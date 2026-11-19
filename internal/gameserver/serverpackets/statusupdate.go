package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpStatusUpdate is the opcode of StatusUpdate.
const OpStatusUpdate = 0x0e

// StatusAttr is the ID of a value that StatusUpdate changes.
type StatusAttr int32

// StatusUpdate attributes (Mobius StatusUpdate). Mobius defines no names
// for StatusExp and StatusSP: it refreshes them with UserInfo instead.
const (
	StatusLevel    StatusAttr = 0x01
	StatusExp      StatusAttr = 0x02
	StatusSTR      StatusAttr = 0x03
	StatusDEX      StatusAttr = 0x04
	StatusCON      StatusAttr = 0x05
	StatusINT      StatusAttr = 0x06
	StatusWIT      StatusAttr = 0x07
	StatusMEN      StatusAttr = 0x08
	StatusCurHP    StatusAttr = 0x09
	StatusMaxHP    StatusAttr = 0x0a
	StatusCurMP    StatusAttr = 0x0b
	StatusMaxMP    StatusAttr = 0x0c
	StatusSP       StatusAttr = 0x0d
	StatusCurLoad  StatusAttr = 0x0e
	StatusMaxLoad  StatusAttr = 0x0f
	StatusPAtk     StatusAttr = 0x11
	StatusAtkSpd   StatusAttr = 0x12
	StatusPDef     StatusAttr = 0x13
	StatusEvasion  StatusAttr = 0x14
	StatusAccuracy StatusAttr = 0x15
	StatusCritical StatusAttr = 0x16
	StatusMAtk     StatusAttr = 0x17
	StatusCastSpd  StatusAttr = 0x18
	StatusMDef     StatusAttr = 0x19
	StatusPvPFlag  StatusAttr = 0x1a
	StatusKarma    StatusAttr = 0x1b
	StatusCurCP    StatusAttr = 0x21
	StatusMaxCP    StatusAttr = 0x22
)

// StatusValue is one changed value of a StatusUpdate.
type StatusValue struct {
	Attr  StatusAttr
	Value int32
}

// StatusUpdate changes some values of an object the client shows, such as
// the HP bar of the target. The values are sent in the given order: Mobius
// sends a new target's HP bar as StatusMaxHP, then StatusCurHP (cut to an
// int).
type StatusUpdate struct {
	ObjectID int32
	Values   []StatusValue
}

// NewStatusUpdatePacket builds StatusUpdate (Mobius StatusUpdate.writeImpl).
func NewStatusUpdatePacket(u StatusUpdate) []byte {
	w := packets.NewWriter(9 + 8*len(u.Values))
	w.WriteUint8(OpStatusUpdate)
	w.WriteInt32(u.ObjectID)
	putCount32(w, len(u.Values))
	for _, v := range u.Values {
		w.WriteInt32(int32(v.Attr))
		w.WriteInt32(v.Value)
	}
	return w.Bytes()
}
