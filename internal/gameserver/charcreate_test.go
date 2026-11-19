package gameserver

import (
	"bytes"
	"context"
	"encoding/binary"
	"log/slog"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/crypt/xor"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Opcodes of the server packets the tests read.
const (
	opCharList       = 0x1f
	opCharCreateOk   = 0x25
	opCharCreateFail = 0x26
)

// testClient speaks the C1 game protocol to a test server.
type testClient struct {
	t      *testing.T
	conn   net.Conn
	cipher *xor.Cipher
}

// connect dials addr and does the handshake (ProtocolVersion, CryptInit).
func connect(t *testing.T, addr string) *testClient {
	t.Helper()
	conn := dial(t, addr)
	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatal(err)
	}
	w := packets.NewWriter(5)
	w.WriteUint8(clientpackets.OpProtocolVersion)
	w.WriteUint32(MinProtocolVersion)
	if err := packets.WritePacket(conn, w.Bytes()); err != nil {
		t.Fatal(err)
	}
	body, err := packets.ReadPacket(conn, packets.MaxBodySize)
	if err != nil {
		t.Fatalf("reading CryptInit: %v", err)
	}
	if len(body) != 2+xor.KeySize || body[0] != 0x00 {
		t.Fatalf("CryptInit = % x", body)
	}
	return &testClient{t: t, conn: conn, cipher: xor.NewCipher([xor.KeySize]byte(body[2:]))}
}

func (c *testClient) send(body []byte) {
	c.t.Helper()
	buf := bytes.Clone(body)
	c.cipher.Encrypt(buf)
	if err := packets.WritePacket(c.conn, buf); err != nil {
		c.t.Fatal(err)
	}
}

func (c *testClient) receive() []byte {
	c.t.Helper()
	body, err := packets.ReadPacket(c.conn, packets.MaxBodySize)
	if err != nil {
		c.t.Fatalf("receive: %v", err)
	}
	c.cipher.Decrypt(body)
	return body
}

// expect reads a packet and checks its opcode.
func (c *testClient) expect(opcode byte) []byte {
	c.t.Helper()
	body := c.receive()
	if body[0] != opcode {
		c.t.Fatalf("got packet % x, want opcode %#x", body, opcode)
	}
	return body
}

func (c *testClient) authLogin(account string) {
	c.t.Helper()
	w := packets.NewWriter(64)
	w.WriteUint8(clientpackets.OpAuthLogin)
	w.WriteString(account)
	for _, key := range []uint32{1, 2, 3, 4} {
		w.WriteUint32(key)
	}
	c.send(w.Bytes())
	c.expect(opCharList)
}

// create sends CharacterCreate and returns the reply.
func (c *testClient) create(name string) []byte {
	c.t.Helper()
	w := packets.NewWriter(64)
	w.WriteUint8(clientpackets.OpCharacterCreate)
	w.WriteString(name)
	// race, sex, class, INT, STR, CON, MEN, DEX, WIT, hair style, hair color, face
	for _, v := range []uint32{1, 1, 18, 1, 2, 3, 4, 5, 6, 2, 3, 1} {
		w.WriteUint32(v)
	}
	c.send(w.Bytes())
	return c.receive()
}

// wantCreated checks that name is accepted: CharCreateOk, then CharList.
func (c *testClient) wantCreated(name string) {
	c.t.Helper()
	if got := c.create(name); got[0] != opCharCreateOk {
		c.t.Fatalf("%q: got % x, want CharCreateOk", name, got)
	}
	c.expect(opCharList)
}

// wantRejected checks that name gets CharCreateFail with code.
func (c *testClient) wantRejected(name string, code serverpackets.CharCreateFailReason) {
	c.t.Helper()
	got := c.create(name)
	if len(got) != 5 || got[0] != opCharCreateFail {
		c.t.Fatalf("%q: got % x, want CharCreateFail", name, got)
	}
	if reason := binary.LittleEndian.Uint32(got[1:]); reason != uint32(code) {
		c.t.Fatalf("%q: reason %#x, want %#x", name, reason, code)
	}
}

