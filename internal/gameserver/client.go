package gameserver

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/IPeter/l2go/internal/crypt/gamecrypt"
	"github.com/IPeter/l2go/internal/game"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/netutil"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/session"
)

const (
	// maxPacketSize bounds a client packet body.
	maxPacketSize = packets.MaxBodySize

	// writeBufferSize is the writer's buffer. Packets queued together go
	// out in as few writes as fit.
	writeBufferSize = 16 << 10

	// listTimeout bounds building the character list after a restart.
	listTimeout = 10 * time.Second
)

// authInfo is what AuthLogin established for a connection.
type authInfo struct {
	account string // lower case
	keys    session.Keys
}

// outItem is one entry of a send queue: a packet body, or the request to
// close the connection once everything queued before it is written.
type outItem struct {
	body  []byte
	close bool
	delay time.Duration // with close: how long to wait after the last write
}

// clientConfig holds the settings of a connection.
type clientConfig struct {
	idleTimeout time.Duration // read timeout before the world; 0 means none
	queueLimit  int           // packets waiting for the writer
	logger      *slog.Logger
}

// client is one game client connection. It implements game.Session.
//
// Two goroutines serve it. The reader (Server.serveClient) reads, decrypts
// and handles the client's packets one at a time. The writer (writeLoop)
// encrypts and writes the queued packets in order. Every packet, from
// KeyPacket on, goes through the queue, so the output cipher sees the
// packets in the order they were queued. Send never blocks: a client whose
// queue overflows is disconnected.
type client struct {
	srv  *Server // nil in tests that only use the queue
	conn net.Conn
	key  [gamecrypt.KeySize]byte
	cfg  clientConfig

	// ctx ends when the connection is closed.
	ctx    context.Context
	cancel context.CancelFunc

	logger atomic.Pointer[slog.Logger]
	auth   atomic.Pointer[authInfo] // set by AuthLogin
	state  atomic.Int64             // a game.State
	// closing is set by CloseAfter and Close: the client's packets are
	// dropped from then on.
	closing atomic.Bool
	// enterSent is set when EnterWorld went to the game, until the next
	// character selection.
	enterSent atomic.Bool

	// Owned by the reader goroutine.
	in       *gamecrypt.Cipher // nil until KeyPacket turns the cipher on
	counted  bool              // counted in Server.players
	selected bool              // a character was selected: call Disconnected at the end

	// slots holds the object IDs of the characters in the last list sent,
	// by slot.
	slotsMu sync.Mutex
	slots   []int32

	// The send queue.
	qmu     sync.Mutex
	queue   []outItem
	qclosed bool          // set by CloseAfter, an overflow or Close: nothing more is queued
	wake    chan struct{} // tells the writer that the queue has items

	// Owned by the writer goroutine.
	out  *gamecrypt.Cipher
	wbuf []byte

	done       chan struct{} // closed by Close
	closeOnce  sync.Once
	writerDone chan struct{} // closed when writeLoop returns

	// bg runs the character list after a restart.
	bgMu     sync.Mutex
	bgClosed bool
	bg       sync.WaitGroup
}

var _ game.Session = (*client)(nil)

// newClient returns a client for conn whose ciphers use key, with its
// writer running. The output cipher sends the first packet (KeyPacket)
// plain and then turns on; the input cipher starts with the first packet
// after KeyPacket (see Server.handshake). The connection's context is
// derived from ctx.
func newClient(ctx context.Context, srv *Server, conn net.Conn, key [gamecrypt.KeySize]byte, cfg clientConfig) *client {
	if cfg.queueLimit <= 0 {
		cfg.queueLimit = DefaultSendQueue
	}
	ctx, cancel := context.WithCancel(ctx)
	c := &client{
		srv:        srv,
		conn:       conn,
		key:        key,
		cfg:        cfg,
		ctx:        ctx,
		cancel:     cancel,
		out:        gamecrypt.NewServerCipher(key),
		wake:       make(chan struct{}, 1),
		done:       make(chan struct{}),
		writerDone: make(chan struct{}),
	}
	c.logger.Store(cfg.logger)
	go c.writeLoop()
	return c
}

// log returns the connection's logger.
func (c *client) log() *slog.Logger { return c.logger.Load() }

// Close closes the connection at once; queued packets are dropped. It is
// safe to call from any goroutine, more than once.
func (c *client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.closing.Store(true)
		c.qmu.Lock()
		c.qclosed = true
		c.queue = nil
		c.qmu.Unlock()
		close(c.done)
		c.cancel()
		err = c.conn.Close()
	})
	return err
}

