package gaz

import "github.com/spf13/viper"

// ConfigManager handles configuration loading, binding, and validation.
type ConfigManager struct {
	v           *viper.Viper
	target      any
	fileName    string
	fileType    string
	searchPaths []string
	envPrefix   string
	profileEnv  string
	defaults    map[string]any
}
