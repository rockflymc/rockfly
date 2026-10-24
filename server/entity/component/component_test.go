package component

import (
	"sync"
	"testing"
	"time"
)

func TestHealth(t *testing.T) {
	h := NewHealth(20, 20)
	if h.Health() != 20 || h.MaxHealth() != 20 {
		t.Fatalf("expected 20/20 health, got %v/%v", h.Health(), h.MaxHealth())
	}
	if h.Dead() {
		t.Fatalf("expected entity to be alive")
	}

	// Add damage
	delta := h.AddHealth(-5)
	if delta != -5 || h.Health() != 15 {
		t.Fatalf("expected 15 health, got %v (delta %v)", h.Health(), delta)
	}

	// Add heal exceeding max
	delta = h.AddHealth(10)
	if delta != 5 || h.Health() != 20 {
		t.Fatalf("expected health clamped to 20, got %v (delta %v)", h.Health(), delta)
	}

	// Fatal damage
	h.AddHealth(-30)
	if !h.Dead() || h.Health() != 0 {
		t.Fatalf("expected dead entity at 0 health, got %v", h.Health())
	}

	// Concurrent access test
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.AddHealth(1)
			h.AddHealth(-1)
		}()
	}
	wg.Wait()
}

func TestAbsorption(t *testing.T) {
	a := NewAbsorption(4.0)
	if a.Absorption() != 4.0 {
		t.Fatalf("expected 4.0 absorption, got %v", a.Absorption())
	}

	// Partial absorption
	rem, absorbed := a.Absorb(2.5)
	if rem != 0 || absorbed != 2.5 || a.Absorption() != 1.5 {
		t.Fatalf("expected rem=0, absorbed=2.5, got rem=%v, absorbed=%v, left=%v", rem, absorbed, a.Absorption())
	}

	// Exhaust absorption with overflow
	rem, absorbed = a.Absorb(3.0)
	if rem != 1.5 || absorbed != 1.5 || a.Absorption() != 0 {
		t.Fatalf("expected rem=1.5, absorbed=1.5, got rem=%v, absorbed=%v, left=%v", rem, absorbed, a.Absorption())
	}
}

func TestDamageImmunity(t *testing.T) {
	imm := NewDamageImmunity()

	// 1. Initial hit deals full damage and establishes window
	eff, vul := imm.Evaluate(5.0, 500*time.Millisecond)
	if !vul || eff != 5.0 {
		t.Fatalf("expected initial hit vulnerable=true, eff=5.0, got %v, %v", vul, eff)
	}
	if !imm.Immune() {
		t.Fatalf("expected entity to be immune")
	}

	// 2. Weaker hit inside window is negated
	eff, vul = imm.Evaluate(3.0, 500*time.Millisecond)
	if vul || eff != 0 {
		t.Fatalf("expected weaker hit negated, got vul=%v, eff=%v", vul, eff)
	}

	// 3. Equal hit inside window is negated
	eff, vul = imm.Evaluate(5.0, 500*time.Millisecond)
	if vul || eff != 0 {
		t.Fatalf("expected equal hit negated, got vul=%v, eff=%v", vul, eff)
	}

	// 4. Stronger hit inside window deals delta (8.0 - 5.0 = 3.0)
	eff, vul = imm.Evaluate(8.0, 500*time.Millisecond)
	if !vul || eff != 3.0 {
		t.Fatalf("expected stronger hit delta=3.0, got vul=%v, eff=%v", vul, eff)
	}
	if imm.LastDamage() != 8.0 {
		t.Fatalf("expected last damage updated to 8.0, got %v", imm.LastDamage())
	}

	// 5. Reset clears immunity
	imm.Reset()
	if imm.Immune() {
		t.Fatalf("expected immunity cleared after reset")
	}
	eff, vul = imm.Evaluate(4.0, 500*time.Millisecond)
	if !vul || eff != 4.0 {
		t.Fatalf("expected full hit after reset, got vul=%v, eff=%v", vul, eff)
	}
}

func TestFallTracker(t *testing.T) {
	f := NewFallTracker()

	// Fall 10 blocks
	f.Accumulate(-4.0)
	f.Accumulate(-6.0)
	if f.FallDistance() != 10.0 {
		t.Fatalf("expected fall distance 10.0, got %v", f.FallDistance())
	}

	// Impact without feather falling: (10 - 3) = 7 damage
	dmg := f.Impact(0)
	if dmg != 7.0 {
		t.Fatalf("expected 7.0 fall damage, got %v", dmg)
	}
	if f.FallDistance() != 0 {
		t.Fatalf("expected fall distance reset after impact, got %v", f.FallDistance())
	}

	// Fall 10 blocks with Feather Falling IV (4 * 12% = 48% reduction): 7 * 0.52 = 3.64
	f.Accumulate(-10.0)
	dmgWithFF := f.Impact(4)
	expected := 7.0 * (1.0 - 0.48)
	if dmgWithFF < expected-0.001 || dmgWithFF > expected+0.001 {
		t.Fatalf("expected %v damage with Feather Falling IV, got %v", expected, dmgWithFF)
	}
}

func TestAirSupply(t *testing.T) {
	a := NewAirSupply(1 * time.Second) // 20 ticks
	if a.AirSupply() != 1*time.Second {
		t.Fatalf("expected 1s air supply, got %v", a.AirSupply())
	}

	// Tick submerged 19 ticks: no drowning
	for i := 0; i < 19; i++ {
		drowning, dmg := a.TickSubmerged()
		if drowning || dmg > 0 {
			t.Fatalf("expected no drowning yet at tick %v", i)
		}
	}

	// Tick 20: air runs out
	drowning, _ := a.TickSubmerged()
	if !drowning {
		t.Fatalf("expected drowning when air supply reaches 0")
	}

	// Next 20 ticks submerged: inflicts 1.0 damage
	var dealtDmg float64
	for i := 0; i < 20; i++ {
		_, dmg := a.TickSubmerged()
		dealtDmg += dmg
	}
	if dealtDmg != 1.0 {
		t.Fatalf("expected 1.0 drowning damage after 20 submerged ticks, got %v", dealtDmg)
	}

	// Replenish
	a.Replenish()
	if a.AirSupply() != 1*time.Second {
		t.Fatalf("expected replenished air supply, got %v", a.AirSupply())
	}
}

func TestFireTracker(t *testing.T) {
	f := NewFireTracker()
	f.SetOnFire(2 * time.Second) // 40 ticks
	if f.OnFireDuration() != 2*time.Second {
		t.Fatalf("expected 2s fire duration, got %v", f.OnFireDuration())
	}

	// Tick 20 times: deals 1.0 fire damage
	var totalDmg float64
	for i := 0; i < 20; i++ {
		burning, dmg := f.Tick()
		if !burning {
			t.Fatalf("expected entity to be burning at tick %v", i)
		}
		totalDmg += dmg
	}
	if totalDmg != 1.0 {
		t.Fatalf("expected 1.0 fire damage after 20 ticks, got %v", totalDmg)
	}

	// Extinguish
	f.Extinguish()
	if f.OnFireDuration() != 0 {
		t.Fatalf("expected 0 fire duration after extinguish, got %v", f.OnFireDuration())
	}
}
