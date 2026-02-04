package logger

import (
	"log/slog"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	require.Equal(t, slog.LevelInfo, cfg.Level)
	require.Equal(t, "info", cfg.levelName)
	require.Equal(t, "text", cfg.Format)
	require.Equal(t, "stdout", cfg.Output)
	require.False(t, cfg.AddSource)
}

func TestConfig_Namespace(t *testing.T) {
	cfg := Config{}
	require.Equal(t, "log", cfg.Namespace())
}

func TestConfig_Flags(t *testing.T) {
	cfg := DefaultConfig()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

	cfg.Flags(fs)

	// Verify flags are registered
	flag := fs.Lookup("log-level")
	require.NotNil(t, flag, "log-level flag should be registered")
	require.Equal(t, "info", flag.DefValue)

	flag = fs.Lookup("log-format")
	require.NotNil(t, flag, "log-format flag should be registered")
	require.Equal(t, "text", flag.DefValue)

	flag = fs.Lookup("log-output")
	require.NotNil(t, flag, "log-output flag should be registered")
	require.Equal(t, "stdout", flag.DefValue)

	flag = fs.Lookup("log-add-source")
	require.NotNil(t, flag, "log-add-source flag should be registered")
	require.Equal(t, "false", flag.DefValue)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		levelName string
		format    string
		wantErr   bool
		errMsg    string
		wantLevel slog.Level
	}{
		{
			name:      "valid debug level",
			levelName: "debug",
			format:    "text",
			wantLevel: slog.LevelDebug,
		},
		{
			name:      "valid info level",
			levelName: "info",
			format:    "json",
			wantLevel: slog.LevelInfo,
		},
		{
			name:      "valid warn level",
			levelName: "warn",
			format:    "text",
			wantLevel: slog.LevelWarn,
		},
		{
			name:      "valid error level",
			levelName: "error",
			format:    "json",
			wantLevel: slog.LevelError,
		},
		{
			name:      "invalid level",
			levelName: "invalid",
			format:    "text",
			wantErr:   true,
			errMsg:    "invalid log level",
		},
		{
			name:      "invalid format",
			levelName: "info",
			format:    "yaml",
			wantErr:   true,
			errMsg:    "invalid log format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				levelName: tt.levelName,
				Format:    tt.format,
			}

			err := cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantLevel, cfg.Level)
			}
		})
	}
}

func TestConfig_SetDefaults(t *testing.T) {
	t.Run("sets all defaults for zero config", func(t *testing.T) {
		cfg := Config{}
		cfg.SetDefaults()

		require.Equal(t, "text", cfg.Format)
		require.Equal(t, "stdout", cfg.Output)
		require.Equal(t, "info", cfg.levelName)
		require.Equal(t, slog.LevelInfo, cfg.Level)
	})

	t.Run("preserves existing values", func(t *testing.T) {
		cfg := Config{
			Format:    "json",
			Output:    "stderr",
			levelName: "debug",
			Level:     slog.LevelDebug,
		}
		cfg.SetDefaults()

		require.Equal(t, "json", cfg.Format)
		require.Equal(t, "stderr", cfg.Output)
		require.Equal(t, "debug", cfg.levelName)
	})
}

func TestConfig_LevelName(t *testing.T) {
	cfg := Config{levelName: "warn"}
	require.Equal(t, "warn", cfg.LevelName())
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantLevel slog.Level
		wantErr   bool
	}{
		{"debug", "debug", slog.LevelDebug, false},
		{"info", "info", slog.LevelInfo, false},
		{"warn", "warn", slog.LevelWarn, false},
		{"error", "error", slog.LevelError, false},
		{"invalid", "trace", slog.LevelInfo, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, err := parseLevel(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantLevel, level)
			}
		})
	}
}
