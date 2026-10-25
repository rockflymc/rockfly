package passive

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/ai"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// Cow represents a peaceful bovine mob that consumes wheat, breeds, panics when struck,
// and produces milk when milked with an empty bucket.
type Cow struct {
	*Animal
}

// NewCow initializes a Cow entity with standard passive livestock behaviors.
func NewCow(handle *world.EntityHandle, baby bool) *Cow {
	c := &Cow{
		Animal: NewAnimal(handle, 10.0, baby),
	}
	c.SetSpeed(0.20)

	isWheat := func(it world.Item) bool {
		_, ok := it.(item.Wheat)
		return ok
	}

	c.GoalSelector().AddGoal(1, ai.NewFloatGoal(c, func() bool { return false }))
	c.GoalSelector().AddGoal(2, NewAnimalPanicGoal(c.Animal, 1.5))
	c.GoalSelector().AddGoal(3, NewAnimalBreedGoal(c.Animal, nil, func(tx *world.Tx, pos mgl64.Vec3) *Animal {
		babyCow := NewCow(nil, true)
		babyCow.SetPosition(pos)
		return babyCow.Animal
	}))
	c.GoalSelector().AddGoal(4, NewAnimalTemptGoal(c.Animal, 10.0, isWheat, nil))
	c.GoalSelector().AddGoal(5, ai.NewRandomStrollGoal(c, 10))
	c.GoalSelector().AddGoal(6, ai.NewLookAtEntityGoal(c, 6.0, nil))

	return c
}

// Tick updates cow autonomous logic.
func (c *Cow) Tick(tx *world.Tx, current int64) {
	c.TickAnimal(tx, current)
}

// Milk attempts to milk the cow using the provided item stack.
// Returns the resulting milk bucket item and true if successful.
func (c *Cow) Milk(held item.Stack) (item.Stack, bool) {
	if c.Baby() {
		return held, false
	}
	if b, ok := held.Item().(item.Bucket); ok && b.Empty() {
		return item.NewStack(item.Bucket{Content: item.MilkBucketContent()}, 1), true
	}
	return held, false
}

// BoundingBox returns the collision box conforming to adult or calf dimensions.
func (c *Cow) BoundingBox() cube.BBox {
	if c.Baby() {
		return cube.Box(-0.225, 0, -0.225, 0.225, 0.7, 0.225)
	}
	return cube.Box(-0.45, 0, -0.45, 0.45, 1.4, 0.45)
}

// Metadata generates the Bedrock protocol entity metadata snapshot.
func (c *Cow) Metadata() protocol.EntityMetadata {
	builder := entity.NewMetadataBuilder()
	if c.Baby() {
		builder.SetDimensions(0.45, 0.7)
	} else {
		builder.SetDimensions(0.9, 1.4)
	}
	return c.SyncAnimalMetadata(builder)
}
