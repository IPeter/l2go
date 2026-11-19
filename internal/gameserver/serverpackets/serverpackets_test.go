package serverpackets

import (
	"bytes"
	"testing"
)

func TestCharCreateFailPacket(t *testing.T) {
	got := NewCharCreateFailPacket(CharCreateFailIncorrectName)
	want := []byte{0x26, 0x04, 0x00, 0x00, 0x00}
	if !bytes.Equal(got, want) {
		t.Errorf("CharCreateFail = % x, want % x", got, want)
	}
}

func TestCharCreateFailReasonByName(t *testing.T) {
	tests := map[string]CharCreateFailReason{
		"creation_failed":      0x00,
		"too_many_characters":  0x01,
		"name_exists":          0x02,
		"name_too_long":        0x03,
		"incorrect_name":       0x04,
		"creation_not_allowed": 0x05,
	}
	for name, want := range tests {
		if got, ok := CharCreateFailReasonByName(name); !ok || got != want {
			t.Errorf("CharCreateFailReasonByName(%q) = %#x, %v; want %#x", name, got, ok, want)
		}
	}
	for _, name := range []string{"", "unknown", "Incorrect_Name"} {
		if _, ok := CharCreateFailReasonByName(name); ok {
			t.Errorf("CharCreateFailReasonByName(%q) found a code", name)
		}
	}
}

func TestCharCreateOkPacket(t *testing.T) {
	if got, want := NewCharCreateOkPacket(), []byte{0x25, 0x01, 0x00, 0x00, 0x00}; !bytes.Equal(got, want) {
		t.Errorf("CharCreateOk = % x, want % x", got, want)
	}
}
