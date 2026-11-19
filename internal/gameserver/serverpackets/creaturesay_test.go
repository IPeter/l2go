package serverpackets

import "testing"

func TestCreatureSayPacket(t *testing.T) {
	// Mobius CreatureSay.writeImpl (name and text form): c 0x4A,
	// d speaker object ID, d chat type, S sender name, S text.
	var g m3Golden
	g.c(0x4a).d(0x10000001).d(2).s("->Elmore").s("hello there")
	s := CreatureSay{ObjectID: 0x10000001, Type: ChatWhisper, SenderName: "->Elmore", Text: "hello there"}
	m3Check(t, "whisper echo", NewCreatureSayPacket(s), g.b)

	// A system line: no speaker.
	var sys m3Golden
	sys.c(0x4a).d(0).d(0).s("SYS").s("ok")
	m3Check(t, "system line", NewCreatureSayPacket(CreatureSay{Type: ChatGeneral, SenderName: "SYS", Text: "ok"}), sys.b)
}

// The chat types are the client IDs of Mobius ChatType.
func TestChatTypes(t *testing.T) {
	types := []ChatType{
		ChatGeneral, ChatShout, ChatWhisper, ChatParty, ChatClan, ChatGM,
		ChatPetitionPlayer, ChatPetitionGM, ChatTrade, ChatAlliance,
		ChatAnnouncement, ChatBoat, ChatFriend, ChatMSN, ChatPartyMatchRoom,
		ChatPartyRoomCommander, ChatPartyRoomAll, ChatHeroVoice,
		ChatCriticalAnnounce, ChatScreenAnnounce, ChatBattlefield, ChatMPCCRoom,
	}
	for i, ct := range types {
		if int(ct) != i {
			t.Errorf("chat type #%d = %d", i, ct)
		}
		if !ct.Valid() {
			t.Errorf("ChatType(%d).Valid() = false", ct)
		}
	}
	for _, ct := range []ChatType{-1, 22, 100} {
		if ct.Valid() {
			t.Errorf("ChatType(%d).Valid() = true", ct)
		}
	}
}