// serveWithScripts starts a game server that uses scripts.
func serveWithScripts(t *testing.T, scripts Scripts) running {
	t.Helper()
	s, err := New(config.Default(), 1, scripts, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	return start(t, s)
}

func TestCharacterCreateRejectedByScript(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"events/rules.lua": `
events.on_character_create(function(ev)
  if ev.name == "Badname" then
    return events.reject("incorrect_name")
  end
  if ev.account == "player1" and ev.name == "Taken" then
    return events.reject("name_exists")
  end
  assert(ev.race == 1 and ev.sex == 1 and ev.class_id == 18, "race, sex, class")
  assert(ev.hair_style == 2 and ev.hair_color == 3 and ev.face == 1, "appearance")
end)
`})
	r := serveWithScripts(t, h.Engine())
	c := connect(t, r.addr)
	c.authLogin("player1")

	c.wantRejected("Badname", serverpackets.CharCreateFailIncorrectName)
	c.wantCreated("Goodname")
	c.wantRejected("Taken", serverpackets.CharCreateFailNameExists)
	for _, l := range h.Logs() {
		if l.Msg == "script handler failed" {
			t.Errorf("a handler failed: %v", l.Attrs)
		}
	}
}

// The shipped default script refuses the Mobius ForbiddenNames.
func TestCharacterCreateDefaultScript(t *testing.T) {
	h := scripttest.Load(t, filepath.Join("..", "..", "scripts"))
	r := serveWithScripts(t, h.Engine())
	c := connect(t, r.addr)
	c.authLogin("player1")

	c.wantRejected("Announcer", serverpackets.CharCreateFailIncorrectName)
	c.wantCreated("Frostwind")
}

// Every reason a script can give arrives as its own code.
func TestCharacterCreateEveryReason(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"events/reason.lua": `
events.on_character_create(function(ev)
  return events.reject(ev.name)
end)
`})
	r := serveWithScripts(t, h.Engine())
	c := connect(t, r.addr)
	c.authLogin("player1")
	for _, reason := range scripting.Reasons(scripting.EventCharacterCreate) {
		code, ok := serverpackets.CharCreateFailReasonByName(reason)
		if !ok {
			t.Fatalf("reason %q has no CharCreateFail code", reason)
		}
		c.wantRejected(reason, code)
	}
}

// Changing the script and reloading changes behaviour without a restart.
func TestCharacterCreateReload(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"events/rules.lua": `
events.on_character_create(function(ev)
  if ev.name == "First" then return events.reject("incorrect_name") end
end)
`})
	r := serveWithScripts(t, h.Engine())
	c := connect(t, r.addr)
	c.authLogin("player1")
	c.wantRejected("First", serverpackets.CharCreateFailIncorrectName)
	c.wantCreated("Second")

	scripttest.WriteScript(t, h.Dir(), "events/rules.lua", `
events.on_character_create(function(ev)
  if ev.name == "Second" then return events.reject("name_exists") end
end)
`)
	if err := h.Engine().Reload(context.Background()); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	c.wantCreated("First")
	c.wantRejected("Second", serverpackets.CharCreateFailNameExists)
}

// A handler error is logged and does not block creation.
func TestCharacterCreateHandlerError(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"events/broken.lua": `
events.on_character_create(function(ev)
  local x = nil
  return x.field
end)
`})
	r := serveWithScripts(t, h.Engine())
	c := connect(t, r.addr)
	c.authLogin("player1")
	c.wantCreated("Anyone")
}

func TestCharacterCreateWithoutScripts(t *testing.T) {
	r := serveWithScripts(t, nil)
	c := connect(t, r.addr)
	c.authLogin("player1")
	c.wantCreated("Announcer")
}

// If the scripts engine is not running, creation fails safely.
func TestCharacterCreateEngineStopped(t *testing.T) {
	dir := scripttest.WriteScripts(t, map[string]string{"a.lua": ""})
	e, err := scripting.New(context.Background(), scripting.Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := e.Run(ctx); err != nil {
		t.Fatal(err)
	}

	r := serveWithScripts(t, e)
	c := connect(t, r.addr)
	c.authLogin("player1")
	c.wantRejected("Anyone", serverpackets.CharCreateFailCreationFailed)
}

func TestAuthLoginTooShortClosesConnection(t *testing.T) {
	r := serveWithScripts(t, nil)
	c := connect(t, r.addr)
	c.send([]byte{clientpackets.OpAuthLogin, 'A', 0}) // unterminated name
	expectClosed(t, c.conn)
}
