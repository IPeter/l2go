package testclient

import (
	"bytes"
	"context"
	"errors"
	"net/netip"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/crypt/logincrypt"
	"github.com/IPeter/l2go/internal/crypt/rsakey"
	"github.com/IPeter/l2go/internal/session"
)

// expectCredentials reads RequestAuthLogin, decrypts its RSA block(s) with
// the server key and checks the account name and password.
func (f *fakeLogin) expectCredentials(user, password string, size int) {
	got := f.recv()
	if want := logincrypt.SessionPacketSize(1 + size); len(got) != want {
		f.fatalf("RequestAuthLogin is %d bytes, want %d for a %d-byte block", len(got), want, size)
	}
	if got[0] != 0x00 {
		f.fatalf("RequestAuthLogin opcode 0x%02x", got[0])
	}
	for i, b := range got[1+size : len(got)-logincrypt.ChecksumSize] {
		if b != 0 {
			f.fatalf("byte %d after the RSA block is 0x%02x", 1+size+i, b)
		}
	}
	dec, err := f.key.DecryptRaw(got[1 : 1+size])
	if err != nil {
		f.fatalf("RSA: %v", err)
	}
	cred, err := rsakey.ParseCredentials(dec)
	if err != nil {
		f.fatalf("credentials: %v", err)
	}
	if cred.User != user || cred.Password != password {
		f.fatalf("credentials %+v, want %q / %q", cred, user, password)
	}
}

