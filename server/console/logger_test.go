package console

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
)

type nopWriteCloser struct {
	io.Writer
}

func (n nopWriteCloser) Close() error {
	return nil
}

func TestStripANSI(t *testing.T) {
	colored := "\033[31mError message\033[0m with \033[36mcyan\033[0m"
	expected := "Error message with cyan"
	got := StripANSI(colored)
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestConsoleHandlerFormatting(t *testing.T) {
	var stdout bytes.Buffer
	var fileOut bytes.Buffer

	h := &Handler{
		stdout:     &stdout,
		fileWriter: nopWriteCloser{&fileOut},
		level:      slog.LevelDebug,
		mu:         &sync.Mutex{},
	}

	record := slog.NewRecord(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC), slog.LevelInfo, "Server started successfully", 0)
	record.AddAttrs(slog.Int("port", 19132))

	if err := h.Handle(context.Background(), record); err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}

	stdoutStr := stdout.String()
	fileStr := fileOut.String()

	// Stdout should contain ANSI formatting
	if !strings.Contains(stdoutStr, "[12:00:00]") {
		t.Errorf("expected timestamp in stdout, got: %s", stdoutStr)
	}
	if !strings.Contains(stdoutStr, "[INFO]") {
		t.Errorf("expected level in stdout, got: %s", stdoutStr)
	}
	if !strings.Contains(stdoutStr, "port=19132") {
		t.Errorf("expected attr in stdout, got: %s", stdoutStr)
	}

	// File output should be clean text
	if strings.Contains(fileStr, "\033[") {
		t.Errorf("expected file output to have no ANSI escapes, got: %s", fileStr)
	}
	if !strings.Contains(fileStr, "[12:00:00] [INFO] Server started successfully port=19132") {
		t.Errorf("unexpected file output: %s", fileStr)
	}
}

type safeBuffer struct {
	bytes.Buffer
	mu sync.Mutex
}

func (s *safeBuffer) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Buffer.Write(p)
}

func (s *safeBuffer) Close() error {
	return nil
}

func TestConsoleHandlerConcurrentWithAttrs(t *testing.T) {
	var stdout safeBuffer
	var fileOut safeBuffer

	h := &Handler{
		stdout:     &stdout,
		fileWriter: &fileOut,
		level:      slog.LevelDebug,
		mu:         &sync.Mutex{},
	}
	logger := slog.New(h)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			subLogger := logger.With("worker", workerID)
			for j := 0; j < 50; j++ {
				subLogger.Info("test message", "iteration", j)
			}
		}(i)
	}
	wg.Wait()

	if err := h.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
}
