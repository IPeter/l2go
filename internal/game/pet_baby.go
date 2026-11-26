package game

import (
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/summon"
	"github.com/IPeter/l2go/internal/world"
)

// Baby pets (Mobius BabyPet): a pet that, in support mode, heals its owner
// when hurt, keeps its buffs up and recharges its MP in combat, on a cast
// task every 2 s (the first after 3 s). Switch Stance (6054) toggles the
// support mode.

// Baby pet skills Mobius treats by ID.
const (
	babyBuffControl  = 5771 // Buff Control: no buffs while it is on the pet
	babyAwakening    = 5753
	babySwitchStance = 6054 // RequestActionUse.SWITCH_STANCE_ID
)

// Baby pet task timing (BabyPet.startCastTask).
const (
	babyCastFirst  = 3 * time.Second
	babyCastPeriod = 2 * time.Second
)

// sumMsgYourPetUses is "Your pet uses $s1.".
const sumMsgYourPetUses serverpackets.SystemMessageID = 1018

// babyState is a baby pet's support state (BabyPet's fields).
type babyState struct {
	supportOff bool // !_bufferMode
	buffs      []*skills.Skill
	majorHeal  *skills.Skill
	minorHeal  *skills.Skill
	recharge   *skills.Skill
	task       world.TimerID
}

// babyPetSupportMode is BabyPet.isInSupportMode; plain pets and servitors
// are always "in support mode" for the action checks.
func (s *Server) babyPetSupportMode(sm *Summon) bool {
	return sm.pet == nil || sm.pet.baby == nil || !sm.pet.baby.supportOff
}

// babyPetSpawned is BabyPet.onSpawn: the pet's skills at its level sorted
// into the major and minor heal, the recharge and the buffs (Buff Control
// and Awakening left out); then the cast task starts.
func (s *Server) babyPetSpawned(sm *Summon) {
	ps := sm.pet
	b := &babyState{}
	ps.baby = b
	seen := make(map[int]bool)
	healPower := 0.0
	for _, row := range ps.data.Skills {
		if seen[row.SkillID] {
			continue
		}
		seen[row.SkillID] = true
		level := s.summonSkillLevel(sm, row.SkillID)
		if level == 0 {
			continue
		}
		sk, ok := s.Skill(row.SkillID, level)
		if !ok || sk.ID == babyBuffControl || sk.ID == babyAwakening {
			continue
		}
		switch {
		case sk.HasEffectType(skills.TypeManaHealByLevel):
			b.recharge = sk
		case sk.HasEffectType(skills.TypeHeal):
			switch {
			case healPower == 0:
				b.majorHeal, b.minorHeal = sk, sk
				healPower = sk.Power
			case sk.Power > healPower:
				b.majorHeal = sk
			default:
				b.minorHeal = sk
			}
		case sk.IsContinuous() && !sk.IsDebuff:
			b.buffs = append(b.buffs, sk)
		}
	}
	s.babyStartTask(sm)
}

// babyStartTask is BabyPet.startCastTask.
func (s *Server) babyStartTask(sm *Summon) {
	b := sm.pet.baby
	if b == nil || b.task != 0 || sm.npc.Dead {
		return
	}
	if b.majorHeal == nil && b.buffs == nil && b.recharge == nil {
		return
	}
	b.task = s.loop.After(babyCastFirst, func() {
		b.task = s.loop.Every(babyCastPeriod, func() { s.babyCastTick(sm) })
		s.babyCastTick(sm)
	})
}

// babyStopTask is BabyPet.stopCastTask.
func (s *Server) babyStopTask(sm *Summon) {
	if sm.pet == nil || sm.pet.baby == nil || sm.pet.baby.task == 0 {
		return
	}
	s.loop.Cancel(sm.pet.baby.task)
	sm.pet.baby.task = 0
}

// babyCastTick is BabyPet.CastTask.run: with a living, controllable owner
// and pet in support mode, not casting: a heal when the owner is hurt (75 %
// of the time the major one under 15 % HP, 25 % of the time the minor one
// under 80 %), else a buff the owner lacks (unless Buff Control is on the
// pet), else, in combat under 60 % MP, the recharge on 61 % of the ticks.
func (s *Server) babyCastTick(sm *Summon) {
	b := sm.pet.baby
	owner, n := sm.owner, sm.npc
	if owner == nil || owner.IsDead() || owner.skill.IsInvul() || b == nil {
		return
	}
	el := n.skill.el
	if n.Dead || s.CastingNow(n) || el.IsAffected(skills.FlagBetrayed) || el.IsAffected(skills.FlagMuted) ||
		summon.Uncontrollable(sm.pet.fed) || b.supportOff || sm.ai.intention == summonCasting {
		return
	}
	if b.majorHeal != nil {
		hp := owner.CurHP / float64(owner.MaxHP())
		improved := s.items.categories != nil && s.items.categories.InCategory(data.CategoryBabyPetGroup, n.Template.ID)
		switch {
		case (improved && hp < 0.3) || (!improved && hp < 0.15):
			sk := b.majorHeal
			if !s.SkillDisabled(n, sk) && s.rnd.Get(100) <= 75 && n.CurMP >= float64(sk.MPConsume) {
				s.babyCast(sm, sk)
				return
			}
		case b.majorHeal != b.minorHeal && ((improved && hp < 0.7) || (!improved && hp < 0.8)):
			sk := b.minorHeal
			if !s.SkillDisabled(n, sk) && s.rnd.Get(100) <= 25 && n.CurMP >= float64(sk.MPConsume) {
				s.babyCast(sm, sk)
				return
			}
		}
	}
	if !el.IsAffectedBySkill(babyBuffControl) {
		var usable []*skills.Skill
		oel := owner.skill.el
		for _, sk := range b.buffs {
			if s.SkillDisabled(n, sk) || n.CurMP < float64(sk.MPConsume) {
				continue
			}
			if info := oel.ByAbnormal(sk.AbnormalType); info != nil && sk.AbnormalLevel <= info.Skill().AbnormalLevel {
				continue
			}
			if slices.Contains(oel.BlockedAbnormals(), sk.AbnormalType) {
				continue
			}
			usable = append(usable, sk)
		}
		if len(usable) > 0 {
			s.babyCast(sm, usable[s.rnd.Get(len(usable))])
			return
		}
	}
	if sk := b.recharge; sk != nil && s.InCombat(owner) && owner.CurMP/float64(owner.skill.MaxMP()) < 0.6 && s.rnd.Get(100) <= 60 {
		if !s.SkillDisabled(n, sk) && n.CurMP >= float64(sk.MPConsume) {
			s.babyCast(sm, sk)
		}
	}
}

// babyCast is BabyPet.castSkill: a pet that does not follow casts only
// within the skill's range of its owner; it casts at its owner ("Your pet
// uses $s1.") and keeps following as it did.
func (s *Server) babyCast(sm *Summon, sk *skills.Skill) {
	follow := sm.follow
	if !follow && !within3D(sm.npc.Position(), sm.owner.Position(), sk.CastRange) {
		return
	}
	sm.npc.ai.target = sm.owner
	s.summonUseMagic(sm, sk, false, false)
	sm.owner.SendSystemMessage(sumMsgYourPetUses, skills.SkillName(sk))
	if follow != sm.follow {
		s.summonSetFollowStatus(sm, follow)
	}
}
