package backoff

import (
	"errors"
	"time"
)

// Operation is a function that is executing by Retry() or RetryNotify().
// It should return nil on success, or an error if the operation failed.
type Operation func() error

// OperationWithData is a function that is executed by RetryWithData() or RetryNotifyWithData().
// It should return the result and nil on success, or the zero value and an error if failed.
type OperationWithData[T any] func() (T, error)

// Notify is a notify-on-error function. It receives an operation error and
// backoff delay if the operation failed (with an error).
//
// NOTE that if the backoff policy stated to stop retrying,
// the notify function isn't called.
type Notify func(error, time.Duration)

// PermanentError signals that the operation should not be retried.
// Wrap an error in PermanentError to stop retry immediately.
type PermanentError struct {
	Err error
}

// Error returns the error message.
func (e *PermanentError) Error() string {
	return e.Err.Error()
}

// Unwrap returns the wrapped error.
func (e *PermanentError) Unwrap() error {
	return e.Err
}

// Is reports whether target is a *PermanentError.
func (e *PermanentError) Is(target error) bool {
	_, ok := target.(*PermanentError)
	return ok
}

// Permanent wraps the given err in a *PermanentError.
// Returns nil if err is nil.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &PermanentError{Err: err}
}

// withEmptyData wraps an Operation to return struct{} for use with generic retry.
func (o Operation) withEmptyData() OperationWithData[struct{}] {
	return func() (struct{}, error) {
		return struct{}{}, o()
	}
}

// Retry the operation o until it does not return error or BackOff stops.
// o is guaranteed to be run at least once.
//
// If o returns a *PermanentError, the operation is not retried, and the
// wrapped error is returned.
//
// Retry sleeps the goroutine for the duration returned by BackOff after a
// failed operation returns.
func Retry(o Operation, b BackOff) error {
	return RetryNotify(o, b, nil)
}

// RetryWithData is like Retry but returns data in the response too.
func RetryWithData[T any](o OperationWithData[T], b BackOff) (T, error) {
	return RetryNotifyWithData(o, b, nil)
}

// RetryNotify calls notify function with the error and wait duration
// for each failed attempt before sleep.
func RetryNotify(operation Operation, b BackOff, notify Notify) error {
	return RetryNotifyWithTimer(operation, b, notify, nil)
}

// RetryNotifyWithData is like RetryNotify but returns data in the response too.
func RetryNotifyWithData[T any](operation OperationWithData[T], b BackOff, notify Notify) (T, error) {
	return doRetryNotify(operation, b, notify, nil)
}

// RetryNotifyWithTimer calls notify function with the error and wait duration using the given Timer
// for each failed attempt before sleep.
// A default timer that uses system timer is used when nil is passed.
func RetryNotifyWithTimer(operation Operation, b BackOff, notify Notify, t Timer) error {
	_, err := doRetryNotify(operation.withEmptyData(), b, notify, t)
	return err
}

// RetryNotifyWithTimerAndData is like RetryNotifyWithTimer but returns data in the response too.
func RetryNotifyWithTimerAndData[T any](operation OperationWithData[T], b BackOff, notify Notify, t Timer) (T, error) {
	return doRetryNotify(operation, b, notify, t)
}

// doRetryNotify is the internal implementation of the retry logic.
func doRetryNotify[T any](operation OperationWithData[T], backOff BackOff, notify Notify, timer Timer) (T, error) {
	var (
		err  error
		next time.Duration
		res  T
	)
	if timer == nil {
		timer = &defaultTimer{}
	}

	defer func() {
		timer.Stop()
	}()

	ctx := getContext(backOff)

	backOff.Reset()
	for {
		res, err = operation()
		if err == nil {
			return res, nil
		}

		var permanent *PermanentError
		if errors.As(err, &permanent) {
			return res, permanent.Err
		}

		if next = backOff.NextBackOff(); next == Stop {
			if cerr := ctx.Err(); cerr != nil {
				return res, cerr //nolint:wrapcheck // returning context error directly is intentional
			}
			return res, err
		}

		if notify != nil {
			notify(err, next)
		}

		timer.Start(next)

		select {
		case <-ctx.Done():
			return res, ctx.Err() //nolint:wrapcheck // returning context error directly is intentional
		case <-timer.C():
		}
	}
}
