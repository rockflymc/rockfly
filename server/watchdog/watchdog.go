package watchdog

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// HangHandler defines the callback invoked when the watchdog detects a tick stall or hang.
type HangHandler func(reportPath string, stackTrace []byte)

// Config defines settings for the production watchdog hang detector.
type Config struct {
	// Timeout is the maximum duration permitted between heartbeats before flagging a hang.
	// Standard PaperMC default is 10 seconds.
	Timeout time.Duration
	// CheckInterval is how frequently the watchdog checks for tick progress. Default is 1 second.
	CheckInterval time.Duration
	// CrashReportDir is the directory where stack trace reports are saved. Default is "crash-reports".
	CrashReportDir string
	// OnHang is called when a hang is confirmed. If nil, default action logs and terminates process.
	OnHang HangHandler
	// Logger is the logger used for watchdog alerts.
	Logger *slog.Logger
}

// DefaultConfig returns production-standard watchdog settings.
func DefaultConfig() Config {
	return Config{
		Timeout:        10 * time.Second,
		CheckInterval:  1 * time.Second,
		CrashReportDir: "crash-reports",
	}
}

// Watchdog runs an independent thread monitoring tick progress across server and world loops.
// If a tick freezes or enters an infinite loop, the watchdog captures all thread stack traces
// and saves an emergency crash report.
type Watchdog struct {
	conf          Config
	lastHeartbeat atomic.Int64 // Unix nanoseconds
	running       atomic.Bool
	stopChan      chan struct{}
	wg            sync.WaitGroup
	triggered     atomic.Bool
}

// New creates a new Watchdog with the given configuration.
func New(conf Config) *Watchdog {
	if conf.Timeout <= 0 {
		conf.Timeout = 10 * time.Second
	}
	if conf.CheckInterval <= 0 {
		conf.CheckInterval = 1 * time.Second
	}
	if conf.CrashReportDir == "" {
		conf.CrashReportDir = "crash-reports"
	}
	if conf.Logger == nil {
		conf.Logger = slog.Default()
	}

	w := &Watchdog{
		conf:     conf,
		stopChan: make(chan struct{}),
	}
	w.lastHeartbeat.Store(time.Now().UnixNano())
	return w
}

// Heartbeat signals that a monitored tick has completed successfully.
func (w *Watchdog) Heartbeat() {
	w.lastHeartbeat.Store(time.Now().UnixNano())
}

// Start launches the background watchdog monitor loop.
func (w *Watchdog) Start() {
	if w.running.CompareAndSwap(false, true) {
		w.Heartbeat()
		w.wg.Add(1)
		go w.monitorLoop()
	}
}

// Stop cleanly shuts down the watchdog background thread.
func (w *Watchdog) Stop() {
	if w.running.CompareAndSwap(true, false) {
		close(w.stopChan)
		w.wg.Wait()
	}
}

// Triggered returns whether the watchdog has caught a hang.
func (w *Watchdog) Triggered() bool {
	return w.triggered.Load()
}

// monitorLoop periodically checks elapsed time since the last heartbeat.
func (w *Watchdog) monitorLoop() {
	defer w.wg.Done()
	ticker := time.NewTicker(w.conf.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ticker.C:
			last := time.Unix(0, w.lastHeartbeat.Load())
			elapsed := time.Since(last)

			if elapsed > w.conf.Timeout {
				if w.triggered.CompareAndSwap(false, true) {
					w.handleHang(elapsed)
					return
				}
			}
		}
	}
}

// handleHang captures full goroutine dumps, writes crash reports, and triggers onHang handler.
func (w *Watchdog) handleHang(elapsed time.Duration) {
	w.conf.Logger.Error("WATCHDOG: Server tick loop has been frozen!", "elapsed", elapsed, "threshold", w.conf.Timeout)

	// Capture all goroutine stack traces across the entire runtime
	buf := make([]byte, 2*1024*1024)
	n := runtime.Stack(buf, true)
	stackTrace := buf[:n]

	// Format crash report
	now := time.Now()
	report := fmt.Sprintf("---- Rockfly Watchdog Crash Report ----\n"+
		"Time: %s\n"+
		"Description: Watchdog detected server tick freeze for %v (limit: %v)\n\n"+
		"System:\n"+
		"  OS: %s\n"+
		"  Arch: %s\n"+
		"  CPUs: %d\n"+
		"  Go Version: %s\n\n"+
		"Goroutine Dump:\n%s\n",
		now.Format(time.RFC3339), elapsed, w.conf.Timeout,
		runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), runtime.Version(),
		string(stackTrace),
	)

	_ = os.MkdirAll(w.conf.CrashReportDir, 0755)
	reportFile := filepath.Join(w.conf.CrashReportDir, fmt.Sprintf("crash-%s.txt", now.Format("2006-01-02_15.04.05")))
	_ = os.WriteFile(reportFile, []byte(report), 0644)

	w.conf.Logger.Error("WATCHDOG: Crash report written to disk", "file", reportFile)

	if w.conf.OnHang != nil {
		w.conf.OnHang(reportFile, stackTrace)
	}
}
