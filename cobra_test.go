package gaz

import (
	"context"
	"errors"
	"testing"

	"github.com/spf13/cobra"
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
	app := New()

	var buildCalled bool
	err := For[*cobraTestService](app.Container()).Provider(func(_ *Container) (*cobraTestService, error) {
		buildCalled = true
		return &cobraTestService{name: "test"}, nil
	})
	s.Require().NoError(err)

	rootCmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, _ []string) error {
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
		},
	}

	app.WithCobra(rootCmd)

	// Execute command
	rootCmd.SetArgs([]string{})
	execErr := rootCmd.Execute()
	s.Require().NoError(execErr)
	s.True(buildCalled, "provider should be called during execution")
}

func (s *CobraSuite) TestWithCobraPreservesExistingHooks() {
	app := New()

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

	app.WithCobra(rootCmd)

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

func (s *CobraSuite) TestWithCobraChaining() {
	app := New()
	rootCmd := &cobra.Command{Use: "test"}

	result := app.WithCobra(rootCmd)
	s.Same(app, result) // Returns same app for chaining
}

func (s *CobraSuite) TestWithCobraSubcommandAccess() {
	app := New()

	// Register a service
	err := For[*cobraTestService](app.Container()).Provider(func(_ *Container) (*cobraTestService, error) {
		return &cobraTestService{name: "from-app"}, nil
	})
	s.Require().NoError(err)

	var resolvedName string

	rootCmd := &cobra.Command{Use: "root"}
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

	app.WithCobra(rootCmd)

	rootCmd.SetArgs([]string{"sub"})
	execErr := rootCmd.Execute()
	s.Require().NoError(execErr)
	s.Equal("from-app", resolvedName)
}

func (s *CobraSuite) TestWithCobraBuildError() {
	app := New()

	buildErr := errors.New("build failed")

	// Register eager service that fails to start - causes Build error
	err := For[*cobraTestService](app.Container()).Eager().Provider(func(_ *Container) (*cobraTestService, error) {
		return nil, buildErr
	})
	s.Require().NoError(err)

	rootCmd := &cobra.Command{
		Use:  "test",
		RunE: func(_ *cobra.Command, _ []string) error { return nil },
	}

	app.WithCobra(rootCmd)

	// Execute command should fail because Build failed
	execErr := rootCmd.Execute()
	s.Require().Error(execErr)
	s.ErrorIs(execErr, buildErr)
}

func (s *CobraSuite) TestWithCobraLifecycleHooksExecuted() {
	app := New()

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

	rootCmd := &cobra.Command{
		Use:  "test",
		RunE: func(_ *cobra.Command, _ []string) error { return nil },
	}

	app.WithCobra(rootCmd)

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
	app := New()

	var receivedArgs []string

	rootCmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Verify access via GetArgs
			// Note: We need to use FromContext because app inside RunE might be closure-captured,
			// but we want to simulate real usage.
			gotApp := FromContext(cmd.Context())
			s.NotNil(gotApp)

			args := GetArgs(gotApp.Container())
			receivedArgs = args
			return nil
		},
	}

	app.WithCobra(rootCmd)

	// Pass arguments
	expectedArgs := []string{"foo", "bar"}
	rootCmd.SetArgs(expectedArgs)

	err := rootCmd.Execute()
	s.Require().NoError(err)

	s.Equal(expectedArgs, receivedArgs)
}

func (s *CobraSuite) TestWithCobraArgsInjectionToService() {
	app := New()

	type argsService struct {
		args []string
	}

	For[*argsService](app.Container()).Provider(func(c *Container) (*argsService, error) {
		cmdArgs, err := Resolve[*CommandArgs](c)
		if err != nil {
			return nil, err
		}
		return &argsService{args: cmdArgs.Args}, nil
	})

	rootCmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, _ []string) error {
			gotApp := FromContext(cmd.Context())
			s.NotNil(gotApp)

			svc, err := Resolve[*argsService](gotApp.Container())
			s.Require().NoError(err)
			s.Equal([]string{"foo", "bar"}, svc.args)
			return nil
		},
	}

	app.WithCobra(rootCmd)
	rootCmd.SetArgs([]string{"foo", "bar"})

	err := rootCmd.Execute()
	s.Require().NoError(err)
}
