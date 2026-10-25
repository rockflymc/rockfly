package pathing

import (
	"container/heap"

	"github.com/df-mc/dragonfly/server/block/cube"
)

// PathNode represents a discrete voxel node evaluated in 3D A* pathfinding.
type PathNode struct {
	Pos    cube.Pos
	G      float64   // Cost from start node to current node
	H      float64   // Estimated heuristic cost to target node
	F      float64   // Total evaluated cost (F = G + H)
	Parent *PathNode // Backtracking pointer to reconstruct the final route

	index int // Internal index in priority queue min-heap
}

// nodeQueue implements container/heap.Interface for prioritizing lowest F cost nodes.
type nodeQueue []*PathNode

func (q nodeQueue) Len() int           { return len(q) }
func (q nodeQueue) Less(i, j int) bool { return q[i].F < q[j].F }
func (q nodeQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

func (q *nodeQueue) Push(x any) {
	n := len(*q)
	node := x.(*PathNode)
	node.index = n
	*q = append(*q, node)
}

func (q *nodeQueue) Pop() any {
	old := *q
	n := len(old)
	node := old[n-1]
	old[n-1] = nil
	node.index = -1
	*q = old[0 : n-1]
	return node
}

// pushNode inserts a PathNode into the min-heap.
func (q *nodeQueue) pushNode(n *PathNode) {
	heap.Push(q, n)
}

// popNode removes and returns the lowest F cost node.
func (q *nodeQueue) popNode() *PathNode {
	return heap.Pop(q).(*PathNode)
}
