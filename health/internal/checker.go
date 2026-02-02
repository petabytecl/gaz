package internal

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultTimeout is the default timeout for health checks (5 seconds).
const DefaultTimeout = 5 * time.Second

// Checker executes health checks and returns aggregated results.
type Checker interface {
	// Check runs all configured health checks and returns the result.
	// The context may contain a deadline that will be respected.
	Check(ctx context.Context) CheckerResult
}

// CheckerResult holds the aggregated health status and details.
type CheckerResult struct {
	// Status is the aggregated availability status.
	Status AvailabilityStatus
	// Details contains per-check results keyed by check name.
	Details map[string]CheckResult
}

// CheckerOption configures the Checker.
type CheckerOption func(*checkerConfig)

// checkerConfig holds the configuration for a checker.
type checkerConfig struct {
	checks         map[string]*internalCheck
	defaultTimeout time.Duration
}

// internalCheck wraps Check with critical flag defaulting.
type internalCheck struct {
	name     string
	check    func(ctx context.Context) error
	timeout  time.Duration
	critical bool
}

// checker implements the Checker interface.
type checker struct {
	checks         map[string]*internalCheck
	defaultTimeout time.Duration
}

// NewChecker creates a new Checker with the given options.
//
//nolint:ireturn // Checker interface is the intended return type for flexibility
func NewChecker(opts ...CheckerOption) Checker {
	cfg := &checkerConfig{
		checks:         make(map[string]*internalCheck),
		defaultTimeout: DefaultTimeout,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return &checker{
		checks:         cfg.checks,
		defaultTimeout: cfg.defaultTimeout,
	}
}

// WithCheck adds a health check to the checker.
// If Critical is not explicitly set (zero value), the check is treated as critical.
func WithCheck(check Check) CheckerOption {
	return func(cfg *checkerConfig) {
		// Determine critical status: default to true if not explicitly set
		critical := true
		if check.criticalSet {
			critical = check.Critical
		} else if check.Critical {
			// Explicitly set to true
			critical = true
		}
		// Note: If user just creates Check{} without setting Critical,
		// criticalSet will be false and Critical will be false,
		// so we default to critical = true (safe default per CONTEXT.md)

		cfg.checks[check.Name] = &internalCheck{
			name:     check.Name,
			check:    check.Check,
			timeout:  check.Timeout,
			critical: critical,
		}
	}
}

// WithTimeout sets the default timeout for checks (default 5s).
func WithTimeout(timeout time.Duration) CheckerOption {
	return func(cfg *checkerConfig) {
		cfg.defaultTimeout = timeout
	}
}

// Check runs all configured health checks and returns the result.
func (c *checker) Check(ctx context.Context) CheckerResult {
	result := CheckerResult{
		Status:  StatusUp, // Default to up - no checks = healthy
		Details: make(map[string]CheckResult),
	}

	if len(c.checks) == 0 {
		// No checks configured - healthy by default (matches alexliesenfeld/health behavior)
		return result
	}

	// Run checks in parallel
	results := c.runChecks(ctx)

	// Aggregate results
	hasCritical := false
	criticalFailed := false

	for name, checkResult := range results {
		result.Details[name] = checkResult

		check := c.checks[name]
		if check.critical {
			hasCritical = true
			if checkResult.Status != StatusUp {
				criticalFailed = true
			}
		}
	}

	// Determine overall status
	switch {
	case !hasCritical:
		// No critical checks, but we have checks - status is up (graceful degradation)
		result.Status = StatusUp
	case criticalFailed:
		result.Status = StatusDown
	default:
		result.Status = StatusUp
	}

	return result
}

// runChecks executes all checks in parallel and returns results.
func (c *checker) runChecks(ctx context.Context) map[string]CheckResult {
	results := make(map[string]CheckResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, check := range c.checks {
		wg.Add(1)
		go func(check *internalCheck) {
			defer wg.Done()
			result := c.executeCheck(ctx, check)
			mu.Lock()
			results[check.name] = result
			mu.Unlock()
		}(check)
	}

	wg.Wait()
	return results
}

// executeCheck runs a single check with timeout and panic recovery.
func (c *checker) executeCheck(ctx context.Context, check *internalCheck) CheckResult {
	timeout := check.timeout
	if timeout == 0 {
		timeout = c.defaultTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result := CheckResult{
		Status:    StatusUp,
		Timestamp: time.Now().UTC(),
	}

	// Panic recovery - wrap check execution
	err := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic: %v", r)
			}
		}()
		return check.check(ctx)
	}()
	if err != nil {
		result.Status = StatusDown
		result.Error = err
	}

	return result
}
