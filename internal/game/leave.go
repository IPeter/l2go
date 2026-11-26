package game

import (
	"context"
	"errors"
	"time"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

const (
	// logoutCloseDelay is how long the connection stays open after
	// LeaveWorld (Mobius closes it a second later).
	logoutCloseDelay = time.Second
	// leaveCallTimeout bounds the wait for the loop when a session enters or
	// leaves the world.
	leaveCallTimeout = 10 * time.Second
)

// Restart takes the player out of the world and back to the character
// screen (Mobius RequestRestart): it is saved and removed, and once the save
// is written the session sends RestartResponse and the character list, so
// that the list shows the saved state. In combat it is refused
// (refuseRestart) (Service).
func (s *Server) Restart(sess Session) {
	s.runFor(sess, func() {
		p := s.playerOf(sess)
		if p == nil || p.state != stateInGame || s.refuseStoreRestart(p) || s.refuseRestart(p) {
			return
		}
		s.log.Info("player restarts", p.logArgs()...)
		s.leave(p, true, func(error) {
			go func() {
				ctx, cancel := context.WithTimeout(s.background(), restartListTimeout)
				defer cancel()
				sess.ReturnToCharacterSelect(ctx)
			}()
		})
	})
}

// Logout saves the player, removes it from the world and closes the
// connection a second after LeaveWorld (Mobius Logout). Without a player the
// connection closes at once; in combat it is refused (refuseLogout)
// (Service).
func (s *Server) Logout(sess Session) {
	s.runFor(sess, func() {
		p := s.playerOf(sess)
		if p == nil {
			sess.CloseAfter(nil, 0)
			return
		}
		if p.state == stateInGame && s.refuseLogout(p) {
			return
		}
		s.log.Info("player logs out", p.logArgs()...)
		s.leave(p, true, nil)
		sess.CloseAfter(leaveWorldPacket, logoutCloseDelay)
	})
}

// Disconnected saves and removes the player of a lost connection (Mobius
// Disconnection.onDisconnection); a player in combat stays 15 s more
// without its session (lingerInCombat) (Service).
func (s *Server) Disconnected(sess Session) {
	s.runFor(sess, func() {
		p := s.playerOf(sess)
		if p == nil {
			return
		}
		if p.state == stateInGame && s.lingerInCombat(p) {
			s.log.Info("player disconnected in combat, staying 15 s", p.logArgs()...)
			return
		}
		s.log.Info("player disconnected", p.logArgs()...)
		s.leave(p, true, nil)
	})
}

// kick takes p out of the game because its account selected a character on
// another connection: p is saved and its connection closes with "Another
// person has logged in with the same account." (Mobius doKickPlayer).
func (s *Server) kick(p *Player) {
	sess := p.session
	s.log.Info("account logged in again, kicking the old session", p.logArgs()...)
	s.leave(p, true, nil)
	if sess != nil {
		sess.CloseAfter(anotherLoginPacket, 0)
	}
}

// shutdown saves and removes every player when the server stops; their
// connections get ServerClose (Mobius Shutdown).
func (s *Server) shutdown() {
	players := make([]*Player, 0, len(s.bySession))
	for _, p := range s.bySession {
		players = append(players, p)
	}
	for _, p := range players {
		sess := p.session
		s.leave(p, true, nil)
		if sess != nil {
			sess.CloseAfter(serverClosePacket, 0)
		}
	}
	s.saveGlobals(true) // quest.go: the scripts' global variables
	s.saveBossesAtShutdown()
	s.saveFishingAtShutdown()  // fishing_champ.go
	s.saveOlympiadAtShutdown() // olympiad.go
	s.waitBoardWrites()        // bbs_store.go
	s.saveSevenSignsAtShutdown() // sevensigns_store.go
	s.log.Info("game world: players saved for shutdown", "players", len(players))
}

// Leave takes p out of the game the way a lost connection does: it is saved
// and removed from the world and the registries, and its session is
// forgotten (but not closed). done, if not nil, runs after the save. Use it
// for players that must go, such as a duplicate login. Loop-only.
func (s *Server) Leave(p *Player, done func(error)) { s.leave(p, true, done) }

// leave takes p out of the game: the OnPlayerLeave hooks run (for a player
// that entered), its timers stop, it leaves the world (DeleteObject to its
// viewers) and the registries, and it forgets its session. With save, a
// snapshot taken before the removal goes to the saver and done runs after
// the write; otherwise done runs at once.
func (s *Server) leave(p *Player, save bool, done func(error)) {
	if p.state == stateLeft || p.leaving {
		// Left already, or a leave hook asked again.
		if done != nil {
			done(nil)
		}
		return
	}
	p.leaving = true
	inGame := p.state == stateInGame
	if inGame {
		s.runLeave(p)
	}

	s.loop.Cancel(p.autosave)
	p.autosave = 0
	for _, id := range p.timers {
		s.loop.Cancel(id)
	}
	p.timers = nil
	if t := p.teleport; t != nil {
		s.loop.Cancel(t.watchdog)
		p.teleport = nil
	}
	p.Move = nil

	var snap storage.Character
	var changes storage.ItemChanges
	if save && inGame {
		// The saves others asked for earlier go first (decision I2).
		s.flushSaveRequests()
		snap, changes = p.Snapshot(), p.takeSaveChanges()
	}
	// The client is detached before the player leaves the world (Mobius
	// Disconnection): its viewers get DeleteObject, the leaving client
	// itself gets nothing for the objects it saw.
	sess := p.session
	p.session = nil
	if p.InWorld() {
		s.RemoveObject(p)
	}
	if inGame {
		s.world.RemovePlayer(p)
		if key := storage.NameKey(p.char.Name); s.byName[key] == p {
			delete(s.byName, key)
		}
	}
	if sess != nil && s.bySession[sess] == p {
		delete(s.bySession, sess)
	}
	if s.byAccount[p.char.Account] == p {
		delete(s.byAccount, p.char.Account)
	}
	p.state = stateLeft

	switch {
	case save && inGame:
		s.saver.enqueue(snap, changes, done)
	case done != nil:
		done(nil)
	}
	if inGame {
		s.log.Info("player left the world", append(p.logArgs(), "online", s.world.PlayerCount())...)
	}
}

// post queues fn on the loop for a packet of sess and returns at once. A
// panic in fn closes the session. When the inbox is full the packet is
// dropped and logged: the client sends faster than the world can follow.
// After the loop stopped packets are dropped quietly.
func (s *Server) post(sess Session, fn func()) {
	if s.loop.PostGuarded(fn, func(any) { sess.CloseAfter(nil, 0) }) {
		return
	}
	select {
	case <-s.loop.Done():
	default:
		s.log.Warn("game: world loop busy, packet dropped", "remote", sess.RemoteAddr())
	}
}

// runFor runs fn on the loop for sess and waits for it, for what must not
// get lost (entering and leaving the world): unlike post it waits for room
// in the inbox. A panic in fn closes the session.
func (s *Server) runFor(sess Session, fn func()) {
	ctx, cancel := context.WithTimeout(s.background(), leaveCallTimeout)
	defer cancel()
	err := s.loop.Call(ctx, fn)
	var perr *world.PanicError
	switch {
	case err == nil:
	case errors.As(err, &perr):
		sess.CloseAfter(nil, 0)
	case errors.Is(err, world.ErrStopped):
		// The server stopped; shutdown saved everyone.
	default:
		s.log.Error("game: world loop did not answer", "remote", sess.RemoteAddr(), "err", err)
	}
}
