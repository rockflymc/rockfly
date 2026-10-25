package boss_test

import (
	"math"
	"testing"

	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/boss"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestBossBar_SyncAndPackets(t *testing.T) {
	bar := boss.NewBossBar(42, "Ancient Guardian", packet.BossEventColourGreen, 0)

	if bar.Title() != "Ancient Guardian" {
		t.Fatalf("expected title Ancient Guardian, got %s", bar.Title())
	}
	if bar.HealthPercentage() != 1.0 {
		t.Fatalf("expected initial health 1.0, got %f", bar.HealthPercentage())
	}

	bar.SetHealthPercentage(0.65)
	if bar.HealthPercentage() != 0.65 {
		t.Fatalf("expected health 0.65, got %f", bar.HealthPercentage())
	}

	// Verify Show packet
	showPk := bar.ShowPacket()
	if showPk.EventType != packet.BossEventShow || showPk.HealthPercentage != 0.65 {
		t.Fatalf("invalid ShowPacket: %v", showPk)
	}

	// Verify Update packet
	updPk := bar.UpdateHealthPacket()
	if updPk.EventType != packet.BossEventHealthPercentage || updPk.HealthPercentage != 0.65 {
		t.Fatalf("invalid UpdateHealthPacket: %v", updPk)
	}

	// Verify Hide packet
	hidePk := bar.HidePacket()
	if hidePk.EventType != packet.BossEventHide {
		t.Fatalf("invalid HidePacket: %v", hidePk)
	}
}

func TestEnderDragon_PhasesAndCrystalHealing(t *testing.T) {
	dragon := boss.NewEnderDragon(nil)
	dragon.SetPosition(mgl64.Vec3{0, 75, 0})

	if dragon.Phase() != boss.DragonPhaseCircling {
		t.Fatalf("expected initial phase Circling, got %v", dragon.Phase())
	}
	if dragon.Health() != 200.0 {
		t.Fatalf("expected max health 200.0, got %f", dragon.Health())
	}

	// Test phase transition
	dragon.SetPhase(boss.DragonPhasePerching)
	if dragon.Phase() != boss.DragonPhasePerching {
		t.Fatalf("expected phase Perching")
	}

	// Damage dragon to 150 HP
	dragon.Hurt(50.0, entity.AttackDamageSource{})
	if dragon.Health() != 150.0 {
		t.Fatalf("expected 150 health after damage, got %f", dragon.Health())
	}

	// Tether crystal for healing
	crystalPos := mgl64.Vec3{20, 85, 20}
	dragon.ConnectCrystal(&crystalPos)
	if dragon.ConnectedCrystal() == nil {
		t.Fatalf("expected connected crystal")
	}

	// Tick dragon: should regenerate +1 HP
	dragon.Tick(nil, 1)
	if dragon.Health() != 151.0 {
		t.Fatalf("expected 151 health after crystal heal tick, got %f", dragon.Health())
	}

	// Reset immunity window to test sever crystal explosion damage
	dragon.ResetImmunity()
	currHealth := dragon.Health()

	// Sever crystal with explosion damage (-10 HP)
	dragon.SeverCrystal(true)
	if dragon.ConnectedCrystal() != nil {
		t.Fatalf("expected crystal severed")
	}
	expectedHealth := currHealth - 10.0
	if dragon.Health() != expectedHealth {
		t.Fatalf("expected %f health after crystal explosion sever, got %f", expectedHealth, dragon.Health())
	}

	// Reset immunity window again for fatal blow
	dragon.ResetImmunity()

	// Fatal blow triggers death sequence
	dragon.Hurt(300.0, entity.AttackDamageSource{})
	if dragon.Phase() != boss.DragonPhaseDying {
		t.Fatalf("expected phase Dying upon fatal damage, got %v", dragon.Phase())
	}
	if dragon.DeathTicks() != 200 {
		t.Fatalf("expected 200 death ticks, got %d", dragon.DeathTicks())
	}
}

