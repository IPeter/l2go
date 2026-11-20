package game

import (
	"github.com/IPeter/l2go/internal/scripting"
)

// The world events the scripts hear of (Plan 11, "Game integration" 3):
// kills (2.5 s after the death, Mobius _onKillDelay), attacks, spawns,
// zones, player deaths and level-ups. Handlers run in script load order,
// each with the engine's time limit; a run nested in another (a reward
// that levels the player up) shares the outer deadline. Failures are
// logged by the engine.

// questDied is the OnDie hook: the kill handlers of an attackable NPC killed
// by a player run after the kill delay, unless the killer left the world
// meanwhile (decision Q6); a player's death tells the quests that asked for
// it and the tutorial ("CE30").
func (s *Server) questDied(victim, killer Creature) {
	if s.scripts == nil {
		return
	}
	switch v := victim.(type) {
	case *Npc:
		p, ok := killer.(*Player)
		if !ok || v.class == nil || !v.class.attackable || !s.scripts.HasKill(v.Template.ID) {
			return
		}
		ref := v.ScriptRef()
		s.loop.After(s.quest.settings.KillEventDelay, func() {
			if p.state == stateInGame {
				_ = s.scripts.Kill(s.background(), ref, p, false)
			}
		})
	case *Player:
		s.questPlayerDied(v, killer)
	}
}

// questPlayerDied runs the death handlers of the quests that asked to hear
// of p's death (Mobius onDeath), then the tutorial's "CE30".
func (s *Server) questPlayerDied(p *Player, killer Creature) {
	var k any
	switch x := killer.(type) {
	case *Player:
		k = x
	case *Npc:
		k = x.ScriptRef()
	}
	for _, name := range append([]string(nil), s.questsOf(p).deathNotify...) {
		_ = s.scripts.Death(s.background(), name, k, p)
	}
	s.tutorialEvent(p, "CE30")
}

// questHit is the OnHit hook: the attack handlers of an NPC a player hit
// (Mobius onAttack, notified asynchronously from addDamage) run right
// after the hit.
func (s *Server) questHit(target, attacker Creature, damage int) {
	if s.scripts == nil {
		return
	}
	n, ok := target.(*Npc)
	p, isPlayer := attacker.(*Player)
	if !ok || !isPlayer || !s.scripts.HasAttack(n.Template.ID) {
		return
	}
	ref := n.ScriptRef()
	s.loop.Post(func() {
		if p.state == stateInGame {
			_ = s.scripts.Attack(s.background(), ref, p, damage, false, nil)
		}
	})
}

// questNpcSpawned runs when an NPC enters the world, also on a respawn:
// its script value resets and the spawn handlers run (Mobius onSpawn).
// Before the scripts started, startScripts runs them for the NPCs there.
func (s *Server) questNpcSpawned(n *Npc) {
	delete(s.quest.scriptValues, n)
	if s.scripts == nil || !s.quest.started {
		return
	}
	_ = s.scripts.Spawn(s.background(), n.ScriptRef())
}

// questNpcGone runs when an NPC leaves the world for good or until its
// respawn: its quest timers stop.
func (s *Server) questNpcGone(n *Npc) {
	delete(s.quest.scriptValues, n)
	if s.scripts != nil {
		s.scripts.NpcGone(n.ScriptRef().ObjectID())
	}
}

// questZone runs the zone handlers when p enters or leaves a zone.
func (s *Server) questZone(p *Player, id int, typ string, enter bool) {
	if s.scripts == nil {
		return
	}
	z := scripting.Zone{ID: id, Type: typ}
	if enter {
		_ = s.scripts.EnterZone(s.background(), p, z)
	} else {
		_ = s.scripts.ExitZone(s.background(), p, z)
	}
}

// questLevelUp tells the tutorial of a level-up ("CE40", Mobius
// PlayerStat.addLevel).
func (s *Server) questLevelUp(p *Player) {
	if p.state == stateInGame {
		s.tutorialEvent(p, "CE40")
	}
}
