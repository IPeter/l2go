package game

import (
	"errors"
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/world"
)

const (
	// targetMaxHeightDiff is how far above or below a player its target may
	// be (Mobius Player.setTarget: "prevents /target exploiting").
	targetMaxHeightDiff = 3000
	// targetTalkTurnDistance is the MoveToPawn distance Mobius sends a
	// player that talks to an NPC, which turns it to the NPC.
	targetTalkTurnDistance = 100
	// targetApproachOffset is how far from an NPC a player walking up to it
	// stops (Mobius PlayerAI.thinkInteract: maybeMoveToPawn(target, 36)).
	targetApproachOffset = 36
	// targetApproachTimeout gives up a walk up to an NPC that never ends.
	targetApproachTimeout = time.Minute
	// targetSocialInterval is the shortest time between two random
	// animations of an NPC (Mobius Npc.MINIMUM_SOCIAL_INTERVAL).
	targetSocialInterval = 6 * time.Second
	// targetTalkAnimations is the number of random animations an NPC picks
	// from when talked to (Mobius onRandomAnimation(Rnd.get(8))).
	targetTalkAnimations = 8
)

// npcApproach is a player walking up to an NPC to talk to it (Mobius
// INTERACT intention).
type npcApproach struct {
	npc *Npc
	// dest is where the walk goes; another destination means the player
	// walks elsewhere. moving is false when the walk had no MoveState.
	dest    world.Position
	moving  bool
	started time.Time
}

// setupTargeting registers the click, attack and target cancel packets and
// the hooks that drop targets.
func (s *Server) setupTargeting() error {
	if err := errors.Join(
		s.Register(clientpackets.OpAction, s.handleAction),
		s.Register(clientpackets.OpAttackRequest, s.handleAttackRequest),
		s.Register(clientpackets.OpRequestTargetCanceld, s.handleTargetCanceld),
	); err != nil {
		return err
	}
	s.OnForget(s.forgetTarget)
	// Mobius teleToLocation drops the target before the player leaves.
	s.OnTeleport(func(p *Player, _ world.Position) { s.SetTarget(p, nil) })
	s.OnPlayerLeave(s.dropTargetOnLeave)
	s.OnTick(s.checkApproaches)
	return nil
}

// handleAction handles Action (0x04), a click on an object (Mobius Action):
// a plain click targets the object, or interacts with the current target;
// a Shift-click shows a GM the object's info and is a plain click without
// interaction for other players. Every click ends with ActionFailed.
func (s *Server) handleAction(p *Player, body []byte) {
	req, err := clientpackets.ParseAction(body)
	if err != nil {
		s.log.Debug("game: bad Action", append(p.logArgs(), "err", err)...)
		return
	}
	obj := s.actionObject(p, req.ObjectID)
	if obj == nil {
		p.SendActionFailed()
		return
	}
	s.onActionRequest(p)
	switch req.ActionID {
	case clientpackets.ActionClick:
		s.onAction(p, obj, true)
	case clientpackets.ActionShiftClick:
		if p.IsGM() {
			s.onActionShift(p, obj)
		} else {
			s.onAction(p, obj, false)
		}
	default:
		s.log.Warn("game: Action with an invalid action ID", append(p.logArgs(), "actionId", req.ActionID)...)
		p.SendActionFailed()
	}
}

// handleAttackRequest handles AttackRequest (0x0A). Mobius treats it as a
// click when the object is not the target yet, and as a forced attack on
// the target (onForcedAttack), except on the player itself, which gets
// ActionFailed (private stores and requests do not exist yet).
func (s *Server) handleAttackRequest(p *Player, body []byte) {
	req, err := clientpackets.ParseAttackRequest(body)
	if err != nil {
		s.log.Debug("game: bad AttackRequest", append(p.logArgs(), "err", err)...)
		return
	}
	obj := s.actionObject(p, req.ObjectID)
	if obj == nil {
		p.SendActionFailed()
		return
	}
	s.onActionRequest(p)
	if p.Target != obj {
		s.onAction(p, obj, true)
		return
	}
	if obj.ID() == p.ID() {
		p.SendActionFailed()
		return
	}
	s.onForcedAttack(p, obj)
}

