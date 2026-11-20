package testclient

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// The quest and tutorial packets as Mobius writes and reads them,
// transcribed from their writeImpl and readImpl.

// mobiusPlaySound is PlaySound.writeImpl: d unknown1, S sound, d unknown3,
// d unknown4, d x, d y, d z, d unknown8 (0).
func mobiusPlaySound(p PlaySound) []byte {
	return msg(0x98).i32(p.Type).str(p.Sound).i32(p.Bound).i32(p.ObjectID).i32(p.X).i32(p.Y).i32(p.Z).i32(p.Reserved).b
}

// mobiusRadarControl is RadarControl.writeImpl: d showRadar, d type, d x,
// d y, d z.
func mobiusRadarControl(r RadarControl) []byte {
	return msg(0xeb).i32(r.Action).i32(r.Type).i32(r.X).i32(r.Y).i32(r.Z).b
}

// mobiusExShowScreenMessage is ExShowScreenMessage.writeImpl: the Ex
// header, ten d, S text, then S text again for a text (-1) or each
// parameter.
func mobiusExShowScreenMessage(m ExShowScreenMessage) []byte {
	w := msg(0xfe).u16(0x38).i32(m.Type).i32(m.SystemMessageID).i32(m.Position).i32(m.Unknown1).i32(m.Size).
		i32(m.Unknown2).i32(m.Unknown3).i32(b2i(m.Effect)).i32(m.Time).i32(b2i(m.Fade)).str(m.Text)
	if m.SystemMessageID == -1 {
		w.str(m.Text)
	} else {
		for _, p := range m.Params {
			w.str(p)
		}
	}
	return w.b
}

// mobiusQuestList is QuestList.writeImpl: h count, then d ID, d cond.
func mobiusQuestList(quests ...Quest) []byte {
	w := msg(0x80).u16(uint16(len(quests)))
	for _, q := range quests {
		w.i32(q.ID).i32(q.State)
	}
	return w.b
}

func TestDecodeQuestPackets(t *testing.T) {
	const page = "<html><body>Click the question mark.<a action=\"link tutorial_close_0\">Close</a></body></html>"
	boat := PlaySound{Sound: "itemsound.ship_arrival_departure", Bound: 1, ObjectID: 0x60000010, X: -96622, Y: 261660, Z: -3610}
	text := ExShowScreenMessage{Type: 1, SystemMessageID: -1, Position: 2, Time: 4500, Text: "The radio signal detector is responding."}
	full := ExShowScreenMessage{Type: 1, SystemMessageID: -1, Position: 5, Size: 1, Effect: true, Time: 3000, Fade: true, Text: "Hi"}
	sys := ExShowScreenMessage{Type: 2, SystemMessageID: 1210, Position: 2, Time: 10000, Params: []string{"Gremlin", "3"}}
	tests := []struct {
		name string
		body []byte
		want Packet
	}{
		{"PlaySound quest", mobiusPlaySound(PlaySound{Sound: SoundQuestAccept}), PlaySound{Sound: SoundQuestAccept}},
		{"PlaySound boat", mobiusPlaySound(boat), boat},
		{"TutorialShowHtml", msg(0xa0).str(page).b, TutorialShowHtml{HTML: page}},
		{"TutorialShowHtml empty", msg(0xa0).str("").b, TutorialShowHtml{}},
		{"TutorialShowQuestionMark", msg(0xa1).i32(1).b, TutorialShowQuestionMark{Mark: 1}},
		{"TutorialEnableClientEvent", msg(0xa2).i32(8).b, TutorialEnableClientEvent{Event: 8}},
		{"TutorialCloseHtml", msg(0xa3).b, TutorialCloseHtml{}},
		{"RadarControl show", mobiusRadarControl(RadarControl{Action: RadarShow, Type: RadarTypeMarker, X: -84023, Y: 244581, Z: -3730}),
			RadarControl{Action: RadarShow, Type: RadarTypeMarker, X: -84023, Y: 244581, Z: -3730}},
		{"RadarControl clear", mobiusRadarControl(RadarControl{Action: RadarClear, Type: RadarTypeClear}),
			RadarControl{Action: RadarClear, Type: RadarTypeClear}},
		{"ExShowQuestMark", msg(0xfe).u16(0x1a).i32(401).b, ExShowQuestMark{QuestID: 401}},
		{"ExShowScreenMessage text", mobiusExShowScreenMessage(text), text},
		{"ExShowScreenMessage options", mobiusExShowScreenMessage(full), full},
		{"ExShowScreenMessage system message", mobiusExShowScreenMessage(sys), sys},
		{"ExShowScreenMessage system message alone", mobiusExShowScreenMessage(ExShowScreenMessage{Type: 2, SystemMessageID: 54}),
			ExShowScreenMessage{Type: 2, SystemMessageID: 54}},
		{"QuestList", mobiusQuestList(Quest{ID: 1, State: 2}, Quest{ID: 255, State: -0x7ffffc01}),
			QuestList{Quests: []Quest{{ID: 1, State: 2}, {ID: 255, State: -0x7ffffc01}}}},
	}
	for _, tt := range tests {
		checkDecode(t, tt.name, tt.body, tt.want)
	}

	if p := (RadarControl{X: 1, Y: 2, Z: 3}).Position(); p != (Position{1, 2, 3}) {
		t.Errorf("RadarControl.Position() = %v", p)
	}
}

