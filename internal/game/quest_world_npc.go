package game

import (
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/world"
)

// The scripts' NPC AI, skills and items (scripting.NpcActions, QuestSkills
// and QuestItems on ScriptOutput) and the player and NPC refs' status,
// weapon, target and noblesse. Everything here runs on the loop, while a
// handler runs or a quest timer fires.

var (
	_ scripting.NpcActions   = (*ScriptOutput)(nil)
	_ scripting.QuestSkills  = (*ScriptOutput)(nil)
	_ scripting.QuestItems   = (*ScriptOutput)(nil)
	_ scripting.NpcStatus    = npcRef{}
	_ scripting.PlayerCombat = (*Player)(nil)
	_ scripting.PlayerNoble  = (*Player)(nil)
	_ scripting.PlayerStatus = (*Player)(nil)
)

// creatureOfRef returns the game's creature behind a PlayerRef or NpcRef,
// or nil.
func creatureOfRef(obj any) Creature {
	switch x := obj.(type) {
	case *Player:
		return x
	case scripting.NpcRef:
		if n := npcOfRef(x); n != nil {
			return n
		}
	}
	return nil
}

// npc returns the connected server and the game's NPC of ref; ok is false
// when either is missing or the NPC is not in the world.
func (o *ScriptOutput) npc(ref scripting.NpcRef) (*Server, *Npc, bool) {
	s := o.srv.Load()
	n := npcOfRef(ref)
	return s, n, s != nil && n != nil && n.InWorld()
}

// fighter returns the server and NPC of ref when the NPC is an attackable
// with a hate list and an attack AI, alive, and target is a creature.
func (o *ScriptOutput) fighter(ref scripting.NpcRef, target any) (*Server, *Npc, Creature, bool) {
	s, n, ok := o.npc(ref)
	t := creatureOfRef(target)
	if !ok || t == nil || n.Dead || n.class == nil || !n.class.attackable {
		return nil, nil, nil, false
	}
	return s, n, t, true
}

// NpcAttack is Mobius getAI().setIntention(ATTACK, target).
func (o *ScriptOutput) NpcAttack(npc scripting.NpcRef, target any) {
	if s, n, t, ok := o.fighter(npc, target); ok {
		s.npcSetAttack(n, t)
	}
}

// NpcAddDamageHate is Mobius Attackable.addDamageHate.
func (o *ScriptOutput) NpcAddDamageHate(npc scripting.NpcRef, target any, damage, hate int64) {
	if _, n, t, ok := o.fighter(npc, target); ok {
		n.AddDamageHate(t, damage, hate)
	}
}

// NpcRemoveAttacker is Mobius getAttackByList().remove(target).
func (o *ScriptOutput) NpcRemoveAttacker(npc scripting.NpcRef, target any) {
	n, t := npcOfRef(npc), creatureOfRef(target)
	if n == nil || t == nil {
		return
	}
	n.ai.attackBy = slices.DeleteFunc(n.ai.attackBy, func(c Creature) bool { return c == t })
}

// NpcAbortAttack is Mobius abortAttack.
func (o *ScriptOutput) NpcAbortAttack(npc scripting.NpcRef) {
	if s, n, ok := o.npc(npc); ok {
		s.AbortAttack(n)
	}
}

// NpcMoveTo is Mobius getAI().setIntention(MOVE_TO, loc): an attackable's
// AI walks there and turns ACTIVE on arrival; any other NPC leaves its
// attack stance and walks there (CreatureAI.onIntentionMoveTo).
func (o *ScriptOutput) NpcMoveTo(npc scripting.NpcRef, loc scripting.Location) {
	s, n, ok := o.npc(npc)
	if !ok || n.Dead {
		return
	}
	pos := world.Position{X: int32(loc.X), Y: int32(loc.Y), Z: int32(loc.Z), Heading: n.Position().Heading} //nolint:gosec // coordinates fit in int32
	if n.class != nil && n.class.attackable {
		s.npcSetMoveTo(n, pos)
		return
	}
	s.StopAutoAttack(n)
	s.AbortAttack(n)
	s.NpcMoveTo(n, pos)
}

