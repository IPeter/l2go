package game

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/storage"
)

// Items that use skills (Plan 10 Phase 7.6):
//
//   - shortcut.go: the shortcut bar, its packets and persistence, and the
//     initial shortcuts of new characters.
//   - shot.go: soulshots and spiritshots, manual and automatic (K14).
//   - item_skill.go: the item handlers of UseItem (ItemSkills, Elixir,
//     CharmOfCourage and the shots), the item reuse stamps and their
//     saving, and herbs used on pick-up (K15).
//   - item_skill_worn.go: the skills of worn items and armor sets (K15).
//   - penalty.go: the grade penalty and EtcStatusUpdate (K16).
//
// Casting (Phase 7.3) casts the items' skills: DoSimultaneousCast for
// potions and herbs, UseMagic for scrolls; it charges the automatic shots
// at a skill's hit through SetShotRecharger.

func init() { registerFeature("itemskills", setupItemSkills) }

// The item handlers this file runs (Mobius handlers/items).
const (
	handlerItemSkills     = "ItemSkills"
	handlerElixir         = "Elixir"
	handlerCharmOfCourage = "CharmOfCourage"
)

// itemReuseRestoreMin is the time left under which a saved item reuse is
// not restored (Mobius restoreItemReuse: remainingTime > 10).
const itemReuseRestoreMin = 10 * time.Millisecond

// exUseSharedGroupItemSubOpcode is the sub-opcode of ExUseSharedGroupItem
// (FE 49).
const exUseSharedGroupItemSubOpcode = 0x49

// itemSkillFeature is the feature's state on the Server (Server.itemSkill).
type itemSkillFeature struct {
	// initial are the initial shortcuts, nil when the data folder has none.
	// Read-only after setup.
	initial *data.InitialShortcuts
}

// playerItemSkill is the feature's state of a player (Player.itemSkill).
// Loop-only.
type playerItemSkill struct {
	bar       shortcutBar // shortcut.go
	autoShots autoShotSet // shot.go
	reuse     itemReuseStamps
	// charm is set by a Charm of Courage until death (Mobius
	// _hasCharmOfCourage); not saved.
	charm bool
	// weaponPenalty and armorPenalty are the grade penalties, 0-4
	// (penalty.go).
	weaponPenalty, armorPenalty int
	// listener applies the worn items' skills and the armor sets
	// (item_skill_worn.go); nil until the player is selected.
	listener *wornItemListener
	// setBonus is the armor set whose base stat bonus is on, nil for none.
	setBonus *data.ArmorSet
	// loaded is set once the shortcuts and the item reuse are loaded; until
	// then a save keeps the stored ones.
	loaded bool
}

// setupItemSkills loads the initial shortcuts and registers the packets
// and hooks.
func setupItemSkills(s *Server) error {
	if dir := s.settings.DataDir; dir != "" {
		switch sc, err := data.LoadInitialShortcuts(dir); {
		case err == nil:
			s.itemSkill.initial = sc
		case errors.Is(err, fs.ErrNotExist):
			s.log.Info("game: no initial shortcuts, new characters start with an empty bar", "file", filepath.Join(dir, data.InitialShortcutsFile))
		default:
			return fmt.Errorf("initial shortcuts: %w", err)
		}
	}
	s.SetShotRecharger(func(p *Player, physical, magic bool) { s.RechargeShots(p, physical, magic) })
	s.OnDie(s.itemSkillsDie)
	// Mobius Player.updateShortcuts after RequestAcquireSkill.
	s.OnSkillLearned(func(p *Player, sk *skills.Skill) { s.UpdateSkillShortcuts(p, sk.ID, sk.Level) })
	return errors.Join(
		s.Register(clientpackets.OpRequestShortcutReg, s.handleShortcutReg),
		s.Register(clientpackets.OpRequestShortcutDel, s.handleShortcutDel),
		s.RegisterEx(clientpackets.ExOpRequestAutoSoulShot, s.handleRequestAutoSoulShot),
	)
}

