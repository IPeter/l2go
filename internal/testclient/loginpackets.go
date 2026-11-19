package testclient

import (
	"fmt"
	"net/netip"

	"github.com/IPeter/l2go/internal/crypt/logincrypt"
	"github.com/IPeter/l2go/internal/crypt/rsakey"
	"github.com/IPeter/l2go/internal/packets"
)

// Login server packet opcodes, from Mobius LoginServerPackets.
const (
	OpInit          byte = 0x00
	OpLoginFail     byte = 0x01
	OpAccountKicked byte = 0x02
	OpLoginOk       byte = 0x03
	OpServerList    byte = 0x04
	OpPlayFail      byte = 0x06
	OpPlayOk        byte = 0x07
	OpGGAuth        byte = 0x0b
)

// Login client packet opcodes, from Mobius LoginClientPackets.
const (
	OpRequestAuthLogin   byte = 0x00
	OpRequestServerLogin byte = 0x02
	OpRequestServerList  byte = 0x05
	OpAuthGameGuard      byte = 0x07
)

// LoginProtocol is the protocol revision in Init.
const LoginProtocol int32 = 0xc621

// InitBodySize is the size of the Init body Mobius writes: opcode, session
// ID, protocol, 128-byte scrambled modulus, four GameGuard words, 16-byte
// Blowfish key and a zero byte.
const InitBodySize = 1 + 4 + 4 + rsakey.BlockSize + 4*4 + logincrypt.KeySize + 1

// MobiusGameGuard holds the four GameGuard words Mobius writes in Init.
var MobiusGameGuard = [4]uint32{0x29dd954e, 0x77c39cfc, 0x97adb620, 0x07bde0f7}

// loginPacketNames maps login server opcodes to Mobius packet names.
var loginPacketNames = map[byte]string{
	OpInit:          "Init",
	OpLoginFail:     "LoginFail",
	OpAccountKicked: "AccountKicked",
	OpLoginOk:       "LoginOk",
	OpServerList:    "ServerList",
	OpPlayFail:      "PlayFail",
	OpPlayOk:        "PlayOk",
	OpGGAuth:        "GGAuth",
}

// LoginFailReason is the reason byte of LoginFail (Mobius LoginFailReason).
type LoginFailReason byte

// LoginFail reasons. Mobius has more (certification and age checks), which
// L2Go does not use.
const (
	LoginFailNoMessage             LoginFailReason = 0x00
	LoginFailSystemErrorLoginLater LoginFailReason = 0x01
	LoginFailUserOrPassWrong       LoginFailReason = 0x02
	LoginFailAccessFailedTryLater  LoginFailReason = 0x04
	LoginFailAccountInfoIncorrect  LoginFailReason = 0x05
	LoginFailNotAuthed             LoginFailReason = 0x06
	LoginFailAccountInUse          LoginFailReason = 0x07
	LoginFailUnder18               LoginFailReason = 0x0c
	LoginFailServerOverloaded      LoginFailReason = 0x0f
	LoginFailServerMaintenance     LoginFailReason = 0x10
	LoginFailTempPassExpired       LoginFailReason = 0x11
	LoginFailGameTimeExpired       LoginFailReason = 0x12
	LoginFailNoTimeLeft            LoginFailReason = 0x13
	LoginFailSystemError           LoginFailReason = 0x14
	LoginFailAccessFailed          LoginFailReason = 0x15
	LoginFailRestrictedIP          LoginFailReason = 0x16
	LoginFailDualBox               LoginFailReason = 0x23
	LoginFailInactive              LoginFailReason = 0x24
)

