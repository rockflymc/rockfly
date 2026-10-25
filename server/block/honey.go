package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// Honey is a block crafted from four honey bottles. It slows movement, reduces jump height,
// and adheres to adjacent blocks in piston mechanisms, with the exception of slime blocks.
type Honey struct {
	solid
	transparent
}

// EntityLand reduces fall damage when landing on a honey block.
func (Honey) EntityLand(_ cube.Pos, _ *world.Tx, e world.Entity, distance *float64) {
	if distance != nil {
		if _, ok := e.(fallDistanceEntity); ok {
			*distance = 0
		}
	}
}

// Friction returns the high friction factor of honey blocks.
func (Honey) Friction() float64 {
	return 0.4
}

// BreakInfo ...
func (h Honey) BreakInfo() BreakInfo {
	return newBreakInfo(0, alwaysHarvestable, nothingEffective, oneOf(h))
}

// EncodeItem ...
func (Honey) EncodeItem() (name string, meta int16) {
	return "minecraft:honey_block", 0
}

// EncodeBlock ...
func (Honey) EncodeBlock() (string, map[string]any) {
	return "minecraft:honey_block", nil
}
