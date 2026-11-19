package testclient

import (
	"cmp"
	"math"
	"slices"
)

// GroundItem is an item lying on the ground, shown by SpawnItem or
// DropItem and gone with GetItem or DeleteObject.
type GroundItem struct {
	ObjectID int32
	// ItemID is the display ID of the item template.
	ItemID    int32
	Pos       Position
	Stackable bool
	Count     int32
	// DropperID is the object that dropped the item (DropItem), 0 for an
	// item shown by SpawnItem.
	DropperID int32
}

// GroundItems returns the items on the ground, sorted by object ID.
func (w *World) GroundItems() []GroundItem {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]GroundItem, 0, len(w.ground))
	for _, g := range w.ground {
		out = append(out, g)
	}
	slices.SortFunc(out, func(a, b GroundItem) int { return cmp.Compare(a.ObjectID, b.ObjectID) })
	return out
}

// GroundItem returns the ground item with the given object ID.
func (w *World) GroundItem(objectID int32) (GroundItem, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	g, ok := w.ground[objectID]
	return g, ok
}

// FindGroundItem returns the ground item of template itemID nearest to the
// character (the lower object ID on a tie).
func (w *World) FindGroundItem(itemID int32) (GroundItem, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	var (
		best     GroundItem
		found    bool
		bestDist = math.Inf(1)
	)
	for _, g := range w.ground {
		if g.ItemID != itemID {
			continue
		}
		d := g.Pos.DistanceTo(w.me.Pos)
		if !found || d < bestDist || (d == bestDist && g.ObjectID < best.ObjectID) {
			best, bestDist, found = g, d, true
		}
	}
	return best, found
}

// Item returns the inventory item with the given object ID.
func (w *World) Item(objectID int32) (Item, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.item(objectID)
}

// item is Item for a caller that holds w.mu.
func (w *World) item(objectID int32) (Item, bool) {
	i := slices.IndexFunc(w.items, func(it Item) bool { return it.ObjectID == objectID })
	if i < 0 {
		return Item{}, false
	}
	return w.items[i], true
}

// FindItem returns the first inventory item of template itemID.
func (w *World) FindItem(itemID int32) (Item, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	i := slices.IndexFunc(w.items, func(it Item) bool { return it.ItemID == itemID })
	if i < 0 {
		return Item{}, false
	}
	return w.items[i], true
}

// Skills returns the known skills: the last SkillList.
func (w *World) Skills() []Skill {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.skills)
}

// SkillLevel returns the level of skill id, 0 when the character does not
// know it.
func (w *World) SkillLevel(id int32) int32 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return skillLevel(w.skills, id)
}

// skillLevel returns the level of skill id in skills, 0 when it is not
// there.
func skillLevel(skills []Skill, id int32) int32 {
	for _, s := range skills {
		if s.ID == id {
			return s.Level
		}
	}
	return 0
}

// Shortcuts returns the shortcut bar, sorted by slot: the last
// ShortcutInit with the ShortcutRegisters after it and the slots
// DeleteShortcut emptied removed.
func (w *World) Shortcuts() []Shortcut {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.shortcuts)
}

// Shortcut returns the shortcut in slot (the slot plus 12 times the page).
func (w *World) Shortcut(slot int32) (Shortcut, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	i := slices.IndexFunc(w.shortcuts, func(s Shortcut) bool { return s.Slot == slot })
	if i < 0 {
		return Shortcut{}, false
	}
	return w.shortcuts[i], true
}

// setShortcut puts s in its slot; the caller holds w.mu.
func (w *World) setShortcut(s Shortcut) {
	i, found := slices.BinarySearchFunc(w.shortcuts, s.Slot, func(e Shortcut, slot int32) int {
		return cmp.Compare(e.Slot, slot)
	})
	if found {
		w.shortcuts[i] = s
		return
	}
	w.shortcuts = slices.Insert(w.shortcuts, i, s)
}

