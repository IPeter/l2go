package game

import (
	"errors"
	"io/fs"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/world"
)

// Walkers (Plan 13, A13): NPCs that walk the routes of Mobius Routes.xml
// (WalkingManager, WalkInfo, ArrivedTask). An NPC that spawns at the point
// of a route target walks the route: node to node, running or walking as
// the node says, waiting 100 + delay × 1000 ms at each node, saying the
// node's line, then going back, round, back to the start by teleport or to
// random nodes as the route's repeat style says. The node events
// (on_node_arrived, on_route_finished) reach the scripts. The fake-player
// route (FPC_Giran_Evi) has no targets, so it stays off, as in Mobius
// without fake players.
func init() { registerFeature("walker", setupWalkers) }

const (
	// walkerRetry is Mobius startMoving's period: a walker that is busy
	// (fighting) tries to start again after 10 s, and an active walk is
	// checked every 10 s, so that a walk a fight stopped goes on.
	walkerRetry = 10 * time.Second
	// walkerFarStart: a walker farther than this from its first node is
	// teleported there (Mobius startMoving).
	walkerFarStart = 3000
	// walkerNodeRadius: an arrival counts for the node only within this
	// (2D) of it (Mobius onArrived).
	walkerNodeRadius = 10
)

// walkerFeature is the walker state on the Server (Server.walkers).
// Loop-only.
type walkerFeature struct {
	routes *data.Routes
	// targeted holds the NPC IDs some route targets: they never wander
	// (Mobius Npc.onSpawn, isTargeted).
	targeted map[int]bool
}

// npcWalker is an NPC's walker state (Mobius WalkingManager's maps for the
// NPC): the active walk and the pending tasks.
type npcWalker struct {
	// info is the walk under way (Mobius _activeRoutes), nil when none.
	info *walkInfo
	// walker is Mobius isWalker: the NPC has started a walk once.
	walker bool
	// start is the pending 10 s start retry, repeat the walk check every
	// 10 s, arrive the wait at a node (Mobius _startMoveTasks,
	// _repeatMoveTasks, _arriveTasks).
	start, repeat, arrive world.TimerID
}

// walkInfo is Mobius WalkInfo: where a walker is on its route.
type walkInfo struct {
	route *data.WalkRoute
	// cursor is the node the walker walks to; step is +1 or -1 (back).
	cursor, step int
	// blocked: the walker waits at a node; suspended and stoppedByAttack are
	// Mobius's flags (a suspended walk does not go on by itself).
	blocked, suspended, stoppedByAttack bool
}

// node returns the node the walker walks to (Mobius getCurrentNode).
func (w *walkInfo) node() *data.RouteNode {
	nodes := w.route.Nodes
	return &nodes[min(max(w.cursor, 0), len(nodes)-1)]
}

// setupWalkers loads the routes of the data folder and starts the walkers
// already spawned (the npc feature spawns the world first).
func setupWalkers(s *Server) error {
	if s.worldData == nil || s.settings.DataDir == "" {
		return nil
	}
	routes, err := data.LoadRoutes(s.settings.DataDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			s.log.Info("game: no walker routes in the data folder", "dir", s.settings.DataDir)
		} else {
			s.log.Warn("game: walker routes not loaded", "err", err)
		}
		return nil
	}
	s.UseWalkerRoutes(routes)
	return nil
}

// UseWalkerRoutes makes the NPCs walk routes (Mobius WalkingManager.load):
// the NPCs in the world that stand on a target's spawn point start now,
// those that spawn later when they spawn. Loop-only (or before Run).
func (s *Server) UseWalkerRoutes(routes *data.Routes) {
	f := &s.walkers
	f.routes = routes
	f.targeted = make(map[int]bool)
	for i := range routes.Routes {
		for _, t := range routes.Routes[i].Targets {
			f.targeted[t.NpcID] = true
			if c := s.ai.classes[t.NpcID]; c != nil {
				c.randomWalk = false
			}
		}
	}
	s.log.Info("walker routes loaded", "routes", len(routes.Routes), "npcs", len(f.targeted))
	for _, sp := range s.npc.spawns {
		if n := sp.npc; n != nil && n.InWorld() {
			s.walkerSpawned(n)
		}
	}
}

