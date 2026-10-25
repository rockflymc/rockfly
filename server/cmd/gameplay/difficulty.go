package gameplay

import (
	"strconv"
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
)

func registerDifficulty() {
	cmd.Register(cmd.New("difficulty", "Sets the game difficulty level", nil,
		Difficulty{},
	))
}

// Difficulty changes the difficulty of the world.
type Difficulty struct {
	Difficulty string `cmd:"difficulty"`
}

func (c Difficulty) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c Difficulty) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if tx == nil {
		o.Errorf("Cannot execute difficulty outside of an active world.")
		return
	}

	diff, name, ok := parseDifficulty(c.Difficulty)
	if !ok {
		o.Errorf("Unknown difficulty '%s' (expected peaceful, easy, normal, or hard)", c.Difficulty)
		return
	}

	tx.World().SetDifficulty(diff)
	o.Printf("The difficulty has been set to %s", name)
}

func parseDifficulty(str string) (world.Difficulty, string, bool) {
	switch strings.ToLower(str) {
	case "peaceful", "p", "0":
		return world.DifficultyPeaceful, "Peaceful", true
	case "easy", "e", "1":
		return world.DifficultyEasy, "Easy", true
	case "normal", "n", "2":
		return world.DifficultyNormal, "Normal", true
	case "hard", "h", "3":
		return world.DifficultyHard, "Hard", true
	default:
		if id, err := strconv.Atoi(str); err == nil {
			if d, ok := world.DifficultyByID(id); ok {
				return d, str, true
			}
		}
		return nil, "", false
	}
}
