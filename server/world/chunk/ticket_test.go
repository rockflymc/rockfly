package chunk

import (
	"testing"
)

func TestChunkLevel_Transitions(t *testing.T) {
	tests := []struct {
		level         int
		expectedState ChunkState
		canBlocks     bool
		canEntities   bool
		isLoaded      bool
	}{
		{LevelEntityTicking, StateEntityTicking, true, true, true},
		{30, StateEntityTicking, true, true, true},
		{LevelTicking, StateTicking, true, false, true},
		{LevelBorder, StateBorder, false, false, true},
		{LevelInaccessible, StateInaccessible, false, false, false},
		{35, StateInaccessible, false, false, false},
		{LevelMax, StateInaccessible, false, false, false},
	}

	for _, tt := range tests {
		st := StateFromLevel(tt.level)
		if st != tt.expectedState {
			t.Errorf("level %d: expected %v, got %v", tt.level, tt.expectedState, st)
		}
		if st.CanTickBlocks() != tt.canBlocks {
			t.Errorf("level %d CanTickBlocks: expected %v, got %v", tt.level, tt.canBlocks, st.CanTickBlocks())
		}
		if st.CanTickEntities() != tt.canEntities {
			t.Errorf("level %d CanTickEntities: expected %v, got %v", tt.level, tt.canEntities, st.CanTickEntities())
		}
		if st.IsLoaded() != tt.isLoaded {
			t.Errorf("level %d IsLoaded: expected %v, got %v", tt.level, tt.isLoaded, st.IsLoaded())
		}
	}
}

func TestTicketManager_NeighborPropagation(t *testing.T) {
	tm := NewTicketManager(200)

	origin := Pos{10, 10}
	ticket := Ticket{
		Type:   TicketPlayer,
		Level:  LevelEntityTicking, // 31
		Radius: 2,                  // radius 2
		Owner:  "player-1",
	}

	tm.AddTicket(origin, ticket)

	// Center should be LevelEntityTicking (31) -> StateEntityTicking
	if lvl := tm.Level(origin); lvl != 31 {
		t.Fatalf("expected origin level 31, got %d", lvl)
	}
	if st := tm.State(origin); st != StateEntityTicking {
		t.Fatalf("expected origin StateEntityTicking, got %v", st)
	}

	// Ring 1: dist 1 -> Level 32 -> StateTicking
	ring1 := []Pos{{11, 10}, {9, 10}, {10, 11}, {10, 9}, {11, 11}}
	for _, p := range ring1 {
		if lvl := tm.Level(p); lvl != 32 {
			t.Errorf("ring 1 pos %v: expected level 32, got %d", p, lvl)
		}
		if st := tm.State(p); st != StateTicking {
			t.Errorf("ring 1 pos %v: expected StateTicking, got %v", p, st)
		}
	}

	// Ring 2: dist 2 -> Level 33 -> StateBorder
	ring2 := []Pos{{12, 10}, {8, 10}, {10, 12}, {10, 8}, {12, 12}}
	for _, p := range ring2 {
		if lvl := tm.Level(p); lvl != 33 {
			t.Errorf("ring 2 pos %v: expected level 33, got %d", p, lvl)
		}
		if st := tm.State(p); st != StateBorder {
			t.Errorf("ring 2 pos %v: expected StateBorder, got %v", p, st)
		}
	}

	// Outside radius: dist 3 -> not covered -> LevelMax -> StateInaccessible
	out := Pos{13, 10}
	if lvl := tm.Level(out); lvl != LevelMax {
		t.Errorf("outside pos %v: expected LevelMax %d, got %d", out, LevelMax, lvl)
	}
	if st := tm.State(out); st != StateInaccessible {
		t.Errorf("outside pos %v: expected StateInaccessible, got %v", out, st)
	}
}

