package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// MaxPistonPushBlocks is the maximum number of blocks that can be pushed or pulled by a piston.
const MaxPistonPushBlocks = 12

// PistonSolver calculates and executes the kinematic movement of blocks and adhesion networks
// when a piston extends or retracts.
type PistonSolver struct {
	tx        *world.Tx
	pistonPos cube.Pos
	facing    cube.Face
	moveDir   cube.Face
	pull      bool

	toMove    []cube.Pos
	toDestroy []cube.Pos
	visited   map[cube.Pos]bool
}

// NewPistonSolver creates a new solver instance for extending or retracting a piston.
func NewPistonSolver(tx *world.Tx, pistonPos cube.Pos, facing cube.Face, pull bool) *PistonSolver {
	moveDir := facing
	if pull {
		moveDir = facing.Opposite()
	}
	return &PistonSolver{
		tx:        tx,
		pistonPos: pistonPos,
		facing:    facing,
		moveDir:   moveDir,
		pull:      pull,
		visited:   make(map[cube.Pos]bool),
	}
}

// Solve calculates the affected blocks and executes the movement if possible.
// Returns true if the operation succeeds, or false if blocked by an immovable block or exceeding push limits.
func (s *PistonSolver) Solve() bool {
	if s.pull {
		if !s.calculatePull() {
			return false
		}
	} else {
		if !s.calculatePush() {
			return false
		}
	}

	// 1. Destroy breakable blocks.
	for _, pos := range s.toDestroy {
		destroyBlock(s.tx, pos)
	}

	// 2. Move blocks to their new positions.
	s.executeMove()
	return true
}

// calculatePush traces the push chain and adhesion networks for piston extension.
func (s *PistonSolver) calculatePush() bool {
	headPos := s.pistonPos.Side(s.facing)
	firstBlock := s.tx.Block(headPos)
	if firstBlock == nil || isPushReplaceable(firstBlock) {
		return true
	}
	if isDestroyableOnPush(firstBlock) {
		s.toDestroy = append(s.toDestroy, headPos)
		return true
	}
	if isImmovable(s.tx, headPos, firstBlock) {
		return false
	}

	s.visited[s.pistonPos] = true
	s.visited[headPos] = true
	queue := []cube.Pos{headPos}
	inMove := make(map[cube.Pos]bool)

	for len(queue) > 0 {
		pos := queue[0]
		queue = queue[1:]

		b := s.tx.Block(pos)
		if b == nil || isPushReplaceable(b) {
			continue
		}
		if isDestroyableOnPush(b) {
			s.toDestroy = append(s.toDestroy, pos)
			continue
		}
		if isImmovable(s.tx, pos, b) {
			return false
		}

		if !inMove[pos] {
			inMove[pos] = true
			s.toMove = append(s.toMove, pos)
			if len(s.toMove) > MaxPistonPushBlocks {
				return false
			}
		}

		// 1. Block directly ahead in moveDir is pushed forward.
		frontPos := pos.Side(s.moveDir)
		if !s.visited[frontPos] {
			s.visited[frontPos] = true
			queue = append(queue, frontPos)
		}

		// 2. Adhesion networks (Slime and Honey).
		if isSlime(b) || isHoney(b) {
			for _, face := range cube.Faces() {
				adjPos := pos.Side(face)
				if s.visited[adjPos] {
					continue
				}
				adjBlock := s.tx.Block(adjPos)
				if adjBlock == nil || isPushReplaceable(adjBlock) {
					continue
				}
				// Slime and Honey do NOT adhere to each other.
				if isSlime(b) && isHoney(adjBlock) {
					continue
				}
				if isHoney(b) && isSlime(adjBlock) {
					continue
				}
				// Glazed terracotta never adheres to slime or honey.
				if isGlazedTerracotta(adjBlock) {
					continue
				}
				// If an adhered neighbour is immovable, the whole structure cannot move.
				if isImmovable(s.tx, adjPos, adjBlock) {
					return false
				}

				s.visited[adjPos] = true
				queue = append(queue, adjPos)
			}
		}
	}

	return true
}

// calculatePull traces the pull chain for sticky piston retraction.
func (s *PistonSolver) calculatePull() bool {
	armPos := s.pistonPos.Side(s.facing)
	targetPos := armPos.Side(s.facing)

	b := s.tx.Block(targetPos)
	if b == nil || isPushReplaceable(b) || isDestroyableOnPush(b) || isImmovable(s.tx, targetPos, b) {
		return true
	}

	s.visited[s.pistonPos] = true
	s.visited[armPos] = true
	s.visited[targetPos] = true

	queue := []cube.Pos{targetPos}
	inMove := make(map[cube.Pos]bool)

	for len(queue) > 0 {
		pos := queue[0]
		queue = queue[1:]

		curr := s.tx.Block(pos)
		if curr == nil || isPushReplaceable(curr) {
			continue
		}
		if isImmovable(s.tx, pos, curr) {
			s.toMove = nil
			return true
		}

		if !inMove[pos] {
			inMove[pos] = true
			s.toMove = append(s.toMove, pos)
			if len(s.toMove) > MaxPistonPushBlocks {
				s.toMove = nil
				return true
			}
		}

		// Destination check for this block.
		destPos := pos.Side(s.moveDir)
		if destPos == s.pistonPos {
			s.toMove = nil
			return true
		}
		if destPos != armPos && !inMove[destPos] {
			destBlock := s.tx.Block(destPos)
			if destBlock != nil && !isPushReplaceable(destBlock) && !isDestroyableOnPush(destBlock) {
				s.toMove = nil
				return true
			}
		}

		// Adhesion check for Slime / Honey.
		if isSlime(curr) || isHoney(curr) {
			for _, face := range cube.Faces() {
				adjPos := pos.Side(face)
				if s.visited[adjPos] {
					continue
				}
				adjBlock := s.tx.Block(adjPos)
				if adjBlock == nil || isPushReplaceable(adjBlock) {
					continue
				}
				if isSlime(curr) && isHoney(adjBlock) {
					continue
				}
				if isHoney(curr) && isSlime(adjBlock) {
					continue
				}
				if isGlazedTerracotta(adjBlock) {
					continue
				}
				if isImmovable(s.tx, adjPos, adjBlock) {
					s.toMove = nil
					return true
				}

				s.visited[adjPos] = true
				queue = append(queue, adjPos)
			}
		}
	}

	return true
}

