package configuredmodule

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/config"
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

func TestProvideValidatedKeepsDefaultsWhenProviderNamespaceMissing(t *testing.T) {
	app := gaz.New().WithConfig(nil, config.WithBackend(config.NewMapBackend(nil)))
	defaults := sampleValidatedConfig{Endpoint: "https://default.example.test"}
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
	require.Equal(t, "https://default.example.test", cfg.Endpoint)
}

func TestProvideValidatedRejectsMalformedProviderOverlay(t *testing.T) {
	app := gaz.New().WithConfig(nil, config.WithBackend(config.NewMapBackend(map[string]any{
		"validated.endpoint": map[string]any{"not": "a string"},
	})))
	defaults := sampleValidatedConfig{Endpoint: "https://default.example.test"}
	app.Use(gaz.NewModule("validated").
		Provide(ProvideValidated[sampleValidatedConfig, *sampleValidatedConfig](
			&defaults,
			"validated config",
			"validated config validate",
		)).
		Build())

	require.NoError(t, app.Build())

	_, err := gaz.Resolve[sampleValidatedConfig](app.Container())
	require.Error(t, err)
	require.ErrorContains(t, err, "load provider config \"validated\"")
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

func TestProvidersRejectNilDefaults(t *testing.T) {
	tests := []struct {
		name    string
		provide func() func(*gaz.Container) error
	}{
		{
			name: "validated",
			provide: func() func(*gaz.Container) error {
				return ProvideValidated[sampleValidatedConfig, *sampleValidatedConfig](
					nil,
					"validated config",
					"validated config validate",
				)
			},
		},
		{
			name: "defaulted",
			provide: func() func(*gaz.Container) error {
				return ProvideDefaulted[sampleDefaultedConfig, *sampleDefaultedConfig](
					nil,
					"defaulted config",
					"validate defaulted config",
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := gaz.New()
			app.Use(gaz.NewModule(tt.name).
				Provide(tt.provide()).
				Build())

			err := app.Build()
			require.Error(t, err)
			require.ErrorContains(t, err, "defaults must not be nil")
		})
	}
}
