package serverpackets

import "testing"

// The golden tests of the quest and tutorial packets, built field by field
// from the Mobius writeImpl.

func TestQuestServerOpcodes(t *testing.T) {
	tests := []struct {
		name      string
		got, want int
	}{
		{"QuestList", OpQuestList, 0x80},
		{"PlaySound", OpPlaySound, 0x98},
		{"TutorialShowHtml", OpTutorialShowHtml, 0xa0},
		{"TutorialShowQuestionMark", OpTutorialShowQuestionMark, 0xa1},
		{"TutorialEnableClientEvent", OpTutorialEnableClientEvent, 0xa2},
		{"TutorialCloseHtml", OpTutorialCloseHtml, 0xa3},
		{"RadarControl", OpRadarControl, 0xeb},
		{"Ex ExShowQuestMark", ExOpShowQuestMark, 0x1a},
		{"Ex ExShowScreenMessage", ExOpShowScreenMessage, 0x38},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s opcode = %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}

// Mobius PlaySound.writeImpl: c 0x98, d unknown1, S sound, d unknown3, d
// unknown4, d x, d y, d z, d unknown8 (always 0).
func TestPlaySoundPacket(t *testing.T) {
	var g m3Golden
	g.c(0x98).d(0).s("ItemSound.quest_accept").d(0).d(0).d(0).d(0).d(0).d(0)
	m3Check(t, "quest sound", NewQuestSoundPacket("ItemSound.quest_accept"), g.b)
	m3Check(t, "quest sound as a struct", NewPlaySoundPacket(PlaySound{Sound: "ItemSound.quest_accept"}), g.b)

	// The boat form: new PlaySound(0, "itemsound.ship_arrival_departure",
	// 1, boatObjectId, x, y, z).
	var boat m3Golden
	boat.c(0x98).d(0).s("itemsound.ship_arrival_departure").d(1).d(0x60000010).d(-96622).d(261660).d(-3610).d(0)
	m3Check(t, "boat sound", NewPlaySoundPacket(PlaySound{Sound: "itemsound.ship_arrival_departure", Bound: 1,
		ObjectID: 0x60000010, X: -96622, Y: 261660, Z: -3610}), boat.b)

	// Hand-counted bytes of a short name: 1 + 4 + ("a" + 0) 4 + 6 * 4.
	m3Check(t, "short name", NewQuestSoundPacket("a"), unhex(t,
		"98 00000000 6100 0000 00000000 00000000 00000000 00000000 00000000 00000000"))
}

// Mobius TutorialShowHtml.writeImpl: c 0xa0, S html.
func TestTutorialShowHtmlPacket(t *testing.T) {
	const page = "<html><body>Welcome to the tutorial!</body></html>"
	var g m3Golden
	g.c(0xa0).s(page)
	m3Check(t, "TutorialShowHtml", NewTutorialShowHtmlPacket(page), g.b)
	m3Check(t, "empty TutorialShowHtml", NewTutorialShowHtmlPacket(""), unhex(t, "a0 0000"))
}

// Mobius TutorialShowQuestionMark.writeImpl: c 0xa1, d mark;
// TutorialEnableClientEvent: c 0xa2, d event; TutorialCloseHtml: c 0xa3.
func TestTutorialSmallPackets(t *testing.T) {
	m3Check(t, "TutorialShowQuestionMark", NewTutorialShowQuestionMarkPacket(1), unhex(t, "a1 01000000"))
	m3Check(t, "TutorialShowQuestionMark 12", NewTutorialShowQuestionMarkPacket(12), new(m3Golden).c(0xa1).d(12).b)
	m3Check(t, "TutorialEnableClientEvent", NewTutorialEnableClientEventPacket(8), unhex(t, "a2 08000000"))
	m3Check(t, "TutorialEnableClientEvent off", NewTutorialEnableClientEventPacket(0), new(m3Golden).c(0xa2).d(0).b)
	m3Check(t, "TutorialCloseHtml", NewTutorialCloseHtmlPacket(), unhex(t, "a3"))
}

// Mobius RadarControl.writeImpl: c 0xeb, d showRadar, d type, d x, d y, d z.
func TestRadarControlPacket(t *testing.T) {
	tests := []struct {
		name string
		r    RadarControl
		want []byte
	}{
		// Radar.addMarker: RadarControl(2, 2, ...) then (0, 1, ...).
		{"clear", RadarControl{Action: RadarClear, Type: RadarTypeClear, X: -84023, Y: 244581, Z: -3730},
			new(m3Golden).c(0xeb).d(2).d(2).d(-84023).d(244581).d(-3730).b},
		{"show", RadarControl{Action: RadarShow, Type: RadarTypeMarker, X: -84023, Y: 244581, Z: -3730},
			new(m3Golden).c(0xeb).d(0).d(1).d(-84023).d(244581).d(-3730).b},
		// Radar.removeMarker: RadarControl(1, 1, ...).
		{"remove", RadarControl{Action: RadarRemove, Type: RadarTypeMarker, X: 1, Y: 2, Z: 3},
			unhex(t, "eb 01000000 01000000 01000000 02000000 03000000")},
	}
	for _, tt := range tests {
		m3Check(t, "RadarControl "+tt.name, NewRadarControlPacket(tt.r), tt.want)
	}
}

// Mobius ExShowQuestMark.writeImpl: c 0xfe, h 0x1a, d quest ID.
func TestExShowQuestMarkPacket(t *testing.T) {
	m3Check(t, "ExShowQuestMark", NewExShowQuestMarkPacket(1), unhex(t, "fe 1a00 01000000"))
	m3Check(t, "ExShowQuestMark 401", NewExShowQuestMarkPacket(401), new(m3Golden).c(0xfe).h(0x1a).d(401).b)
}

// Mobius ExShowScreenMessage.writeImpl: c 0xfe, h 0x38, d type, d system
// message, d position, d unk1, d size, d unk2, d unk3, d effect, d time, d
// fade, S text, then S text again when the system message is -1, or each
// parameter.
func TestExShowScreenMessagePacket(t *testing.T) {
	const text = "The radio signal detector is responding."
	// new ExShowScreenMessage(text, 4500): type 1, -1, top center, the
	// rest zero, and the text twice.
	var g m3Golden
	g.c(0xfe).h(0x38).d(1).d(-1).d(2).d(0).d(0).d(0).d(0).d(0).d(4500).d(0).s(text).s(text)
	m3Check(t, "text", NewExShowScreenMessagePacket(ScreenText(text, 4500)), g.b)

	// new ExShowScreenMessage(text, position, time, size, fade, effect).
	var full m3Golden
	full.c(0xfe).h(0x38).d(1).d(-1).d(5).d(0).d(1).d(0).d(0).d(1).d(3000).d(1).s("Hi").s("Hi")
	m3Check(t, "text with every option", NewExShowScreenMessagePacket(ExShowScreenMessage{Type: ScreenMessageText,
		SystemMessageID: -1, Position: ScreenMiddleCenter, Size: 1, Effect: true, Time: 3000, Fade: true, Text: "Hi"}), full.b)

	// new ExShowScreenMessage(SystemMessageId, position, time, params):
	// type 2, the null text as an empty string, then the parameters.
	var sys m3Golden
	sys.c(0xfe).h(0x38).d(2).d(1210).d(2).d(0).d(0).d(0).d(0).d(0).d(10000).d(0).s("").s("Gremlin").s("3")
	m3Check(t, "system message", NewExShowScreenMessagePacket(ScreenSystemMessage(1210, ScreenTopCenter, 10000, "Gremlin", "3")), sys.b)

	var noParams m3Golden
	noParams.c(0xfe).h(0x38).d(2).d(54).d(7).d(0).d(0).d(0).d(0).d(0).d(1000).d(0).s("")
	m3Check(t, "system message without parameters", NewExShowScreenMessagePacket(ScreenSystemMessage(54, ScreenBottomCenter, 1000)), noParams.b)

	// Hand-counted: header 3, ten d, then "a" + 0 twice.
	m3Check(t, "short text", NewExShowScreenMessagePacket(ScreenText("a", 1)), unhex(t,
		"fe 3800 01000000 ffffffff 02000000 00000000 00000000 00000000 00000000 00000000 01000000 00000000 6100 0000 6100 0000"))
}
