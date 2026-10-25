package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/df-mc/dragonfly/server/plugin/manifest"
	"github.com/df-mc/dragonfly/server/plugin/ts"
	"github.com/pelletier/go-toml"
)

// BuildPlugin bundles a TypeScript plugin or compiles a Go WASM plugin into a distribution folder.
func BuildPlugin(pluginDir, outDir string) (string, time.Duration, error) {
	start := time.Now()

	manifestPath := filepath.Join(pluginDir, "plugin.toml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", 0, fmt.Errorf("plugin.toml missing or unreadable: %w", err)
	}

	var m manifest.PluginManifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return "", 0, fmt.Errorf("invalid plugin.toml: %w", err)
	}

	pluginDir = filepath.Clean(pluginDir)
	if filepath.IsAbs(m.Main) || strings.Contains(m.Main, "..") {
		return "", 0, fmt.Errorf("plugin manifest 'main' must be a relative path inside the plugin directory (got %q)", m.Main)
	}

	if outDir == "" {
		outDir = filepath.Join(pluginDir, "dist")
	}
	outDir = filepath.Clean(outDir)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create output dir %s: %w", outDir, err)
	}

	mainPath := filepath.Join(pluginDir, m.Main)
	isTS := strings.HasSuffix(m.Main, ".ts") || strings.HasSuffix(m.Main, ".js")
	isWASM := strings.HasSuffix(m.Main, ".wasm")

	var artifactName string

	switch {
	case isTS:
		transpiler := ts.NewTranspiler()
		bundledJS, err := transpiler.Bundle(mainPath)
		if err != nil {
			content, readErr := os.ReadFile(mainPath)
			if readErr != nil {
				return "", 0, readErr
			}
			bundledJS, err = transpiler.Transpile(string(content))
			if err != nil {
				return "", 0, fmt.Errorf("bundle failed: %w", err)
			}
		}

		artifactName = "index.js"
		if err := os.WriteFile(filepath.Join(outDir, artifactName), []byte(bundledJS), 0644); err != nil {
			return "", 0, err
		}

		// Write updated plugin.toml
		distManifest := m
		distManifest.Main = artifactName
		distToml, _ := toml.Marshal(distManifest)
		_ = os.WriteFile(filepath.Join(outDir, "plugin.toml"), distToml, 0644)
	case isWASM:
		// Pure Go wasip1 compilation
		artifactName = "plugin.wasm"
		outArtifact, absErr := filepath.Abs(filepath.Join(outDir, artifactName))
		if absErr != nil {
			return "", 0, absErr
		}

		cmd := exec.Command("go", "build", "-o", outArtifact, ".")
		cmd.Dir = pluginDir
		cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm", "CGO_ENABLED=0")
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", 0, fmt.Errorf("go wasip1 build failed: %w\n%s", err, string(out))
		}

		distManifest := m
		distManifest.Main = artifactName
		distToml, _ := toml.Marshal(distManifest)
		_ = os.WriteFile(filepath.Join(outDir, "plugin.toml"), distToml, 0644)
	default:
		return "", 0, fmt.Errorf("unrecognized plugin type: %s", m.Main)
	}

	return filepath.Join(outDir, artifactName), time.Since(start), nil
}
