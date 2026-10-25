package gameplay

import (
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

func registerPlaySound() {
	cmd.Register(cmd.New("playsound", "Plays a sound at a position or to players", nil,
		PlaySoundCmd{},
	))
}

// PlaySoundCmd plays a sound in the world.
type PlaySoundCmd struct {
	Sound    string                   `cmd:"sound"`
	Player   cmd.Optional[[]cmd.Target] `cmd:"player"`
	Position cmd.Optional[mgl64.Vec3] `cmd:"position"`
}

func (c PlaySoundCmd) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c PlaySoundCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if tx == nil {
		o.Errorf("Cannot execute playsound outside of an active world.")
		return
	}

	pos := src.Position()
	if customPos, ok := c.Position.Load(); ok {
		pos = customPos
	} else if targets, ok := c.Player.Load(); ok && len(targets) > 0 {
		pos = targets[0].Position()
	}

	s, ok := parseSound(c.Sound)
	if !ok {
		o.Errorf("Unknown sound '%s'", c.Sound)
		return
	}

	tx.PlaySound(pos, s)
	o.Printf("Played sound %s at %.2f, %.2f, %.2f", c.Sound, pos[0], pos[1], pos[2])
}

func parseSound(name string) (world.Sound, bool) {
	clean := strings.ToLower(strings.TrimSpace(name))
	clean = strings.TrimPrefix(clean, "minecraft:")

	switch clean {
	case "explosion", "random.explode":
		return sound.Explosion{}, true
	case "thunder", "ambient.weather.thunder":
		return sound.Thunder{}, true
	case "teleport", "mob.endermen.portal":
		return sound.Teleport{}, true
	case "levelup", "random.levelup":
		return sound.LevelUp{}, true
	case "burp", "random.burp":
		return sound.Burp{}, true
	case "anvil_use", "random.anvil_use":
		return sound.AnvilUse{}, true
	case "chest_open", "random.chestopen":
		return sound.ChestOpen{}, true
	case "chest_close", "random.chestclosed":
		return sound.ChestClose{}, true
	default:
		return sound.LevelUp{}, true
	}
}
