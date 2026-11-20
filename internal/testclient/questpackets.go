package testclient

import "context"

// Server opcodes of the quest and tutorial packets, from Mobius
// ServerPackets. QuestList (OpQuestList) is decoded in worldpackets.go.
const (
	OpPlaySound                 byte = 0x98
	OpTutorialShowHtml          byte = 0xa0
	OpTutorialShowQuestionMark  byte = 0xa1
	OpTutorialEnableClientEvent byte = 0xa2
	OpTutorialCloseHtml         byte = 0xa3
	OpRadarControl              byte = 0xeb
)

// Ex server sub-opcodes of the quest packets.
const (
	ExOpShowQuestMark     uint16 = 0x1a
	ExOpShowScreenMessage uint16 = 0x38
)

// Client opcodes of the quest and tutorial packets, from Mobius
// ClientPackets. RequestQuestList is OpRequestQuestList.
const (
	OpRequestQuestAbort              byte = 0x64
	OpRequestTutorialLinkHtml        byte = 0x7b
	OpRequestTutorialPassCmdToServer byte = 0x7c
	OpRequestTutorialQuestionMark    byte = 0x7d
	OpRequestTutorialClientEvent     byte = 0x7e
)

// Quest sounds, the names Mobius QuestSound plays most.
const (
	SoundQuestAccept  = "ItemSound.quest_accept"
	SoundQuestMiddle  = "ItemSound.quest_middle"
	SoundQuestFinish  = "ItemSound.quest_finish"
	SoundQuestItemGet = "ItemSound.quest_itemget"
	SoundQuestGiveUp  = "ItemSound.quest_giveup"
	SoundTutorial     = "ItemSound.quest_tutorial"
)

// RadarControl actions and types, as Mobius Radar sends them.
const (
	RadarShow   int32 = 0
	RadarRemove int32 = 1
	RadarClear  int32 = 2

	RadarTypeMarker int32 = 1
	RadarTypeClear  int32 = 2
)

// PlaySound plays a sound (0x98). A quest sound has only Sound set; a boat
// horn is bound to the boat (Bound 1, ObjectID) at a position.
type PlaySound struct {
	Type     int32
	Sound    string
	Bound    int32
	ObjectID int32
	X, Y, Z  int32
	// Reserved is the last int, which Mobius always writes as 0.
	Reserved int32
}

// PacketName implements Packet.
func (PlaySound) PacketName() string { return "PlaySound" }

func readPlaySound(f *fields) Packet {
	var p PlaySound
	p.Type = f.i32()
	p.Sound = f.str()
	p.Bound = f.i32()
	p.ObjectID = f.i32()
	p.X, p.Y, p.Z = f.i32(), f.i32(), f.i32()
	p.Reserved = f.i32()
	return p
}

// TutorialShowHtml opens the tutorial window with a page (0xa0).
type TutorialShowHtml struct {
	HTML string
}

// PacketName implements Packet.
func (TutorialShowHtml) PacketName() string { return "TutorialShowHtml" }

func readTutorialShowHtml(f *fields) Packet { return TutorialShowHtml{HTML: f.str()} }

// TutorialShowQuestionMark shows a tutorial question mark (0xa1); clicking
// it sends its Mark back (ClickQuestionMark).
type TutorialShowQuestionMark struct {
	Mark int32
}

// PacketName implements Packet.
func (TutorialShowQuestionMark) PacketName() string { return "TutorialShowQuestionMark" }

func readTutorialShowQuestionMark(f *fields) Packet {
	return TutorialShowQuestionMark{Mark: f.i32()}
}

// TutorialEnableClientEvent turns a tutorial client event on (0xa2); the
// client then reports it (TutorialClientEvent). 0 turns them off.
type TutorialEnableClientEvent struct {
	Event int32
}

// PacketName implements Packet.
func (TutorialEnableClientEvent) PacketName() string { return "TutorialEnableClientEvent" }

func readTutorialEnableClientEvent(f *fields) Packet {
	return TutorialEnableClientEvent{Event: f.i32()}
}

// TutorialCloseHtml closes the tutorial window (0xa3). It has no fields.
type TutorialCloseHtml struct{}

// PacketName implements Packet.
func (TutorialCloseHtml) PacketName() string { return "TutorialCloseHtml" }

func readTutorialCloseHtml(*fields) Packet { return TutorialCloseHtml{} }

// RadarControl adds or removes a radar marker (0xeb).
type RadarControl struct {
	// Action is RadarShow, RadarRemove or RadarClear.
	Action int32
	// Type is RadarTypeMarker, or RadarTypeClear with RadarClear.
	Type    int32
	X, Y, Z int32
}

// PacketName implements Packet.
func (RadarControl) PacketName() string { return "RadarControl" }

// Position returns the marker's position.
func (r RadarControl) Position() Position { return Position{r.X, r.Y, r.Z} }

func readRadarControl(f *fields) Packet {
	var p RadarControl
	p.Action = f.i32()
	p.Type = f.i32()
	p.X, p.Y, p.Z = f.i32(), f.i32(), f.i32()
	return p
}

