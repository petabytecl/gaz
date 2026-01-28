package health

import (
	"testing"

	"github.com/petabytecl/gaz"
)

func TestModule(t *testing.T) {
	app := gaz.New()

	// Manually register config since module expects it
	err := gaz.For[Config](app.Container()).Instance(DefaultConfig())
	if err != nil {
		t.Fatalf("Register config failed: %v", err)
	}

	// Register module
	app.Module("health", Module)

	// Build
	if err := app.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify components are registered
	c := app.Container()

	if _, err := gaz.Resolve[*Manager](c); err != nil {
		t.Errorf("Manager not resolved: %v", err)
	}

	if _, err := gaz.Resolve[*ShutdownCheck](c); err != nil {
		t.Errorf("ShutdownCheck not resolved: %v", err)
	}

	if _, err := gaz.Resolve[*ManagementServer](c); err != nil {
		t.Errorf("ManagementServer not resolved: %v", err)
	}
}
