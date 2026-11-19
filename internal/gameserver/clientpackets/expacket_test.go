package clientpackets

import (
	"bytes"
	"testing"
)

// Mobius GamePacketHandler reads the sub-opcode of a 0xD0 packet as an
// unsigned short; the packet's own readImpl reads the rest.

func TestParseExPacket(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want ExPacket
	}{
		{"RequestManorList", []byte{0x08, 0x00}, ExPacket{SubOpcode: ExOpRequestManorList, Body: []byte{}}},
		{"with a body", []byte{0x29, 0x00, 0x01, 0x02}, ExPacket{SubOpcode: 0x29, Body: []byte{0x01, 0x02}}},
		{"unsigned", []byte{0xff, 0xff}, ExPacket{SubOpcode: 0xffff, Body: []byte{}}},
	}
	for _, tt := range tests {
		got, err := ParseExPacket(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}

	for _, in := range [][]byte{nil, {}, {0x08}} {
		got, err := ParseExPacket(in)
		m3Failed(t, "short extended packet", got, err)
	}
}

func FuzzParseExPacket(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0x01})
	f.Add([]byte{0x08, 0x00})
	f.Add([]byte{0x29, 0x00, 0x01, 0x02})
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := ParseExPacket(data)
		if len(data) < 2 {
			m3Failed(t, "short extended packet", got, err)
			return
		}
		if err != nil {
			t.Fatalf("% x: %v", data, err)
		}
		// The sub-opcode is the first short, the body everything after it.
		if want := uint16(data[0]) | uint16(data[1])<<8; got.SubOpcode != want {
			t.Errorf("% x: sub-opcode %#x, want %#x", data, got.SubOpcode, want)
		}
		if !bytes.Equal(got.Body, data[2:]) {
			t.Errorf("% x: body % x", data, got.Body)
		}
	})
}

func TestParseExPacketBody(t *testing.T) {
	data := []byte{0x08, 0x00, 0xaa, 0xbb, 0xcc}
	got, err := ParseExPacket(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Body, data[2:]) {
		t.Errorf("Body = % x, want % x", got.Body, data[2:])
	}
	if req, err := ParseRequestManorList(got.Body); err != nil || req != (RequestManorList{}) {
		t.Errorf("ParseRequestManorList = %+v, %v", req, err)
	}
}
