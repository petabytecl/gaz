package gaz

import (
	"fmt"
	"time"

	"github.com/petabytecl/gaz/config"
)

// ConfigFlagType represents the type of a configuration flag value.
// The framework uses this to parse and validate config values correctly.
type ConfigFlagType string

const (
	// ConfigFlagTypeString represents a string configuration value.
	ConfigFlagTypeString ConfigFlagType = "string"

	// ConfigFlagTypeInt represents an integer configuration value.
	ConfigFlagTypeInt ConfigFlagType = "int"

	// ConfigFlagTypeBool represents a boolean configuration value.
	ConfigFlagTypeBool ConfigFlagType = "bool"

	// ConfigFlagTypeDuration represents a time.Duration configuration value.
	// Values are parsed using time.ParseDuration (e.g., "30s", "5m", "1h").
	ConfigFlagTypeDuration ConfigFlagType = "duration"

	// ConfigFlagTypeFloat represents a float64 configuration value.
	ConfigFlagTypeFloat ConfigFlagType = "float"
)

// ConfigFlag defines a configuration key that a provider needs.
// Providers return a slice of ConfigFlag from their ConfigFlags() method
// to declare their configuration requirements.
//
// Example:
//
//	func (r *RedisProvider) ConfigFlags() []gaz.ConfigFlag {
//	    return []gaz.ConfigFlag{
//	        {Key: "host", Type: gaz.ConfigFlagTypeString, Default: "localhost", Description: "Redis server host"},
//	        {Key: "port", Type: gaz.ConfigFlagTypeInt, Default: 6379, Description: "Redis server port"},
//	        {Key: "password", Type: gaz.ConfigFlagTypeString, Required: true, Description: "Redis password"},
//	    }
//	}
type ConfigFlag struct {
	// Key is the config key relative to the provider's namespace.
	// For example, if a provider declares namespace "redis" and key "host",
	// the full config key becomes "redis.host".
	Key string

	// Type specifies how the config value should be parsed.
	// String values are used as-is, while int, bool, duration, and float
	// values are parsed from their string representation.
	Type ConfigFlagType

	// Default is the default value to use if the config key is not set.
	// Set to nil for no default. The type should match the Type field
	// (e.g., int for ConfigFlagTypeInt, time.Duration for ConfigFlagTypeDuration).
	Default any

	// Required indicates whether this config key must be set.
	// If true and the key is not set (via env, file, or flag),
	// the application will fail to start during Build().
	Required bool

	// Description provides help text for this config key.
	// Used in --help output and documentation generation.
	Description string
}

// ConfigProvider is implemented by providers that need configuration.
// When a provider implements this interface, the framework will:
//
//  1. Call ConfigNamespace() to get the prefix for all config keys
//  2. Call ConfigFlags() to collect configuration requirements
//  3. Auto-prefix each key with the namespace (e.g., "redis" + "host" = "redis.host")
//  4. Translate keys for environment variables (e.g., "redis.host" → "REDIS_HOST")
//  5. Validate required flags are set during Build()
//
// Providers define their config needs but do not receive values directly.
// Config values are accessible via ProviderValues, which is injectable.
//
// Example:
//
//	type RedisProvider struct{}
//
//	func (r *RedisProvider) ConfigNamespace() string {
//	    return "redis"
//	}
//
//	func (r *RedisProvider) ConfigFlags() []gaz.ConfigFlag {
//	    return []gaz.ConfigFlag{
//	        {Key: "host", Type: gaz.ConfigFlagTypeString, Default: "localhost", Description: "Redis server host"},
//	        {Key: "port", Type: gaz.ConfigFlagTypeInt, Default: 6379, Description: "Redis server port"},
//	    }
//	}
type ConfigProvider interface {
	// ConfigNamespace returns the namespace prefix for this provider's config keys.
	// All keys returned by ConfigFlags() are automatically prefixed with this namespace.
	// For example, if namespace is "redis" and a key is "host", the full key is "redis.host".
	ConfigNamespace() string

	// ConfigFlags returns the configuration flags this provider needs.
	// Each flag describes a config key with its type, default value, and whether it's required.
	ConfigFlags() []ConfigFlag
}

// ProviderValues provides access to provider-registered configuration values.
// Inject this via the DI container to retrieve config values by their full key.
//
// Example:
//
//	func NewRedisClient(c *gaz.Container) (*RedisClient, error) {
//	    pv := gaz.MustResolve[*gaz.ProviderValues](c)
//	    host := pv.GetString("redis.host")
//	    port := pv.GetInt("redis.port")
//	    return &RedisClient{Host: host, Port: port}, nil
//	}
type ProviderValues struct {
	backend config.Backend
}

// GetString returns a string config value by its full key (e.g., "redis.host").
func (pv *ProviderValues) GetString(key string) string {
	return pv.backend.GetString(key)
}

// GetInt returns an int config value by its full key.
func (pv *ProviderValues) GetInt(key string) int {
	return pv.backend.GetInt(key)
}

// GetBool returns a bool config value by its full key.
func (pv *ProviderValues) GetBool(key string) bool {
	return pv.backend.GetBool(key)
}

// GetDuration returns a duration config value by its full key.
func (pv *ProviderValues) GetDuration(key string) time.Duration {
	return pv.backend.GetDuration(key)
}

// GetFloat64 returns a float64 config value by its full key.
func (pv *ProviderValues) GetFloat64(key string) float64 {
	return pv.backend.GetFloat64(key)
}

// gazUnmarshaler is implemented by backends that support gaz struct tags.
type gazUnmarshaler interface {
	UnmarshalWithGazTag(target any) error
	UnmarshalKeyWithGazTag(key string, target any) error
	HasKey(key string) bool
}

// Unmarshal unmarshals the entire config into target struct.
// Uses "gaz" struct tags for field mapping.
//
// Example:
//
//	type AppConfig struct {
//	    Redis RedisConfig `gaz:"redis"`
//	    DB    DBConfig    `gaz:"database"`
//	}
//
//	var cfg AppConfig
//	if err := pv.Unmarshal(&cfg); err != nil {
//	    return err
//	}
func (pv *ProviderValues) Unmarshal(target any) error {
	if gu, ok := pv.backend.(gazUnmarshaler); ok {
		return gu.UnmarshalWithGazTag(target)
	}
	// Fallback: use standard Unmarshal (uses mapstructure tag)
	return pv.backend.Unmarshal(target)
}

// UnmarshalKey unmarshals config at the given key/namespace into target struct.
// Uses "gaz" struct tags for field mapping.
// Returns config.ErrKeyNotFound if the key/namespace doesn't exist.
//
// Example:
//
//	type RedisConfig struct {
//	    Host string `gaz:"host"`
//	    Port int    `gaz:"port"`
//	}
//
//	var cfg RedisConfig
//	if err := pv.UnmarshalKey("redis", &cfg); err != nil {
//	    if errors.Is(err, config.ErrKeyNotFound) {
//	        // Handle missing namespace
//	    }
//	    return err
//	}
func (pv *ProviderValues) UnmarshalKey(key string, target any) error {
	if gu, ok := pv.backend.(gazUnmarshaler); ok {
		if !gu.HasKey(key) {
			return fmt.Errorf("%w: %s", config.ErrKeyNotFound, key)
		}
		return gu.UnmarshalKeyWithGazTag(key, target)
	}
	// Fallback: use standard UnmarshalKey (uses mapstructure tag)
	// Note: no HasKey check in fallback since Backend doesn't have it
	return pv.backend.UnmarshalKey(key, target)
}
