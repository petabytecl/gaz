package module

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/health"
)

func TestNew(t *testing.T) {
	t.Run("creates module with default config", func(t *testing.T) {
		app := gaz.New()
		app.Use(New())

		err := app.Build()
		require.NoError(t, err)

		// Verify Config resolves with defaults
		cfg, resolveErr := gaz.Resolve[health.Config](app.Container())
		require.NoError(t, resolveErr)
		require.Equal(t, health.DefaultPort, cfg.Port)
		require.Equal(t, health.DefaultLivenessPath, cfg.LivenessPath)
		require.Equal(t, health.DefaultReadinessPath, cfg.ReadinessPath)
		require.Equal(t, health.DefaultStartupPath, cfg.StartupPath)
	})

	t.Run("registers health components", func(t *testing.T) {
		app := gaz.New()
		app.Use(New())

		err := app.Build()
		require.NoError(t, err)

		// Verify all health components are registered
		_, err = gaz.Resolve[*health.ShutdownCheck](app.Container())
		require.NoError(t, err)

		_, err = gaz.Resolve[*health.Manager](app.Container())
		require.NoError(t, err)

		_, err = gaz.Resolve[*health.ManagementServer](app.Container())
		require.NoError(t, err)
	})
}

func TestConfig_Flags(t *testing.T) {
	cfg := health.DefaultConfig()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

	cfg.Flags(fs)

	// Verify flags are registered
	flag := fs.Lookup("health-port")
	require.NotNil(t, flag, "health-port flag should be registered")
	require.Equal(t, "9090", flag.DefValue)

	flag = fs.Lookup("health-liveness-path")
	require.NotNil(t, flag, "health-liveness-path flag should be registered")
	require.Equal(t, "/live", flag.DefValue)

	flag = fs.Lookup("health-readiness-path")
	require.NotNil(t, flag, "health-readiness-path flag should be registered")
	require.Equal(t, "/ready", flag.DefValue)

	flag = fs.Lookup("health-startup-path")
	require.NotNil(t, flag, "health-startup-path flag should be registered")
	require.Equal(t, "/startup", flag.DefValue)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  health.Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			config:  health.DefaultConfig(),
			wantErr: false,
		},
		{
			name: "port 0 fails",
			config: health.Config{
				Port:          0,
				LivenessPath:  "/live",
				ReadinessPath: "/ready",
				StartupPath:   "/startup",
			},
			wantErr: true,
			errMsg:  "port must be greater than 0",
		},
		{
			name: "port over 65535 fails",
			config: health.Config{
				Port:          65536,
				LivenessPath:  "/live",
				ReadinessPath: "/ready",
				StartupPath:   "/startup",
			},
			wantErr: true,
			errMsg:  "port must be less than or equal to 65535",
		},
		{
			name: "negative port fails",
			config: health.Config{
				Port:          -1,
				LivenessPath:  "/live",
				ReadinessPath: "/ready",
				StartupPath:   "/startup",
			},
			wantErr: true,
			errMsg:  "port must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_SetDefaults(t *testing.T) {
	// Zero config
	cfg := health.Config{}
	cfg.SetDefaults()

	require.Equal(t, health.DefaultPort, cfg.Port)
	require.Equal(t, health.DefaultLivenessPath, cfg.LivenessPath)
	require.Equal(t, health.DefaultReadinessPath, cfg.ReadinessPath)
	require.Equal(t, health.DefaultStartupPath, cfg.StartupPath)
}

func TestConfig_SetDefaults_PreservesExisting(t *testing.T) {
	cfg := health.Config{
		Port:          8081,
		LivenessPath:  "/custom/live",
		ReadinessPath: "/custom/ready",
		StartupPath:   "/custom/startup",
	}
	cfg.SetDefaults()

	// Custom values should be preserved
	require.Equal(t, 8081, cfg.Port)
	require.Equal(t, "/custom/live", cfg.LivenessPath)
	require.Equal(t, "/custom/ready", cfg.ReadinessPath)
	require.Equal(t, "/custom/startup", cfg.StartupPath)
}

func TestConfig_Namespace(t *testing.T) {
	cfg := health.Config{}
	require.Equal(t, "health", cfg.Namespace())
}
