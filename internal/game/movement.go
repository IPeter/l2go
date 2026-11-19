package game

import (
	"errors"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/world"
)

// The movement feature (movement*.go): walks, the client's position
// reports, rotation, sitting and the walk/run toggle.
//
// A walk is data (decision W7): Player.Move holds where it goes, and the
// feature's walkTrack holds Mobius's MoveData bookkeeping. On every game
// tick of 100 ms each walking player advances speed × Δticks / 10 units
// with Mobius's formula (walkTrack.step, as MovementTaskManager and
// Creature.updatePosition do), through MoveObject: region changes send the
// visibility packets and the OnPlayerMoved hooks run. The client's
// ValidatePosition corrects the server the way Mobius's does.
//
// movementFeature also holds the state of the chat (chat.go) and GM
// command (gm.go) features.
func init() { registerFeature("movement", setupMovement) }

// movementFeature is the movement, chat and GM-command feature's state on
// the Server (Server.movement). Loop-only.
type movementFeature struct {
	// players holds the movement state of the players in the game.
	players map[*Player]*motion
	// walking lists the players whose walk the tick advances, in the order
	// they started; the tick drops those that stopped.
	walking []*Player
	batch   []*Player // the tick's copy of walking
	// arrived holds the OnPlayerArrived hooks.
	arrived []func(p *Player)
	// actions holds the RequestActionUse handlers by action ID.
	actions map[int32]ActionUse
	// chatRegion returns the map region of a position, which shout and
	// trade chat stay in; nil means chatMapRegion. Tests replace it.
	chatRegion func(pos world.Position) int
	// maxOnline is the most players that were in the game at once.
	maxOnline int
}

// motion is the movement feature's state of one player in the game.
type motion struct {
	// walk follows the walk in progress; it belongs to Player.Move while
	// walk.state is Player.Move.
	walk walkTrack
	// listed is set while the player is in movementFeature.walking.
	listed bool
	// keyboard is set while the player moves with the arrow keys (Mobius
	// cursorKeyMovement): ValidatePosition then takes the client's position.
	keyboard bool
	// pawnStop is set for a walk with an offset, which Mobius makes only to
	// walk up to an object (AbstractAI.moveToPawn): its arrival also sends
	// StopMove (clientStoppedMoving).
	pawnStop bool
	// lastServer is the last position checked against the client's (Mobius
	// lastServerPosition, whose heading stays 0): where an impossible
	// height, or a walk out of the world, puts the player back.
	lastServer world.Position
	// clientZ is the height of the last ValidatePosition (Mobius clientZ).
	clientZ int32
	// sitBusy is set while the animation of sitting down or standing up
	// plays (Mobius sittingInProgress).
	sitBusy bool
	// sitOnArrival makes the player sit down when its walk arrives: a sit
	// request while walking (Mobius NextAction on ARRIVED).
	sitOnArrival bool
	// arriving is set while the arrival hooks run, so that a hook that
	// walks the player to where it stands does not run them again.
	arriving bool
	// nextGlobalChat is the game tick from which the player may shout or
	// trade-chat again in the global chat mode (Mobius flood protector).
	nextGlobalChat int64
}

// setupMovement registers the movement packets, the sit/stand and walk/run
// actions, the tick that advances the walks and the hooks.
func setupMovement(s *Server) error {
	s.OnTick(s.walkTick)
	s.OnPlayerEnter(s.movementEnter)
	s.OnPlayerLeave(s.movementLeave)
	s.OnTeleport(s.movementTeleport)
	return errors.Join(
		s.Register(clientpackets.OpMoveToLocation, s.handleMoveToLocation),
		s.Register(clientpackets.OpValidatePosition, s.handleValidatePosition),
		s.Register(clientpackets.OpCannotMoveAnymore, s.handleCannotMoveAnymore),
		s.Register(clientpackets.OpStartRotating, s.handleStartRotating),
		s.Register(clientpackets.OpFinishRotating, s.handleFinishRotating),
		s.Register(clientpackets.OpRequestActionUse, s.handleRequestActionUse),
		s.Register(opChangeMoveType2, s.handleChangeMoveType2),
		s.Register(opChangeWaitType2, s.handleChangeWaitType2),
		s.RegisterActionUse(clientpackets.ActionUseSitStand, s.actionSitStand),
		s.RegisterActionUse(clientpackets.ActionUseWalkRun, s.actionWalkRun),
	)
}

// motionOf returns the movement state of p, made on first use. A player
// that left gets a state that is thrown away.
func (s *Server) motionOf(p *Player) *motion {
	mv := &s.movement
	if m := mv.players[p]; m != nil {
		return m
	}
	m := &motion{lastServer: lastServerPosition(p.Position())}
	if p.state == stateLeft {
		return m
	}
	if mv.players == nil {
		mv.players = make(map[*Player]*motion)
	}
	mv.players[p] = m
	return m
}

