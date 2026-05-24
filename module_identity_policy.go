package gaz

import "fmt"

// registerModuleIdentity records a module identity for duplicate detection.
// Module identity is the shared policy surface for Use, UseDI, Module, and
// child module application.
func (a *App) registerModuleIdentity(name string) error {
	if a.modules[name] {
		return fmt.Errorf("%w: %s", ErrModuleDuplicate, name)
	}
	a.modules[name] = true
	return nil
}
