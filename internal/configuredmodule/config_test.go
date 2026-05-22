package configuredmodule

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz"
)

const (
	defaultName     = "from-default"
	loadedName      = "from-provider-values"
	mutatedEndpoint = "https://mutated.example.test"
	requiredName    = "configured"
)

type sampleDefaultedConfig struct {
	Name      string `mapstructure:"name"`
	Required  string `mapstructure:"required"`
	Defaulted string `mapstructure:"defaulted"`
}

func (c *sampleDefaultedConfig) Namespace() string {
	return "sample"
}

func (c *sampleDefaultedConfig) SetDefaults() {
	if c.Defaulted == "" {
		c.Defaulted = defaultName
	}
}

func (c *sampleDefaultedConfig) Validate() error {
	if c.Required == "" {
		return errors.New("required is missing")
	}
	return nil
}

type sampleValidatedConfig struct {
	Endpoint string `mapstructure:"endpoint"`
}

func (c *sampleValidatedConfig) Namespace() string {
	return "validated"
}

func (c *sampleValidatedConfig) Validate() error {
	if c.Endpoint == "" {
		return errors.New("endpoint is missing")
	}
	return nil
}

func TestProvideDefaultedLoadsProviderValuesAndAppliesDefaults(t *testing.T) {
	app := gaz.New()
	require.NoError(t, app.MergeConfigMap(map[string]any{
		"sample": map[string]any{
			"name":     loadedName,
			"required": requiredName,
		},
	}))
	defaults := sampleDefaultedConfig{Name: defaultName}
	app.Use(gaz.NewModule("sample").
		Provide(ProvideDefaulted[sampleDefaultedConfig, *sampleDefaultedConfig](
			&defaults,
			"sample config",
			"validate sample config",
		)).
		Build())

	require.NoError(t, app.Build())

	cfg, err := gaz.Resolve[sampleDefaultedConfig](app.Container())
	require.NoError(t, err)
	require.Equal(t, loadedName, cfg.Name)
	require.Equal(t, requiredName, cfg.Required)
	require.Equal(t, defaultName, cfg.Defaulted)
}

func TestProvideDefaultedWrapsValidationContext(t *testing.T) {
	app := gaz.New()
	defaults := sampleDefaultedConfig{}
	app.Use(gaz.NewModule("sample").
		Provide(ProvideDefaulted[sampleDefaultedConfig, *sampleDefaultedConfig](
			&defaults,
			"sample config",
			"validate sample config",
		)).
		Build())

	require.NoError(t, app.Build())

	_, err := gaz.Resolve[sampleDefaultedConfig](app.Container())
	require.Error(t, err)
	require.ErrorContains(t, err, "validate sample config")
}

func TestProvideValidatedLoadsProviderValues(t *testing.T) {
	app := gaz.New()
	require.NoError(t, app.MergeConfigMap(map[string]any{
		"validated": map[string]any{
			"endpoint": "https://example.test",
		},
	}))
	defaults := sampleValidatedConfig{}
	app.Use(gaz.NewModule("validated").
		Provide(ProvideValidated[sampleValidatedConfig, *sampleValidatedConfig](
			&defaults,
			"validated config",
			"validated config validate",
		)).
		Build())

	require.NoError(t, app.Build())

	cfg, err := gaz.Resolve[sampleValidatedConfig](app.Container())
	require.NoError(t, err)
	require.Equal(t, "https://example.test", cfg.Endpoint)
}

func TestProviderUsesLatestDefaultValues(t *testing.T) {
	app := gaz.New()
	defaults := sampleValidatedConfig{Endpoint: "https://initial.example.test"}
	app.Use(gaz.NewModule("validated").
		Provide(ProvideValidated[sampleValidatedConfig, *sampleValidatedConfig](
			&defaults,
			"validated config",
			"validated config validate",
		)).
		Build())

	defaults.Endpoint = mutatedEndpoint

	require.NoError(t, app.Build())

	cfg, err := gaz.Resolve[sampleValidatedConfig](app.Container())
	require.NoError(t, err)
	require.Equal(t, mutatedEndpoint, cfg.Endpoint)
}
