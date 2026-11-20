package inventory

import (
	"fmt"
	"math"
	"slices"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/storage"
)

// Templates looks up item templates; *data.Data implements it.
type Templates interface {
	ItemTemplate(id int) (*data.Item, bool)
}

// IDSource hands out object IDs for new items without blocking;
// *storage.IDPool implements it (decision I3).
type IDSource interface {
	TryNextN(n int) ([]int32, bool)
}

// Config holds the Mobius settings the containers follow.
type Config struct {
	// MultipleItemDrop creates one item per count when items that do not
	// stack are added; otherwise only one (General.ini, true).
	MultipleItemDrop bool
	// MaxAdena caps the adena stack (Player.ini, 2,000,000,000). Other
	// stacks are capped at the int range, as in Mobius.
	MaxAdena int64
	// AutoLootSlotLimit makes ValidateCapacity(0) check the slots too
	// (Player.ini, false).
	AutoLootSlotLimit bool
}

// DefaultConfig returns the Mobius defaults.
func DefaultConfig() Config {
	return Config{MultipleItemDrop: true, MaxAdena: DefaultMaxAdena}
}

// Options are what a container needs besides its owner.
type Options struct {
	Templates Templates // required
	// IDs gives new items their object IDs. Without it, adding by item ID,
	// splitting a stack and transferring a stack fail with ErrNoObjectID.
	IDs IDSource
	// Tracker collects the save diff; nil makes a new one. A character's
	// inventory and warehouse should share one.
	Tracker *Tracker
	Config  Config
}

// Update is one entry of an InventoryUpdate: an item and how it changed.
type Update struct {
	Item   *Item
	Change Change
}

// Container is a set of items with one base location (Mobius
// ItemContainer): the inventory part of an Inventory, or a warehouse.
// Items keep the order they were added in.
type Container struct {
	base      storage.ItemLocation
	items     []*Item
	templates Templates
	ids       IDSource
	tracker   *Tracker
	cfg       Config
	updates   []Update
	loading   bool

	inv      *Inventory // the inventory this container belongs to, or nil
	capacity func() int // the warehouse slot limit, or nil
}

func newContainer(base storage.ItemLocation, opts Options) Container {
	if opts.Tracker == nil {
		opts.Tracker = NewTracker()
	}
	if opts.Config.MaxAdena <= 0 {
		opts.Config.MaxAdena = DefaultMaxAdena
	}
	return Container{base: base, templates: opts.Templates, ids: opts.IDs, tracker: opts.Tracker, cfg: opts.Config}
}

// NewWarehouse returns an empty private warehouse (Mobius PlayerWarehouse,
// location WAREHOUSE). limit returns its slot count: 100, or 120 for
// dwarves, plus the whLimit stat (Player.getWareHouseLimit).
func NewWarehouse(limit func() int, opts Options) *Container {
	c := newContainer(storage.LocWarehouse, opts)
	c.capacity = limit
	return &c
}

// Location returns the base location of the container's items.
func (c *Container) Location() storage.ItemLocation { return c.base }

// Tracker returns the save diff tracker.
func (c *Container) Tracker() *Tracker { return c.tracker }

// Items returns the items in the order they were added. The slice must not
// be changed.
func (c *Container) Items() []*Item { return c.items }

// Len returns the number of items (Mobius getSize).
func (c *Container) Len() int { return len(c.items) }

// ByObjectID returns the item with the object ID, or nil.
func (c *Container) ByObjectID(objectID int32) *Item {
	for _, it := range c.items {
		if it.objectID == objectID {
			return it
		}
	}
	return nil
}

// ByItemID returns the first item of the template, or nil (Mobius
// getItemByItemId).
func (c *Container) ByItemID(itemID int32) *Item {
	for _, it := range c.items {
		if it.ItemID() == itemID {
			return it
		}
	}
	return nil
}

// AllByItemID returns the items of the template.
func (c *Container) AllByItemID(itemID int32) []*Item {
	var out []*Item
	for _, it := range c.items {
		if it.ItemID() == itemID {
			out = append(out, it)
		}
	}
	return out
}

// Count returns how many of the template the container holds: the first
// stack of a stackable item, otherwise the number of items. enchant -1
// matches any enchant level; includeEquipped counts worn items too (Mobius
// getInventoryItemCount).
func (c *Container) Count(itemID, enchant int32, includeEquipped bool) int64 {
	var n int64
	for _, it := range c.items {
		if it.ItemID() != itemID || (enchant >= 0 && it.enchant != enchant) || (!includeEquipped && it.Equipped()) {
			continue
		}
		if it.Stackable() {
			return it.count
		}
		n++
	}
	return n
}

