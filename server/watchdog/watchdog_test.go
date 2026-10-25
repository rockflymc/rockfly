package watchdog

import (
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWatchdog_NormalHeartbeats(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "watchdog_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	w := New(Config{
		Timeout:        100 * time.Millisecond,
		CheckInterval:  20 * time.Millisecond,
		CrashReportDir: tempDir,
	})
	w.Start()
	defer w.Stop()

	// Send heartbeats every 10ms for 200ms
	for i := 0; i < 20; i++ {
		time.Sleep(10 * time.Millisecond)
		w.Heartbeat()
	}

	if w.Triggered() {
		t.Fatal("expected watchdog to NOT trigger when heartbeats are delivered")
	}
}

func TestWatchdog_HangDetectionAndCrashReport(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "watchdog_test_hang_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	var hangTriggered atomic.Bool
	var reportedPath string
	var reportedDump []byte
	var wg sync.WaitGroup
	wg.Add(1)

	w := New(Config{
		Timeout:        60 * time.Millisecond,
		CheckInterval:  15 * time.Millisecond,
		CrashReportDir: tempDir,
		OnHang: func(reportPath string, stackTrace []byte) {
			hangTriggered.Store(true)
			reportedPath = reportPath
			reportedDump = stackTrace
			wg.Done()
		},
	})

	w.Start()
	defer w.Stop()

	// Initial heartbeat, then stop heartbeats to simulate a tick hang
	w.Heartbeat()

	wg.Wait()

	if !hangTriggered.Load() {
		t.Fatal("expected watchdog to detect hang and trigger OnHang callback")
	}

	if reportedPath == "" {
		t.Fatal("expected crash report path to be set")
	}

	// Verify report file was written on disk
	content, err := os.ReadFile(reportedPath)
	if err != nil {
		t.Fatalf("failed to read generated crash report: %v", err)
	}

	strContent := string(content)
	if !strings.Contains(strContent, "Rockfly Watchdog Crash Report") {
		t.Fatal("expected crash report header in file")
	}
	if !strings.Contains(strContent, "Goroutine Dump:") {
		t.Fatal("expected goroutine dump in report")
	}
	if len(reportedDump) == 0 {
		t.Fatal("expected non-empty stack trace dump")
	}
}

func BenchmarkWatchdog_Heartbeat(b *testing.B) {
	w := New(DefaultConfig())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w.Heartbeat()
	}
}
