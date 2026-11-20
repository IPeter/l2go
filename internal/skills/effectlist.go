package skills

import (
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/data"
	sp "github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/stats"
)

// DivineInspiration is the skill whose level raises the buff limit (Mobius
// CreatureStat.DIVINE_INSPIRATION).
const DivineInspiration = 1405

// iconDelay is how long the icon update waits for more changes
// (EffectList.updateEffectIcons).
const iconDelay = 300 * time.Millisecond

// clockPeriod is the period of the buff clock (BuffFinishTask).
const clockPeriod = time.Second

// IconUpdate is one coalesced update of a creature's effect icons, built
// 300 ms after the first change of a burst (EffectList.updateEffectIcons).
// The icons are the effects in use, in list order: buffs (without healing
// potions, which use the short buff), dances, toggles, debuffs; the seconds
// are BuffInfo.Time (-1 for passives and toggles).
type IconUpdate struct {
	// SendAbnormal asks for AbnormalStatusUpdate with Icons; it is false
	// when only the party icons changed (UpdateIcons(true)).
	SendAbnormal bool
	// Icons are the AbnormalStatusUpdate entries; the game also sends them
	// as ExOlympiadSpelledInfo to Olympiad spectators.
	Icons []sp.AbnormalStatusEntry
	// Party are the PartySpelled entries: Icons without toggles for a
	// player, all of them for a summon.
	Party []sp.AbnormalStatusEntry
	// Summon is set when the icons are those of a summon: its owner sends
	// PartySpelled of the summon to itself, and nothing else.
	Summon Creature
}

// ShortBuff is the short buff icon of a player (ShortBuffStatusUpdate): the
// effect of a healing potion. The zero value clears it.
type ShortBuff = sp.ShortBuffStatusUpdate

// EffectList holds the BuffInfos acting on one creature, Mobius EffectList,
// rule for rule (Plan 10 K5, K6): five lists by kind (passives, debuffs,
// dances, toggles, buffs), stacking by abnormal type and level, herbs that
// hide a weaker buff, the buff and dance limits, the effect flags, the
// abnormal visual masks, the coalesced icon updates and the buff clock that
// ends timed effects.
//
// It calls back into the game only through its owner (Creature, Player)
// and the Env's Scheduler. Use it on the world loop only.
type EffectList struct {
	env   *Env
	owner Creature

	buffs, dances, toggles, debuffs, passives []*BuffInfo

	// stacked holds the effect in use of each abnormal type (NONE never).
	stacked map[data.AbnormalType]*BuffInfo
	blocked map[data.AbnormalType]bool

	shortBuff *BuffInfo
	hidden    int

	// The "has effects removed on ..." flags are only ever set, as in
	// Mobius (updateEffectFlags).
	removedOnAction, buffsRemovedOnDamage, debuffsRemovedOnDamage bool

	flags     EffectFlags
	visual    VisualMasks
	partyOnly bool
	iconTimer TimerID

	// nextOwner numbers the BuffInfos, which own their effects' modifiers.
	nextOwner int64

	// The buff clock (BuffFinishTask): the timed BuffInfos in the order
	// they joined it, the timer, and whether the clock was stopped.
	clocked      []*BuffInfo
	clock        TimerID
	clockStopped bool
}

// NewEffectList returns the empty effect list of owner.
func NewEffectList(env *Env, owner Creature) *EffectList {
	return &EffectList{
		env:     env,
		owner:   owner,
		stacked: make(map[data.AbnormalType]*BuffInfo),
		blocked: make(map[data.AbnormalType]bool),
	}
}

// Owner returns the creature the list belongs to.
func (l *EffectList) Owner() Creature { return l.owner }

// Buffs, Dances, Toggles, Debuffs and Passives return the lists in their
// order (oldest first). The slices must not be changed.
func (l *EffectList) Buffs() []*BuffInfo { return l.buffs }

// Dances returns the dances and songs.
func (l *EffectList) Dances() []*BuffInfo { return l.dances }

// Toggles returns the active toggles.
func (l *EffectList) Toggles() []*BuffInfo { return l.toggles }

