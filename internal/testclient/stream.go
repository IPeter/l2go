package testclient

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// received is one server packet: the decrypted bytes and what the decoder
// made of them.
type received struct {
	// raw is the decrypted packet, opcode included. It is never empty.
	raw []byte
	// p is the decoded packet, nil when err is set.
	p   Packet
	err error
}

// known returns the packet for a request method that waits for want: a
// decode error as it is, and an unknown packet or a KeyPacket after the
// handshake as an *UnexpectedPacketError.
func (r received) known(want string) (Packet, error) {
	if r.err != nil {
		return nil, r.err
	}
	switch r.p.(type) {
	case Unknown, KeyPacket:
		return nil, unexpectedGame(r.raw, want)
	}
	return r.p, nil
}

// queue holds the packets the background reader has read and nobody has
// taken yet. It has no size limit unless limit is set.
type queue struct {
	mu      sync.Mutex
	items   []received
	limit   int
	dropped int
	// err is why the reader stopped, set once.
	err error
	// notify has a token whenever items or err may have changed.
	notify chan struct{}
}

func newQueue(limit int) *queue {
	return &queue{limit: limit, notify: make(chan struct{}, 1)}
}

func (q *queue) signal() {
	select {
	case q.notify <- struct{}{}:
	default:
	}
}

func (q *queue) push(r received) {
	q.mu.Lock()
	if q.limit > 0 && len(q.items) >= q.limit {
		q.items[0] = received{}
		q.items = q.items[1:]
		q.dropped++
	}
	q.items = append(q.items, r)
	q.mu.Unlock()
	q.signal()
}

// close records why the reader stopped. Queued packets stay readable.
func (q *queue) close(err error) {
	q.mu.Lock()
	if q.err == nil {
		q.err = err
	}
	q.mu.Unlock()
	q.signal()
}

// pop removes the oldest packet. ok is false when the queue is empty; err
// is then the reader's final error, or nil while it still runs.
func (q *queue) pop() (r received, ok bool, err error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return received{}, false, q.err
	}
	r = q.items[0]
	q.items[0] = received{}
	q.items = q.items[1:]
	if len(q.items) > 0 {
		// Another waiter may be waiting for the next one.
		q.signal()
	}
	return r, true, nil
}

// next returns the oldest packet, waiting for one when the queue is empty.
// Queued packets come first even when ctx is done.
func (q *queue) next(ctx context.Context) (received, error) {
	for {
		r, ok, err := q.pop()
		if ok {
			return r, nil
		}
		if err != nil {
			return received{}, err
		}
		select {
		case <-q.notify:
		case <-ctx.Done():
			return received{}, fmt.Errorf("testclient: %w", ctx.Err())
		}
	}
}

// drain removes and returns every queued packet.
func (q *queue) drain() []received {
	q.mu.Lock()
	defer q.mu.Unlock()
	items := q.items
	q.items = nil
	return items
}

func (q *queue) droppedCount() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.dropped
}

// StartReader starts the background reader. From then on every server
// packet is decoded as soon as it arrives, applied to the World model,
// recorded, and queued for Next, WaitFor, Drain and the request methods.
// EnterWorld starts it; calling it again does nothing. It must not run
// while another method of the connection reads.
//
// The queue has no size limit (see WithQueueLimit): a long-running client
// that does not read every packet should call Drain now and then.
func (c *GameConn) StartReader() {
	if c.q != nil {
		return
	}
	q := newQueue(c.queueLimit)
	done := make(chan struct{})
	c.q, c.done = q, done
	go c.readLoop(q, done)
}

// readLoop reads until the connection fails or closes. The connection has
// no read deadline here: readBody clears it after each direct read.
func (c *GameConn) readLoop(q *queue, done chan<- struct{}) {
	defer close(done)
	for {
		b, err := readFrame(c.conn)
		if err != nil {
			q.close(err)
			return
		}
		if c.dec != nil {
			c.dec.Decrypt(b)
		}
		q.push(c.receive(b))
	}
}

// receive decodes a decrypted server packet, applies it to the world model
// and records it.
func (c *GameConn) receive(b []byte) received {
	p, err := DecodeServerPacket(b)
	c.recvCount.Add(1)
	if err == nil {
		c.world.apply(p)
	}
	c.rec.add(FromServer, b, p, err)
	return received{raw: b, p: p, err: err}
}

// recv returns the next server packet: from the reader's queue once it
// runs, else straight from the connection.
func (c *GameConn) recv(ctx context.Context) (received, error) {
	if c.q != nil {
		return c.q.next(ctx)
	}
	b, err := c.readBody(ctx)
	if err != nil {
		return received{}, err
	}
	return c.receive(b), nil
}

// Next returns the next server packet, in the order the server sent them.
// A packet that does not match its Mobius layout gives an error wrapping
// ErrMalformed; Next can still read the packets after it. When the server
// has closed the connection and every queued packet has been read, the
// error wraps ErrServerClosed.
func (c *GameConn) Next(ctx context.Context) (Packet, error) {
	r, err := c.recv(ctx)
	if err != nil {
		return nil, err
	}
	if r.err != nil {
		return nil, r.err
	}
	return r.p, nil
}

