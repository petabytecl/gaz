package gaz_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/config"
)

type ProviderConfigSuite struct {
	suite.Suite
}

func TestProviderConfigSuite(t *testing.T) {
	suite.Run(t, new(ProviderConfigSuite))
}

// =============================================================================
// Test providers
// =============================================================================

// RedisProvider is a test provider implementing ConfigProvider.
type RedisProvider struct{}

func (r *RedisProvider) ConfigNamespace() string {
	return "redis"
}

func (r *RedisProvider) ConfigFlags() []gaz.ConfigFlag {
	return []gaz.ConfigFlag{
		{
			Key:         "host",
			Type:        gaz.ConfigFlagTypeString,
			Default:     "localhost",
			Description: "Redis server host",
		},
		{Key: "port", Type: gaz.ConfigFlagTypeInt, Default: 6379, Description: "Redis server port"},
	}
}

// CacheProvider is another test provider to test multiple providers.
type CacheProvider struct{}

func (c *CacheProvider) ConfigNamespace() string {
	return "cache"
}

func (c *CacheProvider) ConfigFlags() []gaz.ConfigFlag {
	return []gaz.ConfigFlag{
		{
			Key:         "ttl",
			Type:        gaz.ConfigFlagTypeDuration,
			Default:     time.Minute * 5,
			Description: "Cache TTL",
		},
		{
			Key:         "enabled",
			Type:        gaz.ConfigFlagTypeBool,
			Default:     true,
			Description: "Enable caching",
		},
	}
}

// RequiredConfigProvider has a required config field.
type RequiredConfigProvider struct{}

func (p *RequiredConfigProvider) ConfigNamespace() string {
	return "required"
}

func (p *RequiredConfigProvider) ConfigFlags() []gaz.ConfigFlag {
	return []gaz.ConfigFlag{
		{Key: "api_key", Type: gaz.ConfigFlagTypeString, Required: true, Description: "API key"},
	}
}

// CollidingProvider1 registers cache.host to test collision detection.
type CollidingProvider1 struct{}

func (p *CollidingProvider1) ConfigNamespace() string {
	return "cache"
}

func (p *CollidingProvider1) ConfigFlags() []gaz.ConfigFlag {
	return []gaz.ConfigFlag{
		{Key: "host", Type: gaz.ConfigFlagTypeString, Default: "provider1", Description: "Host"},
	}
}

// CollidingProvider2 also registers cache.host to test collision detection.
type CollidingProvider2 struct{}

func (p *CollidingProvider2) ConfigNamespace() string {
	return "cache"
}

func (p *CollidingProvider2) ConfigFlags() []gaz.ConfigFlag {
	return []gaz.ConfigFlag{
		{Key: "host", Type: gaz.ConfigFlagTypeString, Default: "provider2", Description: "Host"},
	}
}

// AllTypesProvider tests all config flag types.
type AllTypesProvider struct{}

func (p *AllTypesProvider) ConfigNamespace() string {
	return "types"
}

func (p *AllTypesProvider) ConfigFlags() []gaz.ConfigFlag {
	return []gaz.ConfigFlag{
		{
			Key:         "str",
			Type:        gaz.ConfigFlagTypeString,
			Default:     "default-str",
			Description: "String value",
		},
		{Key: "num", Type: gaz.ConfigFlagTypeInt, Default: 42, Description: "Int value"},
		{Key: "flag", Type: gaz.ConfigFlagTypeBool, Default: false, Description: "Bool value"},
		{
			Key:         "timeout",
			Type:        gaz.ConfigFlagTypeDuration,
			Default:     time.Second * 30,
			Description: "Duration value",
		},
		{Key: "rate", Type: gaz.ConfigFlagTypeFloat, Default: 1.5, Description: "Float value"},
	}
}

// NonConfigProvider is a regular provider that doesn't implement ConfigProvider.
type NonConfigProvider struct{}

// DatabaseProvider is used for nested struct unmarshal testing.
type DatabaseProvider struct{}

func (p *DatabaseProvider) ConfigNamespace() string {
	return "database"
}

