package health

import (
	"errors"
	"strings"
	"testing"

	"github.com/petabytecl/gaz/di"
)

func TestModule(t *testing.T) {
	c := di.New()

	// Manually register config since module expects it
	err := di.For[Config](c).Instance(DefaultConfig())
	if err != nil {
		t.Fatalf("Register config failed: %v", err)
	}

	// Register module
	if err := Module(c); err != nil {
		t.Fatalf("Module failed: %v", err)
	}

	// Build
	if err := c.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify components are registered
	if _, err := di.Resolve[*Manager](c); err != nil {
		t.Errorf("Manager not resolved: %v", err)
	}

	if _, err := di.Resolve[*ShutdownCheck](c); err != nil {
		t.Errorf("ShutdownCheck not resolved: %v", err)
	}

	if _, err := di.Resolve[*ManagementServer](c); err != nil {
		t.Errorf("ManagementServer not resolved: %v", err)
	}
}

func TestModule_ShutdownCheckError(t *testing.T) {
	// Create container with ShutdownCheck already registered
	c := di.New()

	// Pre-register ShutdownCheck to cause duplicate
	if err := di.For[*ShutdownCheck](c).Instance(NewShutdownCheck()); err != nil {
		t.Fatalf("Pre-register ShutdownCheck failed: %v", err)
	}

	// Module should succeed (multi-binding is now supported)
	if err := Module(c); err != nil {
		t.Fatalf("Module failed: %v", err)
	}

	// But resolution should be ambiguous
	_, err := di.Resolve[*ShutdownCheck](c)
	if err == nil {
		t.Fatal("Expected error from Resolve, got nil")
	}

	if !errors.Is(err, di.ErrAmbiguous) {
		t.Errorf("Expected ErrAmbiguous, got: %v", err)
	}
}

func TestModule_ManagerError(t *testing.T) {
	// Create container with Manager already registered
	c := di.New()

	// Pre-register Manager to cause duplicate
	if err := di.For[*Manager](c).Instance(NewManager()); err != nil {
		t.Fatalf("Pre-register Manager failed: %v", err)
	}

	// Module should succeed
	if err := Module(c); err != nil {
		t.Fatalf("Module failed: %v", err)
	}

	// But resolution should be ambiguous
	_, err := di.Resolve[*Manager](c)
	if err == nil {
		t.Fatal("Expected error from Resolve, got nil")
	}

	if !errors.Is(err, di.ErrAmbiguous) {
		t.Errorf("Expected ErrAmbiguous, got: %v", err)
	}
}

func TestModule_ManagementServerError(t *testing.T) {
	// Create container with ManagementServer already registered
	c := di.New()

	// Register Config so Module can proceed past ShutdownCheck and Manager
	if err := di.For[Config](c).Instance(DefaultConfig()); err != nil {
		t.Fatalf("Register Config failed: %v", err)
	}

	// Pre-register ManagementServer to cause duplicate
	server := NewManagementServer(DefaultConfig(), NewManager(), NewShutdownCheck(), nil)
	if err := di.For[*ManagementServer](c).Instance(server); err != nil {
		t.Fatalf("Pre-register ManagementServer failed: %v", err)
	}

	// Module should succeed
	if err := Module(c); err != nil {
		t.Fatalf("Module failed: %v", err)
	}

	// But resolution should be ambiguous
	_, err := di.Resolve[*ManagementServer](c)
	if err == nil {
		t.Fatal("Expected error from Resolve, got nil")
	}

	if !errors.Is(err, di.ErrAmbiguous) {
		t.Errorf("Expected ErrAmbiguous, got: %v", err)
	}
}

func TestModule_ConfigNotRegistered(t *testing.T) {
	// Create container without Config registered
	c := di.New()

	// Register module without registering Config first
	if err := Module(c); err != nil {
		t.Fatalf("Module failed: %v", err)
	}

	// Build should fail when ManagementServer provider tries to resolve Config
	err := c.Build()
	if err == nil {
		t.Fatal("Expected error from Build, got nil")
	}

	// The error should indicate Config could not be resolved
	// This is an indirect test - the ManagementServer provider fails to resolve Config
	if !strings.Contains(err.Error(), "Config") {
		t.Errorf("Expected error to mention Config, got: %v", err)
	}
}
