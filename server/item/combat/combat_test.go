package combat_test

import (
	"math"
	"testing"

	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item/combat"
	"github.com/go-gl/mathgl/mgl64"
)

func TestShield_BlockAndAxeDisable(t *testing.T) {
	shield := combat.NewShield()

	defenderPos := mgl64.Vec3{0, 0, 0}
	defenderYaw := 0.0 // Facing positive Z

	// 1. Frontal attack (attacker at Z=2): In blocking arc
	attackerFront := mgl64.Vec3{0, 0, 2}
	blocked, remaining, disabled := shield.BlockDamage(10.0, entity.AttackDamageSource{}, attackerFront, defenderPos, defenderYaw, false)
	if blocked != 10.0 || remaining != 0 || disabled {
		t.Fatalf("expected 100%% frontal damage blocked, got blocked=%f, remaining=%f, disabled=%v", blocked, remaining, disabled)
	}

	// 2. Rear attack (attacker at Z=-2): Outside blocking arc
	attackerBehind := mgl64.Vec3{0, 0, -2}
	blockedRear, remainingRear, _ := shield.BlockDamage(10.0, entity.AttackDamageSource{}, attackerBehind, defenderPos, defenderYaw, false)
	if blockedRear != 0 || remainingRear != 10.0 {
		t.Fatalf("expected rear attack to bypass shield, got blocked=%f, remaining=%f", blockedRear, remainingRear)
	}

	// 3. Axe attack from front: Blocks damage but disables shield for 5 seconds
	_, _, axeDisabled := shield.BlockDamage(12.0, entity.AttackDamageSource{}, attackerFront, defenderPos, defenderYaw, true)
	if !axeDisabled || !shield.Disabled() {
		t.Fatalf("expected shield disabled by axe attack")
	}
	if shield.DisableTicks() != 100 { // 5s = 100 ticks
		t.Fatalf("expected 100 disable ticks, got %d", shield.DisableTicks())
	}

	// 4. Attack while disabled: deals full damage
	blockedWhileDisabled, remainingWhileDisabled, _ := shield.BlockDamage(8.0, entity.AttackDamageSource{}, attackerFront, defenderPos, defenderYaw, false)
	if blockedWhileDisabled != 0 || remainingWhileDisabled != 8.0 {
		t.Fatalf("expected attack to penetrate disabled shield")
	}

	// 5. Cooldown recovery
	for i := 0; i < 100; i++ {
		shield.Tick()
	}
	if shield.Disabled() {
		t.Fatalf("expected shield to recover after 100 ticks")
	}
}

func TestTrident_RiptideAndLoyalty(t *testing.T) {
	trident := combat.Trident{
		LoyaltyLevel:    3,
		RiptideLevel:    2,
		ChannelingLevel: 1,
	}

	// Riptide conditions
	if !trident.CanRiptide(true, false, false) {
		t.Fatalf("expected Riptide allowed while in water")
	}
	if !trident.CanRiptide(false, true, true) {
		t.Fatalf("expected Riptide allowed while raining under open sky")
	}
	if trident.CanRiptide(false, true, false) {
		t.Fatalf("expected Riptide rejected while raining under solid roof")
	}

	// Riptide launch impulse
	impulse := trident.RiptideImpulse(0, 0) // Looking straight ahead
	expectedPower := 1.5 + 0.75*2.0         // 3.0
	if math.Abs(impulse.Len()-expectedPower) > 0.01 {
		t.Fatalf("expected impulse power %f, got %f", expectedPower, impulse.Len())
	}

	// Loyalty return speed
	if math.Abs(trident.LoyaltyReturnSpeed()-1.8) > 0.001 {
		t.Fatalf("expected loyalty speed 1.8 for level 3, got %f", trident.LoyaltyReturnSpeed())
	}

	// Channeling lightning condition
	if !trident.CanChannelLightning(true, true) {
		t.Fatalf("expected Channeling active in thunderstorm under sky")
	}
	if trident.CanChannelLightning(false, true) {
		t.Fatalf("expected Channeling inactive outside thunderstorm")
	}
}

