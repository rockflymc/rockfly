package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureDirectories(t *testing.T) {
	tempDir := t.TempDir()
	opts := DefaultLifecycleOptions()
	opts.BaseDir = tempDir

	if err := EnsureDirectories(opts); err != nil {
		t.Fatalf("unexpected error ensuring directories: %v", err)
	}

	for _, sub := range []string{"plugins", "world", "logs", "config"} {
		p := filepath.Join(tempDir, sub)
		info, err := os.Stat(p)
		if err != nil {
			t.Errorf("expected directory %s to exist: %v", p, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", p)
		}
	}
}

func TestEnsureEULA(t *testing.T) {
	tempDir := t.TempDir()
	opts := DefaultLifecycleOptions()
	opts.BaseDir = tempDir
	opts.AutoAcceptEULA = true

	agreed, err := EnsureEULA(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !agreed {
		t.Errorf("expected eula to be auto-accepted")
	}

	eulaPath := filepath.Join(tempDir, "eula.txt")
	data, err := os.ReadFile(eulaPath)
	if err != nil {
		t.Fatalf("failed to read eula.txt: %v", err)
	}
	if !strings.Contains(string(data), "eula=true") {
		t.Errorf("expected eula.txt to contain eula=true, got: %s", string(data))
	}

	// Test explicit rejection
	if err := os.WriteFile(eulaPath, []byte("eula=false\n"), 0644); err != nil {
		t.Fatalf("failed to write eula.txt: %v", err)
	}
	agreed, err = EnsureEULA(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agreed {
		t.Errorf("expected eula to NOT be agreed when eula=false")
	}
}

func TestEnsureServerProperties(t *testing.T) {
	tempDir := t.TempDir()
	opts := DefaultLifecycleOptions()
	opts.BaseDir = tempDir

	if err := EnsureServerProperties(opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	propPath := filepath.Join(tempDir, "server.properties")
	data, err := os.ReadFile(propPath)
	if err != nil {
		t.Fatalf("failed to read server.properties: %v", err)
	}
	if !strings.Contains(string(data), "server-port=19132") {
		t.Errorf("expected server.properties to contain server-port=19132, got: %s", string(data))
	}
	if !strings.Contains(string(data), "online-mode=false") {
		t.Errorf("expected server.properties to contain online-mode=false, got: %s", string(data))
	}
}

func TestReadOrCreateConfig(t *testing.T) {
	tempDir := t.TempDir()
	opts := DefaultLifecycleOptions()
	opts.BaseDir = tempDir

	conf, err := ReadOrCreateConfig(opts)
	if err != nil {
		t.Fatalf("unexpected error reading/creating config: %v", err)
	}

	if conf.Name != "Rockfly Server" {
		t.Errorf("expected default server name 'Rockfly Server', got %q", conf.Name)
	}
	if !conf.AuthDisabled {
		t.Errorf("expected AuthDisabled to be true (offline mode default), got false")
	}

	configPath := filepath.Join(tempDir, "config.toml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config.toml: %v", err)
	}
	if !strings.Contains(string(data), "Rockfly Server") {
		t.Errorf("expected config.toml to contain 'Rockfly Server', got: %s", string(data))
	}
}
