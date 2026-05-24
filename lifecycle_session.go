package gaz

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// lifecycleSession owns one App runtime session. App remains the public facade;
// this module owns running-state, shutdown triggers, and one-shot stop policy.
type lifecycleSession struct {
	app *App
}

func newLifecycleSession(app *App) *lifecycleSession {
	return &lifecycleSession{app: app}
}

func (a *App) lifecycleSession() *lifecycleSession {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.session == nil {
		a.session = newLifecycleSession(a)
	}
	return a.session
}

func (s *lifecycleSession) Run(ctx context.Context) error {
	if err := s.app.Build(); err != nil {
		return err
	}

	if err := s.enterRunning(); err != nil {
		return err
	}
	defer s.leaveRunning()

	if err := s.startServices(ctx); err != nil {
		return err
	}

	return s.waitForShutdownSignal(ctx)
}

func (s *lifecycleSession) Start(ctx context.Context) error {
	a := s.app

	a.mu.Lock()
	if !a.built {
		a.mu.Unlock()
		if err := a.Build(); err != nil {
			return err
		}
		a.mu.Lock()
	}
	a.mu.Unlock()

	return s.startServices(ctx)
}

func (s *lifecycleSession) Stop(ctx context.Context) error {
	s.app.stopOnce.Do(func() {
		s.app.stopErr = s.doStop(ctx)
	})
	return s.app.stopErr
}

func (s *lifecycleSession) startServices(ctx context.Context) error {
	plan, err := s.app.lifecyclePlan()
	if err != nil {
		return err
	}

	return s.app.lifecycleExecutor(plan).Start(ctx, s.Stop)
}

func (s *lifecycleSession) makePreRunE(
	original func(*cobra.Command, []string) error,
) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, args []string) error {
		if original != nil {
			if err := original(c, args); err != nil {
				return err
			}
		}

		ctx := c.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		if err := s.bootstrap(ctx, c, args); err != nil {
			return err
		}

		c.SetContext(context.WithValue(ctx, contextKey{}, s.app))
		return nil
	}
}

func (s *lifecycleSession) makePostRunE(
	original func(*cobra.Command, []string) error,
) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, args []string) error {
		stopCtx, cancel := context.WithTimeout(context.Background(), s.app.opts.ShutdownTimeout)
		defer cancel()

		stopErr := s.Stop(stopCtx)
		s.leaveRunning()

		if original != nil {
			if err := original(c, args); err != nil {
				return errors.Join(stopErr, err)
			}
		}

		return stopErr
	}
}

func (s *lifecycleSession) bootstrap(ctx context.Context, cmd *cobra.Command, args []string) error {
	_ = For[*CommandArgs](s.app.container).Instance(&CommandArgs{
		Command: cmd,
		Args:    args,
	})

	if s.app.configMgr != nil {
		if err := s.app.configMgr.BindFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("failed to bind flags: %w", err)
		}
	}

	if err := s.enterRunning(); err != nil {
		return err
	}

	success := false
	defer func() {
		if !success {
			s.leaveRunning()
		}
	}()

	if err := s.app.Build(); err != nil {
		return fmt.Errorf("app build failed: %w", err)
	}

	if err := s.Start(ctx); err != nil {
		return fmt.Errorf("app start failed: %w", err)
	}

	success = true
	return nil
}

func (s *lifecycleSession) enterRunning() error {
	s.app.mu.Lock()
	defer s.app.mu.Unlock()

	if s.app.running {
		return errors.New("app is already running")
	}
	s.app.stopCh = make(chan struct{})
	s.app.running = true
	return nil
}

func (s *lifecycleSession) leaveRunning() {
	s.app.mu.Lock()
	s.app.running = false
	s.app.mu.Unlock()
}

