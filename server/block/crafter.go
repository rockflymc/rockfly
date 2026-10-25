package block

import (
	"math/rand/v2"
	"sync"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/internal/nbtconv"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

var (
	_ world.Block                 = Crafter{}
	_ world.Item                  = Crafter{}
	_ world.NeighbourUpdateTicker = Crafter{}
	_ world.ScheduledTicker       = Crafter{}
	_ world.RedstonePowerAction   = Crafter{}
	_ world.NBTer                 = Crafter{}
	_ Container                   = Crafter{}
)

// CrafterOrientation specifies the front facing direction and top orientation of a crafter block.
type CrafterOrientation uint8

const (
	CrafterDownEast CrafterOrientation = iota
	CrafterDownNorth
	CrafterDownSouth
	CrafterDownWest
	CrafterUpEast
	CrafterUpNorth
	CrafterUpSouth
	CrafterUpWest
	CrafterNorthUp
	CrafterSouthUp
	CrafterEastUp
	CrafterWestUp
)

// String returns the Bedrock block state string representation of the crafter orientation.
func (o CrafterOrientation) String() string {
	switch o {
	case CrafterDownEast:
		return "down_east"
	case CrafterDownNorth:
		return "down_north"
	case CrafterDownSouth:
		return "down_south"
	case CrafterDownWest:
		return "down_west"
	case CrafterUpEast:
		return "up_east"
	case CrafterUpNorth:
		return "up_north"
	case CrafterUpSouth:
		return "up_south"
	case CrafterUpWest:
		return "up_west"
	case CrafterNorthUp:
		return "north_up"
	case CrafterSouthUp:
		return "south_up"
	case CrafterEastUp:
		return "east_up"
	case CrafterWestUp:
		return "west_up"
	default:
		return "north_up"
	}
}

// Facing returns the front face where items are ejected from.
func (o CrafterOrientation) Facing() cube.Face {
	switch o {
	case CrafterDownEast, CrafterDownNorth, CrafterDownSouth, CrafterDownWest:
		return cube.FaceDown
	case CrafterUpEast, CrafterUpNorth, CrafterUpSouth, CrafterUpWest:
		return cube.FaceUp
	case CrafterNorthUp:
		return cube.FaceNorth
	case CrafterSouthUp:
		return cube.FaceSouth
	case CrafterEastUp:
		return cube.FaceEast
	case CrafterWestUp:
		return cube.FaceWest
	default:
		return cube.FaceNorth
	}
}

// allCrafterOrientations returns all 12 valid orientations for the crafter.
func allCrafterOrientations() []CrafterOrientation {
	return []CrafterOrientation{
		CrafterDownEast, CrafterDownNorth, CrafterDownSouth, CrafterDownWest,
		CrafterUpEast, CrafterUpNorth, CrafterUpSouth, CrafterUpWest,
		CrafterNorthUp, CrafterSouthUp, CrafterEastUp, CrafterWestUp,
	}
}

// Crafter is a redstone-powered automated crafting block introduced in Minecraft 1.21.
// It can hold up to 9 item stacks, supports toggling individual disabled slots, emits comparator
// signals based on slot usage, and synthesizes recipes when pulsed with redstone.
type Crafter struct {
	solid
	transparent

	// Orientation specifies the 3D facing orientation of the crafter.
	Orientation CrafterOrientation
	// Crafting specifies if the crafter is actively in the process of crafting (visual glow).
	Crafting bool
	// Triggered specifies if the crafter was triggered by redstone.
	Triggered bool
	// DisabledSlots is a 9-bit bitmask marking which slots are disabled.
	DisabledSlots uint16
	// CustomName is the custom display name of the container.
	CustomName string

	inventory *inventory.Inventory
	viewerMu  *sync.RWMutex
	viewers   map[ContainerViewer]struct{}
}

// NewCrafter initializes a new Crafter with an empty 9-slot inventory.
func NewCrafter() Crafter {
	m := &sync.RWMutex{}
	v := make(map[ContainerViewer]struct{})
	return Crafter{
		Orientation: CrafterNorthUp,
		viewerMu:    m,
		viewers:     v,
		inventory: inventory.New(9, func(slot int, before, after item.Stack) {
			m.RLock()
			defer m.RUnlock()
			for viewer := range v {
				viewer.ViewSlotChange(slot, after)
			}
		}),
	}
}

// ContainerSize returns 9, the slot count of a crafter.
func (Crafter) ContainerSize() int {
	return 9
}

// Inventory returns the crafter's 9-slot item inventory.
func (c Crafter) Inventory(*world.Tx, cube.Pos) *inventory.Inventory {
	return c.inventory
}

// AddViewer adds a viewer to the crafter.
func (c Crafter) AddViewer(v ContainerViewer, _ *world.Tx, _ cube.Pos) {
	c.viewerMu.Lock()
	defer c.viewerMu.Unlock()
	c.viewers[v] = struct{}{}
}

// RemoveViewer removes a viewer from the crafter.
func (c Crafter) RemoveViewer(v ContainerViewer, _ *world.Tx, _ cube.Pos) {
	c.viewerMu.Lock()
	defer c.viewerMu.Unlock()
	delete(c.viewers, v)
}

// IsSlotDisabled returns true if the specified slot (0..8) is disabled.
func (c Crafter) IsSlotDisabled(slot int) bool {
	if slot < 0 || slot >= 9 {
		return false
	}
	return (c.DisabledSlots & (1 << slot)) != 0
}

// SetSlotDisabled sets whether a slot (0..8) is locked/disabled.
func (c *Crafter) SetSlotDisabled(slot int, disabled bool) {
	if slot < 0 || slot >= 9 {
		return
	}
	if disabled {
		c.DisabledSlots |= (1 << slot)
	} else {
		c.DisabledSlots &= ^(1 << slot)
	}
}

// CalculateComparatorSignal computes the redstone signal (0..9) emitted into a redstone comparator.
// The signal equals the count of disabled slots plus the count of filled enabled slots.
func (c Crafter) CalculateComparatorSignal() int {
	count := 0
	for i := 0; i < 9; i++ {
		if c.IsSlotDisabled(i) {
			count++
		} else if c.inventory != nil {
			st, _ := c.inventory.Item(i)
			if !st.Empty() {
				count++
			}
		}
	}
	return count
}

// Activate opens the crafter's container interface for the player.
func (c Crafter) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, u item.User, _ *item.UseContext) bool {
	if opener, ok := u.(ContainerOpener); ok {
		opener.OpenBlockContainer(pos, tx)
		return true
	}
	return false
}

