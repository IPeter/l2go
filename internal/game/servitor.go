package game

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/summon"
)

// Servitors (Mobius Servitor and the Summon effect): a summoning skill makes
// a servitor that lives an hour (or its skill's life time), eats its upkeep
// every few minutes, and comes back at login when its owner logged out with
// it (RestoreServitorOnReconnect).

// sumMsgSummonedMonsterUses is "A summoned monster uses $s1." (Mobius
// A_SUMMONED_MONSTER_USES_S1); sumMsgNotEnoughForServitor is "Since you do
// not have enough items to maintain the servitor's stay, the servitor will
// disappear.".
const (
	sumMsgSummonedMonsterUses  serverpackets.SystemMessageID = 1029
	sumMsgNotEnoughForServitor serverpackets.SystemMessageID = 1143
)

// summonServitor is Mobius Summon.onStart for a player owner without a
// summon: the servitor of p.NpcID comes out next to it, named after its
// template and titled with its owner's name, with its life and upkeep, the
// exp of its level, full HP and MP, the owner's heading, running.
func (s *Server) summonServitor(owner *Player, sk *skills.Skill, p skills.ServitorParams) *Summon {
	if s.SummonOf(owner) != nil || s.worldData == nil {
		return nil
	}
	t, ok := s.worldData.Npc(p.NpcID)
	if !ok {
		s.log.Warn("game: a summon skill names a missing NPC", "skill", sk.ID, "npc", p.NpcID)
		return nil
	}
	sm, err := s.newSummon(owner, t, summon.Servitor)
	if err != nil {
		s.log.Warn("game: cannot make a servitor", "npc", p.NpcID, "err", err)
		return nil
	}
	sm.refSkill = sk.ID
	sm.expMult = p.ExpMultiplier
	sm.life = summon.NewLife(p.LifeTime, p.ConsumeItemID, p.ConsumeItemCount, p.ConsumeItemInterval, t.Race == data.RaceSiegeWeapon)
	pos := s.summonSpawnPoint(owner)
	if err := s.spawnSummon(sm, pos); err != nil {
		s.log.Warn("game: cannot spawn a servitor", append(owner.logArgs(), "npc", p.NpcID, "err", err)...)
		s.releaseNpcID(sm.npc)
		return nil
	}
	s.SetNpcRunning(sm.npc, true)
	s.restoreServitorEffects(sm)
	s.servitorLifeTick(sm) // Mobius schedules the life task at once
	if !sm.gone {
		sm.lifeTimer = s.loop.Every(summon.LifeTick, func() { s.servitorLifeTick(sm) })
	}
	return sm
}

// servitorLifeTick is one run of Mobius Servitor.run: the life time runs
// out ("Servitor passed away.", unsummoned), or the upkeep is taken ("A
// summoned monster uses $s1.") or missing (unsummoned); then the owner gets
// SetSummonRemainTime and the effect icons.
func (s *Server) servitorLifeTick(sm *Summon) {
	n, owner := sm.npc, sm.owner
	if n.Dead || !n.InWorld() || sm.gone {
		if sm.lifeTimer != 0 {
			s.loop.Cancel(sm.lifeTimer)
			sm.lifeTimer = 0
		}
		return
	}
	switch sm.life.Tick() {
	case summon.LifeExpired:
		owner.SendSystemMessage(sumMsgServitorPassedAway)
		s.unsummonNow(sm)
		return
	case summon.LifeConsume:
		id, count := int32(sm.life.ItemID), int64(sm.life.Count) //nolint:gosec // item IDs and counts are small
		if sm.life.Consumed(s.DestroyItemByItemID(owner, id, count, false)) {
			owner.SendSystemMessage(sumMsgSummonedMonsterUses, serverpackets.ItemNameParam(id))
		} else {
			owner.SendSystemMessage(sumMsgNotEnoughForServitor)
			s.unsummonNow(sm)
			return
		}
	}
	owner.Send(serverpackets.NewSetSummonRemainTimePacket(serverpackets.SetSummonRemainTime{
		MaxTime: int32(sm.life.Total), RemainingTime: int32(sm.life.Remaining), //nolint:gosec // milliseconds of an hour
	}))
	s.summonIconsLater(sm)
}

// servitorDied is Servitor.doDie's part: the servitor is not restored.
func (s *Server) servitorDied(sm *Summon) {
	o := s.summonOwnerOf(sm.owner)
	if o.saved.Servitor != nil {
		o.saved.Servitor = nil
		s.writeOwnerSummons(sm.owner)
	}
}

