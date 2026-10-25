package ai

import (
	"slices"
	"sync"
)

// PrioritizedGoal pairs a Goal with an integer priority (lower value = higher precedence).
type PrioritizedGoal struct {
	Priority int
	Goal     Goal
	Running  bool
}

// GoalSelector coordinates and prioritizes concurrent mob goals while resolving
// ControlFlag resource conflicts.
type GoalSelector struct {
	mu    sync.Mutex
	goals []*PrioritizedGoal
}

// NewGoalSelector creates an empty GoalSelector.
func NewGoalSelector() *GoalSelector {
	return &GoalSelector{}
}

// AddGoal registers a new goal at the given priority. Lower numbers have higher priority.
func (s *GoalSelector) AddGoal(priority int, g Goal) {
	if s == nil || g == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	pg := &PrioritizedGoal{
		Priority: priority,
		Goal:     g,
		Running:  false,
	}
	s.goals = append(s.goals, pg)
	s.sortGoals()
}

// RemoveGoal removes a registered goal from the selector, stopping it if currently running.
func (s *GoalSelector) RemoveGoal(g Goal) {
	if s == nil || g == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := slices.IndexFunc(s.goals, func(pg *PrioritizedGoal) bool {
		return pg.Goal == g
	})
	if idx >= 0 {
		if s.goals[idx].Running {
			s.goals[idx].Goal.Stop()
		}
		s.goals = slices.Delete(s.goals, idx, idx+1)
	}
}

// Tick executes a single tick of the goal selector.
// High priority goals can preempt lower priority running goals claiming the same ControlFlags.
func (s *GoalSelector) Tick() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Evaluate continuing suitability of currently running goals
	for _, pg := range s.goals {
		if pg.Running && !pg.Goal.CanContinueToUse() {
			pg.Goal.Stop()
			pg.Running = false
		}
	}

	// 2. Schedule eligible goals by priority, resolving control flag conflicts
	var claimedFlags ControlFlag

	for _, pg := range s.goals {
		flags := pg.Goal.Flags()

		if pg.Running {
			// If a higher priority goal already claimed our control flags, we are preempted
			if (claimedFlags & flags) != 0 {
				pg.Goal.Stop()
				pg.Running = false
				continue
			}
			// Still uncontested: continue claiming flags
			claimedFlags |= flags
			continue
		}

		// Not running: check if goal desires to activate
		if !pg.Goal.CanUse() {
			continue
		}

		// Check if any required control flag is already claimed by a higher-priority goal
		if (claimedFlags & flags) != 0 {
			// Flag conflict with a higher priority goal: cannot activate
			continue
		}

		// Activate goal and claim its flags
		pg.Goal.Start()
		pg.Running = true
		claimedFlags |= flags
	}

	// 3. Tick all active running goals
	for _, pg := range s.goals {
		if pg.Running {
			pg.Goal.Tick()
		}
	}
}

// RunningGoals returns all goals currently in active execution.
func (s *GoalSelector) RunningGoals() []Goal {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	var running []Goal
	for _, pg := range s.goals {
		if pg.Running {
			running = append(running, pg.Goal)
		}
	}
	return running
}

func (s *GoalSelector) sortGoals() {
	slices.SortStableFunc(s.goals, func(a, b *PrioritizedGoal) int {
		return a.Priority - b.Priority
	})
}
