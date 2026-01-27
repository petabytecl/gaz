package gaz_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/petabytecl/gaz"
)

// ValidationSuite tests the validation engine functionality.
type ValidationSuite struct {
	suite.Suite
}

func TestValidationSuite(t *testing.T) {
	suite.Run(t, new(ValidationSuite))
}

// TestRequiredValidation tests that required validation tag rejects empty strings.
func (s *ValidationSuite) TestRequiredValidation() {
	type RequiredConfig struct {
		Host string `mapstructure:"host" validate:"required"`
	}

	// Empty value should fail
	var emptyConfig RequiredConfig
	app := gaz.New().WithConfig(&emptyConfig)
	err := app.Build()

	s.Require().Error(err)
	s.Require().ErrorIs(err, gaz.ErrConfigValidation)
	s.Contains(err.Error(), "host")
	s.Contains(err.Error(), "required field cannot be empty")

	// Non-empty value should pass
	nonEmptyConfig := RequiredConfig{Host: "localhost"}
	app2 := gaz.New().WithConfig(&nonEmptyConfig)
	err = app2.Build()

	s.Require().NoError(err)
	s.Equal("localhost", nonEmptyConfig.Host)
}

// TestMinMaxValidation tests that min/max validation tags enforce numeric constraints.
func (s *ValidationSuite) TestMinMaxValidation() {
	type PortConfig struct {
		Port int `mapstructure:"port" validate:"min=1,max=65535"`
	}

	// Value 0 fails min=1
	zeroPort := PortConfig{Port: 0}
	app := gaz.New().WithConfig(&zeroPort)
	err := app.Build()

	s.Require().Error(err)
	s.Require().ErrorIs(err, gaz.ErrConfigValidation)
	s.Contains(err.Error(), "must be at least 1")

	// Value 70000 fails max=65535
	highPort := PortConfig{Port: 70000}
	app2 := gaz.New().WithConfig(&highPort)
	err = app2.Build()

	s.Require().Error(err)
	s.Require().ErrorIs(err, gaz.ErrConfigValidation)
	s.Contains(err.Error(), "must be at most 65535")

	// Value 8080 passes
	validPort := PortConfig{Port: 8080}
	app3 := gaz.New().WithConfig(&validPort)
	err = app3.Build()

	s.Require().NoError(err)
	s.Equal(8080, validPort.Port)
}

// TestOneOfValidation tests that oneof validation tag restricts to allowed values.
func (s *ValidationSuite) TestOneOfValidation() {
	type LogConfig struct {
		Level string `mapstructure:"level" validate:"oneof=debug info warn error"`
	}

	// Invalid value fails
	invalidLevel := LogConfig{Level: "invalid"}
	app := gaz.New().WithConfig(&invalidLevel)
	err := app.Build()

	s.Require().Error(err)
	s.Require().ErrorIs(err, gaz.ErrConfigValidation)
	s.Contains(err.Error(), "must be one of: debug info warn error")

	// Valid value passes
	validLevel := LogConfig{Level: "info"}
	app2 := gaz.New().WithConfig(&validLevel)
	err = app2.Build()

	s.Require().NoError(err)
	s.Equal("info", validLevel.Level)
}

// TestNestedStructValidation tests that nested struct validation works recursively.
func (s *ValidationSuite) TestNestedStructValidation() {
	type DatabaseConfig struct {
		Host string `mapstructure:"host" validate:"required"`
		Port int    `mapstructure:"port" validate:"required,min=1"`
	}

	type AppConfig struct {
		Database DatabaseConfig `mapstructure:"database"`
	}

	// Missing nested field fails with proper namespace path
	emptyNested := AppConfig{}
	app := gaz.New().WithConfig(&emptyNested)
	err := app.Build()

	s.Require().Error(err)
	s.Require().ErrorIs(err, gaz.ErrConfigValidation)
	// Error should show "database.host" namespace
	s.Contains(err.Error(), "database.host")
	s.Contains(err.Error(), "required field cannot be empty")

	// Fully populated nested struct passes
	validNested := AppConfig{
		Database: DatabaseConfig{
			Host: "localhost",
			Port: 5432,
		},
	}
	app2 := gaz.New().WithConfig(&validNested)
	err = app2.Build()

	s.Require().NoError(err)
	s.Equal("localhost", validNested.Database.Host)
	s.Equal(5432, validNested.Database.Port)
}