// lastServerPosition is pos as Mobius keeps lastServerPosition: without
// the heading.
func lastServerPosition(pos world.Position) world.Position {
	return world.Position{X: pos.X, Y: pos.Y, Z: pos.Z}
}

// movementEnter is the OnPlayerEnter hook: the spawn point is the first
// last server position (Mobius sets it when the character loads).
func (s *Server) movementEnter(p *Player) {
	s.motionOf(p).lastServer = lastServerPosition(p.Position())
	s.movement.maxOnline = max(s.movement.maxOnline, s.PlayerCount())
}

// movementLeave is the OnPlayerLeave hook: the player's movement state
// goes; the core clears its walk and the tick drops it from the list. The
// position is where the last tick left it, which is what Mobius saves.
func (s *Server) movementLeave(p *Player) { delete(s.movement.players, p) }

// movementTeleport is the OnTeleport hook: the walk ends where p stands
// (Mobius teleToLocation calls stopMove; its StopMove packet is left out,
// as TeleportToLocation and DeleteObject follow at once), and the
// destination becomes the last server position (Mobius onTeleported).
func (s *Server) movementTeleport(p *Player, to world.Position) {
	m := s.motionOf(p)
	p.Move = nil
	m.pawnStop = false
	m.lastServer = lastServerPosition(to)
}

// MovePlayerTo starts walking p towards dest the way a click does, stopping
// offset units short of it: Mobius moveToLocation with an offset, used to
// walk up to an NPC (an offset of 0 walks all the way). Mobius walks to
// offset - 5 units from dest (a bit closer, against rounding), and the
// height difference counts against the offset.
//
// Like Mobius's moveToLocation it sends nothing about the walk: the caller
// shows it, with MoveToPawn for an NPC (Mobius moveToPawn) or BroadcastMove
// for a place (Mobius moveTo). A walk with an offset gets StopMove when it
// arrives, as after Mobius's moveToPawn. OnPlayerArrived hooks run on
// arrival.
//
// It reports whether a walk started. It returns false, after sending
// ActionFailed, when p cannot walk (it sits, teleports or is not in the
// game), and when p already stands within offset of dest; then the
// OnPlayerArrived hooks have run, as Mobius notifies the AI. Loop-only.
func (s *Server) MovePlayerTo(p *Player, dest world.Position, offset int32) bool {
	if p.state != stateInGame {
		return false
	}
	m := s.motionOf(p)
	if !s.startWalk(p, m, dest, max(offset, 0)) {
		return false
	}
	m.pawnStop = offset > 0
	return true
}

// BroadcastMove shows p's walk to p and the players around: MoveToLocation
// from where p stands to its destination (Mobius broadcastMoveToLocation,
// forced). It does nothing while p stands. Loop-only.
func (s *Server) BroadcastMove(p *Player) {
	if p.state == stateInGame {
		s.broadcastWalk(p, s.motionOf(p), true)
	}
}

// StopPlayer ends p's walk where p stands and sends StopMove to p and the
// players around (Mobius Creature.stopMove(null), which NpcClick calls when
// a player talks to an NPC; it sends StopMove also when p stood still). A
// player out of the world (teleporting) only loses its walk. Loop-only.
func (s *Server) StopPlayer(p *Player) {
	if p.state != stateInGame {
		return
	}
	if !p.InWorld() {
		p.Move = nil
		return
	}
	s.stopWalk(p, s.motionOf(p), nil)
}

// OnPlayerArrived adds fn to the functions run when a player's walk reaches
// its destination (Mobius notifyActionArrived), after the arrival packets;
// also when MovePlayerTo finds the player already there. They do not run
// when a walk is stopped, blocked or replaced by another one. Call it from
// a feature's setup function.
func (s *Server) OnPlayerArrived(fn func(p *Player)) {
	s.movement.arrived = append(s.movement.arrived, fn)
}

// notifyArrived runs the arrival hooks of p, then a sit asked for while it
// walked (Mobius notifyActionArrived and its NextAction).
func (s *Server) notifyArrived(p *Player, m *motion) {
	if m.arriving {
		return
	}
	m.arriving = true
	defer func() { m.arriving = false }()
	for _, fn := range s.movement.arrived {
		s.safely("OnPlayerArrived hook", func() { fn(p) })
		if p.state != stateInGame {
			return
		}
	}
	if m.sitOnArrival {
		m.sitOnArrival = false
		s.useSit(p)
	}
}
