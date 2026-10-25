package gameplay

import (
	"math"
	"strings"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

func registerFill() {
	cmd.Register(cmd.New("fill", "Fills all or parts of a region with a specific block", nil,
		Fill{},
	))
}

// Fill fills a cuboid region with a specified block.
type Fill struct {
	From     mgl64.Vec3           `cmd:"from"`
	To       mgl64.Vec3           `cmd:"to"`
	TileName string               `cmd:"tileName"`
	Mode     cmd.Optional[string] `cmd:"mode"`
}

func (c Fill) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c Fill) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if tx == nil {
		o.Errorf("Cannot execute fill outside of an active world.")
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

	pos1 := cube.PosFromVec3(c.From)
	pos2 := cube.PosFromVec3(c.To)

	minX := min(pos1[0], pos2[0])
	maxX := max(pos1[0], pos2[0])
	minY := min(pos1[1], pos2[1])
	maxY := max(pos1[1], pos2[1])
	minZ := min(pos1[2], pos2[2])
	maxZ := max(pos1[2], pos2[2])

	volume := (maxX - minX + 1) * (maxY - minY + 1) * (maxZ - minZ + 1)
	if volume > 32768 {
		o.Errorf("Cannot fill more than 32768 blocks at once (requested %d)", volume)
		return
	}

	mode := "replace"
	if m, ok := c.Mode.Load(); ok {
		mode = strings.ToLower(m)
	}

	air, _ := world.BlockByName("minecraft:air", nil)
	count := 0
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			for z := minZ; z <= maxZ; z++ {
				p := cube.Pos{x, y, z}
				switch mode {
				case "keep":
					if tx.Block(p) != air {
						continue
					}
				case "outline":
					isBorder := x == minX || x == maxX || y == minY || y == maxY || z == minZ || z == maxZ
					if !isBorder {
						continue
					}
				case "hollow":
					isBorder := x == minX || x == maxX || y == minY || y == maxY || z == minZ || z == maxZ
					if !isBorder {
						tx.SetBlock(p, air, nil)
						count++
						continue
					}
				}
				tx.SetBlock(p, blk, nil)
				count++
			}
		}
	}

	o.Printf("Successfully filled %d block(s)", count)
}

func min(a, b int) int {
	return int(math.Min(float64(a), float64(b)))
}

func max(a, b int) int {
	return int(math.Max(float64(a), float64(b)))
}
