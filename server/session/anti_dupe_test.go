package session

import (
	"sync"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestAntiDupeGuard_ValidateCount(t *testing.T) {
	guard := NewAntiDupeGuard()

	tests := []struct {
		count byte
		valid bool
	}{
		{0, false},
		{1, true},
		{16, true},
		{64, true},
		{127, true},
		{128, false},
		{255, false},
	}

	for _, tt := range tests {
		c, err := guard.ValidateCount(tt.count)
		if tt.valid && err != nil {
			t.Errorf("expected count %d to be valid, got err: %v", tt.count, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("expected count %d to be invalid, got valid %d", tt.count, c)
		}
	}
}

func TestAntiDupeGuard_EmptyActionsRejected(t *testing.T) {
	guard := NewAntiDupeGuard()
	s := &Session{}

	err := guard.ValidateRequest(s, protocol.ItemStackRequest{
		RequestID: 1,
		Actions:   nil,
	})
	if err == nil {
		t.Fatal("expected empty actions request to be rejected")
	}
}

func TestAntiDupeGuard_ConcurrentSlotAccessSequential(t *testing.T) {
	inv := inventory.New(1, nil)
	apple := item.NewStack(item.Apple{}, 1)
	_ = inv.SetItem(0, apple)

	var wg sync.WaitGroup
	var successCount int
	var mu sync.Mutex

	// 2 players trying to take the single item simultaneously
	const competitors = 2
	wg.Add(competitors)

	for i := 0; i < competitors; i++ {
		go func() {
			defer wg.Done()
			mu.Lock()
			defer mu.Unlock()

			it, _ := inv.Item(0)
			if !it.Empty() && it.Count() >= 1 {
				_ = inv.SetItem(0, item.Stack{})
				successCount++
			}
		}()
	}

	wg.Wait()

	if successCount != 1 {
		t.Fatalf("expected exactly 1 player to take the item (preventing dupe), got %d", successCount)
	}

	finalItem, _ := inv.Item(0)
	if !finalItem.Empty() {
		t.Fatal("expected inventory slot to be empty after take")
	}
}

func BenchmarkAntiDupeGuard_ValidateCount(b *testing.B) {
	guard := NewAntiDupeGuard()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = guard.ValidateCount(byte((i % 64) + 1))
	}
}
