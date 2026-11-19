package scenario

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"slices"
	"strconv"

	"github.com/IPeter/l2go/internal/session"
	"github.com/IPeter/l2go/internal/testclient"
)

// Player is one role of a scenario. Its methods are steps: each takes a
// context and returns an error, so a scenario can pass them to Session.Do.
type Player struct {
	Identity
	s *Session

	keys     session.Keys
	gameAddr string
	game     *testclient.GameConn

	list     testclient.CharacterList
	selected testclient.CharSelected
	entered  testclient.EnterWorldResult
}

// Game returns the player's current game connection, nil before Connect.
func (p *Player) Game() *testclient.GameConn { return p.game }

// ObjectID returns the object ID of the player's character, 0 before
// SelectCharacter.
func (p *Player) ObjectID() int32 {
	if p.game == nil {
		return 0
	}
	return p.game.World().Me().ObjectID
}

// Selected returns the last CharSelected.
func (p *Player) Selected() testclient.CharSelected { return p.selected }

// Entered returns what the server sent during the last EnterWorld.
func (p *Player) Entered() testclient.EnterWorldResult { return p.entered }

// Login runs the client's login sequence on a new, recorded login
// connection: Init, AuthGameGuard (GGAuth), RequestAuthLogin (LoginOk),
// RequestServerList (ServerList) and RequestServerLogin (PlayOk). It keeps
// the session keys and the game server's address for Connect.
func (p *Player) Login(ctx context.Context) error {
	t := p.s.target
	st := p.s.newStream("login", p.Role)
	c, err := testclient.DialLogin(ctx, t.LoginAddr)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()
	in := c.Init()
	st.server("Init", testclient.OpInit, in)
	p.s.sessionIDs = append(p.s.sessionIDs, in.SessionID)

	// AuthGameGuard goes out raw, so that the whole GGAuth answer is
	// recorded (the AuthGameGuard method checks it and drops it).
	st.client("AuthGameGuard", testclient.OpAuthGameGuard)
	if err := c.SendRaw(ctx, testclient.OpAuthGameGuard, authGameGuardBody(in.SessionID)); err != nil {
		return err
	}
	gg, err := c.ReadPacket(ctx)
	if err != nil {
		return err
	}
	name, op := loginPacketName(gg)
	st.server(name, op, gg)
	if _, ok := gg.(testclient.GGAuth); !ok {
		return fmt.Errorf("AuthGameGuard was answered with %s", name)
	}

	st.client("RequestAuthLogin", testclient.OpRequestAuthLogin)
	res, err := c.Login(ctx, p.Account, t.Password, false)
	if err != nil {
		return err
	}
	var list *testclient.ServerListResult
	switch {
	case res.Ok != nil:
		st.server("LoginOk", testclient.OpLoginOk, *res.Ok)
	case res.ServerList != nil:
		// Mobius skips LoginOk when its licence screen is off.
		st.server("ServerList", testclient.OpServerList, *res.ServerList)
		list = res.ServerList
	case res.Fail != nil:
		st.server("LoginFail", testclient.OpLoginFail, *res.Fail)
		return fmt.Errorf("login refused: %v", res.Fail.Reason)
	case res.Kicked != nil:
		st.server("AccountKicked", testclient.OpAccountKicked, *res.Kicked)
		return fmt.Errorf("account kicked: %v", res.Kicked.Reason)
	default:
		return errors.New("login: empty answer")
	}

	if list == nil {
		st.client("RequestServerList", testclient.OpRequestServerList)
		l, err := c.ServerList(ctx)
		if lf, ok := errors.AsType[*testclient.LoginFailError](err); ok {
			st.server("LoginFail", testclient.OpLoginFail, testclient.LoginFail{Reason: lf.Reason})
		}
		if err != nil {
			return err
		}
		st.server("ServerList", testclient.OpServerList, l)
		list = &l
	}
	addr, err := gameAddress(*list, t)
	if err != nil {
		return err
	}

	st.client("RequestServerLogin", testclient.OpRequestServerLogin)
	play, err := c.ServerLogin(ctx, t.ServerID)
	if err != nil {
		return err
	}
	switch {
	case play.Ok != nil:
		st.server("PlayOk", testclient.OpPlayOk, *play.Ok)
	case play.Fail != nil:
		st.server("PlayFail", testclient.OpPlayFail, *play.Fail)
		return fmt.Errorf("server login refused: %v", play.Fail.Reason)
	case play.LoginFail != nil:
		st.server("LoginFail", testclient.OpLoginFail, *play.LoginFail)
		return fmt.Errorf("server login refused: %v", play.LoginFail.Reason)
	default:
		return errors.New("server login: empty answer")
	}
	p.keys = c.Keys()
	p.gameAddr = addr
	p.s.keys = append(p.s.keys, p.keys)
	return nil
}

// authGameGuardBody is the body of AuthGameGuard after the opcode: the
// session ID and four GameGuard words, which Mobius reads and ignores.
func authGameGuardBody(sessionID int32) []byte {
	b := make([]byte, 0, 20)
	b = binary.LittleEndian.AppendUint32(b, uint32(sessionID)) //nolint:gosec // same bits as the Java int
	for range 4 {
		b = binary.LittleEndian.AppendUint32(b, 0)
	}
	return b
}

