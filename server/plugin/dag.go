package plugin

import (
	"fmt"
	"sort"
	"strings"
)

// ResolveDependencies computes the topological startup order for a collection of plugin manifests.
// If A depends on B, B is guaranteed to appear before A in the returned order slice.
// Returns an error if any declared dependency is missing or if a circular dependency is detected.
func ResolveDependencies(manifests map[string]PluginManifest) ([]string, error) {
	inDegree := make(map[string]int)
	graph := make(map[string][]string) // dep -> list of plugins that depend on it

	// Initialize nodes
	for name := range manifests {
		inDegree[name] = 0
		graph[name] = nil
	}

	// Verify dependencies and build graph edges
	for name, m := range manifests {
		for _, dep := range m.Dependencies {
			dep = strings.TrimSpace(dep)
			if dep == "" {
				continue
			}
			if _, exists := manifests[dep]; !exists {
				return nil, fmt.Errorf("plugin %q requires missing dependency %q", name, dep)
			}
			graph[dep] = append(graph[dep], name)
			inDegree[name]++
		}
	}

	// Queue nodes with in-degree 0 (no dependencies)
	var queue []string
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}

	// Deterministic sorting for equal-priority independent plugins
	sort.Strings(queue)

	var order []string

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		order = append(order, curr)

		// Decrement in-degree of dependents
		var nextEligible []string
		for _, dependent := range graph[curr] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				nextEligible = append(nextEligible, dependent)
			}
		}

		sort.Strings(nextEligible)
		queue = append(queue, nextEligible...)
	}

	// If order contains fewer plugins than manifests, there is a cycle
	if len(order) < len(manifests) {
		var cycleNodes []string
		for name, deg := range inDegree {
			if deg > 0 {
				cycleNodes = append(cycleNodes, name)
			}
		}
		sort.Strings(cycleNodes)
		return nil, fmt.Errorf("circular dependency detected in plugin graph involving: %s", strings.Join(cycleNodes, ", "))
	}

	return order, nil
}
