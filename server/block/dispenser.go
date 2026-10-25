package block

import (
	"math/rand/v2"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/internal/nbtconv"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

var (
	_ world.Block                 = Dispenser{}
	_ world.Item                  = Dispenser{}
	_ world.NeighbourUpdateTicker = Dispenser{}
	_ world.ScheduledTicker       = Dispenser{}
	_ world.RedstonePowerAction   = Dispenser{}
	_ world.NBTer                 = Dispenser{}
	_ Container                   = Dispenser{}
)

// Dispenser is a redstone-activated block that contextually dispenses items, shoots projectiles,
// places liquids, or equips armor.
type Dispenser struct {
	solid
	transparent

	// Facing is the direction the dispenser shoots or places items into.
	Facing cube.Face
	// Triggered specifies if the dispenser was triggered by redstone.
	Triggered bool
	// CustomName is the custom display name of the dispenser container.
	CustomName string

	inventory *inventory.Inventory
	viewerMu  *sync.RWMutex
	viewers   map[ContainerViewer]struct{}
}

// NewDispenser initializes a new dispenser block with an empty 9-slot inventory.
func NewDispenser() Dispenser {
	m := &sync.RWMutex{}
	v := make(map[ContainerViewer]struct{})
	return Dispenser{
		Facing:    cube.FaceNorth,
		viewerMu:  m,
		viewers:   v,
		inventory: inventory.New(9, func(slot int, before, after item.Stack) {
			m.RLock()
			defer m.RUnlock()
			for viewer := range v {
				viewer.ViewSlotChange(slot, after)
			}
		}),
	}
}

// ContainerSize returns 9, the slot count of a dispenser.
func (Dispenser) ContainerSize() int {
	return 9
}

// Inventory returns the dispenser's 9-slot item inventory.
func (d Dispenser) Inventory(*world.Tx, cube.Pos) *inventory.Inventory {
	return d.inventory
}

// AddViewer adds a viewer to the dispenser.
func (d Dispenser) AddViewer(v ContainerViewer, _ *world.Tx, _ cube.Pos) {
	d.viewerMu.Lock()
	defer d.viewerMu.Unlock()
	d.viewers[v] = struct{}{}
}

// RemoveViewer removes a viewer from the dispenser.
func (d Dispenser) RemoveViewer(v ContainerViewer, _ *world.Tx, _ cube.Pos) {
	d.viewerMu.Lock()
	defer d.viewerMu.Unlock()
	delete(d.viewers, v)
}

// Activate opens the dispenser's container interface for the player.
func (d Dispenser) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, u item.User, _ *item.UseContext) bool {
	if opener, ok := u.(ContainerOpener); ok {
		opener.OpenBlockContainer(pos, tx)
		return true
	}
	return false
}

// UseOnBlock places the dispenser facing away from the player.
func (d Dispenser) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) (used bool) {
	pos, _, used = firstReplaceable(tx, pos, face, d)
	if !used {
		return false
	}
	//noinspection GoAssignmentToReceiver
	d = NewDispenser()
	d.Facing = calculateFace(user, pos).Opposite()
	place(tx, pos, d, user, ctx)
	return placed(ctx)
}

// NeighbourUpdateTick schedules a dispensing tick on redstone rising edges.
func (d Dispenser) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	powered := tx.RedstonePower(pos) > 0
	if powered && !d.Triggered {
		tx.ScheduleBlockUpdate(pos, d, redstoneTicks(2))
	} else if !powered && d.Triggered {
		d.Triggered = false
		tx.SetBlock(pos, d, nil)
	}
}

// RedstonePowerAction schedules dispensing on direct redstone power changes.
func (d Dispenser) RedstonePowerAction(pos cube.Pos, tx *world.Tx, oldPower, newPower int) {
	if newPower > 0 && oldPower == 0 && !d.Triggered {
		tx.ScheduleBlockUpdate(pos, d, redstoneTicks(2))
	} else if newPower == 0 && oldPower > 0 && d.Triggered {
		d.Triggered = false
		tx.SetBlock(pos, d, nil)
	}
}

// ScheduledTick executes dispensing on a redstone pulse.
func (d Dispenser) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	if tx.RedstonePower(pos) > 0 {
		d.Triggered = true
		tx.SetBlock(pos, d, nil)
		d.Dispense(pos, tx)
	}
}