// Adena returns the adena count (Mobius getAdena).
func (c *Container) Adena() int64 {
	if it := c.ByItemID(AdenaID); it != nil {
		return it.count
	}
	return 0
}

// AdenaItem returns the adena stack, or nil (Mobius getAdenaInstance).
func (c *Container) AdenaItem() *Item { return c.ByItemID(AdenaID) }

// TakeUpdates returns the changes since the last call, one per item in the
// order of their first change, for an InventoryUpdate. A later change of an
// item replaces an earlier one, except that an item added and then modified
// stays added, since the client does not know it yet.
func (c *Container) TakeUpdates() []Update {
	u := c.updates
	c.updates = nil
	return u
}

// maxCount is the largest count of a stack of the template.
func (c *Container) maxCount(t *data.Item) int64 {
	if int32(t.ID) == AdenaID { //nolint:gosec // item IDs are below 2^20
		return c.cfg.MaxAdena
	}
	return math.MaxInt32
}

// touch records a change of it, which is in c: the save diff and the update
// list.
func (c *Container) touch(it *Item, ch Change) {
	if c.loading {
		return
	}
	c.tracker.upsert(it)
	c.record(it, ch)
}

// record adds a change to the update list.
func (c *Container) record(it *Item, ch Change) {
	if c.loading {
		return
	}
	for i := range c.updates {
		if c.updates[i].Item.objectID == it.objectID {
			if !(c.updates[i].Change == Added && ch == Modified) {
				c.updates[i] = Update{Item: it, Change: ch}
			}
			return
		}
	}
	c.updates = append(c.updates, Update{Item: it, Change: ch})
}

// insert puts it into the container as it is (Mobius addItem(Item)); an
// inventory equips it when it is marked worn.
func (c *Container) insert(it *Item, w Wearer) {
	c.items = append(c.items, it)
	it.c = c
	if c.inv != nil && it.Equipped() {
		c.inv.equipItem(it, w)
	}
}

// remove takes it out of the container (Mobius removeItem); an inventory
// takes it off first.
func (c *Container) remove(it *Item) bool {
	i := slices.Index(c.items, it)
	if i < 0 {
		return false
	}
	if c.inv != nil {
		for slot, worn := range c.inv.paperdoll {
			if worn == it {
				c.inv.setPaperdoll(storage.PaperdollSlot(slot), nil) //nolint:gosec // a paperdoll index
			}
		}
	}
	c.items = slices.Delete(c.items, i, i+1)
	it.c = nil
	return true
}

// destroyed marks it, which left c, as destroyed (Mobius
// ItemManager.destroyItem).
func (c *Container) destroyed(it *Item) {
	it.loc, it.slot, it.lastChange = LocVoid, storage.PaperdollNone, Removed
	c.tracker.destroy(it)
	c.record(it, Removed)
}

// Add puts an item that is in no container into this one (Mobius
// addItem(process, Item)), for example one picked up from the ground. If
// the container holds a stack of the template, the count goes onto it and
// the item is destroyed; Add then returns the stack. Otherwise the item is
// added at the base location and returned.
func (c *Container) Add(it *Item) (*Item, error) {
	if it.c != nil {
		return nil, ErrInContainer
	}
	if old := c.ByItemID(it.ItemID()); old != nil && old.Stackable() {
		old.changeCount(it.count, c.maxCount(old.tmpl))
		old.lastChange = Modified
		c.touch(old, Modified)
		c.destroyed(it)
		return old, nil
	}
	it.loc, it.slot, it.lastChange = c.base, storage.PaperdollNone, Added
	c.insert(it, Wearer{})
	c.touch(it, Added)
	return it, nil
}

