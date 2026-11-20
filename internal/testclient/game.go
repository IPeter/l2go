package testclient

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/IPeter/l2go/internal/crypt/gamecrypt"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/session"
)

// GameConn is a client connection to the game server.
//
// Its methods are meant for one goroutine at a time, with these exceptions:
// World and Recorder may be used from any goroutine, and the methods that
// only send (Say, SendMove, Action, SendRaw, ...) may run while another
// goroutine waits for packets.
type GameConn struct {
	conn net.Conn
	key  KeyPacket

	// dec decrypts server packets and enc encrypts client packets. They
	// start from the same key and roll apart. Both are nil before KeyPacket
	// and when KeyPacket turns encryption off.
	dec, enc *gamecrypt.Cipher

	// sendMu orders the writes and the output key.
	sendMu sync.Mutex

	world *World
	rec   *Recorder

	// queueLimit is the WithQueueLimit setting.
	queueLimit int

	// q is the background reader's queue, nil until StartReader. done is
	// closed when the reader stops.
	q    *queue
	done chan struct{}

	// sentCount and recvCount count the packets in each direction.
	sentCount, recvCount atomic.Int64

	// clickMu guards lastClick, when sendClick last sent a click.
	clickMu   sync.Mutex
	lastClick time.Time
}

// DialOption configures DialGame.
type DialOption func(*GameConn)

// WithRecorder records every packet of the connection in r, from
// ProtocolVersion and KeyPacket on.
func WithRecorder(r *Recorder) DialOption {
	return func(c *GameConn) { c.rec = r }
}

// WithQueueLimit keeps at most n packets in the background reader's queue.
// When it is full the oldest packet is dropped (the world model and the
// recorder have already seen it) and Dropped counts it. Load-test bots use
// it so that packets they never read do not pile up. n <= 0 means no limit,
// the default.
func WithQueueLimit(n int) DialOption {
	return func(c *GameConn) { c.queueLimit = n }
}

// CreateResult is the answer to CharacterCreate.
type CreateResult struct {
	// Created is true when the server sent CharCreateOk.
	Created bool
	// Reason is the CharCreateFail reason when Created is false.
	Reason CreateFailReason
	// Characters is the updated character list sent with CharCreateOk.
	Characters CharacterList
	// ListBeforeOk is true when the list came before CharCreateOk. Mobius
	// sends the list first (CharacterCreate.initNewChar sends it, then
	// runImpl sends CharCreateOk).
	ListBeforeOk bool
}

// DialGame connects to the game server at addr, sends ProtocolVersion with
// protocol in plain text, reads KeyPacket and sets up the XOR cipher from
// its key half (unless KeyPacket turns encryption off).
//
// When KeyPacket's result is 0, DialGame returns the open connection
// together with an error wrapping ErrWrongProtocol, so the caller can look
// at Key and check with ExpectClosed that the server hangs up. The caller
// must close it. On any other error the connection is closed and nil.
func DialGame(ctx context.Context, addr string, protocol int32, opts ...DialOption) (*GameConn, error) {
	conn, err := dial(ctx, "game", addr)
	if err != nil {
		return nil, err
	}
	c := newGameConn(conn, opts)
	if err := c.handshake(ctx, protocol); err != nil {
		if errors.Is(err, ErrWrongProtocol) {
			return c, err
		}
		_ = conn.Close()
		return nil, err
	}
	return c, nil
}

func newGameConn(conn net.Conn, opts []DialOption) *GameConn {
	c := &GameConn{conn: conn, world: newWorld()}
	for _, o := range opts {
		o(c)
	}
	return c
}

func (c *GameConn) handshake(ctx context.Context, protocol int32) error {
	if err := c.send(ctx, encodeProtocolVersion(protocol)); err != nil {
		return err
	}
	body, err := c.readBody(ctx)
	if err != nil {
		return err
	}
	k, err := parseKeyPacket(body)
	c.recvCount.Add(1)
	if err != nil {
		c.rec.add(FromServer, body, nil, err)
		return err
	}
	c.rec.add(FromServer, body, k, nil)
	c.key = k
	if k.Encryption {
		key, err := gamecrypt.KeyFromHalf(k.KeyHalf[:])
		if err != nil {
			return fmt.Errorf("testclient: %w", err)
		}
		c.dec = gamecrypt.NewClientCipher(key)
		c.enc = gamecrypt.NewClientCipher(key)
	}
	if k.Result == 0 {
		return fmt.Errorf("%w: protocol %d", ErrWrongProtocol, protocol)
	}
	return nil
}

