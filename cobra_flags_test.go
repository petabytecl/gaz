package gaz

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type CobraFlagsSuite struct {
	suite.Suite
}

func TestCobraFlagsSuite(t *testing.T) {
	suite.Run(t, new(CobraFlagsSuite))
}

// testConfigProvider implements ConfigProvider for testing.
type testConfigProvider struct {
	namespace string
	flags     []ConfigFlag
}

func (p *testConfigProvider) ConfigNamespace() string {
	return p.namespace
}

func (p *testConfigProvider) ConfigFlags() []ConfigFlag {
	return p.flags
}

func (s *CobraFlagsSuite) TestRegisterCobraFlagsRegistersFlags() {
	app := New()

	// Register a provider that implements ConfigProvider
	provider := &testConfigProvider{
		namespace: "server",
		flags: []ConfigFlag{
			{Key: "host", Type: ConfigFlagTypeString, Default: "localhost", Description: "Server host"},
			{Key: "port", Type: ConfigFlagTypeInt, Default: 8080, Description: "Server port"},
		},
	}

	err := For[*testConfigProvider](app.Container()).Instance(provider)
	s.Require().NoError(err)

	rootCmd := &cobra.Command{Use: "test"}

	// Register flags
	err = app.RegisterCobraFlags(rootCmd)
	s.Require().NoError(err)

	// Verify flags are registered
	hostFlag := rootCmd.PersistentFlags().Lookup("server-host")
	s.NotNil(hostFlag, "server-host flag should be registered")
	s.Equal("Server host", hostFlag.Usage)
	s.Equal("localhost", hostFlag.DefValue)

	portFlag := rootCmd.PersistentFlags().Lookup("server-port")
	s.NotNil(portFlag, "server-port flag should be registered")
	s.Equal("Server port", portFlag.Usage)
	s.Equal("8080", portFlag.DefValue)
}

func (s *CobraFlagsSuite) TestRegisterCobraFlagsFlagsAppearInHelp() {
	app := New()

	provider := &testConfigProvider{
		namespace: "db",
		flags: []ConfigFlag{
			{Key: "host", Type: ConfigFlagTypeString, Default: "localhost", Description: "Database host"},
		},
	}

	err := For[*testConfigProvider](app.Container()).Instance(provider)
	s.Require().NoError(err)

	rootCmd := &cobra.Command{
		Use:   "myapp",
		Short: "Test app",
		RunE: func(_ *cobra.Command, _ []string) error {
			return nil
		},
	}

	err = app.RegisterCobraFlags(rootCmd)
	s.Require().NoError(err)

	// Capture help output by getting usage string directly
	helpOutput := rootCmd.UsageString()

	s.Contains(helpOutput, "--db-host", "Help should contain --db-host flag")
	s.Contains(helpOutput, "Database host", "Help should contain flag description")
}

func (s *CobraFlagsSuite) TestRegisterCobraFlagsAllTypes() {
	app := New()

	provider := &testConfigProvider{
		namespace: "test",
		flags: []ConfigFlag{
			{Key: "str", Type: ConfigFlagTypeString, Default: "default", Description: "String flag"},
			{Key: "num", Type: ConfigFlagTypeInt, Default: 42, Description: "Int flag"},
			{Key: "flag", Type: ConfigFlagTypeBool, Default: true, Description: "Bool flag"},
			{Key: "dur", Type: ConfigFlagTypeDuration, Default: 30 * time.Second, Description: "Duration flag"},
			{Key: "flt", Type: ConfigFlagTypeFloat, Default: 3.14, Description: "Float flag"},
		},
	}

	err := For[*testConfigProvider](app.Container()).Instance(provider)
	s.Require().NoError(err)

	rootCmd := &cobra.Command{Use: "test"}

	err = app.RegisterCobraFlags(rootCmd)
	s.Require().NoError(err)

	// Verify all flags are registered with correct types
	fs := rootCmd.PersistentFlags()

	strFlag := fs.Lookup("test-str")
	s.NotNil(strFlag)
	s.Equal("string", strFlag.Value.Type())

	numFlag := fs.Lookup("test-num")
	s.NotNil(numFlag)
	s.Equal("int", numFlag.Value.Type())

	boolFlag := fs.Lookup("test-flag")
	s.NotNil(boolFlag)
	s.Equal("bool", boolFlag.Value.Type())

	durFlag := fs.Lookup("test-dur")
	s.NotNil(durFlag)
	s.Equal("duration", durFlag.Value.Type())

	fltFlag := fs.Lookup("test-flt")
	s.NotNil(fltFlag)
	s.Equal("float64", fltFlag.Value.Type())
}

