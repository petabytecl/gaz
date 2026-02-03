package main

import (
	"context"
	"testing"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/gaztest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlugins(t *testing.T) {
	t.Run("AuthPlugin", func(t *testing.T) {
		p := NewAuthPlugin()
		assert.Equal(t, "AuthPlugin", p.Name())
		assert.NoError(t, p.Init(context.Background()))
	})

	t.Run("LoggerPlugin", func(t *testing.T) {
		p := NewLoggerPlugin()
		assert.Equal(t, "LoggerPlugin", p.Name())
		assert.NoError(t, p.Init(context.Background()))
	})

	t.Run("MetricsPlugin", func(t *testing.T) {
		p := NewMetricsPlugin()
		assert.Equal(t, "MetricsPlugin", p.Name())
		assert.NoError(t, p.Init(context.Background()))
	})
}

func TestPluginManager(t *testing.T) {
	// Setup app with all plugins
	app, err := gaztest.New(t).
		WithModules(
			di.NewModuleFunc("auth", func(c *gaz.Container) error {
				return gaz.For[*AuthPlugin](c).Provider(func(_ *gaz.Container) (*AuthPlugin, error) {
					return NewAuthPlugin(), nil
				})
			}),
			di.NewModuleFunc("logging", func(c *gaz.Container) error {
				return gaz.For[*LoggerPlugin](c).Provider(func(_ *gaz.Container) (*LoggerPlugin, error) {
					return NewLoggerPlugin(), nil
				})
			}),
			di.NewModuleFunc("metrics", func(c *gaz.Container) error {
				return gaz.For[*MetricsPlugin](c).Provider(func(_ *gaz.Container) (*MetricsPlugin, error) {
					return NewMetricsPlugin(), nil
				})
			}),
			di.NewModuleFunc("core", func(c *gaz.Container) error {
				return gaz.For[*PluginManager](c).Provider(NewPluginManager)
			}),
		).
		Build()
	require.NoError(t, err)

	manager := gaztest.RequireResolve[*PluginManager](t, app)
	require.NotNil(t, manager)
	assert.Len(t, manager.plugins, 3)

	err = manager.Run(context.Background())
	assert.NoError(t, err)
}

func TestRun(t *testing.T) {
	// Smoke test the main run function
	err := run()
	assert.NoError(t, err)
}
