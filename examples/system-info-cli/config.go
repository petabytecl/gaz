// Package main demonstrates the ConfigProvider pattern with system information collection.
//
// This file implements the ConfigProvider interface for the system-info-cli example.
// It declares configuration flags for refresh interval, output format, and one-shot mode.
package main

import (
	"time"

	"github.com/petabytecl/gaz"
)

// SystemInfoConfig implements ConfigProvider to declare configuration requirements.
// It stores the injected ProviderValues to provide typed accessor methods.
type SystemInfoConfig struct {
	pv *gaz.ProviderValues
}

// ConfigNamespace returns the namespace prefix for all config keys.
// Keys returned by ConfigFlags() are prefixed with this namespace.
// For example: namespace "sysinfo" + key "refresh" = "sysinfo.refresh"
func (c *SystemInfoConfig) ConfigNamespace() string {
	return "sysinfo"
}

// ConfigFlags declares the configuration flags this provider needs.
// Each flag specifies a key, type, default value, and description.
func (c *SystemInfoConfig) ConfigFlags() []gaz.ConfigFlag {
	return []gaz.ConfigFlag{
		{Key: "refresh", Type: gaz.ConfigFlagTypeDuration, Default: 5 * time.Second, Description: "Refresh interval"},
		{Key: "format", Type: gaz.ConfigFlagTypeString, Default: "text", Description: "Output format (text, json)"},
		{Key: "once", Type: gaz.ConfigFlagTypeBool, Default: false, Description: "Run once and exit"},
	}
}

// NewSystemInfoConfig injects ProviderValues during Build().
// This is possible because ProviderValues is registered BEFORE providers run.
// The provider can resolve and store ProviderValues at construction time.
func NewSystemInfoConfig(c *gaz.Container) (*SystemInfoConfig, error) {
	pv, err := gaz.Resolve[*gaz.ProviderValues](c)
	if err != nil {
		return nil, err
	}
	return &SystemInfoConfig{pv: pv}, nil
}

// RefreshInterval returns the refresh interval configuration value.
func (c *SystemInfoConfig) RefreshInterval() time.Duration {
	return c.pv.GetDuration("sysinfo.refresh")
}

// Format returns the output format configuration value.
func (c *SystemInfoConfig) Format() string {
	return c.pv.GetString("sysinfo.format")
}

// Once returns whether to run once and exit.
func (c *SystemInfoConfig) Once() bool {
	return c.pv.GetBool("sysinfo.once")
}
