// Package module provides the gaz.Module for eventbus integration.
package module

import (
	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/eventbus"
)

// New creates an eventbus module that provides eventbus.EventBus.
// This module registers the in-process pub/sub infrastructure.
//
// Usage:
//
//	import eventbusmod "github.com/petabytecl/gaz/eventbus/module"
//
//	app := gaz.New(gaz.WithCobra(rootCmd))
//	app.Use(eventbusmod.New())
//
// The module provides:
//   - *eventbus.EventBus for in-process pub/sub messaging
//
//nolint:ireturn // Module is the expected return type for gaz modules
func New() gaz.Module {
	return gaz.NewModule("eventbus").
		Provide(eventbus.Module).
		Build()
}
