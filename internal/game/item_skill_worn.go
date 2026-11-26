package game

import (
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
)

// The skills of worn items and armor sets (decision K15), as Mobius
// Inventory.ItemSkillsListener and ArmorSetListener: a worn item gives its
// NORMAL skills, and its ON_ENCHANT_4 skill from +4; a complete armor set
// gives its skills, with the set's shield its shield skills, with every
// part at +6 its +6 skills, and its base stat bonus (FuncArmorSet). The
// skills are not saved; SkillList follows every change.

// armorSetOwner owns the modifiers of the armor set's base stat bonus.
// Item owners use object IDs, which are positive.
var armorSetOwner = stats.Owner{Kind: stats.OwnerItem, ID: -1}

// wornItemListener is the inventory listener of a player's worn items: the
// armor set first, then the item's skills, as Mobius adds its listeners.
type wornItemListener struct {
	s *Server
	p *Player
}

var _ inventory.Listener = (*wornItemListener)(nil)

// Equipped is ArmorSetListener.notifyEquiped and
// ItemSkillsListener.notifyEquiped.
func (l *wornItemListener) Equipped(_ *inventory.Inventory, slot storage.PaperdollSlot, it *inventory.Item) {
	l.s.armorSetOn(l.p, slot, it)
	l.s.wornItemSkillsOn(l.p, it, false)
	l.s.refreshArmorSet(l.p, false)
}

// Unequipped is ArmorSetListener.notifyUnequiped and
// ItemSkillsListener.notifyUnequiped; taking an item off also checks the
// grade penalty (Inventory.unEquipItemInSlot).
func (l *wornItemListener) Unequipped(_ *inventory.Inventory, slot storage.PaperdollSlot, it *inventory.Item) {
	l.s.armorSetOff(l.p, slot, it)
	l.s.wornItemSkillsOff(l.p, it)
	l.s.refreshArmorSet(l.p, false)
	l.s.refreshExpertisePenalty(l.p)
}

// itemSkillOf returns the first skill of the given type of an item
// template, nil for none.
func (s *Server) itemSkillOf(t *data.Item, typ data.ItemSkillType) *skills.Skill {
	for _, ref := range t.Skills {
		if ref.Type == typ {
			if sk, ok := s.Skill(ref.ID, ref.Level); ok {
				return sk
			}
		}
	}
	return nil
}

// skillsChanged sends what a change of known skills needs: SkillList and,
// when an active skill came, SkillCoolTime. quiet sends nothing (loading).
func (s *Server) skillsChanged(p *Player, update, stamps, quiet bool) {
	if !update || quiet {
		return
	}
	p.SendSkillList()
	if stamps && p.InGame() {
		p.Send(s.SkillCoolTimePacket(p))
	}
}

// addWornSkill gives p an item's or set's skill (addSkill without store);
// an active skill without a reuse goes under the item's equip reuse delay.
// It reports whether the skill is active.
func (s *Server) addWornSkill(p *Player, sk *skills.Skill, equipDelay int) bool {
	s.AddSkill(p, sk, false)
	if !sk.IsActive() {
		return false
	}
	if s.SkillReuseLeft(p, sk) <= 0 && equipDelay > 0 {
		d := time.Duration(equipDelay) * time.Millisecond
		s.AddSkillReuse(p, sk, d)
		s.DisableSkill(p, sk, d)
	}
	return true
}

// wornItemSkillsOn is ItemSkillsListener.notifyEquiped: the +4 skill from
// enchant 4, then each NORMAL skill.
func (s *Server) wornItemSkillsOn(p *Player, it *inventory.Item, quiet bool) {
	s.augmentOn(p, it, quiet) // augment.go: the augmentation's bonuses come first
	t := it.Template()
	update, stamps := false, false
	if it.Enchant() >= 4 {
		if sk := s.itemSkillOf(t, data.ItemSkillOnEnchant4); sk != nil {
			s.AddSkill(p, sk, false)
			update = true
		}
	}
	for _, ref := range normalSkills(t) {
		sk, ok := s.Skill(ref.ID, ref.Level)
		if !ok {
			s.log.Warn("game: a worn item has a skill the data does not have", append(p.logArgs(), "item", it.ItemID(), "skill", ref.ID, "level", ref.Level)...)
			continue
		}
		if s.addWornSkill(p, sk, t.EquipReuseDelay) {
			stamps = true
		}
		update = true
	}
	s.skillsChanged(p, update, stamps, quiet)
}