// TestQuestPacketLayouts writes packets byte by byte from the Mobius
// writeImpl and checks the test builders and the decoder against them.
func TestQuestPacketLayouts(t *testing.T) {
	sound := []byte{
		0x98,
		0x00, 0x00, 0x00, 0x00, // unknown1
		'a', 0x00, 0x00, 0x00, // "a"
		0x00, 0x00, 0x00, 0x00, // unknown3
		0x00, 0x00, 0x00, 0x00, // unknown4
		0x01, 0x00, 0x00, 0x00, // x
		0x02, 0x00, 0x00, 0x00, // y
		0x03, 0x00, 0x00, 0x00, // z
		0x00, 0x00, 0x00, 0x00, // unknown8
	}
	if got := mobiusPlaySound(PlaySound{Sound: "a", X: 1, Y: 2, Z: 3}); !bytes.Equal(got, sound) {
		t.Errorf("PlaySound builder:\n got % x\nwant % x", got, sound)
	}
	checkDecode(t, "PlaySound bytes", sound, PlaySound{Sound: "a", X: 1, Y: 2, Z: 3})

	screen := []byte{
		0xfe, 0x38, 0x00,
		0x01, 0x00, 0x00, 0x00, // type
		0xff, 0xff, 0xff, 0xff, // system message -1
		0x02, 0x00, 0x00, 0x00, // top center
		0x00, 0x00, 0x00, 0x00, // unknown1
		0x00, 0x00, 0x00, 0x00, // size
		0x00, 0x00, 0x00, 0x00, // unknown2
		0x00, 0x00, 0x00, 0x00, // unknown3
		0x00, 0x00, 0x00, 0x00, // effect
		0x10, 0x27, 0x00, 0x00, // 10000 ms
		0x00, 0x00, 0x00, 0x00, // fade
		'x', 0x00, 0x00, 0x00, // "x"
		'x', 0x00, 0x00, 0x00, // "x" again
	}
	want := ExShowScreenMessage{Type: 1, SystemMessageID: -1, Position: 2, Time: 10000, Text: "x"}
	if got := mobiusExShowScreenMessage(want); !bytes.Equal(got, screen) {
		t.Errorf("ExShowScreenMessage builder:\n got % x\nwant % x", got, screen)
	}
	checkDecode(t, "ExShowScreenMessage bytes", screen, want)
}

