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
// Test humanizeTag coverage (indirectly via ValidateStruct)
// =============================================================================

// Test structs for each validation tag type
type gteConfig struct {
	Value int `mapstructure:"value" validate:"gte=10"`
}

type lteConfig struct {
	Value int `mapstructure:"value" validate:"lte=50"`
}

type gtConfig struct {
	Value int `mapstructure:"value" validate:"gt=0"`
}

type ltConfig struct {
	Value int `mapstructure:"value" validate:"lt=100"`
}

type emailConfig struct {
	Email string `mapstructure:"email" validate:"email"`
}

type urlConfig struct {
	URL string `mapstructure:"url" validate:"url"`
}

type ipConfig struct {
	IP string `mapstructure:"ip" validate:"ip"`
}

type ipv4Config struct {
	IP string `mapstructure:"ip" validate:"ipv4"`
}

type ipv6Config struct {
	IP string `mapstructure:"ip" validate:"ipv6"`
}

type requiredIfConfig struct {
	Field1 string `mapstructure:"field1"`
	Field2 string `mapstructure:"field2" validate:"required_if=Field1 yes"`
}

type requiredUnlessConfig struct {
	Field1 string `mapstructure:"field1"`
	Field2 string `mapstructure:"field2" validate:"required_unless=Field1 no"`
}

type requiredWithConfig struct {
	Field1 string `mapstructure:"field1"`
	Field2 string `mapstructure:"field2" validate:"required_with=Field1"`
}

type requiredWithoutConfig struct {
	Field1 string `mapstructure:"field1"`
	Field2 string `mapstructure:"field2" validate:"required_without=Field1"`
}

type unknownTagConfig struct {
	Value string `mapstructure:"value" validate:"alphanum"`
}

func TestHumanizeTag_Gte_Message(t *testing.T) {
	cfg := gteConfig{Value: 5} // Less than gte=10
	err := config.ValidateStruct(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "greater than or equal to")
	assert.Contains(t, err.Error(), "10")
}

func TestHumanizeTag_Lte_Message(t *testing.T) {
	cfg := lteConfig{Value: 100} // Greater than lte=50
	err := config.ValidateStruct(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "less than or equal to")
	assert.Contains(t, err.Error(), "50")
}

func TestHumanizeTag_Gt_Message(t *testing.T) {
	cfg := gtConfig{Value: 0} // Not greater than gt=0
	err := config.ValidateStruct(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "greater than")
	assert.Contains(t, err.Error(), "0")
}

func TestHumanizeTag_Lt_Message(t *testing.T) {
	cfg := ltConfig{Value: 100} // Not less than lt=100
	err := config.ValidateStruct(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "less than")
	assert.Contains(t, err.Error(), "100")
}

func TestHumanizeTag_Email_Message(t *testing.T) {
	cfg := emailConfig{Email: "invalid-email"}
	err := config.ValidateStruct(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "valid email address")
}

func TestHumanizeTag_URL_Message(t *testing.T) {
	cfg := urlConfig{URL: "not-a-url"}
	err := config.ValidateStruct(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "valid URL")
}

func TestHumanizeTag_IP_Message(t *testing.T) {
	cfg := ipConfig{IP: "invalid-ip"}
	err := config.ValidateStruct(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "valid IP address")
}

func TestHumanizeTag_IPv4_Message(t *testing.T) {
	cfg := ipv4Config{IP: "::1"} // IPv6, not IPv4
	err := config.ValidateStruct(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "valid IPv4 address")
}

func TestHumanizeTag_IPv6_Message(t *testing.T) {
	cfg := ipv6Config{IP: "192.168.1.1"} // IPv4, not IPv6
	err := config.ValidateStruct(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "valid IPv6 address")
}

func TestHumanizeTag_RequiredIf_Message(t *testing.T) {
	cfg := requiredIfConfig{Field1: "yes", Field2: ""} // Field2 required when Field1="yes"
	err := config.ValidateStruct(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required when")
}

func TestHumanizeTag_RequiredUnless_Message(t *testing.T) {
	cfg := requiredUnlessConfig{Field1: "yes", Field2: ""} // Field2 required unless Field1="no"
	err := config.ValidateStruct(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required unless")
}

func TestHumanizeTag_RequiredWith_Message(t *testing.T) {
	cfg := requiredWithConfig{Field1: "present", Field2: ""} // Field2 required when Field1 is present
	err := config.ValidateStruct(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is present")
}

func TestHumanizeTag_RequiredWithout_Message(t *testing.T) {
	cfg := requiredWithoutConfig{Field1: "", Field2: ""} // Field2 required when Field1 is absent
	err := config.ValidateStruct(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is absent")
}

func TestHumanizeTag_UnknownTag_DefaultMessage(t *testing.T) {
	cfg := unknownTagConfig{Value: "not@alphanum!"} // alphanum is not in humanizeTag switch
	err := config.ValidateStruct(&cfg)
	require.Error(t, err)
	// Default case returns "failed <tag> validation"
	assert.Contains(t, err.Error(), "failed")
	assert.Contains(t, err.Error(), "alphanum")
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
