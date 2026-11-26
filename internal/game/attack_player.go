package game

import (
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/world"
)

// A player's intentions (Mobius PlayerAI and CreatureAI) as far as the
// auto-attack needs them: IDLE, ATTACK and MOVE_TO. Sitting is Mobius's
// REST, which refuses attacks and walks; talking to an NPC is INTERACT,
// which the targeting code runs (target.go). The names carry a "player"
// prefix: the monster AI ports the same Mobius methods for NPCs.
//
// An attack thinks (playerThinkAttack): a dead target ends it; a target out
// of reach starts the attack follow, which walks the player up to it every
// 500 ms (Mobius CreatureFollowTaskManager) and repaths at most once a
// second (moveToPawn); in reach the player stops and swings. Every swing
// thinks again at its ready time. A move click during a swing is refused,
// kept and replayed at the ready time.

const (
	// attackFollowPeriod is how often the attack follow runs (Mobius
	// CreatureFollowAttackTask).
	attackFollowPeriod = 500 * time.Millisecond
	// attackFollowSlack is how much farther than its reach a followed
	// target may be when the player stops following to swing (Mobius
	// maybeMoveToPawn: "Allow larger hit range when the target is moving").
	attackFollowSlack = 100
	// attackFollowMax ends a follow whose target is farther (3D).
	attackFollowMax = 3000
	// attackMovingTargetOffset is taken off the reach for a target that
	// moves, down to attackMinFollowOffset.
	attackMovingTargetOffset = 100
	attackMinFollowOffset    = 5
	// attackPawnMinOffset is the smallest moveToPawn offset.
	attackPawnMinOffset = 10
	// attackPawnRepathTicks is how long, in game ticks, a walk to the same
	// pawn is not restarted (Mobius moveToPawn: 1,000 / MILLIS_IN_TICK).
	attackPawnRepathTicks = 10
)

// playerIntention is a player's intention (Mobius Intention).
type playerIntention uint8

const (
	playerIdle playerIntention = iota
	playerAttacking
	playerMovingTo
	playerInteracting
	playerCasting // CAST, cast_player.go
)

// playerAI is a player's intention state (Mobius PlayerAI).
type playerAI struct {
	intention playerIntention
	// target is the attack target (getAttackTarget).
	target Creature
	// follow is the creature the attack follow walks to, followRange how
	// close (getFollowTarget); followed is set while the player is in the
	// follow list.
	follow      Creature
	followRange int32
	followed    bool
	// savedMove is a move click refused during a swing (_savedMoveTo).
	savedMove *world.Position
	// thinking blocks a think within a think (_thinking).
	thinking bool
	// pawnOffset, pawnTarget and pawnTimeout are moveToPawn's state
	// (_clientMovingToPawnOffset, _target, _moveToPawnTimeout). pawnWalking
	// is set while moveToPawn starts a walk.
	pawnOffset  int32
	pawnTarget  world.Object
	pawnTimeout int64
	pawnWalking bool
}

// playerAIOf returns p's intention state, made on first use.
func (s *Server) playerAIOf(p *Player) *playerAI {
	ai := s.attack.players[p]
	if ai == nil {
		ai = &playerAI{}
		if p.state != stateLeft {
			s.attack.players[p] = ai
		}
	}
	return ai
}

// AttackIntention makes p attack target (Mobius setIntentionAttack): p walks
// up to it and swings until it dies or p does something else. A sitting
// player gets ActionFailed instead. Loop-only.
func (s *Server) AttackIntention(p *Player, target Creature) { s.playerIntendAttack(p, target) }

// IdleIntention makes p idle (Mobius setIntentionIdle): it stops attacking
// and walking (StopMove when it walked); its stance stays. Loop-only.
func (s *Server) IdleIntention(p *Player) { s.playerIntendIdle(p) }

// AttackTarget returns the creature p attacks, or nil. Loop-only.
func (s *Server) AttackTarget(p *Player) Creature {
	if ai := s.attack.players[p]; ai != nil && ai.intention == playerAttacking {
		return ai.target
	}
	return nil
}

// playerIntendAttack is Mobius CreatureAI.setIntentionAttack.
func (s *Server) playerIntendAttack(p *Player, target Creature) {
	if target == nil || p.Sitting || s.playerCastBusy(p) {
		p.SendActionFailed()
		return
	}
	ai := s.playerAIOf(p)
	ai.intention = playerAttacking
	ai.target = target
	s.playerStopFollow(p)
	s.playerThink(p)
}

