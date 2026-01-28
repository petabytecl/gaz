package viper_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz/config"
	cfgviper "github.com/petabytecl/gaz/config/viper"
)

// =============================================================================
// Test New() and NewWithViper()
// =============================================================================

func TestNew_ReturnsBackend(t *testing.T) {
	backend := cfgviper.New()
	assert.NotNil(t, backend)
}

func TestNewWithViper_ReturnsBackend(t *testing.T) {
	v := cfgviper.New().Viper()
	backend := cfgviper.NewWithViper(v)
	assert.NotNil(t, backend)
	assert.Same(t, v, backend.Viper())
}

// =============================================================================
// Test config.Backend interface implementation
// =============================================================================

func TestBackend_Get_Set(t *testing.T) {
	backend := cfgviper.New()

	backend.Set("key", "value")
	assert.Equal(t, "value", backend.Get("key"))
}

func TestBackend_GetString(t *testing.T) {
	backend := cfgviper.New()
	backend.Set("host", "localhost")

	assert.Equal(t, "localhost", backend.GetString("host"))
}

func TestBackend_GetInt(t *testing.T) {
	backend := cfgviper.New()
	backend.Set("port", 8080)

	assert.Equal(t, 8080, backend.GetInt("port"))
}

func TestBackend_GetBool(t *testing.T) {
	backend := cfgviper.New()
	backend.Set("debug", true)

	assert.True(t, backend.GetBool("debug"))
}

func TestBackend_GetDuration(t *testing.T) {
	backend := cfgviper.New()
	backend.Set("timeout", "30s")

	assert.Equal(t, 30*time.Second, backend.GetDuration("timeout"))
}

func TestBackend_GetFloat64(t *testing.T) {
	backend := cfgviper.New()
	backend.Set("rate", 1.5)

	assert.Equal(t, 1.5, backend.GetFloat64("rate"))
}

func TestBackend_SetDefault(t *testing.T) {
	backend := cfgviper.New()
	backend.SetDefault("host", "defaulthost")

	assert.Equal(t, "defaulthost", backend.GetString("host"))

	// Explicit set should override default
	backend.Set("host", "explicithost")
	assert.Equal(t, "explicithost", backend.GetString("host"))
}

func TestBackend_IsSet(t *testing.T) {
	backend := cfgviper.New()

	assert.False(t, backend.IsSet("missing"))

	backend.Set("present", "value")
	assert.True(t, backend.IsSet("present"))
}

func TestBackend_Unmarshal(t *testing.T) {
	backend := cfgviper.New()
	backend.Set("host", "testhost")
	backend.Set("port", 9000)

	type cfg struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	}

	var c cfg
	err := backend.Unmarshal(&c)
	require.NoError(t, err)

	assert.Equal(t, "testhost", c.Host)
	assert.Equal(t, 9000, c.Port)
}

func TestBackend_UnmarshalKey(t *testing.T) {
	backend := cfgviper.New()
	backend.Set("database.host", "dbhost")
	backend.Set("database.port", 5432)

	type dbConfig struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	}

	var db dbConfig
	err := backend.UnmarshalKey("database", &db)
	require.NoError(t, err)

	assert.Equal(t, "dbhost", db.Host)
	assert.Equal(t, 5432, db.Port)
}

// =============================================================================
// Test config.Watcher interface implementation
// =============================================================================

func TestBackend_WatchConfig_DoesNotPanic(t *testing.T) {
	backend := cfgviper.New()

	// Just ensure it doesn't panic - actual watching requires a config file
	assert.NotPanics(t, func() {
		backend.WatchConfig()
	})
}

func TestBackend_OnConfigChange_AcceptsCallback(t *testing.T) {
	backend := cfgviper.New()

	assert.NotPanics(t, func() {
		backend.OnConfigChange(func(event any) {
			// Callback registered successfully
			_ = event
		})
	})
	// Note: We can't easily test the callback being called without file changes
}

// =============================================================================
// Test config.Writer interface implementation
// =============================================================================

func TestBackend_WriteConfigAs(t *testing.T) {
	backend := cfgviper.New()
	backend.Set("test", "value")

	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	err := backend.WriteConfigAs(tmpFile)
	require.NoError(t, err)

	// Verify file was written
	data, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), "test")
}

func TestBackend_SafeWriteConfigAs_DoesNotOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.yaml")

	// Create existing file
	err := os.WriteFile(tmpFile, []byte("existing: content\n"), 0o644)
	require.NoError(t, err)

	backend := cfgviper.New()
	backend.Set("new", "value")

	err = backend.SafeWriteConfigAs(tmpFile)
	assert.Error(t, err) // Should error because file exists

	// Verify original content is preserved
	data, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), "existing")
	assert.NotContains(t, string(data), "new")
}

// =============================================================================
// Test config.EnvBinder interface implementation
// =============================================================================

func TestBackend_SetEnvPrefix(t *testing.T) {
	require.NoError(t, os.Setenv("VIPERTEST_HOST", "envhost"))
	defer os.Unsetenv("VIPERTEST_HOST")

	backend := cfgviper.New()
	backend.SetEnvPrefix("VIPERTEST")
	backend.AutomaticEnv()
	_ = backend.BindEnv("host")

	assert.Equal(t, "envhost", backend.GetString("host"))
}

func TestBackend_BindEnv(t *testing.T) {
	require.NoError(t, os.Setenv("CUSTOM_VAR", "customvalue"))
	defer os.Unsetenv("CUSTOM_VAR")

	backend := cfgviper.New()
	err := backend.BindEnv("mykey", "CUSTOM_VAR")
	require.NoError(t, err)

	assert.Equal(t, "customvalue", backend.GetString("mykey"))
}

