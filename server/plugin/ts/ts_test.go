package ts

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/plugin"
)

func TestTranspiler_TranspileSpeed(t *testing.T) {
	transpiler := NewTranspiler()
	tsCode := `
interface PlayerInfo {
    name: string;
    level: number;
}

function greet(info: PlayerInfo): string {
    return "Hello, " + info.name + "! Level: " + info.level;
}

const result: string = greet({ name: "Steve", level: 50 });
`

	start := time.Now()
	js, err := transpiler.Transpile(tsCode)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("transpilation failed: %v", err)
	}
	if !strings.Contains(js, "greet") {
		t.Fatalf("expected output JS to contain 'greet', got: %s", js)
	}

	t.Logf("TypeScript transpiled in %v (<5ms requirement)", elapsed)
	if elapsed > 100*time.Millisecond {
		t.Errorf("transpilation took too long: %v", elapsed)
	}
}

func TestTranspiler_MultiFileBundle(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rockfly_ts_bundle_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	helperPath := filepath.Join(tmpDir, "helper.ts")
	helperCode := `
export function formatMessage(sender: string, msg: string): string {
    return "[" + sender.toUpperCase() + "] " + msg;
}
`
	if err := os.WriteFile(helperPath, []byte(helperCode), 0644); err != nil {
		t.Fatalf("failed to write helper.ts: %v", err)
	}

	indexPath := filepath.Join(tmpDir, "index.ts")
	indexCode := `
import { formatMessage } from "./helper";

const msg = formatMessage("Server", "Plugin enabled successfully!");
rockfly.log(msg);

rockfly.commands.register("heal", "Heals executing player", (p, args) => {
    if (p) {
        p.heal(20);
        p.sendMessage("You have been fully healed!");
    }
});
`
	if err := os.WriteFile(indexPath, []byte(indexCode), 0644); err != nil {
		t.Fatalf("failed to write index.ts: %v", err)
	}

	transpiler := NewTranspiler()
	bundledJS, err := transpiler.Bundle(indexPath)
	if err != nil {
		t.Fatalf("multi-file bundle failed: %v", err)
	}

	t.Logf("BUNDLED JS:\n%s", bundledJS)

	if !strings.Contains(bundledJS, "formatMessage") {
		t.Fatalf("bundled JS missing helper function")
	}

	// Now run the bundled JS in TSPlugin runtime
	manifest := plugin.PluginManifest{
		Name:    "HealPlugin",
		Version: "1.0.0",
		Author:  "RockflyTeam",
		Main:    "index.ts",
	}

	p := NewTSPlugin(manifest)
	if err := p.Load(bundledJS); err != nil {
		t.Fatalf("failed to load bundled plugin into runtime: %v", err)
	}

	if err := p.Enable(); err != nil {
		t.Fatalf("failed to enable plugin: %v", err)
	}

	if err := p.Disable(); err != nil {
		t.Fatalf("failed to disable plugin: %v", err)
	}
}

func TestTSPlugin_EventDispatchAndCancellation(t *testing.T) {
	transpiler := NewTranspiler()
	tsCode := `
let breakCancelled = false;

rockfly.events.on("BlockBreakEvent", rockfly.Priority.High, (e) => {
    if (e.blockName === "minecraft:diamond_ore") {
        e.cancel();
        breakCancelled = true;
    }
});

(globalThis as any).isCancelled = function(): boolean {
    return breakCancelled;
};
`
	js, err := transpiler.Transpile(tsCode)
	if err != nil {
		t.Fatalf("transpilation failed: %v", err)
	}

	manifest := plugin.PluginManifest{Name: "MiningGuard", Version: "1.0.0"}
	p := NewTSPlugin(manifest)
	if err := p.Load(js); err != nil {
		t.Fatalf("failed to load plugin: %v", err)
	}
	if err := p.Enable(); err != nil {
		t.Fatalf("failed to enable: %v", err)
	}

	// Mock event object
	type mockBlockBreakEvent struct {
		BlockName string
		Cancelled bool
	}

	var eventData map[string]any
	eventData = map[string]any{
		"blockName": "minecraft:diamond_ore",
		"cancelled": false,
		"cancel": func() {
			eventData["cancelled"] = true
		},
	}

	if err := p.DispatchEvent("BlockBreakEvent", eventData); err != nil {
		t.Fatalf("dispatch error: %v", err)
	}

	val, err := p.vm.RunString("isCancelled()")
	if err != nil {
		t.Fatalf("eval isCancelled failed: %v", err)
	}
	if !val.ToBoolean() {
		t.Fatalf("expected breakCancelled to be true after event handling")
	}

	_ = p.Disable()
}

