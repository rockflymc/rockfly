package rockfly

import "testing"

func TestSDK_Stub(t *testing.T) {
	Log("Testing SDK logging")
	LogWithLevel(LogLevelWarn, "Warning test")

	muts := []BlockMutation{
		{X: 0, Y: 64, Z: 0, RuntimeID: 1},
	}
	committed := CommitBlockBatch(muts)
	if committed != 1 {
		t.Fatalf("expected 1 committed, got %d", committed)
	}

	_ = GetBlock(0, 64, 0)
}
