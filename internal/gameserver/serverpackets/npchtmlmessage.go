package serverpackets

import (
	"strings"
	"unicode/utf16"

	"github.com/IPeter/l2go/internal/packets"
)

// OpNpcHtmlMessage is the opcode of NpcHtmlMessage.
const OpNpcHtmlMessage = 0x0f

// MaxHTMLLength is the longest page, in UTF-16 code units (Java chars),
// that PrepareHTML keeps. Mobius warns that a longer page crashes the
// client.
const MaxHTMLLength = 8192

// NpcHtmlMessage opens the dialog window with an HTML page.
type NpcHtmlMessage struct {
	// NpcObjectID is the NPC the page belongs to. 0 is a page that is not
	// bound to an NPC: Mobius skips the range check for its links.
	NpcObjectID int32
	// HTML is the page as sent: passed through PrepareHTML, then with its
	// placeholders replaced. Build the player's link cache from this same
	// text.
	HTML string
	// ItemID is the item a page opened from an item belongs to, else 0.
	// Mobius caches the links of item pages in a scope of their own.
	ItemID int32
}

// NewNpcHtmlMessagePacket builds NpcHtmlMessage (Mobius
// NpcHtmlMessage.writeImpl). The page is written as given; PrepareHTML
// applies the Mobius length and wrapping rules.
func NewNpcHtmlMessagePacket(m NpcHtmlMessage) []byte {
	w := packets.NewWriter(11 + 2*len(m.HTML))
	w.WriteUint8(OpNpcHtmlMessage)
	w.WriteInt32(m.NpcObjectID)
	w.WriteString(m.HTML)
	w.WriteInt32(m.ItemID)
	return w.Bytes()
}

// PrepareHTML returns a page the way Mobius AbstractHtmlPacket.setHtml
// stores it:
//   - a page longer than MaxHTMLLength is cut to that length, and not
//     wrapped;
//   - a page without "<html" (case-sensitive) that does not start with
//     `..\L2` is wrapped in "<html><body>" and "</body></html>";
//   - any other page is kept.
//
// Mobius replaces the placeholders (%objectId% and the like) after this
// step, so the page that is sent can be longer than MaxHTMLLength.
func PrepareHTML(html string) string {
	if cut, ok := cutUTF16(html, MaxHTMLLength); ok {
		return cut
	}
	if !strings.Contains(html, "<html") && !strings.HasPrefix(html, `..\L2`) {
		return "<html><body>" + html + "</body></html>"
	}
	return html
}

// cutUTF16 cuts s to at most n UTF-16 code units, like the Java
// s.substring(0, n), and reports whether s was longer. A character that
// takes a surrogate pair across the limit is left out whole, where Java
// keeps its first half. Invalid UTF-8 bytes count as one unit each, as
// packets.Writer sends them.
func cutUTF16(s string, n int) (string, bool) {
	units := 0
	for i, r := range s {
		size := utf16.RuneLen(r)
		if units+size > n {
			return s[:i], true
		}
		units += size
	}
	return s, false
}
