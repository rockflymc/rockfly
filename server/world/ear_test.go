package world

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
)

type dummyEntityType struct {
	name string
}

func (d dummyEntityType) Open(tx *Tx, handle *EntityHandle, data *EntityData) Entity {
	return dummyEntity{handle: handle}
}
func (d dummyEntityType) EncodeEntity() string { return d.name }
func (d dummyEntityType) BBox(e Entity) cube.BBox {
	return cube.Box(-0.3, 0, -0.3, 0.3, 1.8, 0.3)
}
func (d dummyEntityType) DecodeNBT(m map[string]any, data *EntityData) {}
func (d dummyEntityType) EncodeNBT(data *EntityData) map[string]any    { return nil }

type dummyEntity struct {
	handle *EntityHandle
}

func (d dummyEntity) Close() error            { return nil }
func (d dummyEntity) H() *EntityHandle        { return d.handle }
func (d dummyEntity) Position() mgl64.Vec3    { return d.handle.data.Pos }
func (d dummyEntity) Rotation() cube.Rotation { return d.handle.data.Rot }

type dummyImmuneEntity struct {
	dummyEntity
	immune bool
}

func (d dummyImmuneEntity) ActivationImmune() bool {
	return d.immune
}

type dummyCategorizedEntity struct {
	dummyEntity
	cat ActivationCategory
}

func (d dummyCategorizedEntity) ActivationCategory() ActivationCategory {
	return d.cat
}

func TestEARTracker_Categories(t *testing.T) {
	ear := NewEARTracker(DefaultEARConfig())

	cowHandle := EntitySpawnOpts{}.New(dummyEntityType{name: "minecraft:cow"}, testEntityConfig{})
	cow := dummyEntity{handle: cowHandle}
	if cat := ear.categoryOf(cow, "minecraft:cow"); cat != ActivationAnimal {
		t.Fatalf("expected ActivationAnimal, got %v", cat)
	}

	zombieHandle := EntitySpawnOpts{}.New(dummyEntityType{name: "minecraft:zombie"}, testEntityConfig{})
	zombie := dummyEntity{handle: zombieHandle}
	if cat := ear.categoryOf(zombie, "minecraft:zombie"); cat != ActivationMonster {
		t.Fatalf("expected ActivationMonster, got %v", cat)
	}

	itemHandle := EntitySpawnOpts{}.New(dummyEntityType{name: "minecraft:item"}, testEntityConfig{})
	item := dummyEntity{handle: itemHandle}
	if cat := ear.categoryOf(item, "minecraft:item"); cat != ActivationMisc {
		t.Fatalf("expected ActivationMisc, got %v", cat)
	}

	playerHandle := EntitySpawnOpts{}.New(dummyEntityType{name: "minecraft:player"}, testEntityConfig{})
	player := dummyEntity{handle: playerHandle}
	if cat := ear.categoryOf(player, "minecraft:player"); cat != ActivationSpecial {
		t.Fatalf("expected ActivationSpecial, got %v", cat)
	}
}

func TestEARTracker_ProximityActivation(t *testing.T) {
	ear := NewEARTracker(DefaultEARConfig())
	playerPos := []mgl64.Vec3{{0, 64, 0}}

	// Cow within 10 blocks (active)
	closeCow := EntitySpawnOpts{Position: mgl64.Vec3{5, 64, 5}}.New(dummyEntityType{name: "minecraft:cow"}, testEntityConfig{})
	entClose := dummyEntity{handle: closeCow}
	shouldTick, isInactive := ear.ShouldTick(entClose, closeCow, playerPos, 1)
	if !shouldTick || isInactive {
		t.Fatalf("expected close cow to be fully active, got shouldTick=%v isInactive=%v", shouldTick, isInactive)
	}

	// Cow 60 blocks away (outside 32 block range)
	farCow := EntitySpawnOpts{Position: mgl64.Vec3{60, 64, 0}}.New(dummyEntityType{name: "minecraft:cow"}, testEntityConfig{})
	entFar := dummyEntity{handle: farCow}

	activeCount := 0
	for tick := int64(1); tick <= 20; tick++ {
		should, inact := ear.ShouldTick(entFar, farCow, playerPos, tick)
		if should {
			activeCount++
			if !inact {
				t.Fatalf("expected far cow tick to be marked inactive")
			}
		}
	}

	// In 20 ticks, an inactive cow should only tick exactly once
	if activeCount != 1 {
		t.Fatalf("expected inactive cow to tick exactly once in 20 ticks, got %d", activeCount)
	}
}

func TestEARTracker_Immunity(t *testing.T) {
	ear := NewEARTracker(DefaultEARConfig())
	playerPos := []mgl64.Vec3{{0, 64, 0}}

	// Burning far cow (outside 32 blocks)
	burningCow := EntitySpawnOpts{Position: mgl64.Vec3{100, 64, 0}}.New(dummyEntityType{name: "minecraft:cow"}, testEntityConfig{})
	burningCow.data.FireDuration = 5 * time.Second
	entBurning := dummyEntity{handle: burningCow}

	shouldTick, isInactive := ear.ShouldTick(entBurning, burningCow, playerPos, 5)
	if !shouldTick || isInactive {
		t.Fatalf("expected burning cow to remain fully active despite distance")
	}

	// Entity with dynamic immunity interface
	immuneHandle := EntitySpawnOpts{Position: mgl64.Vec3{100, 64, 0}}.New(dummyEntityType{name: "minecraft:zombie"}, testEntityConfig{})
	entImmune := dummyImmuneEntity{dummyEntity: dummyEntity{handle: immuneHandle}, immune: true}

	shouldTick, isInactive = ear.ShouldTick(entImmune, immuneHandle, playerPos, 5)
	if !shouldTick || isInactive {
		t.Fatalf("expected immune entity to remain fully active despite distance")
	}
}

func TestEARTracker_NoPlayers(t *testing.T) {
	ear := NewEARTracker(DefaultEARConfig())
	cowHandle := EntitySpawnOpts{Position: mgl64.Vec3{0, 64, 0}}.New(dummyEntityType{name: "minecraft:cow"}, testEntityConfig{})
	entCow := dummyEntity{handle: cowHandle}

	// No players present -> entity should be inactive
	ticks := 0
	for i := int64(1); i <= 20; i++ {
		should, _ := ear.ShouldTick(entCow, cowHandle, nil, i)
		if should {
			ticks++
		}
	}
	if ticks != 1 {
		t.Fatalf("expected 1 tick in 20 when no players are online, got %d", ticks)
	}
}

func BenchmarkEARTracker_ShouldTick(b *testing.B) {
	ear := NewEARTracker(DefaultEARConfig())
	playerPos := []mgl64.Vec3{
		{0, 64, 0},
		{100, 64, 100},
		{-50, 64, 200},
	}

	handles := make([]*EntityHandle, 1000)
	entities := make([]Entity, 1000)
	for i := 0; i < 1000; i++ {
		handles[i] = EntitySpawnOpts{
			ID:       uuid.New(),
			Position: mgl64.Vec3{float64(i % 150), 64, float64(i / 150 * 50)},
		}.New(dummyEntityType{name: "minecraft:cow"}, testEntityConfig{})
		entities[i] = dummyEntity{handle: handles[i]}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		idx := i % 1000
		ear.ShouldTick(entities[idx], handles[idx], playerPos, int64(i))
	}
}
