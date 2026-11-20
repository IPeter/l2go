package link

import (
	"context"
	"crypto/hmac"
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/session"
)

const (
	testSecret = "s3cret"
	testMax    = 100
)

func testServerConfig() ServerConfig {
	return ServerConfig{
		GameServers:     map[int]string{1: testSecret, 2: "other-secret"},
		RegisterTimeout: 2 * time.Second,
		IdleTimeout:     2 * time.Second,
		WriteTimeout:    time.Second,
	}
}

func testClientConfig(addr string) ClientConfig {
	return ClientConfig{
		Addr:         addr,
		ServerID:     1,
		Secret:       testSecret,
		MaxPlayers:   testMax,
		DialTimeout:  time.Second,
		PingInterval: 50 * time.Millisecond,
		MinBackoff:   10 * time.Millisecond,
		MaxBackoff:   50 * time.Millisecond,
	}
}

// listen listens on addr, retrying for a while so a test can take over the
// port of a server it just stopped.
func listen(t *testing.T, addr string) net.Listener {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			return ln
		}
		if time.Now().After(deadline) {
			t.Fatalf("listen on %s: %v", addr, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

type testServer struct {
	*Server
	addr   string
	cancel context.CancelFunc
	done   chan struct{}
	err    error // Serve's result, set before done is closed
}

func startServer(t *testing.T, addr string, sessions SessionConsumer) *testServer {
	t.Helper()
	return startServerWith(t, addr, testServerConfig(), sessions)
}

// startServerWith serves a link server on addr ("127.0.0.1:0" for any
// port) until stop is called or the test ends.
func startServerWith(t *testing.T, addr string, cfg ServerConfig, sessions SessionConsumer) *testServer {
	t.Helper()
	ln := listen(t, addr)
	ctx, cancel := context.WithCancel(t.Context())
	ts := &testServer{
		Server: NewServer(cfg, sessions, nil),
		addr:   ln.Addr().String(),
		cancel: cancel,
		done:   make(chan struct{}),
	}
	go func() {
		defer close(ts.done)
		ts.err = ts.Serve(ctx, ln)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-ts.done:
		case <-time.After(5 * time.Second):
			t.Error("Serve did not return after cancel")
		}
	})
	return ts
}

// stop cancels Serve and checks that it returns nil.
func (ts *testServer) stop(t *testing.T) {
	t.Helper()
	ts.cancel()
	select {
	case <-ts.done:
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after cancel")
	}
	if ts.err != nil {
		t.Errorf("Serve = %v, want nil", ts.err)
	}
}

func (ts *testServer) online(id int) bool {
	st, _ := ts.Status(id)
	return st.Online
}

func (ts *testServer) players(id int) int {
	st, _ := ts.Status(id)
	return st.Players
}

type testClient struct {
	*Client
	cancel context.CancelFunc
	done   chan struct{}
	err    error // Run's result, set before done is closed
}

// startClient runs a link client until stop is called or the test ends.
func startClient(t *testing.T, cfg ClientConfig) *testClient {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	tc := &testClient{Client: NewClient(cfg, nil), cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(tc.done)
		tc.err = tc.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-tc.done:
		case <-time.After(5 * time.Second):
			t.Error("Run did not return after cancel")
		}
	})
	return tc
}

// wait waits for Run to return on its own and returns its result.
func (tc *testClient) wait(t *testing.T) error {
	t.Helper()
	select {
	case <-tc.done:
		return tc.err
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return")
		return nil
	}
}

// stop cancels Run and returns its result.
func (tc *testClient) stop(t *testing.T) error {
	t.Helper()
	tc.cancel()
	return tc.wait(t)
}

