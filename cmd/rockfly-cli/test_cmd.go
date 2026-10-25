package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/df-mc/dragonfly/server/plugin"
)

// TestPlugin loads the target plugin inside an isolated in-memory Rockfly harness.
func TestPlugin(pluginDir string) (time.Duration, error) {
	start := time.Now()

	absPath, err := filepath.Abs(pluginDir)
	if err != nil {
		return 0, err
	}

	// Create temporary sandbox plugins directory
	sandboxDir, err := os.MkdirTemp("", "rockfly_cli_test_*")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(sandboxDir)

	pluginBase := filepath.Base(absPath)
	destDir := filepath.Join(sandboxDir, pluginBase)

	// Copy plugin files to sandbox
	if err := copyDir(absPath, destDir); err != nil {
		return 0, fmt.Errorf("failed to sandbox plugin files: %w", err)
	}

	mgr := plugin.NewPluginManager(sandboxDir)
	if err := mgr.LoadAll(); err != nil {
		return 0, fmt.Errorf("plugin load failed: %w", err)
	}

	if err := mgr.EnableAll(); err != nil {
		return 0, fmt.Errorf("plugin enable failed: %w", err)
	}

	if err := mgr.DisableAll(); err != nil {
		return 0, fmt.Errorf("plugin disable failed: %w", err)
	}

	return time.Since(start), nil
}

func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if e.Name() == "dist" || e.Name() == ".git" {
				continue
			}
			if err := copyDir(s, d); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(s)
			if err != nil {
				return err
			}
			if err := os.WriteFile(d, data, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}
