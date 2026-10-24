package event

// Priority defines the execution order of event listeners, mirroring PaperMC's EventPriority.
// Listeners with lower priority values execute first, allowing higher priority listeners to override
// or cancel actions before the final outcome is determined.
type Priority int

const (
	// PriorityLowest is executed first. Handlers should use this to set up initial state
	// or perform low-level preparations.
	PriorityLowest Priority = iota
	// PriorityLow is executed second.
	PriorityLow
	// PriorityNormal is the standard default execution priority for event handlers.
	PriorityNormal
	// PriorityHigh is executed fourth, allowing handlers to override standard behavior.
	PriorityHigh
	// PriorityHighest is executed fifth, having the final say in modifying or cancelling the event.
	PriorityHighest
	// PriorityMonitor is executed last. Handlers at this priority MUST NOT modify or cancel
	// the event; it is reserved exclusively for logging, analytics, and outcome observation.
	PriorityMonitor
)

// String returns the uppercase name of the priority.
func (p Priority) String() string {
	switch p {
	case PriorityLowest:
		return "LOWEST"
	case PriorityLow:
		return "LOW"
	case PriorityNormal:
		return "NORMAL"
	case PriorityHigh:
		return "HIGH"
	case PriorityHighest:
		return "HIGHEST"
	case PriorityMonitor:
		return "MONITOR"
	default:
		return "NORMAL"
	}
}
