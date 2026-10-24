package admin

import (
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func registerKick(srv Server) {
	cmd.Register(cmd.New("kick", "Disconnects a player from the server", nil,
		Kick{srv: srv},
	))
}

// Kick disconnects target players with an optional custom reason.
type Kick struct {
	srv    Server                    `cmd:"-"`
	Target []cmd.Target              `cmd:"player"`
	Reason cmd.Optional[cmd.Varargs] `cmd:"reason"`
}

// Allow reports whether the source is permitted to execute /kick.
func (c Kick) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

// Run executes the /kick command.
func (c Kick) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	reason := "Kicked by an operator."
	if r, ok := c.Reason.Load(); ok && string(r) != "" {
		reason = string(r)
	}

	for _, target := range c.Target {
		if p, ok := target.(*player.Player); ok {
			name := p.Name()
			p.Disconnect(reason)
			o.Printf("Kicked %s: %s", name, reason)
		} else if nt, ok := target.(cmd.NamedTarget); ok {
			o.Printf("Target %s is not an active player", nt.Name())
		}
	}
}
