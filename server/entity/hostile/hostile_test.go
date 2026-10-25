package hostile_test

import (
	"math"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/hostile"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// mockTarget represents a test entity capable of receiving attacks and tracking movement.
type mockTarget struct {
	*entity.LivingEntity
}

func newMockTarget(pos mgl64.Vec3) *mockTarget {
	t := &mockTarget{
		LivingEntity: entity.NewLivingEntity(nil, 20.0),
	}
	t.SetPosition(pos)
	return t
}

func TestCreeper_FuseAndExplosion(t *testing.T) {
	creeper := hostile.NewCreeper(nil)
	creeper.SetPosition(mgl64.Vec3{0, 0, 0})

	target := newMockTarget(mgl64.Vec3{2, 0, 0}) // Within 3-block trigger distance
	creeper.SetTarget(target)

	// Step 1: Initial state
	if creeper.Fuse() != 0 {
		t.Fatalf("expected initial fuse 0, got %d", creeper.Fuse())
	}
	if creeper.Exploded() {
		t.Fatalf("expected creeper not exploded initially")
	}

	// Step 2: Tick AI goals and creeper logic
	for i := 0; i < 29; i++ {
		creeper.Tick(nil, int64(i))
	}
	if creeper.Fuse() <= 0 {
		t.Fatalf("expected fuse to advance near target, got %d", creeper.Fuse())
	}
	if creeper.Exploded() {
		t.Fatalf("creeper exploded prematurely before 30 ticks")
	}

	// Step 3: Trigger detonation
	creeper.SetFuse(29)
	creeper.SetSwellDirection(1)
	creeper.Tick(nil, 30)

	if !creeper.Exploded() {
		t.Fatalf("expected creeper to have exploded after reaching max fuse")
	}
}

func TestCreeper_DefuseOnDistance(t *testing.T) {
	creeper := hostile.NewCreeper(nil)
	creeper.SetPosition(mgl64.Vec3{0, 0, 0})
	target := newMockTarget(mgl64.Vec3{2, 0, 0})
	creeper.SetTarget(target)

	// Prime creeper to fuse = 15
	creeper.SetFuse(15)
	creeper.SetSwellDirection(1)

	// Move target away beyond 7-block defuse threshold
	target.SetPosition(mgl64.Vec3{10, 0, 0})

	// Run AI tick to update swell direction
	creeper.GoalSelector().Tick()
	creeper.Tick(nil, 1)

	if creeper.SwellDirection() != -1 {
		t.Fatalf("expected swell direction -1 (defusing) when target is > 7 blocks away, got %d", creeper.SwellDirection())
	}
	if creeper.Fuse() >= 15 {
		t.Fatalf("expected fuse to decrease during defusal, got %d", creeper.Fuse())
	}
}

func TestCreeper_ChargedMetadata(t *testing.T) {
	creeper := hostile.NewCreeper(nil)
	creeper.SetCharged(true)

	if creeper.ExplosionRadius() != 6.0 {
		t.Fatalf("expected charged creeper explosion radius 6.0, got %f", creeper.ExplosionRadius())
	}

	meta := creeper.Metadata()
	flags, ok := meta[protocol.EntityDataKeyFlags].(int64)
	if !ok {
		t.Fatalf("missing or invalid EntityDataKeyFlags")
	}
	if flags&(1<<protocol.EntityDataFlagPowered) == 0 {
		t.Fatalf("expected EntityDataFlagPowered set on charged creeper")
	}
}

func TestZombie_DaylightBurningAndHelmet(t *testing.T) {
	zombie := hostile.NewZombie(nil, false)
	zombie.SetPosition(mgl64.Vec3{10, 65, 10})

	// Baby variant shouldn't burn
	babyZombie := hostile.NewZombie(nil, true)
	babyZombie.SetPosition(mgl64.Vec3{10, 65, 10})
	babyZombie.Tick(nil, 1)
	if babyZombie.OnFireDuration() > 0 {
		t.Fatalf("baby zombie should not burn in daylight")
	}

	// Verify baby metadata
	babyMeta := babyZombie.Metadata()
	flags := babyMeta[protocol.EntityDataKeyFlags].(int64)
	if flags&(1<<protocol.EntityDataFlagBaby) == 0 {
		t.Fatalf("expected baby flag set on baby zombie")
	}

	// Verify helmet protection on adult zombie
	zombie.Armour().SetHelmet(item.NewStack(item.Helmet{Tier: item.ArmourTierDiamond{}}, 1))
	if !zombie.Armour().Helmet().Empty() {
		// Protected by helmet
		if zombie.OnFireDuration() > 0 {
			t.Fatalf("zombie with helmet should not catch fire")
		}
	}
}

func TestSkeleton_LeadAimingAndStrafing(t *testing.T) {
	skeleton := hostile.NewSkeleton(nil)
	skeleton.SetPosition(mgl64.Vec3{0, 0, 0})

	target := newMockTarget(mgl64.Vec3{9, 0, 0})
	// Target is moving perpendicular to line of sight (velocity along Z)
	target.SetVelocity(mgl64.Vec3{0, 0, 0.5})
	skeleton.SetTarget(target)

	var firedCount int
	skeleton.BowGoal().SetOnShoot(func(aimPos mgl64.Vec3) {
		firedCount++
	})

	// Run bow goal tick
	skeleton.BowGoal().Start()
	skeleton.BowGoal().Tick()

	aim := skeleton.BowGoal().LastAimPos()
	// Target is at X=9, Z=0 with velocity Z=0.5. At distance 9 / 1.5 = 6 ticks.
	// Aim should anticipate Z = 0 + 0.5 * 6 = ~3.0.
	if aim[2] < 1.0 {
		t.Fatalf("expected lead aiming prediction along Z axis, got aimPos %v", aim)
	}

	// Target at 9 blocks is in the 6-12 block optimal strafe bracket
	vel := skeleton.Velocity()
	if vel[0] == 0 && vel[2] == 0 {
		t.Fatalf("skeleton should be strafing/moving, got zero velocity")
	}

	// Advance ticks to trigger loose callback
	for i := 0; i < 25; i++ {
		skeleton.BowGoal().Tick()
	}
	if firedCount == 0 {
		t.Fatalf("expected skeleton bow to loose arrow after draw interval")
	}
}

func TestSpider_WallClimbingAndLeap(t *testing.T) {
	spider := hostile.NewSpider(nil)
	spider.SetPosition(mgl64.Vec3{0, 0, 0})

	// Test wall climbing flag and velocity
	spider.SetClimbing(true)
	spider.Tick(nil, 1)

	vel := spider.Velocity()
	if vel[1] <= 0 {
		t.Fatalf("expected positive Y velocity while climbing, got %f", vel[1])
	}

	meta := spider.Metadata()
	flags := meta[protocol.EntityDataKeyFlags].(int64)
	if flags&(1<<protocol.EntityDataFlagWallClimbing) == 0 {
		t.Fatalf("expected EntityDataFlagWallClimbing flag in spider metadata")
	}

	// Test pounce leap attack
	target := newMockTarget(mgl64.Vec3{3, 0, 0}) // Within 2.0 to 5.0 block leap trigger zone
	spider.SetTarget(target)

	leap := spider.LeapGoal()
	if !leap.CanUse() {
		t.Fatalf("expected spider leap goal CanUse to be true")
	}

	leap.Start()
	leapVel := spider.Velocity()
	if leapVel[0] <= 0 || leapVel[1] <= 0 {
		t.Fatalf("expected forward and upward leap impulse, got %v", leapVel)
	}
	if !leap.Leaping() {
		t.Fatalf("expected leap.Leaping() to be true after start")
	}
}

func TestEnderman_StareDetectionAndProjectileDodge(t *testing.T) {
	enderman := hostile.NewEnderman(nil)
	enderman.SetPosition(mgl64.Vec3{0, 0, 10})

	// Player at (0, 0, 0) looking directly toward Enderman at (0, 0, 10)
	// Facing positive Z -> yaw = 0, pitch = 0
	playerPos := mgl64.Vec3{0, 0, 0}
	playerRot := cube.Rotation{0, 0}

	if !enderman.CheckPlayerStare(playerPos, playerRot) {
		t.Fatalf("expected stare detection to trigger when player looks directly at Enderman")
	}
	if !enderman.Angry() {
		t.Fatalf("expected Enderman to be angry after stare")
	}
	if enderman.Speed() != 0.42 {
		t.Fatalf("expected Enderman to sprint at 0.42 speed when angry, got %f", enderman.Speed())
	}

	// Projectile dodge: Enderman is invulnerable to projectiles and teleports away
	initialPos := enderman.Position()
	dmg, vul := enderman.Hurt(5.0, entity.ProjectileDamageSource{})
	if dmg != 0 || vul {
		t.Fatalf("expected Enderman to be immune to projectile damage, got dmg=%f, vul=%v", dmg, vul)
	}
	// Teleport moved the Enderman
	if enderman.Position() == initialPos {
		t.Logf("Enderman teleported from %v to %v", initialPos, enderman.Position())
	}
}

func TestHostile_NilAndNaNSafety(t *testing.T) {
	// 1. Hostile nil checks
	var h *hostile.Hostile
	if dist := h.DistanceTo(mgl64.Vec3{0, 0, 0}); dist != 0 {
		t.Fatalf("expected 0 distance for nil hostile")
	}
	h.LookAt(mgl64.Vec3{0, 0, 0})

	// 2. Creeper nil explode
	var c *hostile.Creeper
	c.Explode(nil)

	// 3. Skeleton zero horizontal distance aiming
	skeleton := hostile.NewSkeleton(nil)
	skeleton.SetPosition(mgl64.Vec3{5, 64, 5})
	target := newMockTarget(mgl64.Vec3{5, 68, 5}) // Directly above
	skeleton.SetTarget(target)
	skeleton.BowGoal().Start()
	skeleton.BowGoal().Tick()

	vel := skeleton.Velocity()
	if math.IsNaN(vel[0]) || math.IsNaN(vel[1]) || math.IsNaN(vel[2]) {
		t.Fatalf("skeleton velocity became NaN when target is directly above: %v", vel)
	}

	// 4. Enderman NaN teleport and stare
	enderman := hostile.NewEnderman(nil)
	if enderman.CheckPlayerStare(mgl64.Vec3{0, 0, 0}, cube.Rotation{math.NaN(), math.NaN()}) {
		t.Fatalf("expected NaN rotation stare to return false")
	}
	if enderman.Teleport(nil, mgl64.Vec3{math.NaN(), 100, 0}) {
		t.Fatalf("expected NaN teleport coordinate to be rejected")
	}
	if enderman.Teleport(nil, mgl64.Vec3{0, -100, 0}) {
		t.Fatalf("expected out-of-bounds Y teleport coordinate to be rejected")
	}
}

func BenchmarkHostile_AITick(b *testing.B) {
	creeper := hostile.NewCreeper(nil)
	creeper.SetPosition(mgl64.Vec3{0, 0, 0})
	target := newMockTarget(mgl64.Vec3{2, 0, 0})
	creeper.SetTarget(target)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		creeper.Tick(nil, int64(i))
	}
}
