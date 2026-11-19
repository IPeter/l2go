package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpPartySpelled is the opcode of PartySpelled.
const OpPartySpelled = 0xee

// PartySpelledType is the kind of creature whose effects PartySpelled
// shows.
type PartySpelledType int32

// PartySpelled creature types (Mobius: isServitor ? 2 : isPet ? 1 : 0).
const (
	PartySpelledPlayer   PartySpelledType = 0
	PartySpelledPet      PartySpelledType = 1
	PartySpelledServitor PartySpelledType = 2
)

// PartySpelled is the effect icons of a party member, shown in the party
// window, or of a player's pet or servitor.
type PartySpelled struct {
	Type     PartySpelledType
	ObjectID int32
	// Effects are the effects in use, as in AbnormalStatusUpdate: the
	// skill's display ID and display level, and the seconds left.
	Effects []AbnormalStatusEntry
}

// NewPartySpelledPacket builds PartySpelled (Mobius
// PartySpelled.writeImpl). Unlike AbnormalStatusUpdate, the number of
// effects is an int. An empty list is valid.
func NewPartySpelledPacket(p PartySpelled) []byte {
	w := packets.NewWriter(13 + 10*len(p.Effects))
	w.WriteUint8(OpPartySpelled)
	w.WriteInt32(int32(p.Type))
	w.WriteInt32(p.ObjectID)
	putCount32(w, len(p.Effects))
	for _, e := range p.Effects {
		w.WriteInt32(e.SkillID)
		putInt16(w, e.SkillLevel)
		w.WriteInt32(e.Seconds)
	}
	return w.Bytes()
}
