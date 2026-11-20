package link

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/session"
)

// Wire format. Every body starts with the opcode; fields follow in order.
//
//	0x01 Challenge            login→game  version u16, challenge [32]byte
//	0x02 Register             game→login  version u16, server ID u8, HMAC-SHA256(secret, challenge) [32]byte, max players u16
//	0x03 RegisterResult       login→game  reason u8 (0 accepted, see Reason)
//	0x04 Status               game→login  players u16, max players u16
//	0x05 VerifySession        game→login  request ID u32, account (UTF-16LE, zero-terminated), login key 1, login key 2, play key 1, play key 2 (i32 each)
//	0x06 VerifySessionResult  login→game  request ID u32, ok u8 (0 or 1)
//	0x07 Ping                 either way  no fields
//	0x08 Pong                 either way  no fields
//	0x09 ChangeAccessLevel    game→login  account (UTF-16LE, zero-terminated), access level i32
const (
	opChallenge           = 0x01
	opRegister            = 0x02
	opRegisterResult      = 0x03
	opStatus              = 0x04
	opVerifySession       = 0x05
	opVerifySessionResult = 0x06
	opPing                = 0x07
	opPong                = 0x08
	opChangeAccessLevel   = 0x09
)

const (
	protocolVersion = 1
	challengeSize   = 32
	macSize         = sha256.Size

	// maxAccountLen is the longest account name, in characters, that a
	// VerifySession may carry.
	maxAccountLen = 64

	// maxMessageSize bounds every message body. The largest valid one is a
	// VerifySession with a maxAccountLen name of supplementary-plane
	// characters: 1 + 4 + 64*4 + 2 + 16 bytes.
	maxMessageSize = 512
)

// errMalformed is wrapped by every decode error.
var errMalformed = errors.New("link: malformed message")

// message is one decoded link message.
type message interface {
	encode() []byte
}

type challengeMsg struct {
	version   uint16
	challenge [challengeSize]byte
}

type registerMsg struct {
	version    uint16
	serverID   uint8
	mac        [macSize]byte
	maxPlayers uint16
}

type registerResultMsg struct {
	reason Reason
}

type statusMsg struct {
	players    uint16
	maxPlayers uint16
}

type verifySessionMsg struct {
	requestID uint32
	account   string
	keys      session.Keys
}

type verifySessionResultMsg struct {
	requestID uint32
	ok        bool
}

type pingMsg struct{}

type pongMsg struct{}

// changeAccessLevelMsg is L2J's ChangeAccessLevel: a GM banned or unbanned
// an account on the game server.
type changeAccessLevelMsg struct {
	account string
	level   int32
}

func (m changeAccessLevelMsg) encode() []byte {
	w := packets.NewWriter(1 + 2*len(m.account) + 2 + 4)
	w.WriteUint8(opChangeAccessLevel)
	w.WriteString(m.account)
	w.WriteInt32(m.level)
	return w.Bytes()
}

func (m challengeMsg) encode() []byte {
	w := packets.NewWriter(1 + 2 + challengeSize)
	w.WriteUint8(opChallenge)
	w.WriteUint16(m.version)
	w.WriteBytes(m.challenge[:])
	return w.Bytes()
}

func (m registerMsg) encode() []byte {
	w := packets.NewWriter(1 + 2 + 1 + macSize + 2)
	w.WriteUint8(opRegister)
	w.WriteUint16(m.version)
	w.WriteUint8(m.serverID)
	w.WriteBytes(m.mac[:])
	w.WriteUint16(m.maxPlayers)
	return w.Bytes()
}

func (m registerResultMsg) encode() []byte {
	return []byte{opRegisterResult, byte(m.reason)}
}

func (m statusMsg) encode() []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(opStatus)
	w.WriteUint16(m.players)
	w.WriteUint16(m.maxPlayers)
	return w.Bytes()
}

func (m verifySessionMsg) encode() []byte {
	w := packets.NewWriter(1 + 4 + 2*len(m.account) + 2 + 16)
	w.WriteUint8(opVerifySession)
	w.WriteUint32(m.requestID)
	w.WriteString(m.account)
	w.WriteInt32(m.keys.LoginOK1)
	w.WriteInt32(m.keys.LoginOK2)
	w.WriteInt32(m.keys.PlayOK1)
	w.WriteInt32(m.keys.PlayOK2)
	return w.Bytes()
}

