package passive

import (
	"math"
	"math/rand"
	"sync"
	"sync/atomic"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/ai"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// Animal serves as the common foundation for peaceful and domestic creatures,
// managing breeding cycles, maturation from baby to adult, bait temptation, and panic behavior.
type Animal struct {
	*entity.LivingEntity

	mu           sync.RWMutex
	age          atomic.Int32 // <0: baby maturing; 0: adult ready; >0: breeding cooldown
	inLove       atomic.Int32 // >0: ticks remaining in love mode
	panicTimer   atomic.Int32
	goalSelector *ai.GoalSelector
}

// NewAnimal initializes the common animal state and behavior coordinator.
func NewAnimal(handle *world.EntityHandle, maxHealth float64, baby bool) *Animal {
	a := &Animal{
		LivingEntity: entity.NewLivingEntity(handle, maxHealth),
		goalSelector: ai.NewGoalSelector(),
	}

	if baby {
		a.age.Store(-24000) // 20 minutes to mature
	} else {
		a.age.Store(0)
	}

	return a
}

// Age returns the animal's life cycle age counter.
func (a *Animal) Age() int32 {
	return a.age.Load()
}

// SetAge sets the animal's life cycle age counter.
func (a *Animal) SetAge(age int32) {
	a.age.Store(age)
}

// Baby returns true if the animal is in its juvenile/child stage.
func (a *Animal) Baby() bool {
	return a.age.Load() < 0
}

// SetBaby configures juvenile status (-24000 ticks for baby, 0 for adult).
func (a *Animal) SetBaby(baby bool) {
	if baby {
		a.age.Store(-24000)
	} else {
		a.age.Store(0)
	}
}

// InLove returns true if the animal is actively primed for breeding.
func (a *Animal) InLove() bool {
	return a.inLove.Load() > 0
}

// InLoveTicks returns the remaining love ticks.
func (a *Animal) InLoveTicks() int32 {
	return a.inLove.Load()
}

// SetInLove primes the animal for breeding with the specified duration in ticks.
func (a *Animal) SetInLove(ticks int32) {
	a.inLove.Store(ticks)
}

// Panicking returns true if the animal is fleeing from damage.
func (a *Animal) Panicking() bool {
	return a.panicTimer.Load() > 0
}

// GoalSelector returns the action goal coordinator.
func (a *Animal) GoalSelector() *ai.GoalSelector {
	return a.goalSelector
}

// Hurt overrides LivingEntity.Hurt to trigger animal panic flight.
func (a *Animal) Hurt(damage float64, src world.DamageSource) (float64, bool) {
	dmg, vul := a.LivingEntity.Hurt(damage, src)
	if vul {
		a.panicTimer.Store(60) // Panic for 3 seconds (60 ticks)
	}
	return dmg, vul
}

// TickAnimal updates life cycle countdowns, panic status, AI goals, and living entity logic.
func (a *Animal) TickAnimal(tx *world.Tx, current int64) {
	a.Tick(tx, current)

	if a.Dead() {
		return
	}

	// 1. Age progression
	currAge := a.age.Load()
	switch {
	case currAge < 0:
		// Maturing
		a.age.Add(1)
	case currAge > 0:
		// Breeding cooldown
		a.age.Add(-1)
	}

	// 2. Love countdown
	if a.inLove.Load() > 0 {
		a.inLove.Add(-1)
	}

	// 3. Panic countdown
	if a.panicTimer.Load() > 0 {
		a.panicTimer.Add(-1)
	}

	// 4. Autonomous AI execution
	a.goalSelector.Tick()
}

// SyncAnimalMetadata populates common animal flags into Bedrock EntityMetadata.
func (a *Animal) SyncAnimalMetadata(builder *entity.MetadataBuilder) protocol.EntityMetadata {
	builder.SyncLiving(a.LivingEntity)
	if a.Baby() {
		builder.SetBaby(true).SetScale(0.5)
	} else {
		builder.SetBaby(false).SetScale(1.0)
	}

	m := builder.Build()
	if a.InLove() {
		m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagInLove)
	}
	return m
}

