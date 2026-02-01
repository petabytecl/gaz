package backoff

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetry_SucceedsOnFirstTry(t *testing.T) {
	attempts := 0
	err := Retry(func() error {
		attempts++
		return nil
	}, &ZeroBackOff{})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetry_RetriesOnTransientErrors(t *testing.T) {
	attempts := 0
	maxAttempts := 3

	err := Retry(func() error {
		attempts++
		if attempts < maxAttempts {
			return errors.New("transient error")
		}
		return nil
	}, WithMaxRetries(&ZeroBackOff{}, uint64(maxAttempts)))
	if err != nil {
		t.Errorf("expected success, got %v", err)
	}
	if attempts != maxAttempts {
		t.Errorf("expected %d attempts, got %d", maxAttempts, attempts)
	}
}

func TestRetry_StopsOnPermanentError(t *testing.T) {
	attempts := 0
	permanentErr := errors.New("permanent error")

	err := Retry(func() error {
		attempts++
		if attempts == 2 {
			return Permanent(permanentErr)
		}
		return errors.New("transient")
	}, WithMaxRetries(&ZeroBackOff{}, 10))

	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != permanentErr.Error() {
		t.Errorf("expected permanent error, got %v", err)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts (stopped at permanent), got %d", attempts)
	}
}

func TestRetry_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	attempts := 0
	err := Retry(func() error {
		attempts++
		if attempts == 2 {
			cancel()
		}
		return errors.New("error")
	}, WithContext(ctx, WithMaxRetries(&ZeroBackOff{}, 10)))

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestRetryWithData_ReturnsDataOnSuccess(t *testing.T) {
	result, err := RetryWithData(func() (int, error) {
		return 42, nil
	}, &ZeroBackOff{})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}

func TestRetryWithData_ReturnsDataAfterRetries(t *testing.T) {
	attempts := 0
	result, err := RetryWithData(func() (string, error) {
		attempts++
		if attempts < 3 {
			return "", errors.New("transient")
		}
		return "success", nil
	}, WithMaxRetries(&ZeroBackOff{}, 5))
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != "success" {
		t.Errorf("expected 'success', got %s", result)
	}
}

func TestRetryNotify_CallsNotifyOnError(t *testing.T) {
	notifyCalls := 0
	attempts := 0

	err := RetryNotify(func() error {
		attempts++
		if attempts < 3 {
			return errors.New("error")
		}
		return nil
	}, WithMaxRetries(&ZeroBackOff{}, 5), func(err error, d time.Duration) {
		notifyCalls++
	})
	if err != nil {
		t.Errorf("expected success, got %v", err)
	}
	// Should be called for each failure before retry (2 failures)
	if notifyCalls != 2 {
		t.Errorf("expected 2 notify calls, got %d", notifyCalls)
	}
}

func TestPermanent_NilReturnsNil(t *testing.T) {
	if Permanent(nil) != nil {
		t.Error("Permanent(nil) should return nil")
	}
}

func TestPermanentError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	pe := &PermanentError{Err: inner}

	if !errors.Is(pe.Unwrap(), inner) {
		t.Error("Unwrap should return inner error")
	}
}

func TestPermanentError_Is(t *testing.T) {
	pe := &PermanentError{Err: errors.New("test")}

	if !errors.Is(pe, &PermanentError{}) {
		t.Error("errors.Is should return true for PermanentError")
	}
}

func TestPermanentError_ErrorsAs(t *testing.T) {
	originalErr := errors.New("test error")
	wrapped := Permanent(originalErr)

	var pe *PermanentError
	if !errors.As(wrapped, &pe) {
		t.Fatal("errors.As should extract PermanentError")
	}

	if !errors.Is(pe.Err, originalErr) {
		t.Error("inner error should be preserved")
	}
}

func TestRetry_ReturnsLastErrorOnStop(t *testing.T) {
	lastErr := errors.New("final error")
	attempts := 0

	// WithMaxRetries(3) allows 3 retries AFTER the first attempt
	// So we get: 1 initial + 3 retries = 4 total attempts
	err := Retry(func() error {
		attempts++
		return lastErr
	}, WithMaxRetries(&ZeroBackOff{}, 3))

	if !errors.Is(err, lastErr) {
		t.Errorf("expected last error, got %v", err)
	}
	// 1 initial attempt + 3 retries = 4 total
	if attempts != 4 {
		t.Errorf("expected 4 attempts (1 initial + 3 retries), got %d", attempts)
	}
}
