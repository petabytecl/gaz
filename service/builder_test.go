package service_test

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/health"
	"github.com/petabytecl/gaz/service"
)

// ServiceBuilderSuite tests the service.Builder API.
type ServiceBuilderSuite struct {
	suite.Suite
}

func TestServiceBuilderSuite(t *testing.T) {
	suite.Run(t, new(ServiceBuilderSuite))
}

func (s *ServiceBuilderSuite) TestBuilder_Basic() {
	// Test: New().Build() returns valid App
	app, err := service.New().Build()

	s.Require().NoError(err)
	s.Require().NotNil(app)
	s.Require().NotNil(app.Container())
}

func (s *ServiceBuilderSuite) TestBuilder_WithConfig() {
	// Test: Config struct is loaded
	type testConfig struct {
		Name string `mapstructure:"name"`
		Port int    `mapstructure:"port"`
	}

	cfg := &testConfig{}

	app, err := service.New().
		WithConfig(cfg).
		Build()

	s.Require().NoError(err)
	s.Require().NotNil(app)

	// Config should be resolvable from container
	resolved, err := gaz.Resolve[*testConfig](app.Container())
	s.Require().NoError(err)
	s.Equal(cfg, resolved)
}

func (s *ServiceBuilderSuite) TestBuilder_WithEnvPrefix() {
	// Test: Env prefix is applied to config
	type testConfig struct {
		Name string `mapstructure:"name"`
	}

	cfg := &testConfig{}

	app, err := service.New().
		WithConfig(cfg).
		WithEnvPrefix("MYAPP").
		Build()

	s.Require().NoError(err)
	s.Require().NotNil(app)
}

func (s *ServiceBuilderSuite) TestBuilder_WithOptions() {
	// Test: gaz.Options are applied to app
	var optionApplied bool

	// We can't directly verify options were applied, but we can test
	// that the builder accepts them without error
	app, err := service.New().
		WithOptions().
		Build()

	s.Require().NoError(err)
	s.Require().NotNil(app)
	// If we reached here, options were accepted (even if empty)
	_ = optionApplied
}

func (s *ServiceBuilderSuite) TestBuilder_Use() {
	// Test: Modules are applied to app
	var moduleApplied bool

	testModule := gaz.NewModule("test-module").
		Provide(func(c *gaz.Container) error {
			moduleApplied = true
			return nil
		}).
		Build()

	app, err := service.New().
		Use(testModule).
		Build()

	s.Require().NoError(err)
	s.Require().NotNil(app)
	s.True(moduleApplied, "module should have been applied")
}

// testConfigWithHealth implements HealthConfigProvider.
type testConfigWithHealth struct {
	AppName string
	Health  health.Config
}

func (c *testConfigWithHealth) HealthConfig() health.Config {
	return c.Health
}

func (s *ServiceBuilderSuite) TestBuilder_HealthAutoRegistration() {
	// Test: Health module auto-registers when config implements HealthConfigProvider
	cfg := &testConfigWithHealth{
		AppName: "test-app",
		Health: health.Config{
			Port:          8081,
			LivenessPath:  "/live",
			ReadinessPath: "/ready",
		},
	}

	app, err := service.New().
		WithConfig(cfg).
		Build()

	s.Require().NoError(err)
	s.Require().NotNil(app)

	// Verify health config was registered
	resolvedCfg, err := gaz.Resolve[health.Config](app.Container())
	s.Require().NoError(err)
	s.Equal(8081, resolvedCfg.Port)
}

func (s *ServiceBuilderSuite) TestBuilder_ChainableMethods() {
	// Test: All methods return *Builder for chaining
	cmd := &cobra.Command{Use: "test"}

	type testConfig struct {
		Name string
	}
	cfg := &testConfig{}

	testModule := gaz.NewModule("chain-test").Build()

	// Build a complex chain - should compile and work
	app, err := service.New().
		WithCmd(cmd).
		WithConfig(cfg).
		WithEnvPrefix("TEST").
		WithOptions().
		Use(testModule).
		Build()

	s.Require().NoError(err)
	s.Require().NotNil(app)
}

func (s *ServiceBuilderSuite) TestBuilder_WithCmd() {
	// Test: Cobra command is attached
	cmd := &cobra.Command{Use: "test"}

	app, err := service.New().
		WithCmd(cmd).
		Build()

	s.Require().NoError(err)
	s.Require().NotNil(app)
	// WithCobra attaches hooks to the command
	// We can verify by checking that PersistentPreRunE is set
	s.NotNil(cmd.PersistentPreRunE, "WithCobra should set PersistentPreRunE")
}

func (s *ServiceBuilderSuite) TestBuilder_NoHealthWithoutProvider() {
	// Test: Health module is NOT registered when config doesn't implement HealthConfigProvider
	type plainConfig struct {
		Name string
	}
	cfg := &plainConfig{Name: "plain"}

	app, err := service.New().
		WithConfig(cfg).
		Build()

	s.Require().NoError(err)
	s.Require().NotNil(app)

	// Health config should NOT be resolvable
	_, err = gaz.Resolve[health.Config](app.Container())
	s.Error(err, "health.Config should not be registered for plain config")
}

func (s *ServiceBuilderSuite) TestBuilder_MultipleModules() {
	// Test: Multiple modules can be added
	var module1Applied, module2Applied bool

	module1 := gaz.NewModule("module-1").
		Provide(func(c *gaz.Container) error {
			module1Applied = true
			return nil
		}).
		Build()

	module2 := gaz.NewModule("module-2").
		Provide(func(c *gaz.Container) error {
			module2Applied = true
			return nil
		}).
		Build()

	app, err := service.New().
		Use(module1).
		Use(module2).
		Build()

	s.Require().NoError(err)
	s.Require().NotNil(app)
	s.True(module1Applied, "module-1 should have been applied")
	s.True(module2Applied, "module-2 should have been applied")
}

// TestBuilder_HealthManagerResolution verifies that when health auto-registers,
// the health.Manager becomes resolvable after app.Build().
func TestBuilder_HealthManagerResolution(t *testing.T) {
	cfg := &testConfigWithHealth{
		AppName: "test-app",
		Health:  health.DefaultConfig(),
	}

	app, err := service.New().
		WithConfig(cfg).
		Build()

	require.NoError(t, err)
	require.NotNil(t, app)

	// Build the app to instantiate eager services
	err = app.Build()
	require.NoError(t, err)

	// Now health.Manager should be resolvable
	manager, err := gaz.Resolve[*health.Manager](app.Container())
	require.NoError(t, err)
	require.NotNil(t, manager)
}
