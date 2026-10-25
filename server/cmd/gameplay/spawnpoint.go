package gameplay

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

func registerSpawnPoint() {
	cmd.Register(cmd.New("spawnpoint", "Sets the spawn point for a player", nil,
		SpawnPointSelf{},
		SpawnPointTarget{},
	))
}

// SpawnPointSelf sets the spawn point of the executing player.
type SpawnPointSelf struct {
	Position cmd.Optional[mgl64.Vec3] `cmd:"position"`
}

func (c SpawnPointSelf) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c SpawnPointSelf) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok {
		o.Errorf("You must specify a target player when executing from console.")
		return
	}

	pos := p.Position()
	if customPos, ok := c.Position.Load(); ok {
		pos = customPos
	}

	if !isValidVec3(pos) {
		o.Errorf("Coordinates must not contain NaN or Infinity")
		return
	}

	blockPos := cube.PosFromVec3(pos)
	tx.World().SetPlayerSpawn(p.UUID(), blockPos)
	o.Printf("Set %s's spawn point to %d, %d, %d", p.Name(), blockPos[0], blockPos[1], blockPos[2])
}

// SpawnPointTarget sets the spawn point for target players.
type SpawnPointTarget struct {
	Player   []cmd.Target             `cmd:"player"`
	Position cmd.Optional[mgl64.Vec3] `cmd:"position"`
}

func (c SpawnPointTarget) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c SpawnPointTarget) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	count := 0
	for _, target := range c.Player {
		if p, ok := target.(*player.Player); ok {
			pos := p.Position()
			if customPos, ok := c.Position.Load(); ok {
				pos = customPos
			}
			if !isValidVec3(pos) {
				continue
			}
			blockPos := cube.PosFromVec3(pos)
			tx.World().SetPlayerSpawn(p.UUID(), blockPos)
			count++
		}
	}

	if count == 0 {
		o.Errorf("No valid target players found.")
		return
	}
	o.Printf("Set spawn point for %d player(s)", count)
}
