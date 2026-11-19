package clientpackets

import (
	"errors"
	"testing"

	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/session"
)

// B10: the packet used to be parsed before the read error was checked.
// Short input must now be an error, not a zero version.
func TestParseProtocolVersion(t *testing.T) {
	for _, v := range []int32{746, PingVersion, 0} {
		w := packets.NewWriter(4)
		w.WriteInt32(v)
		got, err := ParseProtocolVersion(w.Bytes())
		if err != nil || got.Version != v {
			t.Fatalf("ParseProtocolVersion = %+v, %v; want %d", got, err, v)
		}
	}
	for _, n := range []int{0, 1, 3} {
		if _, err := ParseProtocolVersion(make([]byte, n)); !errors.Is(err, packets.ErrShortPacket) {
			t.Errorf("%d bytes: error = %v, want ErrShortPacket", n, err)
		}
	}
}

// The Interlude client sends more data after the version. Mobius reads only
// the version.
func TestParseProtocolVersionIgnoresTrailingData(t *testing.T) {
	w := packets.NewWriter(260)
	w.WriteInt32(746)
	w.WriteBytes(make([]byte, 256))
	got, err := ParseProtocolVersion(w.Bytes())
	if err != nil || got.Version != 746 {
		t.Fatalf("ParseProtocolVersion = %+v, %v; want 746", got, err)
	}
}

// B5: the name used to come out as raw UTF-16 bytes ("N\x00a\x00...").
func TestParseCharacterCreate(t *testing.T) {
	w := packets.NewWriter(64)
	w.WriteString("Hérőke")
	for v := range int32(12) {
		w.WriteInt32(v + 1)
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

// Values are signed: a hacked client can send negative appearance values,
// which the server must see as negative to refuse them.
func TestParseCharacterCreateNegative(t *testing.T) {
	w := packets.NewWriter(64)
	w.WriteString("A")
	for range 12 {
		w.WriteInt32(-1)
	}
	got, err := ParseCharacterCreate(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if got.Face != -1 || got.HairStyle != -1 || got.HairColor != -1 || got.ClassID != -1 {
		t.Errorf("got %+v, want -1 everywhere", got)
	}
}

func TestCharacterCreateFemale(t *testing.T) {
	for sex, want := range map[int32]bool{0: false, 1: true, 2: true, -1: true} {
		if got := (CharacterCreate{Sex: sex}).Female(); got != want {
			t.Errorf("Sex %d: Female() = %v, want %v", sex, got, want)
		}
	}
}

func TestParseAuthLogin(t *testing.T) {
	w := packets.NewWriter(64)
	w.WriteString("Frostwind")
	for _, k := range []int32{2, 1, 3, -4} {
		w.WriteInt32(k)
	}
	got, err := ParseAuthLogin(w.Bytes())
	if err != nil {
		t.Fatalf("ParseAuthLogin: %v", err)
	}
	want := AuthLogin{Account: "Frostwind", PlayKey2: 2, PlayKey1: 1, LoginKey1: 3, LoginKey2: -4}
	if got != want {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
	wantKeys := session.Keys{LoginOK1: 3, LoginOK2: -4, PlayOK1: 1, PlayOK2: 2}
	if k := got.Keys(); k != wantKeys {
		t.Errorf("Keys() = %+v, want %+v", k, wantKeys)
	}

	full := w.Bytes()
	for n := range len(full) {
		if _, err := ParseAuthLogin(full[:n]); err == nil {
			t.Fatalf("%d of %d bytes: no error", n, len(full))
		}
	}
}

// Garbage must give an error, never a panic or a half-filled packet.
func TestParseGarbage(t *testing.T) {
	inputs := [][]byte{
		nil,
		{0x41},                               // half a code unit
		{0x41, 0x00, 0x42},                   // unterminated name, odd length
		{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, // never terminated
		{0x00, 0x00, 0x01},                   // empty name, then a short field
	}
	for _, in := range inputs {
		if a, err := ParseAuthLogin(in); err == nil {
			t.Errorf("ParseAuthLogin(% x) = %+v, want an error", in, a)
		}
		if c, err := ParseCharacterCreate(in); err == nil {
			t.Errorf("ParseCharacterCreate(% x) = %+v, want an error", in, c)
		}
	}
}

func FuzzParsers(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0xea, 0x02, 0x00, 0x00})
	f.Add([]byte{'A', 0, 0, 0, 1, 0, 0, 0})
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ParseProtocolVersion(data)
		_, _ = ParseAuthLogin(data)
		_, _ = ParseCharacterCreate(data)
	})
}