// restoreServitor is CharSummonTable.restoreServitor at login: the skill
// that summoned the stored servitor is applied again at the owner's level of
// it, and the servitor gets its stored HP, MP and life time back.
func (s *Server) restoreServitor(p *Player) {
	o := s.summons.owners[p]
	if o == nil || o.saved.Servitor == nil || s.SummonOf(p) != nil {
		return
	}
	sv := *o.saved.Servitor
	level := p.skill.SkillLevel(int(sv.SkillID))
	sk, ok := s.Skill(int(sv.SkillID), level)
	if !ok {
		o.saved.Servitor = nil
		s.writeOwnerSummons(p)
		return
	}
	sk.ApplyTo(s.skills.env, p.skill, p.skill)
	sm := s.SummonOf(p)
	if sm == nil || sm.kind != summon.Servitor {
		return
	}
	sm.npc.SetCurrentHPMP(float64(sv.CurHP), float64(sv.CurMP))
	sm.life.Remaining = int(sv.Time)
	s.summonStatusUpdate(sm)
}

// servitorStore is Servitor.storeMe: a living servitor whose owner logs out
// is kept to come back (with its HP, MP and life left); any other unsummon
// forgets it (Servitor.unSummon's removeServitor).
func (s *Server) servitorStore(sm *Summon) {
	o := s.summonOwnerOf(sm.owner)
	if sm.refSkill == 0 {
		return
	}
	switch {
	case sm.restore && !sm.npc.Dead && sm.life.Remaining > 0 && restoreServitorOnReconnect:
		o.saved.Servitor = &storage.SavedServitor{
			SkillID: int32(sm.refSkill),             //nolint:gosec // skill IDs are small
			CurHP:   int32(javaRound(sm.npc.CurHP)), //nolint:gosec // HP fit
			CurMP:   int32(javaRound(sm.npc.CurMP)), //nolint:gosec // MP fit
			Time:    int32(sm.life.Remaining),       //nolint:gosec // milliseconds of an hour
		}
	case o.saved.Servitor == nil:
		return
	default:
		o.saved.Servitor = nil
	}
	s.writeOwnerSummons(sm.owner)
}

// servitorStoreEffects is Servitor.storeEffect: the servitor's effects are
// kept, per owner class and summoning skill, for the next servitor of the
// same skill (SummonStoreSkillCooltime).
func (s *Server) servitorStoreEffects(sm *Summon, store bool) {
	if !summonStoreSkillCooltime || sm.owner == nil || sm.refSkill == 0 {
		return
	}
	o := s.summonOwnerOf(sm.owner)
	class := sm.owner.classIndex()
	kept := o.saved.Effects[:0]
	for _, e := range o.saved.Effects {
		if e.ClassIndex != class || e.SummonSkillID != int32(sm.refSkill) { //nolint:gosec // skill IDs are small
			kept = append(kept, e)
		}
	}
	o.saved.Effects = kept
	if store {
		if rows := s.summonEffectsToSave(sm); len(rows) > 0 {
			o.saved.Effects = append(o.saved.Effects, storage.ServitorEffects{
				ClassIndex: class, SummonSkillID: int32(sm.refSkill), Effects: rows, //nolint:gosec // skill IDs are small
			})
		}
	}
	s.writeOwnerSummons(sm.owner)
}

// restoreServitorEffects is Servitor.restoreEffects: the effects a servitor
// of the same skill kept come back on it, and are used up.
func (s *Server) restoreServitorEffects(sm *Summon) {
	if !summonStoreSkillCooltime || sm.owner == nil {
		return
	}
	o := s.summonOwnerOf(sm.owner)
	class := sm.owner.classIndex()
	for i, e := range o.saved.Effects {
		if e.ClassIndex != class || e.SummonSkillID != int32(sm.refSkill) { //nolint:gosec // skill IDs are small
			continue
		}
		s.applySummonEffects(sm, e.Effects)
		o.saved.Effects = append(o.saved.Effects[:i], o.saved.Effects[i+1:]...)
		s.writeOwnerSummons(sm.owner)
		return
	}
}

// summonEffectsToSave are the effects of a summon to keep (Summon
// storeEffect's rules: no heals, toggles or dances, one per reuse hash),
// with their seconds left.
func (s *Server) summonEffectsToSave(sm *Summon) []storage.SavedSkill {
	if sm.npc.skill == nil {
		return nil
	}
	var out []storage.SavedSkill
	for _, info := range sm.npc.skill.el.ToStore() {
		sk := info.Skill()
		out = append(out, storage.SavedSkill{
			SkillID:       int32(sk.ID),       //nolint:gosec // skill IDs are small
			SkillLevel:    int32(sk.Level),    //nolint:gosec // skill levels are small
			RemainingTime: int32(info.Time()), //nolint:gosec // seconds of an effect
			RestoreType:   storage.RestoreEffect,
		})
	}
	return out
}

// applySummonEffects puts saved effects back on a summon (SummonEffectTable
// apply: the skill's effects with the summon as caster and target, for the
// seconds left).
func (s *Server) applySummonEffects(sm *Summon, rows []storage.SavedSkill) {
	if sm.npc.skill == nil {
		return
	}
	a := sm.npc.skill
	for _, r := range rows {
		sk, ok := s.Skill(int(r.SkillID), int(r.SkillLevel))
		if !ok || !sk.HasEffects(data.ScopeGeneral) {
			continue
		}
		sk.ApplyEffects(s.skills.env, a, a, false, false, false, int(r.RemainingTime))
	}
}
