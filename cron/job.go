package cron

import (
	"context"
	"time"
)

// CronJob defines the interface for scheduled tasks with lifecycle management.
//
// Jobs are scheduled using cron expressions and execute with context support
// for cancellation and timeout. They integrate with gaz's lifecycle system
// for automatic discovery, startup, and graceful shutdown.
//
// # Contract
//
// Implementations must follow these rules:
//
//   - Name() must return a non-empty, unique string identifier. This name is used
//     for logging and debugging.
//
//   - Schedule() must return a valid cron expression (e.g., "*/5 * * * *") or
//     predefined schedule (e.g., "@hourly", "@daily"). Return an empty string
//     to disable the job without removing it from registration.
//
//   - Timeout() returns the maximum duration for job execution. Return 0 for no
//     timeout. When timeout expires, the context passed to Run() is cancelled.
//
//   - Run() executes the job. The context is cancelled on app shutdown or timeout.
//     Implementations must check ctx.Done() in long-running operations and exit
//     promptly when cancelled.
//
// # Example
//
//	type ReportJob struct {
//	    mailer  *mail.Client
//	    queries *db.Queries
//	}
//
//	func (j *ReportJob) Name() string { return "daily-report" }
//
//	func (j *ReportJob) Schedule() string { return "@daily" }
//
//	func (j *ReportJob) Timeout() time.Duration { return 10 * time.Minute }
//
//	func (j *ReportJob) Run(ctx context.Context) error {
//	    data, err := j.queries.GetDailyStats(ctx)
//	    if err != nil {
//	        return err
//	    }
//	    return j.mailer.SendReport(ctx, data)
//	}
//
//nolint:revive // CronJob stutters but renaming would break public API
type CronJob interface {
	// Name returns a human-readable identifier for logging.
	//
	// The name should be descriptive and unique among registered jobs.
	// It is used in log messages to identify which job is starting,
	// finishing, or encountering errors.
	Name() string

	// Schedule returns the cron expression or predefined schedule.
	//
	// Standard 5-field cron expressions are supported:
	//   "*/5 * * * *"  - Every 5 minutes
	//   "0 * * * *"    - Every hour at minute 0
	//   "0 0 * * *"    - Every day at midnight
	//   "0 0 * * 0"    - Every Sunday at midnight
	//
	// Predefined schedules are also supported:
	//   "@yearly"      - Once a year (midnight, Jan 1)
	//   "@monthly"     - Once a month (midnight, first day)
	//   "@weekly"      - Once a week (midnight, Sunday)
	//   "@daily"       - Once a day (midnight)
	//   "@hourly"      - Once an hour
	//   "@every 5m"    - Every 5 minutes
	//
	// Return an empty string to disable this job. The job remains
	// registered but won't be scheduled for execution.
	Schedule() string

	// Timeout returns the maximum duration for job execution.
	//
	// If the job runs longer than this duration, the context passed to
	// Run() is cancelled. Return 0 for no timeout limit.
	//
	// Choose timeouts based on expected job duration plus margin for
	// variability. Too short causes premature cancellation; too long
	// delays detection of stuck jobs.
	Timeout() time.Duration

	// Run executes the job with the given context.
	//
	// The context is derived from the application context and is cancelled
	// when the app shuts down or when the job's timeout expires. Long-running
	// operations must check ctx.Done() and exit promptly when cancelled.
	//
	// Return nil on success. Errors are logged but don't trigger immediate
	// retries - the job simply runs again at the next scheduled time.
	//
	// Panics are recovered and logged with stack traces. The app won't
	// crash due to a panicking job.
	Run(ctx context.Context) error
}
