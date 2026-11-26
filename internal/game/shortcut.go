package game

import (
	"cmp"
	"slices"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/storage"
)

// The shortcut bar (Mobius Shortcuts): slots 0-11 on pages 0-10, kept by
// slot + page × 12 in the order of Mobius's ConcurrentHashMap. A skill
// shortcut takes the level the player knows; an item shortcut needs the
// item and goes with it. Deleting a shot's shortcut turns its automatic use
// off and resends the whole bar. The shortcuts of the active class are
// saved with the character (character_shortcuts, decision X3).

const (
	// shortcutsPerPage is MAX_SHORTCUTS_PER_BAR.
	shortcutsPerPage = 12
	// shortcutMaxPage is the last page RequestShortcutReg and
	// RequestShortcutDel accept.
	shortcutMaxPage = 10
)

// shortcut is one slot of the bar (Mobius Shortcut).
type shortcut struct {
	slot, page int32
	typ        int32 // storage.ShortcutItem, ...
	id, level  int32
}

// key is the slot number across pages, the bar's map key.
func (sc shortcut) key() int { return int(sc.slot + sc.page*shortcutsPerPage) }

// packet is the shortcut as ShortcutInit and ShortcutRegister send it.
func (sc shortcut) packet() serverpackets.Shortcut {
	return serverpackets.Shortcut{
		Type: serverpackets.ShortcutType(sc.typ), Slot: sc.slot, Page: sc.page, ID: sc.id, Level: sc.level,
	}
}

// shortcutBar is a player's bar of the active class, and the saved rows of
// its other classes.
type shortcutBar struct {
	byKey map[int]shortcut
	order chmOrder
	other []storage.Shortcut
}

// put stores sc in its slot (Mobius _shortcuts.put).
func (b *shortcutBar) put(sc shortcut) {
	if b.byKey == nil {
		b.byKey = make(map[int]shortcut)
	}
	k := sc.key()
	if _, ok := b.byKey[k]; !ok {
		b.order.put(k)
	}
	b.byKey[k] = sc
}

// remove clears a slot and returns what was there.
func (b *shortcutBar) remove(slot, page int32) (shortcut, bool) {
	k := int(slot + page*shortcutsPerPage)
	sc, ok := b.byKey[k]
	if ok {
		delete(b.byKey, k)
		b.order.remove(k)
	}
	return sc, ok
}

// all returns the shortcuts in Mobius's map order.
func (b *shortcutBar) all() []shortcut {
	keys := b.order.keys()
	out := make([]shortcut, len(keys))
	for i, k := range keys {
		out[i] = b.byKey[k]
	}
	return out
}

// shortcutInitPacket builds ShortcutInit with p's whole bar.
func shortcutInitPacket(p *Player) []byte {
	all := p.itemSkill.bar.all()
	list := make([]serverpackets.Shortcut, len(all))
	for i, sc := range all {
		list[i] = sc.packet()
	}
	return serverpackets.NewShortcutInitPacket(list)
}

// loadShortcuts restores the bar of the active class (Shortcuts.restoreMe)
// in the order Mobius's table gives the rows (by slot, then page). An item
// shortcut whose item is gone is dropped; Mobius deletes it with packets the
// client does not see yet.
func (s *Server) loadShortcuts(p *Player) {
	b := &p.itemSkill.bar
	var rows []storage.Shortcut
	classIndex := p.classIndex()
	for _, r := range p.char.Shortcuts {
		if r.ClassIndex != classIndex {
			b.other = append(b.other, r)
			continue
		}
		rows = append(rows, r)
	}
	slices.SortStableFunc(rows, func(x, y storage.Shortcut) int {
		return cmp.Or(cmp.Compare(x.Slot, y.Slot), cmp.Compare(x.Page, y.Page))
	})
	for _, r := range rows {
		if r.Type == storage.ShortcutItem && p.inv.ByObjectID(r.ID) == nil {
			continue
		}
		b.put(shortcut{slot: r.Slot, page: r.Page, typ: r.Type, id: r.ID, level: r.Level})
	}
}

