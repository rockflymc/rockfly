package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

var (
	_ world.Block                 = PistonArm{}
	_ world.NeighbourUpdateTicker = PistonArm{}
)

// PistonArm is the extended head of a piston or sticky piston.
type PistonArm struct {
	empty
	transparent

	// Facing is the direction the piston head extends towards.
	Facing cube.Face
	// Sticky indicates whether this is the head of a sticky piston.
	Sticky bool
}

// NeighbourUpdateTick removes the piston arm if its base piston is missing or no longer pointing at it.
func (a PistonArm) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if tx == nil || pos.OutOfBounds(tx.Range()) {
		return
	}
	basePos := pos.Side(a.Facing.Opposite())
	if basePos.OutOfBounds(tx.Range()) {
		tx.SetBlock(pos, Air{}, nil)
		return
	}
	if p, ok := tx.Block(basePos).(Piston); !ok || p.Facing != a.Facing {
		tx.SetBlock(pos, Air{}, nil)
	}
}

// BreakInfo ...
func (a PistonArm) BreakInfo() BreakInfo {
	return newBreakInfo(0.5, alwaysHarvestable, nothingEffective, nil).withBreakHandler(func(pos cube.Pos, tx *world.Tx, _ item.User) {
		if tx == nil || pos.OutOfBounds(tx.Range()) {
			return
		}
		basePos := pos.Side(a.Facing.Opposite())
		if basePos.OutOfBounds(tx.Range()) {
			return
		}
		if p, ok := tx.Block(basePos).(Piston); ok && p.Facing == a.Facing {
			tx.SetBlock(basePos, Air{}, nil)
			if create := tx.World().EntityRegistry().Config().Item; create != nil {
				dropItem(tx, item.NewStack(p, 1), basePos.Vec3Centre())
			}
		}
	})
}

// EncodeBlock ...
func (a PistonArm) EncodeBlock() (string, map[string]any) {
	name := "minecraft:piston_arm_collision"
	if a.Sticky {
		name = "minecraft:sticky_piston_arm_collision"
	}
	return name, map[string]any{"facing_direction": int32(a.Facing)}
}

// allPistonArms returns all permutations of piston arms.
func allPistonArms() (arms []world.Block) {
	for _, f := range cube.Faces() {
		for _, sticky := range []bool{false, true} {
			arms = append(arms, PistonArm{Facing: f, Sticky: sticky})
		}
	}
	return arms
}
