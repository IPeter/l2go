package gameserver

import (
	"log/slog"
	"net"
	"time"

	"github.com/IPeter/l2go/internal/crypt/xor"
	"github.com/IPeter/l2go/internal/netutil"
	"github.com/IPeter/l2go/internal/packets"
)

// maxPacketSize bounds a client packet body.
const maxPacketSize = packets.MaxBodySize

// client is one game client. Only its own handler goroutine uses it,
// except Close.
type client struct {
	conn        net.Conn
	cipher      *xor.Cipher
	idleTimeout time.Duration
	logger      *slog.Logger

	account string // from AuthLogin; empty before it
}

func newClient(conn net.Conn, idleTimeout time.Duration, logger *slog.Logger) *client {
	return &client{conn: conn, cipher: xor.NewCipher(xor.StaticKey), idleTimeout: idleTimeout, logger: logger}
}

// Close closes the connection. It is safe to call from any goroutine.
func (c *client) Close() error { return c.conn.Close() }

func (c *client) read() ([]byte, error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(c.idleTimeout)); err != nil {
		return nil, err
	}
	body, err := packets.ReadPacket(c.conn, maxPacketSize)
	if err != nil {
		return nil, err
	}
	c.logger.Debug("packet received", "raw", netutil.Hex(body))
	return body, nil
}

// receivePlain reads one unencrypted packet and returns its opcode and the
// rest of the body.
func (c *client) receivePlain() (opcode byte, data []byte, err error) {
	body, err := c.read()
	if err != nil {
		return 0, nil, err
	}
	return body[0], body[1:], nil
}

// receive reads and decrypts one packet.
func (c *client) receive() (opcode byte, data []byte, err error) {
	body, err := c.read()
	if err != nil {
		return 0, nil, err
	}
	c.cipher.Decrypt(body)
	c.logger.Debug("packet decrypted", "data", netutil.Hex(body))
	return body[0], body[1:], nil
}

// sendPlain sends body unencrypted.
func (c *client) sendPlain(body []byte) error {
	c.logger.Debug("packet sent", "data", netutil.Hex(body))
	if err := c.conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		return err
	}
	return packets.WritePacket(c.conn, body)
}

// send encrypts a copy of body and sends it.
func (c *client) send(body []byte) error {
	c.logger.Debug("packet sent", "data", netutil.Hex(body))
	buf := append([]byte(nil), body...)
	c.cipher.Encrypt(buf)
	if err := c.conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		return err
	}
	return packets.WritePacket(c.conn, buf)
}