// Dispense executes contextual dispensing based on the item in a randomly chosen non-empty slot.
func (d Dispenser) Dispense(pos cube.Pos, tx *world.Tx) {
	if d.inventory == nil {
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
	targetPos := pos.Side(d.Facing)
	spawnPos := targetPos.Vec3Centre()
	directionVec := faceVec3(d.Facing)

	switch it := st.Item().(type) {
	case item.Bucket:
		if liquid, ok := it.Content.Liquid(); ok {
			// Places liquid into world if target is air or replaceable.
			targetBlock := tx.Block(targetPos)
			if targetBlock == nil || isPushReplaceable(targetBlock) {
				tx.SetLiquid(targetPos, liquid.WithDepth(8, false))
				_ = d.inventory.SetItem(chosenSlot, item.NewStack(item.Bucket{}, 1))
				tx.PlaySound(pos.Vec3Centre(), sound.Click{})
				return
			}
		} else {
			// Empty bucket: scoops up liquid if present at target.
			if liq, ok := tx.Liquid(targetPos); ok {
				tx.SetLiquid(targetPos, nil)
				_ = d.inventory.SetItem(chosenSlot, item.NewStack(item.Bucket{Content: item.LiquidBucketContent(liq.WithDepth(8, false))}, 1))
				tx.PlaySound(pos.Vec3Centre(), sound.Click{})
				return
			}
		}
	case item.BoneMeal:
		if bm, ok := tx.Block(targetPos).(item.BoneMealAffected); ok {
			res := bm.BoneMeal(targetPos, tx)
			if res != item.BoneMealResultNone {
				_ = d.inventory.SetItem(chosenSlot, st.Grow(-1))
				tx.PlaySound(pos.Vec3Centre(), sound.Click{})
				return
			}
		}
	case item.Arrow:
		reg := tx.World().EntityRegistry().Config()
		if reg.Arrow != nil {
			_ = d.inventory.SetItem(chosenSlot, st.Grow(-1))
			opts := world.EntitySpawnOpts{Position: spawnPos, Velocity: directionVec.Mul(1.5)}
			tx.AddEntity(reg.Arrow(opts, world.ArrowSpawnConfig{Damage: 2.0}))
			tx.PlaySound(pos.Vec3Centre(), sound.Click{})
			return
		}
	case TNT:
		reg := tx.World().EntityRegistry().Config()
		if reg.TNT != nil {
			_ = d.inventory.SetItem(chosenSlot, st.Grow(-1))
			opts := world.EntitySpawnOpts{Position: spawnPos, Velocity: directionVec.Mul(0.1)}
			tx.AddEntity(reg.TNT(opts, time.Second*4))
			tx.PlaySound(pos.Vec3Centre(), sound.PowerOn{})
			return
		}
	}

	// Default fallback: ejects item into the world with velocity.
	toDrop := st.Grow(-st.Count() + 1)
	_ = d.inventory.SetItem(chosenSlot, st.Grow(-1))
	dropVelocity := directionVec.Mul(0.3)
	dropItemWithVelocity(tx, toDrop, spawnPos, dropVelocity)
	tx.PlaySound(pos.Vec3Centre(), sound.Click{})
}

// BreakInfo ...
func (d Dispenser) BreakInfo() BreakInfo {
	return newBreakInfo(3.5, pickaxeHarvestable, pickaxeEffective, oneOf(NewDispenser())).withBreakHandler(func(pos cube.Pos, tx *world.Tx, _ item.User) {
		if d.inventory != nil {
			for _, i := range d.inventory.Clear() {
				dropItem(tx, i, pos.Vec3Centre())
			}
		}
	})
}

// EncodeItem ...
func (Dispenser) EncodeItem() (name string, meta int16) {
	return "minecraft:dispenser", 0
}

// EncodeBlock ...
func (d Dispenser) EncodeBlock() (string, map[string]any) {
	return "minecraft:dispenser", map[string]any{
		"facing_direction": int32(d.Facing),
		"triggered_bit":    boolByte(d.Triggered),
	}
}

// EncodeNBT encodes dispenser inventory into NBT.
func (d Dispenser) EncodeNBT() map[string]any {
	if d.inventory == nil {
		facing, triggered, customName := d.Facing, d.Triggered, d.CustomName
		//noinspection GoAssignmentToReceiver
		d = NewDispenser()
		d.Facing, d.Triggered, d.CustomName = facing, triggered, customName
	}
	m := map[string]any{
		"Items": nbtconv.InvToNBT(d.inventory),
		"id":    "Dispenser",
	}
	if d.CustomName != "" {
		m["CustomName"] = d.CustomName
	}
	return m
}

// DecodeNBT decodes dispenser inventory from NBT.
func (d Dispenser) DecodeNBT(data map[string]any) any {
	facing, triggered := d.Facing, d.Triggered
	//noinspection GoAssignmentToReceiver
	d = NewDispenser()
	d.Facing = facing
	d.Triggered = triggered
	d.CustomName = nbtconv.String(data, "CustomName")
	nbtconv.InvFromNBT(d.inventory, nbtconv.Slice(data, "Items"))
	return d
}

// allDispensers returns all permutations of dispensers across all 6 faces and triggered states.
func allDispensers() (dispensers []world.Block) {
	for _, f := range cube.Faces() {
		dispensers = append(dispensers, Dispenser{Facing: f, Triggered: false})
		dispensers = append(dispensers, Dispenser{Facing: f, Triggered: true})
	}
	return dispensers
}
