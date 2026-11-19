package testclient

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"net"

	"github.com/IPeter/l2go/internal/crypt/logincrypt"
	"github.com/IPeter/l2go/internal/crypt/rsakey"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/session"
)

// LoginConn is a client connection to the login server. It is not safe for
// concurrent use.
type LoginConn struct {
	conn  net.Conn
	crypt *logincrypt.ClientSession
	init  Init
	pub   *rsa.PublicKey
	keys  session.Keys
}

// LoginResult is the answer to RequestAuthLogin. Exactly one field is set.
type LoginResult struct {
	Ok     *LoginOk
	Fail   *LoginFail
	Kicked *AccountKicked
	// ServerList is set when the server skips LoginOk and sends the list
	// at once, as Mobius does when its licence screen is off. L2Go always
	// sends LoginOk.
	ServerList *ServerListResult
}

func (r LoginResult) String() string {
	switch {
	case r.Ok != nil:
		return fmt.Sprintf("LoginOk%+v", *r.Ok)
	case r.Fail != nil:
		return fmt.Sprintf("LoginFail{Reason:%v}", r.Fail.Reason)
	case r.Kicked != nil:
		return fmt.Sprintf("AccountKicked{Reason:%v}", r.Kicked.Reason)
	case r.ServerList != nil:
		return fmt.Sprintf("ServerList%+v", *r.ServerList)
	default:
		return "LoginResult{}"
	}
}

// PlayResult is the answer to RequestServerLogin. Exactly one field is
// set.
type PlayResult struct {
	Ok   *PlayOk
	Fail *PlayFail
	// LoginFail is set when the server answers with LoginFail, as Mobius
	// does for wrong login keys or when the login server is down.
	LoginFail *LoginFail
}

func (r PlayResult) String() string {
	switch {
	case r.Ok != nil:
		return fmt.Sprintf("PlayOk%+v", *r.Ok)
	case r.Fail != nil:
		return fmt.Sprintf("PlayFail{Reason:%v}", r.Fail.Reason)
	case r.LoginFail != nil:
		return fmt.Sprintf("LoginFail{Reason:%v}", r.LoginFail.Reason)
	default:
		return "PlayResult{}"
	}
}

