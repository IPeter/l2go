// Package serverpackets builds packets sent by the login server to the
// client (Interlude protocol). Each builder returns the plain body, opcode
// first; the caller pads, adds the checksum and encrypts it (see
// internal/crypt/logincrypt).
//
// The layouts follow the writeImpl methods of the L2J Mobius Interlude
// login server packets.
package serverpackets

// Opcodes of the server packets (Mobius LoginServerPackets).
const (
	OpInit          = 0x00
	OpLoginFail     = 0x01
	OpAccountKicked = 0x02
	OpLoginOk       = 0x03
	OpServerList    = 0x04
	OpPlayFail      = 0x06
	OpPlayOk        = 0x07
	OpGGAuth        = 0x0b
)

// LoginFailReason is the reason byte of LoginFail. The values are a subset
// of Mobius LoginFailReason.
type LoginFailReason uint8

// LoginFail reasons.
const (
	LoginFailSystemErrorLoginLater LoginFailReason = 0x01
	LoginFailUserOrPassWrong       LoginFailReason = 0x02
	LoginFailAccessFailedTryLater  LoginFailReason = 0x04
	LoginFailAccountInUse          LoginFailReason = 0x07
	LoginFailServerOverloaded      LoginFailReason = 0x0f
	LoginFailServerMaintenance     LoginFailReason = 0x10
	LoginFailSystemError           LoginFailReason = 0x14
	LoginFailAccessFailed          LoginFailReason = 0x15
)

// PlayFailReason is the reason byte of PlayFail. The values are a subset of
// Mobius PlayFailReason.
type PlayFailReason uint8

// PlayFail reasons.
const (
	PlayFailSystemErrorLoginLater PlayFailReason = 0x01
	PlayFailAccessFailedTryLater  PlayFailReason = 0x04
	PlayFailAccountInUse          PlayFailReason = 0x07
	PlayFailServerOverloaded      PlayFailReason = 0x0f
	PlayFailServerMaintenance     PlayFailReason = 0x10
	PlayFailSystemError           PlayFailReason = 0x14
	PlayFailAccessFailed          PlayFailReason = 0x15
)

// AccountKickedReason is the reason of AccountKicked (Mobius
// AccountKickedReason).
type AccountKickedReason uint32

// AccountKicked reasons.
const (
	KickedDataStealer       AccountKickedReason = 0x01
	KickedGenericViolation  AccountKickedReason = 0x08
	Kicked7DaysSuspended    AccountKickedReason = 0x10
	KickedPermanentlyBanned AccountKickedReason = 0x20
)
