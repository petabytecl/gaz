package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz/config"
	cfgviper "github.com/petabytecl/gaz/config/viper"
)

// =============================================================================
// Mock Backend for testing Manager in isolation
// =============================================================================

type mockBackend struct {
	data     map[string]any
	defaults map[string]any
}

func newMockBackend() *mockBackend {
	return &mockBackend{
		data:     make(map[string]any),
		defaults: make(map[string]any),
	}
}

func (m *mockBackend) Get(key string) any {
	if v, ok := m.data[key]; ok {
		return v
	}
	return m.defaults[key]
}

func (m *mockBackend) GetString(key string) string {
	v := m.Get(key)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func (m *mockBackend) GetInt(key string) int {
	v := m.Get(key)
	if i, ok := v.(int); ok {
		return i
	}
	return 0
}

func (m *mockBackend) GetBool(key string) bool {
	v := m.Get(key)
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

func (m *mockBackend) GetDuration(key string) time.Duration {
	v := m.Get(key)
	if d, ok := v.(time.Duration); ok {
		return d
	}
	return 0
}

func (m *mockBackend) GetFloat64(key string) float64 {
	v := m.Get(key)
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}

func (m *mockBackend) Set(key string, value any) {
	m.data[key] = value
}

func (m *mockBackend) SetDefault(key string, value any) {
	m.defaults[key] = value
}

func (m *mockBackend) IsSet(key string) bool {
	_, ok := m.data[key]
	if ok {
		return true
	}
	_, ok = m.defaults[key]
	return ok
}

func (m *mockBackend) Unmarshal(target any) error {
	// Simple mock implementation - doesn't actually unmarshal
	return nil
}

func (m *mockBackend) UnmarshalKey(key string, target any) error {
	return nil
}

// =============================================================================
// Test New() and NewWithBackend()
// =============================================================================

func TestNew_WithBackend_ReturnsManager(t *testing.T) {
	backend := newMockBackend()
	mgr := config.New(config.WithBackend(backend))

	assert.NotNil(t, mgr)
	assert.Equal(t, backend, mgr.Backend())
}

func TestNew_WithoutBackend_Panics(t *testing.T) {
	assert.Panics(t, func() {
		config.New() // No backend provided
	})
}

func TestNewWithBackend_ReturnsManager(t *testing.T) {
	backend := newMockBackend()
	mgr := config.NewWithBackend(backend)

	assert.NotNil(t, mgr)
	assert.Equal(t, backend, mgr.Backend())
}

func TestNewWithBackend_NilBackend_Panics(t *testing.T) {
	assert.Panics(t, func() {
		config.NewWithBackend(nil)
	})
}

func TestNew_WithOptions_AppliesOptions(t *testing.T) {
	backend := cfgviper.New()
	defaults := map[string]any{"foo": "bar"}

	mgr := config.New(
		config.WithBackend(backend),
		config.WithName("myconfig"),
		config.WithType("json"),
		config.WithEnvPrefix("MYAPP"),
		config.WithSearchPaths(".", "./config"),
		config.WithDefaults(defaults),
	)

	assert.NotNil(t, mgr)

	// Load to apply defaults to the backend
	err := mgr.Load()
	assert.NoError(t, err)

	// Verify defaults were applied to backend
	assert.True(t, backend.IsSet("foo"))
	assert.Equal(t, "bar", backend.GetString("foo"))
}

// =============================================================================
// Test Load()
// =============================================================================

func TestLoad_WithMissingConfigFile_NoError(t *testing.T) {
	// Using viper backend because mock doesn't implement configReader
	backend := cfgviper.New()
	mgr := config.NewWithBackend(backend,
		config.WithName("nonexistent"),
		config.WithSearchPaths(t.TempDir()),
	)

	err := mgr.Load()
	assert.NoError(t, err) // Missing config file is OK
}

func TestLoad_WithValidConfigFile(t *testing.T) {
	backend := cfgviper.New()
	testdataDir := filepath.Join("testdata")

	mgr := config.NewWithBackend(backend,
		config.WithName("config"),
		config.WithSearchPaths(testdataDir),
	)

	err := mgr.Load()
	require.NoError(t, err)

	// Verify values were loaded
	assert.Equal(t, "testhost", backend.GetString("host"))
	assert.Equal(t, 9000, backend.GetInt("port"))
	assert.True(t, backend.GetBool("debug"))
}

func TestLoad_WithDefaults_AppliesDefaults(t *testing.T) {
	backend := cfgviper.New()
	mgr := config.NewWithBackend(backend,
		config.WithName("nonexistent"),
		config.WithSearchPaths(t.TempDir()),
		config.WithDefaults(map[string]any{
			"host": "defaulthost",
			"port": 8080,
		}),
	)

	err := mgr.Load()
	require.NoError(t, err)

	assert.Equal(t, "defaulthost", backend.GetString("host"))
	assert.Equal(t, 8080, backend.GetInt("port"))
}

func TestLoad_WithEnvPrefix_BindsEnvVars(t *testing.T) {
	require.NoError(t, os.Setenv("CFGTEST_HOST", "envhost"))
	defer os.Unsetenv("CFGTEST_HOST")

	backend := cfgviper.New()
	mgr := config.NewWithBackend(backend,
		config.WithName("nonexistent"),
		config.WithSearchPaths(t.TempDir()),
		config.WithEnvPrefix("CFGTEST"),
	)

	err := mgr.Load()
	require.NoError(t, err)

	// After binding, AutomaticEnv should pick up the env var
	assert.Equal(t, "envhost", backend.GetString("host"))
}

func TestLoad_WithProfileConfig_MergesProfile(t *testing.T) {
	require.NoError(t, os.Setenv("CFG_PROFILE", "local"))
	defer os.Unsetenv("CFG_PROFILE")

	backend := cfgviper.New()
	testdataDir := filepath.Join("testdata")

	mgr := config.NewWithBackend(backend,
		config.WithName("config"),
		config.WithSearchPaths(testdataDir),
		config.WithProfileEnv("CFG_PROFILE"),
	)

	err := mgr.Load()
	require.NoError(t, err)

	// Profile overrides host, but base keeps port
	assert.Equal(t, "localhost", backend.GetString("host"))
	assert.Equal(t, 9000, backend.GetInt("port")) // From base config
	assert.False(t, backend.GetBool("debug"))     // Overridden by profile
}

// =============================================================================
// Test LoadInto()
// =============================================================================

type testConfig struct {
	Host  string `mapstructure:"host"`
	Port  int    `mapstructure:"port"`
	Debug bool   `mapstructure:"debug"`
}

func (c *testConfig) Default() {
	if c.Host == "" {
		c.Host = "defaulthost"
	}
	if c.Port == 0 {
		c.Port = 8080
	}
}

type validatorConfig struct {
	Host string `mapstructure:"host" validate:"required"`
	Port int    `mapstructure:"port" validate:"min=1,max=65535"`
}

type customValidatorConfig struct {
	Port int `mapstructure:"port"`
}

func (c *customValidatorConfig) Validate() error {
	if c.Port < 0 {
		return errors.New("port must be positive")
	}
	return nil
}

func TestLoadInto_UnmarshalsIntoStruct(t *testing.T) {
	backend := cfgviper.New()
	testdataDir := filepath.Join("testdata")

	mgr := config.NewWithBackend(backend,
		config.WithName("config"),
		config.WithSearchPaths(testdataDir),
	)

	var cfg testConfig
	err := mgr.LoadInto(&cfg)
	require.NoError(t, err)

	assert.Equal(t, "testhost", cfg.Host)
	assert.Equal(t, 9000, cfg.Port)
	assert.True(t, cfg.Debug)
}

func TestLoadInto_CallsDefaulter(t *testing.T) {
	backend := cfgviper.New()
	mgr := config.NewWithBackend(backend,
		config.WithName("nonexistent"),
		config.WithSearchPaths(t.TempDir()),
	)

	var cfg testConfig
	err := mgr.LoadInto(&cfg)
	require.NoError(t, err)

	// Defaulter should have set defaults
	assert.Equal(t, "defaulthost", cfg.Host)
	assert.Equal(t, 8080, cfg.Port)
}

func TestLoadInto_ValidatesStructTags(t *testing.T) {
	backend := cfgviper.New()
	mgr := config.NewWithBackend(backend,
		config.WithName("nonexistent"),
		config.WithSearchPaths(t.TempDir()),
	)

	var cfg validatorConfig
	err := mgr.LoadInto(&cfg)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, config.ErrConfigValidation))
}