func TestDecodeQuestPacketErrors(t *testing.T) {
	head := func(sys int32) *wire {
		return msg(0xfe).u16(0x38).i32(1).i32(sys).i32(2).i32(0).i32(0).i32(0).i32(0).i32(0).i32(100).i32(0)
	}
	tests := []struct {
		name   string
		body   []byte
		substr string
	}{
		{"PlaySound short", msg(0x98).i32(0).str("a").i32(0).i32(0).i32(0).i32(0).i32(0).b, "PlaySound: packets: short packet"},
		{"TutorialShowQuestionMark extra", msg(0xa1).i32(1).u8(0).b, "TutorialShowQuestionMark: 1 bytes after the last field"},
		{"TutorialEnableClientEvent short", msg(0xa2).u16(1).b, "TutorialEnableClientEvent: packets: short packet"},
		{"TutorialCloseHtml extra", msg(0xa3).i32(0).b, "TutorialCloseHtml: 4 bytes after the last field"},
		{"RadarControl short", msg(0xeb).i32(0).i32(1).i32(2).i32(3).b, "RadarControl: packets: short packet"},
		{"ExShowQuestMark short", msg(0xfe).u16(0x1a).u16(1).b, "ExShowQuestMark: packets: short packet"},
		{"ExShowScreenMessage effect 2", msg(0xfe).u16(0x38).i32(1).i32(-1).i32(2).i32(0).i32(0).i32(0).i32(0).i32(2).
			i32(100).i32(0).str("a").str("a").b, "ExShowScreenMessage: effect flag is 2"},
		{"ExShowScreenMessage fade 3", msg(0xfe).u16(0x38).i32(1).i32(-1).i32(2).i32(0).i32(0).i32(0).i32(0).i32(0).
			i32(100).i32(3).str("a").str("a").b, "ExShowScreenMessage: fade flag is 3"},
		{"ExShowScreenMessage text once", head(-1).str("a").b, "ExShowScreenMessage: packets: short packet"},
		{"ExShowScreenMessage texts differ", head(-1).str("a").str("b").b, `ExShowScreenMessage: second text "b" differs from "a"`},
		{"ExShowScreenMessage text extra", head(-1).str("a").str("a").u16(0).b, "ExShowScreenMessage: 2 bytes after the last field"},
		{"ExShowScreenMessage parameter cut", head(54).str("").u16('x').b, "ExShowScreenMessage: packets: short packet"},
	}
	for _, tt := range tests {
		_, err := DecodeServerPacket(tt.body)
		if err == nil || !strings.Contains(err.Error(), tt.substr) {
			t.Errorf("%s: got %v, want an error containing %q", tt.name, err, tt.substr)
			continue
		}
		if !errors.Is(err, ErrMalformed) {
			t.Errorf("%s: %v does not wrap ErrMalformed", tt.name, err)
		}
	}
}

// The client packets of the quests as Mobius reads them: RequestQuestAbort
// d quest; RequestTutorialLinkHtml and RequestTutorialPassCmdToServer S
// bypass; RequestTutorialQuestionMark d number; RequestTutorialClientEvent
// d event.
func TestEncodeQuestRequests(t *testing.T) {
	tests := []struct {
		name string
		got  []byte
		want []byte
	}{
		{"RequestQuestAbort", encodeInt(OpRequestQuestAbort, 401), []byte{0x64, 0x91, 0x01, 0x00, 0x00}},
		{"RequestTutorialLinkHtml", encodeString(OpRequestTutorialLinkHtml, "close"), msg(0x7b).str("close").b},
		{"RequestTutorialPassCmdToServer", encodeString(OpRequestTutorialPassCmdToServer, "_bbshome"), msg(0x7c).str("_bbshome").b},
		{"RequestTutorialQuestionMark", encodeInt(OpRequestTutorialQuestionMark, 1), []byte{0x7d, 0x01, 0x00, 0x00, 0x00}},
		{"RequestTutorialClientEvent", encodeInt(OpRequestTutorialClientEvent, 8), msg(0x7e).i32(8).b},
	}
	for _, tt := range tests {
		if !bytes.Equal(tt.got, tt.want) {
			t.Errorf("%s:\n got % x\nwant % x", tt.name, tt.got, tt.want)
		}
		if got := clientPacketName(tt.got); got != tt.name {
			t.Errorf("clientPacketName(% x) = %q, want %q", tt.got, got, tt.name)
		}
	}
}