var loginFailNames = map[LoginFailReason]string{
	LoginFailNoMessage:             "no message",
	LoginFailSystemErrorLoginLater: "system error, log in later",
	LoginFailUserOrPassWrong:       "user or password wrong",
	LoginFailAccessFailedTryLater:  "access failed, try later",
	LoginFailAccountInfoIncorrect:  "account info incorrect",
	LoginFailNotAuthed:             "not authed",
	LoginFailAccountInUse:          "account in use",
	LoginFailUnder18:               "under 18",
	LoginFailServerOverloaded:      "server overloaded",
	LoginFailServerMaintenance:     "server maintenance",
	LoginFailTempPassExpired:       "temporary password expired",
	LoginFailGameTimeExpired:       "game time expired",
	LoginFailNoTimeLeft:            "no time left",
	LoginFailSystemError:           "system error",
	LoginFailAccessFailed:          "access failed",
	LoginFailRestrictedIP:          "restricted IP",
	LoginFailDualBox:               "dual box",
	LoginFailInactive:              "inactive",
}

func (r LoginFailReason) String() string {
	if s, ok := loginFailNames[r]; ok {
		return fmt.Sprintf("0x%02x (%s)", byte(r), s)
	}
	return fmt.Sprintf("0x%02x", byte(r))
}

// PlayFailReason is the reason byte of PlayFail (Mobius PlayFailReason). It
// uses the same codes as LoginFailReason.
type PlayFailReason byte

// PlayFail reasons used by Mobius and L2Go.
const (
	PlayFailNoMessage             PlayFailReason = 0x00
	PlayFailSystemErrorLoginLater PlayFailReason = 0x01
	PlayFailUserOrPassWrong       PlayFailReason = 0x02
	PlayFailAccessFailedTryLater  PlayFailReason = 0x04
	PlayFailAccountInfoIncorrect  PlayFailReason = 0x05
	PlayFailAccountInUse          PlayFailReason = 0x07
	PlayFailUnder18               PlayFailReason = 0x0c
	PlayFailServerOverloaded      PlayFailReason = 0x0f
	PlayFailServerMaintenance     PlayFailReason = 0x10
	PlayFailSystemError           PlayFailReason = 0x14
	PlayFailAccessFailed          PlayFailReason = 0x15
	PlayFailRestrictedIP          PlayFailReason = 0x16
)

func (r PlayFailReason) String() string {
	if s, ok := loginFailNames[LoginFailReason(r)]; ok {
		return fmt.Sprintf("0x%02x (%s)", byte(r), s)
	}
	return fmt.Sprintf("0x%02x", byte(r))
}

// AccountKickedReason is the reason of AccountKicked (Mobius
// AccountKickedReason). It is a 4-byte field.
type AccountKickedReason int32

// AccountKicked reasons.
const (
	AccountKickedDataStealer       AccountKickedReason = 0x01
	AccountKickedGenericViolation  AccountKickedReason = 0x08
	AccountKicked7DaysSuspended    AccountKickedReason = 0x10
	AccountKickedPermanentlyBanned AccountKickedReason = 0x20
)

var accountKickedNames = map[AccountKickedReason]string{
	AccountKickedDataStealer:       "data stealer",
	AccountKickedGenericViolation:  "generic violation",
	AccountKicked7DaysSuspended:    "7 days suspended",
	AccountKickedPermanentlyBanned: "permanently banned",
}

func (r AccountKickedReason) String() string {
	if s, ok := accountKickedNames[r]; ok {
		return fmt.Sprintf("0x%02x (%s)", int32(r), s)
	}
	return fmt.Sprintf("0x%02x", int32(r))
}

// Init is the first login server packet (Mobius Init).
type Init struct {
	SessionID        int32
	Protocol         int32
	ScrambledModulus [rsakey.BlockSize]byte
	// GameGuard holds the four GameGuard words; Mobius sends
	// MobiusGameGuard.
	GameGuard   [4]uint32
	BlowfishKey [logincrypt.KeySize]byte
}

// GGAuth answers AuthGameGuard (Mobius GGAuth).
type GGAuth struct {
	// Response is the session ID from Init.
	Response int32
	// Reserved holds four words that Mobius writes as zero.
	Reserved [4]int32
}

