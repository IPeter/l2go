package loginserver

import (
	"bytes"
	"context"
	"errors"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/link"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
)

// End-to-end tests: a real Server on 127.0.0.1:0 with an in-memory account
// store, driven by the fake Interlude client. Game servers register on the
// link with link.Client, as the real game server does.

const (
	secret1 = "secret-1"
	secret7 = "secret-7"
)

// e2eConfig has game server 1 and, if withTesting, testing server 7. The
// test client connects from 127.0.0.1, an internal network, so it gets
// the internal IPs.
func e2eConfig(withTesting bool) config.Config {
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.Secret = secret1
	gs.InternalIP = "10.1.2.3"
	gs.ExternalIP = "203.0.113.7"
	gs.Port = 7777
	gs.Options.MaxPlayers = 1000
	if withTesting {
		ts := cfg.GameServers[0]
		ts.ID = 7
		ts.Name = "Test"
		ts.Secret = secret7
		ts.Port = 7778
		ts.Options.Testing = true
		cfg.GameServers = append(cfg.GameServers, ts)
	}
	return cfg
}

// e2e is a running login server. Its methods that take a *testing.T may be
// used from subtests.
type e2e struct {
	t     *testing.T // the top-level test, for setup
	s     *Server
	r     running
	store storage.AccountStore
}

func startE2E(t *testing.T, cfg config.Config, store storage.AccountStore) *e2e {
	t.Helper()
	if store == nil {
		store = memstore.NewAccounts()
	}
	s := newTestServer(t, cfg, store)
	return &e2e{t: t, s: s, r: start(t, s), store: store}
}

func testContext(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// addAccount stores an account with the given password and level.
func (e *e2e) addAccount(user, password string, level storage.AccessLevel) {
	e.t.Helper()
	acc := storage.Account{Username: user, Password: hash(e.t, password), AccessLevel: level}
	if err := e.store.Create(e.t.Context(), acc); err != nil {
		e.t.Fatal(err)
	}
}

// gameServer registers game server id on the link and waits until the
// login server sees it online. stop disconnects it.
func (e *e2e) gameServer(id int, secret string, maxPlayers int) (c *link.Client, stop func()) {
	e.t.Helper()
	c = link.NewClient(link.ClientConfig{
		Addr:       e.r.gsAddr,
		ServerID:   id,
		Secret:     secret,
		MaxPlayers: maxPlayers,
		MinBackoff: 10 * time.Millisecond,
		MaxBackoff: 50 * time.Millisecond,
	}, nil)
	ctx, cancel := context.WithCancel(e.t.Context())
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx) }()
	stop = sync.OnceFunc(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				e.t.Errorf("link client %d: %v", id, err)
			}
		case <-time.After(5 * time.Second):
			e.t.Errorf("link client %d did not stop", id)
		}
	})
	e.t.Cleanup(stop)
	waitFor(e.t, "game server registered", func() bool {
		st, _ := e.s.link.Status(id)
		return st.Online
	})
	return c, stop
}

