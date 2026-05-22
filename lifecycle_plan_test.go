package gaz

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz/cron"
	"github.com/petabytecl/gaz/worker"
)

type lifecyclePlanDependency struct{}

func (s *lifecyclePlanDependency) OnStart(context.Context) error { return nil }
func (s *lifecyclePlanDependency) OnStop(context.Context) error  { return nil }

type lifecyclePlanDependent struct {
	dependency *lifecyclePlanDependency
}

func (s *lifecyclePlanDependent) OnStart(context.Context) error { return nil }
func (s *lifecyclePlanDependent) OnStop(context.Context) error  { return nil }

type lifecyclePlanTransient struct{}

func (s *lifecyclePlanTransient) OnStart(context.Context) error { return nil }
func (s *lifecyclePlanTransient) OnStop(context.Context) error  { return nil }

type lifecyclePlanPlain struct{}

type lifecyclePlanWorker struct {
	name string
}

func (w *lifecyclePlanWorker) Name() string                  { return w.name }
func (w *lifecyclePlanWorker) OnStart(context.Context) error { return nil }
func (w *lifecyclePlanWorker) OnStop(context.Context) error  { return nil }

type lifecyclePlanCronJob struct{}

func (j *lifecyclePlanCronJob) Name() string              { return "cron-job" }
func (j *lifecyclePlanCronJob) Schedule() string          { return "@every 1m" }
func (j *lifecyclePlanCronJob) Timeout() time.Duration    { return time.Second }
func (j *lifecyclePlanCronJob) Run(context.Context) error { return nil }

type lifecyclePlanInvalidCronJob struct{}

func (j *lifecyclePlanInvalidCronJob) Name() string              { return "invalid-cron-job" }
func (j *lifecyclePlanInvalidCronJob) Schedule() string          { return "not-a-schedule" }
func (j *lifecyclePlanInvalidCronJob) Timeout() time.Duration    { return time.Second }
func (j *lifecyclePlanInvalidCronJob) Run(context.Context) error { return nil }

type lifecyclePlanLazyPlain struct{}

type lifecyclePlanRuntimeHelper struct{}

type lifecyclePlanCountedService struct {
	onStart func()
}

func (s *lifecyclePlanCountedService) OnStart(context.Context) error {
	if s.onStart != nil {
		s.onStart()
	}
	return nil
}

func (s *lifecyclePlanCountedService) OnStop(context.Context) error { return nil }

type lifecyclePlanCountedDependent struct {
	dependency *lifecyclePlanCountedService
	onStart    func()
}

func (s *lifecyclePlanCountedDependent) OnStart(context.Context) error {
	if s.onStart != nil {
		s.onStart()
	}
	return nil
}

func (s *lifecyclePlanCountedDependent) OnStop(context.Context) error { return nil }

func TestLifecyclePlanOwnsSelectionAndOrderPolicy(t *testing.T) {
	c := NewContainer()

	require.NoError(t, For[*lifecyclePlanDependency](c).Named("dependency").
		ProviderFunc(func(*Container) *lifecyclePlanDependency {
			return &lifecyclePlanDependency{}
		}))

	require.NoError(t, For[*lifecyclePlanDependent](c).Named("dependent").
		Provider(func(c *Container) (*lifecyclePlanDependent, error) {
			dep, err := Resolve[*lifecyclePlanDependency](c, Named("dependency"))
			if err != nil {
				return nil, err
			}
			return &lifecyclePlanDependent{dependency: dep}, nil
		}))

	require.NoError(t, For[*lifecyclePlanTransient](c).Named("transient").Transient().
		ProviderFunc(func(*Container) *lifecyclePlanTransient {
			return &lifecyclePlanTransient{}
		}))

	require.NoError(t, For[*lifecyclePlanPlain](c).Named("plain").
		ProviderFunc(func(*Container) *lifecyclePlanPlain {
			return &lifecyclePlanPlain{}
		}))

	require.NoError(t, For[*lifecyclePlanWorker](c).Named("worker").
		Instance(&lifecyclePlanWorker{name: "worker"}))

	require.NoError(t, For[cron.CronJob](c).Named("cron-job").Transient().
		ProviderFunc(func(*Container) cron.CronJob {
			return &lifecyclePlanCronJob{}
		}))

	require.NoError(t, c.Build())

	plan, err := newLifecyclePlan(c)
	require.NoError(t, err)

	assert.Contains(t, plan.services, "dependency")
	assert.Contains(t, plan.services, "dependent")
	assert.Contains(t, plan.services, "plain")
	assert.NotContains(t, plan.services, "transient")
	assert.NotContains(t, plan.services, "worker")
	assert.NotContains(t, plan.services, "cron-job")

	require.NoError(t, plan.resolveLifecycleServices(c))

	assert.Equal(t, [][]string{{"dependency"}, {"dependent"}}, plan.startupOrder)
	assert.Equal(t, [][]string{{"dependent"}, {"dependency"}}, plan.shutdownOrder)

	flattened := flattenLifecycleOrder(plan.startupOrder)
	assert.NotContains(t, flattened, "plain")
	assert.NotContains(t, flattened, "transient")
	assert.NotContains(t, flattened, "worker")
	assert.NotContains(t, flattened, "cron-job")

	require.Len(t, plan.workerParticipants, 1)
	assert.Equal(t, "worker", plan.workerParticipants[0].serviceName)

	require.Len(t, plan.cronJobs, 1)
	assert.Equal(t, "cron-job", plan.cronJobs[0].serviceName)
	assert.True(t, plan.cronJobs[0].transient)
}

