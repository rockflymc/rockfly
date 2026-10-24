package world

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/event"
	"github.com/go-gl/mathgl/mgl64"
)

// LiquidFlowEvent is dispatched when a liquid attempts to flow from one position into another.
type LiquidFlowEvent struct {
	event.BaseCancellable
	World    *World
	From     cube.Pos
	Into     cube.Pos
	Liquid   Liquid
	Replaced Block
}

// LiquidDecayEvent is dispatched when a liquid decays due to missing source block.
type LiquidDecayEvent struct {
	event.BaseCancellable
	World  *World
	Pos    cube.Pos
	Before Liquid
	After  Liquid
}

// LiquidHardenEvent is dispatched when liquids collide and produce a solid block (e.g. lava + water).
type LiquidHardenEvent struct {
	event.BaseCancellable
	World          *World
	HardenedPos    cube.Pos
	LiquidHardened Block
	OtherLiquid    Block
	NewBlock       Block
}

// SoundPlayEvent is dispatched when a sound is played in the world.
type SoundPlayEvent struct {
	event.BaseCancellable
	World *World
	Sound Sound
	Pos   mgl64.Vec3
}

// FireSpreadEvent is dispatched when fire spreads to an adjacent position.
type FireSpreadEvent struct {
	event.BaseCancellable
	World *World
	From  cube.Pos
	To    cube.Pos
}

// BlockBurnEvent is dispatched when a flammable block is consumed by fire.
type BlockBurnEvent struct {
	event.BaseCancellable
	World *World
	Pos   cube.Pos
}

// CropTrampleEvent is dispatched when an entity tramples a crop block.
type CropTrampleEvent struct {
	event.BaseCancellable
	World *World
	Pos   cube.Pos
}

// LeavesDecayEvent is dispatched when leaves decay naturally without adjacent wood.
type LeavesDecayEvent struct {
	event.BaseCancellable
	World *World
	Pos   cube.Pos
}

// PortalCreateEvent is dispatched when a portal structure is formed.
type PortalCreateEvent struct {
	event.BaseCancellable
	World      *World
	PortalType Dimension
	Positions  []cube.Pos
}

// PortalActivateEvent is dispatched when a portal is activated.
type PortalActivateEvent struct {
	event.BaseCancellable
	World      *World
	PortalType Dimension
	Positions  []cube.Pos
}

// EntitySpawnEvent is dispatched when an entity is spawned into a world.
type EntitySpawnEvent struct {
	Tx     *Tx
	World  *World
	Entity Entity
}

// EntityDespawnEvent is dispatched when an entity is removed from a world.
type EntityDespawnEvent struct {
	Tx     *Tx
	World  *World
	Entity Entity
}

// ExplosionEvent is dispatched when an explosion occurs in a world.
type ExplosionEvent struct {
	event.BaseCancellable
	World          *World
	Source         ExplosionSource
	Entities       *[]Entity
	Blocks         *[]cube.Pos
	ItemDropChance *float64
	SpawnFire      *bool
}

// RedstoneUpdateEvent is dispatched when a redstone power mutation is proposed.
type RedstoneUpdateEvent struct {
	event.BaseCancellable
	World  *World
	Update RedstoneUpdate
}

// WorldCloseEvent is dispatched when a world begins closing before saving chunks.
type WorldCloseEvent struct {
	Tx    *Tx
	World *World
}
