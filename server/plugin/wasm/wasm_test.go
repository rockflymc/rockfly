package wasm

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/plugin"
)

func TestDecodeBlockBatch_BoundsChecking(t *testing.T) {
	mem := make([]byte, 1024)

	// Valid batch
	muts := []BlockMutation{
		{X: 10, Y: 64, Z: -20, RuntimeID: 1},
		{X: 11, Y: 65, Z: -19, RuntimeID: 2},
	}
	if err := EncodeBlockBatch(mem, 64, muts); err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	decoded, err := DecodeBlockBatch(mem, 64, 2)
	if err != nil {
		t.Fatalf("decode valid batch failed: %v", err)
	}
	if len(decoded) != 2 {
		t.Fatalf("expected 2 mutations, got %d", len(decoded))
	}
	if decoded[0] != muts[0] || decoded[1] != muts[1] {
		t.Fatalf("decoded mismatch: got %+v, expected %+v", decoded, muts)
	}

	// Out of bounds count
	_, err = DecodeBlockBatch(mem, 1000, 10) // 1000 + 160 = 1160 > 1024
	if err == nil {
		t.Fatalf("expected ErrOutOfBounds for buffer overflow")
	}

	// Exceed max batch limit
	_, err = DecodeBlockBatch(mem, 0, MaxBatchSize+1)
	if err == nil {
		t.Fatalf("expected ErrBatchTooLarge")
	}

	// Zero count
	zeroMuts, err := DecodeBlockBatch(mem, 0, 0)
	if err != nil || len(zeroMuts) != 0 {
		t.Fatalf("expected nil for 0 count")
	}
}

func TestWASMPlugin_10kBlockMutationsGoGuest(t *testing.T) {
	// Create temporary directory for Go wasip1 guest
	tmpDir, err := os.MkdirTemp("", "rockfly_wasm_guest_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	guestSrc := `package main

import (
	"unsafe"
)

//go:wasmimport rockfly_env rockfly_commit_block_batch
func rockfly_commit_block_batch(ptr uint32, count uint32) uint32

//go:wasmimport rockfly_env rockfly_log
func rockfly_log(level uint32, ptr uint32, length uint32)

//go:wasmimport rockfly_env rockfly_get_block
func rockfly_get_block(x int32, y int32, z int32) uint32

type BlockMutation struct {
	X         int32
	Y         int32
	Z         int32
	RuntimeID uint32
}

func logMessage(msg string) {
	if len(msg) == 0 {
		return
	}
	ptr := uint32(uintptr(unsafe.Pointer(unsafe.StringData(msg))))
	rockfly_log(0, ptr, uint32(len(msg)))
}

func main() {
	logMessage("WASM Guest starting 10k batch block generation...")

	// Query host for existing block at (0, 64, 0)
	_ = rockfly_get_block(0, 64, 0)

	const count = 10000
	mutations := make([]BlockMutation, count)
	for i := 0; i < count; i++ {
		mutations[i] = BlockMutation{
			X:         int32(i),
			Y:         int32(64 + (i % 32)),
			Z:         int32(i % 16),
			RuntimeID: uint32(100 + (i % 10)),
		}
	}

	ptr := uint32(uintptr(unsafe.Pointer(&mutations[0])))
	committed := rockfly_commit_block_batch(ptr, count)
	if committed == count {
		logMessage("WASM Guest successfully committed all 10k blocks!")
	}
}
`
	srcFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(srcFile, []byte(guestSrc), 0644); err != nil {
		t.Fatalf("failed to write guest src: %v", err)
	}

	wasmFile := filepath.Join(tmpDir, "plugin.wasm")
	buildCmd := exec.Command("go", "build", "-o", wasmFile, srcFile)
	buildCmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm", "CGO_ENABLED=0")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to compile wasm guest: %v\nOutput: %s", err, string(out))
	}

	wasmBytes, err := os.ReadFile(wasmFile)
	if err != nil {
		t.Fatalf("failed to read compiled wasm: %v", err)
	}

	ctx := context.Background()
	manifest := plugin.PluginManifest{
		Name:    "WasmWorldGen",
		Version: "1.0.0",
		Author:  "RockflyTeam",
		Main:    "plugin.wasm",
	}

	wasmPlugin, err := NewWASMPlugin(ctx, manifest)
	if err != nil {
		t.Fatalf("failed to init wasm plugin: %v", err)
	}
	defer wasmPlugin.Close()

	var receivedCount int
	var decodeDuration time.Duration

	wasmPlugin.SetBlockGetter(func(x, y, z int32) uint32 {
		return 42 // Mock bedrock runtime ID
	})

	wasmPlugin.SetBatchHandler(func(mutations []BlockMutation) error {
		start := time.Now()
		receivedCount = len(mutations)
		decodeDuration = time.Since(start)

		// Verify contents
		for i, m := range mutations {
			expectedX := int32(i)
			expectedY := int32(64 + (i % 32))
			expectedZ := int32(i % 16)
			expectedID := uint32(100 + (i % 10))

			if m.X != expectedX || m.Y != expectedY || m.Z != expectedZ || m.RuntimeID != expectedID {
				t.Errorf("mutation[%d] mismatch: got %+v", i, m)
				break
			}
		}
		return nil
	})

	// Load and run WASM module
	startTotal := time.Now()
	if err := wasmPlugin.Load(wasmBytes); err != nil {
		t.Fatalf("failed to load wasm module: %v", err)
	}
	totalElapsed := time.Since(startTotal)

	if receivedCount != 10000 {
		t.Fatalf("expected 10000 mutations, received %d", receivedCount)
	}

	t.Logf("Successfully committed %d blocks in single transaction (validation: %v, total load+run: %v)",
		receivedCount, decodeDuration, totalElapsed)

	if decodeDuration > 10*time.Millisecond {
		t.Errorf("batch decode/verification took too long: %v", decodeDuration)
	}

	if err := wasmPlugin.Enable(); err != nil {
		t.Fatalf("enable failed: %v", err)
	}
	if err := wasmPlugin.Disable(); err != nil {
		t.Fatalf("disable failed: %v", err)
	}
}
