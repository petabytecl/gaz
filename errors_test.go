package gaz_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/config"
)

// =============================================================================
// ResolutionError tests
// =============================================================================

func TestResolutionError_Error_WithoutChain(t *testing.T) {
	cause := errors.New("service not found")
	err := &gaz.ResolutionError{
		ServiceName: "UserService",
		Chain:       nil,
		Cause:       cause,
	}

	msg := err.Error()
	assert.Contains(t, msg, "UserService")
	assert.Contains(t, msg, "service not found")
	assert.NotContains(t, msg, "chain:")
}

func TestResolutionError_Error_WithChain(t *testing.T) {
	cause := errors.New("database connection failed")
	err := &gaz.ResolutionError{
		ServiceName: "Database",
		Chain:       []string{"App", "UserService", "Database"},
		Cause:       cause,
	}

	msg := err.Error()
	assert.Contains(t, msg, "Database")
	assert.Contains(t, msg, "chain: App -> UserService -> Database")
	assert.Contains(t, msg, "database connection failed")
}

func TestResolutionError_Unwrap(t *testing.T) {
	cause := gaz.ErrDINotFound
	err := &gaz.ResolutionError{
		ServiceName: "SomeService",
		Chain:       []string{"Parent", "SomeService"},
		Cause:       cause,
	}

	// errors.Is should work through Unwrap
	assert.True(t, errors.Is(err, gaz.ErrDINotFound))

	// errors.Unwrap should return the cause
	assert.Equal(t, cause, errors.Unwrap(err))
}

// =============================================================================
// LifecycleError tests
// =============================================================================

func TestLifecycleError_Error(t *testing.T) {
	cause := errors.New("connection refused")
	err := &gaz.LifecycleError{
		ServiceName: "DatabaseService",
		Phase:       "start",
		Cause:       cause,
	}

	msg := err.Error()
	assert.Contains(t, msg, "DatabaseService")
	assert.Contains(t, msg, "start")
	assert.Contains(t, msg, "connection refused")
}

func TestLifecycleError_Error_StopPhase(t *testing.T) {
	cause := errors.New("timeout closing connections")
	err := &gaz.LifecycleError{
		ServiceName: "HTTPServer",
		Phase:       "stop",
		Cause:       cause,
	}

	msg := err.Error()
	assert.Contains(t, msg, "HTTPServer")
	assert.Contains(t, msg, "stop")
	assert.Contains(t, msg, "timeout closing connections")
}

func TestLifecycleError_Unwrap(t *testing.T) {
	cause := errors.New("internal error")
	err := &gaz.LifecycleError{
		ServiceName: "Service",
		Phase:       "start",
		Cause:       cause,
	}

	// errors.Is should work through Unwrap
	assert.True(t, errors.Is(err, cause))

	// errors.Unwrap should return the cause
	assert.Equal(t, cause, errors.Unwrap(err))
}

// =============================================================================
// NewFieldError and NewValidationError wrapper tests
// =============================================================================

func TestNewFieldError_CreatesFieldError(t *testing.T) {
	fe := gaz.NewFieldError("Database.Host", "required", "", "Host is required")

	// Verify the FieldError is created correctly
	assert.Equal(t, "Database.Host", fe.Namespace)
	assert.Equal(t, "required", fe.Tag)
	assert.Equal(t, "", fe.Param)
	assert.Equal(t, "Host is required", fe.Message)
}

func TestNewValidationError_CreatesValidationError(t *testing.T) {
	fieldErrors := []gaz.FieldError{
		gaz.NewFieldError("Database.Host", "required", "", "Host is required"),
		gaz.NewFieldError("Database.Port", "min", "1", "Port must be at least 1"),
	}

	ve := gaz.NewValidationError(fieldErrors)

	// Verify the ValidationError is created
	require.NotNil(t, ve)

	// ValidationError should wrap ErrConfigValidation
	assert.True(t, errors.Is(ve, config.ErrConfigValidation))

	// Check error message includes field information
	msg := ve.Error()
	assert.Contains(t, msg, "Database.Host")
	assert.Contains(t, msg, "Database.Port")
}

func TestValidationError_Errors(t *testing.T) {
	fieldErrors := []gaz.FieldError{
		gaz.NewFieldError("Server.Port", "min", "1", "Port must be positive"),
	}

	ve := gaz.NewValidationError(fieldErrors)

	// Access fields from the validation error
	errs := ve.Errors
	require.Len(t, errs, 1)
	assert.Equal(t, "Server.Port", errs[0].Namespace)
}

// =============================================================================
// Sentinel Error alias tests
// =============================================================================

func TestSentinelErrorAliases(t *testing.T) {
	// Test that gaz error aliases match their source
	assert.True(t, errors.Is(gaz.ErrDINotFound, gaz.ErrNotFound))
	assert.True(t, errors.Is(gaz.ErrDICycle, gaz.ErrCycle))
	assert.True(t, errors.Is(gaz.ErrDIDuplicate, gaz.ErrDuplicate))
	assert.True(t, errors.Is(gaz.ErrDINotSettable, gaz.ErrNotSettable))
	assert.True(t, errors.Is(gaz.ErrDITypeMismatch, gaz.ErrTypeMismatch))
	assert.True(t, errors.Is(gaz.ErrDIAlreadyBuilt, gaz.ErrAlreadyBuilt))
	assert.True(t, errors.Is(gaz.ErrDIInvalidProvider, gaz.ErrInvalidProvider))
}