func (tc *testClient) returned() bool {
	select {
	case <-tc.done:
		return true
	default:
		return false
	}
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting until %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func dial(t *testing.T, addr string) net.Conn {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	return conn
}

// rawHandshake registers by hand over a plain connection. makeReg builds
// the Register message from the challenge. It returns the connection and
// the login server's answer.
func rawHandshake(t *testing.T, addr string, makeReg func(challengeMsg) registerMsg) (net.Conn, Reason) {
	t.Helper()
	conn := dial(t, addr)
	msg, err := readMsg(conn)
	if err != nil {
		t.Fatalf("read challenge: %v", err)
	}
	ch, ok := msg.(challengeMsg)
	if !ok {
		t.Fatalf("got %T, want a challenge", msg)
	}
	if ch.version != protocolVersion {
		t.Errorf("challenge version = %d, want %d", ch.version, protocolVersion)
	}
	if err := writeMsg(conn, time.Second, makeReg(ch)); err != nil {
		t.Fatal(err)
	}
	msg, err = readMsg(conn)
	if err != nil {
		t.Fatalf("read register result: %v", err)
	}
	res, ok := msg.(registerResultMsg)
	if !ok {
		t.Fatalf("got %T, want a register result", msg)
	}
	return conn, res.reason
}

func rawRegister(t *testing.T, addr string, id uint8, secret string) (net.Conn, Reason) {
	t.Helper()
	return rawHandshake(t, addr, func(ch challengeMsg) registerMsg {
		reg := registerMsg{version: protocolVersion, serverID: id, maxPlayers: testMax}
		copy(reg.mac[:], computeMAC([]byte(secret), ch.challenge))
		return reg
	})
}

// expectClosed checks that the peer closes conn without sending anything.
func expectClosed(t *testing.T, conn net.Conn) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	msg, err := readMsg(conn)
	if err == nil {
		t.Fatalf("got %T, want the connection closed", msg)
	}
	if errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatal("the connection was not closed")
	}
}

// acceptRegistered plays a login server by hand: it accepts one client on
// ln, checks its registration and accepts it.
func acceptRegistered(t *testing.T, ln net.Listener) net.Conn {
	t.Helper()
	conn := acceptConn(t, ln)
	ch := challengeMsg{version: protocolVersion}
	ch.challenge[0] = 1
	if err := writeMsg(conn, time.Second, ch); err != nil {
		t.Fatal(err)
	}
	msg, err := readMsg(conn)
	if err != nil {
		t.Fatalf("read register: %v", err)
	}
	reg, ok := msg.(registerMsg)
	if !ok {
		t.Fatalf("got %T, want a register message", msg)
	}
	if !hmac.Equal(reg.mac[:], computeMAC([]byte(testSecret), ch.challenge)) {
		t.Error("the client sent a wrong MAC")
	}
	if reg.version != protocolVersion || reg.serverID != 1 || reg.maxPlayers != testMax {
		t.Errorf("register = version %d, server %d, max %d; want %d, 1, %d",
			reg.version, reg.serverID, reg.maxPlayers, protocolVersion, testMax)
	}
	if err := writeMsg(conn, time.Second, registerResultMsg{}); err != nil {
		t.Fatal(err)
	}
	return conn
}

func acceptConn(t *testing.T, ln net.Listener) net.Conn {
	t.Helper()
	if tl, ok := ln.(*net.TCPListener); ok {
		_ = tl.SetDeadline(time.Now().Add(5 * time.Second))
	}
	conn, err := ln.Accept()
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	return conn
}

func rejectedReason(err error) (Reason, bool) {
	var rejected *RejectedError
	if !errors.As(err, &rejected) {
		return 0, false
	}
	return rejected.Reason, true
}

func TestRegister(t *testing.T) {
	srv := startServer(t, "127.0.0.1:0", session.NewStore(time.Minute))
	if st, ok := srv.Status(1); !ok || st != (ServerStatus{}) {
		t.Fatalf("before registering: Status(1) = %+v, %v; want offline, true", st, ok)
	}

	c := startClient(t, testClientConfig(srv.addr))
	eventually(t, "the client is connected", c.Connected)
	eventually(t, "server 1 is online", func() bool { return srv.online(1) })

	if st, _ := srv.Status(1); st != (ServerStatus{Online: true, Players: 0, MaxPlayers: testMax}) {
		t.Errorf("Status(1) = %+v", st)
	}
	all := srv.Statuses()
	if len(all) != 2 || !all[1].Online || all[2].Online {
		t.Errorf("Statuses = %+v; want 1 online and 2 offline", all)
	}
	if _, ok := srv.Status(3); ok {
		t.Error("Status(3) reports a server that is not configured")
	}
}

