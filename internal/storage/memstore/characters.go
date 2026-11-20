package memstore

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/IPeter/l2go/internal/storage"
)

// Characters is an in-memory storage.CharacterStore and
// storage.CharacterSaver. Like the MongoDB store it keeps the items and the
// quests apart from the characters, each with its owner, so items move
// between characters the same way. Every write checks first and then changes
// everything at once, which makes it as atomic as a transaction. The zero
// value is ready to use and it is safe for concurrent use.
type Characters struct {
	mu      sync.Mutex
	byID    map[int32]storage.Character // without items
	byName  map[string]int32            // storage.NameKey(name) → object ID
	items   map[int32]ownedItem         // item object ID → item
	byOwner map[int32]map[int32]struct{}
	quests  map[int32]map[string]map[string]string // character → quest name → vars

	// failNext, if set, fails the next Create, Update, Save or Delete that
	// gets past its checks, before it changes anything. Tests in this
	// package use it to run the shared failed-write tests.
	failNext error
}

// ownedItem is a stored item, normalized, and the character that owns it.
type ownedItem struct {
	owner int32
	item  storage.Item
}

var (
	_ storage.CharacterStore = (*Characters)(nil)
	_ storage.CharacterSaver = (*Characters)(nil)
)

// NewCharacters returns an empty store.
func NewCharacters() *Characters {
	return &Characters{}
}

// init makes the maps of the zero value. The caller holds s.mu.
func (s *Characters) init() {
	if s.byID == nil {
		s.byID = make(map[int32]storage.Character)
		s.byName = make(map[string]int32)
		s.items = make(map[int32]ownedItem)
		s.byOwner = make(map[int32]map[int32]struct{})
		s.quests = make(map[int32]map[string]map[string]string)
	}
}

// takeFault returns and clears the injected failure. The caller holds s.mu.
func (s *Characters) takeFault() error {
	err := s.failNext
	s.failNext = nil
	return err
}

// Create implements storage.CharacterStore.
func (s *Characters) Create(ctx context.Context, c *storage.Character) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.init()
	key := storage.NameKey(c.Name)
	if _, ok := s.byName[key]; ok {
		return storage.ErrExists
	}
	if _, ok := s.byID[c.ObjectID]; ok {
		return storage.ErrExists
	}
	seen := make(map[int32]bool, len(c.Items))
	for _, it := range c.Items {
		if _, ok := s.items[it.ObjectID]; ok || seen[it.ObjectID] {
			return fmt.Errorf("%w: item object ID %d", storage.ErrExists, it.ObjectID)
		}
		seen[it.ObjectID] = true
	}
	if err := s.takeFault(); err != nil {
		return err
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = normalizeTime(time.Now())
	}
	stored := *c
	stored.Items, stored.Quests = nil, nil
	stored.Vars = cloneVars(c.Vars)
	stored.Skills, stored.SavedSkills = cloneList(c.Skills), cloneList(c.SavedSkills)
	stored.Shortcuts, stored.ItemReuse = cloneList(c.Shortcuts), cloneList(c.ItemReuse)
	stored.CreatedAt = normalizeTime(stored.CreatedAt)
	stored.DeleteTime = normalizeTime(stored.DeleteTime)
	stored.LastAccess = normalizeTime(stored.LastAccess)
	s.byID[c.ObjectID] = stored
	s.byName[key] = c.ObjectID
	for _, it := range c.Items {
		s.putItem(c.ObjectID, it)
	}
	delete(s.quests, c.ObjectID)
	for _, q := range c.Quests {
		s.putQuest(c.ObjectID, q)
	}
	return nil
}

// ListByAccount implements storage.CharacterStore.
func (s *Characters) ListByAccount(ctx context.Context, account string) ([]storage.Character, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var list []storage.Character
	for _, c := range s.byID {
		if c.Account == account {
			list = append(list, s.load(c))
		}
	}
	slices.SortFunc(list, func(a, b storage.Character) int {
		return cmp.Or(a.CreatedAt.Compare(b.CreatedAt), cmp.Compare(a.ObjectID, b.ObjectID))
	})
	return list, nil
}

// CountByAccount implements storage.CharacterStore.
func (s *Characters) CountByAccount(ctx context.Context, account string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, c := range s.byID {
		if c.Account == account {
			n++
		}
	}
	return n, nil
}

// NameExists implements storage.CharacterStore.
func (s *Characters) NameExists(ctx context.Context, name string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.byName[storage.NameKey(name)]
	return ok, nil
}

