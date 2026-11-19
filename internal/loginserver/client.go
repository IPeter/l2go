package loginserver

import (
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"

	"github.com/IPeter/l2go/internal/crypt/logincrypt"
	"github.com/IPeter/l2go/internal/crypt/rsakey"
	"github.com/IPeter/l2go/internal/netutil"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/session"
	"github.com/IPeter/l2go/internal/storage"
)

// maxPacketSize bounds a client packet body. The largest Interlude login
// packet is RequestAuthLogin with two RSA blocks (272 bytes encrypted).
const maxPacketSize = 1024

// connState is where a connection is in the login sequence. Each client
// packet is accepted in one state only (Mobius ConnectionState).
type connState int

const (
	// stateConnected: Init was sent; AuthGameGuard is expected.
	stateConnected connState = iota
	// stateAuthedGG: the GameGuard handshake is done; RequestAuthLogin is
	// expected.
	stateAuthedGG
	// stateAuthedLogin: the account is logged in; RequestServerList and
	// RequestServerLogin are accepted.
	stateAuthedLogin
)

func (s connState) String() string {
	switch s {
	case stateConnected:
		return "connected"
	case stateAuthedGG:
		return "GameGuard ok"
	case stateAuthedLogin:
		return "logged in"
	default:
		return fmt.Sprintf("state %d", int(s))
	}
}

// client is one game client connected to the login server. Only its own
// handler goroutine uses it, except Close.
type client struct {
	conn        net.Conn
	crypt       *logincrypt.Session
	rsa         *rsakey.KeyPair
	sessionID   int32
	blowfishKey [logincrypt.KeySize]byte
	idleTimeout time.Duration
	logger      *slog.Logger

	state   connState
	account storage.Account // set when state is stateAuthedLogin
	keys    session.Keys    // issued at login
}

// newClient sets up a connection: a random session ID and Blowfish session
// key read from rnd, and the RSA key pair kp.
func newClient(conn net.Conn, kp *rsakey.KeyPair, rnd io.Reader, idleTimeout time.Duration, logger *slog.Logger) (*client, error) {
	var b [4 + logincrypt.KeySize]byte
	if _, err := io.ReadFull(rnd, b[:]); err != nil {
		return nil, fmt.Errorf("generate session ID and key: %w", err)
	}
	c := &client{
		conn:        conn,
		rsa:         kp,
		sessionID:   int32(binary.LittleEndian.Uint32(b[:4])), //nolint:gosec // random bits
		idleTimeout: idleTimeout,
		logger:      logger,
	}
	copy(c.blowfishKey[:], b[4:])
	cs, err := logincrypt.NewSession(c.blowfishKey[:], rnd)
	if err != nil {
		return nil, err
	}
	c.crypt = cs
	return c, nil
}

// Close closes the connection. It is safe to call from any goroutine.
func (c *client) Close() error { return c.conn.Close() }

// accessLevel returns the logged-in account's level, or player level.
func (c *client) accessLevel() storage.AccessLevel {
	if c.state != stateAuthedLogin {
		return storage.AccessLevelPlayer
	}
	return c.account.AccessLevel
}

// receive reads one packet, decrypts it and checks its checksum. It returns
// the opcode and the rest of the body, including padding and checksum.
func (c *client) receive() (opcode byte, data []byte, err error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(c.idleTimeout)); err != nil {
		return 0, nil, err
	}
	body, err := packets.ReadPacket(c.conn, maxPacketSize)
	if err != nil {
		return 0, nil, err
	}
	c.logger.Debug("packet received", "raw", netutil.Hex(body))
	if err := c.crypt.DecryptClientPacket(body); err != nil {
		return 0, nil, err
	}
	c.logger.Debug("packet decrypted", "data", netutil.Hex(body))
	return body[0], body[1:], nil
}

// send encrypts body and sends it. The first packet of a connection (Init)
// goes out with the static key; every later one with the session key and a
// checksum.
func (c *client) send(body []byte) error {
	c.logger.Debug("packet sent", "data", netutil.Hex(body))
	buf, err := c.crypt.EncryptServerPacket(body)
	if err != nil {
		return err
	}
	if err := c.conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		return err
	}
	return packets.WritePacket(c.conn, buf)
}