// Debuffs returns the debuffs.
func (l *EffectList) Debuffs() []*BuffInfo { return l.debuffs }

// Passives returns the passive skills' effects.
func (l *EffectList) Passives() []*BuffInfo { return l.passives }

// Effects returns the buffs, dances, toggles and debuffs in that order, as
// a new slice (EffectList.getEffects). Passives are not included.
func (l *EffectList) Effects() []*BuffInfo {
	all := make([]*BuffInfo, 0, len(l.buffs)+len(l.dances)+len(l.toggles)+len(l.debuffs))
	all = append(all, l.buffs...)
	all = append(all, l.dances...)
	all = append(all, l.toggles...)
	return append(all, l.debuffs...)
}

// IsEmpty reports a list without buffs, dances, toggles and debuffs.
func (l *EffectList) IsEmpty() bool {
	return len(l.buffs) == 0 && len(l.dances) == 0 && len(l.toggles) == 0 && len(l.debuffs) == 0
}

// listOf returns the list a skill's effects go to (getEffectList).
func (l *EffectList) listOf(sk *Skill) *[]*BuffInfo {
	switch {
	case sk.IsPassive():
		return &l.passives
	case sk.IsDebuff:
		return &l.debuffs
	case sk.IsDance():
		return &l.dances
	case sk.IsToggle():
		return &l.toggles
	}
	return &l.buffs
}

// BuffCount is getBuffCount: the buffs, less the hidden ones and the short
// buff.
func (l *EffectList) BuffCount() int {
	if len(l.buffs) == 0 {
		return 0
	}
	n := len(l.buffs) - l.hidden
	if l.shortBuff != nil {
		n--
	}
	return n
}

// DanceCount is the number of dances and songs.
func (l *EffectList) DanceCount() int { return len(l.dances) }

// HiddenCount is the number of buffs a herb hides.
func (l *EffectList) HiddenCount() int { return l.hidden }

// ShortBuff returns the BuffInfo shown as the short buff, nil if none.
func (l *EffectList) ShortBuff() *BuffInfo { return l.shortBuff }

// Flags returns the effect flags of the effects (computeEffectFlags).
func (l *EffectList) Flags() EffectFlags { return l.flags }

// IsAffected reports whether an effect sets the flag (isAffected).
func (l *EffectList) IsAffected(f EffectFlag) bool { return l.flags.Has(f) }

// Visual returns the abnormal visual masks the effects show.
func (l *EffectList) Visual() VisualMasks { return l.visual }

// BySkillID returns the BuffInfo of a skill, looking in buffs, dances,
// toggles, debuffs and passives in that order (getBuffInfoBySkillId).
func (l *EffectList) BySkillID(id int) *BuffInfo {
	for _, list := range [...][]*BuffInfo{l.buffs, l.dances, l.toggles, l.debuffs, l.passives} {
		for _, b := range list {
			if b.skill.ID == id {
				return b
			}
		}
	}
	return nil
}

// IsAffectedBySkill reports whether a skill's effects are on the creature.
func (l *EffectList) IsAffectedBySkill(id int) bool { return l.BySkillID(id) != nil }

// ByAbnormal returns the BuffInfo in use of an abnormal type
// (getBuffInfoByAbnormalType).
func (l *EffectList) ByAbnormal(t data.AbnormalType) *BuffInfo { return l.stacked[t] }

// IsAffectedByAbnormal reports whether an effect of the abnormal type is in
// use.
func (l *EffectList) IsAffectedByAbnormal(t data.AbnormalType) bool {
	_, ok := l.stacked[t]
	return ok
}

// FirstEffect returns the first BuffInfo with an effect of the type, in
// buffs, dances, toggles and debuffs (getFirstEffect).
func (l *EffectList) FirstEffect(t EffectType) *BuffInfo {
	for _, list := range [...][]*BuffInfo{l.buffs, l.dances, l.toggles, l.debuffs} {
		for _, b := range list {
			for _, e := range b.effects {
				if e.Type() == t {
					return b
				}
			}
		}
	}
	return nil
}

