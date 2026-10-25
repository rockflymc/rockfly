package gameplay

import (
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

func registerSetBlock() {
	cmd.Register(cmd.New("setblock", "Changes a block to another block", nil,
		SetBlock{},
	))
}

// SetBlock changes the block at a given coordinate.
type SetBlock struct {
	Position mgl64.Vec3           `cmd:"position"`
	TileName string               `cmd:"tileName"`
	Mode     cmd.Optional[string] `cmd:"mode"`
}

func (c SetBlock) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c SetBlock) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if tx == nil {
		o.Errorf("Cannot execute setblock outside of an active world.")
		return
	}

	blockName := strings.ToLower(strings.TrimSpace(c.TileName))
	if !strings.Contains(blockName, ":") {
		blockName = "minecraft:" + blockName
	}

	blk, ok := world.BlockByName(blockName, nil)
	if !ok {
		o.Errorf("Unknown block '%s'", c.TileName)
		return
	}

	pos := cube.PosFromVec3(c.Position)
	mode := "replace"
	if m, ok := c.Mode.Load(); ok {
		mode = strings.ToLower(m)
	}

	current := tx.Block(pos)
	if mode == "keep" {
		if air, ok := world.BlockByName("minecraft:air", nil); ok && current != air {
			o.Errorf("The block could not be placed because destination is not air.")
			return
		}
	}

	tx.SetBlock(pos, blk, nil)
	o.Printf("Block placed at %d, %d, %d", pos[0], pos[1], pos[2])
}
