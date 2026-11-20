package skills

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/stats"
)

// targetHandler is a Mobius target type handler (ITargetTypeHandler).
type targetHandler func(env *Env, s *Skill, caster Creature, onlyFirst bool, target Creature) []Creature

// targetHandlers are the ported target types (K11). Clan types come with
// M9; summon, pet, GROUND and HOLY with M11; UNLOCKABLE with M10. NONE and
// CORPSE have no Mobius handler.
var targetHandlers = map[data.TargetType]targetHandler{
	data.TargetSelf:          targetSelf,
	data.TargetOne:           targetOne,
	data.TargetAura:          targetAura,
	data.TargetArea:          targetArea,
	data.TargetParty:         targetParty,
	data.TargetFrontArea:     targetFrontArea,
	data.TargetPcBody:        targetPcBody,
	data.TargetCorpseMob:     targetCorpseMob,
	data.TargetPartyMember:   targetPartyMember,
	data.TargetAreaCorpseMob: targetAreaCorpseMob,
	data.TargetBehindAura:    targetBehindAura,
	data.TargetFrontAura:     targetFrontAura,
	data.TargetPartyNotme:    targetPartyNotMe,
	data.TargetAuraCorpseMob: targetAuraCorpseMob,
}

// TargetTypePorted reports whether a target type has a handler.
func TargetTypePorted(t data.TargetType) bool {
	_, ok := targetHandlers[t]
	return ok
}

// Targets is Skill.getTargetList(caster, onlyFirst, target): the creatures
// the skill acts on, target being the caster's current target. A handler
// tells the caster why it found no target where Mobius does. A target type
// without a handler gives none and the text "Target type of skill is not
// currently handled.".
func (s *Skill) Targets(env *Env, caster Creature, onlyFirst bool, target Creature) []Creature {
	h, ok := targetHandlers[s.TargetType]
	if !ok {
		if caster != nil {
			caster.SendText(TextTargetTypeNotHandled)
		}
		return nil
	}
	return h(env, s, caster, onlyFirst, target)
}

// TargetsOfType is the handler of target type t run for the skill
// (ITargetTypeHandler.getTargetList), as the trigger effects ask it; nil
// for a type without a handler.
func (s *Skill) TargetsOfType(env *Env, t data.TargetType, caster Creature, onlyFirst bool, target Creature) []Creature {
	h, ok := targetHandlers[t]
	if !ok {
		return nil
	}
	return h(env, s, caster, onlyFirst, target)
}

// FirstTarget is Skill.getFirstOfTargetList with the caster's own target:
// the first creature the skill would act on, nil if none.
func (s *Skill) FirstTarget(env *Env, caster Creature) Creature {
	if t := s.Targets(env, caster, true, caster.Target()); len(t) > 0 {
		return t[0]
	}
	return nil
}

func incorrectTarget(c Creature) []Creature {
	send(c, MsgIncorrectTarget)
	return nil
}

// srcInArena is the area handlers' arena test: a PvP zone that is not a
// siege zone.
func srcInArena(c Creature) bool { return c.InsideZone(ZonePvP) && !c.InsideZone(ZoneSiege) }

// visibleInRange and visible ask the world, nil without one.
func visibleInRange(env *Env, c Creature, rng int) []Creature {
	if env.World == nil {
		return nil
	}
	return env.World.VisibleInRange(c, rng)
}

func visible(env *Env, c Creature) []Creature {
	if env.World == nil {
		return nil
	}
	return env.World.Visible(c)
}

// canSee asks the world for line of sight; true without one.
func canSee(env *Env, from, to Creature) bool {
	return env.World == nil || env.World.CanSee(from, to)
}