// loadWornItems is the part of Mobius Player.restore that the item
// listeners do while the inventory loads: the worn items' skills and the
// armor set, before the saved skills. It runs on the loop at character
// selection, before loadSkills.
func (s *Server) loadWornItems(p *Player) {
	if p.inv.Inventory == nil {
		return
	}
	l := &wornItemListener{s: s, p: p}
	p.itemSkill.listener = l
	for slot := range storage.PaperdollSlot(storage.PaperdollTotalSlots) {
		if it := p.inv.Paperdoll(slot); it != nil && it.Slot() == slot {
			s.wornItemSkillsOn(p, it, true)
		}
	}
	s.loadArmorSet(p)
	s.refreshArmorSet(p, true)
	p.inv.AddListener(l)
}

// loadItemState is the rest of Mobius Player.load after the skills: the
// shortcuts, the item reuse (with ExUseSharedGroupItem for a shared group in
// the inventory) and the grade penalty. It runs on the loop at character
// selection, after loadSkills.
func (s *Server) loadItemState(p *Player) {
	s.loadShortcuts(p)
	s.restoreItemReuse(p)
	p.itemSkill.loaded = true
	s.refreshExpertisePenalty(p)
}

// itemSkillsDie is the OnDie hook: a Charm of Courage ends with death
// (Playable.doDie; reviving on the spot needs sieges).
func (s *Server) itemSkillsDie(victim, _ Creature) {
	p, ok := victim.(*Player)
	if !ok || !p.itemSkill.charm {
		return
	}
	p.itemSkill.charm = false
	if p.InGame() {
		p.Send(etcStatusPacket(p))
	}
}

// itemReuse is an item reuse stamp (Mobius TimeStamp of an item).
type itemReuse struct {
	itemID, objectID int32
	group            int
	delay            time.Duration
	end              time.Time
}

// itemReuseStamps are a player's item reuse stamps by object ID, in Mobius's
// map order. The zero value is empty.
type itemReuseStamps struct {
	byObject map[int32]itemReuse
	order    chmOrder
}

func (r *itemReuseStamps) put(st itemReuse) {
	if r.byObject == nil {
		r.byObject = make(map[int32]itemReuse)
	}
	if _, ok := r.byObject[st.objectID]; !ok {
		r.order.put(int(st.objectID))
	}
	r.byObject[st.objectID] = st
}

func (r *itemReuseStamps) all() []itemReuse {
	keys := r.order.keys()
	out := make([]itemReuse, len(keys))
	for i, k := range keys {
		out[i] = r.byObject[int32(k)] //nolint:gosec // object IDs
	}
	return out
}

// addItemReuse is Creature.addTimeStampItem: it goes under reuse for delay
// from now, or until end when end is not zero (a restored stamp).
func (s *Server) addItemReuse(p *Player, it *inventory.Item, delay time.Duration, end time.Time) {
	if end.IsZero() {
		end = s.Now().Add(delay)
	}
	t := it.Template()
	p.itemSkill.reuse.put(itemReuse{itemID: it.ItemID(), objectID: it.ObjectID(), group: t.SharedReuseGroup, delay: delay, end: end})
}

// ItemReuseLeft returns how long the item with the object ID stays under
// reuse for p, 0 when it is ready (Mobius getItemRemainingReuseTime).
// Loop-only.
func (s *Server) ItemReuseLeft(p *Player, objectID int32) time.Duration {
	st, ok := p.itemSkill.reuse.byObject[objectID]
	if !ok {
		return 0
	}
	return max(st.end.Sub(s.Now()), 0)
}

// groupReuseLeft is Creature.getReuseDelayOnGroup: the time left of the
// first stamp of the shared reuse group that still runs.
func (s *Server) groupReuseLeft(p *Player, group int) time.Duration {
	if group <= 0 {
		return 0
	}
	now := s.Now()
	for _, st := range p.itemSkill.reuse.all() {
		if st.group == group && now.Before(st.end) {
			return st.end.Sub(now)
		}
	}
	return 0
}

