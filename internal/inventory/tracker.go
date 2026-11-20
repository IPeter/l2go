package inventory

import (
	"slices"

	"github.com/IPeter/l2go/internal/storage"
)

// Tracker collects the changes to one character's items since its last
// save (decision I2): the items to upsert, the IDs of items that no longer
// exist, and the IDs of items that left the character. A character's
// inventory and warehouse share one tracker, so an item moving between them
// is one upsert.
//
// An item that leaves and comes back before a save is an upsert, not a
// delete: each kind of change replaces the earlier ones of the same item.
type Tracker struct {
	upserts   []*Item
	destroyed []int32
	left      []int32
}

// NewTracker returns an empty tracker.
func NewTracker() *Tracker { return &Tracker{} }

// upsert records a new or changed item.
func (t *Tracker) upsert(it *Item) {
	id := it.objectID
	t.destroyed = deleteID(t.destroyed, id)
	t.left = deleteID(t.left, id)
	it.dirty = true
	if !slices.Contains(t.upserts, it) {
		t.upserts = append(t.upserts, it)
	}
}

// destroy records an item that no longer exists.
func (t *Tracker) destroy(it *Item) {
	t.forget(it)
	t.left = deleteID(t.left, it.objectID)
	if !slices.Contains(t.destroyed, it.objectID) {
		t.destroyed = append(t.destroyed, it.objectID)
	}
}

// leave records an item that left the character: dropped, given away.
func (t *Tracker) leave(it *Item) {
	t.forget(it)
	if !slices.Contains(t.destroyed, it.objectID) && !slices.Contains(t.left, it.objectID) {
		t.left = append(t.left, it.objectID)
	}
}

// forget drops a pending upsert of it.
func (t *Tracker) forget(it *Item) {
	if i := slices.Index(t.upserts, it); i >= 0 {
		t.upserts = slices.Delete(t.upserts, i, i+1)
	}
	it.dirty = false
}

// Pending reports whether there are changes to save.
func (t *Tracker) Pending() bool {
	return len(t.upserts) > 0 || len(t.destroyed) > 0 || len(t.left) > 0
}

// Changes returns the changes since the last Take, as storage's Save takes
// them, without resetting them. Upserts hold the items as they are now, in
// the order of their first change.
func (t *Tracker) Changes() storage.ItemChanges {
	var ch storage.ItemChanges
	if len(t.upserts) > 0 {
		ch.Upserts = make([]storage.Item, len(t.upserts))
		for i, it := range t.upserts {
			ch.Upserts[i] = it.Storage()
		}
	}
	ch.Destroyed = slices.Clone(t.destroyed)
	ch.Left = slices.Clone(t.left)
	return ch
}

// Take returns the changes since the last Take and starts a new diff: the
// saver calls it when it snapshots the character. A failed save is the
// saver's to retry with the snapshot it took.
func (t *Tracker) Take() storage.ItemChanges {
	ch := t.Changes()
	for _, it := range t.upserts {
		it.dirty = false
	}
	t.upserts, t.destroyed, t.left = nil, nil, nil
	return ch
}

func deleteID(ids []int32, id int32) []int32 {
	if i := slices.Index(ids, id); i >= 0 {
		return slices.Delete(ids, i, i+1)
	}
	return ids
}
