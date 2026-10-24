package admin

import (
	"bytes"
	"iter"
	"log/slog"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/console"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

type mockAdminServer struct {
	maxPlayers int
	players    []*player.Player
}

func (m *mockAdminServer) MaxPlayerCount() int {
	return m.maxPlayers
}

func (m *mockAdminServer) PlayerCount() int {
	return len(m.players)
}

func (m *mockAdminServer) Players(tx *world.Tx) iter.Seq[*player.Player] {
	return func(yield func(*player.Player) bool) {
		for _, p := range m.players {
			if !yield(p) {
				break
			}
		}
	}
}

func (m *mockAdminServer) PlayerByName(name string) (*world.EntityHandle, bool) {
	return nil, false
}

type testCaptureSource struct {
	name     string
	messages []string
	errors   []string
}

func (s *testCaptureSource) Name() string {
	return s.name
}

func (s *testCaptureSource) Position() mgl64.Vec3 {
	return mgl64.Vec3{}
}

func (s *testCaptureSource) SendCommandOutput(o *cmd.Output) {
	for _, m := range o.Messages() {
		s.messages = append(s.messages, m.String())
	}
	for _, e := range o.Errors() {
		s.errors = append(s.errors, e.Error())
	}
}

func TestAdminCommandsRegistration(t *testing.T) {
	srv := &mockAdminServer{maxPlayers: 20}
	RegisterAll(srv)

	expected := []string{"op", "deop", "kick", "ban", "pardon", "ban-ip", "pardon-ip", "list", "say", "version"}
	commands := cmd.Commands()

	for _, name := range expected {
		if _, ok := commands[name]; !ok {
			t.Errorf("expected command /%s to be registered", name)
		}
	}
}

func TestVersionCommand(t *testing.T) {
	c, ok := cmd.ByAlias("version")
	if !ok {
		t.Fatalf("version command not found")
	}

	src := &testCaptureSource{name: "CONSOLE"}
	c.Execute("", src, nil)

	if len(src.messages) == 0 {
		t.Fatalf("expected output from /version")
	}
	if !strings.Contains(src.messages[0], "Rockfly version") {
		t.Errorf("expected message to mention Rockfly version, got %q", src.messages[0])
	}
}

func TestOpAndDeopCommands(t *testing.T) {
	opCmd, ok := cmd.ByAlias("op")
	if !ok {
		t.Fatalf("op command not found")
	}

	consoleSrc := console.NewSource(slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)))
	opCmd.Execute("Alex 4", consoleSrc, nil)

	if !player.IsOp("Alex") {
		t.Errorf("expected Alex to be operator after /op")
	}

	deopCmd, ok := cmd.ByAlias("deop")
	if !ok {
		t.Fatalf("deop command not found")
	}

	deopCmd.Execute("Alex", consoleSrc, nil)
	if player.IsOp("Alex") {
		t.Errorf("expected Alex to no longer be operator after /deop")
	}
}

func TestBanAndPardonCommands(t *testing.T) {
	banCmd, ok := cmd.ByAlias("ban")
	if !ok {
		t.Fatalf("ban command not found")
	}

	consoleSrc := console.NewSource(slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)))
	banCmd.Execute("Griefer99 Breaking rules", consoleSrc, nil)

	banned, reason := player.DefaultBanManager().IsPlayerBanned("Griefer99")
	if !banned || reason != "Breaking rules" {
		t.Errorf("expected Griefer99 to be banned with reason, got %v (%q)", banned, reason)
	}

	pardonCmd, ok := cmd.ByAlias("pardon")
	if !ok {
		t.Fatalf("pardon command not found")
	}

	pardonCmd.Execute("Griefer99", consoleSrc, nil)
	banned, _ = player.DefaultBanManager().IsPlayerBanned("Griefer99")
	if banned {
		t.Errorf("expected Griefer99 to be unbanned after /pardon")
	}
}

