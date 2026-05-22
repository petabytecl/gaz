package gaz

import (
	"fmt"
	"log/slog"
	"sort"

	"github.com/petabytecl/gaz/cron"
	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/worker"
)

// lifecyclePlan is the structural runtime plan derived from the DI container graph.
// It owns lifecycle policy; App owns execution, logging, rollback, and deadlines.
type lifecyclePlan struct {
	services           map[string]di.ServiceWrapper
	startupOrder       [][]string
	shutdownOrder      [][]string
	workerParticipants []lifecycleWorkerParticipant
	cronJobs           []lifecycleCronJob
}

type lifecycleWorkerParticipant struct {
	serviceName string
}

type lifecycleCronJob struct {
	serviceName string
	transient   bool
}

const runtimeParticipantErrorCapacity = 2

func (a *App) lifecyclePlan() (*lifecyclePlan, error) {
	a.mu.Lock()
	plan := a.cachedLifecyclePlan
	a.mu.Unlock()
	if plan != nil {
		return plan, nil
	}

	planned, err := newLifecyclePlan(a.container)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	if a.cachedLifecyclePlan == nil {
		a.cachedLifecyclePlan = planned
		plan = planned
	} else {
		plan = a.cachedLifecyclePlan
	}
	a.mu.Unlock()

	return plan, nil
}

func newLifecyclePlan(container *Container) (*lifecyclePlan, error) {
	services := collectLifecyclePlanServices(container)

	startupOrder, err := ComputeStartupOrder(container.GetGraph(), services)
	if err != nil {
		return nil, err
	}

	workerParticipants, cronJobs := collectRuntimeParticipants(container)

	return &lifecyclePlan{
		services:           services,
		startupOrder:       startupOrder,
		shutdownOrder:      ComputeShutdownOrder(startupOrder),
		workerParticipants: workerParticipants,
		cronJobs:           cronJobs,
	}, nil
}

func collectLifecyclePlanServices(container *Container) map[string]di.ServiceWrapper {
	services := make(map[string]di.ServiceWrapper)

	container.ForEachService(func(name string, svc di.ServiceWrapper) {
		if svc.IsTransient() {
			return
		}

		if isWorkerParticipant(svc) || isCronJobParticipant(svc) {
			return
		}

		services[name] = svc
	})

	return services
}

func (p *lifecyclePlan) resolveLifecycleServices(container *Container) error {
	names := p.lifecycleServiceNames()
	for _, name := range names {
		if _, err := container.ResolveByName(name, nil); err != nil {
			return fmt.Errorf("lifecycle plan resolving %s: %w", name, err)
		}
	}

	startupOrder, err := ComputeStartupOrder(container.GetGraph(), p.services)
	if err != nil {
		return err
	}

	p.startupOrder = startupOrder
	p.shutdownOrder = ComputeShutdownOrder(startupOrder)
	return nil
}

