package world

import (
	"errors"
	"fmt"
	"math"
	"sync"
)

// Errors of IDAllocator.
var (
	// ErrNoIDs means every runtime ID is in use.
	ErrNoIDs = errors.New("world: no free runtime object IDs")
	// ErrNotAllocated means an ID was released that is not in use.
	ErrNotAllocated = errors.New("world: object ID not allocated")
)

// idReuseAfter is how many released IDs wait before the oldest is reused, so
// a client is unlikely to mix up an old object with a new one.
const idReuseAfter = 1 << 16

// IDAllocator hands out the IDs of runtime-only objects (decision W3), from
// FirstRuntimeID up. Released IDs are reused oldest first once more than
// 65536 of them wait, or when the range runs out. Safe for concurrent use.
type IDAllocator struct {
	mu         sync.Mutex
	first      int64
	next       int64 // the next fresh ID
	last       int64 // the last ID of the range
	reuseAfter int
	queue      []ObjectID // released IDs, oldest from head
	head       int
	released   map[ObjectID]struct{}
	inUse      int
}

// NewIDAllocator returns an allocator for the runtime range
// FirstRuntimeID–0x7FFFFFFF.
func NewIDAllocator() *IDAllocator {
	return newIDAllocator(FirstRuntimeID, math.MaxInt32, idReuseAfter)
}

func newIDAllocator(first, last ObjectID, reuseAfter int) *IDAllocator {
	return &IDAllocator{
		first:      int64(first),
		next:       int64(first),
		last:       int64(last),
		reuseAfter: reuseAfter,
		released:   make(map[ObjectID]struct{}),
	}
}

// Next returns an unused ID, or ErrNoIDs.
func (a *IDAllocator) Next() (ObjectID, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	waiting := len(a.queue) - a.head
	fresh := a.next <= a.last
	switch {
	case waiting > a.reuseAfter || (waiting > 0 && !fresh):
		id := a.queue[a.head]
		a.head++
		delete(a.released, id)
		a.compact()
		a.inUse++
		return id, nil
	case fresh:
		id := ObjectID(a.next)
		a.next++
		a.inUse++
		return id, nil
	}
	return 0, ErrNoIDs
}

// Release returns id for reuse. Releasing an ID that is not in use returns
// an error wrapping ErrNotAllocated and changes nothing.
func (a *IDAllocator) Release(id ObjectID) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if int64(id) < a.first || int64(id) >= a.next {
		return fmt.Errorf("%w: %#x", ErrNotAllocated, int32(id))
	}
	if _, dup := a.released[id]; dup {
		return fmt.Errorf("%w: %#x (released twice)", ErrNotAllocated, int32(id))
	}
	a.released[id] = struct{}{}
	a.queue = append(a.queue, id)
	a.inUse--
	return nil
}

// InUse returns the number of IDs handed out and not released.
func (a *IDAllocator) InUse() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.inUse
}

// compact drops the consumed head of the queue once it is most of it.
func (a *IDAllocator) compact() {
	if a.head < 1024 || a.head*2 < len(a.queue) {
		return
	}
	n := copy(a.queue, a.queue[a.head:])
	a.queue = a.queue[:n]
	a.head = 0
}
