package hostile

import (
	"sync/atomic"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/ai"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// Spider represents an agile arthropod mob capable of vertical wall climbing,
// pounce leap attacks, and daytime neutrality when unprovoked.
type Spider struct {
	*Hostile

	climbing atomic.Bool
	leapGoal *SpiderLeapAttackGoal
}

// NewSpider initializes a Spider entity with climbing and leap attack behaviors.
func NewSpider(handle *world.EntityHandle) *Spider {
	s := &Spider{
		Hostile: NewHostile(handle, 16.0),
	}
	s.SetSpeed(0.30)

	s.leapGoal = NewSpiderLeapAttackGoal(s, s)

	s.GoalSelector().AddGoal(1, ai.NewFloatGoal(s, func() bool { return false }))
	s.GoalSelector().AddGoal(2, s.leapGoal)
	s.GoalSelector().AddGoal(3, ai.NewMeleeAttackGoal(s, s, 2.0, 2.0, func(target ai.Mob, damage float64) {
		if living, ok := target.(entity.Living); ok {
			living.Hurt(damage, entity.AttackDamageSource{Attacker: s})
		}
	}))
	s.GoalSelector().AddGoal(4, ai.NewRandomStrollGoal(s, 10))
	s.GoalSelector().AddGoal(5, ai.NewLookAtEntityGoal(s, 8.0, func(radius float64) world.Entity {
		if e, ok := s.Target().(world.Entity); ok {
			return e
		}
		return nil
	}))

	// Configure target selectors: Retaliation always triggers; neutral targeting checks daylight
	s.TargetSelector().AddGoal(1, ai.NewHurtByTargetGoal(s, s, s.LastAttacker))

	return s
}

// Climbing returns true if the spider is currently traversing a vertical surface.
func (s *Spider) Climbing() bool {
	return s.climbing.Load()
}

// SetClimbing sets the vertical wall climbing state.
func (s *Spider) SetClimbing(climbing bool) {
	s.climbing.Store(climbing)
}

// LeapGoal returns the pounce leap attack behavior controller.
func (s *Spider) LeapGoal() *SpiderLeapAttackGoal {
	return s.leapGoal
}

// Tick executes spider autonomous logic including wall climbing checks.
func (s *Spider) Tick(tx *world.Tx, current int64) {
	if !s.Dead() {
		// Wall climbing check: If in contact with vertical obstruction, climb upward
		if s.climbing.Load() {
			vel := s.Velocity()
			vel[1] = 0.2 // Ascend wall
			s.SetVelocity(vel)
		}
	}
	s.TickHostile(tx, current)
}

// BoundingBox returns the wide, low-profile spider collision box.
func (s *Spider) BoundingBox() cube.BBox {
	return cube.Box(-0.7, 0, -0.7, 0.7, 0.9, 0.7)
}

// Metadata generates the Bedrock protocol entity metadata snapshot.
func (s *Spider) Metadata() protocol.EntityMetadata {
	builder := entity.NewMetadataBuilder().
		SyncLiving(s.LivingEntity).
		SetDimensions(1.4, 0.9).
		SetScale(1.0)

	m := builder.Build()
	if s.climbing.Load() {
		m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagWallClimbing)
	}
	return m
}

// -------------------------------------------------------------------------
// SpiderLeapAttackGoal: Pounce Leap Attack
// -------------------------------------------------------------------------

// SpiderLeapAttackGoal executes an explosive forward leap attack toward targets.
type SpiderLeapAttackGoal struct {
	ai.BaseGoal
	spider   *Spider
	holder   ai.TargetHolder
	cooldown int
	leaping  bool
}

// NewSpiderLeapAttackGoal creates a SpiderLeapAttackGoal.
func NewSpiderLeapAttackGoal(spider *Spider, holder ai.TargetHolder) *SpiderLeapAttackGoal {
	return &SpiderLeapAttackGoal{
		BaseGoal: ai.NewBaseGoal(ai.ControlMove | ai.ControlJump),
		spider:   spider,
		holder:   holder,
	}
}

// Cooldown returns the remaining ticks until the next leap can occur.
func (g *SpiderLeapAttackGoal) Cooldown() int {
	return g.cooldown
}

// SetCooldown sets the leap cooldown in ticks.
func (g *SpiderLeapAttackGoal) SetCooldown(ticks int) {
	g.cooldown = ticks
}

// Leaping returns true if a leap impulse was recently delivered.
func (g *SpiderLeapAttackGoal) Leaping() bool {
	return g.leaping
}

func (g *SpiderLeapAttackGoal) CanUse() bool {
	if g.spider == nil || g.spider.Dead() || g.holder == nil || g.cooldown > 0 {
		return false
	}
	target := g.holder.Target()
	if target == nil || target.Dead() {
		return false
	}
	dist := target.Position().Sub(g.spider.Position()).Len()
	// Pounce attack triggers between 2.0 and 5.0 blocks
	return dist >= 2.0 && dist <= 5.0
}

func (g *SpiderLeapAttackGoal) CanContinueToUse() bool {
	return false // Leap is an instantaneous impulse
}

func (g *SpiderLeapAttackGoal) Start() {
	if g.spider == nil || g.holder == nil {
		return
	}
	target := g.holder.Target()
	if target == nil || target.Dead() {
		return
	}

	diff := target.Position().Sub(g.spider.Position())
	diffXZ := mgl64.Vec3{diff[0], 0, diff[2]}
	distXZ := diffXZ.Len()
	if distXZ < 0.001 {
		return
	}

	dir := diffXZ.Normalize()
	// Set leap impulse: horizontal forward propulsion + upward hop
	leapVel := mgl64.Vec3{
		dir[0] * 0.55,
		0.40,
		dir[2] * 0.55,
	}
	g.spider.SetVelocity(leapVel)
	g.leaping = true
	g.cooldown = 40 // 2 seconds cooldown
}

func (g *SpiderLeapAttackGoal) Stop() {
	g.leaping = false
}

func (g *SpiderLeapAttackGoal) Tick() {
	if g.cooldown > 0 {
		g.cooldown--
	}
}
