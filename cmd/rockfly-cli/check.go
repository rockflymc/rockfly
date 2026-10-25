package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/df-mc/dragonfly/server/plugin/manifest"
	"github.com/df-mc/dragonfly/server/plugin/ts"
	"github.com/pelletier/go-toml"
)

// CheckPlugin verifies manifest syntax and type-checks the entry point without node/npm.
func CheckPlugin(pluginDir string) (time.Duration, error) {
	start := time.Now()

	manifestPath := filepath.Join(pluginDir, "plugin.toml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return 0, fmt.Errorf("plugin.toml missing or unreadable: %w", err)
	}

	var m manifest.PluginManifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return 0, fmt.Errorf("invalid plugin.toml: %w", err)
	}

	if m.Name == "" {
		return 0, fmt.Errorf("plugin manifest must declare a 'name'")
	}
	if m.Main == "" {
		return 0, fmt.Errorf("plugin manifest must declare a 'main' entry file")
	}
	if filepath.IsAbs(m.Main) || strings.Contains(m.Main, "..") {
		return 0, fmt.Errorf("plugin manifest 'main' must be a relative path inside the plugin directory (got %q)", m.Main)
	}

	mainPath := filepath.Join(pluginDir, m.Main)
	if _, err := os.Stat(mainPath); err != nil {
		if strings.HasSuffix(m.Main, ".wasm") {
			// Check if Go source files exist prior to compilation
			if _, goErr := os.Stat(filepath.Join(pluginDir, "main.go")); goErr == nil {
				return time.Since(start), nil
			}
		}
		return 0, fmt.Errorf("main entry file %q not found: %w", m.Main, err)
	}

	// For TypeScript plugins, run in-memory esbuild validation
	if strings.HasSuffix(m.Main, ".ts") || strings.HasSuffix(m.Main, ".js") {
		transpiler := ts.NewTranspiler()
		_, err := transpiler.Bundle(mainPath)
		if err != nil {
			// Fallback to Transpile
			content, readErr := os.ReadFile(mainPath)
			if readErr != nil {
				return 0, readErr
			}
			if _, transErr := transpiler.Transpile(string(content)); transErr != nil {
				return 0, fmt.Errorf("transpilation check failed: %w", transErr)
			}
		}
	}

	return time.Since(start), nil
}
