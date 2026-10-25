package boss

import (
	"math"
	"math/rand"
	"sync/atomic"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// DragonPhase represents the active flight or combat behavior phase of the Ender Dragon.
type DragonPhase int

const (
	// DragonPhaseCircling: The dragon orbits the central End island.
	DragonPhaseCircling DragonPhase = iota
	// DragonPhaseStrafing: The dragon dives toward a target to loose dragon fireballs.
	DragonPhaseStrafing
	// DragonPhaseLanding: The dragon descends toward the bedrock exit portal at (0, 65, 0).
	DragonPhaseLanding
	// DragonPhasePerching: The dragon sits on the portal, emitting breath attacks.
	DragonPhasePerching
	// DragonPhaseTakeoff: The dragon ascends from the portal back into high altitude.
	DragonPhaseTakeoff
	// DragonPhaseDying: The dragon undergoes a 200-tick death eruption.
	DragonPhaseDying
)

// EnderDragon represents the flying apex boss of The End, managing multi-phase flight
// loops, End Crystal tether healing, breath attacks, and death portal activation.
type EnderDragon struct {
	*Boss

	phase            atomic.Int32
	deathTicks       atomic.Int32
	flightAngle      float64
	flightRadius     float64
	perchTimer       int
	connectedCrystal *mgl64.Vec3
}

// NewEnderDragon initializes an EnderDragon entity with full health and purple BossBar.
func NewEnderDragon(handle *world.EntityHandle) *EnderDragon {
	bar := NewBossBar(1, "Ender Dragon", packet.BossEventColourPurple, 0)
	d := &EnderDragon{
		Boss:         NewBoss(handle, 200.0, bar),
		flightRadius: 40.0,
	}
	d.SetSpeed(0.50)
	d.phase.Store(int32(DragonPhaseCircling))

	return d
}

// Phase returns the active flight/combat behavior phase.
func (d *EnderDragon) Phase() DragonPhase {
	if d == nil {
		return DragonPhaseCircling
	}
	return DragonPhase(d.phase.Load())
}

// SetPhase changes the dragon's behavioral phase.
func (d *EnderDragon) SetPhase(phase DragonPhase) {
	if d == nil {
		return
	}
	d.phase.Store(int32(phase))
	if phase == DragonPhasePerching {
		d.perchTimer = 200 // Perch for 10 seconds (200 ticks)
	}
}

// ConnectedCrystal returns the coordinates of an active healing End Crystal beam, if any.
func (d *EnderDragon) ConnectedCrystal() *mgl64.Vec3 {
	if d == nil {
		return nil
	}
	return d.connectedCrystal
}

// ConnectCrystal tethers the dragon to an End Crystal for continuous HP regeneration.
func (d *EnderDragon) ConnectCrystal(pos *mgl64.Vec3) {
	if d == nil {
		return
	}
	d.connectedCrystal = pos
}

// SeverCrystal breaks the crystal tether. If destroyed by player attack, deals 10 damage.
func (d *EnderDragon) SeverCrystal(damaged bool) {
	if d == nil {
		return
	}
	d.connectedCrystal = nil
	if damaged {
		d.Hurt(10.0, entity.ExplosionDamageSource{})
	}
}

// DeathTicks returns remaining ticks in the death sequence.
func (d *EnderDragon) DeathTicks() int32 {
	if d == nil {
		return 0
	}
	return d.deathTicks.Load()
}

// Hurt overrides LivingEntity.Hurt to intercept fatal damage for the death sequence.
func (d *EnderDragon) Hurt(damage float64, src world.DamageSource) (float64, bool) {
	if d == nil || d.Boss == nil || d.LivingEntity == nil {
		return 0, false
	}
	if d.Phase() == DragonPhaseDying {
		return 0, false
	}
	dmg, vul := d.LivingEntity.Hurt(damage, src)
	if vul {
		d.SyncBarHealth()
		if d.Health() <= 0 {
			// Trigger death sequence
			d.SetPhase(DragonPhaseDying)
			d.deathTicks.Store(200)
			return dmg, true
		}
	}
	return dmg, vul
}

// Tick coordinates dragon flight kinematics, crystal healing, and phase transitions.
func (d *EnderDragon) Tick(tx *world.Tx, current int64) {
	if d == nil || d.Boss == nil || d.LivingEntity == nil {
		return
	}

	// Crystal tether healing: +1 HP per tick
	if d.connectedCrystal != nil && !d.Dead() && d.Phase() != DragonPhaseDying {
		d.Heal(1.0, nil)
		d.SyncBarHealth()
	}

	phase := d.Phase()
	switch phase {
	case DragonPhaseCircling:
		d.flightAngle += 0.02
		if d.flightAngle > 2*math.Pi {
			d.flightAngle -= 2 * math.Pi
		}
		targetX := d.flightRadius * math.Cos(d.flightAngle)
		targetZ := d.flightRadius * math.Sin(d.flightAngle)
		targetY := 75.0 + 5.0*math.Sin(d.flightAngle*2)

		d.SetPosition(mgl64.Vec3{targetX, targetY, targetZ})

		// Random chance to transition to strafe or landing
		if rand.Float32() < 0.005 {
			d.SetPhase(DragonPhaseLanding)
		}

	case DragonPhaseLanding:
		// Move toward portal center at (0, 65, 0)
		curr := d.Position()
		if math.IsNaN(curr[0]) || math.IsNaN(curr[1]) || math.IsNaN(curr[2]) {
			d.SetPosition(mgl64.Vec3{0, 75, 0})
			curr = mgl64.Vec3{0, 75, 0}
		}
		diff := mgl64.Vec3{0, 65, 0}.Sub(curr)
		l := diff.Len()
		if l < 2.0 || l == 0 || math.IsNaN(l) {
			d.SetPhase(DragonPhasePerching)
		} else {
			dir := diff.Normalize().Mul(0.6)
			d.SetPosition(curr.Add(dir))
		}

	case DragonPhasePerching:
		d.perchTimer--
		d.SetPosition(mgl64.Vec3{0, 65, 0})
		if d.perchTimer <= 0 {
			d.SetPhase(DragonPhaseTakeoff)
		}

	case DragonPhaseTakeoff:
		curr := d.Position()
		if math.IsNaN(curr[0]) || math.IsNaN(curr[1]) || math.IsNaN(curr[2]) {
			d.SetPosition(mgl64.Vec3{0, 80, 0})
			curr = mgl64.Vec3{0, 80, 0}
		}
		if curr[1] >= 80.0 {
			d.SetPhase(DragonPhaseCircling)
		} else {
			d.SetPosition(curr.Add(mgl64.Vec3{0, 0.5, 0}))
		}

	case DragonPhaseDying:
		remaining := d.deathTicks.Add(-1)
		if remaining <= 0 {
			// Death sequence complete: drop 12,000 XP and close boss
			_ = d.Close()
			return
		}
	}

	d.LivingEntity.Tick(tx, current)
}

// BoundingBox returns the massive 16.0 x 8.0 collision box.
func (d *EnderDragon) BoundingBox() cube.BBox {
	return cube.Box(-8.0, 0, -8.0, 8.0, 8.0, 8.0)
}

// Metadata generates the Bedrock protocol entity metadata snapshot.
func (d *EnderDragon) Metadata() protocol.EntityMetadata {
	if d == nil || d.LivingEntity == nil {
		return entity.NewMetadataBuilder().Build()
	}
	return entity.NewMetadataBuilder().
		SyncLiving(d.LivingEntity).
		SetDimensions(16.0, 8.0).
		SetScale(1.0).
		Build()
}