func TestBackend_AutomaticEnv(t *testing.T) {
	require.NoError(t, os.Setenv("AUTOTEST_DEBUG", "true"))
	defer os.Unsetenv("AUTOTEST_DEBUG")

	backend := cfgviper.New()
	backend.SetEnvPrefix("AUTOTEST")
	backend.AutomaticEnv()
	_ = backend.BindEnv("debug")

	assert.True(t, backend.GetBool("debug"))
}

func TestBackend_SetEnvKeyReplacer(t *testing.T) {
	require.NoError(t, os.Setenv("APP_DATABASE__HOST", "dbhost"))
	defer os.Unsetenv("APP_DATABASE__HOST")

	backend := cfgviper.New()
	backend.SetEnvPrefix("APP")
	backend.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
	backend.AutomaticEnv()
	_ = backend.BindEnv("database.host")

	assert.Equal(t, "dbhost", backend.GetString("database.host"))
}

func TestBackend_SetStringsReplacer(t *testing.T) {
	require.NoError(t, os.Setenv("APP2_SERVER__PORT", "9000"))
	defer os.Unsetenv("APP2_SERVER__PORT")

	backend := cfgviper.New()
	backend.SetEnvPrefix("APP2")
	backend.SetStringsReplacer(strings.NewReplacer(".", "__"))
	backend.AutomaticEnv()
	_ = backend.BindEnv("server.port")

	assert.Equal(t, 9000, backend.GetInt("server.port"))
}

// =============================================================================
// Test viper-specific methods
// =============================================================================

func TestBackend_SetConfigName_SetConfigType_AddConfigPath(t *testing.T) {
	backend := cfgviper.New()
	backend.SetConfigName("config")
	backend.SetConfigType("yaml")
	backend.AddConfigPath("testdata")

	err := backend.ReadInConfig()
	require.NoError(t, err)

	assert.Equal(t, "viperhost", backend.GetString("host"))
	assert.Equal(t, 3000, backend.GetInt("port"))
}

func TestBackend_ReadInConfig_WithValidFile(t *testing.T) {
	backend := cfgviper.New()
	backend.SetConfigName("config")
	backend.SetConfigType("yaml")
	backend.AddConfigPath("testdata")

	err := backend.ReadInConfig()
	require.NoError(t, err)

	assert.Equal(t, "viperhost", backend.GetString("host"))
	assert.True(t, backend.GetBool("debug"))
}

func TestBackend_MergeInConfig(t *testing.T) {
	backend := cfgviper.New()
	backend.SetConfigName("config")
	backend.SetConfigType("yaml")
	backend.AddConfigPath("testdata")

	// Read base config
	err := backend.ReadInConfig()
	require.NoError(t, err)

	// Change to profile config and merge
	backend.SetConfigName("config.prod")
	err = backend.MergeInConfig()
	require.NoError(t, err)

	// Merged values
	assert.Equal(t, "prodhost", backend.GetString("host")) // Overridden
	assert.Equal(t, 3000, backend.GetInt("port"))          // From base
}

func TestBackend_ConfigFileUsed(t *testing.T) {
	backend := cfgviper.New()
	backend.SetConfigName("config")
	backend.SetConfigType("yaml")
	backend.AddConfigPath("testdata")

	err := backend.ReadInConfig()
	require.NoError(t, err)

	configFile := backend.ConfigFileUsed()
	assert.Contains(t, configFile, "config.yaml")
}

func TestBackend_AllSettings(t *testing.T) {
	backend := cfgviper.New()
	backend.Set("host", "localhost")
	backend.Set("port", 8080)

	settings := backend.AllSettings()
	assert.Equal(t, "localhost", settings["host"])
	assert.Equal(t, 8080, settings["port"])
}

func TestBackend_Viper_ReturnsUnderlyingViper(t *testing.T) {
	backend := cfgviper.New()
	v := backend.Viper()

	assert.NotNil(t, v)

	// Set via viper directly should reflect in backend
	v.Set("direct", "value")
	assert.Equal(t, "value", backend.GetString("direct"))
}

// =============================================================================
// Test IsConfigFileNotFoundError
// =============================================================================

func TestIsConfigFileNotFoundError_ReturnsTrue(t *testing.T) {
	backend := cfgviper.New()
	backend.SetConfigName("nonexistent")
	backend.SetConfigType("yaml")
	backend.AddConfigPath(t.TempDir())

	err := backend.ReadInConfig()
	require.Error(t, err)

	assert.True(t, cfgviper.IsConfigFileNotFoundError(err))
	assert.True(t, backend.IsConfigFileNotFoundError(err))
}

func TestIsConfigFileNotFoundError_ReturnsFalse(t *testing.T) {
	err := os.ErrPermission
	assert.False(t, cfgviper.IsConfigFileNotFoundError(err))
}

// =============================================================================
// Test interface compliance
// =============================================================================

func TestBackend_ImplementsBackend(t *testing.T) {
	var _ config.Backend = (*cfgviper.Backend)(nil)
}

func TestBackend_ImplementsWatcher(t *testing.T) {
	var _ config.Watcher = (*cfgviper.Backend)(nil)
}

func TestBackend_ImplementsWriter(t *testing.T) {
	var _ config.Writer = (*cfgviper.Backend)(nil)
}

func TestBackend_ImplementsEnvBinder(t *testing.T) {
	var _ config.EnvBinder = (*cfgviper.Backend)(nil)
}
