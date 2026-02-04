package gaz

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/suite"
)

type CobraSuite struct {
	suite.Suite
}

func TestCobraSuite(t *testing.T) {
	suite.Run(t, new(CobraSuite))
}

// cobraTestService is a test helper type for Cobra tests.
type cobraTestService struct {
	name    string
	onStart func()
	onStop  func()
}

// OnStart implements di.Starter for cobraTestService.
func (s *cobraTestService) OnStart(_ context.Context) error {
	if s.onStart != nil {
		s.onStart()
	}
	return nil
}

// OnStop implements di.Stopper for cobraTestService.
func (s *cobraTestService) OnStop(_ context.Context) error {
	if s.onStop != nil {
		s.onStop()
	}
	return nil
}

func (s *CobraSuite) TestWithCobraBuildsAndStartsApp() {
	rootCmd := &cobra.Command{
		Use: "test",
	}

	// Create app with WithCobra option
	app := New(WithCobra(rootCmd))

	var buildCalled bool
	err := For[*cobraTestService](app.Container()).Provider(func(_ *Container) (*cobraTestService, error) {
		buildCalled = true
		return &cobraTestService{name: "test"}, nil
	})
	s.Require().NoError(err)

	// Set RunE that verifies app is in context
	rootCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		// Access app from context
		gotApp := FromContext(cmd.Context())
		s.NotNil(gotApp)
		s.Same(app, gotApp)

		// Resolve service
		svc, err := Resolve[*cobraTestService](gotApp.Container())
		s.Require().NoError(err)
		s.NotNil(svc)
		s.Equal("test", svc.name)

		return nil
	}

	// Execute command
	rootCmd.SetArgs([]string{})
	execErr := rootCmd.Execute()
	s.Require().NoError(execErr)
	s.True(buildCalled, "provider should be called during execution")
}

func (s *CobraSuite) TestWithCobraPreservesExistingHooks() {
	var preRunCalled, postRunCalled bool

	rootCmd := &cobra.Command{
		Use: "test",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			preRunCalled = true
			return nil
		},
		PersistentPostRunE: func(_ *cobra.Command, _ []string) error {
			postRunCalled = true
			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return nil
		},
	}

	_ = New(WithCobra(rootCmd))

	rootCmd.SetArgs([]string{})
	err := rootCmd.Execute()
	s.Require().NoError(err)

	s.True(preRunCalled, "original PersistentPreRunE should be called")
	s.True(postRunCalled, "original PersistentPostRunE should be called")
}

func (s *CobraSuite) TestFromContextReturnsNilWhenNoApp() {
	ctx := context.Background()
	app := FromContext(ctx)
	s.Nil(app)
}

func (s *CobraSuite) TestWithCobraAsOption() {
	rootCmd := &cobra.Command{Use: "test"}

	// WithCobra is now an Option, not a method
	app := New(WithCobra(rootCmd))
	s.NotNil(app)
	s.NotNil(rootCmd.PersistentPreRunE, "WithCobra should set PersistentPreRunE")
}

func (s *CobraSuite) TestWithCobraSubcommandAccess() {
	rootCmd := &cobra.Command{Use: "root"}

	// Create app with WithCobra option
	app := New(WithCobra(rootCmd))

	// Register a service
	err := For[*cobraTestService](app.Container()).Provider(func(_ *Container) (*cobraTestService, error) {
		return &cobraTestService{name: "from-app"}, nil
	})
	s.Require().NoError(err)

	var resolvedName string

	subCmd := &cobra.Command{
		Use: "sub",
		RunE: func(cmd *cobra.Command, _ []string) error {
			gotApp := FromContext(cmd.Context())
			svc, err := Resolve[*cobraTestService](gotApp.Container())
			if err != nil {
				return err
			}
			resolvedName = svc.name
			return nil
		},
	}
	rootCmd.AddCommand(subCmd)

	rootCmd.SetArgs([]string{"sub"})
	execErr := rootCmd.Execute()
	s.Require().NoError(execErr)
	s.Equal("from-app", resolvedName)
}

func (s *CobraSuite) TestWithCobraBuildError() {
	rootCmd := &cobra.Command{
		Use:  "test",
		RunE: func(_ *cobra.Command, _ []string) error { return nil },
	}

	app := New(WithCobra(rootCmd))

	buildErr := errors.New("build failed")

	// Register eager service that fails to start - causes Build error
	err := For[*cobraTestService](app.Container()).Eager().Provider(func(_ *Container) (*cobraTestService, error) {
		return nil, buildErr
	})
	s.Require().NoError(err)

	// Execute command should fail because Build failed
	execErr := rootCmd.Execute()
	s.Require().Error(execErr)
	s.ErrorIs(execErr, buildErr)
}