// NpcSetRunning is Mobius setRunning.
func (o *ScriptOutput) NpcSetRunning(npc scripting.NpcRef) {
	if s, n, ok := o.npc(npc); ok {
		s.SetNpcRunning(n, true)
	}
}

// NpcDie is Mobius doDie(killer); a killer that is not a creature of the
// game counts as the NPC itself.
func (o *ScriptOutput) NpcDie(npc scripting.NpcRef, killer any) {
	s, n, ok := o.npc(npc)
	if !ok {
		return
	}
	k := creatureOfRef(killer)
	if k == nil {
		k = n
	}
	s.Kill(n, k)
}

// NpcSetRespawn is Mobius getSpawn().startRespawn() and stopRespawn(). An
// NPC from q:add_spawn has a spawn too in Mobius, with no respawn delay;
// here it gets a spawn slot at its spawn point when its respawn starts.
func (o *ScriptOutput) NpcSetRespawn(npc scripting.NpcRef, on bool) {
	s, n := o.srv.Load(), npcOfRef(npc)
	if s == nil || n == nil {
		return
	}
	sp := n.spawn
	if sp == nil {
		if !on {
			return
		}
		home, _, ok := s.npcHome(n)
		if !ok {
			home = n.Position()
		}
		sp = &npcSpawn{tmpl: n.Template, point: home, lastPoint: home, npc: n}
		s.addSpawnSlot(sp)
		n.spawn = sp
	}
	sp.respawn = on
}

// NpcCast is Mobius doCast(skill), at target or at the NPC's target.
func (o *ScriptOutput) NpcCast(npc scripting.NpcRef, skill scripting.Skill, target any) {
	s, n, ok := o.npc(npc)
	if !ok || n.Dead {
		return
	}
	sk, found := s.Skill(skill.ID, skill.Level)
	if !found {
		return
	}
	if t := creatureOfRef(target); t != nil {
		s.NpcCast(n, t, sk)
		return
	}
	s.DoCast(n, sk)
}

// NpcSetTarget is Mobius npc.setTarget(target).
func (o *ScriptOutput) NpcSetTarget(npc scripting.NpcRef, target any) {
	s, n, ok := o.npc(npc)
	if !ok || n.skill == nil {
		return
	}
	n.skill.SetTarget(s.skillCreatureOf(creatureOfRef(target)))
}

// NpcTeleport is Mobius teleToLocation(x, y, z).
func (o *ScriptOutput) NpcTeleport(npc scripting.NpcRef, loc scripting.Location) {
	if s, n, ok := o.npc(npc); ok {
		s.TeleportNpc(n, world.Position{X: int32(loc.X), Y: int32(loc.Y), Z: int32(loc.Z)}) //nolint:gosec // coordinates fit in int32
	}
}

// NpcDropItem is Mobius npc.dropItem(player, id, count).
func (o *ScriptOutput) NpcDropItem(npc scripting.NpcRef, p scripting.PlayerRef, itemID int, count int64) {
	s, n, ok := o.npc(npc)
	pl := playerOfRef(p)
	if !ok || pl == nil || count <= 0 {
		return
	}
	t, found := s.data.ItemTemplate(itemID)
	if !found {
		return
	}
	s.npcDropItem(n, pl, t, int(min(count, 1<<31-1)))
}

// NpcExists reports whether an NPC of the template is in the world.
func (o *ScriptOutput) NpcExists(npcID int) bool {
	s := o.srv.Load()
	if s == nil {
		return false
	}
	for obj := range s.world.Objects() {
		if n, ok := obj.(*Npc); ok && n.Template.ID == npcID {
			return true
		}
	}
	return false
}