func (p *lifecyclePlan) lifecycleServiceNames() []string {
	names := make([]string, 0, len(p.services))
	for name, svc := range p.services {
		if svc.HasLifecycle() {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func collectRuntimeParticipants(container *Container) ([]lifecycleWorkerParticipant, []lifecycleCronJob) {
	var workerParticipants []lifecycleWorkerParticipant
	var cronJobs []lifecycleCronJob

	container.ForEachService(func(name string, svc di.ServiceWrapper) {
		if isWorkerParticipant(svc) {
			if isFrameworkEventBus(svc) {
				return
			}
			workerParticipants = append(workerParticipants, lifecycleWorkerParticipant{
				serviceName: name,
			})
			return
		}

		cronJobs = appendCronJobParticipant(name, svc, cronJobs)
	})

	return workerParticipants, cronJobs
}

func isWorkerParticipant(svc di.ServiceWrapper) bool {
	st := svc.ServiceType()
	return st != nil && st.Implements(workerType)
}

func isFrameworkEventBus(svc di.ServiceWrapper) bool {
	st := svc.ServiceType()
	return st == eventBusType
}

func isCronJobParticipant(svc di.ServiceWrapper) bool {
	if svc.TypeName() != di.TypeName[cron.CronJob]() {
		return false
	}

	st := svc.ServiceType()
	return st != nil && st.Implements(cronJobType)
}

func appendCronJobParticipant(
	name string,
	svc di.ServiceWrapper,
	jobs []lifecycleCronJob,
) []lifecycleCronJob {
	if !isCronJobParticipant(svc) {
		return jobs
	}

	return append(jobs, lifecycleCronJob{
		serviceName: name,
		transient:   svc.IsTransient(),
	})
}

func (p *lifecyclePlan) registerRuntimeParticipants(
	container *Container,
	workerMgr *worker.Manager,
	eventBus worker.Worker,
	scheduler *cron.Scheduler,
	log *slog.Logger,
) []error {
	if log == nil {
		log = slog.Default()
	}

	errs := make([]error, 0, runtimeParticipantErrorCapacity)
	errs = append(errs, p.registerWorkers(container, workerMgr)...)
	errs = append(errs, p.registerEventBus(workerMgr, eventBus)...)
	errs = append(errs, p.registerCronJobs(container, scheduler, log)...)
	errs = append(errs, p.registerScheduler(workerMgr, scheduler)...)
	return errs
}

func (p *lifecyclePlan) registerWorkers(
	container *Container,
	workerMgr *worker.Manager,
) []error {
	if workerMgr == nil {
		return nil
	}

	errs := make([]error, 0, len(p.workerParticipants))
	for _, participant := range p.workerParticipants {
		instance, err := container.ResolveByName(participant.serviceName, nil)
		if err != nil {
			errs = append(errs, fmt.Errorf(
				"registering worker participant %s: resolve: %w",
				participant.serviceName,
				err,
			))
			continue
		}

		w, ok := instance.(worker.Worker)
		if !ok {
			errs = append(errs, fmt.Errorf(
				"registering worker participant %s: resolved %T does not implement worker.Worker",
				participant.serviceName,
				instance,
			))
			continue
		}

		if regErr := workerMgr.Register(w); regErr != nil {
			errs = append(errs, fmt.Errorf(
				"registering worker participant %s (worker %s): %w",
				participant.serviceName,
				w.Name(),
				regErr,
			))
		}
	}

	return errs
}

func (p *lifecyclePlan) registerEventBus(
	workerMgr *worker.Manager,
	eventBus worker.Worker,
) []error {
	if workerMgr == nil || eventBus == nil {
		return nil
	}

	if err := workerMgr.Register(eventBus); err != nil {
		return []error{fmt.Errorf("registering eventbus: %w", err)}
	}
	return nil
}

func (p *lifecyclePlan) registerCronJobs(
	container *Container,
	scheduler *cron.Scheduler,
	log *slog.Logger,
) []error {
	if scheduler == nil {
		return nil
	}

	errs := make([]error, 0, len(p.cronJobs))
	for _, participant := range p.cronJobs {
		instance, err := container.ResolveByName(participant.serviceName, nil)
		if err != nil {
			errs = append(errs, fmt.Errorf(
				"registering cron job participant %s: resolve: %w",
				participant.serviceName,
				err,
			))
			continue
		}

		job, ok := instance.(cron.CronJob)
		if !ok {
			errs = append(errs, fmt.Errorf(
				"registering cron job participant %s: resolved %T does not implement cron.CronJob",
				participant.serviceName,
				instance,
			))
			continue
		}

		if !participant.transient {
			log.Warn("CronJob should be transient",
				"name", participant.serviceName,
			)
		}

		if regErr := scheduler.RegisterJob(
			participant.serviceName,
			job.Name(),
			job.Schedule(),
			job.Timeout(),
		); regErr != nil {
			errs = append(errs, fmt.Errorf(
				"registering cron job participant %s (job %s): %w",
				participant.serviceName,
				job.Name(),
				regErr,
			))
		}
	}

	return errs
}

func (p *lifecyclePlan) registerScheduler(
	workerMgr *worker.Manager,
	scheduler *cron.Scheduler,
) []error {
	if workerMgr == nil || scheduler == nil || scheduler.JobCount() == 0 {
		return nil
	}

	if err := workerMgr.Register(scheduler); err != nil {
		return []error{fmt.Errorf("registering scheduler: %w", err)}
	}
	return nil
}
