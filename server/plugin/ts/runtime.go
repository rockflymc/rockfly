package ts

import (
	"fmt"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/plugin"
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
	mu sync.RWMutex

	manifest  plugin.PluginManifest
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
func NewTSPlugin(manifest plugin.PluginManifest) *TSPlugin {
	vm := goja.New()
	p := &TSPlugin{
		manifest: manifest,
		vm:       vm,
		timeout:  50 * time.Millisecond, // Strict 50ms execution watchdog
	}
	_ = p.bindRockflyAPI()
	return p
}

// Manifest returns the plugin manifest metadata.
func (p *TSPlugin) Manifest() plugin.PluginManifest {
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
	// Execute with watchdog guard
	timer := time.AfterFunc(p.timeout, func() {
		p.vm.Interrupt("plugin load execution timed out")
	})

	_, err := p.vm.RunString(jsSource)
	timer.Stop()

	if err != nil {
		p.vm.ClearInterrupt()
		return fmt.Errorf("failed to run plugin script: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

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
	if p.enabled {
		p.mu.Unlock()
		return nil
	}
	p.enabled = true
	onEnable := p.onEnable
	p.mu.Unlock()

	if onEnable != nil {
		if _, err := p.callWithTimeout(onEnable, goja.Undefined()); err != nil {
			p.mu.Lock()
			p.enabled = false
			p.mu.Unlock()
			return fmt.Errorf("onEnable failed: %w", err)
		}
	}

	return nil
}

// Disable deactivates the plugin, clears all listeners, and calls onDisable.
func (p *TSPlugin) Disable() error {
	p.mu.Lock()
	if !p.enabled {
		p.mu.Unlock()
		return nil
	}
	onDisable := p.onDisable
	p.listeners = nil
	p.eventHandlers = nil
	p.enabled = false
	p.mu.Unlock()

	if onDisable != nil {
		_, _ = p.callWithTimeout(onDisable, goja.Undefined())
	}

	return nil
}

// callWithTimeout executes a JS callable with strict timeout watchdog (vm.Interrupt).
func (p *TSPlugin) callWithTimeout(fn goja.Callable, this goja.Value, args ...goja.Value) (goja.Value, error) {
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
	defer timer.Stop()

	res, err := fn(this, args...)
	if err != nil {
		p.vm.ClearInterrupt()
		return nil, err
	}

	return res, nil
}
