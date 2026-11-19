// Package testclient is a fake Interlude (protocol 746) client for
// end-to-end tests of the login and game servers.
//
// It is written from the L2J Mobius Interlude sources (the writeImpl and
// readImpl methods of each packet), not from the L2Go servers, so a test
// that passes against it shows that the servers speak the protocol the way
// Mobius does. It shares only the crypto and framing packages with them.
//
// Login side: DialLogin reads Init and switches to the session key, then
// AuthGameGuard, Login, ServerList and ServerLogin follow the client's
// order. LoginAndSelect runs the whole sequence and returns the session
// keys. Game side: DialGame sends ProtocolVersion and reads KeyPacket, then
// AuthLogin, NewCharacter and CreateCharacter send their requests and parse
// the replies. Ping checks the version -2 ping.
//
// Every call is synchronous and takes a context. Its deadline becomes the
// connection deadline, and cancelling it aborts a blocked read or write.
// After an aborted call the connection may be out of step and should be
// closed.
//
// Parsing is strict. Each server packet must have exactly the fields Mobius
// writes: game packets may not carry extra bytes, and login packets must
// have Mobius's padding size with only zeros after the body. A packet the
// call does not expect gives an *UnexpectedPacketError with its opcode, and
// a malformed one an error that wraps ErrMalformed. SendRaw, ReadRaw and
// ReadPacket are there for negative tests.
package testclient

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"syscall"
	"time"

	"github.com/IPeter/l2go/internal/packets"
)

var (
	// ErrServerClosed is returned when the server closes the connection
	// while the client waits for a packet.
	ErrServerClosed = errors.New("testclient: server closed the connection")

	// ErrMalformed is wrapped by every error about a server packet whose
	// layout does not match Mobius.
	ErrMalformed = errors.New("testclient: malformed packet")

	// ErrWrongProtocol is returned by DialGame when the game server answers
	// ProtocolVersion with a KeyPacket whose result is 0.
	ErrWrongProtocol = errors.New("testclient: game server rejected the protocol version")
)

// UnexpectedPacketError reports a server packet the call did not expect,
// or one with an unknown opcode.
type UnexpectedPacketError struct {
	// Server is "login" or "game".
	Server string
	// Opcode is the packet's first byte.
	Opcode byte
	// Name is the packet's Mobius name, or "" for an unknown opcode.
	Name string
	// Want describes what the call was waiting for.
	Want string
	// Body is the packet after the opcode. For login packets it includes
	// the zero padding but not the checksum.
	Body []byte
}

func (e *UnexpectedPacketError) Error() string {
	name := e.Name
	if name == "" {
		name = "unknown packet"
	}
	msg := fmt.Sprintf("testclient: %s server sent %s (opcode 0x%02x, %d more bytes: %s)",
		e.Server, name, e.Opcode, len(e.Body), hexPrefix(e.Body))
	if e.Want != "" {
		msg += ", want " + e.Want
	}
	return msg
}

// hexPrefix returns the first bytes of b in hex, enough for an error
// message.
func hexPrefix(b []byte) string {
	const limit = 48
	if len(b) <= limit {
		return hex.EncodeToString(b)
	}
	return hex.EncodeToString(b[:limit]) + "..."
}

// malformed returns an error wrapping ErrMalformed for the named packet.
func malformed(name string, format string, args ...any) error {
	return fmt.Errorf("%w: %s: %s", ErrMalformed, name, fmt.Sprintf(format, args...))
}

// readerError wraps a packets.Reader error for the named packet.
func readerError(name string, err error) error {
	return fmt.Errorf("%w: %s: %w", ErrMalformed, name, err)
}

// boolValue converts a 0/1 field to a bool. Mobius writes no other values.
func boolValue[T int32 | uint8](name, field string, v T) (bool, error) {
	switch v {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, malformed(name, "%s is %d, want 0 or 1", field, v)
	}
}

// pastDeadline aborts blocked I/O when set as a connection deadline.
var pastDeadline = time.Unix(1, 0)

// run calls fn with the connection deadline taken from ctx. Cancelling ctx
// while fn runs aborts its I/O. A failure caused by ctx wraps ctx.Err().
func run(ctx context.Context, conn net.Conn, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("testclient: %w", err)
	}
	deadline, hasDeadline := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		return fmt.Errorf("testclient: set deadline: %w", err)
	}
	fired := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		_ = conn.SetDeadline(pastDeadline)
		close(fired)
	})
	err := fn()
	if !stop() {
		<-fired
	}
	_ = conn.SetDeadline(time.Time{})
	if err == nil {
		return nil
	}
	ctxErr := ctx.Err()
	if ctxErr == nil && hasDeadline && errors.Is(err, os.ErrDeadlineExceeded) && !time.Now().Before(deadline) {
		// The connection deadline can fire just before the context's own
		// timer does.
		ctxErr = context.DeadlineExceeded
	}
	if ctxErr != nil && !errors.Is(err, ctxErr) {
		return fmt.Errorf("%w: %w", ctxErr, err)
	}
	return err
}

// readFrame reads one length-prefixed packet body. A connection closed by
// the server gives an error wrapping ErrServerClosed.
func readFrame(conn net.Conn) ([]byte, error) {
	b, err := packets.ReadPacket(conn, packets.MaxBodySize)
	if err != nil {
		if closedByPeer(err) {
			return nil, fmt.Errorf("%w: %w", ErrServerClosed, err)
		}
		return nil, fmt.Errorf("testclient: %w", err)
	}
	return b, nil
}

// closedByPeer reports whether err means the other end closed the
// connection.
func closedByPeer(err error) bool {
	return errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNRESET)
}

// dial opens a TCP connection to addr.
func dial(ctx context.Context, what, addr string) (net.Conn, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("testclient: dial %s server %s: %w", what, addr, err)
	}
	return conn, nil
}

// expectClosed reads once and returns nil if the server has closed the
// connection. A packet that arrives instead is reported by unexpected.
func expectClosed(read func() (byte, []byte, error), unexpected func(op byte, body []byte) error) error {
	op, body, err := read()
	if err == nil {
		return unexpected(op, body)
	}
	if errors.Is(err, ErrServerClosed) {
		return nil
	}
	return fmt.Errorf("testclient: server did not close the connection: %w", err)
}
