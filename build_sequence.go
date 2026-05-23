package gaz

import (
	"errors"
	"fmt"
)

// buildPhase names a unit of work in the build pipeline.
type buildPhase struct {
	name  string
	run   func() error
	gated bool // only runs if no prior phase produced an error
}

// buildPhaseOrder returns the named build phases in their required execution
// order. The ordering contracts documented here are verified by tests.
//
// Ordering contracts:
//   - register-provider-values before collect-provider-configs
//   - initialize-logger before initialize-subsystems
//   - build-container before create-lifecycle-plan
//   - create-lifecycle-plan before resolve-lifecycle-services
//   - resolve-lifecycle-services before register-runtime-participants
func (a *App) buildPhaseOrder() []buildPhase {
	return []buildPhase{
		{name: "register-provider-values", run: a.registerProviderValuesEarly},
		{name: "initialize-logger", run: a.initializeLogger},
		{name: "initialize-subsystems", run: a.initializeSubsystems},
		{name: "collect-provider-configs", run: a.collectProviderConfigs},
		{name: "auto-register-health", run: a.autoRegisterHealth},
		{name: "build-container", run: a.buildContainer},
		{name: "create-lifecycle-plan", run: a.createLifecyclePlan, gated: true},
		{name: "resolve-lifecycle-services", run: a.resolveLifecycleServicesStep, gated: true},
		{name: "register-runtime-participants", run: a.registerRuntimeParticipantsStep, gated: true},
	}
}

// runBuildPhases executes the build pipeline in order.
// Non-gated phases always run and accumulate errors so users see all
// problems at once. Gated phases only run if no prior error occurred.
func (a *App) runBuildPhases() error {
	var errs []error
	errs = append(errs, a.buildErrors...)

	for _, phase := range a.buildPhaseOrder() {
		if phase.gated && len(errs) > 0 {
			break
		}
		if err := phase.run(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (a *App) buildContainer() error {
	if err := a.container.Build(); err != nil {
		return fmt.Errorf("building container: %w", err)
	}
	return nil
}

func (a *App) createLifecyclePlan() error {
	plan, err := newLifecyclePlan(a.container)
	if err != nil {
		return err
	}
	a.cachedLifecyclePlan = plan
	return nil
}

func (a *App) resolveLifecycleServicesStep() error {
	return a.cachedLifecyclePlan.resolveLifecycleServices(a.container)
}

func (a *App) registerRuntimeParticipantsStep() error {
	errs := a.cachedLifecyclePlan.registerRuntimeParticipants(
		a.container,
		a.workerMgr,
		a.eventBus,
		a.scheduler,
		a.getLogger(),
	)
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("registering runtime participants: %w", errors.Join(errs...))
}
