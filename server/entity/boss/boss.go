package boss

import (
	"math"
	"sync"

	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// BossBar coordinates the network synchronization of boss health bars, titles, colors,
// and screen darkening for major boss encounters.
type BossBar struct {
	mu           sync.RWMutex
	bossID       int64
	title        string
	health       float32 // 0.0 to 1.0
	colour       uint8
	overlay      uint8
	darkenScreen bool
}

// NewBossBar initializes a new BossBar descriptor.
func NewBossBar(bossID int64, title string, colour, overlay uint8) *BossBar {
	return &BossBar{
		bossID:  bossID,
		title:   title,
		health:  1.0,
		colour:  colour,
		overlay: overlay,
	}
}

// Title returns the title displayed above the health bar.
func (b *BossBar) Title() string {
	if b == nil {
		return ""
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.title
}

// SetTitle updates the title text.
func (b *BossBar) SetTitle(title string) {
	if b == nil {
		return
	}
	b.mu.Lock()
	b.title = title
	b.mu.Unlock()
}

// HealthPercentage returns the current health ratio in [0.0, 1.0].
func (b *BossBar) HealthPercentage() float32 {
	if b == nil {
		return 0
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.health
}

// SetHealthPercentage updates the boss health ratio clamped between 0.0 and 1.0.
func (b *BossBar) SetHealthPercentage(h float32) {
	if b == nil {
		return
	}
	if math.IsNaN(float64(h)) || math.IsInf(float64(h), 0) || h < 0 {
		h = 0
	} else if h > 1 {
		h = 1
	}
	b.mu.Lock()
	b.health = h
	b.mu.Unlock()
}

// Colour returns the boss bar color constant.
func (b *BossBar) Colour() uint8 {
	if b == nil {
		return 0
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.colour
}

// SetColour updates the boss bar color.
func (b *BossBar) SetColour(colour uint8) {
	if b == nil {
		return
	}
	b.mu.Lock()
	b.colour = colour
	b.mu.Unlock()
}

// ShowPacket constructs the packet.BossEvent packet to display the bar on clients.
func (b *BossBar) ShowPacket() *packet.BossEvent {
	if b == nil {
		return nil
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	return &packet.BossEvent{
		BossEntityUniqueID: b.bossID,
		EventType:          packet.BossEventShow,
		BossBarTitle:       b.title,
		HealthPercentage:   b.health,
		Colour:             b.colour,
		Overlay:            b.overlay,
	}
}

// UpdateHealthPacket constructs an incremental update packet for health changes.
func (b *BossBar) UpdateHealthPacket() *packet.BossEvent {
	if b == nil {
		return nil
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	return &packet.BossEvent{
		BossEntityUniqueID: b.bossID,
		EventType:          packet.BossEventHealthPercentage,
		HealthPercentage:   b.health,
	}
}

// HidePacket constructs the removal packet to dismiss the bar from clients.
func (b *BossBar) HidePacket() *packet.BossEvent {
	if b == nil {
		return nil
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	return &packet.BossEvent{
		BossEntityUniqueID: b.bossID,
		EventType:          packet.BossEventHide,
	}
}

// Boss serves as the common base for formidable world encounters, pairing LivingEntity
// attributes with dedicated BossBar life bars.
type Boss struct {
	*entity.LivingEntity
	bar *BossBar
}

// NewBoss initializes a boss foundation.
func NewBoss(handle *world.EntityHandle, maxHealth float64, bar *BossBar) *Boss {
	return &Boss{
		LivingEntity: entity.NewLivingEntity(handle, maxHealth),
		bar:          bar,
	}
}

// Bar returns the boss's synchronized BossBar.
func (b *Boss) Bar() *BossBar {
	if b == nil {
		return nil
	}
	return b.bar
}

// SyncBarHealth synchronizes the current health of the LivingEntity into the BossBar.
func (b *Boss) SyncBarHealth() {
	if b == nil || b.LivingEntity == nil || b.bar == nil {
		return
	}
	maxH := b.MaxHealth()
	if maxH <= 0 || math.IsNaN(maxH) || math.IsInf(maxH, 0) {
		b.bar.SetHealthPercentage(0)
		return
	}
	ratio := float32(b.Health() / maxH)
	b.bar.SetHealthPercentage(ratio)
}
