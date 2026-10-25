package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/plugin/ts"
	"github.com/df-mc/dragonfly/server/plugin/wasm"
	"github.com/pelletier/go-toml"
)

type pluginEntry struct {
	plugin   Plugin
	manifest PluginManifest
	mainPath string
	isWASM   bool
}

// PluginManager coordinates discovery, DAG dependency resolution, lifecycle, and hot-reload.
type PluginManager struct {
	reloadMu sync.Mutex
	mu       sync.RWMutex

	pluginsDir string
	plugins    map[string]Plugin
	manifests  map[string]PluginManifest
	entries    map[string]*pluginEntry
	order      []string // Topological startup order

	transpiler *ts.Transpiler
	ctx        context.Context
	cancel     context.CancelFunc

	batchHandler func(mutations []wasm.BlockMutation) error
	blockGetter  func(x, y, z int32) uint32
}

// NewPluginManager creates an uninitialized PluginManager for the target plugins directory.
func NewPluginManager(pluginsDir string) *PluginManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &PluginManager{
		pluginsDir: pluginsDir,
		plugins:    make(map[string]Plugin),
		manifests:  make(map[string]PluginManifest),
		entries:    make(map[string]*pluginEntry),
		transpiler: ts.NewTranspiler(),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// SetBatchHandler configures the transaction callback for WASM block mutations.
func (m *PluginManager) SetBatchHandler(fn func(mutations []wasm.BlockMutation) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.batchHandler = fn
}

// SetBlockGetter configures block inquiry for WASM plugins.
func (m *PluginManager) SetBlockGetter(fn func(x, y, z int32) uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blockGetter = fn
}

// Order returns a copy of the topologically sorted plugin execution order.
func (m *PluginManager) Order() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]string, len(m.order))
	copy(res, m.order)
	return res
}

// Plugins returns a snapshot slice of all registered plugins in topological order.
func (m *PluginManager) Plugins() []Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []Plugin
	for _, name := range m.order {
		if p, ok := m.plugins[name]; ok {
			list = append(list, p)
		}
	}
	return list
}

// Plugin returns a registered plugin by name.
func (m *PluginManager) Plugin(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.plugins[name]
	return p, ok
}

// Discover scans the plugins directory and builds candidate manifests.
func (m *PluginManager) Discover() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := os.MkdirAll(m.pluginsDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugins dir: %w", err)
	}

	// Always ensure rockfly.d.ts is up to date in plugins directory
	_ = ts.GenerateDTS(m.pluginsDir)

	entries, err := os.ReadDir(m.pluginsDir)
	if err != nil {
		return fmt.Errorf("failed to read plugins dir: %w", err)
	}

	m.manifests = make(map[string]PluginManifest)
	m.entries = make(map[string]*pluginEntry)

	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || name == "rockfly.d.ts" {
			continue
		}

		fullPath := filepath.Join(m.pluginsDir, name)

		if e.IsDir() {
			// Subdirectory plugin with plugin.toml or package.json
			manifestPath := filepath.Join(fullPath, "plugin.toml")
			var manifest PluginManifest

			if data, err := os.ReadFile(manifestPath); err == nil {
				if err := toml.Unmarshal(data, &manifest); err != nil {
					return fmt.Errorf("invalid plugin.toml in %s: %w", name, err)
				}
			} else {
				// Default manifest if plugin.toml not present
				manifest = PluginManifest{
					Name:    name,
					Version: "1.0.0",
				}
			}

			if manifest.Name == "" {
				manifest.Name = name
			}

			// Locate entry point
			mainFile := manifest.Main
			if mainFile == "" {
				candidates := []string{"index.ts", "index.js", "main.ts", "main.js", "plugin.wasm", "main.wasm"}
				for _, c := range candidates {
					if _, err := os.Stat(filepath.Join(fullPath, c)); err == nil {
						mainFile = c
						break
					}
				}
			}

			if mainFile == "" {
				continue // Skip directories without a recognized entry file
			}

			mainFullPath := filepath.Join(fullPath, mainFile)
			manifest.Main = mainFile
			isWASM := strings.HasSuffix(mainFile, ".wasm")

			m.manifests[manifest.Name] = manifest
			m.entries[manifest.Name] = &pluginEntry{
				manifest: manifest,
				mainPath: mainFullPath,
				isWASM:   isWASM,
			}
		} else {
			// Standalone file in plugins directory (.ts, .js, .wasm)
			ext := strings.ToLower(filepath.Ext(name))
			if ext != ".ts" && ext != ".js" && ext != ".wasm" {
				continue
			}

			baseName := strings.TrimSuffix(name, ext)
			manifest := PluginManifest{
				Name:    baseName,
				Version: "1.0.0",
				Main:    name,
			}

			m.manifests[baseName] = manifest
			m.entries[baseName] = &pluginEntry{
				manifest: manifest,
				mainPath: fullPath,
				isWASM:   ext == ".wasm",
			}
		}
	}

	return nil
}