// handleTargetCanceld handles RequestTargetCanceld (0x37): the player drops
// its target, or with Unselect 0 aborts a cast first (castTargetCanceled).
func (s *Server) handleTargetCanceld(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestTargetCanceld(body)
	if err != nil {
		s.log.Debug("game: bad RequestTargetCanceld", append(p.logArgs(), "err", err)...)
		return
	}
	s.castTargetCanceled(p, req.Unselect)
}

// actionObject returns the object a click of p names, with the checks of
// Mobius Action: it exists, a non-GM may target it, it is in p's instance
// and p sees it. It returns nil otherwise, and while p is out of the world.
func (s *Server) actionObject(p *Player, id int32) world.Object {
	if !p.InWorld() {
		return nil
	}
	var obj world.Object
	if t := p.Target; t != nil && int32(t.ID()) == id {
		obj = t
	} else {
		o, ok := s.world.Find(world.ObjectID(id))
		if !ok {
			return nil
		}
		obj = o
	}
	if n, ok := obj.(*Npc); ok && !n.Template.Status.Targetable && !p.IsGM() {
		return nil
	}
	if targetInstance(obj) != p.InstanceID() || !s.world.CanSee(p, obj) {
		return nil
	}
	return obj
}

// targetInstance returns the instance of an object.
func targetInstance(o world.Object) int32 {
	if b, ok := o.(interface{ InstanceID() int32 }); ok {
		return b.InstanceID()
	}
	return 0
}

// onAction runs the click handler of the object's kind, then sends
// ActionFailed (Mobius WorldObject.onAction).
func (s *Server) onAction(p *Player, obj world.Object, interact bool) {
	switch o := obj.(type) {
	case *Npc:
		s.npcClick(p, o, interact)
	case *Player:
		s.playerClick(p, o, interact)
	case *GroundItem:
		s.itemClick(p, o)
	}
	p.SendActionFailed()
}

// onActionShift handles a GM's Shift-click (Mobius onActionShift): an NPC
// becomes the target and its info page opens; a player becomes the target
// (its character info is a GM command of a later milestone).
func (s *Server) onActionShift(p *Player, obj world.Object) {
	switch o := obj.(type) {
	case *Npc:
		s.SetTarget(p, o)
		s.sendNpcInfoPage(p, o)
	case *Player:
		if p.Target != obj {
			s.SetTarget(p, o)
		}
	}
	p.SendActionFailed()
}

// npcClick is Mobius NpcClick: the first click targets the NPC; a click on
// the target attacks it when it is attackable and alive, and talks to it
// when it is not attackable, walking up to it first when it is out of
// reach. Every click makes the NPC the player's last NPC (Mobius
// setLastFolkNPC).
func (s *Server) npcClick(p *Player, n *Npc, interact bool) {
	p.LastNpc = n.ID()
	if p.Target != world.Object(n) {
		s.SetTarget(p, n)
		return
	}
	if !interact {
		return
	}
	if n.autoAttackableBy(p) {
		if !n.IsDead() {
			s.playerIntendAttack(p, n)
		}
		return
	}
	if !s.npcCanInteract(p, n) {
		s.approachNpc(p, n)
		return
	}
	s.talkToNpc(p, n)
}

// playerClick is Mobius PlayerClick: the first click targets the player. A
// click on the target attacks it when p may attack it without Ctrl;
// otherwise p gets ActionFailed (following comes later; private stores and
// cursed weapons do not exist yet).
func (s *Server) playerClick(p, target *Player, interact bool) {
	if p.Target != world.Object(target) {
		s.SetTarget(p, target)
		return
	}
	if !interact {
		return
	}
	if target.autoAttackableBy(p) {
		s.playerIntendAttack(p, target)
		s.onActionRequest(p)
		return
	}
	p.SendActionFailed()
}