// wornItemSkillsOff is ItemSkillsListener.notifyUnequiped: the +4 skill and
// the NORMAL skills go (their effects end for passives); for armor, the
// skills of the other worn items that are now missing come back; the
// item's unequip skill is used on the player.
func (s *Server) wornItemSkillsOff(p *Player, it *inventory.Item) {
	s.augmentOff(p, it, false) // augment.go: the augmentation's bonuses go first
	t := it.Template()
	update, stamps := false, false
	if it.Enchant() >= 4 {
		if sk := s.itemSkillOf(t, data.ItemSkillOnEnchant4); sk != nil {
			s.RemoveSkill(p, sk.ID, false, sk.IsPassive())
			update = true
		}
	}
	for _, ref := range normalSkills(t) {
		if sk, ok := s.Skill(ref.ID, ref.Level); ok {
			s.RemoveSkill(p, sk.ID, false, sk.IsPassive())
			update = true
		}
	}
	if t.Armor != nil {
		for _, other := range p.inv.Items() {
			if !other.Equipped() || other == it {
				continue
			}
			for _, ref := range normalSkills(other.Template()) {
				if p.SkillLevel(ref.ID) != 0 {
					continue
				}
				if sk, ok := s.Skill(ref.ID, ref.Level); ok {
					// Mobius uses the equip delay of the item taken off.
					if s.addWornSkill(p, sk, t.EquipReuseDelay) {
						stamps = true
					}
					update = true
				}
			}
		}
	}
	if sk := s.itemSkillOf(t, data.ItemSkillOnUnequip); sk != nil {
		sk.Activate(s.SkillEnv(), p.skill, []skills.Creature{p.skill})
	}
	s.skillsChanged(p, update, stamps, false)
}

// armorSetContainsItem is ArmorSet.containItem: the item is the set's part
// for the slot.
func armorSetContainsItem(set *data.ArmorSet, slot storage.PaperdollSlot, id int) bool {
	has := func(ids []int) bool {
		for _, v := range ids {
			if v == id {
				return true
			}
		}
		return false
	}
	switch slot {
	case storage.PaperdollChest:
		return set.Chest == id
	case storage.PaperdollLegs:
		return has(set.Legs)
	case storage.PaperdollHead:
		return has(set.Head)
	case storage.PaperdollGloves:
		return has(set.Gloves)
	case storage.PaperdollFeet:
		return has(set.Feet)
	}
	return false
}

// wornID returns the template ID worn in a slot, 0 for none.
func wornID(p *Player, slot storage.PaperdollSlot) (int, int32) {
	if it := p.inv.Paperdoll(slot); it != nil {
		return int(it.ItemID()), it.Enchant()
	}
	return 0, 0
}

// armorSetComplete is ArmorSet.containAll(player).
func armorSetComplete(p *Player, set *data.ArmorSet) bool {
	legs, _ := wornID(p, storage.PaperdollLegs)
	head, _ := wornID(p, storage.PaperdollHead)
	gloves, _ := wornID(p, storage.PaperdollGloves)
	feet, _ := wornID(p, storage.PaperdollFeet)
	return set.ContainsAll(set.Chest, legs, head, gloves, feet)
}

// armorSetShield is ArmorSet.containShield(player).
func armorSetShield(p *Player, set *data.ArmorSet) bool {
	id, _ := wornID(p, storage.PaperdollLHand)
	return id != 0 && set.ContainsShield(id)
}

// armorSetEnchanted6 is ArmorSet.isEnchanted6: the set is complete and
// each of its parts is at +6 or more.
func armorSetEnchanted6(p *Player, set *data.ArmorSet) bool {
	if !armorSetComplete(p, set) {
		return false
	}
	if _, e := wornID(p, storage.PaperdollChest); e < 6 {
		return false
	}
	for _, part := range []struct {
		ids  []int
		slot storage.PaperdollSlot
	}{
		{set.Legs, storage.PaperdollLegs},
		{set.Gloves, storage.PaperdollGloves},
		{set.Head, storage.PaperdollHead},
		{set.Feet, storage.PaperdollFeet},
	} {
		if len(part.ids) == 0 {
			continue
		}
		if id, e := wornID(p, part.slot); id == 0 || e < 6 {
			return false
		}
	}
	return true
}

// chestSet returns the armor set of p's chest item.
func (s *Server) chestSet(p *Player) (*data.ArmorSet, bool) {
	d := s.SkillData()
	if d == nil {
		return nil, false
	}
	id, _ := wornID(p, storage.PaperdollChest)
	if id == 0 {
		return nil, false
	}
	return d.ArmorSet(id)
}

// addSetSkills adds a list of set skills; it reports whether one was added
// and whether one was active.
func (s *Server) addSetSkills(p *Player, refs []data.SkillRef, equipDelay int, withStamps bool) (update, stamps bool) {
	for _, ref := range refs {
		sk, ok := s.Skill(ref.ID, ref.Level)
		if !ok {
			s.log.Warn("game: an armor set has a skill the data does not have", append(p.logArgs(), "skill", ref.ID, "level", ref.Level)...)
			continue
		}
		if withStamps {
			if s.addWornSkill(p, sk, equipDelay) {
				stamps = true
			}
		} else {
			s.AddSkill(p, sk, false)
		}
		update = true
	}
	return update, stamps
}

