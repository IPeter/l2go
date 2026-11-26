package gameserver

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/crypt/gamecrypt"
	"github.com/IPeter/l2go/internal/game"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/link"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/session"
	"github.com/IPeter/l2go/internal/storage/memstore"
)

var discard = slog.New(slog.DiscardHandler)

func testKey(t *testing.T) [gamecrypt.KeySize]byte {
	t.Helper()
	key, err := gamecrypt.KeyFromHalf([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	if err != nil {
		t.Fatal(err)
	}
	return key
}

// pipeClient returns a client on one end of a pipe, with its writer
// running, and the peer on the other end. The client is closed when the
// test ends.
func pipeClient(t *testing.T, queueLimit int) (*client, *peer) {
	t.Helper()
	server, other := net.Pipe()
	key := testKey(t)
	c := newClient(t.Context(), nil, server, key, clientConfig{queueLimit: queueLimit, logger: discard})
	t.Cleanup(func() {
		_ = c.Close()
		<-c.writerDone
		_ = other.Close()
	})
	return c, &peer{conn: other, key: key}
}

// peer reads a client's packets on the other end of the connection, as the
// game client does: the first packet (KeyPacket) is plain, the rest
// encrypted.
type peer struct {
	conn net.Conn
	key  [gamecrypt.KeySize]byte
	dec  *gamecrypt.Cipher
}

func (p *peer) read() ([]byte, error) {
	b, err := packets.ReadPacket(p.conn, packets.MaxBodySize)
	if err != nil {
		return nil, err
	}
	if p.dec == nil {
		p.dec = gamecrypt.NewClientCipher(p.key)
		return b, nil
	}
	p.dec.Decrypt(b)
	return b, nil
}

func (p *peer) mustRead(t *testing.T) []byte {
	t.Helper()
	b, err := p.read()
	if err != nil {
		t.Fatalf("reading a packet: %v", err)
	}
	return b
}

// keyBody stands in for KeyPacket, the plain first packet.
var keyBody = []byte{0x2e, 1, 2, 3}

// Packets arrive in the order they were queued, encrypted after the first,
// and the bodies handed to Send are not changed.
func TestSendKeepsOrder(t *testing.T) {
	c, p := pipeClient(t, 4096)
	const n = 2000
	bodies := make([][]byte, n)
	for i := range bodies {
		bodies[i] = []byte{0x01, byte(i), byte(i >> 8), 0x55, 0xaa}
	}
	want := make([][]byte, n)
	for i, b := range bodies {
		want[i] = slices.Clone(b)
	}
	c.Send(keyBody)
	go func() {
		for _, b := range bodies {
			c.Send(b)
		}
	}()
	if got := p.mustRead(t); !bytes.Equal(got, keyBody) {
		t.Fatalf("first packet = %x, want %x sent plain", got, keyBody)
	}
	for i := range n {
		if got := p.mustRead(t); !bytes.Equal(got, want[i]) {
			t.Fatalf("packet %d = %x, want %x", i, got, want[i])
		}
	}
	for i := range bodies {
		if !bytes.Equal(bodies[i], want[i]) {
			t.Fatalf("Send changed body %d: %x, want %x", i, bodies[i], want[i])
		}
	}
}

// One body can go to many clients, each with its own cipher state.
func TestSendSharedBody(t *testing.T) {
	a, pa := pipeClient(t, 16)
	b, pb := pipeClient(t, 16)
	shared := []byte{0x03, 9, 8, 7, 6, 5}
	orig := slices.Clone(shared)
	a.Send(keyBody)
	a.Send([]byte{0x04, 1}) // a's cipher has moved on before the shared body
	a.Send(shared)
	b.Send(keyBody)
	b.Send(shared)
	for name, p := range map[string]*peer{"a": pa, "b": pb} {
		p.mustRead(t)
		if p == pa {
			p.mustRead(t)
		}
		if got := p.mustRead(t); !bytes.Equal(got, orig) {
			t.Errorf("%s: shared body arrived as %x, want %x", name, got, orig)
		}
	}
	if !bytes.Equal(shared, orig) {
		t.Errorf("shared body changed to %x", shared)
	}
}

// Send never blocks. A client whose queue is full is disconnected.
func TestSendOverflowCloses(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c, p := pipeClient(t, 4)
		c.Send(keyBody) // the writer takes it and blocks: the peer does not read
		synctest.Wait()
		for range 4 {
			c.Send([]byte{0x05})
		}
		synctest.Wait()
		select {
		case <-c.done:
			t.Fatal("closed before the queue was full")
		default:
		}
		c.Send([]byte{0x06})
		synctest.Wait()
		select {
		case <-c.done:
		default:
			t.Fatal("still open after the queue overflowed")
		}
		<-c.writerDone
		if _, err := p.read(); err == nil {
			t.Error("the peer could still read")
		}
		c.Send([]byte{0x07}) // dropped, no panic
	})
}

