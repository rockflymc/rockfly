package gameplay

import (
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func registerKill() {
	cmd.Register(cmd.New("kill", "Kills entities (players, mobs, etc.)", nil,
		KillSelf{},
		KillTarget{},
	))
}

// KillSelf kills the executing player.
type KillSelf struct{}

func (c KillSelf) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c KillSelf) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok {
		o.Errorf("You must specify a target when executing from console.")
		return
	}
	p.Hurt(100000, entity.VoidDamageSource{})
	o.Printf("Killed %s", p.Name())
}

// KillTarget kills specified target entities or players.
type KillTarget struct {
	Target []cmd.Target `cmd:"target"`
}

func (c KillTarget) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c KillTarget) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	count := 0
	for _, target := range c.Target {
		if p, ok := target.(*player.Player); ok {
			p.Hurt(100000, entity.VoidDamageSource{})
			count++
		} else if ent, ok := target.(world.Entity); ok {
			_ = ent.Close()
			count++
		}
	}
	if count == 0 {
		o.Errorf("No valid targets found to kill.")
		return
	}
	o.Printf("Killed %d entity/player(s)", count)
}
