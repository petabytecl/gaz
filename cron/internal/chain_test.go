package internal

import (
	"bytes"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type syncBuffer struct {
	b bytes.Buffer
	m sync.Mutex
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Write(p) //nolint:wrapcheck
}

func (b *syncBuffer) String() string {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.String()
}

func newDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func appendingJob(slice *[]int, value int) Job { //nolint:ireturn,nolintlint // test helper
	var m sync.Mutex
	return FuncJob(func() {
		m.Lock()
		*slice = append(*slice, value)
		m.Unlock()
	})
}

func appendingWrapper(slice *[]int, value int) JobWrapper {
	return func(j Job) Job {
		return FuncJob(func() {
			appendingJob(slice, value).Run()
			j.Run()
		})
	}
}

func TestDelayIfStillRunningLogs(t *testing.T) {
	t.Parallel()
	var buf syncBuffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	var j countJob
	// First job blocks for 50ms
	j.delay = 50 * time.Millisecond

	// Threshold 10ms
	wrappedJob := NewChain(delayIfStillRunning(logger, 10*time.Millisecond)).Then(&j)

	// Start first run
	go wrappedJob.Run()

	// Wait until first job has started before launching the second
	require.Eventually(t, func() bool {
		return j.Started() >= 1
	}, time.Second, time.Millisecond)

	// Start second run. It should wait for the first to finish.
	// The wait exceeds the 10ms threshold, so it should log.
	done := make(chan struct{})
	go func() {
		wrappedJob.Run()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("second job did not complete within timeout")
	}

	if !strings.Contains(buf.String(), "delay") {
		t.Errorf("expected log message about delay, got: %s", buf.String())
	}
}

func TestChain(t *testing.T) {
	t.Parallel()
	var nums []int
	var (
		append1 = appendingWrapper(&nums, 1)
		append2 = appendingWrapper(&nums, 2)
		append3 = appendingWrapper(&nums, 3)
		append4 = appendingJob(&nums, 4)
	)
	NewChain(append1, append2, append3).Then(append4).Run()
	if !reflect.DeepEqual(nums, []int{1, 2, 3, 4}) {
		t.Error("unexpected order of calls:", nums)
	}
}

func TestChainRecover(t *testing.T) {
	t.Parallel()
	panickingJob := FuncJob(func() {
		panic("panickingJob panics")
	})

	t.Run("panic exits job by default", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if err := recover(); err == nil {
				t.Errorf("panic expected, but none received")
			}
		}()
		NewChain().Then(panickingJob).
			Run()
	})

	t.Run("Recovering JobWrapper recovers", func(t *testing.T) {
		t.Parallel()
		NewChain(Recover(newDiscardLogger())).
			Then(panickingJob).
			Run()
	})

	t.Run("composed with the *IfStillRunning wrappers", func(t *testing.T) {
		t.Parallel()
		NewChain(Recover(newDiscardLogger())).
			Then(panickingJob).
			Run()
	})
}

type countJob struct {
	m       sync.Mutex
	started int
	done    int
	delay   time.Duration
}

func (j *countJob) Run() {
	j.m.Lock()
	j.started++
	j.m.Unlock()
	time.Sleep(j.delay) //nolint:timesleep // simulates real job execution time, not test synchronization
	j.m.Lock()
	j.done++
	j.m.Unlock()
}

func (j *countJob) Started() int {
	defer j.m.Unlock()
	j.m.Lock()
	return j.started
}

func (j *countJob) Done() int {
	defer j.m.Unlock()
	j.m.Lock()
	return j.done
}

