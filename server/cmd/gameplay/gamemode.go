package gameplay

import (
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func registerGameMode() {
	cmd.Register(cmd.New("gamemode", "Sets a player's game mode", []string{"gm"},
		GameModeSelf{},
		GameModeTarget{},
	))
}

// GameModeSelf changes the game mode of the executing player.
type GameModeSelf struct {
	Mode string `cmd:"gameMode"`
}

func (c GameModeSelf) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok {
		o.Errorf("You must specify a target player when executing from console.")
		return
	}
	mode, ok := parseGameMode(c.Mode)
	if !ok {
		o.Errorf("Unknown game mode '%s'", c.Mode)
		return
	}
	p.SetGameMode(mode)
	o.Printf("Your game mode has been set to %s", c.Mode)
}

// GameModeTarget changes the game mode of one or more specified players.
type GameModeTarget struct {
	Mode   string       `cmd:"gameMode"`
	Target []cmd.Target `cmd:"player"`
}

func (c GameModeTarget) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	mode, ok := parseGameMode(c.Mode)
	if !ok {
		o.Errorf("Unknown game mode '%s'", c.Mode)
		return
	}
	count := 0
	for _, t := range c.Target {
		if p, ok := t.(*player.Player); ok {
			p.SetGameMode(mode)
			count++
		}
	}
	if count == 0 {
		o.Errorf("No valid players found to update game mode.")
		return
	}
	o.Printf("Set game mode to %s for %d player(s)", c.Mode, count)
}

func parseGameMode(str string) (world.GameMode, bool) {
	switch strings.ToLower(str) {
	case "survival", "s", "0":
		return world.GameModeSurvival, true
	case "creative", "c", "1":
		return world.GameModeCreative, true
	case "adventure", "a", "2":
		return world.GameModeAdventure, true
	case "spectator", "sp", "3":
		return world.GameModeSpectator, true
	default:
		return nil, false
	}
}
