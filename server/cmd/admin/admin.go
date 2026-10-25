package admin

import (
	"iter"
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// Server represents the server capabilities required by administrative commands.
type Server interface {
	MaxPlayerCount() int
	PlayerCount() int
	Players(tx *world.Tx) iter.Seq[*player.Player]
	PlayerByName(name string) (*world.EntityHandle, bool)
}

// sourceName extracts the readable name of a command source, defaulting to "Server".
func sourceName(src cmd.Source) string {
	if src == nil {
		return "Server"
	}
	if nt, ok := src.(cmd.NamedTarget); ok {
		name := nt.Name()
		if strings.EqualFold(name, "CONSOLE") {
			return "Server"
		}
		return name
	}
	return "Server"
}

// isAllowed checks if a command source is permitted to execute administrative commands.
func isAllowed(src cmd.Source) bool {
	if src == nil {
		return false
	}
	if c, ok := src.(interface{ IsConsole() bool }); ok && c.IsConsole() {
		return true
	}
	if nt, ok := src.(cmd.NamedTarget); ok && strings.EqualFold(nt.Name(), "CONSOLE") {
		return true
	}
	if p, ok := src.(*player.Player); ok {
		return p.Op()
	}
	if nt, ok := src.(cmd.NamedTarget); ok {
		return player.IsOp(nt.Name())
	}
	return false
}

// RegisterAll registers all core administrative and moderation commands into the cmd registry.
func RegisterAll(srv Server) {
	registerOp(srv)
	registerDeop(srv)
	registerKick(srv)
	registerBan(srv)
	registerPardon()
	registerBanIP(srv)
	registerPardonIP()
	registerList(srv)
	registerSay(srv)
	registerVersion()
	registerWhitelist()
	registerLocate()
	registerTimings()
	registerReload()
}
