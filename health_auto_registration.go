package gaz

import (
	"fmt"

	"github.com/petabytecl/gaz/health"
)

// autoRegisterHealth detects whether the config target implements
// health.HealthConfigProvider and, when the health module has not already
// been applied explicitly, registers it automatically.
//
// Keeping this policy in its own file (instead of inline in Build) gives
// the health auto-registration rule locality: config-provider detection,
// module identity, and duplicate-avoidance all live together and can be
// tested independently of the full build pipeline.
func (a *App) autoRegisterHealth() error {
	if a.configTarget == nil {
		return nil
	}

	hp, ok := a.configTarget.(health.HealthConfigProvider)
	if !ok {
		return nil
	}

	// Explicit health module already applied — skip auto-registration.
	if a.modules[health.ModuleName] {
		return nil
	}

	cfg := hp.HealthConfig()

	if err := For[health.Config](a.container).Instance(cfg); err != nil {
		return fmt.Errorf("auto-register health config: %w", err)
	}

	mod := NewModule(health.ModuleName).
		Provide(health.Module).
		Build()
	if err := a.registerModuleIdentity(health.ModuleName); err != nil {
		return err
	}
	if err := mod.Apply(a); err != nil {
		return fmt.Errorf("auto-register health module: %w", err)
	}

	return nil
}