// BlockAbnormals adds abnormal types that the list refuses
// (addBlockedAbnormalTypes).
func (l *EffectList) BlockAbnormals(types ...data.AbnormalType) {
	for _, t := range types {
		l.blocked[t] = true
	}
}

// UnblockAbnormals removes blocked abnormal types; it reports whether one
// was blocked.
func (l *EffectList) UnblockAbnormals(types ...data.AbnormalType) bool {
	changed := false
	for _, t := range types {
		if l.blocked[t] {
			delete(l.blocked, t)
			changed = true
		}
	}
	return changed
}

// doesStack reports whether a list of the skill's kind has an effect of its
// abnormal type (doesStack).
func (l *EffectList) doesStack(sk *Skill) bool {
	if sk.AbnormalType == data.AbnormalNone || l.IsEmpty() {
		return false
	}
	for _, b := range *l.listOf(sk) {
		if b.skill.AbnormalType == sk.AbnormalType {
			return true
		}
	}
	return false
}

// MaxBuffCount is the buff limit of the owner: MaxBuffAmount plus its
// Divine Inspiration level (CreatureStat.getMaxBuffCount).
func (l *EffectList) MaxBuffCount() int {
	return l.env.Config.MaxBuffAmount + l.owner.SkillLevel(DivineInspiration)
}

// Add puts a BuffInfo in the list and starts its effects (EffectList.add):
//
//   - A blocked abnormal type is refused.
//   - A passive needs its preconditions; it replaces the passive of the same
//     skill ID and starts even on a dead creature.
//   - Nothing else lands on a dead creature.
//   - Abnormal type NONE: an effect of the same skill ends (worn off) and the
//     new one goes to the end of the list.
//   - Another abnormal type: a level at least the stacked one's replaces it
//     (a herb hides it instead, stats off and time running), a lower level is
//     dropped silently.
//   - A new buff or dance that does not stack ends the oldest buff or dance
//     (the oldest dance, for a dance over the dance limit) when the list is
//     full. Healing potions do not count, nor debuffs, toggles and the Seven
//     Signs skills.
func (l *EffectList) Add(info *BuffInfo) {
	if info == nil || info.skill == nil {
		return
	}
	sk := info.skill
	if l.blocked[sk.AbnormalType] {
		return
	}
	info.list = l
	if sk.IsPassive() {
		if sk.AbnormalType != data.AbnormalNone {
			l.env.log().Warn("skills: passive skill with an abnormal type", "skill", sk.String(), "abnormal", sk.AbnormalType.String())
		}
		if !sk.CheckCondition(l.env, info.effector, info.effected, false) {
			return
		}
		for _, b := range slices.Clone(l.passives) {
			if b.skill.ID == sk.ID {
				b.inUse = false
				b.removeStats()
				l.passives = removeInfo(l.passives, b)
			}
		}
		info.owner = passiveOwner(sk.ID)
		l.passives = append(l.passives, info)
		info.initializeEffects()
		return
	}
	if info.effected.IsDead() {
		return
	}
	if sk.AbnormalType == data.AbnormalNone {
		l.StopSkill(FinishNormal, sk.ID)
	} else {
		if stacked, ok := l.stacked[sk.AbnormalType]; ok {
			if stacked == nil || sk.AbnormalLevel < stacked.skill.AbnormalLevel {
				return // a lesser buff is not added
			}
			if sk.AbnormalInstant {
				if stacked.skill.AbnormalInstant {
					l.StopAbnormal(FinishNormal, sk.AbnormalType)
					stacked = l.stacked[sk.AbnormalType]
				}
				if stacked != nil {
					stacked.inUse = false
					stacked.removeStats()
					l.hidden++
				}
			} else {
				if stacked.skill.AbnormalInstant {
					l.StopAbnormal(FinishNormal, sk.AbnormalType)
				}
				l.StopAbnormal(FinishNormal, sk.AbnormalType)
			}
		}
		l.stacked[sk.AbnormalType] = info
	}

	list := l.listOf(sk)
	if !sk.IsDebuff && !sk.IsToggle() && !sk.Is7Signs() && !l.doesStack(sk) {
		l.evictForLimit(sk)
	}
	*list = append(*list, info)
	l.nextOwner++
	info.owner = stats.Owner{Kind: stats.OwnerEffect, ID: l.nextOwner}
	info.initializeEffects()
	l.updateEffectList(true)
}