func (s *CobraSuite) TestWithCobraLifecycleHooksExecuted() {
	rootCmd := &cobra.Command{
		Use:  "test",
		RunE: func(_ *cobra.Command, _ []string) error { return nil },
	}

	app := New(WithCobra(rootCmd))

	var startCalled, stopCalled bool
	// Service implements di.Starter and di.Stopper interfaces - no fluent hooks needed
	err := For[*cobraTestService](app.Container()).Named("test").Eager().
		Provider(func(_ *Container) (*cobraTestService, error) {
			return &cobraTestService{
				onStart: func() { startCalled = true },
				onStop:  func() { stopCalled = true },
			}, nil
		})
	s.Require().NoError(err)

	rootCmd.SetArgs([]string{})
	execErr := rootCmd.Execute()
	s.Require().NoError(execErr)
	s.True(startCalled, "OnStart hook should be called in PreRunE")
	s.True(stopCalled, "OnStop hook should be called in PostRunE")
}

func (s *CobraSuite) TestStartWithoutBuildCallsBuild() {
	app := New()

	regErr := For[*cobraTestService](app.Container()).Provider(func(_ *Container) (*cobraTestService, error) {
		return &cobraTestService{name: "auto-built"}, nil
	})
	s.Require().NoError(regErr)

	// Call Start without Build first - should auto-build
	err := app.Start(context.Background())
	s.Require().NoError(err)

	// Service should be resolvable
	svc, err := Resolve[*cobraTestService](app.Container())
	s.Require().NoError(err)
	s.Equal("auto-built", svc.name)
}

func (s *CobraSuite) TestWithCobraArgsInjection() {
	rootCmd := &cobra.Command{
		Use: "test",
	}

	_ = New(WithCobra(rootCmd))

	var receivedArgs []string

	rootCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		// Verify access via GetArgs
		// Note: We need to use FromContext because app inside RunE might be closure-captured,
		// but we want to simulate real usage.
		gotApp := FromContext(cmd.Context())
		s.NotNil(gotApp)

		args := GetArgs(gotApp.Container())
		receivedArgs = args
		return nil
	}

	// Pass arguments
	expectedArgs := []string{"foo", "bar"}
	rootCmd.SetArgs(expectedArgs)

	err := rootCmd.Execute()
	s.Require().NoError(err)

	s.Equal(expectedArgs, receivedArgs)
}

func (s *CobraSuite) TestWithCobraArgsInjectionToService() {
	rootCmd := &cobra.Command{
		Use: "test",
	}

	_ = New(WithCobra(rootCmd))

	type argsService struct {
		args []string
	}

	// We need to get the app from context since we don't store it
	rootCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		gotApp := FromContext(cmd.Context())
		s.NotNil(gotApp)

		For[*argsService](gotApp.Container()).Provider(func(c *Container) (*argsService, error) {
			cmdArgs, err := Resolve[*CommandArgs](c)
			if err != nil {
				return nil, err
			}
			return &argsService{args: cmdArgs.Args}, nil
		})

		svc, err := Resolve[*argsService](gotApp.Container())
		s.Require().NoError(err)
		s.Equal([]string{"foo", "bar"}, svc.args)
		return nil
	}

	rootCmd.SetArgs([]string{"foo", "bar"})

	err := rootCmd.Execute()
	s.Require().NoError(err)
}

func (s *CobraSuite) TestWithCobraAppliesDeferredFlags() {
	cmd := &cobra.Command{Use: "test"}

	app := New(WithCobra(cmd))

	var flagApplied bool
	// Simulate a flag function registered after WithCobra option
	app.AddFlagsFn(func(flags *pflag.FlagSet) {
		flags.Bool("test-flag", false, "a test flag")
		flagApplied = true
	})

	// Execute command to trigger flag application in PersistentPreRunE
	cmd.RunE = func(_ *cobra.Command, _ []string) error { return nil }
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	s.Require().NoError(err)

	// Verify flag was applied during PersistentPreRunE
	s.True(flagApplied)
	f := cmd.PersistentFlags().Lookup("test-flag")
	s.NotNil(f)
}

func (s *CobraSuite) TestWithCobraInjectsDefaultRunE() {
	cmd := &cobra.Command{Use: "test"}

	// cmd has no Run or RunE initially
	s.Nil(cmd.Run)
	s.Nil(cmd.RunE)

	app := New(WithCobra(cmd))

	// Should now have a RunE
	s.Nil(cmd.Run)
	s.NotNil(cmd.RunE)

	// Verify the default RunE waits for signal
	// We run it in a goroutine and stop the app to unblock it
	done := make(chan error)
	go func() {
		done <- cmd.Execute()
	}()

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	ctx := context.Background()
	err := app.Stop(ctx)
	s.Require().NoError(err)

	select {
	case err := <-done:
		s.Require().NoError(err)
	case <-time.After(1 * time.Second):
		s.Fail("RunE did not return after App.Stop()")
	}
}