func TestLifecyclePlanDoesNotResolvePlainServicesDuringPlanning(t *testing.T) {
	c := NewContainer()
	var resolutions atomic.Int32

	require.NoError(t, For[*lifecyclePlanLazyPlain](c).Named("lazy-plain").
		ProviderFunc(func(*Container) *lifecyclePlanLazyPlain {
			resolutions.Add(1)
			return &lifecyclePlanLazyPlain{}
		}))

	require.NoError(t, c.Build())

	plan, err := newLifecyclePlan(c)
	require.NoError(t, err)

	assert.Zero(t, resolutions.Load())
	assert.Contains(t, plan.services, "lazy-plain")
	assert.Empty(t, plan.startupOrder)
}

func TestLifecyclePlanDoesNotResolveRuntimeParticipantsDuringPlanning(t *testing.T) {
	c := NewContainer()
	var workerResolutions atomic.Int32
	var cronResolutions atomic.Int32

	require.NoError(t, For[*lifecyclePlanRuntimeHelper](c).Named("runtime-helper").
		ProviderFunc(func(*Container) *lifecyclePlanRuntimeHelper {
			return &lifecyclePlanRuntimeHelper{}
		}))

	require.NoError(t, For[*lifecyclePlanWorker](c).Named("worker").
		Provider(func(c *Container) (*lifecyclePlanWorker, error) {
			workerResolutions.Add(1)
			_, err := Resolve[*lifecyclePlanRuntimeHelper](c, Named("runtime-helper"))
			if err != nil {
				return nil, err
			}
			return &lifecyclePlanWorker{name: "worker"}, nil
		}))

	require.NoError(t, For[cron.CronJob](c).Named("cron-job").Transient().
		ProviderFunc(func(*Container) cron.CronJob {
			cronResolutions.Add(1)
			return &lifecyclePlanCronJob{}
		}))

	require.NoError(t, c.Build())

	plan, err := newLifecyclePlan(c)
	require.NoError(t, err)

	assert.Zero(t, workerResolutions.Load())
	assert.Zero(t, cronResolutions.Load())
	assert.NotContains(t, plan.services, "worker")
	assert.NotContains(t, plan.services, "cron-job")
	require.Len(t, plan.workerParticipants, 1)
	require.Len(t, plan.cronJobs, 1)
	assert.Equal(t, "worker", plan.workerParticipants[0].serviceName)
	assert.Equal(t, "cron-job", plan.cronJobs[0].serviceName)
	assert.Zero(t, workerResolutions.Load())
	assert.Zero(t, cronResolutions.Load())
}

func TestAppBuildCachesFullLifecyclePlan(t *testing.T) {
	app := New()

	require.NoError(t, For[*lifecyclePlanDependency](app.Container()).Named("dependency").
		ProviderFunc(func(*Container) *lifecyclePlanDependency {
			return &lifecyclePlanDependency{}
		}))

	require.NoError(t, For[*lifecyclePlanWorker](app.Container()).Named("worker").
		Instance(&lifecyclePlanWorker{name: "worker"}))

	require.NoError(t, For[cron.CronJob](app.Container()).Named("cron-job").Transient().
		ProviderFunc(func(*Container) cron.CronJob {
			return &lifecyclePlanCronJob{}
		}))

	require.NoError(t, app.Build())

	plan := app.cachedLifecyclePlan
	require.NotNil(t, plan)
	assert.Contains(t, plan.services, "dependency")
	assert.Equal(t, [][]string{{"dependency"}}, plan.startupOrder)
	assert.Equal(t, [][]string{{"dependency"}}, plan.shutdownOrder)

	require.Len(t, plan.workerParticipants, 1)
	assert.Equal(t, "worker", plan.workerParticipants[0].serviceName)

	require.Len(t, plan.cronJobs, 1)
	assert.Equal(t, "cron-job", plan.cronJobs[0].serviceName)
}