// evictForLimit ends the oldest buff or dance when a new one would pass
// the buff or dance limit.
func (l *EffectList) evictForLimit(sk *Skill) {
	total := l.BuffCount() + l.DanceCount()
	maxDances := l.env.Config.MaxDanceAmount
	dances := l.DanceCount()
	buffsFull := total >= l.MaxBuffCount() && !sk.IsHealingPotion()
	dancesFull := sk.IsDance() && dances >= maxDances
	if !buffsFull && !dancesFull {
		return
	}
	var oldest *BuffInfo
	for _, list := range [...][]*BuffInfo{l.buffs, l.dances} {
		for _, b := range list {
			if !b.inUse {
				continue
			}
			if sk.IsDance() && dances >= maxDances && !b.skill.IsDance() {
				continue
			}
			if oldest == nil || b.periodStart < oldest.periodStart {
				oldest = b
			}
		}
	}
	if oldest != nil {
		l.stopAndRemove(true, FinishRemoved, oldest, l.listOf(oldest.skill))
	}
}

// Remove ends a BuffInfo and takes it out of the list (EffectList.remove).
// The finish type picks the message and whether END effects run.
func (l *EffectList) Remove(ft FinishType, info *BuffInfo) {
	if info == nil || info.skill == nil {
		return
	}
	l.stopAndRemove(true, ft, info, l.listOf(info.skill))
	l.updateEffectList(true)
}

// StopSkill ends the effects of a skill (stopSkillEffects by skill ID).
func (l *EffectList) StopSkill(ft FinishType, id int) {
	if info := l.BySkillID(id); info != nil {
		l.Remove(ft, info)
	}
}

// StopAbnormal ends the effect in use of an abnormal type; it reports
// whether there was one (stopSkillEffects by abnormal type).
func (l *EffectList) StopAbnormal(ft FinishType, t data.AbnormalType) bool {
	old, ok := l.stacked[t]
	if !ok {
		return false
	}
	delete(l.stacked, t)
	l.Remove(ft, old)
	return true
}

// stopAndRemove takes info out of list and ends it (stopAndRemove). A
// BuffInfo no longer in the list was already stopped and is left alone. A
// herb that ends gives the hidden buff of its abnormal type back. END
// effects run unless the info was removed.
func (l *EffectList) stopAndRemove(broadcast bool, ft FinishType, info *BuffInfo, list *[]*BuffInfo) {
	if info == nil || !slices.Contains(*list, info) {
		return
	}
	*list = removeInfo(*list, info)
	info.stopAllEffects(ft, broadcast)
	sk := info.skill
	if !info.inUse {
		l.hidden--
	} else if l.stacked[sk.AbnormalType] == info {
		delete(l.stacked, sk.AbnormalType)
	}
	if sk.AbnormalInstant {
		for _, b := range l.buffs {
			if b.skill.AbnormalType == sk.AbnormalType && !b.inUse {
				b.inUse = true
				b.addStats()
				l.stacked[b.skill.AbnormalType] = b
				l.hidden--
				break
			}
		}
	}
	if ft != FinishRemoved {
		sk.ApplyEffectScope(data.ScopeEnd, info, true, false)
	}
}

// removeInfo returns list without info, keeping the order.
func removeInfo(list []*BuffInfo, info *BuffInfo) []*BuffInfo {
	if i := slices.Index(list, info); i >= 0 {
		return slices.Delete(list, i, i+1)
	}
	return list
}

// stopList ends every BuffInfo of a list for which keep is false, with the
// removed finish type.
func (l *EffectList) stopList(broadcast bool, list *[]*BuffInfo, keep func(*BuffInfo) bool) {
	for _, b := range slices.Clone(*list) {
		if keep == nil || !keep(b) {
			l.stopAndRemove(broadcast, FinishRemoved, b, list)
		}
	}
}

