package starlight

// PackedQueue is a high-performance ring buffer for PackedNode values.
// Because PackedNode is a primitive uint64, elements are stored contiguously in memory
// without pointer indirection, achieving maximal cache efficiency.
type PackedQueue struct {
	nodes []PackedNode
	head  int
	tail  int
	size  int
}

// NewPackedQueue creates a new PackedQueue with the specified initial capacity.
func NewPackedQueue(capacity int) *PackedQueue {
	if capacity < 16 {
		capacity = 16
	}
	return &PackedQueue{
		nodes: make([]PackedNode, capacity),
	}
}

// Push appends a node to the queue, growing the underlying buffer if needed.
func (q *PackedQueue) Push(n PackedNode) {
	if q.size == len(q.nodes) {
		q.grow()
	}
	q.nodes[q.tail] = n
	q.tail = (q.tail + 1) % len(q.nodes)
	q.size++
}

// Pop removes and returns the oldest node from the queue.
func (q *PackedQueue) Pop() (PackedNode, bool) {
	if q.size == 0 {
		return 0, false
	}
	n := q.nodes[q.head]
	q.head = (q.head + 1) % len(q.nodes)
	q.size--
	return n, true
}

// Empty returns whether the queue has no elements.
func (q *PackedQueue) Empty() bool {
	return q.size == 0
}

// Len returns the number of nodes currently in the queue.
func (q *PackedQueue) Len() int {
	return q.size
}

// Clear resets the queue indices without releasing allocated slice capacity.
func (q *PackedQueue) Clear() {
	q.head = 0
	q.tail = 0
	q.size = 0
}

// grow doubles the ring buffer storage and aligns head at index 0.
func (q *PackedQueue) grow() {
	newCap := len(q.nodes) << 1
	newNodes := make([]PackedNode, newCap)
	if q.head < q.tail {
		copy(newNodes, q.nodes[q.head:q.tail])
	} else {
		n := copy(newNodes, q.nodes[q.head:])
		copy(newNodes[n:], q.nodes[:q.tail])
	}
	q.head = 0
	q.tail = q.size
	q.nodes = newNodes
}
