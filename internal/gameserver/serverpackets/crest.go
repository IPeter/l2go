package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// Opcodes of the crest packets (Mobius ServerPackets). The large clan
// crest is the extended packet ExOpPledgeEmblem.
const (
	OpPledgeCrest = 0x6c
	OpAllyCrest   = 0xae
)

// NewPledgeCrestPacket builds PledgeCrest, the image of a clan crest, the
// answer to RequestPledgeCrest (Mobius PledgeCrest.writeImpl). nil data, a
// crest that does not exist, is written as length 0.
func NewPledgeCrestPacket(crestID int32, data []byte) []byte {
	w := packets.NewWriter(9 + len(data))
	w.WriteUint8(OpPledgeCrest)
	putCrest(w, crestID, data)
	return w.Bytes()
}

// NewAllyCrestPacket builds AllyCrest, the image of an alliance crest, the
// answer to RequestAllyCrest (Mobius AllyCrest.writeImpl).
func NewAllyCrestPacket(crestID int32, data []byte) []byte {
	w := packets.NewWriter(9 + len(data))
	w.WriteUint8(OpAllyCrest)
	putCrest(w, crestID, data)
	return w.Bytes()
}

// NewExPledgeEmblemPacket builds ExPledgeEmblem, the image of a large clan
// crest, the answer to RequestExPledgeCrestLarge (Mobius
// ExPledgeEmblem.writeImpl): a zero word comes before the crest.
func NewExPledgeEmblemPacket(crestID int32, data []byte) []byte {
	w := newExWriter(ExOpPledgeEmblem, 12+len(data))
	w.WriteInt32(0)
	putCrest(w, crestID, data)
	return w.Bytes()
}

// putCrest writes d crest ID, d length and the image.
func putCrest(w *packets.Writer, crestID int32, data []byte) {
	w.WriteInt32(crestID)
	putCount32(w, len(data))
	w.WriteBytes(data)
}
