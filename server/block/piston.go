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
	_ world.Block                 = Piston{}
	_ world.Item                  = Piston{}
	_ world.NeighbourUpdateTicker = Piston{}
	_ world.ScheduledTicker       = Piston{}
	_ world.RedstonePowerAction   = Piston{}
)

// Piston is a mechanical block capable of pushing up to 12 blocks when powered by redstone.
// Sticky pistons can also pull a block and its connected slime/honey adhesion networks back upon retraction.
type Piston struct {
	solid
	transparent

	// Facing is the direction the piston faces and extends towards.
	Facing cube.Face
	// Sticky indicates whether this piston is sticky.
	Sticky bool
}

// Extended returns whether the piston is currently physically extended with a piston arm in the world.
func (p Piston) Extended(pos cube.Pos, tx *world.Tx) bool {
	headPos := pos.Side(p.Facing)
	arm, ok := tx.Block(headPos).(PistonArm)
	return ok && arm.Facing == p.Facing
}

// Extend pushes the blocks in front of the piston and places the extended piston arm.
func (p Piston) Extend(pos cube.Pos, tx *world.Tx) bool {
	if p.Extended(pos, tx) {
		return false
	}
	solver := NewPistonSolver(tx, pos, p.Facing, false)
	if !solver.Solve() {
		return false
	}
	headPos := pos.Side(p.Facing)
	tx.SetBlock(headPos, PistonArm{Facing: p.Facing, Sticky: p.Sticky}, nil)
	tx.PlaySound(pos.Vec3Centre(), sound.PowerOn{})
	return true
}

// Retract removes the piston arm and, if sticky, pulls back the block in front of it and its adhesion network.
func (p Piston) Retract(pos cube.Pos, tx *world.Tx) bool {
	if !p.Extended(pos, tx) {
		return false
	}
	headPos := pos.Side(p.Facing)
	tx.SetBlock(headPos, Air{}, nil)
	if p.Sticky {
		solver := NewPistonSolver(tx, pos, p.Facing, true)
		solver.Solve()
	}
	tx.PlaySound(pos.Vec3Centre(), sound.PowerOff{})
	return true
}

// UseOnBlock places the piston oriented facing away from the player.
func (p Piston) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) (used bool) {
	pos, _, used = firstReplaceable(tx, pos, face, p)
	if !used {
		return false
	}
	p.Facing = calculateFace(user, pos).Opposite()
	place(tx, pos, p, user, ctx)
	return placed(ctx)
}

// NeighbourUpdateTick schedules an extension or retraction when redstone power status changes.
func (p Piston) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	powered := tx.RedstonePower(pos) > 0
	extended := p.Extended(pos, tx)
	if powered != extended {
		tx.ScheduleBlockUpdate(pos, p, redstoneTicks(1))
	}
}

// RedstonePowerAction schedules an update when immediate redstone power state changes.
func (p Piston) RedstonePowerAction(pos cube.Pos, tx *world.Tx, _, _ int) {
	p.NeighbourUpdateTick(pos, pos, tx)
}

// ScheduledTick executes piston extension or retraction after a 1 redstone tick delay.
func (p Piston) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	powered := tx.RedstonePower(pos) > 0
	extended := p.Extended(pos, tx)
	if powered && !extended {
		p.Extend(pos, tx)
	} else if !powered && extended {
		p.Retract(pos, tx)
	}
}

// BreakInfo ...
func (p Piston) BreakInfo() BreakInfo {
	return newBreakInfo(0.5, pickaxeHarvestable, pickaxeEffective, oneOf(p)).withBreakHandler(func(pos cube.Pos, tx *world.Tx, _ item.User) {
		headPos := pos.Side(p.Facing)
		if arm, ok := tx.Block(headPos).(PistonArm); ok && arm.Facing == p.Facing {
			tx.SetBlock(headPos, Air{}, nil)
		}
	})
}

// EncodeItem ...
func (p Piston) EncodeItem() (name string, meta int16) {
	if p.Sticky {
		return "minecraft:sticky_piston", 0
	}
	return "minecraft:piston", 0
}

// EncodeBlock ...
func (p Piston) EncodeBlock() (string, map[string]any) {
	name := "minecraft:piston"
	if p.Sticky {
		name = "minecraft:sticky_piston"
	}
	return name, map[string]any{"facing_direction": int32(p.Facing)}
}

// allPistons returns all permutations of pistons and sticky pistons across all faces.
func allPistons() (pistons []world.Block) {
	for _, f := range cube.Faces() {
		for _, sticky := range []bool{false, true} {
			pistons = append(pistons, Piston{Facing: f, Sticky: sticky})
		}
	}
	return pistons
}
