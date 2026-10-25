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
	_ world.Block                     = TripwireHook{}
	_ world.Item                      = TripwireHook{}
	_ world.ScheduledTicker           = TripwireHook{}
	_ world.NeighbourUpdateTicker     = TripwireHook{}
	_ world.RedstonePowerSource       = TripwireHook{}
	_ world.RedstoneStrongPowerSource = TripwireHook{}
	_ world.RedstoneWeakBlockPowerer  = TripwireHook{}
)

// TripwireHook is a redstone component that connects to another tripwire hook via string (tripwire)
// to detect entities passing through the beam and emit strong redstone power (15).
type TripwireHook struct {
	transparent
	empty

	// Facing is the horizontal direction the hook projects its line towards.
	Facing cube.Direction
	// Attached specifies if this hook is connected to an opposite hook via tripwire string.
	Attached bool
	// Powered specifies if an entity is currently tripping the wire.
	Powered bool
}

// RedstoneWeaklyPowersBlocks indicates whether weak power can conduct to adjacent blocks.
func (TripwireHook) RedstoneWeaklyPowersBlocks() bool {
	return true
}

// RedstonePower emits signal strength 15 when powered.
func (h TripwireHook) RedstonePower(_ cube.Pos, _ *world.Tx, _ cube.Face) int {
	if h.Powered {
		return 15
	}
	return 0
}

// RedstoneStrongPower emits strong signal strength into the wall block it is mounted on.
func (h TripwireHook) RedstoneStrongPower(pos cube.Pos, tx *world.Tx, face cube.Face) int {
	if h.Powered && face == h.Facing.Opposite().Face() {
		return 15
	}
	return 0
}

// CheckLine inspects the tripwire line in front of the hook and updates attached & powered states.
func (h TripwireHook) CheckLine(pos cube.Pos, tx *world.Tx) (attached bool, powered bool) {
	stepDir := h.Facing.Face()
	oppDir := h.Facing.Opposite()

	var wirePositions []cube.Pos
	otherHookPos := cube.Pos{}
	foundOtherHook := false

	// Trace up to 40 blocks in front of the hook.
	currPos := pos
	for i := 1; i <= 40; i++ {
		currPos = currPos.Side(stepDir)
		currBlock := tx.Block(currPos)
		if currBlock == nil {
			break
		}

		if str, ok := currBlock.(String); ok {
			wirePositions = append(wirePositions, currPos)
			// Check if an entity is inside this wire.
			box := cube.Box(
				float64(currPos[0]), float64(currPos[1]), float64(currPos[2]),
				float64(currPos[0]+1), float64(currPos[1]+1), float64(currPos[2]+1),
			)
			for range tx.EntitiesWithin(box) {
				powered = true
			}
			_ = str
			continue
		}

		if other, ok := currBlock.(TripwireHook); ok && other.Facing == oppDir {
			foundOtherHook = true
			otherHookPos = currPos
			break
		}

		// Line is broken by any other block.
		break
	}

	attached = foundOtherHook && len(wirePositions) > 0
	if !attached {
		powered = false
	}

	// Update self.
	if h.Attached != attached || h.Powered != powered {
		h.Attached = attached
		h.Powered = powered
		tx.SetBlock(pos, h, nil)
	}

	// Update other hook.
	if foundOtherHook {
		if other, ok := tx.Block(otherHookPos).(TripwireHook); ok {
			if other.Attached != attached || other.Powered != powered {
				other.Attached = attached
				other.Powered = powered
				tx.SetBlock(otherHookPos, other, nil)
			}
		}
	}

	// Update all intervening strings.
	for _, wPos := range wirePositions {
		if str, ok := tx.Block(wPos).(String); ok {
			if str.Attached != attached || str.Powered != powered {
				str.Attached = attached
				str.Powered = powered
				tx.SetBlock(wPos, str, nil)
			}
		}
	}

	return attached, powered
}

// NeighbourUpdateTick validates attachment to wall and checks the tripwire line.
func (h TripwireHook) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	wallPos := pos.Side(h.Facing.Opposite().Face())
	if !tx.Block(wallPos).Model().FaceSolid(wallPos, h.Facing.Face(), tx) {
		tx.SetBlock(pos, Air{}, nil)
		dropItem(tx, item.NewStack(TripwireHook{}, 1), pos.Vec3Centre())
		return
	}
	h.CheckLine(pos, tx)
}

// ScheduledTick turns off power when an entity departs the wire.
func (h TripwireHook) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	h.CheckLine(pos, tx)
}

// UseOnBlock places the tripwire hook attached to the side of a solid block.
func (h TripwireHook) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) (used bool) {
	if face.Axis() == cube.Y {
		return false
	}
	pos, _, used = firstReplaceable(tx, pos, face, h)
	if !used {
		return false
	}
	wallPos := pos.Side(face.Opposite())
	if !tx.Block(wallPos).Model().FaceSolid(wallPos, face, tx) {
		return false
	}

	h.Facing = face.Direction()
	place(tx, pos, h, user, ctx)
	h.CheckLine(pos, tx)
	tx.PlaySound(pos.Vec3Centre(), sound.Click{})
	return placed(ctx)
}

// BreakInfo ...
func (h TripwireHook) BreakInfo() BreakInfo {
	return newBreakInfo(0.5, alwaysHarvestable, axeEffective, oneOf(TripwireHook{})).withBreakHandler(func(pos cube.Pos, tx *world.Tx, _ item.User) {
		h.CheckLine(pos, tx)
	})
}

// EncodeItem ...
func (TripwireHook) EncodeItem() (name string, meta int16) {
	return "minecraft:tripwire_hook", 0
}

// EncodeBlock ...
func (h TripwireHook) EncodeBlock() (string, map[string]any) {
	return "minecraft:tripwire_hook", map[string]any{
		"direction":    int32(h.Facing),
		"attached_bit": boolByte(h.Attached),
		"powered_bit":  boolByte(h.Powered),
	}
}

// allTripwireHooks returns all 16 permutations of tripwire hooks.
func allTripwireHooks() (hooks []world.Block) {
	for _, d := range cube.Directions() {
		for _, attached := range []bool{false, true} {
			for _, powered := range []bool{false, true} {
				hooks = append(hooks, TripwireHook{
					Facing:   d,
					Attached: attached,
					Powered:  powered,
				})
			}
		}
	}
	return hooks
}