// -------------------------------------------------------------------------
// AnimalPanicGoal: Rapid Fleeing on Injury
// -------------------------------------------------------------------------

// AnimalPanicGoal forces the animal to sprint in randomized evasion directions when injured.
type AnimalPanicGoal struct {
	ai.BaseGoal
	animal     *Animal
	speedBoost float64
	escapeDir  mgl64.Vec3
}

// NewAnimalPanicGoal creates an AnimalPanicGoal.
func NewAnimalPanicGoal(animal *Animal, speedBoost float64) *AnimalPanicGoal {
	return &AnimalPanicGoal{
		BaseGoal:   ai.NewBaseGoal(ai.ControlMove),
		animal:     animal,
		speedBoost: speedBoost,
	}
}

func (g *AnimalPanicGoal) CanUse() bool {
	return !g.animal.Dead() && g.animal.Panicking()
}

func (g *AnimalPanicGoal) CanContinueToUse() bool {
	return g.CanUse()
}

func (g *AnimalPanicGoal) Start() {
	// Pick random escape angle
	angle := rand.Float64() * 2 * math.Pi
	g.escapeDir = mgl64.Vec3{math.Cos(angle), 0, math.Sin(angle)}
}

func (g *AnimalPanicGoal) Stop() {}

func (g *AnimalPanicGoal) Tick() {
	vel := g.animal.Velocity()
	speed := g.animal.Speed() * g.speedBoost
	vel[0] = g.escapeDir[0] * speed
	vel[2] = g.escapeDir[2] * speed
	g.animal.SetVelocity(vel)
}

// -------------------------------------------------------------------------
// AnimalTemptGoal: Follow Bait
// -------------------------------------------------------------------------

// AnimalTemptGoal attracts the animal toward players holding their favorite bait item.
type AnimalTemptGoal struct {
	ai.BaseGoal
	animal          *Animal
	detectionRadius float64
	isBaitItem      func(item world.Item) bool
	findPlayer      func(radius float64) ai.Mob
	targetPlayer    ai.Mob
}

// NewAnimalTemptGoal creates an AnimalTemptGoal.
func NewAnimalTemptGoal(animal *Animal, radius float64, isBait func(item world.Item) bool, findPlayer func(radius float64) ai.Mob) *AnimalTemptGoal {
	return &AnimalTemptGoal{
		BaseGoal:        ai.NewBaseGoal(ai.ControlMove | ai.ControlLook),
		animal:          animal,
		detectionRadius: radius,
		isBaitItem:      isBait,
		findPlayer:      findPlayer,
	}
}

// SetTargetPlayer manually sets the tempt target player (useful for unit testing).
func (g *AnimalTemptGoal) SetTargetPlayer(p ai.Mob) {
	g.targetPlayer = p
}

func (g *AnimalTemptGoal) CanUse() bool {
	if g.animal.Dead() || g.animal.Panicking() || g.animal.InLove() {
		return false
	}
	if g.targetPlayer != nil && !g.targetPlayer.Dead() {
		return true
	}
	if g.findPlayer == nil {
		return false
	}
	g.targetPlayer = g.findPlayer(g.detectionRadius)
	return g.targetPlayer != nil && !g.targetPlayer.Dead()
}

func (g *AnimalTemptGoal) CanContinueToUse() bool {
	if g.animal.Dead() || g.animal.Panicking() || g.targetPlayer == nil || g.targetPlayer.Dead() {
		return false
	}
	dist := g.targetPlayer.Position().Sub(g.animal.Position()).Len()
	return dist <= g.detectionRadius*1.5
}

func (g *AnimalTemptGoal) Start() {}

func (g *AnimalTemptGoal) Stop() {
	g.targetPlayer = nil
}

