package link

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/IPeter/l2go/internal/session"
)

// Client defaults, used for zero ClientConfig fields.
const (
	defaultDialTimeout  = 5 * time.Second
	defaultPingInterval = 5 * time.Second
	defaultMinBackoff   = time.Second
	defaultMaxBackoff   = 30 * time.Second
)

// ClientConfig configures the game server side of the link.
type ClientConfig struct {
	// Addr is the login server's address for game servers (host:port).
	Addr string
	// ServerID is this game server's ID, 1-255.
	ServerID int
	// Secret is the secret the login server has for ServerID.
	Secret string
	// MaxPlayers is reported to the login server (clamped to 0-65535).
	MaxPlayers int

	// DialTimeout bounds connecting plus registering. Default 5s.
	DialTimeout time.Duration
	// PingInterval is how often the client pings the login server.
	// Default 5s.
	PingInterval time.Duration
	// ReadTimeout drops the link when the login server sends nothing for
	// this long. Default 3 × PingInterval.
	ReadTimeout time.Duration
	// WriteTimeout bounds each write. Default 10s.
	WriteTimeout time.Duration
	// MinBackoff is the first delay before reconnecting; it doubles after
	// each failed attempt up to MaxBackoff. Defaults 1s and 30s.
	MinBackoff time.Duration
	MaxBackoff time.Duration
}

// Client is the game server side of the link. Run keeps it registered with
// the login server; the other methods are safe for concurrent use.
type Client struct {
	cfg    ClientConfig
	logger *slog.Logger

	players     atomic.Uint32 // last SetPlayers value, 0-65535
	statusDirty chan struct{} // capacity 1: a Status is due
	running     atomic.Bool   // Run is active

	mu   sync.Mutex
	conn *clientConn // nil while not registered
}

// NewClient returns a client for cfg. Call Run to connect. A nil logger
// discards logs.
func NewClient(cfg ClientConfig, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	cfg.DialTimeout = orDefault(cfg.DialTimeout, defaultDialTimeout)
	cfg.PingInterval = orDefault(cfg.PingInterval, defaultPingInterval)
	cfg.ReadTimeout = orDefault(cfg.ReadTimeout, 3*cfg.PingInterval)
	cfg.WriteTimeout = orDefault(cfg.WriteTimeout, defaultWriteTimeout)
	cfg.MinBackoff = orDefault(cfg.MinBackoff, defaultMinBackoff)
	cfg.MaxBackoff = max(orDefault(cfg.MaxBackoff, defaultMaxBackoff), cfg.MinBackoff)
	cfg.MaxPlayers = clampPlayers(cfg.MaxPlayers)
	return &Client{
		cfg:         cfg,
		logger:      logger.With("loginserver", cfg.Addr),
		statusDirty: make(chan struct{}, 1),
	}
}

// Connected reports whether the client is registered with the login
// server right now.
func (c *Client) Connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

// SetPlayers sets the player count reported to the login server (clamped
// to 0-65535). Updates are coalesced: the login server gets the latest
// value soon, not every value. While disconnected, the value is sent after
// the next registration.
func (c *Client) SetPlayers(n int) {
	c.players.Store(uint32(clampPlayers(n))) //nolint:gosec // clamped to 0-65535
	select {
	case c.statusDirty <- struct{}{}:
	default:
	}
}

// VerifySession asks the login server whether k are the current session
// keys of account, which uses them up on success. It returns
// ErrNotConnected (possibly wrapped) while the link is down, and ctx's
// error if ctx ends first. An account name the link cannot carry (empty,
// longer than 64 characters or containing NUL) is reported as not
// verified without asking.
func (c *Client) VerifySession(ctx context.Context, account string, k session.Keys) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if !validAccount(account) {
		return false, nil
	}
	c.mu.Lock()
	cc := c.conn
	c.mu.Unlock()
	if cc == nil {
		return false, ErrNotConnected
	}
	id, result, ok := cc.addPending()
	if !ok {
		return false, ErrNotConnected
	}
	defer cc.removePending(id)

	if err := cc.send(verifySessionMsg{requestID: id, account: account, keys: k}); err != nil {
		return false, fmt.Errorf("%w: %w", ErrNotConnected, err)
	}
	select {
	case verified := <-result:
		return verified, nil
	case <-ctx.Done():
		return false, ctx.Err()
	case <-cc.done:
		select {
		case verified := <-result: // answered just before the link went down
			return verified, nil
		default:
			return false, ErrNotConnected
		}
	}
}

