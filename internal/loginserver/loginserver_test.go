package loginserver

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"slices"
	"sync"
	"testing"
	"testing/iotest"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/crypt/rsakey"
	"github.com/IPeter/l2go/internal/link"
	"github.com/IPeter/l2go/internal/loginserver/clientpackets"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
)

// testKeys is one RSA key pair shared by all tests; generating keys is
// slow.
var testKeys = sync.OnceValues(func() (*rsakey.Pool, error) { return rsakey.New(1) })

func newTestServer(t *testing.T, cfg config.Config, store storage.AccountStore) *Server {
	t.Helper()
	keys, err := testKeys()
	if err != nil {
		t.Fatalf("RSA keys: %v", err)
	}
	s, err := newServer(cfg, store, slog.New(slog.DiscardHandler), keys)
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	s.bcryptCost = bcrypt.MinCost
	return s
}

func hash(t *testing.T, password string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return string(h)
}

type brokenStore struct{ err error }

func (b brokenStore) FindByUsername(context.Context, string) (storage.Account, error) {
	return storage.Account{}, b.err
}
func (b brokenStore) Create(context.Context, storage.Account) error { return b.err }
func (b brokenStore) SetLastServer(context.Context, string, int) error {
	return b.err
}
func (b brokenStore) SetAccessLevel(context.Context, string, storage.AccessLevel) error {
	return b.err
}

// fakeStatuses replaces the link in unit tests.
type fakeStatuses map[int]link.ServerStatus

func (f fakeStatuses) Status(id int) (link.ServerStatus, bool) {
	st, ok := f[id]
	return st, ok
}

func (f fakeStatuses) Statuses() map[int]link.ServerStatus {
	out := make(map[int]link.ServerStatus, len(f))
	for id, st := range f {
		out[id] = st
	}
	return out
}

func TestNewRejectsBadInternalNetwork(t *testing.T) {
	cfg := config.Default()
	cfg.LoginServer.InternalNetworks = []string{"not-a-cidr"}
	if _, err := New(cfg, memstore.NewAccounts(), slog.New(slog.DiscardHandler)); err == nil {
		t.Fatal("New succeeded with an invalid internal network")
	}
}

