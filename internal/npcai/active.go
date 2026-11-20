package npcai

import (
	"math"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/skills"
)

// RingPoint is the point Mobius picks around a leader for its minions
// (thinkActive's follow, MinionList's spawn and teleport): x =
// Between(2 × minRadius, 2 × radius), then the distance Between(x, 2 ×
// radius) and y = (int) sqrt(distance² − x²); each coordinate lands past
// the center (+ value − radius) when the value is above radius + minRadius,
// before it (− value + minRadius) otherwise. Z is the center's.
func RingPoint(r rnd.Source, center Point, minRadius, radius int) Point {
	x := r.Between(minRadius*2, radius*2)
	y := r.Between(x, radius*2)
	y = int(math.Sqrt(float64(y*y - x*x)))
	return Point{
		X: ringCoord(center.X, x, minRadius, radius),
		Y: ringCoord(center.Y, y, minRadius, radius),
		Z: center.Z,
	}
}

func ringCoord(c int32, v, minRadius, radius int) int32 {
	if v > radius+minRadius {
		return c + int32(v-radius) //nolint:gosec // within twice the radius
	}
	return c - int32(v-minRadius) //nolint:gosec // as above
}

// FollowOffsetOf is how far a minion lets its leader go: 500 for a raid
// minion, 200 for others.
func FollowOffsetOf(npc Npc) int {
	if npc.Rank() == RankRaidMinion {
		return RaidFollowOffset
	}
	return FollowOffset
}

// SpawnPoint is where MinionList puts a minion around its master (spawn,
// respawn and the master's teleport): RingPoint with the master's collision
// radius (getCollisionRadius, the grown one when grown) truncated plus 30 as
// the inner radius and 200 as the radius.
func SpawnPoint(r rnd.Source, master Point, masterCollision float64) Point {
	return RingPoint(r, master, int(masterCollision)+SpawnMinRadius, SpawnRadius)
}

// Follow is thinkActive's branch for a minion whose leader lives (the game
// syncs walking or running with the leader first). Farther than its follow
// offset (2D), it walks to a RingPoint around the leader (inner radius 30)
// with ActMoveTo; otherwise, on Get(30) == 0, it casts a buff on itself
// (Buff). Either way the think ends.
func (t *Thinker) Follow(npc Npc, leader Creature) Decision {
	offset := FollowOffsetOf(npc)
	if dist2D(npc, leader) > float64(offset) {
		return Decision{Action: ActMoveTo, To: RingPoint(t.Rnd, leader.Loc(), FollowMinRadius, offset)}
	}
	if t.Rnd.Get(RandomWalkRate) == 0 {
		return Buff(npc)
	}
	return none
}

// Buff is thinkActive's buff before a walk: the first BUFF skill the NPC
// does not have on itself, with more MP than it takes, and for a
// LIFE_FORCE_OTHERS skill only when hurt, cast on itself (ActCast with the
// NPC as its target; Mobius neither stops nor gives the target back). A
// wandering NPC (not a minion) calls it after its Get(30) == 0 draw and
// walks only on ActNone.
func Buff(npc Npc) Decision {
	if sk := buffSkill(npc); sk != nil {
		return Decision{Action: ActCast, Skill: sk, Target: npc}
	}
	return none
}

func buffSkill(npc Npc) *skills.Skill {
	for _, sk := range npc.AIProfile().Skills(ScopeBuff) {
		if npc.AffectedBySkill(sk.ID) {
			continue
		}
		if npc.CurrentMP() <= float64(sk.MPConsume) {
			continue
		}
		if sk.AbnormalType == data.AbnormalLifeForceOthers && npc.CurrentHP() >= float64(npc.MaxHP()) {
			continue
		}
		return sk
	}
	return nil
}
