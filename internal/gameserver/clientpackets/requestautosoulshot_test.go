package clientpackets

import "testing"

// Mobius RequestAutoSoulShot.readImpl: d item ID, d type (1 on, 0 off)
// (8 bytes after the 0xD0 0x0005 header).
func TestParseRequestAutoSoulShot(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want RequestAutoSoulShot
	}{
		{"on", new(m67Body).d(1835).d(AutoSoulShotOn).b, RequestAutoSoulShot{ItemID: 1835, Type: 1}},
		{"off", new(m67Body).d(3947).d(AutoSoulShotOff).b, RequestAutoSoulShot{ItemID: 3947, Type: 0}},
		// Not checked here: Mobius does nothing for other types.
		{"type 2", new(m67Body).d(1835).d(2).b, RequestAutoSoulShot{ItemID: 1835, Type: 2}},
		{"trailing data", new(m67Body).d(5790).d(1).c(0).b, RequestAutoSoulShot{ItemID: 5790, Type: 1}},
		{"garbage", []byte{0xff, 0xff, 0xff, 0xff, 0xfe, 0xff, 0xff, 0xff}, RequestAutoSoulShot{ItemID: -1, Type: -2}},
	}
	for _, tt := range tests {
		got, err := ParseRequestAutoSoulShot(tt.body)
		m67Parsed(t, tt.name, got, err, tt.want)
	}
	m67Truncations(t, "RequestAutoSoulShot", new(m67Body).d(1835).d(1).b, ParseRequestAutoSoulShot)
}

// The body comes from ParseExPacket, after the sub-opcode.
func TestParseRequestAutoSoulShotEx(t *testing.T) {
	data := append([]byte{0x05, 0x00}, new(m67Body).d(1835).d(1).b...)
	ex, err := ParseExPacket(data)
	if err != nil {
		t.Fatal(err)
	}
	if ex.SubOpcode != ExOpRequestAutoSoulShot {
		t.Fatalf("sub-opcode %#x, want %#x", ex.SubOpcode, ExOpRequestAutoSoulShot)
	}
	got, err := ParseRequestAutoSoulShot(ex.Body)
	m67Parsed(t, "Ex body", got, err, RequestAutoSoulShot{ItemID: 1835, Type: AutoSoulShotOn})
}

func FuzzParseRequestAutoSoulShot(f *testing.F) {
	f.Add([]byte{})
	f.Add(new(m67Body).d(1835).d(1).b)
	f.Add(new(m67Body).d(3947).d(0).c(7).b)
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := ParseRequestAutoSoulShot(data)
		if !m67Fuzzed(t, data, got, err, 8, false, nil) {
			return
		}
		if want := (RequestAutoSoulShot{ItemID: m67Int(data, 0), Type: m67Int(data, 4)}); got != want {
			t.Errorf("% x: got %+v, want %+v", data, got, want)
		}
	})
}
