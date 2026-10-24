package player

import (
	"path/filepath"
	"testing"
)

func TestOperatorManager(t *testing.T) {
	tempDir := t.TempDir()
	opsPath := filepath.Join(tempDir, "ops.json")

	mgr := NewOperatorManager(opsPath)
	if err := mgr.Load(); err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if mgr.IsOp("Steve") {
		t.Errorf("expected Steve to not be an op initially")
	}

	if err := mgr.SetOp("Steve", "1234-uuid", 4, true); err != nil {
		t.Fatalf("failed to set op: %v", err)
	}

	if !mgr.IsOp("Steve") || !mgr.IsOp("steve") || !mgr.IsOp("1234-uuid") {
		t.Errorf("expected Steve to be an op by name and uuid")
	}

	if lvl := mgr.OpLevel("steve"); lvl != 4 {
		t.Errorf("expected op level 4, got %d", lvl)
	}

	if !mgr.BypassesPlayerLimit("steve") {
		t.Errorf("expected Steve to bypass player limit")
	}

	// Verify persistence by reloading in a new manager
	mgr2 := NewOperatorManager(opsPath)
	if err := mgr2.Load(); err != nil {
		t.Fatalf("failed to reload ops: %v", err)
	}
	if !mgr2.IsOp("steve") {
		t.Errorf("expected persisted op for steve")
	}

	// Test Deop
	if err := mgr2.Deop("Steve"); err != nil {
		t.Fatalf("failed to deop: %v", err)
	}
	if mgr2.IsOp("steve") {
		t.Errorf("expected steve to no longer be op")
	}
}

func TestBanManager(t *testing.T) {
	tempDir := t.TempDir()
	pBanPath := filepath.Join(tempDir, "banned-players.json")
	ipBanPath := filepath.Join(tempDir, "banned-ips.json")

	mgr := NewBanManager(pBanPath, ipBanPath)
	if err := mgr.Load(); err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	// Test Player Ban
	if err := mgr.BanPlayer("Griefer", "uuid-griefer", "Griefing spawn", "Console"); err != nil {
		t.Fatalf("failed to ban player: %v", err)
	}

	banned, reason := mgr.IsPlayerBanned("griefer")
	if !banned || reason != "Griefing spawn" {
		t.Errorf("expected player to be banned with reason, got banned=%v reason=%q", banned, reason)
	}

	// Test IP Ban
	if err := mgr.BanIP("192.168.1.100", "Bot spam", "Console"); err != nil {
		t.Fatalf("failed to ban IP: %v", err)
	}

	bannedIP, reasonIP := mgr.IsIPBanned("192.168.1.100")
	if !bannedIP || reasonIP != "Bot spam" {
		t.Errorf("expected IP to be banned with reason, got banned=%v reason=%q", bannedIP, reasonIP)
	}

	// Test Reload Persistence
	mgr2 := NewBanManager(pBanPath, ipBanPath)
	if err := mgr2.Load(); err != nil {
		t.Fatalf("failed to reload bans: %v", err)
	}
	if banned, _ := mgr2.IsPlayerBanned("uuid-griefer"); !banned {
		t.Errorf("expected persisted player ban by uuid")
	}
	if banned, _ := mgr2.IsIPBanned("192.168.1.100"); !banned {
		t.Errorf("expected persisted IP ban")
	}

	// Test Pardons
	if err := mgr2.PardonPlayer("Griefer"); err != nil {
		t.Fatalf("failed to pardon player: %v", err)
	}
	if banned, _ := mgr2.IsPlayerBanned("griefer"); banned {
		t.Errorf("expected player to be unbanned")
	}

	if err := mgr2.PardonIP("192.168.1.100"); err != nil {
		t.Fatalf("failed to pardon IP: %v", err)
	}
	if banned, _ := mgr2.IsIPBanned("192.168.1.100"); banned {
		t.Errorf("expected IP to be unbanned")
	}
}
