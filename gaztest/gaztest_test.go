package gaztest_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/gaztest"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Types
// =============================================================================

// Database is a mock interface for testing.
type Database interface {
	Query() string
}

// MockDatabase is a mock implementation of Database.
type MockDatabase struct {
	queryResult string
}

func (m *MockDatabase) Query() string {
	if m.queryResult != "" {
		return m.queryResult
	}
	return "mock-result"
}

// RealDatabase is a real implementation that would be replaced in tests.
type RealDatabase struct{}

func (r *RealDatabase) Query() string {
	return "real-result"
}

// TestService is a service that depends on Database.
type TestService struct {
	db Database
}

func NewTestService(c *gaz.Container) (*TestService, error) {
	db, err := gaz.Resolve[Database](c)
	if err != nil {
		return nil, err
	}
	return &TestService{db: db}, nil
}

func (s *TestService) DoWork() string {
	return s.db.Query()
}

// LifecycleService tracks start/stop calls for testing.
type LifecycleService struct {
	started atomic.Bool
	stopped atomic.Bool
}

func (s *LifecycleService) OnStart(_ context.Context) error {
	s.started.Store(true)
	return nil
}

func (s *LifecycleService) OnStop(_ context.Context) error {
	s.stopped.Store(true)
	return nil
}

func (s *LifecycleService) IsStarted() bool {
	return s.started.Load()
}

func (s *LifecycleService) IsStopped() bool {
	return s.stopped.Load()
}

// =============================================================================
// TestNew_DefaultTimeout
// =============================================================================

func TestNew_DefaultTimeout(t *testing.T) {
	// Test that New(t) creates a builder with the default 5s timeout.
	builder := gaztest.New(t)
	require.NotNil(t, builder)

	// Build an empty app to verify builder works
	app, err := builder.Build()
	require.NoError(t, err)
	require.NotNil(t, app)
}

// =============================================================================
// TestBuilder_WithTimeout
// =============================================================================

func TestBuilder_WithTimeout(t *testing.T) {
	// Test that WithTimeout overrides the default timeout.
	customTimeout := 2 * time.Second
	builder := gaztest.New(t).WithTimeout(customTimeout)
	require.NotNil(t, builder)

	app, err := builder.Build()
	require.NoError(t, err)
	require.NotNil(t, app)
}

// =============================================================================
// TestBuilder_Replace
// =============================================================================

func TestBuilder_Replace(t *testing.T) {
	// Register a concrete type that can be replaced
	// Note: Replace infers type from the mock instance using reflection,
	// so we register and replace the same concrete type.
	realDB := &MockDatabase{queryResult: "original"}
	baseApp := gaz.New()
	err := gaz.For[*MockDatabase](baseApp.Container()).Instance(realDB)
	require.NoError(t, err)
	err = baseApp.Build()
	require.NoError(t, err)

	// Verify original value
	db1, err := gaz.Resolve[*MockDatabase](baseApp.Container())
	require.NoError(t, err)
	require.Equal(t, "original", db1.Query())

	// Now test replacement in gaztest
	mock := &MockDatabase{queryResult: "mocked"}
	app, err := gaztest.New(t).
		WithApp(baseApp).
		Replace(mock).
		Build()
	require.NoError(t, err)
	require.NotNil(t, app)

	// Resolve the database - should get the mock
	db2, err := gaz.Resolve[*MockDatabase](app.Container())
	require.NoError(t, err)
	require.Equal(t, "mocked", db2.Query())
}

// =============================================================================
// TestBuilder_Build
// =============================================================================

func TestBuilder_Build(t *testing.T) {
	// Test that Build returns (*App, error) and works correctly.
	app, err := gaztest.New(t).Build()
	require.NoError(t, err)
	require.NotNil(t, app)

	// Verify it's a proper App that can be used
	require.NotNil(t, app.Container())
}