// walkerTargeted reports whether a route targets NPCs of the template
// (Mobius isTargeted).
func (s *Server) walkerTargeted(npcID int) bool { return s.walkers.targeted[npcID] }

// isWalker is Mobius Npc.isWalker.
func (n *Npc) isWalker() bool { return n.walker != nil && n.walker.walker }

// walkerSpawned is Mobius WalkingManager.onSpawn: an NPC whose spawn point
// is a route target's starts that route.
func (s *Server) walkerSpawned(n *Npc) {
	f := &s.walkers
	if f.routes == nil || !f.targeted[n.Template.ID] || n.spawn == nil {
		return
	}
	at := n.spawn.lastPoint
	if route, ok := f.routes.RouteAt(n.Template.ID, int(at.X), int(at.Y), int(at.Z)); ok {
		s.walkerStart(n, route)
	}
}

// walkerIdle is Mobius's "intention ACTIVE or IDLE" of startMoving. An NPC
// the monster AI does not drive is idle while it stands.
func walkerIdle(n *Npc) bool {
	if n.class != nil && n.class.attackable {
		return n.ai.intention == NpcIdle || n.ai.intention == NpcActive
	}
	return n.Move == nil
}

// walkerStart is Mobius WalkingManager.startMoving: a new walk starts at the
// first node (the second if the NPC stands on the first) and is checked
// every 10 s; a busy NPC tries again in 10 s; an active walk goes on to its
// node unless it waits at one or is suspended.
func (s *Server) walkerStart(n *Npc, route *data.WalkRoute) {
	if route == nil || n.Dead || !n.InWorld() {
		return
	}
	w := n.walker
	if w == nil {
		w = &npcWalker{}
		n.walker = w
	}
	if w.info == nil {
		if !walkerIdle(n) {
			if w.start == 0 {
				w.start = s.loop.After(walkerRetry, func() {
					w.start = 0
					s.walkerStart(n, route)
				})
			}
			return
		}
		info := &walkInfo{route: route, step: 1}
		node := info.node()
		pos := n.Position()
		if pos.X == int32(node.X) && pos.Y == int32(node.Y) { //nolint:gosec // world coordinates fit
			s.walkerNext(n, info)
			node = info.node()
		}
		if to := walkerNodePos(node); !(pos.DistanceSq3D(to) < walkerFarStart*walkerFarStart) {
			s.log.Warn("game: a walker is too far from its route; teleporting it there",
				"route", route.Name, "npc", n.Template.ID, "pos", pos, "node", to)
			s.TeleportNpc(n, to)
		}
		s.SetNpcRunning(n, node.Run)
		s.walkerMoveTo(n, node)
		if w.repeat == 0 {
			w.repeat = s.loop.Every(walkerRetry, func() { s.walkerStart(n, route) })
		}
		w.walker = true
		w.info = info
		return
	}
	info := w.info
	if !walkerIdle(n) || info.blocked || info.suspended {
		return
	}
	info.blocked = true
	node := info.node()
	s.SetNpcRunning(n, node.Run)
	s.walkerMoveTo(n, node)
	info.blocked = false
	info.stoppedByAttack = false
}

// walkerNodePos is a node's position.
func walkerNodePos(node *data.RouteNode) world.Position {
	return world.Position{X: int32(node.X), Y: int32(node.Y), Z: int32(node.Z)} //nolint:gosec // world coordinates fit
}

// walkerMoveTo is Mobius setIntentionMoveTo(node): the monster AI's MOVE_TO
// for an attackable, a plain walk for other NPCs.
func (s *Server) walkerMoveTo(n *Npc, node *data.RouteNode) {
	to := walkerNodePos(node)
	if n.class != nil && n.class.attackable {
		s.npcSetMoveTo(n, to)
		return
	}
	s.NpcMoveTo(n, to)
}