// playerIntendIdle is Mobius setIntentionIdle (PlayerAI's
// setIntentionActive too): no follow, no attack target, a walk stops with
// StopMove, and the stance shown stays until the scan ends it.
func (s *Server) playerIntendIdle(p *Player) {
	ai := s.playerAIOf(p)
	s.playerStopFollow(p)
	ai.intention = playerIdle
	ai.target = nil
	s.playerStopMoving(p)
	s.StopAutoAttack(p)
}

// playerIntendMove is PlayerAI.setIntentionMoveTo for a move click to dest.
// It reports whether p may walk there now. During a swing it refuses with
// ActionFailed and keeps the click for the ready time. Otherwise the attack
// ends: no follow, intention MOVE_TO, the stance shown stays.
func (s *Server) playerIntendMove(p *Player, dest world.Position) bool {
	ai := s.playerAIOf(p)
	if s.AttackingNow(p) || s.playerCastBusy(p) {
		p.SendActionFailed()
		ai.savedMove = &dest
		return false
	}
	s.playerStopFollow(p)
	ai.savedMove = nil
	ai.intention = playerMovingTo
	ai.pawnOffset = 0
	s.StopAutoAttack(p)
	return true
}

// playerIntendInteract is the start of Mobius setIntentionInteract, when p
// walks up to an NPC to talk: the attack and its follow end.
func (s *Server) playerIntendInteract(p *Player) {
	ai := s.playerAIOf(p)
	s.playerStopFollow(p)
	s.StopAutoAttack(p)
	ai.intention = playerInteracting
	ai.target = nil
}

// playerIntendRest is PlayerAI.setIntentionRest, when p sits down: the
// saved move and the attack end (the caller stops the walk).
func (s *Server) playerIntendRest(p *Player) {
	ai := s.playerAIOf(p)
	s.playerStopFollow(p)
	ai.savedMove = nil
	ai.intention = playerIdle
	ai.target = nil
	ai.pawnOffset = 0
}

// playerStopMoving is Mobius clientStopMoving(null): a walking player stops
// with StopMove.
func (s *Server) playerStopMoving(p *Player) {
	if p.Move != nil {
		s.StopPlayer(p)
	}
	if ai := s.attack.players[p]; ai != nil {
		ai.pawnOffset = 0
	}
}

// playerThink is PlayerAI.notifyActionThink: ATTACK thinks here, CAST in
// cast_player.go (a cast thinks even inside a think).
func (s *Server) playerThink(p *Player) {
	ai := s.playerAIOf(p)
	if (ai.thinking && ai.intention != playerCasting) || p.state != stateInGame || !p.InWorld() {
		return
	}
	was := ai.thinking
	ai.thinking = true
	defer func() { ai.thinking = was }()
	switch ai.intention {
	case playerAttacking:
		s.playerThinkAttack(p, ai)
	case playerCasting:
		s.playerThinkCast(p, ai)
	}
}

// playerThinkAttack is PlayerAI.thinkAttack.
func (s *Server) playerThinkAttack(p *Player, ai *playerAI) {
	t := ai.target
	if t == nil {
		return
	}
	if t.IsDead() {
		s.playerIntendIdle(p)
		return
	}
	if s.playerMaybeMoveToPawn(p, ai, t, p.stats.PhysicalAttackRange()) {
		return
	}
	s.playerStopMoving(p)
	s.doAttack(p, t)
}

// playerMaybeMoveToPawn is Mobius maybeMoveToPawn: it reports whether p
// must still walk up to t, within offset plus both collision radii (2D),
// and starts the attack follow when it must.
func (s *Server) playerMaybeMoveToPawn(p *Player, ai *playerAI, t Creature, offset int) bool {
	if offset < 0 {
		return false
	}
	reach := offset + attackCollision(p) + attackCollision(t)
	if !inRadius2D(p.Position(), t.Position(), reach) {
		if playerFollowing(ai) {
			if !inRadius2D(p.Position(), t.Position(), reach+attackFollowSlack) {
				return true
			}
			s.playerStopFollow(p)
			return false
		}
		if p.dead || p.Teleporting() || !(p.stats.MoveSpeed() > 0) {
			if ai.intention == playerAttacking {
				s.playerIntendIdle(p)
			}
			return true
		}
		s.playerStopFollow(p)
		off := offset
		if creatureMoving(t) {
			off -= attackMovingTargetOffset
		}
		s.playerStartFollow(p, t, int32(max(off, attackMinFollowOffset))) //nolint:gosec // a weapon's reach
		return true
	}
	if playerFollowing(ai) {
		s.playerStopFollow(p)
	}
	return false
}

