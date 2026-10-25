package ai

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// Mob represents an entity capable of autonomous AI navigation and combat.
type Mob interface {
	Position() mgl64.Vec3
	SetPosition(pos mgl64.Vec3)
	Velocity() mgl64.Vec3
	SetVelocity(vel mgl64.Vec3)
	Rotation() cube.Rotation
	SetRotation(rot cube.Rotation)
	Speed() float64
	SetSpeed(s float64)
	Health() float64
	Dead() bool
	OnFireDuration() time.Duration
}

// TargetHolder provides access to the mob's active combat target.
type TargetHolder interface {
	Target() Mob
	SetTarget(target Mob)
}

// -------------------------------------------------------------------------
// 1. FloatGoal
// -------------------------------------------------------------------------

// FloatGoal propels submerged mobs upward to prevent drowning.
type FloatGoal struct {
	BaseGoal
	mob     Mob
	inWater func() bool
}

// NewFloatGoal creates a FloatGoal for a mob.
func NewFloatGoal(mob Mob, inWater func() bool) *FloatGoal {
	return &FloatGoal{
		BaseGoal: NewBaseGoal(ControlJump),
		mob:      mob,
		inWater:  inWater,
	}
}

func (g *FloatGoal) CanUse() bool {
	if g.mob == nil || g.mob.Dead() || g.inWater == nil {
		return false
	}
	return g.inWater()
}

func (g *FloatGoal) CanContinueToUse() bool {
	return g.CanUse()
}

func (g *FloatGoal) Tick() {
	if g.mob == nil {
		return
	}
	vel := g.mob.Velocity()
	if vel[1] < 0.25 {
		vel[1] = 0.25
		g.mob.SetVelocity(vel)
	}
}

// -------------------------------------------------------------------------
// 2. PanicGoal
// -------------------------------------------------------------------------

// PanicGoal accelerates the mob away from danger when hurt or on fire.
type PanicGoal struct {
	BaseGoal
	mob           Mob
	speedMod      float64
	panicTicks    int
	maxPanicTicks int
	isPanicking   func() bool
	originalSpeed float64
}

// NewPanicGoal creates a PanicGoal that boosts mob speed when hurt or burning.
func NewPanicGoal(mob Mob, speedMod float64, isPanicking func() bool) *PanicGoal {
	if speedMod <= 0 || math.IsNaN(speedMod) || math.IsInf(speedMod, 0) {
		speedMod = 1.25
	}
	return &PanicGoal{
		BaseGoal:      NewBaseGoal(ControlMove),
		mob:           mob,
		speedMod:      speedMod,
		maxPanicTicks: 60, // 3 seconds of panic run
		isPanicking:   isPanicking,
	}
}

func (g *PanicGoal) CanUse() bool {
	if g.mob == nil || g.mob.Dead() {
		return false
	}
	if g.mob.OnFireDuration() > 0 {
		return true
	}
	return g.isPanicking != nil && g.isPanicking()
}

func (g *PanicGoal) CanContinueToUse() bool {
	if g.mob == nil || g.mob.Dead() {
		return false
	}
	return g.panicTicks > 0
}

func (g *PanicGoal) Start() {
	if g.mob == nil {
		return
	}
	g.panicTicks = g.maxPanicTicks
	g.originalSpeed = g.mob.Speed()
	g.mob.SetSpeed(g.originalSpeed * g.speedMod)
}

func (g *PanicGoal) Stop() {
	if g.mob == nil {
		return
	}
	g.panicTicks = 0
	g.mob.SetSpeed(g.originalSpeed)
}

func (g *PanicGoal) Tick() {
	if g.mob == nil {
		return
	}
	g.panicTicks--
	// Random escape impulse
	vel := g.mob.Velocity()
	angle := rand.Float64() * 2 * math.Pi
	vel[0] = math.Cos(angle) * g.mob.Speed()
	vel[2] = math.Sin(angle) * g.mob.Speed()
	g.mob.SetVelocity(vel)
}

// -------------------------------------------------------------------------
// 3. MeleeAttackGoal
// -------------------------------------------------------------------------

// MeleeAttackGoal pursues and strikes the mob's active combat target within reach.
type MeleeAttackGoal struct {
	BaseGoal
	mob         Mob
	holder      TargetHolder
	attackRange float64
	damage      float64
	cooldown    int
	onAttack    func(target Mob, damage float64)
}

// NewMeleeAttackGoal creates a MeleeAttackGoal.
func NewMeleeAttackGoal(mob Mob, holder TargetHolder, attackRange, damage float64, onAttack func(target Mob, damage float64)) *MeleeAttackGoal {
	return &MeleeAttackGoal{
		BaseGoal:    NewBaseGoal(ControlMove | ControlAttack),
		mob:         mob,
		holder:      holder,
		attackRange: attackRange,
		damage:      damage,
		onAttack:    onAttack,
	}
}

func (g *MeleeAttackGoal) CanUse() bool {
	if g.mob == nil || g.mob.Dead() || g.holder == nil {
		return false
	}
	target := g.holder.Target()
	return target != nil && !target.Dead()
}

func (g *MeleeAttackGoal) CanContinueToUse() bool {
	return g.CanUse()
}

