package gameplay

import (
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/title"
	"github.com/df-mc/dragonfly/server/world"
)

func registerTitle() {
	cmd.Register(cmd.New("title", "Controls on-screen titles", nil,
		TitleCmd{},
	))
}

// TitleCmd controls player screen titles.
type TitleCmd struct {
	Player []cmd.Target              `cmd:"player"`
	Action string                    `cmd:"titleLocation"`
	Text   cmd.Optional[cmd.Varargs] `cmd:"titleText"`
}

func (c TitleCmd) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c TitleCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	rawText := ""
	if t, ok := c.Text.Load(); ok {
		rawText = string(t)
	}

	action := strings.ToLower(c.Action)
	var t title.Title

	switch action {
	case "title":
		t = title.New(rawText)
	case "subtitle":
		t = title.New("").WithSubtitle(rawText)
	case "actionbar":
		t = title.New("").WithActionText(rawText)
	case "clear", "reset":
		t = title.New("")
	default:
		o.Errorf("Unknown title location '%s' (expected title, subtitle, actionbar, clear, or reset)", c.Action)
		return
	}

	count := 0
	for _, target := range c.Player {
		if p, ok := target.(*player.Player); ok {
			p.SendTitle(t)
			count++
		}
	}

	if count == 0 {
		o.Errorf("No valid players found.")
		return
	}
	o.Printf("Sent title to %d player(s)", count)
}
