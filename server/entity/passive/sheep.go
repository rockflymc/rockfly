package passive

import (
	"math/rand"
	"sync/atomic"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/ai"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// Sheep represents a peaceful domestic mob that produces colored wool, eats grass to regrow
// its fleece, and can be sheared using shears or dyed into any of the 16 Minecraft colors.
type Sheep struct {
	*Animal

	colour  item.Colour
	sheared atomic.Bool
}

// NewSheep initializes a Sheep entity with specified wool color.
func NewSheep(handle *world.EntityHandle, colour item.Colour, baby bool) *Sheep {
	s := &Sheep{
		Animal: NewAnimal(handle, 8.0, baby),
		colour: colour,
	}
	s.SetSpeed(0.23)

	isWheat := func(it world.Item) bool {
		_, ok := it.(item.Wheat)
		return ok
	}

	s.GoalSelector().AddGoal(1, ai.NewFloatGoal(s, func() bool { return false }))
	s.GoalSelector().AddGoal(2, NewAnimalPanicGoal(s.Animal, 1.5))
	s.GoalSelector().AddGoal(3, NewAnimalBreedGoal(s.Animal, nil, func(tx *world.Tx, pos mgl64.Vec3) *Animal {
		babySheep := NewSheep(nil, s.colour, true)
		babySheep.SetPosition(pos)
		return babySheep.Animal
	}))
	s.GoalSelector().AddGoal(4, NewAnimalTemptGoal(s.Animal, 10.0, isWheat, nil))
	s.GoalSelector().AddGoal(5, ai.NewRandomStrollGoal(s, 10))
	s.GoalSelector().AddGoal(6, ai.NewLookAtEntityGoal(s, 6.0, nil))

	return s
}

// Colour returns the sheep's wool color.
func (s *Sheep) Colour() item.Colour {
	return s.colour
}

// SetColour configures the sheep's wool color.
func (s *Sheep) SetColour(colour item.Colour) {
	s.colour = colour
}

// Sheared returns true if the sheep's wool has been removed.
func (s *Sheep) Sheared() bool {
	return s.sheared.Load()
}

// SetSheared updates the sheared wool state.
func (s *Sheep) SetSheared(sheared bool) {
	s.sheared.Store(sheared)
}

// Shear attempts to harvest the sheep's wool using shears.
// Returns the dropped wool stacks and true if shearing succeeded.
func (s *Sheep) Shear(held item.Stack) ([]item.Stack, bool) {
	if s.Baby() || s.Sheared() {
		return nil, false
	}
	if _, ok := held.Item().(item.Shears); ok {
		s.SetSheared(true)
		count := 1 + rand.Intn(3) // 1 to 3 wool blocks
		wool := item.NewStack(block.Wool{Colour: s.colour}, count)
		return []item.Stack{wool}, true
	}
	return nil, false
}

// EatGrass simulates grazing, regrowing wool and accelerating baby growth.
func (s *Sheep) EatGrass() {
	s.SetSheared(false)
	if s.Baby() {
		// Advance juvenile growth by 60 seconds (1200 ticks)
		age := s.Age() + 1200
		if age > 0 {
			age = 0
		}
		s.SetAge(age)
	}
}

// Tick coordinates sheep autonomous behaviors.
func (s *Sheep) Tick(tx *world.Tx, current int64) {
	s.TickAnimal(tx, current)
}

// BoundingBox returns the collision box conforming to adult or lamb dimensions.
func (s *Sheep) BoundingBox() cube.BBox {
	if s.Baby() {
		return cube.Box(-0.225, 0, -0.225, 0.225, 0.65, 0.225)
	}
	return cube.Box(-0.45, 0, -0.45, 0.45, 1.3, 0.45)
}

// Metadata generates the Bedrock protocol entity metadata snapshot.
func (s *Sheep) Metadata() protocol.EntityMetadata {
	builder := entity.NewMetadataBuilder()
	if s.Baby() {
		builder.SetDimensions(0.45, 0.65)
	} else {
		builder.SetDimensions(0.9, 1.3)
	}

	m := s.SyncAnimalMetadata(builder)
	m[protocol.EntityDataKeyColorIndex] = byte(s.colour.Uint8())
	if s.Sheared() {
		m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagSheared)
	}
	return m
}
