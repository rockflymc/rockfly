package gameplay

import (
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
)

func registerDefaultGameMode() {
	cmd.Register(cmd.New("defaultgamemode", "Sets the default game mode for new players", nil,
		DefaultGameMode{},
	))
}

// DefaultGameMode sets the default game mode of the world.
type DefaultGameMode struct {
	Mode string `cmd:"gameMode"`
}

func (c DefaultGameMode) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c DefaultGameMode) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if tx == nil {
		o.Errorf("Cannot execute defaultgamemode outside of an active world.")
		return
	}

	mode, ok := parseGameMode(c.Mode)
	if !ok {
		o.Errorf("Unknown game mode '%s'", c.Mode)
		return
	}

	tx.World().SetDefaultGameMode(mode)
	o.Printf("The world's default game mode is now %s", c.Mode)
}