// Run connects to the login server, registers and keeps the link alive
// until ctx is done, then returns nil. When the link fails it reconnects
// with exponential backoff. It returns a *RejectedError if the login
// server refuses the registration for a reason retrying cannot fix (wrong
// secret, unknown ID, other protocol version), and an error for an
// invalid config. Only one Run may be active at a time.
func (c *Client) Run(ctx context.Context) error {
	if err := c.validate(); err != nil {
		return err
	}
	if !c.running.CompareAndSwap(false, true) {
		return errors.New("link: Run is already running")
	}
	defer c.running.Store(false)

	backoff := c.cfg.MinBackoff
	for {
		registered, err := c.runConn(ctx)
		if ctx.Err() != nil {
			return nil
		}
		var rejected *RejectedError
		if errors.As(err, &rejected) && rejected.Reason != ReasonAlreadyRegistered {
			c.logger.Error("the login server refused the registration", "reason", rejected.Reason.String())
			return rejected
		}
		if registered {
			backoff = c.cfg.MinBackoff
		}
		delay := jitter(backoff)
		if registered {
			c.logger.Warn("lost the link to the login server", "err", err, "retryIn", delay)
		} else {
			c.logger.Warn("cannot register with the login server", "err", err, "retryIn", delay)
		}
		t := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			t.Stop()
			return nil
		case <-t.C:
		}
		backoff = min(2*backoff, c.cfg.MaxBackoff)
	}
}

func (c *Client) validate() error {
	switch {
	case c.cfg.Addr == "":
		return errors.New("link: no login server address")
	case c.cfg.ServerID < 1 || c.cfg.ServerID > 255:
		return fmt.Errorf("link: server ID %d is not in 1-255", c.cfg.ServerID)
	case c.cfg.Secret == "":
		return errors.New("link: empty secret")
	}
	return nil
}

// runConn makes one connection and serves it until it fails or ctx is
// done. registered reports whether the registration succeeded.
func (c *Client) runConn(ctx context.Context) (registered bool, err error) {
	nc, err := c.connect(ctx)
	if err != nil {
		return false, err
	}
	cc := &clientConn{
		nc:           nc,
		writeTimeout: c.cfg.WriteTimeout,
		pending:      make(map[uint32]chan bool),
		done:         make(chan struct{}),
	}
	c.mu.Lock()
	c.conn = cc
	c.mu.Unlock()
	c.logger.Info("registered with the login server", "server", c.cfg.ServerID)

	var readErr error
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		readErr = cc.readLoop(c.cfg.ReadTimeout)
	}()

	err = c.keepAlive(ctx, cc, readDone)

	c.mu.Lock()
	c.conn = nil
	c.mu.Unlock()
	cc.close()
	<-readDone
	if err == nil {
		err = readErr
	}
	return true, err
}

// connect dials the login server and registers. Cancelling ctx aborts it.
func (c *Client) connect(ctx context.Context) (net.Conn, error) {
	d := net.Dialer{Timeout: c.cfg.DialTimeout}
	nc, err := d.DialContext(ctx, "tcp", c.cfg.Addr)
	if err != nil {
		return nil, err
	}
	stop := context.AfterFunc(ctx, func() { _ = nc.Close() })
	defer stop()
	if err := c.register(nc); err != nil {
		_ = nc.Close()
		return nil, err
	}
	return nc, nil
}

// register answers the login server's challenge.
func (c *Client) register(nc net.Conn) error {
	if err := nc.SetDeadline(time.Now().Add(c.cfg.DialTimeout)); err != nil {
		return err
	}
	msg, err := readMsg(nc)
	if err != nil {
		return fmt.Errorf("read challenge: %w", err)
	}
	ch, ok := msg.(challengeMsg)
	if !ok {
		return fmt.Errorf("expected a challenge, got %T", msg)
	}
	if ch.version != protocolVersion {
		return &RejectedError{Reason: ReasonUnsupportedVersion}
	}
	reg := registerMsg{
		version:    protocolVersion,
		serverID:   uint8(c.cfg.ServerID),    //nolint:gosec // validated to 1-255
		maxPlayers: uint16(c.cfg.MaxPlayers), //nolint:gosec // clamped to 0-65535
	}
	copy(reg.mac[:], computeMAC([]byte(c.cfg.Secret), ch.challenge))
	if err := writeMsg(nc, c.cfg.DialTimeout, reg); err != nil {
		return fmt.Errorf("send register: %w", err)
	}
	msg, err = readMsg(nc)
	if err != nil {
		return fmt.Errorf("read register result: %w", err)
	}
	res, ok := msg.(registerResultMsg)
	if !ok {
		return fmt.Errorf("expected a register result, got %T", msg)
	}
	if res.reason != reasonNone {
		return &RejectedError{Reason: res.reason}
	}
	return nil
}

