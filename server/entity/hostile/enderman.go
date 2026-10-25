package hostile

import (
	"math"
	"math/rand"
	"sync/atomic"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/ai"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// Enderman represents an imposing, tall neutral/hostile mob capable of dimensional teleportation,
// projectile deflection, and aggressive retaliation upon direct eye contact.
type Enderman struct {
	*Hostile

	angry        atomic.Bool
	screaming    atomic.Bool
	teleportGoal *EndermanAvoidWaterGoal
}

// NewEnderman initializes an Enderman entity with teleportation and gaze detection behaviors.
func NewEnderman(handle *world.EntityHandle) *Enderman {
	e := &Enderman{
		Hostile: NewHostile(handle, 40.0),
	}
	e.SetSpeed(0.30)

	e.teleportGoal = NewEndermanAvoidWaterGoal(e)

	e.GoalSelector().AddGoal(1, e.teleportGoal)
	e.GoalSelector().AddGoal(2, ai.NewMeleeAttackGoal(e, e, 2.0, 7.0, func(target ai.Mob, damage float64) {
		if living, ok := target.(entity.Living); ok {
			living.Hurt(damage, entity.AttackDamageSource{Attacker: e})
		}
	}))
	e.GoalSelector().AddGoal(3, ai.NewRandomStrollGoal(e, 10))
	e.GoalSelector().AddGoal(4, ai.NewLookAtEntityGoal(e, 16.0, func(radius float64) world.Entity {
		if ent, ok := e.Target().(world.Entity); ok {
			return ent
		}
		return nil
	}))

	// Configure target selectors
	e.TargetSelector().AddGoal(1, ai.NewHurtByTargetGoal(e, e, e.LastAttacker))

	return e
}

// Angry returns true if the enderman was provoked by a stare or attack.
func (e *Enderman) Angry() bool {
	return e.angry.Load()
}

// SetAngry updates the enderman's agitation state and modifies locomotion speed.
func (e *Enderman) SetAngry(angry bool) {
	e.angry.Store(angry)
	if angry {
		e.SetSpeed(0.42) // 1.4x sprint when provoked
	} else {
		e.SetSpeed(0.30)
	}
}

// Screaming returns true if the enderman is emitting its aggro scream.
func (e *Enderman) Screaming() bool {
	return e.screaming.Load()
}

// SetScreaming updates the scream audio/animation state.
func (e *Enderman) SetScreaming(s bool) {
	e.screaming.Store(s)
}

// CheckPlayerStare evaluates whether a player's gaze vector intersects with the Enderman's head.
func (e *Enderman) CheckPlayerStare(playerPos mgl64.Vec3, playerRot cube.Rotation) bool {
	if e == nil || e.Dead() {
		return false
	}
	if math.IsNaN(playerRot.Yaw()) || math.IsNaN(playerRot.Pitch()) || math.IsInf(playerRot.Yaw(), 0) || math.IsInf(playerRot.Pitch(), 0) {
		return false
	}
	playerEye := playerPos.Add(mgl64.Vec3{0, 1.62, 0})
	endermanHead := e.Position().Add(mgl64.Vec3{0, 2.5, 0})

	toEnderman := endermanHead.Sub(playerEye)
	dist := toEnderman.Len()
	if dist > 64.0 || dist < 0.001 {
		return false
	}

	// Calculate player forward look unit vector from yaw & pitch
	yawRad := -playerRot.Yaw() * math.Pi / 180.0
	pitchRad := -playerRot.Pitch() * math.Pi / 180.0

	cosPitch := math.Cos(pitchRad)
	lookDir := mgl64.Vec3{
		-math.Sin(yawRad) * cosPitch,
		math.Sin(pitchRad),
		math.Cos(yawRad) * cosPitch,
	}

	dot := lookDir.Dot(toEnderman.Normalize())
	// dot > 0.95 corresponds to an angle within ~18 degrees of direct eye contact
	if dot > 0.95 {
		e.SetAngry(true)
		e.SetScreaming(true)
		return true
	}
	return false
}

// Teleport moves the enderman to target coordinate if vertical clearance exists.
func (e *Enderman) Teleport(tx *world.Tx, dest mgl64.Vec3) bool {
	if e == nil {
		return false
	}
	if math.IsNaN(dest[0]) || math.IsNaN(dest[1]) || math.IsNaN(dest[2]) ||
		math.IsInf(dest[0], 0) || math.IsInf(dest[1], 0) || math.IsInf(dest[2], 0) {
		return false
	}
	if dest[1] < -64 || dest[1] > 320 {
		return false
	}
	if tx != nil {
		pos := cube.PosFromVec3(dest)
		// Destination floor must be solid, and 3 blocks above must not be solid
		bFloor := tx.Block(pos.Side(cube.FaceDown))
		if bFloor == nil || len(bFloor.Model().BBox(pos.Side(cube.FaceDown), tx)) == 0 {
			return false
		}
		for dy := 0; dy < 3; dy++ {
			b := tx.Block(pos.Add(cube.Pos{0, dy, 0}))
			if b != nil && len(b.Model().BBox(pos.Add(cube.Pos{0, dy, 0}), tx)) > 0 {
				return false
			}
		}
	}

	e.SetPosition(dest)
	return true
}

// TeleportRandomly picks a random location within radius and relocates the Enderman.
func (e *Enderman) TeleportRandomly(tx *world.Tx, radius float64) bool {
	if e == nil {
		return false
	}
	if radius <= 0 || math.IsNaN(radius) || math.IsInf(radius, 0) {
		radius = 16.0
	}
	currPos := e.Position()

	for attempt := 0; attempt < 16; attempt++ {
		offsetX := (rand.Float64()*2 - 1) * radius
		offsetY := (rand.Float64()*2 - 1) * (radius / 2)
		offsetZ := (rand.Float64()*2 - 1) * radius

		candidate := currPos.Add(mgl64.Vec3{offsetX, offsetY, offsetZ})
		if e.Teleport(tx, candidate) {
			return true
		}
	}
	return false
}

// Hurt overrides LivingEntity.Hurt to dodge all incoming projectiles with immediate teleportation.
func (e *Enderman) Hurt(damage float64, src world.DamageSource) (float64, bool) {
	if _, ok := src.(entity.ProjectileDamageSource); ok {
		// Endermen are completely invulnerable to projectiles; immediately teleport away
		e.TeleportRandomly(nil, 16.0)
		return 0, false
	}
	dmg, vul := e.Hostile.Hurt(damage, src)
	if vul {
		e.SetAngry(true)
		// Small chance to teleport when struck in melee
		if rand.Float32() < 0.15 {
			e.TeleportRandomly(nil, 16.0)
		}
	}
	return dmg, vul
}

// Tick executes autonomous Enderman logic including water damage avoidance.
func (e *Enderman) Tick(tx *world.Tx, current int64) {
	if !e.Dead() {
		// Water and rain sensitivity
		if tx != nil {
			pos := cube.PosFromVec3(e.Position())
			if tx.RainingAt(pos) {
				highest := tx.HighestLightBlocker(pos.X(), pos.Z())
				if pos.Y() >= highest {
					// Direct rain contact
					e.Hurt(1.0, entity.AttackDamageSource{})
					e.TeleportRandomly(tx, 32.0)
				}
			}
		}
	}
	e.TickHostile(tx, current)
}

// BoundingBox returns the tall, slender enderman collision box.
func (e *Enderman) BoundingBox() cube.BBox {
	return cube.Box(-0.3, 0, -0.3, 0.3, 2.9, 0.3)
}

// Metadata generates the Bedrock protocol entity metadata snapshot.
func (e *Enderman) Metadata() protocol.EntityMetadata {
	builder := entity.NewMetadataBuilder().
		SyncLiving(e.LivingEntity).
		SetDimensions(0.6, 2.9).
		SetScale(1.0)

	m := builder.Build()
	if e.angry.Load() {
		m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagAngry)
	}
	return m
}

// -------------------------------------------------------------------------
// EndermanAvoidWaterGoal
// -------------------------------------------------------------------------

// EndermanAvoidWaterGoal forces an immediate random teleport when the Enderman touches water.
type EndermanAvoidWaterGoal struct {
	ai.BaseGoal
	enderman *Enderman
}

// NewEndermanAvoidWaterGoal creates an EndermanAvoidWaterGoal.
func NewEndermanAvoidWaterGoal(enderman *Enderman) *EndermanAvoidWaterGoal {
	return &EndermanAvoidWaterGoal{
		BaseGoal: ai.NewBaseGoal(ai.ControlMove | ai.ControlJump),
		enderman: enderman,
	}
}

func (g *EndermanAvoidWaterGoal) CanUse() bool {
	return false
}

func (g *EndermanAvoidWaterGoal) CanContinueToUse() bool {
	return false
}

func (g *EndermanAvoidWaterGoal) Start() {
	g.enderman.TeleportRandomly(nil, 32.0)
}
