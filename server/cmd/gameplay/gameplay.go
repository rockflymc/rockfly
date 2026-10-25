package gameplay

import (
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/cmd/admin"
	"github.com/df-mc/dragonfly/server/player"
)

// isAllowed checks if a command source is permitted to execute gameplay/cheats commands.
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

// RegisterAll registers all gameplay commands into the cmd registry.
func RegisterAll(srv admin.Server) {
	registerGameMode()
	registerGive()
	registerTeleport()
	registerTime()
	registerWeather()
	registerSummon()
	registerSetBlock()
	registerFill()
	registerClone()
	registerEffect()
	registerEnchant()
	registerClear()
	registerKill()
	registerDifficulty()
	registerDefaultGameMode()
	registerSpawnPoint()
	registerSetWorldSpawn()
	registerTitle()
	registerParticle()
	registerPlaySound()
}
