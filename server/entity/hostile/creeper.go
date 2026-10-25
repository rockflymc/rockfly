package hostile

import (
	"sync/atomic"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/ai"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// Creeper represents a silent, explosive hostile mob that swells and detonates near targets.
type Creeper struct {
	*Hostile

	fuse           int
	maxFuse        int
	swellDirection int
	charged        bool
	ignited        bool
	exploded       atomic.Bool
	swellGoal      *CreeperSwellGoal
}

// NewCreeper initializes a Creeper entity with standard fuse and detonation behaviors.
func NewCreeper(handle *world.EntityHandle) *Creeper {
	c := &Creeper{
		Hostile: NewHostile(handle, 20.0),
		maxFuse: 30, // 30 ticks = 1.5 seconds fuse
	}
	c.SetSpeed(0.25)

	c.swellGoal = NewCreeperSwellGoal(c)

	c.GoalSelector().AddGoal(1, ai.NewFloatGoal(c, func() bool { return false }))
	c.GoalSelector().AddGoal(2, c.swellGoal)
	// Approach target to enter detonation range
	c.GoalSelector().AddGoal(3, ai.NewMeleeAttackGoal(c, c, 2.5, 0.0, nil))
	c.GoalSelector().AddGoal(4, ai.NewRandomStrollGoal(c, 10))
	c.GoalSelector().AddGoal(5, ai.NewLookAtEntityGoal(c, 8.0, func(radius float64) world.Entity {
		if e, ok := c.Target().(world.Entity); ok {
			return e
		}
		return nil
	}))

	// Configure target selectors
	c.TargetSelector().AddGoal(1, ai.NewHurtByTargetGoal(c, c, c.LastAttacker))

	return c
}

// Fuse returns the current fuse counter in ticks.
func (c *Creeper) Fuse() int {
	return c.fuse
}

// SetFuse manually sets the current fuse ticks.
func (c *Creeper) SetFuse(ticks int) {
	c.fuse = ticks
}

// MaxFuse returns the total fuse duration required to trigger detonation.
func (c *Creeper) MaxFuse() int {
	return c.maxFuse
}

// SwellDirection returns +1 if priming, -1 if defusing, or 0 if idle.
func (c *Creeper) SwellDirection() int {
	return c.swellDirection
}

// SetSwellDirection sets the current swelling progression direction.
func (c *Creeper) SetSwellDirection(dir int) {
	c.swellDirection = dir
}

// Charged returns true if this creeper is supercharged by lightning.
func (c *Creeper) Charged() bool {
	return c.charged
}

// SetCharged configures the supercharged status.
func (c *Creeper) SetCharged(charged bool) {
	c.charged = charged
}

// Ignited returns true if the creeper was manually primed (e.g. flint and steel).
func (c *Creeper) Ignited() bool {
	return c.ignited
}

// SetIgnited manually primes the creeper fuse.
func (c *Creeper) SetIgnited(ignited bool) {
	c.ignited = ignited
}

// Exploded returns true if the creeper has detonated.
func (c *Creeper) Exploded() bool {
	return c.exploded.Load()
}

// ExplosionRadius returns 3.0 blocks for normal, or 6.0 blocks for charged creepers.
func (c *Creeper) ExplosionRadius() float64 {
	if c.charged {
		return 6.0
	}
	return 3.0
}

// Explode detonates the creeper, inflicting radial explosion damage to nearby entities.
func (c *Creeper) Explode(tx *world.Tx) {
	if c == nil || !c.exploded.CompareAndSwap(false, true) {
		return
	}
	radius := c.ExplosionRadius()
	pos := c.Position()

	if tx != nil {
		box := cube.Box(pos[0]-radius, pos[1]-radius, pos[2]-radius, pos[0]+radius, pos[1]+radius, pos[2]+radius)
		for e := range tx.EntitiesWithin(box) {
			if e == c {
				continue
			}
			dist := e.Position().Sub(pos).Len()
			if dist <= radius {
				factor := 1.0 - (dist / radius)
				if factor < 0 {
					factor = 0
				} else if factor > 1 {
					factor = 1
				}
				maxDamage := 43.0
				if c.charged {
					maxDamage = 85.0
				}
				damage := factor * maxDamage
				if living, ok := e.(entity.Living); ok {
					living.Hurt(damage, entity.ExplosionDamageSource{})
					living.KnockBack(pos, factor*1.5, 0.4)
				}
			}
		}
	}
	_ = c.Close()
}

// Tick coordinates fuse progression and AI execution.
func (c *Creeper) Tick(tx *world.Tx, current int64) {
	if c.exploded.Load() {
		return
	}

	// Update fuse ticks based on swelling direction
	if c.swellDirection > 0 {
		c.fuse++
		if c.fuse >= c.maxFuse {
			c.Explode(tx)
			return
		}
	} else if c.swellDirection < 0 {
		c.fuse--
		if c.fuse < 0 {
			c.fuse = 0
		}
	}

	c.TickHostile(tx, current)
}

// BoundingBox returns the standard creeper collision box.
func (c *Creeper) BoundingBox() cube.BBox {
	return cube.Box(-0.3, 0, -0.3, 0.3, 1.7, 0.3)
}

// Metadata generates the Bedrock protocol entity metadata snapshot.
func (c *Creeper) Metadata() protocol.EntityMetadata {
	builder := entity.NewMetadataBuilder().
		SyncLiving(c.LivingEntity).
		SetDimensions(0.6, 1.7).
		SetScale(1.0)

	m := builder.Build()
	m[protocol.EntityDataKeyFuseTime] = int32(c.fuse)
	m[protocol.EntityDataKeySwell] = int32(c.fuse)
	m[protocol.EntityDataKeySwellDirection] = int32(c.swellDirection)

	if c.charged {
		m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagPowered)
	}
	if c.fuse > 0 || c.ignited {
		m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagIgnited)
	}
	return m
}

