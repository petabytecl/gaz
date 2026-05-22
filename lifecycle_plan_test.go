package gaz

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func (w *lifecyclePlanWorker) Name() string { return w.name }
func (w *lifecyclePlanWorker) OnStart(context.Context) error {
	return nil
}
func (w *lifecyclePlanWorker) OnStop(context.Context) error {
	return nil
}

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

	require.NoError(t, c.Build())

	plan, err := newLifecyclePlan(c)
	require.NoError(t, err)

	assert.Contains(t, plan.services, "dependency")
	assert.Contains(t, plan.services, "dependent")
	assert.Contains(t, plan.services, "plain")
	assert.NotContains(t, plan.services, "transient")
	assert.NotContains(t, plan.services, "worker")

	assert.Equal(t, [][]string{{"dependency"}, {"dependent"}}, plan.startupOrder)
	assert.Equal(t, [][]string{{"dependent"}, {"dependency"}}, plan.shutdownOrder)

	flattened := flattenLifecycleOrder(plan.startupOrder)
	assert.NotContains(t, flattened, "plain")
	assert.NotContains(t, flattened, "transient")
	assert.NotContains(t, flattened, "worker")
}

func flattenLifecycleOrder(order [][]string) []string {
	result := make([]string, 0)
	for _, layer := range order {
		result = append(result, layer...)
	}
	return result
}
