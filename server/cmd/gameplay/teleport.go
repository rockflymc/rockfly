package gameplay

import (
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

func registerTeleport() {
	cmd.Register(cmd.New("teleport", "Teleports players or entities", []string{"tp"},
		TeleportPos{},
		TeleportTarget{},
		TeleportTargetPos{},
		TeleportTargetTarget{},
	))
}

// TeleportPos teleports the executing player to a specific position.
type TeleportPos struct {
	Destination mgl64.Vec3 `cmd:"destination"`
}

func (c TeleportPos) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c TeleportPos) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok {
		o.Errorf("You must specify a target player when executing from console.")
		return
	}
	p.Teleport(c.Destination)
	o.Printf("Teleported %s to %.2f, %.2f, %.2f", p.Name(), c.Destination[0], c.Destination[1], c.Destination[2])
}

// TeleportTarget teleports the executing player to another target entity.
type TeleportTarget struct {
	Destination []cmd.Target `cmd:"destination"`
}

func (c TeleportTarget) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c TeleportTarget) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok {
		o.Errorf("You must specify a target player when executing from console.")
		return
	}
	if len(c.Destination) == 0 {
		o.Errorf("No destination target found.")
		return
	}
	destPos := c.Destination[0].Position()
	p.Teleport(destPos)
	o.Printf("Teleported %s to %.2f, %.2f, %.2f", p.Name(), destPos[0], destPos[1], destPos[2])
}

// TeleportTargetPos teleports target entities to a specific position.
type TeleportTargetPos struct {
	Victim      []cmd.Target `cmd:"victim"`
	Destination mgl64.Vec3   `cmd:"destination"`
}

func (c TeleportTargetPos) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c TeleportTargetPos) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	count := 0
	for _, target := range c.Victim {
		if p, ok := target.(*player.Player); ok {
			p.Teleport(c.Destination)
			count++
		}
	}
	if count == 0 {
		o.Errorf("No valid players found to teleport.")
		return
	}
	o.Printf("Teleported %d player(s) to %.2f, %.2f, %.2f", count, c.Destination[0], c.Destination[1], c.Destination[2])
}

// TeleportTargetTarget teleports target entities to a destination entity.
type TeleportTargetTarget struct {
	Victim      []cmd.Target `cmd:"victim"`
	Destination []cmd.Target `cmd:"destination"`
}

func (c TeleportTargetTarget) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c TeleportTargetTarget) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if len(c.Destination) == 0 {
		o.Errorf("No destination target found.")
		return
	}
	destPos := c.Destination[0].Position()
	count := 0
	for _, target := range c.Victim {
		if p, ok := target.(*player.Player); ok {
			p.Teleport(destPos)
			count++
		}
	}
	if count == 0 {
		o.Errorf("No valid players found to teleport.")
		return
	}
	o.Printf("Teleported %d player(s) to destination", count)
}
