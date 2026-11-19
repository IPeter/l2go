package gameserver

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/packets"
)

func newTestServer(t *testing.T, cfg config.Config) *Server {
	t.Helper()
	s, err := New(cfg, 1, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s
}

func TestNewUnknownID(t *testing.T) {
	if _, err := New(config.Default(), 2, slog.New(slog.DiscardHandler)); err == nil {
		t.Fatal("New succeeded for a game server ID that is not configured")
	}
}

func TestSupportedProtocol(t *testing.T) {
	for v, want := range map[uint32]bool{0: false, 418: false, 419: true, 746: true} {
		if got := supportedProtocol(v); got != want {
			t.Errorf("supportedProtocol(%d) = %v, want %v", v, got, want)
		}
	}
}

// B9: the game server used to start without a login server and panic on
// the first write to the nil connection.
func TestRunWithoutLoginServer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close() // nothing listens on port any more

	cfg := config.Default()
	cfg.LoginServer.Host = "127.0.0.1"
	cfg.LoginServer.GameServerListen = ":" + strconv.Itoa(port)
	s := newTestServer(t, cfg)
	if err := s.Run(t.Context()); err == nil {
		t.Fatal("Run succeeded without a login server")
	}
}

type running struct {
	addr      string
	loginSide net.Conn // the login server's end of the link
	stop      func() error
}

// start runs s.Serve over a pipe to a fake login server that reads the
// register packet and then drains the link.
func start(t *testing.T, s *Server) running {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	gameSide, loginSide := net.Pipe()
	go func() { _, _ = io.Copy(io.Discard, loginSide) }()

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- s.Serve(ctx, ln, gameSide) }()
	stop := sync.OnceValue(func() error {
		cancel()
		select {
		case err := <-done:
			return err
		case <-time.After(5 * time.Second):
			return errors.New("server did not stop within 5s")
		}
	})
	t.Cleanup(func() {
		_ = stop()
		_ = loginSide.Close()
	})
	return running{addr: ln.Addr().String(), loginSide: loginSide, stop: stop}
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

func TestRegistersWithLoginServer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	gameSide, loginSide := net.Pipe()
	defer loginSide.Close()

	s := newTestServer(t, config.Default())
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- s.Serve(ctx, ln, gameSide) }()

	if err := loginSide.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	body, err := packets.ReadPacket(loginSide, packets.MaxBodySize)
	if err != nil {
		t.Fatalf("reading the register packet: %v", err)
	}
	if len(body) == 0 {
		t.Error("empty register packet")
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Serve = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return")
	}
}

// B6, B7, B8: clients register under their own IDs, a leaving client
// removes only itself, and cancelling closes everything.
func TestServeShutdown(t *testing.T) {
	s := newTestServer(t, config.Default())
	r := start(t, s)

	const n = 4
	conns := make([]net.Conn, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			conn, err := net.DialTimeout("tcp", r.addr, 5*time.Second)
			if err != nil {
				t.Error(err)
				return
			}
			conns[i] = conn
			_ = s.OnlineClients()
		})
	}
	wg.Wait()
	for _, c := range conns {
		if c == nil {
			t.FailNow()
		}
		t.Cleanup(func() { _ = c.Close() })
	}
	waitFor(t, "all clients registered", func() bool { return s.OnlineClients() == n })

	_ = conns[0].Close()
	waitFor(t, "one client removed", func() bool { return s.OnlineClients() == n-1 })

	if err := r.stop(); err != nil {
		t.Fatalf("Serve = %v, want nil after cancel", err)
	}
	for _, c := range conns[1:] {
		expectClosed(t, c)
	}
	if got := s.OnlineClients(); got != 0 {
		t.Errorf("%d clients still registered after shutdown", got)
	}
	if c, err := net.DialTimeout("tcp", r.addr, time.Second); err == nil {
		_ = c.Close()
		t.Error("listener still accepts connections")
	}
}

// Losing the login server does not stop the game server.
func TestLoginLinkLossKeepsServing(t *testing.T) {
	s := newTestServer(t, config.Default())
	r := start(t, s)
	_ = r.loginSide.Close()

	dial(t, r.addr)
	waitFor(t, "client registered", func() bool { return s.OnlineClients() == 1 })
	if err := r.stop(); err != nil {
		t.Fatalf("Serve = %v", err)
	}
}

func TestIdleTimeout(t *testing.T) {
	s := newTestServer(t, config.Default())
	s.idleTimeout = 100 * time.Millisecond
	r := start(t, s)

	conn := dial(t, r.addr)
	expectClosed(t, conn)
	waitFor(t, "client removed", func() bool { return s.OnlineClients() == 0 })
}

// B2 and B10: an empty or short first packet used to crash the server or be
// used before the read error was checked. Now only that connection closes.
func TestBadFirstPacketClosesOnlyThatConnection(t *testing.T) {
	s := newTestServer(t, config.Default())
	r := start(t, s)
	dial(t, r.addr) // a well-behaved client that must stay connected

	for _, wire := range [][]byte{
		{0x02, 0x00},             // empty body
		{0x00, 0x00},             // size 0
		{0x03, 0x00, 0x00},       // opcode without the version
		{0x05, 0x00, 0x00, 1, 2}, // truncated version
	} {
		bad := dial(t, r.addr)
		if _, err := bad.Write(wire); err != nil {
			t.Fatal(err)
		}
		expectClosed(t, bad)
	}
	waitFor(t, "only the good client left", func() bool { return s.OnlineClients() == 1 })
}
