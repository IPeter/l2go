package memstore

import (
	"cmp"
	"context"
	"errors"
	"maps"
	"slices"

	"github.com/IPeter/l2go/internal/storage"
)

// WriteBatch implements storage.BatchWriter.
func (s *Characters) WriteBatch(ctx context.Context, b storage.Batch) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writeBatch(&b)
}

// writeBatch checks b, then applies it in the order storage.Batch gives. The
// caller holds s.mu.
func (s *Characters) writeBatch(b *storage.Batch) error {
	s.init()
	for i := range b.Saves {
		if _, ok := s.byID[b.Saves[i].Character.ObjectID]; !ok {
			return storage.ErrNotFound
		}
	}
	if err := s.checkClanNames(b); err != nil {
		return err
	}
	if err := s.takeFault(); err != nil {
		return err
	}
	for i := range b.Saves {
		s.applySave(&b.Saves[i].Character, b.Saves[i].Changes)
	}
	for _, o := range b.Items {
		s.applyItems(o.Owner, o.Upserts, o.Destroyed, o.Left)
	}
	for _, id := range b.DeletedClans {
		s.deleteClan(id)
	}
	for _, c := range b.Clans {
		s.putClan(c)
	}
	for _, a := range b.AllyCrests {
		if a.AllyID == 0 {
			continue
		}
		for id, c := range s.clans {
			if c.AllyID == a.AllyID {
				c.AllyCrestID = a.CrestID
				s.clans[id] = c
			}
		}
	}
	for _, id := range b.DeletedCrests {
		delete(s.crests, id)
	}
	for _, c := range b.Crests {
		c.Data = slices.Clone(c.Data)
		s.crests[c.ID] = c
	}
	for _, r := range b.Relations.Removed {
		delete(s.relations, r)
	}
	for _, r := range b.Relations.Added {
		s.relations[r] = struct{}{}
	}
	for _, p := range b.Patches {
		if c, ok := s.byID[p.ObjectID]; ok {
			p.Apply(&c)
			s.byID[p.ObjectID] = c
		}
	}
	return nil
}

// checkClanNames returns storage.ErrExists if a clan of b would take the
// name of another clan, ignoring case, at the point the batch writes it.
// The caller holds s.mu.
func (s *Characters) checkClanNames(b *storage.Batch) error {
	if len(b.Clans) == 0 {
		return nil
	}
	names := maps.Clone(s.clanByName)
	keyOf := make(map[int32]string, len(s.clans))
	for id, c := range s.clans {
		keyOf[id] = storage.NameKey(c.Name)
	}
	for _, id := range b.DeletedClans {
		if k, ok := keyOf[id]; ok {
			delete(names, k)
			delete(keyOf, id)
		}
	}
	for _, c := range b.Clans {
		if k, ok := keyOf[c.ID]; ok {
			delete(names, k)
		}
		k := storage.NameKey(c.Name)
		if _, taken := names[k]; taken {
			return storage.ErrExists
		}
		names[k] = c.ID
		keyOf[c.ID] = k
	}
	return nil
}

// putClan stores a copy of c, replacing the clan with its ID. The caller
// holds s.mu.
func (s *Characters) putClan(c storage.Clan) {
	if old, ok := s.clans[c.ID]; ok {
		delete(s.clanByName, storage.NameKey(old.Name))
	}
	s.clans[c.ID] = cloneClan(c)
	s.clanByName[storage.NameKey(c.Name)] = c.ID
}

// deleteClan removes the clan, its items, and its ID from the wars of the
// other clans. The caller holds s.mu.
func (s *Characters) deleteClan(id int32) {
	if c, ok := s.clans[id]; ok {
		delete(s.clanByName, storage.NameKey(c.Name))
		delete(s.clans, id)
	}
	for itemID := range s.byOwner[id] {
		s.deleteItem(itemID)
	}
	for other, c := range s.clans {
		if slices.Contains(c.WarsDeclared, id) {
			c.WarsDeclared = cloneList(slices.DeleteFunc(slices.Clone(c.WarsDeclared), func(w int32) bool { return w == id }))
			s.clans[other] = c
		}
	}
}

// cloneClan returns a copy of c that shares no slice with it, with nil for
// empty lists, as from MongoDB.
func cloneClan(c storage.Clan) storage.Clan {
	c.Ranks = cloneList(c.Ranks)
	c.Units = cloneList(c.Units)
	c.Skills = cloneList(c.Skills)
	c.WarsDeclared = cloneList(c.WarsDeclared)
	return c
}

// Clans returns the clan store that reads the clans of s.
func (s *Characters) Clans() *Clans { return &Clans{db: s} }

// Clans is the in-memory storage.ClanStore of a Characters store. Clans are
// written with Characters.WriteBatch.
type Clans struct{ db *Characters }

var _ storage.ClanStore = (*Clans)(nil)

// LoadClans implements storage.ClanStore.
func (c *Clans) LoadClans(ctx context.Context) ([]storage.Clan, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.db.mu.Lock()
	defer c.db.mu.Unlock()
	if len(c.db.clans) == 0 {
		return nil, nil
	}
	list := make([]storage.Clan, 0, len(c.db.clans))
	for _, id := range slices.Sorted(maps.Keys(c.db.clans)) {
		list = append(list, cloneClan(c.db.clans[id]))
	}
	return list, nil
}