func TestNewGeneratesKeyPairs(t *testing.T) {
	cfg := config.Default()
	cfg.LoginServer.RSAKeyPairs = 2
	s, err := New(cfg, memstore.NewAccounts(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if s.keys.Len() != 2 {
		t.Errorf("key pool has %d pairs, want 2", s.keys.Len())
	}

	cfg.LoginServer.RSAKeyPairs = 0
	if _, err := New(cfg, memstore.NewAccounts(), slog.New(slog.DiscardHandler)); err == nil {
		t.Error("New succeeded with no RSA key pairs")
	}
}

func TestNormalizeUsername(t *testing.T) {
	tests := map[string]string{
		"alice":   "alice",
		"Alice":   "alice",
		"BOB_99":  "bob_99",
		"":        "",
		"MiXeD12": "mixed12",
	}
	for in, want := range tests {
		if got := normalizeUsername(in); got != want {
			t.Errorf("normalizeUsername(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAuthenticate(t *testing.T) {
	store := memstore.NewAccounts()
	ctx := t.Context()
	for _, acc := range []storage.Account{
		{Username: "player", Password: hash(t, "good"), AccessLevel: storage.AccessLevelPlayer},
		{Username: "admin", Password: hash(t, "good"), AccessLevel: storage.AccessLevelAdmin},
		{Username: "banned", Password: hash(t, "good"), AccessLevel: storage.AccessLevelBanned},
	} {
		if err := store.Create(ctx, acc); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name       string
		autoCreate bool
		user, pass string
		wantErr    error
		wantLevel  storage.AccessLevel
	}{
		{"existing player", true, "player", "good", nil, storage.AccessLevelPlayer},
		{"existing admin", false, "admin", "good", nil, storage.AccessLevelAdmin},
		{"wrong password", true, "player", "bad", errWrongCredentials, 0},
		{"banned", true, "banned", "good", errAccessDenied, 0},
		{"unknown without auto-create", false, "nobody", "x", errWrongCredentials, 0},
		{"empty username", true, "", "x", errWrongCredentials, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.LoginServer.AutoCreate = tt.autoCreate
			s := newTestServer(t, cfg, store)
			acc, err := s.authenticate(ctx, tt.user, tt.pass)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("authenticate error = %v, want %v", err, tt.wantErr)
			}
			if err == nil && (acc.Username != tt.user || acc.AccessLevel != tt.wantLevel) {
				t.Errorf("account = %+v, want %s with level %d", acc, tt.user, tt.wantLevel)
			}
		})
	}

	// None of the above may have created an account.
	if _, err := store.FindByUsername(ctx, "nobody"); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("unknown user was created: %v", err)
	}
	if _, err := store.FindByUsername(ctx, ""); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("empty user was created: %v", err)
	}
}

func TestAuthenticateAutoCreate(t *testing.T) {
	store := memstore.NewAccounts()
	s := newTestServer(t, config.Default(), store)
	ctx := t.Context()

	acc, err := s.authenticate(ctx, "newbie", "pw")
	if err != nil {
		t.Fatalf("first login: %v", err)
	}
	if acc.AccessLevel != storage.AccessLevelPlayer {
		t.Errorf("new account level = %d, want player", acc.AccessLevel)
	}
	stored, err := store.FindByUsername(ctx, "newbie")
	if err != nil {
		t.Fatalf("account not stored: %v", err)
	}
	if stored.Password == "pw" || bcrypt.CompareHashAndPassword([]byte(stored.Password), []byte("pw")) != nil {
		t.Errorf("stored password %q is not a bcrypt hash of the password", stored.Password)
	}

	if _, err := s.authenticate(ctx, "newbie", "pw"); err != nil {
		t.Errorf("second login: %v", err)
	}
	if _, err := s.authenticate(ctx, "newbie", "other"); !errors.Is(err, errWrongCredentials) {
		t.Errorf("login with another password = %v, want errWrongCredentials", err)
	}
	st := s.Stats()
	if st.AccountsCreated != 1 || st.Logins != 1 || st.FailedLogins != 1 {
		t.Errorf("stats = %+v, want 1 created, 1 login, 1 failed", st)
	}
}

// Concurrent first logins with one new username must all succeed and
// create a single account.
func TestAuthenticateConcurrentAutoCreate(t *testing.T) {
	store := memstore.NewAccounts()
	s := newTestServer(t, config.Default(), store)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if _, err := s.authenticate(t.Context(), "racer", "pw"); err != nil {
				t.Errorf("authenticate: %v", err)
			}
		})
	}
	wg.Wait()
	if got := s.Stats().AccountsCreated; got != 1 {
		t.Errorf("AccountsCreated = %d, want 1", got)
	}
}

func TestAuthenticateStoreError(t *testing.T) {
	boom := errors.New("database down")
	s := newTestServer(t, config.Default(), brokenStore{boom})
	_, err := s.authenticate(t.Context(), "user", "pw")
	if !errors.Is(err, boom) {
		t.Fatalf("authenticate = %v, want it to wrap %v", err, boom)
	}
	if errors.Is(err, errWrongCredentials) || errors.Is(err, errAccessDenied) {
		t.Errorf("a store failure must not look like bad credentials: %v", err)
	}
}

func TestAuthenticateCreateError(t *testing.T) {
	boom := errors.New("disk full")
	s := newTestServer(t, config.Default(), createFails{memstore.NewAccounts(), boom})
	if _, err := s.authenticate(t.Context(), "user", "pw"); !errors.Is(err, boom) {
		t.Fatalf("authenticate = %v, want it to wrap %v", err, boom)
	}
	if got := s.Stats().AccountCreateFailures; got != 1 {
		t.Errorf("AccountCreateFailures = %d, want 1", got)
	}
}

type createFails struct {
	*memstore.Accounts
	err error
}

func (c createFails) Create(context.Context, storage.Account) error { return c.err }

// testingServerConfig returns the default config plus testing server 7.
func testingServerConfig() config.Config {
	cfg := config.Default()
	testServer := cfg.GameServers[0]
	testServer.ID = 7
	testServer.Secret = "secret-7"
	testServer.Options.Testing = true
	cfg.GameServers = append(cfg.GameServers, testServer)
	return cfg
}

