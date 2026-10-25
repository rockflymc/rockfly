package entity

import (
	"image/color"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestLivingEntity_HurtAndImmunity(t *testing.T) {
	e := NewLivingEntity(nil, 20.0)

	// Hit 1: 4.0 damage (vulnerable, takes 4.0)
	dealt, vul := e.Hurt(4.0, AttackDamageSource{})
	if !vul || dealt != 4.0 || e.Health() != 16.0 {
		t.Fatalf("expected 4.0 dealt, 16.0 health, got dealt=%v, health=%v", dealt, e.Health())
	}

	// Hit 2: 2.0 damage inside immunity frame (invulnerable, 0 dealt)
	dealt, vul = e.Hurt(2.0, AttackDamageSource{})
	if vul || dealt != 0 || e.Health() != 16.0 {
		t.Fatalf("expected 0 dealt during immunity frame, got dealt=%v, vul=%v", dealt, vul)
	}

	// Hit 3: 6.0 damage inside immunity frame (vulnerable to overwrite, delta = 6.0 - 4.0 = 2.0 dealt)
	dealt, vul = e.Hurt(6.0, AttackDamageSource{})
	if !vul || dealt != 2.0 || e.Health() != 14.0 {
		t.Fatalf("expected 2.0 delta dealt, 14.0 health, got dealt=%v, health=%v", dealt, e.Health())
	}
}

func TestLivingEntity_Absorption(t *testing.T) {
	e := NewLivingEntity(nil, 20.0)
	e.SetAbsorption(4.0)

	// Hit 1: 3.0 damage absorbs entirely from absorption
	dealt, vul := e.Hurt(3.0, AttackDamageSource{})
	if !vul || dealt != 0 || e.Health() != 20.0 || e.Absorption() != 1.0 {
		t.Fatalf("expected 0 health dealt, 1.0 absorption remaining, got dealt=%v, hp=%v, abs=%v", dealt, e.Health(), e.Absorption())
	}

	// Reset immunity window to simulate next hit outside window
	e.immunity.Reset()

	// Hit 2: 3.0 damage absorbs 1.0 from absorption, inflicts 2.0 to health
	dealt, vul = e.Hurt(3.0, AttackDamageSource{})
	if !vul || dealt != 2.0 || e.Health() != 18.0 || e.Absorption() != 0 {
		t.Fatalf("expected 2.0 health dealt, 0 absorption, got dealt=%v, hp=%v, abs=%v", dealt, e.Health(), e.Absorption())
	}
}

func TestLivingEntity_KnockBack(t *testing.T) {
	e := NewLivingEntity(nil, 20.0)
	e.SetPosition(mgl64.Vec3{10, 0, 10})

	// Attacker at (10, 0, 8), so knockback should push in +Z direction
	attackerPos := mgl64.Vec3{10, 0, 8}
	e.KnockBack(attackerPos, 0.4, 0.3)

	vel := e.Velocity()
	if vel[2] <= 0 || vel[1] != 0.3 {
		t.Fatalf("expected positive Z velocity and 0.3 Y velocity, got %v", vel)
	}
}

func TestLivingEntity_ArmorMitigation(t *testing.T) {
	e := NewLivingEntity(nil, 20.0)
	e.Armour().SetChestplate(item.NewStack(item.Chestplate{Tier: item.ArmourTierDiamond{}}, 1))

	// Attack damage is reduced by armor
	dealt, vul := e.Hurt(10.0, AttackDamageSource{})
	if !vul || dealt >= 10.0 {
		t.Fatalf("expected armor to reduce damage below 10.0, got dealt=%v", dealt)
	}
}

func TestLivingEntity_Effects(t *testing.T) {
	e := NewLivingEntity(nil, 20.0)

	// Add Resistance I (level 1 = 20% reduction)
	e.AddEffect(effect.New(effect.Resistance, 1, 10*time.Second))

	dealt, vul := e.Hurt(10.0, AttackDamageSource{})
	expected := 10.0 * 0.80 // 8.0 damage
	if !vul || dealt < expected-0.01 || dealt > expected+0.01 {
		t.Fatalf("expected 8.0 damage with Resistance I, got %v", dealt)
	}

	// Add Fire Resistance
	e.AddEffect(effect.New(effect.FireResistance, 1, 10*time.Second))
	e.immunity.Reset()

	_, vulFire := e.Hurt(5.0, AttackDamageSource{})
	if !vulFire {
		t.Fatalf("attack damage should still hurt")
	}

	// Now with void damage (ignores totem and armor)
	e.immunity.Reset()
	dealtVoid, vulVoid := e.Hurt(5.0, VoidDamageSource{})
	if !vulVoid || dealtVoid != 5.0 {
		t.Fatalf("expected void damage to bypass resistance, got %v", dealtVoid)
	}
}

func TestLivingEntity_BedrockMetadataSync(t *testing.T) {
	e := NewLivingEntity(nil, 20.0)
	e.SetNameTag("Zombie")
	e.SetAlwaysShowNameTag(true)
	e.SetOnFire(5 * time.Second)
	e.SetAirSupply(10 * time.Second)

	builder := NewMetadataBuilder().
		SetDimensions(0.6, 1.8).
		SetScale(1.0).
		SetEffectParticles(color.RGBA{R: 255, G: 0, B: 0, A: 255}, false).
		SyncLiving(e)

	meta := builder.Build()

	if meta[protocol.EntityDataKeyName] != "Zombie" {
		t.Fatalf("expected name tag Zombie in metadata, got %v", meta[protocol.EntityDataKeyName])
	}
	if meta[protocol.EntityDataKeyAlwaysShowNameTag] != byte(1) {
		t.Fatalf("expected alwaysShowNameTag=1, got %v", meta[protocol.EntityDataKeyAlwaysShowNameTag])
	}
	if meta[protocol.EntityDataKeyWidth] != float32(0.6) {
		t.Fatalf("expected width 0.6, got %v", meta[protocol.EntityDataKeyWidth])
	}
}

func TestLivingEntity_Snapshot(t *testing.T) {
	e := NewLivingEntity(nil, 20.0)
	e.SetPosition(mgl64.Vec3{10, 64, 10})
	e.SetFallDistance(0)

	snap := e.Snapshot()
	if snap.Pos != (mgl64.Vec3{10, 64, 10}) {
		t.Fatalf("expected snapshot pos (10, 64, 10), got %v", snap.Pos)
	}
	if snap.Health != 20.0 || snap.MaxHealth != 20.0 {
		t.Fatalf("expected snapshot hp 20.0/20.0, got %v/%v", snap.Health, snap.MaxHealth)
	}
	if !snap.OnGround {
		t.Fatalf("expected snapshot OnGround=true with fall distance 0")
	}

	e.SetFallDistance(5.0)
	snap2 := e.Snapshot()
	if snap2.OnGround {
		t.Fatalf("expected snapshot OnGround=false with fall distance > 0")
	}
}

func TestLivingEntity_NaNAndNilSafety(t *testing.T) {
	e := NewLivingEntity(nil, 20.0)

	// NaN damage should be ignored and not corrupt state
	dealt, vul := e.Hurt(1e300*1e300-1e300*1e300, AttackDamageSource{}) // NaN
	if vul || dealt != 0 || e.Health() != 20.0 {
		t.Fatalf("expected NaN damage to be rejected, got dealt=%v, hp=%v", dealt, e.Health())
	}

	// Negative damage should be rejected
	dealt, vul = e.Hurt(-5.0, AttackDamageSource{})
	if vul || dealt != 0 {
		t.Fatalf("expected negative damage to be rejected")
	}

	// Nil entity method calls should not panic
	var nilEnt *LivingEntity
	if nilEnt.Health() != 0 || nilEnt.MaxHealth() != 0 || !nilEnt.Dead() {
		t.Fatalf("nil entity should report 0 health and dead=true")
	}
	nilSnap := nilEnt.Snapshot()
	if nilSnap.Health != 0 {
		t.Fatalf("nil entity snapshot should have 0 health")
	}
}

func BenchmarkLivingEntity_Hurt(b *testing.B) {
	e := NewLivingEntity(nil, 1000000.0)
	src := AttackDamageSource{}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e.Hurt(1.0, src)
	}
}
