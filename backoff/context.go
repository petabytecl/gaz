package backoff

import (
	"context"
	"time"
)

// Context is a BackOff that is associated with a context.Context.
// It will return Stop if the context is cancelled.
type Context interface {
	BackOff
	// Context returns the context associated with this backoff.
	Context() context.Context
}

// backOffContext implements the Context interface.
type backOffContext struct {
	BackOff
	ctx context.Context //nolint:containedctx // required for context-aware backoff
}

// Context returns the context associated with this backoff.
func (b *backOffContext) Context() context.Context {
	return b.ctx
}

// NextBackOff returns Stop if the context is cancelled, otherwise delegates
// to the embedded BackOff.
func (b *backOffContext) NextBackOff() time.Duration {
	select {
	case <-b.ctx.Done():
		return Stop
	default:
		return b.BackOff.NextBackOff()
	}
}

// WithContext returns a Context that wraps the given BackOff with context awareness.
// The backoff will return Stop when the context is cancelled.
//
// If the provided BackOff is already context-aware, this will unwrap and re-wrap
// with the new context to avoid double-wrapping.
//
// Panics if ctx is nil.
//
//nolint:ireturn // returns Context interface by design for API consistency
func WithContext(ctx context.Context, backOff BackOff) Context {
	if ctx == nil {
		panic("nil context")
	}

	// Unwrap if already wrapped to avoid double-wrapping
	if b, ok := backOff.(*backOffContext); ok {
		return &backOffContext{
			BackOff: b.BackOff,
			ctx:     ctx,
		}
	}

	return &backOffContext{
		BackOff: backOff,
		ctx:     ctx,
	}
}

// getContext extracts the context from a BackOff if it implements Context,
// otherwise returns context.Background().
// This helper is used by Ticker and retry functions.
func getContext(b BackOff) context.Context {
	if bc, ok := b.(*backOffContext); ok {
		return bc.Context()
	}

	// Also check through backOffTries wrapper
	if bt, ok := b.(*backOffTries); ok {
		return getContext(bt.delegate)
	}

	return context.Background()
}

// Interface compliance assertion.
var _ Context = (*backOffContext)(nil)