// armorSetOn is ArmorSetListener.notifyEquiped: a part of the chest's set
// completing it gives the set skills (active ones under the item's equip
// delay), the shield skills with the set's shield and the +6 skills; the
// set's shield alone gives the shield skills.
func (s *Server) armorSetOn(p *Player, slot storage.PaperdollSlot, it *inventory.Item) {
	set, ok := s.chestSet(p)
	if !ok {
		return
	}
	var update, stamps bool
	id := int(it.ItemID())
	switch {
	case armorSetContainsItem(set, slot, id):
		if !armorSetComplete(p, set) {
			break
		}
		update, stamps = s.addSetSkills(p, set.Skills, it.Template().EquipReuseDelay, true)
		if armorSetShield(p, set) {
			if u, _ := s.addSetSkills(p, set.ShieldSkills, 0, false); u {
				update = true
			}
		}
		if armorSetEnchanted6(p, set) {
			if u, _ := s.addSetSkills(p, set.Enchant6Skills, 0, false); u {
				update = true
			}
		}
	case set.ContainsShield(id):
		update, _ = s.addSetSkills(p, set.ShieldSkills, 0, false)
	}
	s.skillsChanged(p, update, stamps, false)
}

// armorSetOff is ArmorSetListener.notifyUnequiped: taking off the chest or
// a part of the chest's set removes the set, shield and +6 skills; taking
// off the set's shield removes the shield skills.
func (s *Server) armorSetOff(p *Player, slot storage.PaperdollSlot, it *inventory.Item) {
	d := s.SkillData()
	if d == nil {
		return
	}
	id := int(it.ItemID())
	var set *data.ArmorSet
	var lists [][]data.SkillRef
	if slot == storage.PaperdollChest {
		var ok bool
		if set, ok = d.ArmorSet(id); !ok {
			return
		}
		lists = [][]data.SkillRef{set.Skills, set.ShieldSkills, set.Enchant6Skills}
	} else {
		var ok bool
		if set, ok = s.chestSet(p); !ok {
			return
		}
		switch {
		case armorSetContainsItem(set, slot, id):
			lists = [][]data.SkillRef{set.Skills, set.ShieldSkills, set.Enchant6Skills}
		case set.ContainsShield(id):
			lists = [][]data.SkillRef{set.ShieldSkills}
		default:
			return
		}
	}
	for _, refs := range lists {
		for _, ref := range refs {
			if sk, ok := s.Skill(ref.ID, ref.Level); ok {
				s.RemoveSkill(p, sk.ID, false, sk.IsPassive())
			}
		}
	}
	p.SendSkillList()
}

// loadArmorSet gives a loading player the skills of a complete armor set.
func (s *Server) loadArmorSet(p *Player) {
	set, ok := s.chestSet(p)
	if !ok || !armorSetComplete(p, set) {
		return
	}
	chest := p.inv.Paperdoll(storage.PaperdollChest)
	s.addSetSkills(p, set.Skills, chest.Template().EquipReuseDelay, true)
	if armorSetShield(p, set) {
		s.addSetSkills(p, set.ShieldSkills, 0, false)
	}
	if armorSetEnchanted6(p, set) {
		s.addSetSkills(p, set.Enchant6Skills, 0, false)
	}
}

// refreshArmorSet is FuncArmorSet: while the chest's set is complete its
// base stat bonus (STR, CON, ...) is on, as modifiers at order 1; quiet
// leaves the new numbers for the caller to show.
func (s *Server) refreshArmorSet(p *Player, quiet bool) {
	set, ok := s.chestSet(p)
	if !ok || !armorSetComplete(p, set) {
		set = nil
	}
	if set == p.itemSkill.setBonus {
		return
	}
	removed := p.stats.RemoveOwner(armorSetOwner) > 0
	p.itemSkill.setBonus = set
	var mods []stats.Modifier
	if set != nil {
		for _, b := range []struct {
			st stats.Stat
			v  int
		}{
			{stats.STR, set.STR}, {stats.DEX, set.DEX}, {stats.INT, set.INT},
			{stats.MEN, set.MEN}, {stats.CON, set.CON}, {stats.WIT, set.WIT},
		} {
			if b.v != 0 {
				mods = append(mods, stats.Modifier{Stat: b.st, Op: stats.OpAdd, Order: 1, Value: float64(b.v), Owner: armorSetOwner})
			}
		}
	}
	if len(mods) > 0 {
		if err := p.stats.AddModifiers(mods...); err != nil {
			s.log.Warn("game: armor set bonus refused", append(p.logArgs(), "err", err)...)
		}
	}
	if !quiet && (removed || len(mods) > 0) {
		p.RefreshStats()
	}
}
