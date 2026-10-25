package passive_test

import (
	"math"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/ai"
	"github.com/df-mc/dragonfly/server/entity/passive"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type mockPlayer struct {
	*entity.LivingEntity
}

func newMockPlayer(pos mgl64.Vec3) *mockPlayer {
	p := &mockPlayer{
		LivingEntity: entity.NewLivingEntity(nil, 20.0),
	}
	p.SetPosition(pos)
	return p
}

func TestAnimal_BreedingAndMaturation(t *testing.T) {
	cow1 := passive.NewCow(nil, false)
	cow1.SetPosition(mgl64.Vec3{0, 0, 0})

	cow2 := passive.NewCow(nil, false)
	cow2.SetPosition(mgl64.Vec3{1.0, 0, 0})

	// Prime both into love mode
	cow1.SetInLove(600)
	cow2.SetInLove(600)

	var spawnedBaby *passive.Animal
	breedGoal := passive.NewAnimalBreedGoal(cow1.Animal, func(radius float64) *passive.Animal {
		return cow2.Animal
	}, func(tx *world.Tx, pos mgl64.Vec3) *passive.Animal {
		baby := passive.NewCow(nil, true).Animal
		baby.SetPosition(pos)
		spawnedBaby = baby
		return baby
	})

	if !breedGoal.CanUse() {
		t.Fatalf("expected breedGoal.CanUse to be true")
	}

	breedGoal.Start()
	breedGoal.Tick()

	if spawnedBaby == nil {
		t.Fatalf("expected baby cow to spawn upon breeding")
	}
	if !spawnedBaby.Baby() {
		t.Fatalf("expected spawned animal to be baby")
	}
	if cow1.InLove() || cow2.InLove() {
		t.Fatalf("expected parents to exit love mode after breeding")
	}
	if cow1.Age() <= 0 || cow2.Age() <= 0 {
		t.Fatalf("expected parents to enter breeding cooldown")
	}

	// Test juvenile maturation countdown
	spawnedBaby.SetAge(-2)
	spawnedBaby.TickAnimal(nil, 1)
	if spawnedBaby.Age() != -1 {
		t.Fatalf("expected age -1, got %d", spawnedBaby.Age())
	}
	spawnedBaby.TickAnimal(nil, 2)
	if spawnedBaby.Baby() {
		t.Fatalf("expected animal to have reached adulthood when age == 0")
	}
}

func TestAnimal_PanicOnDamage(t *testing.T) {
	cow := passive.NewCow(nil, false)
	cow.SetPosition(mgl64.Vec3{0, 0, 0})

	// Animal struck
	cow.Hurt(2.0, entity.AttackDamageSource{})
	if !cow.Panicking() {
		t.Fatalf("expected cow to enter panic state upon damage")
	}

	panicGoal := passive.NewAnimalPanicGoal(cow.Animal, 1.5)
	if !panicGoal.CanUse() {
		t.Fatalf("expected panicGoal.CanUse to be true while panicking")
	}

	panicGoal.Start()
	panicGoal.Tick()

	vel := cow.Velocity()
	if vel[0] == 0 && vel[2] == 0 {
		t.Fatalf("expected non-zero flee velocity during panic")
	}
}

func TestAnimal_TemptGoal(t *testing.T) {
	cow := passive.NewCow(nil, false)
	cow.SetPosition(mgl64.Vec3{0, 0, 0})

	player := newMockPlayer(mgl64.Vec3{5, 0, 0})

	temptGoal := passive.NewAnimalTemptGoal(cow.Animal, 10.0, func(it world.Item) bool {
		return true
	}, func(radius float64) ai.Mob {
		return player
	})
	temptGoal.SetTargetPlayer(player)

	if !temptGoal.CanUse() {
		t.Fatalf("expected tempt goal CanUse to be true")
	}

	temptGoal.Tick()
	vel := cow.Velocity()
	if vel[0] <= 0 {
		t.Fatalf("expected cow to move toward player holding bait, got velocity %v", vel)
	}
}

func TestSheep_ShearingAndRegrowth(t *testing.T) {
	sheep := passive.NewSheep(nil, item.ColourRed(), false)

	// Shearing with shears
	drops, ok := sheep.Shear(item.NewStack(item.Shears{}, 1))
	if !ok || len(drops) == 0 {
		t.Fatalf("expected successful shearing with shears")
	}
	if !sheep.Sheared() {
		t.Fatalf("expected sheep to be sheared")
	}

	wool, ok := drops[0].Item().(block.Wool)
	if !ok || wool.Colour != item.ColourRed() {
		t.Fatalf("expected red wool dropped, got %v", drops[0])
	}

	// Verify protocol metadata
	meta := sheep.Metadata()
	flags := meta[protocol.EntityDataKeyFlags].(int64)
	if flags&(1<<protocol.EntityDataFlagSheared) == 0 {
		t.Fatalf("expected EntityDataFlagSheared in metadata")
	}

	// Regrow fleece by eating grass
	sheep.EatGrass()
	if sheep.Sheared() {
		t.Fatalf("expected wool to regrow after eating grass")
	}
}

func TestCow_Milking(t *testing.T) {
	cow := passive.NewCow(nil, false)

	// Milk with empty bucket
	emptyBucket := item.NewStack(item.Bucket{}, 1)
	result, ok := cow.Milk(emptyBucket)
	if !ok {
		t.Fatalf("expected milking adult cow to succeed")
	}
	bucket, ok := result.Item().(item.Bucket)
	if !ok || bucket.Empty() {
		t.Fatalf("expected filled milk bucket")
	}

	// Calves cannot be milked
	calf := passive.NewCow(nil, true)
	_, calfMilked := calf.Milk(emptyBucket)
	if calfMilked {
		t.Fatalf("calf should not be milkable")
	}
}

func TestPig_Saddling(t *testing.T) {
	pig := passive.NewPig(nil, false)
	if pig.Saddled() {
		t.Fatalf("pig should not be saddled by default")
	}

	if !pig.Saddle() {
		t.Fatalf("expected saddling adult pig to succeed")
	}
	if !pig.Saddled() {
		t.Fatalf("expected pig to be saddled")
	}

	meta := pig.Metadata()
	flags := meta[protocol.EntityDataKeyFlags].(int64)
	if flags&(1<<protocol.EntityDataFlagSaddled) == 0 {
		t.Fatalf("expected EntityDataFlagSaddled in metadata")
	}
}

func TestChicken_SlowFallAndEgg(t *testing.T) {
	chicken := passive.NewChicken(nil, false)

	// Simulate falling with downward speed
	chicken.SetVelocity(mgl64.Vec3{0, -0.8, 0})
	chicken.Tick(nil, 1)

	vel := chicken.Velocity()
	if vel[1] < -0.05 {
		t.Fatalf("expected slow-fall flutter capping downward velocity to -0.05, got %f", vel[1])
	}
	if chicken.FallDistance() != 0 {
		t.Fatalf("expected chicken fall distance reset to 0")
	}

	// Egg laying
	egg, laid := chicken.LayEgg()
	if !laid || egg.Empty() {
		t.Fatalf("expected chicken to produce egg")
	}
}

func TestWolf_TamingAndSitting(t *testing.T) {
	wolf := passive.NewWolf(nil, false)
	if wolf.MaxHealth() != 8.0 {
		t.Fatalf("expected wild wolf max health 8.0, got %f", wolf.MaxHealth())
	}

	// Domesticate wolf
	wolf.SetTamed(true, "PlayerOwner123")
	if !wolf.Tamed() {
		t.Fatalf("expected wolf to be tamed")
	}
	if wolf.MaxHealth() != 20.0 {
		t.Fatalf("expected tamed wolf max health 20.0, got %f", wolf.MaxHealth())
	}
	if wolf.Owner() != "PlayerOwner123" {
		t.Fatalf("expected owner PlayerOwner123, got %s", wolf.Owner())
	}

	// Toggle sitting
	wolf.SetSitting(true)
	wolf.SetVelocity(mgl64.Vec3{0.5, 0, 0.5})
	wolf.Tick(nil, 1)

	vel := wolf.Velocity()
	if vel[0] != 0 || vel[2] != 0 {
		t.Fatalf("expected sitting wolf horizontal velocity to be suppressed, got %v", vel)
	}

	meta := wolf.Metadata()
	flags := meta[protocol.EntityDataKeyFlags].(int64)
	if flags&(1<<protocol.EntityDataFlagSitting) == 0 {
		t.Fatalf("expected EntityDataFlagSitting in metadata")
	}
}

func TestAnimal_NilAndNaNSafety(t *testing.T) {
	// 1. Nil animal panic goal
	var a *passive.Animal
	panicGoal := passive.NewAnimalPanicGoal(a, 1.5)
	if panicGoal.CanUse() {
		t.Fatalf("expected CanUse to be false for nil animal")
	}
	panicGoal.Tick()

	// 2. Zero vector distance tempting
	cow := passive.NewCow(nil, false)
	player := newMockPlayer(mgl64.Vec3{10, 64, 10})
	cow.SetPosition(mgl64.Vec3{10, 64, 10})

	tempt := passive.NewAnimalTemptGoal(cow.Animal, 10.0, nil, nil)
	tempt.SetTargetPlayer(player)
	tempt.Tick()

	vel := cow.Velocity()
	if math.IsNaN(vel[0]) || math.IsNaN(vel[1]) || math.IsNaN(vel[2]) {
		t.Fatalf("cow velocity became NaN during tempt at zero distance: %v", vel)
	}

	// 3. Invalid speedBoost
	badPanic := passive.NewAnimalPanicGoal(cow.Animal, math.NaN())
	if badPanic == nil {
		t.Fatalf("expected non-nil panic goal")
	}
}

func BenchmarkAnimal_AITick(b *testing.B) {
	cow := passive.NewCow(nil, false)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cow.Tick(nil, int64(i))
	}
}
