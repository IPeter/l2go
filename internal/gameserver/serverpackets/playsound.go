package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpPlaySound is the opcode of PlaySound.
const OpPlaySound = 0x98

// PlaySound plays a sound on the client: a quest sound such as
// "ItemSound.quest_accept" (Mobius QuestSound) with every other field zero,
// or a boat's horn bound to the boat at a position.
type PlaySound struct {
	// Type is 0 for quests and boats (Mobius "unknown1").
	Type int32
	// Sound is the sound name.
	Sound string
	// Bound is 1 for a sound bound to an object (a boat), 0 for quests
	// (Mobius "unknown3").
	Bound int32
	// ObjectID is the object the sound is bound to, 0 for quests (Mobius
	// "unknown4").
	ObjectID int32
	// X, Y and Z are where the sound plays, zero for quests.
	X, Y, Z int32
}

// NewPlaySoundPacket builds PlaySound (Mobius PlaySound.writeImpl): d type,
// S sound, d bound, d object, d x, d y, d z, and a d Mobius always writes
// as 0.
func NewPlaySoundPacket(p PlaySound) []byte {
	w := packets.NewWriter(1 + 2*len(p.Sound) + 2 + 7*4)
	w.WriteUint8(OpPlaySound)
	w.WriteInt32(p.Type)
	w.WriteString(p.Sound)
	w.WriteInt32(p.Bound)
	w.WriteInt32(p.ObjectID)
	w.WriteInt32(p.X)
	w.WriteInt32(p.Y)
	w.WriteInt32(p.Z)
	w.WriteInt32(0)
	return w.Bytes()
}

// NewQuestSoundPacket builds the PlaySound of a quest sound (Mobius
// new PlaySound(name)): the name, every other field zero.
func NewQuestSoundPacket(sound string) []byte {
	return NewPlaySoundPacket(PlaySound{Sound: sound})
}
