package chunk

import (
	"sync"
)

// TicketType represents the origin/purpose of a chunk ticket.
type TicketType string

const (
	// TicketPlayer represents a ticket originating from a player/loader viewing chunks.
	TicketPlayer TicketType = "player"
	// TicketForced represents a permanent ticket placed by administrative commands (/forceload).
	TicketForced TicketType = "forced"
	// TicketPortal represents a temporary entity-ticking ticket created when teleporting through portals.
	TicketPortal TicketType = "portal"
	// TicketSpawn represents a ticket keeping the world spawn area loaded and ticking.
	TicketSpawn TicketType = "spawn"
	// TicketPlugin represents a ticket granted to a third-party plugin or extension.
	TicketPlugin TicketType = "plugin"
	// TicketFuture represents a ticket held for asynchronous processing (e.g. world gen, pathfinding).
	TicketFuture TicketType = "future"
)

// Ticket represents a load/tick ticket assigned to a chunk.
type Ticket struct {
	// Type is the origin/purpose of this ticket.
	Type TicketType
	// Level is the target ticket level (e.g. LevelEntityTicking=31).
	Level int
	// Radius is the propagation radius in chunks. If 0, only applies to the origin chunk.
	Radius int
	// Expiry is the absolute world tick when this ticket expires (0 for permanent).
	Expiry int64
	// Owner is an optional identifier for the holder of the ticket (e.g. player UUID, loader pointer).
	Owner any
}

// Expired returns true if the ticket has expired at currentTick.
func (t Ticket) Expired(currentTick int64) bool {
	return t.Expiry > 0 && currentTick >= t.Expiry
}

// StateChangeHandler is a callback function invoked when a chunk transitions between states.
type StateChangeHandler func(pos Pos, from, to ChunkState)

// TicketManager orchestrates chunk tickets, level propagation, linger hysteresis, and unload reaping.
type TicketManager struct {
	mu sync.RWMutex

	// tickets maps chunk coordinates to the tickets directly assigned to that chunk.
	tickets map[Pos][]Ticket

	// levels maps chunk coordinates to their calculated effective ticket level.
	levels map[Pos]int

	// states maps chunk coordinates to their current ChunkState.
	states map[Pos]ChunkState

	// lingers maps chunk coordinates to the tick when their unload linger period expires.
	lingers map[Pos]int64

	// defaultLingerDuration is the default linger duration in world ticks (e.g. 200 ticks = 10s).
	defaultLingerDuration int64

	// handlers is a slice of callbacks invoked on chunk state transitions.
	handlers []StateChangeHandler
}

// NewTicketManager creates a new TicketManager with the specified linger duration.
func NewTicketManager(defaultLingerTicks int64) *TicketManager {
	if defaultLingerTicks <= 0 {
		defaultLingerTicks = 200 // default 10 seconds
	}
	return &TicketManager{
		tickets:               make(map[Pos][]Ticket),
		levels:                make(map[Pos]int),
		states:                make(map[Pos]ChunkState),
		lingers:               make(map[Pos]int64),
		defaultLingerDuration: defaultLingerTicks,
	}
}

// OnStateChange registers a callback to be invoked whenever a chunk transitions state.
func (m *TicketManager) OnStateChange(handler StateChangeHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = append(m.handlers, handler)
}

// AddTicket adds a ticket to the given chunk position and propagates its level outward.
func (m *TicketManager) AddTicket(origin Pos, ticket Ticket) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tickets[origin] = append(m.tickets[origin], ticket)

	// Propagate ticket level across radius.
	radius := int32(ticket.Radius)
	for dx := -radius; dx <= radius; dx++ {
		for dz := -radius; dz <= radius; dz++ {
			p := Pos{origin[0] + dx, origin[1] + dz}
			dist := int(max(abs(dx), abs(dz)))
			targetLevel := ticket.Level + dist
			if targetLevel > LevelMax {
				targetLevel = LevelMax
			}

			// Clear any active linger timer since this chunk is actively targeted.
			delete(m.lingers, p)

			m.updateChunkLevel(p, targetLevel)
		}
	}
}

// RemoveTicket removes any tickets matching the given type and owner from origin.
// Returns true if at least one ticket was removed.
func (m *TicketManager) RemoveTicket(origin Pos, ticketType TicketType, owner any, currentTick int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	ticketList, ok := m.tickets[origin]
	if !ok || len(ticketList) == 0 {
		return false
	}

	var removedRadius int
	found := false
	filtered := ticketList[:0]
	for _, t := range ticketList {
		if t.Type == ticketType && (owner == nil || t.Owner == owner) {
			found = true
			if t.Radius > removedRadius {
				removedRadius = t.Radius
			}
		} else {
			filtered = append(filtered, t)
		}
	}

	if !found {
		return false
	}

	if len(filtered) == 0 {
		delete(m.tickets, origin)
	} else {
		m.tickets[origin] = filtered
	}

	// Re-evaluate affected area.
	radius := int32(removedRadius)
	for dx := -radius; dx <= radius; dx++ {
		for dz := -radius; dz <= radius; dz++ {
			p := Pos{origin[0] + dx, origin[1] + dz}
			m.recomputeChunkLevel(p, currentTick)
		}
	}

	return true
}