// B3: server ID 0 wrapped around to 255 and indexed past the server list.
// The server must also be online on the link; testing and full servers only
// take admins.
func TestCanPlay(t *testing.T) {
	cfg := testingServerConfig()
	cfg.GameServers[0].Options.MaxPlayers = 100
	s := newTestServer(t, cfg, memstore.NewAccounts())

	tests := []struct {
		name     string
		statuses fakeStatuses
		id       int
		level    storage.AccessLevel
		want     bool
	}{
		{"ID 0", fakeStatuses{1: {Online: true}}, 0, storage.AccessLevelAdmin, false},
		{"unknown ID", fakeStatuses{1: {Online: true}}, 2, storage.AccessLevelAdmin, false},
		{"ID 255", fakeStatuses{1: {Online: true}}, 255, storage.AccessLevelAdmin, false},
		{"online", fakeStatuses{1: {Online: true}}, 1, storage.AccessLevelPlayer, true},
		{"offline", fakeStatuses{1: {MaxPlayers: 100}}, 1, storage.AccessLevelAdmin, false},
		{"never registered", fakeStatuses{}, 1, storage.AccessLevelAdmin, false},
		{"testing, player", fakeStatuses{7: {Online: true}}, 7, storage.AccessLevelPlayer, false},
		{"testing, admin", fakeStatuses{7: {Online: true}}, 7, storage.AccessLevelAdmin, true},
		{"full by report, player", fakeStatuses{1: {Online: true, Players: 50, MaxPlayers: 50}}, 1, storage.AccessLevelPlayer, false},
		{"full by report, admin", fakeStatuses{1: {Online: true, Players: 50, MaxPlayers: 50}}, 1, storage.AccessLevelAdmin, true},
		{"full by config, player", fakeStatuses{1: {Online: true, Players: 100}}, 1, storage.AccessLevelPlayer, false},
		{"not full", fakeStatuses{1: {Online: true, Players: 99}}, 1, storage.AccessLevelPlayer, true},
	}
	for _, tt := range tests {
		s.statuses = tt.statuses
		if got := s.canPlay(tt.id, tt.level) == ""; got != tt.want {
			t.Errorf("%s: canPlay(%d, %d) = %q, want ok=%v", tt.name, tt.id, tt.level, s.canPlay(tt.id, tt.level), tt.want)
		}
	}
	if s.Stats().HackAttempts == 0 {
		t.Error("unknown server IDs were not counted as hack attempts")
	}
}

func TestServerListAddresses(t *testing.T) {
	cfg := config.Default()
	cfg.GameServers[0].ID = 3
	cfg.GameServers[0].InternalIP = "10.0.0.5"
	cfg.GameServers[0].ExternalIP = "203.0.113.9"
	cfg.LoginServer.InternalNetworks = []string{"127.0.0.0/8", "192.168.0.0/16"}
	s := newTestServer(t, cfg, memstore.NewAccounts())

	tests := []struct {
		client string
		want   string
	}{
		{"127.0.0.1", "10.0.0.5"},
		{"::ffff:127.0.0.1", "10.0.0.5"},
		{"192.168.1.20", "10.0.0.5"},
		{"172.18.0.1", "203.0.113.9"}, // a Docker gateway is external
		{"::1", "203.0.113.9"},
	}
	for _, tt := range tests {
		list := s.serverList(netip.MustParseAddr(tt.client), storage.AccessLevelPlayer)
		if len(list) != 1 {
			t.Fatalf("server list has %d entries, want 1", len(list))
		}
		if got := list[0].IP.String(); got != tt.want {
			t.Errorf("client %s gets %s, want %s", tt.client, got, tt.want)
		}
		if list[0].ID != 3 {
			t.Errorf("ID = %d, want the configured ID 3", list[0].ID)
		}
	}
}

