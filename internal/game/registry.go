package game

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The registries below are filled while the server is set up: by the core in
// New and by the features in their setup functions (registerFeature). Call
// the Register and On functions there, or on the loop.

// Handler handles a client packet of a player in the game, on the loop. body
// is the packet after its opcode (after the sub-opcode for an Ex packet):
// the input of the clientpackets parsers. The handler may keep it.
type Handler func(p *Player, body []byte)

// Register sets the handler of the client packet with the given opcode. A
// packet without a handler is dropped, as Mobius drops unknown packets. It
// returns an error wrapping ErrDuplicate when the opcode has one.
func (s *Server) Register(opcode byte, h Handler) error {
	if h == nil {
		return fmt.Errorf("game: nil handler for opcode 0x%02x", opcode)
	}
	if s.handlers[opcode] != nil {
		return fmt.Errorf("%w: handler of opcode 0x%02x", ErrDuplicate, opcode)
	}
	s.handlers[opcode] = h
	return nil
}

// RegisterEx sets the handler of the Ex client packet (opcode 0xD0) with the
// given sub-opcode, like Register.
func (s *Server) RegisterEx(subOpcode uint16, h Handler) error {
	if h == nil {
		return fmt.Errorf("game: nil handler for Ex packet 0x%04x", subOpcode)
	}
	if s.exHandlers[subOpcode] != nil {
		return fmt.Errorf("%w: handler of Ex packet 0x%04x", ErrDuplicate, subOpcode)
	}
	s.exHandlers[subOpcode] = h
	return nil
}

// Handle runs the handler of an in-world packet on the loop (Service). The
// packet is dropped unless the session's player is in the game.
func (s *Server) Handle(sess Session, opcode byte, body []byte) {
	b := bytes.Clone(body)
	s.post(sess, func() {
		if p := s.inGame(sess, "0x%02x", opcode); p != nil {
			if h := s.handlers[opcode]; h != nil {
				h(p, b)
			} else {
				s.log.Debug("game: packet without a handler, ignored", append(p.logArgs(), "opcode", fmt.Sprintf("0x%02x", opcode))...)
			}
		}
	})
}

// HandleEx runs the handler of an in-world Ex packet on the loop, like
// Handle (Service).
func (s *Server) HandleEx(sess Session, subOpcode uint16, body []byte) {
	b := bytes.Clone(body)
	s.post(sess, func() {
		if p := s.inGame(sess, "0xd0:0x%04x", subOpcode); p != nil {
			if h := s.exHandlers[subOpcode]; h != nil {
				h(p, b)
			} else {
				s.log.Debug("game: Ex packet without a handler, ignored", append(p.logArgs(), "subOpcode", fmt.Sprintf("0x%04x", subOpcode))...)
			}
		}
	})
}

// inGame returns the session's player when it is in the game; otherwise it
// logs the dropped packet and returns nil. Mobius allows every in-world
// packet only in the IN_GAME state.
func (s *Server) inGame(sess Session, format string, op any) *Player {
	p := s.playerOf(sess)
	if p == nil || p.state != stateInGame {
		s.log.Debug("game: packet without a player in the game, dropped", "packet", fmt.Sprintf(format, op), "remote", sess.RemoteAddr())
		return nil
	}
	return p
}

// GMCommand handles a GM command, typed in the chat as "//name args" and
// sent as SendBypassBuildCmd. args are the words after the name.
type GMCommand func(p *Player, args []string)

type gmCommand struct {
	minAccess storage.AccessLevel
	fn        GMCommand
}

// RegisterGMCommand adds the GM command name: what is typed after "//",
// matched exactly (Mobius admin_ + name). Players need access level
// minAccess, and at least 1: non-GMs are ignored (Mobius
// AdminCommandHandler).
func (s *Server) RegisterGMCommand(name string, minAccess storage.AccessLevel, fn GMCommand) error {
	switch {
	case name == "" || strings.ContainsAny(name, " \t"):
		return fmt.Errorf("game: bad GM command name %q", name)
	case fn == nil:
		return fmt.Errorf("game: nil GM command %q", name)
	}
	if _, ok := s.gmCommands[name]; ok {
		return fmt.Errorf("%w: GM command %q", ErrDuplicate, name)
	}
	s.gmCommands[name] = gmCommand{minAccess: max(minAccess, storage.AccessLevelAdmin), fn: fn}
	return nil
}