// GetClan implements storage.ClanStore.
func (c *Clans) GetClan(ctx context.Context, id int32) (storage.Clan, error) {
	if err := ctx.Err(); err != nil {
		return storage.Clan{}, err
	}
	c.db.mu.Lock()
	defer c.db.mu.Unlock()
	clan, ok := c.db.clans[id]
	if !ok {
		return storage.Clan{}, storage.ErrNotFound
	}
	return cloneClan(clan), nil
}

// ClanMembers implements storage.ClanStore.
func (c *Clans) ClanMembers(ctx context.Context, clanID int32) ([]storage.ClanMember, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.db.mu.Lock()
	defer c.db.mu.Unlock()
	var list []storage.ClanMember
	for _, ch := range c.db.byID {
		if ch.ClanID == 0 || (clanID != 0 && ch.ClanID != clanID) {
			continue
		}
		list = append(list, storage.ClanMember{
			ObjectID:         ch.ObjectID,
			ClanID:           ch.ClanID,
			Name:             ch.Name,
			Level:            ch.Level,
			ClassID:          ch.ClassID,
			Race:             ch.Race,
			Sex:              ch.Sex,
			Title:            ch.Title,
			PowerGrade:       ch.PowerGrade,
			PledgeType:       ch.PledgeType,
			Apprentice:       ch.Apprentice,
			Sponsor:          ch.Sponsor,
			LvlJoinedAcademy: ch.LvlJoinedAcademy,
			WantsPeace:       ch.WantsPeace,
		})
	}
	slices.SortFunc(list, func(a, b storage.ClanMember) int {
		return cmp.Or(cmp.Compare(a.ClanID, b.ClanID), cmp.Compare(a.ObjectID, b.ObjectID))
	})
	return list, nil
}

// ClanItems implements storage.ClanStore.
func (c *Clans) ClanItems(ctx context.Context, clanID int32) ([]storage.Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.db.mu.Lock()
	defer c.db.mu.Unlock()
	return c.db.ownedItems(clanID), nil
}

// Crests returns the crest store of s.
func (s *Characters) Crests() *Crests { return &Crests{db: s} }

// Crests is the in-memory storage.CrestStore of a Characters store. Crest
// IDs are unique within it only; they start over in a new process. Crests
// are written with Characters.WriteBatch.
type Crests struct{ db *Characters }

var _ storage.CrestStore = (*Crests)(nil)

var errCrestIDsExhausted = errors.New("memstore: crest IDs exhausted")

// NextCrestID implements storage.CrestStore.
func (c *Crests) NextCrestID(ctx context.Context) (int32, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	c.db.mu.Lock()
	defer c.db.mu.Unlock()
	if c.db.crestsUsed == 1<<31-1 {
		return 0, errCrestIDsExhausted
	}
	c.db.crestsUsed++
	return c.db.crestsUsed, nil
}

// LoadCrests implements storage.CrestStore.
func (c *Crests) LoadCrests(ctx context.Context) ([]storage.Crest, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.db.mu.Lock()
	defer c.db.mu.Unlock()
	if len(c.db.crests) == 0 {
		return nil, nil
	}
	list := make([]storage.Crest, 0, len(c.db.crests))
	for _, id := range slices.Sorted(maps.Keys(c.db.crests)) {
		cr := c.db.crests[id]
		cr.Data = slices.Clone(cr.Data)
		list = append(list, cr)
	}
	return list, nil
}

// Relations returns the relation store of s.
func (s *Characters) Relations() *Relations { return &Relations{db: s} }

// Relations is the in-memory storage.RelationStore of a Characters store.
// Relations are written with Characters.WriteBatch.
type Relations struct{ db *Characters }

var _ storage.RelationStore = (*Relations)(nil)

// Relations implements storage.RelationStore.
func (r *Relations) Relations(ctx context.Context, owner int32) ([]storage.Relation, error) {
	return r.find(ctx, func(rel storage.Relation) bool { return rel.Owner == owner }, func(a, b storage.Relation) int {
		return cmp.Or(cmp.Compare(a.Type, b.Type), cmp.Compare(a.Target, b.Target))
	})
}

// RelationsTo implements storage.RelationStore.
func (r *Relations) RelationsTo(ctx context.Context, target int32) ([]storage.Relation, error) {
	return r.find(ctx, func(rel storage.Relation) bool { return rel.Target == target }, func(a, b storage.Relation) int {
		return cmp.Or(cmp.Compare(a.Type, b.Type), cmp.Compare(a.Owner, b.Owner))
	})
}

// find returns the relations that match, sorted.
func (r *Relations) find(ctx context.Context, match func(storage.Relation) bool, order func(a, b storage.Relation) int) ([]storage.Relation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.db.mu.Lock()
	defer r.db.mu.Unlock()
	var list []storage.Relation
	for rel := range r.db.relations {
		if match(rel) {
			list = append(list, rel)
		}
	}
	slices.SortFunc(list, order)
	return list, nil
}
