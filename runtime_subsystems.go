package gaz

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/petabytecl/gaz/cron"
	"github.com/petabytecl/gaz/eventbus"
	"github.com/petabytecl/gaz/worker"
)

// runtimeSubsystems owns construction of the App-managed runtime participants:
// worker manager, cron scheduler, and event bus. Keeping construction in its
// own module makes it testable without a full App instance.
type runtimeSubsystems struct {
	workerMgr  *worker.Manager
	scheduler  *cron.Scheduler
	cronCtx    context.Context
	cronCancel context.CancelFunc
	eventBus   *eventbus.EventBus
}

// runtimeSubsystemsDeps are the inputs needed to construct the runtime subsystems.
type runtimeSubsystemsDeps struct {
	logger          *slog.Logger
	container       *Container
	shutdownTimeout time.Duration
	stopFunc        func(context.Context) error
}

// newRuntimeSubsystems creates worker manager, cron scheduler, and event bus.
// It also registers the event bus in the DI container.
func newRuntimeSubsystems(deps runtimeSubsystemsDeps) (*runtimeSubsystems, error) {
	log := deps.logger
	if log == nil {
		log = slog.Default()
	}

	mgr := worker.NewManager(log)
	mgr.SetCriticalFailHandler(criticalFailHandler(log, deps.shutdownTimeout, deps.stopFunc))

	cronCtx, cronCancel := context.WithCancel(context.Background())
	sched := cron.NewScheduler(deps.container, cronCtx, log)

	bus := eventbus.New(log)

	if err := For[*eventbus.EventBus](deps.container).Instance(bus); err != nil {
		cronCancel()
		return nil, fmt.Errorf("register eventbus: %w", err)
	}

	return &runtimeSubsystems{
		workerMgr:  mgr,
		scheduler:  sched,
		cronCtx:    cronCtx,
		cronCancel: cronCancel,
		eventBus:   bus,
	}, nil
}

func criticalFailHandler(log *slog.Logger, timeout time.Duration, stopFunc func(context.Context) error) func() {
	return func() {
		log.Error("critical worker failed, initiating shutdown")
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			if err := stopFunc(ctx); err != nil {
				log.Warn("critical-fail shutdown completed with errors", slog.Any("error", err))
			}
		}()
	}
}