func TestAppBuildDoesNotResolveLifecycleServices(t *testing.T) {
	app := New()
	var dependencyResolutions atomic.Int32
	var dependentResolutions atomic.Int32
	var startMu sync.Mutex
	var starts []string

	recordStart := func(name string) func() {
		return func() {
			startMu.Lock()
			defer startMu.Unlock()
			starts = append(starts, name)
		}
	}

	require.NoError(t, For[*lifecyclePlanCountedService](app.Container()).Named("z-dependency").
		ProviderFunc(func(*Container) *lifecyclePlanCountedService {
			dependencyResolutions.Add(1)
			return &lifecyclePlanCountedService{
				onStart: recordStart("z-dependency"),
			}
		}))

	require.NoError(t, For[*lifecyclePlanCountedDependent](app.Container()).Named("a-dependent").
		Provider(func(c *Container) (*lifecyclePlanCountedDependent, error) {
			dependentResolutions.Add(1)
			dep, err := Resolve[*lifecyclePlanCountedService](c, Named("z-dependency"))
			if err != nil {
				return nil, err
			}
			return &lifecyclePlanCountedDependent{
				dependency: dep,
				onStart:    recordStart("a-dependent"),
			}, nil
		}))

	require.NoError(t, app.Build())
	assert.Zero(t, dependencyResolutions.Load())
	assert.Zero(t, dependentResolutions.Load())

	plan := app.cachedLifecyclePlan
	require.NotNil(t, plan)
	assert.Contains(t, plan.services, "z-dependency")
	assert.Contains(t, plan.services, "a-dependent")

	require.NoError(t, app.Start(context.Background()))
	assert.Equal(t, int32(1), dependencyResolutions.Load())
	assert.Equal(t, int32(1), dependentResolutions.Load())

	startMu.Lock()
	started := append([]string(nil), starts...)
	startMu.Unlock()
	assert.Equal(t, []string{"z-dependency", "a-dependent"}, started)

	require.NoError(t, app.Stop(context.Background()))
}

func TestAppBuildFailsWhenWorkerParticipantCannotResolve(t *testing.T) {
	app := New()
	workerErr := errors.New("worker provider failed")

	require.NoError(t, For[*lifecyclePlanWorker](app.Container()).Named("broken-worker").
		Provider(func(*Container) (*lifecyclePlanWorker, error) {
			return nil, workerErr
		}))

	err := app.Build()
	require.Error(t, err)
	assert.ErrorIs(t, err, workerErr)
	assert.Contains(t, err.Error(), "registering worker participant broken-worker: resolve")
}

func TestAppBuildFailsWhenWorkerParticipantResolvesWrongType(t *testing.T) {
	app := New()

	require.NoError(t, For[worker.Worker](app.Container()).Named("nil-worker").
		ProviderFunc(func(*Container) worker.Worker {
			return nil
		}))

	err := app.Build()
	require.Error(t, err)
	assert.Contains(
		t,
		err.Error(),
		"registering worker participant nil-worker: resolved <nil> does not implement worker.Worker",
	)
}

func TestAppBuildFailsWhenCronParticipantCannotResolve(t *testing.T) {
	app := New()
	cronErr := errors.New("cron provider failed")

	require.NoError(t, For[cron.CronJob](app.Container()).Named("broken-cron").Transient().
		Provider(func(*Container) (cron.CronJob, error) {
			return nil, cronErr
		}))

	err := app.Build()
	require.Error(t, err)
	assert.ErrorIs(t, err, cronErr)
	assert.Contains(t, err.Error(), "registering cron job participant broken-cron: resolve")
}

func TestAppBuildFailsWhenCronParticipantResolvesWrongType(t *testing.T) {
	app := New()

	require.NoError(t, For[cron.CronJob](app.Container()).Named("nil-cron").Transient().
		ProviderFunc(func(*Container) cron.CronJob {
			return nil
		}))

	err := app.Build()
	require.Error(t, err)
	assert.Contains(
		t,
		err.Error(),
		"registering cron job participant nil-cron: resolved <nil> does not implement cron.CronJob",
	)
}

func TestAppBuildFailsWhenCronScheduleIsInvalid(t *testing.T) {
	app := New()

	require.NoError(t, For[cron.CronJob](app.Container()).Named("invalid-cron").Transient().
		ProviderFunc(func(*Container) cron.CronJob {
			return &lifecyclePlanInvalidCronJob{}
		}))

	err := app.Build()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "registering cron job participant invalid-cron (job invalid-cron-job)")
	assert.Contains(t, err.Error(), "invalid schedule")
}

func TestRegisterWorkersErrorIncludesServiceAndWorkerNames(t *testing.T) {
	c := NewContainer()
	require.NoError(t, For[*lifecyclePlanWorker](c).Named("worker-binding").
		Instance(&lifecyclePlanWorker{name: "runtime-worker"}))

	mgr := worker.NewManager(slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, mgr.Start(context.Background()))
	t.Cleanup(func() {
		require.NoError(t, mgr.Stop(context.Background()))
	})

	plan := &lifecyclePlan{
		workerParticipants: []lifecycleWorkerParticipant{{
			serviceName: "worker-binding",
		}},
	}

	errs := plan.registerWorkers(c, mgr)
	require.Len(t, errs, 1)
	assert.ErrorIs(t, errs[0], worker.ErrManagerAlreadyRunning)
	assert.Contains(
		t,
		errs[0].Error(),
		"registering worker participant worker-binding (worker runtime-worker)",
	)
}

func flattenLifecycleOrder(order [][]string) []string {
	size := 0
	for _, layer := range order {
		size += len(layer)
	}

	result := make([]string, 0, size)
	for _, layer := range order {
		result = append(result, layer...)
	}
	return result
}
