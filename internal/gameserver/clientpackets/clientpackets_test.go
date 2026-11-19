package clientpackets

import (
	"errors"
	"testing"

	"github.com/IPeter/l2go/internal/packets"
)

// B10: the packet used to be parsed before the read error was checked.
// Short input must now be an error, not a zero version.
func TestParseProtocolVersion(t *testing.T) {
	w := packets.NewWriter(4)
	w.WriteUint32(419)
	got, err := ParseProtocolVersion(w.Bytes())
	if err != nil || got.Version != 419 {
		t.Fatalf("ParseProtocolVersion = %+v, %v; want 419", got, err)
	}
	for _, n := range []int{0, 3} {
		if _, err := ParseProtocolVersion(make([]byte, n)); !errors.Is(err, packets.ErrShortPacket) {
			t.Errorf("%d bytes: error = %v, want ErrShortPacket", n, err)
		}
	}
}

// B5: the name used to come out as raw UTF-16 bytes ("N\x00a\x00...").
func TestParseCharacterCreate(t *testing.T) {
	w := packets.NewWriter(64)
	w.WriteString("Hérőke")
	for v := range uint32(12) {
		w.WriteUint32(v + 1)
	}
	got, err := ParseCharacterCreate(w.Bytes())
	if err != nil {
		t.Fatalf("ParseCharacterCreate: %v", err)
	}
	want := CharacterCreate{
		Name: "Hérőke", Race: 1, Sex: 2, ClassID: 3,
		INT: 4, STR: 5, CON: 6, MEN: 7, DEX: 8, WIT: 9,
		HairStyle: 10, HairColor: 11, Face: 12,
	}
	if got != want {
		t.Errorf("got %+v\nwant %+v", got, want)
	}

	// Every truncation is an error.
	full := w.Bytes()
	for n := range len(full) {
		if _, err := ParseCharacterCreate(full[:n]); err == nil {
			t.Fatalf("%d of %d bytes: no error", n, len(full))
		}
	}
}

func TestParseAuthLogin(t *testing.T) {
	w := packets.NewWriter(64)
	w.WriteString("Frostwind")
	for _, k := range []uint32{2, 1, 3, 4} {
		w.WriteUint32(k)
	}
	got, err := ParseAuthLogin(w.Bytes())
	if err != nil {
		t.Fatalf("ParseAuthLogin: %v", err)
	}
	want := AuthLogin{Account: "Frostwind", PlayKey2: 2, PlayKey1: 1, LoginKey1: 3, LoginKey2: 4}
	if got != want {
		t.Errorf("got %+v\nwant %+v", got, want)
	}

	full := w.Bytes()
	for n := range len(full) {
		if _, err := ParseAuthLogin(full[:n]); err == nil {
			t.Fatalf("%d of %d bytes: no error", n, len(full))
		}
	}
}

func FuzzParsers(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0xa3, 0x01, 0x00, 0x00})
	f.Add([]byte{'A', 0, 0, 0, 1, 0, 0, 0})
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ParseProtocolVersion(data)
		_, _ = ParseAuthLogin(data)
		_, _ = ParseCharacterCreate(data)
	})
}