func (s *CobraFlagsSuite) TestRegisterCobraFlagsSkipsDuplicates() {
	app := New()

	provider := &testConfigProvider{
		namespace: "server",
		flags: []ConfigFlag{
			{Key: "host", Type: ConfigFlagTypeString, Default: "localhost", Description: "Server host"},
		},
	}

	err := For[*testConfigProvider](app.Container()).Instance(provider)
	s.Require().NoError(err)

	rootCmd := &cobra.Command{Use: "test"}

	// Pre-register a flag with the same name
	rootCmd.PersistentFlags().String("server-host", "preexisting", "Preexisting host")

	// RegisterCobraFlags should skip the duplicate
	err = app.RegisterCobraFlags(rootCmd)
	s.Require().NoError(err)

	// Original flag should be preserved
	hostFlag := rootCmd.PersistentFlags().Lookup("server-host")
	s.Equal("preexisting", hostFlag.DefValue, "Original flag should be preserved")
}

func (s *CobraFlagsSuite) TestRegisterCobraFlagsIdempotent() {
	app := New()

	provider := &testConfigProvider{
		namespace: "server",
		flags: []ConfigFlag{
			{Key: "host", Type: ConfigFlagTypeString, Default: "localhost", Description: "Server host"},
		},
	}

	err := For[*testConfigProvider](app.Container()).Instance(provider)
	s.Require().NoError(err)

	rootCmd := &cobra.Command{Use: "test"}

	// Call twice - should not error
	err = app.RegisterCobraFlags(rootCmd)
	s.Require().NoError(err)

	err = app.RegisterCobraFlags(rootCmd)
	s.Require().NoError(err) // Second call should succeed (idempotent)
}

func (s *CobraFlagsSuite) TestRegisterCobraFlagsWithBuildIntegration() {
	app := New()

	provider := &testConfigProvider{
		namespace: "server",
		flags: []ConfigFlag{
			{Key: "host", Type: ConfigFlagTypeString, Default: "localhost", Description: "Server host"},
		},
	}

	err := For[*testConfigProvider](app.Container()).Instance(provider)
	s.Require().NoError(err)

	rootCmd := &cobra.Command{Use: "test"}

	// Register flags first
	err = app.RegisterCobraFlags(rootCmd)
	s.Require().NoError(err)

	// Then build - should not duplicate work
	err = app.Build()
	s.Require().NoError(err)

	// Verify provider is still resolvable
	resolved, err := Resolve[*testConfigProvider](app.Container())
	s.Require().NoError(err)
	s.Equal("server", resolved.ConfigNamespace())
}

func (s *CobraFlagsSuite) TestRegisterCobraFlagsWithCobraLifecycle() {
	app := New()

	provider := &testConfigProvider{
		namespace: "app",
		flags: []ConfigFlag{
			{Key: "name", Type: ConfigFlagTypeString, Default: "myapp", Description: "App name"},
		},
	}

	err := For[*testConfigProvider](app.Container()).Instance(provider)
	s.Require().NoError(err)

	var capturedName string

	rootCmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, _ []string) error {
			gotApp := FromContext(cmd.Context())
			pv := MustResolve[*ProviderValues](gotApp.Container())
			capturedName = pv.GetString("app.name")
			return nil
		},
	}

	// Full lifecycle: RegisterCobraFlags -> WithCobra -> Execute
	err = app.RegisterCobraFlags(rootCmd)
	s.Require().NoError(err)

	app.WithCobra(rootCmd)

	rootCmd.SetArgs([]string{})
	execErr := rootCmd.Execute()
	s.Require().NoError(execErr)

	// Default value should be used (no flag override)
	s.Equal("myapp", capturedName)
}