// LoginOk accepts the credentials (Mobius LoginOk).
type LoginOk struct {
	LoginOK1, LoginOK2 int32
	// Fields holds the six words after the keys. Mobius writes 0, 0,
	// 0x3ea, 0, 0, 0.
	Fields [6]int32
	// Tail holds the last 16 bytes. Mobius writes zeros.
	Tail [16]byte
}

// LoginFail refuses a login request (Mobius LoginFail).
type LoginFail struct {
	Reason LoginFailReason
}

// AccountKicked refuses a banned account (Mobius AccountKicked).
type AccountKicked struct {
	Reason AccountKickedReason
}

// PlayOk accepts RequestServerLogin and carries the play keys (Mobius
// PlayOk).
type PlayOk struct {
	PlayOK1, PlayOK2 int32
}

// PlayFail refuses RequestServerLogin (Mobius PlayFail).
type PlayFail struct {
	Reason PlayFailReason
}

// ServerListResult is the ServerList packet (Mobius ServerList).
type ServerListResult struct {
	// LastServer is the ID of the server the account used last.
	LastServer byte
	Servers    []ServerInfo
	// Unknown is the 2-byte word after the server entries. Mobius writes 0.
	Unknown uint16
	// Characters is the trailing "characters per server" section. Mobius
	// writes a zero count when it has no data.
	Characters []ServerCharacters
}

// ServerInfo is one entry of ServerList.
type ServerInfo struct {
	ID   byte
	IP   netip.Addr
	Port int32
	// AgeLimit is 0, 15 or 18.
	AgeLimit       byte
	PvP            bool
	CurrentPlayers uint16
	MaxPlayers     uint16
	// Up is false when the server is down.
	Up bool
	// ServerType is a bit set: 1 normal, 2 relax, 4 public test, 8 no
	// label, 16 character creation restricted, 32 event, 64 free.
	ServerType int32
	Brackets   bool
}

// ServerCharacters is one entry of the trailing ServerList section.
type ServerCharacters struct {
	ServerID   byte
	Characters byte
	// DeleteTimers holds, for each character waiting for deletion, the
	// seconds until it is deleted.
	DeleteTimers []int32
}

// LoginFailError is returned when the login server answers with LoginFail
// where the call expected another packet.
type LoginFailError struct {
	Reason LoginFailReason
}

func (e *LoginFailError) Error() string {
	return fmt.Sprintf("testclient: login server sent LoginFail, reason %v", e.Reason)
}

// encodeAuthGameGuard builds AuthGameGuard: the session ID and four
// GameGuard words (Mobius reads them and ignores them; zeros here).
func encodeAuthGameGuard(sessionID int32) []byte {
	w := packets.NewWriter(21)
	w.WriteUint8(OpAuthGameGuard)
	w.WriteInt32(sessionID)
	for range 4 {
		w.WriteInt32(0)
	}
	return w.Bytes()
}

// encodeRequestAuthLogin builds RequestAuthLogin: the opcode followed by
// one or two RSA blocks.
func encodeRequestAuthLogin(rsaBlocks []byte) []byte {
	w := packets.NewWriter(1 + len(rsaBlocks))
	w.WriteUint8(OpRequestAuthLogin)
	w.WriteBytes(rsaBlocks)
	return w.Bytes()
}

// encodeRequestServerList builds RequestServerList with the login keys.
func encodeRequestServerList(loginOK1, loginOK2 int32) []byte {
	w := packets.NewWriter(9)
	w.WriteUint8(OpRequestServerList)
	w.WriteInt32(loginOK1)
	w.WriteInt32(loginOK2)
	return w.Bytes()
}

// encodeRequestServerLogin builds RequestServerLogin: the login keys and
// the chosen server ID.
func encodeRequestServerLogin(loginOK1, loginOK2 int32, serverID byte) []byte {
	w := packets.NewWriter(10)
	w.WriteUint8(OpRequestServerLogin)
	w.WriteInt32(loginOK1)
	w.WriteInt32(loginOK2)
	w.WriteUint8(serverID)
	return w.Bytes()
}