// Create adds count items of the template (Mobius addItem(process, itemId,
// count)): onto the stack the container holds, else as a new stack, else,
// for items that do not stack, as count new items (one if MultipleItemDrop
// is off) at the template's default enchant level. All new object IDs are
// taken before anything changes. It returns the stack or the last new item.
func (c *Container) Create(itemID int32, count int64) (*Item, error) {
	t, ok := c.templates.ItemTemplate(int(itemID))
	if !ok {
		return nil, fmt.Errorf("%w: %d", ErrUnknownItem, itemID)
	}
	if count <= 0 {
		return nil, ErrInvalidCount
	}
	if old := c.ByItemID(itemID); old != nil && old.Stackable() {
		old.changeCount(count, c.maxCount(t))
		old.lastChange = Modified
		c.touch(old, Modified)
		return old, nil
	}
	n := int64(1)
	if !t.Stackable && c.cfg.MultipleItemDrop {
		n = count
	}
	ids, err := c.newIDs(n)
	if err != nil {
		return nil, err
	}
	var it *Item
	for _, id := range ids {
		it = c.createWithID(t, count, id)
	}
	return it, nil
}

// createWithID adds a new item of the template with the object ID.
func (c *Container) createWithID(t *data.Item, count int64, id int32) *Item {
	it := NewItem(id, t, min(count, c.maxCount(t)))
	it.enchant = int32(t.DefaultEnchant) //nolint:gosec // small template values
	it.loc, it.lastChange = c.base, Added
	c.insert(it, Wearer{})
	c.touch(it, Added)
	return it
}

// newIDs takes n object IDs from the ID source.
func (c *Container) newIDs(n int64) ([]int32, error) {
	if c.ids == nil || n > math.MaxInt32 {
		return nil, ErrNoObjectID
	}
	ids, ok := c.ids.TryNextN(int(n))
	if !ok || int64(len(ids)) != n {
		return nil, ErrNoObjectID
	}
	return ids, nil
}

// Transfer moves count of the item with the object ID into target (Mobius
// transferItem): an item that does not stack moves whole, keeping its
// object ID; a stack shrinks (or is destroyed when all of it goes) and
// its count goes onto the target's stack, or into a new item there with a
// new object ID. count is capped at what the item has. It returns the item
// in target.
func (c *Container) Transfer(objectID int32, count int64, target *Container) (*Item, error) {
	if target == nil || target == c {
		return nil, ErrNotFound
	}
	src := c.ByObjectID(objectID)
	if src == nil {
		return nil, ErrNotFound
	}
	if count <= 0 {
		return nil, ErrInvalidCount
	}
	count = min(count, src.count)
	var dst *Item
	if src.Stackable() {
		dst = target.ByItemID(src.ItemID())
	}
	if src.count == count && dst == nil && !src.Stackable() {
		c.remove(src)
		c.tracker.leave(src)
		c.record(src, Removed)
		return target.Add(src)
	}
	var newID int32
	if dst == nil {
		ids, err := target.newIDs(1)
		if err != nil {
			return nil, err
		}
		newID = ids[0]
	}
	if src.count > count {
		src.changeCount(-count, math.MaxInt64)
		src.lastChange = Modified
		c.touch(src, Modified)
	} else {
		c.remove(src)
		c.destroyed(src)
	}
	if dst != nil {
		dst.changeCount(count, target.maxCount(dst.tmpl))
		dst.lastChange = Modified
		target.touch(dst, Modified)
		return dst, nil
	}
	return target.createWithID(src.tmpl, count, newID), nil
}

// Destroy destroys count of the item (Mobius destroyItem): the stack
// shrinks, or the item is taken off, removed and destroyed when count is
// all of it. More than the item has fails with ErrInvalidCount. It returns
// the item.
func (c *Container) Destroy(it *Item, count int64) (*Item, error) {
	if it == nil || it.c != c {
		return nil, ErrNotFound
	}
	if count <= 0 || count > it.count {
		return nil, ErrInvalidCount
	}
	if it.count > count {
		it.changeCount(-count, math.MaxInt64)
		it.lastChange = Modified
		c.touch(it, Modified)
		return it, nil
	}
	c.remove(it)
	c.destroyed(it)
	return it, nil
}

// DestroyByObjectID destroys count of the item with the object ID.
func (c *Container) DestroyByObjectID(objectID int32, count int64) (*Item, error) {
	return c.Destroy(c.ByObjectID(objectID), count)
}