func testContext(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

var testServers = []ServerInfo{{
	ID: 1, IP: netip.MustParseAddr("127.0.0.1"), Port: 7777,
	CurrentPlayers: 3, MaxPlayers: 100, Up: true, ServerType: 1,
}}

func TestLoginAndSelect(t *testing.T) {
	addr := startFakeLogin(t, func(f *fakeLogin) {
		f.sendInit()
		f.expect(msg(0x07).i32(testSessionID).i32(0).i32(0).i32(0).i32(0).b)
		f.send(mobiusGGAuth(testSessionID))
		f.expectCredentials("player1", "pass1", 128)
		f.send(mobiusLoginOk(testLoginOK1, testLoginOK2))
		f.expect(msg(0x05).i32(testLoginOK1).i32(testLoginOK2).b)
		f.send(mobiusServerList(1, testServers, nil))
		f.expect(msg(0x02).i32(testLoginOK1).i32(testLoginOK2).u8(1).b)
		f.send(mobiusPlayOk(testPlayOK1, testPlayOK2))
		f.drain()
	})
	got, err := LoginAndSelect(testContext(t), addr, "player1", "pass1", 1)
	if err != nil {
		t.Fatal(err)
	}
	want := session.Keys{LoginOK1: testLoginOK1, LoginOK2: testLoginOK2, PlayOK1: testPlayOK1, PlayOK2: testPlayOK2}
	if got != want {
		t.Errorf("keys %+v, want %+v", got, want)
	}
}

func TestLoginStepByStep(t *testing.T) {
	chars := []ServerCharacters{{ServerID: 1, Characters: 2, DeleteTimers: []int32{3600}}}
	addr := startFakeLogin(t, func(f *fakeLogin) {
		f.sendInit()
		f.expect(msg(0x07).i32(testSessionID).i32(0).i32(0).i32(0).i32(0).b)
		f.send(mobiusGGAuth(testSessionID))
		f.expectCredentials("averyveryverylongaccountname", "pw", 256)
		f.send(mobiusLoginOk(testLoginOK1, testLoginOK2))
		f.expect(msg(0x05).i32(testLoginOK1).i32(testLoginOK2).b)
		f.send(mobiusServerList(1, testServers, chars))
		f.expect(msg(0x02).i32(testLoginOK1).i32(testLoginOK2).u8(1).b)
		f.send(mobiusPlayOk(testPlayOK1, testPlayOK2))
		f.drain()
	})
	ctx := testContext(t)
	c, err := DialLogin(ctx, addr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	in := c.Init()
	if in.SessionID != testSessionID || in.Protocol != LoginProtocol ||
		in.GameGuard != MobiusGameGuard || in.BlowfishKey != testBlowfishKey {
		t.Errorf("Init %+v", in)
	}
	kp, err := testKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	if c.PublicKey().N.Cmp(kp.PublicKey().N) != 0 || c.PublicKey().E != 65537 {
		t.Error("RSA key from Init differs from the server key")
	}

	if err := c.AuthGameGuard(ctx); err != nil {
		t.Fatal(err)
	}
	res, err := c.Login(ctx, "averyveryverylongaccountname", "pw", true)
	if err != nil {
		t.Fatal(err)
	}
	wantOk := LoginOk{LoginOK1: testLoginOK1, LoginOK2: testLoginOK2, Fields: [6]int32{0, 0, 0x3ea, 0, 0, 0}}
	if res.Ok == nil || *res.Ok != wantOk {
		t.Fatalf("Login = %v, want %+v", res, wantOk)
	}
	list, err := c.ServerList(ctx)
	if err != nil {
		t.Fatal(err)
	}
	wantList := ServerListResult{LastServer: 1, Servers: testServers, Characters: chars}
	if !reflect.DeepEqual(list, wantList) {
		t.Errorf("ServerList\n got %+v\nwant %+v", list, wantList)
	}
	play, err := c.ServerLogin(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if play.Ok == nil || *play.Ok != (PlayOk{PlayOK1: testPlayOK1, PlayOK2: testPlayOK2}) {
		t.Errorf("ServerLogin = %v", play)
	}
	want := session.Keys{LoginOK1: testLoginOK1, LoginOK2: testLoginOK2, PlayOK1: testPlayOK1, PlayOK2: testPlayOK2}
	if c.Keys() != want {
		t.Errorf("Keys() = %+v, want %+v", c.Keys(), want)
	}
}

func TestLoginResults(t *testing.T) {
	tests := []struct {
		name  string
		reply []byte
		check func(t *testing.T, r LoginResult)
	}{
		{
			name:  "LoginFail",
			reply: mobiusLoginFail(0x02),
			check: func(t *testing.T, r LoginResult) {
				if r.Fail == nil || r.Fail.Reason != LoginFailUserOrPassWrong || r.Ok != nil {
					t.Errorf("got %v", r)
				}
			},
		},
		{
			name:  "AccountKicked",
			reply: mobiusAccountKicked(0x20),
			check: func(t *testing.T, r LoginResult) {
				if r.Kicked == nil || r.Kicked.Reason != AccountKickedPermanentlyBanned {
					t.Errorf("got %v", r)
				}
			},
		},
		{
			// Mobius with ShowLicence off answers with the list at once.
			name:  "ServerList instead of LoginOk",
			reply: mobiusServerList(1, testServers, nil),
			check: func(t *testing.T, r LoginResult) {
				if r.ServerList == nil || !reflect.DeepEqual(r.ServerList.Servers, testServers) {
					t.Errorf("got %v", r)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := startFakeLogin(t, func(f *fakeLogin) {
				f.sendInit()
				f.expectCredentials("user", "password", 128)
				f.send(tt.reply)
				f.drain()
			})
			ctx := testContext(t)
			c, err := DialLogin(ctx, addr)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = c.Close() }()
			res, err := c.Login(ctx, "user", "password", false)
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, res)
			if c.Keys() != (session.Keys{}) {
				t.Errorf("keys stored without LoginOk: %+v", c.Keys())
			}
		})
	}
}

func TestServerLoginResults(t *testing.T) {
	tests := []struct {
		name  string
		reply []byte
		want  PlayResult
	}{
		{"PlayFail", mobiusPlayFail(0x0f), PlayResult{Fail: &PlayFail{Reason: PlayFailServerOverloaded}}},
		{"LoginFail", mobiusLoginFail(0x15), PlayResult{LoginFail: &LoginFail{Reason: LoginFailAccessFailed}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := startFakeLogin(t, func(f *fakeLogin) {
				f.sendInit()
				// The keys set with SetLoginKeys, and server ID 9.
				f.expect(msg(0x02).i32(5).i32(-6).u8(9).b)
				f.send(tt.reply)
				f.drain()
			})
			ctx := testContext(t)
			c, err := DialLogin(ctx, addr)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = c.Close() }()
			c.SetLoginKeys(5, -6)
			got, err := c.ServerLogin(ctx, 9)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoginFailErrors(t *testing.T) {
	t.Run("ServerList", func(t *testing.T) {
		addr := startFakeLogin(t, func(f *fakeLogin) {
			f.sendInit()
			f.expect(msg(0x05).i32(0).i32(0).b)
			f.send(mobiusLoginFail(0x15))
			f.drain()
		})
		ctx := testContext(t)
		c, err := DialLogin(ctx, addr)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = c.Close() }()
		_, err = c.ServerList(ctx)
		var lfe *LoginFailError
		if !errors.As(err, &lfe) || lfe.Reason != LoginFailAccessFailed {
			t.Errorf("got %v, want LoginFailError with reason 0x15", err)
		}
	})
	t.Run("AuthGameGuard", func(t *testing.T) {
		addr := startFakeLogin(t, func(f *fakeLogin) {
			f.sendInit()
			f.recv()
			f.send(mobiusLoginFail(0x15))
			f.drain()
		})
		ctx := testContext(t)
		c, err := DialLogin(ctx, addr)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = c.Close() }()
		err = c.AuthGameGuard(ctx)
		var lfe *LoginFailError
		if !errors.As(err, &lfe) || lfe.Reason != LoginFailAccessFailed {
			t.Errorf("got %v, want LoginFailError with reason 0x15", err)
		}
	})
}

func TestAuthGameGuardWrongResponse(t *testing.T) {
	addr := startFakeLogin(t, func(f *fakeLogin) {
		f.sendInit()
		f.recv()
		f.send(mobiusGGAuth(testSessionID + 1))
		f.drain()
	})
	ctx := testContext(t)
	c, err := DialLogin(ctx, addr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	if err := c.AuthGameGuard(ctx); !errors.Is(err, ErrMalformed) {
		t.Errorf("got %v, want ErrMalformed", err)
	}
}

func TestLoginServerCloses(t *testing.T) {
	addr := startFakeLogin(t, func(f *fakeLogin) {
		f.sendInit()
		f.recv()
	})
	ctx := testContext(t)
	c, err := DialLogin(ctx, addr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	if err := c.AuthGameGuard(ctx); !errors.Is(err, ErrServerClosed) {
		t.Errorf("got %v, want ErrServerClosed", err)
	}
}

func TestLoginUnexpectedPackets(t *testing.T) {
	addr := startFakeLogin(t, func(f *fakeLogin) {
		f.sendInit()
		f.recv()
		f.send(mobiusPlayOk(1, 2))
		f.recv()
		f.send(msg(0x42).u32(0xdeadbeef).b)
		f.drain()
	})
	ctx := testContext(t)
	c, err := DialLogin(ctx, addr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	_, err = c.ServerList(ctx)
	var upe *UnexpectedPacketError
	if !errors.As(err, &upe) || upe.Name != "PlayOk" || upe.Want != "ServerList" || upe.Server != "login" {
		t.Errorf("PlayOk instead of ServerList: got %#v", err)
	}

	_, err = c.ServerList(ctx)
	if !errors.As(err, &upe) || upe.Opcode != 0x42 || upe.Name != "" || !strings.Contains(err.Error(), "opcode 0x42") {
		t.Errorf("unknown opcode: got %v", err)
	}
	if !bytes.HasPrefix(upe.Body, []byte{0xef, 0xbe, 0xad, 0xde}) {
		t.Errorf("unknown packet body % x", upe.Body)
	}
}

func TestLoginRaw(t *testing.T) {
	addr := startFakeLogin(t, func(f *fakeLogin) {
		f.sendInit()
		f.expect([]byte{0x05, 1, 2, 3})
		f.send(msg(0x42).u32(0xdeadbeef).b)
		f.send(mobiusLoginFail(0x01))
		f.send(mobiusPlayFail(0x01))
	})
	ctx := testContext(t)
	c, err := DialLogin(ctx, addr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	if err := c.SendRaw(ctx, 0x05, []byte{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
	op, body, err := c.ReadRaw(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// A 5-byte body is padded to 24 bytes, 20 before the checksum: the
	// opcode, the 4 data bytes and 15 zeros.
	wantBody := append([]byte{0xef, 0xbe, 0xad, 0xde}, make([]byte, 15)...)
	if op != 0x42 || !bytes.Equal(body, wantBody) {
		t.Errorf("ReadRaw = 0x%02x % x, want 0x42 % x", op, body, wantBody)
	}
	p, err := c.ReadPacket(ctx)
	if err != nil || p != (LoginFail{Reason: LoginFailSystemErrorLoginLater}) {
		t.Errorf("ReadPacket = %#v, %v", p, err)
	}
	// The PlayFail arrives before the close.
	var upe *UnexpectedPacketError
	if err := c.ExpectClosed(ctx); !errors.As(err, &upe) || upe.Name != "PlayFail" {
		t.Errorf("ExpectClosed with a packet pending: got %v", err)
	}
	if err := c.ExpectClosed(ctx); err != nil {
		t.Errorf("ExpectClosed: %v", err)
	}
}

func TestLoginContext(t *testing.T) {
	addr := startFakeLogin(t, func(f *fakeLogin) {
		f.sendInit()
		f.drain()
	})
	c, err := DialLogin(testContext(t), addr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	if err := c.AuthGameGuard(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("deadline: got %v, want context.DeadlineExceeded", err)
	}

	ctx, cancel = context.WithCancel(t.Context())
	time.AfterFunc(50*time.Millisecond, cancel)
	if err := c.AuthGameGuard(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("cancel: got %v, want context.Canceled", err)
	}

	if err := c.AuthGameGuard(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled before the call: got %v, want context.Canceled", err)
	}
}