// CheckAreaOffensive is Skill.checkForAreaOffensiveSkills: whether an area
// skill may hit target. Never a dead target or the caster. A player caster
// spares its own summon, players in observer mode, its siege side (negative
// skills), players in a peace zone (negative skills), its party, and
// outside arenas its clan and alliance and whom checkPvpSkill forbids. A
// monster spares other monsters. The target must be in sight. Events, auto
// play and command channels do not exist here.
func CheckAreaOffensive(env *Env, caster, target Creature, s *Skill, inArena bool) bool {
	if target == nil || target.IsDead() || same(target, caster) {
		return false
	}
	player := asPlayer(caster)
	tp := asPlayer(target)
	if player != nil {
		if tp != nil {
			if tp == player {
				return false
			}
			if tp.InObserverMode() {
				return false
			}
			if s.HasNegativeEffect() && player.SiegeState() > 0 && caster.InsideZone(ZoneSiege) &&
				player.SiegeState() == tp.SiegeState() && player.SiegeSide() == tp.SiegeSide() {
				return false
			}
			if s.HasNegativeEffect() && target.InsideZone(ZonePeace) {
				return false
			}
			if inParty(env, caster) && inParty(env, target) && sameParty(env, caster, target) {
				return false
			}
			if !inArena && !(target.InsideZone(ZonePvP) && !target.InsideZone(ZoneSiege)) {
				if player.AllyID() != 0 && player.AllyID() == tp.AllyID() {
					return false
				}
				if player.ClanID() != 0 && player.ClanID() == tp.ClanID() {
					return false
				}
				if !player.CheckPvPSkill(tp, s) {
					return false
				}
			}
		}
	} else if tp == nil && isAttackable(target) && isAttackable(caster) {
		return false
	}
	return canSee(env, caster, target)
}

// addCharacter is Skill.addCharacter: target counts when its death state
// is isDead and it is within radius (with collision, 3D) of the caster.
func addCharacter(caster, target Creature, radius int, isDead bool) bool {
	if target == nil || isDead != target.IsDead() {
		return false
	}
	return radius <= 0 || inRange(radius, caster, target, true)
}

// summonOf returns the summon of a player, nil without one or a world.
func summonOf(env *Env, owner Creature) Creature {
	if env.World == nil {
		return nil
	}
	return env.World.Summon(owner)
}

func targetSelf(_ *Env, _ *Skill, caster Creature, _ bool, _ Creature) []Creature {
	return []Creature{caster}
}

func targetOne(_ *Env, s *Skill, caster Creature, _ bool, target Creature) []Creature {
	if target == nil || target.IsDead() || (same(target, caster) && s.HasNegativeEffect()) {
		return incorrectTarget(caster)
	}
	return []Creature{target}
}

func targetAura(env *Env, s *Skill, caster Creature, onlyFirst bool, _ Creature) []Creature {
	var list []Creature
	inArena := srcInArena(caster)
	for _, obj := range visibleInRange(env, caster, s.AffectRange) {
		// Doors are hit only when stealthy, which no Interlude door is.
		if !isAttackable(obj) && !isPlayable(obj) {
			continue
		}
		if !CheckAreaOffensive(env, caster, obj, s, inArena) {
			continue
		}
		if isPlayable(caster) && isAttackable(obj) && !s.HasNegativeEffect() {
			continue
		}
		list = append(list, obj)
		if onlyFirst {
			return list
		}
	}
	return list
}

// areaOrigin is the common start of AREA and FRONT_AREA: the target check,
// and the origin of the area (the target for a skill with a cast range,
// else the caster). done is set when the list is final.
func areaOrigin(env *Env, s *Skill, caster Creature, onlyFirst bool, target Creature, firstBeforeAdd bool) (list []Creature, origin Creature, done bool) {
	if target == nil || ((same(target, caster) || target.IsAlikeDead()) && s.CastRange >= 0) || !(isAttackable(target) || isPlayable(target)) {
		return incorrectTarget(caster), nil, true
	}
	if s.CastRange < 0 {
		return nil, caster, false
	}
	if !CheckAreaOffensive(env, caster, target, s, srcInArena(caster)) {
		return nil, nil, true
	}
	if onlyFirst && firstBeforeAdd {
		return []Creature{target}, nil, true
	}
	list = append(list, target)
	if onlyFirst {
		return list, nil, true
	}
	return list, target, false
}