func (s *lifecycleSession) waitForShutdownSignal(ctx context.Context) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case <-ctx.Done():
		s.app.Logger.InfoContext(ctx, "Shutting down gracefully...", "reason", "context cancelled")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.app.opts.ShutdownTimeout)
		defer cancel()
		return s.Stop(shutdownCtx)

	case sig := <-sigCh:
		return s.handleSignalShutdown(ctx, sig, sigCh)

	case <-s.app.stopCh:
		return nil
	}
}

func (s *lifecycleSession) handleSignalShutdown(
	ctx context.Context,
	sig os.Signal,
	sigCh <-chan os.Signal,
) error {
	if sig == os.Interrupt {
		s.app.Logger.InfoContext(ctx, "Shutting down gracefully...", "hint", "Ctrl+C again to force")
	} else {
		s.app.Logger.InfoContext(ctx, "Shutting down gracefully...", "signal", sig.String())
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.app.opts.ShutdownTimeout)
	defer cancel()

	shutdownErr := make(chan error, 1)
	shutdownDone := make(chan struct{})

	go func() {
		defer close(shutdownDone)
		shutdownErr <- s.Stop(shutdownCtx)
	}()

	watcherDone := make(chan struct{})
	if sig == os.Interrupt {
		go func() {
			defer close(watcherDone)
			select {
			case <-sigCh:
				s.app.Logger.ErrorContext(ctx, "Received second interrupt, forcing exit")
				callExitFunc(1)
			case <-shutdownDone:
			}
		}()
	} else {
		close(watcherDone)
	}

	err := <-shutdownErr
	<-watcherDone
	return err
}

func (s *lifecycleSession) doStop(ctx context.Context) error {
	wasRunning, wasBuilt := s.stopState()
	if !wasBuilt {
		return nil
	}

	if s.app.cronCancel != nil {
		s.app.cronCancel()
	}

	done := s.startForceExitTimer()

	errs, err := s.stopPlannedLifecycle(ctx)
	if err != nil {
		close(done)
		return err
	}

	if closeErr := s.closeLogger(); closeErr != nil {
		errs = append(errs, closeErr)
	}

	close(done)
	s.signalRunStopped(wasRunning)

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (s *lifecycleSession) stopState() (bool, bool) {
	s.app.mu.Lock()
	wasRunning := s.app.running
	wasBuilt := s.app.built
	s.app.mu.Unlock()
	return wasRunning, wasBuilt
}

func (s *lifecycleSession) startForceExitTimer() chan struct{} {
	done := make(chan struct{})
	timer := time.NewTimer(s.app.opts.ShutdownTimeout)
	go func() {
		select {
		case <-done:
			timer.Stop()
			return
		case <-timer.C:
			select {
			case <-done:
				return
			default:
				msg := fmt.Sprintf(
					"shutdown: global timeout %s exceeded, forcing exit",
					s.app.opts.ShutdownTimeout,
				)
				s.app.getLogger().Error(msg)
				_, _ = fmt.Fprintln(os.Stderr, msg)
				callExitFunc(1)
			}
		}
	}()
	return done
}

func (s *lifecycleSession) stopPlannedLifecycle(ctx context.Context) ([]error, error) {
	plan, err := s.app.lifecyclePlan()
	if err != nil {
		return nil, err
	}

	var errs []error
	if stopErr := s.app.lifecycleExecutor(plan).Stop(ctx); stopErr != nil {
		errs = append(errs, stopErr)
	}

	return errs, nil
}

func (s *lifecycleSession) closeLogger() error {
	if s.app.logCloser != nil {
		if closeErr := s.app.logCloser.Close(); closeErr != nil {
			return fmt.Errorf("closing logger: %w", closeErr)
		}
	}
	return nil
}

func (s *lifecycleSession) signalRunStopped(wasRunning bool) {
	if wasRunning {
		s.app.mu.Lock()
		select {
		case <-s.app.stopCh:
		default:
			close(s.app.stopCh)
		}
		s.app.mu.Unlock()
	}
}
