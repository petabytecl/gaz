// Package cron provides scheduled task support for Go applications using robfig/cron.
//
// This package defines the [CronJob] interface and supporting types for managing
// scheduled tasks. Jobs integrate with gaz's lifecycle system for automatic
// startup and graceful shutdown.
//
// # CronJob Interface
//
// The [CronJob] interface defines four methods for scheduled task execution:
//
//   - Name() - Returns a human-readable identifier for logging
//   - Schedule() - Returns the cron expression or predefined schedule
//   - Timeout() - Returns the execution timeout (0 for none)
//   - Run(ctx) - Executes the job with a context for cancellation
//
// # Implementing a CronJob
//
// Jobs are responsible for respecting context cancellation. The context is
// cancelled on application shutdown or when the job's timeout expires.
//
// Example of a simple scheduled job:
//
//	type CleanupJob struct {
//	    db *sql.DB
//	}
//
//	func (j *CleanupJob) Name() string { return "cleanup" }
//
//	func (j *CleanupJob) Schedule() string { return "@daily" }
//
//	func (j *CleanupJob) Timeout() time.Duration { return 5 * time.Minute }
//
//	func (j *CleanupJob) Run(ctx context.Context) error {
//	    _, err := j.db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at < NOW()")
//	    return err
//	}
//
// # Schedule Expressions
//
// Schedule expressions follow the standard 5-field cron format:
//
//	┌───────────── minute (0-59)
//	│ ┌───────────── hour (0-23)
//	│ │ ┌───────────── day of month (1-31)
//	│ │ │ ┌───────────── month (1-12)
//	│ │ │ │ ┌───────────── day of week (0-6, Sunday=0)
//	│ │ │ │ │
//	* * * * *
//
// Predefined schedules are also supported:
//
//   - @yearly (or @annually) - Run once a year at midnight on Jan 1
//   - @monthly - Run once a month at midnight on the first day
//   - @weekly - Run once a week at midnight on Sunday
//   - @daily (or @midnight) - Run once a day at midnight
//   - @hourly - Run once an hour at the beginning of the hour
//   - @every <duration> - Run at fixed intervals (e.g., @every 5m)
//
// # Registration Pattern
//
// Jobs are registered as transient providers and discovered during app.Build():
//
//	di.For[cron.CronJob](c).Transient().Provider(NewCleanupJob)
//
// Jobs returning an empty string from Schedule() are not scheduled (soft disable).
//
// # Concurrency and Lifecycle
//
//   - Overlapping job runs are skipped by default (SkipIfStillRunning)
//   - Jobs gracefully complete on shutdown (scheduler waits for running jobs)
//   - Panics are recovered, logged with stack trace, and don't crash the app
//   - Each execution resolves a fresh job instance from the container (transient)
package cron