func TestRegisterWrongSecret(t *testing.T) {
	srv := startServer(t, "127.0.0.1:0", nil)

	cfg := testClientConfig(srv.addr)
	cfg.Secret = "wrong"
	c := startClient(t, cfg)
	err := c.wait(t)
	if reason, ok := rejectedReason(err); !ok || reason != ReasonWrongSecret {
		t.Fatalf("Run = %v, want a RejectedError with ReasonWrongSecret", err)
	}
	if c.Connected() {
		t.Error("the client reports connected")
	}

	conn, reason := rawRegister(t, srv.addr, 1, "wrong")
	if reason != ReasonWrongSecret {
		t.Errorf("raw register reason = %v, want %v", reason, ReasonWrongSecret)
	}
	expectClosed(t, conn)

	// Another server's secret does not work either.
	conn, reason = rawRegister(t, srv.addr, 1, "other-secret")
	if reason != ReasonWrongSecret {
		t.Errorf("register with server 2's secret: reason = %v, want %v", reason, ReasonWrongSecret)
	}
	expectClosed(t, conn)

	if srv.online(1) {
		t.Error("server 1 is online after failed registrations")
	}
}

func TestRegisterUnknownServer(t *testing.T) {
	srv := startServer(t, "127.0.0.1:0", nil)

	cfg := testClientConfig(srv.addr)
	cfg.ServerID = 9
	c := startClient(t, cfg)
	err := c.wait(t)
	if reason, ok := rejectedReason(err); !ok || reason != ReasonUnknownServer {
		t.Fatalf("Run = %v, want a RejectedError with ReasonUnknownServer", err)
	}

	conn, reason := rawRegister(t, srv.addr, 0, testSecret)
	if reason != ReasonUnknownServer {
		t.Errorf("register as server 0: reason = %v, want %v", reason, ReasonUnknownServer)
	}
	expectClosed(t, conn)
	if _, ok := srv.Status(9); ok {
		t.Error("an unknown server appeared in the status list")
	}
}

func TestRegisterDuplicate(t *testing.T) {
	srv := startServer(t, "127.0.0.1:0", nil)
	a := startClient(t, testClientConfig(srv.addr))
	eventually(t, "the first client is connected", a.Connected)

	conn, reason := rawRegister(t, srv.addr, 1, testSecret)
	if reason != ReasonAlreadyRegistered {
		t.Errorf("second registration: reason = %v, want %v", reason, ReasonAlreadyRegistered)
	}
	expectClosed(t, conn)

	// A second client with the same ID keeps retrying without taking over.
	b := startClient(t, testClientConfig(srv.addr))
	time.Sleep(200 * time.Millisecond)
	if b.Connected() {
		t.Error("a second client registered with a taken ID")
	}
	if b.returned() {
		t.Fatalf("Run = %v; it should keep retrying while the ID is taken", b.err)
	}
	if !a.Connected() || !srv.online(1) {
		t.Fatal("the first client lost its registration")
	}

	// Once the first one leaves, the second one gets in.
	if err := a.stop(t); err != nil {
		t.Errorf("Run = %v, want nil", err)
	}
	eventually(t, "the second client registers", b.Connected)
	if !srv.online(1) {
		t.Error("server 1 is offline")
	}
}

func TestRegisterUnsupportedVersion(t *testing.T) {
	srv := startServer(t, "127.0.0.1:0", nil)
	conn, reason := rawHandshake(t, srv.addr, func(ch challengeMsg) registerMsg {
		reg := registerMsg{version: protocolVersion + 1, serverID: 1}
		copy(reg.mac[:], computeMAC([]byte(testSecret), ch.challenge))
		return reg
	})
	if reason != ReasonUnsupportedVersion {
		t.Errorf("reason = %v, want %v", reason, ReasonUnsupportedVersion)
	}
	expectClosed(t, conn)
}