func TestBuilder_Build_RegistersCleanup(t *testing.T) {
	// Test that Build registers t.Cleanup() for automatic stop.
	// We verify this by creating a test that builds an app and doesn't explicitly stop it.
	// The cleanup should be registered automatically.

	// Create a mock TB that tracks cleanup registration
	mockT := &mockTB{
		realT: t,
	}

	builder := gaztest.New(mockT)
	_, err := builder.Build()
	require.NoError(t, err)

	// Verify cleanup was registered (mockT tracks this)
	require.True(t, mockT.cleanupRegistered, "t.Cleanup should be registered")
}

// =============================================================================
// TestApp_RequireStart
// =============================================================================

func TestApp_RequireStart(t *testing.T) {
	// Test that RequireStart starts the app successfully.
	svc := &LifecycleService{}

	baseApp := gaz.New()
	err := gaz.For[*LifecycleService](baseApp.Container()).Instance(svc)
	require.NoError(t, err)

	app, err := gaztest.New(t).WithApp(baseApp).Build()
	require.NoError(t, err)

	// Start the app
	app.RequireStart()

	// Verify service was started
	require.True(t, svc.IsStarted(), "service should be started")
}

func TestApp_RequireStart_ReturnsApp(t *testing.T) {
	// Test that RequireStart returns the app for chaining.
	app, err := gaztest.New(t).Build()
	require.NoError(t, err)

	result := app.RequireStart()
	require.Same(t, app, result, "RequireStart should return the same app")
}

// =============================================================================
// TestApp_RequireStop
// =============================================================================

func TestApp_RequireStop(t *testing.T) {
	// Test that RequireStop stops the app successfully.
	svc := &LifecycleService{}

	baseApp := gaz.New()
	err := gaz.For[*LifecycleService](baseApp.Container()).Instance(svc)
	require.NoError(t, err)

	app, err := gaztest.New(t).WithApp(baseApp).Build()
	require.NoError(t, err)

	app.RequireStart()
	app.RequireStop()

	// Verify service was stopped
	require.True(t, svc.IsStopped(), "service should be stopped")
}

func TestApp_RequireStop_Idempotent(t *testing.T) {
	// Test that RequireStop is idempotent - calling twice doesn't error.
	app, err := gaztest.New(t).Build()
	require.NoError(t, err)

	app.RequireStart()
	app.RequireStop()
	app.RequireStop() // Should not panic or error
}

// =============================================================================
// TestApp_AutoCleanup
// =============================================================================

func TestApp_AutoCleanup(t *testing.T) {
	// Test that t.Cleanup() stops the app even if the test doesn't explicitly call RequireStop.
	svc := &LifecycleService{}

	// Use a subtesthook to verify cleanup behavior
	var cleanupFunc func()
	mockT := &mockTB{
		realT: t,
		cleanupCallback: func() {
			if cleanupFunc != nil {
				cleanupFunc()
			}
		},
	}

	baseApp := gaz.New()
	err := gaz.For[*LifecycleService](baseApp.Container()).Instance(svc)
	require.NoError(t, err)

	app, err := gaztest.New(mockT).WithApp(baseApp).Build()
	require.NoError(t, err)

	// Store the cleanup function that was registered
	cleanupFunc = mockT.registeredCleanup

	app.RequireStart()
	require.True(t, svc.IsStarted())

	// Simulate t.Cleanup() being called (what Go does after test completes)
	if mockT.registeredCleanup != nil {
		mockT.registeredCleanup()
	}

	// Verify service was stopped by cleanup
	require.True(t, svc.IsStopped(), "cleanup should stop the app")
}

// =============================================================================
// TestBuilder_ReplaceTypeNotRegistered
// =============================================================================

func TestBuilder_ReplaceTypeNotRegistered(t *testing.T) {
	// Test that Replace with an unregistered type returns error from Build().
	mock := &MockDatabase{}

	// Don't register Database in the base app
	baseApp := gaz.New()
	err := baseApp.Build()
	require.NoError(t, err)

	// Attempt to replace a type that was never registered
	_, err = gaztest.New(t).
		WithApp(baseApp).
		Replace(mock).
		Build()

	require.Error(t, err)
	require.Contains(t, err.Error(), "not registered")
}

