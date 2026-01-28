package config_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz/config"
)

// =============================================================================
// Test ValidateStruct
// =============================================================================

type validConfig struct {
	Host string `mapstructure:"host" validate:"required"`
	Port int    `mapstructure:"port" validate:"min=1,max=65535"`
}

type invalidRequiredConfig struct {
	Host string `mapstructure:"host" validate:"required"`
}

type invalidMinConfig struct {
	Port int `mapstructure:"port" validate:"min=1"`
}

type invalidMaxConfig struct {
	Count int `mapstructure:"count" validate:"max=100"`
}

type nestedConfig struct {
	Server struct {
		Host string `mapstructure:"host" validate:"required"`
	} `mapstructure:"server"`
}

type oneOfConfig struct {
	Level string `mapstructure:"level" validate:"required,oneof=debug info warn error"`
}

func TestValidateStruct_ValidStruct_ReturnsNil(t *testing.T) {
	cfg := validConfig{
		Host: "localhost",
		Port: 8080,
	}

	err := config.ValidateStruct(&cfg)
	assert.NoError(t, err)
}

func TestValidateStruct_MissingRequired_ReturnsError(t *testing.T) {
	cfg := invalidRequiredConfig{
		Host: "", // Empty, but required
	}

	err := config.ValidateStruct(&cfg)
	require.Error(t, err)
	assert.True(t, errors.Is(err, config.ErrConfigValidation))

	// Check error message contains field name
	assert.Contains(t, err.Error(), "host")
	assert.Contains(t, err.Error(), "required")
}

func TestValidateStruct_MinViolation_ReturnsError(t *testing.T) {
	cfg := invalidMinConfig{
		Port: 0, // Less than min=1
	}

	err := config.ValidateStruct(&cfg)
	require.Error(t, err)
	assert.True(t, errors.Is(err, config.ErrConfigValidation))

	assert.Contains(t, err.Error(), "port")
	assert.Contains(t, err.Error(), "at least")
}

func TestValidateStruct_MaxViolation_ReturnsError(t *testing.T) {
	cfg := invalidMaxConfig{
		Count: 200, // Greater than max=100
	}

	err := config.ValidateStruct(&cfg)
	require.Error(t, err)
	assert.True(t, errors.Is(err, config.ErrConfigValidation))

	assert.Contains(t, err.Error(), "count")
	assert.Contains(t, err.Error(), "at most")
}

func TestValidateStruct_NestedStruct_ValidatesNested(t *testing.T) {
	cfg := nestedConfig{}
	// Server.Host is empty but required

	err := config.ValidateStruct(&cfg)
	require.Error(t, err)
	assert.True(t, errors.Is(err, config.ErrConfigValidation))

	assert.Contains(t, err.Error(), "server.host")
}

func TestValidateStruct_OneOf_ValidValue(t *testing.T) {
	cfg := oneOfConfig{
		Level: "info",
	}

	err := config.ValidateStruct(&cfg)
	assert.NoError(t, err)
}

func TestValidateStruct_OneOf_InvalidValue(t *testing.T) {
	cfg := oneOfConfig{
		Level: "verbose", // Not in oneof list
	}

	err := config.ValidateStruct(&cfg)
	require.Error(t, err)
	assert.True(t, errors.Is(err, config.ErrConfigValidation))

	assert.Contains(t, err.Error(), "level")
	assert.Contains(t, err.Error(), "one of")
}

// =============================================================================
// Test ValidationErrors and FieldError
// =============================================================================

func TestValidationErrors_Error_FormatsCorrectly(t *testing.T) {
	fieldErrors := []config.FieldError{
		{Namespace: "Config.host", Tag: "required", Message: "required field cannot be empty"},
		{Namespace: "Config.port", Tag: "min", Param: "1", Message: "must be at least 1"},
	}

	ve := config.NewValidationErrors(fieldErrors)
	errStr := ve.Error()

	assert.Contains(t, errStr, "validation failed")
	assert.Contains(t, errStr, "Config.host")
	assert.Contains(t, errStr, "Config.port")
}

func TestValidationErrors_Unwrap_ReturnsErrConfigValidation(t *testing.T) {
	ve := config.NewValidationErrors(nil)

	assert.True(t, errors.Is(ve, config.ErrConfigValidation))
}

func TestFieldError_String_WithTag(t *testing.T) {
	fe := config.NewFieldError("Config.host", "required", "", "required field cannot be empty")
	s := fe.String()

	assert.Contains(t, s, "Config.host")
	assert.Contains(t, s, "required field cannot be empty")
	assert.Contains(t, s, `validate:"required"`)
}

func TestFieldError_String_WithoutTag(t *testing.T) {
	fe := config.FieldError{
		Namespace: "Config.host",
		Message:   "custom error",
	}
	s := fe.String()

	assert.Contains(t, s, "Config.host")
	assert.Contains(t, s, "custom error")
	assert.NotContains(t, s, "validate:")
}

// =============================================================================
// Test mapstructure tag name in error messages
// =============================================================================

type mapstructureTagConfig struct {
	DatabaseHost string `mapstructure:"db_host" validate:"required"`
}

func TestValidateStruct_UsesMapstructureTagInErrorMessage(t *testing.T) {
	cfg := mapstructureTagConfig{
		DatabaseHost: "", // Empty but required
	}

	err := config.ValidateStruct(&cfg)
	require.Error(t, err)

	// Should use mapstructure name "db_host", not Go field name "DatabaseHost"
	assert.Contains(t, err.Error(), "db_host")
	assert.NotContains(t, err.Error(), "DatabaseHost")
}

// =============================================================================
// Test multiple validation errors
// =============================================================================

type multiErrorConfig struct {
	Host string `mapstructure:"host" validate:"required"`
	Port int    `mapstructure:"port" validate:"required,min=1"`
	Name string `mapstructure:"name" validate:"required"`
}

func TestValidateStruct_MultipleErrors_ReportsAll(t *testing.T) {
	cfg := multiErrorConfig{
		Host: "",
		Port: 0,
		Name: "",
	}

	err := config.ValidateStruct(&cfg)
	require.Error(t, err)

	// Should report all three errors
	assert.Contains(t, err.Error(), "host")
	assert.Contains(t, err.Error(), "port")
	assert.Contains(t, err.Error(), "name")

	// Check we can access the individual errors
	var ve config.ValidationErrors
	require.True(t, errors.As(err, &ve))
	assert.Len(t, ve.Errors, 3)
}
