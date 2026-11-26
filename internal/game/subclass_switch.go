package game

import (
	"cmp"
	"slices"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/subclass"
	"github.com/IPeter/l2go/internal/summon"
)

// Adding, replacing and switching subclasses (Mobius Player.addSubClass,
// modifySubClass and setActiveClass). All of it is loop-only.

// addSubclass is Mobius Player.addSubClass: a subclass of classID at index
// (1-3) that is free, while p has fewer than MaxSubclass, starting at the
// base subclass level with no SP; its class's skills up to level 40 are
// stored for the new index (the highest level of each skill the tree gives
// by then). It does not switch to it. It reports whether it was added.
func (s *Server) addSubclass(p *Player, classID int, index int32) bool {
	cfg := s.subclass.cfg
	if len(p.char.Subclasses) >= cfg.MaxSubclass || index < 1 || index > subclass.MaxIndex || p.subclassOf(index) != nil {
		return false
	}
	if _, ok := s.data.ClassTemplate(classID); !ok || s.classes[classID] == nil {
		s.log.Warn("game: a subclass of an unknown class", append(p.logArgs(), "class", classID)...)
		return false
	}
	sub := storage.Subclass{
		ClassIndex: index,
		ClassID:    int32(classID), //nolint:gosec // class IDs are small
		Level:      int32(cfg.BaseLevel),
		Exp:        s.ExpForLevel(cfg.BaseLevel),
	}
	subs := append(slices.Clone(p.char.Subclasses), sub)
	slices.SortFunc(subs, func(a, b storage.Subclass) int { return cmp.Compare(a.ClassIndex, b.ClassIndex) })
	p.char.Subclasses = subs

	// The rows a class of this index may have left are not the new class's.
	s.dropClassRows(p, index)
	if d := s.skills.data; d != nil {
		tree, _ := d.SkillTree(classID)
		levels := make(map[int]int)
		var order []int
		for _, l := range tree {
			if l.GetLevel > subclassSkillLevel {
				continue
			}
			if _, ok := s.Skill(l.SkillID, l.Level); !ok {
				continue
			}
			prev, ok := levels[l.SkillID]
			if ok && prev > l.Level {
				continue
			}
			if !ok {
				order = append(order, l.SkillID)
			}
			levels[l.SkillID] = l.Level
		}
		for _, id := range order {
			p.char.Skills = append(p.char.Skills, storage.CharacterSkill{ClassIndex: index, ID: int32(id), Level: int32(levels[id])}) //nolint:gosec // small
		}
	}
	s.RequestSave(p)
	return true
}

// subclassSkillLevel is the level up to which a new subclass gets its
// class's skills (Mobius addSubClass: getGetLevel() <= 40).
const subclassSkillLevel = 40

// dropClassRows forgets what Mobius modifySubClass deletes for a class
// index: its dyes, shortcuts, saved effects and skills. (Mobius keeps the
// dwarven recipes of the index, and so does the crafting feature.)
func (s *Server) dropClassRows(p *Player, index int32) {
	p.char.Skills = slices.DeleteFunc(slices.Clone(p.char.Skills), func(r storage.CharacterSkill) bool { return r.ClassIndex == index })
	p.char.SavedSkills = slices.DeleteFunc(slices.Clone(p.char.SavedSkills), func(r storage.SavedSkill) bool { return r.ClassIndex == index })
	p.char.Shortcuts = slices.DeleteFunc(slices.Clone(p.char.Shortcuts), func(r storage.Shortcut) bool { return r.ClassIndex == index })
	p.itemSkill.bar.other = slices.DeleteFunc(p.itemSkill.bar.other, func(r storage.Shortcut) bool { return r.ClassIndex == index })
	if _, ok := p.char.Hennas[index]; ok {
		hennas := storage.CloneHennas(p.char.Hennas)
		delete(hennas, index)
		p.char.Hennas = storage.CloneHennas(hennas)
	}
}

// modifySubclass is Mobius Player.modifySubClass: the subclass of index is
// erased with its dyes, shortcuts, saved effects and skills, and a subclass
// of newClassID takes its index (addSubclass). The caller switches to it.
// It reports whether the subclass existed and was replaced.
func (s *Server) modifySubclass(p *Player, index int32, newClassID int) bool {
	if p.subclassOf(index) == nil {
		return false
	}
	if p.classIndex() != index {
		// Mobius stores the other classes as they are; the active class is
		// written by the switch that follows.
		s.storeActiveClass(p)
	}
	p.char.Subclasses = slices.DeleteFunc(slices.Clone(p.char.Subclasses), func(sub storage.Subclass) bool { return sub.ClassIndex == index })
	s.dropClassRows(p, index)
	return s.addSubclass(p, newClassID, index)
}

