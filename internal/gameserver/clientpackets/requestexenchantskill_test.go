package clientpackets

import "testing"

// Mobius RequestExEnchantSkillInfo.readImpl and RequestExEnchantSkill.readImpl:
// d skill id, d skill level (8 bytes after the 0xD0 0x0006 or 0x0007
// header).
func TestParseRequestExEnchantSkill(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want RequestExEnchantSkill
	}{
		{"triple slash +1", new(m67Body).d(1).d(101).b, RequestExEnchantSkill{SkillID: 1, Level: 101}},
		{"cost route", new(m67Body).d(1).d(141).b, RequestExEnchantSkill{SkillID: 1, Level: 141}},
		{"trailing data", new(m67Body).d(8).d(102).c(0).b, RequestExEnchantSkill{SkillID: 8, Level: 102}},
	}
	for _, tt := range tests {
		got, err := ParseRequestExEnchantSkill(tt.body)
		m67Parsed(t, tt.name, got, err, tt.want)
	}
	m67Truncations(t, "RequestExEnchantSkill", new(m67Body).d(1).d(101).b, ParseRequestExEnchantSkill)

	for _, sub := range []uint16{ExOpRequestExEnchantSkillInfo, ExOpRequestExEnchantSkill} {
		data := append([]byte{byte(sub), 0x00}, new(m67Body).d(1).d(101).b...)
		ex, err := ParseExPacket(data)
		if err != nil {
			t.Fatal(err)
		}
		if ex.SubOpcode != sub {
			t.Fatalf("sub-opcode %#x, want %#x", ex.SubOpcode, sub)
		}
		got, err := ParseRequestExEnchantSkill(ex.Body)
		m67Parsed(t, "Ex body", got, err, RequestExEnchantSkill{SkillID: 1, Level: 101})
	}
	if ExOpRequestExEnchantSkillInfo != 0x06 || ExOpRequestExEnchantSkill != 0x07 {
		t.Error("sub-opcodes differ from Mobius ExClientPackets")
	}
}
