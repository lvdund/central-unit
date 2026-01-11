package context

import "sync/atomic"

// IdGenerator provides thread-safe unique ID generation using atomic operations.
// This replaces the bare int64 fields with "TODO implement mutex" comments.
type IdGenerator struct {
	counter int64
}

// NewIdGenerator creates a new generator starting at the given value.
// IDs will be generated starting from start+1.
func NewIdGenerator(start int64) *IdGenerator {
	return &IdGenerator{counter: start}
}

// Next returns the next unique ID (atomic increment).
// Thread-safe for concurrent use.
func (g *IdGenerator) Next() int64 {
	return atomic.AddInt64(&g.counter, 1)
}

// Current returns the current counter value without incrementing.
// Thread-safe for concurrent use.
func (g *IdGenerator) Current() int64 {
	return atomic.LoadInt64(&g.counter)
}

// Reset resets the counter to a new starting value.
// Use with caution - typically only during testing.
func (g *IdGenerator) Reset(value int64) {
	atomic.StoreInt64(&g.counter, value)
}
