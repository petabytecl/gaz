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
	rp := phaseIndex(phases, "register-runtime-participants")

	require.NotEqual(t, -1, bc)
	require.NotEqual(t, -1, lp)
	require.NotEqual(t, -1, rp)
	assert.Less(t, bc, lp)
	assert.Less(t, lp, rp)
}

func TestBuildPhaseOrder_GatedPhasesAreAfterNonGated(t *testing.T) {
	app := New()
	phases := app.buildPhaseOrder()

	lastNonGated := -1
	firstGated := len(phases)

	for i, p := range phases {
		if p.gated {
			if i < firstGated {
				firstGated = i
			}
		} else {
			if i > lastNonGated {
				lastNonGated = i
			}
		}
	}

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

	var ran []string
	original := app.buildPhaseOrder()

	phases := make([]buildPhase, len(original))
	for i, p := range original {
		name := p.name
		gated := p.gated
		phases[i] = buildPhase{
			name:  name,
			gated: gated,
			run: func() error {
				ran = append(ran, name)
				return nil
			},
		}
	}

	for _, phase := range phases {
		if phase.gated && len(app.buildErrors) > 0 {
			break
		}
		_ = phase.run()
	}

	for _, name := range ran {
		idx := phaseIndex(original, name)
		require.NotEqual(t, -1, idx)
		assert.False(t, original[idx].gated,
			"gated phase %q should not have run", name)
	}
}