// reuseText is Mobius's "There are ... remaining in S1's re-use time.".
func reuseText(left time.Duration, name string) string {
	ms := left.Milliseconds()
	hours, minutes, seconds := ms/3600000, (ms%3600000)/60000, (ms/1000)%60
	switch {
	case hours > 0:
		return fmt.Sprintf("There are %d hour(s), %d minute(s), and %d second(s) remaining in %s's re-use time.", hours, minutes, seconds, name)
	case minutes > 0:
		return fmt.Sprintf("There are %d minute(s), %d second(s) remaining in %s's re-use time.", minutes, seconds, name)
	}
	return fmt.Sprintf("There are %d second(s) remaining in %s's re-use time.", seconds, name)
}

// exUseSharedGroupItemPacket builds ExUseSharedGroupItem (Mobius: the item,
// the group, the seconds left and the full reuse in seconds). The
// serverpackets package has no codec for it yet.
func exUseSharedGroupItemPacket(itemID int32, group int, left, total time.Duration) []byte {
	w := packets.NewWriter(19)
	w.WriteUint8(serverpackets.OpExPacket)
	w.WriteUint16(exUseSharedGroupItemSubOpcode)
	w.WriteInt32(itemID)
	w.WriteInt32(int32(group))                       //nolint:gosec // small group IDs
	w.WriteInt32(int32(left.Milliseconds() / 1000))  //nolint:gosec // seconds of a reuse
	w.WriteInt32(int32(total.Milliseconds() / 1000)) //nolint:gosec // seconds of a reuse
	return w.Bytes()
}

// itemUseReady is the part of Mobius UseItem between the item's conditions
// and the use: a stunned, paralyzed or sleeping player does nothing; an
// item under reuse (its own or its shared group's) is refused with the time
// left.
func (s *Server) itemUseReady(p *Player, it *inventory.Item) bool {
	el := p.Effects()
	if el.IsAffected(skills.FlagStunned) || el.IsAffected(skills.FlagParalyzed) || el.IsAffected(skills.FlagSleep) {
		return false
	}
	t := it.Template()
	if t.ReuseDelay <= 0 {
		return true
	}
	total := time.Duration(t.ReuseDelay) * time.Millisecond
	for _, left := range []time.Duration{s.ItemReuseLeft(p, it.ObjectID()), s.groupReuseLeft(p, t.SharedReuseGroup)} {
		if left > 0 {
			p.SendMessage(reuseText(left, t.Name))
			if t.SharedReuseGroup > 0 {
				p.Send(exUseSharedGroupItemPacket(it.ItemID(), t.SharedReuseGroup, left, total))
			}
			return false
		}
	}
	return true
}

// equipWhileCasting is UseItem's refusal to put items on or off during a
// cast: "You may not equip items while casting or performing a skill." and
// ActionFailed. It reports whether it refused.
func (s *Server) equipWhileCasting(p *Player) bool {
	if !s.CastingNow(p) && !s.CastingSimultaneouslyNow(p) {
		return false
	}
	p.SendSystemMessage(sysMsgNoEquipWhileCasting)
	p.SendActionFailed()
	return true
}

// System messages of the item handlers.
const (
	sysMsgNoEquipWhileCasting serverpackets.SystemMessageID = 104 // You may not equip items while casting or performing a skill.
	sysMsgUnsuitableTerms     serverpackets.SystemMessageID = 113 // $s1 cannot be used due to unsuitable terms.
)

// useItemHandler runs the handler of an item that is not worn (Mobius
// UseItem: handler.onItemUse, then the item's reuse stamp and
// ExUseSharedGroupItem when it was used). It reports false for a handler
// that does not exist yet.
func (s *Server) useItemHandler(p *Player, it *inventory.Item, ctrl bool) bool {
	t := it.Template()
	used, known := s.runItemHandler(p, it, ctrl)
	if !known {
		return false
	}
	if used && t.ReuseDelay > 0 {
		d := time.Duration(t.ReuseDelay) * time.Millisecond
		s.addItemReuse(p, it, d, time.Time{})
		if t.SharedReuseGroup > 0 {
			p.Send(exUseSharedGroupItemPacket(it.ItemID(), t.SharedReuseGroup, d, d))
		}
	}
	return true
}

