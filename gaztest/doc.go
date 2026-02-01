// Package gaztest provides test utilities for gaz applications.
// It enables easy testing with automatic cleanup, mock injection,
// and assertion methods that fail tests on error.
//
// For a complete testing guide, see the README.md file in this package.
//
// # Basic Usage
//
//	func TestMyService(t *testing.T) {
//	    app, err := gaztest.New(t).Build()
//	    require.NoError(t, err)
//
//	    app.RequireStart()
//	    defer app.RequireStop()
//
//	    // ... test logic
//	}
//
// # With Modules (v3 Pattern)
//
// Use WithModules for testing modules without a pre-built app:
//
//	func TestWithModules(t *testing.T) {
//	    app, err := gaztest.New(t).
//	        WithModules(health.NewModule(), worker.NewModule()).
//	        Build()
//	    require.NoError(t, err)
//
//	    app.RequireStart()
//	    defer app.RequireStop()
//
//	    // Modules are registered and started
//	}
//
// # Type-Safe Resolution (v3 Pattern)
//
// Use RequireResolve for type-safe service resolution that fails the test on error:
//
//	func TestRequireResolve(t *testing.T) {
//	    app, err := gaztest.New(t).Build()
//	    require.NoError(t, err)
//
//	    app.RequireStart()
//	    defer app.RequireStop()
//
//	    // Fails test if resolution fails - no error check needed
//	    svc := gaztest.RequireResolve[*MyService](t, app)
//	    // use svc directly
//	}
//
// # With Mock Replacement
//
//	func TestWithMock(t *testing.T) {
//	    // First create an app with registered services
//	    baseApp := gaz.New()
//	    gaz.For[Database](baseApp.Container()).Instance(&RealDatabase{})
//	    baseApp.Build()
//
//	    // Then create test app with mock replacement
//	    mock := &MockDatabase{}
//	    app, err := gaztest.New(t).
//	        WithApp(baseApp).
//	        Replace(mock).
//	        Build()
//	    require.NoError(t, err)
//
//	    app.RequireStart()
//	    // ... test logic using mock
//	}
//
// # Configuration Injection
//
// Use WithConfigMap for injecting test configuration values:
//
//	func TestWithConfig(t *testing.T) {
//	    app, err := gaztest.New(t).
//	        WithConfigMap(map[string]any{
//	            "worker.pool_size": 2,
//	            "health.port":      0,
//	        }).
//	        Build()
//	    require.NoError(t, err)
//	    // ...
//	}
//
// # Subsystem Test Helpers
//
// Each subsystem provides test helpers in a testing.go file:
//
//   - health: TestConfig, MockRegistrar, RequireHealthy
//   - worker: MockWorker, SimpleWorker, RequireWorkerStarted
//   - cron: MockJob, SimpleJob, RequireJobRan
//   - config: MapBackend, TestManager, RequireConfigLoaded
//   - eventbus: TestBus, TestSubscriber, RequireEventsReceived
//
// # Custom Timeout
//
//	func TestWithTimeout(t *testing.T) {
//	    app, err := gaztest.New(t).
//	        WithTimeout(10 * time.Second).
//	        Build()
//	    require.NoError(t, err)
//	    // ...
//	}
package gaztest
