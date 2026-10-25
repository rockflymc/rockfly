package console

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
)

// Server represents the minimal server interface needed by the REPL to execute
// commands and initiate shutdown.
type Server interface {
	World() *world.World
	Close() error
}

// REPL provides an interactive read-eval-print-loop for the server console.
type REPL struct {
	reader      *bufio.Reader
	log         *slog.Logger
	source      Source
	historyPath string
	history     []string
	historyMu   sync.Mutex
	closed      atomic.Bool
	stop        chan struct{}
}

// NewREPL creates a new interactive console REPL.
func NewREPL(r io.Reader, log *slog.Logger, historyPath string) *REPL {
	if r == nil {
		r = os.Stdin
	}
	if log == nil {
		log = slog.Default()
	}
	if historyPath == "" {
		historyPath = ".rockfly_history"
	}

	repl := &REPL{
		reader:      bufio.NewReader(r),
		log:         log,
		source:      NewSource(log),
		historyPath: historyPath,
		stop:        make(chan struct{}),
	}
	repl.loadHistory()
	return repl
}

// loadHistory loads command history from the history file.
func (r *REPL) loadHistory() {
	data, err := os.ReadFile(r.historyPath)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	r.historyMu.Lock()
	defer r.historyMu.Unlock()
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			r.history = append(r.history, l)
		}
	}
}

// saveHistory appends commands to the persistent history file.
func (r *REPL) recordHistory(command string) {
	command = strings.TrimSpace(command)
	if command == "" {
		return
	}

	r.historyMu.Lock()
	defer r.historyMu.Unlock()

	// Avoid duplicate consecutive commands
	if len(r.history) > 0 && r.history[len(r.history)-1] == command {
		return
	}

	r.history = append(r.history, command)
	// Cap in-memory history to 1000 commands
	if len(r.history) > 1000 {
		r.history = r.history[len(r.history)-1000:]
	}

	f, err := os.OpenFile(r.historyPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		_, _ = f.WriteString(command + "\n")
		_ = f.Close()
	}
}

// History returns a copy of the current command history.
func (r *REPL) History() []string {
	r.historyMu.Lock()
	defer r.historyMu.Unlock()
	copied := make([]string, len(r.history))
	copy(copied, r.history)
	return copied
}

// Complete returns auto-completion candidates for the given command prefix.
func (r *REPL) Complete(line string) []string {
	line = strings.TrimPrefix(line, "/")
	line = strings.TrimSpace(line)
	if strings.Contains(line, " ") {
		return nil
	}
	line = strings.ToLower(line)

	candidates := []string{"stop", "exit", "help", "version"}
	for name := range cmd.Commands() {
		candidates = append(candidates, name)
	}

	sort.Strings(candidates)

	var matches []string
	seen := make(map[string]struct{})
	for _, c := range candidates {
		if strings.HasPrefix(strings.ToLower(c), line) {
			if _, ok := seen[c]; !ok {
				seen[c] = struct{}{}
				matches = append(matches, c)
			}
		}
	}
	return matches
}

// ExecuteLine parses and executes a single command line against the given Server.
func (r *REPL) ExecuteLine(line string, srv Server) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	// Strip leading slash if user typed /command
	line = strings.TrimPrefix(line, "/")
	r.recordHistory(line)

	parts := strings.SplitN(line, " ", 2)
	name := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	switch name {
	case "stop", "exit":
		r.log.Info("Stop command received from console, shutting down...")
		if err := srv.Close(); err != nil {
			r.log.Error("error during console stop", "error", err)
		}
		return
	case "help":
		r.printHelp()
		return
	}

	command, ok := cmd.ByAlias(name)
	if !ok {
		r.log.Error(fmt.Sprintf("Unknown command: %q. Type 'help' for a list of available commands.", name))
		return
	}

	if srv.World() == nil {
		r.log.Error("cannot execute command: server world is not yet initialized")
		return
	}

	_, err := world.Call(context.Background(), srv.World(), func(tx *world.Tx) (struct{}, error) {
		defer func() {
			if rec := recover(); rec != nil {
				r.log.Error("panic recovered during console command execution", "panic", fmt.Sprint(rec))
			}
		}()
		command.Execute(args, r.source, tx)
		return struct{}{}, nil
	})
	if err != nil {
		r.log.Error("failed to execute command on world transaction", "error", err)
	}
}

// printHelp lists all available commands in console.
func (r *REPL) printHelp() {
	var names []string
	seen := make(map[string]struct{})
	for name := range cmd.Commands() {
		if _, ok := seen[name]; !ok {
			seen[name] = struct{}{}
			names = append(names, name)
		}
	}
	sort.Strings(names)

	r.log.Info("--- Available Console Commands ---")
	r.log.Info("stop, exit - Gracefully shuts down the server")
	r.log.Info("help - Displays this help message")
	for _, name := range names {
		if command, ok := cmd.ByAlias(name); ok {
			r.log.Info(fmt.Sprintf("/%s - %s", name, command.Description()))
		}
	}
}

// Start launches the interactive readline loop in a background goroutine.
func (r *REPL) Start(srv Server) {
	go func() {
		for {
			if r.closed.Load() {
				return
			}

			line, err := r.reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return
				}
				if r.closed.Load() {
					return
				}
				continue
			}

			r.ExecuteLine(line, srv)
		}
	}()
}

// Close stops the interactive console reader loop.
func (r *REPL) Close() {
	if r.closed.CompareAndSwap(false, true) {
		close(r.stop)
	}
}