func TestLoadInto_CallsCustomValidator(t *testing.T) {
	backend := cfgviper.New()
	backend.Set("port", -1)

	mgr := config.NewWithBackend(backend,
		config.WithName("nonexistent"),
		config.WithSearchPaths(t.TempDir()),
	)

	var cfg customValidatorConfig
	err := mgr.LoadInto(&cfg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "port must be positive")
}

func TestLoadInto_WithNilTarget_NoError(t *testing.T) {
	backend := cfgviper.New()
	mgr := config.NewWithBackend(backend)

	err := mgr.LoadInto(nil)
	assert.NoError(t, err)
}

func TestLoadInto_WithEnvVars_BindsToStruct(t *testing.T) {
	require.NoError(t, os.Setenv("LOADTEST_HOST", "envhost"))
	require.NoError(t, os.Setenv("LOADTEST_PORT", "9999"))
	defer func() {
		os.Unsetenv("LOADTEST_HOST")
		os.Unsetenv("LOADTEST_PORT")
	}()

	backend := cfgviper.New()
	mgr := config.NewWithBackend(backend,
		config.WithName("nonexistent"),
		config.WithSearchPaths(t.TempDir()),
		config.WithEnvPrefix("LOADTEST"),
	)

	var cfg testConfig
	err := mgr.LoadInto(&cfg)
	require.NoError(t, err)

	assert.Equal(t, "envhost", cfg.Host)
	assert.Equal(t, 9999, cfg.Port)
}