// UseOnBlock places the crafter with appropriate orientation facing away from the player.
func (c Crafter) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) (used bool) {
	pos, _, used = firstReplaceable(tx, pos, face, c)
	if !used {
		return false
	}
	//noinspection GoAssignmentToReceiver
	c = NewCrafter()
	facing := calculateFace(user, pos).Opposite()
	switch facing {
	case cube.FaceDown:
		c.Orientation = CrafterDownNorth
	case cube.FaceUp:
		c.Orientation = CrafterUpNorth
	case cube.FaceNorth:
		c.Orientation = CrafterNorthUp
	case cube.FaceSouth:
		c.Orientation = CrafterSouthUp
	case cube.FaceWest:
		c.Orientation = CrafterWestUp
	case cube.FaceEast:
		c.Orientation = CrafterEastUp
	}

	place(tx, pos, c, user, ctx)
	return placed(ctx)
}

// NeighbourUpdateTick schedules crafting on redstone rising edges.
func (c Crafter) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	powered := tx.RedstonePower(pos) > 0
	if powered && !c.Triggered {
		tx.ScheduleBlockUpdate(pos, c, redstoneTicks(2))
	} else if !powered && c.Triggered {
		c.Triggered = false
		tx.SetBlock(pos, c, nil)
	}
}

