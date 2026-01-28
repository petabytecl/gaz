package gaz

import (
	"errors"
	"sort"

	"github.com/petabytecl/gaz/di"
)

// ComputeStartupOrder calculates the order in which services should be started.
// It returns a list of layers, where each layer contains services that can be started in parallel.
//
// graph: A map where key is the service name and value is the list of dependencies.
// services: A map of service wrappers to check for lifecycle hooks.
func ComputeStartupOrder(
	graph map[string][]string,
	services map[string]di.ServiceWrapper,
) ([][]string, error) {
	// 1. Build reverse graph (dependency -> dependents) and pending counts (dependent -> count)
	reverseGraph := make(map[string][]string)
	pendingCounts := make(map[string]int)

	// Initialize pending counts for all services
	for name := range services {
		pendingCounts[name] = 0
	}

	// Update counts based on dependency graph
	for node, deps := range graph {
		// Ensure node exists in pendingCounts (it should if services map is complete)
		if _, ok := pendingCounts[node]; !ok {
			// If node is in graph but not in services, it might be an issue, but let's track it
			pendingCounts[node] = 0
		}
		pendingCounts[node] = len(deps)

		for _, dep := range deps {
			reverseGraph[dep] = append(reverseGraph[dep], node)
		}
	}

	totalNodes := len(pendingCounts)

	// 2. Initialize queue with nodes having 0 dependencies
	var currentLayer []string
	for node, count := range pendingCounts {
		if count == 0 {
			currentLayer = append(currentLayer, node)
		}
	}

	// Sort for determinism
	sort.Strings(currentLayer)

	var fullOrder [][]string
	processedCount := 0

	// 3. Process layers
	for len(currentLayer) > 0 {
		fullOrder = append(fullOrder, currentLayer)
		processedCount += len(currentLayer)

		var nextLayer []string
		for _, node := range currentLayer {
			// Notify dependents
			for _, dependent := range reverseGraph[node] {
				pendingCounts[dependent]--
				if pendingCounts[dependent] == 0 {
					nextLayer = append(nextLayer, dependent)
				}
			}
		}

		// Sort next layer for determinism
		sort.Strings(nextLayer)
		currentLayer = nextLayer
	}

	// 4. Check for cycles
	if processedCount < totalNodes {
		return nil, errors.New("circular dependency detected")
	}

	// 5. Filter services that don't need lifecycle management
	var filteredOrder [][]string
	for _, layer := range fullOrder {
		var filteredLayer []string
		for _, name := range layer {
			svc, exists := services[name]
			// Only include if service exists and has lifecycle hooks
			if exists && svc.HasLifecycle() {
				filteredLayer = append(filteredLayer, name)
			}
		}
		if len(filteredLayer) > 0 {
			filteredOrder = append(filteredOrder, filteredLayer)
		}
	}

	return filteredOrder, nil
}

// ComputeShutdownOrder reverses the startup order for safe shutdown.
func ComputeShutdownOrder(startupOrder [][]string) [][]string {
	n := len(startupOrder)
	shutdownOrder := make([][]string, n)
	for i, layer := range startupOrder {
		shutdownOrder[n-1-i] = layer
	}
	return shutdownOrder
}