func TestTicketManager_LingerReaper(t *testing.T) {
	lingerTicks := int64(100)
	tm := NewTicketManager(lingerTicks)

	origin := Pos{0, 0}
	ticket := Ticket{
		Type:   TicketForced,
		Level:  LevelEntityTicking,
		Radius: 1,
		Owner:  "admin",
	}

	tm.AddTicket(origin, ticket)

	if tm.CanUnload(origin, 0) {
		t.Fatal("expected chunk with active ticket to NOT be unloadable")
	}

	// Remove ticket at tick 50
	currentTick := int64(50)
	removed := tm.RemoveTicket(origin, TicketForced, "admin", currentTick)
	if !removed {
		t.Fatal("expected ticket to be removed successfully")
	}

	// Chunk should now be Inaccessible, but lingering until tick 50 + 100 = 150
	if st := tm.State(origin); st != StateInaccessible {
		t.Fatalf("expected chunk to be Inaccessible, got %v", st)
	}
	if !tm.HasLinger(origin, currentTick) {
		t.Fatal("expected chunk to have active linger")
	}
	if tm.CanUnload(origin, currentTick) {
		t.Fatal("expected chunk to NOT be unloadable while lingering at tick 50")
	}
	if tm.CanUnload(origin, 149) {
		t.Fatal("expected chunk to NOT be unloadable at tick 149")
	}

	// At tick 150 (linger expired), chunk becomes unloadable
	if !tm.CanUnload(origin, 150) {
		t.Fatal("expected chunk to be unloadable at tick 150")
	}

	// Reap at tick 150 should evict origin
	_, evictable := tm.Reap(150)
	found := false
	for _, p := range evictable {
		if p == origin {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected origin %v in evictable chunks %v", origin, evictable)
	}
}

func TestTicketManager_LingerCancelOnReAdd(t *testing.T) {
	tm := NewTicketManager(100)
	origin := Pos{5, 5}

	tm.AddTicket(origin, Ticket{Type: TicketPlayer, Level: 31, Radius: 1, Owner: "p1"})
	tm.RemoveTicket(origin, TicketPlayer, "p1", 10)

	if !tm.HasLinger(origin, 10) {
		t.Fatal("expected linger after remove")
	}

	// Re-add ticket before linger expires (e.g. player stepped back into chunk)
	tm.AddTicket(origin, Ticket{Type: TicketPlayer, Level: 31, Radius: 1, Owner: "p1"})

	if tm.HasLinger(origin, 20) {
		t.Fatal("expected linger to be cancelled upon re-adding ticket")
	}
	if st := tm.State(origin); st != StateEntityTicking {
		t.Fatalf("expected state restored to StateEntityTicking, got %v", st)
	}
}

func TestTicketManager_PlayerTeleportRamping(t *testing.T) {
	tm := NewTicketManager(50)

	var transitions []string
	tm.OnStateChange(func(pos Pos, from, to ChunkState) {
		if pos == (Pos{0, 0}) {
			transitions = append(transitions, from.String()+"->"+to.String())
		}
	})

	// Player starts at (0, 0)
	tm.AddTicket(Pos{0, 0}, Ticket{Type: TicketPlayer, Level: 31, Radius: 0, Owner: "steve"})

	if tm.State(Pos{0, 0}) != StateEntityTicking {
		t.Fatalf("expected StateEntityTicking, got %v", tm.State(Pos{0, 0}))
	}

	// Player teleports to (50, 50) at tick 10
	tm.RemoveTicket(Pos{0, 0}, TicketPlayer, "steve", 10)
	tm.AddTicket(Pos{50, 50}, Ticket{Type: TicketPlayer, Level: 31, Radius: 0, Owner: "steve"})

	// Origin (0,0) degrades to Inaccessible with linger
	if tm.State(Pos{0, 0}) != StateInaccessible {
		t.Fatalf("expected (0, 0) to be Inaccessible, got %v", tm.State(Pos{0, 0}))
	}
	if !tm.HasLinger(Pos{0, 0}, 10) {
		t.Fatal("expected (0, 0) to linger after teleport")
	}

	// Target (50, 50) ramps to EntityTicking
	if tm.State(Pos{50, 50}) != StateEntityTicking {
		t.Fatalf("expected (50, 50) to be EntityTicking, got %v", tm.State(Pos{50, 50}))
	}
}

func BenchmarkTicketManager_Propagation(b *testing.B) {
	tm := NewTicketManager(200)
	origin := Pos{0, 0}
	ticket := Ticket{
		Type:   TicketPlayer,
		Level:  LevelEntityTicking,
		Radius: 8, // 17x17 chunk grid = 289 chunks
		Owner:  "bench-player",
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tm.AddTicket(origin, ticket)
		tm.RemoveTicket(origin, TicketPlayer, "bench-player", int64(i))
	}
}

func BenchmarkTicketManager_Reap(b *testing.B) {
	tm := NewTicketManager(10)
	for x := int32(0); x < 10; x++ {
		for z := int32(0); z < 10; z++ {
			tm.AddTicket(Pos{x, z}, Ticket{
				Type:   TicketPortal,
				Level:  31,
				Radius: 0,
				Expiry: 50,
			})
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tm.Reap(100)
	}
}
