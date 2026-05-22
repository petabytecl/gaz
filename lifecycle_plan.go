package gaz

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/petabytecl/gaz/cron"
	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/worker"
)

// lifecyclePlan is the pure runtime plan derived from the DI container graph.
// It owns lifecycle policy; App owns execution, logging, rollback, and deadlines.
type lifecyclePlan struct {
	services           map[string]di.ServiceWrapper
	startupOrder       [][]string
	shutdownOrder      [][]string
	workerParticipants []worker.Worker
	cronJobs           []lifecycleCronJob
}

type lifecycleCronJob struct {
	serviceName string
	jobName     string
	schedule    string
	timeout     time.Duration
	transient   bool
}

const runtimeParticipantErrorCapacity = 2

func newLifecyclePlan(container *Container) (*lifecyclePlan, error) {
	services, workerParticipants, cronJobs, err := collectLifecyclePlanParticipants(
		container,
		true,
	)
	if err != nil {
		return nil, err
	}

	startupOrder, err := ComputeStartupOrder(container.GetGraph(), services)
	if err != nil {
		return nil, err
	}

	return &lifecyclePlan{
		services:           services,
		startupOrder:       startupOrder,
		shutdownOrder:      ComputeShutdownOrder(startupOrder),
		workerParticipants: workerParticipants,
		cronJobs:           cronJobs,
	}, nil
}

func newRuntimeParticipantPlan(container *Container) *lifecyclePlan {
	_, workerParticipants, cronJobs, _ := collectLifecyclePlanParticipants(container, false)
	return &lifecyclePlan{
		workerParticipants: workerParticipants,
		cronJobs:           cronJobs,
	}
}

func collectLifecyclePlanParticipants(
	container *Container,
	resolveLifecycle bool,
) (map[string]di.ServiceWrapper, []worker.Worker, []lifecycleCronJob, error) {
	services := make(map[string]di.ServiceWrapper)
	var workerParticipants []worker.Worker
	var cronJobs []lifecycleCronJob
	var resolveErr error

	container.ForEachService(func(name string, svc di.ServiceWrapper) {
		if resolveErr != nil {
			return
		}
		if svc.IsTransient() {
			cronJobs = appendCronJobParticipant(container, name, svc, cronJobs)
			return
		}

		if isWorkerParticipant(svc) {
			if isFrameworkEventBus(svc) {
				return
			}
			workerParticipants = appendWorkerParticipant(container, name, workerParticipants)
			return
		}

		cronJobs = appendCronJobParticipant(container, name, svc, cronJobs)
		if resolveLifecycle && svc.HasLifecycle() {
			if _, err := container.ResolveByName(name, nil); err != nil {
				resolveErr = fmt.Errorf("lifecycle plan resolving %s: %w", name, err)
				return
			}
		}
		services[name] = svc
	})

	return services, workerParticipants, cronJobs, resolveErr
}

func isWorkerParticipant(svc di.ServiceWrapper) bool {
	st := svc.ServiceType()
	return st != nil && st.Implements(workerType)
}

func isFrameworkEventBus(svc di.ServiceWrapper) bool {
	st := svc.ServiceType()
	return st == eventBusType
}

func appendWorkerParticipant(
	container *Container,
	name string,
	participants []worker.Worker,
) []worker.Worker {
	instance, err := container.ResolveByName(name, nil)
	if err != nil {
		return participants
	}
	if w, ok := instance.(worker.Worker); ok {
		return append(participants, w)
	}
	return participants
}

func appendCronJobParticipant(
	container *Container,
	name string,
	svc di.ServiceWrapper,
	jobs []lifecycleCronJob,
) []lifecycleCronJob {
	if svc.TypeName() != di.TypeName[cron.CronJob]() {
		return jobs
	}

	st := svc.ServiceType()
	if st == nil || !st.Implements(cronJobType) {
		return jobs
	}

	instance, err := container.ResolveByName(name, nil)
	if err != nil {
		return jobs
	}

	job, ok := instance.(cron.CronJob)
	if !ok {
		return jobs
	}

	return append(jobs, lifecycleCronJob{
		serviceName: name,
		jobName:     job.Name(),
		schedule:    job.Schedule(),
		timeout:     job.Timeout(),
		transient:   svc.IsTransient(),
	})
}

func (p *lifecyclePlan) registerRuntimeParticipants(
	workerMgr *worker.Manager,
	eventBus worker.Worker,
	scheduler *cron.Scheduler,
	log *slog.Logger,
) []error {
	if log == nil {
		log = slog.Default()
	}

	errs := make([]error, 0, runtimeParticipantErrorCapacity)
	p.registerWorkers(workerMgr, log)
	errs = append(errs, p.registerEventBus(workerMgr, eventBus)...)
	p.registerCronJobs(scheduler, log)
	errs = append(errs, p.registerScheduler(workerMgr, scheduler)...)
	return errs
}

func (p *lifecyclePlan) registerWorkers(workerMgr *worker.Manager, log *slog.Logger) {
	if workerMgr == nil {
		return
	}

	for _, w := range p.workerParticipants {
		if regErr := workerMgr.Register(w); regErr != nil {
			log.Warn("failed to register worker",
				"name", w.Name(),
				"error", regErr,
			)
		}
	}
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

func (p *lifecyclePlan) registerCronJobs(scheduler *cron.Scheduler, log *slog.Logger) {
	if scheduler == nil {
		return
	}

	for _, job := range p.cronJobs {
		if !job.transient {
			log.Warn("CronJob should be transient",
				"name", job.serviceName,
			)
		}

		if regErr := scheduler.RegisterJob(
			job.serviceName,
			job.jobName,
			job.schedule,
			job.timeout,
		); regErr != nil {
			log.Warn("failed to register cron job",
				"name", job.jobName,
				"error", regErr,
			)
		}
	}
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
