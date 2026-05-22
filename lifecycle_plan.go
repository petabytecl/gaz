package gaz

import (
	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/worker"
)

// lifecyclePlan is the pure runtime plan derived from the DI container graph.
// It owns lifecycle policy; App owns execution, logging, rollback, and deadlines.
type lifecyclePlan struct {
	services      map[string]di.ServiceWrapper
	startupOrder  [][]string
	shutdownOrder [][]string
}

func newLifecyclePlan(container *Container) (*lifecyclePlan, error) {
	services := collectLifecyclePlanServices(container)

	startupOrder, err := ComputeStartupOrder(container.GetGraph(), services)
	if err != nil {
		return nil, err
	}

	return &lifecyclePlan{
		services:      services,
		startupOrder:  startupOrder,
		shutdownOrder: ComputeShutdownOrder(startupOrder),
	}, nil
}

func collectLifecyclePlanServices(container *Container) map[string]di.ServiceWrapper {
	services := make(map[string]di.ServiceWrapper)
	container.ForEachService(func(name string, svc di.ServiceWrapper) {
		if svc.IsTransient() {
			return
		}
		if resolvesAsWorker(container, name) {
			return
		}
		services[name] = svc
	})
	return services
}

func resolvesAsWorker(container *Container, name string) bool {
	instance, err := container.ResolveByName(name, nil)
	if err != nil {
		return false
	}
	_, isWorker := instance.(worker.Worker)
	return isWorker
}
