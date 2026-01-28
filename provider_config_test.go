package gaz_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/petabytecl/gaz"
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

// =============================================================================
// Tests
// =============================================================================

func (s *ProviderConfigSuite) TestBasicConfigProvider() {
	// Provider implements ConfigProvider, values accessible via ProviderValues
	app := gaz.New().
		WithConfig(&struct{}{}, gaz.WithEnvPrefix("TEST_BASIC"))

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
		WithConfig(&struct{}{}, gaz.WithEnvPrefix("TEST_NS"))

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
		WithConfig(&struct{}{}, gaz.WithEnvPrefix("TEST_COLLISION"))

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
		WithConfig(&struct{}{}, gaz.WithEnvPrefix("TEST_REQUIRED_MISSING"))

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
		WithConfig(&struct{}{}, gaz.WithEnvPrefix("TEST_REQUIRED_SET"))

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
		WithConfig(&struct{}{}, gaz.WithEnvPrefix("TEST_DEFAULT"))

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
		WithConfig(&struct{}{}, gaz.WithEnvPrefix("TEST_TYPES"))

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
		WithConfig(&struct{}{}, gaz.WithEnvPrefix("TEST_MULTI"))

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
	// App without WithConfig - provider config features are skipped
	app := gaz.New()

	err := gaz.For[*RedisProvider](app.Container()).ProviderFunc(func(_ *gaz.Container) *RedisProvider {
		return &RedisProvider{}
	})
	s.Require().NoError(err)

	// Should build successfully
	err = app.Build()
	s.Require().NoError(err)

	// ProviderValues should not be registered when no ConfigManager
	_, err = gaz.Resolve[*gaz.ProviderValues](app.Container())
	s.Require().Error(err)
	s.ErrorIs(err, gaz.ErrNotFound)
}

func (s *ProviderConfigSuite) TestNonConfigProvider() {
	// Provider that doesn't implement ConfigProvider is ignored
	app := gaz.New().
		WithConfig(&struct{}{}, gaz.WithEnvPrefix("TEST_NON"))

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
		WithConfig(&struct{}{}, gaz.WithEnvPrefix("TEST_ENV"))

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