// parseInit parses the decrypted Init packet: the 170-byte body, zero
// padding, the XOR key word at len-8 and a zero word at len-4, in the
// size Mobius LoginEncryption gives it.
func parseInit(plain []byte) (Init, error) {
	var in Init
	if len(plain) == 0 {
		return in, malformed("Init", "empty packet")
	}
	if plain[0] != OpInit {
		return in, unexpectedLogin(plain[:max(len(plain)-8, 1)], "Init")
	}
	if want := logincrypt.StaticPacketSize(InitBodySize); len(plain) != want {
		return in, malformed("Init", "packet is %d bytes after decryption, Mobius sends %d (%d-byte body)",
			len(plain), want, InitBodySize)
	}
	r := packets.NewReader(plain[1:InitBodySize])
	in.SessionID = r.ReadInt32()
	in.Protocol = r.ReadInt32()
	copy(in.ScrambledModulus[:], r.ReadBytes(rsakey.BlockSize))
	for i := range in.GameGuard {
		in.GameGuard[i] = r.ReadUint32()
	}
	copy(in.BlowfishKey[:], r.ReadBytes(logincrypt.KeySize))
	term := r.ReadUint8()
	if err := r.Err(); err != nil {
		return in, readerError("Init", err)
	}
	if term != 0 {
		return in, malformed("Init", "last body byte is 0x%02x, want 0", term)
	}
	if in.Protocol != LoginProtocol {
		return in, malformed("Init", "protocol revision 0x%x, want 0x%x", in.Protocol, LoginProtocol)
	}
	keyAt := len(plain) - 8
	for i, b := range plain[InitBodySize:keyAt] {
		if b != 0 {
			return in, malformed("Init", "padding byte at offset %d is 0x%02x, want 0", InitBodySize+i, b)
		}
	}
	for i, b := range plain[keyAt+4:] {
		if b != 0 {
			return in, malformed("Init", "byte at offset %d after the XOR key is 0x%02x, want 0", keyAt+4+i, b)
		}
	}
	return in, nil
}

// unexpectedLogin builds the error for a login packet the call did not
// want. data is the decrypted packet without its checksum.
func unexpectedLogin(data []byte, want string) *UnexpectedPacketError {
	e := &UnexpectedPacketError{Server: "login", Want: want}
	if len(data) > 0 {
		e.Opcode = data[0]
		e.Name = loginPacketNames[data[0]]
		e.Body = append([]byte(nil), data[1:]...)
	}
	return e
}

// decodeLoginPacket parses a decrypted login packet that followed Init.
// plain is the whole decrypted packet: body, zero padding and checksum.
//
// It returns one of GGAuth, LoginOk, LoginFail, AccountKicked,
// ServerListResult, PlayOk or PlayFail. An unknown opcode, or Init,
// gives an *UnexpectedPacketError.
func decodeLoginPacket(plain []byte) (any, error) {
	if len(plain) <= logincrypt.ChecksumSize {
		return nil, malformed("login packet", "%d bytes after decryption", len(plain))
	}
	data := plain[:len(plain)-logincrypt.ChecksumSize]
	op := data[0]
	name, known := loginPacketNames[op]
	if !known || op == OpInit {
		return nil, unexpectedLogin(data, "")
	}
	r := packets.NewReader(data[1:])
	var p any
	switch op {
	case OpLoginFail:
		p = LoginFail{Reason: LoginFailReason(r.ReadUint8())}
	case OpAccountKicked:
		p = AccountKicked{Reason: AccountKickedReason(r.ReadInt32())}
	case OpLoginOk:
		var v LoginOk
		v.LoginOK1 = r.ReadInt32()
		v.LoginOK2 = r.ReadInt32()
		for i := range v.Fields {
			v.Fields[i] = r.ReadInt32()
		}
		copy(v.Tail[:], r.ReadBytes(len(v.Tail)))
		p = v
	case OpServerList:
		v, err := readServerList(r)
		if err != nil {
			return nil, err
		}
		p = v
	case OpPlayFail:
		p = PlayFail{Reason: PlayFailReason(r.ReadUint8())}
	case OpPlayOk:
		p = PlayOk{PlayOK1: r.ReadInt32(), PlayOK2: r.ReadInt32()}
	case OpGGAuth:
		var v GGAuth
		v.Response = r.ReadInt32()
		for i := range v.Reserved {
			v.Reserved[i] = r.ReadInt32()
		}
		p = v
	}
	if err := r.Err(); err != nil {
		return nil, readerError(name, err)
	}
	if err := checkLoginPadding(name, plain, len(data)-r.Len()); err != nil {
		return nil, err
	}
	return p, nil
}

