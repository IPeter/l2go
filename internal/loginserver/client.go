package loginserver

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"

	"github.com/IPeter/l2go/internal/crypt/blowfish"
	"github.com/IPeter/l2go/internal/crypt/checksum"
	"github.com/IPeter/l2go/internal/loginserver/clientpackets"
	"github.com/IPeter/l2go/internal/netutil"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/storage"
)

// staticKey is the Blowfish key of the C1 login protocol, including its
// trailing zero byte.
var staticKey = []byte("[;'.]94-31==-%&@!^+]\x00")

// maxPacketSize bounds a client packet body.
const maxPacketSize = 1024

var errChecksum = errors.New("invalid packet checksum")

// sessionIDSize is the length of the random session ID; the first
// clientpackets.SessionIDSize bytes go to the client.
const sessionIDSize = 16

// client is one game client connected to the login server. Only its own
// handler goroutine uses it, except Close.
type client struct {
	conn        net.Conn
	cipher      *blowfish.Cipher
	sessionID   [sessionIDSize]byte
	account     *storage.Account // set after a successful login
	idleTimeout time.Duration
	logger      *slog.Logger
}

// newSessionID reads a random session ID from rnd.
func newSessionID(rnd io.Reader) ([sessionIDSize]byte, error) {
	var id [sessionIDSize]byte
	if _, err := io.ReadFull(rnd, id[:]); err != nil {
		return id, fmt.Errorf("generate session ID: %w", err)
	}
	return id, nil
}

func newClient(conn net.Conn, cipher *blowfish.Cipher, rnd io.Reader, idleTimeout time.Duration, logger *slog.Logger) (*client, error) {
	id, err := newSessionID(rnd)
	if err != nil {
		return nil, err
	}
	return &client{conn: conn, cipher: cipher, sessionID: id, idleTimeout: idleTimeout, logger: logger}, nil
}

// Close closes the connection. It is safe to call from any goroutine.
func (c *client) Close() error { return c.conn.Close() }

// sessionMatches reports whether the client is logged in and echoed the
// right session ID.
func (c *client) sessionMatches(sid []byte) bool {
	return c.account != nil && subtle.ConstantTimeCompare(c.sessionID[:clientpackets.SessionIDSize], sid) == 1
}

// accessLevel returns the logged-in account's level, or player level.
func (c *client) accessLevel() storage.AccessLevel {
	if c.account == nil {
		return storage.AccessLevelPlayer
	}
	return c.account.AccessLevel
}

// receive reads one packet, decrypts it and checks its checksum. It returns
// the opcode and the rest of the body (including padding and checksum).
func (c *client) receive() (opcode byte, data []byte, err error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(c.idleTimeout)); err != nil {
		return 0, nil, err
	}
	body, err := packets.ReadPacket(c.conn, maxPacketSize)
	if err != nil {
		return 0, nil, err
	}
	c.logger.Debug("packet received", "raw", netutil.Hex(body))
	if err := c.cipher.Decrypt(body, body); err != nil {
		return 0, nil, err
	}
	if !checksum.Verify(body) {
		return 0, nil, errChecksum
	}
	c.logger.Debug("packet decrypted", "data", netutil.Hex(body))
	return body[0], body[1:], nil
}

// sendPlain sends body as is.
func (c *client) sendPlain(body []byte) error {
	c.logger.Debug("packet sent", "data", netutil.Hex(body))
	if err := c.conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		return err
	}
	return packets.WritePacket(c.conn, body)
}

// send pads body, adds the checksum and encrypts it before sending.
func (c *client) send(body []byte) error {
	c.logger.Debug("packet sent", "data", netutil.Hex(body))
	buf := sealPacket(body)
	if err := c.cipher.Encrypt(buf, buf); err != nil {
		return err
	}
	if err := c.conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		return err
	}
	return packets.WritePacket(c.conn, buf)
}

// sealPacket returns a copy of body with room for the checksum, zero
// padding to a whole number of Blowfish blocks, and the checksum set.
func sealPacket(body []byte) []byte {
	n := len(body) + checksum.Size
	n += (blowfish.BlockSize - n%blowfish.BlockSize) % blowfish.BlockSize
	buf := make([]byte, n)
	copy(buf, body)
	_ = checksum.Put(buf) // n is a multiple of 8 and at least 8
	return buf
}
