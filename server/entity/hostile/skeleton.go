package hostile

import (
	"math"
	"math/rand"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/ai"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// Skeleton represents a hostile undead mob equipped with a bow that strafes around targets,
// performs lead projectile aiming calculations, and burns in direct sunlight.
type Skeleton struct {
	*Hostile
	bowGoal *BowAttackGoal
}

// NewSkeleton initializes a Skeleton entity with default attributes and bow combat AI.
func NewSkeleton(handle *world.EntityHandle) *Skeleton {
	s := &Skeleton{
		Hostile: NewHostile(handle, 20.0),
	}
	s.SetSpeed(0.25)

	// Configure bow attack goal with 15-block range and 20-tick (1s) draw interval
	s.bowGoal = NewBowAttackGoal(s, s, 15.0, 20, func(aimPos mgl64.Vec3) {
		// Callback when arrow is loosed
	})

	s.GoalSelector().AddGoal(1, ai.NewFloatGoal(s, func() bool { return false }))
	s.GoalSelector().AddGoal(2, s.bowGoal)
	s.GoalSelector().AddGoal(3, ai.NewRandomStrollGoal(s, 10))
	s.GoalSelector().AddGoal(4, ai.NewLookAtEntityGoal(s, 8.0, func(radius float64) world.Entity {
		if e, ok := s.Target().(world.Entity); ok {
			return e
		}
		return nil
	}))

	// Configure target selectors
	s.TargetSelector().AddGoal(1, ai.NewHurtByTargetGoal(s, s, s.LastAttacker))

	return s
}

// Tick executes skeleton autonomous behaviors including daylight combustion and AI.
func (s *Skeleton) Tick(tx *world.Tx, current int64) {
	if !s.Dead() {
		CheckDaylightBurning(s, s.LivingEntity, tx)
	}
	s.TickHostile(tx, current)
}

// BowGoal returns the active bow combat behavior controller.
func (s *Skeleton) BowGoal() *BowAttackGoal {
	return s.bowGoal
}

// BoundingBox returns the standard skeleton collision box.
func (s *Skeleton) BoundingBox() cube.BBox {
	return cube.Box(-0.3, 0, -0.3, 0.3, 1.99, 0.3)
}

// Metadata generates the Bedrock protocol entity metadata snapshot.
func (s *Skeleton) Metadata() protocol.EntityMetadata {
	return entity.NewMetadataBuilder().
		SyncLiving(s.LivingEntity).
		SetDimensions(0.6, 1.99).
		SetScale(1.0).
		Build()
}

// -------------------------------------------------------------------------
// BowAttackGoal: Ranged Combat & Strafing
// -------------------------------------------------------------------------

// BowAttackGoal coordinates ranged strafing combat and predictive arrow shooting.
type BowAttackGoal struct {
	ai.BaseGoal
	mob             ai.Mob
	holder          ai.TargetHolder
	attackRange     float64
	drawInterval    int
	chargeTicks     int
	strafeTimer     int
	strafeClockwise bool
	onShoot         func(aimPos mgl64.Vec3)
	lastAimPos      mgl64.Vec3
}

// NewBowAttackGoal creates a BowAttackGoal.
func NewBowAttackGoal(mob ai.Mob, holder ai.TargetHolder, attackRange float64, drawInterval int, onShoot func(aimPos mgl64.Vec3)) *BowAttackGoal {
	return &BowAttackGoal{
		BaseGoal:        ai.NewBaseGoal(ai.ControlMove | ai.ControlLook | ai.ControlAttack),
		mob:             mob,
		holder:          holder,
		attackRange:     attackRange,
		drawInterval:    drawInterval,
		onShoot:         onShoot,
		strafeClockwise: rand.Float32() < 0.5,
	}
}

// SetOnShoot updates the arrow loose callback.
func (g *BowAttackGoal) SetOnShoot(fn func(aimPos mgl64.Vec3)) {
	g.onShoot = fn
}

// ChargeTicks returns the current bow pull progress in ticks.
func (g *BowAttackGoal) ChargeTicks() int {
	return g.chargeTicks
}

// LastAimPos returns the most recently computed predictive aiming coordinate.
func (g *BowAttackGoal) LastAimPos() mgl64.Vec3 {
	return g.lastAimPos
}

func (g *BowAttackGoal) CanUse() bool {
	if g.mob == nil || g.mob.Dead() || g.holder == nil {
		return false
	}
	target := g.holder.Target()
	if target == nil || target.Dead() {
		return false
	}
	dist := target.Position().Sub(g.mob.Position()).Len()
	return dist <= g.attackRange
}

func (g *BowAttackGoal) CanContinueToUse() bool {
	return g.CanUse()
}

func (g *BowAttackGoal) Start() {
	g.chargeTicks = 0
	g.strafeTimer = 0
}

func (g *BowAttackGoal) Stop() {
	g.chargeTicks = 0
}

func (g *BowAttackGoal) Tick() {
	if g.mob == nil || g.holder == nil {
		return
	}
	target := g.holder.Target()
	if target == nil || target.Dead() {
		return
	}

	diff := target.Position().Sub(g.mob.Position())
	dist := diff.Len()
	if dist < 0.001 {
		return
	}

	// 1. Predictive aiming calculation (lead shooting)
	// Arrow flight speed: ~1.5 blocks/tick
	const arrowSpeed = 1.5
	flightTicks := dist / arrowSpeed
	predictedPos := target.Position().Add(target.Velocity().Mul(flightTicks))
	// Offset aiming point to chest/head height
	predictedPos[1] += 0.9
	g.lastAimPos = predictedPos

	// Rotate mob toward predicted position
	diffAim := predictedPos.Sub(g.mob.Position())
	hDist := diffAim[0]*diffAim[0] + diffAim[2]*diffAim[2]
	yaw := float64(0)
	if hDist > 0.0001 {
		yaw = -mgl64.RadToDeg(math.Atan2(diffAim[0], diffAim[2]))
	}
	pitch := -mgl64.RadToDeg(math.Atan2(diffAim[1], math.Sqrt(hDist)))
	g.mob.SetRotation(cube.Rotation{yaw, pitch})

	// 2. Combat movement & strafing
	hLen := math.Hypot(diff[0], diff[2])
	if hLen < 0.001 {
		return
	}
	dirXZ := mgl64.Vec3{diff[0] / hLen, 0, diff[2] / hLen}
	perp := mgl64.Vec3{-dirXZ[2], 0, dirXZ[0]}
	if !g.strafeClockwise {
		perp = perp.Mul(-1)
	}

	g.strafeTimer++
	if g.strafeTimer >= 40 {
		g.strafeTimer = 0
		g.strafeClockwise = !g.strafeClockwise
	}

	vel := g.mob.Velocity()
	speed := g.mob.Speed()

	switch {
	case dist > 12.0:
		// Advance toward target
		vel[0] = dirXZ[0] * speed
		vel[2] = dirXZ[2] * speed
	case dist < 6.0:
		// Retreat backwards
		vel[0] = -dirXZ[0] * speed
		vel[2] = -dirXZ[2] * speed
	default:
		// Strafe sideways while holding ground
		vel[0] = perp[0] * speed * 0.8
		vel[2] = perp[2] * speed * 0.8
	}
	g.mob.SetVelocity(vel)

	// 3. Bow charging and shooting
	g.chargeTicks++
	if g.chargeTicks >= g.drawInterval {
		g.chargeTicks = 0
		if g.onShoot != nil {
			g.onShoot(predictedPos)
		}
	}
}
