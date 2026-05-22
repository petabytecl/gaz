package gaz

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz/cron"
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

type lifecyclePlanLazyPlain struct{}

type lifecyclePlanRuntimeHelper struct{}

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

	assert.Equal(t, [][]string{{"dependency"}, {"dependent"}}, plan.startupOrder)
	assert.Equal(t, [][]string{{"dependent"}, {"dependency"}}, plan.shutdownOrder)

	flattened := flattenLifecycleOrder(plan.startupOrder)
	assert.NotContains(t, flattened, "plain")
	assert.NotContains(t, flattened, "transient")
	assert.NotContains(t, flattened, "worker")
	assert.NotContains(t, flattened, "cron-job")

	runtimePlan := newRuntimeParticipantPlan(c)
	require.Len(t, runtimePlan.workerParticipants, 1)
	assert.Equal(t, "worker", runtimePlan.workerParticipants[0].Name())

	require.Len(t, runtimePlan.cronJobs, 1)
	assert.Equal(t, "cron-job", runtimePlan.cronJobs[0].serviceName)
	assert.Equal(t, "cron-job", runtimePlan.cronJobs[0].jobName)
	assert.True(t, runtimePlan.cronJobs[0].transient)
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

	runtimePlan := newRuntimeParticipantPlan(c)
	require.Len(t, runtimePlan.workerParticipants, 1)
	require.Len(t, runtimePlan.cronJobs, 1)
	assert.Equal(t, int32(1), workerResolutions.Load())
	assert.Equal(t, int32(1), cronResolutions.Load())
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