// deleteShortcut empties slot. The server does not confirm RequestShortcutDel,
// so DeleteShortcut calls it.
func (w *World) deleteShortcut(slot int32) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.shortcuts = slices.DeleteFunc(w.shortcuts, func(s Shortcut) bool { return s.Slot == slot })
}

// Effects returns the buff and debuff icons of the character: the last
// AbnormalStatusUpdate.
func (w *World) Effects() []Effect {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.effects)
}

// Effect returns the effect of skill skillID among the character's icons.
func (w *World) Effect(skillID int32) (Effect, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	i := slices.IndexFunc(w.effects, func(e Effect) bool { return e.SkillID == skillID })
	if i < 0 {
		return Effect{}, false
	}
	return w.effects[i], true
}

// ShortBuff returns the short buff icon: the last ShortBuffStatusUpdate,
// unless it removed the icon.
func (w *World) ShortBuff() (ShortBuffStatusUpdate, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.shortBuff, w.shortBuff.SkillID != 0
}

// PartySpelled returns the last PartySpelled about objectID: the buffs of a
// party member, pet or servitor.
func (w *World) PartySpelled(objectID int32) (PartySpelled, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	p, ok := w.party[objectID]
	if ok {
		p.Effects = slices.Clone(p.Effects)
	}
	return p, ok
}

// AutoSoulShots returns the item IDs of the shots in automatic use
// (ExAutoSoulShot), sorted.
func (w *World) AutoSoulShots() []int32 {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]int32, 0, len(w.autoShots))
	for id := range w.autoShots {
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}

// applyPlay handles the packets of items on the ground, combat, skills,
// shortcuts, shots and buffs; the caller holds w.mu.
func (w *World) applyPlay(p Packet) {
	switch v := p.(type) {
	case SpawnItem:
		w.ground[v.ObjectID] = GroundItem{ObjectID: v.ObjectID, ItemID: v.ItemID, Pos: v.Position(),
			Stackable: v.Stackable, Count: v.Count}
	case DropItem:
		w.ground[v.ObjectID] = GroundItem{ObjectID: v.ObjectID, ItemID: v.ItemID, Pos: v.Position(),
			Stackable: v.Stackable, Count: v.Count, DropperID: v.DropperID}
	case GetItem:
		delete(w.ground, v.ObjectID)
	case Die:
		if v.ObjectID != 0 && v.ObjectID == w.me.ObjectID {
			w.me.Dead, w.me.Death = true, v
		} else if o, ok := w.objects[v.ObjectID]; ok {
			o.Dead = true
		}
	case Revive:
		if v.ObjectID != 0 && v.ObjectID == w.me.ObjectID {
			w.me.Dead, w.me.Death = false, Die{}
		} else if o, ok := w.objects[v.ObjectID]; ok {
			o.Dead = false
		}
	case AutoAttackStart:
		w.setInCombat(v.ObjectID, true)
	case AutoAttackStop:
		w.setInCombat(v.ObjectID, false)
	case SkillList:
		w.skills = slices.Clone(v.Skills)
	case ShortcutInit:
		w.shortcuts = slices.Clone(v.Shortcuts)
		slices.SortStableFunc(w.shortcuts, func(a, b Shortcut) int { return cmp.Compare(a.Slot, b.Slot) })
	case ShortcutRegister:
		w.setShortcut(v.Shortcut)
	case AbnormalStatusUpdate:
		w.effects = slices.Clone(v.Effects)
	case ShortBuffStatusUpdate:
		w.shortBuff = v
	case PartySpelled:
		v.Effects = slices.Clone(v.Effects)
		w.party[v.ObjectID] = v
	case ExAutoSoulShot:
		if v.Enabled {
			w.autoShots[v.ItemID] = true
		} else {
			delete(w.autoShots, v.ItemID)
		}
	}
}

// setInCombat sets the combat stance of the character or of a visible
// object; the caller holds w.mu.
func (w *World) setInCombat(id int32, on bool) {
	if id != 0 && id == w.me.ObjectID {
		w.me.InCombat = on
		return
	}
	if o, ok := w.objects[id]; ok {
		o.InCombat = on
	}
}
