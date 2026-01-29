package health

import (
	"strings"
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

func TestModule_ShutdownCheckError(t *testing.T) {
	// Create container with ShutdownCheck already registered
	c := gaz.NewContainer()

	// Pre-register ShutdownCheck to cause duplicate error
	if err := gaz.For[*ShutdownCheck](c).Instance(NewShutdownCheck()); err != nil {
		t.Fatalf("Pre-register ShutdownCheck failed: %v", err)
	}

	// Module should fail when trying to register ShutdownCheck
	err := Module(c)
	if err == nil {
		t.Fatal("Expected error from Module, got nil")
	}

	if !strings.Contains(err.Error(), "register shutdown check") {
		t.Errorf("Expected error to contain 'register shutdown check', got: %v", err)
	}
}

func TestModule_ManagerError(t *testing.T) {
	// Create container with Manager already registered
	c := gaz.NewContainer()

	// Pre-register Manager to cause duplicate error
	if err := gaz.For[*Manager](c).Instance(NewManager()); err != nil {
		t.Fatalf("Pre-register Manager failed: %v", err)
	}

	// Module should fail when trying to register Manager
	// First ShutdownCheck succeeds, then Manager fails
	err := Module(c)
	if err == nil {
		t.Fatal("Expected error from Module, got nil")
	}

	if !strings.Contains(err.Error(), "register manager") {
		t.Errorf("Expected error to contain 'register manager', got: %v", err)
	}
}

func TestModule_ManagementServerError(t *testing.T) {
	// Create container with ManagementServer already registered
	c := gaz.NewContainer()

	// Register Config so Module can proceed past ShutdownCheck and Manager
	if err := gaz.For[Config](c).Instance(DefaultConfig()); err != nil {
		t.Fatalf("Register Config failed: %v", err)
	}

	// Pre-register ManagementServer to cause duplicate error
	server := NewManagementServer(DefaultConfig(), NewManager(), NewShutdownCheck())
	if err := gaz.For[*ManagementServer](c).Instance(server); err != nil {
		t.Fatalf("Pre-register ManagementServer failed: %v", err)
	}

	// Module should fail when trying to register ManagementServer
	err := Module(c)
	if err == nil {
		t.Fatal("Expected error from Module, got nil")
	}

	if !strings.Contains(err.Error(), "register management server") {
		t.Errorf("Expected error to contain 'register management server', got: %v", err)
	}
}

func TestModule_ConfigNotRegistered(t *testing.T) {
	// Create container without Config registered
	app := gaz.New()

	// Register module without registering Config first
	app.Module("health", Module)

	// Build should fail when ManagementServer provider tries to resolve Config
	err := app.Build()
	if err == nil {
		t.Fatal("Expected error from Build, got nil")
	}

	// The error should indicate Config could not be resolved
	// This is an indirect test - the ManagementServer provider fails to resolve Config
	if !strings.Contains(err.Error(), "Config") {
		t.Errorf("Expected error to mention Config, got: %v", err)
	}
}
