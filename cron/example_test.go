package cron_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/petabytecl/gaz/cron"
	"github.com/petabytecl/gaz/di"
)

// CleanupJob demonstrates implementing the CronJob interface.
// Jobs define Name, Schedule, Timeout, and Run methods.
type CleanupJob struct {
	runCount int
}

func (j *CleanupJob) Name() string     { return "cleanup" }
func (j *CleanupJob) Schedule() string { return "0 * * * *" } // Every hour
func (j *CleanupJob) Timeout() time.Duration {
	return 5 * time.Minute
}

func (j *CleanupJob) Run(ctx context.Context) error {
	j.runCount++
	fmt.Println("running cleanup job")
	return nil
}

// Example_job demonstrates implementing the CronJob interface.
// Jobs define Name, Schedule, Timeout, and Run methods for scheduled execution.
func Example_job() {
	job := &CleanupJob{}

	fmt.Println("name:", job.Name())
	fmt.Println("schedule:", job.Schedule())
	fmt.Println("timeout:", job.Timeout())
	_ = job.Run(context.Background())
	// Output:
	// name: cleanup
	// schedule: 0 * * * *
	// timeout: 5m0s
	// running cleanup job
}

// ExampleNewScheduler demonstrates creating a cron scheduler.
// The scheduler uses cron/internal with DI-aware job execution.
// Note: In real applications, use gaz.New() which creates the scheduler automatically.
func ExampleNewScheduler() {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	resolver := cron.NewMockResolver()

	scheduler := cron.NewScheduler(resolver, context.Background(), logger)

	fmt.Println("scheduler created")
	fmt.Println("running:", scheduler.IsRunning())
	// Output:
	// scheduler created
	// running: false
}

// ExampleScheduler_RegisterJob demonstrates registering jobs with the scheduler.
// Jobs are registered with a service name for DI resolution and a schedule.
func ExampleScheduler_RegisterJob() {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	resolver := cron.NewMockResolver()
	scheduler := cron.NewScheduler(resolver, context.Background(), logger)

	err := scheduler.RegisterJob("*CleanupJob", "cleanup", "@hourly", 5*time.Minute)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("job registered")
	fmt.Println("job count:", scheduler.JobCount())
	// Output:
	// job registered
	// job count: 1
}

// ExampleScheduler_OnStart demonstrates starting the cron scheduler.
// The scheduler begins executing jobs according to their schedules.
// Note: This example shows lifecycle but cannot verify job execution timing.
func ExampleScheduler_OnStart() {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	resolver := cron.NewMockResolver()
	scheduler := cron.NewScheduler(resolver, context.Background(), logger)

	ctx := context.Background()
	err := scheduler.OnStart(ctx)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("running:", scheduler.IsRunning())

	_ = scheduler.OnStop(ctx)
	fmt.Println("running after stop:", scheduler.IsRunning())
	// Output:
	// running: true
	// running after stop: false
}

// ExampleModule demonstrates the cron module function for DI.
// The module integrates cron scheduling into a gaz application.
func ExampleModule() {
	c := di.New()

	// Register logger (normally done by gaz.New())
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	_ = di.For[*slog.Logger](c).Instance(logger)

	// Apply cron module
	err := cron.Module(c)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("module registered scheduler")
	// Output: module registered scheduler
}

// ExampleSimpleJob demonstrates using SimpleJob for testing.
// SimpleJob tracks Run calls without mock complexity.
func ExampleSimpleJob() {
	job := cron.NewSimpleJob("test-job", "@every 1m")

	fmt.Println("name:", job.Name())
	fmt.Println("schedule:", job.Schedule())
	fmt.Println("timeout:", job.Timeout())
	fmt.Println("run count before:", job.RunCount.Load())

	_ = job.Run(context.Background())
	fmt.Println("run count after:", job.RunCount.Load())
	// Output:
	// name: test-job
	// schedule: @every 1m
	// timeout: 30s
	// run count before: 0
	// run count after: 1
}

// ExampleMockJob demonstrates using MockJob for testing.
// MockJob uses testify/mock for flexible expectation setup.
func ExampleMockJob() {
	m := cron.NewMockJob("my-job")

	// MockJob comes with default expectations:
	// - Name() returns the given name
	// - Schedule() returns "@every 1m"
	// - Timeout() returns 30s
	// - Run() returns nil
	fmt.Println("name:", m.Name())
	fmt.Println("schedule:", m.Schedule())
	// Output:
	// name: my-job
	// schedule: @every 1m
}