// npcCanInteract reports whether p may talk to n now (Mobius
// Npc.canInteract): it does not sit, is in n's instance and within the
// interaction range.
func (s *Server) npcCanInteract(p *Player, n *Npc) bool {
	return !p.Sitting && p.InstanceID() == n.InstanceID() &&
		p.Position().InRange3D(n.Position(), s.npcInteractionRange())
}

// npcInteractionRange is Settings.NPCs.InteractionRange, or Mobius's 250.
func (s *Server) npcInteractionRange() int32 {
	if r := s.settings.NPCs.InteractionRange; r > 0 {
		return int32(min(r, 1<<20)) //nolint:gosec // clamped
	}
	return config.DefaultNPCInteractionRange
}

// talkToNpc is the end of Mobius NpcClick in range. A walking player stops
// first: Mobius sets the AI active, which ends any walk, a walk up to an NPC
// included, with StopMove to the player and the players around
// (StopPlayer). Then MoveToPawn turns the player to the NPC, the NPC may
// play a random animation, and the interaction hooks open its dialog.
func (s *Server) talkToNpc(p *Player, n *Npc) {
	delete(s.npc.approaches, p)
	if p.Move != nil {
		s.playerIntendIdle(p) // stops the walk (StopPlayer) and any attack
	}
	p.Send(npcMoveToPawnPacket(p, n, targetTalkTurnDistance))
	s.npcTalkAnimation(n)
	s.runNpcInteract(p, n)
}

// npcTalkAnimation plays a random animation of n for the players around, at
// most every 6 seconds (Mobius Npc.onRandomAnimation).
func (s *Server) npcTalkAnimation(n *Npc) {
	if !n.randomAnimation || s.npc.rnd == nil {
		return
	}
	now := s.Now()
	if !n.lastSocial.IsZero() && now.Sub(n.lastSocial) <= targetSocialInterval {
		return
	}
	n.lastSocial = now
	s.world.Broadcast(n, serverpackets.NewSocialActionPacket(serverpackets.SocialAction{
		ObjectID: int32(n.ID()),
		ActionID: int32(s.npc.rnd.IntN(targetTalkAnimations)), //nolint:gosec // below 8
	}), false)
}

// approachNpc walks p up to n to talk to it (Mobius INTERACT intention and
// maybeMoveToPawn): the movement feature moves p, the players around see
// MoveToPawn, and once p reached n a tick runs the talk (checkApproaches).
// A sitting player does not move; a refused move does nothing.
func (s *Server) approachNpc(p *Player, n *Npc) {
	s.npc.ensureMaps()
	delete(s.npc.approaches, p)
	if p.Sitting {
		return
	}
	s.playerIntendInteract(p) // the INTERACT intention ends an attack
	if !s.MovePlayerTo(p, n.Position(), targetApproachOffset) {
		return
	}
	a := &npcApproach{npc: n, started: s.Now()}
	if m := p.Move; m != nil {
		a.dest, a.moving = m.Destination, true
	}
	s.npc.approaches[p] = a
	s.world.Broadcast(p, npcMoveToPawnPacket(p, n, targetApproachOffset), true)
}

// checkApproaches is the tick hook of the walks up to NPCs: a player that
// reached its NPC, or stopped, talks to it if it can (Mobius thinkInteract
// and doInteract); one that walks elsewhere, lost the NPC as its target or
// left gives up.
func (s *Server) checkApproaches(now time.Time) {
	for p, a := range s.npc.approaches {
		n := a.npc
		switch {
		case !p.InGame() || !p.InWorld() || p.Target != world.Object(n) || !n.InWorld():
			delete(s.npc.approaches, p)
		case s.npcReached(p, n), p.Move == nil:
			delete(s.npc.approaches, p)
			s.arriveAtNpc(p, n)
		case !a.moving || p.Move.Destination != a.dest, now.Sub(a.started) > targetApproachTimeout:
			delete(s.npc.approaches, p)
		}
	}
}

