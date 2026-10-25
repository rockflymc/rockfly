package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/cmd/admin"
	"github.com/df-mc/dragonfly/server/plugin/ts"
)

func TestDAG_TopologicalSort(t *testing.T) {
	manifests := map[string]PluginManifest{
		"Economy": {
			Name:         "Economy",
			Dependencies: []string{"Core"},
		},
		"Core": {
			Name: "Core",
		},
		"Shop": {
			Name:         "Shop",
			Dependencies: []string{"Economy", "Core"},
		},
		"Chat": {
			Name: "Chat",
		},
	}

	order, err := ResolveDependencies(manifests)
	if err != nil {
		t.Fatalf("unexpected DAG sort error: %v", err)
	}

	indices := make(map[string]int)
	for i, name := range order {
		indices[name] = i
	}

	// Core must come before Economy
	if indices["Core"] > indices["Economy"] {
		t.Errorf("Core index %d > Economy index %d", indices["Core"], indices["Economy"])
	}
	// Economy must come before Shop
	if indices["Economy"] > indices["Shop"] {
		t.Errorf("Economy index %d > Shop index %d", indices["Economy"], indices["Shop"])
	}
}

func TestDAG_CircularDependency(t *testing.T) {
	manifests := map[string]PluginManifest{
		"PluginA": {
			Name:         "PluginA",
			Dependencies: []string{"PluginB"},
		},
		"PluginB": {
			Name:         "PluginB",
			Dependencies: []string{"PluginC"},
		},
		"PluginC": {
			Name:         "PluginC",
			Dependencies: []string{"PluginA"},
		},
	}

	_, err := ResolveDependencies(manifests)
	if err == nil {
		t.Fatalf("expected circular dependency error, got nil")
	}
	if !strings.Contains(err.Error(), "circular dependency detected") {
		t.Errorf("expected error to mention circular dependency, got: %v", err)
	}
}

func TestDAG_MissingDependency(t *testing.T) {
	manifests := map[string]PluginManifest{
		"Addon": {
			Name:         "Addon",
			Dependencies: []string{"NonExistent"},
		},
	}

	_, err := ResolveDependencies(manifests)
	if err == nil {
		t.Fatalf("expected missing dependency error, got nil")
	}
	if !strings.Contains(err.Error(), "missing dependency") {
		t.Errorf("expected missing dependency error, got: %v", err)
	}
}

func TestPluginManager_InterdependentTSHotReload(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rockfly_pm_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 1. Create CorePlugin folder
	coreDir := filepath.Join(tmpDir, "CorePlugin")
	if err := os.MkdirAll(coreDir, 0755); err != nil {
		t.Fatalf("mkdir CorePlugin failed: %v", err)
	}
	coreManifest := `name = "CorePlugin"
version = "1.0.0"
main = "index.ts"
`
	if err := os.WriteFile(filepath.Join(coreDir, "plugin.toml"), []byte(coreManifest), 0644); err != nil {
		t.Fatalf("write core manifest failed: %v", err)
	}
	coreCode := `
let coreBootTime = Date.now();
rockfly.log("CorePlugin loaded successfully at " + coreBootTime);
`
	if err := os.WriteFile(filepath.Join(coreDir, "index.ts"), []byte(coreCode), 0644); err != nil {
		t.Fatalf("write core index.ts failed: %v", err)
	}

	// 2. Create MinigamePlugin folder (depends on CorePlugin)
	mgDir := filepath.Join(tmpDir, "MinigamePlugin")
	if err := os.MkdirAll(mgDir, 0755); err != nil {
		t.Fatalf("mkdir MinigamePlugin failed: %v", err)
	}
	mgManifest := `name = "MinigamePlugin"
version = "1.0.0"
main = "index.ts"
dependencies = ["CorePlugin"]
`
	if err := os.WriteFile(filepath.Join(mgDir, "plugin.toml"), []byte(mgManifest), 0644); err != nil {
		t.Fatalf("write mg manifest failed: %v", err)
	}
	mgCodeV1 := `
let state = "V1";
rockfly.events.on("PlayerJoinEvent", rockfly.Priority.Normal, (e) => {
    rockfly.log("Player joined minigame: " + state);
});
(globalThis as any).getState = function() { return state; };
`
	if err := os.WriteFile(filepath.Join(mgDir, "index.ts"), []byte(mgCodeV1), 0644); err != nil {
		t.Fatalf("write mg index.ts v1 failed: %v", err)
	}

	// 3. Initialize PluginManager
	pm := NewPluginManager(tmpDir)
	admin.SetPluginReloader(pm)

	if err := pm.LoadAll(); err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	order := pm.Order()
	if len(order) != 2 {
		t.Fatalf("expected 2 plugins in order, got %d (%v)", len(order), order)
	}
	if order[0] != "CorePlugin" || order[1] != "MinigamePlugin" {
		t.Fatalf("expected order [CorePlugin, MinigamePlugin], got %v", order)
	}

	if err := pm.EnableAll(); err != nil {
		t.Fatalf("EnableAll failed: %v", err)
	}

	// Assert V1 state
	mgPlugin, ok := pm.Plugin("MinigamePlugin")
	if !ok {
		t.Fatalf("MinigamePlugin not found")
	}
	tsPlugin := mgPlugin.(*ts.TSPlugin)
	val, err := tsPlugin.Eval("getState()")
	if err != nil {
		t.Fatalf("eval getState failed: %v", err)
	}
	if val.String() != "V1" {
		t.Fatalf("expected state V1, got %v", val.String())
	}

	// 4. Update MinigamePlugin source to V2
	mgCodeV2 := `
let state = "V2_UPDATED";
rockfly.events.on("PlayerJoinEvent", rockfly.Priority.Normal, (e) => {
    rockfly.log("Player joined minigame updated: " + state);
});
(globalThis as any).getState = function() { return state; };
`
	if err := os.WriteFile(filepath.Join(mgDir, "index.ts"), []byte(mgCodeV2), 0644); err != nil {
		t.Fatalf("write mg index.ts v2 failed: %v", err)
	}

	// 5. Trigger live hot reload via ReloadAll
	count, dur, err := pm.ReloadAll()
	if err != nil {
		t.Fatalf("ReloadAll failed: %v", err)
	}

	t.Logf("Hot-reloaded %d plugins in %v", count, dur)
	if count != 2 {
		t.Errorf("expected 2 reloaded plugins, got %d", count)
	}
	if dur > 100*time.Millisecond {
		t.Errorf("reload took too long: %v (>100ms)", dur)
	}

	// Verify updated logic in V2
	mgPluginV2, ok := pm.Plugin("MinigamePlugin")
	if !ok {
		t.Fatalf("MinigamePlugin not found after reload")
	}
	tsPluginV2 := mgPluginV2.(*ts.TSPlugin)
	valV2, err := tsPluginV2.Eval("getState()")
	if err != nil {
		t.Fatalf("eval getState v2 failed: %v", err)
	}
	if valV2.String() != "V2_UPDATED" {
		t.Fatalf("expected state V2_UPDATED after hot-reload, got %v", valV2.String())
	}

	// Verify rockfly.d.ts auto-generated
	dtsPath := filepath.Join(tmpDir, "rockfly.d.ts")
	if _, err := os.Stat(dtsPath); os.IsNotExist(err) {
		t.Errorf("rockfly.d.ts was not auto-generated")
	}
}
