package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCLI_TypeScriptWorkflow(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rockfly_cli_test_ts_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pluginDir := filepath.Join(tmpDir, "SampleTS")

	// 1. Scaffold TypeScript plugin
	if err := ScaffoldPlugin("SampleTS", "ts", pluginDir); err != nil {
		t.Fatalf("scaffold ts failed: %v", err)
	}

	// Verify generated files
	for _, expectedFile := range []string{"plugin.toml", "index.ts", "tsconfig.json", "rockfly.d.ts"} {
		p := filepath.Join(pluginDir, expectedFile)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Fatalf("expected file %s does not exist", expectedFile)
		}
	}

	// 2. Check plugin
	durCheck, err := CheckPlugin(pluginDir)
	if err != nil {
		t.Fatalf("check ts failed: %v", err)
	}
	t.Logf("TS plugin check passed in %v", durCheck)

	// 3. Build plugin
	outDir := filepath.Join(pluginDir, "dist")
	artifact, durBuild, err := BuildPlugin(pluginDir, outDir)
	if err != nil {
		t.Fatalf("build ts failed: %v", err)
	}
	t.Logf("TS plugin built at %s in %v", artifact, durBuild)

	if _, err := os.Stat(artifact); os.IsNotExist(err) {
		t.Fatalf("built artifact %s not found", artifact)
	}

	// 4. Test plugin in harness
	durTest, err := TestPlugin(pluginDir)
	if err != nil {
		t.Fatalf("test ts plugin failed: %v", err)
	}
	t.Logf("TS plugin tested in harness in %v", durTest)
}

func TestCLI_GoScaffoldWorkflow(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rockfly_cli_test_go_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pluginDir := filepath.Join(tmpDir, "SampleGo")

	// 1. Scaffold Go plugin
	if err := ScaffoldPlugin("SampleGo", "go", pluginDir); err != nil {
		t.Fatalf("scaffold go failed: %v", err)
	}

	for _, expectedFile := range []string{"plugin.toml", "main.go", "go.mod"} {
		p := filepath.Join(pluginDir, expectedFile)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Fatalf("expected file %s does not exist", expectedFile)
		}
	}

	// 2. Check plugin
	durCheck, err := CheckPlugin(pluginDir)
	if err != nil {
		t.Fatalf("check go failed: %v", err)
	}
	t.Logf("Go plugin check passed in %v", durCheck)
}

func TestCLI_SecuritySanitization(t *testing.T) {
	// 1. Invalid plugin names with path traversal or invalid characters
	badNames := []string{"../../escaped", "my/plugin", "plugin\\name", "test*plugin", ""}
	for _, bad := range badNames {
		if err := ScaffoldPlugin(bad, "ts", ""); err == nil {
			t.Errorf("expected error scaffolding plugin with invalid name %q", bad)
		}
	}

	// 2. Manifest with path traversal in 'main'
	tmpDir, err := os.MkdirTemp("", "rockfly_cli_test_sec_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	badManifest := `name = "Malicious"
version = "1.0.0"
main = "../../../etc/passwd"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "plugin.toml"), []byte(badManifest), 0644); err != nil {
		t.Fatalf("failed to write bad manifest: %v", err)
	}

	if _, err := CheckPlugin(tmpDir); err == nil {
		t.Errorf("expected CheckPlugin to reject path traversal in manifest 'main'")
	}

	if _, _, err := BuildPlugin(tmpDir, ""); err == nil {
		t.Errorf("expected BuildPlugin to reject path traversal in manifest 'main'")
	}
}
