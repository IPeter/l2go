package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpEtcStatusUpdate is the opcode of EtcStatusUpdate.
const OpEtcStatusUpdate = 0xf3

// EtcStatusUpdate is the row of small status icons: charges, weight and
// grade penalties, chat block, danger area, charm of courage and death
// penalty.
type EtcStatusUpdate struct {
	Charges       int32 // force charge level, 0-7
	WeightPenalty int32 // 0-4: none, 50%, 66.6%, 80%, 100%
	// ChatBlocked shows the chat block icon: message refusal, a chat ban or
	// silence mode.
	ChatBlocked       bool
	DangerArea        bool
	GradePenalty      bool // a weapon or armor grade penalty
	CharmOfCourage    bool
	DeathPenaltyLevel int32 // 0-15
}

// NewEtcStatusUpdatePacket builds EtcStatusUpdate (Mobius
// EtcStatusUpdate.writeImpl).
func NewEtcStatusUpdatePacket(e EtcStatusUpdate) []byte {
	w := packets.NewWriter(29)
	w.WriteUint8(OpEtcStatusUpdate)
	w.WriteInt32(e.Charges)
	w.WriteInt32(e.WeightPenalty)
	putBool32(w, e.ChatBlocked)
	putBool32(w, e.DangerArea)
	putBool32(w, e.GradePenalty)
	putBool32(w, e.CharmOfCourage)
	w.WriteInt32(e.DeathPenaltyLevel)
	return w.Bytes()
}