func TestClientRejectsOtherVersion(t *testing.T) {
	ln := listen(t, "127.0.0.1:0")
	t.Cleanup(func() { _ = ln.Close() })
	c := startClient(t, testClientConfig(ln.Addr().String()))

	conn := acceptConn(t, ln)
	if err := writeMsg(conn, time.Second, challengeMsg{version: protocolVersion + 1}); err != nil {
		t.Fatal(err)
	}
	err := c.wait(t)
	if reason, ok := rejectedReason(err); !ok || reason != ReasonUnsupportedVersion {
		t.Fatalf("Run = %v, want a RejectedError with ReasonUnsupportedVersion", err)
	}
}

func TestUnregisteredConnectionIsClosed(t *testing.T) {
	cfg := testServerConfig()
	cfg.RegisterTimeout = 100 * time.Millisecond
	srv := startServerWith(t, "127.0.0.1:0", cfg, nil)

	conn := dial(t, srv.addr)
	if _, err := readMsg(conn); err != nil {
		t.Fatalf("read challenge: %v", err)
	}
	expectClosed(t, conn) // we never register
}

func TestHandshakeRejectsOtherMessages(t *testing.T) {
	srv := startServer(t, "127.0.0.1:0", nil)
	conn := dial(t, srv.addr)
	if _, err := readMsg(conn); err != nil {
		t.Fatalf("read challenge: %v", err)
	}
	if err := writeMsg(conn, time.Second, verifySessionMsg{requestID: 1, account: "alice"}); err != nil {
		t.Fatal(err)
	}
	expectClosed(t, conn)
}

func TestVerifySession(t *testing.T) {
	store := session.NewStore(time.Minute)
	srv := startServer(t, "127.0.0.1:0", store)
	c := startClient(t, testClientConfig(srv.addr))
	eventually(t, "the client is connected", c.Connected)
	ctx := t.Context()

	verify := func(account string, k session.Keys, want bool) {
		t.Helper()
		got, err := c.VerifySession(ctx, account, k)
		if err != nil || got != want {
			t.Errorf("VerifySession(%q) = %v, %v; want %v, nil", account, got, err, want)
		}
	}

	alice, err := store.Issue("alice")
	if err != nil {
		t.Fatal(err)
	}
	verify("alice", alice, true)
	verify("alice", alice, false) // keys are single use

	bob, err := store.Issue("bob")
	if err != nil {
		t.Fatal(err)
	}
	wrong := bob
	wrong.PlayOK1++
	verify("bob", wrong, false)
	verify("alice", bob, false) // someone else's keys
	verify("bob", bob, true)    // wrong attempts do not burn the keys

	verify("nobody", alice, false)
	verify("", alice, false)
	verify(strings.Repeat("x", maxAccountLen+1), alice, false)
	verify("a\x00b", alice, false)

	if !c.Connected() {
		t.Error("the link dropped during session checks")
	}
}

// accessRecorder records the access levels a link server gets.
type accessRecorder struct {
	mu  sync.Mutex
	got []string
}

func (r *accessRecorder) SetAccessLevel(account string, level int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.got = append(r.got, fmt.Sprintf("%s=%d", account, level))
	return nil
}

func (r *accessRecorder) list() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.got)
}

// ChangeAccessLevel reaches the login server's setter in order; it fails
// while the link is down and for names the link cannot carry.
func TestChangeAccessLevel(t *testing.T) {
	rec := &accessRecorder{}
	cfg := testServerConfig()
	cfg.AccessLevels = rec
	srv := startServerWith(t, "127.0.0.1:0", cfg, nil)

	down := NewClient(testClientConfig(srv.addr), nil)
	if err := down.ChangeAccessLevel("alice", -1); !errors.Is(err, ErrNotConnected) {
		t.Errorf("ChangeAccessLevel before Run = %v, want ErrNotConnected", err)
	}

	c := startClient(t, testClientConfig(srv.addr))
	eventually(t, "the client is connected", c.Connected)
	if err := c.ChangeAccessLevel("", -1); err == nil {
		t.Error("ChangeAccessLevel with an empty account succeeded")
	}
	for _, step := range []struct {
		account string
		level   int
	}{{"alice", -1}, {"bob", 0}, {"alice", 0}} {
		if err := c.ChangeAccessLevel(step.account, step.level); err != nil {
			t.Fatalf("ChangeAccessLevel(%s, %d) = %v", step.account, step.level, err)
		}
	}
	want := []string{"alice=-1", "bob=0", "alice=0"}
	eventually(t, "the access levels arrive", func() bool { return slices.Equal(rec.list(), want) })
	if !c.Connected() {
		t.Error("the link dropped after ChangeAccessLevel")
	}
}