// storeActiveClass is the store() step of Mobius setActiveClass: the live
// skills, shortcuts and progress of the active class go into p's stored
// rows (where the inactive classes' rows are), and its saved effects: the
// skill reuse only, unless SubclassStoreSkillCooltime keeps the effects
// too.
func (s *Server) storeActiveClass(p *Player) {
	c := p.char
	c.Level = int32(p.Level()) //nolint:gosec // levels are small
	p.storeSkills(&c)
	if p.itemSkill.loaded {
		p.storeShortcuts(&c)
		p.char.Shortcuts = c.Shortcuts
	}
	p.char.Skills = c.Skills
	if !s.subclass.cfg.StoreSkillCooltime {
		index := p.classIndex()
		var rows []storage.SavedSkill
		for _, r := range p.char.SavedSkills {
			if r.ClassIndex != index {
				rows = append(rows, r)
			}
		}
		c.SavedSkills = append(rows, s.reuseToSave(p)...)
	}
	p.char.SavedSkills = c.SavedSkills
	p.storeSubclass(&c)
	p.char.Subclasses = c.Subclasses
	p.char.Level = c.Level
}

// reuseToSave is Mobius storeEffect(false): every reuse stamp that has not
// passed, as a reuse row of the active class.
func (s *Server) reuseToSave(p *Player) []storage.SavedSkill {
	var out []storage.SavedSkill
	for _, r := range p.skill.activeReuse(s.Now()) {
		out = append(out, storage.SavedSkill{
			ClassIndex:    p.classIndex(),
			SkillID:       int32(r.id),    //nolint:gosec // skill IDs are small
			SkillLevel:    int32(r.level), //nolint:gosec // skill levels are small
			RemainingTime: -1,
			ReuseDelay:    r.delay.Milliseconds(),
			ReuseEnd:      r.end.UnixMilli(),
			RestoreType:   storage.RestoreReuse,
		})
	}
	return out
}