type movedBlock struct {
	pos    cube.Pos
	target cube.Pos
	block  world.Block
}

// executeMove shifts all blocks in s.toMove into their destination positions.
func (s *PistonSolver) executeMove() {
	if len(s.toMove) == 0 {
		return
	}

	saved := make([]movedBlock, 0, len(s.toMove))
	for _, pos := range s.toMove {
		b := s.tx.Block(pos)
		saved = append(saved, movedBlock{
			pos:    pos,
			target: pos.Side(s.moveDir),
			block:  b,
		})
	}

	// Clear all original positions first to avoid collision.
	for _, item := range saved {
		s.tx.SetBlock(item.pos, Air{}, nil)
	}

	// Place all blocks at target positions.
	for _, item := range saved {
		s.tx.SetBlock(item.target, item.block, nil)
	}

	// Nudge entities standing in or on moved blocks.
	moveVec := faceVec3(s.moveDir)
	for _, item := range saved {
		box := cube.Box(
			float64(item.target[0]), float64(item.target[1]), float64(item.target[2]),
			float64(item.target[0]+1), float64(item.target[1]+2), float64(item.target[2]+1),
		)
		for e := range s.tx.EntitiesWithin(box) {
			if m, ok := e.(interface{ Move(mgl64.Vec3) }); ok {
				m.Move(moveVec)
			}
			if isSlime(item.block) && s.moveDir == cube.FaceUp {
				if v, ok := e.(interface{ SetVelocity(mgl64.Vec3) }); ok {
					v.SetVelocity(mgl64.Vec3{0, 1.0, 0})
				}
			}
		}
	}
}

// faceVec3 converts a cube.Face into a unit 3D movement vector.
func faceVec3(f cube.Face) mgl64.Vec3 {
	switch f {
	case cube.FaceUp:
		return mgl64.Vec3{0, 1, 0}
	case cube.FaceDown:
		return mgl64.Vec3{0, -1, 0}
	case cube.FaceNorth:
		return mgl64.Vec3{0, 0, -1}
	case cube.FaceSouth:
		return mgl64.Vec3{0, 0, 1}
	case cube.FaceWest:
		return mgl64.Vec3{-1, 0, 0}
	case cube.FaceEast:
		return mgl64.Vec3{1, 0, 0}
	}
	return mgl64.Vec3{}
}

// isPushReplaceable returns true if a block can be replaced by a pushed block without breaking into items.
func isPushReplaceable(b world.Block) bool {
	switch b.(type) {
	case Air, Water, Lava:
		return true
	}
	return false
}

// isDestroyableOnPush returns true if a block breaks and drops as an item when pushed by a piston.
func isDestroyableOnPush(b world.Block) bool {
	if pb, ok := b.(interface{ PistonBreak() bool }); ok {
		return pb.PistonBreak()
	}
	switch b.(type) {
	case Torch, CopperTorch, RedstoneTorch:
		return true
	case RedstoneWire, Repeater, Comparator:
		return true
	case Flower, ShortGrass, DoubleTallGrass, DeadBush:
		return true
	case Carpet, SugarCane, Cactus:
		return true
	}
	return false
}

// isImmovable returns true if a block cannot be moved by pistons under any circumstances.
func isImmovable(tx *world.Tx, pos cube.Pos, b world.Block) bool {
	if imm, ok := b.(interface{ PistonImmovable() bool }); ok {
		return imm.PistonImmovable()
	}
	switch v := b.(type) {
	case Bedrock, Obsidian, ReinforcedDeepslate, Barrier, EndPortal, EndPortalFrame, Portal:
		return true
	case PistonArm:
		return true
	case Piston:
		// Extended pistons cannot be moved.
		headPos := pos.Side(v.Facing)
		if arm, ok := tx.Block(headPos).(PistonArm); ok && arm.Facing == v.Facing {
			return true
		}
	}
	return false
}

// isSlime returns true if the block is a slime block.
func isSlime(b world.Block) bool {
	_, ok := b.(Slime)
	return ok
}

// isHoney returns true if the block is a honey block.
func isHoney(b world.Block) bool {
	_, ok := b.(Honey)
	return ok
}

// isGlazedTerracotta returns true if the block is glazed terracotta.
func isGlazedTerracotta(b world.Block) bool {
	_, ok := b.(GlazedTerracotta)
	return ok
}

// destroyBlock breaks a block, drops its items, and clears it to Air.
func destroyBlock(tx *world.Tx, pos cube.Pos) {
	b := tx.Block(pos)
	if b == nil || isPushReplaceable(b) {
		return
	}
	if bi, ok := b.(interface{ BreakInfo() BreakInfo }); ok {
		info := bi.BreakInfo()
		if create := tx.World().EntityRegistry().Config().Item; create != nil {
			for _, drop := range info.Drops(item.ToolNone{}, nil) {
				dropItem(tx, drop, pos.Vec3Centre())
			}
		}
	}
	tx.SetBlock(pos, Air{}, nil)
}
