package console

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// ANSI color escape codes for terminal formatting.
const (
	ansiReset   = "\033[0m"
	ansiGray    = "\033[90m"
	ansiCyan    = "\033[36m"
	ansiYellow  = "\033[33m"
	ansiRed     = "\033[31m"
	ansiMagenta = "\033[35m"
	ansiGreen   = "\033[32m"
	ansiBold    = "\033[1m"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// StripANSI removes all ANSI escape sequences from a string.
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// Handler is a high-throughput, colorized slog.Handler designed for Minecraft server consoles.
// It outputs ANSI color-formatted logs to stdout and clean, plain text logs to logs/latest.log.
type Handler struct {
	stdout     io.Writer
	fileWriter io.Writer
	level      slog.Level
	mu         sync.Mutex
	attrs      []slog.Attr
	groups     []string
}

// HandlerOptions configures the ConsoleHandler.
type HandlerOptions struct {
	Level   slog.Level
	Stdout  io.Writer
	LogsDir string
	LogFile string
}

// NewHandler creates a new ConsoleHandler.
func NewHandler(opts HandlerOptions) *Handler {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.LogsDir == "" {
		opts.LogsDir = "logs"
	}
	if opts.LogFile == "" {
		opts.LogFile = "latest.log"
	}

	var fileWriter io.Writer
	_ = os.MkdirAll(opts.LogsDir, 0755)
	logPath := filepath.Join(opts.LogsDir, opts.LogFile)
	if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fileWriter = f
	}

	return &Handler{
		stdout:     opts.Stdout,
		fileWriter: fileWriter,
		level:      opts.Level,
	}
}

// Enabled reports whether the handler handles records at the given level.
func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle formats and outputs a log record.
func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	timestamp := r.Time.Format("15:04:05")

	var levelBadge string
	var levelPlain string
	switch {
	case r.Level < slog.LevelInfo:
		levelBadge = fmt.Sprintf("%s[DEBUG]%s", ansiMagenta, ansiReset)
		levelPlain = "[DEBUG]"
	case r.Level < slog.LevelWarn:
		levelBadge = fmt.Sprintf("%s[INFO]%s", ansiCyan, ansiReset)
		levelPlain = "[INFO]"
	case r.Level < slog.LevelError:
		levelBadge = fmt.Sprintf("%s[WARN]%s", ansiYellow, ansiReset)
		levelPlain = "[WARN]"
	default:
		levelBadge = fmt.Sprintf("%s[ERROR]%s", ansiRed, ansiReset)
		levelPlain = "[ERROR]"
	}

	var attrParts []string
	for _, a := range h.attrs {
		attrParts = append(attrParts, fmt.Sprintf("%s=%v", a.Key, a.Value.Any()))
	}
	r.Attrs(func(a slog.Attr) bool {
		attrParts = append(attrParts, fmt.Sprintf("%s=%v", a.Key, a.Value.Any()))
		return true
	})

	attrStr := ""
	if len(attrParts) > 0 {
		attrStr = " " + strings.Join(attrParts, " ")
	}

	// Console line with ANSI colors
	consoleLine := fmt.Sprintf("%s[%s]%s %s %s%s\n", ansiGray, timestamp, ansiReset, levelBadge, r.Message, attrStr)

	// File line (plain text, stripped of ANSI)
	fileLine := fmt.Sprintf("[%s] %s %s%s\n", timestamp, levelPlain, StripANSI(r.Message), attrStr)

	h.mu.Lock()
	defer h.mu.Unlock()

	_, _ = h.stdout.Write([]byte(consoleLine))
	if h.fileWriter != nil {
		_, _ = h.fileWriter.Write([]byte(fileLine))
	}
	return nil
}

// WithAttrs returns a new Handler with the given attributes added.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()

	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &Handler{
		stdout:     h.stdout,
		fileWriter: h.fileWriter,
		level:      h.level,
		attrs:      newAttrs,
		groups:     h.groups,
	}
}

// WithGroup returns a new Handler with the given group name appended.
func (h *Handler) WithGroup(name string) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()

	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name

	return &Handler{
		stdout:     h.stdout,
		fileWriter: h.fileWriter,
		level:      h.level,
		attrs:      h.attrs,
		groups:     newGroups,
	}
}