func TestMace_SmashAttackScalingAndBounce(t *testing.T) {
	mace := combat.NewMace()

	// Normal ground hit (fall distance <= 1.5)
	if mace.IsSmashAttack(1.0) {
		t.Fatalf("fall distance <= 1.5 should not trigger smash attack")
	}
	if mace.CalculateSmashDamage(1.0) != 6.0 {
		t.Fatalf("expected base 6.0 damage, got %f", mace.CalculateSmashDamage(1.0))
	}

	// Smash attack: 5.5 blocks fall distance = 4 blocks fallen past 1.5
	// Damage = 6.0 (base) + 3 * 4.0 (Tier 1) + 1 * 2.0 (Tier 2) = 20.0
	if !mace.IsSmashAttack(5.5) {
		t.Fatalf("expected smash attack triggered at 5.5 blocks fall")
	}
	damage4 := mace.CalculateSmashDamage(5.5)
	if damage4 != 20.0 {
		t.Fatalf("expected 20.0 damage for 4 blocks fallen, got %f", damage4)
	}

	// Smash attack: 11.5 blocks fall distance = 10 blocks fallen past 1.5
	// Damage = 6.0 + 3*4.0 + 5*2.0 + 2*1.0 = 30.0
	damage10 := mace.CalculateSmashDamage(11.5)
	if damage10 != 30.0 {
		t.Fatalf("expected 30.0 damage for 10 blocks fallen, got %f", damage10)
	}

	// Density IV enchantment (+0.5 per block fallen per level = +2.0 per block)
	densityMace := combat.Mace{DensityLevel: 4}
	densityDamage := densityMace.CalculateSmashDamage(11.5)
	// Base 30.0 + (10 * 0.5 * 4 = 20.0) = 50.0
	if densityDamage != 50.0 {
		t.Fatalf("expected 50.0 damage with Density IV, got %f", densityDamage)
	}

	// Wind Burst bounce
	if mace.WindBurstBounceVelocity() != 0.8 {
		t.Fatalf("expected standard bounce 0.8, got %f", mace.WindBurstBounceVelocity())
	}
	windBurstMace := combat.Mace{WindBurstLevel: 3}
	if windBurstMace.WindBurstBounceVelocity() != 2.0 {
		t.Fatalf("expected Wind Burst III bounce 2.0, got %f", windBurstMace.WindBurstBounceVelocity())
	}
}

func TestCombat_NilAndNaNSafety(t *testing.T) {
	// 1. Mace NaN and Inf safety
	mace := combat.Mace{DensityLevel: 300, WindBurstLevel: 20}
	if mace.IsSmashAttack(math.NaN()) || mace.IsSmashAttack(math.Inf(1)) {
		t.Fatalf("expected IsSmashAttack to return false for NaN/Inf fall distance")
	}
	nanDmg := mace.CalculateSmashDamage(math.NaN())
	if nanDmg != 6.0 {
		t.Fatalf("expected base damage 6.0 for NaN fall distance, got %f", nanDmg)
	}
	// Density clamped to 255, WindBurst clamped to 10
	if mace.WindBurstBounceVelocity() != 4.8 {
		t.Fatalf("expected clamped wind burst bounce 4.8, got %f", mace.WindBurstBounceVelocity())
	}

	// 2. Shield NaN & Arc safety
	shield := combat.NewShield()
	shield.Disable(-100) // negative duration ignored
	if shield.Disabled() {
		t.Fatalf("expected shield not disabled by negative duration")
	}

	nanAttacker := mgl64.Vec3{math.NaN(), 0, 1}
	inArc := combat.InBlockingArc(nanAttacker, mgl64.Vec3{0, 0, 0}, 0)
	if inArc {
		t.Fatalf("expected InBlockingArc to return false on NaN coordinates")
	}

	b, rem, dis := shield.BlockDamage(math.NaN(), entity.AttackDamageSource{}, mgl64.Vec3{0, 0, 1}, mgl64.Vec3{0, 0, 0}, 0, false)
	if b != 0 || rem != 0 || dis {
		t.Fatalf("expected 0 damage on NaN block damage call")
	}

	// 3. Trident NaN safety
	trident := combat.Trident{RiptideLevel: 20, LoyaltyLevel: 20, ChannelingLevel: 20}
	nanImpulse := trident.RiptideImpulse(math.NaN(), 0)
	if nanImpulse.Len() != 0 {
		t.Fatalf("expected zero impulse on NaN yaw/pitch")
	}
	if trident.LoyaltyReturnSpeed() != 6.0 { // clamped to level 10 -> 0.6 * 10 = 6.0
		t.Fatalf("expected loyalty return speed 6.0 (clamped), got %f", trident.LoyaltyReturnSpeed())
	}
}

func BenchmarkMace_SmashCalculation(b *testing.B) {
	mace := combat.Mace{DensityLevel: 5, WindBurstLevel: 3}
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = mace.CalculateSmashDamage(15.5)
		_ = mace.WindBurstBounceVelocity()
	}
}

func BenchmarkShield_BlockingArcEvaluation(b *testing.B) {
	attacker := mgl64.Vec3{2, 0, 3}
	defender := mgl64.Vec3{0, 0, 0}
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = combat.InBlockingArc(attacker, defender, 45.0)
	}
}