// ExampleMockResolver demonstrates using MockResolver for testing.
// MockResolver mocks the container resolution for job execution.
func ExampleMockResolver() {
	resolver := cron.NewMockResolver()

	// Set up an expectation
	resolver.On("ResolveByName", "*MyJob", []string{}).Return(&CleanupJob{}, nil)

	// Resolve - would be called by the scheduler when running jobs
	result, err := resolver.ResolveByName("*MyJob", []string{})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	job := result.(*CleanupJob)
	fmt.Println("resolved job name:", job.Name())
	// Output: resolved job name: cleanup
}

// ExampleTestScheduler demonstrates creating a test scheduler.
// TestScheduler creates a Scheduler with a discard logger for tests.
func ExampleTestScheduler() {
	scheduler := cron.TestScheduler(nil, nil)

	err := scheduler.RegisterJob("*TestJob", "test", "@every 5m", 30*time.Second)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("job count:", scheduler.JobCount())
	// Output: job count: 1
}

// Example_cronExpression demonstrates common cron schedule expressions.
// These are the most frequently used patterns in applications.
func Example_cronExpression() {
	expressions := []struct {
		expr string
		desc string
	}{
		{"@every 5m", "Every 5 minutes"},
		{"@hourly", "Every hour"},
		{"@daily", "Daily at midnight"},
		{"@weekly", "Weekly on Sunday"},
		{"0 * * * *", "Top of every hour"},
		{"0 0 * * *", "Daily at midnight (cron format)"},
		{"0 0 * * 0", "Sunday at midnight"},
		{"*/5 * * * *", "Every 5 minutes (cron format)"},
		{"0 9-17 * * 1-5", "Weekdays 9am-5pm hourly"},
	}

	for _, e := range expressions {
		fmt.Printf("%s - %s\n", e.expr, e.desc)
	}
	// Output:
	// @every 5m - Every 5 minutes
	// @hourly - Every hour
	// @daily - Daily at midnight
	// @weekly - Weekly on Sunday
	// 0 * * * * - Top of every hour
	// 0 0 * * * - Daily at midnight (cron format)
	// 0 0 * * 0 - Sunday at midnight
	// */5 * * * * - Every 5 minutes (cron format)
	// 0 9-17 * * 1-5 - Weekdays 9am-5pm hourly
}

// Example_disabledJob demonstrates disabling a job with empty schedule.
// Jobs can be soft-disabled by returning an empty string from Schedule().
func Example_disabledJob() {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	resolver := cron.NewMockResolver()
	scheduler := cron.NewScheduler(resolver, context.Background(), logger)

	// Empty schedule disables the job without error
	err := scheduler.RegisterJob("*DisabledJob", "disabled", "", 0)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("job count:", scheduler.JobCount())
	fmt.Println("job disabled (not scheduled)")
	// Output:
	// job count: 0
	// job disabled (not scheduled)
}

// Example_moduleIntegration demonstrates using Module with di.Container.
// This shows how cron module integrates into the DI system.
func Example_moduleIntegration() {
	c := di.New()

	// Register logger (normally done by gaz.New())
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	_ = di.For[*slog.Logger](c).Instance(logger)

	// Apply cron module
	if err := cron.Module(c); err != nil {
		fmt.Println("error:", err)
		return
	}

	// Build container
	if err := c.Build(); err != nil {
		fmt.Println("build error:", err)
		return
	}

	// Resolve Scheduler
	sched, err := di.Resolve[*cron.Scheduler](c)
	if err != nil {
		fmt.Println("resolve error:", err)
		return
	}

	fmt.Println("module applied")
	fmt.Println("scheduler running:", sched.IsRunning())
	// Output:
	// module applied
	// scheduler running: false
}

// Example_jobWithTimeout demonstrates a job with custom timeout.
// Timeout is enforced by the scheduler - context is cancelled when exceeded.
func Example_jobWithTimeout() {
	job := cron.NewSimpleJob("slow-job", "@daily")
	job.SetTimeout(10 * time.Minute)

	fmt.Println("name:", job.Name())
	fmt.Println("timeout:", job.Timeout())
	// Output:
	// name: slow-job
	// timeout: 10m0s
}

// Example_healthCheck demonstrates scheduler health check.
// HealthCheck returns nil when running, error when stopped.
func Example_healthCheck() {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	resolver := cron.NewMockResolver()
	scheduler := cron.NewScheduler(resolver, context.Background(), logger)

	// Not running yet
	err := scheduler.HealthCheck(context.Background())
	if err != nil {
		fmt.Println("health check before start:", err)
	}

	// Start the scheduler
	_ = scheduler.OnStart(context.Background())
	err = scheduler.HealthCheck(context.Background())
	if err != nil {
		fmt.Println("error:", err)
	} else {
		fmt.Println("health check after start: ok")
	}

	_ = scheduler.OnStop(context.Background())
	// Output:
	// health check before start: cron: scheduler not running
	// health check after start: ok
}