// TestMapstructureFieldNames tests that error messages show mapstructure field names, not Go field names.
func (s *ValidationSuite) TestMapstructureFieldNames() {
	type DBConfig struct {
		DBHost string `mapstructure:"db_host" validate:"required"`
	}

	// Error message should show "db_host" not "DBHost"
	emptyHost := DBConfig{}
	app := gaz.New().WithConfig(&emptyHost)
	err := app.Build()

	s.Require().Error(err)
	s.Require().ErrorIs(err, gaz.ErrConfigValidation)
	// Error should use mapstructure name
	s.Contains(err.Error(), "db_host")
	// Error should NOT contain Go field name
	s.NotContains(err.Error(), "DBHost")
}

// TestAllErrorsCollected tests that all validation errors are collected and shown together.
func (s *ValidationSuite) TestAllErrorsCollected() {
	type MultiErrorConfig struct {
		Host string `mapstructure:"host" validate:"required"`
		Port int    `mapstructure:"port" validate:"required,min=1"`
		Name string `mapstructure:"name" validate:"required"`
	}

	// Config with multiple invalid fields
	invalidConfig := MultiErrorConfig{}
	app := gaz.New().WithConfig(&invalidConfig)
	err := app.Build()

	s.Require().Error(err)
	s.Require().ErrorIs(err, gaz.ErrConfigValidation)

	errStr := err.Error()

	// All errors should be reported
	s.Contains(errStr, "host")
	s.Contains(errStr, "port")
	s.Contains(errStr, "name")

	// Multiple lines expected (newline separated)
	lines := strings.Split(errStr, "\n")
	s.GreaterOrEqual(len(lines), 3, "expected multiple error lines")
}

// TestRequiredIfValidation tests cross-field validation with required_if tag.
func (s *ValidationSuite) TestRequiredIfValidation() {
	type AuthConfig struct {
		Type     string `mapstructure:"type"     validate:"required,oneof=none basic oauth"`
		Username string `mapstructure:"username" validate:"required_if=Type basic"`
		Password string `mapstructure:"password" validate:"required_if=Type basic"`
		Token    string `mapstructure:"token"    validate:"required_if=Type oauth"`
	}

	// Type="basic" without username/password fails
	basicNoAuth := AuthConfig{Type: "basic"}
	app := gaz.New().WithConfig(&basicNoAuth)
	err := app.Build()

	s.Require().Error(err)
	s.Require().ErrorIs(err, gaz.ErrConfigValidation)
	s.Contains(err.Error(), "username")
	s.Contains(err.Error(), "password")

	// Type="basic" with username/password passes
	basicWithAuth := AuthConfig{
		Type:     "basic",
		Username: "admin",
		Password: "secret",
	}
	app2 := gaz.New().WithConfig(&basicWithAuth)
	err = app2.Build()

	s.Require().NoError(err)

	// Type="oauth" without token fails
	oauthNoToken := AuthConfig{Type: "oauth"}
	app3 := gaz.New().WithConfig(&oauthNoToken)
	err = app3.Build()

	s.Require().Error(err)
	s.Require().ErrorIs(err, gaz.ErrConfigValidation)
	s.Contains(err.Error(), "token")

	// Type="oauth" with token passes
	oauthWithToken := AuthConfig{
		Type:  "oauth",
		Token: "abc123",
	}
	app4 := gaz.New().WithConfig(&oauthWithToken)
	err = app4.Build()

	s.Require().NoError(err)

	// Type="none" passes without any auth fields
	noneAuth := AuthConfig{Type: "none"}
	app5 := gaz.New().WithConfig(&noneAuth)
	err = app5.Build()

	s.Require().NoError(err)
}

