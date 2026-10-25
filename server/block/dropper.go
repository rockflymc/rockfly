package block

import (
	"math"
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
	_ world.Block                 = Dropper{}
	_ world.Item                  = Dropper{}
	_ world.NeighbourUpdateTicker = Dropper{}
	_ world.ScheduledTicker       = Dropper{}
	_ world.RedstonePowerAction   = Dropper{}
	_ world.NBTer                 = Dropper{}
	_ Container                   = Dropper{}
)

// Dropper is a redstone-activated container block that either transfers items directly into an adjacent container
// or ejects them into the world as dropped item entities.
type Dropper struct {
	solid
	transparent

	// Facing is the direction the dropper dispenses items into.
	Facing cube.Face
	// Triggered specifies if the dropper was triggered by a redstone signal.
	Triggered bool
	// CustomName is the custom display name of the dropper container.
	CustomName string

	inventory *inventory.Inventory
	viewerMu  *sync.RWMutex
	viewers   map[ContainerViewer]struct{}
}

// NewDropper initializes a new dropper block with an empty 9-slot inventory.
func NewDropper() Dropper {
	m := &sync.RWMutex{}
	v := make(map[ContainerViewer]struct{})
	return Dropper{
		Facing:   cube.FaceNorth,
		viewerMu: m,
		viewers:  v,
		inventory: inventory.New(9, func(slot int, before, after item.Stack) {
			m.RLock()
			defer m.RUnlock()
			for viewer := range v {
				viewer.ViewSlotChange(slot, after)
			}
		}),
	}
}

// ContainerSize returns 9, the slot count of a dropper.
func (Dropper) ContainerSize() int {
	return 9
}

// Inventory returns the dropper's 9-slot item inventory.
func (d Dropper) Inventory(*world.Tx, cube.Pos) *inventory.Inventory {
	return d.inventory
}

// AddViewer adds a viewer to the dropper.
func (d Dropper) AddViewer(v ContainerViewer, _ *world.Tx, _ cube.Pos) {
	if d.viewerMu == nil {
		return
	}
	d.viewerMu.Lock()
	defer d.viewerMu.Unlock()
	d.viewers[v] = struct{}{}
}

// RemoveViewer removes a viewer from the dropper.
func (d Dropper) RemoveViewer(v ContainerViewer, _ *world.Tx, _ cube.Pos) {
	if d.viewerMu == nil {
		return
	}
	d.viewerMu.Lock()
	defer d.viewerMu.Unlock()
	delete(d.viewers, v)
}

// Activate opens the dropper's container interface for the player.
func (d Dropper) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, u item.User, _ *item.UseContext) bool {
	if opener, ok := u.(ContainerOpener); ok {
		opener.OpenBlockContainer(pos, tx)
		return true
	}
	return false
}

// UseOnBlock places the dropper facing away from the player.
func (d Dropper) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) (used bool) {
	pos, _, used = firstReplaceable(tx, pos, face, d)
	if !used {
		return false
	}
	//noinspection GoAssignmentToReceiver
	d = NewDropper()
	d.Facing = calculateFace(user, pos).Opposite()
	place(tx, pos, d, user, ctx)
	return placed(ctx)
}

// NeighbourUpdateTick schedules an ejection tick on redstone rising edges.
func (d Dropper) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if tx == nil || pos.OutOfBounds(tx.Range()) {
		return
	}
	powered := tx.RedstonePower(pos) > 0
	if powered && !d.Triggered {
		tx.ScheduleBlockUpdate(pos, d, redstoneTicks(2))
	} else if !powered && d.Triggered {
		d.Triggered = false
		tx.SetBlock(pos, d, nil)
	}
}

// RedstonePowerAction schedules an ejection on direct redstone power changes.
func (d Dropper) RedstonePowerAction(pos cube.Pos, tx *world.Tx, oldPower, newPower int) {
	if tx == nil || pos.OutOfBounds(tx.Range()) {
		return
	}
	if newPower > 0 && oldPower == 0 && !d.Triggered {
		tx.ScheduleBlockUpdate(pos, d, redstoneTicks(2))
	} else if newPower == 0 && oldPower > 0 && d.Triggered {
		d.Triggered = false
		tx.SetBlock(pos, d, nil)
	}
}

// ScheduledTick executes the dropping action on a redstone pulse.
func (d Dropper) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	if tx == nil || pos.OutOfBounds(tx.Range()) {
		return
	}
	if tx.RedstonePower(pos) > 0 {
		d.Triggered = true
		tx.SetBlock(pos, d, nil)
		d.Drop(pos, tx)
	}
}