func (s *CobraFlagsSuite) TestRegisterCobraFlagsCliOverride() {
	app := New()

	provider := &testConfigProvider{
		namespace: "server",
		flags: []ConfigFlag{
			{Key: "port", Type: ConfigFlagTypeInt, Default: 8080, Description: "Server port"},
		},
	}

	err := For[*testConfigProvider](app.Container()).Instance(provider)
	s.Require().NoError(err)

	var capturedPort int

	rootCmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, _ []string) error {
			gotApp := FromContext(cmd.Context())
			pv := MustResolve[*ProviderValues](gotApp.Container())
			capturedPort = pv.GetInt("server.port")
			return nil
		},
	}

	err = app.RegisterCobraFlags(rootCmd)
	s.Require().NoError(err)

	app.WithCobra(rootCmd)

	// Pass flag on command line to override default
	rootCmd.SetArgs([]string{"--server-port=9090"})
	execErr := rootCmd.Execute()
	s.Require().NoError(execErr)

	// CLI flag should override default
	s.Equal(9090, capturedPort)
}

func (s *CobraFlagsSuite) TestConfigKeyToFlagName() {
	// Test the key transformation function
	s.Equal("server-host", configKeyToFlagName("server.host"))
	s.Equal("database-pool-size", configKeyToFlagName("database.pool.size"))
	s.Equal("simple", configKeyToFlagName("simple"))
}

func (s *CobraFlagsSuite) TestRegisterCobraFlagsUnknownTypeDefaultsToString() {
	app := New()

	provider := &testConfigProvider{
		namespace: "test",
		flags: []ConfigFlag{
			{Key: "unknown", Type: ConfigFlagType("unknown-type"), Default: "fallback", Description: "Unknown type"},
		},
	}

	err := For[*testConfigProvider](app.Container()).Instance(provider)
	s.Require().NoError(err)

	rootCmd := &cobra.Command{Use: "test"}

	err = app.RegisterCobraFlags(rootCmd)
	s.Require().NoError(err)

	// Unknown type should fallback to string
	flag := rootCmd.PersistentFlags().Lookup("test-unknown")
	s.NotNil(flag)
	s.Equal("string", flag.Value.Type())
	s.Equal("fallback", flag.DefValue)
}

func (s *CobraFlagsSuite) TestRegisterCobraFlagsNoProviders() {
	app := New()

	// No providers registered - should still work
	rootCmd := &cobra.Command{Use: "test"}

	err := app.RegisterCobraFlags(rootCmd)
	s.Require().NoError(err)
}

func (s *CobraFlagsSuite) TestRegisterCobraFlagsMultipleProviders() {
	app := New()

	// Register multiple providers with different namespaces
	serverProvider := &testConfigProvider{
		namespace: "server",
		flags: []ConfigFlag{
			{Key: "host", Type: ConfigFlagTypeString, Default: "localhost", Description: "Server host"},
		},
	}
	dbProvider := &testConfigProvider{
		namespace: "database",
		flags: []ConfigFlag{
			{Key: "host", Type: ConfigFlagTypeString, Default: "db.local", Description: "Database host"},
		},
	}

	err := For[*testConfigProvider](app.Container()).Named("server").Instance(serverProvider)
	s.Require().NoError(err)

	// Need to use a different type or name for the second provider
	err = For[*testConfigProvider](app.Container()).Named("database").Instance(dbProvider)
	s.Require().NoError(err)

	rootCmd := &cobra.Command{Use: "test"}

	err = app.RegisterCobraFlags(rootCmd)
	s.Require().NoError(err)

	// Both should be registered with unique flag names
	serverFlag := rootCmd.PersistentFlags().Lookup("server-host")
	s.NotNil(serverFlag, "server-host should be registered")
	s.Equal("localhost", serverFlag.DefValue)

	dbFlag := rootCmd.PersistentFlags().Lookup("database-host")
	s.NotNil(dbFlag, "database-host should be registered")
	s.Equal("db.local", dbFlag.DefValue)
}

