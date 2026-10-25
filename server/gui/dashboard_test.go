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
	// Double unsubscribe should not panic
	b.Unsubscribe(ch)
}

func TestDashboardAuthAndCSRF(t *testing.T) {
	d := NewDashboardServer(&mockDashboardServer{}, DashboardOptions{
		AuthToken: "secret123",
	})

	// Test missing auth -> 401
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/commands", bytes.NewBufferString(`{"command": "help"}`))
	w1 := httptest.NewRecorder()
	d.handleCommands(w1, req1)
	if w1.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", w1.Result().StatusCode)
	}

	// Test valid auth but foreign CSRF origin -> 403
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/commands", bytes.NewBufferString(`{"command": "help"}`))
	req2.Header.Set("Authorization", "Bearer secret123")
	req2.Header.Set("Origin", "http://malicious-site.com")
	req2.Host = "127.0.0.1:8080"
	w2 := httptest.NewRecorder()
	d.handleCommands(w2, req2)
	if w2.Result().StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for cross-origin request, got %d", w2.Result().StatusCode)
	}

	// Test valid auth and matching origin -> 200
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/commands", bytes.NewBufferString(`{"command": "help"}`))
	req3.Header.Set("Authorization", "Bearer secret123")
	req3.Header.Set("Origin", "http://127.0.0.1:8080")
	req3.Host = "127.0.0.1:8080"
	w3 := httptest.NewRecorder()
	d.handleCommands(w3, req3)
	if w3.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w3.Result().StatusCode)
	}
}
