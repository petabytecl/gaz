package gaz

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInterpretConfigFlags_ExplicitConfigPath(t *testing.T) {
	tmp := t.TempDir()
	cfgFile := filepath.Join(tmp, "config.yaml")
	require.NoError(t, os.WriteFile(cfgFile, []byte("key: value"), 0o644))

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.String("config", cfgFile, "")

	result, err := interpretConfigFlags(fs, "myapp")
	require.NoError(t, err)
	require.NotEmpty(t, result.options)
}

func TestInterpretConfigFlags_MissingConfigPathReturnsError(t *testing.T) {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.String("config", "/nonexistent/config.yaml", "")

	_, err := interpretConfigFlags(fs, "myapp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

func TestInterpretConfigFlags_EmptyConfigPathBuildsSearchPaths(t *testing.T) {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.String("config", "", "")

	result, err := interpretConfigFlags(fs, "myapp")
	require.NoError(t, err)
	require.NotEmpty(t, result.options)
}

func TestInterpretConfigFlags_EnvPrefix(t *testing.T) {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.String("config", "", "")
	fs.String("env-prefix", "MYAPP", "")

	result, err := interpretConfigFlags(fs, "myapp")
	require.NoError(t, err)
	require.NotEmpty(t, result.options)
}

func TestInterpretConfigFlags_StrictTrue(t *testing.T) {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.String("config", "", "")
	fs.Bool("config-strict", true, "")

	result, err := interpretConfigFlags(fs, "myapp")
	require.NoError(t, err)
	require.NotNil(t, result.strictMode)
	assert.True(t, *result.strictMode)
}

func TestInterpretConfigFlags_StrictFalse(t *testing.T) {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.String("config", "", "")
	fs.Bool("config-strict", false, "")

	result, err := interpretConfigFlags(fs, "myapp")
	require.NoError(t, err)
	require.NotNil(t, result.strictMode)
	assert.False(t, *result.strictMode)
}

func TestInterpretConfigFlags_NoConfigFlagIsNoOp(t *testing.T) {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

	result, err := interpretConfigFlags(fs, "myapp")
	require.NoError(t, err)
	assert.Empty(t, result.options)
	assert.Nil(t, result.strictMode)
}

func TestConfigSearchPaths_IncludesCwd(t *testing.T) {
	paths := configSearchPaths("myapp")
	require.NotEmpty(t, paths)
	assert.Equal(t, ".", paths[0])
}

func TestConfigSearchPaths_IncludesXDGDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-test")

	paths := configSearchPaths("myapp")
	require.Len(t, paths, 2)
	assert.Equal(t, "/tmp/xdg-test/myapp", paths[1])
}

func TestConfigSearchPaths_EmptyAppNameSkipsXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-test")

	paths := configSearchPaths("")
	assert.Len(t, paths, 1)
	assert.Equal(t, ".", paths[0])
}
