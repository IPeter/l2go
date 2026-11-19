// Package netutil holds connection handling shared by the servers.
package netutil

import (
	"io"
	"sync"
)

// Registry tracks live connections by a unique ID. It is safe for
// concurrent use. The zero value is ready to use.
type Registry[T io.Closer] struct {
	mu     sync.Mutex
	nextID uint64
	items  map[uint64]T
	closed bool
}

// Add registers v and returns its ID. After CloseAll it refuses new items
// and returns false; the caller then owns v and must close it.
func (r *Registry[T]) Add(v T) (id uint64, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return 0, false
	}
	if r.items == nil {
		r.items = make(map[uint64]T)
	}
	r.nextID++
	r.items[r.nextID] = v
	return r.nextID, true
}

// Remove unregisters the item with the given ID. It does not close it.
func (r *Registry[T]) Remove(id uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.items, id)
}

// Get returns the item with the given ID.
func (r *Registry[T]) Get(id uint64) (T, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.items[id]
	return v, ok
}

// Len returns the number of registered items.
func (r *Registry[T]) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.items)
}

// CloseAll closes every registered item and makes later Add calls fail.
// Items stay registered until their owners Remove them.
func (r *Registry[T]) CloseAll() {
	r.mu.Lock()
	r.closed = true
	items := make([]T, 0, len(r.items))
	for _, v := range r.items {
		items = append(items, v)
	}
	r.mu.Unlock()

	for _, v := range items {
		_ = v.Close()
	}
}