// StopAll ends every buff, dance, toggle and debuff, not the passives
// (stopAllEffects).
func (l *EffectList) StopAll() {
	l.StopAllBuffs(false)
	l.StopAllDances(false)
	l.StopAllToggles(false)
	l.StopAllDebuffs(false)
	clear(l.stacked)
	l.updateEffectList(true)
	l.owner.UpdateAbnormalEffect()
}

// StopAllExceptStayAfterDeath ends the effects that do not outlast death
// (stopAllEffectsExceptThoseThatLastThroughDeath).
func (l *EffectList) StopAllExceptStayAfterDeath() {
	stay := func(b *BuffInfo) bool { return b.skill.StayAfterDeath }
	l.stopKinds(stay)
}

// StopAllNotStayOnSubclassChange ends the effects that do not survive a
// subclass change.
func (l *EffectList) StopAllNotStayOnSubclassChange() {
	stay := func(b *BuffInfo) bool { return b.skill.StayOnSubclassChange }
	l.stopKinds(stay)
}

// stopKinds ends the buffs, debuffs, dances and toggles keep refuses, in
// Mobius's order, and updates if any list had effects.
func (l *EffectList) stopKinds(keep func(*BuffInfo) bool) {
	update := false
	for _, list := range [...]*[]*BuffInfo{&l.buffs, &l.debuffs, &l.dances, &l.toggles} {
		if len(*list) > 0 {
			l.stopList(true, list, keep)
			update = true
		}
	}
	l.updateEffectList(update)
}

// StopAllWithoutExclusions ends everything, passives included
// (stopAllEffectsWithoutExclusions): when a creature leaves the world.
func (l *EffectList) StopAllWithoutExclusions(update, broadcast bool) {
	for _, list := range [...]*[]*BuffInfo{&l.buffs, &l.dances, &l.toggles, &l.debuffs, &l.passives} {
		l.stopList(broadcast, list, nil)
	}
	if update {
		l.updateEffectList(broadcast)
	}
}

// StopAllBuffs ends every buff.
func (l *EffectList) StopAllBuffs(update bool) {
	l.stopList(update, &l.buffs, nil)
	l.updateEffectList(update)
}

// StopAllToggles ends every toggle.
func (l *EffectList) StopAllToggles(update bool) {
	if len(l.toggles) == 0 {
		return
	}
	l.stopList(update, &l.toggles, nil)
	l.updateEffectList(update)
}

// StopAllDances ends every dance and song.
func (l *EffectList) StopAllDances(update bool) {
	if len(l.dances) == 0 {
		return
	}
	l.stopList(update, &l.dances, nil)
	l.updateEffectList(update)
}

// StopAllDebuffs ends every debuff.
func (l *EffectList) StopAllDebuffs(update bool) {
	if len(l.debuffs) == 0 {
		return
	}
	l.stopList(update, &l.debuffs, nil)
	l.updateEffectList(update)
}

// StopEffectType ends the BuffInfos with an effect of the type
// (stopEffects).
func (l *EffectList) StopEffectType(t EffectType) {
	update := false
	for _, list := range [...]*[]*BuffInfo{&l.buffs, &l.dances, &l.toggles, &l.debuffs} {
		if len(*list) == 0 {
			continue
		}
		for _, b := range slices.Clone(*list) {
			for _, e := range b.effects {
				if e.Type() == t {
					l.stopAndRemove(true, FinishRemoved, b, l.listOf(b.skill))
				}
			}
		}
		update = true
	}
	l.updateEffectList(update)
}

// StopOnAction ends the effects removed by any action but moving, on an
// attack or a cast (stopEffectsOnAction).
func (l *EffectList) StopOnAction() {
	if !l.removedOnAction {
		return
	}
	keep := func(b *BuffInfo) bool { return !b.skill.RemovedOnAnyActionExceptMove }
	l.stopKinds(keep)
}