// storeShortcuts puts the bar into a snapshot: the other classes' rows as
// loaded, then the active class's in slot and page order.
func (p *Player) storeShortcuts(c *storage.Character) {
	b := &p.itemSkill.bar
	rows := slices.Clone(b.other)
	mine := b.all()
	slices.SortFunc(mine, func(x, y shortcut) int {
		return cmp.Or(cmp.Compare(x.slot, y.slot), cmp.Compare(x.page, y.page))
	})
	for _, sc := range mine {
		rows = append(rows, storage.Shortcut{ClassIndex: p.classIndex(), Slot: sc.slot, Page: sc.page, Type: sc.typ, ID: sc.id, Level: sc.level})
	}
	c.Shortcuts = rows
}

// registerShortcut is Mobius Shortcuts.registerShortcut: an item shortcut
// needs the item in the inventory; the shortcut replaces the slot's.
func (s *Server) registerShortcut(p *Player, sc shortcut) {
	if sc.typ == storage.ShortcutItem && p.inv.ByObjectID(sc.id) == nil {
		return
	}
	p.itemSkill.bar.put(sc)
	s.RequestSave(p)
}

// deleteShortcut is Mobius Shortcuts.deleteShortcut: the slot is cleared;
// for a shot's shortcut the shot's automatic use goes off; then the whole
// bar and every shot still automatic are resent. gone is an item that just
// left the inventory, which Mobius still finds there.
func (s *Server) deleteShortcut(p *Player, slot, page int32, gone *inventory.Item) {
	old, ok := p.itemSkill.bar.remove(slot, page)
	if !ok {
		return
	}
	s.RequestSave(p)
	if old.typ == storage.ShortcutItem {
		it := p.inv.ByObjectID(old.id)
		if it == nil && gone != nil && gone.ObjectID() == old.id {
			it = gone
		}
		if it != nil && it.Template().Etc != nil && it.Template().Etc.Type == data.EtcShot &&
			p.itemSkill.autoShots.remove(it.ItemID()) {
			p.Send(serverpackets.NewExAutoSoulShotPacket(serverpackets.ExAutoSoulShot{ItemID: it.ItemID()}))
		}
	}
	p.Send(shortcutInitPacket(p))
	for _, id := range p.itemSkill.autoShots.list() {
		p.Send(serverpackets.NewExAutoSoulShotPacket(serverpackets.ExAutoSoulShot{ItemID: id, Enabled: true}))
	}
}

// removeItemShortcut is Mobius removeItemFromShortcut: the first shortcut
// of an item that left the inventory goes.
func (s *Server) removeItemShortcut(p *Player, it *inventory.Item) {
	for _, sc := range p.itemSkill.bar.all() {
		if sc.typ == storage.ShortcutItem && sc.id == it.ObjectID() {
			s.deleteShortcut(p, sc.slot, sc.page, it)
			return
		}
	}
}

// itemsRemoved runs removeItemShortcut for the items the inventory updates
// report removed.
func (s *Server) itemsRemoved(p *Player, ups []inventory.Update) {
	for _, u := range ups {
		if u.Change == inventory.Removed && u.Item != nil && p.inv.ByObjectID(u.Item.ObjectID()) == nil {
			s.removeItemShortcut(p, u.Item)
		}
	}
}

// deleteSkillShortcuts removes the shortcuts of a skill p forgot for good
// (Mobius removeSkill with store); the skills 3080-3259 keep theirs.
func (s *Server) deleteSkillShortcuts(p *Player, id int) {
	if id >= 3080 && id <= 3259 {
		return
	}
	for _, sc := range p.itemSkill.bar.all() {
		if sc.typ == storage.ShortcutSkill && int(sc.id) == id {
			s.deleteShortcut(p, sc.slot, sc.page, nil)
		}
	}
}

// UpdateSkillShortcuts puts a new level on the shortcuts of a skill (Mobius
// Player.updateShortcuts): each gets ShortcutRegister and is saved. Learning
// calls it after a new level (RequestAcquireSkill). Loop-only.
func (s *Server) UpdateSkillShortcuts(p *Player, id, level int) {
	for _, sc := range p.itemSkill.bar.all() {
		if sc.typ == storage.ShortcutSkill && int(sc.id) == id {
			sc.level = int32(level) //nolint:gosec // skill levels are small
			p.Send(serverpackets.NewShortcutRegisterPacket(sc.packet()))
			s.registerShortcut(p, sc)
		}
	}
}