// WaitFor reads packets until match accepts one, and returns it. It drops
// the packets match refuses (the world model and the recorder still see
// them). When the context ends or the connection closes before a match,
// the error is a *WaitError naming the packets read meanwhile. A malformed
// packet ends the wait with its error.
//
// WaitFor also takes packets that were queued before the call: Drain first
// when an older packet of the same kind may still be queued.
func (c *GameConn) WaitFor(ctx context.Context, match func(Packet) bool) (Packet, error) {
	return c.await(ctx, "a matching packet", match)
}

// Drain removes the packets queued so far and returns them, without
// waiting. Packets that could not be decoded are left out of the list and
// reported in the error. Before the reader runs, Drain returns nothing.
func (c *GameConn) Drain() ([]Packet, error) {
	if c.q == nil {
		return nil, nil
	}
	var (
		ps   []Packet
		errs []string
	)
	for _, r := range c.q.drain() {
		if r.err != nil {
			errs = append(errs, r.err.Error())
			continue
		}
		ps = append(ps, r.p)
	}
	if len(errs) > 0 {
		return ps, fmt.Errorf("%w: %d drained packets: %s", ErrMalformed, len(errs), strings.Join(errs, "; "))
	}
	return ps, nil
}

// Dropped returns how many packets the queue dropped because of
// WithQueueLimit.
func (c *GameConn) Dropped() int {
	if c.q == nil {
		return 0
	}
	return c.q.droppedCount()
}

// maxSeenNames limits the packet names a WaitError lists.
const maxSeenNames = 64

// WaitError reports that a wait ended before the packet it waited for
// came: the context ended or the connection closed. It wraps that error,
// so errors.Is(err, context.DeadlineExceeded) and errors.Is(err,
// ErrServerClosed) work.
type WaitError struct {
	// Want describes the packet waited for.
	Want string
	// Seen names the packets read during the wait, at most 64 of them;
	// SeenCount counts all.
	Seen      []string
	SeenCount int
	Err       error
}

func (e *WaitError) Error() string {
	msg := fmt.Sprintf("testclient: waiting for %s: %v", e.Want, e.Err)
	if e.SeenCount == 0 {
		return msg + " (no packets meanwhile)"
	}
	more := ""
	if e.SeenCount > len(e.Seen) {
		more = fmt.Sprintf(", ... %d more", e.SeenCount-len(e.Seen))
	}
	return fmt.Sprintf("%s (read meanwhile: %s%s)", msg, strings.Join(e.Seen, ", "), more)
}

func (e *WaitError) Unwrap() error { return e.Err }

// see adds a packet name to the list of packets read during the wait.
func (e *WaitError) see(name string) {
	e.SeenCount++
	if len(e.Seen) < maxSeenNames {
		e.Seen = append(e.Seen, name)
	}
}

// await reads packets until match accepts one and returns it, dropping the
// others.
func (c *GameConn) await(ctx context.Context, want string, match func(Packet) bool) (Packet, error) {
	we := &WaitError{Want: want}
	for {
		r, err := c.recv(ctx)
		if err != nil {
			we.Err = err
			return nil, we
		}
		if r.err != nil {
			return nil, r.err
		}
		if match(r.p) {
			return r.p, nil
		}
		we.see(r.p.PacketName())
	}
}

// typeName returns the packet name of T.
func typeName[T Packet]() string {
	var zero T
	if any(zero) == nil {
		return "a packet"
	}
	return zero.PacketName()
}

// Await waits like WaitFor for the next packet of type T that match
// accepts; a nil match accepts any T. T is one of the packet types of this
// package, such as CreatureSay or NpcHtmlMessage.
func Await[T Packet](ctx context.Context, c *GameConn, match func(T) bool) (T, error) {
	p, err := c.await(ctx, typeName[T](), func(p Packet) bool {
		v, ok := p.(T)
		return ok && (match == nil || match(v))
	})
	if err != nil {
		var zero T
		return zero, err
	}
	return p.(T), nil
}

// Expect reads the next packet and returns it when it is a T. Any other
// packet gives an *UnexpectedPacketError; use it to check the exact order
// of the server's packets.
func Expect[T Packet](ctx context.Context, c *GameConn) (T, error) {
	var zero T
	r, err := c.recv(ctx)
	if err != nil {
		return zero, err
	}
	if r.err != nil {
		return zero, r.err
	}
	if v, ok := r.p.(T); ok {
		return v, nil
	}
	return zero, unexpectedGame(r.raw, typeName[T]())
}

// Find returns the first packet of type T in ps.
func Find[T Packet](ps []Packet) (T, bool) {
	for _, p := range ps {
		if v, ok := p.(T); ok {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// FindAll returns the packets of type T in ps, in order.
func FindAll[T Packet](ps []Packet) []T {
	var out []T
	for _, p := range ps {
		if v, ok := p.(T); ok {
			out = append(out, v)
		}
	}
	return out
}

// Names returns the packet names of ps, in order.
func Names(ps []Packet) []string {
	names := make([]string, len(ps))
	for i, p := range ps {
		names[i] = p.PacketName()
	}
	return names
}