// runItemHandler is IItemHandler.onItemUse of the item's handler: used
// reports that it was used, known that the handler exists.
func (s *Server) runItemHandler(p *Player, it *inventory.Item, ctrl bool) (used, known bool) {
	switch it.Template().Etc.Handler {
	case handlerSoulShots, handlerSpiritShot, handlerBlessedSpiritShot:
		return s.useShot(p, it), true
	case handlerItemSkills, handlerElixir:
		return s.useItemSkills(p, it, ctrl), true
	case handlerCharmOfCourage:
		return s.useCharmOfCourage(p, it), true
	}
	return false, false
}

// itemSkillReuseOK is ItemSkillsTemplate.checkReuse: the item's reuse
// without a skill, else the skill's; a refusal tells the time left, with the
// item's name for an item or a static skill, else the skill's.
func (s *Server) itemSkillReuseOK(p *Player, it *inventory.Item, sk *skills.Skill) bool {
	var left time.Duration
	name := it.Template().Name
	if sk == nil {
		left = s.ItemReuseLeft(p, it.ObjectID())
	} else {
		left = s.SkillReuseLeft(p, sk)
		if !sk.IsStatic() {
			name = sk.Name
		}
	}
	if left <= 0 {
		return true
	}
	p.SendMessage(reuseText(left, name))
	return false
}

// itemTypeIs reports the item's etc item type.
func itemTypeIs(t *data.Item, et data.EtcItemType) bool { return t.Etc != nil && t.Etc.Type == et }

// useItemSkills is Mobius ItemSkillsTemplate.onItemUse (ItemSkills and
// Elixir; the Olympiad and events do not exist yet): the item's reuse; then
// for each of its skills the conditions on p's target, the skill's reuse,
// a disabled skill, no other cast for anything but a potion, elixir or
// scroll; a simultaneous skill, or a static one of an item with an
// immediate effect, is cast at once in the second slot (potions, herbs),
// the others go through UseMagic after the intention went idle (scrolls).
// A capsule or SKILL_REDUCE item with an immediate effect and no skill that
// consumes an item is used up here; the skill's itemConsumeId is taken when
// its cast starts.
func (s *Server) useItemSkills(p *Player, it *inventory.Item, ctrl bool) bool {
	if !s.itemSkillReuseOK(p, it, nil) {
		return false
	}
	t := it.Template()
	refs := normalSkills(t)
	if len(refs) == 0 {
		s.log.Info("game: the item has no skill for its handler", append(p.logArgs(), "item", it.ItemID())...)
		return false
	}
	env := s.SkillEnv()
	consumeSkill := false
	for _, ref := range refs {
		sk, ok := s.Skill(ref.ID, ref.Level)
		if !ok {
			continue
		}
		if sk.ItemConsumeID > 0 {
			consumeSkill = true
		}
		if !sk.CheckCondition(env, p.skill, p.skill.Target(), false) {
			return false
		}
		if !s.itemSkillReuseOK(p, it, sk) || s.SkillDisabled(p, sk) {
			return false
		}
		if !itemTypeIs(t, data.EtcPotion) && !itemTypeIs(t, data.EtcElixir) && !itemTypeIs(t, data.EtcScroll) && s.CastingNow(p) {
			return false
		}
		if sk.SimultaneousCast || ((t.ImmediateEffect || t.ExImmediateEffect) && sk.IsStatic()) {
			s.DoSimultaneousCast(p, sk)
			continue
		}
		s.IdleIntention(p)
		if !s.UseMagic(p, sk, ctrl, false) {
			return false
		}
	}
	consume := (t.DefaultAction == data.ActionCapsule || t.DefaultAction == data.ActionSkillReduce) && !consumeSkill && t.ImmediateEffect
	if consume && !s.DestroyItem(p, it, 1, false) {
		p.SendSystemMessage(sysMsgIncorrectCount)
		return false
	}
	return true
}