// LoadAll discovers plugins, resolves the dependency DAG, compiles/transpiles, and initializes plugins.
func (m *PluginManager) LoadAll() error {
	if err := m.Discover(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Compute topological startup order
	order, err := ResolveDependencies(m.manifests)
	if err != nil {
		return err
	}
	m.order = order

	// Instantiate each plugin according to DAG order
	for _, name := range m.order {
		entry := m.entries[name]

		if entry.isWASM {
			// Tier 2: Wazero WebAssembly plugin
			wasmBytes, err := os.ReadFile(entry.mainPath)
			if err != nil {
				return fmt.Errorf("failed to read wasm plugin %s: %w", name, err)
			}

			wp, err := wasm.NewWASMPlugin(m.ctx, entry.manifest)
			if err != nil {
				return fmt.Errorf("failed to init wasm plugin %s: %w", name, err)
			}
			wp.SetBatchHandler(m.batchHandler)
			wp.SetBlockGetter(m.blockGetter)

			if err := wp.Load(wasmBytes); err != nil {
				wp.Close()
				return fmt.Errorf("failed to load wasm plugin %s: %w", name, err)
			}

			entry.plugin = wp
			m.plugins[name] = wp
		} else {
			// Tier 1: Embedded TypeScript / JavaScript plugin via esbuild + goja
			bundledJS, err := m.transpiler.Bundle(entry.mainPath)
			if err != nil {
				// Fallback to Transpile if single file without entry directory
				data, readErr := os.ReadFile(entry.mainPath)
				if readErr != nil {
					return fmt.Errorf("failed to read ts script %s: %w", name, readErr)
				}
				bundledJS, err = m.transpiler.Transpile(string(data))
				if err != nil {
					return fmt.Errorf("failed to bundle/transpile ts plugin %s: %w", name, err)
				}
			}

			tp := ts.NewTSPlugin(entry.manifest)
			if err := tp.Load(bundledJS); err != nil {
				return fmt.Errorf("failed to load ts plugin %s: %w", name, err)
			}

			entry.plugin = tp
			m.plugins[name] = tp
		}
	}

	return nil
}

// EnableAll activates all plugins in topological dependency order.
func (m *PluginManager) EnableAll() error {
	m.mu.RLock()
	order := make([]string, len(m.order))
	copy(order, m.order)
	plugins := make(map[string]Plugin, len(m.plugins))
	for k, v := range m.plugins {
		plugins[k] = v
	}
	m.mu.RUnlock()

	for _, name := range order {
		p, ok := plugins[name]
		if !ok {
			continue
		}
		if err := p.Enable(); err != nil {
			return fmt.Errorf("failed to enable plugin %s: %w", name, err)
		}
	}
	return nil
}

// DisableAll deactivates all active plugins in REVERSE topological order.
func (m *PluginManager) DisableAll() error {
	m.mu.RLock()
	order := make([]string, len(m.order))
	copy(order, m.order)
	plugins := make(map[string]Plugin, len(m.plugins))
	for k, v := range m.plugins {
		plugins[k] = v
	}
	m.mu.RUnlock()

	var firstErr error
	// Disable in reverse order (dependents first, dependencies last)
	for i := len(order) - 1; i >= 0; i-- {
		name := order[i]
		p, ok := plugins[name]
		if !ok {
			continue
		}
		if err := p.Disable(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to disable plugin %s: %w", name, err)
		}
	}

	return firstErr
}

// ReloadAll performs an atomic, live hot-reload: disables active plugins, closes runtimes,
// re-scans the plugins directory, resolves DAG dependencies, and enables the new instances.
func (m *PluginManager) ReloadAll() (int, time.Duration, error) {
	m.reloadMu.Lock()
	defer m.reloadMu.Unlock()

	start := time.Now()

	// 1. Disable active plugins in reverse order
	_ = m.DisableAll()

	m.mu.Lock()
	// 2. Close WASM runtimes if any
	for _, p := range m.plugins {
		if wp, ok := p.(*wasm.WASMPlugin); ok {
			_ = wp.Close()
		}
	}
	m.plugins = make(map[string]Plugin)
	m.manifests = make(map[string]PluginManifest)
	m.entries = make(map[string]*pluginEntry)
	m.order = nil
	m.mu.Unlock()

	// 3. Load all plugins anew
	if err := m.LoadAll(); err != nil {
		return 0, time.Since(start), err
	}

	// 4. Enable all newly loaded plugins
	if err := m.EnableAll(); err != nil {
		return 0, time.Since(start), err
	}

	elapsed := time.Since(start)
	return len(m.Order()), elapsed, nil
}