// playerFollowing is Mobius isFollowing.
func playerFollowing(ai *playerAI) bool { return ai.follow != nil && ai.followed }

// playerStartFollow starts the attack follow of p to t at rng (Mobius
// startFollow(target, range) and addAttackFollow), following at once.
func (s *Server) playerStartFollow(p *Player, t Creature, rng int32) {
	s.playerStopFollow(p)
	ai := s.playerAIOf(p)
	ai.follow, ai.followRange = t, rng
	if !ai.followed {
		ai.followed = true
		s.attack.following = append(s.attack.following, p)
	}
	s.playerFollow(p)
}

// playerStopFollow ends p's attack follow (Mobius stopFollow).
func (s *Server) playerStopFollow(p *Player) {
	ai := s.attack.players[p]
	if ai == nil {
		return
	}
	ai.follow = nil
	if ai.followed {
		ai.followed = false
		s.attack.following = slices.DeleteFunc(s.attack.following, func(x *Player) bool { return x == p })
	}
}

// playerFollowTick is the attack follow task (every 500 ms).
func (s *Server) playerFollowTick() {
	if len(s.attack.following) == 0 {
		return
	}
	for _, p := range slices.Clone(s.attack.following) {
		if ai := s.attack.players[p]; ai != nil && ai.followed {
			s.playerFollow(p)
		}
	}
}

// playerFollow is CreatureFollowTaskManager.follow for p: out of range p
// walks up to its target (no farther than 3,000); a player that stands
// meanwhile, in range or not, gets an arrival so that the attack goes on.
func (s *Server) playerFollow(p *Player) {
	ai := s.playerAIOf(p)
	if p.state != stateInGame || !p.InWorld() {
		return
	}
	t := ai.follow
	if t == nil {
		s.playerIntendIdle(p)
		return
	}
	if !inRadius3D(p.Position(), t.Position(), int(ai.followRange)) {
		if !inRadius3D(p.Position(), t.Position(), attackFollowMax) {
			s.playerIntendIdle(p)
			return
		}
		s.playerMoveToPawn(p, ai, t, ai.followRange)
		if p.Move == nil && (ai.intention == playerAttacking || ai.intention == playerCasting) {
			s.playerArrived(p, ai)
		}
	} else if p.Move == nil && (ai.intention == playerAttacking || ai.intention == playerCasting) {
		s.playerArrived(p, ai)
	}
}

// playerMoveToPawn is Mobius AbstractAI.moveToPawn: p walks to pawn's
// position, offset away, and p and the players around get MoveToPawn. A
// walk to the same pawn at the same offset is not restarted within a
// second. A player that cannot move or swings gets ActionFailed.
func (s *Server) playerMoveToPawn(p *Player, ai *playerAI, pawn Creature, offset int32) {
	if p.dead || p.Teleporting() || s.AttackingNow(p) {
		p.SendActionFailed()
		return
	}
	offset = max(offset, attackPawnMinOffset)
	tick := gameTick(s.Now())
	if p.Move != nil && ai.pawnTarget == world.Object(pawn) {
		if ai.pawnOffset == offset && tick < ai.pawnTimeout {
			return
		}
		// A route is kept 2 s at least (pathing.go).
		if ai.pawnOffset != offset && s.onRoute(p) && tick < ai.pawnTimeout+attackPawnRepathTicks {
			return
		}
	}
	ai.pawnOffset, ai.pawnTarget, ai.pawnTimeout = offset, pawn, tick+attackPawnRepathTicks
	ai.pawnWalking = true
	s.MovePlayerTo(p, pawn.Position(), offset)
	ai.pawnWalking = false
	if p.state == stateInGame && p.InWorld() && s.broadcastPawnWalk(p, pawn, offset) {
		ai.pawnOffset = 0
	}
}

// attackArrived is the OnPlayerArrived hook: Mobius notifyActionArrived for
// the intentions.
func (s *Server) attackArrived(p *Player) {
	if ai := s.attack.players[p]; ai != nil {
		s.playerArrived(p, ai)
	}
}