// Get implements storage.CharacterStore.
func (s *Characters) Get(ctx context.Context, objectID int32) (storage.Character, error) {
	if err := ctx.Err(); err != nil {
		return storage.Character{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.byID[objectID]
	if !ok {
		return storage.Character{}, storage.ErrNotFound
	}
	return s.load(c), nil
}

// GetByName implements storage.CharacterStore.
func (s *Characters) GetByName(ctx context.Context, name string) (storage.Character, error) {
	if err := ctx.Err(); err != nil {
		return storage.Character{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.byName[storage.NameKey(name)]
	if !ok {
		return storage.Character{}, storage.ErrNotFound
	}
	return s.load(s.byID[id]), nil
}

// Rename implements storage.CharacterStore.
func (s *Characters) Rename(ctx context.Context, objectID int32, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.byID[objectID]
	if !ok {
		return storage.ErrNotFound
	}
	key := storage.NameKey(name)
	if id, taken := s.byName[key]; taken && id != objectID {
		return storage.ErrExists
	}
	delete(s.byName, storage.NameKey(c.Name))
	c.Name = name
	s.byID[objectID] = c
	s.byName[key] = objectID
	return nil
}

// load returns a copy of the stored character c with its items, variables
// and quests. The caller holds s.mu.
func (s *Characters) load(c storage.Character) storage.Character {
	c.Items = s.ownedItems(c.ObjectID)
	c.Vars = cloneVars(c.Vars)
	c.Skills, c.SavedSkills = cloneList(c.Skills), cloneList(c.SavedSkills)
	c.Shortcuts, c.ItemReuse = cloneList(c.Shortcuts), cloneList(c.ItemReuse)
	c.Quests = s.ownedQuests(c.ObjectID)
	return c
}

// Update implements storage.CharacterStore.
func (s *Characters) Update(ctx context.Context, c *storage.Character) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.updateFields(c); err != nil {
		return err
	}
	keep := make(map[int32]bool, len(c.Items))
	for _, it := range c.Items {
		keep[it.ObjectID] = true
	}
	for id := range s.byOwner[c.ObjectID] {
		if !keep[id] {
			s.deleteItem(id)
		}
	}
	for _, it := range c.Items {
		s.putItem(c.ObjectID, it)
	}
	delete(s.quests, c.ObjectID)
	for _, q := range c.Quests {
		s.putQuest(c.ObjectID, q)
	}
	return nil
}

// Save implements storage.CharacterSaver.
func (s *Characters) Save(ctx context.Context, c *storage.Character, changes storage.ItemChanges) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.updateFields(c); err != nil {
		return err
	}
	for _, id := range changes.Destroyed {
		s.deleteItem(id)
	}
	for _, id := range changes.Left {
		if e, ok := s.items[id]; ok && e.owner == c.ObjectID {
			s.deleteItem(id)
		}
	}
	for _, it := range changes.Upserts {
		s.putItem(c.ObjectID, it)
	}
	for _, name := range changes.Quests.Deleted {
		delete(s.quests[c.ObjectID], name)
	}
	for _, q := range changes.Quests.Upserts {
		s.putQuest(c.ObjectID, q)
	}
	if len(s.quests[c.ObjectID]) == 0 {
		delete(s.quests, c.ObjectID)
	}
	return nil
}

// updateFields stores the fields Update saves, or returns ErrNotFound or
// the injected failure. The caller holds s.mu.
func (s *Characters) updateFields(c *storage.Character) error {
	old, ok := s.byID[c.ObjectID]
	if !ok {
		return storage.ErrNotFound
	}
	if err := s.takeFault(); err != nil {
		return err
	}
	stored := *c
	stored.Items, stored.Quests = nil, nil
	stored.Vars = cloneVars(c.Vars)
	stored.Skills, stored.SavedSkills = cloneList(c.Skills), cloneList(c.SavedSkills)
	stored.Shortcuts, stored.ItemReuse = cloneList(c.Shortcuts), cloneList(c.ItemReuse)
	stored.Account, stored.Name, stored.Race, stored.CreatedAt = old.Account, old.Name, old.Race, old.CreatedAt
	stored.Newbie = old.Newbie
	stored.DeleteTime = normalizeTime(stored.DeleteTime)
	stored.LastAccess = normalizeTime(stored.LastAccess)
	s.byID[c.ObjectID] = stored
	return nil
}

// Delete implements storage.CharacterStore.
func (s *Characters) Delete(ctx context.Context, objectID int32) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.byID[objectID]
	if !ok {
		return storage.ErrNotFound
	}
	if err := s.takeFault(); err != nil {
		return err
	}
	for id := range s.byOwner[objectID] {
		s.deleteItem(id)
	}
	delete(s.quests, objectID)
	delete(s.byID, objectID)
	delete(s.byName, storage.NameKey(c.Name))
	return nil
}

// SetDeleteTime implements storage.CharacterStore.
func (s *Characters) SetDeleteTime(ctx context.Context, objectID int32, t time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.byID[objectID]
	if !ok {
		return storage.ErrNotFound
	}
	c.DeleteTime = normalizeTime(t)
	s.byID[objectID] = c
	return nil
}

// putItem stores it, normalized, with owner as its owner, taking it from
// its previous owner. The caller holds s.mu.
func (s *Characters) putItem(owner int32, it storage.Item) {
	s.init()
	if e, ok := s.items[it.ObjectID]; ok && e.owner != owner {
		delete(s.byOwner[e.owner], it.ObjectID)
	}
	s.items[it.ObjectID] = ownedItem{owner: owner, item: it.Normalized()}
	ids := s.byOwner[owner]
	if ids == nil {
		ids = make(map[int32]struct{})
		s.byOwner[owner] = ids
	}
	ids[it.ObjectID] = struct{}{}
}

// deleteItem removes the item, if it exists. The caller holds s.mu.
func (s *Characters) deleteItem(id int32) {
	e, ok := s.items[id]
	if !ok {
		return
	}
	delete(s.items, id)
	delete(s.byOwner[e.owner], id)
	if len(s.byOwner[e.owner]) == 0 {
		delete(s.byOwner, e.owner)
	}
}

// ownedItems returns copies of the owner's items in object ID order, or nil
// if it has none, as from MongoDB. The caller holds s.mu.
func (s *Characters) ownedItems(owner int32) []storage.Item {
	ids := s.byOwner[owner]
	if len(ids) == 0 {
		return nil
	}
	items := make([]storage.Item, 0, len(ids))
	for id := range ids {
		items = append(items, s.items[id].item)
	}
	slices.SortFunc(items, func(a, b storage.Item) int { return cmp.Compare(a.ObjectID, b.ObjectID) })
	return items
}

// putQuest stores a copy of q as one of owner's quests, replacing the one
// with the same name. The caller holds s.mu.
func (s *Characters) putQuest(owner int32, q storage.Quest) {
	s.init()
	byName := s.quests[owner]
	if byName == nil {
		byName = make(map[string]map[string]string)
		s.quests[owner] = byName
	}
	byName[q.Name] = cloneVars(q.Vars)
}

// ownedQuests returns copies of the owner's quests in name order, or nil if
// it has none, as from MongoDB. The caller holds s.mu.
func (s *Characters) ownedQuests(owner int32) []storage.Quest {
	byName := s.quests[owner]
	if len(byName) == 0 {
		return nil
	}
	quests := make([]storage.Quest, 0, len(byName))
	for _, name := range slices.Sorted(maps.Keys(byName)) {
		quests = append(quests, storage.Quest{Name: name, Vars: cloneVars(byName[name])})
	}
	return quests
}

// cloneVars returns a copy of vars, or nil if it is empty, as from MongoDB.
func cloneVars(vars map[string]string) map[string]string {
	if len(vars) == 0 {
		return nil
	}
	return maps.Clone(vars)
}

// cloneList returns a copy of list, or nil if it is empty, as from MongoDB.
func cloneList[T any](list []T) []T {
	if len(list) == 0 {
		return nil
	}
	return slices.Clone(list)
}

// normalizeTime returns t as MongoDB would store it: millisecond precision,
// in UTC, without a monotonic clock reading.
func normalizeTime(t time.Time) time.Time {
	return t.UTC().Truncate(time.Millisecond)
}

// IDs is an in-memory storage.IDAllocator. IDs are unique within one IDs
// value only; they start over in a new process. The zero value is ready to
// use and it is safe for concurrent use.
type IDs struct {
	mu   sync.Mutex
	used int64 // how many IDs were handed out
}

var _ storage.IDAllocator = (*IDs)(nil)

// NewIDs returns an allocator that starts at storage.FirstObjectID.
func NewIDs() *IDs {
	return &IDs{}
}

var errIDsExhausted = errors.New("memstore: object IDs exhausted")

// NextID implements storage.IDAllocator.
func (a *IDs) NextID(ctx context.Context) (int32, error) {
	ids, err := a.NextIDs(ctx, 1)
	if err != nil {
		return 0, err
	}
	return ids[0], nil
}

// NextIDs implements storage.IDAllocator. IDs end below
// storage.FirstRuntimeObjectID.
func (a *IDs) NextIDs(ctx context.Context, n int) ([]int32, error) {
	if n < 1 {
		return nil, fmt.Errorf("memstore: NextIDs(%d): n must be at least 1", n)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	first := int64(storage.FirstObjectID) + a.used
	if first+int64(n) > int64(storage.FirstRuntimeObjectID) {
		return nil, errIDsExhausted
	}
	a.used += int64(n)
	ids := make([]int32, n)
	for i := range ids {
		ids[i] = int32(first + int64(i)) //nolint:gosec // range checked against storage.FirstRuntimeObjectID above
	}
	return ids, nil
}
