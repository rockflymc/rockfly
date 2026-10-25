package gameplay

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

func registerSetWorldSpawn() {
	cmd.Register(cmd.New("setworldspawn", "Sets the world spawn", nil,
		SetWorldSpawn{},
	))
}

// SetWorldSpawn sets the world's spawn location.
type SetWorldSpawn struct {
	Position cmd.Optional[mgl64.Vec3] `cmd:"position"`
}

func (c SetWorldSpawn) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c SetWorldSpawn) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if tx == nil {
		o.Errorf("Cannot execute setworldspawn outside of an active world.")
		return
	}

	pos := src.Position()
	if customPos, ok := c.Position.Load(); ok {
		pos = customPos
	}

	if !isValidVec3(pos) {
		o.Errorf("Coordinates must not contain NaN or Infinity")
		return
	}

	blockPos := cube.PosFromVec3(pos)
	tx.World().SetSpawn(blockPos)
	o.Printf("Set the world spawn to %d, %d, %d", blockPos[0], blockPos[1], blockPos[2])
}
