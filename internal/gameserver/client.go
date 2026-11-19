package gameserver

import (
	"log/slog"
	"net"
	"time"

	"github.com/IPeter/l2go/internal/crypt/gamecrypt"
	"github.com/IPeter/l2go/internal/netutil"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/session"
)

// maxPacketSize bounds a client packet body.
const maxPacketSize = packets.MaxBodySize

// connState is where a client is in the protocol (Mobius ConnectionState).
type connState int

const (
	// stateConnected: KeyPacket was sent; only AuthLogin is accepted.
	stateConnected connState = iota
	// stateAuthenticated: the login server confirmed the session keys; the
	// character packets are accepted.
	stateAuthenticated
)

// client is one game client. Only its own handler goroutine uses it,
// except Close.
type client struct {
	conn        net.Conn
	key         [gamecrypt.KeySize]byte // its first half goes out in KeyPacket
	cipher      *gamecrypt.Cipher
	idleTimeout time.Duration
	logger      *slog.Logger

	state   connState
	account string       // lower case, set by AuthLogin
	keys    session.Keys // set by AuthLogin
}

// newClient returns a client whose cipher uses key. The cipher stays off
// until the first packet sent (KeyPacket), so the first packet in each
// direction is plain.
func newClient(conn net.Conn, key [gamecrypt.KeySize]byte, idleTimeout time.Duration, logger *slog.Logger) *client {
	return &client{
		conn:        conn,
		key:         key,
		cipher:      gamecrypt.NewServerCipher(key),
		idleTimeout: idleTimeout,
		logger:      logger,
	}
}

// Close closes the connection. It is safe to call from any goroutine.
func (c *client) Close() error { return c.conn.Close() }

// receive reads and decrypts one packet and returns its opcode and the rest
// of the body.
func (c *client) receive() (opcode byte, data []byte, err error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(c.idleTimeout)); err != nil {
		return 0, nil, err
	}
	body, err := packets.ReadPacket(c.conn, maxPacketSize)
	if err != nil {
		return 0, nil, err
	}
	c.logger.Debug("packet received", "raw", netutil.Hex(body))
	if c.cipher.Enabled() {
		c.cipher.Decrypt(body)
		c.logger.Debug("packet decrypted", "data", netutil.Hex(body))
	}
	return body[0], body[1:], nil
}

// send encrypts a copy of body and sends it. The first packet (KeyPacket)
// goes out plain and turns the cipher on.
func (c *client) send(body []byte) error {
	c.logger.Debug("packet sent", "data", netutil.Hex(body))
	buf := append([]byte(nil), body...)
	c.cipher.Encrypt(buf)
	if err := c.conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		return err
	}
	return packets.WritePacket(c.conn, buf)
}