// receive reads and decrypts one packet and returns its opcode and the rest
// of the body. Before the world, a client that sends nothing for the idle
// timeout is cut off; in the world there is no timeout, as in Mobius (a
// player standing still sends nothing).
func (c *client) receive() (opcode byte, data []byte, err error) {
	var deadline time.Time
	if st := c.State(); c.cfg.idleTimeout > 0 && st != game.StateEntering && st != game.StateInGame {
		deadline = time.Now().Add(c.cfg.idleTimeout)
	}
	if err := c.conn.SetReadDeadline(deadline); err != nil {
		return 0, nil, err
	}
	body, err := packets.ReadPacket(c.conn, maxPacketSize)
	if err != nil {
		return 0, nil, err
	}
	if c.in != nil {
		c.in.Decrypt(body)
	}
	c.log().Debug("packet received", "data", netutil.Hex(body))
	return body[0], body[1:], nil
}

// Send implements world.Sink: it queues body for the writer and never
// blocks. body must not change afterwards; the writer encrypts a copy, so
// one body can go to many clients. If the queue is full, the client cannot
// keep up and the connection is closed. After CloseAfter or Close, Send
// drops body.
func (c *client) Send(body []byte) {
	if len(body) == 0 {
		return
	}
	c.qmu.Lock()
	if c.qclosed {
		c.qmu.Unlock()
		return
	}
	if n := len(c.queue); n >= c.cfg.queueLimit {
		c.qclosed = true
		c.qmu.Unlock()
		c.log().Warn("client cannot keep up, closing the connection", "queued", n)
		_ = c.Close()
		return
	}
	c.queue = append(c.queue, outItem{body: body})
	c.qmu.Unlock()
	c.signal()
}

// CloseAfter implements game.Session: it queues final (if not empty), and
// once the writer has sent it and everything queued before it, waits delay
// and closes the connection. Packets queued later are dropped, and the
// client's packets are ignored from now on (Mobius closes a client 1 s
// after LeaveWorld). Only the first call counts.
func (c *client) CloseAfter(final []byte, delay time.Duration) {
	c.closing.Store(true)
	c.qmu.Lock()
	if c.qclosed {
		c.qmu.Unlock()
		return
	}
	c.qclosed = true
	if len(final) > 0 {
		c.queue = append(c.queue, outItem{body: final})
	}
	c.queue = append(c.queue, outItem{close: true, delay: delay})
	c.qmu.Unlock()
	c.signal()
}

// signal wakes the writer.
func (c *client) signal() {
	select {
	case c.wake <- struct{}{}:
	default: // a wake-up is already pending
	}
}

// writeLoop writes the queued packets until the connection closes. It
// takes the whole queue at once, writes it through a buffer and flushes
// when the queue is empty. A write error, or the close item of CloseAfter,
// closes the connection.
func (c *client) writeLoop() {
	defer close(c.writerDone)
	defer func() { _ = c.Close() }()
	w := bufio.NewWriterSize(c.conn, writeBufferSize)
	var batch []outItem
	for {
		select {
		case <-c.done:
			return
		case <-c.wake:
		}
		c.qmu.Lock()
		batch, c.queue = c.queue, batch[:0]
		c.qmu.Unlock()
		if len(batch) == 0 {
			continue
		}
		if err := c.conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
			c.writeFailed(err)
			return
		}
		for i := range batch {
			it := batch[i]
			batch[i] = outItem{} // do not keep the body alive
			if it.close {
				if err := w.Flush(); err != nil {
					c.writeFailed(err)
					return
				}
				c.linger(it.delay)
				return
			}
			if err := c.writePacket(w, it.body); err != nil {
				c.writeFailed(err)
				return
			}
		}
		if err := w.Flush(); err != nil {
			c.writeFailed(err)
			return
		}
		if cap(batch) > maxKeptBatch {
			batch = nil // let a burst's array go
		}
	}
}

// maxKeptBatch is the largest batch array the writer keeps for reuse.
const maxKeptBatch = 256