// DestroyByItemID destroys count items of the template. A warehouse
// destroys from the first item of the template (Mobius
// ItemContainer.destroyItemByItemId). An inventory picks an item that is
// not worn when it has one, and destroys count unworn items one by one for
// items that do not stack, failing when it has fewer (Mobius
// PlayerInventory.destroyItemByItemId). It returns the last item it
// destroyed from.
func (c *Container) DestroyByItemID(itemID int32, count int64) (*Item, error) {
	if c.inv == nil {
		it := c.ByItemID(itemID)
		if it == nil {
			return nil, ErrNotFound
		}
		return c.Destroy(it, count)
	}
	items := c.AllByItemID(itemID)
	var pick *Item
	for _, it := range items {
		pick = it
		if !it.Equipped() {
			break
		}
	}
	if pick == nil {
		return nil, ErrNotFound
	}
	if !pick.Stackable() && count > 1 {
		if c.Count(itemID, -1, false) < count {
			return nil, ErrInvalidCount
		}
		var done int64
		for _, it := range items {
			if it.Equipped() {
				continue
			}
			if _, err := c.Destroy(it, 1); err == nil {
				if done++; done == count {
					return it, nil
				}
			}
		}
	}
	return c.Destroy(pick, count)
}

// ValidateCapacity reports whether slots more items fit: an inventory
// checks its owner's inventory limit (always passing for 0 slots unless
// AutoLootSlotLimit is set), a warehouse its limit (Mobius
// PlayerInventory and PlayerWarehouse validateCapacity).
func (c *Container) ValidateCapacity(slots int64) bool {
	switch {
	case c.inv != nil:
		if slots == 0 && !c.cfg.AutoLootSlotLimit {
			return true
		}
		return int64(len(c.items))+slots <= int64(c.inv.owner.InventoryLimit())
	case c.capacity != nil:
		return int64(len(c.items))+slots <= int64(c.capacity())
	}
	return true
}

// Want is an item and a count that is to come into a container.
type Want struct {
	ItemID int32
	Count  int64
}

// SlotsNeeded returns how many slots the items take in this container: an
// item that does not stack takes one per count, a stackable one takes one
// unless the container already holds it (Mobius SendWareHouseDepositList,
// SendWareHouseWithDrawList). An unknown template takes one per count.
func (c *Container) SlotsNeeded(items []Want) int64 {
	var slots int64
	for _, w := range items {
		t, ok := c.templates.ItemTemplate(int(w.ItemID))
		switch {
		case !ok || !t.Stackable:
			slots += max(w.Count, 0)
		case c.ByItemID(w.ItemID) == nil:
			slots++
		}
	}
	return slots
}

// Load fills the empty container with the stored items at its base
// location and returns those it skipped for want of a template, which stay
// stored as they are (Mobius ItemContainer.restore). Stacks of one template
// are merged into the first, the others destroyed. Only real changes, such
// as merged stacks, end up in the save diff, and no updates are recorded.
func (c *Container) Load(items []storage.Item) (skipped []storage.Item) {
	return c.load(items, Wearer{}, func(si storage.Item) bool { return si.Normalized().Loc == c.base })
}

// load is Load for a filter of the stored items. An inventory equips the
// worn items afterwards, in slot order.
func (c *Container) load(items []storage.Item, w Wearer, keep func(storage.Item) bool) (skipped []storage.Item) {
	orig := make(map[*Item]storage.Item)
	var worn []*Item
	c.loading = true
	for _, si := range items {
		if !keep(si) {
			continue
		}
		t, ok := c.templates.ItemTemplate(int(si.ItemID))
		if !ok {
			skipped = append(skipped, si)
			continue
		}
		it := fromStorage(si, t)
		orig[it] = it.Storage()
		if c.inv != nil && it.Equipped() && t.Hero && !w.GM && !w.Hero {
			it.loc, it.slot = c.base, storage.PaperdollNone // Mobius Inventory.restore
		}
		if old := c.ByItemID(it.ItemID()); old != nil && it.Stackable() && old.Stackable() {
			old.changeCount(it.count, c.maxCount(t))
			it.loc, it.slot, it.lastChange = LocVoid, storage.PaperdollNone, Removed
			c.tracker.destroy(it)
			continue
		}
		if it.Equipped() {
			worn = append(worn, it)
			it.loc, it.slot = c.base, storage.PaperdollNone
		}
		c.insert(it, w)
	}
	if c.inv != nil {
		slices.SortStableFunc(worn, func(a, b *Item) int { return int(orig[a].Slot) - int(orig[b].Slot) })
		for _, it := range worn {
			c.inv.equipItem(it, w)
		}
	}
	c.loading = false
	for _, it := range c.items {
		if o, ok := orig[it]; ok && it.Storage() != o {
			c.tracker.upsert(it)
		}
	}
	return skipped
}
