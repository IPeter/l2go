package clientpackets

import (
	"testing"

	"github.com/IPeter/l2go/internal/packets"
)

func TestParseDlgAnswer(t *testing.T) {
	w := packets.NewWriter(12)
	w.WriteInt32(1842)
	w.WriteInt32(1)
	w.WriteInt32(268435460)
	got, err := ParseDlgAnswer(w.Bytes())
	if err != nil || got != (DlgAnswer{MessageID: 1842, Answer: 1, RequesterID: 268435460}) {
		t.Fatalf("ParseDlgAnswer = %+v, %v", got, err)
	}
	if _, err := ParseDlgAnswer(w.Bytes()[:8]); err == nil {
		t.Fatal("a short DlgAnswer parsed")
	}
}
