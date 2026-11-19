package testclient

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/IPeter/l2go/internal/crypt/gamecrypt"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/session"
)

// GameConn is a client connection to the game server. It is not safe for
// concurrent use.
type GameConn struct {
	conn net.Conn
	// cipher is nil before KeyPacket and when KeyPacket turns encryption
	// off.
	cipher *gamecrypt.Cipher
	key    KeyPacket
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
func DialGame(ctx context.Context, addr string, protocol int32) (*GameConn, error) {
	conn, err := dial(ctx, "game", addr)
	if err != nil {
		return nil, err
	}
	c := &GameConn{conn: conn}
	if err := c.handshake(ctx, protocol); err != nil {
		if errors.Is(err, ErrWrongProtocol) {
			return c, err
		}
		_ = conn.Close()
		return nil, err
	}
	return c, nil
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
	if err != nil {
		return err
	}
	c.key = k
	if k.Encryption {
		key, err := gamecrypt.KeyFromHalf(k.KeyHalf[:])
		if err != nil {
			return fmt.Errorf("testclient: %w", err)
		}
		c.cipher = gamecrypt.NewClientCipher(key)
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
	c := &GameConn{conn: conn}
	defer func() { _ = c.Close() }()
	if err := c.send(ctx, encodeProtocolVersion(PingProtocol)); err != nil {
		return err
	}
	return c.ExpectClosed(ctx)
}

// Key returns the KeyPacket read by DialGame.
func (c *GameConn) Key() KeyPacket { return c.key }

// Close closes the connection.
func (c *GameConn) Close() error { return c.conn.Close() }

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

// ReadRaw reads and decrypts one packet and returns its opcode and the
// rest of it.
func (c *GameConn) ReadRaw(ctx context.Context) (opcode byte, body []byte, err error) {
	b, err := c.readBody(ctx)
	if err != nil {
		return 0, nil, err
	}
	return b[0], b[1:], nil
}

// ReadPacket reads one packet and parses it. It returns one of
// CharacterList, GameLoginFail, NewCharacterSuccess, CharCreateOk or
// CharCreateFail.
func (c *GameConn) ReadPacket(ctx context.Context) (any, error) {
	b, err := c.readBody(ctx)
	if err != nil {
		return nil, err
	}
	return decodeGamePacket(b)
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

// send encrypts body (when the cipher is on) and writes it.
func (c *GameConn) send(ctx context.Context, body []byte) error {
	b := append([]byte(nil), body...)
	if c.cipher != nil {
		c.cipher.Encrypt(b)
	}
	return run(ctx, c.conn, func() error { return packets.WritePacket(c.conn, b) })
}

// readBody reads one packet and decrypts it (when the cipher is on).
func (c *GameConn) readBody(ctx context.Context) ([]byte, error) {
	var body []byte
	err := run(ctx, c.conn, func() error {
		b, err := readFrame(c.conn)
		if err != nil {
			return err
		}
		if c.cipher != nil {
			c.cipher.Decrypt(b)
		}
		body = b
		return nil
	})
	return body, err
}

// read reads and parses one packet. body is the raw decrypted packet, for
// error messages.
func (c *GameConn) read(ctx context.Context, want string) (p any, body []byte, err error) {
	body, err = c.readBody(ctx)
	if err != nil {
		return nil, nil, err
	}
	p, err = decodeGamePacket(body)
	if err != nil {
		var upe *UnexpectedPacketError
		if errors.As(err, &upe) {
			upe.Want = want
		}
		return nil, nil, err
	}
	return p, body, nil
}

// request sends req and reads and parses the answer.
func (c *GameConn) request(ctx context.Context, req []byte, want string) (p any, body []byte, err error) {
	if err := c.send(ctx, req); err != nil {
		return nil, nil, err
	}
	return c.read(ctx, want)
}