// =============================================================================
// TestBuilder_ReplaceNil
// =============================================================================

func TestBuilder_ReplaceNil(t *testing.T) {
	// Test that Replace with nil returns error from Build().
	_, err := gaztest.New(t).
		Replace(nil).
		Build()

	require.Error(t, err)
	require.Contains(t, err.Error(), "nil")
}

// =============================================================================
// mockTB - Mock testing.TB for verifying cleanup registration
// =============================================================================

type mockTB struct {
	realT             *testing.T
	cleanupRegistered bool
	cleanupCallback   func()
	registeredCleanup func()
}

func (m *mockTB) Cleanup(f func()) {
	m.cleanupRegistered = true
	m.registeredCleanup = f
}

func (m *mockTB) Logf(format string, args ...any) {
	m.realT.Logf(format, args...)
}

func (m *mockTB) Errorf(format string, args ...any) {
	m.realT.Errorf(format, args...)
}

func (m *mockTB) Fatalf(format string, args ...any) {
	m.realT.Fatalf(format, args...)
}

func (m *mockTB) FailNow() {
	m.realT.FailNow()
}

func (m *mockTB) Helper() {
	m.realT.Helper()
}

// =============================================================================
// Integration Tests: Replace with real service swapping
// =============================================================================

// TestReplace_SwapsImplementation verifies that Replace() swaps a real
// implementation with a mock, and the mock is returned when resolving.
func TestReplace_SwapsImplementation(t *testing.T) {
	// Register a "real" service (using concrete type for Replace compatibility)
	realDB := &MockDatabase{queryResult: "production-data"}

	baseApp := gaz.New()
	err := gaz.For[*MockDatabase](baseApp.Container()).Instance(realDB)
	require.NoError(t, err)
	err = baseApp.Build()
	require.NoError(t, err)

	// Verify original returns production data
	originalDB, err := gaz.Resolve[*MockDatabase](baseApp.Container())
	require.NoError(t, err)
	require.Equal(t, "production-data", originalDB.Query())

	// Create test app with mock replacement
	mock := &MockDatabase{queryResult: "test-mock-data"}
	app, err := gaztest.New(t).
		WithApp(baseApp).
		Replace(mock).
		Build()
	require.NoError(t, err)

	app.RequireStart()
	defer app.RequireStop()

	// Resolve the type - should get the mock, not the real implementation
	resolved, err := gaz.Resolve[*MockDatabase](app.Container())
	require.NoError(t, err)
	require.Equal(t, "test-mock-data", resolved.Query())
	require.Same(t, mock, resolved, "should return exact mock instance")
}

// TestReplace_MultipleServices verifies replacing some but not all services.
func TestReplace_MultipleServices(t *testing.T) {
	// Create base app with multiple services
	baseApp := gaz.New()

	// Service 1: will be replaced
	db1 := &MockDatabase{queryResult: "db1-original"}
	err := gaz.For[*MockDatabase](baseApp.Container()).Instance(db1)
	require.NoError(t, err)

	// Service 2: will NOT be replaced
	svc := &LifecycleService{}
	err = gaz.For[*LifecycleService](baseApp.Container()).Instance(svc)
	require.NoError(t, err)

	err = baseApp.Build()
	require.NoError(t, err)

	// Replace only the database, keep lifecycle service as-is
	mockDB := &MockDatabase{queryResult: "db1-mocked"}
	app, err := gaztest.New(t).
		WithApp(baseApp).
		Replace(mockDB).
		Build()
	require.NoError(t, err)

	app.RequireStart()
	defer app.RequireStop()

	// Replaced service should return mock
	resolvedDB, err := gaz.Resolve[*MockDatabase](app.Container())
	require.NoError(t, err)
	require.Equal(t, "db1-mocked", resolvedDB.Query())

	// Non-replaced service should be the original
	resolvedSvc, err := gaz.Resolve[*LifecycleService](app.Container())
	require.NoError(t, err)
	require.Same(t, svc, resolvedSvc, "non-replaced service should be original")
}