// MagicSkillUse broadcasts MagicSkillUse of caster on target to the players
// around the caster, the caster included.
func (o *ScriptOutput) MagicSkillUse(caster, target any, skillID, level int, hitTime, reuseDelay time.Duration) {
	s := o.srv.Load()
	c := creatureOfRef(caster)
	if s == nil || c == nil || !creatureInWorld(c) {
		return
	}
	t := creatureOfRef(target)
	if t == nil {
		t = c
	}
	s.world.Broadcast(c, castUsePacket(c, t, skillID, level, int(hitTime/time.Millisecond), int(reuseDelay/time.Millisecond)), true)
}

// ----- skills and items -----

// SkillExists reports whether the skill level exists.
func (o *ScriptOutput) SkillExists(id, level int) bool {
	s := o.srv.Load()
	if s == nil {
		return false
	}
	_, ok := s.Skill(id, level)
	return ok
}

// IsAffectedBySkill is Mobius isAffectedBySkill.
func (o *ScriptOutput) IsAffectedBySkill(obj any, skillID int) bool {
	switch c := creatureOfRef(obj).(type) {
	case *Player:
		return c.skill != nil && c.Effects().IsAffectedBySkill(skillID)
	case *Npc:
		return c.skill != nil && c.Effects().IsAffectedBySkill(skillID)
	}
	return false
}

// ApplyEffects is Mobius skill.applyEffects(effector, effected).
func (o *ScriptOutput) ApplyEffects(skill scripting.Skill, effector, effected any) {
	s := o.srv.Load()
	if s == nil {
		return
	}
	sk, ok := s.Skill(skill.ID, skill.Level)
	a, b := s.skillCreatureOf(creatureOfRef(effector)), s.skillCreatureOf(creatureOfRef(effected))
	if !ok || a == nil || b == nil {
		return
	}
	sk.ApplyEffects(s.SkillEnv(), a, b, false, false, true, 0)
}

// ItemTemplate returns an item template of the item data.
func (o *ScriptOutput) ItemTemplate(id int) (scripting.ItemTemplate, bool) {
	s := o.srv.Load()
	if s == nil {
		return scripting.ItemTemplate{}, false
	}
	t, ok := s.data.ItemTemplate(id)
	if !ok {
		return scripting.ItemTemplate{}, false
	}
	return scripting.ItemTemplate{ID: t.ID, Name: t.Name, Stackable: t.Stackable}, true
}

// ----- the refs -----

// Level is the NPC's level (scripting.NpcStatus).
func (r npcRef) Level() int { return r.n.Level() }

// CurrentHP is the NPC's current HP.
func (r npcRef) CurrentHP() float64 { return r.n.CurHP }

// MaxHP is the NPC's maximum HP.
func (r npcRef) MaxHP() float64 { return float64(r.n.MaxHP()) }

// ActiveWeaponID is the item ID of the player's right-hand weapon, 0 for
// none (scripting.PlayerCombat).
func (p *Player) ActiveWeaponID() int {
	if t := p.srv.activeWeaponItem(p); t != nil {
		return t.ID
	}
	return 0
}

// TargetObject is the player's target as scripts see it: a player, an NPC
// or nil.
func (p *Player) TargetObject() any {
	switch t := p.Target.(type) {
	case *Player:
		return t
	case *Npc:
		return t.ScriptRef()
	}
	return nil
}

// PkKills is the PK count (scripting.PlayerStatus).
func (p *Player) PkKills() int { return int(p.char.PKKills) }

// SetPkKills is Mobius setPkKills; the character is saved soon after.
func (p *Player) SetPkKills(n int) {
	p.char.PKKills = int32(max(0, min(n, 1<<31-1))) //nolint:gosec // clamped
	p.srv.RequestSave(p)
}

// ActiveQuests returns the IDs of the started quests (Mobius
// getAllActiveQuests; the QuestList entries).
func (p *Player) ActiveQuests() []int {
	entries := p.srv.questsOf(p).book.Active()
	ids := make([]int, len(entries))
	for i, e := range entries {
		ids[i] = int(e.ID)
	}
	return ids
}