func (s *CobraFlagsSuite) TestRegisterCobraFlagsStringOverride() {
	app := New()

	provider := &testConfigProvider{
		namespace: "app",
		flags: []ConfigFlag{
			{Key: "name", Type: ConfigFlagTypeString, Default: "default-name", Description: "App name"},
		},
	}

	err := For[*testConfigProvider](app.Container()).Instance(provider)
	s.Require().NoError(err)

	var capturedName string

	rootCmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, _ []string) error {
			gotApp := FromContext(cmd.Context())
			pv := MustResolve[*ProviderValues](gotApp.Container())
			capturedName = pv.GetString("app.name")
			return nil
		},
	}

	err = app.RegisterCobraFlags(rootCmd)
	s.Require().NoError(err)

	app.WithCobra(rootCmd)

	// Override via CLI flag
	rootCmd.SetArgs([]string{"--app-name=override-name"})
	execErr := rootCmd.Execute()
	s.Require().NoError(execErr)

	s.Equal("override-name", capturedName)
}

func (s *CobraFlagsSuite) TestRegisterCobraFlagsBoolOverride() {
	app := New()

	provider := &testConfigProvider{
		namespace: "app",
		flags: []ConfigFlag{
			{Key: "debug", Type: ConfigFlagTypeBool, Default: false, Description: "Debug mode"},
		},
	}

	err := For[*testConfigProvider](app.Container()).Instance(provider)
	s.Require().NoError(err)

	var capturedDebug bool

	rootCmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, _ []string) error {
			gotApp := FromContext(cmd.Context())
			pv := MustResolve[*ProviderValues](gotApp.Container())
			capturedDebug = pv.GetBool("app.debug")
			return nil
		},
	}

	err = app.RegisterCobraFlags(rootCmd)
	s.Require().NoError(err)

	app.WithCobra(rootCmd)

	// Override via CLI flag
	rootCmd.SetArgs([]string{"--app-debug=true"})
	execErr := rootCmd.Execute()
	s.Require().NoError(execErr)

	s.True(capturedDebug)
}

func (s *CobraFlagsSuite) TestRegisterCobraFlagsDurationOverride() {
	app := New()

	provider := &testConfigProvider{
		namespace: "app",
		flags: []ConfigFlag{
			{Key: "timeout", Type: ConfigFlagTypeDuration, Default: 30 * time.Second, Description: "Timeout"},
		},
	}

	err := For[*testConfigProvider](app.Container()).Instance(provider)
	s.Require().NoError(err)

	var capturedTimeout time.Duration

	rootCmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, _ []string) error {
			gotApp := FromContext(cmd.Context())
			pv := MustResolve[*ProviderValues](gotApp.Container())
			capturedTimeout = pv.GetDuration("app.timeout")
			return nil
		},
	}

	err = app.RegisterCobraFlags(rootCmd)
	s.Require().NoError(err)

	app.WithCobra(rootCmd)

	// Override via CLI flag
	rootCmd.SetArgs([]string{"--app-timeout=5m"})
	execErr := rootCmd.Execute()
	s.Require().NoError(execErr)

	s.Equal(5*time.Minute, capturedTimeout)
}

func (s *CobraFlagsSuite) TestRegisterCobraFlagsFloatOverride() {
	app := New()

	provider := &testConfigProvider{
		namespace: "app",
		flags: []ConfigFlag{
			{Key: "rate", Type: ConfigFlagTypeFloat, Default: 1.0, Description: "Rate limit"},
		},
	}

	err := For[*testConfigProvider](app.Container()).Instance(provider)
	s.Require().NoError(err)

	var capturedRate float64

	rootCmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, _ []string) error {
			gotApp := FromContext(cmd.Context())
			pv := MustResolve[*ProviderValues](gotApp.Container())
			capturedRate = pv.GetFloat64("app.rate")
			return nil
		},
	}

	err = app.RegisterCobraFlags(rootCmd)
	s.Require().NoError(err)

	app.WithCobra(rootCmd)

	// Override via CLI flag
	rootCmd.SetArgs([]string{"--app-rate=2.5"})
	execErr := rootCmd.Execute()
	s.Require().NoError(execErr)

	s.InDelta(2.5, capturedRate, 0.001)
}
