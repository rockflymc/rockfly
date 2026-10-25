package wasm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/plugin"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// WASMPlugin represents an active WebAssembly plugin executing inside Wazero.
type WASMPlugin struct {
	mu sync.RWMutex

	manifest  plugin.PluginManifest
	ctx       context.Context
	cancel    context.CancelFunc
	runtime   wazero.Runtime
	module    api.Module
	onEnable  api.Function
	onDisable api.Function
	enabled   bool

	batchHandler func(mutations []BlockMutation) error
	blockGetter  func(x, y, z int32) uint32
}

// NewWASMPlugin initializes a new Wazero runtime and configures the host environment.
func NewWASMPlugin(ctx context.Context, manifest plugin.PluginManifest) (*WASMPlugin, error) {
	pluginCtx, cancel := context.WithCancel(ctx)
	// Pure-Go Wazero runtime (CGO_ENABLED=0 compliant)
	config := wazero.NewRuntimeConfig()
	rt := wazero.NewRuntimeWithConfig(pluginCtx, config)

	// Instantiate WASI snapshot preview 1 for standard guest runtime support
	_, err := wasi_snapshot_preview1.Instantiate(pluginCtx, rt)
	if err != nil {
		rt.Close(pluginCtx)
		cancel()
		return nil, fmt.Errorf("failed to instantiate WASI snapshot preview 1: %w", err)
	}

	p := &WASMPlugin{
		manifest: manifest,
		ctx:      pluginCtx,
		cancel:   cancel,
		runtime:  rt,
	}

	// Register Rockfly host functions under "rockfly_env"
	if err := p.registerHostABI(pluginCtx); err != nil {
		p.Close()
		return nil, fmt.Errorf("failed to register host ABI: %w", err)
	}

	return p, nil
}

// Manifest returns the plugin's metadata.
func (p *WASMPlugin) Manifest() plugin.PluginManifest {
	return p.manifest
}

// SetBatchHandler sets the callback that applies block mutations atomically in a world transaction.
func (p *WASMPlugin) SetBatchHandler(fn func(mutations []BlockMutation) error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.batchHandler = fn
}

// SetBlockGetter sets the block inquiry callback.
func (p *WASMPlugin) SetBlockGetter(fn func(x, y, z int32) uint32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.blockGetter = fn
}

// Load compiles and instantiates the WASM binary bytes.
func (p *WASMPlugin) Load(wasmBytes []byte) error {
	compiled, err := p.runtime.CompileModule(p.ctx, wasmBytes)
	if err != nil {
		return fmt.Errorf("wasm module compilation failed: %w", err)
	}

	config := wazero.NewModuleConfig().WithName(p.manifest.Name)
	mod, err := p.runtime.InstantiateModule(p.ctx, compiled, config)
	if err != nil {
		return fmt.Errorf("wasm module instantiation failed: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.module = mod

	// Look up lifecycle hooks if exported
	p.onEnable = mod.ExportedFunction("onEnable")
	if p.onEnable == nil {
		p.onEnable = mod.ExportedFunction("on_enable")
	}
	p.onDisable = mod.ExportedFunction("onDisable")
	if p.onDisable == nil {
		p.onDisable = mod.ExportedFunction("on_disable")
	}

	return nil
}

// Enable calls the guest's onEnable hook if exported.
func (p *WASMPlugin) Enable() error {
	p.mu.Lock()
	if p.enabled {
		p.mu.Unlock()
		return nil
	}
	p.enabled = true
	onEnable := p.onEnable
	p.mu.Unlock()

	if onEnable != nil {
		ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
		defer cancel()
		if _, err := onEnable.Call(ctx); err != nil {
			p.mu.Lock()
			p.enabled = false
			p.mu.Unlock()
			return fmt.Errorf("wasm onEnable failed: %w", err)
		}
	}
	return nil
}

// Disable calls the guest's onDisable hook if exported.
func (p *WASMPlugin) Disable() error {
	p.mu.Lock()
	if !p.enabled {
		p.mu.Unlock()
		return nil
	}
	onDisable := p.onDisable
	p.enabled = false
	p.mu.Unlock()

	if onDisable != nil {
		ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
		defer cancel()
		_, _ = onDisable.Call(ctx)
	}
	return nil
}

// Close shuts down the WASM plugin instance and frees the runtime.
func (p *WASMPlugin) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.enabled = false
	bgCtx := context.Background()
	if p.module != nil {
		_ = p.module.Close(bgCtx)
		p.module = nil
	}
	if p.runtime != nil {
		_ = p.runtime.Close(bgCtx)
		p.runtime = nil
	}
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	return nil
}

// Module returns the active Wazero module instance.
func (p *WASMPlugin) Module() api.Module {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.module
}

// CallFunction invokes an exported guest function with the provided arguments.
func (p *WASMPlugin) CallFunction(name string, params ...uint64) ([]uint64, error) {
	return p.CallFunctionWithTimeout(name, 5*time.Second, params...)
}

// CallFunctionWithTimeout invokes an exported guest function with a watchdog deadline.
func (p *WASMPlugin) CallFunctionWithTimeout(name string, timeout time.Duration, params ...uint64) ([]uint64, error) {
	p.mu.RLock()
	mod := p.module
	p.mu.RUnlock()

	if mod == nil {
		return nil, fmt.Errorf("wasm module not loaded")
	}

	fn := mod.ExportedFunction(name)
	if fn == nil {
		return nil, fmt.Errorf("function %q not exported by wasm module", name)
	}

	callCtx, cancel := context.WithTimeout(p.ctx, timeout)
	defer cancel()

	return fn.Call(callCtx, params...)
}
