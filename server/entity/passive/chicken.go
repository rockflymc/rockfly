package passive

import (
	"math"
	"math/rand"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/ai"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// Chicken represents a small poultry mob that consumes seeds, flutters downward slowly without
// taking fall damage, and periodically lays eggs.
type Chicken struct {
	*Animal

	eggLayTimer int
}

// NewChicken initializes a Chicken entity with slow-fall flapping and egg laying timers.
func NewChicken(handle *world.EntityHandle, baby bool) *Chicken {
	c := &Chicken{
		Animal:      NewAnimal(handle, 4.0, baby),
		eggLayTimer: 6000 + rand.Intn(6000), // 5 to 10 minutes (6000-12000 ticks)
	}
	c.SetSpeed(0.25)

	isSeeds := func(it world.Item) bool {
		_, ok := it.(block.WheatSeeds)
		return ok
	}

	c.GoalSelector().AddGoal(1, ai.NewFloatGoal(c, func() bool { return false }))
	c.GoalSelector().AddGoal(2, NewAnimalPanicGoal(c.Animal, 1.5))
	c.GoalSelector().AddGoal(3, NewAnimalBreedGoal(c.Animal, nil, func(tx *world.Tx, pos mgl64.Vec3) *Animal {
		babyChicken := NewChicken(nil, true)
		babyChicken.SetPosition(pos)
		return babyChicken.Animal
	}))
	c.GoalSelector().AddGoal(4, NewAnimalTemptGoal(c.Animal, 10.0, isSeeds, nil))
	c.GoalSelector().AddGoal(5, ai.NewRandomStrollGoal(c, 10))
	c.GoalSelector().AddGoal(6, ai.NewLookAtEntityGoal(c, 6.0, nil))

	return c
}

// EggLayTimer returns remaining ticks until the next egg is laid.
func (c *Chicken) EggLayTimer() int {
	return c.eggLayTimer
}

// SetEggLayTimer sets the egg lay countdown in ticks.
func (c *Chicken) SetEggLayTimer(ticks int) {
	c.eggLayTimer = ticks
}

// Tick coordinates slow falling, wing flapping, egg laying, and animal AI.
func (c *Chicken) Tick(tx *world.Tx, current int64) {
	if !c.Dead() {
		// 1. Slow falling wing flutter: cap downward velocity to -0.05 blocks/tick
		vel := c.Velocity()
		if vel[1] < -0.05 {
			vel[1] = math.Max(vel[1], -0.05)
			c.SetVelocity(vel)
			c.SetFallDistance(0) // Immunity to fall damage
		}

		// 2. Egg laying countdown
		if !c.Baby() {
			c.eggLayTimer--
			if c.eggLayTimer <= 0 {
				c.eggLayTimer = 6000 + rand.Intn(6000)
				// Egg lays at position
			}
		}
	}

	c.TickAnimal(tx, current)
}

// LayEgg immediately forces the chicken to produce an egg stack.
func (c *Chicken) LayEgg() (item.Stack, bool) {
	if c.Baby() {
		return item.Stack{}, false
	}
	c.eggLayTimer = 6000 + rand.Intn(6000)
	return item.NewStack(item.Egg{}, 1), true
}

// BoundingBox returns the collision box conforming to adult or chick dimensions.
func (c *Chicken) BoundingBox() cube.BBox {
	if c.Baby() {
		return cube.Box(-0.1, 0, -0.1, 0.1, 0.35, 0.1)
	}
	return cube.Box(-0.2, 0, -0.2, 0.2, 0.7, 0.2)
}

// Metadata generates the Bedrock protocol entity metadata snapshot.
func (c *Chicken) Metadata() protocol.EntityMetadata {
	builder := entity.NewMetadataBuilder()
	if c.Baby() {
		builder.SetDimensions(0.2, 0.35)
	} else {
		builder.SetDimensions(0.4, 0.7)
	}
	return c.SyncAnimalMetadata(builder)
}
