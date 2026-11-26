package game

import (
	"errors"
	"slices"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/summon"
	"github.com/IPeter/l2go/internal/world"
)

// The summon actions of the action window (Mobius RequestActionUse: 15-23,
// 32-54 and 1000-1040; summon.Actions lists them).

// Texts Mobius sends with sendMessage for the summon actions.
const (
	summonTextNoPet      = "You do not have a pet."
	summonTextNoServitor = "You do not have a servitor."
	summonTextHelperPet  = "The hunting helper pet cannot be returned because there is not much time remaining until it leaves."
	summonTextAuxMode    = "A pet on auxiliary mode cannot use skills."
)

// registerSummonActions registers a handler for every summon action.
func (s *Server) registerSummonActions() error {
	ids := make([]int32, 0, len(summon.Actions))
	for id := range summon.Actions {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	var errs []error
	for _, id := range ids {
		def := summon.Actions[id]
		errs = append(errs, s.RegisterActionUse(id, func(p *Player, req clientpackets.RequestActionUse) {
			s.summonAction(p, req, def)
		}))
	}
	return errors.Join(errs...)
}

// summonAction runs one summon action for p (RequestActionUse.runImpl).
func (s *Server) summonAction(p *Player, req clientpackets.RequestActionUse, def summon.ActionDef) {
	if p.IsDead() || p.skill.fakeDeath {
		p.SendActionFailed()
		return
	}
	sm := s.SummonOf(p)
	target, _ := p.Target.(Creature)
	switch def.Kind {
	case summon.ActFollow:
		if s.validateSummon(p, sm, def) {
			s.summonFollowToggle(sm)
		}
	case summon.ActAttack:
		if s.validateSummon(p, sm, def) && s.summonCanAttack(sm, req.CtrlPressed) {
			s.summonSetAttack(sm, target)
		}
	case summon.ActStop:
		if s.validateSummon(p, sm, def) && !npcMovementDisabled(sm.npc) {
			s.summonSetActive(sm)
		}
	case summon.ActUnsummonPet:
		s.actionUnsummonPet(p, sm, def)
	case summon.ActUnsummonServitor:
		if !s.validateSummon(p, sm, def) {
			return
		}
		switch {
		case npcMovementDisabled(sm.npc):
			p.SendSystemMessage(sumMsgPetInDistress)
		case s.AttackingNow(sm.npc) || s.summonInCombat(sm):
			p.SendSystemMessage(sumMsgServitorInBattle)
		default:
			s.unsummonNow(sm)
		}
	case summon.ActMoveTo:
		if s.validateSummon(p, sm, def) && p.Target != nil && p.Target != world.Object(sm.npc) && !npcMovementDisabled(sm.npc) {
			sm.follow = false
			s.summonSetMoveTo(sm, p.Target.Position())
		}
	case summon.ActMount:
		s.mountPlayer(p, sm) // pet_ride.go
	case summon.ActDoorSkill:
		if def.SkillID == 4230 && !s.validateSummon(p, sm, def) {
			return
		}
		if _, ok := p.Target.(*Door); !ok {
			if def.SkillID == 4230 {
				p.SendSystemMessage(sumMsgInvalidTarget)
			}
			return
		}
		s.summonActionSkillByID(p, sm, def, req, target)
	case summon.ActSkillByID:
		s.summonActionSkillByID(p, sm, def, req, target)
	case summon.ActSkillByName:
		s.summonActionSkillByName(p, sm, def, req, target)
	case summon.ActSinEaterTalk:
		if s.validateSummon(p, sm, def) && sm.npc.Template.ID == summon.SinEaterID {
			line := summon.SinEaterLines[s.rnd.Get(len(summon.SinEaterLines))]
			s.world.Broadcast(sm.npc, serverpackets.NewNpcSayPacket(serverpackets.NpcSay{
				ObjectID: int32(sm.npc.ID()), Type: serverpackets.ChatGeneral,
				NpcID: int32(sm.npc.Template.DisplayID), Text: line, //nolint:gosec // NPC IDs are small
			}), false)
		}
	}
}

// validateSummon is RequestActionUse.validateSummon: the action's kind of
// summon, controllable and not betrayed; otherwise p is told.
func (s *Server) validateSummon(p *Player, sm *Summon, def summon.ActionDef) bool {
	if sm != nil && def.ValidFor(sm.kind) {
		if sm.pet != nil && summon.Uncontrollable(sm.pet.fed) {
			p.SendSystemMessage(sumMsgPetUnresponsive)
			return false
		}
		if sm.npc.skill != nil && sm.npc.skill.el.IsAffected(skills.FlagBetrayed) {
			p.SendSystemMessage(sumMsgPetUnresponsive)
			return false
		}
		return true
	}
	if def.Pet {
		p.SendMessage(summonTextNoPet)
	} else {
		p.SendMessage(summonTextNoServitor)
	}
	return false
}

// summonFollowToggle is SummonAI.notifyFollowStatusChange.
func (s *Server) summonFollowToggle(sm *Summon) {
	sm.ai.startFollow = !sm.ai.startFollow
	switch sm.ai.intention {
	case summonActive, summonFollowing, summonIdle, summonMovingTo, summonPickingUp:
		s.summonSetFollowStatus(sm, sm.ai.startFollow)
	}
}

// summonCanAttack is Mobius Summon.canAttack(ctrl): the owner's target, not
// the summon or the owner; not a passive summon; not betrayed; a pet at most
// 20 levels above its owner; outside peace zones; not a target the owner
// may not attack without Ctrl (unless an NPC: the summon follows it); a door
// only for a siege weapon; within 3,000 of the owner and in sight.
func (s *Server) summonCanAttack(sm *Summon, ctrl bool) bool {
	owner := sm.owner
	t, ok := owner.Target.(Creature)
	if !ok || t == nil || t == Creature(sm.npc) || t == Creature(owner) {
		return false
	}
	if summon.IsPassive(sm.npc.Template.ID) {
		owner.SendActionFailed()
		return false
	}
	if sm.npc.skill != nil && sm.npc.skill.el.IsAffected(skills.FlagBetrayed) {
		owner.SendSystemMessage(sumMsgPetUnresponsive)
		owner.SendActionFailed()
		return false
	}
	if sm.pet != nil && sm.level()-owner.Level() > summon.LevelGapControl {
		owner.SendSystemMessage(sumMsgPetTooHighLevel)
		owner.SendActionFailed()
		return false
	}
	if !owner.IsGM() && s.insidePeaceZone(owner, t) {
		owner.SendSystemMessage(sysMsgPeacefulZone)
		return false
	}
	_, npcTarget := t.(*Npc)
	if tp, isPlayer := t.(*Player); isPlayer && !ctrl && !tp.autoAttackableBy(owner) && !npcTarget {
		sm.follow = false
		s.summonSetFollow(sm, t)
		owner.SendSystemMessage(sumMsgInvalidTarget)
		return false
	}
	if _, door := owner.Target.(*Door); door && sm.npc.Template.Race != data.RaceSiegeWeapon {
		return false // only siege weapons attack doors and walls
	}
	if owner.Position().Distance3D(t.Position()) > summon.MaxCommandRange {
		s.summonSetFollow(sm, owner)
		owner.SendSystemMessage(sysMsgTargetOutOfRange)
		return false
	}
	if !s.canSee(sm.npc, t) {
		s.summonSetFollow(sm, owner)
		owner.SendSystemMessage(sysMsgCannotSeeTarget)
		return false
	}
	return true
}

// actionUnsummonPet is action 19 (Unsummon Pet).
func (s *Server) actionUnsummonPet(p *Player, sm *Summon, def summon.ActionDef) {
	if !s.validateSummon(p, sm, def) {
		return
	}
	switch {
	case sm.npc.Dead:
		p.SendSystemMessage(sumMsgDeadPetCannotBeSentBack)
	case npcMovementDisabled(sm.npc):
		p.SendSystemMessage(sumMsgPetInDistress)
	case s.AttackingNow(sm.npc) || s.summonInCombat(sm):
		p.SendSystemMessage(sumMsgPetCannotGoBackInBattle)
	case sm.pet == nil:
		p.SendMessage(summonTextHelperPet)
	case !summon.CanSendBack(sm.pet.fed, sm.pet.maxMeal()):
		p.SendSystemMessage(sumMsgHungryPetNoRestore)
	default:
		s.unsummonNow(sm)
	}
}

// summonCanControl is RequestActionUse.canControl.
func (s *Server) summonCanControl(p *Player, sm *Summon) bool {
	if sm.kind == summon.BabyPet && !s.babyPetSupportMode(sm) {
		p.SendMessage(summonTextAuxMode)
		return false
	}
	if sm.pet != nil && sm.level()-p.Level() > summon.LevelGapControl {
		p.SendSystemMessage(sumMsgPetTooHighLevel)
		return false
	}
	return true
}

// summonActionSkillByID is RequestActionUse.useSkill(skillId): the skill at
// the level the summon has of it.
func (s *Server) summonActionSkillByID(p *Player, sm *Summon, def summon.ActionDef, req clientpackets.RequestActionUse, target Creature) {
	if !s.validateSummon(p, sm, def) || !s.summonCanControl(p, sm) {
		return
	}
	if level := s.summonSkillLevel(sm, def.SkillID); level > 0 {
		if sk, ok := s.Skill(def.SkillID, level); ok {
			if def.OnOwner {
				target = p
			}
			sm.npc.ai.target = target
			s.summonUseMagic(sm, sk, req.CtrlPressed, req.ShiftPressed)
		}
	}
	s.summonSwitchMode(sm, def.SkillID)
}

// summonSwitchMode is Summon.switchMode after Switch Stance (6054): a baby
// pet toggles its support mode.
func (s *Server) summonSwitchMode(sm *Summon, skillID int) {
	if skillID == babySwitchStance && sm.pet != nil && sm.pet.baby != nil {
		sm.pet.baby.supportOff = !sm.pet.baby.supportOff
	}
}

// summonActionSkillByName is RequestActionUse.useSkill(skillName): the skill
// the template's parameters name.
func (s *Server) summonActionSkillByName(p *Player, sm *Summon, def summon.ActionDef, req clientpackets.RequestActionUse, target Creature) {
	if !s.validateSummon(p, sm, def) || !s.summonCanControl(p, sm) {
		return
	}
	ns, ok := sm.npc.Template.Parameters.Skills[def.SkillName]
	if !ok {
		return
	}
	sk, ok := s.Skill(ns.ID, ns.Level)
	if !ok {
		return
	}
	if def.OnOwner {
		target = p
	}
	sm.npc.ai.target = target
	s.summonUseMagic(sm, sk, req.CtrlPressed, req.ShiftPressed)
	s.summonSwitchMode(sm, sk.ID)
}

// summonSkillLevel is the level of a skill a summon uses by ID: a pet's from
// its pet data, a servitor's from PetSkillData.
func (s *Server) summonSkillLevel(sm *Summon, skillID int) int {
	maxLevel := func(id int) int {
		if s.skills.data == nil {
			return 0
		}
		return s.skillMaxLevel(id)
	}
	if sm.pet != nil && sm.pet.data != nil {
		rows := make([]summon.SkillRow, 0, len(sm.pet.data.Skills))
		for _, r := range sm.pet.data.Skills {
			rows = append(rows, summon.SkillRow{SkillID: r.SkillID, SkillLevel: r.SkillLevel, MinLevel: r.MinLevel})
		}
		return summon.PetAvailableLevel(rows, skillID, sm.level(), maxLevel)
	}
	return summon.SummonAvailableLevel(s.summonSkillRows(sm.npc.Template.ID), skillID, sm.level(), maxLevel)
}

// summonSkillRows returns the PetSkillData rows of an NPC.
func (s *Server) summonSkillRows(npcID int) []summon.SkillRow {
	if s.summons.data == nil {
		return nil
	}
	var rows []summon.SkillRow
	for _, r := range s.summons.data.PetSkills(npcID) {
		rows = append(rows, summon.SkillRow{SkillID: r.SkillID, SkillLevel: r.SkillLevel})
	}
	return rows
}

// skillMaxLevel returns the highest level of a skill in the catalog (Mobius
// SkillData.getMaxLevel), 0 when unknown.
func (s *Server) skillMaxLevel(id int) int {
	level := 0
	for l := 1; l <= 100; l++ {
		if _, ok := s.Skill(id, l); !ok {
			break
		}
		level = l
	}
	return level
}

// summonUseMagic is Mobius Summon.useMagic: a living summon not casting
// takes the CAST intention with sk at the skill's target (its owner for
// OWNER_PET, itself for the self and area types, else the first of the
// skill's targets), after the reuse, MP, HP, line and harmful-skill checks.
func (s *Server) summonUseMagic(sm *Summon, sk *skills.Skill, force, dontMove bool) bool {
	n, owner := sm.npc, sm.owner
	if sk == nil || n.Dead || owner == nil || sk.IsPassive() || s.CastingNow(n) {
		return false
	}
	b := &n.skill.skillBase
	var target Creature
	switch sk.TargetType {
	case data.TargetOwnerPet:
		target = owner
	case data.TargetParty, data.TargetAura, data.TargetFrontAura, data.TargetBehindAura, data.TargetSelf,
		data.TargetAuraCorpseMob, data.TargetCommandChannel:
		target = n
	default:
		if list := sk.Targets(s.skills.env, b.self, true, b.Target()); len(list) > 0 {
			target = s.gameCreature(list[0])
		}
	}
	if target == nil {
		owner.SendSystemMessage(sumMsgTargetCannotBeFound)
		return false
	}
	if s.SkillDisabled(n, sk) {
		owner.SendSystemMessage(sysMsgNotReadyForReuse, skills.SkillName(sk))
		return false
	}
	env := s.skills.env
	if b.CurrentMP() < float64(skills.MPConsume(env, b.self, sk)+skills.MPInitialConsume(b.self, sk)) {
		owner.SendSystemMessage(sysMsgNotEnoughMP)
		return false
	}
	if b.CurrentHP() <= float64(sk.HPConsume) {
		owner.SendSystemMessage(sysMsgNotEnoughHP)
		return false
	}
	if sk.HasNegativeEffect() {
		if target == Creature(owner) {
			return false
		}
		if !owner.IsGM() && s.insidePeaceZone(owner, target) {
			owner.SendSystemMessage(sysMsgPeacefulZone)
			return false
		}
		if d, isDoor := target.(*Door); isDoor {
			if !d.autoAttackableBy(owner) { // door_combat.go
				return false
			}
		} else if _, isNpc := target.(*Npc); !isNpc {
			tp, isPlayer := target.(*Player)
			aoe := sk.TargetType == data.TargetAura || sk.TargetType == data.TargetFrontAura || sk.TargetType == data.TargetBehindAura ||
				sk.TargetType == data.TargetClan || sk.TargetType == data.TargetParty || sk.TargetType == data.TargetSelf
			if isPlayer && !tp.autoAttackableBy(owner) && !force && !aoe {
				return false
			}
		}
	}
	_ = dontMove
	s.summonSetCast(sm, sk, target)
	return true
}