// Ping sends ProtocolVersion with version -2 and returns nil if the server
// closes the connection without answering.
func Ping(ctx context.Context, addr string) error {
	conn, err := dial(ctx, "game", addr)
	if err != nil {
		return err
	}
	c := newGameConn(conn, nil)
	defer func() { _ = c.Close() }()
	if err := c.send(ctx, encodeProtocolVersion(PingProtocol)); err != nil {
		return err
	}
	return c.ExpectClosed(ctx)
}

// Key returns the KeyPacket read by DialGame.
func (c *GameConn) Key() KeyPacket { return c.key }

// World returns the client's view of the world, which the connection keeps
// up to date from every server packet it reads.
func (c *GameConn) World() *World { return c.world }

// Recorder returns the recorder given with WithRecorder, or nil.
func (c *GameConn) Recorder() *Recorder { return c.rec }

// PacketsSent returns the number of packets the client has sent.
func (c *GameConn) PacketsSent() int64 { return c.sentCount.Load() }

// PacketsReceived returns the number of server packets the client has read.
func (c *GameConn) PacketsReceived() int64 { return c.recvCount.Load() }

// Close closes the connection and waits for the background reader, if it
// runs, to stop.
func (c *GameConn) Close() error {
	err := c.conn.Close()
	if c.done != nil {
		<-c.done
	}
	return err
}

// AuthLogin sends AuthLogin with account (sent in lower case, as Mobius
// stores it) and the four session keys, and returns the character list
// from CharSelectionInfo. A game server LoginFail answer gives a
// *GameLoginFailError.
func (c *GameConn) AuthLogin(ctx context.Context, account string, k session.Keys) (CharacterList, error) {
	const want = "CharSelectionInfo"
	p, body, err := c.request(ctx, encodeAuthLogin(account, k), want)
	if err != nil {
		return CharacterList{}, err
	}
	switch v := p.(type) {
	case CharacterList:
		return v, nil
	case GameLoginFail:
		return CharacterList{}, &GameLoginFailError{Reason: v.Reason}
	}
	return CharacterList{}, unexpectedGame(body, want)
}

// NewCharacter sends NewCharacter and returns the starting classes from
// NewCharacterSuccess.
func (c *GameConn) NewCharacter(ctx context.Context) ([]CharTemplate, error) {
	const want = "NewCharacterSuccess"
	p, body, err := c.request(ctx, encodeNewCharacter(), want)
	if err != nil {
		return nil, err
	}
	if v, ok := p.(NewCharacterSuccess); ok {
		return v.Templates, nil
	}
	return nil, unexpectedGame(body, want)
}

// CreateCharacter sends CharacterCreate. On success it reads both
// CharCreateOk and the updated CharSelectionInfo, in either order. A
// CharCreateFail answer is a result with Created false, not an error.
func (c *GameConn) CreateCharacter(ctx context.Context, req CreateRequest) (CreateResult, error) {
	const want = "CharCreateOk, CharSelectionInfo or CharCreateFail"
	p, body, err := c.request(ctx, encodeCharacterCreate(req), want)
	if err != nil {
		return CreateResult{}, err
	}
	switch v := p.(type) {
	case CharCreateFail:
		return CreateResult{Reason: v.Reason}, nil
	case CharCreateOk:
		if err := checkCreateOk(v); err != nil {
			return CreateResult{}, err
		}
		const next = "CharSelectionInfo after CharCreateOk"
		p, body, err := c.read(ctx, next)
		if err != nil {
			return CreateResult{}, err
		}
		list, ok := p.(CharacterList)
		if !ok {
			return CreateResult{}, unexpectedGame(body, next)
		}
		return CreateResult{Created: true, Characters: list}, nil
	case CharacterList:
		const next = "CharCreateOk after CharSelectionInfo"
		p, body, err := c.read(ctx, next)
		if err != nil {
			return CreateResult{}, err
		}
		ok, isOk := p.(CharCreateOk)
		if !isOk {
			return CreateResult{}, unexpectedGame(body, next)
		}
		if err := checkCreateOk(ok); err != nil {
			return CreateResult{}, err
		}
		return CreateResult{Created: true, Characters: v, ListBeforeOk: true}, nil
	}
	return CreateResult{}, unexpectedGame(body, want)
}

