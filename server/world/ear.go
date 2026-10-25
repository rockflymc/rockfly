package world

import (
	"strings"

	"github.com/go-gl/mathgl/mgl64"
)

// ActivationCategory represents the category of an entity for Entity Activation Range (EAR).
type ActivationCategory uint8

const (
	// ActivationAnimal applies to passive/neutral animals, villagers, and farm mobs.
	ActivationAnimal ActivationCategory = iota
	// ActivationMonster applies to hostile monsters.
	ActivationMonster
	// ActivationMisc applies to ambient mobs, dropped items, and projectiles.
	ActivationMisc
	// ActivationSpecial applies to players, bosses, and always-active entities.
	ActivationSpecial
)

// EARConfig defines configuration parameters for Paper-style Entity Activation Range.
type EARConfig struct {
	// Configured indicates whether EARConfig has been explicitly populated.
	Configured bool
	// Enabled determines if Entity Activation Range is active.
	Enabled bool
	// AnimalRange is the activation radius in blocks for animals/passive mobs (default: 32).
	AnimalRange float64
	// MonsterRange is the activation radius in blocks for monsters/hostile mobs (default: 32).
	MonsterRange float64
	// MiscRange is the activation radius in blocks for miscellaneous entities (default: 16).
	MiscRange float64
	// InactiveTickRate specifies the interval in ticks at which inactive entities are ticked (default: 20).
	InactiveTickRate int64
}

// DefaultEARConfig returns standard Paper-style default EAR settings.
func DefaultEARConfig() EARConfig {
	return EARConfig{
		Configured:       true,
		Enabled:          true,
		AnimalRange:      32.0,
		MonsterRange:     32.0,
		MiscRange:        16.0,
		InactiveTickRate: 20,
	}
}

// CategorizedEntity is an optional interface that an entity can implement to explicitly
// declare its EAR activation category.
type CategorizedEntity interface {
	ActivationCategory() ActivationCategory
}

// ImmuneEntity is an optional interface that an entity can implement to declare dynamic
// immunity to EAR deactivation (e.g. while in combat, fleeing, or interacting).
type ImmuneEntity interface {
	ActivationImmune() bool
}

// InactiveTickerEntity is an optional interface for entities that perform lightweight
// updates (e.g. gravity, drowning, despawn timers) while inactive without executing full AI.
type InactiveTickerEntity interface {
	InactiveTick(tx *Tx, tick int64)
}

// EARTracker tracks and evaluates entity activation states for a World.
type EARTracker struct {
	conf EARConfig
}

// NewEARTracker creates a new EAR tracker with the given configuration.
func NewEARTracker(conf EARConfig) *EARTracker {
	if !conf.Configured {
		conf = DefaultEARConfig()
	}
	return &EARTracker{conf: conf}
}

// Config returns the active EAR configuration.
func (ear *EARTracker) Config() EARConfig {
	if ear == nil {
		return DefaultEARConfig()
	}
	return ear.conf
}

// ShouldTick determines if an entity should tick on the given world tick.
// Returns shouldTick (bool) and isInactiveTick (bool).
func (ear *EARTracker) ShouldTick(e Entity, handle *EntityHandle, players []mgl64.Vec3, currentTick int64) (shouldTick bool, isInactiveTick bool) {
	if ear == nil || !ear.conf.Enabled {
		return true, false
	}
	if handle == nil || handle.Type() == nil {
		return true, false
	}

	// Players and always-active entities always tick.
	typeStr := handle.Type().EncodeEntity()
	if typeStr == "minecraft:player" {
		return true, false
	}

	cat := ear.categoryOf(e, typeStr)
	if cat == ActivationSpecial {
		return true, false
	}

	// Immunity checks:
	// 1. Dynamic immunity interface
	if immune, ok := e.(ImmuneEntity); ok && immune.ActivationImmune() {
		return true, false
	}

	// 2. Burning entities remain active
	if handle.data.FireDuration > 0 {
		return true, false
	}

	// If no players are present, entity is inactive.
	if len(players) == 0 {
		return ear.evalInactive(handle, currentTick)
	}

	// Proximity check against all players
	rangeLimit := ear.rangeFor(cat)
	rangeSq := rangeLimit * rangeLimit
	pos := handle.data.Pos

	for _, pPos := range players {
		dx := pos[0] - pPos[0]
		dz := pos[2] - pPos[2]
		dy := pos[1] - pPos[1]
		distSq := dx*dx + dz*dz + dy*dy
		if distSq <= rangeSq {
			return true, false
		}
	}

	// Entity is outside activation range.
	return ear.evalInactive(handle, currentTick)
}