// =============================================================================
// Test Backend()
// =============================================================================

func TestBackend_ReturnsUnderlyingBackend(t *testing.T) {
	backend := newMockBackend()
	mgr := config.NewWithBackend(backend)

	assert.Same(t, backend, mgr.Backend())
}

// =============================================================================
// Test RegisterProviderFlags and ValidateProviderFlags
// =============================================================================

func TestRegisterProviderFlags_SetsDefaults(t *testing.T) {
	backend := cfgviper.New()
	mgr := config.NewWithBackend(backend)

	flags := []config.ConfigFlag{
		{Key: "host", Default: "localhost"},
		{Key: "port", Default: 8080},
	}

	err := mgr.RegisterProviderFlags("myapp", flags)
	require.NoError(t, err)

	assert.Equal(t, "localhost", backend.GetString("myapp.host"))
	assert.Equal(t, 8080, backend.GetInt("myapp.port"))
}

func TestValidateProviderFlags_ReturnsErrorsForMissing(t *testing.T) {
	backend := cfgviper.New()
	mgr := config.NewWithBackend(backend)

	flags := []config.ConfigFlag{
		{Key: "host", Required: true},
		{Key: "port", Required: true},
	}

	errs := mgr.ValidateProviderFlags("myapp", flags)
	assert.Len(t, errs, 2)
}

func TestValidateProviderFlags_NoErrorsWhenSet(t *testing.T) {
	backend := cfgviper.New()
	backend.Set("myapp.host", "localhost")
	backend.Set("myapp.port", 8080)

	mgr := config.NewWithBackend(backend)

	flags := []config.ConfigFlag{
		{Key: "host", Required: true},
		{Key: "port", Required: true},
	}

	errs := mgr.ValidateProviderFlags("myapp", flags)
	assert.Len(t, errs, 0)
}
