package block

import (
	"math/rand/v2"
	"sync"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

var (
	_ world.Block                     = SculkSensor{}
	_ world.Item                      = SculkSensor{}
	_ world.ScheduledTicker           = SculkSensor{}
	_ world.RedstonePowerSource       = SculkSensor{}
	_ world.RedstoneStrongPowerSource = SculkSensor{}
	_ world.RedstoneWeakBlockPowerer  = SculkSensor{}

	sculkMu      sync.RWMutex
	sculkFreqMap = make(map[cube.Pos]int)
)

// VibrationFrequency represents standard vanilla vibration frequencies (1-15).
type VibrationFrequency int

const (
	VibrationStep            VibrationFrequency = 1
	VibrationFlap            VibrationFrequency = 2
	VibrationSwim            VibrationFrequency = 3
	VibrationElytraGlide     VibrationFrequency = 4
	VibrationEntityHitGround VibrationFrequency = 5
	VibrationSplashing       VibrationFrequency = 6
	VibrationProjectileShoot VibrationFrequency = 7
	VibrationProjectileLand  VibrationFrequency = 8
	VibrationEntityDamage    VibrationFrequency = 9
	VibrationEquipArmor      VibrationFrequency = 10
	VibrationEntityInteract  VibrationFrequency = 11
	VibrationEntityAttack    VibrationFrequency = 12
	VibrationMobDie          VibrationFrequency = 13
	VibrationContainerAction VibrationFrequency = 14
	VibrationExplosion       VibrationFrequency = 15
)

// SculkSensor is a block that detects vibrations and outputs redstone power.
// It also has a calibrated variant that filters vibrations by frequency using redstone input.
type SculkSensor struct {
	solid
	transparent

	// Calibrated indicates whether this is a calibrated sculk sensor.
	Calibrated bool
	// Facing is the horizontal direction the calibrated sensor faces.
	Facing cube.Direction
	// Phase is the current phase of the sculk sensor (0: Inactive, 1: Active, 2: Cooldown).
	Phase int
	// Frequency is the vibration frequency that triggered the sensor (1-15).
	Frequency int
}

// RedstoneWeaklyPowersBlocks indicates whether weak power can conduct to adjacent blocks.
func (SculkSensor) RedstoneWeaklyPowersBlocks() bool {
	return true
}

// RedstonePower emits signal strength equal to the vibration frequency when active (Phase == 1).
func (s SculkSensor) RedstonePower(pos cube.Pos, _ *world.Tx, _ cube.Face) int {
	if s.Phase == 1 {
		sculkMu.RLock()
		defer sculkMu.RUnlock()
		f := s.Frequency
		if mapped, ok := sculkFreqMap[pos]; ok {
			f = mapped
		}
		if f < 0 {
			return 0
		}
		if f > 15 {
			return 15
		}
		return f
	}
	return 0
}

// RedstoneStrongPower emits strong signal strength into adjacent blocks when active.
func (s SculkSensor) RedstoneStrongPower(pos cube.Pos, tx *world.Tx, face cube.Face) int {
	return s.RedstonePower(pos, tx, face)
}

// Trigger activates the sculk sensor with the given vibration frequency if not on cooldown or filtered.
func (s SculkSensor) Trigger(pos cube.Pos, tx *world.Tx, freq VibrationFrequency) bool {
	if s.Phase != 0 || tx == nil || pos.OutOfBounds(tx.Range()) {
		return false
	}
	intFreq := int(freq)
	if intFreq < 1 || intFreq > 15 {
		return false
	}

	// Calibrated check: filters vibrations based on redstone signal entering the rear face.
	if s.Calibrated {
		rearFace := s.Facing.Opposite().Face()
		requiredFreq := tx.RedstonePowerFrom(pos, rearFace)
		if requiredFreq > 0 && intFreq != requiredFreq {
			return false
		}
	}

	sculkMu.Lock()
	sculkFreqMap[pos] = intFreq
	sculkMu.Unlock()

	s.Phase = 1
	s.Frequency = intFreq
	tx.SetBlock(pos, s, nil)
	tx.ScheduleBlockUpdate(pos, s, redstoneTicks(15)) // 30 game ticks
	tx.PlaySound(pos.Vec3Centre(), sound.Click{})
	return true
}

// ScheduledTick advances the phase from Active -> Cooldown -> Inactive.
func (s SculkSensor) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	if tx == nil || pos.OutOfBounds(tx.Range()) {
		return
	}
	switch s.Phase {
	case 1:
		// Active -> Cooldown (Phase 2, 10 game ticks)
		sculkMu.Lock()
		delete(sculkFreqMap, pos)
		sculkMu.Unlock()

		s.Phase = 2
		s.Frequency = 0
		tx.SetBlock(pos, s, nil)
		tx.ScheduleBlockUpdate(pos, s, redstoneTicks(5))
	case 2:
		// Cooldown -> Inactive (Phase 0)
		s.Phase = 0
		tx.SetBlock(pos, s, nil)
	}
}

// UseOnBlock places the sculk sensor on a block surface.
func (s SculkSensor) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) (used bool) {
	pos, _, used = firstReplaceable(tx, pos, face, s)
	if !used {
		return false
	}
	s.Facing = user.Rotation().Direction().Opposite()
	place(tx, pos, s, user, ctx)
	return placed(ctx)
}

// BreakInfo ...
func (s SculkSensor) BreakInfo() BreakInfo {
	return newBreakInfo(1.5, alwaysHarvestable, hoeEffective, oneOf(s)).withBreakHandler(func(pos cube.Pos, _ *world.Tx, _ item.User) {
		sculkMu.Lock()
		delete(sculkFreqMap, pos)
		sculkMu.Unlock()
	})
}

// EncodeItem ...
func (s SculkSensor) EncodeItem() (name string, meta int16) {
	if s.Calibrated {
		return "minecraft:calibrated_sculk_sensor", 0
	}
	return "minecraft:sculk_sensor", 0
}

// EncodeBlock ...
func (s SculkSensor) EncodeBlock() (string, map[string]any) {
	if s.Calibrated {
		return "minecraft:calibrated_sculk_sensor", map[string]any{
			"minecraft:cardinal_direction": s.Facing.String(),
			"sculk_sensor_phase":           int32(s.Phase),
		}
	}
	return "minecraft:sculk_sensor", map[string]any{
		"sculk_sensor_phase": int32(s.Phase),
	}
}

// allSculkSensors returns all permutations of standard and calibrated sculk sensors.
func allSculkSensors() (sensors []world.Block) {
	// Standard sculk sensor: 3 phases
	for phase := 0; phase <= 2; phase++ {
		sensors = append(sensors, SculkSensor{Phase: phase})
	}
	// Calibrated sculk sensor: 4 directions x 3 phases = 12
	for _, d := range cube.Directions() {
		for phase := 0; phase <= 2; phase++ {
			sensors = append(sensors, SculkSensor{
				Calibrated: true,
				Facing:     d,
				Phase:      phase,
			})
		}
	}
	return sensors
}