// setActiveClass is Mobius Player.setActiveClass: p plays the class of
// index (0 for the base class). In Mobius's order: the augment bonuses come
// off, a cast stops, the class left is stored (unless store is false: its
// subclass was just replaced), the reuse stamps and charges go, a servitor
// is sent away, the class template, level, exp and SP change, every skill
// goes, the effects but those that stay through death and a class change
// end with the toggles and cubics, the dwarven recipes, the Death Penalty,
// the stored and auto-get skills, the noble, hero and clan skills and the
// worn items' skills come, then the saved effects of the class
// (EtcStatusUpdate), quest 422 ends, the dyes of the class come
// (HennaInfo), HP, MP and CP keep within their maximum, UserInfo, the
// shortcut bar of the class (ShortcutInit), SocialAction, SkillCoolTime
// and ExStorageMaxCount. It reports false, changing nothing, for an index p
// has no class of.
func (s *Server) setActiveClass(p *Player, index int32, store bool) bool {
	classID := p.char.BaseClassID
	if index != baseClassIndex {
		sub := p.subclassOf(index)
		if sub == nil {
			s.log.Warn("game: switching to a subclass the player does not have", append(p.logArgs(), "classIndex", index)...)
			return false
		}
		classID = sub.ClassID
	}
	class, ok := s.data.ClassTemplate(int(classID))
	tmpl := s.classes[int(classID)]
	if !ok || tmpl == nil {
		s.log.Error("game: no template for the class of a subclass switch", append(p.logArgs(), "class", classID)...)
		return false
	}
	f := &s.subclass
	s.RemoveAugmentBonuses(p) // augment.go: Mobius re-applies them only when an item is put on again
	s.AbortCast(p)
	out := p.classIndex()
	if store {
		s.storeActiveClass(p)
		if out == baseClassIndex {
			p.char.BaseLevel, p.char.BaseExp, p.char.BaseSP = p.char.Level, p.char.Exp, p.char.SP
		}
	}
	s.ClearSkillReuse(p)
	p.skill.charges = 0
	if sm := s.SummonOf(p); sm != nil && sm.Kind() == summon.Servitor {
		s.Unsummon(p)
	}

	// The class template and the progress of the new class.
	var level int32
	var exp int64
	var sp int32
	if index == baseClassIndex {
		level, exp, sp = p.char.BaseLevel, p.char.BaseExp, p.char.BaseSP
		p.char.BaseLevel, p.char.BaseExp, p.char.BaseSP = 0, 0, 0
	} else {
		sub := p.subclassOf(index)
		level = int32(f.cfg.ClampLevel(int(sub.Level), s.MaxLevel())) //nolint:gosec // levels are small
		exp, sp = sub.Exp, sub.SP
	}
	if err := p.stats.SetTemplate(tmpl); err != nil {
		s.log.Error("game: subclass switch failed", append(p.logArgs(), "class", classID, "err", err)...)
		return false
	}
	p.char.ClassIndex, p.char.ClassID = index, classID
	p.char.Level, p.char.Exp, p.char.SP = max(level, 1), exp, sp
	p.stats.SetLevel(int(p.char.Level))
	p.class = class
	s.partyClassChanged(p) // party_window.go: the party's level and window

	// Every skill goes; the effects end but those that stay.
	for _, sk := range p.Skills() {
		p.skill.removeSkill(sk.ID, true)
	}
	p.skill.stored = nil
	el := p.skill.el
	el.StopAllExceptStayAfterDeath()
	el.StopAllNotStayOnSubclassChange()
	el.StopAllToggles(false)
	s.StopCubics(p)
	// The dwarven recipe book follows the class index (craft.go).
	s.applyDeathPenalty(p)

	// The skills of the new class.
	s.restoreStoredSkills(p) // skill_persist.go
	s.RewardSkills(p)
	if p.IsNoble() {
		s.giveNobleSkills(p)
	}
	if p.IsHero() {
		s.setHero(p, true)
	}
	s.giveClanSkills(p) // clan_skills.go
	s.reapplyWornItemSkills(p)
	s.skillsEnter(p) // skill_persist.go: the class's saved reuse and effects
	p.Send(etcStatusPacket(p))

	if st := s.QuestState(p, subclass.QuestRepentYourSins); st != nil {
		st.ExitQuest(true, false)
	}
	s.SwapHennas(p, index) // henna.go: the class's dyes and HennaInfo

	v := p.stats.View()
	if p.CurHP > float64(v.MaxHP) {
		p.SetCurrentHP(float64(v.MaxHP))
	}
	if p.CurMP > float64(v.MaxMP) {
		p.SetCurrentMP(float64(v.MaxMP))
	}
	if p.CurCP > float64(v.MaxCP) {
		p.SetCurrentCP(float64(v.MaxCP))
	}
	s.refreshOverloaded(p)
	s.refreshExpertisePenalty(p)
	p.BroadcastUserInfo()
	p.char.ExpBeforeDeath = 0

	p.itemSkill.bar = shortcutBar{}
	s.loadShortcuts(p) // shortcut.go: the class's bar
	p.Send(shortcutInitPacket(p))
	if p.InWorld() {
		s.world.Broadcast(p, serverpackets.NewSocialActionPacket(serverpackets.SocialAction{
			ObjectID: p.ObjectID(),
			ActionID: serverpackets.SocialActionLevelUp,
		}), true)
	}
	p.Send(s.SkillCoolTimePacket(p))
	p.Send(s.storageMaxCount(p))
	s.RequestSave(p)
	return true
}

// reapplyWornItemSkills is Mobius getInventory().applyItemSkills() after a
// class switch took every skill: the worn items' skills and the armor set's
// come back.
func (s *Server) reapplyWornItemSkills(p *Player) {
	if p.inv.Inventory == nil {
		return
	}
	for slot := range storage.PaperdollSlot(storage.PaperdollTotalSlots) {
		if it := p.inv.Paperdoll(slot); it != nil && it.Slot() == slot {
			s.wornItemSkillsOn(p, it, true)
		}
	}
	s.loadArmorSet(p)
}

// subclassClassChanged follows a class change (ChangeClass, Mobius
// setPlayerClass): on a subclass its entry takes the new class; on the base
// class the base class follows, as the class change quests set it.
func (s *Server) subclassClassChanged(p *Player) {
	if !p.IsSubClassActive() {
		p.char.BaseClassID = p.char.ClassID
		return
	}
	// A save may still hold the old slice: change a copy.
	subs := slices.Clone(p.char.Subclasses)
	for i := range subs {
		if subs[i].ClassIndex == p.classIndex() {
			subs[i].ClassID = p.char.ClassID
		}
	}
	p.char.Subclasses = subs
}
