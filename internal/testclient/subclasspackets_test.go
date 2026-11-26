package testclient

import (
	"encoding/binary"
	"testing"
)

// mobiusExEnchantSkillList is ExEnchantSkillList.writeImpl: the Ex header,
// d count, then d id, d next level, d sp, q exp.
func mobiusExEnchantSkillList(skills ...EnchantSkill) []byte {
	w := msg(0xfe).u16(0x17).i32(int32(len(skills)))
	for _, s := range skills {
		w.i32(s.ID).i32(s.NextLevel).i32(s.SP).i64(s.Exp)
	}
	return w.b
}

// mobiusExEnchantSkillInfo is ExEnchantSkillInfo.writeImpl: the Ex header,
// d id, d level, d sp, q exp, d rate, d count, then d type, d id, d count,
// d unknown.
func mobiusExEnchantSkillInfo(i ExEnchantSkillInfo) []byte {
	w := msg(0xfe).u16(0x18).i32(i.SkillID).i32(i.Level).i32(i.SPCost).i64(i.ExpCost).i32(i.Rate).i32(int32(len(i.Requirements)))
	for _, r := range i.Requirements {
		w.i32(r.Type).i32(r.ItemID).i32(r.Count).i32(r.Unknown)
	}
	return w.b
}

func TestDecodeEnchantSkillPackets(t *testing.T) {
	list := ExEnchantSkillList{Skills: []EnchantSkill{
		{ID: 1, NextLevel: 101, SP: 306000, Exp: 3060000},
		{ID: 1, NextLevel: 141, SP: 306000, Exp: 3060000},
	}}
	info := ExEnchantSkillInfo{SkillID: 1, Level: 101, SPCost: 306000, ExpCost: 3060000, Rate: 82,
		Requirements: []EnchantRequirement{{Type: 4, ItemID: 6622, Count: 1}}}
	tests := []struct {
		name string
		body []byte
		want Packet
	}{
		{"ExEnchantSkillList", mobiusExEnchantSkillList(list.Skills...), list},
		{"ExEnchantSkillList empty", mobiusExEnchantSkillList(), ExEnchantSkillList{}},
		{"ExEnchantSkillInfo", mobiusExEnchantSkillInfo(info), info},
		{"ExEnchantSkillInfo without a book", mobiusExEnchantSkillInfo(ExEnchantSkillInfo{SkillID: 1, Level: 102, Rate: 80}),
			ExEnchantSkillInfo{SkillID: 1, Level: 102, Rate: 80}},
	}
	for _, tt := range tests {
		checkDecode(t, tt.name, tt.body, tt.want)
	}
	if s, ok := list.Find(1); !ok || s.NextLevel != 101 {
		t.Errorf("Find(1) = %+v, %v", s, ok)
	}
	// Sizes counted from writeImpl: 2 + 4 + 20 per skill; 2 + 28 + 16 per item.
	if n := len(mobiusExEnchantSkillList(EnchantSkill{})) - 1; n != 2+4+20 {
		t.Errorf("ExEnchantSkillList of 1 is %d bytes", n)
	}
	if n := len(mobiusExEnchantSkillInfo(ExEnchantSkillInfo{Requirements: []EnchantRequirement{{}}})) - 1; n != 2+28+16 {
		t.Errorf("ExEnchantSkillInfo of 1 is %d bytes", n)
	}
}

func TestEncodeEnchantSkillRequest(t *testing.T) {
	b := encodeEnchantSkillRequest(ExOpRequestExEnchantSkill, 1, 101)
	if len(b) != 11 || b[0] != OpExClient || binary.LittleEndian.Uint16(b[1:]) != 0x07 ||
		binary.LittleEndian.Uint32(b[3:]) != 1 || binary.LittleEndian.Uint32(b[7:]) != 101 {
		t.Errorf("RequestExEnchantSkill = % x", b)
	}
	if b := encodeEnchantSkillRequest(ExOpRequestExEnchantSkillInfo, 1, 141); binary.LittleEndian.Uint16(b[1:]) != 0x06 {
		t.Errorf("RequestExEnchantSkillInfo = % x", b)
	}
}
