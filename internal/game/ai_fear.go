package game

import (
	"math"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/world"
)

// Fear (Plan 13, decision A5): Mobius AttackableAI.notifyActionAfraid and
// CreatureAI.notifyActionAfraid. A Fear effect tells the AI on its start and
// on each of its ticks, and the NPC runs 500 away from the caster. A RunAway
// effect sets a fear time instead: the NPC then runs every 5 s
// (FEAR_TICKS) until the time is used up, shown with the TURN_FLEE effect,
// and turns IDLE at the end.

const (
	// fearTicks is FEAR_TICKS: the period of the fear task in seconds, and
	// what each run takes off the fear time.
	fearTicks = 5
	// fearRange is how far a frightened creature runs (FEAR_RANGE).
	fearRange = 500
	// headingPerDegree converts a client heading to degrees
	// (LocationUtil.convertHeadingToDegree).
	headingPerDegree = 182.044444444
)

// npcAfraid is Mobius AttackableAI.notifyActionAfraid(effector, start).
func (s *Server) npcAfraid(n *Npc, effector Creature, start bool) {
	if n.skill == nil || n.class == nil || !n.class.attackable {
		return
	}
	a, x := n.skill, &n.ai.x
	if a.fearTime > 0 && x.fearTimer == 0 {
		var id world.TimerID
		run := func() {
			if x.fearTimer != id || effector == nil {
				return
			}
			if !n.InWorld() {
				s.stopFear(n)
				return
			}
			a.fearTime -= fearTicks
			s.npcAfraid(n, effector, start)
		}
		id = s.loop.Every(fearTicks*time.Second, run)
		x.fearTimer = id
		// scheduleAtFixedRate(task, 0, 5000): the first run comes at once.
		s.loop.After(0, run)
		s.setNpcVisual(n, data.VisualTurnFlee, true)
		return
	}
	s.npcFlee(n, effector, start)
	if (n.Dead || a.fearTime <= 0) && x.fearTimer != 0 {
		s.stopFear(n)
		s.npcSetIdle(n)
	}
}

// stopFear cancels the fear task and its visual effect.
func (s *Server) stopFear(n *Npc) {
	x := &n.ai.x
	if x.fearTimer == 0 {
		return
	}
	s.loop.Cancel(x.fearTimer)
	x.fearTimer = 0
	s.setNpcVisual(n, data.VisualTurnFlee, false)
}

// npcFlee is Mobius CreatureAI.notifyActionAfraid: the NPC runs fearRange
// away, from the effector when the fear starts, else straight ahead,
// stopping at the first obstacle with pathfinding on. An NPC that casts or
// cannot act does not move (setIntentionMoveTo).
func (s *Server) npcFlee(n *Npc, effector Creature, start bool) {
	if !canHandleAction(n) || effector == nil {
		return
	}
	pos := n.Position()
	var degrees float64
	if start {
		degrees = angleFrom(effector.Position(), pos)
	} else {
		degrees = float64(pos.Heading) / headingPerDegree
	}
	rad := degrees * math.Pi / 180
	to := world.Position{
		X: int32(float64(pos.X) + fearRange*math.Cos(rad)),
		Y: int32(float64(pos.Y) + fearRange*math.Sin(rad)),
		Z: pos.Z,
	}
	s.SetNpcRunning(n, true)
	if s.gs.Geodata.Pathfinding {
		to = s.validLocation(pos, to)
	}
	if (n.skill != nil && allSkillsDisabled(&n.skill.skillBase)) || s.CastingNow(n) {
		return
	}
	s.npcSetMoveTo(n, to)
}

// angleFrom is LocationUtil.calculateAngleFrom: the direction from a to b in
// degrees, 0 to 360.
func angleFrom(a, b world.Position) float64 {
	deg := math.Atan2(float64(b.Y)-float64(a.Y), float64(b.X)-float64(a.X)) * 180 / math.Pi
	if deg < 0 {
		deg += 360
	}
	return deg
}

// setNpcVisual starts or stops an abnormal visual effect on n (Mobius
// startAbnormalVisualEffect, stopAbnormalVisualEffect) and shows it.
func (s *Server) setNpcVisual(n *Npc, ave data.AbnormalVisualEffect, on bool) {
	if n.skill == nil {
		return
	}
	adm := &n.skill.admin
	before := adm.aves
	if on {
		adm.aves |= 1 << ave
	} else {
		adm.aves &^= 1 << ave
	}
	if adm.aves != before && n.InWorld() {
		n.BroadcastInfo()
	}
}
