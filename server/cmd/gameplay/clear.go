package gameplay

import (
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func registerClear() {
	cmd.Register(cmd.New("clear", "Clears items from player inventory", nil,
		ClearSelf{},
		ClearTarget{},
	))
}

// ClearSelf clears the inventory of the executing player.
type ClearSelf struct{}

func (c ClearSelf) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c ClearSelf) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok {
		o.Errorf("You must specify a player target when executing from console.")
		return
	}
	cleared := len(p.Inventory().Clear())
	_ = p.Armour().Clear()
	o.Printf("Cleared the inventory of %s, removing items", p.Name())
	_ = cleared
}

// ClearTarget clears the inventory of specified target players.
type ClearTarget struct {
	Player []cmd.Target `cmd:"player"`
}

func (c ClearTarget) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c ClearTarget) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	count := 0
	for _, target := range c.Player {
		if p, ok := target.(*player.Player); ok {
			p.Inventory().Clear()
			p.Armour().Clear()
			count++
		}
	}
	if count == 0 {
		o.Errorf("No valid target players found.")
		return
	}
	o.Printf("Cleared the inventory of %d player(s)", count)
}