// Responses are matched to requests by ID, not by order.
func TestVerifySessionConcurrent(t *testing.T) {
	store := session.NewStore(time.Minute)
	srv := startServer(t, "127.0.0.1:0", store)
	c := startClient(t, testClientConfig(srv.addr))
	eventually(t, "the client is connected", c.Connected)

	const n = 50
	keys := make([]session.Keys, n)
	for i := range n {
		k, err := store.Issue(fmt.Sprintf("user%d", i))
		if err != nil {
			t.Fatal(err)
		}
		keys[i] = k
	}
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			k := keys[i]
			want := i%2 == 0
			if !want {
				k.PlayOK2 ^= 1
			}
			got, err := c.VerifySession(t.Context(), fmt.Sprintf("user%d", i), k)
			if err != nil || got != want {
				t.Errorf("user%d: VerifySession = %v, %v; want %v, nil", i, got, err, want)
			}
		})
	}
	wg.Wait()
}

func TestVerifySessionWithoutSessionStore(t *testing.T) {
	srv := startServer(t, "127.0.0.1:0", nil)
	c := startClient(t, testClientConfig(srv.addr))
	eventually(t, "the client is connected", c.Connected)
	got, err := c.VerifySession(t.Context(), "alice", session.Keys{})
	if got || err != nil {
		t.Errorf("VerifySession = %v, %v; want false, nil", got, err)
	}
}

func TestVerifySessionNotConnected(t *testing.T) {
	keys := session.Keys{LoginOK1: 1, LoginOK2: 2, PlayOK1: 3, PlayOK2: 4}

	// Run was never called.
	idle := NewClient(testClientConfig("127.0.0.1:1"), nil)
	if _, err := idle.VerifySession(t.Context(), "alice", keys); !errors.Is(err, ErrNotConnected) {
		t.Errorf("before Run: VerifySession error = %v, want ErrNotConnected", err)
	}

	// Run is trying, but nothing listens.
	ln := listen(t, "127.0.0.1:0")
	addr := ln.Addr().String()
	_ = ln.Close()
	down := startClient(t, testClientConfig(addr))
	time.Sleep(50 * time.Millisecond)
	if down.Connected() {
		t.Fatal("connected to a closed port")
	}
	if _, err := down.VerifySession(t.Context(), "alice", keys); !errors.Is(err, ErrNotConnected) {
		t.Errorf("login server down: VerifySession error = %v, want ErrNotConnected", err)
	}

	// The login server goes away.
	srv := startServer(t, "127.0.0.1:0", session.NewStore(time.Minute))
	c := startClient(t, testClientConfig(srv.addr))
	eventually(t, "the client is connected", c.Connected)
	srv.stop(t)
	eventually(t, "the client notices the lost link", func() bool { return !c.Connected() })
	if _, err := c.VerifySession(t.Context(), "alice", keys); !errors.Is(err, ErrNotConnected) {
		t.Errorf("after the login server stopped: VerifySession error = %v, want ErrNotConnected", err)
	}
}