// GMCommands returns the names of the GM commands, sorted.
func (s *Server) GMCommands() []string {
	names := make([]string, 0, len(s.gmCommands))
	for name := range s.gmCommands {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// RunGMCommand runs a GM command line (without the slashes) for p, with
// Mobius's checks and answers: a non-GM is ignored, an unknown command gets
// "The command '...' does not exist!", a GM without the access level gets
// "You don't have the access rights to use this command!". Loop-only.
func (s *Server) RunGMCommand(p *Player, line string) {
	if !p.IsGM() {
		return
	}
	name, rest, _ := strings.Cut(line, " ")
	cmd, ok := s.gmCommands[name]
	if !ok {
		p.SendMessage("The command '" + name + "' does not exist!")
		s.log.Warn("game: unknown GM command", append(p.logArgs(), "command", name)...)
		return
	}
	if p.AccessLevel() < cmd.minAccess {
		p.SendMessage("You don't have the access rights to use this command!")
		s.log.Warn("game: GM command without the access level", append(p.logArgs(), "command", name, "need", cmd.minAccess)...)
		return
	}
	s.log.Info("GM command", append(p.logArgs(), "command", line)...)
	cmd.fn(p, strings.Fields(rest))
}

// handleBuildCmd handles SendBypassBuildCmd (0x5B), a "//command" line.
func (s *Server) handleBuildCmd(p *Player, body []byte) {
	req, err := clientpackets.ParseSendBypassBuildCmd(body)
	if err != nil {
		s.log.Debug("game: bad SendBypassBuildCmd", append(p.logArgs(), "err", err)...)
		return
	}
	s.RunGMCommand(p, req.Command)
}

// UserCommand handles a user command such as /loc, which the client sends
// as BypassUserCmd with the command's number.
type UserCommand func(p *Player)

// RegisterUserCommand adds the user command with the given number (Mobius
// IUserCommandHandler COMMAND_IDS, for example 0 for /loc).
func (s *Server) RegisterUserCommand(id int32, fn UserCommand) error {
	if fn == nil {
		return fmt.Errorf("game: nil user command %d", id)
	}
	if _, ok := s.userCommands[id]; ok {
		return fmt.Errorf("%w: user command %d", ErrDuplicate, id)
	}
	s.userCommands[id] = fn
	return nil
}

// handleUserCmd handles BypassUserCmd (0xAA). An unknown command tells GMs
// that it is not implemented, as Mobius does.
func (s *Server) handleUserCmd(p *Player, body []byte) {
	req, err := clientpackets.ParseBypassUserCmd(body)
	if err != nil {
		s.log.Debug("game: bad BypassUserCmd", append(p.logArgs(), "err", err)...)
		return
	}
	fn, ok := s.userCommands[req.Command]
	if !ok {
		if p.IsGM() {
			p.SendMessage(fmt.Sprintf("User commandID %d not implemented yet.", req.Command))
		}
		return
	}
	fn(p)
}

// hooks are the functions features add to game events.
type hooks struct {
	enter    []func(p *Player)
	leave    []func(p *Player)
	moved    []func(p *Player, from, to world.Position)
	forget   []func(viewer, object world.Object)
	teleport []func(p *Player, to world.Position)
}

// OnPlayerEnter adds fn to the functions run when a player enters the world
// (Mobius onPlayerEnter): it is in the world, its first packets are sent,
// and SkillList, the second UserInfo, ValidateLocation and ActionFailed
// follow. Use it to look at the zones around, notify friends, start tasks.
func (s *Server) OnPlayerEnter(fn func(p *Player)) { s.hooks.enter = append(s.hooks.enter, fn) }

// OnPlayerLeave adds fn to the functions run when a player leaves the game
// (restart, logout, disconnection, kick, shutdown), before it is saved and
// removed from the world: stop its movement there (set the final position
// with MoveObject), cancel its tasks, clear references to it.
func (s *Server) OnPlayerLeave(fn func(p *Player)) { s.hooks.leave = append(s.hooks.leave, fn) }

// OnPlayerMoved adds fn to the functions run after a player's position
// changed in the world (MoveObject, and the end of a teleport), with the
// old and the new position: zones, map regions. It also runs when the player
// appears in the world on EnterWorld, with from equal to to, before
// EtcStatusUpdate (Mobius onSpawn revalidates the zones there).
func (s *Server) OnPlayerMoved(fn func(p *Player, from, to world.Position)) {
	s.hooks.moved = append(s.hooks.moved, fn)
}

// OnForget adds fn to the functions run, for every pair of objects that
// lost sight of each other, once per direction: viewer no longer sees
// object (Mobius removeVisibleObject and switchRegion clear targets and make
// AI forget). The DeleteObject packets are already sent. Targeting uses it
// to drop a target that went out of sight.
func (s *Server) OnForget(fn func(viewer, object world.Object)) {
	s.hooks.forget = append(s.hooks.forget, fn)
}

// OnTeleport adds fn to the functions run when a player starts a teleport,
// while it is still in the world at its old position: stop its movement,
// cancel its target (Mobius teleToLocation does both).
func (s *Server) OnTeleport(fn func(p *Player, to world.Position)) {
	s.hooks.teleport = append(s.hooks.teleport, fn)
}

// OnTick adds fn to the functions run on every tick of the loop
// (world.Loop.OnTick), before the coalesced updates are sent.
func (s *Server) OnTick(fn func(now time.Time)) world.HookID { return s.loop.OnTick(fn) }

func (s *Server) runEnter(p *Player) {
	for _, fn := range s.hooks.enter {
		s.safely("OnPlayerEnter hook", func() { fn(p) })
	}
}

func (s *Server) runLeave(p *Player) {
	for _, fn := range s.hooks.leave {
		s.safely("OnPlayerLeave hook", func() { fn(p) })
	}
}

func (s *Server) runMoved(p *Player, from, to world.Position) {
	for _, fn := range s.hooks.moved {
		s.safely("OnPlayerMoved hook", func() { fn(p, from, to) })
	}
}

func (s *Server) runForget(viewer, object world.Object) {
	for _, fn := range s.hooks.forget {
		s.safely("OnForget hook", func() { fn(viewer, object) })
	}
}

func (s *Server) runTeleport(p *Player, to world.Position) {
	for _, fn := range s.hooks.teleport {
		s.safely("OnTeleport hook", func() { fn(p, to) })
	}
}
