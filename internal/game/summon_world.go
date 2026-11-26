package game

import (
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/summon"
)

// The summoning effects' side of the world (skills.SummonWorld).

var _ skills.SummonWorld = skillWorld{}

// summonPlayerOf returns the player behind a skills.Creature, nil for
// anything else.
func (w skillWorld) summonPlayerOf(c skills.Creature) *Player {
	p, _ := w.s.gameCreature(c).(*Player)
	return p
}

// Summon returns the summon of a player (or of a summon's owner), nil
// without one.
func (w skillWorld) Summon(owner skills.Creature) skills.Creature {
	p := w.s.actingPlayer(w.s.gameCreature(owner))
	if p == nil {
		return nil
	}
	if sm := w.s.SummonOf(p); sm != nil {
		return sm.npc.skill
	}
	return nil
}

// CanSummon is ConditionPlayerCanSummon's test (Mobius): no summon (nor one
// waiting to be restored), not riding, not spawn protected.
func (w skillWorld) CanSummon(c skills.Creature) bool {
	s := w.s
	p := s.actingPlayer(s.gameCreature(c))
	if p == nil || s.SpawnProtected(p) {
		return false
	}
	o := s.summons.owners[p]
	switch {
	case o != nil && o.saved.Servitor != nil && restoreServitorOnReconnect,
		o != nil && o.restorePet != nil && restorePetOnReconnect,
		s.SummonOf(p) != nil:
		p.SendSystemMessage(sumMsgMultiplePets)
		return false
	case s.mounted(p):
		return false
	}
	return true
}

// SummonServitor is Mobius Summon.onStart.
func (w skillWorld) SummonServitor(effector skills.Creature, sk *skills.Skill, p skills.ServitorParams) {
	if pl := w.summonPlayerOf(effector); pl != nil {
		w.s.summonServitor(pl, sk, p)
	}
}

// SummonPet is Mobius SummonPet.onStart (pet.go).
func (w skillWorld) SummonPet(effector, _ skills.Creature, _ *skills.Skill) {
	if pl := w.summonPlayerOf(effector); pl != nil {
		w.s.summonPetFromCollar(pl)
	}
}

// SummonCubic is Mobius SummonCubic.onStart (cubic.go).
func (w skillWorld) SummonCubic(effector, effected skills.Creature, sk *skills.Skill, p skills.CubicParams) {
	if pl := w.summonPlayerOf(effected); pl != nil {
		w.s.summonCubic(pl, effector != effected, sk, p)
	}
}

// SetMaxCubics is CubicMastery's PlayerStat.setMaxCubicCount.
func (w skillWorld) SetMaxCubics(c skills.Creature, count int) {
	if pl := w.summonPlayerOf(c); pl != nil {
		w.s.summonOwnerOf(pl).maxCubics = count
	}
}

// Unsummon is Mobius Unsummon.onStart: the summon stops, loses its effects
// and goes; its owner is told "Your servitor has vanished!".
func (w skillWorld) Unsummon(c skills.Creature) {
	s := w.s
	sm := summonOfNpc(s.gameCreature(c))
	if sm == nil {
		return
	}
	n := sm.npc
	s.AbortAttack(n)
	s.AbortCast(n)
	if n.skill != nil {
		n.skill.el.StopAll()
	}
	owner := sm.owner
	s.unsummon(owner)
	owner.SendSystemMessage(sumMsgServitorVanished)
}

// SummonObject is player:summon() (scripting.PlayerSummon, Mobius
// getSummon): the summon as an NpcRef, nil without one. Loop-only.
func (p *Player) SummonObject() any {
	if p.srv == nil {
		return nil
	}
	if sm := p.srv.SummonOf(p); sm != nil {
		return sm.npc.ScriptRef()
	}
	return nil
}

var (
	_ scripting.PlayerSummon = (*Player)(nil)
	_                        = summon.Servitor
)
