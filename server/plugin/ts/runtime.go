package ts

import (
	"fmt"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/plugin/manifest"
	"github.com/dop251/goja"
)

type tsEventListener struct {
	eventName string
	priority  event.Priority
	callback  goja.Callable
	plugin    *TSPlugin
}

// TSPlugin represents an active TypeScript/JavaScript plugin executing on the Go heap.
type TSPlugin struct {
	mu sync.Mutex

	manifest  manifest.PluginManifest
	vm        *goja.Runtime
	onEnable  goja.Callable
	onDisable goja.Callable

	listeners     []func()
	eventHandlers []*tsEventListener
	commands      []string

	timeout time.Duration
	enabled bool
}

// NewTSPlugin initializes a new TSPlugin with reasonable defaults.
func NewTSPlugin(m manifest.PluginManifest) *TSPlugin {
	vm := goja.New()
	p := &TSPlugin{
		manifest: m,
		vm:       vm,
		timeout:  50 * time.Millisecond, // Strict 50ms execution watchdog
	}
	_ = p.bindRockflyAPI()
	return p
}

// Manifest returns the plugin manifest metadata.
func (p *TSPlugin) Manifest() manifest.PluginManifest {
	return p.manifest
}

// SetTimeout sets the execution timeout per callback to prevent infinite loops.
func (p *TSPlugin) SetTimeout(d time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.timeout = d
}

// Load evaluates the bundled JavaScript source code and extracts lifecycle hooks.
func (p *TSPlugin) Load(jsSource string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Execute with watchdog guard
	timer := time.AfterFunc(p.timeout, func() {
		p.vm.Interrupt("plugin load execution timed out")
	})
	defer func() {
		timer.Stop()
		p.vm.ClearInterrupt()
	}()

	_, err := p.vm.RunString(jsSource)
	if err != nil {
		return fmt.Errorf("failed to run plugin script: %w", err)
	}

	// Extract lifecycle hooks if exported
	if fn, ok := goja.AssertFunction(p.vm.Get("onEnable")); ok {
		p.onEnable = fn
	}
	if fn, ok := goja.AssertFunction(p.vm.Get("onDisable")); ok {
		p.onDisable = fn
	}

	return nil
}

// Enable activates the plugin and calls onEnable.
func (p *TSPlugin) Enable() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.enabled {
		return nil
	}
	p.enabled = true

	if p.onEnable != nil {
		if _, err := p.callWithTimeoutLocked(p.onEnable, goja.Undefined()); err != nil {
			p.enabled = false
			return fmt.Errorf("onEnable failed: %w", err)
		}
	}

	return nil
}

// Disable deactivates the plugin, clears all listeners, and calls onDisable.
func (p *TSPlugin) Disable() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.enabled {
		return nil
	}

	// Clean up event listeners
	for _, unsub := range p.listeners {
		if unsub != nil {
			unsub()
		}
	}
	p.listeners = nil
	p.eventHandlers = nil
	p.enabled = false

	if p.onDisable != nil {
		_, _ = p.callWithTimeoutLocked(p.onDisable, goja.Undefined())
	}

	return nil
}

// callWithTimeoutLocked executes a JS callable with strict timeout watchdog and panic recovery.
// MUST be called with p.mu held.
func (p *TSPlugin) callWithTimeoutLocked(fn goja.Callable, this goja.Value, args ...goja.Value) (res goja.Value, err error) {
	if fn == nil {
		return goja.Undefined(), nil
	}

	timeout := p.timeout
	if timeout <= 0 {
		timeout = 50 * time.Millisecond
	}

	timer := time.AfterFunc(timeout, func() {
		p.vm.Interrupt("script execution watchdog timeout exceeded")
	})
	defer func() {
		timer.Stop()
		p.vm.ClearInterrupt()
		if r := recover(); r != nil {
			err = fmt.Errorf("plugin execution panic: %v", r)
		}
	}()

	return fn(this, args...)
}

// callWithTimeout executes a JS callable with strict timeout watchdog and acquires p.mu.
func (p *TSPlugin) callWithTimeout(fn goja.Callable, this goja.Value, args ...goja.Value) (goja.Value, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.callWithTimeoutLocked(fn, this, args...)
}

// VM returns the underlying Goja runtime for inspection or evaluation.
func (p *TSPlugin) VM() *goja.Runtime {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.vm
}

// Eval executes a JavaScript expression on the plugin's runtime.
func (p *TSPlugin) Eval(src string) (goja.Value, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.vm.RunString(src)
}
