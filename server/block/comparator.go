package block

import (
	"math"
	"math/rand/v2"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/internal/nbtconv"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

var (
	_ world.Block                    = Comparator{}
	_ world.Item                     = Comparator{}
	_ world.ScheduledTicker          = Comparator{}
	_ world.NeighbourUpdateTicker    = Comparator{}
	_ world.RedstonePowerSource      = Comparator{}
	_ world.RedstoneWeakBlockPowerer = Comparator{}
	_ world.NBTer                    = Comparator{}
)

// ComparatorMode represents the operating mode of a redstone comparator.
type ComparatorMode uint8

const (
	// ComparatorComparison compares the rear input against the side inputs.
	// If rear >= maxSide, output is rear; otherwise output is 0.
	ComparatorComparison ComparatorMode = iota
	// ComparatorSubtraction subtracts the maximum side input from the rear input:
	// output = max(0, rear - maxSide).
	ComparatorSubtraction
)

// Comparator is a redstone component used to maintain, compare, or subtract signal strengths,
// or to measure the fullness of container blocks.
type Comparator struct {
	transparent
	empty

	// Facing is the horizontal direction the comparator outputs towards.
	Facing cube.Direction
	// Mode is the operating mode of the comparator (Comparison or Subtraction).
	Mode ComparatorMode
	// Powered indicates whether the comparator is currently outputting a signal (> 0).
	Powered bool
	// OutputSignal is the signal strength currently emitted by the comparator (0-15).
	OutputSignal int
}

// Model returns the flat diode bounding box (2 pixels high).
func (Comparator) Model() world.BlockModel {
	return diodeModel{}
}

// HasLiquidDrops returns true so water washes away the comparator.
func (Comparator) HasLiquidDrops() bool {
	return true
}

// SideClosed ...
func (Comparator) SideClosed(cube.Pos, cube.Pos, *world.Tx) bool {
	return false
}

// BreakInfo defines drops and break behavior.
func (c Comparator) BreakInfo() BreakInfo {
	return newBreakInfo(0, alwaysHarvestable, nothingEffective, oneOf(Comparator{}))
}

// UseOnBlock places the comparator on top of a solid surface facing away from the player.
func (c Comparator) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) bool {
	if tx == nil {
		return false
	}
	pos, face, used := firstReplaceable(tx, pos, face, c)
	if !used || face != cube.FaceUp {
		return false
	}
	if !comparatorSupported(tx, pos) {
		return false
	}
	if user != nil {
		c.Facing = user.Rotation().Direction().Opposite()
	}
	c.Mode = ComparatorComparison
	c.Powered = false
	c.OutputSignal = 0

	place(tx, pos, c, user, ctx)
	ok := placed(ctx)
	if ok {
		if c.calculateOutput(pos, tx) > 0 {
			tx.ScheduleBlockUpdate(pos, c, redstoneTicks(1))
		}
	}
	return ok
}

// Activate right-clicks toggle the comparator between Comparison and Subtraction modes.
func (c Comparator) Activate(pos cube.Pos, _ cube.Face, tx *world.Tx, _ item.User, _ *item.UseContext) bool {
	if tx == nil {
		return false
	}
	if c.Mode == ComparatorComparison {
		c.Mode = ComparatorSubtraction
	} else {
		c.Mode = ComparatorComparison
	}
	tx.SetBlock(pos, c, nil)
	tx.PlaySound(pos.Vec3Centre(), sound.Click{})
	tx.ScheduleBlockUpdate(pos, c, redstoneTicks(1))
	return true
}

// NeighbourUpdateTick checks for breakage when unsupported and schedules output recalculation.
func (c Comparator) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	if tx == nil {
		return
	}
	if !comparatorSupported(tx, pos) {
		breakBlock(c, pos, tx)
		return
	}
	targetOutput := c.calculateOutput(pos, tx)
	if targetOutput != c.OutputSignal {
		tx.ScheduleBlockUpdate(pos, c, redstoneTicks(1))
	}
}

// ScheduledTick recalculates output after the standard 1-redstone-tick delay and updates block state.
func (c Comparator) ScheduledTick(pos cube.Pos, tx *world.Tx, _ *rand.Rand) {
	if tx == nil {
		return
	}
	current, ok := tx.Block(pos).(Comparator)
	if !ok {
		return
	}
	output := current.calculateOutput(pos, tx)
	powered := output > 0
	if current.OutputSignal != output || current.Powered != powered {
		current.OutputSignal = output
		current.Powered = powered
		tx.SetBlock(pos, current, &world.SetOpts{DisableRedstoneUpdates: true})
		tx.Redstone().ScheduleUpdate(pos)
	}
}