func TestBanIPAndPardonIPCommands(t *testing.T) {
	banIPCmd, ok := cmd.ByAlias("ban-ip")
	if !ok {
		t.Fatalf("ban-ip command not found")
	}

	consoleSrc := console.NewSource(slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)))
	banIPCmd.Execute("10.0.0.99 DDOS attempt", consoleSrc, nil)

	banned, reason := player.DefaultBanManager().IsIPBanned("10.0.0.99")
	if !banned || reason != "DDOS attempt" {
		t.Errorf("expected IP 10.0.0.99 to be banned, got %v (%q)", banned, reason)
	}

	pardonIPCmd, ok := cmd.ByAlias("pardon-ip")
	if !ok {
		t.Fatalf("pardon-ip command not found")
	}

	pardonIPCmd.Execute("10.0.0.99", consoleSrc, nil)
	banned, _ = player.DefaultBanManager().IsIPBanned("10.0.0.99")
	if banned {
		t.Errorf("expected IP 10.0.0.99 to be unbanned after /pardon-ip")
	}
}

func TestListCommand(t *testing.T) {
	listCmd, ok := cmd.ByAlias("list")
	if !ok {
		t.Fatalf("list command not found")
	}

	src := &testCaptureSource{name: "CONSOLE"}
	listCmd.Execute("", src, nil)

	if len(src.messages) == 0 {
		t.Fatalf("expected message from /list")
	}
	if !strings.Contains(src.messages[0], "players online") {
		t.Errorf("expected output to mention players online, got %q", src.messages[0])
	}
}

func TestSayCommand(t *testing.T) {
	sayCmd, ok := cmd.ByAlias("say")
	if !ok {
		t.Fatalf("say command not found")
	}

	src := &testCaptureSource{name: "CONSOLE"}
	sayCmd.Execute("Server restart in 5 minutes", src, nil)

	if len(src.messages) == 0 {
		t.Fatalf("expected output from /say")
	}
	if !strings.Contains(src.messages[0], "[Server] Server restart in 5 minutes") {
		t.Errorf("expected formatted broadcast, got %q", src.messages[0])
	}
}

func TestAdminPermissionsCheck(t *testing.T) {
	regularUser := &testCaptureSource{name: "RegularPlayer"}
	if isAllowed(regularUser) {
		t.Errorf("regular user should not be allowed admin commands")
	}

	consoleSrc := console.NewSource(slog.Default())
	if !isAllowed(consoleSrc) {
		t.Errorf("console source should be allowed admin commands")
	}

	_ = player.DefaultOperatorManager().SetOp("OpPlayer", "", 4, false)
	opUser := &testCaptureSource{name: "OpPlayer"}
	if !isAllowed(opUser) {
		t.Errorf("operator user should be allowed admin commands")
	}
}

func TestWhitelistCommand(t *testing.T) {
	wlCmd, ok := cmd.ByAlias("whitelist")
	if !ok {
		t.Fatalf("whitelist command not found")
	}

	consoleSrc := console.NewSource(slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)))

	// Test on
	wlCmd.Execute("on", consoleSrc, nil)
	if !player.WhitelistEnabled() {
		t.Errorf("expected whitelist to be enabled after /whitelist on")
	}

	// Test add
	wlCmd.Execute("add Notch", consoleSrc, nil)
	if !player.DefaultWhitelistManager().IsWhitelisted("Notch", "") {
		t.Errorf("expected Notch to be whitelisted after /whitelist add Notch")
	}

	// Test list
	src := &testCaptureSource{name: "CONSOLE"}
	wlCmd.Execute("list", src, nil)
	if len(src.messages) == 0 || !strings.Contains(src.messages[0], "Notch") {
		t.Errorf("expected whitelist list output to contain Notch, got %v", src.messages)
	}

	// Test remove
	wlCmd.Execute("remove Notch", consoleSrc, nil)
	if player.DefaultWhitelistManager().IsWhitelisted("Notch", "") {
		t.Errorf("expected Notch to be removed after /whitelist remove Notch")
	}

	// Test off
	wlCmd.Execute("off", consoleSrc, nil)
	if player.WhitelistEnabled() {
		t.Errorf("expected whitelist to be disabled after /whitelist off")
	}
}
