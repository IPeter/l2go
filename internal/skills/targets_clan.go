package skills

import "github.com/IPeter/l2go/internal/data"

// The clan target types of M9 (Plan 12 Phase 7): CLAN, CLAN_MEMBER,
// CORPSE_CLAN and PARTY_CLAN (Mobius handlers/skill/targets). Their player
// side needs a World that is a ClanWorld; their NPC side asks it about the
// NPC clans (the template's social clans, not player clans).
func init() {
	targetHandlers[data.TargetClan] = targetClan
	targetHandlers[data.TargetClanMember] = targetClanMember
	targetHandlers[data.TargetCorpseClan] = targetCorpseClan
	targetHandlers[data.TargetPartyClan] = targetPartyClan
}

// ClanWorld is a World that knows the player clans and the NPC clans. The
// clan target types find no clan members, and no NPC of the caster's clan,
// when the World does not implement it.
type ClanWorld interface {
	// ClanMembers returns the members of c's clan who are in the game, in
	// the clan's member order, c included (Mobius Clan.getMembers with
	// getPlayer != null); nil when c is not a player in a clan.
	ClanMembers(c Creature) []Creature
	// NpcHasClans reports whether the NPC's template has clans (Mobius
	// getTemplate().getClans() not empty).
	NpcHasClans(npc Creature) bool
	// NpcInMyClan is Mobius npc.isInMyClan(other): both are NPCs and npc's
	// template clans match other's.
	NpcInMyClan(npc, other Creature) bool
}

func clanWorld(env *Env) ClanWorld {
	if env == nil || env.World == nil {
		return nil
	}
	cw, _ := env.World.(ClanWorld)
	return cw
}

// clanMembers returns the members of c's clan in the game, nil without a
// ClanWorld.
func clanMembers(env *Env, c Creature) []Creature {
	if cw := clanWorld(env); cw != nil {
		return cw.ClanMembers(c)
	}
	return nil
}

// npcClanFriends is the NPC side of CLAN and CORPSE_CLAN: the NPC, then the
// NPCs within the cast range that are in its clan, up to limit; a limit of
// 0 is no limit unless alwaysCheckLimit is set (CORPSE_CLAN compares the
// size with it anyway).
func npcClanFriends(env *Env, s *Skill, npc Creature, limit int, alwaysCheckLimit bool) []Creature {
	list := []Creature{npc}
	cw := clanWorld(env)
	if cw == nil || !cw.NpcHasClans(npc) {
		return list
	}
	for _, obj := range visibleInRange(env, npc, s.CastRange) {
		if !isNpc(obj) || !cw.NpcInMyClan(npc, obj) {
			continue
		}
		if (alwaysCheckLimit || limit > 0) && len(list) >= limit {
			break
		}
		list = append(list, obj)
	}
	return list
}

// targetClan is CLAN: a player caster, its summon and the living members of
// its clan in the game within the affect range, each preceded by its summon;
// in the Olympiad the caster alone. An NPC caster gets itself and the NPCs
// of its clan within the cast range. Duels and events do not exist here.
func targetClan(env *Env, s *Skill, caster Creature, onlyFirst bool, _ Creature) []Creature {
	if isPlayable(caster) {
		player := asPlayer(caster)
		if player == nil {
			return nil
		}
		list := []Creature{caster}
		if player.InOlympiad() || onlyFirst {
			return list
		}
		radius := s.AffectRange
		if sm := summonOf(env, caster); addCharacter(caster, sm, radius, false) {
			list = append(list, sm)
		}
		for _, m := range clanMembers(env, caster) {
			mp := asPlayer(m)
			if m == nil || same(m, caster) || mp == nil {
				continue
			}
			if !player.CheckPvPSkill(mp, s) {
				continue
			}
			if sm := summonOf(env, m); addCharacter(caster, sm, radius, false) {
				list = append(list, sm)
			}
			if !addCharacter(caster, m, radius, false) {
				continue
			}
			list = append(list, m)
		}
		return list
	}
	if isNpc(caster) {
		return npcClanFriends(env, s, caster, s.RollAffectLimit(env.Rand), false)
	}
	return nil
}