// RedstonePowerAction schedules crafting on direct redstone power changes.
func (c Crafter) RedstonePowerAction(pos cube.Pos, tx *world.Tx, oldPower, newPower int) {
	if newPower > 0 && oldPower == 0 && !c.Triggered {
		tx.ScheduleBlockUpdate(pos, c, redstoneTicks(2))
	} else if newPower == 0 && oldPower > 0 && c.Triggered {
		c.Triggered = false
		tx.SetBlock(pos, c, nil)
	}
}

// ScheduledTick executes automated crafting on a redstone pulse.
func (c Crafter) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	if c.Crafting {
		c.Crafting = false
		tx.SetBlock(pos, c, nil)
		return
	}

	if tx.RedstonePower(pos) > 0 {
		c.Triggered = true
		c.Craft(pos, tx)
	}
}

// Craft checks the 3x3 crafting matrix against registered recipes and synthesizes the item.
func (c Crafter) Craft(pos cube.Pos, tx *world.Tx) bool {
	if c.inventory == nil {
		return false
	}

	grid := make([]item.Stack, 9)
	for i := 0; i < 9; i++ {
		grid[i], _ = c.inventory.Item(i)
	}

	for _, r := range recipe.Recipes() {
		if r.Block() != "" && r.Block() != "crafting_table" {
			continue
		}

		if matchContributing, ok := matchRecipe(r, grid); ok {
			// Consume 1 item from each contributing slot.
			for _, slot := range matchContributing {
				st, _ := c.inventory.Item(slot)
				_ = c.inventory.SetItem(slot, st.Grow(-1))
			}

			output := r.Output()[0]
			targetPos := pos.Side(c.Orientation.Facing())
			targetBlock := tx.Block(targetPos)

			// Attempt transfer into adjacent container.
			transferred := false
			if container, isContainer := targetBlock.(Container); isContainer {
				if _, err := container.Inventory(tx, targetPos).AddItem(output); err == nil {
					transferred = true
				}
			}

			if !transferred {
				spawnPos := targetPos.Vec3Centre()
				dropVel := faceVec3(c.Orientation.Facing()).Mul(0.3)
				dropItemWithVelocity(tx, output, spawnPos, dropVel)
			}

			c.Crafting = true
			tx.SetBlock(pos, c, nil)
			tx.ScheduleBlockUpdate(pos, c, redstoneTicks(1))
			tx.PlaySound(pos.Vec3Centre(), sound.Click{})
			return true
		}
	}

	// Crafting failed: play click sound.
	tx.SetBlock(pos, c, nil)
	tx.PlaySound(pos.Vec3Centre(), sound.Click{})
	return false
}

// matchRecipe checks if the 3x3 grid satisfies a given recipe, returning the contributing slots.
func matchRecipe(r recipe.Recipe, grid []item.Stack) ([]int, bool) {
	switch rep := r.(type) {
	case recipe.Shaped:
		w, h := rep.Shape().Width(), rep.Shape().Height()
		if w > 3 || h > 3 {
			return nil, false
		}
		for offY := 0; offY <= 3-h; offY++ {
			for offX := 0; offX <= 3-w; offX++ {
				var contributing []int
				matched := true

				for y := 0; y < 3; y++ {
					for x := 0; x < 3; x++ {
						slot := y*3 + x
						st := grid[slot]

						inPattern := y >= offY && y < offY+h && x >= offX && x < offX+w
						if inPattern {
							reqIndex := (y-offY)*w + (x - offX)
							req := rep.Input()[reqIndex]
							if !matchingRecipeItem(st, req) {
								matched = false
								break
							}
							if !st.Empty() {
								contributing = append(contributing, slot)
							}
						} else if !st.Empty() {
							matched = false
							break
						}
					}
					if !matched {
						break
					}
				}

				if matched {
					return contributing, true
				}
			}
		}
	case recipe.Shapeless:
		inputs := rep.Input()
		var nonEmptySlots []int
		for i, st := range grid {
			if !st.Empty() {
				nonEmptySlots = append(nonEmptySlots, i)
			}
		}
		if len(nonEmptySlots) != len(inputs) {
			return nil, false
		}

		matchedIndices := make(map[int]bool)
		for _, req := range inputs {
			found := false
			for _, slot := range nonEmptySlots {
				if matchedIndices[slot] {
					continue
				}
				if matchingRecipeItem(grid[slot], req) {
					matchedIndices[slot] = true
					found = true
					break
				}
			}
			if !found {
				return nil, false
			}
		}
		return nonEmptySlots, true
	}
	return nil, false
}

