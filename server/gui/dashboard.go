package gui

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// Server represents the server capabilities required by the web dashboard.
type Server interface {
	MaxPlayerCount() int
	PlayerCount() int
	Players(tx *world.Tx) iter.Seq[*player.Player]
	PlayerByName(name string) (*world.EntityHandle, bool)
	World() *world.World
	Close() error
}

// DashboardOptions configures the embedded web administration dashboard.
type DashboardOptions struct {
	Addr          string
	MaxLogHistory int
	AuthToken     string
}

// DefaultDashboardOptions returns standard configuration for the dashboard.
func DefaultDashboardOptions() DashboardOptions {
	return DashboardOptions{
		Addr:          "127.0.0.1:8080",
		MaxLogHistory: 200,
		AuthToken:     "",
	}
}

// WebSource implements cmd.Source and cmd.NamedTarget for commands issued via the web console.
type WebSource struct {
	mu       sync.Mutex
	messages []string
	errors   []string
}

// NewWebSource creates a new WebSource.
func NewWebSource() *WebSource {
	return &WebSource{}
}

// Name returns "CONSOLE".
func (s *WebSource) Name() string { return "CONSOLE" }

// IsConsole identifies this source as the console.
func (s *WebSource) IsConsole() bool { return true }

// Position returns origin.
func (s *WebSource) Position() mgl64.Vec3 { return mgl64.Vec3{} }

// SendCommandOutput captures messages and errors returned from command execution.
func (s *WebSource) SendCommandOutput(o *cmd.Output) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, msg := range o.Messages() {
		s.messages = append(s.messages, msg.String())
	}
	for _, err := range o.Errors() {
		s.errors = append(s.errors, err.Error())
	}
}

// Results returns captured messages and errors.
func (s *WebSource) Results() ([]string, []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.messages, s.errors
}

// DashboardServer serves the embedded web dashboard and real-time management endpoints.
type DashboardServer struct {
	srv         Server
	opts        DashboardOptions
	listener    net.Listener
	httpServer  *http.Server
	broadcaster *LogBroadcaster
	startTime   time.Time
}

// NewDashboardServer creates a new DashboardServer instance.
func NewDashboardServer(srv Server, opts DashboardOptions) *DashboardServer {
	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:8080"
	}
	if opts.MaxLogHistory <= 0 {
		opts.MaxLogHistory = 200
	}

	return &DashboardServer{
		srv:         srv,
		opts:        opts,
		broadcaster: NewLogBroadcaster(opts.MaxLogHistory),
		startTime:   time.Now(),
	}
}

// Start binds the HTTP listener and launches the web server.
func (d *DashboardServer) Start() error {
	ln, err := net.Listen("tcp", d.opts.Addr)
	if err != nil {
		return fmt.Errorf("listen dashboard on %s: %w", d.opts.Addr, err)
	}
	d.listener = ln

	mux := http.NewServeMux()
	mux.HandleFunc("/", d.handleIndex)
	mux.HandleFunc("/api/v1/status", d.handleStatus)
	mux.HandleFunc("/api/v1/players", d.handlePlayers)
	mux.HandleFunc("/api/v1/commands", d.handleCommands)
	mux.HandleFunc("/api/v1/stream", d.handleStream)

	d.httpServer = &http.Server{
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // Disable write timeout for SSE streams
	}

	go func() {
		_ = d.httpServer.Serve(ln)
	}()
	return nil
}

// Close gracefully closes the dashboard HTTP server.
func (d *DashboardServer) Close() error {
	if d.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return d.httpServer.Shutdown(ctx)
	}
	return nil
}

// Addr returns the bound network address of the dashboard server.
func (d *DashboardServer) Addr() string {
	if d.listener != nil {
		return d.listener.Addr().String()
	}
	return d.opts.Addr
}

// BroadcastLog forwards a console log line to all connected web dashboard sessions.
func (d *DashboardServer) BroadcastLog(line string) {
	if d.broadcaster != nil {
		d.broadcaster.Broadcast(line)
	}
}

