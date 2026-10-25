package hostile

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/ai"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// Zombie represents a hostile undead mob that pursues players and villagers in melee combat
// and ignites in direct sunlight unless wearing head armor or being a baby.
type Zombie struct {
	*Hostile
	baby bool
}

// NewZombie initializes a Zombie entity with standard or baby attributes.
func NewZombie(handle *world.EntityHandle, baby bool) *Zombie {
	z := &Zombie{
		Hostile: NewHostile(handle, 20.0),
		baby:    baby,
	}

	speed := 0.23
	if baby {
		speed = 0.30 // Baby zombies are ~30% faster
	}
	z.SetSpeed(speed)

	// Configure action goals
	z.GoalSelector().AddGoal(1, ai.NewFloatGoal(z, func() bool {
		return false
	}))
	z.GoalSelector().AddGoal(2, ai.NewMeleeAttackGoal(z, z, 1.8, 3.0, func(target ai.Mob, damage float64) {
		if living, ok := target.(entity.Living); ok {
			living.Hurt(damage, entity.AttackDamageSource{Attacker: z})
		}
	}))
	z.GoalSelector().AddGoal(3, ai.NewRandomStrollGoal(z, 10))
	z.GoalSelector().AddGoal(4, ai.NewLookAtEntityGoal(z, 8.0, func(radius float64) world.Entity {
		if e, ok := z.Target().(world.Entity); ok {
			return e
		}
		return nil
	}))

	// Configure target selectors
	z.TargetSelector().AddGoal(1, ai.NewHurtByTargetGoal(z, z, z.LastAttacker))

	return z
}

// Baby returns true if this zombie is a child variant.
func (z *Zombie) Baby() bool {
	return z.baby
}

// SetBaby configures the baby variant state.
func (z *Zombie) SetBaby(baby bool) {
	if z == nil {
		return
	}
	z.baby = baby
	if baby {
		z.SetSpeed(0.30)
	} else {
		z.SetSpeed(0.23)
	}
}

// Tick executes zombie autonomous logic, including daylight combustion and AI routines.
func (z *Zombie) Tick(tx *world.Tx, current int64) {
	if z == nil {
		return
	}
	if !z.Dead() && !z.baby {
		CheckDaylightBurning(z, z.LivingEntity, tx)
	}
	z.TickHostile(tx, current)
}

// BoundingBox returns the collision box conforming to adult or baby dimensions.
func (z *Zombie) BoundingBox() cube.BBox {
	if z.baby {
		return cube.Box(-0.15, 0, -0.15, 0.15, 0.95, 0.15)
	}
	return cube.Box(-0.3, 0, -0.3, 0.3, 1.9, 0.3)
}

// Metadata generates the Bedrock protocol entity metadata snapshot.
func (z *Zombie) Metadata() protocol.EntityMetadata {
	builder := entity.NewMetadataBuilder().SyncLiving(z.LivingEntity)
	if z.baby {
		builder.SetDimensions(0.3, 0.95).SetBaby(true).SetScale(0.5)
	} else {
		builder.SetDimensions(0.6, 1.9).SetBaby(false).SetScale(1.0)
	}
	return builder.Build()
}
