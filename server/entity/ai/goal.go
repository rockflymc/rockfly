package ai

// ControlFlag is a bitmask representing the physical action channels a Goal controls.
// Goals claiming the same ControlFlag cannot run concurrently unless prioritized.
type ControlFlag uint8

const (
	// ControlMove controls walking, path following, and horizontal locomotion.
	ControlMove ControlFlag = 1 << iota
	// ControlLook controls head rotation, yaw/pitch aiming, and gaze direction.
	ControlLook
	// ControlAttack controls weapon striking, biting, and damage delivery.
	ControlAttack
	// ControlJump controls vertical propulsion and jumping over obstacles.
	ControlJump
)

// Goal defines an individual autonomous behavior for living entities.
type Goal interface {
	// CanUse checks if the goal meets prerequisites to activate this tick.
	CanUse() bool
	// CanContinueToUse checks if an already running goal can remain active this tick.
	CanContinueToUse() bool
	// Start is invoked once when the goal transitions from inactive to running.
	Start()
	// Stop is invoked once when the goal is finished, interrupted, or preempted.
	Stop()
	// Tick executes one behavioral step of the active goal.
	Tick()
	// Flags returns the ControlFlags claimed by this goal.
	Flags() ControlFlag
}

// BaseGoal provides a default no-op implementation of Goal methods.
type BaseGoal struct {
	flags ControlFlag
}

// NewBaseGoal initializes a BaseGoal claiming the specified ControlFlags.
func NewBaseGoal(flags ControlFlag) BaseGoal {
	return BaseGoal{flags: flags}
}

// Flags returns the ControlFlags claimed by this goal.
func (b BaseGoal) Flags() ControlFlag {
	return b.flags
}

// Start default no-op.
func (b BaseGoal) Start() {}

// Stop default no-op.
func (b BaseGoal) Stop() {}

// Tick default no-op.
func (b BaseGoal) Tick() {}