func (p *DatabaseProvider) ConfigFlags() []gaz.ConfigFlag {
	return []gaz.ConfigFlag{
		{Key: "host", Type: gaz.ConfigFlagTypeString, Default: "db.local", Description: "Database host"},
		{Key: "pool_max", Type: gaz.ConfigFlagTypeInt, Default: 10, Description: "Max pool size"},
		{Key: "pool_idle", Type: gaz.ConfigFlagTypeInt, Default: 5, Description: "Idle pool size"},
	}
}

// PartialProvider is used for partial fill testing.
type PartialProvider struct{}

func (p *PartialProvider) ConfigNamespace() string {
	return "partial"
}

func (p *PartialProvider) ConfigFlags() []gaz.ConfigFlag {
	return []gaz.ConfigFlag{
		{Key: "host", Type: gaz.ConfigFlagTypeString, Default: "myhost", Description: "Host"},
		// port deliberately has no default to test partial fill
	}
}

// ServerProvider is used for full unmarshal testing.
type ServerProvider struct{}

func (p *ServerProvider) ConfigNamespace() string {
	return "server"
}

func (p *ServerProvider) ConfigFlags() []gaz.ConfigFlag {
	return []gaz.ConfigFlag{
		{Key: "host", Type: gaz.ConfigFlagTypeString, Default: "0.0.0.0", Description: "Server host"},
		{Key: "port", Type: gaz.ConfigFlagTypeInt, Default: 8080, Description: "Server port"},
	}
}

// DebugProvider is used for full unmarshal testing.
type DebugProvider struct{}

func (p *DebugProvider) ConfigNamespace() string {
	return ""
}

func (p *DebugProvider) ConfigFlags() []gaz.ConfigFlag {
	return []gaz.ConfigFlag{
		{Key: "debug", Type: gaz.ConfigFlagTypeBool, Default: true, Description: "Debug mode"},
	}
}

// =============================================================================
// Tests
// =============================================================================

func (s *ProviderConfigSuite) TestBasicConfigProvider() {
	// Provider implements ConfigProvider, values accessible via ProviderValues
	app := gaz.New().
		WithConfig(&struct{}{}, config.WithEnvPrefix("TEST_BASIC"))

	err := gaz.For[*RedisProvider](app.Container()).ProviderFunc(func(_ *gaz.Container) *RedisProvider {
		return &RedisProvider{}
	})
	s.Require().NoError(err)

	err = app.Build()
	s.Require().NoError(err)

	// Resolve ProviderValues
	pv, err := gaz.Resolve[*gaz.ProviderValues](app.Container())
	s.Require().NoError(err)
	s.NotNil(pv)

	// Verify default value is accessible
	s.Equal("localhost", pv.GetString("redis.host"))
	s.Equal(6379, pv.GetInt("redis.port"))
}

func (s *ProviderConfigSuite) TestNamespacePrefixing() {
	// Keys are prefixed with namespace
	s.T().Setenv("REDIS_HOST", "custom-host")
	s.T().Setenv("REDIS_PORT", "9999")

	app := gaz.New().
		WithConfig(&struct{}{}, config.WithEnvPrefix("TEST_NS"))

	err := gaz.For[*RedisProvider](app.Container()).ProviderFunc(func(_ *gaz.Container) *RedisProvider {
		return &RedisProvider{}
	})
	s.Require().NoError(err)

	err = app.Build()
	s.Require().NoError(err)

	pv, err := gaz.Resolve[*gaz.ProviderValues](app.Container())
	s.Require().NoError(err)

	// Env var REDIS_HOST overrides default for redis.host
	s.Equal("custom-host", pv.GetString("redis.host"))
	s.Equal(9999, pv.GetInt("redis.port"))
}

func (s *ProviderConfigSuite) TestKeyCollision() {
	// Two providers with same full key fails Build()
	app := gaz.New().
		WithConfig(&struct{}{}, config.WithEnvPrefix("TEST_COLLISION"))

	err := gaz.For[*CollidingProvider1](app.Container()).ProviderFunc(func(_ *gaz.Container) *CollidingProvider1 {
		return &CollidingProvider1{}
	})
	s.Require().NoError(err)

	err = gaz.For[*CollidingProvider2](app.Container()).ProviderFunc(func(_ *gaz.Container) *CollidingProvider2 {
		return &CollidingProvider2{}
	})
	s.Require().NoError(err)

	err = app.Build()
	s.Require().Error(err)
	s.Require().ErrorIs(err, gaz.ErrConfigKeyCollision)
	s.Contains(err.Error(), "cache.host")
}