// walkerNext is Mobius WalkInfo.calculateNextNode: a random route picks
// another node at random; otherwise the cursor steps on, and past the last
// node the route is finished (on_route_finished) and starts over as its
// repeat style says, or the walk ends.
func (s *Server) walkerNext(n *Npc, info *walkInfo) {
	count := len(info.route.Nodes)
	if info.route.RepeatStyle == data.RepeatRandom {
		if count <= 1 {
			info.cursor = 0
			return
		}
		base := info.cursor
		for info.cursor == base {
			info.cursor = s.rnd.Get(count)
		}
		return
	}
	info.cursor += info.step
	if info.cursor == count {
		s.walkerScriptEvent(n, "route_finished")
		if !info.route.Repeats() {
			s.walkerCancel(n)
			return
		}
		switch info.route.RepeatStyle {
		case data.RepeatBack:
			info.step = -1
			info.cursor = count - 2
		case data.RepeatCycle:
			info.step, info.cursor = 1, 0
		case data.RepeatConveyor:
			if sp := n.spawn; sp != nil {
				s.TeleportNpc(n, sp.lastPoint)
			}
			info.step, info.cursor = 1, 0
		}
		return
	}
	if info.cursor == -1 {
		info.cursor, info.step = 1, 1
	}
}

// walkerCancel is Mobius cancelMoving: the walk ends and is no longer
// checked.
func (s *Server) walkerCancel(n *Npc) {
	w := n.walker
	if w == nil {
		return
	}
	w.info = nil
	if w.repeat != 0 {
		s.loop.Cancel(w.repeat)
		w.repeat = 0
	}
}

// walkerGone is Mobius WalkingManager.onDeath, run when the NPC leaves the
// world (Npc.onDecay): the walk ends. Unlike Mobius, the pending start and
// node tasks end too, so that nothing walks an NPC that is not there.
func (s *Server) walkerGone(n *Npc) {
	w := n.walker
	if w == nil {
		return
	}
	s.walkerCancel(n)
	for _, id := range []*world.TimerID{&w.start, &w.arrive} {
		if *id != 0 {
			s.loop.Cancel(*id)
			*id = 0
		}
	}
}

// walkerArrived is Mobius WalkingManager.onArrived, which notifyActionArrived
// calls for every NPC: at its node (within 10) the walker picks the next
// node, says the node's line and walks on after the node's wait.
func (s *Server) walkerArrived(n *Npc) {
	w := n.walker
	if w == nil || w.info == nil || !n.InWorld() {
		return
	}
	s.walkerScriptEvent(n, "node_arrived")
	info := w.info
	nodes := info.route.Nodes
	if info.cursor < 0 || info.cursor >= len(nodes) {
		return
	}
	node := &nodes[info.cursor]
	if !within2D(n.Position(), walkerNodePos(node), walkerNodeRadius) {
		return
	}
	s.walkerNext(n, info)
	info.blocked = true
	if node.Chat != "" {
		s.npcBroadcastSay(n, node.Chat)
	}
	if w.arrive == 0 {
		w.arrive = s.loop.After(node.Wait(), func() {
			w.arrive = 0
			info.blocked = false
			s.walkerStart(n, info.route)
		})
	}
}

// walkerScriptEvent runs the scripts' on_node_arrived(npc) or
// on_route_finished(npc) when one listens for the NPC's template (Mobius
// OnNpcMoveNodeArrived, OnNpcMoveRouteFinished).
func (s *Server) walkerScriptEvent(n *Npc, event string) {
	if s.scripts == nil || !s.scripts.Listens(event, n.Template.ID) {
		return
	}
	ctx, ref := s.background(), n.ScriptRef()
	switch event {
	case "node_arrived":
		_ = s.scripts.NodeArrived(ctx, ref) // failures are logged
	case "route_finished":
		_ = s.scripts.RouteFinished(ctx, ref)
	}
}

// npcBroadcastSay is Mobius npc.broadcastSay(NPC_GENERAL, text).
func (s *Server) npcBroadcastSay(n *Npc, text string) {
	s.world.Broadcast(n, serverpackets.NewNpcSayPacket(serverpackets.NpcSay{
		ObjectID: int32(n.ID()),                //nolint:gosec // object IDs fit in int32
		Type:     serverpackets.ChatGeneral,
		NpcID:    int32(n.Template.DisplayID), //nolint:gosec // NPC IDs fit in int32
		Text:     text,
	}), false)
}