// TestQuestFlow walks the quest and tutorial packets against a fake server
// in Mobius's order: the tutorial's question mark and page, a radar marker,
// the client event, accepting a quest (QuestList, ExShowQuestMark,
// PlaySound), a screen message, the quest list and aborting the quest.
func TestQuestFlow(t *testing.T) {
	pos := Position{-71338, 258271, -3104}
	const tutorialPage = "<html><body>Welcome!<a action=\"link tutorial_close_0\">Close</a></body></html>"
	radar := RadarControl{Action: RadarShow, Type: RadarTypeMarker, X: -71424, Y: 258336, Z: -3109}
	screen := ExShowScreenMessage{Type: 1, SystemMessageID: -1, Position: 2, Time: 4500, Text: "Talk to the Newbie Helper."}

	addr := startFakeGame(t, func(f *fakeGame) {
		f.enterQuick(pos, nil, mobiusQuestList(Quest{ID: 255, State: 0}))

		// Q255: the voice, the question mark, then its page and a marker.
		f.send(mobiusPlaySound(PlaySound{Sound: "tutorial_voice_001a"}))
		f.send(msg(0xa1).i32(1).b)
		f.expect(msg(0x7d).i32(1).b)
		f.send(msg(0xa0).str(tutorialPage).b)
		f.send(mobiusRadarControl(RadarControl{Action: RadarClear, Type: RadarTypeClear, X: radar.X, Y: radar.Y, Z: radar.Z}))
		f.send(mobiusRadarControl(radar))
		f.expect(msg(0x7b).str("tutorial_close_0").b)
		f.send(msg(0xa3).b)
		f.send(msg(0xa2).i32(8).b)
		f.expect(msg(0x7e).i32(8).b)
		f.expect(msg(0x7c).str("_bbshome").b)

		// Accepting Q001 at its NPC: QuestList, the quest mark, the sound.
		f.send(mobiusQuestList(Quest{ID: 255, State: 0}, Quest{ID: 1, State: 1}))
		f.send(msg(0xfe).u16(0x1a).i32(1).b)
		f.send(mobiusPlaySound(PlaySound{Sound: SoundQuestAccept}))
		f.send(mobiusExShowScreenMessage(screen))

		// The quest log, then aborting Q001.
		f.expect([]byte{0x63})
		f.send(mobiusQuestList(Quest{ID: 255, State: 0}, Quest{ID: 1, State: 1}))
		f.expect(msg(0x64).i32(1).b)
		f.send(mobiusQuestList(Quest{ID: 255, State: 0}))
		f.drain()
	})

	ctx := testContext(t)
	c := clientEnterQuick(t, ctx, addr)

	voice, err := Await[PlaySound](ctx, c, nil)
	if err != nil || voice.Sound != "tutorial_voice_001a" {
		t.Fatalf("the tutorial voice = %+v, %v", voice, err)
	}
	mark, err := Await[TutorialShowQuestionMark](ctx, c, nil)
	if err != nil || mark.Mark != 1 {
		t.Fatalf("TutorialShowQuestionMark = %+v, %v", mark, err)
	}
	page, err := c.OpenQuestionMark(ctx, mark.Mark)
	if err != nil || page.HTML != tutorialPage {
		t.Fatalf("OpenQuestionMark = %+v, %v", page, err)
	}
	got, err := Await(ctx, c, func(r RadarControl) bool { return r.Action == RadarShow })
	if err != nil || got != radar {
		t.Fatalf("RadarControl = %+v, %v", got, err)
	}
	if err := c.TutorialLink(ctx, "tutorial_close_0"); err != nil {
		t.Fatal(err)
	}
	if _, err := Await[TutorialCloseHtml](ctx, c, nil); err != nil {
		t.Fatal(err)
	}
	ev, err := Await[TutorialEnableClientEvent](ctx, c, nil)
	if err != nil || ev.Event != 8 {
		t.Fatalf("TutorialEnableClientEvent = %+v, %v", ev, err)
	}
	if err := c.TutorialClientEvent(ctx, ev.Event); err != nil {
		t.Fatal(err)
	}
	if err := c.TutorialBypass(ctx, "_bbshome"); err != nil {
		t.Fatal(err)
	}

	list, err := Await(ctx, c, func(l QuestList) bool { return len(l.Quests) == 2 })
	if err != nil {
		t.Fatal(err)
	}
	if want := []Quest{{ID: 255}, {ID: 1, State: 1}}; !reflect.DeepEqual(list.Quests, want) {
		t.Errorf("QuestList = %+v, want %+v", list.Quests, want)
	}
	qm, err := Expect[ExShowQuestMark](ctx, c)
	if err != nil || qm.QuestID != 1 {
		t.Fatalf("ExShowQuestMark = %+v, %v", qm, err)
	}
	accept, err := Expect[PlaySound](ctx, c)
	if err != nil || accept.Sound != SoundQuestAccept {
		t.Fatalf("PlaySound = %+v, %v", accept, err)
	}
	sm, err := Expect[ExShowScreenMessage](ctx, c)
	if err != nil || !reflect.DeepEqual(sm, screen) {
		t.Fatalf("ExShowScreenMessage = %+v, %v", sm, err)
	}

	log, err := c.Quests(ctx)
	if err != nil || len(log.Quests) != 2 {
		t.Fatalf("Quests = %+v, %v", log, err)
	}
	after, err := c.AbortQuest(ctx, 1)
	if err != nil {
		t.Fatalf("AbortQuest: %v", err)
	}
	if want := []Quest{{ID: 255}}; !reflect.DeepEqual(after.Quests, want) {
		t.Errorf("QuestList after the abort = %+v, want %+v", after.Quests, want)
	}
}
