package entity

import (
	"image/color"
	"time"

	"github.com/df-mc/dragonfly/server/internal/nbtconv"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// MetadataBuilder simplifies the construction and mutation of Bedrock EntityMetadata.
type MetadataBuilder struct {
	m protocol.EntityMetadata
}

// NewMetadataBuilder initializes an empty MetadataBuilder with default entity flags.
func NewMetadataBuilder() *MetadataBuilder {
	m := protocol.NewEntityMetadata()
	m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagHasGravity)
	m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagClimb)
	m[protocol.EntityDataKeyEffectColor] = int32(0)
	m[protocol.EntityDataKeyEffectAmbience] = byte(0)
	m[protocol.EntityDataKeyColorIndex] = byte(0)
	return &MetadataBuilder{m: m}
}

// Build returns the underlying protocol.EntityMetadata.
func (b *MetadataBuilder) Build() protocol.EntityMetadata {
	return b.m
}

// SetDimensions sets entity bounding box width and height.
func (b *MetadataBuilder) SetDimensions(width, height float64) *MetadataBuilder {
	b.m[protocol.EntityDataKeyWidth] = float32(width)
	b.m[protocol.EntityDataKeyHeight] = float32(height)
	return b
}

// SetScale sets the visual render scale factor of the entity.
func (b *MetadataBuilder) SetScale(scale float64) *MetadataBuilder {
	b.m[protocol.EntityDataKeyScale] = float32(scale)
	return b
}

// SetNameTag configures the visible name tag of the entity.
func (b *MetadataBuilder) SetNameTag(name string, alwaysShow bool) *MetadataBuilder {
	b.m[protocol.EntityDataKeyName] = name
	if len(name) > 0 {
		b.m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagShowName)
	} else {
		b.m.UnsetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagShowName)
	}
	if alwaysShow {
		b.m[protocol.EntityDataKeyAlwaysShowNameTag] = byte(1)
		b.m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagAlwaysShowName)
	} else {
		b.m[protocol.EntityDataKeyAlwaysShowNameTag] = byte(0)
		b.m.UnsetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagAlwaysShowName)
	}
	return b
}

// SetAirSupply configures the oxygen level for underwater display.
func (b *MetadataBuilder) SetAirSupply(air, maxAir time.Duration, breathing bool) *MetadataBuilder {
	b.m[protocol.EntityDataKeyAirSupply] = int16(air.Milliseconds() / 50)
	b.m[protocol.EntityDataKeyAirSupplyMax] = int16(maxAir.Milliseconds() / 50)
	if breathing {
		b.m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagBreathing)
	} else {
		b.m.UnsetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagBreathing)
	}
	return b
}

// SetOnFire sets or unsets the on-fire visual flag.
func (b *MetadataBuilder) SetOnFire(onFire bool) *MetadataBuilder {
	if onFire {
		b.m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagOnFire)
	} else {
		b.m.UnsetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagOnFire)
	}
	return b
}

// SetInvisible sets or unsets invisibility.
func (b *MetadataBuilder) SetInvisible(invisible bool) *MetadataBuilder {
	if invisible {
		b.m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagInvisible)
	} else {
		b.m.UnsetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagInvisible)
	}
	return b
}

// SetBaby sets or unsets the baby visual state.
func (b *MetadataBuilder) SetBaby(baby bool) *MetadataBuilder {
	if baby {
		b.m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagBaby)
	} else {
		b.m.UnsetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagBaby)
	}
	return b
}

// SetNoAI sets or unsets the NoAI (immobile) flag.
func (b *MetadataBuilder) SetNoAI(noAI bool) *MetadataBuilder {
	if noAI {
		b.m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagNoAI)
	} else {
		b.m.UnsetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagNoAI)
	}
	return b
}

// SetSwimming sets or unsets the swimming flag.
func (b *MetadataBuilder) SetSwimming(swimming bool) *MetadataBuilder {
	if swimming {
		b.m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagSwimming)
	} else {
		b.m.UnsetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagSwimming)
	}
	return b
}

// SetGliding sets or unsets the gliding (elytra) flag.
func (b *MetadataBuilder) SetGliding(gliding bool) *MetadataBuilder {
	if gliding {
		b.m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagGliding)
	} else {
		b.m.UnsetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagGliding)
	}
	return b
}

// SetSneaking sets or unsets sneaking.
func (b *MetadataBuilder) SetSneaking(sneaking bool) *MetadataBuilder {
	if sneaking {
		b.m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagSneaking)
	} else {
		b.m.UnsetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagSneaking)
	}
	return b
}

// SetSprinting sets or unsets sprinting.
func (b *MetadataBuilder) SetSprinting(sprinting bool) *MetadataBuilder {
	if sprinting {
		b.m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagSprinting)
	} else {
		b.m.UnsetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagSprinting)
	}
	return b
}

// SetEffectParticles sets the potion particle color and ambience.
func (b *MetadataBuilder) SetEffectParticles(c color.RGBA, ambient bool) *MetadataBuilder {
	b.m[protocol.EntityDataKeyEffectColor] = nbtconv.Int32FromRGBA(c)
	if ambient {
		b.m[protocol.EntityDataKeyEffectAmbience] = byte(1)
	} else {
		b.m[protocol.EntityDataKeyEffectAmbience] = byte(0)
	}
	return b
}

// SyncLiving populates metadata with the living entity's attributes and component states.
func (b *MetadataBuilder) SyncLiving(l *LivingEntity) *MetadataBuilder {
	b.SetNameTag(l.NameTag(), l.AlwaysShowNameTag())
	b.SetOnFire(l.OnFireDuration() > 0)
	b.SetAirSupply(l.AirSupply(), l.MaxAirSupply(), l.AirSupply() > 0)
	return b
}