// handleShortcutReg handles RequestShortcutReg (0x33) as Mobius does: a page
// outside 0-10 is ignored; an item, action, macro or recipe goes on the
// slot with level -1 and the client gets ShortcutRegister (also when the
// item is missing and nothing was stored); a skill takes the level p knows,
// and nothing happens for an unknown skill.
func (s *Server) handleShortcutReg(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestShortcutReg(body)
	if err != nil {
		s.log.Debug("game: bad RequestShortcutReg", append(p.logArgs(), "err", err)...)
		return
	}
	if req.Page < 0 || req.Page > shortcutMaxPage {
		return
	}
	sc := shortcut{slot: req.Slot, page: req.Page, typ: req.Type, id: req.ID, level: -1}
	switch req.Type {
	case storage.ShortcutItem, storage.ShortcutAction, storage.ShortcutMacro, storage.ShortcutRecipe:
	case storage.ShortcutSkill:
		level := p.SkillLevel(int(req.ID))
		if level <= 0 {
			return
		}
		sc.level = int32(level) //nolint:gosec // skill levels are small
	default:
		return
	}
	s.registerShortcut(p, sc)
	p.Send(serverpackets.NewShortcutRegisterPacket(sc.packet()))
}

// handleShortcutDel handles RequestShortcutDel (0x35): the slot is cleared
// (deleteShortcut); the client needs no answer.
func (s *Server) handleShortcutDel(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestShortcutDel(body)
	if err != nil {
		s.log.Debug("game: bad RequestShortcutDel", append(p.logArgs(), "err", err)...)
		return
	}
	if req.Page < 0 || req.Page > shortcutMaxPage {
		return
	}
	s.deleteShortcut(p, req.Slot, req.Page, nil)
}

// NewCharacterShortcuts returns the shortcut bar of a new character (Mobius
// InitialShortcutData.registerAllShortcuts at CharacterCreate): the
// shortcuts of every class, then the class's own. An item shortcut points
// at the character's first item of the template and is left out without
// one; a skill shortcut needs the skill among c.Skills, with the level the
// data gives; macros wait for macros and are left out. Call it once the
// character has its items and skills. Safe on any goroutine.
func (s *Server) NewCharacterShortcuts(c *storage.Character) []storage.Shortcut {
	initial := s.itemSkill.initial
	if initial == nil {
		return nil
	}
	var out []storage.Shortcut
	for _, sc := range initial.For(int(c.ClassID)) {
		row := storage.Shortcut{ClassIndex: baseClassIndex, Slot: int32(sc.Slot), Page: int32(sc.Page), ID: int32(sc.ID), Level: int32(sc.Level)} //nolint:gosec // small data values
		switch sc.Type {
		case data.ShortcutTypeItem:
			i := slices.IndexFunc(c.Items, func(it storage.Item) bool { return it.ItemID == int32(sc.ID) }) //nolint:gosec // item IDs are small
			if i < 0 {
				continue
			}
			row.Type, row.ID = storage.ShortcutItem, c.Items[i].ObjectID
		case data.ShortcutTypeSkill:
			if !slices.ContainsFunc(c.Skills, func(k storage.CharacterSkill) bool { return k.ClassIndex == baseClassIndex && k.ID == int32(sc.ID) }) { //nolint:gosec // skill IDs are small
				continue
			}
			row.Type = storage.ShortcutSkill
		case data.ShortcutTypeAction:
			row.Type = storage.ShortcutAction
		case data.ShortcutTypeRecipe:
			row.Type = storage.ShortcutRecipe
		default:
			continue // macros: no macro presets yet
		}
		// A later shortcut of the same slot replaces an earlier one.
		out = slices.DeleteFunc(out, func(o storage.Shortcut) bool { return o.Slot == row.Slot && o.Page == row.Page })
		out = append(out, row)
	}
	return out
}