// npcReached reports whether p came close enough to n to talk (Mobius
// maybeMoveToPawn: within the offset plus both collision radii, in 2D).
func (s *Server) npcReached(p *Player, n *Npc) bool {
	reach := float64(targetApproachOffset) + float64(int(p.lastView.CollisionRadius)) + float64(int(n.statView().CollisionRadius))
	return p.Position().Distance2D(n.Position()) <= reach
}

// arriveAtNpc ends a walk up to an NPC: Mobius doInteract clicks the target
// again, which talks when the player is in range. Out of range the walk is
// given up instead of starting another one.
func (s *Server) arriveAtNpc(p *Player, n *Npc) {
	p.LastNpc = n.ID()
	if !n.autoAttackableBy(p) && s.npcCanInteract(p, n) {
		s.talkToNpc(p, n)
	}
	p.SendActionFailed()
}

// npcMoveToPawnPacket is MoveToPawn of mover walking up to target until it is
// distance away.
func npcMoveToPawnPacket(mover, target world.Object, distance int32) []byte {
	m, t := mover.Position(), target.Position()
	return serverpackets.NewMoveToPawnPacket(serverpackets.MoveToPawn{
		ObjectID: int32(mover.ID()),
		TargetID: int32(target.ID()),
		Distance: distance,
		X:        m.X, Y: m.Y, Z: m.Z,
		TargetX: t.X, TargetY: t.Y, TargetZ: t.Z,
	})
}

// SetTarget makes o the target of p, or drops p's target when o is nil
// (Mobius Player.setTarget). A target more than 3000 above or below p, or
// not in the world, counts as nil. Choosing the current target again only
// sends its ValidateLocation. A new NPC or player target gets p its
// ValidateLocation, MyTargetSelected (coloured by the level difference
// when p can attack it) and its HP bar, while the players around get
// TargetSelected; p keeps getting the HP bar while the target stays.
// Dropping the target sends TargetUnselected to p and the players around.
// Loop-only.
func (s *Server) SetTarget(p *Player, o world.Object) {
	if o != nil && !s.canTarget(p, o) {
		o = nil
	}
	old := p.Target
	if old != nil {
		if old == o {
			if o.ID() != p.ID() {
				p.Send(validateLocation(o))
			}
			return
		}
		s.unwatch(old, p)
	}
	if o != nil && targetIsCreature(o) {
		if o.ID() != p.ID() {
			p.Send(validateLocation(o))
		}
		p.Send(serverpackets.NewMyTargetSelectedPacket(serverpackets.MyTargetSelected{
			ObjectID: int32(o.ID()),
			Color:    targetColor(p, o),
		}))
		s.watch(o, p)
		if bar := targetHPBar(o); bar != nil {
			p.Send(bar)
		}
		pos := p.Position()
		s.world.Broadcast(p, serverpackets.NewTargetSelectedPacket(serverpackets.TargetSelected{
			ObjectID: p.ObjectID(),
			TargetID: int32(o.ID()),
			X:        pos.X, Y: pos.Y, Z: pos.Z,
		}), false)
	}
	if o == nil && old != nil {
		pos := p.Position()
		s.world.Broadcast(p, serverpackets.NewTargetUnselectedPacket(serverpackets.TargetUnselected{
			ObjectID: p.ObjectID(),
			X:        pos.X, Y: pos.Y, Z: pos.Z,
		}), true)
	}
	p.Target = o
	if a := s.npc.approaches[p]; a != nil && world.Object(a.npc) != o {
		delete(s.npc.approaches, p)
	}
}

// canTarget reports whether p may have o as its target (Mobius
// setTarget's checks without parties and vehicles).
func (s *Server) canTarget(p *Player, o world.Object) bool {
	if targetAbsDiff(o.Position().Z, p.Position().Z) > targetMaxHeightDiff {
		return false
	}
	if b, ok := o.(interface{ InWorld() bool }); ok && !b.InWorld() {
		return false
	}
	return true
}

