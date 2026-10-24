package gui

import (
	"bytes"
	"encoding/json"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

type mockDashboardServer struct {
	maxPlayers int
	closed     bool
}

func (m *mockDashboardServer) MaxPlayerCount() int { return m.maxPlayers }
func (m *mockDashboardServer) PlayerCount() int    { return 0 }
func (m *mockDashboardServer) Players(tx *world.Tx) iter.Seq[*player.Player] {
	return func(yield func(*player.Player) bool) {}
}
func (m *mockDashboardServer) PlayerByName(name string) (*world.EntityHandle, bool) {
	return nil, false
}
func (m *mockDashboardServer) World() *world.World { return nil }
func (m *mockDashboardServer) Close() error {
	m.closed = true
	return nil
}

func TestDashboardIndexAsset(t *testing.T) {
	d := NewDashboardServer(&mockDashboardServer{maxPlayers: 20}, DashboardOptions{})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	d.handleIndex(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Rockfly Server Dashboard") {
		t.Errorf("expected dashboard HTML title, got %s", body)
	}
}

func TestDashboardStatusEndpoint(t *testing.T) {
	d := NewDashboardServer(&mockDashboardServer{maxPlayers: 25}, DashboardOptions{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	w := httptest.NewRecorder()

	d.handleStatus(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var status StatusPayload
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("failed to decode JSON status: %v", err)
	}

	if status.MaxPlayers != 25 {
		t.Errorf("expected max players 25, got %d", status.MaxPlayers)
	}
	if status.TPS != 20.0 {
		t.Errorf("expected 20.0 TPS, got %v", status.TPS)
	}
}

type testDummyCommand struct{}

func (testDummyCommand) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	o.Printf("Hello from dummy command!")
}

func TestDashboardCommandExecution(t *testing.T) {
	cmd.Register(cmd.New("dummytest", "Test command", nil, testDummyCommand{}))

	d := NewDashboardServer(&mockDashboardServer{}, DashboardOptions{})

	payload := `{"command": "dummytest"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/commands", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	d.handleCommands(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var res commandResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode command response: %v", err)
	}

	if len(res.Messages) == 0 || !strings.Contains(res.Messages[0], "Hello from dummy command!") {
		t.Errorf("expected message from dummy command, got %v", res.Messages)
	}
}

func TestDashboardBroadcaster(t *testing.T) {
	b := NewLogBroadcaster(10)
	ch, hist := b.Subscribe()
	if len(hist) != 0 {
		t.Errorf("expected empty initial history")
	}

	b.Broadcast("Line 1")
	b.Broadcast("Line 2")

	line1 := <-ch
	line2 := <-ch
	if line1 != "Line 1" || line2 != "Line 2" {
		t.Errorf("expected broadcast lines, got %q, %q", line1, line2)
	}

	b.Unsubscribe(ch)
}
