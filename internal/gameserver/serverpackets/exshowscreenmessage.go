package serverpackets

// ExOpShowScreenMessage is the sub-opcode of ExShowScreenMessage.
const ExOpShowScreenMessage = 0x38

// Screen message positions (Mobius ExShowScreenMessage constants).
const (
	ScreenTopLeft      int32 = 1
	ScreenTopCenter    int32 = 2
	ScreenTopRight     int32 = 3
	ScreenMiddleLeft   int32 = 4
	ScreenMiddleCenter int32 = 5
	ScreenMiddleRight  int32 = 6
	ScreenBottomCenter int32 = 7
	ScreenBottomRight  int32 = 8
)

// Screen message types: the Mobius constructors use ScreenMessageText for
// a text and ScreenMessageSystem for a system message.
const (
	ScreenMessageText   int32 = 1
	ScreenMessageSystem int32 = 2
)

// ExShowScreenMessage shows a text or a system message on the screen for a
// while.
type ExShowScreenMessage struct {
	// Type is ScreenMessageText or ScreenMessageSystem.
	Type int32
	// SystemMessageID is the system message, or -1 for a text.
	SystemMessageID int32
	// Position is one of the Screen positions.
	Position int32
	// Unknown1, Unknown2 and Unknown3 are 0 in every Mobius constructor
	// but the raw one.
	Unknown1 int32
	// Size is the font size: 0 normal, 1 small.
	Size     int32
	Unknown2 int32
	Unknown3 int32
	// Effect shows the upper effect (only at ScreenTopCenter).
	Effect bool
	// Time is how long the message stays, in milliseconds.
	Time int32
	// Fade fades the message out.
	Fade bool
	// Text is the text of a text message; empty for a system message
	// (Mobius writes its null text as an empty string).
	Text string
	// Params are the string parameters of a system message.
	Params []string
}

// ScreenText returns the message Mobius Quest.showOnScreenMsg(player, text,
// time) sends: new ExShowScreenMessage(text, time), a text at the top
// center.
func ScreenText(text string, time int32) ExShowScreenMessage {
	return ExShowScreenMessage{Type: ScreenMessageText, SystemMessageID: -1, Position: ScreenTopCenter, Text: text, Time: time}
}

// ScreenSystemMessage returns the message of Mobius
// new ExShowScreenMessage(systemMsg, position, time, params...).
func ScreenSystemMessage(id, position, time int32, params ...string) ExShowScreenMessage {
	return ExShowScreenMessage{Type: ScreenMessageSystem, SystemMessageID: id, Position: position, Time: time, Params: params}
}

// NewExShowScreenMessagePacket builds ExShowScreenMessage (Mobius
// ExShowScreenMessage.writeImpl, without its multilingual branch): d type,
// d system message, d position, d unknown1, d size, d unknown2, d
// unknown3, d effect, d time, d fade, S text, then the text once more for a
// text message (system message -1), or each parameter as S.
func NewExShowScreenMessagePacket(m ExShowScreenMessage) []byte {
	size := 40 + 2*len(m.Text) + 2
	if m.SystemMessageID == -1 {
		size += 2*len(m.Text) + 2
	}
	for _, p := range m.Params {
		size += 2*len(p) + 2
	}
	w := newExWriter(ExOpShowScreenMessage, size)
	w.WriteInt32(m.Type)
	w.WriteInt32(m.SystemMessageID)
	w.WriteInt32(m.Position)
	w.WriteInt32(m.Unknown1)
	w.WriteInt32(m.Size)
	w.WriteInt32(m.Unknown2)
	w.WriteInt32(m.Unknown3)
	putBool32(w, m.Effect)
	w.WriteInt32(m.Time)
	putBool32(w, m.Fade)
	w.WriteString(m.Text)
	if m.SystemMessageID == -1 {
		w.WriteString(m.Text)
	} else {
		for _, p := range m.Params {
			w.WriteString(p)
		}
	}
	return w.Bytes()
}
