package world

import (
	"context"
	"testing"

	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/go-gl/mathgl/mgl64"
)

type dummyWorldViewer struct {
	NopViewer
}

func TestWorld_ChunkTicketsAndStateIntegration(t *testing.T) {
	w := newTestWorld()
	defer w.Close()

	viewer := &dummyWorldViewer{}
	loader := NewLoader(4, w, viewer)

	pos1 := mgl64.Vec3{0, 64, 0}
	chunkPos1 := ChunkPos{0, 0}

	w.Do(func(tx *Tx) {
		loader.Move(tx, pos1)
		loader.Load(tx, 50)
	}).Wait(context.Background())

	// At chunk (0, 0), ticket level should be LevelEntityTicking (31)
	if lvl := w.ChunkLevel(chunkPos1); lvl != chunk.LevelEntityTicking {
		t.Fatalf("expected chunk level %d at origin, got %d", chunk.LevelEntityTicking, lvl)
	}
	if st := w.ChunkState(chunkPos1); st != chunk.StateEntityTicking {
		t.Fatalf("expected StateEntityTicking at origin, got %v", st)
	}

	// Neighbor at distance 1 (1, 0) should be LevelTicking (32)
	chunkPosNeighbor := ChunkPos{1, 0}
	if lvl := w.ChunkLevel(chunkPosNeighbor); lvl != chunk.LevelTicking {
		t.Fatalf("expected neighbor level %d, got %d", chunk.LevelTicking, lvl)
	}
	if st := w.ChunkState(chunkPosNeighbor); st != chunk.StateTicking {
		t.Fatalf("expected neighbor StateTicking, got %v", st)
	}

	// Far neighbor at distance 2 (2, 0) should be LevelBorder (33)
	chunkPosBorder := ChunkPos{2, 0}
	if lvl := w.ChunkLevel(chunkPosBorder); lvl != chunk.LevelBorder {
		t.Fatalf("expected border level %d, got %d", chunk.LevelBorder, lvl)
	}
	if st := w.ChunkState(chunkPosBorder); st != chunk.StateBorder {
		t.Fatalf("expected border StateBorder, got %v", st)
	}

	// Player teleports to chunk (50, 50)
	pos2 := mgl64.Vec3{50 * 16, 64, 50 * 16}
	chunkPos2 := ChunkPos{50, 50}

	w.Do(func(tx *Tx) {
		loader.Move(tx, pos2)
	}).Wait(context.Background())

	// Old chunk (0, 0) should degrade to StateInaccessible but be lingering
	if st := w.ChunkState(chunkPos1); st != chunk.StateInaccessible {
		t.Fatalf("expected old chunk to become Inaccessible, got %v", st)
	}
	if !w.TicketManager().HasLinger(chunk.Pos(chunkPos1), 0) {
		t.Fatal("expected old chunk to have active linger timer")
	}

	// New chunk (50, 50) should ramp up to StateEntityTicking
	if st := w.ChunkState(chunkPos2); st != chunk.StateEntityTicking {
		t.Fatalf("expected new chunk to become StateEntityTicking, got %v", st)
	}

	// Closing loader
	w.Do(func(tx *Tx) {
		loader.Close(tx)
	}).Wait(context.Background())

	// New chunk (50, 50) should now also become Inaccessible and linger
	if st := w.ChunkState(chunkPos2); st != chunk.StateInaccessible {
		t.Fatalf("expected closed chunk to become Inaccessible, got %v", st)
	}
}

func TestWorld_CloseUnusedChunksWithLinger(t *testing.T) {
	w := newTestWorld()
	defer w.Close()

	pos := ChunkPos{5, 5}
	c := w.emptyColumn()

	w.Do(func(tx *Tx) {
		w.chunks[pos] = c
		// Add a temporary ticket expiring at tick 100
		w.TicketManager().AddTicket(chunk.Pos(pos), chunk.Ticket{
			Type:   chunk.TicketForced,
			Level:  chunk.LevelEntityTicking,
			Radius: 0,
			Expiry: 100,
		})

		// At tick 50, ticket is active -> cannot unload
		w.closeUnusedChunks(tx)
		if _, ok := w.chunks[pos]; !ok {
			t.Fatal("expected chunk to NOT be closed while active ticket exists")
		}

		// Advance past expiry: ticket expires at 100, linger starts (default 200 ticks, expires at 300)
		w.set.CurrentTick = 150
		w.closeUnusedChunks(tx)
		if _, ok := w.chunks[pos]; !ok {
			t.Fatal("expected chunk to NOT be closed while lingering at tick 150")
		}

		// Advance past linger expiry (tick 350)
		w.set.CurrentTick = 350
		w.closeUnusedChunks(tx)
		if _, ok := w.chunks[pos]; ok {
			t.Fatal("expected chunk to be closed after linger expired at tick 350")
		}
	}).Wait(context.Background())
}

func newTestWorld() *World {
	c := Config{
		Dim:         Overworld,
		Synchronous: true,
	}
	return c.New()
}
