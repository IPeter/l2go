package game

import (
	"fmt"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/packets"
)

const (
	// opChangeMoveType2 and opChangeWaitType2 are the client packets of the
	// /walk, /run, /sit and /stand chat commands (Mobius ChangeMoveType2 and
	// ChangeWaitType2). Each carries one int: 1 to run or stand.
	opChangeMoveType2 = 0x1c
	opChangeWaitType2 = 0x1d

	// sitAnimation is how long sitting down or standing up takes (Mobius
	// SitDownTask and StandUpTask). Meanwhile the player can neither sit
	// nor stand, and a player standing up still counts as sitting.
	sitAnimation = 2500 * time.Millisecond
)

// ActionUse handles an action of the client's action window
// (RequestActionUse), such as sit/stand, walk/run or a social action.
type ActionUse func(p *Player, req clientpackets.RequestActionUse)

// RegisterActionUse sets the handler of the RequestActionUse action with
// the given ID (Mobius RequestActionUse's cases; 0 sit/stand and 1 walk/run
// are the movement feature's). Mobius ignores the actions it does not know.
// It returns an error wrapping ErrDuplicate when the action has one. Call it
// from a feature's setup function.
func (s *Server) RegisterActionUse(actionID int32, fn ActionUse) error {
	if fn == nil {
		return fmt.Errorf("game: nil handler for action %d", actionID)
	}
	mv := &s.movement
	if _, ok := mv.actions[actionID]; ok {
		return fmt.Errorf("%w: handler of action %d", ErrDuplicate, actionID)
	}
	if mv.actions == nil {
		mv.actions = make(map[int32]ActionUse)
	}
	mv.actions[actionID] = fn
	return nil
}

// handleRequestActionUse handles RequestActionUse (0x45), a click in the
// action window, with the handler of its action (RegisterActionUse).
func (s *Server) handleRequestActionUse(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestActionUse(body)
	if err != nil {
		s.log.Debug("game: bad RequestActionUse", append(p.logArgs(), "err", err)...)
		return
	}
	fn := s.movement.actions[req.ActionID]
	if fn == nil {
		s.log.Debug("game: action without a handler, ignored", append(p.logArgs(), "actionId", req.ActionID)...)
		return
	}
	fn(p, req)
}

// actionSitStand is action 0 (Mobius RequestActionUse): a standing player
// sits down and a sitting one stands up; a walking player sits down when
// the walk arrives.
func (s *Server) actionSitStand(p *Player, _ clientpackets.RequestActionUse) {
	if p.Sitting || p.Move == nil {
		s.useSit(p)
		return
	}
	s.motionOf(p).sitOnArrival = true
}

// actionWalkRun is action 1: the player switches between walking and
// running.
func (s *Server) actionWalkRun(p *Player, _ clientpackets.RequestActionUse) {
	s.setRunning(p, !p.Running())
}

// useSit sits p down, or stands it up when it sits (Mobius
// RequestActionUse.useSit, without chairs).
func (s *Server) useSit(p *Player) {
	if p.Sitting {
		s.standUp(p)
	} else {
		s.sitDown(p)
	}
}

// sitDown makes p sit (Mobius Player.sitDown): a walk stops (StopMove), and
// p and the players around get ChangeWaitType. Nothing happens while the
// animation of sitting down or standing up plays.
func (s *Server) sitDown(p *Player) {
	m := s.motionOf(p)
	if m.sitBusy || p.Sitting || !p.InWorld() || s.AttackingNow(p) {
		// Mobius isAttackDisabled: no sitting down during a swing.
		return
	}
	s.playerIntendRest(p)
	if p.Move != nil {
		s.stopWalk(p, m, nil)
	}
	m.sitOnArrival = false
	p.Sitting = true
	m.sitBusy = true
	s.world.Broadcast(p, changeWaitTypePacket(p, serverpackets.WaitSitting), true)
	p.after(sitAnimation, func() {
		if m := s.movement.players[p]; m != nil {
			m.sitBusy = false
		}
	})
}

// standUp makes a sitting p stand up (Mobius Player.standUp): p and the
// players around get ChangeWaitType at once, and p counts as sitting, unable
// to walk, until the animation ends.
func (s *Server) standUp(p *Player) {
	m := s.motionOf(p)
	if m.sitBusy || !p.Sitting || !p.InWorld() || p.InStoreMode() { // store.go: a store keeps its owner seated
		return
	}
	m.sitBusy = true
	s.world.Broadcast(p, changeWaitTypePacket(p, serverpackets.WaitStanding), true)
	p.after(sitAnimation, func() {
		p.Sitting = false
		if m := s.movement.players[p]; m != nil {
			m.sitBusy = false
		}
	})
}

// changeWaitTypePacket is ChangeWaitType of p at its position.
func changeWaitTypePacket(p *Player, t serverpackets.WaitType) []byte {
	pos := p.Position()
	return serverpackets.NewChangeWaitTypePacket(serverpackets.ChangeWaitType{
		ObjectID: p.ObjectID(),
		Type:     t,
		X:        pos.X, Y: pos.Y, Z: pos.Z,
	})
}

// setRunning switches p between running and walking (Mobius
// Creature.setRunning): p and the players around get ChangeMoveType, then p
// its UserInfo and they its CharInfo. A walk goes on at the new speed.
func (s *Server) setRunning(p *Player, running bool) {
	if p.Running() == running {
		return
	}
	p.SetRunning(running)
	if walk := p.Move; walk != nil {
		walk.Running = running
	}
	if p.stats.RunSpeed() != 0 {
		s.world.Broadcast(p, serverpackets.NewChangeMoveTypePacket(serverpackets.ChangeMoveType{
			ObjectID: p.ObjectID(),
			Running:  running,
		}), true)
	}
	p.BroadcastUserInfo()
}

// handleChangeMoveType2 handles ChangeMoveType2 (0x1C), the /run and /walk
// commands (Mobius ChangeMoveType2).
func (s *Server) handleChangeMoveType2(p *Player, body []byte) {
	run, err := readFlagPacket(body)
	if err != nil {
		s.log.Debug("game: bad ChangeMoveType2", append(p.logArgs(), "err", err)...)
		return
	}
	s.setRunning(p, run)
}

// handleChangeWaitType2 handles ChangeWaitType2 (0x1D), the /stand and /sit
// commands (Mobius ChangeWaitType2, without chairs).
func (s *Server) handleChangeWaitType2(p *Player, body []byte) {
	stand, err := readFlagPacket(body)
	if err != nil {
		s.log.Debug("game: bad ChangeWaitType2", append(p.logArgs(), "err", err)...)
		return
	}
	if stand {
		s.standUp(p)
	} else {
		s.sitDown(p)
	}
}

// readFlagPacket parses the one int of ChangeMoveType2 and ChangeWaitType2:
// true when it is 1, as Mobius reads it.
func readFlagPacket(body []byte) (bool, error) {
	r := packets.NewReader(body)
	v := r.ReadInt32()
	if err := r.Err(); err != nil {
		return false, err
	}
	return v == 1, nil
}
