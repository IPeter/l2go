package serverpackets

import (
	"bytes"
	"testing"

	"github.com/IPeter/l2go/internal/packets"
)

// ConfirmDlg is SystemMessage's body after its own opcode, then the time
// and the requester (Mobius ConfirmDlg.writeImpl).
func TestConfirmDlgGolden(t *testing.T) {
	got := NewConfirmDlgPacket(ConfirmDlg{
		Message: SystemMessage{ID: 1842, Params: []SystemMessageParam{TextParam("Al"), ZoneNameParam(1, -2, 3)}},
		TimeMs:  30000, RequesterID: 0x10203040,
	})
	w := packets.NewWriter(64)
	w.WriteUint8(0xed)
	w.WriteInt32(1842)
	w.WriteInt32(2)
	w.WriteInt32(0) // text
	w.WriteString("Al")
	w.WriteInt32(7) // zone name
	w.WriteInt32(1)
	w.WriteInt32(-2)
	w.WriteInt32(3)
	w.WriteInt32(30000)
	w.WriteInt32(0x10203040)
	if want := w.Bytes(); !bytes.Equal(got, want) {
		t.Fatalf("ConfirmDlg\n got % x\nwant % x", got, want)
	}
}
