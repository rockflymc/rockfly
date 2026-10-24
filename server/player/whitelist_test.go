package player

import (
	"path/filepath"
	"testing"
)

func TestWhitelistManager(t *testing.T) {
	tempDir := t.TempDir()
	wlPath := filepath.Join(tempDir, "whitelist.json")

	mgr := NewWhitelistManager(wlPath)
	if err := mgr.Load(); err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if mgr.Enabled() {
		t.Errorf("expected whitelist to be disabled by default")
	}

	mgr.SetEnabled(true)
	if !mgr.Enabled() {
		t.Errorf("expected whitelist to be enabled")
	}

	if mgr.IsWhitelisted("Alex", "") {
		t.Errorf("expected Alex not to be whitelisted yet")
	}

	if err := mgr.Add("Alex", "uuid-alex-123"); err != nil {
		t.Fatalf("failed to add Alex: %v", err)
	}

	if !mgr.IsWhitelisted("alex", "") || !mgr.IsWhitelisted("", "uuid-alex-123") {
		t.Errorf("expected Alex to be whitelisted by name and uuid")
	}

	// Verify persistence in a new manager
	mgr2 := NewWhitelistManager(wlPath)
	if err := mgr2.Load(); err != nil {
		t.Fatalf("failed to reload whitelist: %v", err)
	}
	if !mgr2.IsWhitelisted("Alex", "") {
		t.Errorf("expected persisted whitelist entry for Alex")
	}

	if err := mgr2.Remove("alex"); err != nil {
		t.Fatalf("failed to remove Alex: %v", err)
	}
	if mgr2.IsWhitelisted("Alex", "") {
		t.Errorf("expected Alex to be removed from whitelist")
	}
}