func TestWither_ArmorAndProjectileImmunity(t *testing.T) {
	wither := boss.NewWither(nil)
	wither.SetPosition(mgl64.Vec3{0, 50, 0})

	// Initial invulnerability
	if wither.InvulnerableTicks() != 220 {
		t.Fatalf("expected initial 220 invulnerability ticks, got %d", wither.InvulnerableTicks())
	}
	// Damage while invulnerable does nothing
	dmg, vul := wither.Hurt(20.0, entity.AttackDamageSource{})
	if dmg != 0 || vul {
		t.Fatalf("wither should be invulnerable during spawn chargeup")
	}

	// Clear invulnerability
	wither.SetInvulnerableTicks(0)

	// Inflict damage down to 140 HP (<= 50% of 300)
	wither.Hurt(160.0, entity.AttackDamageSource{})
	if !wither.WitherArmor() {
		t.Fatalf("expected Wither Armor active at <= 50%% health, health is %f", wither.Health())
	}

	// Verify Wither Armor metadata flag
	meta := wither.Metadata()
	flags := meta[protocol.EntityDataKeyFlags].(int64)
	if flags&(1<<protocol.EntityDataFlagPowered) == 0 {
		t.Fatalf("expected EntityDataFlagPowered on Wither with active armor")
	}

	// Projectile attack while Wither Armor is active is completely deflected
	projDmg, projVul := wither.Hurt(15.0, entity.ProjectileDamageSource{})
	if projDmg != 0 || projVul {
		t.Fatalf("Wither Armor should completely deflect projectile damage, got dmg=%f, vul=%v", projDmg, projVul)
	}

	// Verify Nether Star drops
	drops := wither.Drops()
	if len(drops) != 1 {
		t.Fatalf("expected 1 drop, got %d", len(drops))
	}
	if _, ok := drops[0].Item().(item.NetherStar); !ok {
		t.Fatalf("expected Nether Star drop, got %v", drops[0])
	}
}

func TestBoss_NilAndNaNSafety(t *testing.T) {
	// 1. BossBar nil receiver safety
	var nilBar *boss.BossBar
	if nilBar.Title() != "" {
		t.Fatalf("expected empty title on nil BossBar")
	}
	nilBar.SetTitle("Dragon")
	if nilBar.HealthPercentage() != 0 {
		t.Fatalf("expected 0 health percentage on nil BossBar")
	}
	nilBar.SetHealthPercentage(0.5)
	if nilBar.Colour() != 0 {
		t.Fatalf("expected 0 colour on nil BossBar")
	}
	nilBar.SetColour(1)
	if nilBar.ShowPacket() != nil || nilBar.UpdateHealthPacket() != nil || nilBar.HidePacket() != nil {
		t.Fatalf("expected nil packets on nil BossBar")
	}

	// 2. BossBar NaN and Inf clamp
	bar := boss.NewBossBar(1, "Test", 0, 0)
	bar.SetHealthPercentage(float32(math.NaN()))
	if bar.HealthPercentage() != 0 {
		t.Fatalf("expected NaN health percentage to clamp to 0")
	}
	bar.SetHealthPercentage(float32(math.Inf(1)))
	if bar.HealthPercentage() != 0 {
		t.Fatalf("expected +Inf health percentage to clamp to 0")
	}
	bar.SetHealthPercentage(1.5)
	if bar.HealthPercentage() != 1.0 {
		t.Fatalf("expected >1.0 health percentage to clamp to 1.0")
	}
	bar.SetHealthPercentage(-0.5)
	if bar.HealthPercentage() != 0.0 {
		t.Fatalf("expected <0 health percentage to clamp to 0.0")
	}

	// 3. EnderDragon nil receiver safety
	var nilDragon *boss.EnderDragon
	if nilDragon.Phase() != boss.DragonPhaseCircling {
		t.Fatalf("expected default phase Circling on nil dragon")
	}
	nilDragon.SetPhase(boss.DragonPhasePerching)
	if nilDragon.ConnectedCrystal() != nil {
		t.Fatalf("expected nil connected crystal on nil dragon")
	}
	nilDragon.ConnectCrystal(&mgl64.Vec3{0, 0, 0})
	nilDragon.SeverCrystal(true)
	if nilDragon.DeathTicks() != 0 {
		t.Fatalf("expected 0 death ticks on nil dragon")
	}
	nilDragon.Tick(nil, 1)

	// 4. Wither nil receiver safety
	var nilWither *boss.Wither
	if nilWither.InvulnerableTicks() != 0 {
		t.Fatalf("expected 0 invulnerable ticks on nil wither")
	}
	nilWither.SetInvulnerableTicks(10)
	if nilWither.WitherArmor() {
		t.Fatalf("expected false wither armor on nil wither")
	}
	nilWither.Tick(nil, 1)
	if nilWither.Metadata() == nil {
		t.Fatalf("expected non-nil metadata map on nil wither")
	}
}

func BenchmarkEnderDragon_Tick(b *testing.B) {
	dragon := boss.NewEnderDragon(nil)
	dragon.SetPosition(mgl64.Vec3{0, 75, 0})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		dragon.Tick(nil, int64(i))
	}
}