func TestServerListStatus(t *testing.T) {
	cfg := testingServerConfig()
	cfg.GameServers[0].Options.MaxPlayers = 1000
	s := newTestServer(t, cfg, memstore.NewAccounts())
	client := netip.MustParseAddr("127.0.0.1")

	s.statuses = fakeStatuses{}
	list := s.serverList(client, storage.AccessLevelPlayer)
	if len(list) != 1 {
		t.Fatalf("player sees %d servers, want 1 (the testing server is hidden)", len(list))
	}
	if got := list[0]; got.Up || got.Players != 0 || got.MaxPlayers != 1000 || got.Port != 7777 {
		t.Errorf("never registered: %+v, want down, 0 players, the configured 1000 max", got)
	}

	s.statuses = fakeStatuses{1: {Online: true, Players: 42, MaxPlayers: 500}, 7: {Online: true}}
	list = s.serverList(client, storage.AccessLevelPlayer)
	if got := list[0]; !got.Up || got.Players != 42 || got.MaxPlayers != 500 {
		t.Errorf("online: %+v, want up with 42/500", got)
	}

	list = s.serverList(client, storage.AccessLevelAdmin)
	if len(list) != 2 || list[1].ID != 7 || !list[1].Up {
		t.Errorf("admin sees %+v, want servers 1 and 7, both up", list)
	}

	s.statuses = fakeStatuses{1: {Players: 0, MaxPlayers: 500}}
	if got := s.serverList(client, storage.AccessLevelPlayer)[0]; got.Up || got.MaxPlayers != 500 {
		t.Errorf("offline: %+v, want down with the last reported 500 max", got)
	}

	s.statuses = fakeStatuses{1: {Online: true, Players: 70000, MaxPlayers: 70000}}
	if got := s.serverList(client, storage.AccessLevelPlayer)[0]; got.Players != 0xffff || got.MaxPlayers != 0xffff {
		t.Errorf("huge counts: %+v, want them clamped to 65535", got)
	}
}

func TestStatsCountsOnlineGameServers(t *testing.T) {
	s := newTestServer(t, testingServerConfig(), memstore.NewAccounts())
	s.statuses = fakeStatuses{1: {Online: true}, 7: {MaxPlayers: 10}}
	if got := s.Stats().GameServers; got != 1 {
		t.Errorf("GameServers = %d, want 1", got)
	}
}

// B12: NewClient returned nil when crypto/rand failed, and the accept loop
// then dereferenced it.
func TestNewClientRandomFailure(t *testing.T) {
	keys, err := testKeys()
	if err != nil {
		t.Fatal(err)
	}
	boom := errors.New("no entropy")
	c, err := newClient(nil, keys.Pick(), iotest.ErrReader(boom), time.Minute, slog.New(slog.DiscardHandler))
	if !errors.Is(err, boom) || c != nil {
		t.Fatalf("newClient = %v, %v; want nil, %v", c, err, boom)
	}
}

func TestNewClientRandomValues(t *testing.T) {
	s := newTestServer(t, config.Default(), memstore.NewAccounts())
	logger := slog.New(slog.DiscardHandler)
	a, err := newClient(nil, s.keys.Pick(), s.rnd, time.Minute, logger)
	if err != nil {
		t.Fatal(err)
	}
	b, err := newClient(nil, s.keys.Pick(), s.rnd, time.Minute, logger)
	if err != nil {
		t.Fatal(err)
	}
	if a.sessionID == b.sessionID && a.blowfishKey == b.blowfishKey {
		t.Error("two connections got the same session ID and Blowfish key")
	}
	if a.blowfishKey == [16]byte{} {
		t.Error("Blowfish key is all zeros")
	}
	if a.state != stateConnected {
		t.Errorf("new connection is in state %v, want connected", a.state)
	}
}

// Each packet is accepted in one state only. Everything else, including
// unknown opcodes, is ignored as in Mobius: no error, no answer, no state
// change.
func TestHandlePacketWrongState(t *testing.T) {
	s := newTestServer(t, config.Default(), memstore.NewAccounts())
	allowed := map[connState][]byte{
		stateConnected:   {clientpackets.OpAuthGameGuard},
		stateAuthedGG:    {clientpackets.OpRequestAuthLogin},
		stateAuthedLogin: {clientpackets.OpRequestServerList, clientpackets.OpRequestServerLogin},
	}
	for state, ok := range allowed {
		for op := range 256 {
			opcode := byte(op)
			if slices.Contains(ok, opcode) {
				continue
			}
			// conn is nil: any attempt to send would panic.
			c := &client{state: state, logger: slog.New(slog.DiscardHandler)}
			if err := s.handlePacket(t.Context(), c, opcode, make([]byte, 300)); err != nil {
				t.Errorf("opcode 0x%02x in state %v: error = %v, want it ignored", opcode, state, err)
			}
			if c.state != state {
				t.Errorf("opcode 0x%02x moved the state from %v to %v", opcode, state, c.state)
			}
		}
	}
}

