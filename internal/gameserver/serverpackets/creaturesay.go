package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpCreatureSay is the opcode of CreatureSay.
const OpCreatureSay = 0x4a

// ChatType is a chat channel as the client numbers it (the client IDs of
// Mobius ChatType). Say2 carries the same numbers.
type ChatType int32

// Chat types (Mobius ChatType).
const (
	ChatGeneral            ChatType = 0
	ChatShout              ChatType = 1 // "!" prefix
	ChatWhisper            ChatType = 2 // '"' prefix; Say2 then carries the target name
	ChatParty              ChatType = 3
	ChatClan               ChatType = 4
	ChatGM                 ChatType = 5
	ChatPetitionPlayer     ChatType = 6
	ChatPetitionGM         ChatType = 7
	ChatTrade              ChatType = 8 // "+" prefix
	ChatAlliance           ChatType = 9
	ChatAnnouncement       ChatType = 10
	ChatBoat               ChatType = 11
	ChatFriend             ChatType = 12
	ChatMSN                ChatType = 13
	ChatPartyMatchRoom     ChatType = 14
	ChatPartyRoomCommander ChatType = 15
	ChatPartyRoomAll       ChatType = 16
	ChatHeroVoice          ChatType = 17
	ChatCriticalAnnounce   ChatType = 18
	ChatScreenAnnounce     ChatType = 19
	ChatBattlefield        ChatType = 20
	ChatMPCCRoom           ChatType = 21
)

// Valid reports whether t is a chat type Mobius knows. Mobius disconnects
// a player whose Say2 has any other type.
func (t ChatType) Valid() bool { return t >= ChatGeneral && t <= ChatMPCCRoom }

// CreatureSay is a chat line shown in the chat window, and over the
// speaker's head for nearby chat.
type CreatureSay struct {
	ObjectID   int32 // the speaker; 0 when there is none
	Type       ChatType
	SenderName string // for a whisper echoed to its sender, "->" + target name
	Text       string
}

// NewCreatureSayPacket builds CreatureSay in the sender name and text form
// (Mobius CreatureSay.writeImpl). The boat form, a character ID and a
// system message ID instead of the name and text, is not needed yet.
func NewCreatureSayPacket(s CreatureSay) []byte {
	w := packets.NewWriter(13 + 2*(len(s.SenderName)+len(s.Text)))
	w.WriteUint8(OpCreatureSay)
	w.WriteInt32(s.ObjectID)
	w.WriteInt32(int32(s.Type))
	w.WriteString(s.SenderName)
	w.WriteString(s.Text)
	return w.Bytes()
}