func TestVerifySessionRespectsContext(t *testing.T) {
	ln := listen(t, "127.0.0.1:0")
	t.Cleanup(func() { _ = ln.Close() })
	cfg := testClientConfig(ln.Addr().String())
	cfg.ReadTimeout = 10 * time.Second
	c := startClient(t, cfg)

	// A login server that registers the client, answers pings and ignores
	// everything else.
	conn := acceptRegistered(t, ln)
	_ = conn.SetDeadline(time.Time{})
	go func() {
		for {
			msg, err := readMsg(conn)
			if err != nil {
				return
			}
			if _, ok := msg.(pingMsg); ok {
				if err := writeMsg(conn, time.Second, pongMsg{}); err != nil {
					return
				}
			}
		}
	}()
	eventually(t, "the client is connected", c.Connected)

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	got, err := c.VerifySession(ctx, "alice", session.Keys{})
	if got || !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("unanswered VerifySession = %v, %v; want false, DeadlineExceeded", got, err)
	}

	done, cancelDone := context.WithCancel(t.Context())
	cancelDone()
	got, err = c.VerifySession(done, "alice", session.Keys{})
	if got || !errors.Is(err, context.Canceled) {
		t.Errorf("VerifySession with a cancelled context = %v, %v; want false, Canceled", got, err)
	}
	if !c.Connected() {
		t.Error("the link dropped")
	}
}

func TestClientAnswersPing(t *testing.T) {
	ln := listen(t, "127.0.0.1:0")
	t.Cleanup(func() { _ = ln.Close() })
	cfg := testClientConfig(ln.Addr().String())
	cfg.PingInterval = time.Hour // the client stays quiet on its own
	c := startClient(t, cfg)
	c.SetPlayers(3)

	conn := acceptRegistered(t, ln)
	msg, err := readMsg(conn)
	if err != nil {
		t.Fatal(err)
	}
	if want := (statusMsg{players: 3, maxPlayers: testMax}); msg != want {
		t.Errorf("first message after registering = %+v, want %+v", msg, want)
	}
	if err := writeMsg(conn, time.Second, pingMsg{}); err != nil {
		t.Fatal(err)
	}
	msg, err = readMsg(conn)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := msg.(pongMsg); !ok {
		t.Errorf("answer to a ping = %T, want a pong", msg)
	}
}

func TestStatusUpdates(t *testing.T) {
	srv := startServer(t, "127.0.0.1:0", nil)
	c := startClient(t, testClientConfig(srv.addr))
	eventually(t, "the client is connected", c.Connected)

	c.SetPlayers(5)
	c.SetPlayers(7)
	eventually(t, "the server sees 7 players", func() bool { return srv.players(1) == 7 })
	if st, _ := srv.Status(1); st != (ServerStatus{Online: true, Players: 7, MaxPlayers: testMax}) {
		t.Errorf("Status(1) = %+v", st)
	}

	c.SetPlayers(-3)
	eventually(t, "a negative count is sent as 0", func() bool { return srv.players(1) == 0 })
	c.SetPlayers(1 << 20)
	eventually(t, "a huge count is capped", func() bool { return srv.players(1) == 65535 })
	if st := srv.Statuses()[1]; st.Players != 65535 {
		t.Errorf("Statuses()[1] = %+v", st)
	}
}

func TestServerProtocolByHand(t *testing.T) {
	srv := startServer(t, "127.0.0.1:0", session.NewStore(time.Minute))
	conn, reason := rawRegister(t, srv.addr, 2, "other-secret")
	if reason != reasonNone {
		t.Fatalf("register reason = %v", reason)
	}
	eventually(t, "server 2 is online", func() bool { return srv.online(2) })

	if err := writeMsg(conn, time.Second, pingMsg{}); err != nil {
		t.Fatal(err)
	}
	if msg, err := readMsg(conn); err != nil || msg != (pongMsg{}) {
		t.Fatalf("answer to a ping = %+v, %v; want a pong", msg, err)
	}

	if err := writeMsg(conn, time.Second, statusMsg{players: 5, maxPlayers: 50}); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the status arrives", func() bool {
		st, _ := srv.Status(2)
		return st == ServerStatus{Online: true, Players: 5, MaxPlayers: 50}
	})

	if err := writeMsg(conn, time.Second, verifySessionMsg{requestID: 77, account: "nobody"}); err != nil {
		t.Fatal(err)
	}
	if msg, err := readMsg(conn); err != nil || msg != (verifySessionResultMsg{requestID: 77, ok: false}) {
		t.Fatalf("answer to VerifySession = %+v, %v; want request 77 not ok", msg, err)
	}

	// A message that only the login server sends is a protocol error.
	if err := writeMsg(conn, time.Second, challengeMsg{}); err != nil {
		t.Fatal(err)
	}
	expectClosed(t, conn)
	eventually(t, "server 2 goes offline", func() bool { return !srv.online(2) })
	if st, _ := srv.Status(2); st != (ServerStatus{MaxPlayers: 50}) {
		t.Errorf("Status(2) after disconnect = %+v; want offline, 0 players, max kept", st)
	}
}