// CloseAfter sends what was queued and the final packet, waits, then
// closes. Later packets are dropped.
func TestCloseAfter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c, p := pipeClient(t, 16)
		c.Send(keyBody)
		c.Send([]byte{0x10, 1})
		c.CloseAfter([]byte{0x7e}, time.Second)
		c.Send([]byte{0x11}) // after CloseAfter: dropped
		c.CloseAfter([]byte{0x12}, 0)
		if !c.closing.Load() {
			t.Error("closing is not set: the client's packets would still be handled")
		}
		for _, want := range [][]byte{keyBody, {0x10, 1}, {0x7e}} {
			if got := p.mustRead(t); !bytes.Equal(got, want) {
				t.Fatalf("got %x, want %x", got, want)
			}
		}
		start := time.Now()
		_, err := p.read()
		if !errors.Is(err, io.EOF) {
			t.Fatalf("read after the final packet: %v, want EOF", err)
		}
		if d := time.Since(start); d != time.Second {
			t.Errorf("closed %v after the final packet, want 1s", d)
		}
		<-c.writerDone
	})
}

// CloseAfter without a packet and delay closes once the queue is written.
func TestCloseAfterFlushes(t *testing.T) {
	c, p := pipeClient(t, 16)
	c.Send(keyBody)
	c.Send([]byte{0x13})
	c.CloseAfter(nil, 0)
	p.mustRead(t)
	if got := p.mustRead(t); !bytes.Equal(got, []byte{0x13}) {
		t.Fatalf("got %x, want 13", got)
	}
	if _, err := p.read(); !errors.Is(err, io.EOF) {
		t.Fatalf("read after the queue: %v, want EOF", err)
	}
}

// Close drops the queue and stops the writer.
func TestCloseDropsQueue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c, p := pipeClient(t, 16)
		c.Send(keyBody) // blocks the writer
		synctest.Wait()
		c.Send([]byte{0x20})
		_ = c.Close()
		<-c.writerDone
		c.Send([]byte{0x21})
		c.CloseAfter([]byte{0x22}, time.Second)
		if _, err := p.read(); err == nil {
			t.Error("the peer could still read")
		}
		if c.ctx.Err() == nil {
			t.Error("the connection's context is still live")
		}
	})
}

// A body too large to frame is skipped; the cipher stays in step.
func TestWriterSkipsOversizedBody(t *testing.T) {
	c, p := pipeClient(t, 16)
	c.Send(keyBody)
	c.Send(make([]byte, packets.MaxBodySize+1))
	c.Send([]byte{0x30, 1, 2})
	p.mustRead(t)
	if got := p.mustRead(t); !bytes.Equal(got, []byte{0x30, 1, 2}) {
		t.Fatalf("got %x, want 30 01 02", got)
	}
}

func TestSendIgnoresEmptyBody(t *testing.T) {
	c, p := pipeClient(t, 16)
	c.Send(keyBody)
	c.Send(nil)
	c.Send([]byte{})
	c.Send([]byte{0x31})
	p.mustRead(t)
	if got := p.mustRead(t); !bytes.Equal(got, []byte{0x31}) {
		t.Fatalf("got %x, want 31", got)
	}
}

