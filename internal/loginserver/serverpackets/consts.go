// Package serverpackets builds packets sent by the login server to the
// client (C1 protocol). Each builder returns the plain body; the caller
// adds the checksum and encryption.
package serverpackets

// Reasons for LoginFail and PlayFail.
const (
	ReasonSystemError     = 0x01
	ReasonPassWrong       = 0x02
	ReasonUserOrPassWrong = 0x03
	ReasonAccessFailed    = 0x04
	ReasonInfoWrong       = 0x05
	ReasonAccountInUse    = 0x07
	ReasonMaintenance     = 0x10
	ReasonChangeTmpPass   = 0x11
	ReasonExpired         = 0x12
	ReasonNoTimeLeft      = 0x13
)
