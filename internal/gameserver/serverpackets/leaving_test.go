package serverpackets

import "testing"

// The packets without fields, or with one flag or code: Mobius writes the
// opcode and at most one int.

func TestActionFailedPacket(t *testing.T) {
	m3Check(t, "ActionFailed", NewActionFailedPacket(), unhex(t, "25"))
}

func TestLeaveWorldPacket(t *testing.T) {
	m3Check(t, "LeaveWorld", NewLeaveWorldPacket(), unhex(t, "7e"))
}

func TestRestartResponsePacket(t *testing.T) {
	// c 0x5F, d result.
	m3Check(t, "RestartResponse ok", NewRestartResponsePacket(true), unhex(t, "5f 01000000"))
	m3Check(t, "RestartResponse refused", NewRestartResponsePacket(false), unhex(t, "5f 00000000"))
}

func TestCharDeleteSuccessPacket(t *testing.T) {
	m3Check(t, "CharDeleteSuccess", NewCharDeleteSuccessPacket(), unhex(t, "23"))
}

func TestCharDeleteFailPacket(t *testing.T) {
	// c 0x24, d reason (Mobius CharDeleteFail.REASON_*).
	tests := []struct {
		reason CharDeleteFailReason
		want   string
	}{
		{CharDeleteFailDeletionFailed, "24 01000000"},
		{CharDeleteFailClanMember, "24 02000000"},
		{CharDeleteFailClanLeader, "24 03000000"},
	}
	for _, tt := range tests {
		m3Check(t, "CharDeleteFail", NewCharDeleteFailPacket(tt.reason), unhex(t, tt.want))
	}
}
