package main

import (
	"testing"

	"github.com/df-mc/dragonfly/server"
)

func TestMainLifecycleBootstrap(t *testing.T) {
	tempDir := t.TempDir()
	opts := server.DefaultLifecycleOptions()
	opts.BaseDir = tempDir
	opts.AutoAcceptEULA = true

	srv, err := server.Bootstrap(opts)
	if err != nil {
		t.Fatalf("failed to bootstrap server: %v", err)
	}
	if srv == nil {
		t.Fatal("expected non-nil server instance")
	}
}
