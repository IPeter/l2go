package clientpackets

import (
	"testing"

	"github.com/IPeter/l2go/internal/packets"
)

func TestBBSOpcodes(t *testing.T) {
	if OpRequestShowBoard != 0x57 || OpRequestBBSwrite != 0x22 {
		t.Errorf("opcodes %#x %#x, want 0x57 0x22", OpRequestShowBoard, OpRequestBBSwrite)
	}
}

// Mobius RequestShowBoard.readImpl: d unused.
func TestParseRequestShowBoard(t *testing.T) {
	w := packets.NewWriter(4)
	w.WriteInt32(1)
	req, err := ParseRequestShowBoard(w.Bytes())
	if err != nil || req.Unused != 1 {
		t.Errorf("ParseRequestShowBoard = %+v, %v", req, err)
	}
	if _, err := ParseRequestShowBoard([]byte{1, 0}); err == nil {
		t.Error("a short RequestShowBoard parsed")
	}
}

// Mobius RequestBBSwrite.readImpl: S url, S arg1 ... S arg5.
func TestParseRequestBBSwrite(t *testing.T) {
	w := packets.NewWriter(64)
	for _, s := range []string{"Mail", "Send", "%postId%", "Bob;Ann", "Hello", "Line 1\nLine 2"} {
		w.WriteString(s)
	}
	req, err := ParseRequestBBSwrite(w.Bytes())
	want := RequestBBSwrite{URL: "Mail", Arg1: "Send", Arg2: "%postId%", Arg3: "Bob;Ann", Arg4: "Hello", Arg5: "Line 1\nLine 2"}
	if err != nil || req != want {
		t.Errorf("ParseRequestBBSwrite = %+v, %v; want %+v", req, err, want)
	}
	if _, err := ParseRequestBBSwrite(w.Bytes()[:10]); err == nil {
		t.Error("a short RequestBBSwrite parsed")
	}
}