// -------------------------------------------------------------------------
// CreeperSwellGoal
// -------------------------------------------------------------------------

// CreeperSwellGoal coordinates target proximity detection, fuse countdown, and defusing.
type CreeperSwellGoal struct {
	ai.BaseGoal
	creeper *Creeper
}

// NewCreeperSwellGoal creates a CreeperSwellGoal.
func NewCreeperSwellGoal(creeper *Creeper) *CreeperSwellGoal {
	return &CreeperSwellGoal{
		BaseGoal: ai.NewBaseGoal(0), // Swelling does not conflict with moving/looking
		creeper:  creeper,
	}
}

func (g *CreeperSwellGoal) CanUse() bool {
	if g.creeper == nil || g.creeper.Dead() || g.creeper.Exploded() {
		return false
	}
	if g.creeper.Ignited() {
		return true
	}
	target := g.creeper.Target()
	return target != nil && !target.Dead()
}

func (g *CreeperSwellGoal) CanContinueToUse() bool {
	return g.CanUse()
}

func (g *CreeperSwellGoal) Start() {
	if g.creeper != nil {
		g.creeper.SetSwellDirection(1)
	}
}

func (g *CreeperSwellGoal) Stop() {
	if g.creeper != nil && !g.creeper.Ignited() {
		g.creeper.SetSwellDirection(-1)
	}
}

func (g *CreeperSwellGoal) Tick() {
	if g.creeper == nil {
		return
	}
	if g.creeper.Ignited() {
		g.creeper.SetSwellDirection(1)
		return
	}

	target := g.creeper.Target()
	if target == nil || target.Dead() {
		g.creeper.SetSwellDirection(-1)
		return
	}

	dist := target.Position().Sub(g.creeper.Position()).Len()
	if dist <= 3.0 {
		// Within explosive fuse trigger radius
		g.creeper.SetSwellDirection(1)
	} else if dist > 7.0 {
		// Target has fled beyond fuse retention distance; defuse
		g.creeper.SetSwellDirection(-1)
	}
}
