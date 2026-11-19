package testclient

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/IPeter/l2go/internal/session"
)

var testGameKeys = session.Keys{LoginOK1: 0x0a, LoginOK2: 0x0b, PlayOK1: 0x0c, PlayOK2: 0x0d}

// testAuthLogin is AuthLogin for "player1" with testGameKeys, in Mobius
// AuthLogin.readImpl order.
func testAuthLogin() []byte {
	return msg(0x08).str("player1").i32(0x0d).i32(0x0c).i32(0x0a).i32(0x0b).b
}

// TestGameFlow runs the whole game handshake and character screen against
// a fake server with encryption on, so the rolling keys of both directions
// are checked over many packets.
func TestGameFlow(t *testing.T) {
	templates := testTemplates()
	hero := testChar("Hero", 0x10000001)
	newbie := testChar("Newbie", 0x10000100)
	newbie.Active = false
	req := CreateRequest{
		Name: "Newbie", Race: 0, Sex: 1, ClassID: 0,
		INT: 21, STR: 40, CON: 43, MEN: 25, DEX: 30, WIT: 11,
		HairStyle: 6, HairColor: 3, Face: 2,
	}
	req2 := req
	req2.Name = "Second"
	req3 := req
	req3.Name = "Hero"

	addr := startFakeGame(t, func(f *fakeGame) {
		f.expect(msg(0x00).i32(746).b)
		f.sendKey(1, true)
		f.expect(testAuthLogin())
		f.send(mobiusCharSelectionInfo([]CharInfo{hero}))
		f.expect([]byte{0x0e})
		f.send(mobiusNewCharacterSuccess(templates))
		// Mobius order: the list first, then CharCreateOk.
		f.expect(mobiusCharacterCreate(req))
		f.send(mobiusCharSelectionInfo([]CharInfo{hero, newbie}))
		f.send(msg(0x19).i32(1).b)
		// The order the plan describes: CharCreateOk, then the list.
		f.expect(mobiusCharacterCreate(req2))
		f.send(msg(0x19).i32(1).b)
		f.send(mobiusCharSelectionInfo([]CharInfo{hero, newbie}))
		f.expect(mobiusCharacterCreate(req3))
		f.send(msg(0x1a).i32(2).b)
		f.drain()
	})

	ctx := testContext(t)
	c, err := DialGame(ctx, addr, InterludeProtocol)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	wantKey := KeyPacket{Result: 1, KeyHalf: testKeyHalf, Encryption: true, ServerID: 1, Unknown: 1}
	if c.Key() != wantKey {
		t.Errorf("Key() = %+v, want %+v", c.Key(), wantKey)
	}

	list, err := c.AuthLogin(ctx, "Player1", testGameKeys)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(list, CharacterList{Characters: []CharInfo{hero}}) {
		t.Errorf("AuthLogin list\n got %+v\nwant %+v", list, hero)
	}

	got, err := c.NewCharacter(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, templates) {
		t.Errorf("NewCharacter\n got %+v\nwant %+v", got, templates)
	}

	wantList := CharacterList{Characters: []CharInfo{hero, newbie}}
	res, err := c.CreateCharacter(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Created || !res.ListBeforeOk || !reflect.DeepEqual(res.Characters, wantList) {
		t.Errorf("CreateCharacter (list first) = %+v", res)
	}

	res, err = c.CreateCharacter(ctx, req2)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Created || res.ListBeforeOk || !reflect.DeepEqual(res.Characters, wantList) {
		t.Errorf("CreateCharacter (CharCreateOk first) = %+v", res)
	}

	res, err = c.CreateCharacter(ctx, req3)
	if err != nil {
		t.Fatal(err)
	}
	if res.Created || res.Reason != CreateFailNameExists {
		t.Errorf("CreateCharacter (name exists) = %+v", res)
	}
}

func TestGameEncryptionOff(t *testing.T) {
	hero := testChar("Hero", 0x10000001)
	addr := startFakeGame(t, func(f *fakeGame) {
		f.expect(msg(0x00).i32(746).b)
		f.sendKey(1, false)
		f.expect(testAuthLogin())
		f.send(mobiusCharSelectionInfo([]CharInfo{hero}))
		f.drain()
	})
	ctx := testContext(t)
	c, err := DialGame(ctx, addr, InterludeProtocol)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	if c.Key().Encryption {
		t.Error("Key().Encryption is true")
	}
	list, err := c.AuthLogin(ctx, "player1", testGameKeys)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Characters) != 1 || list.Characters[0].Name != "Hero" {
		t.Errorf("list %+v", list)
	}
}