// useCharmOfCourage is Mobius CharmOfCourage.onItemUse: the charm's grade
// must reach the player's level band (below 20, 40, 52, 61, 76, else S);
// one charm is used up, the player has the charm until death and gets
// EtcStatusUpdate.
func (s *Server) useCharmOfCourage(p *Player, it *inventory.Item) bool {
	var band int
	switch lvl := p.Level(); {
	case lvl < 20:
		band = 0
	case lvl < 40:
		band = 1
	case lvl < 52:
		band = 2
	case lvl < 61:
		band = 3
	case lvl < 76:
		band = 4
	default:
		band = 5
	}
	if it.Template().Grade.Level() < band {
		p.SendSystemMessage(sysMsgUnsuitableTerms, s.itemNameParam(it.ItemID()))
		return false
	}
	if !s.DestroyItem(p, it, 1, false) {
		return false
	}
	p.itemSkill.charm = true
	p.Send(etcStatusPacket(p))
	return true
}

// useHerb is the herb part of Mobius Player.doPickupItem: an item with an
// immediate effect runs its handler on pick-up and is destroyed; it never
// enters the inventory.
func (s *Server) useHerb(p *Player, it *inventory.Item) {
	if t := it.Template(); t.Etc != nil && t.Etc.Handler != "" {
		if _, known := s.runItemHandler(p, it, false); !known {
			s.log.Warn("game: no item handler for a herb", append(p.logArgs(), "item", it.ItemID(), "handler", t.Etc.Handler)...)
		}
	}
}

// castHerbSkills is the herb part of Mobius Player.addItem (auto-loot): each
// of the item's skills is cast at once in the simultaneous slot, then the
// player's info goes out.
func (s *Server) castHerbSkills(p *Player, t *data.Item) {
	for _, ref := range normalSkills(t) {
		if sk, ok := s.Skill(ref.ID, ref.Level); ok {
			s.DoSimultaneousCast(p, sk)
		}
	}
	if p.InGame() {
		p.BroadcastUserInfo()
	}
}

// storeItemReuse puts the item reuse that still runs into a snapshot
// (Mobius storeItemReuseDelay).
func (p *Player) storeItemReuse(c *storage.Character) {
	now := p.srv.Now()
	var rows []storage.ItemReuse
	for _, st := range p.itemSkill.reuse.all() {
		if now.Before(st.end) {
			rows = append(rows, storage.ItemReuse{ItemID: st.itemID, ObjectID: st.objectID, ReuseDelay: st.delay.Milliseconds(), ReuseEnd: st.end.UnixMilli()})
		}
	}
	c.ItemReuse = rows
}

// restoreItemReuse is Mobius restoreItemReuse: each saved reuse goes on the
// first item of its template in the inventory, else in the warehouse, when
// the item has a reuse delay and more than 10 ms are left;
// ExUseSharedGroupItem tells the client of a shared group in the inventory.
// The rows are used up.
func (s *Server) restoreItemReuse(p *Player) {
	now := s.Now()
	for _, r := range p.char.ItemReuse {
		it, inInventory := p.inv.ByItemID(r.ItemID), true
		if it == nil && p.items.warehouse != nil {
			it, inInventory = p.items.warehouse.ByItemID(r.ItemID), false
		}
		if it == nil || it.Template().ReuseDelay <= 0 {
			continue
		}
		end := time.UnixMilli(r.ReuseEnd)
		left := end.Sub(now)
		if left <= itemReuseRestoreMin {
			continue
		}
		delay := time.Duration(r.ReuseDelay) * time.Millisecond
		s.addItemReuse(p, it, delay, end)
		if t := it.Template(); inInventory && t.Etc != nil && t.SharedReuseGroup > 0 {
			p.Send(exUseSharedGroupItemPacket(it.ItemID(), t.SharedReuseGroup, left, delay))
		}
	}
	p.char.ItemReuse = nil
}