// Drop chooses a random item from the dropper and transfers it into an adjacent container or drops it.
func (d Dropper) Drop(pos cube.Pos, tx *world.Tx) {
	if d.inventory == nil || tx == nil || pos.OutOfBounds(tx.Range()) {
		return
	}

	targetPos := pos.Side(d.Facing)
	if targetPos.OutOfBounds(tx.Range()) {
		return
	}

	var nonEmpty []int
	for slot, st := range d.inventory.Slots() {
		if !st.Empty() {
			nonEmpty = append(nonEmpty, slot)
		}
	}
	if len(nonEmpty) == 0 {
		tx.PlaySound(pos.Vec3Centre(), sound.Click{})
		return
	}

	chosenSlot := nonEmpty[rand.IntN(len(nonEmpty))]
	st, _ := d.inventory.Item(chosenSlot)
	toDrop := st.Grow(-st.Count() + 1)
	_ = d.inventory.SetItem(chosenSlot, st.Grow(-1))

	targetBlock := tx.Block(targetPos)

	// Check if target block has an inventory to transfer into.
	if c, ok := targetBlock.(Container); ok {
		if inv := c.Inventory(tx, targetPos); inv != nil {
			if _, err := inv.AddItem(toDrop); err == nil {
				tx.PlaySound(pos.Vec3Centre(), sound.Click{})
				return
			}
		}
	}

	// Drop as item entity in front of the dropper.
	spawnPos := targetPos.Vec3Centre()
	dropVelocity := faceVec3(d.Facing).Mul(0.3)
	dropItemWithVelocity(tx, toDrop, spawnPos, dropVelocity)
	tx.PlaySound(pos.Vec3Centre(), sound.Click{})
}

// BreakInfo ...
func (d Dropper) BreakInfo() BreakInfo {
	return newBreakInfo(3.5, pickaxeHarvestable, pickaxeEffective, oneOf(NewDropper())).withBreakHandler(func(pos cube.Pos, tx *world.Tx, _ item.User) {
		if tx == nil || pos.OutOfBounds(tx.Range()) {
			return
		}
		if d.inventory != nil {
			for _, i := range d.inventory.Clear() {
				dropItem(tx, i, pos.Vec3Centre())
			}
		}
	})
}

// EncodeItem ...
func (Dropper) EncodeItem() (name string, meta int16) {
	return "minecraft:dropper", 0
}

// EncodeBlock ...
func (d Dropper) EncodeBlock() (string, map[string]any) {
	return "minecraft:dropper", map[string]any{
		"facing_direction": int32(d.Facing),
		"triggered_bit":    boolByte(d.Triggered),
	}
}

// EncodeNBT encodes dropper inventory into NBT.
func (d Dropper) EncodeNBT() map[string]any {
	if d.inventory == nil {
		facing, triggered, customName := d.Facing, d.Triggered, d.CustomName
		//noinspection GoAssignmentToReceiver
		d = NewDropper()
		d.Facing, d.Triggered, d.CustomName = facing, triggered, customName
	}
	m := map[string]any{
		"Items": nbtconv.InvToNBT(d.inventory),
		"id":    "Dropper",
	}
	if d.CustomName != "" {
		m["CustomName"] = d.CustomName
	}
	return m
}

// DecodeNBT decodes dropper inventory from NBT.
func (d Dropper) DecodeNBT(data map[string]any) any {
	facing, triggered := d.Facing, d.Triggered
	//noinspection GoAssignmentToReceiver
	d = NewDropper()
	d.Facing = facing
	d.Triggered = triggered
	d.CustomName = nbtconv.String(data, "CustomName")
	nbtconv.InvFromNBT(d.inventory, nbtconv.Slice(data, "Items"))
	return d
}

// allDroppers returns all permutations of droppers across all 6 faces and triggered states.
func allDroppers() (droppers []world.Block) {
	for _, f := range cube.Faces() {
		droppers = append(droppers, Dropper{Facing: f, Triggered: false})
		droppers = append(droppers, Dropper{Facing: f, Triggered: true})
	}
	return droppers
}

// dropItemWithVelocity spawns a dropped item entity into the world with initial velocity.
func dropItemWithVelocity(tx *world.Tx, it item.Stack, pos mgl64.Vec3, vel mgl64.Vec3) {
	if it.Empty() || tx == nil {
		return
	}
	if math.IsNaN(pos[0]) || math.IsNaN(pos[1]) || math.IsNaN(pos[2]) || math.IsInf(pos[0], 0) || math.IsInf(pos[1], 0) || math.IsInf(pos[2], 0) {
		return
	}
	if math.IsNaN(vel[0]) || math.IsNaN(vel[1]) || math.IsNaN(vel[2]) || math.IsInf(vel[0], 0) || math.IsInf(vel[1], 0) || math.IsInf(vel[2], 0) {
		return
	}
	create := tx.World().EntityRegistry().Config().Item
	if create == nil {
		return
	}
	opts := world.EntitySpawnOpts{Position: pos, Velocity: vel}
	tx.AddEntity(create(opts, it))
}
