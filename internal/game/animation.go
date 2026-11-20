package game

import (
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/world"
)

// Random animations (Plan 13, A13): Mobius RandomAnimationTaskManager. The
// NPCs of a region that turns active wait 5–60 s (monsters: their own
// bounds) and then, when idle, play SocialAction 2 or 3; each animation
// draws the next wait. An NPC plays at most one animation (random or on a
// talk) per 6 s. Chests, NPCs with the CORPSE AI, grand bosses and NPCs
// whose template or spawn turns them off never play one. Like Mobius, only
// the NPCs present when their region turns on are scheduled; one that
// spawns into an active region waits for the next activation.
func init() { registerFeature("animation", setupAnimation) }

const (
	// animationScan is how often the pending animations are looked at.
	animationScan = time.Second
	// animationFirst and animationLast are Rnd.get(2, 3): the social
	// actions of a random animation.
	animationFirst, animationLast = 2, 3
)

// animationFeature is the random animation state on the Server
// (Server.animation). Loop-only.
type animationFeature struct {
	minNpc, maxNpc         time.Duration
	minMonster, maxMonster time.Duration
	// pending holds the scheduled NPCs in the order they came (Mobius
	// PENDING_ANIMATIONS); due holds when each plays next.
	pending []*Npc
	due     map[*Npc]time.Time
}

// setupAnimation reads the bounds and starts the scan.
func setupAnimation(s *Server) error {
	w := s.gs.World
	f := &s.animation
	f.minNpc, f.maxNpc = time.Duration(w.MinNpcAnimation), time.Duration(w.MaxNpcAnimation)
	f.minMonster, f.maxMonster = time.Duration(w.MinMonsterAnimation), time.Duration(w.MaxMonsterAnimation)
	f.due = make(map[*Npc]time.Time)
	s.world.OnRegionActivation(s.animationRegionSwitched)
	s.loop.Every(animationScan, s.animationTick)
	return nil
}

// hasRandomAnimation is Mobius hasRandomAnimation: animations are on (the
// maximum wait is not 0), the NPC's own switch is on (its template and the
// spawn's disableRandomAnimation; never with the CORPSE AI), and it is no
// chest nor grand boss.
func (s *Server) hasRandomAnimation(n *Npc) bool {
	if !n.randomAnimation {
		return false
	}
	switch n.Template.Type {
	case "Chest", "GrandBoss":
		return false
	}
	if n.class != nil && n.class.attackable {
		return s.animation.maxMonster > 0
	}
	return s.animation.maxNpc > 0
}

// animationWait draws the wait before an NPC's next animation (Mobius
// Rnd.get(min, max) seconds).
func (s *Server) animationWait(n *Npc) time.Duration {
	f := &s.animation
	lo, hi := f.minNpc, f.maxNpc
	if n.class != nil && n.class.attackable {
		lo, hi = f.minMonster, f.maxMonster
	}
	return time.Duration(s.rnd.Between(int(lo/time.Second), int(hi/time.Second))) * time.Second
}

// animationRegionSwitched is Mobius WorldRegion.switchAI's part: the NPCs of
// a region that turns on are scheduled, those of one that turns off are
// dropped.
func (s *Server) animationRegionSwitched(_ world.RegionKey, active bool, objects []world.Object) {
	f := &s.animation
	if active {
		now := s.Now()
		for _, o := range objects {
			n, ok := o.(*Npc)
			if !ok || !s.hasRandomAnimation(n) {
				continue
			}
			if _, ok := f.due[n]; ok {
				continue // Mobius putIfAbsent
			}
			f.due[n] = now.Add(s.animationWait(n))
			f.pending = append(f.pending, n)
		}
		return
	}
	dropped := false
	for _, o := range objects {
		if n, ok := o.(*Npc); ok {
			if _, ok := f.due[n]; ok {
				delete(f.due, n)
				dropped = true
			}
		}
	}
	if dropped {
		f.pending = slices.DeleteFunc(f.pending, func(n *Npc) bool {
			_, ok := f.due[n]
			return !ok
		})
	}
}

// animationTick is Mobius RandomAnimationTaskManager.run: every NPC whose
// time came plays an animation if it is in an active region, alive, out of
// combat, standing and not stunned, asleep or paralyzed, and draws its next
// time either way.
func (s *Server) animationTick() {
	f := &s.animation
	now := s.Now()
	for _, n := range f.pending {
		if !now.After(f.due[n]) {
			continue
		}
		if s.animationIdle(n) {
			s.npcRandomAnimation(n, s.rnd.Between(animationFirst, animationLast))
		}
		f.due[n] = now.Add(s.animationWait(n))
	}
}

// animationIdle is the check of RandomAnimationTaskManager.run.
func (s *Server) animationIdle(n *Npc) bool {
	if !n.InWorld() || n.Dead || !s.world.RegionActive(n) || n.Move != nil || s.InCombat(n) {
		return false
	}
	return n.skill == nil || !allSkillsDisabled(&n.skill.skillBase)
}

// npcRandomAnimation is Mobius Npc.onRandomAnimation: the players around see
// the social action, unless the NPC played one less than 6 s ago.
func (s *Server) npcRandomAnimation(n *Npc, action int) {
	now := s.Now()
	if !n.lastSocial.IsZero() && now.Sub(n.lastSocial) <= targetSocialInterval {
		return
	}
	n.lastSocial = now
	s.world.Broadcast(n, serverpackets.NewSocialActionPacket(serverpackets.SocialAction{
		ObjectID: int32(n.ID()), //nolint:gosec // object IDs fit in int32
		ActionID: int32(action), //nolint:gosec // 2 or 3
	}), false)
}
