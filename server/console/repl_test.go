package console

import (
	"bytes"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/world"
)

type mockServer struct {
	closed bool
}

func (m *mockServer) World() *world.World { return nil }
func (m *mockServer) Close() error {
	m.closed = true
	return nil
}

func TestREPLHistory(t *testing.T) {
	tempDir := t.TempDir()
	histPath := filepath.Join(tempDir, ".test_history")

	r := NewREPL(bytes.NewBuffer(nil), slog.Default(), histPath)
	r.recordHistory("help")
	r.recordHistory("list")
	r.recordHistory("list") // Consecutive duplicate should be ignored

	hist := r.History()
	if len(hist) != 2 {
		t.Fatalf("expected 2 history entries, got %d: %v", len(hist), hist)
	}
	if hist[0] != "help" || hist[1] != "list" {
		t.Errorf("unexpected history: %v", hist)
	}

	// Verify persistence
	r2 := NewREPL(bytes.NewBuffer(nil), slog.Default(), histPath)
	hist2 := r2.History()
	if len(hist2) != 2 || hist2[0] != "help" || hist2[1] != "list" {
		t.Errorf("persisted history mismatch: %v", hist2)
	}
}

func TestREPLComplete(t *testing.T) {
	r := NewREPL(bytes.NewBuffer(nil), slog.Default(), "")

	matches := r.Complete("st")
	found := false
	for _, m := range matches {
		if m == "stop" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'stop' in matches for prefix 'st', got: %v", matches)
	}

	matchesHelp := r.Complete("/he")
	foundHelp := false
	for _, m := range matchesHelp {
		if m == "help" {
			foundHelp = true
			break
		}
	}
	if !foundHelp {
		t.Errorf("expected 'help' in matches for prefix '/he', got: %v", matchesHelp)
	}
}

func TestREPLExecuteStop(t *testing.T) {
	mock := &mockServer{}
	r := NewREPL(bytes.NewBuffer(nil), slog.Default(), "")

	r.ExecuteLine("stop", mock)
	if !mock.closed {
		t.Errorf("expected server to be closed after 'stop' command")
	}

	mock2 := &mockServer{}
	r.ExecuteLine("/exit", mock2)
	if !mock2.closed {
		t.Errorf("expected server to be closed after '/exit' command")
	}
}

func TestREPLInteractiveLoop(t *testing.T) {
	pipeR, pipeW := io.Pipe()
	mock := &mockServer{}
	r := NewREPL(pipeR, slog.Default(), filepath.Join(t.TempDir(), "hist"))

	r.Start(mock)

	_, _ = pipeW.Write([]byte("stop\n"))

	// Give goroutine a moment to read and process
	time.Sleep(50 * time.Millisecond)

	if !mock.closed {
		t.Errorf("expected mock server to be closed via interactive pipe")
	}

	r.Close()
	_ = pipeW.Close()
}
