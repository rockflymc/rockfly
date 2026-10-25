package block

import (
	"math/rand/v2"
	"sync"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/internal/nbtconv"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

var (
	_ world.Block                     = TrappedChest{}
	_ world.Item                      = TrappedChest{}
	_ world.ScheduledTicker           = TrappedChest{}
	_ world.RedstonePowerSource       = TrappedChest{}
	_ world.RedstoneStrongPowerSource = TrappedChest{}
	_ world.RedstoneWeakBlockPowerer  = TrappedChest{}
	_ world.NBTer                     = TrappedChest{}
	_ Container                       = TrappedChest{}
)

// TrappedChest is a redstone-emitting variant of the regular chest. It emits a redstone signal
// whose strength equals the number of players currently viewing the chest (capped at 15).
type TrappedChest struct {
	chest
	transparent
	bass

	// Facing is the horizontal direction the trapped chest faces.
	Facing cube.Direction
	// CustomName is the custom display name of the trapped chest.
	CustomName string

	inventory *inventory.Inventory
	viewerMu  *sync.RWMutex
	viewers   map[ContainerViewer]struct{}
}

// NewTrappedChest initializes a new trapped chest with a 27-slot inventory.
func NewTrappedChest() TrappedChest {
	m := &sync.RWMutex{}
	v := make(map[ContainerViewer]struct{})
	return TrappedChest{
		Facing:   cube.North,
		viewerMu: m,
		viewers:  v,
		inventory: inventory.New(27, func(slot int, before, after item.Stack) {
			m.RLock()
			defer m.RUnlock()
			for viewer := range v {
				viewer.ViewSlotChange(slot, after)
			}
		}),
	}
}

// ContainerSize returns 27.
func (TrappedChest) ContainerSize() int {
	return 27
}

// Inventory returns the trapped chest's inventory.
func (c TrappedChest) Inventory(*world.Tx, cube.Pos) *inventory.Inventory {
	return c.inventory
}

// RedstoneWeaklyPowersBlocks indicates whether weak power can conduct to adjacent blocks.
func (TrappedChest) RedstoneWeaklyPowersBlocks() bool {
	return true
}

// RedstonePower emits signal strength equal to the viewer count (clamped to 15).
func (c TrappedChest) RedstonePower(_ cube.Pos, _ *world.Tx, _ cube.Face) int {
	if c.viewerMu == nil {
		return 0
	}
	c.viewerMu.RLock()
	defer c.viewerMu.RUnlock()
	count := len(c.viewers)
	if count > 15 {
		count = 15
	}
	return count
}

// RedstoneStrongPower strongly powers the block underneath the chest.
func (c TrappedChest) RedstoneStrongPower(pos cube.Pos, tx *world.Tx, face cube.Face) int {
	if face == cube.FaceDown {
		return c.RedstonePower(pos, tx, face)
	}
	return 0
}

// AddViewer adds a viewer and plays the open chest animation and sound.
func (c TrappedChest) AddViewer(v ContainerViewer, tx *world.Tx, pos cube.Pos) {
	c.viewerMu.Lock()
	defer c.viewerMu.Unlock()
	if len(c.viewers) == 0 {
		for _, viewer := range tx.Viewers(pos.Vec3()) {
			viewer.ViewBlockAction(pos, OpenAction{})
		}
		tx.PlaySound(pos.Vec3Centre(), sound.ChestOpen{})
	}
	c.viewers[v] = struct{}{}
	tx.ScheduleBlockUpdate(pos, c, redstoneTicks(1))
}

// RemoveViewer removes a viewer and closes the chest when empty.
func (c TrappedChest) RemoveViewer(v ContainerViewer, tx *world.Tx, pos cube.Pos) {
	c.viewerMu.Lock()
	defer c.viewerMu.Unlock()
	delete(c.viewers, v)
	if len(c.viewers) == 0 {
		for _, viewer := range tx.Viewers(pos.Vec3()) {
			viewer.ViewBlockAction(pos, CloseAction{})
		}
		tx.PlaySound(pos.Vec3Centre(), sound.ChestClose{})
	}
	tx.ScheduleBlockUpdate(pos, c, redstoneTicks(1))
}

// ScheduledTick updates the block state when viewer count changes.
func (c TrappedChest) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	tx.SetBlock(pos, c, nil)
}

// Activate opens the trapped chest container for a player.
func (c TrappedChest) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, u item.User, _ *item.UseContext) bool {
	if opener, ok := u.(ContainerOpener); ok {
		opener.OpenBlockContainer(pos, tx)
		return true
	}
	return false
}

// UseOnBlock places the trapped chest facing away from the player.
func (c TrappedChest) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) (used bool) {
	pos, _, used = firstReplaceable(tx, pos, face, c)
	if !used {
		return false
	}
	//noinspection GoAssignmentToReceiver
	c = NewTrappedChest()
	c.Facing = user.Rotation().Direction().Opposite()
	place(tx, pos, c, user, ctx)
	return placed(ctx)
}

// BreakInfo ...
func (c TrappedChest) BreakInfo() BreakInfo {
	return newBreakInfo(2.5, alwaysHarvestable, axeEffective, oneOf(NewTrappedChest())).withBreakHandler(func(pos cube.Pos, tx *world.Tx, _ item.User) {
		if c.inventory != nil {
			for _, i := range c.inventory.Clear() {
				dropItem(tx, i, pos.Vec3Centre())
			}
		}
	})
}

// EncodeItem ...
func (TrappedChest) EncodeItem() (name string, meta int16) {
	return "minecraft:trapped_chest", 0
}

// EncodeBlock ...
func (c TrappedChest) EncodeBlock() (string, map[string]any) {
	return "minecraft:trapped_chest", map[string]any{
		"minecraft:cardinal_direction": c.Facing.String(),
	}
}

// EncodeNBT ...
func (c TrappedChest) EncodeNBT() map[string]any {
	if c.inventory == nil {
		facing, customName := c.Facing, c.CustomName
		//noinspection GoAssignmentToReceiver
		c = NewTrappedChest()
		c.Facing, c.CustomName = facing, customName
	}
	m := map[string]any{
		"Items": nbtconv.InvToNBT(c.inventory),
		"id":    "Chest",
	}
	if c.CustomName != "" {
		m["CustomName"] = c.CustomName
	}
	return m
}

// DecodeNBT ...
func (c TrappedChest) DecodeNBT(data map[string]any) any {
	facing := c.Facing
	//noinspection GoAssignmentToReceiver
	c = NewTrappedChest()
	c.Facing = facing
	c.CustomName = nbtconv.String(data, "CustomName")
	nbtconv.InvFromNBT(c.inventory, nbtconv.Slice(data, "Items"))
	return c
}

// allTrappedChests returns all 4 directional trapped chests.
func allTrappedChests() (chests []world.Block) {
	for _, d := range cube.Directions() {
		chests = append(chests, TrappedChest{Facing: d})
	}
	return chests
}
