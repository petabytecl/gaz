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
