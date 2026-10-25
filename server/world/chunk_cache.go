package world

import (
	"container/list"
	"sync"
)

// cacheEntry holds a chunk position and its cached Column.
type cacheEntry struct {
	pos ChunkPos
	col *Column
}

// ChunkCache is a thread-safe LRU cache for Column structures.
// It keeps recently accessed chunks in memory to eliminate redundant disk reads
// during high-speed player flight and frequent border crossing.
type ChunkCache struct {
	mu       sync.Mutex
	capacity int
	items    map[ChunkPos]*list.Element
	evictList *list.List
}

// NewChunkCache creates a new ChunkCache with the specified capacity.
func NewChunkCache(capacity int) *ChunkCache {
	if capacity <= 0 {
		capacity = 512
	}
	return &ChunkCache{
		capacity:  capacity,
		items:     make(map[ChunkPos]*list.Element),
		evictList: list.New(),
	}
}

// Get returns the cached Column for the specified ChunkPos, moving it to the front of the LRU list.
func (c *ChunkCache) Get(pos ChunkPos) (*Column, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[pos]
	if !ok {
		return nil, false
	}
	c.evictList.MoveToFront(elem)
	return elem.Value.(*cacheEntry).col, true
}

// Put inserts or updates a Column in the cache, evicting the least recently used item if full.
func (c *ChunkCache) Put(pos ChunkPos, col *Column) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[pos]; ok {
		c.evictList.MoveToFront(elem)
		elem.Value.(*cacheEntry).col = col
		return
	}

	for c.evictList.Len() >= c.capacity {
		c.evictOldest()
	}

	entry := &cacheEntry{pos: pos, col: col}
	elem := c.evictList.PushFront(entry)
	c.items[pos] = elem
}

// Remove removes the chunk at pos from the cache.
func (c *ChunkCache) Remove(pos ChunkPos) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[pos]; ok {
		c.evictList.Remove(elem)
		delete(c.items, pos)
	}
}

// Len returns the number of items currently in the cache.
func (c *ChunkCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.evictList.Len()
}

// Clear flushes all entries from the cache.
func (c *ChunkCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[ChunkPos]*list.Element)
	c.evictList.Init()
}

// evictOldest removes the least recently used entry. Caller must hold c.mu.
func (c *ChunkCache) evictOldest() {
	elem := c.evictList.Back()
	if elem != nil {
		c.evictList.Remove(elem)
		entry := elem.Value.(*cacheEntry)
		delete(c.items, entry.pos)
	}
}