func (g *MeleeAttackGoal) Start() {
	g.cooldown = 0
}

func (g *MeleeAttackGoal) Stop() {}

func (g *MeleeAttackGoal) Tick() {
	if g.mob == nil || g.holder == nil {
		return
	}
	target := g.holder.Target()
	if target == nil || target.Dead() {
		return
	}

	diff := target.Position().Sub(g.mob.Position())
	dist := diff.Len()

	if g.cooldown > 0 {
		g.cooldown--
	}

	if dist <= g.attackRange {
		// Within attack reach
		if g.cooldown <= 0 {
			if g.onAttack != nil {
				g.onAttack(target, g.damage)
			}
			g.cooldown = 20 // 1 hit per second (20 ticks)
		}
	} else {
		// Move toward target
		diff[1] = 0
		if diff.Len() > 0 {
			dir := diff.Normalize()
			vel := g.mob.Velocity()
			vel[0] = dir[0] * g.mob.Speed()
			vel[2] = dir[2] * g.mob.Speed()
			g.mob.SetVelocity(vel)
		}
	}
}

// -------------------------------------------------------------------------
// 4. RandomStrollGoal
// -------------------------------------------------------------------------

// RandomStrollGoal navigates wandering wander paths when idle.
type RandomStrollGoal struct {
	BaseGoal
	mob         Mob
	walkTicks   int
	interval    int
	timer       int
	strollSpeed float64
}

// NewRandomStrollGoal creates a RandomStrollGoal.
func NewRandomStrollGoal(mob Mob, interval int) *RandomStrollGoal {
	return &RandomStrollGoal{
		BaseGoal: NewBaseGoal(ControlMove),
		mob:      mob,
		interval: interval,
	}
}

func (g *RandomStrollGoal) CanUse() bool {
	if g.mob == nil || g.mob.Dead() {
		return false
	}
	g.timer++
	return g.timer >= g.interval
}

func (g *RandomStrollGoal) CanContinueToUse() bool {
	if g.mob == nil || g.mob.Dead() {
		return false
	}
	return g.walkTicks > 0
}

func (g *RandomStrollGoal) Start() {
	if g.mob == nil {
		return
	}
	g.timer = 0
	g.walkTicks = 40 + rand.IntN(40) // Walk 2 to 4 seconds
	angle := rand.Float64() * 2 * math.Pi
	vel := g.mob.Velocity()
	vel[0] = math.Cos(angle) * g.mob.Speed() * 0.7
	vel[2] = math.Sin(angle) * g.mob.Speed() * 0.7
	g.mob.SetVelocity(vel)
}

func (g *RandomStrollGoal) Stop() {
	if g.mob == nil {
		return
	}
	g.walkTicks = 0
	vel := g.mob.Velocity()
	vel[0] = 0
	vel[2] = 0
	g.mob.SetVelocity(vel)
}

func (g *RandomStrollGoal) Tick() {
	g.walkTicks--
}

// -------------------------------------------------------------------------
// 5. LookAtEntityGoal
// -------------------------------------------------------------------------

// LookAtEntityGoal rotates mob yaw and pitch to face a nearby entity.
type LookAtEntityGoal struct {
	BaseGoal
	mob        Mob
	radius     float64
	findTarget func(radius float64) world.Entity
	lookTarget world.Entity
	lookTicks  int
}

// NewLookAtEntityGoal creates a LookAtEntityGoal.
func NewLookAtEntityGoal(mob Mob, radius float64, findTarget func(radius float64) world.Entity) *LookAtEntityGoal {
	return &LookAtEntityGoal{
		BaseGoal:   NewBaseGoal(ControlLook),
		mob:        mob,
		radius:     radius,
		findTarget: findTarget,
	}
}

func (g *LookAtEntityGoal) CanUse() bool {
	if g.mob == nil || g.mob.Dead() || g.findTarget == nil {
		return false
	}
	target := g.findTarget(g.radius)
	if target == nil {
		return false
	}
	g.lookTarget = target
	return true
}

func (g *LookAtEntityGoal) CanContinueToUse() bool {
	if g.mob == nil || g.mob.Dead() || g.lookTarget == nil || g.lookTicks <= 0 {
		return false
	}
	dist := g.lookTarget.Position().Sub(g.mob.Position()).Len()
	return dist <= g.radius
}

func (g *LookAtEntityGoal) Start() {
	g.lookTicks = 40 + rand.IntN(20) // Look for 2 to 3 seconds
}

func (g *LookAtEntityGoal) Stop() {
	g.lookTarget = nil
	g.lookTicks = 0
}

func (g *LookAtEntityGoal) Tick() {
	if g.mob == nil || g.lookTarget == nil {
		return
	}
	g.lookTicks--
	diff := g.lookTarget.Position().Sub(g.mob.Position())
	horizontalDist := math.Hypot(diff[0], diff[2])
	if horizontalDist == 0 && diff[1] == 0 {
		return
	}
	yaw := -math.Atan2(diff[0], diff[2]) * 180 / math.Pi
	pitch := -math.Atan2(diff[1], horizontalDist) * 180 / math.Pi

	g.mob.SetRotation(cube.Rotation{yaw, pitch})
}
