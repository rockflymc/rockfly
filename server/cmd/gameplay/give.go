package gameplay

import (
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func registerGive() {
	cmd.Register(cmd.New("give", "Gives an item to one or more players", nil,
		Give{},
	))
}

// Give handles giving items to target players.
type Give struct {
	Player []cmd.Target      `cmd:"player"`
	Item   string            `cmd:"item"`
	Amount cmd.Optional[int] `cmd:"amount"`
	Meta   cmd.Optional[int] `cmd:"meta"`
}

func (c Give) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c Give) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	itemName := strings.ToLower(strings.TrimSpace(c.Item))
	if !strings.Contains(itemName, ":") {
		itemName = "minecraft:" + itemName
	}

	meta := int16(0)
	if m, ok := c.Meta.Load(); ok {
		meta = int16(m)
	}

	amount := 1
	if a, ok := c.Amount.Load(); ok && a > 0 {
		amount = a
	}

	var it world.Item
	if found, ok := world.ItemByName(itemName, meta); ok {
		it = found
	} else if blk, ok := world.BlockByName(itemName, nil); ok {
		if bit, ok := blk.(world.Item); ok {
			it = bit
		}
	}

	if it == nil {
		o.Errorf("Unknown item '%s'", c.Item)
		return
	}

	st := item.NewStack(it, amount)
	count := 0
	for _, target := range c.Player {
		if p, ok := target.(*player.Player); ok {
			_, _ = p.Inventory().AddItem(st)
			count++
		}
	}

	if count == 0 {
		o.Errorf("No valid players found to give items to.")
		return
	}
	o.Printf("Gave %d %s to %d player(s)", amount, c.Item, count)
}