// ExShowQuestMark marks a quest of the quest log as updated (FE 1A).
type ExShowQuestMark struct {
	QuestID int32
}

// PacketName implements Packet.
func (ExShowQuestMark) PacketName() string { return "ExShowQuestMark" }

func readExShowQuestMark(f *fields) Packet { return ExShowQuestMark{QuestID: f.i32()} }

// ExShowScreenMessage shows a text or a system message on the screen (FE
// 38).
type ExShowScreenMessage struct {
	// Type is 1 for a text, 2 for a system message (the Mobius
	// constructors).
	Type int32
	// SystemMessageID is -1 for a text.
	SystemMessageID int32
	// Position is 1 (top left) to 8 (bottom right); 2 is the top center.
	Position int32
	Unknown1 int32
	// Size is the font size: 0 normal, 1 small.
	Size     int32
	Unknown2 int32
	Unknown3 int32
	Effect   bool
	// Time is in milliseconds.
	Time int32
	Fade bool
	// Text is the text of a text message (Mobius writes it twice), empty
	// for a system message.
	Text string
	// Params are the parameters of a system message.
	Params []string
}

// PacketName implements Packet.
func (ExShowScreenMessage) PacketName() string { return "ExShowScreenMessage" }

func readExShowScreenMessage(f *fields) Packet {
	var p ExShowScreenMessage
	p.Type = f.i32()
	p.SystemMessageID = f.i32()
	p.Position = f.i32()
	p.Unknown1 = f.i32()
	p.Size = f.i32()
	p.Unknown2 = f.i32()
	p.Unknown3 = f.i32()
	p.Effect = f.flag32("effect flag")
	p.Time = f.i32()
	p.Fade = f.flag32("fade flag")
	p.Text = f.str()
	if p.SystemMessageID == -1 {
		if again := f.str(); f.ok() && again != p.Text {
			f.failf("second text %q differs from %q", again, p.Text)
		}
		return p
	}
	for f.ok() && f.r.Len() > 0 {
		p.Params = append(p.Params, f.str())
	}
	return p
}

// RequestQuestAbort asks to abort a quest (RequestQuestAbort) without
// waiting. Mobius answers with QuestList, or not at all for an unknown,
// missing or completed quest.
func (c *GameConn) RequestQuestAbort(ctx context.Context, questID int32) error {
	return c.send(ctx, encodeInt(OpRequestQuestAbort, questID))
}

// AbortQuest aborts a quest from the quest log (RequestQuestAbort) and
// returns the QuestList that answers it. A refused abort ends with the
// context; negative tests use RequestQuestAbort.
func (c *GameConn) AbortQuest(ctx context.Context, questID int32) (QuestList, error) {
	if err := c.RequestQuestAbort(ctx, questID); err != nil {
		return QuestList{}, err
	}
	return Await[QuestList](ctx, c, nil)
}

// Quests asks for the quest list (RequestQuestList) and returns the
// QuestList that answers.
func (c *GameConn) Quests(ctx context.Context) (QuestList, error) {
	if err := c.RequestQuestList(ctx); err != nil {
		return QuestList{}, err
	}
	return Await[QuestList](ctx, c, nil)
}

// TutorialLink clicks a "link" of the tutorial window
// (RequestTutorialLinkHtml) without waiting; "close" closes it.
func (c *GameConn) TutorialLink(ctx context.Context, link string) error {
	return c.send(ctx, encodeString(OpRequestTutorialLinkHtml, link))
}

// TutorialBypass clicks a "bypass" of the tutorial window
// (RequestTutorialPassCmdToServer) without waiting.
func (c *GameConn) TutorialBypass(ctx context.Context, cmd string) error {
	return c.send(ctx, encodeString(OpRequestTutorialPassCmdToServer, cmd))
}

// ClickQuestionMark clicks the tutorial question mark mark
// (RequestTutorialQuestionMark) without waiting.
func (c *GameConn) ClickQuestionMark(ctx context.Context, mark int32) error {
	return c.send(ctx, encodeInt(OpRequestTutorialQuestionMark, mark))
}

// OpenQuestionMark clicks the tutorial question mark mark and returns the
// TutorialShowHtml that answers it. The tutorial answers some marks with
// other packets only, so such a call ends with the context; use
// ClickQuestionMark there.
func (c *GameConn) OpenQuestionMark(ctx context.Context, mark int32) (TutorialShowHtml, error) {
	if err := c.ClickQuestionMark(ctx, mark); err != nil {
		return TutorialShowHtml{}, err
	}
	return Await[TutorialShowHtml](ctx, c, nil)
}

// TutorialClientEvent reports a tutorial client event
// (RequestTutorialClientEvent) without waiting, as the client does after
// TutorialEnableClientEvent.
func (c *GameConn) TutorialClientEvent(ctx context.Context, event int32) error {
	return c.send(ctx, encodeInt(OpRequestTutorialClientEvent, event))
}
