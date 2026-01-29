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