func (s *ProviderConfigSuite) TestRequiredMissing() {
	// Required flag not set fails Build()
	app := gaz.New().
		WithConfig(&struct{}{}, config.WithEnvPrefix("TEST_REQUIRED_MISSING"))

	err := gaz.For[*RequiredConfigProvider](app.Container()).ProviderFunc(func(_ *gaz.Container) *RequiredConfigProvider {
		return &RequiredConfigProvider{}
	})
	s.Require().NoError(err)

	err = app.Build()
	s.Require().Error(err)
	s.Contains(err.Error(), "required.api_key")
	s.Contains(err.Error(), "required config key")
}

func (s *ProviderConfigSuite) TestRequiredSet() {
	// Required flag set passes Build()
	s.T().Setenv("REQUIRED_API_KEY", "my-secret-key")

	app := gaz.New().
		WithConfig(&struct{}{}, config.WithEnvPrefix("TEST_REQUIRED_SET"))

	err := gaz.For[*RequiredConfigProvider](app.Container()).ProviderFunc(func(_ *gaz.Container) *RequiredConfigProvider {
		return &RequiredConfigProvider{}
	})
	s.Require().NoError(err)

	err = app.Build()
	s.Require().NoError(err)

	pv, err := gaz.Resolve[*gaz.ProviderValues](app.Container())
	s.Require().NoError(err)
	s.Equal("my-secret-key", pv.GetString("required.api_key"))
}

func (s *ProviderConfigSuite) TestDefaultValue() {
	// Default value used when not set
	app := gaz.New().
		WithConfig(&struct{}{}, config.WithEnvPrefix("TEST_DEFAULT"))

	err := gaz.For[*RedisProvider](app.Container()).ProviderFunc(func(_ *gaz.Container) *RedisProvider {
		return &RedisProvider{}
	})
	s.Require().NoError(err)

	err = app.Build()
	s.Require().NoError(err)

	pv, err := gaz.Resolve[*gaz.ProviderValues](app.Container())
	s.Require().NoError(err)

	// Default values from ConfigFlags
	s.Equal("localhost", pv.GetString("redis.host"))
	s.Equal(6379, pv.GetInt("redis.port"))
}

func (s *ProviderConfigSuite) TestAllTypes() {
	// All ConfigFlagType values work
	s.T().Setenv("TYPES_STR", "env-string")
	s.T().Setenv("TYPES_NUM", "100")
	s.T().Setenv("TYPES_FLAG", "true")
	s.T().Setenv("TYPES_TIMEOUT", "1m30s")
	s.T().Setenv("TYPES_RATE", "3.14")

	app := gaz.New().
		WithConfig(&struct{}{}, config.WithEnvPrefix("TEST_TYPES"))

	err := gaz.For[*AllTypesProvider](app.Container()).ProviderFunc(func(_ *gaz.Container) *AllTypesProvider {
		return &AllTypesProvider{}
	})
	s.Require().NoError(err)

	err = app.Build()
	s.Require().NoError(err)

	pv, err := gaz.Resolve[*gaz.ProviderValues](app.Container())
	s.Require().NoError(err)

	s.Equal("env-string", pv.GetString("types.str"))
	s.Equal(100, pv.GetInt("types.num"))
	s.True(pv.GetBool("types.flag"))
	s.Equal(90*time.Second, pv.GetDuration("types.timeout"))
	s.InDelta(3.14, pv.GetFloat64("types.rate"), 0.001)
}

func (s *ProviderConfigSuite) TestMultipleProviders() {
	// Multiple providers with different namespaces
	s.T().Setenv("REDIS_HOST", "redis-server")
	s.T().Setenv("CACHE_TTL", "10m")

	app := gaz.New().
		WithConfig(&struct{}{}, config.WithEnvPrefix("TEST_MULTI"))

	err := gaz.For[*RedisProvider](app.Container()).ProviderFunc(func(_ *gaz.Container) *RedisProvider {
		return &RedisProvider{}
	})
	s.Require().NoError(err)

	err = gaz.For[*CacheProvider](app.Container()).ProviderFunc(func(_ *gaz.Container) *CacheProvider {
		return &CacheProvider{}
	})
	s.Require().NoError(err)

	err = app.Build()
	s.Require().NoError(err)

	pv, err := gaz.Resolve[*gaz.ProviderValues](app.Container())
	s.Require().NoError(err)

	// Redis provider values
	s.Equal("redis-server", pv.GetString("redis.host"))
	s.Equal(6379, pv.GetInt("redis.port"))

	// Cache provider values
	s.Equal(10*time.Minute, pv.GetDuration("cache.ttl"))
	s.True(pv.GetBool("cache.enabled"))
}

