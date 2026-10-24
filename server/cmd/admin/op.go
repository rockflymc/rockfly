package admin

import (
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func registerOp(srv Server) {
	cmd.Register(cmd.New("op", "Grants operator status to a player", nil,
		Op{srv: srv},
		OpOffline{srv: srv},
	))
}

// Op handles granting operator status to an online target player.
type Op struct {
	srv    Server            `cmd:"-"`
	Target []cmd.Target      `cmd:"player"`
	Level  cmd.Optional[int] `cmd:"level"`
}

// Allow reports whether the source is permitted to execute /op.
func (c Op) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

// Run executes the /op command for matched target players.
func (c Op) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	lvl := 4
	if l, ok := c.Level.Load(); ok {
		if l < 1 || l > 4 {
			o.Errorf("Operator level must be between 1 and 4")
			return
		}
		lvl = l
	}

	for _, target := range c.Target {
		if p, ok := target.(*player.Player); ok {
			p.SetOp(true)
			p.SetOpLevel(lvl)
			_ = player.DefaultOperatorManager().SetOp(p.Name(), p.UUID().String(), lvl, false)
			p.Message("§eYou are now op!")
			o.Printf("Made %s a server operator (level %d)", p.Name(), lvl)
		} else if nt, ok := target.(cmd.NamedTarget); ok {
			_ = player.DefaultOperatorManager().SetOp(nt.Name(), "", lvl, false)
			o.Printf("Made %s a server operator (level %d)", nt.Name(), lvl)
		}
	}
}

// OpOffline handles granting operator status to a player by name (online or offline).
type OpOffline struct {
	srv    Server            `cmd:"-"`
	Player string            `cmd:"player"`
	Level  cmd.Optional[int] `cmd:"level"`
}

// Allow reports whether the source is permitted to execute /op.
func (c OpOffline) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

// Run executes the /op command for a player specified by name.
func (c OpOffline) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	lvl := 4
	if l, ok := c.Level.Load(); ok {
		if l < 1 || l > 4 {
			o.Errorf("Operator level must be between 1 and 4")
			return
		}
		lvl = l
	}

	if c.srv != nil {
		if handle, ok := c.srv.PlayerByName(c.Player); ok {
			if e, ok := handle.Entity(tx); ok {
				if p, ok := e.(*player.Player); ok {
					p.SetOp(true)
					p.SetOpLevel(lvl)
					_ = player.DefaultOperatorManager().SetOp(p.Name(), p.UUID().String(), lvl, false)
					p.Message("§eYou are now op!")
					o.Printf("Made %s a server operator (level %d)", p.Name(), lvl)
					return
				}
			}
		}
	}

	_ = player.DefaultOperatorManager().SetOp(c.Player, "", lvl, false)
	o.Printf("Made %s a server operator (level %d)", c.Player, lvl)
}
