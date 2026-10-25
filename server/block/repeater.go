package block

import (
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

var (
	_ world.Block                  = Repeater{}
	_ world.Item                   = Repeater{}
	_ world.ScheduledTicker        = Repeater{}
	_ world.NeighbourUpdateTicker  = Repeater{}
	_ world.RedstonePowerSource    = Repeater{}
	_ world.RedstoneStrongPowerSource = Repeater{}
	_ world.RedstoneWeakBlockPowerer = Repeater{}
)

// Repeater is a redstone component that functions as a diode, signal repeater, delay gate, and locking latch.
type Repeater struct {
	transparent
	empty

	// Facing is the horizontal direction the repeater outputs towards.
	Facing cube.Direction
	// Delay is the delay of the repeater in redstone ticks (1-4). 1 redstone tick equals 2 game ticks (100ms).
	Delay int
	// Powered indicates whether the repeater is currently outputting a full strength 15 redstone signal.
	Powered bool
}

// Model returns the flat diode bounding box (2 pixels high).
func (Repeater) Model() world.BlockModel {
	return diodeModel{}
}

// HasLiquidDrops returns true so water washes away the repeater.
func (Repeater) HasLiquidDrops() bool {
	return true
}

// SideClosed ...
func (Repeater) SideClosed(cube.Pos, cube.Pos, *world.Tx) bool {
	return false
}

// BreakInfo defines drops and break behavior.
func (r Repeater) BreakInfo() BreakInfo {
	return newBreakInfo(0, alwaysHarvestable, nothingEffective, oneOf(Repeater{Delay: 1}))
}

// UseOnBlock places the repeater on top of a solid surface facing away from the player.
func (r Repeater) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	pos, face, used := firstReplaceable(tx, pos, face, r)
	if !used || face != cube.FaceUp {
		return false
	}
	if !repeaterSupported(tx, pos) {
		return false
	}
	r.Facing = user.Rotation().Direction().Opposite()
	if r.Delay < 1 || r.Delay > 4 {
		r.Delay = 1
	}
	r.Powered = false

	place(tx, pos, r, user, ctx)
	ok := placed(ctx)
	if ok {
		if r.shouldBePowered(pos, tx) {
			tx.ScheduleBlockUpdate(pos, r, redstoneTicks(r.Delay))
		}
	}
	return ok
}

// Activate right-clicks cycle the repeater delay: 1 -> 2 -> 3 -> 4 -> 1.
func (r Repeater) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, _ item.User, _ *item.UseContext) bool {
	r.Delay = (r.Delay % 4) + 1
	tx.SetBlock(pos, r, nil)
	tx.PlaySound(pos.Vec3Centre(), sound.Click{})
	return true
}

// NeighbourUpdateTick handles breaking when unsupported and schedules state toggles when input changes.
func (r Repeater) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if !repeaterSupported(tx, pos) {
		breakBlock(r, pos, tx)
		return
	}
	if r.SideLocked(pos, tx) {
		return
	}
	shouldPower := r.shouldBePowered(pos, tx)
	if shouldPower != r.Powered {
		tx.ScheduleBlockUpdate(pos, r, redstoneTicks(r.Delay))
	}
}

// ScheduledTick evaluates rear redstone input after the configured delay and updates powered state.
func (r Repeater) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	if tx == nil {
		return
	}
	current, ok := tx.Block(pos).(Repeater)
	if !ok {
		return
	}
	if current.SideLocked(pos, tx) {
		return
	}
	shouldPower := current.shouldBePowered(pos, tx)
	if shouldPower != current.Powered {
		current.Powered = shouldPower
		tx.SetBlock(pos, current, &world.SetOpts{DisableRedstoneUpdates: true})
		tx.Redstone().ScheduleUpdate(pos)
	}
}

