package gameplay

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

func registerClone() {
	cmd.Register(cmd.New("clone", "Clones blocks from one region to another", nil,
		Clone{},
	))
}

// Clone clones blocks from source to destination.
type Clone struct {
	From        mgl64.Vec3 `cmd:"begin"`
	To          mgl64.Vec3 `cmd:"end"`
	Destination mgl64.Vec3 `cmd:"destination"`
}

func (c Clone) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c Clone) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if tx == nil {
		o.Errorf("Cannot execute clone outside of an active world.")
		return
	}

	pos1 := cube.PosFromVec3(c.From)
	pos2 := cube.PosFromVec3(c.To)
	dest := cube.PosFromVec3(c.Destination)

	minX := min(pos1[0], pos2[0])
	maxX := max(pos1[0], pos2[0])
	minY := min(pos1[1], pos2[1])
	maxY := max(pos1[1], pos2[1])
	minZ := min(pos1[2], pos2[2])
	maxZ := max(pos1[2], pos2[2])

	volume := (maxX - minX + 1) * (maxY - minY + 1) * (maxZ - minZ + 1)
	if volume > 32768 {
		o.Errorf("Cannot clone more than 32768 blocks at once (requested %d)", volume)
		return
	}

	type blockEntry struct {
		offset cube.Pos
		b      world.Block
	}
	var entries []blockEntry
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			for z := minZ; z <= maxZ; z++ {
				srcPos := cube.Pos{x, y, z}
				b := tx.Block(srcPos)
				offset := cube.Pos{x - minX, y - minY, z - minZ}
				entries = append(entries, blockEntry{offset: offset, b: b})
			}
		}
	}

	for _, entry := range entries {
		targetPos := dest.Add(entry.offset)
		tx.SetBlock(targetPos, entry.b, nil)
	}

	o.Printf("Successfully cloned %d block(s)", len(entries))
}