func (d *DashboardServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := Assets.ReadFile("assets/index.html")
	if err != nil {
		http.Error(w, "Asset not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

// StatusPayload represents the live server health payload.
type StatusPayload struct {
	TPS              float64 `json:"tps"`
	MSPT             float64 `json:"mspt"`
	Uptime           string  `json:"uptime"`
	PlayerCount      int     `json:"player_count"`
	MaxPlayers       int     `json:"max_players"`
	MemAlloc         uint64  `json:"mem_alloc"`
	MemSys           uint64  `json:"mem_sys"`
	GCCycles         uint32  `json:"gc_cycles"`
	GoVersion        string  `json:"go_version"`
	OSArch           string  `json:"os_arch"`
	WhitelistEnabled bool    `json:"whitelist_enabled"`
}

// PlayerInfo represents connected player metadata for the dashboard.
type PlayerInfo struct {
	Name    string `json:"name"`
	UUID    string `json:"uuid"`
	Addr    string `json:"addr"`
	Op      bool   `json:"op"`
	OpLevel int    `json:"op_level"`
}

func (d *DashboardServer) getStatusPayload() StatusPayload {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	count := 0
	maxCount := 20
	if d.srv != nil {
		count = d.srv.PlayerCount()
		maxCount = d.srv.MaxPlayerCount()
	}

	uptimeDuration := time.Since(d.startTime).Round(time.Second)

	return StatusPayload{
		TPS:              20.0,
		MSPT:             0.5,
		Uptime:           uptimeDuration.String(),
		PlayerCount:      count,
		MaxPlayers:       maxCount,
		MemAlloc:         m.Alloc,
		MemSys:           m.Sys,
		GCCycles:         m.NumGC,
		GoVersion:        runtime.Version(),
		OSArch:           runtime.GOOS + "/" + runtime.GOARCH,
		WhitelistEnabled: player.WhitelistEnabled(),
	}
}

func (d *DashboardServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(d.getStatusPayload())
}

func (d *DashboardServer) handlePlayers(w http.ResponseWriter, r *http.Request) {
	var list []PlayerInfo
	if d.srv != nil {
		for p := range d.srv.Players(nil) {
			addrStr := ""
			if p.Addr() != nil {
				addrStr = p.Addr().String()
			}
			list = append(list, PlayerInfo{
				Name:    p.Name(),
				UUID:    p.UUID().String(),
				Addr:    addrStr,
				Op:      p.Op(),
				OpLevel: p.OpLevel(),
			})
		}
	}
	if list == nil {
		list = []PlayerInfo{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

type commandRequest struct {
	Command string `json:"command"`
}

type commandResponse struct {
	Messages []string `json:"messages"`
	Errors   []string `json:"errors"`
}

// isAuthorized verifies the request against the configured AuthToken.
func (d *DashboardServer) isAuthorized(r *http.Request) bool {
	if d.opts.AuthToken == "" {
		return true
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") && strings.TrimPrefix(auth, "Bearer ") == d.opts.AuthToken {
		return true
	}
	if r.Header.Get("X-Auth-Token") == d.opts.AuthToken {
		return true
	}
	if r.URL.Query().Get("token") == d.opts.AuthToken {
		return true
	}
	return false
}

// formatSSEData replaces unescaped newlines to preserve SSE protocol boundary integrity.
func formatSSEData(line string) string {
	return strings.ReplaceAll(line, "\n", " ")
}

func (d *DashboardServer) handleCommands(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !d.isAuthorized(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// CSRF Origin verification: reject cross-origin posts
	if origin := r.Header.Get("Origin"); origin != "" {
		if !strings.Contains(origin, r.Host) {
			http.Error(w, "Cross-origin requests forbidden", http.StatusForbidden)
			return
		}
	}

	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		http.Error(w, "Read error", http.StatusBadRequest)
		return
	}

	var req commandRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	line := strings.TrimSpace(req.Command)
	line = strings.TrimPrefix(line, "/")
	if line == "" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(commandResponse{})
		return
	}

	parts := strings.SplitN(line, " ", 2)
	cmdName := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	src := NewWebSource()

	// Built-in special handling
	if cmdName == "stop" || cmdName == "exit" {
		d.BroadcastLog("[Dashboard] Stop command received, shutting down server...")
		go func() {
			time.Sleep(100 * time.Millisecond)
			if d.srv != nil {
				_ = d.srv.Close()
			}
		}()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(commandResponse{
			Messages: []string{"Shutting down server..."},
		})
		return
	}

	command, ok := cmd.ByAlias(cmdName)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(commandResponse{
			Errors: []string{fmt.Sprintf("Unknown command: %s", cmdName)},
		})
		return
	}

	if d.srv != nil && d.srv.World() != nil {
		_, _ = world.Call(context.Background(), d.srv.World(), func(tx *world.Tx) (bool, error) {
			defer func() {
				if rec := recover(); rec != nil {
					src.mu.Lock()
					src.errors = append(src.errors, fmt.Sprintf("panic: %v", rec))
					src.mu.Unlock()
				}
			}()
			command.Execute(args, src, tx)
			return true, nil
		})
	} else {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					src.mu.Lock()
					src.errors = append(src.errors, fmt.Sprintf("panic: %v", rec))
					src.mu.Unlock()
				}
			}()
			command.Execute(args, src, nil)
		}()
	}

	msgs, errs := src.Results()
	for _, m := range msgs {
		d.BroadcastLog(m)
	}
	for _, e := range errs {
		d.BroadcastLog("[ERROR] " + e)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(commandResponse{
		Messages: msgs,
		Errors:   errs,
	})
}

func (d *DashboardServer) handleStream(w http.ResponseWriter, r *http.Request) {
	if !d.isAuthorized(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch, history := d.broadcaster.Subscribe()
	defer d.broadcaster.Unsubscribe(ch)

	// Send initial log history
	for _, h := range history {
		_, _ = fmt.Fprintf(w, "event: log\ndata: %s\n\n", formatSSEData(h))
	}

	// Send initial metrics
	initialMetrics, _ := json.Marshal(d.getStatusPayload())
	_, _ = fmt.Fprintf(w, "event: metrics\ndata: %s\n\n", string(initialMetrics))
	flusher.Flush()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case line, ok := <-ch:
			if !ok {
				return
			}
			_, _ = fmt.Fprintf(w, "event: log\ndata: %s\n\n", formatSSEData(line))
			flusher.Flush()
		case <-ticker.C:
			metricsData, _ := json.Marshal(d.getStatusPayload())
			_, _ = fmt.Fprintf(w, "event: metrics\ndata: %s\n\n", string(metricsData))
			flusher.Flush()
		}
	}
}