func checkCreateOk(v CharCreateOk) error {
	if v.Result != 1 {
		return malformed("CharCreateOk", "result %d, Mobius sends 1", v.Result)
	}
	return nil
}

// SendRaw sends a packet made of opcode and body, encrypted if the cipher
// is on.
func (c *GameConn) SendRaw(ctx context.Context, opcode byte, body []byte) error {
	b := make([]byte, 0, 1+len(body))
	b = append(b, opcode)
	b = append(b, body...)
	return c.send(ctx, b)
}

// ReadRaw reads the next packet and returns its opcode and the rest of it,
// decrypted. Once the background reader runs, it takes the packet from the
// queue; either way the world model and the recorder see it.
func (c *GameConn) ReadRaw(ctx context.Context) (opcode byte, body []byte, err error) {
	r, err := c.recv(ctx)
	if err != nil {
		return 0, nil, err
	}
	return r.raw[0], r.raw[1:], nil
}

// ReadPacket reads the next packet and parses it. It returns any packet
// DecodeServerPacket parses, such as CharacterList, CharSelected or
// UserInfo. An unknown opcode, or a KeyPacket after the handshake, gives an
// *UnexpectedPacketError.
func (c *GameConn) ReadPacket(ctx context.Context) (any, error) {
	r, err := c.recv(ctx)
	if err != nil {
		return nil, err
	}
	p, err := r.known("")
	if err != nil {
		return nil, err
	}
	return p, nil
}

// ExpectClosed returns nil if the server closes the connection before
// sending anything else. A packet that arrives instead is returned as an
// *UnexpectedPacketError.
func (c *GameConn) ExpectClosed(ctx context.Context) error {
	return expectClosed(
		func() (byte, []byte, error) { return c.ReadRaw(ctx) },
		func(op byte, body []byte) error {
			return unexpectedGame(append([]byte{op}, body...), "connection close")
		})
}

// send records body, encrypts a copy of it (when the cipher is on) and
// writes it. Only the write deadline follows ctx, so a send does not
// disturb the background reader.
func (c *GameConn) send(ctx context.Context, body []byte) error {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	c.rec.add(FromClient, body, nil, nil)
	c.sentCount.Add(1)
	b := append([]byte(nil), body...)
	if c.enc != nil {
		c.enc.Encrypt(b)
	}
	return runWith(ctx, c.conn.SetWriteDeadline, func() error { return packets.WritePacket(c.conn, b) })
}

// readBody reads one packet straight from the connection and decrypts it
// (when the cipher is on). Only the handshake and recv use it, before the
// background reader starts.
func (c *GameConn) readBody(ctx context.Context) ([]byte, error) {
	var body []byte
	err := runWith(ctx, c.conn.SetReadDeadline, func() error {
		b, err := readFrame(c.conn)
		if err != nil {
			return err
		}
		if c.dec != nil {
			c.dec.Decrypt(b)
		}
		body = b
		return nil
	})
	return body, err
}

// read reads the next packet for a request that waits for want. body is
// the raw packet, for error messages.
func (c *GameConn) read(ctx context.Context, want string) (p Packet, body []byte, err error) {
	r, err := c.recv(ctx)
	if err != nil {
		return nil, nil, err
	}
	p, err = r.known(want)
	if err != nil {
		return nil, nil, err
	}
	return p, r.raw, nil
}

// request sends req and reads the answer.
func (c *GameConn) request(ctx context.Context, req []byte, want string) (p Packet, body []byte, err error) {
	if err := c.send(ctx, req); err != nil {
		return nil, nil, err
	}
	return c.read(ctx, want)
}