func (s *ProviderConfigSuite) TestNoConfigManager() {
	// App without explicit WithConfig - config is auto-initialized with convention defaults
	app := gaz.New()

	err := gaz.For[*RedisProvider](app.Container()).ProviderFunc(func(_ *gaz.Container) *RedisProvider {
		return &RedisProvider{}
	})
	s.Require().NoError(err)

	// Should build successfully
	err = app.Build()
	s.Require().NoError(err)

	// ProviderValues IS registered because config is auto-initialized
	pv, err := gaz.Resolve[*gaz.ProviderValues](app.Container())
	s.Require().NoError(err)
	s.NotNil(pv)

	// ConfigProvider flags should work with auto-initialized config
	s.Equal("localhost", pv.GetString("redis.host"))
	s.Equal(6379, pv.GetInt("redis.port"))
}

func (s *ProviderConfigSuite) TestNonConfigProvider() {
	// Provider that doesn't implement ConfigProvider is ignored
	app := gaz.New().
		WithConfig(&struct{}{}, config.WithEnvPrefix("TEST_NON"))

	err := gaz.For[*NonConfigProvider](app.Container()).ProviderFunc(func(_ *gaz.Container) *NonConfigProvider {
		return &NonConfigProvider{}
	})
	s.Require().NoError(err)

	err = gaz.For[*RedisProvider](app.Container()).ProviderFunc(func(_ *gaz.Container) *RedisProvider {
		return &RedisProvider{}
	})
	s.Require().NoError(err)

	err = app.Build()
	s.Require().NoError(err)

	pv, err := gaz.Resolve[*gaz.ProviderValues](app.Container())
	s.Require().NoError(err)

	// RedisProvider config should work
	s.Equal("localhost", pv.GetString("redis.host"))
}

func (s *ProviderConfigSuite) TestEnvVarTranslation() {
	// Test that redis.host becomes REDIS_HOST
	s.T().Setenv("REDIS_HOST", "env-translated-host")

	app := gaz.New().
		WithConfig(&struct{}{}, config.WithEnvPrefix("TEST_ENV"))

	err := gaz.For[*RedisProvider](app.Container()).ProviderFunc(func(_ *gaz.Container) *RedisProvider {
		return &RedisProvider{}
	})
	s.Require().NoError(err)

	err = app.Build()
	s.Require().NoError(err)

	pv, err := gaz.Resolve[*gaz.ProviderValues](app.Container())
	s.Require().NoError(err)

	// Env var REDIS_HOST should override redis.host
	s.Equal("env-translated-host", pv.GetString("redis.host"))
}

// =============================================================================
// Unmarshal tests
// =============================================================================

func (s *ProviderConfigSuite) TestProviderValues_UnmarshalKey_SimpleStruct() {
	// Unmarshal simple struct with gaz tags
	s.T().Setenv("REDIS_HOST", "localhost")
	s.T().Setenv("REDIS_PORT", "6379")

	app := gaz.New().
		WithConfig(&struct{}{}, config.WithEnvPrefix("TEST_UNMARSHAL_SIMPLE"))

	err := gaz.For[*RedisProvider](app.Container()).ProviderFunc(func(_ *gaz.Container) *RedisProvider {
		return &RedisProvider{}
	})
	s.Require().NoError(err)

	err = app.Build()
	s.Require().NoError(err)

	pv := gaz.MustResolve[*gaz.ProviderValues](app.Container())

	target := &struct {
		Host string `gaz:"host"`
		Port int    `gaz:"port"`
	}{}
	err = pv.UnmarshalKey("redis", target)
	s.Require().NoError(err)

	s.Equal("localhost", target.Host)
	s.Equal(6379, target.Port)
}