// calculateOutput determines the target output signal based on mode, rear input, and side inputs.
func (c Comparator) calculateOutput(pos cube.Pos, tx *world.Tx) int {
	if tx == nil || pos.OutOfBounds(tx.Range()) {
		return 0
	}
	rearSignal := c.readRearSignal(pos, tx)
	if rearSignal < 0 {
		rearSignal = 0
	} else if rearSignal > 15 {
		rearSignal = 15
	}

	sideSignal := c.readMaxSideSignal(pos, tx)
	if sideSignal < 0 {
		sideSignal = 0
	} else if sideSignal > 15 {
		sideSignal = 15
	}

	if c.Mode == ComparatorComparison {
		if rearSignal >= sideSignal {
			return rearSignal
		}
		return 0
	}
	// Subtraction mode.
	diff := rearSignal - sideSignal
	if diff < 0 {
		return 0
	} else if diff > 15 {
		return 15
	}
	return diff
}

// readRearSignal reads the strongest signal arriving from the rear (redstone wire/source or container fullness).
func (c Comparator) readRearSignal(pos cube.Pos, tx *world.Tx) int {
	if tx == nil || pos.OutOfBounds(tx.Range()) {
		return 0
	}
	rearFace := c.Facing.Opposite().Face()
	rearPos := pos.Side(rearFace)
	if rearPos.OutOfBounds(tx.Range()) {
		return 0
	}

	redstonePower := tx.RedstonePowerFrom(pos, rearFace)
	containerSignal := c.readContainerAt(rearPos, tx)

	// In vanilla, containers can also be read through an opaque solid conductive block directly behind the comparator.
	if containerSignal == 0 {
		rearBlock := tx.Block(rearPos)
		if rearBlock != nil && rearBlock.Model().FaceSolid(rearPos, rearFace, tx) {
			behindSolidPos := rearPos.Side(rearFace)
			if !behindSolidPos.OutOfBounds(tx.Range()) {
				containerSignal = c.readContainerAt(behindSolidPos, tx)
			}
		}
	}

	if containerSignal > redstonePower {
		return containerSignal
	}
	return redstonePower
}

// readContainerAt inspects pos for a container or lectern and computes its comparator signal.
func (Comparator) readContainerAt(pos cube.Pos, tx *world.Tx) int {
	if tx == nil || pos.OutOfBounds(tx.Range()) {
		return 0
	}
	b := tx.Block(pos)
	if b == nil {
		return 0
	}
	if container, ok := b.(Container); ok {
		return CalculateContainerSignal(container.Inventory(tx, pos))
	}
	if lectern, ok := b.(Lectern); ok {
		return CalculateLecternSignal(lectern)
	}
	return 0
}

// readMaxSideSignal reads the strongest signal arriving from the left or right sides.
func (c Comparator) readMaxSideSignal(pos cube.Pos, tx *world.Tx) int {
	if tx == nil || pos.OutOfBounds(tx.Range()) {
		return 0
	}
	leftFace := c.Facing.RotateLeft().Face()
	rightFace := c.Facing.RotateRight().Face()

	leftPower := tx.RedstonePowerFrom(pos, leftFace)
	rightPower := tx.RedstonePowerFrom(pos, rightFace)

	// Check direct adjacent repeater or comparator pointing into this side.
	leftBlockPower := c.readDirectSideComponent(pos.Side(leftFace), tx, leftFace.Opposite())
	rightBlockPower := c.readDirectSideComponent(pos.Side(rightFace), tx, rightFace.Opposite())

	return max(leftPower, rightPower, leftBlockPower, rightBlockPower)
}

// readDirectSideComponent inspects sidePos for an adjacent repeater or comparator facing towardsThis.
func (Comparator) readDirectSideComponent(sidePos cube.Pos, tx *world.Tx, towardsThis cube.Face) int {
	if tx == nil || sidePos.OutOfBounds(tx.Range()) {
		return 0
	}
	b := tx.Block(sidePos)
	if b == nil {
		return 0
	}
	if r, ok := b.(Repeater); ok {
		if r.Facing.Face() == towardsThis && r.Powered {
			return 15
		}
	}
	if comp, ok := b.(Comparator); ok {
		if comp.Facing.Face() == towardsThis && comp.Powered {
			sig := comp.OutputSignal
			if sig < 0 {
				return 0
			} else if sig > 15 {
				return 15
			}
			return sig
		}
	}
	return 0
}