// targetClanMember is CLAN_MEMBER: for an NPC caster, the first NPC of its
// clan within the cast range, else itself; nothing for anyone else.
func targetClanMember(env *Env, s *Skill, caster Creature, _ bool, _ Creature) []Creature {
	if !isNpc(caster) {
		return nil
	}
	cw := clanWorld(env)
	if cw == nil || !cw.NpcHasClans(caster) {
		return []Creature{caster}
	}
	for _, obj := range visibleInRange(env, caster, s.CastRange) {
		if isNpc(obj) && cw.NpcInMyClan(caster, obj) {
			return []Creature{obj}
		}
	}
	return []Creature{caster}
}

// targetCorpseClan is CORPSE_CLAN: the dead members of a player caster's
// clan in the game within the affect range, not in a siege zone without a
// siege, up to the affect limit; in the Olympiad the caster alone. An NPC
// caster gets itself and the NPCs of its clan within the cast range, up to
// the affect limit (Mobius compares the size with the limit even when it is
// 0, so such a skill finds the NPC alone).
func targetCorpseClan(env *Env, s *Skill, caster Creature, onlyFirst bool, _ Creature) []Creature {
	if isPlayable(caster) {
		player := asPlayer(caster)
		if player == nil {
			return nil
		}
		if player.InOlympiad() {
			return []Creature{caster}
		}
		var list []Creature
		radius := s.AffectRange
		limit := s.RollAffectLimit(env.Rand)
		for _, m := range clanMembers(env, caster) {
			mp := asPlayer(m)
			if m == nil || same(m, caster) || mp == nil {
				continue
			}
			if !player.CheckPvPSkill(mp, s) {
				continue
			}
			if !addCharacter(caster, m, radius, true) {
				continue
			}
			if m.InsideZone(ZoneSiege) && !mp.IsInSiege() {
				continue
			}
			list = append(list, m)
			if onlyFirst {
				return list
			}
			if limit > 0 && len(list) >= limit {
				break
			}
		}
		return list
	}
	if isNpc(caster) {
		return npcClanFriends(env, s, caster, s.RollAffectLimit(env.Rand), true)
	}
	return nil
}

// targetPartyClan is PARTY_CLAN: a player caster, its summon, and the
// living players within the affect range who are in its clan or its party,
// each preceded by its summon, up to the affect limit. In the Olympiad only
// other Olympiad players count (the game and side checks come with the
// Olympiad, M11).
func targetPartyClan(env *Env, s *Skill, caster Creature, onlyFirst bool, _ Creature) []Creature {
	player := asPlayer(caster)
	if player == nil {
		return nil
	}
	list := []Creature{caster}
	if onlyFirst {
		return list
	}
	radius := s.AffectRange
	hasClan := player.ClanID() != 0
	hasParty := inParty(env, caster)
	if sm := summonOf(env, caster); addCharacter(caster, sm, radius, false) {
		list = append(list, sm)
	}
	if !hasClan && !hasParty {
		return list
	}
	limit := s.RollAffectLimit(env.Rand)
	for _, obj := range visibleInRange(env, caster, radius) {
		op := asPlayer(obj)
		if !isPlayer(obj) || op == nil {
			continue
		}
		if player.InOlympiad() && !op.InOlympiad() {
			continue
		}
		sameClan := hasClan && op.ClanID() == player.ClanID()
		sameGroup := hasParty && inParty(env, obj) && sameParty(env, caster, obj)
		if !sameClan && !sameGroup {
			continue
		}
		if !player.CheckPvPSkill(op, s) {
			continue
		}
		if sm := summonOf(env, obj); addCharacter(caster, sm, radius, false) {
			list = append(list, sm)
		}
		if !addCharacter(caster, obj, radius, false) {
			continue
		}
		list = append(list, obj)
		if limit > 0 && len(list) >= limit {
			break
		}
	}
	return list
}