// Packets too short for their layout are ignored in their own state too,
// as Mobius does when readImpl fails.
func TestHandlePacketShort(t *testing.T) {
	s := newTestServer(t, config.Default(), memstore.NewAccounts())
	tests := []struct {
		state  connState
		opcode byte
		size   int
	}{
		{stateConnected, clientpackets.OpAuthGameGuard, 19},
		{stateAuthedGG, clientpackets.OpRequestAuthLogin, 127},
		{stateAuthedLogin, clientpackets.OpRequestServerList, 7},
		{stateAuthedLogin, clientpackets.OpRequestServerLogin, 8},
	}
	for _, tt := range tests {
		c := &client{state: tt.state, logger: slog.New(slog.DiscardHandler)}
		if err := s.handlePacket(t.Context(), c, tt.opcode, make([]byte, tt.size)); err != nil {
			t.Errorf("%d-byte packet 0x%02x: error = %v, want it ignored", tt.size, tt.opcode, err)
		}
		if c.state != tt.state {
			t.Errorf("packet 0x%02x moved the state from %v to %v", tt.opcode, tt.state, c.state)
		}
	}
}

// An RSA block that is not smaller than the modulus cannot be decrypted;
// the packet is dropped and counted.
func TestHandleAuthLoginUndecryptable(t *testing.T) {
	s := newTestServer(t, config.Default(), memstore.NewAccounts())
	c := &client{state: stateAuthedGG, rsa: s.keys.Pick(), logger: slog.New(slog.DiscardHandler)}
	block := bytes.Repeat([]byte{0xff}, 128)
	if err := s.handlePacket(t.Context(), c, clientpackets.OpRequestAuthLogin, block); err != nil {
		t.Fatalf("error = %v, want the packet ignored", err)
	}
	if c.state != stateAuthedGG {
		t.Errorf("state = %v, want GameGuard ok", c.state)
	}
	if s.Stats().HackAttempts != 1 {
		t.Errorf("HackAttempts = %d, want 1", s.Stats().HackAttempts)
	}
}

func TestConnStateString(t *testing.T) {
	for _, st := range []connState{stateConnected, stateAuthedGG, stateAuthedLogin, connState(9)} {
		if st.String() == "" {
			t.Errorf("state %d has no name", int(st))
		}
	}
}

// --- lifecycle ---

type running struct {
	clientAddr string
	gsAddr     string
	stop       func() error
}

func listen(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return ln
}

// start runs s.Serve until the test ends or stop is called.
func start(t *testing.T, s *Server) running {
	t.Helper()
	clientLn, gsLn := listen(t), listen(t)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- s.Serve(ctx, clientLn, gsLn) }()
	stop := sync.OnceValue(func() error {
		cancel()
		select {
		case err := <-done:
			return err
		case <-time.After(5 * time.Second):
			return errors.New("server did not stop within 5s")
		}
	})
	t.Cleanup(func() { _ = stop() })
	return running{clientAddr: clientLn.Addr().String(), gsAddr: gsLn.Addr().String(), stop: stop}
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

