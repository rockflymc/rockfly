package block

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

var (
	_ world.Block                 = BubbleColumn{}
	_ world.Item                  = BubbleColumn{}
	_ world.Liquid                = BubbleColumn{}
	_ world.ScheduledTicker       = BubbleColumn{}
	_ world.NeighbourUpdateTicker = BubbleColumn{}
	_ EntityInsider               = BubbleColumn{}
)

// BubbleColumn is a water column created by Soul Sand (upward ocean geyser) or Magma Block (downward whirlpool).
type BubbleColumn struct {
	empty
	transparent

	// DragDown specifies if this bubble column is a downward vortex pulling entities downward.
	DragDown bool
}

// LiquidDepth returns 8 (full source depth).
func (BubbleColumn) LiquidDepth() int {
	return 8
}

// SpreadDecay returns 1.
func (BubbleColumn) SpreadDecay() int {
	return 1
}

// WithDepth returns the bubble column itself.
func (b BubbleColumn) WithDepth(int, bool) world.Liquid {
	return b
}

// LiquidFalling returns false.
func (BubbleColumn) LiquidFalling() bool {
	return false
}

// BlastResistance returns 100.
func (BubbleColumn) BlastResistance() float64 {
	return 100.0
}

// LiquidType returns "water".
func (BubbleColumn) LiquidType() string {
	return "water"
}

// Harden returns false.
func (BubbleColumn) Harden(cube.Pos, *world.Tx, *cube.Pos) bool {
	return false
}

// LiquidRemoveBlock does nothing for bubble columns.
func (BubbleColumn) LiquidRemoveBlock(cube.Pos, *world.Tx, world.Block) {}

// EntityInside applies liquid velocity dynamics to entities inside the bubble column.
func (b BubbleColumn) EntityInside(pos cube.Pos, _ *world.Tx, e world.Entity) {
	if e == nil {
		return
	}
	if v, ok := e.(interface {
		Velocity() mgl64.Vec3
		SetVelocity(mgl64.Vec3)
	}); ok {
		vel := v.Velocity()
		if math.IsNaN(vel[0]) || math.IsInf(vel[0], 0) {
			vel[0] = 0
		}
		if math.IsNaN(vel[1]) || math.IsInf(vel[1], 0) {
			vel[1] = 0
		}
		if math.IsNaN(vel[2]) || math.IsInf(vel[2], 0) {
			vel[2] = 0
		}
		if b.DragDown {
			vel[1] = max(-0.7, vel[1]-0.15)
		} else {
			vel[1] = min(0.7, vel[1]+0.15)
		}
		v.SetVelocity(vel)
	}

	// Upward bubble columns replenish oxygen and prevent drowning.
	if !b.DragDown {
		if air, ok := e.(interface{ SetAirSupply(time.Duration) }); ok {
			air.SetAirSupply(time.Second * 15)
		}
	}
}

// NeighbourUpdateTick validates the column base and propagates bubble columns upward.
func (b BubbleColumn) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if tx == nil || pos.OutOfBounds(tx.Range()) {
		return
	}
	belowPos := pos.Side(cube.FaceDown)
	if belowPos.OutOfBounds(tx.Range()) {
		tx.SetBlock(pos, Water{Depth: 8, Still: true}, nil)
		return
	}
	below := tx.Block(belowPos)

	// Validate base block:
	valid := false
	if b.DragDown {
		if _, ok := below.(Magma); ok {
			valid = true
		} else if bc, ok := below.(BubbleColumn); ok && bc.DragDown {
			valid = true
		}
	} else {
		if _, ok := below.(SoulSand); ok {
			valid = true
		} else if bc, ok := below.(BubbleColumn); ok && !bc.DragDown {
			valid = true
		}
	}

	if !valid {
		// Dissolve back into normal still water.
		tx.SetBlock(pos, Water{Depth: 8, Still: true}, nil)
		return
	}

	// Propagate upward into any still water block.
	abovePos := pos.Side(cube.FaceUp)
	if !abovePos.OutOfBounds(tx.Range()) {
		above := tx.Block(abovePos)
		if w, ok := above.(Water); ok && w.Depth == 8 && w.Still {
			tx.SetBlock(abovePos, BubbleColumn{DragDown: b.DragDown}, nil)
			tx.ScheduleBlockUpdate(abovePos, BubbleColumn{DragDown: b.DragDown}, redstoneTicks(1))
		}
	}
}

// ScheduledTick ensures propagation along the column continues.
func (b BubbleColumn) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	if tx == nil || pos.OutOfBounds(tx.Range()) {
		return
	}
	b.NeighbourUpdateTick(pos, pos, tx)
}

// CheckCreateBubbleColumn checks if water at pos should transform into a bubble column over soul sand or magma.
func CheckCreateBubbleColumn(pos cube.Pos, tx *world.Tx) bool {
	if tx == nil || pos.OutOfBounds(tx.Range()) {
		return false
	}
	belowPos := pos.Side(cube.FaceDown)
	if belowPos.OutOfBounds(tx.Range()) {
		return false
	}
	below := tx.Block(belowPos)

	if _, ok := below.(SoulSand); ok {
		tx.SetBlock(pos, BubbleColumn{DragDown: false}, nil)
		tx.ScheduleBlockUpdate(pos, BubbleColumn{DragDown: false}, redstoneTicks(1))
		return true
	} else if bc, ok := below.(BubbleColumn); ok && !bc.DragDown {
		tx.SetBlock(pos, BubbleColumn{DragDown: false}, nil)
		tx.ScheduleBlockUpdate(pos, BubbleColumn{DragDown: false}, redstoneTicks(1))
		return true
	}

	if _, ok := below.(Magma); ok {
		tx.SetBlock(pos, BubbleColumn{DragDown: true}, nil)
		tx.ScheduleBlockUpdate(pos, BubbleColumn{DragDown: true}, redstoneTicks(1))
		return true
	} else if bc, ok := below.(BubbleColumn); ok && bc.DragDown {
		tx.SetBlock(pos, BubbleColumn{DragDown: true}, nil)
		tx.ScheduleBlockUpdate(pos, BubbleColumn{DragDown: true}, redstoneTicks(1))
		return true
	}

	return false
}

// BreakInfo ...
func (BubbleColumn) BreakInfo() BreakInfo {
	return newBreakInfo(100.0, alwaysHarvestable, nothingEffective, nil)
}

// EncodeItem ...
func (BubbleColumn) EncodeItem() (name string, meta int16) {
	return "minecraft:bubble_column", 0
}

// EncodeBlock ...
func (b BubbleColumn) EncodeBlock() (string, map[string]any) {
	return "minecraft:bubble_column", map[string]any{
		"drag_down": boolByte(b.DragDown),
	}
}

// allBubbleColumns returns all 2 permutations of bubble columns.
func allBubbleColumns() []world.Block {
	return []world.Block{
		BubbleColumn{DragDown: false},
		BubbleColumn{DragDown: true},
	}
}
