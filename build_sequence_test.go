package gaz

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func phaseIndex(phases []buildPhase, name string) int {
	for i, p := range phases {
		if p.name == name {
			return i
		}
	}
	return -1
}

func TestBuildPhaseOrder_LoadConfigBeforeProviderValues(t *testing.T) {
	app := New()
	phases := app.buildPhaseOrder()

	lc := phaseIndex(phases, "load-config")
	pv := phaseIndex(phases, "register-provider-values")

	require.NotEqual(t, -1, lc)
	require.NotEqual(t, -1, pv)
	assert.Less(t, lc, pv)
}

func TestBuildPhaseOrder_ProviderValuesBeforeCollectConfigs(t *testing.T) {
	app := New()
	phases := app.buildPhaseOrder()

	pv := phaseIndex(phases, "register-provider-values")
	cc := phaseIndex(phases, "collect-provider-configs")

	require.NotEqual(t, -1, pv)
	require.NotEqual(t, -1, cc)
	assert.Less(t, pv, cc)
}

func TestBuildPhaseOrder_LoggerBeforeSubsystems(t *testing.T) {
	app := New()
	phases := app.buildPhaseOrder()

	lg := phaseIndex(phases, "initialize-logger")
	ss := phaseIndex(phases, "initialize-subsystems")

	require.NotEqual(t, -1, lg)
	require.NotEqual(t, -1, ss)
	assert.Less(t, lg, ss)
}

func TestBuildPhaseOrder_ContainerBuildBeforeLifecyclePlan(t *testing.T) {
	app := New()
	phases := app.buildPhaseOrder()

	bc := phaseIndex(phases, "build-container")
	lp := phaseIndex(phases, "create-lifecycle-plan")

	require.NotEqual(t, -1, bc)
	require.NotEqual(t, -1, lp)
	assert.Less(t, bc, lp)
}

func TestBuildPhaseOrder_LifecyclePlanBeforeResolveBeforeParticipants(t *testing.T) {
	app := New()
	phases := app.buildPhaseOrder()

	lp := phaseIndex(phases, "create-lifecycle-plan")
	rs := phaseIndex(phases, "resolve-lifecycle-services")
	rp := phaseIndex(phases, "register-runtime-participants")

	require.NotEqual(t, -1, lp)
	require.NotEqual(t, -1, rs)
	require.NotEqual(t, -1, rp)
	assert.Less(t, lp, rs)
	assert.Less(t, rs, rp)
}

func TestBuildPhaseOrder_GatedPhasesAreAfterNonGated(t *testing.T) {
	app := New()
	phases := app.buildPhaseOrder()

	var gatedCount, nonGatedCount int
	lastNonGated := -1
	firstGated := len(phases)

	for i, p := range phases {
		if p.gated {
			gatedCount++
			if i < firstGated {
				firstGated = i
			}
		} else {
			nonGatedCount++
			if i > lastNonGated {
				lastNonGated = i
			}
		}
	}

	require.Greater(t, gatedCount, 0, "at least one gated phase must exist")
	require.Greater(t, nonGatedCount, 0, "at least one non-gated phase must exist")
	assert.Less(t, lastNonGated, firstGated,
		"all non-gated phases must precede gated phases")
}

func TestBuildIdempotent(t *testing.T) {
	app := New()

	require.NoError(t, app.Build())
	require.NoError(t, app.Build())
}

func TestRunBuildPhases_GatedPhasesSkippedOnError(t *testing.T) {
	app := New()
	app.buildErrors = []error{assert.AnError}

	err := app.runBuildPhases()
	require.Error(t, err)

	// Verify gated phases did not execute by checking their side effects:
	// lifecycle plan should be nil because createLifecyclePlan is gated.
	assert.Nil(t, app.cachedLifecyclePlan,
		"gated phase create-lifecycle-plan should not have run")
}