// DialLogin connects to the login server at addr, reads and checks Init,
// and switches to the session Blowfish key it carries.
func DialLogin(ctx context.Context, addr string) (*LoginConn, error) {
	conn, err := dial(ctx, "login", addr)
	if err != nil {
		return nil, err
	}
	c := &LoginConn{conn: conn, crypt: logincrypt.NewClientSession()}
	if err := c.readInit(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return c, nil
}

func (c *LoginConn) readInit(ctx context.Context) error {
	plain, err := c.readPlain(ctx)
	if err != nil {
		return err
	}
	in, err := parseInit(plain)
	if err != nil {
		return err
	}
	pub, err := rsakey.PublicKeyFromScrambled(in.ScrambledModulus[:])
	if err != nil {
		return malformed("Init", "RSA key: %v", err)
	}
	if err := c.crypt.SetSessionKey(in.BlowfishKey[:]); err != nil {
		return malformed("Init", "Blowfish key: %v", err)
	}
	c.init, c.pub = in, pub
	return nil
}

// Init returns the Init packet read by DialLogin.
func (c *LoginConn) Init() Init { return c.init }

// PublicKey returns the RSA key from Init.
func (c *LoginConn) PublicKey() *rsa.PublicKey { return c.pub }

// Keys returns the session keys received so far: the login keys after
// LoginOk, the play keys after PlayOk.
func (c *LoginConn) Keys() session.Keys { return c.keys }

// SetLoginKeys replaces the login keys that ServerList and ServerLogin
// send, for tests with wrong keys.
func (c *LoginConn) SetLoginKeys(loginOK1, loginOK2 int32) {
	c.keys.LoginOK1, c.keys.LoginOK2 = loginOK1, loginOK2
}

// Close closes the connection.
func (c *LoginConn) Close() error { return c.conn.Close() }

// AuthGameGuard sends AuthGameGuard with the session ID from Init and
// expects GGAuth carrying the same session ID. A LoginFail answer gives a
// *LoginFailError.
func (c *LoginConn) AuthGameGuard(ctx context.Context) error {
	const want = "GGAuth"
	p, data, err := c.request(ctx, encodeAuthGameGuard(c.init.SessionID), want)
	if err != nil {
		return err
	}
	switch v := p.(type) {
	case GGAuth:
		if v.Response != c.init.SessionID {
			return malformed(want, "response 0x%08x, want the session ID 0x%08x", v.Response, c.init.SessionID)
		}
		return nil
	case LoginFail:
		return &LoginFailError{Reason: v.Reason}
	}
	return unexpectedLogin(data, want)
}

// Login sends RequestAuthLogin with the credentials in an RSA block of 128
// bytes, or 256 bytes when newAuth is set. A LoginOk answer stores the
// login keys for ServerList and ServerLogin.
func (c *LoginConn) Login(ctx context.Context, user, password string, newAuth bool) (LoginResult, error) {
	const want = "LoginOk, LoginFail or AccountKicked"
	size := rsakey.BlockSize
	if newAuth {
		size = 2 * rsakey.BlockSize
	}
	block, err := rsakey.CredentialsBlock(user, password, size)
	if err != nil {
		return LoginResult{}, fmt.Errorf("testclient: %w", err)
	}
	enc, err := rsakey.EncryptRaw(c.pub, block)
	if err != nil {
		return LoginResult{}, fmt.Errorf("testclient: %w", err)
	}
	p, data, err := c.request(ctx, encodeRequestAuthLogin(enc), want)
	if err != nil {
		return LoginResult{}, err
	}
	switch v := p.(type) {
	case LoginOk:
		c.keys.LoginOK1, c.keys.LoginOK2 = v.LoginOK1, v.LoginOK2
		return LoginResult{Ok: &v}, nil
	case LoginFail:
		return LoginResult{Fail: &v}, nil
	case AccountKicked:
		return LoginResult{Kicked: &v}, nil
	case ServerListResult:
		return LoginResult{ServerList: &v}, nil
	}
	return LoginResult{}, unexpectedLogin(data, want)
}

// ServerList sends RequestServerList with the login keys and returns the
// list. A LoginFail answer gives a *LoginFailError.
func (c *LoginConn) ServerList(ctx context.Context) (ServerListResult, error) {
	const want = "ServerList"
	p, data, err := c.request(ctx, encodeRequestServerList(c.keys.LoginOK1, c.keys.LoginOK2), want)
	if err != nil {
		return ServerListResult{}, err
	}
	switch v := p.(type) {
	case ServerListResult:
		return v, nil
	case LoginFail:
		return ServerListResult{}, &LoginFailError{Reason: v.Reason}
	}
	return ServerListResult{}, unexpectedLogin(data, want)
}

// ServerLogin sends RequestServerLogin for serverID with the login keys. A
// PlayOk answer stores the play keys.
func (c *LoginConn) ServerLogin(ctx context.Context, serverID byte) (PlayResult, error) {
	const want = "PlayOk, PlayFail or LoginFail"
	req := encodeRequestServerLogin(c.keys.LoginOK1, c.keys.LoginOK2, serverID)
	p, data, err := c.request(ctx, req, want)
	if err != nil {
		return PlayResult{}, err
	}
	switch v := p.(type) {
	case PlayOk:
		c.keys.PlayOK1, c.keys.PlayOK2 = v.PlayOK1, v.PlayOK2
		return PlayResult{Ok: &v}, nil
	case PlayFail:
		return PlayResult{Fail: &v}, nil
	case LoginFail:
		return PlayResult{LoginFail: &v}, nil
	}
	return PlayResult{}, unexpectedLogin(data, want)
}

// SendRaw sends a packet made of opcode and body, with the usual padding,
// checksum and encryption.
func (c *LoginConn) SendRaw(ctx context.Context, opcode byte, body []byte) error {
	b := make([]byte, 0, 1+len(body))
	b = append(b, opcode)
	b = append(b, body...)
	return c.send(ctx, b)
}

// ReadRaw reads and decrypts one packet and returns its opcode and the
// rest of it. The rest includes the zero padding but not the checksum.
func (c *LoginConn) ReadRaw(ctx context.Context) (opcode byte, body []byte, err error) {
	plain, err := c.readPlain(ctx)
	if err != nil {
		return 0, nil, err
	}
	data := plain[:len(plain)-logincrypt.ChecksumSize]
	return data[0], data[1:], nil
}

// ReadPacket reads one packet and parses it. It returns one of GGAuth,
// LoginOk, LoginFail, AccountKicked, ServerListResult, PlayOk or PlayFail.
func (c *LoginConn) ReadPacket(ctx context.Context) (any, error) {
	plain, err := c.readPlain(ctx)
	if err != nil {
		return nil, err
	}
	return decodeLoginPacket(plain)
}

// ExpectClosed returns nil if the server closes the connection before
// sending anything else. A packet that arrives instead is returned as an
// *UnexpectedPacketError.
func (c *LoginConn) ExpectClosed(ctx context.Context) error {
	return expectClosed(
		func() (byte, []byte, error) { return c.ReadRaw(ctx) },
		func(op byte, body []byte) error {
			return unexpectedLogin(append([]byte{op}, body...), "connection close")
		})
}

// send encrypts body and writes it.
func (c *LoginConn) send(ctx context.Context, body []byte) error {
	enc, err := c.crypt.EncryptClientPacket(body)
	if err != nil {
		return fmt.Errorf("testclient: %w", err)
	}
	return run(ctx, c.conn, func() error { return packets.WritePacket(c.conn, enc) })
}

// readPlain reads one packet and decrypts it in place.
func (c *LoginConn) readPlain(ctx context.Context) ([]byte, error) {
	var plain []byte
	err := run(ctx, c.conn, func() error {
		b, err := readFrame(c.conn)
		if err != nil {
			return err
		}
		if err := c.crypt.DecryptServerPacket(b); err != nil {
			return fmt.Errorf("%w: login packet of %d bytes: %w", ErrMalformed, len(b), err)
		}
		plain = b
		return nil
	})
	return plain, err
}

// request sends req and reads and parses the answer. data is the answer
// without its checksum, for error messages.
func (c *LoginConn) request(ctx context.Context, req []byte, want string) (p any, data []byte, err error) {
	if err := c.send(ctx, req); err != nil {
		return nil, nil, err
	}
	plain, err := c.readPlain(ctx)
	if err != nil {
		return nil, nil, err
	}
	p, err = decodeLoginPacket(plain)
	if err != nil {
		var upe *UnexpectedPacketError
		if errors.As(err, &upe) {
			upe.Want = want
		}
		return nil, nil, err
	}
	return p, plain[:len(plain)-logincrypt.ChecksumSize], nil
}

// LoginAndSelect runs the whole login sequence the way the client does:
// Init, AuthGameGuard, RequestAuthLogin (128-byte block), RequestServerList
// and RequestServerLogin for serverID. It closes the login connection and
// returns all four session keys for the game server.
func LoginAndSelect(ctx context.Context, addr, user, password string, serverID byte) (session.Keys, error) {
	c, err := DialLogin(ctx, addr)
	if err != nil {
		return session.Keys{}, err
	}
	defer func() { _ = c.Close() }()
	if err := c.AuthGameGuard(ctx); err != nil {
		return session.Keys{}, err
	}
	res, err := c.Login(ctx, user, password, false)
	if err != nil {
		return session.Keys{}, err
	}
	switch {
	case res.Ok != nil:
		if _, err := c.ServerList(ctx); err != nil {
			return session.Keys{}, err
		}
	case res.ServerList != nil:
		// The list came instead of LoginOk.
	case res.Fail != nil:
		return session.Keys{}, &LoginFailError{Reason: res.Fail.Reason}
	default:
		return session.Keys{}, fmt.Errorf("testclient: login refused: %v", res)
	}
	play, err := c.ServerLogin(ctx, serverID)
	if err != nil {
		return session.Keys{}, err
	}
	switch {
	case play.Ok != nil:
		return c.Keys(), nil
	case play.LoginFail != nil:
		return session.Keys{}, &LoginFailError{Reason: play.LoginFail.Reason}
	default:
		return session.Keys{}, fmt.Errorf("testclient: server login refused: %v", play)
	}
}