// rawConn connects and reads Init, without the GameGuard handshake.
func (e *e2e) rawConn(t *testing.T, ctx context.Context) *testclient.LoginConn {
	t.Helper()
	c, err := testclient.DialLogin(ctx, e.r.clientAddr)
	if err != nil {
		t.Fatalf("DialLogin: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// dial connects and passes the GameGuard handshake.
func (e *e2e) dial(t *testing.T, ctx context.Context) *testclient.LoginConn {
	t.Helper()
	c := e.rawConn(t, ctx)
	if err := c.AuthGameGuard(ctx); err != nil {
		t.Fatalf("AuthGameGuard: %v", err)
	}
	return c
}

// login logs in on a new connection and returns the answer.
func (e *e2e) login(t *testing.T, ctx context.Context, user, password string, newAuth bool) (*testclient.LoginConn, testclient.LoginResult) {
	t.Helper()
	c := e.dial(t, ctx)
	res, err := c.Login(ctx, user, password, newAuth)
	if err != nil {
		t.Fatalf("Login(%q): %v", user, err)
	}
	return c, res
}

// loggedIn logs in on a new connection and expects LoginOk.
func (e *e2e) loggedIn(t *testing.T, ctx context.Context, user, password string) *testclient.LoginConn {
	t.Helper()
	c, res := e.login(t, ctx, user, password, false)
	if res.Ok == nil {
		t.Fatalf("Login(%q) = %v, want LoginOk", user, res)
	}
	return c
}

// expectLoginClosed checks that the server closes c without sending
// anything else.
func expectLoginClosed(t *testing.T, ctx context.Context, c *testclient.LoginConn) {
	t.Helper()
	if err := c.ExpectClosed(ctx); err != nil {
		t.Errorf("connection not closed: %v", err)
	}
}

func TestE2ENewAccount(t *testing.T) {
	e := startE2E(t, e2eConfig(false), nil)
	ctx := testContext(t)

	c := e.loggedIn(t, ctx, "Newbie", "pw1")
	if in := c.Init(); in.GameGuard != testclient.MobiusGameGuard {
		t.Errorf("Init GameGuard words = %x, want Mobius's", in.GameGuard)
	}
	acc, err := e.store.FindByUsername(ctx, "newbie")
	if err != nil {
		t.Fatalf("account not created under the lowercased name: %v", err)
	}
	if acc.AccessLevel != storage.AccessLevelPlayer {
		t.Errorf("new account level = %d, want player", acc.AccessLevel)
	}
	if _, err := e.store.FindByUsername(ctx, "Newbie"); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("account also stored as typed: %v", err)
	}
	k := c.Keys()
	if !e.s.sessions.CheckLogin("newbie", k.LoginOK1, k.LoginOK2) {
		t.Error("the LoginOk keys are not the issued session keys")
	}
	if st := e.s.Stats(); st.AccountsCreated != 1 {
		t.Errorf("AccountsCreated = %d, want 1", st.AccountsCreated)
	}
}

func TestE2EExistingAccount(t *testing.T) {
	e := startE2E(t, e2eConfig(false), nil)
	e.addAccount("alice", "secret", storage.AccessLevelPlayer)
	ctx := testContext(t)

	e.loggedIn(t, ctx, "alice", "secret")
	e.loggedIn(t, ctx, "ALICE", "secret") // names are case-insensitive, as in Mobius
	if st := e.s.Stats(); st.AccountsCreated != 0 || st.Logins != 2 {
		t.Errorf("stats = %+v, want 2 logins and no new account", st)
	}
}

func TestE2EWrongPassword(t *testing.T) {
	e := startE2E(t, e2eConfig(false), nil)
	e.addAccount("alice", "secret", storage.AccessLevelPlayer)
	ctx := testContext(t)

	c, res := e.login(t, ctx, "alice", "wrong", false)
	if res.Fail == nil || res.Fail.Reason != testclient.LoginFailUserOrPassWrong {
		t.Fatalf("Login = %v, want LoginFail user or password wrong", res)
	}
	expectLoginClosed(t, ctx, c)
	if st := e.s.Stats(); st.FailedLogins != 1 {
		t.Errorf("FailedLogins = %d, want 1", st.FailedLogins)
	}
}

func TestE2EUnknownAccountWithoutAutoCreate(t *testing.T) {
	cfg := e2eConfig(false)
	cfg.LoginServer.AutoCreate = false
	e := startE2E(t, cfg, nil)
	ctx := testContext(t)

	c, res := e.login(t, ctx, "nobody", "pw", false)
	if res.Fail == nil || res.Fail.Reason != testclient.LoginFailUserOrPassWrong {
		t.Fatalf("Login = %v, want LoginFail user or password wrong", res)
	}
	expectLoginClosed(t, ctx, c)
}

func TestE2EBannedAccount(t *testing.T) {
	e := startE2E(t, e2eConfig(false), nil)
	e.addAccount("cheater", "pw", storage.AccessLevelBanned)
	ctx := testContext(t)

	c, res := e.login(t, ctx, "cheater", "pw", false)
	if res.Kicked == nil || res.Kicked.Reason != testclient.AccountKickedPermanentlyBanned {
		t.Fatalf("Login = %v, want AccountKicked permanently banned", res)
	}
	expectLoginClosed(t, ctx, c)
}

func TestE2EStoreError(t *testing.T) {
	e := startE2E(t, e2eConfig(false), brokenStore{errors.New("database down")})
	ctx := testContext(t)

	c, res := e.login(t, ctx, "anyone", "pw", false)
	if res.Fail == nil || res.Fail.Reason != testclient.LoginFailSystemErrorLoginLater {
		t.Fatalf("Login = %v, want LoginFail system error", res)
	}
	expectLoginClosed(t, ctx, c)
}

// The newer client sends the credentials in two RSA blocks (256 bytes),
// with room for longer account names.
func TestE2ENewAuthLogin(t *testing.T) {
	e := startE2E(t, e2eConfig(false), nil)
	e.addAccount("bob", "pw", storage.AccessLevelPlayer)
	ctx := testContext(t)

	for _, user := range []string{"bob", "AVeryVeryLongAccountNameOver14"} {
		c, res := e.login(t, ctx, user, "pw", true)
		if res.Ok == nil {
			t.Fatalf("Login(%q, newAuth) = %v, want LoginOk", user, res)
		}
		if _, err := c.ServerList(ctx); err != nil {
			t.Errorf("ServerList after the 256-byte login: %v", err)
		}
	}
	if _, err := e.store.FindByUsername(ctx, "averyverylongaccountnameover14"); err != nil {
		t.Errorf("long account not created: %v", err)
	}

	c, res := e.login(t, ctx, "bob", "wrong", true)
	if res.Fail == nil || res.Fail.Reason != testclient.LoginFailUserOrPassWrong {
		t.Errorf("wrong password with newAuth = %v, want LoginFail", res)
	}
	expectLoginClosed(t, ctx, c)
}

func TestE2EWrongSessionKeys(t *testing.T) {
	e := startE2E(t, e2eConfig(false), nil)
	e.gameServer(1, secret1, 100)
	ctx := testContext(t)

	t.Run("server list", func(t *testing.T) {
		c := e.loggedIn(t, ctx, "keys1", "pw")
		k := c.Keys()
		c.SetLoginKeys(k.LoginOK1+1, k.LoginOK2)
		_, err := c.ServerList(ctx)
		var lf *testclient.LoginFailError
		if !errors.As(err, &lf) || lf.Reason != testclient.LoginFailAccessFailed {
			t.Fatalf("ServerList = %v, want LoginFail access failed", err)
		}
		expectLoginClosed(t, ctx, c)
	})

	t.Run("server login", func(t *testing.T) {
		c := e.loggedIn(t, ctx, "keys2", "pw")
		k := c.Keys()
		c.SetLoginKeys(k.LoginOK1, ^k.LoginOK2)
		res, err := c.ServerLogin(ctx, 1)
		if err != nil {
			t.Fatal(err)
		}
		if res.LoginFail == nil || res.LoginFail.Reason != testclient.LoginFailAccessFailed {
			t.Fatalf("ServerLogin = %v, want LoginFail access failed", res)
		}
		expectLoginClosed(t, ctx, c)
	})

	t.Run("keys of another account", func(t *testing.T) {
		a := e.loggedIn(t, ctx, "keys3", "pw")
		b := e.loggedIn(t, ctx, "keys4", "pw")
		ka := a.Keys()
		b.SetLoginKeys(ka.LoginOK1, ka.LoginOK2)
		if _, err := b.ServerList(ctx); err == nil {
			t.Fatal("ServerList with another account's keys succeeded")
		}
		if _, err := a.ServerList(ctx); err != nil {
			t.Errorf("the owner of the keys lost its list: %v", err)
		}
	})

	t.Run("replaced by a new login", func(t *testing.T) {
		old := e.loggedIn(t, ctx, "keys5", "pw")
		cur := e.loggedIn(t, ctx, "keys5", "pw")
		if _, err := old.ServerList(ctx); err == nil {
			t.Error("the first connection's keys still work after a second login")
		}
		if _, err := cur.ServerList(ctx); err != nil {
			t.Errorf("the second connection's keys: %v", err)
		}
	})

	if got := e.s.Stats().HackAttempts; got < 3 {
		t.Errorf("HackAttempts = %d, want at least 3", got)
	}
}

// A packet in the wrong state, an unknown one or one too short for its
// layout is ignored, as in Mobius: no answer, and the connection carries on
// in the same state. Each case sends such a packet and then the right next
// request on the same connection. Had the server answered the ignored
// packet, that answer would arrive in place of the expected one; had it
// closed the connection, the next request would fail.
func TestE2EPacketsOutOfOrder(t *testing.T) {
	e := startE2E(t, e2eConfig(false), nil)
	ctx := testContext(t)

	send := func(t *testing.T, c *testclient.LoginConn, opcode byte, body []byte) {
		t.Helper()
		if err := c.SendRaw(ctx, opcode, body); err != nil {
			t.Fatal(err)
		}
	}
	gameGuard := func(c *testclient.LoginConn) []byte {
		id := c.Init().SessionID
		return append([]byte{byte(id), byte(id >> 8), byte(id >> 16), byte(id >> 24)}, make([]byte, 16)...)
	}
	authGameGuard := func(t *testing.T, c *testclient.LoginConn) {
		t.Helper()
		if err := c.AuthGameGuard(ctx); err != nil {
			t.Fatalf("AuthGameGuard after the ignored packet: %v", err)
		}
	}
	login := func(t *testing.T, c *testclient.LoginConn, user string) {
		t.Helper()
		res, err := c.Login(ctx, user, "pw", false)
		if err != nil || res.Ok == nil {
			t.Fatalf("Login after the ignored packet = %v, %v; want LoginOk", res, err)
		}
	}
	serverList := func(t *testing.T, c *testclient.LoginConn) {
		t.Helper()
		if _, err := c.ServerList(ctx); err != nil {
			t.Fatalf("ServerList after the ignored packet: %v", err)
		}
	}
	zeroBlock := make([]byte, 128) // decrypts to empty credentials

	t.Run("login before GameGuard", func(t *testing.T) {
		c := e.rawConn(t, ctx)
		send(t, c, testclient.OpRequestAuthLogin, zeroBlock)
		authGameGuard(t, c)
		login(t, c, "order1")
	})
	t.Run("server list and server login before GameGuard", func(t *testing.T) {
		c := e.rawConn(t, ctx)
		send(t, c, testclient.OpRequestServerList, make([]byte, 8))
		send(t, c, testclient.OpRequestServerLogin, append(make([]byte, 8), 1))
		authGameGuard(t, c)
	})
	t.Run("GameGuard twice", func(t *testing.T) {
		c := e.dial(t, ctx)
		send(t, c, testclient.OpAuthGameGuard, gameGuard(c))
		login(t, c, "order2")
	})
	t.Run("server list and server login before login", func(t *testing.T) {
		c := e.dial(t, ctx)
		send(t, c, testclient.OpRequestServerList, make([]byte, 8))
		send(t, c, testclient.OpRequestServerLogin, append(make([]byte, 8), 1))
		login(t, c, "order3")
	})
	t.Run("login twice", func(t *testing.T) {
		c := e.loggedIn(t, ctx, "order4", "pw")
		send(t, c, testclient.OpRequestAuthLogin, zeroBlock)
		serverList(t, c)
	})
	t.Run("GameGuard after login", func(t *testing.T) {
		c := e.loggedIn(t, ctx, "order5", "pw")
		send(t, c, testclient.OpAuthGameGuard, gameGuard(c))
		serverList(t, c)
	})
	t.Run("unknown opcodes after login", func(t *testing.T) {
		c := e.loggedIn(t, ctx, "order6", "pw")
		send(t, c, 0x0e, nil) // PI agreement check, sent by later clients
		send(t, c, 0x0f, []byte{1})
		send(t, c, 0x99, []byte{1, 2, 3})
		serverList(t, c)
	})
	t.Run("unknown opcode first", func(t *testing.T) {
		c := e.rawConn(t, ctx)
		send(t, c, 0x99, []byte{1, 2, 3})
		authGameGuard(t, c)
	})
	t.Run("short packets", func(t *testing.T) {
		c := e.rawConn(t, ctx)
		send(t, c, testclient.OpAuthGameGuard, nil) // 15 bytes with padding, 20 needed
		authGameGuard(t, c)
		send(t, c, testclient.OpRequestAuthLogin, make([]byte, 10)) // 128 needed
		login(t, c, "order7")
	})
	t.Run("RSA block out of range", func(t *testing.T) {
		c := e.dial(t, ctx)
		block := bytes.Repeat([]byte{0xff}, 128) // larger than any 1024-bit modulus
		send(t, c, testclient.OpRequestAuthLogin, block)
		login(t, c, "order8")
	})

	// AuthGameGuard with the wrong session ID is answered (Mobius
	// AuthGameGuard.run), and L2Go then closes the connection.
	t.Run("GameGuard with a wrong session ID", func(t *testing.T) {
		c := e.rawConn(t, ctx)
		body := gameGuard(c)
		body[0]++
		send(t, c, testclient.OpAuthGameGuard, body)
		p, err := c.ReadPacket(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if lf, ok := p.(testclient.LoginFail); !ok || lf.Reason != testclient.LoginFailAccessFailed {
			t.Fatalf("answer = %#v, want LoginFail access failed", p)
		}
		expectLoginClosed(t, ctx, c)
	})

	if got := e.s.Stats().HackAttempts; got < 2 {
		t.Errorf("HackAttempts = %d, want at least 2 (RSA block, GameGuard session ID)", got)
	}
}

// B3: server ID 0 and IDs that are not configured are refused, without
// touching any other server.
func TestE2EServerLoginBadIDs(t *testing.T) {
	e := startE2E(t, e2eConfig(false), nil)
	e.gameServer(1, secret1, 100)
	ctx := testContext(t)

	for _, id := range []byte{0, 2, 128, 255} {
		c := e.loggedIn(t, ctx, "badid", "pw")
		res, err := c.ServerLogin(ctx, id)
		if err != nil {
			t.Fatalf("ServerLogin(%d): %v", id, err)
		}
		if res.Fail == nil || res.Fail.Reason != testclient.PlayFailServerOverloaded {
			t.Errorf("ServerLogin(%d) = %v, want PlayFail", id, res)
		}
		expectLoginClosed(t, ctx, c)
	}
	if got := e.s.Stats().HackAttempts; got < 4 {
		t.Errorf("HackAttempts = %d, want at least 4", got)
	}

	// The real server still works.
	c := e.loggedIn(t, ctx, "badid", "pw")
	if res, err := c.ServerLogin(ctx, 1); err != nil || res.Ok == nil {
		t.Errorf("ServerLogin(1) = %v, %v; want PlayOk", res, err)
	}
}

func TestE2EServerLoginOffline(t *testing.T) {
	e := startE2E(t, e2eConfig(false), nil)
	ctx := testContext(t)

	c := e.loggedIn(t, ctx, "early", "pw")
	res, err := c.ServerLogin(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if res.Fail == nil || res.Fail.Reason != testclient.PlayFailServerOverloaded {
		t.Fatalf("ServerLogin to an offline server = %v, want PlayFail", res)
	}
	expectLoginClosed(t, ctx, c)
}

func TestE2ETestingServer(t *testing.T) {
	e := startE2E(t, e2eConfig(true), nil)
	e.addAccount("gm", "pw", storage.AccessLevelAdmin)
	e.gameServer(1, secret1, 100)
	e.gameServer(7, secret7, 100)
	ctx := testContext(t)

	player := e.loggedIn(t, ctx, "player", "pw")
	list, err := player.ServerList(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Servers) != 1 || list.Servers[0].ID != 1 {
		t.Errorf("player sees %+v, want only server 1", list.Servers)
	}
	res, err := player.ServerLogin(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if res.Fail == nil || res.Fail.Reason != testclient.PlayFailServerOverloaded {
		t.Errorf("player on the testing server = %v, want PlayFail", res)
	}
	expectLoginClosed(t, ctx, player)

	gm := e.loggedIn(t, ctx, "gm", "pw")
	list, err = gm.ServerList(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Servers) != 2 || list.Servers[1].ID != 7 || !list.Servers[1].Up {
		t.Errorf("admin sees %+v, want servers 1 and 7, both up", list.Servers)
	}
	res, err = gm.ServerLogin(ctx, 7)
	if err != nil || res.Ok == nil {
		t.Errorf("admin on the testing server = %v, %v; want PlayOk", res, err)
	}
}

// F4: the list shows a game server as up with its players while it is
// registered on the link, and as down after it disconnects.
func TestE2EServerListLiveStatus(t *testing.T) {
	e := startE2E(t, e2eConfig(false), nil)
	ctx := testContext(t)
	c := e.loggedIn(t, ctx, "watcher", "pw")

	server := func() testclient.ServerInfo {
		t.Helper()
		list, err := c.ServerList(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(list.Servers) != 1 {
			t.Fatalf("list has %d servers, want 1", len(list.Servers))
		}
		if list.Unknown != 0 || len(list.Characters) != 0 || list.LastServer != storage.DefaultLastServer {
			t.Errorf("trailer = %d, %+v, last %d; want 0, no character data, 1", list.Unknown, list.Characters, list.LastServer)
		}
		return list.Servers[0]
	}

	before := server()
	want := testclient.ServerInfo{
		ID: 1, IP: netip.MustParseAddr("10.1.2.3"), Port: 7777, PvP: true,
		MaxPlayers: 1000, ServerType: 1,
	}
	if before != want {
		t.Errorf("before registration: %+v, want %+v", before, want)
	}

	gs, stop := e.gameServer(1, secret1, 500)
	gs.SetPlayers(42)
	waitFor(t, "player count", func() bool {
		st, _ := e.s.link.Status(1)
		return st.Players == 42
	})
	up := server()
	if !up.Up || up.CurrentPlayers != 42 || up.MaxPlayers != 500 {
		t.Errorf("registered: %+v, want up with 42/500", up)
	}
	if got := e.s.Stats().GameServers; got != 1 {
		t.Errorf("Stats().GameServers = %d, want 1", got)
	}

	stop()
	waitFor(t, "game server offline", func() bool {
		st, _ := e.s.link.Status(1)
		return !st.Online
	})
	down := server()
	if down.Up || down.CurrentPlayers != 0 || down.MaxPlayers != 500 {
		t.Errorf("after disconnect: %+v, want down with 0/500", down)
	}
}

// F5/F6: the keys from PlayOk let the player into the game server once.
// The account name is the lowercased one, which the game client sends.
func TestE2EPlayKeysReachGameServer(t *testing.T) {
	e := startE2E(t, e2eConfig(false), nil)
	gs, _ := e.gameServer(1, secret1, 100)
	ctx := testContext(t)

	keys, err := testclient.LoginAndSelect(ctx, e.r.clientAddr, "Hero", "pw", 1)
	if err != nil {
		t.Fatalf("LoginAndSelect: %v", err)
	}
	// LoginAndSelect closed the login connection; the keys stay valid.
	ok, err := gs.VerifySession(ctx, "hero", keys)
	if err != nil || !ok {
		t.Fatalf("VerifySession = %v, %v; want true", ok, err)
	}
	if ok, _ := gs.VerifySession(ctx, "hero", keys); ok {
		t.Error("the keys worked twice")
	}

	keys, err = testclient.LoginAndSelect(ctx, e.r.clientAddr, "hero", "pw", 1)
	if err != nil {
		t.Fatal(err)
	}
	wrong := keys
	wrong.PlayOK1++
	if ok, _ := gs.VerifySession(ctx, "hero", wrong); ok {
		t.Error("wrong play keys were accepted")
	}
	if ok, _ := gs.VerifySession(ctx, "Hero", keys); ok {
		t.Error("the keys were accepted for a differently spelled account")
	}
	if ok, _ := gs.VerifySession(ctx, "hero", keys); !ok {
		t.Error("the right keys were refused after the failed attempts")
	}
}

// The server list preselects the account's last server: 1 for a new
// account, as the Mobius column default, then the server it last joined.
// As in Mobius, the last server is stored when RequestServerLogin is
// accepted, and the connection that joined keeps the value it read at login.
func TestE2ELastServer(t *testing.T) {
	e := startE2E(t, e2eConfig(true), nil)
	e.addAccount("gm", "pw", storage.AccessLevelAdmin)
	e.gameServer(1, secret1, 100)
	e.gameServer(7, secret7, 100)
	ctx := testContext(t)

	lastServer := func(t *testing.T, c *testclient.LoginConn) byte {
		t.Helper()
		list, err := c.ServerList(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return list.LastServer
	}
	stored := func(t *testing.T) int {
		t.Helper()
		acc, err := e.store.FindByUsername(ctx, "gm")
		if err != nil {
			t.Fatal(err)
		}
		return acc.LastServer
	}

	c := e.loggedIn(t, ctx, "gm", "pw")
	if got := lastServer(t, c); got != storage.DefaultLastServer {
		t.Errorf("new account: LastServer = %d, want %d", got, storage.DefaultLastServer)
	}
	if res, err := c.ServerLogin(ctx, 7); err != nil || res.Ok == nil {
		t.Fatalf("ServerLogin(7) = %v, %v; want PlayOk", res, err)
	}
	if got := stored(t); got != 7 {
		t.Errorf("stored LastServer = %d, want 7", got)
	}
	if got := lastServer(t, c); got != storage.DefaultLastServer {
		t.Errorf("same connection after joining 7: LastServer = %d, want the value read at login", got)
	}

	c = e.loggedIn(t, ctx, "gm", "pw")
	if got := lastServer(t, c); got != 7 {
		t.Errorf("next login: LastServer = %d, want 7", got)
	}

	// A refused server login keeps the last server.
	player := e.loggedIn(t, ctx, "player", "pw")
	if res, err := player.ServerLogin(ctx, 7); err != nil || res.Fail == nil {
		t.Fatalf("player on the testing server = %v, %v; want PlayFail", res, err)
	}
	if acc, err := e.store.FindByUsername(ctx, "player"); err != nil || acc.LastServer != storage.DefaultLastServer {
		t.Errorf("after PlayFail: %+v, %v; want LastServer %d", acc, err, storage.DefaultLastServer)
	}
}

// A last server that cannot be stored is logged; the client still gets
// PlayOk, as in Mobius.
func TestE2ELastServerWriteFails(t *testing.T) {
	e := startE2E(t, e2eConfig(true), lastServerFails{memstore.NewAccounts(), errors.New("database down")})
	e.addAccount("gm", "pw", storage.AccessLevelAdmin)
	e.gameServer(7, secret7, 100)
	ctx := testContext(t)

	c := e.loggedIn(t, ctx, "gm", "pw")
	if res, err := c.ServerLogin(ctx, 7); err != nil || res.Ok == nil {
		t.Fatalf("ServerLogin(7) = %v, %v; want PlayOk", res, err)
	}
}

type lastServerFails struct {
	*memstore.Accounts
	err error
}

func (l lastServerFails) SetLastServer(context.Context, string, int) error { return l.err }

// After PlayOk the connection stays open, as in Mobius; the client closes
// it. A second RequestServerLogin is answered again.
func TestE2EPlayOkKeepsConnection(t *testing.T) {
	e := startE2E(t, e2eConfig(false), nil)
	e.gameServer(1, secret1, 100)
	ctx := testContext(t)

	c := e.loggedIn(t, ctx, "stay", "pw")
	for range 2 {
		res, err := c.ServerLogin(ctx, 1)
		if err != nil || res.Ok == nil {
			t.Fatalf("ServerLogin = %v, %v; want PlayOk", res, err)
		}
	}
	if k := c.Keys(); !e.s.sessions.Consume("stay", k) {
		t.Error("PlayOk did not carry the issued play keys")
	}
}
