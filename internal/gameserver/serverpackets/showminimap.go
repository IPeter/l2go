package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpShowMiniMap is the opcode of ShowMiniMap.
const OpShowMiniMap = 0x9d

// MiniMapID is the map Mobius opens in answer to RequestShowMiniMap.
const MiniMapID = 1665

// Seven Signs periods (Mobius SevenSigns), which the map shows.
const (
	SevenSignsRecruiting     = 0 // the competition is recruiting
	SevenSignsCompetition    = 1
	SevenSignsResults        = 2 // the competition results
	SevenSignsSealValidation = 3
)

// ShowMiniMap opens the map window.
type ShowMiniMap struct {
	MapID int32
	// Period is the current Seven Signs period.
	Period uint8
}

// NewShowMiniMapPacket builds ShowMiniMap (Mobius ShowMiniMap.writeImpl).
func NewShowMiniMapPacket(m ShowMiniMap) []byte {
	w := packets.NewWriter(6)
	w.WriteUint8(OpShowMiniMap)
	w.WriteInt32(m.MapID)
	w.WriteUint8(m.Period)
	return w.Bytes()
}
