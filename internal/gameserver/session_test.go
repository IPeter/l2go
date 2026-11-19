package gameserver

import (
	"bytes"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/game"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/testclient"
)

// enterWorldBody is a full EnterWorld body (Mobius reads 104 bytes).
func enterWorldBody() []byte { return make([]byte, 104) }

// selectBody is a CharacterSelect body for slot.
func selectBody(slot int32) []byte {
	w := packets.NewWriter(18)
	w.WriteInt32(slot)
	w.WriteUint16(0)
	w.WriteInt32(0)
	w.WriteInt32(0)
	w.WriteInt32(0)
	return w.Bytes()
}

// withCharacters logs account in and creates a Human Fighter for each name.
// It returns the connection on the character screen and the last list.
func withCharacters(t *testing.T, e *env, account string, names ...string) (*testclient.GameConn, testclient.CharacterList) {
	t.Helper()
	c, list := e.login(t, account)
	for _, name := range names {
		list = wantCreated(t, c, humanFighter(name)).Characters
	}
	return c, list
}

// expectNext reads the next packet and fails the test unless it is a T.
func expectNext[T testclient.Packet](t *testing.T, c *testclient.GameConn) T {
	t.Helper()
	v, err := testclient.Expect[T](testCtx(t), c)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// enter selects the character in slot 0 and enters the world. The fake game
// answers EnterWorld with ActionFailed.
func enter(t *testing.T, e *env, c *testclient.GameConn) {
	t.Helper()
	if _, err := c.SelectCharacter(testCtx(t), 0); err != nil {
		t.Fatalf("SelectCharacter: %v", err)
	}
	if err := c.SendRaw(testCtx(t), clientpackets.OpEnterWorld, enterWorldBody()); err != nil {
		t.Fatal(err)
	}
	expectNext[testclient.ActionFailed](t, c)
}

func TestCharacterSelectGoesToTheGame(t *testing.T) {
	e := newEnv(t, envOptions{})
	c, list := withCharacters(t, e, "player1", "First", "Second")
	cs, err := c.SelectCharacter(testCtx(t), 1)
	if err != nil {
		t.Fatalf("SelectCharacter: %v", err)
	}
	want, _ := list.Find("Second")
	if cs.Name != "Second" || cs.ObjectID != want.ObjectID {
		t.Errorf("selected %q %#x, want Second %#x", cs.Name, cs.ObjectID, want.ObjectID)
	}
	calls := e.game.callsTo("SelectCharacter")
	if len(calls) != 1 || calls[0].slot != 1 {
		t.Fatalf("SelectCharacter calls = %+v, want one for slot 1", calls)
	}
	s := calls[0].session
	if s.Account() != "player1" || s.State() != game.StateEntering || s.PlayKey1() == 0 {
		t.Errorf("session account %q state %v play key %d", s.Account(), s.State(), s.PlayKey1())
	}
	if s.RemoteAddr() == "" {
		t.Error("RemoteAddr is empty")
	}
}

// A slot that is not in the last list gets no answer, as in Mobius, and
// does not reach the game.
func TestCharacterSelectEmptySlot(t *testing.T) {
	e := newEnv(t, envOptions{})
	c, _ := withCharacters(t, e, "player1", "Only")
	sendAll(t, c, []rawPacket{
		{clientpackets.OpCharacterSelect, selectBody(1)},
		{clientpackets.OpCharacterSelect, selectBody(-1)},
	})
	if _, err := c.NewCharacter(testCtx(t)); err != nil {
		t.Fatalf("NewCharacter: %v", err)
	}
	if m := e.game.methods(); len(m) != 0 {
		t.Errorf("the game got %v", m)
	}
}

// A failed selection gets no answer; the client stays on the character
// screen and can try again.
func TestCharacterSelectFailure(t *testing.T) {
	e := newEnv(t, envOptions{})
	c, _ := withCharacters(t, e, "player1", "Broken")
	e.game.setSelectErr(errors.New("cannot load"))
	sendAll(t, c, []rawPacket{{clientpackets.OpCharacterSelect, selectBody(0)}})
	e.game.waitCalls(t, "SelectCharacter", 1)
	if _, err := c.NewCharacter(testCtx(t)); err != nil {
		t.Fatalf("NewCharacter after a failed selection: %v", err)
	}
	e.game.setSelectErr(nil)
	if _, err := c.SelectCharacter(testCtx(t), 0); err != nil {
		t.Fatalf("second SelectCharacter: %v", err)
	}
}

func TestCharacterSelectMalformedClosesConnection(t *testing.T) {
	e := newEnv(t, envOptions{})
	c, _ := withCharacters(t, e, "player1", "Someone")
	sendAll(t, c, []rawPacket{{clientpackets.OpCharacterSelect, []byte{0, 0, 0, 0}}})
	if err := c.ExpectClosed(testCtx(t)); err != nil {
		t.Fatal(err)
	}
	if e.srv.HackAttempts() == 0 {
		t.Error("the malformed packet was not counted")
	}
}

// While the character loads (StateEntering), only EnterWorld is accepted.
// The game gets it once, however often the client sends it.
func TestEnteringAcceptsOnlyEnterWorld(t *testing.T) {
	e := newEnv(t, envOptions{})
	c, _ := withCharacters(t, e, "player1", "Loader")
	if _, err := c.SelectCharacter(testCtx(t), 0); err != nil {
		t.Fatalf("SelectCharacter: %v", err)
	}
	sendAll(t, c, []rawPacket{
		{clientpackets.OpNewCharacter, nil},
		{clientpackets.OpCharacterSelect, selectBody(0)},
		{clientpackets.OpCharacterDelete, make([]byte, 4)},
		{clientpackets.OpLogout, nil}, // not allowed while entering
		{clientpackets.OpRequestRestart, nil},
		{clientpackets.OpMoveToLocation, make([]byte, 28)},
		{clientpackets.OpEx, []byte{0x08, 0x00}},
		{clientpackets.OpEnterWorld, enterWorldBody()},
		{clientpackets.OpEnterWorld, enterWorldBody()},
	})
	expectNext[testclient.ActionFailed](t, c) // the fake game's answer to EnterWorld
	if got := e.game.methods(); !slices.Equal(got, []string{"SelectCharacter", "EnterWorld"}) {
		t.Errorf("the game got %v, want SelectCharacter, EnterWorld", got)
	}
}

func TestEnterWorldMalformedClosesConnection(t *testing.T) {
	e := newEnv(t, envOptions{})
	c, _ := withCharacters(t, e, "player1", "Shorty")
	if _, err := c.SelectCharacter(testCtx(t), 0); err != nil {
		t.Fatalf("SelectCharacter: %v", err)
	}
	sendAll(t, c, []rawPacket{{clientpackets.OpEnterWorld, make([]byte, 100)}})
	if err := c.ExpectClosed(testCtx(t)); err != nil {
		t.Fatal(err)
	}
	if n := len(e.game.callsTo("EnterWorld")); n != 0 {
		t.Errorf("%d EnterWorld calls, want 0", n)
	}
	// The selected character is still released.
	e.game.waitCalls(t, "Disconnected", 1)
}

// In the world, the character screen's packets are dropped and every other
// packet goes to the game with the bytes after its opcode.
func TestInGameRouting(t *testing.T) {
	e := newEnv(t, envOptions{})
	c, _ := withCharacters(t, e, "player1", "Walker")
	enter(t, e, c)

	sendAll(t, c, []rawPacket{
		{clientpackets.OpNewCharacter, nil},
		{clientpackets.OpCharacterCreate, []byte{1, 2, 3}},
		{clientpackets.OpCharacterSelect, selectBody(0)},
		{clientpackets.OpCharacterDelete, make([]byte, 4)},
		{clientpackets.OpCharacterRestore, make([]byte, 4)},
		{clientpackets.OpAuthLogin, authLoginBody("player1", e.issue(t, "player1"))},
		{clientpackets.OpEnterWorld, enterWorldBody()},
		{clientpackets.OpProtocolVersion, []byte{0xea, 0x02, 0, 0}},
	})
	move := []byte{1, 2, 3, 4, 5}
	say := []byte{'h', 0, 'i', 0, 0, 0, 0, 0, 0, 0}
	sendAll(t, c, []rawPacket{
		{clientpackets.OpMoveToLocation, move},
		{clientpackets.OpEx, []byte{0x08, 0x00}},
		{clientpackets.OpEx, []byte{0x34, 0x12, 9, 8, 7}},
		{clientpackets.OpSay2, say},
		{0x7f, nil},
	})
	for range 5 {
		expectNext[testclient.ActionFailed](t, c)
	}

	handled := e.game.callsTo("Handle")
	want := []gameCall{
		{opcode: clientpackets.OpMoveToLocation, body: move},
		{opcode: clientpackets.OpSay2, body: say},
		{opcode: 0x7f, body: []byte{}},
	}
	if len(handled) != len(want) {
		t.Fatalf("Handle calls = %+v, want %d", handled, len(want))
	}
	for i, w := range want {
		if handled[i].opcode != w.opcode || !bytes.Equal(handled[i].body, w.body) {
			t.Errorf("Handle %d = 0x%02x %v, want 0x%02x %v", i, handled[i].opcode, handled[i].body, w.opcode, w.body)
		}
	}
	ex := e.game.callsTo("HandleEx")
	if len(ex) != 2 || ex[0].sub != 0x08 || len(ex[0].body) != 0 ||
		ex[1].sub != 0x1234 || !bytes.Equal(ex[1].body, []byte{9, 8, 7}) {
		t.Errorf("HandleEx calls = %+v, want sub 0x08 with no body, then 0x1234 with 9 8 7", ex)
	}
	if n := len(e.game.callsTo("SelectCharacter")); n != 1 {
		t.Errorf("%d SelectCharacter calls, want 1", n)
	}
	if n := len(e.game.callsTo("EnterWorld")); n != 1 {
		t.Errorf("%d EnterWorld calls, want 1", n)
	}
}

// An extended packet without its sub-opcode is malformed.
func TestExPacketMalformedClosesConnection(t *testing.T) {
	e := newEnv(t, envOptions{})
	c, _ := withCharacters(t, e, "player1", "Extender")
	enter(t, e, c)
	sendAll(t, c, []rawPacket{{clientpackets.OpEx, []byte{0x08}}})
	if err := c.ExpectClosed(testCtx(t)); err != nil {
		t.Fatal(err)
	}
	if n := len(e.game.callsTo("HandleEx")); n != 0 {
		t.Errorf("%d HandleEx calls, want 0", n)
	}
}

// RequestRestart goes to the game, which calls ReturnToCharacterSelect:
// RestartResponse(true), the list, and the character screen again.
func TestRestart(t *testing.T) {
	e := newEnv(t, envOptions{})
	c, _ := withCharacters(t, e, "player1", "Restarter", "Other")
	enter(t, e, c)

	list, err := c.Restart(testCtx(t))
	if err != nil {
		t.Fatalf("Restart: %v", err)
	}
	if len(list.Characters) != 2 {
		t.Errorf("list after restart has %d characters, want 2", len(list.Characters))
	}
	calls := e.game.callsTo("Restart")
	if len(calls) != 1 {
		t.Fatalf("%d Restart calls, want 1", len(calls))
	}
	if st := calls[0].session.State(); st != game.StateAuthenticated {
		t.Errorf("state after restart = %v, want authenticated", stateName(st))
	}

	// The character screen works again, and in-world packets are dropped.
	sendAll(t, c, []rawPacket{{clientpackets.OpMoveToLocation, make([]byte, 28)}})
	if _, err := c.NewCharacter(testCtx(t)); err != nil {
		t.Fatalf("NewCharacter after restart: %v", err)
	}
	cs, err := c.SelectCharacter(testCtx(t), 1)
	if err != nil {
		t.Fatalf("SelectCharacter after restart: %v", err)
	}
	if cs.Name != "Other" {
		t.Errorf("selected %q, want Other", cs.Name)
	}
	if n := len(e.game.callsTo("Handle")); n != 0 {
		t.Errorf("%d Handle calls on the character screen, want 0", n)
	}
	// EnterWorld is accepted again for the new selection.
	if err := c.SendRaw(testCtx(t), clientpackets.OpEnterWorld, enterWorldBody()); err != nil {
		t.Fatal(err)
	}
	expectNext[testclient.ActionFailed](t, c)
	if n := len(e.game.callsTo("EnterWorld")); n != 2 {
		t.Errorf("%d EnterWorld calls, want 2", n)
	}
}

// Logout in the world goes to the game, which sends LeaveWorld and closes
// the connection after a delay. Packets after it are dropped.
func TestLogoutInGame(t *testing.T) {
	e := newEnv(t, envOptions{})
	e.game.setLogoutDelay(200 * time.Millisecond)
	c, _ := withCharacters(t, e, "player1", "Leaver")
	enter(t, e, c)

	sendAll(t, c, []rawPacket{{clientpackets.OpLogout, nil}})
	start := time.Now()
	expectNext[testclient.LeaveWorld](t, c)
	sendAll(t, c, []rawPacket{{clientpackets.OpMoveToLocation, make([]byte, 28)}})
	if err := c.ExpectClosed(testCtx(t)); err != nil {
		t.Fatalf("after LeaveWorld: %v", err)
	}
	if d := time.Since(start); d < 150*time.Millisecond {
		t.Errorf("closed %v after LeaveWorld, want about 200ms", d)
	}
	if n := len(e.game.callsTo("Handle")); n != 0 {
		t.Errorf("%d Handle calls after Logout, want 0", n)
	}
	logouts := e.game.callsTo("Logout")
	if len(logouts) != 1 {
		t.Fatalf("%d Logout calls, want 1", len(logouts))
	}
	// The end of the connection is reported too; the game ignores it for a
	// player that already left.
	d := e.game.waitCalls(t, "Disconnected", 1)
	if d[0].session != logouts[0].session {
		t.Error("Disconnected got another session than Logout")
	}
	waitFor(t, "0 players", func() bool { return e.srv.Players() == 0 })
}

// Logout on the character screen closes the connection at once, as in
// Mobius; no character was selected, so the game hears nothing.
func TestLogoutOnTheCharacterScreen(t *testing.T) {
	e := newEnv(t, envOptions{})
	c, _ := e.login(t, "player1")
	sendAll(t, c, []rawPacket{{clientpackets.OpLogout, nil}})
	if err := c.ExpectClosed(testCtx(t)); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "0 players", func() bool { return e.srv.Players() == 0 && e.srv.OnlineClients() == 0 })
	if m := e.game.methods(); len(m) != 0 {
		t.Errorf("the game got %v", m)
	}
}