// loginPacketName returns the Mobius name and opcode of a decoded login
// server packet.
func loginPacketName(v any) (string, byte) {
	switch v.(type) {
	case testclient.GGAuth:
		return "GGAuth", testclient.OpGGAuth
	case testclient.LoginOk:
		return "LoginOk", testclient.OpLoginOk
	case testclient.LoginFail:
		return "LoginFail", testclient.OpLoginFail
	case testclient.AccountKicked:
		return "AccountKicked", testclient.OpAccountKicked
	case testclient.ServerListResult:
		return "ServerList", testclient.OpServerList
	case testclient.PlayOk:
		return "PlayOk", testclient.OpPlayOk
	case testclient.PlayFail:
		return "PlayFail", testclient.OpPlayFail
	}
	return fmt.Sprintf("%T", v), 0
}

// gameAddress returns where to connect for t: the override, or the
// address the server list gives for t.ServerID.
func gameAddress(list testclient.ServerListResult, t Target) (string, error) {
	if t.GameAddr != "" {
		return t.GameAddr, nil
	}
	for _, srv := range list.Servers {
		if srv.ID != t.ServerID {
			continue
		}
		if !srv.Up {
			return "", fmt.Errorf("game server %d is down in the server list", t.ServerID)
		}
		return net.JoinHostPort(srv.IP.String(), strconv.Itoa(int(srv.Port))), nil
	}
	return "", fmt.Errorf("game server %d is not in the server list", t.ServerID)
}

// Connect opens a recorded game connection to the server Login chose and
// sends AuthLogin with the session keys. The character list it gets back
// is kept for Create and Select.
func (p *Player) Connect(ctx context.Context) error {
	if p.gameAddr == "" {
		return errors.New("connect before login")
	}
	st := p.s.newStream("game", p.Role)
	st.rec = testclient.NewRecorder()
	gc, err := testclient.DialGame(ctx, p.gameAddr, testclient.InterludeProtocol, testclient.WithRecorder(st.rec))
	if gc != nil {
		// Also when the protocol was refused: the connection is open.
		p.s.conns = append(p.s.conns, gc)
	}
	if err != nil {
		return err
	}
	p.game = gc
	list, err := gc.AuthLogin(ctx, p.Account, p.keys)
	if err != nil {
		return err
	}
	p.list = list
	return nil
}

// Create opens the character creation screen (NewCharacter) and creates a
// Human Fighter with the player's character name, unless the list already
// has it.
func (p *Player) Create(ctx context.Context) error {
	if p.game == nil {
		return errors.New("create before connect")
	}
	if _, ok := p.list.Find(p.Char); ok {
		return nil
	}
	if _, err := p.game.NewCharacter(ctx); err != nil {
		return err
	}
	res, err := p.game.CreateCharacter(ctx, HumanFighter(p.Char))
	if err != nil {
		return err
	}
	if !res.Created {
		return fmt.Errorf("character creation refused: %v", res.Reason)
	}
	p.list = res.Characters
	return nil
}

// Select selects the player's character from the last character list.
func (p *Player) Select(ctx context.Context) error {
	if p.game == nil {
		return errors.New("select before connect")
	}
	slot := slices.IndexFunc(p.list.Characters, func(c testclient.CharInfo) bool { return c.Name == p.Char })
	if slot < 0 {
		return fmt.Errorf("character %s is not in the character list", p.Char)
	}
	// After a restart the reader runs: leave nothing stale in its queue
	// for SelectCharacter to trip over. The recorder has it already.
	_, _ = p.game.Drain()
	cs, err := p.game.SelectCharacter(ctx, int32(slot)) //nolint:gosec // at most 7 characters
	if err != nil {
		return err
	}
	p.selected = cs
	return nil
}

// EnterWorld sends EnterWorld and waits for the end of the enter-world
// sequence.
func (p *Player) EnterWorld(ctx context.Context) error {
	if p.game == nil {
		return errors.New("enter before connect")
	}
	res, err := p.game.EnterWorld(ctx)
	p.entered = res
	return err
}

// Restart returns to the character screen (RequestRestart) and keeps the
// character list that follows.
func (p *Player) Restart(ctx context.Context) error {
	if p.game == nil {
		return errors.New("restart before connect")
	}
	list, err := p.game.Restart(ctx)
	if err != nil {
		return err
	}
	p.list = list
	return nil
}

// Logout logs out (Logout, LeaveWorld) and waits for the server to close
// the connection.
func (p *Player) Logout(ctx context.Context) error {
	if p.game == nil {
		return errors.New("logout before connect")
	}
	return p.game.Logout(ctx)
}

// HumanFighter is the CharacterCreate of a male Human Fighter (class 0)
// with the template's stats, as the client sends it.
func HumanFighter(name string) testclient.CreateRequest {
	return testclient.CreateRequest{
		Name: name, Race: 0, Sex: 0, ClassID: 0,
		INT: 21, STR: 40, CON: 43, MEN: 25, DEX: 30, WIT: 11,
		HairStyle: 1, HairColor: 2, Face: 1,
	}
}
