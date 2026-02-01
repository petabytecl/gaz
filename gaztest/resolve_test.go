package gaztest_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/gaztest"
)

// =============================================================================
// TestRequireResolve_Success
// =============================================================================

func TestRequireResolve_Success(t *testing.T) {
	// Create a base app with a registered service
	baseApp := gaz.New()
	db := &MockDatabase{queryResult: "test-data"}
	err := gaz.For[*MockDatabase](baseApp.Container()).Instance(db)
	require.NoError(t, err)

	// Build test app
	app, err := gaztest.New(t).WithApp(baseApp).Build()
	require.NoError(t, err)

	// RequireResolve should return the service
	resolved := gaztest.RequireResolve[*MockDatabase](t, app)
	require.NotNil(t, resolved)
	require.Equal(t, "test-data", resolved.Query())
	require.Same(t, db, resolved, "should return the same instance")
}

// =============================================================================
// TestRequireResolve_NotRegistered
// =============================================================================

func TestRequireResolve_NotRegistered(t *testing.T) {
	// Create an empty app with no registered services
	app, err := gaztest.New(t).Build()
	require.NoError(t, err)

	// Use mock TB to capture Fatalf call
	mockT := &fatalfCatcher{realT: t}

	// RequireResolve should call Fatalf when type is not registered
	_ = gaztest.RequireResolve[*MockDatabase](mockT, app)

	require.True(t, mockT.fatalfCalled, "Fatalf should have been called")
	require.Contains(t, mockT.fatalfMessage, "RequireResolve")
	require.Contains(t, mockT.fatalfMessage, "MockDatabase")
}

// =============================================================================
// fatalfCatcher - Mock TB that captures Fatalf calls without terminating
// =============================================================================

type fatalfCatcher struct {
	realT         *testing.T
	fatalfCalled  bool
	fatalfMessage string
}

func (m *fatalfCatcher) Logf(format string, args ...any) {
	m.realT.Logf(format, args...)
}

func (m *fatalfCatcher) Errorf(format string, args ...any) {
	m.realT.Errorf(format, args...)
}

func (m *fatalfCatcher) Fatalf(format string, args ...any) {
	m.fatalfCalled = true
	m.fatalfMessage = fmt.Sprintf(format, args...)
	// Don't actually fail - we just want to capture the call
}

func (m *fatalfCatcher) FailNow() {
	m.fatalfCalled = true
}

func (m *fatalfCatcher) Cleanup(f func()) {
	m.realT.Cleanup(f)
}

func (m *fatalfCatcher) Helper() {
	m.realT.Helper()
}
