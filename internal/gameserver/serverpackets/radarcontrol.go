package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpRadarControl is the opcode of RadarControl.
const OpRadarControl = 0xeb

// RadarControl actions, as Mobius Radar sends them: a marker is added with
// (RadarShow, RadarTypeMarker), removed with (RadarRemove,
// RadarTypeMarker), and all markers go with (RadarClear, RadarTypeClear).
const (
	RadarShow   int32 = 0
	RadarRemove int32 = 1
	RadarClear  int32 = 2

	RadarTypeMarker int32 = 1
	RadarTypeClear  int32 = 2
)

// RadarControl adds or removes a marker on the client's radar and map.
type RadarControl struct {
	// Action is RadarShow, RadarRemove or RadarClear.
	Action int32
	// Type is RadarTypeMarker, or RadarTypeClear with RadarClear.
	Type    int32
	X, Y, Z int32
}

// NewRadarControlPacket builds RadarControl (Mobius RadarControl.writeImpl):
// d action, d type, d x, d y, d z.
func NewRadarControlPacket(r RadarControl) []byte {
	w := packets.NewWriter(21)
	w.WriteUint8(OpRadarControl)
	w.WriteInt32(r.Action)
	w.WriteInt32(r.Type)
	w.WriteInt32(r.X)
	w.WriteInt32(r.Y)
	w.WriteInt32(r.Z)
	return w.Bytes()
}
