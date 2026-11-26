package testclient

import (
	"bytes"
	"reflect"
	"testing"
)

var testBoardTabs = [8]string{"bypass _bbshome", "bypass _bbsgetfav", "bypass _bbsloc", "bypass _bbsclan",
	"bypass _bbsmemo", "bypass _bbsmail", "bypass _bbsfriends", "bypass bbs_add_fav"}

func showBoardWire(show byte, content string) []byte {
	w := msg(0x6e).u8(show)
	for _, tab := range testBoardTabs {
		w.str(tab)
	}
	return w.str(content).b
}

// The board packets decode from the Mobius writeImpl layouts.
func TestDecodeBoardPackets(t *testing.T) {
	for _, tt := range []struct {
		name string
		raw  []byte
		want Packet
	}{
		// ShowBoard: c show, 8 × S tab bypass, S content.
		{"ShowBoard", showBoardWire(1, "101\u0008<html>x</html>"),
			ShowBoard{Show: true, Tabs: testBoardTabs, Content: "101\u0008<html>x</html>"}},
		{"ShowBoard", showBoardWire(0, ""), ShowBoard{Tabs: testBoardTabs}},
		// ExMailArrived: nothing after the sub-opcode.
		{"ExMailArrived", msg(0xfe).u16(0x2d).b, ExMailArrived{}},
	} {
		got, err := DecodeServerPacket(tt.raw)
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: got %+v, want %+v", tt.name, got, tt.want)
		}
		if got.PacketName() != tt.name {
			t.Errorf("%s: named %q", tt.name, got.PacketName())
		}
	}
	if _, err := DecodeServerPacket(showBoardWire(1, "x")[:20]); err == nil {
		t.Error("a short ShowBoard decoded")
	}
}

// BoardPage joins the parts of a page and drops the "null" of a missing
// part (Mobius ShowBoard(null, "102")).
func TestBoardPage(t *testing.T) {
	ps := []Packet{
		ShowBoard{Show: true, Content: "101\u0008old"},
		ShowBoard{Show: true, Content: "101\u0008<html>a"},
		ShowBoard{Show: true, Content: "102\u0008b</html>"},
		ShowBoard{Show: true, Content: "103\u0008null"},
	}
	if html, ok := BoardPage(ps); !ok || html != "<html>ab</html>" {
		t.Errorf("BoardPage = %q, %v", html, ok)
	}
	if _, ok := BoardPage(nil); ok {
		t.Error("BoardPage of nothing is ok")
	}
	if html, ok := BoardPage([]Packet{ShowBoard{Content: "1001\u0008<html>edit</html>"}, ShowBoard{Content: "1002\u00080\u0008"}}); !ok || html != "<html>edit</html>" {
		t.Errorf("BoardPage(1001) = %q, %v", html, ok)
	}
}

func TestBoardRequests(t *testing.T) {
	got := encodeRequestBBSwrite("Mail", [5]string{"Send", "0", "Bob", "Hi", "Text"})
	want := msg(0x22).str("Mail").str("Send").str("0").str("Bob").str("Hi").str("Text").b
	if !bytes.Equal(got, want) {
		t.Errorf("RequestBBSwrite = % x, want % x", got, want)
	}
	if name := clientPacketName(got); name != "RequestBBSwrite" {
		t.Errorf("named %q", name)
	}
	if !bytes.Equal(encodeInts(OpRequestShowBoard, 0), msg(0x57).i32(0).b) {
		t.Error("RequestShowBoard layout")
	}
	if name := clientPacketName(msg(0x57).i32(0).b); name != "RequestShowBoard" {
		t.Errorf("named %q", name)
	}
}