func TestChainDelayIfStillRunning(t *testing.T) {
	t.Parallel()
	t.Run("runs immediately", func(t *testing.T) {
		t.Parallel()
		var j countJob
		wrappedJob := NewChain(DelayIfStillRunning(newDiscardLogger())).Then(&j)
		go wrappedJob.Run()
		require.Eventually(t, func() bool {
			return j.Done() == 1
		}, time.Second, time.Millisecond)
	})

	t.Run("second run immediate if first done", func(t *testing.T) {
		t.Parallel()
		var j countJob
		wrappedJob := NewChain(DelayIfStillRunning(newDiscardLogger())).Then(&j)
		go wrappedJob.Run()
		// Wait for first to complete before launching second
		require.Eventually(t, func() bool {
			return j.Done() >= 1
		}, time.Second, time.Millisecond)
		go wrappedJob.Run()
		require.Eventually(t, func() bool {
			return j.Done() == 2
		}, time.Second, time.Millisecond)
	})

	t.Run("second run delayed if first not done", func(t *testing.T) {
		t.Parallel()
		var j countJob
		j.delay = 10 * time.Millisecond
		wrappedJob := NewChain(DelayIfStillRunning(newDiscardLogger())).Then(&j)
		go wrappedJob.Run()
		// Wait for first to start
		require.Eventually(t, func() bool {
			return j.Started() >= 1
		}, time.Second, time.Millisecond)
		go wrappedJob.Run()

		// First job should be started but not yet done
		require.Equal(t, 0, j.Done(), "first job should not be done yet")

		// Verify that both jobs eventually complete
		require.Eventually(t, func() bool {
			return j.Started() == 2 && j.Done() == 2
		}, time.Second, time.Millisecond)
	})
}

func TestChainSkipIfStillRunning(t *testing.T) {
	t.Parallel()
	t.Run("runs immediately", func(t *testing.T) {
		t.Parallel()
		var j countJob
		wrappedJob := NewChain(SkipIfStillRunning(newDiscardLogger())).Then(&j)
		go wrappedJob.Run()
		require.Eventually(t, func() bool {
			return j.Done() == 1
		}, time.Second, time.Millisecond)
	})

	t.Run("second run immediate if first done", func(t *testing.T) {
		t.Parallel()
		var j countJob
		wrappedJob := NewChain(SkipIfStillRunning(newDiscardLogger())).Then(&j)
		go wrappedJob.Run()
		// Wait for first to complete before launching second
		require.Eventually(t, func() bool {
			return j.Done() >= 1
		}, time.Second, time.Millisecond)
		go wrappedJob.Run()
		require.Eventually(t, func() bool {
			return j.Done() == 2
		}, time.Second, time.Millisecond)
	})

	t.Run("second run skipped if first not done", func(t *testing.T) {
		t.Parallel()
		var j countJob
		j.delay = 10 * time.Millisecond
		wrappedJob := NewChain(SkipIfStillRunning(newDiscardLogger())).Then(&j)
		go wrappedJob.Run()
		// Wait for first to start
		require.Eventually(t, func() bool {
			return j.Started() >= 1
		}, time.Second, time.Millisecond)
		// Launch second (should be skipped since first is still running)
		go wrappedJob.Run()

		// Verify that first job started but not done yet
		require.Equal(t, 0, j.Done(), "first job should not be done yet")

		// Verify that the first job completes and second was skipped
		require.Eventually(t, func() bool {
			return j.Done() == 1
		}, time.Second, time.Millisecond)
		// Second job should never have started
		require.Equal(t, 1, j.Started(), "second job should have been skipped")
	})

	t.Run("skip 10 jobs on rapid fire", func(t *testing.T) {
		t.Parallel()
		var j countJob
		j.delay = 10 * time.Millisecond
		wrappedJob := NewChain(SkipIfStillRunning(newDiscardLogger())).Then(&j)
		for range 11 {
			go wrappedJob.Run()
		}
		require.Eventually(t, func() bool {
			return j.Done() == 1
		}, time.Second, time.Millisecond)
	})

	t.Run("different jobs independent", func(t *testing.T) {
		t.Parallel()
		var j1, j2 countJob
		j1.delay = 10 * time.Millisecond
		j2.delay = 10 * time.Millisecond
		chain := NewChain(SkipIfStillRunning(newDiscardLogger()))
		wrappedJob1 := chain.Then(&j1)
		wrappedJob2 := chain.Then(&j2)
		for range 11 {
			go wrappedJob1.Run()
			go wrappedJob2.Run()
		}
		require.Eventually(t, func() bool {
			return j1.Done() == 1 && j2.Done() == 1
		}, time.Second, time.Millisecond)
	})
}