// shouldBePowered checks if the block behind the repeater is providing redstone power.
func (r Repeater) shouldBePowered(pos cube.Pos, tx *world.Tx) bool {
	rearFace := r.Facing.Opposite().Face()
	return tx.RedstonePowerFrom(pos, rearFace) > 0
}

// SideLocked checks if an adjacent powered repeater or comparator is facing into this repeater's side.
func (r Repeater) SideLocked(pos cube.Pos, tx *world.Tx) bool {
	leftFace := r.Facing.RotateLeft().Face()
	rightFace := r.Facing.RotateRight().Face()

	return r.isSideLocking(pos.Side(leftFace), tx, leftFace.Opposite()) ||
		r.isSideLocking(pos.Side(rightFace), tx, rightFace.Opposite())
}

// isSideLocking reports whether the block at sidePos is a powered repeater or active comparator pointing into pos.
func (Repeater) isSideLocking(sidePos cube.Pos, tx *world.Tx, towardsThis cube.Face) bool {
	if sidePos.OutOfBounds(tx.Range()) {
		return false
	}
	b := tx.Block(sidePos)
	if other, ok := b.(Repeater); ok {
		return other.Facing.Face() == towardsThis && other.Powered
	}
	if comp, ok := b.(Comparator); ok {
		return comp.Facing.Face() == towardsThis && comp.Powered
	}
	return false
}

// RedstonePower emits full signal 15 strictly from the front face when powered.
func (r Repeater) RedstonePower(_ cube.Pos, _ *world.Tx, face cube.Face) int {
	if r.Powered && face == r.Facing.Face() {
		return 15
	}
	return 0
}

// RedstoneStrongPower emits strong signal 15 strictly from the front face when powered.
func (r Repeater) RedstoneStrongPower(pos cube.Pos, tx *world.Tx, face cube.Face) int {
	return r.RedstonePower(pos, tx, face)
}

// RedstoneWeaklyPowersBlocks allows adjacent conductive blocks in front to be weakly powered.
func (Repeater) RedstoneWeaklyPowersBlocks() bool {
	return true
}

// EncodeItem encodes the inventory item representation.
func (Repeater) EncodeItem() (name string, meta int16) {
	return "minecraft:repeater", 0
}

// EncodeBlock maps the repeater into canonical Bedrock block states.
func (r Repeater) EncodeBlock() (string, map[string]any) {
	name := "minecraft:unpowered_repeater"
	if r.Powered {
		name = "minecraft:powered_repeater"
	}
	delay := int32(r.Delay - 1)
	if delay < 0 || delay > 3 {
		delay = 0
	}
	return name, map[string]any{
		"minecraft:cardinal_direction": r.Facing.String(),
		"repeater_delay":               delay,
	}
}

// allRepeaters generates all 32 combinations (4 directions x 4 delays x 2 powered states).
func allRepeaters() (repeaters []world.Block) {
	for _, f := range cube.Directions() {
		for delay := 1; delay <= 4; delay++ {
			for _, powered := range []bool{false, true} {
				repeaters = append(repeaters, Repeater{Facing: f, Delay: delay, Powered: powered})
			}
		}
	}
	return
}

// repeaterSupported reports whether the support block beneath pos can hold a repeater.
func repeaterSupported(tx *world.Tx, pos cube.Pos) bool {
	below := pos.Side(cube.FaceDown)
	if below.OutOfBounds(tx.Range()) {
		return false
	}
	return tx.Block(below).Model().FaceSolid(below, cube.FaceUp, tx)
}

// diodeModel provides the flat 2-pixel slab model for repeaters and comparators.
type diodeModel struct{}

// BBox returns a flat box bounding box.
func (diodeModel) BBox(cube.Pos, world.BlockSource) []cube.BBox {
	return []cube.BBox{cube.Box(0, 0, 0, 1, 0.125, 1)}
}

// FaceSolid always returns false for diode components.
func (diodeModel) FaceSolid(cube.Pos, cube.Face, world.BlockSource) bool {
	return false
}
