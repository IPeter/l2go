package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpConfirmDlg is the opcode of ConfirmDlg (Mobius CONFIRM_DLG).
const OpConfirmDlg = 0xed

// ConfirmDlg is a yes/no question to the player, answered with DlgAnswer:
// a system message with its parameters, how long the client shows it and
// who asks.
type ConfirmDlg struct {
	Message SystemMessage
	// TimeMs is how long the window stays, in milliseconds (Mobius
	// addTime); 0 for no limit.
	TimeMs int32
	// RequesterID is the object ID of the player who asks (Mobius
	// addRequesterId); DlgAnswer sends it back.
	RequesterID int32
}

// NewConfirmDlgPacket builds ConfirmDlg (Mobius ConfirmDlg.writeImpl): the
// message as SystemMessage writes it (d ID, d count, the parameters), then
// d time and d requester ID.
func NewConfirmDlgPacket(c ConfirmDlg) []byte {
	msg := NewSystemMessagePacket(c.Message)
	w := packets.NewWriter(len(msg) + 8)
	w.WriteUint8(OpConfirmDlg)
	w.WriteBytes(msg[1:])
	w.WriteInt32(c.TimeMs)
	w.WriteInt32(c.RequesterID)
	return w.Bytes()
}