// writePacket frames, encrypts and writes one body. The first packet
// (KeyPacket) goes out plain and turns the output cipher on. A body that
// cannot be framed is a server bug: it is logged and skipped, and the
// cipher stays in step because it never saw it.
func (c *client) writePacket(w io.Writer, body []byte) error {
	if len(body) > packets.MaxBodySize {
		c.log().Error("packet too large, not sent", "opcode", opcodeString(body[0]), "size", len(body))
		return nil
	}
	c.log().Debug("packet sent", "data", netutil.Hex(body))
	n := packets.HeaderSize + len(body)
	if cap(c.wbuf) < n {
		c.wbuf = make([]byte, n, max(n, 2*cap(c.wbuf)))
	}
	buf := c.wbuf[:n]
	binary.LittleEndian.PutUint16(buf, uint16(n)) //nolint:gosec // n <= HeaderSize+MaxBodySize fits in uint16
	copy(buf[packets.HeaderSize:], body)
	c.out.Encrypt(buf[packets.HeaderSize:])
	if _, err := w.Write(buf); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

// writeFailed logs why the writer stopped, unless the connection was
// closed on purpose.
func (c *client) writeFailed(err error) {
	select {
	case <-c.done:
	default:
		c.log().Info("cannot write to the client, closing the connection", "err", err)
	}
}

// linger waits d, or until the connection is closed.
func (c *client) linger(d time.Duration) {
	if d <= 0 {
		return
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
	case <-c.done:
	}
}

// background runs f on its own goroutine, tracked until the connection's
// handler ends. After that it does nothing.
func (c *client) background(f func()) {
	c.bgMu.Lock()
	defer c.bgMu.Unlock()
	if c.bgClosed {
		return
	}
	c.bg.Go(f)
}

// stopBackground waits for the goroutines of background and refuses new
// ones.
func (c *client) stopBackground() {
	c.bgMu.Lock()
	c.bgClosed = true
	c.bgMu.Unlock()
	c.bg.Wait()
}

// setSlots remembers the characters of the list just sent, by slot.
func (c *client) setSlots(ids []int32) {
	c.slotsMu.Lock()
	defer c.slotsMu.Unlock()
	c.slots = ids
}

// slotObjectID returns the object ID of the character in slot of the last
// list sent (Mobius GameClient.getObjectIdForSlot).
func (c *client) slotObjectID(slot int32) (int32, bool) {
	c.slotsMu.Lock()
	defer c.slotsMu.Unlock()
	if slot < 0 || int(slot) >= len(c.slots) {
		return 0, false
	}
	return c.slots[slot], true
}

// Account implements game.Session: the lower case account name, or "" before
// AuthLogin.
func (c *client) Account() string {
	if a := c.auth.Load(); a != nil {
		return a.account
	}
	return ""
}

// PlayKey1 implements game.Session.
func (c *client) PlayKey1() int32 {
	if a := c.auth.Load(); a != nil {
		return a.keys.PlayOK1
	}
	return 0
}

// State implements game.Session.
func (c *client) State() game.State { return game.State(c.state.Load()) }

// SetState implements game.Session. Entering a new selection allows one
// EnterWorld again.
func (c *client) SetState(st game.State) {
	if st == game.StateEntering {
		c.enterSent.Store(false)
	}
	if old := game.State(c.state.Swap(int64(st))); old != st {
		c.log().Debug("connection state changed", "from", stateName(old), "to", stateName(st))
	}
}

// RemoteAddr implements game.Session.
func (c *client) RemoteAddr() string { return c.conn.RemoteAddr().String() }

// ReturnToCharacterSelect implements game.Session, as the end of Mobius
// RequestRestart: it sets StateAuthenticated, sends RestartResponse(true)
// and then the character list. Listing reads the database, so the list is
// built on its own goroutine, bound to the connection rather than to ctx;
// the call does not block. Call it after the character was saved, so the
// list shows the saved state. If the list cannot be built, the connection
// is closed.
func (c *client) ReturnToCharacterSelect(context.Context) {
	c.SetState(game.StateAuthenticated)
	c.Send(serverpackets.NewRestartResponsePacket(true))
	c.background(func() {
		ctx, cancel := context.WithTimeout(c.ctx, listTimeout)
		defer cancel()
		if err := c.srv.sendCharacterList(ctx, c); err != nil {
			if c.ctx.Err() == nil {
				c.log().Warn("cannot send the character list after a restart, closing the connection", "err", err)
			}
			c.CloseAfter(nil, 0)
		}
	})
}

// stateName names a connection state for logs.
func stateName(st game.State) string {
	switch st {
	case game.StateConnected:
		return "connected"
	case game.StateAuthenticated:
		return "authenticated"
	case game.StateEntering:
		return "entering"
	case game.StateInGame:
		return "in game"
	}
	return fmt.Sprintf("state %d", int(st))
}