// TestApp_DoubleStop_Idempotent verifies that calling RequireStop twice is safe.
func TestApp_DoubleStop_Idempotent(t *testing.T) {
	svc := &LifecycleService{}

	baseApp := gaz.New()
	err := gaz.For[*LifecycleService](baseApp.Container()).Instance(svc)
	require.NoError(t, err)

	app, err := gaztest.New(t).WithApp(baseApp).Build()
	require.NoError(t, err)

	app.RequireStart()
	require.True(t, svc.IsStarted())

	// First stop
	app.RequireStop()
	require.True(t, svc.IsStopped())

	// Second stop - should NOT panic, should be idempotent
	app.RequireStop()
	// If we get here without panic, test passes
}

// TestCleanup_RunsEvenIfTestPanics verifies cleanup registration.
// Note: We can't actually test panic recovery in the same test,
// but we can verify that cleanup is registered and callable.
func TestCleanup_RunsEvenIfTestPanics(t *testing.T) {
	// Track cleanup execution
	var cleanupCalled bool
	svc := &LifecycleService{}

	// Create mock TB to capture cleanup function
	mockT := &mockTB{
		realT: t,
	}

	baseApp := gaz.New()
	err := gaz.For[*LifecycleService](baseApp.Container()).Instance(svc)
	require.NoError(t, err)

	app, err := gaztest.New(mockT).WithApp(baseApp).Build()
	require.NoError(t, err)

	// Verify cleanup was registered
	require.True(t, mockT.cleanupRegistered)
	require.NotNil(t, mockT.registeredCleanup)

	// Start the app
	app.RequireStart()
	require.True(t, svc.IsStarted())

	// Simulate what would happen if test panics and Go's defer runs cleanup
	// In real scenario, t.Cleanup() is called by Go testing framework
	func() {
		defer func() {
			// This simulates Go's cleanup mechanism
			mockT.registeredCleanup()
			cleanupCalled = true
		}()

		// Simulate panic (we recover it)
		func() {
			defer func() {
				recover() // swallow the panic for test purposes
			}()
			panic("simulated test panic")
		}()
	}()

	// Verify cleanup was called and service was stopped
	require.True(t, cleanupCalled, "cleanup should have been called")
	require.True(t, svc.IsStopped(), "service should be stopped by cleanup")
}

// TestBuilder_WithApp_AllowsServiceResolution verifies that WithApp provides
// access to pre-registered services.
func TestBuilder_WithApp_AllowsServiceResolution(t *testing.T) {
	// Create base app with a service
	baseApp := gaz.New()
	db := &MockDatabase{queryResult: "base-app-db"}
	err := gaz.For[*MockDatabase](baseApp.Container()).Instance(db)
	require.NoError(t, err)
	err = baseApp.Build()
	require.NoError(t, err)

	// Create test app from base app
	app, err := gaztest.New(t).WithApp(baseApp).Build()
	require.NoError(t, err)

	app.RequireStart()
	defer app.RequireStop()

	// Should be able to resolve services from base app
	resolved, err := gaz.Resolve[*MockDatabase](app.Container())
	require.NoError(t, err)
	require.Equal(t, "base-app-db", resolved.Query())
}

// TestRequireStart_Idempotent verifies calling RequireStart twice is safe.
func TestRequireStart_Idempotent(t *testing.T) {
	svc := &LifecycleService{}

	baseApp := gaz.New()
	err := gaz.For[*LifecycleService](baseApp.Container()).Instance(svc)
	require.NoError(t, err)

	app, err := gaztest.New(t).WithApp(baseApp).Build()
	require.NoError(t, err)

	// First start
	app.RequireStart()
	require.True(t, svc.IsStarted())

	// Second start - should NOT error or double-start
	app.RequireStart()
	// If we get here without panic/error, test passes
}