// RedstonePower emits signal strength from the front face when powered.
func (c Comparator) RedstonePower(_ cube.Pos, _ *world.Tx, face cube.Face) int {
	if c.Powered && face == c.Facing.Face() {
		sig := c.OutputSignal
		if sig < 0 {
			return 0
		} else if sig > 15 {
			return 15
		}
		return sig
	}
	return 0
}

// RedstoneWeaklyPowersBlocks allows adjacent conductive blocks in front to be weakly powered.
func (Comparator) RedstoneWeaklyPowersBlocks() bool {
	return true
}

// EncodeItem encodes the inventory item representation.
func (Comparator) EncodeItem() (name string, meta int16) {
	return "minecraft:comparator", 0
}

// EncodeBlock maps the comparator into canonical Bedrock block states.
func (c Comparator) EncodeBlock() (string, map[string]any) {
	name := "minecraft:unpowered_comparator"
	lit := byte(0)
	if c.Powered {
		name = "minecraft:powered_comparator"
		lit = byte(1)
	}
	subtract := byte(0)
	if c.Mode == ComparatorSubtraction {
		subtract = byte(1)
	}
	return name, map[string]any{
		"minecraft:cardinal_direction": c.Facing.String(),
		"output_lit_bit":               lit,
		"output_subtract_bit":          subtract,
	}
}

// EncodeNBT serializes comparator output signal strength to block entity NBT.
func (c Comparator) EncodeNBT() map[string]any {
	return map[string]any{
		"id":           "Comparator",
		"OutputSignal": int32(c.OutputSignal),
	}
}

// DecodeNBT deserializes comparator output signal strength from block entity NBT.
func (c Comparator) DecodeNBT(m map[string]any) any {
	sig := int(nbtconv.Int32(m, "OutputSignal"))
	if sig < 0 {
		sig = 0
	} else if sig > 15 {
		sig = 15
	}
	c.OutputSignal = sig
	c.Powered = c.OutputSignal > 0
	return c
}

// allComparators generates all 16 canonical combinations (4 directions x 2 modes x 2 powered states).
func allComparators() (comparators []world.Block) {
	for _, f := range cube.Directions() {
		for _, mode := range []ComparatorMode{ComparatorComparison, ComparatorSubtraction} {
			for _, powered := range []bool{false, true} {
				comparators = append(comparators, Comparator{Facing: f, Mode: mode, Powered: powered})
			}
		}
	}
	return
}

// comparatorSupported reports whether the support block beneath pos can hold a comparator.
func comparatorSupported(tx *world.Tx, pos cube.Pos) bool {
	if tx == nil || pos.OutOfBounds(tx.Range()) {
		return false
	}
	below := pos.Side(cube.FaceDown)
	if below.OutOfBounds(tx.Range()) {
		return false
	}
	b := tx.Block(below)
	if b == nil {
		return false
	}
	return b.Model().FaceSolid(below, cube.FaceUp, tx)
}

// CalculateContainerSignal computes the 0-15 comparator signal strength from a container inventory.
func CalculateContainerSignal(inv *inventory.Inventory) int {
	if inv == nil {
		return 0
	}
	slots := inv.Slots()
	if len(slots) == 0 {
		return 0
	}
	var (
		itemCount float64
		empty     = true
	)
	for _, stack := range slots {
		if stack.Empty() {
			continue
		}
		count := stack.Count()
		if count <= 0 {
			continue
		}
		empty = false
		maxCount := stack.MaxCount()
		if maxCount <= 0 {
			maxCount = 64
		}
		itemCount += float64(count) / float64(maxCount)
	}
	if empty {
		return 0
	}
	factor := itemCount / float64(len(slots))
	if math.IsNaN(factor) || math.IsInf(factor, 0) || factor <= 0 {
		return 0
	}
	val := int(math.Floor(1.0 + factor*14.0))
	if val < 0 {
		return 0
	} else if val > 15 {
		return 15
	}
	return val
}

// CalculateLecternSignal computes the comparator signal strength from a lectern with a book.
func CalculateLecternSignal(l Lectern) int {
	if l.Book.Empty() {
		return 0
	}
	r, ok := l.Book.Item().(readableBook)
	if !ok {
		return 0
	}
	pages := r.TotalPages()
	if pages <= 1 {
		return 15
	}
	page := l.Page
	if page < 0 {
		page = 0
	} else if page >= pages {
		page = pages - 1
	}
	val := int(math.Floor(float64(page)/float64(pages-1)*14.0)) + 1
	if val < 0 {
		return 0
	} else if val > 15 {
		return 15
	}
	return val
}
