package serverpackets

import "testing"

func TestSendMacroListPacketEmpty(t *testing.T) {
	// The list EnterWorld sends without macros: c 0xE7, d revision (2),
	// c 0, c count 0, c 0 (no macro).
	m3Check(t, "empty SendMacroList", NewSendMacroListPacket(SendMacroList{Revision: 2}),
		unhex(t, "e7 02000000 00 00 00"))
}

func TestSendMacroListPacket(t *testing.T) {
	l := SendMacroList{
		Revision: 5,
		Count:    3,
		Macro: &Macro{
			ID: 1001, Name: "Buff", Description: "self buffs", Acronym: "BF", Icon: 4,
			Commands: []MacroCommand{
				{Type: MacroCommandSkill, D1: 1204, D2: 0, Command: "/useskill Wind Walk"},
				{Type: MacroCommandShortcut, D1: 0, D2: 13, Command: ""},
			},
		},
	}
	// Mobius SendMacroList.writeImpl: c 0xE7, d revision, c 0, c count,
	// c 1 (has a macro), d macro ID, S name, S description, S acronym,
	// c icon, c command count, then per command c index from 1, c type,
	// d d1, c d2, S command.
	var g m3Golden
	g.c(0xe7).d(5).c(0).c(3).c(1)
	g.d(1001).s("Buff").s("self buffs").s("BF").c(4).c(2)
	g.c(1).c(1).d(1204).c(0).s("/useskill Wind Walk")
	g.c(2).c(4).d(0).c(13).s("")
	m3Check(t, "SendMacroList", NewSendMacroListPacket(l), g.b)
}
