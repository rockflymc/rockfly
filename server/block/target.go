package block

import (
	"math"
	"math/rand/v2"
	"sync"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

var (
	_ world.Block                     = Target{}
	_ world.Item                      = Target{}
	_ world.ScheduledTicker           = Target{}
	_ world.RedstonePowerSource       = Target{}
	_ world.RedstoneStrongPowerSource = Target{}
	_ world.RedstoneWeakBlockPowerer  = Target{}

	targetMu       sync.RWMutex
	targetPowerMap = make(map[cube.Pos]int)
)

// Target is a block that provides a redstone signal when hit by a projectile. The closer the projectile
// hits to the center of the block face, the higher the redstone signal strength (1-15).
type Target struct {
	solid
}

// RedstoneWeaklyPowersBlocks indicates whether weak power can conduct to adjacent blocks.
func (Target) RedstoneWeaklyPowersBlocks() bool {
	return true
}

// RedstonePower emits signal strength from all faces according to the current target power.
func (t Target) RedstonePower(pos cube.Pos, _ *world.Tx, _ cube.Face) int {
	targetMu.RLock()
	defer targetMu.RUnlock()
	return targetPowerMap[pos]
}

// RedstoneStrongPower emits strong signal strength into adjacent blocks.
func (t Target) RedstoneStrongPower(pos cube.Pos, tx *world.Tx, face cube.Face) int {
	return t.RedstonePower(pos, tx, face)
}

// CalculateTargetPower computes the signal strength (1-15) based on the distance from the face center.
func CalculateTargetPower(impactPos mgl64.Vec3, blockPos cube.Pos, face cube.Face) int {
	if math.IsNaN(impactPos[0]) || math.IsNaN(impactPos[1]) || math.IsNaN(impactPos[2]) ||
		math.IsInf(impactPos[0], 0) || math.IsInf(impactPos[1], 0) || math.IsInf(impactPos[2], 0) {
		return 1
	}
	center := blockPos.Vec3Centre()
	var off1, off2 float64
	switch face.Axis() {
	case cube.X:
		off1 = impactPos[1] - center[1]
		off2 = impactPos[2] - center[2]
	case cube.Y:
		off1 = impactPos[0] - center[0]
		off2 = impactPos[2] - center[2]
	case cube.Z:
		off1 = impactPos[0] - center[0]
		off2 = impactPos[1] - center[1]
	}

	dist := math.Sqrt(off1*off1 + off2*off2)
	if math.IsNaN(dist) || math.IsInf(dist, 0) {
		return 1
	}
	// Bullseye within radius 0.07 yields power 15.
	if dist <= 0.07 {
		return 15
	}
	power := 15 - int(math.Floor(dist*28.0))
	if power < 1 {
		return 1
	}
	if power > 15 {
		return 15
	}
	return power
}

// Hit activates the target block with redstone power calculated from impact position.
func (t Target) Hit(pos cube.Pos, tx *world.Tx, impactPos mgl64.Vec3, face cube.Face, isArrow bool) int {
	power := CalculateTargetPower(impactPos, pos, face)
	targetMu.Lock()
	targetPowerMap[pos] = power
	targetMu.Unlock()

	if tx == nil || pos.OutOfBounds(tx.Range()) {
		return power
	}

	ticks := redstoneTicks(4)
	if isArrow {
		ticks = redstoneTicks(8)
	}
	tx.ScheduleBlockUpdate(pos, t, ticks)
	return power
}

// ScheduledTick turns the target off after the hit duration expires.
func (t Target) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	targetMu.Lock()
	delete(targetPowerMap, pos)
	targetMu.Unlock()
}

// BreakInfo ...
func (t Target) BreakInfo() BreakInfo {
	return newBreakInfo(0.5, alwaysHarvestable, hoeEffective, oneOf(Target{})).withBreakHandler(func(pos cube.Pos, _ *world.Tx, _ item.User) {
		targetMu.Lock()
		delete(targetPowerMap, pos)
		targetMu.Unlock()
	})
}

// EncodeItem ...
func (Target) EncodeItem() (name string, meta int16) {
	return "minecraft:target", 0
}

// EncodeBlock ...
func (t Target) EncodeBlock() (string, map[string]any) {
	return "minecraft:target", nil
}