// playerArrived is CreatureAI.notifyActionArrived: a walk up to a pawn ends
// with StopMove (a walk that arrived on its own sent it already), MOVE_TO
// becomes IDLE, and the intention thinks.
func (s *Server) playerArrived(p *Player, ai *playerAI) {
	if s.CastingNow(p) || s.CastingSimultaneouslyNow(p) {
		return // Mobius notifyActionArrived ignores arrivals during a cast
	}
	if ai.pawnOffset > 0 {
		if ai.pawnWalking || (p.Move == nil && !s.walkArrivingNow(p)) {
			s.broadcastStop(p)
		}
		ai.pawnOffset = 0
	}
	if ai.intention == playerMovingTo {
		s.playerIntendIdle(p)
	}
	s.playerThink(p)
}

// walkArrivingNow reports whether p's arrival hooks run right now: the walk
// already sent the StopMove of its pawn walk.
func (s *Server) walkArrivingNow(p *Player) bool {
	m := s.movement.players[p]
	return m != nil && m.arriving
}

// playerReadyToAct is PlayerAI.notifyActionReadyToAct: a move kept from a
// swing goes first, then the intention thinks.
func (s *Server) playerReadyToAct(p *Player) {
	ai := s.playerAIOf(p)
	if dest := ai.savedMove; dest != nil {
		ai.savedMove = nil
		switch {
		case p.Sitting:
			p.SendActionFailed()
		case s.playerIntendMove(p, *dest):
			m := s.motionOf(p)
			if s.startWalk(p, m, *dest, 0) {
				s.broadcastWalk(p, m, false)
			}
		}
	}
	if !s.CastingNow(p) && !s.CastingSimultaneouslyNow(p) {
		s.playerThink(p)
	}
}

// attackLeave is the OnPlayerLeave hook: p's swing, intention and stance
// end without packets.
func (s *Server) attackLeave(p *Player) {
	s.playerStopFollow(p)
	delete(s.attack.players, p)
	if st := s.attack.states[p]; st != nil {
		for _, id := range st.timers {
			s.loop.Cancel(id)
		}
		delete(s.attack.states, p)
	}
	s.dropStance(p)
}

// attackTeleport is the OnTeleport hook: Mobius teleToLocation aborts the
// attack and sets the intention ACTIVE (the movement feature stops the
// walk).
func (s *Server) attackTeleport(p *Player) {
	s.AbortAttack(p)
	ai := s.attack.players[p]
	if ai == nil {
		return
	}
	s.playerStopFollow(p)
	ai.intention, ai.target, ai.pawnOffset = playerIdle, nil, 0
	s.StopAutoAttack(p)
}

// attackForget is the OnForget hook (Mobius notifyActionForgetObject): a
// player that lost sight of its attack target drops it, and goes idle
// unless the target is dead; one that lost its follow target stops.
func (s *Server) attackForget(viewer, object world.Object) {
	p, ok := viewer.(*Player)
	if !ok {
		return
	}
	ai := s.attack.players[p]
	if ai == nil {
		return
	}
	if ai.target != nil && world.Object(ai.target) == object {
		ai.target = nil
		if c, ok := object.(Creature); !ok || !c.IsDead() {
			s.playerIntendIdle(p)
		}
	}
	if ai.follow != nil && world.Object(ai.follow) == object {
		s.playerStopMoving(p)
		s.playerStopFollow(p)
		s.playerIntendIdle(p)
	}
}

// attackDie is the OnDie hook: a dead player's intention ends (Mobius
// clientNotifyDead: IDLE, no targets, no follow).
func (s *Server) attackDie(victim, _ Creature) {
	p, ok := victim.(*Player)
	if !ok {
		return
	}
	ai := s.attack.players[p]
	if ai == nil {
		return
	}
	s.playerStopFollow(p)
	ai.intention, ai.target, ai.pawnOffset = playerIdle, nil, 0
}

// inRadius2D is Mobius isInsideRadius2D: within r on the ground plane, edge
// included.
func inRadius2D(a, b world.Position, r int) bool {
	dx, dy := int64(a.X)-int64(b.X), int64(a.Y)-int64(b.Y)
	return dx*dx+dy*dy <= int64(r)*int64(r)
}

// inRadius3D is Mobius isInsideRadius3D.
func inRadius3D(a, b world.Position, r int) bool {
	return a.DistanceSq3D(b) <= int64(r)*int64(r)
}