// A lost connection is reported to the game once, with the same session,
// when a character was selected on it.
func TestDisconnectedAfterSelection(t *testing.T) {
	e := newEnv(t, envOptions{})
	c, _ := withCharacters(t, e, "player1", "Dropper")
	enter(t, e, c)
	_ = c.Close()
	d := e.game.waitCalls(t, "Disconnected", 1)
	if d[0].session != e.game.callsTo("EnterWorld")[0].session {
		t.Error("Disconnected got another session than EnterWorld")
	}
	waitFor(t, "client removed", func() bool { return e.srv.OnlineClients() == 0 })
	if n := len(e.game.callsTo("Disconnected")); n != 1 {
		t.Errorf("%d Disconnected calls, want 1", n)
	}

	// Without a selection there is nothing to report.
	c2, _ := e.login(t, "player2")
	_ = c2.Close()
	waitFor(t, "client removed", func() bool { return e.srv.OnlineClients() == 0 })
	if n := len(e.game.callsTo("Disconnected")); n != 1 {
		t.Errorf("%d Disconnected calls after a client without selection, want 1", n)
	}
}

// Shutting down reports the players in the world to the game before Serve
// returns, so they can be saved.
func TestShutdownReportsPlayersInTheWorld(t *testing.T) {
	e := newEnv(t, envOptions{})
	c, _ := withCharacters(t, e, "player1", "Stayer")
	enter(t, e, c)
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
	if n := len(e.game.callsTo("Disconnected")); n != 1 {
		t.Errorf("%d Disconnected calls when Serve returned, want 1", n)
	}
}

// The idle timeout applies before the world only: a player standing still
// sends nothing.
func TestNoIdleTimeoutInTheWorld(t *testing.T) {
	e := newEnv(t, envOptions{server: func(s *Server) { s.idleTimeout = 250 * time.Millisecond }})
	c, _ := withCharacters(t, e, "player1", "Statue")
	enter(t, e, c)
	time.Sleep(750 * time.Millisecond)
	sendAll(t, c, []rawPacket{{clientpackets.OpMoveToLocation, make([]byte, 28)}})
	expectNext[testclient.ActionFailed](t, c)
}

// A client that idles on the character screen is still cut off.
func TestIdleTimeoutOnTheCharacterScreen(t *testing.T) {
	e := newEnv(t, envOptions{server: func(s *Server) { s.idleTimeout = 100 * time.Millisecond }})
	c, _ := e.login(t, "player1")
	if err := c.ExpectClosed(testCtx(t)); err != nil {
		t.Fatal(err)
	}
}
