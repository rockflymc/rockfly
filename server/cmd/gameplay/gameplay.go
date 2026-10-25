package gameplay

import (
	"math"
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/cmd/admin"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/go-gl/mathgl/mgl64"
)

// isValidVec3 checks if a 3D vector contains real, finite numbers.
func isValidVec3(v mgl64.Vec3) bool {
	return !math.IsNaN(v[0]) && !math.IsNaN(v[1]) && !math.IsNaN(v[2]) &&
		!math.IsInf(v[0], 0) && !math.IsInf(v[1], 0) && !math.IsInf(v[2], 0)
}

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