func targetArea(env *Env, s *Skill, caster Creature, onlyFirst bool, target Creature) []Creature {
	list, origin, done := areaOrigin(env, s, caster, onlyFirst, target, true)
	if done {
		return list
	}
	inArena := srcInArena(caster)
	limit := s.RollAffectLimit(env.Rand)
	for _, obj := range visible(env, caster) {
		if !(isAttackable(obj) || isPlayable(obj)) || same(obj, origin) {
			continue
		}
		if !inRange(s.AffectRange, origin, obj, true) {
			continue
		}
		if !CheckAreaOffensive(env, caster, obj, s, inArena) {
			continue
		}
		if limit > 0 && len(list) >= limit {
			continue
		}
		list = append(list, obj)
	}
	return list
}

func targetFrontArea(env *Env, s *Skill, caster Creature, onlyFirst bool, target Creature) []Creature {
	list, origin, done := areaOrigin(env, s, caster, onlyFirst, target, false)
	if done {
		return list
	}
	if !same(caster, origin) {
		cl, ol := caster.Location(), origin.Location()
		caster.SetHeading(stats.HeadingTo(cl.X, cl.Y, ol.X, ol.Y))
	}
	inArena := srcInArena(caster)
	limit := s.RollAffectLimit(env.Rand)
	for _, obj := range visible(env, caster) {
		if !(isAttackable(obj) || isPlayable(obj)) || same(obj, origin) {
			continue
		}
		if !inRange(s.AffectRange, origin, obj, true) {
			continue
		}
		if !isInFrontOf(obj, caster) {
			continue
		}
		if !CheckAreaOffensive(env, caster, obj, s, inArena) {
			continue
		}
		if limit > 0 && len(list) >= limit {
			continue
		}
		list = append(list, obj)
	}
	return list
}

// targetAuraSide is FRONT_AURA (front) and BEHIND_AURA (behind).
func targetAuraSide(env *Env, s *Skill, caster Creature, onlyFirst bool, front bool) []Creature {
	var list []Creature
	inArena := srcInArena(caster)
	limit := s.RollAffectLimit(env.Rand)
	for _, obj := range visibleInRange(env, caster, s.AffectRange) {
		if !(isAttackable(obj) || isPlayable(obj)) {
			continue
		}
		if front && !isInFrontOf(obj, caster) || !front && !isBehind(obj, caster) {
			continue
		}
		if !CheckAreaOffensive(env, caster, obj, s, inArena) {
			continue
		}
		list = append(list, obj)
		if onlyFirst {
			return list
		}
		if limit > 0 && len(list) >= limit {
			break
		}
	}
	return list
}

func targetFrontAura(env *Env, s *Skill, caster Creature, onlyFirst bool, _ Creature) []Creature {
	return targetAuraSide(env, s, caster, onlyFirst, true)
}

func targetBehindAura(env *Env, s *Skill, caster Creature, onlyFirst bool, _ Creature) []Creature {
	return targetAuraSide(env, s, caster, onlyFirst, false)
}

// targetParty is PARTY: the caster, its summon, and the living party
// members and their summons within the affect range. A summon as the
// caster (M11) gets itself and the party members only.
func targetParty(env *Env, s *Skill, caster Creature, onlyFirst bool, _ Creature) []Creature {
	list := []Creature{caster}
	if onlyFirst {
		return list
	}
	radius := s.AffectRange
	if isPlayer(caster) {
		if sm := summonOf(env, caster); addCharacter(caster, sm, radius, false) {
			list = append(list, sm)
		}
	}
	if !inParty(env, caster) {
		return list
	}
	for _, m := range env.World.Party(caster) {
		if m == nil || same(m, caster) {
			continue
		}
		if addCharacter(caster, m, radius, false) {
			list = append(list, m)
		}
		if sm := summonOf(env, m); addCharacter(caster, sm, radius, false) {
			list = append(list, sm)
		}
	}
	return list
}

