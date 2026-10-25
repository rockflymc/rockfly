package pathing

import (
	"math"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// BlockProvider provides thread-safe voxel queries for pathfinding.
type BlockProvider interface {
	Block(pos cube.Pos) world.Block
}

// AStarConfig customizes A* search limits and entity navigation capabilities.
type AStarConfig struct {
	MaxNodes int
	MaxJump  int
	MaxFall  int
	CanSwim  bool
}

// DefaultAStarConfig returns standard mob navigation parameters.
func DefaultAStarConfig() AStarConfig {
	return AStarConfig{
		MaxNodes: 1500,
		MaxJump:  1,
		MaxFall:  3,
		CanSwim:  true,
	}
}

// AStar implements 3D voxel A* pathfinding with jump, fall, diagonal corner clearance,
// and hazard cost evaluation.
type AStar struct {
	conf AStarConfig
}

// NewAStar creates an AStar pathfinder with the provided configuration.
func NewAStar(conf AStarConfig) *AStar {
	if conf.MaxNodes <= 0 {
		conf.MaxNodes = 1500
	}
	if conf.MaxJump <= 0 {
		conf.MaxJump = 1
	}
	if conf.MaxFall <= 0 {
		conf.MaxFall = 3
	}
	return &AStar{conf: conf}
}

// FindPath calculates the shortest 3D voxel route between start and target.
// Returns a slice of cube.Pos waypoints starting at the step after start and ending at target
// (or the closest reachable node if target is unreachable).
func (a *AStar) FindPath(start, target cube.Pos, provider BlockProvider) []cube.Pos {
	if start == target {
		return nil
	}

	openQueue := make(nodeQueue, 0, 128)
	openSet := make(map[cube.Pos]*PathNode)
	closedSet := make(map[cube.Pos]bool)

	startNode := &PathNode{
		Pos: start,
		G:   0,
		H:   distance3D(start, target),
	}
	startNode.F = startNode.G + startNode.H
	openQueue.pushNode(startNode)
	openSet[start] = startNode

	closestNode := startNode
	minDistance := startNode.H

	nodesEvaluated := 0

	for openQueue.Len() > 0 && nodesEvaluated < a.conf.MaxNodes {
		nodesEvaluated++
		current := openQueue.popNode()
		delete(openSet, current.Pos)
		closedSet[current.Pos] = true

		// Check if target reached
		if current.Pos == target {
			return reconstructPath(current)
		}

		// Track closest node in case target is unreachable
		if current.H < minDistance {
			minDistance = current.H
			closestNode = current
		}

		// Explore neighbors
		neighbors := a.getNeighbors(current.Pos, provider)
		for _, next := range neighbors {
			if closedSet[next.pos] {
				continue
			}

			tentativeG := current.G + next.cost
			existing, inOpen := openSet[next.pos]

			if !inOpen {
				h := distance3D(next.pos, target)
				neighborNode := &PathNode{
					Pos:    next.pos,
					G:      tentativeG,
					H:      h,
					F:      tentativeG + h,
					Parent: current,
				}
				openQueue.pushNode(neighborNode)
				openSet[next.pos] = neighborNode
			} else if tentativeG < existing.G {
				existing.G = tentativeG
				existing.F = tentativeG + existing.H
				existing.Parent = current
				// Re-balance heap node
				openQueue.Swap(existing.index, existing.index)
			}
		}
	}

	// Target unreachable within budget: return path to closest reachable node
	if closestNode != startNode {
		return reconstructPath(closestNode)
	}
	return nil
}

type neighborMove struct {
	pos  cube.Pos
	cost float64
}

var cardinalDirections = [][2]int{
	{1, 0}, {-1, 0}, {0, 1}, {0, -1},
}

var diagonalDirections = [][2]int{
	{1, 1}, {1, -1}, {-1, 1}, {-1, -1},
}

func (a *AStar) getNeighbors(pos cube.Pos, provider BlockProvider) []neighborMove {
	var neighbors []neighborMove

	// 1. Cardinal horizontal moves
	for _, dir := range cardinalDirections {
		dx, dz := dir[0], dir[1]
		dest := pos.Add(cube.Pos{dx, 0, dz})

		// A. Flat move (same Y)
		if a.isStandable(dest, provider) {
			penalty := a.hazardPenalty(dest, provider)
			neighbors = append(neighbors, neighborMove{pos: dest, cost: 1.0 + penalty})
			continue
		}

		// B. Jump up (Y + 1)
		jumpDest := dest.Add(cube.Pos{0, 1, 0})
		// Need head clearance above current position to jump up
		if a.isPassable(pos.Add(cube.Pos{0, 2, 0}), provider) && a.isStandable(jumpDest, provider) {
			penalty := a.hazardPenalty(jumpDest, provider)
			neighbors = append(neighbors, neighborMove{pos: jumpDest, cost: 1.4 + penalty})
			continue
		}

		// C. Step down / fall (Y - 1 to Y - MaxFall)
		for fall := 1; fall <= a.conf.MaxFall; fall++ {
			fallDest := dest.Add(cube.Pos{0, -fall, 0})
			// All air blocks between dest and fallDest must be passable
			if !a.isPassable(dest.Add(cube.Pos{0, -(fall - 1), 0}), provider) {
				break
			}
			if a.isStandable(fallDest, provider) {
				penalty := a.hazardPenalty(fallDest, provider)
				neighbors = append(neighbors, neighborMove{pos: fallDest, cost: 1.0 + float64(fall)*0.3 + penalty})
				break
			}
		}
	}

	// 2. Diagonal horizontal moves (same Y)
	for _, dir := range diagonalDirections {
		dx, dz := dir[0], dir[1]
		dest := pos.Add(cube.Pos{dx, 0, dz})

		// Corner clearance: both adjacent cardinal voxels must be passable to avoid clipping through wall corners
		card1 := pos.Add(cube.Pos{dx, 0, 0})
		card2 := pos.Add(cube.Pos{0, 0, dz})
		if !a.isPassable(card1, provider) || !a.isPassable(card2, provider) {
			continue
		}

		if a.isStandable(dest, provider) {
			penalty := a.hazardPenalty(dest, provider)
			neighbors = append(neighbors, neighborMove{pos: dest, cost: 1.414 + penalty})
		}
	}

	return neighbors
}

// isStandable returns true if the entity can safely stand at footPos:
// foot block is passable, head block (footPos + 1) is passable, and floor below (footPos - 1) is solid.
func (a *AStar) isStandable(footPos cube.Pos, provider BlockProvider) bool {
	if !a.isPassable(footPos, provider) {
		return false
	}
	if !a.isPassable(footPos.Add(cube.Pos{0, 1, 0}), provider) {
		return false
	}
	// Floor beneath feet must be solid
	floor := footPos.Add(cube.Pos{0, -1, 0})
	return a.isSolid(floor, provider)
}

// isPassable checks if an entity can move through pos (air, non-solid vegetation).
func (a *AStar) isPassable(pos cube.Pos, provider BlockProvider) bool {
	b := provider.Block(pos)
	if b == nil {
		return true
	}
	switch b.(type) {
	case block.Air:
		return true
	case block.ShortGrass, block.Fern, block.Flower, block.DoubleFlower, block.DeadBush:
		return true
	case block.Torch, block.RedstoneWire:
		return true
	case block.Water:
		return a.conf.CanSwim
	default:
		// Full blocks with solid models are impassable
		return false
	}
}

// isSolid checks if the block provides a walkable surface to stand on.
func (a *AStar) isSolid(pos cube.Pos, provider BlockProvider) bool {
	b := provider.Block(pos)
	if b == nil {
		return false
	}
	switch b.(type) {
	case block.Air:
		return false
	case block.ShortGrass, block.Fern, block.Flower, block.DoubleFlower, block.DeadBush, block.Torch, block.RedstoneWire:
		return false
	case block.Water, block.Lava:
		return false
	default:
		return true
	}
}

// hazardPenalty returns cost penalties for environmental hazards (fire, lava, cactus, etc.).
func (a *AStar) hazardPenalty(pos cube.Pos, provider BlockProvider) float64 {
	b := provider.Block(pos)
	if b == nil {
		return 0
	}
	floor := provider.Block(pos.Add(cube.Pos{0, -1, 0}))

	switch b.(type) {
	case block.Fire, block.Lava:
		return 1000.0
	case block.Cactus:
		return 50.0
	case block.Water:
		return 2.5
	}

	switch floor.(type) {
	case block.Magma, block.Cactus:
		return 50.0
	}
	return 0
}

func distance3D(a, b cube.Pos) float64 {
	dx := float64(a[0] - b[0])
	dy := float64(a[1] - b[1])
	dz := float64(a[2] - b[2])
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func reconstructPath(node *PathNode) []cube.Pos {
	var path []cube.Pos
	curr := node
	for curr != nil && curr.Parent != nil {
		path = append(path, curr.Pos)
		curr = curr.Parent
	}
	// Reverse so path goes from start to target
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}
