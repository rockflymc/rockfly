package passive

import (
	"math/rand"
	"sync/atomic"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/ai"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// Wolf represents a domesticable predator mob capable of taming with bones,
// collar customization, sitting commands, and defending its master against hostiles.
type Wolf struct {
	*Animal

	tamed        atomic.Bool
	sitting      atomic.Bool
	angry        atomic.Bool
	owner        atomic.Pointer[string]
	collarColour item.Colour
}

// NewWolf initializes a wild or tamed Wolf entity.
func NewWolf(handle *world.EntityHandle, baby bool) *Wolf {
	w := &Wolf{
		Animal:       NewAnimal(handle, 8.0, baby), // 8.0 health wild
		collarColour: item.ColourRed(),
	}
	w.SetSpeed(0.30)

	w.GoalSelector().AddGoal(1, ai.NewFloatGoal(w, func() bool { return false }))
	w.GoalSelector().AddGoal(2, ai.NewRandomStrollGoal(w, 10))
	w.GoalSelector().AddGoal(3, ai.NewLookAtEntityGoal(w, 8.0, nil))

	return w
}

// Tamed returns true if the wolf has been domesticated by a player.
func (w *Wolf) Tamed() bool {
	return w.tamed.Load()
}

// SetTamed sets the domesticated state and scales maximum health to 20.0.
func (w *Wolf) SetTamed(tamed bool, owner string) {
	w.tamed.Store(tamed)
	if tamed {
		w.owner.Store(&owner)
		w.SetMaxHealth(20.0)
		w.SetHealth(20.0)
		w.SetSpeed(0.30)
	} else {
		w.owner.Store(nil)
		w.SetMaxHealth(8.0)
		w.SetHealth(8.0)
	}
}

// Owner returns the owner identifier if tamed.
func (w *Wolf) Owner() string {
	ptr := w.owner.Load()
	if ptr == nil {
		return ""
	}
	return *ptr
}

// Sitting returns true if ordered to stay in place.
func (w *Wolf) Sitting() bool {
	return w.sitting.Load()
}

// SetSitting configures the sitting posture.
func (w *Wolf) SetSitting(sitting bool) {
	w.sitting.Store(sitting)
}

// ToggleSitting toggles between sitting and following states.
func (w *Wolf) ToggleSitting() bool {
	if !w.Tamed() {
		return false
	}
	curr := w.sitting.Load()
	w.sitting.Store(!curr)
	return !curr
}

// CollarColour returns the collar dye color.
func (w *Wolf) CollarColour() item.Colour {
	return w.collarColour
}

// SetCollarColour configures the collar dye color.
func (w *Wolf) SetCollarColour(colour item.Colour) {
	w.collarColour = colour
}

// Angry returns true if the wild wolf is hostile towards an attacker.
func (w *Wolf) Angry() bool {
	return w.angry.Load()
}

// SetAngry configures wild hostility.
func (w *Wolf) SetAngry(angry bool) {
	w.angry.Store(angry)
}

// TameWithBone attempts taming with a 33% success probability matching vanilla.
// Returns consumed=true if bone was accepted, and tamed=true if domesticity was unlocked.
func (w *Wolf) TameWithBone(held item.Stack, owner string) (consumed bool, tamed bool) {
	if w.Tamed() || w.Baby() {
		return false, false
	}
	if _, ok := held.Item().(item.Bone); ok {
		if rand.Float32() < 0.33 {
			w.SetTamed(true, owner)
			return true, true
		}
		return true, false
	}
	return false, false
}

// Tick coordinates sitting suppression and wolf autonomous behaviors.
func (w *Wolf) Tick(tx *world.Tx, current int64) {
	if w.sitting.Load() {
		// Stop horizontal movement while sitting
		vel := w.Velocity()
		vel[0] = 0
		vel[2] = 0
		w.SetVelocity(vel)
	}
	w.TickAnimal(tx, current)
}

// BoundingBox returns the collision box conforming to adult or pup dimensions.
func (w *Wolf) BoundingBox() cube.BBox {
	if w.Baby() {
		return cube.Box(-0.15, 0, -0.15, 0.15, 0.425, 0.15)
	}
	return cube.Box(-0.3, 0, -0.3, 0.3, 0.85, 0.3)
}

// Metadata generates the Bedrock protocol entity metadata snapshot.
func (w *Wolf) Metadata() protocol.EntityMetadata {
	builder := entity.NewMetadataBuilder()
	if w.Baby() {
		builder.SetDimensions(0.3, 0.425)
	} else {
		builder.SetDimensions(0.6, 0.85)
	}

	m := w.SyncAnimalMetadata(builder)
	if w.Tamed() {
		m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagTamed)
		m[protocol.EntityDataKeyColorIndex] = byte(w.collarColour.Uint8())
		if owner := w.Owner(); owner != "" {
			m[protocol.EntityDataKeyOwner] = owner
		}
	}
	if w.Sitting() {
		m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagSitting)
	}
	if w.Angry() {
		m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagAngry)
	}
	return m
}
