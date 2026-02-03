package health

import (
	"errors"
	"strings"
	"testing"
	"time"

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

//nolint:gocyclo,cyclop // Test function with many subtests
func TestNewModule(t *testing.T) {
	t.Run("zero arguments uses defaults", func(t *testing.T) {
		c := di.New()
		m := NewModule()

		// Verify module name
		if m.Name() != "health" {
			t.Errorf("Expected module name 'health', got %q", m.Name())
		}

		// Register module
		if err := m.Register(c); err != nil {
			t.Fatalf("Register failed: %v", err)
		}

		// Build
		if err := c.Build(); err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		// Verify Config was registered with defaults
		cfg, err := di.Resolve[Config](c)
		if err != nil {
			t.Fatalf("Resolve Config failed: %v", err)
		}
		if cfg.Port != 9090 {
			t.Errorf("Expected port 9090, got %d", cfg.Port)
		}
		if cfg.LivenessPath != "/live" {
			t.Errorf("Expected LivenessPath '/live', got %q", cfg.LivenessPath)
		}
		if cfg.ReadinessPath != "/ready" {
			t.Errorf("Expected ReadinessPath '/ready', got %q", cfg.ReadinessPath)
		}
		if cfg.StartupPath != "/startup" {
			t.Errorf("Expected StartupPath '/startup', got %q", cfg.StartupPath)
		}
	})

	t.Run("WithPort overrides port", func(t *testing.T) {
		c := di.New()
		m := NewModule(WithPort(8081))

		if err := m.Register(c); err != nil {
			t.Fatalf("Register failed: %v", err)
		}

		if err := c.Build(); err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		cfg, err := di.Resolve[Config](c)
		if err != nil {
			t.Fatalf("Resolve Config failed: %v", err)
		}
		if cfg.Port != 8081 {
			t.Errorf("Expected port 8081, got %d", cfg.Port)
		}
	})

	t.Run("multiple options combine", func(t *testing.T) {
		c := di.New()
		m := NewModule(
			WithPort(8082),
			WithLivenessPath("/health/live"),
			WithReadinessPath("/health/ready"),
			WithStartupPath("/health/startup"),
		)

		if err := m.Register(c); err != nil {
			t.Fatalf("Register failed: %v", err)
		}

		if err := c.Build(); err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		cfg, err := di.Resolve[Config](c)
		if err != nil {
			t.Fatalf("Resolve Config failed: %v", err)
		}
		if cfg.Port != 8082 {
			t.Errorf("Expected port 8082, got %d", cfg.Port)
		}
		if cfg.LivenessPath != "/health/live" {
			t.Errorf("Expected LivenessPath '/health/live', got %q", cfg.LivenessPath)
		}
		if cfg.ReadinessPath != "/health/ready" {
			t.Errorf("Expected ReadinessPath '/health/ready', got %q", cfg.ReadinessPath)
		}
		if cfg.StartupPath != "/health/startup" {
			t.Errorf("Expected StartupPath '/health/startup', got %q", cfg.StartupPath)
		}
	})

	t.Run("registers all health components", func(t *testing.T) {
		c := di.New()
		m := NewModule()

		if err := m.Register(c); err != nil {
			t.Fatalf("Register failed: %v", err)
		}

		if err := c.Build(); err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		// Verify all components registered
		if _, err := di.Resolve[*ShutdownCheck](c); err != nil {
			t.Errorf("ShutdownCheck not resolved: %v", err)
		}

		if _, err := di.Resolve[*Manager](c); err != nil {
			t.Errorf("Manager not resolved: %v", err)
		}

		if _, err := di.Resolve[*ManagementServer](c); err != nil {
			t.Errorf("ManagementServer not resolved: %v", err)
		}
	})

	t.Run("WithGRPC registers GRPCServer", func(t *testing.T) {
		c := di.New()
		m := NewModule(WithGRPC())

		if err := m.Register(c); err != nil {
			t.Fatalf("Register failed: %v", err)
		}

		if err := c.Build(); err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		// Verify GRPCServer is registered
		if _, err := di.Resolve[*GRPCServer](c); err != nil {
			t.Errorf("GRPCServer not resolved: %v", err)
		}
	})

	t.Run("WithGRPCInterval configures interval", func(t *testing.T) {
		c := di.New()
		m := NewModule(WithGRPC(), WithGRPCInterval(10*time.Second))

		if err := m.Register(c); err != nil {
			t.Fatalf("Register failed: %v", err)
		}

		if err := c.Build(); err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		// Verify GRPCServer is registered
		server, err := di.Resolve[*GRPCServer](c)
		if err != nil {
			t.Fatalf("GRPCServer not resolved: %v", err)
		}

		// Check interval
		if server.interval != 10*time.Second {
			t.Errorf("Expected interval 10s, got %v", server.interval)
		}
	})

	t.Run("without WithGRPC does not register GRPCServer", func(t *testing.T) {
		c := di.New()
		m := NewModule() // No WithGRPC()

		if err := m.Register(c); err != nil {
			t.Fatalf("Register failed: %v", err)
		}

		if err := c.Build(); err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		// Verify GRPCServer is NOT registered
		if di.Has[*GRPCServer](c) {
			t.Error("GRPCServer should not be registered without WithGRPC()")
		}
	})
}
