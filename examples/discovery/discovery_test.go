package main

import (
	"testing"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/gaztest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscovery(t *testing.T) {
	t.Run("ResolveAll finds all implementations", func(t *testing.T) {
		app, err := gaztest.New(t).
			WithModules(di.NewModuleFunc("test-module", func(c *gaz.Container) error {
				// Register AuthPlugin
				if err := gaz.For[*AuthPlugin](c).Provider(func(_ *gaz.Container) (*AuthPlugin, error) {
					return NewAuthPlugin(), nil
				}); err != nil {
					return err
				}

				// Register LoggerPlugin
				if err := gaz.For[*LoggerPlugin](c).Provider(func(_ *gaz.Container) (*LoggerPlugin, error) {
					return NewLoggerPlugin(), nil
				}); err != nil {
					return err
				}

				return nil
			})).
			Build()
		require.NoError(t, err)

		// Verify ResolveAll finds both as Plugin
		plugins, err := gaz.ResolveAll[Plugin](app.Container())
		require.NoError(t, err)
		assert.Len(t, plugins, 2)

		// Verify names
		names := make(map[string]bool)
		for _, p := range plugins {
			names[p.Name()] = true
		}
		assert.True(t, names["AuthPlugin"])
		assert.True(t, names["LoggerPlugin"])
	})

	t.Run("ResolveAll finds nothing if none registered", func(t *testing.T) {
		app, err := gaztest.New(t).Build()
		require.NoError(t, err)

		plugins, err := gaz.ResolveAll[Plugin](app.Container())
		require.NoError(t, err)
		assert.Len(t, plugins, 0)
	})
}

func TestResolveGroup(t *testing.T) {
	t.Run("ResolveGroup filters by group name", func(t *testing.T) {
		app, err := gaztest.New(t).
			WithModules(di.NewModuleFunc("test-module", func(c *gaz.Container) error {
				// Register AuthPlugin in "system" group
				if err := gaz.For[*AuthPlugin](c).InGroup("system").Provider(func(_ *gaz.Container) (*AuthPlugin, error) {
					return NewAuthPlugin(), nil
				}); err != nil {
					return err
				}

				// Register LoggerPlugin in "system" group
				if err := gaz.For[*LoggerPlugin](c).InGroup("system").Provider(func(_ *gaz.Container) (*LoggerPlugin, error) {
					return NewLoggerPlugin(), nil
				}); err != nil {
					return err
				}

				// Register MetricsPlugin in "user" group
				if err := gaz.For[*MetricsPlugin](c).InGroup("user").Provider(func(_ *gaz.Container) (*MetricsPlugin, error) {
					return NewMetricsPlugin(), nil
				}); err != nil {
					return err
				}

				return nil
			})).
			Build()
		require.NoError(t, err)

		// Resolve "system" group plugins
		systemPlugins, err := gaz.ResolveGroup[Plugin](app.Container(), "system")
		require.NoError(t, err)
		assert.Len(t, systemPlugins, 2)

		names := make(map[string]bool)
		for _, p := range systemPlugins {
			names[p.Name()] = true
		}
		assert.True(t, names["AuthPlugin"])
		assert.True(t, names["LoggerPlugin"])
		assert.False(t, names["MetricsPlugin"])

		// Resolve "user" group plugins
		userPlugins, err := gaz.ResolveGroup[Plugin](app.Container(), "user")
		require.NoError(t, err)
		assert.Len(t, userPlugins, 1)
		assert.Equal(t, "MetricsPlugin", userPlugins[0].Name())
	})
}
