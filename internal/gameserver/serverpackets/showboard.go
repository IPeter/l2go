package serverpackets

import (
	"strings"

	"github.com/IPeter/l2go/internal/packets"
)

// The community board packets (Plan 14 M11o): ShowBoard opens the board
// (Alt+B) with a page, ExMailArrived blinks the mail icon.

// OpShowBoard is the opcode of ShowBoard.
const OpShowBoard = 0x6e

// ExOpMailArrived is the sub-opcode of ExMailArrived (FE 2D).
const ExOpMailArrived = 0x2d

// BoardTabs are the bypasses of the board's tab buttons, in the order
// ShowBoard writes them: home, favorites, region, clan, memo, mail, friends
// and "add to favorites" (Mobius ShowBoard.writeImpl).
var BoardTabs = [8]string{
	"bypass _bbshome",
	"bypass _bbsgetfav",
	"bypass _bbsloc",
	"bypass _bbsclan",
	"bypass _bbsmemo",
	"bypass _bbsmail",
	"bypass _bbsfriends",
	"bypass bbs_add_fav",
}

// boardSeparator ends the part ID of a ShowBoard content and each value of
// a multiedit fill.
const boardSeparator = "\u0008"

// BoardContent is the content of a ShowBoard that carries a page part:
// the part ID ("101", "102", "103", or "1001" for a page with a multiedit
// field) and the HTML (Mobius ShowBoard(String, String)). Mobius writes a
// missing part with the Java string "null"; pass that literally.
func BoardContent(id, html string) string { return id + boardSeparator + html }

// BoardFillContent is the content of a ShowBoard that fills the multiedit
// field of the page sent before it (Mobius ShowBoard(List<String>)): "1002",
// then each value, each followed by the separator.
func BoardFillContent(values []string) string {
	var b strings.Builder
	b.WriteString("1002" + boardSeparator)
	for _, v := range values {
		b.WriteString(v)
		b.WriteString(boardSeparator)
	}
	return b.String()
}

// NewShowBoardPacket builds ShowBoard (Mobius ShowBoard.writeImpl): c 1, the
// eight tab bypasses as S, then S content (BoardContent or
// BoardFillContent).
func NewShowBoardPacket(content string) []byte {
	return showBoard(1, content)
}

// NewHideBoardPacket builds the ShowBoard that closes the board (Mobius
// ShowBoard()): c 0, the tab bypasses, an empty content.
func NewHideBoardPacket() []byte {
	return showBoard(0, "")
}

func showBoard(show uint8, content string) []byte {
	size := 2 + 2*len(content)
	for _, tab := range BoardTabs {
		size += 2 + 2*len(tab)
	}
	w := packets.NewWriter(1 + size)
	w.WriteUint8(OpShowBoard)
	w.WriteUint8(show)
	for _, tab := range BoardTabs {
		w.WriteString(tab)
	}
	w.WriteString(content)
	return w.Bytes()
}

// NewExMailArrivedPacket builds ExMailArrived (Mobius ExMailArrived): no
// fields, it makes the client blink the mail icon.
func NewExMailArrivedPacket() []byte {
	return newExWriter(ExOpMailArrived, 0).Bytes()
}