// checkLoginPadding checks that the n-byte body of a decrypted login packet
// is followed only by zeros up to the checksum, and that the packet has
// the size Mobius LoginEncryption gives an n-byte body. A missing or extra
// field shows up here, because the zero padding itself reads like fields.
func checkLoginPadding(name string, plain []byte, n int) error {
	if want := logincrypt.SessionPacketSize(n); len(plain) != want {
		return malformed(name, "%d-byte body arrived as %d bytes, Mobius sends %d (missing or extra fields?)",
			n, len(plain), want)
	}
	end := len(plain) - logincrypt.ChecksumSize
	for i, b := range plain[n:end] {
		if b != 0 {
			return malformed(name, "byte at offset %d after the %d-byte body is 0x%02x, want zero padding", n+i, n, b)
		}
	}
	return nil
}

// readServerList parses ServerList after its opcode, as Mobius
// ServerList.writeImpl writes it:
//
//	c count, c last server
//	count × [c id, 4×c IP, d port, c age limit, c PvP, h players,
//	         h max players, c up, d server type, c brackets]
//	h 0
//	c n, n × [c server ID, c characters, c k, k × d seconds to deletion]
func readServerList(r *packets.Reader) (ServerListResult, error) {
	const name = "ServerList"
	var s ServerListResult
	count := int(r.ReadUint8())
	s.LastServer = r.ReadUint8()
	for i := range count {
		var si ServerInfo
		si.ID = r.ReadUint8()
		ip := r.ReadBytes(4)
		si.Port = r.ReadInt32()
		si.AgeLimit = r.ReadUint8()
		pvp := r.ReadUint8()
		si.CurrentPlayers = r.ReadUint16()
		si.MaxPlayers = r.ReadUint16()
		up := r.ReadUint8()
		si.ServerType = r.ReadInt32()
		brackets := r.ReadUint8()
		if err := r.Err(); err != nil {
			return s, readerError(fmt.Sprintf("%s server %d", name, i), err)
		}
		si.IP = netip.AddrFrom4([4]byte(ip))
		var err error
		if si.PvP, err = boolValue(name, "PvP flag", pvp); err != nil {
			return s, err
		}
		if si.Up, err = boolValue(name, "status", up); err != nil {
			return s, err
		}
		if si.Brackets, err = boolValue(name, "brackets flag", brackets); err != nil {
			return s, err
		}
		s.Servers = append(s.Servers, si)
	}
	s.Unknown = r.ReadUint16()
	n := int(r.ReadUint8())
	for i := range n {
		var sc ServerCharacters
		sc.ServerID = r.ReadUint8()
		sc.Characters = r.ReadUint8()
		k := int(r.ReadUint8())
		for range k {
			sc.DeleteTimers = append(sc.DeleteTimers, r.ReadInt32())
		}
		if err := r.Err(); err != nil {
			return s, readerError(fmt.Sprintf("%s characters entry %d", name, i), err)
		}
		s.Characters = append(s.Characters, sc)
	}
	return s, nil
}
