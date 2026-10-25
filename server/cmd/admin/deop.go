package admin

import (
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func registerDeop(srv Server) {
	cmd.Register(cmd.New("deop", "Revokes operator status from a player", nil,
		Deop{srv: srv},
		DeopOffline{srv: srv},
	))
}

// Deop revokes operator status from matched target players.
type Deop struct {
	srv    Server       `cmd:"-"`
	Target []cmd.Target `cmd:"player"`
}

// Allow reports whether the source is permitted to execute /deop.
func (c Deop) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

// Run executes the /deop command for matched targets.
func (c Deop) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	for _, target := range c.Target {
		if p, ok := target.(*player.Player); ok {
			p.SetOp(false)
			p.SetOpLevel(0)
			_ = player.DefaultOperatorManager().Deop(p.Name())
			if p.UUID().String() != "" {
				_ = player.DefaultOperatorManager().Deop(p.UUID().String())
			}
			p.Message("§eYou are no longer op!")
			o.Printf("Made %s no longer a server operator", p.Name())
		} else if nt, ok := target.(cmd.NamedTarget); ok {
			_ = player.DefaultOperatorManager().Deop(nt.Name())
			o.Printf("Made %s no longer a server operator", nt.Name())
		}
	}
}

// DeopOffline revokes operator status from a player by name.
type DeopOffline struct {
	srv    Server `cmd:"-"`
	Player string `cmd:"player"`
}

// Allow reports whether the source is permitted to execute /deop.
func (c DeopOffline) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

// Run executes the /deop command for a player specified by name.
func (c DeopOffline) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if c.srv != nil {
		if handle, ok := c.srv.PlayerByName(c.Player); ok {
			if e, ok := handle.Entity(tx); ok {
				if p, ok := e.(*player.Player); ok {
					p.SetOp(false)
					p.SetOpLevel(0)
					_ = player.DefaultOperatorManager().Deop(p.Name())
					p.Message("§eYou are no longer op!")
					o.Printf("Made %s no longer a server operator", p.Name())
					return
				}
			}
		}
	}

	_ = player.DefaultOperatorManager().Deop(c.Player)
	o.Printf("Made %s no longer a server operator", c.Player)
}