// keepAlive sends the status and pings until ctx is done, the reader stops
// or a write fails.
func (c *Client) keepAlive(ctx context.Context, cc *clientConn, readDone <-chan struct{}) error {
	// The registration carried no player count, so always send one.
	select {
	case <-c.statusDirty:
	default:
	}
	if err := cc.send(c.status()); err != nil {
		return err
	}
	ticker := time.NewTicker(c.cfg.PingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-readDone:
			return nil
		case <-ticker.C:
			if err := cc.send(pingMsg{}); err != nil {
				return err
			}
		case <-c.statusDirty:
			if err := cc.send(c.status()); err != nil {
				return err
			}
		}
	}
}

func (c *Client) status() statusMsg {
	return statusMsg{
		players:    uint16(c.players.Load()), //nolint:gosec // stored clamped to 0-65535
		maxPlayers: uint16(c.cfg.MaxPlayers), //nolint:gosec // clamped to 0-65535
	}
}

// clientConn is one registered connection to the login server.
type clientConn struct {
	nc           net.Conn
	writeTimeout time.Duration

	writeMu sync.Mutex // serialises writes

	mu      sync.Mutex
	nextID  uint32
	pending map[uint32]chan bool // VerifySession waiters by request ID
	closed  bool
	done    chan struct{} // closed by close
}

// send writes one message. A failed write closes the connection, which
// stops the reader and tears the link down.
func (cc *clientConn) send(m message) error {
	cc.writeMu.Lock()
	defer cc.writeMu.Unlock()
	if err := writeMsg(cc.nc, cc.writeTimeout, m); err != nil {
		_ = cc.nc.Close()
		return err
	}
	return nil
}

// readLoop handles messages from the login server until the connection
// fails.
func (cc *clientConn) readLoop(timeout time.Duration) error {
	for {
		if err := cc.nc.SetReadDeadline(time.Now().Add(timeout)); err != nil {
			return err
		}
		msg, err := readMsg(cc.nc)
		if err != nil {
			return err
		}
		switch m := msg.(type) {
		case verifySessionResultMsg:
			cc.deliver(m.requestID, m.ok)
		case pingMsg:
			if err := cc.send(pongMsg{}); err != nil {
				return err
			}
		case pongMsg:
		default:
			return fmt.Errorf("unexpected %T from the login server", msg)
		}
	}
}

// addPending registers a VerifySession waiter. ok is false once the
// connection is closed.
func (cc *clientConn) addPending() (id uint32, result chan bool, ok bool) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	if cc.closed {
		return 0, nil, false
	}
	for {
		cc.nextID++
		if _, used := cc.pending[cc.nextID]; !used {
			break
		}
	}
	result = make(chan bool, 1)
	cc.pending[cc.nextID] = result
	return cc.nextID, result, true
}

func (cc *clientConn) removePending(id uint32) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	delete(cc.pending, id)
}

// deliver hands a result to its waiter. Results nobody waits for any more
// (the caller gave up) are dropped.
func (cc *clientConn) deliver(id uint32, ok bool) {
	cc.mu.Lock()
	result, found := cc.pending[id]
	delete(cc.pending, id)
	cc.mu.Unlock()
	if found {
		result <- ok // buffered, and each ID is delivered at most once
	}
}

// close wakes every waiter and closes the connection.
func (cc *clientConn) close() {
	cc.mu.Lock()
	if !cc.closed {
		cc.closed = true
		close(cc.done)
	}
	cc.mu.Unlock()
	_ = cc.nc.Close()
}

// jitter returns a random delay in [d/2, d].
func jitter(d time.Duration) time.Duration {
	return d/2 + rand.N(d/2+1) //nolint:gosec // timing jitter, not security
}

func clampPlayers(n int) int {
	return min(max(n, 0), math.MaxUint16)
}