func targetAbsDiff(a, b int32) int64 {
	d := int64(a) - int64(b)
	if d < 0 {
		return -d
	}
	return d
}

// targetIsCreature reports whether o is a creature: an NPC or a player.
func targetIsCreature(o world.Object) bool {
	switch o.(type) {
	case *Npc, *Player:
		return true
	}
	return false
}

// targetColor is MyTargetSelected's colour: p's level minus the target's
// when p can attack the target with a click, else 0.
func targetColor(p *Player, o world.Object) int16 {
	switch t := o.(type) {
	case *Npc:
		if t.autoAttackableBy(p) {
			return int16(p.Level() - t.Level()) //nolint:gosec // levels are small
		}
	case *Player:
		if t.autoAttackableBy(p) {
			return int16(p.Level() - t.Level()) //nolint:gosec // levels are small
		}
	}
	return 0
}

// targetHPBar is the HP bar of a creature: StatusUpdate with its maximum
// and current HP (cut to an int), as Mobius sends a new target's.
func targetHPBar(o world.Object) []byte {
	var maxHP int32
	var cur float64
	switch t := o.(type) {
	case *Npc:
		maxHP, cur = t.MaxHP(), t.CurHP
	case *Player:
		maxHP, cur = t.Stats().View().MaxHP, t.CurHP
	default:
		return nil
	}
	return serverpackets.NewStatusUpdatePacket(serverpackets.StatusUpdate{
		ObjectID: int32(o.ID()),
		Values: []serverpackets.StatusValue{
			{Attr: serverpackets.StatusMaxHP, Value: maxHP},
			{Attr: serverpackets.StatusCurHP, Value: int32(cur)},
		},
	})
}

// BroadcastHP sends o's HP bar to the players targeting it (Mobius
// broadcastStatusUpdate). An NPC does it itself in SetCurrentHP; call it
// after changing the HP of a player. Loop-only.
func (s *Server) BroadcastHP(o world.Object) { s.sendHPBar(o) }

// sendHPBar sends o's HP bar to the players targeting it.
func (s *Server) sendHPBar(o world.Object) {
	list := s.npc.watchers[o.ID()]
	if len(list) == 0 {
		return
	}
	bar := targetHPBar(o)
	if bar == nil {
		return
	}
	for _, p := range list {
		p.Send(bar)
	}
}

// watch adds p to the players that get o's HP bar (Mobius
// addStatusListener).
func (s *Server) watch(o world.Object, p *Player) {
	s.npc.ensureMaps()
	id := o.ID()
	if !slices.Contains(s.npc.watchers[id], p) {
		s.npc.watchers[id] = append(s.npc.watchers[id], p)
	}
}

// unwatch removes p from the players that get o's HP bar.
func (s *Server) unwatch(o world.Object, p *Player) {
	id := o.ID()
	list := slices.DeleteFunc(s.npc.watchers[id], func(x *Player) bool { return x == p })
	if len(list) == 0 {
		delete(s.npc.watchers, id)
		return
	}
	s.npc.watchers[id] = list
}

// forgetTarget is the OnForget hook: a player that lost sight of its target
// drops it (Mobius removeVisibleObject and switchRegion call setTarget(null)).
func (s *Server) forgetTarget(viewer, object world.Object) {
	if p, ok := viewer.(*Player); ok && p.Target != nil && p.Target == object {
		s.SetTarget(p, nil)
	}
}

// dropTargetOnLeave is the OnPlayerLeave hook: the leaving player drops its
// target and its walk up to an NPC without packets; the players targeting
// it drop it when it leaves their sight.
func (s *Server) dropTargetOnLeave(p *Player) {
	delete(s.npc.approaches, p)
	if t := p.Target; t != nil {
		s.unwatch(t, p)
		p.Target = nil
	}
}