func (s *ProviderConfigSuite) TestProviderValues_UnmarshalKey_NestedStruct() {
	// Test UnmarshalKey with struct that has multiple fields using gaz tags
	app := gaz.New().
		WithConfig(&struct{}{}, config.WithEnvPrefix("TEST_UNMARSHAL_NESTED"))

	// Register a provider that declares the database namespace with flat keys
	err := gaz.For[*DatabaseProvider](app.Container()).ProviderFunc(func(_ *gaz.Container) *DatabaseProvider {
		return &DatabaseProvider{}
	})
	s.Require().NoError(err)

	err = app.Build()
	s.Require().NoError(err)

	pv := gaz.MustResolve[*gaz.ProviderValues](app.Container())

	// Test that struct fields are unmarshaled correctly using gaz tags
	target := &struct {
		Host     string `gaz:"host"`
		PoolMax  int    `gaz:"pool_max"`
		PoolIdle int    `gaz:"pool_idle"`
	}{}
	err = pv.UnmarshalKey("database", target)
	s.Require().NoError(err)

	s.Equal("db.local", target.Host)
	s.Equal(10, target.PoolMax)
	s.Equal(5, target.PoolIdle)
}

func (s *ProviderConfigSuite) TestProviderValues_UnmarshalKey_MissingNamespace() {
	// Missing namespace returns ErrKeyNotFound
	app := gaz.New().
		WithConfig(&struct{}{}, config.WithEnvPrefix("TEST_UNMARSHAL_MISSING"))

	err := app.Build()
	s.Require().NoError(err)

	pv := gaz.MustResolve[*gaz.ProviderValues](app.Container())

	target := &struct {
		Host string `gaz:"host"`
	}{}
	err = pv.UnmarshalKey("nonexistent", target)

	s.Require().Error(err)
	s.ErrorIs(err, config.ErrKeyNotFound)
	s.Contains(err.Error(), "nonexistent")
}

func (s *ProviderConfigSuite) TestProviderValues_UnmarshalKey_PartialFill() {
	// Partial fill leaves unset fields at zero value
	// PartialProvider only declares "host" with default "myhost"
	// The target struct has "port" which is not in config - should stay zero
	app := gaz.New().
		WithConfig(&struct{}{}, config.WithEnvPrefix("TEST_UNMARSHAL_PARTIAL"))

	// Register provider for partial namespace
	err := gaz.For[*PartialProvider](app.Container()).ProviderFunc(func(_ *gaz.Container) *PartialProvider {
		return &PartialProvider{}
	})
	s.Require().NoError(err)

	err = app.Build()
	s.Require().NoError(err)

	pv := gaz.MustResolve[*gaz.ProviderValues](app.Container())

	target := &struct {
		Host string `gaz:"host"`
		Port int    `gaz:"port"`
	}{}
	err = pv.UnmarshalKey("partial", target)
	s.Require().NoError(err)

	s.Equal("myhost", target.Host)
	s.Equal(0, target.Port) // zero value because not in config
}

func (s *ProviderConfigSuite) TestProviderValues_Unmarshal() {
	// Full config unmarshaling using UnmarshalKey for namespaced config
	// This demonstrates the recommended pattern: use UnmarshalKey for specific namespaces
	type RedisConfig struct {
		Host string `gaz:"host"`
		Port int    `gaz:"port"`
	}

	app := gaz.New().
		WithConfig(&struct{}{}, config.WithEnvPrefix("TEST_UNMARSHAL_FULL"))

	// Register provider for redis namespace
	err := gaz.For[*RedisProvider](app.Container()).ProviderFunc(func(_ *gaz.Container) *RedisProvider {
		return &RedisProvider{}
	})
	s.Require().NoError(err)

	err = app.Build()
	s.Require().NoError(err)

	pv := gaz.MustResolve[*gaz.ProviderValues](app.Container())

	// UnmarshalKey is the recommended way to get namespaced config
	var cfg RedisConfig
	err = pv.UnmarshalKey("redis", &cfg)
	s.Require().NoError(err)

	s.Equal("localhost", cfg.Host)
	s.Equal(6379, cfg.Port)

	// Unmarshal (full config) can unmarshal to a map to inspect all settings
	var allConfig map[string]any
	err = pv.Unmarshal(&allConfig)
	s.Require().NoError(err)
	s.Contains(allConfig, "redis")
}
