package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
)

var (
	_ world.Block                 = SoulSand{}
	_ world.NeighbourUpdateTicker = SoulSand{}
)

// SoulSand is a block found naturally only in the Nether. SoulSand slows movement of mobs & players
// and generates upward bubble columns in water.
type SoulSand struct {
	solid
}

// NeighbourUpdateTick triggers creation of upward bubble columns in water above.
func (s SoulSand) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	abovePos := pos.Side(cube.FaceUp)
	if w, ok := tx.Block(abovePos).(Water); ok && w.Depth == 8 && w.Still {
		CheckCreateBubbleColumn(abovePos, tx)
	}
}

// SoilFor ...
func (s SoulSand) SoilFor(block world.Block) bool {
	flower, ok := block.(Flower)
	return ok && flower.Type == WitherRose()
}

// Instrument ...
func (s SoulSand) Instrument() sound.Instrument {
	return sound.CowBell()
}

// BreakInfo ...
func (s SoulSand) BreakInfo() BreakInfo {
	return newBreakInfo(0.5, alwaysHarvestable, shovelEffective, oneOf(s))
}

// EncodeItem ...
func (SoulSand) EncodeItem() (name string, meta int16) {
	return "minecraft:soul_sand", 0
}

// EncodeBlock ...
func (SoulSand) EncodeBlock() (string, map[string]any) {
	return "minecraft:soul_sand", nil
}
