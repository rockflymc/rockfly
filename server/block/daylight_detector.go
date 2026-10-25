package block

import (
	"math"
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

var (
	_ world.Block                     = DaylightDetector{}
	_ world.Item                      = DaylightDetector{}
	_ world.ScheduledTicker           = DaylightDetector{}
	_ world.NeighbourUpdateTicker     = DaylightDetector{}
	_ world.RedstonePowerSource       = DaylightDetector{}
	_ world.RedstoneStrongPowerSource = DaylightDetector{}
	_ world.RedstoneWeakBlockPowerer  = DaylightDetector{}
)

// DaylightDetector is a redstone sensor that emits redstone power proportional to daylight.
// Right-clicking toggles inverted mode, transforming it into a moonlight/night sensor.
type DaylightDetector struct {
	transparent
	empty

	// Inverted indicates whether the detector is inverted to activate in low light and nighttime.
	Inverted bool
	// Power is the redstone signal level emitted (0-15).
	Power int
}

// RedstoneWeaklyPowersBlocks indicates whether weak power can conduct to adjacent blocks.
func (DaylightDetector) RedstoneWeaklyPowersBlocks() bool {
	return true
}

// RedstonePower emits signal strength from all faces according to the current light power.
func (d DaylightDetector) RedstonePower(_ cube.Pos, _ *world.Tx, _ cube.Face) int {
	return d.Power
}

// RedstoneStrongPower emits strong signal strength into adjacent blocks.
func (d DaylightDetector) RedstoneStrongPower(pos cube.Pos, tx *world.Tx, face cube.Face) int {
	return d.RedstonePower(pos, tx, face)
}

// CalculateDaylightPower computes the redstone signal (0..15) based on world time of day.
func CalculateDaylightPower(time int, inverted bool) int {
	t := time % 24000
	rad := float64(t-6000) * 2.0 * math.Pi / 24000.0
	cos := math.Cos(rad)
	if cos < 0 {
		cos = 0
	}
	level := int(math.Round(cos * 15.0))
	if inverted {
		return 15 - level
	}
	return level
}

// Activate toggles inverted mode when a player interacts with the detector.
func (d DaylightDetector) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, _ item.User, _ *item.UseContext) bool {
	d.Inverted = !d.Inverted
	d.Power = CalculateDaylightPower(tx.World().Time(), d.Inverted)
	tx.SetBlock(pos, d, nil)
	tx.PlaySound(pos.Vec3Centre(), sound.Click{})
	return true
}

// UseOnBlock places the daylight detector on a block surface.
func (d DaylightDetector) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) (used bool) {
	pos, _, used = firstReplaceable(tx, pos, face, d)
	if !used {
		return false
	}
	d.Power = CalculateDaylightPower(tx.World().Time(), d.Inverted)
	place(tx, pos, d, user, ctx)
	return placed(ctx)
}

// NeighbourUpdateTick recalculates daylight power and schedules updates.
func (d DaylightDetector) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	tx.ScheduleBlockUpdate(pos, d, redstoneTicks(10))
}

// ScheduledTick periodically updates the emitted daylight signal.
func (d DaylightDetector) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	newPower := CalculateDaylightPower(tx.World().Time(), d.Inverted)
	if newPower != d.Power {
		d.Power = newPower
		tx.SetBlock(pos, d, nil)
	}
	tx.ScheduleBlockUpdate(pos, d, redstoneTicks(10))
}

// BreakInfo ...
func (d DaylightDetector) BreakInfo() BreakInfo {
	return newBreakInfo(0.2, alwaysHarvestable, axeEffective, oneOf(DaylightDetector{}))
}

// EncodeItem ...
func (DaylightDetector) EncodeItem() (name string, meta int16) {
	return "minecraft:daylight_detector", 0
}

// EncodeBlock ...
func (d DaylightDetector) EncodeBlock() (string, map[string]any) {
	name := "minecraft:daylight_detector"
	if d.Inverted {
		name = "minecraft:daylight_detector_inverted"
	}
	return name, map[string]any{"redstone_signal": int32(d.Power)}
}

// allDaylightDetectors returns all 32 combinations of daylight detector states.
func allDaylightDetectors() (detectors []world.Block) {
	for _, inverted := range []bool{false, true} {
		for power := 0; power <= 15; power++ {
			detectors = append(detectors, DaylightDetector{
				Inverted: inverted,
				Power:    power,
			})
		}
	}
	return detectors
}