func dial(t *testing.T, addr string) net.Conn {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// readFirstPacket reads the packet the server sends on connect.
func readFirstPacket(t *testing.T, conn net.Conn) {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := packets.ReadPacket(conn, packets.MaxBodySize); err != nil {
		t.Fatalf("reading the first packet: %v", err)
	}
}

// expectClosed checks that the server closed conn. Anything still sent
// before the close is skipped.
func expectClosed(t *testing.T, conn net.Conn) {
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

// B7, B8: cancelling the context stops both accept loops, closes every
// connection and waits for the handlers, with clients coming and going
// concurrently.
func TestServeShutdown(t *testing.T) {
	s := newTestServer(t, config.Default(), memstore.NewAccounts())
	r := start(t, s)

	const n = 5
	conns := make([]net.Conn, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			conn, err := net.DialTimeout("tcp", r.clientAddr, 5*time.Second)
			if err != nil {
				t.Error(err)
				return
			}
			conns[i] = conn
			_ = s.Stats() // concurrent reads of the counters
		})
	}
	wg.Wait()
	for _, c := range conns {
		if c == nil {
			t.FailNow()
		}
		t.Cleanup(func() { _ = c.Close() })
		readFirstPacket(t, c)
	}
	gs := dial(t, r.gsAddr) // an unregistered game server connection
	readFirstPacket(t, gs)  // the link's challenge
	waitFor(t, "all clients registered", func() bool { return s.Stats().OnlineClients == n })

	// A client that leaves is removed without touching the others (B6).
	_ = conns[0].Close()
	waitFor(t, "one client removed", func() bool { return s.Stats().OnlineClients == n-1 })

	if err := r.stop(); err != nil {
		t.Fatalf("Serve = %v, want nil after cancel", err)
	}
	for _, c := range conns[1:] {
		expectClosed(t, c)
	}
	expectClosed(t, gs)
	if st := s.Stats(); st.OnlineClients != 0 || st.GameServers != 0 {
		t.Errorf("after shutdown: %d clients, %d game servers registered", st.OnlineClients, st.GameServers)
	}
	if c, err := net.DialTimeout("tcp", r.clientAddr, time.Second); err == nil {
		_ = c.Close()
		t.Error("client listener still accepts connections")
	}
	if c, err := net.DialTimeout("tcp", r.gsAddr, time.Second); err == nil {
		_ = c.Close()
		t.Error("game server listener still accepts connections")
	}
}

func TestIdleTimeout(t *testing.T) {
	s := newTestServer(t, config.Default(), memstore.NewAccounts())
	s.idleTimeout = 100 * time.Millisecond
	r := start(t, s)

	conn := dial(t, r.clientAddr)
	readFirstPacket(t, conn)
	expectClosed(t, conn)
	waitFor(t, "client removed", func() bool { return s.Stats().OnlineClients == 0 })
}

// B2 at the server level: a packet with an empty body used to crash the
// whole server. Now only that connection is closed. The same goes for
// bodies that are not whole Blowfish blocks or fail the checksum.
func TestBadFramesCloseOnlyThatConnection(t *testing.T) {
	s := newTestServer(t, config.Default(), memstore.NewAccounts())
	r := start(t, s)

	good := dial(t, r.clientAddr)
	readFirstPacket(t, good)

	bodies := [][]byte{
		{0x02, 0x00},       // empty body
		{0x03, 0x00, 0x00}, // one byte
		append([]byte{0x0a, 0x00}, make([]byte, 8)...),  // a block that fails the checksum
		append([]byte{0x0e, 0x00}, make([]byte, 12)...), // not a whole block
	}
	for _, wire := range bodies {
		bad := dial(t, r.clientAddr)
		readFirstPacket(t, bad)
		if _, err := bad.Write(wire); err != nil {
			t.Fatal(err)
		}
		expectClosed(t, bad)
	}

	gs := dial(t, r.gsAddr)
	if _, err := gs.Write([]byte{0x02, 0x00}); err != nil {
		t.Fatal(err)
	}
	expectClosed(t, gs)

	waitFor(t, "only the good client left", func() bool { return s.Stats().OnlineClients == 1 })
	if err := r.stop(); err != nil {
		t.Fatalf("Serve = %v", err)
	}
}

// B9: listen errors used to be printed and ignored.
func TestRunListenErrors(t *testing.T) {
	tests := map[string]func(*config.Config){
		"client listen":      func(c *config.Config) { c.LoginServer.ClientListen = "127.0.0.1:99999" },
		"game server listen": func(c *config.Config) { c.LoginServer.GameServerListen = "127.0.0.1:99999" },
	}
	for name, modify := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := config.Default()
			cfg.LoginServer.ClientListen = "127.0.0.1:0"
			cfg.LoginServer.GameServerListen = "127.0.0.1:0"
			modify(&cfg)
			s := newTestServer(t, cfg, memstore.NewAccounts())
			if err := s.Run(t.Context()); err == nil {
				t.Fatal("Run succeeded with an invalid listen address")
			}
		})
	}
}