func (m verifySessionResultMsg) encode() []byte {
	w := packets.NewWriter(6)
	w.WriteUint8(opVerifySessionResult)
	w.WriteUint32(m.requestID)
	if m.ok {
		w.WriteUint8(1)
	} else {
		w.WriteUint8(0)
	}
	return w.Bytes()
}

func (pingMsg) encode() []byte { return []byte{opPing} }

func (pongMsg) encode() []byte { return []byte{opPong} }

// decode parses one message body. It rejects unknown opcodes, short or
// over-long bodies and out-of-range values, and never panics.
func decode(body []byte) (message, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("%w: empty body", errMalformed)
	}
	r := packets.NewReader(body[1:])
	var m message
	switch body[0] {
	case opChallenge:
		var c challengeMsg
		c.version = r.ReadUint16()
		copy(c.challenge[:], r.ReadBytes(challengeSize))
		m = c
	case opRegister:
		var g registerMsg
		g.version = r.ReadUint16()
		g.serverID = r.ReadUint8()
		copy(g.mac[:], r.ReadBytes(macSize))
		g.maxPlayers = r.ReadUint16()
		m = g
	case opRegisterResult:
		m = registerResultMsg{reason: Reason(r.ReadUint8())}
	case opStatus:
		m = statusMsg{players: r.ReadUint16(), maxPlayers: r.ReadUint16()}
	case opVerifySession:
		v := verifySessionMsg{requestID: r.ReadUint32(), account: r.ReadString()}
		v.keys.LoginOK1 = r.ReadInt32()
		v.keys.LoginOK2 = r.ReadInt32()
		v.keys.PlayOK1 = r.ReadInt32()
		v.keys.PlayOK2 = r.ReadInt32()
		if r.Err() == nil && !validAccount(v.account) {
			return nil, fmt.Errorf("%w: invalid account name", errMalformed)
		}
		m = v
	case opVerifySessionResult:
		v := verifySessionResultMsg{requestID: r.ReadUint32()}
		switch r.ReadUint8() {
		case 0:
		case 1:
			v.ok = true
		default:
			if r.Err() == nil {
				return nil, fmt.Errorf("%w: invalid ok flag", errMalformed)
			}
		}
		m = v
	case opPing:
		m = pingMsg{}
	case opPong:
		m = pongMsg{}
	case opChangeAccessLevel:
		c := changeAccessLevelMsg{account: r.ReadString(), level: r.ReadInt32()}
		if r.Err() == nil && !validAccount(c.account) {
			return nil, fmt.Errorf("%w: invalid account name", errMalformed)
		}
		m = c
	default:
		return nil, fmt.Errorf("%w: unknown opcode 0x%02x", errMalformed, body[0])
	}
	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("%w: opcode 0x%02x: %w", errMalformed, body[0], err)
	}
	if r.Len() != 0 {
		return nil, fmt.Errorf("%w: opcode 0x%02x: %d trailing bytes", errMalformed, body[0], r.Len())
	}
	return m, nil
}

// validAccount reports whether a VerifySession may carry account.
func validAccount(account string) bool {
	return account != "" &&
		utf8.RuneCountInString(account) <= maxAccountLen &&
		!strings.ContainsRune(account, 0)
}

// computeMAC returns HMAC-SHA256(secret, challenge).
func computeMAC(secret []byte, challenge [challengeSize]byte) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write(challenge[:])
	return h.Sum(nil)
}

// readMsg reads and decodes one message.
func readMsg(r io.Reader) (message, error) {
	body, err := packets.ReadPacket(r, maxMessageSize)
	if err != nil {
		return nil, err
	}
	return decode(body)
}

// writeMsg writes one message with a write deadline of timeout from now.
func writeMsg(conn net.Conn, timeout time.Duration, m message) error {
	if err := conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}
	return packets.WritePacket(conn, m.encode())
}