func TestDisconnectMarksOffline(t *testing.T) {
	srv := startServer(t, "127.0.0.1:0", nil)
	c := startClient(t, testClientConfig(srv.addr))
	eventually(t, "server 1 is online", func() bool { return srv.online(1) })
	c.SetPlayers(3)
	eventually(t, "the server sees 3 players", func() bool { return srv.players(1) == 3 })

	if err := c.stop(t); err != nil {
		t.Errorf("Run = %v, want nil", err)
	}
	if c.Connected() {
		t.Error("the client reports connected after Run returned")
	}
	eventually(t, "server 1 goes offline", func() bool { return !srv.online(1) })
	if st, _ := srv.Status(1); st != (ServerStatus{MaxPlayers: testMax}) {
		t.Errorf("Status(1) = %+v; want offline, 0 players, max kept", st)
	}

	// A connection that just drops.
	conn, reason := rawRegister(t, srv.addr, 2, "other-secret")
	if reason != reasonNone {
		t.Fatalf("register reason = %v", reason)
	}
	eventually(t, "server 2 is online", func() bool { return srv.online(2) })
	_ = conn.Close()
	eventually(t, "server 2 goes offline", func() bool { return !srv.online(2) })
}

func TestIdleGameServerIsDropped(t *testing.T) {
	cfg := testServerConfig()
	cfg.IdleTimeout = 100 * time.Millisecond
	srv := startServerWith(t, "127.0.0.1:0", cfg, nil)

	conn, reason := rawRegister(t, srv.addr, 2, "other-secret")
	if reason != reasonNone {
		t.Fatalf("register reason = %v", reason)
	}
	expectClosed(t, conn) // we never send anything
	eventually(t, "server 2 goes offline", func() bool { return !srv.online(2) })
}

func TestReconnectAfterServerRestart(t *testing.T) {
	store := session.NewStore(time.Minute)
	first := startServer(t, "127.0.0.1:0", store)
	addr := first.addr
	c := startClient(t, testClientConfig(addr))
	eventually(t, "the client is connected", c.Connected)
	c.SetPlayers(12)
	eventually(t, "the first server sees 12 players", func() bool { return first.players(1) == 12 })

	first.stop(t)
	if first.online(1) {
		t.Error("server 1 is still online after Serve returned")
	}
	eventually(t, "the client notices the lost link", func() bool { return !c.Connected() })

	second := startServer(t, addr, store)
	eventually(t, "the client registers again", func() bool { return second.online(1) })
	eventually(t, "the player count is sent again", func() bool { return second.players(1) == 12 })
	eventually(t, "the client is connected", c.Connected)

	k, err := store.Issue("alice")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := c.VerifySession(t.Context(), "alice", k); !got || err != nil {
		t.Errorf("VerifySession after reconnecting = %v, %v; want true, nil", got, err)
	}
}

func TestClientConnectsWhenServerStartsLater(t *testing.T) {
	ln := listen(t, "127.0.0.1:0")
	addr := ln.Addr().String()
	_ = ln.Close()

	c := startClient(t, testClientConfig(addr))
	c.SetPlayers(4)
	time.Sleep(50 * time.Millisecond)

	srv := startServer(t, addr, nil)
	eventually(t, "the client registers", c.Connected)
	eventually(t, "the player count arrives", func() bool { return srv.players(1) == 4 })
}