// StopOnDamage ends the effects removed by damage (stopEffectsOnDamage);
// awake is false for damage that does not wake.
func (l *EffectList) StopOnDamage(awake bool) {
	if !awake {
		return
	}
	update := false
	keep := func(b *BuffInfo) bool { return !b.skill.RemovedOnDamage }
	if l.buffsRemovedOnDamage {
		for _, list := range [...]*[]*BuffInfo{&l.buffs, &l.dances, &l.toggles} {
			if len(*list) > 0 {
				l.stopList(true, list, keep)
				update = true
			}
		}
	}
	if l.debuffsRemovedOnDamage && len(l.debuffs) > 0 {
		l.stopList(true, &l.debuffs, keep)
		update = true
	}
	l.updateEffectList(update)
}

// UpdateIcons schedules an icon update (updateEffectIcons); partyOnly
// leaves out AbnormalStatusUpdate the next time.
func (l *EffectList) UpdateIcons(partyOnly bool) {
	if partyOnly {
		l.partyOnly = true
	}
	l.updateEffectList(true)
}

// updateEffectList updates the icons and the effect flags (updateEffectList).
func (l *EffectList) updateEffectList(update bool) {
	if update {
		l.updateEffectIcons()
		l.computeEffectFlags()
	}
}

// updateEffectIcons schedules the icon packets of a playable owner, 300 ms
// after the first change of a burst.
func (l *EffectList) updateEffectIcons() {
	if !isPlayable(l.owner) {
		return
	}
	l.updateRemovalFlags()
	if l.iconTimer == 0 {
		l.iconTimer = l.env.Sched.After(iconDelay, l.sendIcons)
	}
}

// sendIcons sends the coalesced icon update.
func (l *EffectList) sendIcons() {
	l.iconTimer = 0
	p := l.owner.AsPlayer()
	if p == nil {
		return
	}
	var u IconUpdate
	player := isPlayer(l.owner)
	if player {
		if l.partyOnly {
			l.partyOnly = false
		} else {
			u.SendAbnormal = true
		}
	} else if isSummon(l.owner) {
		u.Summon = l.owner
	} else {
		return
	}
	add := func(b *BuffInfo) {
		if !b.inUse {
			return
		}
		e := iconOf(b)
		if !b.skill.IsHealingPotion() {
			u.Icons = append(u.Icons, e)
		}
		if !player || !b.skill.IsToggle() {
			u.Party = append(u.Party, e)
		}
	}
	for _, b := range l.buffs {
		if b.skill.IsHealingPotion() {
			l.shortBuffUpdate(b)
		} else {
			add(b)
		}
	}
	for _, list := range [...][]*BuffInfo{l.dances, l.toggles, l.debuffs} {
		for _, b := range list {
			add(b)
		}
	}
	if !u.SendAbnormal {
		u.Icons = nil
	}
	p.SendEffectIcons(u)
}

// iconOf is the icon entry of a BuffInfo.
func iconOf(b *BuffInfo) sp.AbnormalStatusEntry {
	return sp.AbnormalStatusEntry{
		SkillID:    int32(b.skill.DisplayID),    //nolint:gosec // skill IDs are small
		SkillLevel: int16(b.skill.DisplayLevel), //nolint:gosec // skill levels are small
		Seconds:    int32(b.Time()),             //nolint:gosec // seconds of an effect
	}
}

// shortBuffUpdate sets the short buff of a player and sends it
// (shortBuffStatusUpdate); nil resets it.
func (l *EffectList) shortBuffUpdate(info *BuffInfo) {
	if !isPlayer(l.owner) {
		return
	}
	l.shortBuff = info
	sb := ShortBuff{}
	if info != nil {
		sb = ShortBuff{
			SkillID:    int32(info.skill.ID),    //nolint:gosec // skill IDs are small
			SkillLevel: int32(info.skill.Level), //nolint:gosec // skill levels are small
			Seconds:    int32(info.Time()),      //nolint:gosec // seconds of an effect
		}
	}
	l.owner.AsPlayer().SendShortBuff(sb)
}

