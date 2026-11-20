package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// Opcodes of the tutorial packets (Mobius ServerPackets 0xA0-0xA3).
const (
	OpTutorialShowHtml          = 0xa0
	OpTutorialShowQuestionMark  = 0xa1
	OpTutorialEnableClientEvent = 0xa2
	OpTutorialCloseHtml         = 0xa3
)

// NewTutorialShowHtmlPacket builds TutorialShowHtml (Mobius
// TutorialShowHtml.writeImpl): S page. It opens the tutorial window; its
// links come back as RequestTutorialLinkHtml and its bypasses as
// RequestTutorialPassCmdToServer. The page is written as given; Mobius
// prepares it like an NPC page (see PrepareHTML).
func NewTutorialShowHtmlPacket(html string) []byte {
	w := packets.NewWriter(3 + 2*len(html))
	w.WriteUint8(OpTutorialShowHtml)
	w.WriteString(html)
	return w.Bytes()
}

// NewTutorialShowQuestionMarkPacket builds TutorialShowQuestionMark
// (Mobius TutorialShowQuestionMark.writeImpl): d mark. The client shows a
// blinking question mark; clicking it sends RequestTutorialQuestionMark
// with the same number.
func NewTutorialShowQuestionMarkPacket(mark int32) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpTutorialShowQuestionMark)
	w.WriteInt32(mark)
	return w.Bytes()
}

// NewTutorialEnableClientEventPacket builds TutorialEnableClientEvent
// (Mobius TutorialEnableClientEvent.writeImpl): d event. The client then
// reports that event with RequestTutorialClientEvent; 0 turns the events
// off.
func NewTutorialEnableClientEventPacket(event int32) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpTutorialEnableClientEvent)
	w.WriteInt32(event)
	return w.Bytes()
}

// NewTutorialCloseHtmlPacket builds TutorialCloseHtml (Mobius
// TutorialCloseHtml.writeImpl): the opcode alone. It closes the tutorial
// window; Mobius also forgets the links of its page.
func NewTutorialCloseHtmlPacket() []byte { return []byte{OpTutorialCloseHtml} }
