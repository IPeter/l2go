package serverpackets

import "testing"

// Mobius ShowBoard.writeImpl: c 0x6E, c show (1), then S for each of the
// eight tab bypasses, then S content; ShowBoard(html, id) makes the content
// id + U+0008 + html, so a missing part is "102\bnull".
func TestShowBoardPacket(t *testing.T) {
	tabs := func(g *m3Golden) *m3Golden {
		return g.s("bypass _bbshome").s("bypass _bbsgetfav").s("bypass _bbsloc").s("bypass _bbsclan").
			s("bypass _bbsmemo").s("bypass _bbsmail").s("bypass _bbsfriends").s("bypass bbs_add_fav")
	}
	var g m3Golden
	tabs(g.c(0x6e).c(1)).s("101\u0008<html><body>Hi</body></html>")
	m3Check(t, "ShowBoard 101", NewShowBoardPacket(BoardContent("101", "<html><body>Hi</body></html>")), g.b)

	var null m3Golden
	tabs(null.c(0x6e).c(1)).s("102\u0008null")
	m3Check(t, "ShowBoard 102 null", NewShowBoardPacket(BoardContent("102", "null")), null.b)

	var hide m3Golden
	tabs(hide.c(0x6e).c(0)).s("")
	m3Check(t, "ShowBoard hide", NewHideBoardPacket(), hide.b)
}

// Mobius ShowBoard(List<String>): "1002" + U+0008, then every value
// followed by U+0008 (HtmlUtil.fillMultiEditContent).
func TestBoardFillContent(t *testing.T) {
	got := BoardFillContent([]string{"0", "Alice", "line1\nline2", "0"})
	if want := "1002\u00080\u0008Alice\u0008line1\nline2\u00080\u0008"; got != want {
		t.Errorf("BoardFillContent = %q, want %q", got, want)
	}
	if got := BoardFillContent(nil); got != "1002\u0008" {
		t.Errorf("BoardFillContent(nil) = %q", got)
	}
}

// Mobius ExMailArrived.writeImpl: c 0xFE, h 0x2D and nothing else.
func TestExMailArrivedPacket(t *testing.T) {
	m3Check(t, "ExMailArrived", NewExMailArrivedPacket(), unhex(t, "fe 2d00"))
}
