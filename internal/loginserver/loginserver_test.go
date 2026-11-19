package loginserver

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"sync"
	"testing"
	"testing/iotest"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
)

func newTestServer(t *testing.T, cfg config.Config, store storage.AccountStore) *Server {
	t.Helper()
	s, err := New(cfg, store, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New: %v", err)
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

func TestNewRejectsBadInternalNetwork(t *testing.T) {
	cfg := config.Default()
	cfg.LoginServer.InternalNetworks = []string{"not-a-cidr"}
	if _, err := New(cfg, memstore.NewAccounts(), slog.New(slog.DiscardHandler)); err == nil {
		t.Fatal("New succeeded with an invalid internal network")
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

// B3: server ID 0 wrapped around to 255 and indexed past the server list.
func TestCanPlay(t *testing.T) {
	cfg := config.Default()
	testServer := cfg.GameServers[0]
	testServer.ID = 7
	testServer.Options.Testing = true
	cfg.GameServers = append(cfg.GameServers, testServer)
	s := newTestServer(t, cfg, memstore.NewAccounts())

	tests := []struct {
		id    uint8
		level storage.AccessLevel
		want  bool
	}{
		{0, storage.AccessLevelAdmin, false},
		{2, storage.AccessLevelAdmin, false},
		{255, storage.AccessLevelAdmin, false},
		{1, storage.AccessLevelPlayer, true},
		{7, storage.AccessLevelPlayer, false},
		{7, storage.AccessLevelAdmin, true},
	}
	for _, tt := range tests {
		if got := s.canPlay(tt.id, tt.level); got != tt.want {
			t.Errorf("canPlay(%d, %d) = %v, want %v", tt.id, tt.level, got, tt.want)
		}
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
		list := s.serverList(netip.MustParseAddr(tt.client))
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

// B12: NewClient returned nil when crypto/rand failed, and the accept loop
// then dereferenced it.
func TestNewClientRandomFailure(t *testing.T) {
	boom := errors.New("no entropy")
	c, err := newClient(nil, nil, iotest.ErrReader(boom), time.Minute, slog.New(slog.DiscardHandler))
	if !errors.Is(err, boom) || c != nil {
		t.Fatalf("newClient = %v, %v; want nil, %v", c, err, boom)
	}
}

func TestSessionIDs(t *testing.T) {
	a, err := newSessionID(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	b, err := newSessionID(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Error("two session IDs are equal")
	}
}

func TestSessionMatches(t *testing.T) {
	c := &client{sessionID: [sessionIDSize]byte{1, 2, 3, 4, 5, 6, 7, 8, 9}}
	sid := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	if c.sessionMatches(sid) {
		t.Error("session matches before login")
	}
	c.account = &storage.Account{Username: "x"}
	if !c.sessionMatches(sid) {
		t.Error("session does not match after login")
	}
	for _, bad := range [][]byte{nil, sid[:7], {1, 2, 3, 4, 5, 6, 7, 9}} {
		if c.sessionMatches(bad) {
			t.Errorf("session matches %v", bad)
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

// expectClosed checks that the server closed conn.
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
	link := dial(t, r.gsAddr)
	waitFor(t, "all connections registered", func() bool {
		st := s.Stats()
		return st.OnlineClients == n && st.GameServers == 1
	})

	// A client that leaves is removed without touching the others (B6).
	_ = conns[0].Close()
	waitFor(t, "one client removed", func() bool { return s.Stats().OnlineClients == n-1 })

	if err := r.stop(); err != nil {
		t.Fatalf("Serve = %v, want nil after cancel", err)
	}
	for _, c := range conns[1:] {
		expectClosed(t, c)
	}
	expectClosed(t, link)
	if st := s.Stats(); st.OnlineClients != 0 || st.GameServers != 0 {
		t.Errorf("after shutdown: %d clients, %d game servers registered", st.OnlineClients, st.GameServers)
	}
	if c, err := net.DialTimeout("tcp", r.clientAddr, time.Second); err == nil {
		_ = c.Close()
		t.Error("client listener still accepts connections")
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
// whole server. Now only that connection is closed.
func TestEmptyPacketClosesOnlyThatConnection(t *testing.T) {
	s := newTestServer(t, config.Default(), memstore.NewAccounts())
	r := start(t, s)

	good := dial(t, r.clientAddr)
	readFirstPacket(t, good)

	for _, wire := range [][]byte{{0x02, 0x00}, {0x00, 0x00}, {0x01, 0x00}, {0x03, 0x00, 0x00}} {
		bad := dial(t, r.clientAddr)
		readFirstPacket(t, bad)
		if _, err := bad.Write(wire); err != nil {
			t.Fatal(err)
		}
		expectClosed(t, bad)
	}

	link := dial(t, r.gsAddr)
	if _, err := link.Write([]byte{0x02, 0x00}); err != nil {
		t.Fatal(err)
	}
	expectClosed(t, link)

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