func TestShutdown(t *testing.T) {
	before := runtime.NumGoroutine()

	srv := startServer(t, "127.0.0.1:0", session.NewStore(time.Minute))
	c := startClient(t, testClientConfig(srv.addr))
	eventually(t, "the client is connected", c.Connected)

	srv.stop(t)
	for id, st := range srv.Statuses() {
		if st.Online {
			t.Errorf("server %d is online after Serve returned", id)
		}
	}
	eventually(t, "the client notices", func() bool { return !c.Connected() })

	if err := c.stop(t); err != nil {
		t.Errorf("Run = %v, want nil", err)
	}
	if _, err := c.VerifySession(t.Context(), "alice", session.Keys{}); !errors.Is(err, ErrNotConnected) {
		t.Errorf("VerifySession after Run returned: error = %v, want ErrNotConnected", err)
	}
	eventually(t, "all goroutines exit", func() bool { return runtime.NumGoroutine() <= before })
}

func TestRunStopsDuringBackoff(t *testing.T) {
	ln := listen(t, "127.0.0.1:0")
	addr := ln.Addr().String()
	_ = ln.Close()

	cfg := testClientConfig(addr)
	cfg.MinBackoff = time.Hour
	cfg.MaxBackoff = time.Hour
	c := startClient(t, cfg)
	time.Sleep(50 * time.Millisecond) // the first dial fails; Run waits
	if err := c.stop(t); err != nil {
		t.Errorf("Run = %v, want nil", err)
	}
}

func TestRunStopsDuringHandshake(t *testing.T) {
	ln := listen(t, "127.0.0.1:0")
	t.Cleanup(func() { _ = ln.Close() })

	cfg := testClientConfig(ln.Addr().String())
	cfg.DialTimeout = time.Hour
	c := startClient(t, cfg)
	acceptConn(t, ln) // a login server that never sends the challenge

	start := time.Now()
	if err := c.stop(t); err != nil {
		t.Errorf("Run = %v, want nil", err)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("Run took %v to stop", d)
	}
}

func TestRunInvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		change func(*ClientConfig)
	}{
		{"no address", func(c *ClientConfig) { c.Addr = "" }},
		{"server ID 0", func(c *ClientConfig) { c.ServerID = 0 }},
		{"server ID 256", func(c *ClientConfig) { c.ServerID = 256 }},
		{"empty secret", func(c *ClientConfig) { c.Secret = "" }},
	}
	for _, tt := range tests {
		cfg := testClientConfig("127.0.0.1:1")
		tt.change(&cfg)
		err := NewClient(cfg, nil).Run(t.Context())
		if err == nil {
			t.Errorf("%s: Run = nil, want an error", tt.name)
		}
		if _, ok := rejectedReason(err); ok {
			t.Errorf("%s: Run = %v, want a config error, not a rejection", tt.name, err)
		}
	}
}

func TestRunTwice(t *testing.T) {
	srv := startServer(t, "127.0.0.1:0", nil)
	c := startClient(t, testClientConfig(srv.addr))
	eventually(t, "the client is connected", c.Connected)
	if err := c.Run(t.Context()); err == nil {
		t.Error("a second concurrent Run returned nil, want an error")
	}
	if !c.Connected() {
		t.Error("the second Run broke the first")
	}
}

func TestServeFailsWhenListenerCloses(t *testing.T) {
	ln := listen(t, "127.0.0.1:0")
	srv := NewServer(testServerConfig(), nil, nil)
	done := make(chan error, 1)
	go func() { done <- srv.Serve(t.Context(), ln) }()
	_ = ln.Close()
	select {
	case err := <-done:
		if !errors.Is(err, net.ErrClosed) {
			t.Errorf("Serve = %v, want net.ErrClosed", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return")
	}
}

func TestNewServerIgnoresInvalidEntries(t *testing.T) {
	srv := NewServer(ServerConfig{GameServers: map[int]string{0: "x", 256: "x", 3: "", 4: "ok"}}, nil, nil)
	all := srv.Statuses()
	if len(all) != 1 {
		t.Fatalf("Statuses = %+v; want only server 4", all)
	}
	if _, ok := all[4]; !ok {
		t.Errorf("Statuses = %+v; want server 4", all)
	}
}