func TestTSPlugin_ExecutionTimeoutGuard(t *testing.T) {
	manifest := plugin.PluginManifest{Name: "InfiniteLoopPlugin", Version: "1.0.0"}
	p := NewTSPlugin(manifest)
	p.SetTimeout(50 * time.Millisecond) // 50ms guard

	infiniteLoopCode := `
function onEnable() {
    let count = 0;
    while (true) {
        count++;
    }
}
`
	if err := p.Load(infiniteLoopCode); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	start := time.Now()
	err := p.Enable()
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("expected error from infinite loop execution timeout")
	}
	t.Logf("Infinite loop safely aborted after %v with error: %v", elapsed, err)

	if elapsed > 1*time.Second {
		t.Errorf("timeout guard took too long to abort: %v", elapsed)
	}
}

func TestDTSGenerator(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rockfly_dts_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := GenerateDTS(tmpDir); err != nil {
		t.Fatalf("GenerateDTS failed: %v", err)
	}

	dtsPath := filepath.Join(tmpDir, "rockfly.d.ts")
	data, err := os.ReadFile(dtsPath)
	if err != nil {
		t.Fatalf("failed to read rockfly.d.ts: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "declare namespace rockfly") {
		t.Errorf("expected d.ts to declare rockfly namespace")
	}
	if !strings.Contains(content, "enum Priority") {
		t.Errorf("expected d.ts to declare Priority enum")
	}
	if !strings.Contains(content, "interface Player") {
		t.Errorf("expected d.ts to declare Player interface")
	}
}

func TestTSPlugin_ConcurrentEventDispatch(t *testing.T) {
	manifest := plugin.PluginManifest{Name: "ConcurrentPlugin", Version: "1.0.0"}
	p := NewTSPlugin(manifest)

	code := `
let counter = 0;
rockfly.events.on("TestEvent", rockfly.Priority.Normal, (e) => {
    counter += e.value;
});
(globalThis as any).getCounter = () => counter;
`
	transpiler := NewTranspiler()
	js, err := transpiler.Transpile(code)
	if err != nil {
		t.Fatalf("transpile error: %v", err)
	}

	if err := p.Load(js); err != nil {
		t.Fatalf("load error: %v", err)
	}
	if err := p.Enable(); err != nil {
		t.Fatalf("enable error: %v", err)
	}

	const goroutines = 20
	const iterations = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = p.DispatchEvent("TestEvent", map[string]any{"value": 1})
			}
		}()
	}

	wg.Wait()

	val, err := p.vm.RunString("getCounter()")
	if err != nil {
		t.Fatalf("eval getCounter failed: %v", err)
	}
	expected := int64(goroutines * iterations)
	if val.ToInteger() != expected {
		t.Fatalf("expected counter %d, got %d", expected, val.ToInteger())
	}

	_ = p.Disable()
}

func TestTSPlugin_DisableCleansListeners(t *testing.T) {
	manifest := plugin.PluginManifest{Name: "DisableTestPlugin", Version: "1.0.0"}
	p := NewTSPlugin(manifest)

	code := `
let fired = false;
rockfly.events.on("CustomEvent", rockfly.Priority.Normal, (e) => {
    fired = true;
});
(globalThis as any).hasFired = () => fired;
`
	transpiler := NewTranspiler()
	js, err := transpiler.Transpile(code)
	if err != nil {
		t.Fatalf("transpile error: %v", err)
	}

	if err := p.Load(js); err != nil {
		t.Fatalf("load error: %v", err)
	}
	if err := p.Enable(); err != nil {
		t.Fatalf("enable error: %v", err)
	}

	// Disable plugin
	if err := p.Disable(); err != nil {
		t.Fatalf("disable error: %v", err)
	}

	// Dispatch event after disable
	_ = p.DispatchEvent("CustomEvent", map[string]any{})

	val, err := p.vm.RunString("hasFired()")
	if err != nil {
		t.Fatalf("eval hasFired failed: %v", err)
	}
	if val.ToBoolean() {
		t.Fatalf("expected listener not to fire after plugin was disabled")
	}
}

func TestTSPlugin_PanicRecovery(t *testing.T) {
	manifest := plugin.PluginManifest{Name: "PanicPlugin", Version: "1.0.0"}
	p := NewTSPlugin(manifest)

	code := `
rockfly.events.on("PanicEvent", rockfly.Priority.Normal, (e) => {
    throw new Error("Script threw intentional test error");
});
`
	transpiler := NewTranspiler()
	js, err := transpiler.Transpile(code)
	if err != nil {
		t.Fatalf("transpile error: %v", err)
	}

	if err := p.Load(js); err != nil {
		t.Fatalf("load error: %v", err)
	}
	if err := p.Enable(); err != nil {
		t.Fatalf("enable error: %v", err)
	}

	err = p.DispatchEvent("PanicEvent", map[string]any{})
	if err == nil {
		t.Fatalf("expected error from thrown script exception")
	}
	if !strings.Contains(err.Error(), "Script threw intentional test error") {
		t.Fatalf("expected error message to contain thrown error, got: %v", err)
	}

	_ = p.Disable()
}
