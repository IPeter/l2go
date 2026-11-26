package clientpackets

import "testing"

// The sub-opcodes are those of Mobius ExClientPackets (Interlude).
func TestVariationOpcodes(t *testing.T) {
	for _, tt := range []struct {
		name      string
		got, want int
	}{
		{"RequestConfirmTargetItem", ExOpRequestConfirmTargetItem, 0x29},
		{"RequestConfirmRefinerItem", ExOpRequestConfirmRefinerItem, 0x2a},
		{"RequestConfirmGemStone", ExOpRequestConfirmGemStone, 0x2b},
		{"RequestRefine", ExOpRequestRefine, 0x2c},
		{"RequestConfirmCancelItem", ExOpRequestConfirmCancelItem, 0x2d},
		{"RequestRefineCancel", ExOpRequestRefineCancel, 0x2e},
	} {
		if tt.got != tt.want {
			t.Errorf("%s: sub-opcode %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}

// Mobius RequestConfirmTargetItem, RequestConfirmCancelItem and
// RequestRefineCancel.readImpl: d object ID.
func TestParseRequestObjectID(t *testing.T) {
	for _, tt := range []struct {
		name string
		body []byte
		want int32
	}{
		{"object", new(m67Body).d(0x10000123).b, 0x10000123},
		{"trailing data", new(m67Body).d(7).c(1).b, 7},
		{"negative", new(m67Body).d(-1).b, -1},
	} {
		got, err := ParseRequestObjectID(tt.body)
		m67Parsed(t, tt.name, got, err, tt.want)
	}
	m67Truncations(t, "object ID", new(m67Body).d(1).b, ParseRequestObjectID)
}

// Mobius RequestConfirmRefinerItem.readImpl: d target, d refiner.
func TestParseRequestConfirmRefinerItem(t *testing.T) {
	body := new(m67Body).d(0x10000123).d(0x10000124).b
	got, err := ParseRequestConfirmRefinerItem(body)
	m67Parsed(t, "refiner", got, err, RequestConfirmRefinerItem{TargetObjectID: 0x10000123, RefinerObjectID: 0x10000124})
	m67Truncations(t, "RequestConfirmRefinerItem", body, ParseRequestConfirmRefinerItem)
}

// Mobius RequestConfirmGemStone and RequestRefine.readImpl: d target, d
// refiner, d gemstone, d gemstone count.
func TestParseRequestRefine(t *testing.T) {
	body := new(m67Body).d(0x10000123).d(0x10000124).d(0x10000125).d(20).b
	got, err := ParseRequestRefine(body)
	m67Parsed(t, "refine", got, err, RequestRefine{TargetObjectID: 0x10000123, RefinerObjectID: 0x10000124, GemstoneObjectID: 0x10000125, GemstoneCount: 20})
	m67Truncations(t, "RequestRefine", body, ParseRequestRefine)

	// The body comes from ParseExPacket, after the sub-opcode.
	ex, err := ParseExPacket(append([]byte{ExOpRequestRefine, 0x00}, body...))
	if err != nil || ex.SubOpcode != ExOpRequestRefine {
		t.Fatalf("ParseExPacket = %+v, %v", ex, err)
	}
	got, err = ParseRequestRefine(ex.Body)
	m67Parsed(t, "Ex body", got, err, RequestRefine{TargetObjectID: 0x10000123, RefinerObjectID: 0x10000124, GemstoneObjectID: 0x10000125, GemstoneCount: 20})
}

func FuzzParseRequestRefine(f *testing.F) {
	f.Add([]byte{})
	f.Add(new(m67Body).d(1).d(2).d(3).d(4).b)
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := ParseRequestRefine(data)
		if !m67Fuzzed(t, data, got, err, 16, false, nil) {
			return
		}
		want := RequestRefine{TargetObjectID: m67Int(data, 0), RefinerObjectID: m67Int(data, 4), GemstoneObjectID: m67Int(data, 8), GemstoneCount: m67Int(data, 12)}
		if got != want {
			t.Errorf("% x: got %+v, want %+v", data, got, want)
		}
	})
}