// Level returns the current effective ticket level of the chunk at pos.
func (m *TicketManager) Level(pos Pos) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	lvl, ok := m.levels[pos]
	if !ok {
		return LevelMax
	}
	return lvl
}

// State returns the current ChunkState of the chunk at pos.
func (m *TicketManager) State(pos Pos) ChunkState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	st, ok := m.states[pos]
	if !ok {
		return StateInaccessible
	}
	return st
}

// HasLinger returns whether pos currently has an active linger timer.
func (m *TicketManager) HasLinger(pos Pos, currentTick int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	expiry, ok := m.lingers[pos]
	return ok && currentTick < expiry
}

// CanUnload returns whether pos is safe to unload (inaccessible and not lingering).
func (m *TicketManager) CanUnload(pos Pos, currentTick int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// If chunk has active tickets or is not inaccessible, it cannot unload.
	if st, ok := m.states[pos]; ok && st > StateInaccessible {
		return false
	}

	// If lingering, do not unload yet.
	if expiry, ok := m.lingers[pos]; ok && currentTick < expiry {
		return false
	}

	return true
}

// Reap processes expired tickets and returns chunks that have finished lingering and are ready for unload.
func (m *TicketManager) Reap(currentTick int64) (expiredCount int, evictable []Pos) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Purge expired tickets.
	for pos, tickets := range m.tickets {
		var active []Ticket
		var maxRadius int
		for _, t := range tickets {
			if t.Expired(currentTick) {
				expiredCount++
				if t.Radius > maxRadius {
					maxRadius = t.Radius
				}
			} else {
				active = append(active, t)
			}
		}

		if len(active) == len(tickets) {
			continue
		}

		if len(active) == 0 {
			delete(m.tickets, pos)
		} else {
			m.tickets[pos] = active
		}

		// Recompute levels for affected radius.
		rad := int32(maxRadius)
		for dx := -rad; dx <= rad; dx++ {
			for dz := -rad; dz <= rad; dz++ {
				p := Pos{pos[0] + dx, pos[1] + dz}
				m.recomputeChunkLevel(p, currentTick)
			}
		}
	}

	// 2. Identify expired lingers ready for eviction.
	for pos, expiry := range m.lingers {
		if currentTick >= expiry {
			if st, ok := m.states[pos]; !ok || st <= StateInaccessible {
				evictable = append(evictable, pos)
				delete(m.lingers, pos)
				delete(m.levels, pos)
				delete(m.states, pos)
			}
		}
	}

	return expiredCount, evictable
}

// updateChunkLevel unconditionally sets pos to newLevel if newLevel is lower (better) than existing.
func (m *TicketManager) updateChunkLevel(pos Pos, newLevel int) {
	currentLevel, exists := m.levels[pos]
	if exists && currentLevel <= newLevel {
		return
	}

	m.levels[pos] = newLevel
	oldState, stateExists := m.states[pos]
	if !stateExists {
		oldState = StateInaccessible
	}
	newState := StateFromLevel(newLevel)
	m.states[pos] = newState

	if oldState != newState {
		for _, h := range m.handlers {
			h(pos, oldState, newState)
		}
	}
}

// recomputeChunkLevel recalculates the best effective ticket level for pos across all sources.
func (m *TicketManager) recomputeChunkLevel(pos Pos, currentTick int64) {
	bestLevel := LevelMax

	// Scan tickets in surrounding radius that could influence pos.
	for origin, tickets := range m.tickets {
		dist := int(pos.Distance(origin))
		for _, t := range tickets {
			if dist <= t.Radius {
				lvl := t.Level + dist
				if lvl < bestLevel {
					bestLevel = lvl
				}
			}
		}
	}

	oldState, stateExists := m.states[pos]
	if !stateExists {
		oldState = StateInaccessible
	}

	if bestLevel >= LevelMax {
		delete(m.levels, pos)
		m.states[pos] = StateInaccessible
		newState := StateInaccessible

		// Chunk degraded to Inaccessible: start linger timer to prevent thrashing.
		if oldState > StateInaccessible {
			m.lingers[pos] = currentTick + m.defaultLingerDuration
		}

		if oldState != newState {
			for _, h := range m.handlers {
				h(pos, oldState, newState)
			}
		}
		return
	}

	m.levels[pos] = bestLevel
	newState := StateFromLevel(bestLevel)
	m.states[pos] = newState

	// If now active, cancel any pending linger.
	if newState > StateInaccessible {
		delete(m.lingers, pos)
	}

	if oldState != newState {
		for _, h := range m.handlers {
			h(pos, oldState, newState)
		}
	}
}

func abs(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
