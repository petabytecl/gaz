package gaztest_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/gaztest"
)

// Example demonstrates the basic usage pattern for gaztest.
// This is the simplest way to create a test app with automatic cleanup.
//
// In an actual test function:
//
//	func TestMyFeature(t *testing.T) {
//	    app, err := gaztest.New(t).Build()
//	    require.NoError(t, err)
//
//	    app.RequireStart()
//	    defer app.RequireStop()
//
//	    // ... test logic ...
//	}
func Example() {
	// In a real test, use the *testing.T provided by the test function.
	// For this example, we use a mock that implements gaztest.TB.
	t := &testing.T{}

	// Create a new test app - cleanup is registered automatically
	app, err := gaztest.New(t).Build()
	if err != nil {
		fmt.Println("build failed:", err)
		return
	}

	// Start the app - fails test if error occurs
	app.RequireStart()

	// ... run your test assertions here ...

	// Stop is optional - t.Cleanup() will handle it automatically
	// but explicit stop is fine if you need early cleanup
	app.RequireStop()

	// Note: In real tests, use require/assert for verification
}

// Example_withTimeout demonstrates how to set a custom timeout.
// The default timeout is 5 seconds, which may need to be increased
// for slow-starting services or decreased for faster feedback.
func Example_withTimeout() {
	t := &testing.T{}

	// Set a custom 10 second timeout for start/stop operations
	app, err := gaztest.New(t).
		WithTimeout(10 * time.Second).
		Build()
	if err != nil {
		fmt.Println("build failed:", err)
		return
	}

	app.RequireStart()
	defer app.RequireStop()

	// Test logic with services that need more startup time...
}

// Example_withApp demonstrates using a pre-configured gaz.App.
// This is useful when you have services already registered that you
// want to test with, optionally replacing some with mocks.
func Example_withApp() {
	t := &testing.T{}

	// Create and configure a base app
	baseApp := gaz.New()

	// Register a service (in real tests, this might be your production service)
	type MyService struct {
		Name string
	}
	svc := &MyService{Name: "production"}
	if err := gaz.For[*MyService](baseApp.Container()).Instance(svc); err != nil {
		fmt.Println("register failed:", err)
		return
	}
	if err := baseApp.Build(); err != nil {
		fmt.Println("base build failed:", err)
		return
	}

	// Create test app using the base app
	app, err := gaztest.New(t).
		WithApp(baseApp).
		Build()
	if err != nil {
		fmt.Println("test build failed:", err)
		return
	}

	app.RequireStart()
	defer app.RequireStop()

	// Resolve and use the service
	resolved, _ := gaz.Resolve[*MyService](app.Container())
	_ = resolved // Use in your test assertions
}

// Example_replace demonstrates mock injection.
// Replace swaps a registered type with a mock implementation,
// allowing you to test components in isolation.
func Example_replace() {
	t := &testing.T{}

	// Create base app with "real" service
	baseApp := gaz.New()

	type EmailSender struct {
		SendCount int
		TestMode  bool
	}
	realSender := &EmailSender{TestMode: false}
	if err := gaz.For[*EmailSender](baseApp.Container()).Instance(realSender); err != nil {
		fmt.Println("register failed:", err)
		return
	}
	if err := baseApp.Build(); err != nil {
		fmt.Println("base build failed:", err)
		return
	}

	// Create mock for testing
	mockSender := &EmailSender{TestMode: true}

	// Create test app with mock replacement
	app, err := gaztest.New(t).
		WithApp(baseApp).
		Replace(mockSender).
		Build()
	if err != nil {
		fmt.Println("test build failed:", err)
		return
	}

	app.RequireStart()
	defer app.RequireStop()

	// Resolve the service - should get the mock, not the real implementation
	resolved, _ := gaz.Resolve[*EmailSender](app.Container())
	_ = resolved // resolved.TestMode == true
}

// TestExample_BasicUsage is a complete test demonstrating the typical pattern.
// This runs as an actual test and shows how gaztest integrates with testing.T.
func TestExample_BasicUsage(t *testing.T) {
	// gaztest.New(t) creates a builder that:
	// - Uses 5 second default timeout for start/stop
	// - Registers t.Cleanup() for automatic stop when test ends
	app, err := gaztest.New(t).Build()
	if err != nil {
		t.Fatalf("failed to build app: %v", err)
	}

	// RequireStart starts the app or fails the test
	app.RequireStart()

	// Optional: explicitly stop (cleanup would handle this anyway)
	defer app.RequireStop()

	// Access the container to resolve services
	container := app.Container()
	if container == nil {
		t.Fatal("container should not be nil")
	}

	t.Log("Test app running with container access")
}

// TestExample_MockReplacement demonstrates replacing a service with a mock.
func TestExample_MockReplacement(t *testing.T) {
	// Step 1: Create base app with "production" service
	type Database struct {
		Name string
	}

	baseApp := gaz.New()
	prodDB := &Database{Name: "postgresql"}
	if err := gaz.For[*Database](baseApp.Container()).Instance(prodDB); err != nil {
		t.Fatalf("failed to register: %v", err)
	}
	if err := baseApp.Build(); err != nil {
		t.Fatalf("failed to build base: %v", err)
	}

	// Step 2: Create test app with mock replacement
	mockDB := &Database{Name: "mock-db"}
	app, err := gaztest.New(t).
		WithApp(baseApp).
		Replace(mockDB).
		Build()
	if err != nil {
		t.Fatalf("failed to build test app: %v", err)
	}

	app.RequireStart()
	defer app.RequireStop()

	// Step 3: Verify the mock is returned
	resolved, err := gaz.Resolve[*Database](app.Container())
	if err != nil {
		t.Fatalf("failed to resolve: %v", err)
	}

	if resolved.Name != "mock-db" {
		t.Errorf("expected mock-db, got %s", resolved.Name)
	}

	t.Logf("Successfully replaced production DB with mock: %s", resolved.Name)
}
