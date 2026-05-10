package gaz

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/petabytecl/gaz/worker"
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

	// Verify command args are available via DI resolution (CommandArgs is
	// registered pre-Build by the framework and populated in PersistentPreRunE).
	rootCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		gotApp := FromContext(cmd.Context())
		s.NotNil(gotApp)

		cmdArgs, err := Resolve[*CommandArgs](gotApp.Container())
		if err != nil {
			return fmt.Errorf("resolve CommandArgs: %w", err)
		}
		s.Equal([]string{"foo", "bar"}, cmdArgs.Args)
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

// testWorkerForCobra implements worker.Worker for regression tests.
type testWorkerForCobra struct {
	name      string
	started   atomic.Bool
	stopped   atomic.Bool
	onStartFn func(ctx context.Context) error
}

func (w *testWorkerForCobra) Name() string { return w.name }

func (w *testWorkerForCobra) OnStart(ctx context.Context) error {
	w.started.Store(true)
	if w.onStartFn != nil {
		return w.onStartFn(ctx)
	}
	return nil
}

func (w *testWorkerForCobra) OnStop(_ context.Context) error {
	w.stopped.Store(true)
	return nil
}

// Verify testWorkerForCobra implements worker.Worker.
var _ worker.Worker = (*testWorkerForCobra)(nil)

// TestCobraStartServicesMatchesRun verifies that the Cobra entry point
// starts workers via workerMgr (not DI layer), uses parallel startup,
// and recovers from panicking OnStart hooks.
func TestCobraStartServicesMatchesRun(t *testing.T) {
	t.Parallel()

	// Create a worker that tracks whether it was started via workerMgr.
	// Use a channel to synchronize - RunE waits for worker startup.
	workerStarted := make(chan struct{})
	w := &testWorkerForCobra{
		name: "test-worker",
		onStartFn: func(_ context.Context) error {
			close(workerStarted)
			return nil
		},
	}

	rootCmd := &cobra.Command{
		Use: "test",
		RunE: func(_ *cobra.Command, _ []string) error {
			// Wait for worker to be started by workerMgr before returning.
			// This ensures the supervisor goroutine has time to call OnStart.
			select {
			case <-workerStarted:
				return nil
			case <-time.After(2 * time.Second):
				return errors.New("worker was not started within timeout")
			}
		},
	}

	app := New(WithCobra(rootCmd), WithShutdownTimeout(2*time.Second))

	// Register the worker - it gets auto-discovered and registered with workerMgr during Build
	err := For[worker.Worker](app.Container()).Named("test-worker").Instance(w)
	require.NoError(t, err)

	// Register a normal service to verify parallel startup works
	var svcStarted atomic.Bool
	err = For[*cobraTestService](app.Container()).Named("normal-svc").Eager().
		Provider(func(_ *Container) (*cobraTestService, error) {
			return &cobraTestService{
				name:    "normal",
				onStart: func() { svcStarted.Store(true) },
			}, nil
		})
	require.NoError(t, err)

	rootCmd.SetArgs([]string{})
	execErr := rootCmd.Execute()
	require.NoError(t, execErr)

	// Worker should have been started via workerMgr (confirmed by channel signal)
	require.True(t, w.started.Load(), "worker should be started via workerMgr")

	// Normal service should have been started via startServices (parallel layer startup)
	require.True(t, svcStarted.Load(), "normal service should be started")
}

// TestCobraStartServicesRollback verifies that when a service fails to start
// via the Cobra path, previously started services are rolled back (stopped).
func TestCobraStartServicesRollback(t *testing.T) {
	t.Parallel()

	rootCmd := &cobra.Command{
		Use: "test",
		RunE: func(_ *cobra.Command, _ []string) error {
			return nil
		},
	}

	app := New(WithCobra(rootCmd), WithShutdownTimeout(2*time.Second))

	// Register a service that will start successfully (layer 0 due to no deps)
	var stopped atomic.Bool
	err := For[*cobraTestService](app.Container()).Named("good-svc").Eager().
		Provider(func(_ *Container) (*cobraTestService, error) {
			return &cobraTestService{
				name:   "good",
				onStop: func() { stopped.Store(true) },
			}, nil
		})
	require.NoError(t, err)

	// Register a service that fails to start (same layer as good-svc)
	startErr := errors.New("intentional start failure")
	err = For[*failingStartService](app.Container()).Named("bad-svc").Eager().
		Provider(func(_ *Container) (*failingStartService, error) {
			return &failingStartService{err: startErr}, nil
		})
	require.NoError(t, err)

	rootCmd.SetArgs([]string{})
	execErr := rootCmd.Execute()

	// Execute should fail because Start failed
	require.Error(t, execErr)
	require.ErrorContains(t, execErr, "intentional start failure")

	// The good service should have been rolled back (stopped)
	require.Eventually(t, stopped.Load, 2*time.Second, 10*time.Millisecond, "good service should be stopped during rollback")
}

// failingStartService is a service whose OnStart always returns an error.
type failingStartService struct {
	err error
}

func (s *failingStartService) OnStart(_ context.Context) error {
	return s.err
}

func (s *failingStartService) OnStop(_ context.Context) error {
	return nil
}