func (g *AnimalTemptGoal) Tick() {
	if g.targetPlayer == nil {
		return
	}
	diff := g.targetPlayer.Position().Sub(g.animal.Position())
	dist := diff.Len()

	// Rotate toward player
	hDist := diff[0]*diff[0] + diff[2]*diff[2]
	yaw := float64(0)
	if hDist > 0.0001 {
		yaw = -mgl64.RadToDeg(math.Atan2(diff[0], diff[2]))
	}
	pitch := -mgl64.RadToDeg(math.Atan2(diff[1], math.Sqrt(hDist)))
	g.animal.SetRotation(cube.Rotation{yaw, pitch})

	// Advance toward player until within comfortable reach (2.0 blocks)
	vel := g.animal.Velocity()
	if dist > 2.0 {
		dir := mgl64.Vec3{diff[0], 0, diff[2]}.Normalize()
		speed := g.animal.Speed()
		vel[0] = dir[0] * speed
		vel[2] = dir[2] * speed
	} else {
		vel[0] = 0
		vel[2] = 0
	}
	g.animal.SetVelocity(vel)
}

// -------------------------------------------------------------------------
// AnimalBreedGoal: Pair and Spawn Offspring
// -------------------------------------------------------------------------

// AnimalBreedGoal guides two in-love animals toward each other to produce offspring.
type AnimalBreedGoal struct {
	ai.BaseGoal
	animal    *Animal
	findMate  func(radius float64) *Animal
	spawnBaby func(tx *world.Tx, pos mgl64.Vec3) *Animal
	mate      *Animal
}

// NewAnimalBreedGoal creates an AnimalBreedGoal.
func NewAnimalBreedGoal(animal *Animal, findMate func(radius float64) *Animal, spawnBaby func(tx *world.Tx, pos mgl64.Vec3) *Animal) *AnimalBreedGoal {
	return &AnimalBreedGoal{
		BaseGoal:  ai.NewBaseGoal(ai.ControlMove | ai.ControlLook),
		animal:    animal,
		findMate:  findMate,
		spawnBaby: spawnBaby,
	}
}

// SetMate manually designates a partner animal.
func (g *AnimalBreedGoal) SetMate(m *Animal) {
	g.mate = m
}

func (g *AnimalBreedGoal) CanUse() bool {
	if g.animal.Dead() || !g.animal.InLove() || g.animal.Age() != 0 {
		return false
	}
	if g.mate != nil && !g.mate.Dead() && g.mate.InLove() && g.mate.Age() == 0 {
		return true
	}
	if g.findMate == nil {
		return false
	}
	g.mate = g.findMate(8.0)
	return g.mate != nil && !g.mate.Dead() && g.mate.InLove() && g.mate.Age() == 0
}

func (g *AnimalBreedGoal) CanContinueToUse() bool {
	return g.mate != nil && !g.mate.Dead() && g.mate.InLove() && g.animal.InLove()
}

func (g *AnimalBreedGoal) Start() {}

func (g *AnimalBreedGoal) Stop() {
	g.mate = nil
}

func (g *AnimalBreedGoal) Tick() {
	if g.mate == nil {
		return
	}
	diff := g.mate.Position().Sub(g.animal.Position())
	dist := diff.Len()

	if dist > 1.5 {
		// Move toward mate
		dir := mgl64.Vec3{diff[0], 0, diff[2]}.Normalize()
		speed := g.animal.Speed()
		vel := g.animal.Velocity()
		vel[0] = dir[0] * speed
		vel[2] = dir[2] * speed
		g.animal.SetVelocity(vel)
	} else {
		// Mate proximity reached: produce offspring!
		midpoint := g.animal.Position().Add(g.mate.Position()).Mul(0.5)

		// Reset love and enter cooldown
		g.animal.SetInLove(0)
		g.mate.SetInLove(0)
		g.animal.SetAge(6000) // 5 minute breeding cooldown
		g.mate.SetAge(6000)

		if g.spawnBaby != nil {
			g.spawnBaby(nil, midpoint)
		}
		g.mate = nil
	}
}
