package gaz

import (
	"context"
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
	name string
}

func (s *CobraSuite) TestWithCobraBuildsAndStartsApp() {
	app := New()

	var buildCalled bool
	app.ProvideSingleton(func(_ *Container) (*cobraTestService, error) {
		buildCalled = true
		return &cobraTestService{name: "test"}, nil
	})

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
	err := rootCmd.Execute()
	s.Require().NoError(err)
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
	app.ProvideSingleton(func(_ *Container) (*cobraTestService, error) {
		return &cobraTestService{name: "from-app"}, nil
	})

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
	err := rootCmd.Execute()
	s.Require().NoError(err)
	s.Equal("from-app", resolvedName)
}

func (s *CobraSuite) TestWithCobraBuildError() {
	app := New()

	// Register duplicate to cause build error
	app.ProvideSingleton(func(_ *Container) (*cobraTestService, error) {
		return &cobraTestService{}, nil
	})
	app.ProvideSingleton(func(_ *Container) (*cobraTestService, error) {
		return &cobraTestService{}, nil
	})

	rootCmd := &cobra.Command{
		Use:  "test",
		RunE: func(_ *cobra.Command, _ []string) error { return nil },
	}

	app.WithCobra(rootCmd)

	rootCmd.SetArgs([]string{})
	err := rootCmd.Execute()
	s.Require().Error(err)
	s.Contains(err.Error(), "app build failed")
}

func (s *CobraSuite) TestWithCobraLifecycleHooksExecuted() {
	app := New()

	var startCalled, stopCalled bool
	err := For[*cobraTestService](app.Container()).Named("test").Eager().
		OnStart(func(_ context.Context, _ *cobraTestService) error {
			startCalled = true
			return nil
		}).
		OnStop(func(_ context.Context, _ *cobraTestService) error {
			stopCalled = true
			return nil
		}).
		Provider(func(_ *Container) (*cobraTestService, error) {
			return &cobraTestService{}, nil
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

	app.ProvideSingleton(func(_ *Container) (*cobraTestService, error) {
		return &cobraTestService{name: "auto-built"}, nil
	})

	// Call Start without Build first - should auto-build
	err := app.Start(context.Background())
	s.Require().NoError(err)

	// Service should be resolvable
	svc, err := Resolve[*cobraTestService](app.Container())
	s.Require().NoError(err)
	s.Equal("auto-built", svc.name)
}
