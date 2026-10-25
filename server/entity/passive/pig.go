package passive

import (
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

// Pig represents a peaceful livestock mob that consumes carrots, potatoes, and beetroots,
// and can be equipped with a saddle to become mountable.
type Pig struct {
	*Animal

	saddled atomic.Bool
}

// NewPig initializes a Pig entity with standard passive attributes.
func NewPig(handle *world.EntityHandle, baby bool) *Pig {
	p := &Pig{
		Animal: NewAnimal(handle, 10.0, baby),
	}
	p.SetSpeed(0.25)

	isBait := func(it world.Item) bool {
		switch it.(type) {
		case block.Carrot, block.Potato, item.Beetroot:
			return true
		default:
			return false
		}
	}

	p.GoalSelector().AddGoal(1, ai.NewFloatGoal(p, func() bool { return false }))
	p.GoalSelector().AddGoal(2, NewAnimalPanicGoal(p.Animal, 1.5))
	p.GoalSelector().AddGoal(3, NewAnimalBreedGoal(p.Animal, nil, func(tx *world.Tx, pos mgl64.Vec3) *Animal {
		babyPig := NewPig(nil, true)
		babyPig.SetPosition(pos)
		return babyPig.Animal
	}))
	p.GoalSelector().AddGoal(4, NewAnimalTemptGoal(p.Animal, 10.0, isBait, nil))
	p.GoalSelector().AddGoal(5, ai.NewRandomStrollGoal(p, 10))
	p.GoalSelector().AddGoal(6, ai.NewLookAtEntityGoal(p, 6.0, nil))

	return p
}

// Saddled returns true if the pig is equipped with a saddle.
func (p *Pig) Saddled() bool {
	return p.saddled.Load()
}

// SetSaddled updates the saddled state.
func (p *Pig) SetSaddled(saddled bool) {
	p.saddled.Store(saddled)
}

// Saddle equips a saddle onto an adult pig.
func (p *Pig) Saddle() bool {
	if p.Baby() || p.Saddled() {
		return false
	}
	p.SetSaddled(true)
	return true
}

// Tick coordinates pig autonomous behavior.
func (p *Pig) Tick(tx *world.Tx, current int64) {
	p.TickAnimal(tx, current)
}

// BoundingBox returns the collision box conforming to adult or piglet dimensions.
func (p *Pig) BoundingBox() cube.BBox {
	if p.Baby() {
		return cube.Box(-0.225, 0, -0.225, 0.225, 0.45, 0.225)
	}
	return cube.Box(-0.45, 0, -0.45, 0.45, 0.9, 0.45)
}

// Metadata generates the Bedrock protocol entity metadata snapshot.
func (p *Pig) Metadata() protocol.EntityMetadata {
	builder := entity.NewMetadataBuilder()
	if p.Baby() {
		builder.SetDimensions(0.45, 0.45)
	} else {
		builder.SetDimensions(0.9, 0.9)
	}

	m := p.SyncAnimalMetadata(builder)
	if p.Saddled() {
		m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagSaddled)
	}
	return m
}
