package serverpackets

import "testing"

// Mobius EquipUpdate.writeImpl: c 0x4B; d change; d object ID; d the slot
// number of the template's body part.
func TestEquipUpdatePacket(t *testing.T) {
	m3Check(t, "a sword put on",
		NewEquipUpdatePacket(EquipUpdate{Change: 1, ObjectID: 0x10000002, BodyPart: 0x80}),
		unhex(t, "4b 01000000 02000010 07000000"))
	m3Check(t, "a two-handed sword",
		NewEquipUpdatePacket(EquipUpdate{Change: 1, ObjectID: 0x10000004, BodyPart: 0x4000}),
		unhex(t, "4b 01000000 04000010 0e000000"))
	m3Check(t, "earrings taken off",
		NewEquipUpdatePacket(EquipUpdate{ObjectID: 0x10000003, BodyPart: 0x6}),
		unhex(t, "4b 00000000 03000010 00000000"))
}

// The slot numbers of the Mobius switch, by body part mask (Mobius
// BodyPart). The masks it does not list are 0.
func TestEquipUpdateSlot(t *testing.T) {
	tests := []struct {
		part       string
		mask, want int32
	}{
		{"none", 0, 0},
		{"underwear", 0x0001, 0},
		{"rear", 0x0002, 0x02},
		{"lear", 0x0004, 0x01},
		{"rear;lear", 0x0006, 0},
		{"neck", 0x0008, 0x03},
		{"rfinger", 0x0010, 0x04},
		{"lfinger", 0x0020, 0x05},
		{"rfinger;lfinger", 0x0030, 0},
		{"head", 0x0040, 0x06},
		{"rhand", 0x0080, 0x07},
		{"lhand", 0x0100, 0x08},
		{"gloves", 0x0200, 0x09},
		{"chest", 0x0400, 0x0a},
		{"legs", 0x0800, 0x0b},
		{"feet", 0x1000, 0x0c},
		{"back", 0x2000, 0x0d},
		{"lrhand", 0x4000, 0x0e},
		{"fullarmor", 0x8000, 0},
		{"hair", 0x010000, 0x0f},
		{"alldress", 0x020000, 0},
		{"hair2", 0x040000, 0},
		{"hairall", 0x080000, 0},
		{"wolf", -100, 0},
	}
	for _, tt := range tests {
		if got := equipUpdateSlot(tt.mask); got != tt.want {
			t.Errorf("%s (mask %#x): slot %#x, want %#x", tt.part, tt.mask, got, tt.want)
		}
	}
}