// targetPartyMember is PARTY_MEMBER: a living target that is the caster,
// in its party, its summon or its owner.
func targetPartyMember(env *Env, s *Skill, caster Creature, _ bool, target Creature) []Creature {
	if target == nil {
		return incorrectTarget(caster)
	}
	if !target.IsDead() {
		switch {
		case same(target, caster),
			inParty(env, caster) && inParty(env, target) && sameParty(env, caster, target),
			isPlayer(caster) && isSummon(target) && same(summonOf(env, caster), target),
			isSummon(caster) && isPlayer(target) && same(summonOf(env, target), caster):
			return []Creature{target}
		}
	}
	if asPlayer(caster) != nil && isPlayer(target) {
		send(caster, MsgS1CannotBeUsedUnsuitable, SkillName(s))
	}
	return nil
}

// targetPartyNotMe is PARTY_NOTME: the other living party members within
// PartyRange and the affect range, each followed by its living summon.
func targetPartyNotMe(env *Env, s *Skill, caster Creature, _ bool, _ Creature) []Creature {
	if !inParty(env, caster) {
		return nil
	}
	var list []Creature
	for _, m := range env.World.Party(caster) {
		if m == nil || m.IsDead() || same(m, caster) || !inRange(env.Config.PartyRange, caster, m, true) {
			continue
		}
		if s.AffectRange > 0 && !inRange(s.AffectRange, caster, m, true) {
			continue
		}
		list = append(list, m)
		if sm := summonOf(env, m); sm != nil && !sm.IsDead() {
			list = append(list, sm)
		}
	}
	return list
}

// targetPcBody is PC_BODY: a dead player or pet targeted by a player. A
// resurrection may not raise a player in a siege zone without a siege, nor
// a festival participant.
func targetPcBody(_ *Env, s *Skill, caster Creature, _ bool, target Creature) []Creature {
	if target != nil && target.IsDead() && isPlayer(caster) && (isPlayer(target) || isPet(target)) {
		good := true
		if s.HasEffectType(TypeResurrection) && isPlayer(target) {
			tp := target.AsPlayer()
			if target.InsideZone(ZoneSiege) && !tp.IsInSiege() {
				good = false
				send(caster, MsgCannotResurrectInSiege)
			}
			if tp.IsFestivalParticipant() {
				good = false
				caster.SendText(TextNoFestivalResurrect)
			}
		}
		if good {
			return []Creature{target}
		}
	}
	return incorrectTarget(caster)
}

// targetCorpseMob is CORPSE_MOB: a dead attackable; a drain skill refuses a
// corpse too old to consume.
func targetCorpseMob(env *Env, s *Skill, caster Creature, _ bool, target Creature) []Creature {
	if target == nil || !isAttackable(target) || !target.IsDead() {
		return incorrectTarget(caster)
	}
	if s.HasEffectType(TypeHPDrain) && env.World != nil && env.World.OldCorpse(target, caster) {
		return nil
	}
	return []Creature{target}
}

// targetAreaCorpseMob is AREA_CORPSE_MOB: a dead attackable target, then
// the attackables and playables around it that the skill may hit.
func targetAreaCorpseMob(env *Env, s *Skill, caster Creature, onlyFirst bool, target Creature) []Creature {
	if target == nil || !isAttackable(target) || !target.IsDead() {
		return incorrectTarget(caster)
	}
	list := []Creature{target}
	if onlyFirst {
		return list
	}
	inArena := srcInArena(caster)
	for _, obj := range visible(env, caster) {
		if !(isAttackable(obj) || isPlayable(obj)) || !inRange(s.AffectRange, target, obj, true) {
			continue
		}
		if !CheckAreaOffensive(env, caster, obj, s, inArena) {
			continue
		}
		list = append(list, obj)
	}
	return list
}

// targetAuraCorpseMob is AURA_CORPSE_MOB: the dead attackables around the
// caster.
func targetAuraCorpseMob(env *Env, s *Skill, caster Creature, onlyFirst bool, _ Creature) []Creature {
	var list []Creature
	limit := s.RollAffectLimit(env.Rand)
	for _, obj := range visibleInRange(env, caster, s.AffectRange) {
		if !isAttackable(obj) || !obj.IsDead() {
			continue
		}
		list = append(list, obj)
		if onlyFirst {
			return list
		}
		if limit > 0 && len(list) >= limit {
			break
		}
	}
	return list
}