func (ear *EARTracker) evalInactive(handle *EntityHandle, currentTick int64) (shouldTick bool, isInactiveTick bool) {
	rate := ear.conf.InactiveTickRate
	if rate <= 0 {
		rate = 20
	}
	// Stagger inactive ticking across ticks using UUID salt to prevent GC/CPU spikes.
	salt := int64(handle.id[15])
	if (currentTick+salt)%rate == 0 {
		return true, true
	}
	return false, false
}

func (ear *EARTracker) rangeFor(cat ActivationCategory) float64 {
	switch cat {
	case ActivationAnimal:
		return ear.conf.AnimalRange
	case ActivationMonster:
		return ear.conf.MonsterRange
	case ActivationMisc:
		return ear.conf.MiscRange
	default:
		return 32.0
	}
}

func (ear *EARTracker) categoryOf(e Entity, typeStr string) ActivationCategory {
	if catEntity, ok := e.(CategorizedEntity); ok {
		return catEntity.ActivationCategory()
	}

	// Special entities
	if strings.Contains(typeStr, "dragon") || strings.Contains(typeStr, "wither") || strings.Contains(typeStr, "player") {
		return ActivationSpecial
	}

	// Monsters
	if isMonsterType(typeStr) {
		return ActivationMonster
	}

	// Animals
	if isAnimalType(typeStr) {
		return ActivationAnimal
	}

	return ActivationMisc
}

func isMonsterType(t string) bool {
	switch t {
	case "minecraft:zombie", "minecraft:skeleton", "minecraft:creeper", "minecraft:spider",
		"minecraft:cave_spider", "minecraft:enderman", "minecraft:witch", "minecraft:slime",
		"minecraft:magma_cube", "minecraft:blaze", "minecraft:ghast", "minecraft:warden",
		"minecraft:breeze", "minecraft:pillager", "minecraft:drowned", "minecraft:husk",
		"minecraft:stray", "minecraft:piglin", "minecraft:piglin_brute", "minecraft:zombified_piglin",
		"minecraft:phantom", "minecraft:shulker", "minecraft:silverfish", "minecraft:endermite",
		"minecraft:ravager", "minecraft:evoker", "minecraft:vindicator", "minecraft:vex":
		return true
	default:
		return false
	}
}

func isAnimalType(t string) bool {
	switch t {
	case "minecraft:cow", "minecraft:pig", "minecraft:sheep", "minecraft:chicken",
		"minecraft:horse", "minecraft:donkey", "minecraft:mule", "minecraft:llama",
		"minecraft:trader_llama", "minecraft:wolf", "minecraft:cat", "minecraft:ocelot",
		"minecraft:fox", "minecraft:rabbit", "minecraft:parrot", "minecraft:villager",
		"minecraft:wandering_trader", "minecraft:iron_golem", "minecraft:snow_golem",
		"minecraft:sniffer", "minecraft:armadillo", "minecraft:allay", "minecraft:axolotl",
		"minecraft:bee", "minecraft:camel", "minecraft:frog", "minecraft:goat",
		"minecraft:mooshroom", "minecraft:panda", "minecraft:polar_bear", "minecraft:strider",
		"minecraft:turtle":
		return true
	default:
		return false
	}
}
