package combat

import (
	"math"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/go-gl/mathgl/mgl64"
)

// Trident represents a versatile oceanic combat tool capable of melee strikes,
// ranged impaling, Loyalty return flight, Riptide jet propulsion, and Channeling lightning strikes.
type Trident struct {
	LoyaltyLevel    int
	RiptideLevel    int
	ChannelingLevel int
}

// NewTrident initializes a standard unenchanted Trident.
func NewTrident() Trident {
	return Trident{}
}

// AttackDamage returns 9.0 base melee attack power.
func (Trident) AttackDamage() float64 {
	return 9.0
}

// MaxCount returns 1.
func (Trident) MaxCount() int {
	return 1
}

// DurabilityInfo returns trident durability configuration (250 uses).
func (Trident) DurabilityInfo() item.DurabilityInfo {
	return item.DurabilityInfo{
		MaxDurability: 250,
		BrokenItem: func() item.Stack {
			return item.Stack{}
		},
		AttackDurability: 1,
		BreakDurability:  2,
	}
}

// CanRiptide checks whether environmental conditions (submerged in water or exposed to rain) allow Riptide propulsion.
func (t Trident) CanRiptide(inWater bool, raining bool, skyAccess bool) bool {
	if t.RiptideLevel <= 0 {
		return false
	}
	return inWater || (raining && skyAccess)
}

// RiptideImpulse calculates the physical launch velocity imparted on the thrower.
func (t Trident) RiptideImpulse(yaw, pitch float64) mgl64.Vec3 {
	riptide := t.RiptideLevel
	if riptide <= 0 {
		return mgl64.Vec3{}
	}
	if riptide > 10 {
		riptide = 10
	}
	if math.IsNaN(yaw) || math.IsInf(yaw, 0) || math.IsNaN(pitch) || math.IsInf(pitch, 0) {
		return mgl64.Vec3{}
	}

	yawRad := -yaw * math.Pi / 180.0
	pitchRad := -pitch * math.Pi / 180.0
	cosPitch := math.Cos(pitchRad)

	lookDir := mgl64.Vec3{
		-math.Sin(yawRad) * cosPitch,
		math.Sin(pitchRad),
		math.Cos(yawRad) * cosPitch,
	}

	power := 1.5 + 0.75*float64(riptide)
	return lookDir.Mul(power)
}

// LoyaltyReturnSpeed returns the velocity at which a thrown trident flies back to its thrower.
func (t Trident) LoyaltyReturnSpeed() float64 {
	loyalty := t.LoyaltyLevel
	if loyalty <= 0 {
		return 0
	}
	if loyalty > 10 {
		loyalty = 10
	}
	return 0.6 * float64(loyalty)
}

// CanChannelLightning checks whether Channeling summons lightning upon impact during a thunderstorm.
func (t Trident) CanChannelLightning(thundering bool, skyAccess bool) bool {
	if t.ChannelingLevel <= 0 {
		return false
	}
	return thundering && skyAccess
}

// EncodeItem returns the Minecraft Bedrock identifier.
func (Trident) EncodeItem() (name string, meta int16) {
	return "minecraft:trident", 0
}