// TestConfigManagerValidation tests validation via ConfigManager.Load().
func (s *ValidationSuite) TestConfigManagerValidation() {
	type ServerConfig struct {
		Host string `mapstructure:"host" validate:"required"`
		Port int    `mapstructure:"port" validate:"min=1,max=65535"`
	}

	// Use ConfigManager.Load() with validation tags
	emptyConfig := ServerConfig{}
	app := gaz.New().WithConfig(&emptyConfig)
	err := app.Build()

	// Validation runs and fails
	s.Require().Error(err)
	s.Require().ErrorIs(err, gaz.ErrConfigValidation)
}

// TestValidationAfterDefaults tests that defaults are applied before validation.
func (s *ValidationSuite) TestValidationAfterDefaults() {
	// Config struct with Default() that sets required values
	defaulted := &defaulterConfig{}
	app := gaz.New().WithConfig(defaulted)
	err := app.Build()

	// Without defaults: would fail required validation
	// With defaults applied first: passes validation
	s.Require().NoError(err)
	s.Equal("localhost", defaulted.Host)
	s.Equal(8080, defaulted.Port)
}

// defaulterConfig implements Defaulter interface.
type defaulterConfig struct {
	Host string `mapstructure:"host" validate:"required"`
	Port int    `mapstructure:"port" validate:"required,min=1"`
}

func (c *defaulterConfig) Default() {
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port == 0 {
		c.Port = 8080
	}
}

// TestValidationBeforeCustomValidate tests that tag validation runs before Validate() method.
func (s *ValidationSuite) TestValidationBeforeCustomValidate() {
	// Config struct with both validate tags AND Validate() method
	// Tag validation runs first. If tags fail, Validate() method is NOT called.
	invalidConfig := &validatorConfig{}
	app := gaz.New().WithConfig(invalidConfig)
	err := app.Build()

	// Tag validation fails first (required field empty)
	s.Require().Error(err)
	s.Require().ErrorIs(err, gaz.ErrConfigValidation)
	// Custom Validate() should NOT have been called
	s.False(invalidConfig.validateCalled)
}

// TestValidationPassesThenCustomValidate tests Validate() is called when tags pass.
func (s *ValidationSuite) TestValidationPassesThenCustomValidate() {
	// If tags pass, Validate() method IS called
	validConfig := &validatorConfig{Host: "localhost", Port: 8080}
	app := gaz.New().WithConfig(validConfig)
	err := app.Build()

	// Tags pass, custom Validate() is called and passes
	s.Require().NoError(err)
	s.True(validConfig.validateCalled)
}

// validatorConfig implements both tag validation and Validate() method.
type validatorConfig struct {
	Host           string `mapstructure:"host" validate:"required"`
	Port           int    `mapstructure:"port" validate:"min=1"`
	validateCalled bool
}

func (c *validatorConfig) Validate() error {
	c.validateCalled = true
	return nil
}

// TestErrorMessageFormat tests the error message format.
func (s *ValidationSuite) TestErrorMessageFormat() {
	type FormatConfig struct {
		Host string `mapstructure:"host" validate:"required"`
	}

	emptyConfig := FormatConfig{}
	app := gaz.New().WithConfig(&emptyConfig)
	err := app.Build()

	s.Require().Error(err)

	errStr := err.Error()

	// Verify format: {namespace}: {message} (validate:"{tag}")
	// Should contain the namespace path
	s.Contains(errStr, "host")
	// Should contain humanized message
	s.Contains(errStr, "required field cannot be empty")
	// Should contain the tag reference
	s.Contains(errStr, `validate:"required"`)
}
