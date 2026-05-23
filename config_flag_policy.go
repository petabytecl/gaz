package gaz

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"

	"github.com/petabytecl/gaz/config"
	cfgviper "github.com/petabytecl/gaz/config/viper"
)

// configFlagResult holds the output of config flag interpretation.
type configFlagResult struct {
	options    []config.Option
	strictMode *bool // nil = unchanged from app default
}

// interpretConfigFlags reads Cobra config flags (--config, --env-prefix,
// --config-strict) and produces config manager options and strict-mode state.
// This keeps flag interpretation logic out of App, making it testable
// without constructing a full App instance.
func interpretConfigFlags(flags *pflag.FlagSet, appName string) (configFlagResult, error) {
	configFlag := flags.Lookup("config")
	if configFlag == nil {
		return configFlagResult{}, nil
	}

	opts := []config.Option{config.WithBackend(cfgviper.New())}

	configPath := configFlag.Value.String()
	if configPath != "" {
		if _, err := os.Stat(configPath); err != nil {
			return configFlagResult{}, fmt.Errorf("config: file not found: %s", configPath)
		}
		opts = append(opts, config.WithConfigFile(configPath))
	} else {
		opts = append(opts, config.WithSearchPaths(configSearchPaths(appName)...))
	}

	if envPrefixFlag := flags.Lookup("env-prefix"); envPrefixFlag != nil {
		if v := envPrefixFlag.Value.String(); v != "" {
			opts = append(opts, config.WithEnvPrefix(v))
		}
	}

	var strict *bool
	if strictFlag := flags.Lookup("config-strict"); strictFlag != nil {
		switch strictFlag.Value.String() {
		case "true":
			v := true
			strict = &v
		case "false":
			v := false
			strict = &v
		}
	}

	return configFlagResult{options: opts, strictMode: strict}, nil
}

func configSearchPaths(appName string) []string {
	paths := []string{"."}

	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		if home, err := os.UserHomeDir(); err == nil {
			xdgConfig = filepath.Join(home, ".config")
		}
	}
	if xdgConfig != "" && appName != "" {
		paths = append(paths, filepath.Join(xdgConfig, appName))
	}

	return paths
}
