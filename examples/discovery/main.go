// Package main demonstrates the discovery pattern using ResolveAll.
//
// This pattern allows for "plugin" style architectures where multiple implementations
// of an interface are registered independently, and a consumer can discover and use
// all of them without knowing their concrete types.
//
// Run with: go run .
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/petabytecl/gaz"
)

// Plugin is the common interface that our plugins will implement.
type Plugin interface {
	Name() string
	Init(ctx context.Context) error
}

// --- Auth Plugin ---

type AuthPlugin struct{}

func NewAuthPlugin() *AuthPlugin {
	return &AuthPlugin{}
}

func (p *AuthPlugin) Name() string {
	return "AuthPlugin"
}

func (p *AuthPlugin) Init(ctx context.Context) error {
	fmt.Println("Initializing AuthPlugin...")
	return nil
}

// --- Logger Plugin ---

type LoggerPlugin struct{}

func NewLoggerPlugin() *LoggerPlugin {
	return &LoggerPlugin{}
}

func (p *LoggerPlugin) Name() string {
	return "LoggerPlugin"
}

func (p *LoggerPlugin) Init(ctx context.Context) error {
	fmt.Println("Initializing LoggerPlugin...")
	return nil
}

// --- Metrics Plugin ---

type MetricsPlugin struct{}

func NewMetricsPlugin() *MetricsPlugin {
	return &MetricsPlugin{}
}

func (p *MetricsPlugin) Name() string {
	return "MetricsPlugin"
}

func (p *MetricsPlugin) Init(ctx context.Context) error {
	fmt.Println("Initializing MetricsPlugin...")
	return nil
}

// --- Plugin Manager ---

// PluginManager is responsible for discovering and managing plugins.
type PluginManager struct {
	plugins []Plugin
}

// NewPluginManager discovers all registered Plugins in the container.
// It uses gaz.ResolveAll to find everything that implements the Plugin interface.
func NewPluginManager(c *gaz.Container) (*PluginManager, error) {
	// Discovery happens here!
	plugins, err := gaz.ResolveAll[Plugin](c)
	if err != nil {
		return nil, fmt.Errorf("failed to discover plugins: %w", err)
	}

	return &PluginManager{
		plugins: plugins,
	}, nil
}

func (pm *PluginManager) Run(ctx context.Context) error {
	fmt.Printf("PluginManager found %d plugins:\n", len(pm.plugins))

	for _, p := range pm.plugins {
		fmt.Printf("- Found: %s\n", p.Name())
		if err := p.Init(ctx); err != nil {
			return fmt.Errorf("plugin %s init failed: %w", p.Name(), err)
		}
	}
	return nil
}

func run() error {
	app := gaz.New()

	// Register plugins as their concrete types.
	// Note: We don't register them as 'Plugin', but as '*AuthPlugin', etc.
	// ResolveAll[Plugin] will find them because they implement the interface.

	// Register AuthPlugin
	app.Module("auth", func(c *gaz.Container) error {
		return gaz.For[*AuthPlugin](c).Provider(func(_ *gaz.Container) (*AuthPlugin, error) {
			return NewAuthPlugin(), nil
		})
	})

	// Register LoggerPlugin
	app.Module("logging", func(c *gaz.Container) error {
		return gaz.For[*LoggerPlugin](c).Provider(func(_ *gaz.Container) (*LoggerPlugin, error) {
			return NewLoggerPlugin(), nil
		})
	})

	// Register MetricsPlugin
	app.Module("metrics", func(c *gaz.Container) error {
		return gaz.For[*MetricsPlugin](c).Provider(func(_ *gaz.Container) (*MetricsPlugin, error) {
			return NewMetricsPlugin(), nil
		})
	})

	// Register PluginManager which consumes the plugins
	app.Module("core", func(c *gaz.Container) error {
		return gaz.For[*PluginManager](c).Provider(NewPluginManager)
	})

	// Build the container
	if err := app.Build(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Run the manager
	manager, err := gaz.Resolve[*PluginManager](app.Container())
	if err != nil {
		return fmt.Errorf("resolve manager failed: %w", err)
	}

	return manager.Run(context.Background())
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