func TestSessionState(t *testing.T) {
	c, _ := pipeClient(t, 16)
	if c.State() != game.StateConnected || c.Account() != "" || c.PlayKey1() != 0 {
		t.Errorf("new client: state %v account %q key %d", c.State(), c.Account(), c.PlayKey1())
	}
	c.auth.Store(&authInfo{account: "player1", keys: session.Keys{PlayOK1: 77}})
	for _, st := range []game.State{game.StateAuthenticated, game.StateEntering, game.StateInGame, game.StateAuthenticated} {
		c.SetState(st)
		if got := c.State(); got != st {
			t.Errorf("State() = %v after SetState(%v)", got, st)
		}
	}
	if c.Account() != "player1" || c.PlayKey1() != 77 {
		t.Errorf("account %q key %d", c.Account(), c.PlayKey1())
	}
}

// The connection states of Mobius ClientPackets.
func TestAllowed(t *testing.T) {
	const (
		con = game.StateConnected
		aut = game.StateAuthenticated
		ent = game.StateEntering
		ing = game.StateInGame
	)
	tests := []struct {
		opcode byte
		states []game.State
	}{
		{clientpackets.OpAuthLogin, []game.State{con}},
		{clientpackets.OpNewCharacter, []game.State{aut}},
		{clientpackets.OpCharacterCreate, []game.State{aut}},
		{clientpackets.OpCharacterDelete, []game.State{aut}},
		{clientpackets.OpCharacterRestore, []game.State{aut}},
		{clientpackets.OpCharacterSelect, []game.State{aut}},
		{clientpackets.OpLogout, []game.State{aut, ing}},
		{clientpackets.OpEnterWorld, []game.State{ent}},
		{clientpackets.OpRequestPledgeCrest, []game.State{aut, ing}},
		{clientpackets.OpRequestRestart, []game.State{ing}},
		{clientpackets.OpEx, []game.State{ing}},
		{clientpackets.OpMoveToLocation, []game.State{ing}},
		{clientpackets.OpSay2, []game.State{ing}},
		{0xff, []game.State{ing}},
	}
	for _, tt := range tests {
		for _, st := range []game.State{con, aut, ent, ing} {
			if got, want := allowed(tt.opcode, st), slices.Contains(tt.states, st); got != want {
				t.Errorf("allowed(0x%02x, %s) = %v, want %v", tt.opcode, stateName(st), got, want)
			}
		}
	}
}

// newTestServer returns a server with a fake game, for tests that call its
// handlers directly.
func newTestServer(t *testing.T) (*Server, *fakeGame) {
	t.Helper()
	chars := memstore.NewCharacters()
	g := newFakeGame(chars, nil)
	s, err := New(config.Default(), 1, Deps{
		Characters: chars,
		IDs:        memstore.NewIDs(),
		Data:       loadData(t),
		Login:      link.NewClient(link.ClientConfig{}, nil),
		Game:       g,
	}, discard)
	if err != nil {
		t.Fatal(err)
	}
	return s, g
}

// EnterWorld reaches the game once per selection, even while the game has
// not entered the world yet.
func TestEnterWorldOncePerSelection(t *testing.T) {
	s, g := newTestServer(t)
	g.holdEnter = true
	c, _ := pipeClient(t, 16)
	c.SetState(game.StateEntering)
	for range 3 {
		if err := s.handlePacket(t.Context(), c, clientpackets.OpEnterWorld, enterWorldBody()); err != nil {
			t.Fatalf("handlePacket: %v", err)
		}
	}
	if n := len(g.callsTo("EnterWorld")); n != 1 {
		t.Fatalf("%d EnterWorld calls, want 1", n)
	}
	c.SetState(game.StateEntering) // the next selection
	if err := s.handlePacket(t.Context(), c, clientpackets.OpEnterWorld, enterWorldBody()); err != nil {
		t.Fatalf("handlePacket: %v", err)
	}
	if n := len(g.callsTo("EnterWorld")); n != 2 {
		t.Errorf("%d EnterWorld calls after a new selection, want 2", n)
	}
}

// After CloseAfter the client's packets are dropped.
func TestPacketsDroppedWhileClosing(t *testing.T) {
	s, g := newTestServer(t)
	c, _ := pipeClient(t, 16)
	c.SetState(game.StateInGame)
	c.CloseAfter(nil, time.Hour)
	if err := s.handlePacket(t.Context(), c, clientpackets.OpMoveToLocation, make([]byte, 28)); err != nil {
		t.Fatalf("handlePacket: %v", err)
	}
	if m := g.methods(); len(m) != 0 {
		t.Errorf("the game got %v while closing", m)
	}
}
