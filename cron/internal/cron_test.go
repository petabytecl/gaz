package internal

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Many tests schedule a job for every second, and then wait at most a second
// for it to run.  This amount is just slightly larger than 1 second to
// compensate for a few milliseconds of runtime.
//
// Timing note: These tests are inherently limited by cron's 1-second minimum
// schedule resolution. Each test needs at least one cron tick (~1s). Channel-based
// signaling replaces time.Sleep where possible, and t.Parallel() is used on
// independent tests (each creates its own Cron instance) to reduce wall-clock time.
const defaultWait = 1*time.Second + 50*time.Millisecond

type syncWriter struct {
	wr bytes.Buffer
	m  sync.Mutex
}

func (sw *syncWriter) Write(data []byte) (int, error) {
	sw.m.Lock()
	defer sw.m.Unlock()
	return sw.wr.Write(data) //nolint:wrapcheck // test helper
}

func (sw *syncWriter) String() string {
	sw.m.Lock()
	defer sw.m.Unlock()
	return sw.wr.String()
}

func newBufLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestFuncPanicRecovery(t *testing.T) {
	t.Parallel()
	panicked := make(chan struct{}, 1)
	cron := New(WithParser(secondParser),
		WithChain(Recover(newBufLogger())))
	cron.Start()
	defer cron.Stop()
	_, err := cron.AddFunc("* * * * * ?", func() {
		select {
		case panicked <- struct{}{}:
		default:
		}
		panic("YOLO")
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Job panics but recovers - no crash
	select {
	case <-panicked:
	case <-time.After(defaultWait):
		t.Fatal("expected job to run and panic")
	}
}

type DummyJob struct{}

func (d DummyJob) Run() {
	panic("YOLO")
}

func TestJobPanicRecovery(t *testing.T) {
	t.Parallel()
	ran := make(chan struct{}, 1)
	cron := New(WithParser(secondParser),
		WithChain(Recover(newBufLogger())))
	cron.Start()
	defer cron.Stop()
	_, err := cron.AddJob("* * * * * ?", FuncJob(func() {
		select {
		case ran <- struct{}{}:
		default:
		}
		panic("YOLO")
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Job panics but recovers - no crash
	select {
	case <-ran:
	case <-time.After(defaultWait):
		t.Fatal("expected job to run and panic")
	}
}

// Start and stop cron with no entries.
func TestNoEntries(t *testing.T) {
	t.Parallel()
	cron := newWithSeconds()
	cron.Start()

	select {
	case <-time.After(defaultWait):
		t.Fatal("expected cron will be stopped immediately")
	case <-stop(cron):
	}
}

// Start, stop, then add an entry. Verify entry doesn't run.
func TestStopCausesJobsToNotRun(t *testing.T) {
	t.Parallel()
	wg := &sync.WaitGroup{}
	wg.Add(1)

	cron := newWithSeconds()
	cron.Start()
	cron.Stop()
	_, _ = cron.AddFunc("* * * * * ?", func() { wg.Done() })

	select {
	case <-time.After(defaultWait):
		// No job ran!
	case <-wait(wg):
		t.Fatal("expected stopped cron does not run any job")
	}
}

// Add a job, start cron, expect it runs.
func TestAddBeforeRunning(t *testing.T) {
	t.Parallel()
	wg := &sync.WaitGroup{}
	wg.Add(1)

	cron := newWithSeconds()
	_, _ = cron.AddFunc("* * * * * ?", func() { wg.Done() })
	cron.Start()
	defer cron.Stop()

	// Give cron 2 seconds to run our job (which is always activated).
	select {
	case <-time.After(defaultWait):
		t.Fatal("expected job runs")
	case <-wait(wg):
	}
}

// Start cron, add a job, expect it runs.
func TestAddWhileRunning(t *testing.T) {
	t.Parallel()
	wg := &sync.WaitGroup{}
	wg.Add(1)

	cron := newWithSeconds()
	cron.Start()
	defer cron.Stop()
	_, _ = cron.AddFunc("* * * * * ?", func() { wg.Done() })

	select {
	case <-time.After(defaultWait):
		t.Fatal("expected job runs")
	case <-wait(wg):
	}
}

// Test for #34. Adding a job after calling start results in multiple job invocations.
func TestAddWhileRunningWithDelay(t *testing.T) {
	t.Parallel()
	cron := newWithSeconds()
	cron.Start()
	defer cron.Stop()
	// Original test used 5s delay. A 1s delay is sufficient to verify the
	// bug fix: adding a job after some delay should not trigger multiple invocations.
	time.Sleep(1 * time.Second)
	var calls int64
	_, _ = cron.AddFunc("* * * * * *", func() { atomic.AddInt64(&calls, 1) })

	<-time.After(defaultWait)
	if atomic.LoadInt64(&calls) != 1 {
		t.Errorf("called %d times, expected 1\n", calls)
	}
}

// Add a job, remove a job, start cron, expect nothing runs.
func TestRemoveBeforeRunning(t *testing.T) {
	t.Parallel()
	wg := &sync.WaitGroup{}
	wg.Add(1)

	cron := newWithSeconds()
	id, _ := cron.AddFunc("* * * * * ?", func() { wg.Done() })
	cron.Remove(id)
	cron.Start()
	defer cron.Stop()

	select {
	case <-time.After(defaultWait):
		// Success, shouldn't run
	case <-wait(wg):
		t.FailNow()
	}
}

// Start cron, add a job, remove it, expect it doesn't run.
func TestRemoveWhileRunning(t *testing.T) {
	t.Parallel()
	wg := &sync.WaitGroup{}
	wg.Add(1)

	cron := newWithSeconds()
	cron.Start()
	defer cron.Stop()
	id, _ := cron.AddFunc("* * * * * ?", func() { wg.Done() })
	cron.Remove(id)

	select {
	case <-time.After(defaultWait):
	case <-wait(wg):
		t.FailNow()
	}
}

// Test timing with Entries.
func TestSnapshotEntries(t *testing.T) {
	t.Parallel()
	wg := &sync.WaitGroup{}
	wg.Add(1)

	cron := newWithSeconds()
	_, _ = cron.AddFunc("* * * * * ?", func() { wg.Done() })
	cron.Start()
	defer cron.Stop()

	// Call Entries mid-run and verify the job still fires.
	cron.Entries()

	select {
	case <-time.After(defaultWait):
		t.Error("expected job runs")
	case <-wait(wg):
	}
}

// Test that the entries are correctly sorted.
// Add a bunch of long-in-the-future entries, and an immediate entry, and ensure
// that the immediate entry runs immediately.
// Also: Test that multiple jobs run in the same instant.
func TestMultipleEntries(t *testing.T) {
	t.Parallel()
	wg := &sync.WaitGroup{}
	wg.Add(2)

	cron := newWithSeconds()
	_, _ = cron.AddFunc("0 0 0 1 1 ?", func() {})
	_, _ = cron.AddFunc("* * * * * ?", func() { wg.Done() })
	id1, _ := cron.AddFunc("* * * * * ?", func() { t.Fatal() })
	id2, _ := cron.AddFunc("* * * * * ?", func() { t.Fatal() })
	_, _ = cron.AddFunc("0 0 0 31 12 ?", func() {})
	_, _ = cron.AddFunc("* * * * * ?", func() { wg.Done() })

	cron.Remove(id1)
	cron.Start()
	cron.Remove(id2)
	defer cron.Stop()

	select {
	case <-time.After(defaultWait):
		t.Error("expected job run in proper order")
	case <-wait(wg):
	}
}

// Test running the same job twice.
func TestRunningJobTwice(t *testing.T) {
	t.Parallel()
	wg := &sync.WaitGroup{}
	wg.Add(2)

	cron := newWithSeconds()
	_, _ = cron.AddFunc("0 0 0 1 1 ?", func() {})
	_, _ = cron.AddFunc("0 0 0 31 12 ?", func() {})
	_, _ = cron.AddFunc("* * * * * ?", func() { wg.Done() })

	cron.Start()
	defer cron.Stop()

	select {
	case <-time.After(2 * defaultWait):
		t.Error("expected job fires 2 times")
	case <-wait(wg):
	}
}

func TestRunningMultipleSchedules(t *testing.T) {
	t.Parallel()
	wg := &sync.WaitGroup{}
	wg.Add(2)

	cron := newWithSeconds()
	_, _ = cron.AddFunc("0 0 0 1 1 ?", func() {})
	_, _ = cron.AddFunc("0 0 0 31 12 ?", func() {})
	_, _ = cron.AddFunc("* * * * * ?", func() { wg.Done() })
	cron.Schedule(Every(time.Minute), FuncJob(func() {}))
	cron.Schedule(Every(time.Second), FuncJob(func() { wg.Done() }))
	cron.Schedule(Every(time.Hour), FuncJob(func() {}))

	cron.Start()
	defer cron.Stop()

	select {
	case <-time.After(2 * defaultWait):
		t.Error("expected job fires 2 times")
	case <-wait(wg):
	}
}

// Test that the cron is run in the local time zone (as opposed to UTC).
func TestLocalTimezone(t *testing.T) {
	t.Parallel()
	wg := &sync.WaitGroup{}
	wg.Add(2)

	now := time.Now()
	// FIX: Issue #205
	// This calculation doesn't work in seconds 58 or 59.
	// Take the easy way out and sleep.
	if now.Second() >= 58 {
		time.Sleep(2 * time.Second)
		now = time.Now()
	}
	spec := fmt.Sprintf("%d,%d %d %d %d %d ?",
		now.Second()+1, now.Second()+2, now.Minute(), now.Hour(), now.Day(), now.Month())

	cron := newWithSeconds()
	_, _ = cron.AddFunc(spec, func() { wg.Done() })
	cron.Start()
	defer cron.Stop()

	select {
	case <-time.After(defaultWait * 2):
		t.Error("expected job fires 2 times")
	case <-wait(wg):
	}
}

// Test that the cron is run in the given time zone (as opposed to local).
func TestNonLocalTimezone(t *testing.T) {
	t.Parallel()
	wg := &sync.WaitGroup{}
	wg.Add(2)

	loc, err := time.LoadLocation("Atlantic/Cape_Verde")
	if err != nil {
		t.Logf("Failed to load time zone Atlantic/Cape_Verde: %+v", err)
		t.Fail()
	}

	now := time.Now().In(loc)
	// FIX: Issue #205
	// This calculation doesn't work in seconds 58 or 59.
	// Take the easy way out and sleep.
	if now.Second() >= 58 {
		time.Sleep(2 * time.Second)
		now = time.Now().In(loc)
	}
	spec := fmt.Sprintf("%d,%d %d %d %d %d ?",
		now.Second()+1, now.Second()+2, now.Minute(), now.Hour(), now.Day(), now.Month())

	cron := New(WithLocation(loc), WithParser(secondParser))
	_, _ = cron.AddFunc(spec, func() { wg.Done() })
	cron.Start()
	defer cron.Stop()

	select {
	case <-time.After(defaultWait * 2):
		t.Error("expected job fires 2 times")
	case <-wait(wg):
	}
}

// Test that calling stop before start silently returns without
// blocking the stop channel.
func TestStopWithoutStart(t *testing.T) {
	t.Parallel()
	cron := New()
	cron.Stop()
}

type testJob struct {
	wg   *sync.WaitGroup
	name string
}

func (t testJob) Run() {
	t.wg.Done()
}

// Test that adding an invalid job spec returns an error.
func TestInvalidJobSpec(t *testing.T) {
	t.Parallel()
	cron := New()
	_, err := cron.AddJob("this will not parse", nil)
	if err == nil {
		t.Errorf("expected an error with invalid spec, got nil")
	}
}

// Test blocking run method behaves as Start().
func TestBlockingRun(t *testing.T) {
	t.Parallel()
	wg := &sync.WaitGroup{}
	wg.Add(1)

	cron := newWithSeconds()
	_, _ = cron.AddFunc("* * * * * ?", func() { wg.Done() })

	unblockChan := make(chan struct{})

	go func() {
		cron.Run()
		close(unblockChan)
	}()
	defer cron.Stop()

	select {
	case <-time.After(defaultWait):
		t.Error("expected job fires")
	case <-unblockChan:
		t.Error("expected that Run() blocks")
	case <-wait(wg):
	}
}

// Test that double-running is a no-op.
func TestStartNoop(t *testing.T) {
	t.Parallel()
	tickChan := make(chan struct{}, 2)

	cron := newWithSeconds()
	_, _ = cron.AddFunc("* * * * * ?", func() {
		tickChan <- struct{}{}
	})

	cron.Start()
	defer cron.Stop()

	// Wait for the first firing to ensure the runner is going
	<-tickChan

	cron.Start()

	<-tickChan

	// Fail if this job fires again in a short period, indicating a double-run
	select {
	case <-time.After(time.Millisecond):
	case <-tickChan:
		t.Error("expected job fires exactly twice")
	}
}

// Simple test using Runnables.
func TestJob(t *testing.T) {
	t.Parallel()
	wg := &sync.WaitGroup{}
	wg.Add(1)

	cron := newWithSeconds()
	_, _ = cron.AddJob("0 0 0 30 Feb ?", testJob{wg, "job0"})
	_, _ = cron.AddJob("0 0 0 1 1 ?", testJob{wg, "job1"})
	job2, _ := cron.AddJob("* * * * * ?", testJob{wg, "job2"})
	_, _ = cron.AddJob("1 0 0 1 1 ?", testJob{wg, "job3"})
	cron.Schedule(Every(5*time.Second+5*time.Nanosecond), testJob{wg, "job4"})
	job5 := cron.Schedule(Every(5*time.Minute), testJob{wg, "job5"})

	// Test getting an Entry pre-Start.
	if actualName := cron.Entry(job2).Job.(testJob).name; actualName != "job2" {
		t.Error("wrong job retrieved:", actualName)
	}
	if actualName := cron.Entry(job5).Job.(testJob).name; actualName != "job5" {
		t.Error("wrong job retrieved:", actualName)
	}

	cron.Start()
	defer cron.Stop()

	select {
	case <-time.After(defaultWait):
		t.FailNow()
	case <-wait(wg):
	}

	// Ensure the entries are in the right order.
	expecteds := []string{"job2", "job4", "job5", "job1", "job3", "job0"}

	actuals := make([]string, 0, len(cron.Entries()))
	for _, entry := range cron.Entries() {
		actuals = append(actuals, entry.Job.(testJob).name)
	}

	for i, expected := range expecteds {
		if actuals[i] != expected {
			t.Fatalf("Jobs not in the right order.  (expected) %s != %s (actual)", expecteds, actuals)
		}
	}

	// Test getting Entries.
	if actualName := cron.Entry(job2).Job.(testJob).name; actualName != "job2" {
		t.Error("wrong job retrieved:", actualName)
	}
	if actualName := cron.Entry(job5).Job.(testJob).name; actualName != "job5" {
		t.Error("wrong job retrieved:", actualName)
	}
}

// Issue #206
// Ensure that the next run of a job after removing an entry is accurate.
func TestScheduleAfterRemoval(t *testing.T) {
	t.Parallel()
	var wg1 sync.WaitGroup
	var wg2 sync.WaitGroup
	wg1.Add(1)
	wg2.Add(1)

	// The first time this job is run, set a timer and remove the other job
	// 750ms later. Correct behavior would be to still run the job again in
	// 250ms, but the bug would cause it to run instead 1s later.

	var calls int
	var mu sync.Mutex

	cron := newWithSeconds()
	hourJob := cron.Schedule(Every(time.Hour), FuncJob(func() {}))
	cron.Schedule(Every(time.Second), FuncJob(func() {
		mu.Lock()
		defer mu.Unlock()
		switch calls {
		case 0:
			wg1.Done()
			calls++
		case 1:
			time.Sleep(750 * time.Millisecond)
			cron.Remove(hourJob)
			calls++
		case 2:
			calls++
			wg2.Done()
		case 3:
			panic("unexpected 3rd call")
		}
	}))

	cron.Start()
	defer cron.Stop()

	// the first run might be any length of time 0 - 1s, since the schedule
	// rounds to the second. wait for the first run to true up.
	wg1.Wait()

	select {
	case <-time.After(2 * defaultWait):
		t.Error("expected job fires 2 times")
	case <-wait(&wg2):
	}
}

type ZeroSchedule struct{}

func (*ZeroSchedule) Next(time.Time) time.Time {
	return time.Time{}
}

// Tests that job without time does not run

// Test Entry.Valid().
func TestEntryValid(t *testing.T) {
	t.Parallel()
	e := Entry{}
	if e.Valid() {
		t.Error("zero entry should be invalid")
	}
	e.ID = 1
	if !e.Valid() {
		t.Error("non-zero entry should be valid")
	}
}

// Test Cron.Location().
func TestCronLocation(t *testing.T) {
	t.Parallel()
	loc, _ := time.LoadLocation("America/New_York")
	cron := New(WithLocation(loc))
	if cron.Location() != loc {
		t.Error("expected correct location")
	}
}

// Test Cron.Entry() with non-existent ID.
func TestCronEntryNotFound(t *testing.T) {
	t.Parallel()
	cron := New()
	cron.Start()
	defer cron.Stop()
	e := cron.Entry(123)
	if e.Valid() {
		t.Error("expected invalid entry for non-existent ID")
	}
}

// Test Cron.Run() idempotency.
func TestCronRunIdempotent(t *testing.T) {
	t.Parallel()
	cron := New()
	cron.Start() // Sets running = true
	defer cron.Stop()

	// Call Run() while already running. It should return immediately.
	done := make(chan struct{})
	go func() {
		cron.Run()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(defaultWait):
		t.Error("Run() should return immediately if already running")
	}
}

//nolint:gocognit,funlen // complex test from vendored robfig/cron with multiple subtests
func TestStopAndWait(t *testing.T) {
	t.Parallel()
	t.Run("nothing running, returns immediately", func(t *testing.T) {
		t.Parallel()
		cron := newWithSeconds()
		cron.Start()
		ctx := cron.Stop()
		select {
		case <-ctx.Done():
		case <-time.After(time.Millisecond):
			t.Error("context was not done immediately")
		}
	})

	t.Run("repeated calls to Stop", func(t *testing.T) {
		t.Parallel()
		cron := newWithSeconds()
		cron.Start()
		_ = cron.Stop()
		time.Sleep(time.Millisecond)
		ctx := cron.Stop()
		select {
		case <-ctx.Done():
		case <-time.After(time.Millisecond):
			t.Error("context was not done immediately")
		}
	})

	t.Run("a couple fast jobs added, still returns immediately", func(t *testing.T) {
		t.Parallel()
		cron := newWithSeconds()
		_, _ = cron.AddFunc("* * * * * *", func() {})
		cron.Start()
		_, _ = cron.AddFunc("* * * * * *", func() {})
		_, _ = cron.AddFunc("* * * * * *", func() {})
		_, _ = cron.AddFunc("* * * * * *", func() {})
		time.Sleep(time.Second)
		ctx := cron.Stop()
		select {
		case <-ctx.Done():
		case <-time.After(time.Millisecond):
			t.Error("context was not done immediately")
		}
	})

	t.Run("a couple fast jobs and a slow job added, waits for slow job", func(t *testing.T) {
		t.Parallel()
		cron := newWithSeconds()
		_, _ = cron.AddFunc("* * * * * *", func() {})
		started := make(chan struct{}, 1)
		cron.Start()
		_, _ = cron.AddFunc("* * * * * *", func() {
			select {
			case started <- struct{}{}:
			default:
			}
			time.Sleep(500 * time.Millisecond)
		})
		_, _ = cron.AddFunc("* * * * * *", func() {})

		// Wait for the slow job to actually start running
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("slow job did not start")
		}

		ctx := cron.Stop()

		// Verify that it is not done for at least 150ms
		select {
		case <-ctx.Done():
			t.Error("context was done too quickly immediately")
		case <-time.After(150 * time.Millisecond):
			// expected, because the slow job is still running
		}

		// Verify that it IS done in the next 600ms (giving buffer)
		select {
		case <-ctx.Done():
			// expected
		case <-time.After(600 * time.Millisecond):
			t.Error("context not done after job should have completed")
		}
	})

	t.Run("repeated calls to stop, waiting for completion and after", func(t *testing.T) {
		t.Parallel()
		cron := newWithSeconds()
		_, _ = cron.AddFunc("* * * * * *", func() {})
		started := make(chan struct{}, 1)
		_, _ = cron.AddFunc("* * * * * *", func() {
			select {
			case started <- struct{}{}:
			default:
			}
			time.Sleep(750 * time.Millisecond)
		})
		cron.Start()
		_, _ = cron.AddFunc("* * * * * *", func() {})

		// Wait for the slow job to actually start running
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("slow job did not start")
		}

		// Now stop while the slow job is definitely running
		ctx := cron.Stop()
		ctx2 := cron.Stop()

		// Verify that it is not done for at least 200ms
		select {
		case <-ctx.Done():
			t.Error("context was done too quickly immediately")
		case <-ctx2.Done():
			t.Error("context2 was done too quickly immediately")
		case <-time.After(200 * time.Millisecond):
			// expected, because the slow job is still running
		}

		// Verify that it IS done in the next 800ms (giving buffer)
		select {
		case <-ctx.Done():
			// expected
		case <-time.After(800 * time.Millisecond):
			t.Error("context not done after job should have completed")
		}

		// Verify that ctx2 is also done.
		select {
		case <-ctx2.Done():
			// expected
		case <-time.After(time.Millisecond):
			t.Error("context2 not done even though context1 is")
		}

		// Verify that a new context retrieved from stop is immediately done.
		ctx3 := cron.Stop()
		select {
		case <-ctx3.Done():
			// expected
		case <-time.After(time.Millisecond):
			t.Error("context not done even when cron Stop is completed")
		}
	})
}

func TestMultiThreadedStartAndStop(t *testing.T) {
	t.Parallel()
	cron := New()
	go cron.Run()
	time.Sleep(2 * time.Millisecond)
	cron.Stop()
}

func wait(wg *sync.WaitGroup) chan bool {
	ch := make(chan bool)
	go func() {
		wg.Wait()
		ch <- true
	}()
	return ch
}

func stop(cron *Cron) chan bool {
	ch := make(chan bool)
	go func() {
		cron.Stop()
		ch <- true
	}()
	return ch
}

// newWithSeconds returns a Cron with the seconds field enabled.
func newWithSeconds() *Cron {
	return New(WithParser(secondParser), WithChain())
}
