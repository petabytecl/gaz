package gaz

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/pflag"

	"github.com/petabytecl/gaz/config"
	cfgviper "github.com/petabytecl/gaz/config/viper"
)

// configFlagResult holds the output of config flag interpretation.
type configFlagResult struct {
	options    []config.Option
	strictMode *bool // nil when the flag was not explicitly set by the user
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
		if err := validateConfigFile(configPath); err != nil {
			return configFlagResult{}, err
		}
		opts = append(opts, config.WithConfigFile(configPath))
	} else {
		opts = append(opts, config.WithSearchPaths(config.DefaultSearchPaths(appName)...))
	}

	if envPrefixFlag := flags.Lookup("env-prefix"); envPrefixFlag != nil {
		if v := envPrefixFlag.Value.String(); v != "" {
			opts = append(opts, config.WithEnvPrefix(v))
		}
	}

	var strict *bool
	if strictFlag := flags.Lookup("config-strict"); strictFlag != nil && strictFlag.Changed {
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

func validateConfigFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("config: file not found: %s", path)
		}
		return fmt.Errorf("config: cannot access %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("config: path is a directory, not a file: %s", path)
	}
	return nil
}
