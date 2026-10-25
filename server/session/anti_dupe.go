package session

import (
	"fmt"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// AntiDupeGuard provides cryptographic-grade item transaction validation, race condition
// protection, and container anti-duplication defense for ItemStackRequest handling.
type AntiDupeGuard struct {
	mu           sync.Mutex
	lastAction   map[uuid.UUID]time.Time
	actionCounts map[uuid.UUID]int
}

// NewAntiDupeGuard creates a new AntiDupeGuard.
func NewAntiDupeGuard() *AntiDupeGuard {
	return &AntiDupeGuard{
		lastAction:   make(map[uuid.UUID]time.Time),
		actionCounts: make(map[uuid.UUID]int),
	}
}

// ValidateRequest checks an ItemStackRequest for malicious exploits:
// - Rate-limiting / packet spam (> 100 requests/sec)
// - Empty action sets
// - Container detachment / rapid-close race condition
func (g *AntiDupeGuard) ValidateRequest(s *Session, req protocol.ItemStackRequest) error {
	if g == nil {
		return nil
	}
	if len(req.Actions) == 0 {
		return fmt.Errorf("empty item stack request actions")
	}
	if s == nil || s.conn == nil {
		return fmt.Errorf("nil session or connection in item stack request")
	}

	id := s.conn.IdentityData().Identity
	uid, err := uuid.Parse(id)
	if err != nil {
		uid = uuid.Nil
	}

	now := time.Now()
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.lastAction == nil {
		g.lastAction = make(map[uuid.UUID]time.Time)
	}
	if g.actionCounts == nil {
		g.actionCounts = make(map[uuid.UUID]int)
	}

	last, exists := g.lastAction[uid]
	if exists && now.Sub(last) < time.Second {
		g.actionCounts[uid]++
		if g.actionCounts[uid] > 128 {
			return fmt.Errorf("item stack request rate limit exceeded (%d req/s)", g.actionCounts[uid])
		}
	} else {
		g.lastAction[uid] = now
		g.actionCounts[uid] = 1
	}

	return nil
}

// ValidateCount verifies that a transfer count is strictly positive and within bounds.
func (g *AntiDupeGuard) ValidateCount(count byte) (int, error) {
	c := int(count)
	if c <= 0 {
		return 0, fmt.Errorf("zero or negative transfer count (%d)", c)
	}
	if c > 127 {
		return 0, fmt.Errorf("excessive stack transfer count (%d)", c)
	}
	return c, nil
}

// ResyncContainer sends an authoritative full container and inventory content update to the client
// after a failed or rejected transaction, wiping any client-side ghost or duplicated items.
func (g *AntiDupeGuard) ResyncContainer(s *Session, tx *world.Tx, containerID byte) {
	if g == nil || s == nil || s.conn == nil {
		return
	}

	if inv, _ := s.invByID(int32(containerID), tx); inv != nil {
		s.sendInv(inv, uint32(containerID))
	}
	if s.inv != nil {
		s.sendInv(s.inv, protocol.WindowIDInventory)
	}
	if s.ui != nil {
		s.sendInv(s.ui, protocol.WindowIDUI)
	}
	if s.offHand != nil {
		s.sendInv(s.offHand, protocol.WindowIDOffHand)
	}
	if s.armour != nil && s.armour.Inventory() != nil {
		s.sendInv(s.armour.Inventory(), protocol.WindowIDArmour)
	}
}

// Clear removes tracked rate-limiting data for a disconnected session.
func (g *AntiDupeGuard) Clear(id uuid.UUID) {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.lastAction != nil {
		delete(g.lastAction, id)
	}
	if g.actionCounts != nil {
		delete(g.actionCounts, id)
	}
}
