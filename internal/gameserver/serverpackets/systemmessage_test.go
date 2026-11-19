package serverpackets

import "testing"

func TestSystemMessagePacketNoParams(t *testing.T) {
	// c 0x64, d message ID, d parameter count.
	m3Check(t, "welcome", NewSystemMessagePacket(SystemMessage{ID: SysMsgWelcome}), unhex(t, "64 22000000 00000000"))
}

func TestSystemMessagePacketParams(t *testing.T) {
	m := SystemMessage{
		ID: 9999,
		Params: []SystemMessageParam{
			TextParam("Frostwind"),
			NumberParam(-5),
			NPCNameParam(30001),
			ItemNameParam(57),
			SkillNameParam(1177, 5),
			CastleNameParam(3),
			LongNumberParam(5000000000),
			ZoneNameParam(-71338, 258271, -3104),
			PlayerNameParam("Elmore"),
			{Type: ParamElementName, Value: 2},
			{Type: ParamInstanceName, Value: 11},
			{Type: ParamDoorName, Value: 24190001},
			{Type: ParamSystemString, Value: 1281},
			{Type: 8, Value: 77}, // a type Mobius has no case for: only the type is sent
		},
	}
	// Mobius SystemMessage.writeImpl: c 0x64, d ID, d count, then per
	// parameter d type and:
	//   text, player name: S
	//   long number: q
	//   item, castle, number, NPC, element, system string, instance, door: d
	//   skill name: d ID, d level
	//   zone name: d x, d y, d z
	var g m3Golden
	g.c(0x64).d(9999).d(14)
	g.d(0).s("Frostwind")
	g.d(1).d(-5)
	g.d(2).d(1030001) // NPC ID + 1,000,000
	g.d(3).d(57)
	g.d(4).d(1177).d(5)
	g.d(5).d(3)
	g.d(6).q(5000000000)
	g.d(7).d(-71338).d(258271).d(-3104)
	g.d(12).s("Elmore")
	g.d(9).d(2)
	g.d(10).d(11)
	g.d(11).d(24190001)
	g.d(13).d(1281)
	g.d(8)
	m3Check(t, "SystemMessage", NewSystemMessagePacket(m), g.b)
}

// Free text, as Mobius Player.sendMessage sends it.
func TestSystemMessageTextPacket(t *testing.T) {
	var g m3Golden
	g.c(0x64).d(1983).d(1).d(0).s("The command 'foo' does not exist!")
	m3Check(t, "text", NewSystemMessageTextPacket("The command 'foo' does not exist!"), g.b)
}

// The message IDs are those of Mobius SystemMessageId.
func TestSystemMessageIDs(t *testing.T) {
	tests := []struct {
		got  SystemMessageID
		want int32
	}{
		{SysMsgWelcome, 34},
		{SysMsgCannotExitInCombat, 101},
		{SysMsgCannotRestartInCombat, 102},
		{SysMsgPlayerNotOnline, 145},
		{SysMsgChattingProhibited, 147},
		{SysMsgMessageRefusal, 176},
		{SysMsgFriendLoggedIn, 503},
		{SysMsgCurrentLocationTalkingIsland, 910},
		{SysMsgChatTooLong, 1078},
		{SysMsgText, 1983},
	}
	for _, tt := range tests {
		if int32(tt.got) != tt.want {
			t.Errorf("message ID %d, want %d", tt.got, tt.want)
		}
	}
}
