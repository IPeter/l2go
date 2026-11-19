package gameserver

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/link"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/session"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
)

const testSecret = "test-secret"

// testData is the real data/ folder, loaded once.
var testData = sync.OnceValues(func() (*data.Data, error) {
	return data.Load(filepath.Join("..", "..", "data"))
})

func loadData(t *testing.T) *data.Data {
	t.Helper()
	d, err := testData()
	if err != nil {
		t.Fatalf("loading data/: %v", err)
	}
	return d
}

// testCtx bounds one test's client calls.
func testCtx(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func listen(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return ln
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// envOptions change the test environment.
type envOptions struct {
	scripts Scripts
	// login replaces the real link to a link.Server.
	login LoginLink
	// chars and ids are shared with an earlier environment (a restart).
	chars *memstore.Characters
	ids   *memstore.IDs
	// configure changes the game server config.
	configure func(*config.GameServer)
	// server changes the server before it starts serving.
	server func(*Server)
}

// env is a running game server with an in-memory store, the real game
// data, and a real login link to a link.Server whose session store issues
// the keys.
type env struct {
	srv      *Server
	addr     string
	sessions *session.Store
	linkSrv  *link.Server // nil when envOptions.login is set
	chars    *memstore.Characters
	ids      *memstore.IDs
	stop     func() error
}

func newEnv(t *testing.T, opts envOptions) *env {
	t.Helper()
	e := &env{chars: opts.chars, ids: opts.ids, sessions: session.NewStore(0)}
	if e.chars == nil {
		e.chars = memstore.NewCharacters()
	}
	if e.ids == nil {
		e.ids = memstore.NewIDs()
	}

	// The link outlives the game server: this cleanup runs last.
	bgCtx, stopBackground := context.WithCancel(context.Background())
	var bg sync.WaitGroup
	t.Cleanup(func() {
		stopBackground()
		bg.Wait()
	})

	login := opts.login
	if login == nil {
		ln := listen(t)
		e.linkSrv = link.NewServer(link.ServerConfig{GameServers: map[int]string{1: testSecret}}, e.sessions, nil)
		bg.Go(func() { _ = e.linkSrv.Serve(bgCtx, ln) })
		lc := link.NewClient(link.ClientConfig{
			Addr:       ln.Addr().String(),
			ServerID:   1,
			Secret:     testSecret,
			MaxPlayers: 100,
			MinBackoff: 10 * time.Millisecond,
			MaxBackoff: 100 * time.Millisecond,
		}, nil)
		bg.Go(func() { _ = lc.Run(bgCtx) })
		waitFor(t, "the link to register", lc.Connected)
		login = lc
	}

	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.Secret = testSecret
	if opts.configure != nil {
		opts.configure(gs)
	}
	s, err := New(cfg, 1, Deps{
		Characters: e.chars,
		IDs:        e.ids,
		Data:       loadData(t),
		Login:      login,
		Scripts:    opts.scripts,
	}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if opts.server != nil {
		opts.server(s)
	}
	e.srv = s

	ln := listen(t)
	e.addr = ln.Addr().String()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Serve(ctx, ln) }()
	e.stop = sync.OnceValue(func() error {
		cancel()
		select {
		case err := <-done:
			return err
		case <-time.After(5 * time.Second):
			return errors.New("server did not stop within 5s")
		}
	})
	t.Cleanup(func() { _ = e.stop() })
	return e
}

// issue makes the login server's session keys for account.
func (e *env) issue(t *testing.T, account string) session.Keys {
	t.Helper()
	k, err := e.sessions.Issue(account)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// dial connects and does the handshake with protocol 746.
func (e *env) dial(t *testing.T) *testclient.GameConn {
	t.Helper()
	c, err := testclient.DialGame(testCtx(t), e.addr, testclient.InterludeProtocol)
	if err != nil {
		t.Fatalf("DialGame: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// login connects and authenticates account with fresh session keys.
func (e *env) login(t *testing.T, account string) (*testclient.GameConn, testclient.CharacterList) {
	t.Helper()
	keys := e.issue(t, account)
	c := e.dial(t)
	list, err := c.AuthLogin(testCtx(t), account, keys)
	if err != nil {
		t.Fatalf("AuthLogin(%q): %v", account, err)
	}
	return c, list
}

// players returns the player count the login server knows.
func (e *env) players() int {
	st, _ := e.linkSrv.Status(1)
	return st.Players
}

// authLoginBody builds an AuthLogin body without the opcode, keeping the
// account's letter case (the test client lowercases it).
func authLoginBody(account string, k session.Keys) []byte {
	w := packets.NewWriter(64)
	w.WriteString(account)
	w.WriteInt32(k.PlayOK2)
	w.WriteInt32(k.PlayOK1)
	w.WriteInt32(k.LoginOK1)
	w.WriteInt32(k.LoginOK2)
	return w.Bytes()
}

// wantLoginFail checks that AuthLogin got LoginFail and the server hung up.
func wantLoginFail(t *testing.T, c *testclient.GameConn, err error) {
	t.Helper()
	var lf *testclient.GameLoginFailError
	if !errors.As(err, &lf) {
		t.Fatalf("AuthLogin error = %v, want LoginFail", err)
	}
	if lf.Reason != testclient.GameLoginFailSystemErrorLoginLater {
		t.Errorf("LoginFail reason %d, want %d", lf.Reason, testclient.GameLoginFailSystemErrorLoginLater)
	}
	if err := c.ExpectClosed(testCtx(t)); err != nil {
		t.Error(err)
	}
}

func TestNewChecksArguments(t *testing.T) {
	d := loadData(t)
	full := Deps{
		Characters: memstore.NewCharacters(),
		IDs:        memstore.NewIDs(),
		Data:       d,
		Login:      link.NewClient(link.ClientConfig{}, nil),
	}
	logger := slog.New(slog.DiscardHandler)
	if _, err := New(config.Default(), 1, full, logger); err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := New(config.Default(), 2, full, logger); err == nil {
		t.Error("New succeeded for a game server ID that is not configured")
	}
	for name, change := range map[string]func(*Deps){
		"no characters": func(d *Deps) { d.Characters = nil },
		"no IDs":        func(d *Deps) { d.IDs = nil },
		"no data":       func(d *Deps) { d.Data = nil },
		"no login":      func(d *Deps) { d.Login = nil },
	} {
		deps := full
		change(&deps)
		if _, err := New(config.Default(), 1, deps, logger); err == nil {
			t.Errorf("%s: New succeeded", name)
		}
	}
}

func TestPing(t *testing.T) {
	e := newEnv(t, envOptions{})
	if err := testclient.Ping(testCtx(t), e.addr); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	waitFor(t, "the client to be removed", func() bool { return e.srv.OnlineClients() == 0 })
}

func TestHandshake(t *testing.T) {
	e := newEnv(t, envOptions{})
	c := e.dial(t)
	k := c.Key()
	if k.Result != 1 || !k.Encryption || k.ServerID != 1 || k.Unknown != 1 || k.ObfuscationKey != 0 {
		t.Errorf("KeyPacket = %+v, want result 1, encryption on, server 1, unknown 1, no obfuscation", k)
	}
	if k.KeyHalf == ([8]byte{}) {
		t.Error("KeyPacket carries an all-zero key")
	}
	// The cipher works both ways: an encrypted request gets an answer the
	// client can decrypt.
	if _, err := c.AuthLogin(testCtx(t), "player1", e.issue(t, "player1")); err != nil {
		t.Fatalf("AuthLogin: %v", err)
	}
}

// Every connection gets its own key.
func TestHandshakeKeysDiffer(t *testing.T) {
	e := newEnv(t, envOptions{})
	a, b := e.dial(t), e.dial(t)
	if a.Key().KeyHalf == b.Key().KeyHalf {
		t.Error("two connections got the same key")
	}
}

func TestWrongProtocol(t *testing.T) {
	e := newEnv(t, envOptions{})
	ctx := testCtx(t)
	for _, v := range []int32{0, 419, 745, 747} {
		c, err := testclient.DialGame(ctx, e.addr, v)
		if !errors.Is(err, testclient.ErrWrongProtocol) {
			if c != nil {
				_ = c.Close()
			}
			t.Fatalf("protocol %d: DialGame error = %v, want ErrWrongProtocol", v, err)
		}
		if c.Key().Result != 0 {
			t.Errorf("protocol %d: KeyPacket result %d, want 0", v, c.Key().Result)
		}
		if err := c.ExpectClosed(ctx); err != nil {
			t.Errorf("protocol %d: %v", v, err)
		}
		_ = c.Close()
	}
}

// AllowedProtocols comes from the config.
func TestAllowedProtocolsFromConfig(t *testing.T) {
	e := newEnv(t, envOptions{configure: func(gs *config.GameServer) {
		gs.AllowedProtocols = []int32{740, 746}
	}})
	ctx := testCtx(t)
	c, err := testclient.DialGame(ctx, e.addr, 740)
	if err != nil {
		t.Fatalf("protocol 740: %v", err)
	}
	_ = c.Close()
}

// B2 and B10: an empty or short first packet used to crash the server or be
// used before the read error was checked. Now only that connection closes.
func TestBadFirstPacketClosesOnlyThatConnection(t *testing.T) {
	e := newEnv(t, envOptions{})
	good := e.dial(t) // a well-behaved client that must stay connected

	for _, wire := range [][]byte{
		{0x02, 0x00},             // empty body
		{0x00, 0x00},             // size 0
		{0x03, 0x00, 0x00},       // opcode without the version
		{0x05, 0x00, 0x00, 1, 2}, // truncated version
		{0x07, 0x00, 0x08, 'A', 0x00, 0x00, 0x00}, // AuthLogin before ProtocolVersion
	} {
		bad := rawDial(t, e.addr)
		if _, err := bad.Write(wire); err != nil {
			t.Fatal(err)
		}
		expectRawClosed(t, bad)
	}
	waitFor(t, "only the good client left", func() bool { return e.srv.OnlineClients() == 1 })
	if e.srv.HackAttempts() == 0 {
		t.Error("bad first packets were not counted as hack attempts")
	}
	keys := e.issue(t, "player1")
	if _, err := good.AuthLogin(testCtx(t), "player1", keys); err != nil {
		t.Fatalf("the good client stopped working: %v", err)
	}
}

func TestAuthLogin(t *testing.T) {
	e := newEnv(t, envOptions{})
	ctx := testCtx(t)
	keys := e.issue(t, "player1")
	c := e.dial(t)
	list, err := c.AuthLogin(ctx, "player1", keys)
	if err != nil {
		t.Fatalf("AuthLogin: %v", err)
	}
	if len(list.Characters) != 0 {
		t.Errorf("new account lists %d characters", len(list.Characters))
	}

	// The player count reaches the login server, and drops on disconnect.
	waitFor(t, "1 player", func() bool { return e.srv.Players() == 1 && e.players() == 1 })
	_ = c.Close()
	waitFor(t, "0 players", func() bool { return e.srv.Players() == 0 && e.players() == 0 })
}

// Mobius lowercases the account name, and the login server issues keys for
// the lowercased name.
func TestAuthLoginLowercasesAccount(t *testing.T) {
	e := newEnv(t, envOptions{})
	ctx := testCtx(t)
	keys := e.issue(t, "player1")
	c := e.dial(t)
	if err := c.SendRaw(ctx, clientpackets.OpAuthLogin, authLoginBody("PlAyEr1", keys)); err != nil {
		t.Fatal(err)
	}
	p, err := c.ReadPacket(ctx)
	if err != nil {
		t.Fatalf("reading the answer: %v", err)
	}
	if _, ok := p.(testclient.CharacterList); !ok {
		t.Fatalf("got %T, want CharacterList", p)
	}
	res, err := c.CreateCharacter(ctx, humanFighter("Casey"))
	if err != nil || !res.Created {
		t.Fatalf("CreateCharacter = %+v, %v", res, err)
	}
	if ch, _ := res.Characters.Find("Casey"); ch.Account != "player1" {
		t.Errorf("listed account %q, want player1", ch.Account)
	}
	stored, err := e.chars.ListByAccount(ctx, "player1")
	if err != nil || len(stored) != 1 {
		t.Fatalf("stored under player1: %v, %v", stored, err)
	}
}

func TestAuthLoginInvalidKeys(t *testing.T) {
	e := newEnv(t, envOptions{})
	ctx := testCtx(t)
	keys := e.issue(t, "player1")
	tests := map[string]session.Keys{
		"wrong play key 1":  {LoginOK1: keys.LoginOK1, LoginOK2: keys.LoginOK2, PlayOK1: keys.PlayOK1 + 1, PlayOK2: keys.PlayOK2},
		"wrong login key 2": {LoginOK1: keys.LoginOK1, LoginOK2: keys.LoginOK2 + 1, PlayOK1: keys.PlayOK1, PlayOK2: keys.PlayOK2},
		"play keys swapped": {LoginOK1: keys.LoginOK1, LoginOK2: keys.LoginOK2, PlayOK1: keys.PlayOK2, PlayOK2: keys.PlayOK1},
		"zero":              {},
	}
	for name, k := range tests {
		t.Run(name, func(t *testing.T) {
			c := e.dial(t)
			_, err := c.AuthLogin(ctx, "player1", k)
			wantLoginFail(t, c, err)
		})
	}
	// Keys issued for another account do not work.
	c := e.dial(t)
	_, err := c.AuthLogin(ctx, "player2", keys)
	wantLoginFail(t, c, err)
	if e.srv.Players() != 0 {
		t.Errorf("%d players after failed logins", e.srv.Players())
	}
}

// Keys work once: the login server forgets them after a successful check.
func TestAuthLoginReusedKeys(t *testing.T) {
	e := newEnv(t, envOptions{})
	ctx := testCtx(t)
	keys := e.issue(t, "player1")
	first := e.dial(t)
	if _, err := first.AuthLogin(ctx, "player1", keys); err != nil {
		t.Fatalf("first AuthLogin: %v", err)
	}
	second := e.dial(t)
	_, err := second.AuthLogin(ctx, "player1", keys)
	wantLoginFail(t, second, err)
}

// While the link to the login server is down, AuthLogin fails cleanly.
func TestAuthLoginLinkDown(t *testing.T) {
	// A link client that never runs is not connected.
	down := link.NewClient(link.ClientConfig{Addr: "127.0.0.1:1", ServerID: 1, Secret: testSecret}, nil)
	e := newEnv(t, envOptions{login: down})
	c := e.dial(t)
	_, err := c.AuthLogin(testCtx(t), "player1", session.Keys{PlayOK1: 1})
	wantLoginFail(t, c, err)
}

// blockingLink never answers a session check.
type blockingLink struct{}

func (blockingLink) VerifySession(ctx context.Context, _ string, _ session.Keys) (bool, error) {
	<-ctx.Done()
	return false, ctx.Err()
}

func (blockingLink) SetPlayers(int) {}

func TestAuthLoginLinkTimeout(t *testing.T) {
	e := newEnv(t, envOptions{
		login:  blockingLink{},
		server: func(s *Server) { s.verifyTimeout = 50 * time.Millisecond },
	})
	c := e.dial(t)
	_, err := c.AuthLogin(testCtx(t), "player1", session.Keys{PlayOK1: 1})
	wantLoginFail(t, c, err)
}

func TestAuthLoginMalformedClosesConnection(t *testing.T) {
	e := newEnv(t, envOptions{})
	ctx := testCtx(t)
	for name, body := range map[string][]byte{
		"unterminated name": {'A', 0},
		"no keys":           {'A', 0, 0, 0},
		"empty account":     authLoginBody("", session.Keys{}),
	} {
		c := e.dial(t)
		if err := c.SendRaw(ctx, clientpackets.OpAuthLogin, body); err != nil {
			t.Fatal(err)
		}
		if err := c.ExpectClosed(ctx); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// Character packets before AuthLogin are ignored, as in Mobius: no answer,
// and the connection stays open.
func TestPacketsBeforeAuthLoginAreIgnored(t *testing.T) {
	e := newEnv(t, envOptions{})
	ctx := testCtx(t)
	c := e.dial(t)
	for _, op := range []byte{
		clientpackets.OpNewCharacter,
		clientpackets.OpCharacterCreate,
		clientpackets.OpCharacterSelect,
		0x7f, // unknown
	} {
		if err := c.SendRaw(ctx, op, []byte{1, 2, 3}); err != nil {
			t.Fatal(err)
		}
	}
	// AuthLogin's answer is the next packet: nothing answered the others.
	if _, err := c.AuthLogin(ctx, "player1", e.issue(t, "player1")); err != nil {
		t.Fatalf("AuthLogin: %v", err)
	}
}

// After AuthLogin, packets that are not supported yet, unknown packets and
// a second AuthLogin are ignored.
func TestUnsupportedPacketsAreIgnored(t *testing.T) {
	e := newEnv(t, envOptions{})
	ctx := testCtx(t)
	c, _ := e.login(t, "player1")
	for _, raw := range []struct {
		op   byte
		body []byte
	}{
		{clientpackets.OpCharacterDelete, []byte{0, 0, 0, 0}},
		{clientpackets.OpCharacterSelect, []byte{0, 0, 0, 0}},
		{clientpackets.OpCharacterRestore, []byte{0, 0, 0, 0}},
		{clientpackets.OpProtocolVersion, []byte{0xea, 0x02, 0, 0}},
		{clientpackets.OpAuthLogin, authLoginBody("player1", session.Keys{})},
		{0xd0, []byte{0x01, 0x00}}, // extended packet
		{0x7f, nil},
	} {
		if err := c.SendRaw(ctx, raw.op, raw.body); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := c.NewCharacter(ctx); err != nil {
		t.Fatalf("NewCharacter after ignored packets: %v", err)
	}
	if e.srv.Players() != 1 {
		t.Errorf("%d players, want 1", e.srv.Players())
	}
}

func rawDial(t *testing.T, addr string) net.Conn {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// expectRawClosed checks that the server closed conn.
func expectRawClosed(t *testing.T, conn net.Conn) {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	_, err := io.ReadAll(conn)
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		t.Fatal("connection is still open")
	}
}

// B6, B7, B8: clients register under their own IDs, a leaving client
// removes only itself, and cancelling closes everything.
func TestServeShutdown(t *testing.T) {
	e := newEnv(t, envOptions{})

	const n = 4
	conns := make([]net.Conn, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			conn, err := net.DialTimeout("tcp", e.addr, 5*time.Second)
			if err != nil {
				t.Error(err)
				return
			}
			conns[i] = conn
			_ = e.srv.OnlineClients()
		})
	}
	wg.Wait()
	for _, c := range conns {
		if c == nil {
			t.FailNow()
		}
		t.Cleanup(func() { _ = c.Close() })
	}
	waitFor(t, "all clients registered", func() bool { return e.srv.OnlineClients() == n })

	_ = conns[0].Close()
	waitFor(t, "one client removed", func() bool { return e.srv.OnlineClients() == n-1 })

	if err := e.stop(); err != nil {
		t.Fatalf("Serve = %v, want nil after cancel", err)
	}
	for _, c := range conns[1:] {
		expectRawClosed(t, c)
	}
	if got := e.srv.OnlineClients(); got != 0 {
		t.Errorf("%d clients still registered after shutdown", got)
	}
	if c, err := net.DialTimeout("tcp", e.addr, time.Second); err == nil {
		_ = c.Close()
		t.Error("listener still accepts connections")
	}
}

// Shutting down with authenticated players reports zero players.
func TestShutdownDropsPlayers(t *testing.T) {
	e := newEnv(t, envOptions{})
	e.login(t, "player1")
	e.login(t, "player2")
	waitFor(t, "2 players", func() bool { return e.srv.Players() == 2 })
	if err := e.stop(); err != nil {
		t.Fatal(err)
	}
	if got := e.srv.Players(); got != 0 {
		t.Errorf("%d players after shutdown", got)
	}
}

func TestIdleTimeout(t *testing.T) {
	e := newEnv(t, envOptions{server: func(s *Server) { s.idleTimeout = 100 * time.Millisecond }})
	conn := rawDial(t, e.addr)
	expectRawClosed(t, conn)
	waitFor(t, "client removed", func() bool { return e.srv.OnlineClients() == 0 })
}

// Run fails cleanly when the port is taken.
func TestRunListenError(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	cfg := config.Default()
	cfg.GameServers[0].Port = ln.Addr().(*net.TCPAddr).Port
	s, err := New(cfg, 1, Deps{
		Characters: memstore.NewCharacters(),
		IDs:        memstore.NewIDs(),
		Data:       loadData(t),
		Login:      blockingLink{},
	}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Run(t.Context()); err == nil {
		t.Fatal("Run succeeded on a port in use")
	}
}