// matchingRecipeItem checks if a candidate stack satisfies an expected recipe item.
func matchingRecipeItem(has item.Stack, expected recipe.Item) bool {
	if expected == nil || expected.Empty() {
		return has.Empty()
	}
	if has.Empty() {
		return false
	}
	switch exp := expected.(type) {
	case item.Stack:
		_, variants := exp.Value("variants")
		if !variants {
			return has.Comparable(exp)
		}
		name1, _ := has.Item().EncodeItem()
		name2, _ := exp.Item().EncodeItem()
		return name1 == name2
	case recipe.ItemTag:
		name, _ := has.Item().EncodeItem()
		return exp.Contains(name)
	}
	return false
}

// BreakInfo ...
func (c Crafter) BreakInfo() BreakInfo {
	return newBreakInfo(3.5, pickaxeHarvestable, pickaxeEffective, oneOf(NewCrafter())).withBreakHandler(func(pos cube.Pos, tx *world.Tx, _ item.User) {
		if c.inventory != nil {
			for _, i := range c.inventory.Clear() {
				dropItem(tx, i, pos.Vec3Centre())
			}
		}
	})
}

// EncodeItem ...
func (Crafter) EncodeItem() (name string, meta int16) {
	return "minecraft:crafter", 0
}

// EncodeBlock ...
func (c Crafter) EncodeBlock() (string, map[string]any) {
	return "minecraft:crafter", map[string]any{
		"orientation":   c.Orientation.String(),
		"crafting":      boolByte(c.Crafting),
		"triggered_bit": boolByte(c.Triggered),
	}
}

// EncodeNBT encodes crafter inventory and disabled slots into NBT.
func (c Crafter) EncodeNBT() map[string]any {
	if c.inventory == nil {
		orient, crafting, triggered, disabled, customName := c.Orientation, c.Crafting, c.Triggered, c.DisabledSlots, c.CustomName
		//noinspection GoAssignmentToReceiver
		c = NewCrafter()
		c.Orientation, c.Crafting, c.Triggered, c.DisabledSlots, c.CustomName = orient, crafting, triggered, disabled, customName
	}
	m := map[string]any{
		"Items":          nbtconv.InvToNBT(c.inventory),
		"disabled_slots": int32(c.DisabledSlots),
		"id":             "Crafter",
	}
	if c.CustomName != "" {
		m["CustomName"] = c.CustomName
	}
	return m
}

// DecodeNBT decodes crafter inventory and disabled slots from NBT.
func (c Crafter) DecodeNBT(data map[string]any) any {
	orient, crafting, triggered := c.Orientation, c.Crafting, c.Triggered
	//noinspection GoAssignmentToReceiver
	c = NewCrafter()
	c.Orientation = orient
	c.Crafting = crafting
	c.Triggered = triggered
	c.DisabledSlots = uint16(nbtconv.Int32(data, "disabled_slots"))
	c.CustomName = nbtconv.String(data, "CustomName")
	nbtconv.InvFromNBT(c.inventory, nbtconv.Slice(data, "Items"))
	return c
}

// allCrafters returns all 48 permutations of crafters.
func allCrafters() (crafters []world.Block) {
	for _, o := range allCrafterOrientations() {
		for _, crafting := range []bool{false, true} {
			for _, triggered := range []bool{false, true} {
				crafters = append(crafters, Crafter{
					Orientation: o,
					Crafting:    crafting,
					Triggered:   triggered,
				})
			}
		}
	}
	return crafters
}
