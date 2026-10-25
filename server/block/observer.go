package block

import (
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

var (
	_ world.Block                     = Observer{}
	_ world.Item                      = Observer{}
	_ world.NeighbourUpdateTicker     = Observer{}
	_ world.ScheduledTicker           = Observer{}
	_ world.RedstonePowerSource       = Observer{}
	_ world.RedstoneStrongPowerSource = Observer{}
	_ world.RedstoneWeakBlockPowerer  = Observer{}
)

// Observer is a redstone component that observes changes in the block facing it and emits a 1-redstone-tick
// strong signal (15) from its rear face.
type Observer struct {
	solid
	transparent

	// Facing is the direction the observer's sensor face points towards.
	Facing cube.Face
	// Powered indicates whether the observer is currently emitting its pulse.
	Powered bool
}

// RedstoneWeaklyPowersBlocks indicates whether weak power can conduct to adjacent blocks.
func (Observer) RedstoneWeaklyPowersBlocks() bool {
	return true
}

// RedstonePower emits signal strength 15 strictly from the rear face when powered.
func (o Observer) RedstonePower(_ cube.Pos, _ *world.Tx, face cube.Face) int {
	if o.Powered && face == o.Facing.Opposite() {
		return 15
	}
	return 0
}

// RedstoneStrongPower emits strong signal strength 15 from the rear face when powered.
func (o Observer) RedstoneStrongPower(pos cube.Pos, tx *world.Tx, face cube.Face) int {
	return o.RedstonePower(pos, tx, face)
}

// NeighbourUpdateTick schedules an observation pulse whenever the block in front of its sensor changes.
func (o Observer) NeighbourUpdateTick(pos, changedNeighbour cube.Pos, tx *world.Tx) {
	if changedNeighbour == pos.Side(o.Facing) && !o.Powered {
		tx.ScheduleBlockUpdate(pos, o, redstoneTicks(1))
	}
}

// ScheduledTick turns the observer on for a 1-redstone-tick pulse, or turns it off when the pulse concludes.
func (o Observer) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	if !o.Powered {
		o.Powered = true
		tx.SetBlock(pos, o, nil)
		tx.ScheduleBlockUpdate(pos, o, redstoneTicks(1))
	} else {
		o.Powered = false
		tx.SetBlock(pos, o, nil)
	}
}

// UseOnBlock places the observer oriented with its sensor facing the block it was placed against.
func (o Observer) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) (used bool) {
	pos, _, used = firstReplaceable(tx, pos, face, o)
	if !used {
		return false
	}
	o.Facing = calculateFace(user, pos).Opposite()
	place(tx, pos, o, user, ctx)
	return placed(ctx)
}

// BreakInfo ...
func (o Observer) BreakInfo() BreakInfo {
	return newBreakInfo(3.0, pickaxeHarvestable, pickaxeEffective, oneOf(o))
}

// EncodeItem ...
func (Observer) EncodeItem() (name string, meta int16) {
	return "minecraft:observer", 0
}

// EncodeBlock ...
func (o Observer) EncodeBlock() (string, map[string]any) {
	return "minecraft:observer", map[string]any{
		"minecraft:facing_direction": o.Facing.String(),
		"powered_bit":                boolByte(o.Powered),
	}
}

// allObservers returns all permutations of observers across all faces and power states.
func allObservers() (observers []world.Block) {
	for _, f := range cube.Faces() {
		observers = append(observers, Observer{Facing: f, Powered: false})
		observers = append(observers, Observer{Facing: f, Powered: true})
	}
	return observers
}