// updateRemovalFlags notes effects removed on action or damage
// (updateEffectFlags). The flags are never cleared.
func (l *EffectList) updateRemovalFlags() {
	for _, list := range [...][]*BuffInfo{l.buffs, l.toggles} {
		for _, b := range list {
			if b.skill.RemovedOnAnyActionExceptMove {
				l.removedOnAction = true
			}
			if b.skill.RemovedOnDamage {
				l.buffsRemovedOnDamage = true
			}
		}
	}
	for _, b := range l.debuffs {
		if b.skill.RemovedOnDamage {
			l.debuffsRemovedOnDamage = true
		}
	}
}

// computeEffectFlags recomputes the effect flags of the buffs, debuffs,
// dances and toggles.
func (l *EffectList) computeEffectFlags() {
	var flags EffectFlags
	for _, list := range [...][]*BuffInfo{l.buffs, l.debuffs, l.dances, l.toggles} {
		for _, b := range list {
			for _, e := range b.effects {
				flags |= e.Flags()
			}
		}
	}
	l.flags = flags
}

// ToStore returns the effects a player saves (Player.storeEffect): no
// LIFE_FORCE_OTHERS heals, toggles and dances as configured
// (StoreToggles, StoreDances), each skill level once.
func (l *EffectList) ToStore() []*BuffInfo {
	var out []*BuffInfo
	seen := make(map[int]bool)
	for _, b := range l.Effects() {
		sk := b.skill
		switch {
		case sk.AbnormalType == data.AbnormalLifeForceOthers,
			sk.IsToggle() && !l.env.Config.StoreToggles,
			sk.IsDance() && !l.env.Config.StoreDances,
			seen[sk.ReuseHash()]:
			continue
		}
		seen[sk.ReuseHash()] = true
		out = append(out, b)
	}
	return out
}

// clockAdd puts a timed BuffInfo on the buff clock (addBuffInfoTime). The
// clock starts with a run at once, as BuffFinishTask's first run has no
// delay.
func (l *EffectList) clockAdd(b *BuffInfo) {
	if !b.onClock {
		l.clocked = append(l.clocked, b)
	}
	b.onClock, b.clockCount = true, 0
	if l.clock == 0 && !l.clockStopped {
		l.startClock()
	}
}

// clockRemove takes a BuffInfo off the buff clock (removeBuffInfoTime); the
// clock stops when nothing is left on it.
func (l *EffectList) clockRemove(b *BuffInfo) {
	if !b.onClock {
		return
	}
	b.onClock = false
	l.clocked = removeInfo(l.clocked, b)
	if len(l.clocked) == 0 && l.clock != 0 {
		l.env.Sched.Cancel(l.clock)
		l.clock = 0
	}
}

// startClock runs the clock once and then every second.
func (l *EffectList) startClock() {
	l.clock = l.env.Sched.Every(clockPeriod, l.clockTick)
	l.clockTick()
}

// clockTick counts a second for every timed BuffInfo and ends those whose
// count passes their abnormal time (BuffFinishRunnable).
func (l *EffectList) clockTick() {
	var ended []*BuffInfo
	for _, b := range l.clocked {
		b.clockCount++
		if b.clockCount > b.abnormalTime {
			ended = append(ended, b)
		}
	}
	for _, b := range ended {
		l.clockRemove(b)
		if b.effected != nil {
			l.Remove(FinishNormal, b)
		}
	}
}

// StopClock stops the buff clock (BuffFinishTask.stop): timed effects stop
// counting, as while a player teleports or leaves.
func (l *EffectList) StopClock() {
	l.clockStopped = true
	if l.clock != 0 {
		l.env.Sched.Cancel(l.clock)
		l.clock = 0
	}
}

// StartClock starts the buff clock again (BuffFinishTask.start).
func (l *EffectList) StartClock() {
	l.clockStopped = false
	if len(l.clocked) > 0 && l.clock == 0 {
		l.startClock()
	}
}

// CancelTimers cancels the icon timer and the buff clock, when the creature
// is removed from the world.
func (l *EffectList) CancelTimers() {
	if l.iconTimer != 0 {
		l.env.Sched.Cancel(l.iconTimer)
		l.iconTimer = 0
	}
	l.StopClock()
}