func TestDialGameWrongProtocol(t *testing.T) {
	addr := startFakeGame(t, func(f *fakeGame) {
		f.expect(msg(0x00).i32(999).b)
		f.sendKey(0, true)
	})
	ctx := testContext(t)
	c, err := DialGame(ctx, addr, 999)
	if !errors.Is(err, ErrWrongProtocol) {
		t.Fatalf("got %v, want ErrWrongProtocol", err)
	}
	if c == nil {
		t.Fatal("no connection returned with ErrWrongProtocol")
	}
	defer func() { _ = c.Close() }()
	if c.Key().Result != 0 || c.Key().KeyHalf != testKeyHalf {
		t.Errorf("Key() = %+v", c.Key())
	}
	if err := c.ExpectClosed(ctx); err != nil {
		t.Errorf("ExpectClosed: %v", err)
	}
}

func TestPing(t *testing.T) {
	t.Run("server closes", func(t *testing.T) {
		addr := startFakeGame(t, func(f *fakeGame) {
			f.expect([]byte{0x00, 0xfe, 0xff, 0xff, 0xff})
		})
		if err := Ping(testContext(t), addr); err != nil {
			t.Errorf("Ping: %v", err)
		}
	})
	t.Run("server answers", func(t *testing.T) {
		addr := startFakeGame(t, func(f *fakeGame) {
			f.recv()
			f.sendKey(1, false)
			f.drain()
		})
		err := Ping(testContext(t), addr)
		var upe *UnexpectedPacketError
		if !errors.As(err, &upe) || upe.Name != "KeyPacket" || upe.Want != "connection close" {
			t.Errorf("got %v, want KeyPacket as an unexpected packet", err)
		}
	})
}

func TestAuthLoginRefused(t *testing.T) {
	t.Run("LoginFail", func(t *testing.T) {
		addr := startFakeGame(t, func(f *fakeGame) {
			f.recv()
			f.sendKey(1, true)
			f.expect(testAuthLogin())
			f.send(msg(0x14).i32(1).b)
			f.drain()
		})
		ctx := testContext(t)
		c, err := DialGame(ctx, addr, InterludeProtocol)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = c.Close() }()
		_, err = c.AuthLogin(ctx, "player1", testGameKeys)
		var lfe *GameLoginFailError
		if !errors.As(err, &lfe) || lfe.Reason != GameLoginFailSystemErrorLoginLater {
			t.Errorf("got %v, want GameLoginFailError with reason 1", err)
		}
	})
	t.Run("server closes", func(t *testing.T) {
		addr := startFakeGame(t, func(f *fakeGame) {
			f.recv()
			f.sendKey(1, true)
			f.recv()
		})
		ctx := testContext(t)
		c, err := DialGame(ctx, addr, InterludeProtocol)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = c.Close() }()
		if _, err := c.AuthLogin(ctx, "player1", testGameKeys); !errors.Is(err, ErrServerClosed) {
			t.Errorf("got %v, want ErrServerClosed", err)
		}
	})
}

func TestCreateCharacterIncomplete(t *testing.T) {
	addr := startFakeGame(t, func(f *fakeGame) {
		f.recv()
		f.sendKey(1, true)
		f.recv()
		f.send(msg(0x19).i32(1).b)
		f.send(msg(0x1a).i32(0).b)
		f.drain()
	})
	ctx := testContext(t)
	c, err := DialGame(ctx, addr, InterludeProtocol)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	_, err = c.CreateCharacter(ctx, CreateRequest{Name: "x"})
	var upe *UnexpectedPacketError
	if !errors.As(err, &upe) || upe.Name != "CharCreateFail" || !strings.Contains(upe.Want, "after CharCreateOk") {
		t.Errorf("got %v", err)
	}
}

func TestGameRaw(t *testing.T) {
	addr := startFakeGame(t, func(f *fakeGame) {
		f.recv()
		f.sendKey(1, true)
		f.expect([]byte{0x0e})
		f.send(msg(0x42).u32(1).b)
		f.send(msg(0x42).u32(2).b)
		f.send(msg(0x19).i32(1).b)
		f.drain()
	})
	ctx := testContext(t)
	c, err := DialGame(ctx, addr, InterludeProtocol)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	if err := c.SendRaw(ctx, 0x0e, nil); err != nil {
		t.Fatal(err)
	}
	op, body, err := c.ReadRaw(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if op != 0x42 || !reflect.DeepEqual(body, []byte{1, 0, 0, 0}) {
		t.Errorf("ReadRaw = 0x%02x % x", op, body)
	}
	if _, err := c.ReadPacket(ctx); err == nil || !strings.Contains(err.Error(), "opcode 0x42") {
		t.Errorf("ReadPacket of an unknown packet: got %v", err)
	}
	p, err := c.ReadPacket(ctx)
	if err != nil || p != (CharCreateOk{Result: 1}) {
		t.Errorf("ReadPacket = %#v, %v", p, err)
	}
}
