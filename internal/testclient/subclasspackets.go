package testclient

import (
	"context"

	"github.com/IPeter/l2go/internal/packets"
)

// Skill enchanting (M11e): the trainer's enchant window and its requests.

// Sub-opcodes of the skill enchant packets (Mobius ExClientPackets and
// ServerPackets).
const (
	ExOpRequestExEnchantSkillInfo = 0x06
	ExOpRequestExEnchantSkill     = 0x07
	ExOpEnchantSkillList          = 0x17
	ExOpEnchantSkillInfo          = 0x18
)

// EnchantSkill is one skill of ExEnchantSkillList.
type EnchantSkill struct {
	ID, NextLevel, SP int32
	Exp               int64
}

// ExEnchantSkillList is the trainer's skill enchant window (Ex 0x17).
type ExEnchantSkillList struct {
	Skills []EnchantSkill
}

// PacketName implements Packet.
func (ExEnchantSkillList) PacketName() string { return "ExEnchantSkillList" }

// Find returns the entry of a skill.
func (l ExEnchantSkillList) Find(id int32) (EnchantSkill, bool) {
	for _, s := range l.Skills {
		if s.ID == id {
			return s, true
		}
	}
	return EnchantSkill{}, false
}

func readExEnchantSkillList(f *fields) Packet {
	var p ExEnchantSkillList
	n := f.count32("skill count")
	for i := 0; i < n && f.ok(); i++ {
		p.Skills = append(p.Skills, EnchantSkill{ID: f.i32(), NextLevel: f.i32(), SP: f.i32(), Exp: f.i64()})
	}
	return p
}

// EnchantRequirement is an item ExEnchantSkillInfo asks for.
type EnchantRequirement struct {
	Type, ItemID, Count, Unknown int32
}

// ExEnchantSkillInfo is the cost and chance of an enchant level (Ex 0x18).
type ExEnchantSkillInfo struct {
	SkillID, Level, SPCost int32
	ExpCost                int64
	Rate                   int32
	Requirements           []EnchantRequirement
}

// PacketName implements Packet.
func (ExEnchantSkillInfo) PacketName() string { return "ExEnchantSkillInfo" }

func readExEnchantSkillInfo(f *fields) Packet {
	p := ExEnchantSkillInfo{SkillID: f.i32(), Level: f.i32(), SPCost: f.i32(), ExpCost: f.i64(), Rate: f.i32()}
	n := f.count32("requirement count")
	for i := 0; i < n && f.ok(); i++ {
		p.Requirements = append(p.Requirements, EnchantRequirement{Type: f.i32(), ItemID: f.i32(), Count: f.i32(), Unknown: f.i32()})
	}
	return p
}

// encodeEnchantSkillRequest builds RequestExEnchantSkillInfo or
// RequestExEnchantSkill: the Ex header, d skill, d level.
func encodeEnchantSkillRequest(sub uint16, skillID, level int32) []byte {
	w := packets.NewWriter(11)
	w.WriteUint8(OpExClient)
	w.WriteUint16(sub)
	w.WriteInt32(skillID)
	w.WriteInt32(level)
	return w.Bytes()
}

// EnchantSkillInfo asks the trainer for the cost and chance of an enchant
// level (RequestExEnchantSkillInfo) and returns the answer. Mobius answers
// nothing for a level it does not offer, so such a call ends with the
// context.
func (c *GameConn) EnchantSkillInfo(ctx context.Context, skillID, level int32) (ExEnchantSkillInfo, error) {
	if err := c.send(ctx, encodeEnchantSkillRequest(ExOpRequestExEnchantSkillInfo, skillID, level)); err != nil {
		return ExEnchantSkillInfo{}, err
	}
	return Await(ctx, c, func(i ExEnchantSkillInfo) bool { return i.SkillID == skillID && i.Level == level })
}

// EnchantSkill enchants a skill at the trainer (RequestExEnchantSkill) and
// returns the enchant window Mobius sends again afterwards.
func (c *GameConn) EnchantSkill(ctx context.Context, skillID, level int32) (ExEnchantSkillList, error) {
	if err := c.send(ctx, encodeEnchantSkillRequest(ExOpRequestExEnchantSkill, skillID, level)); err != nil {
		return ExEnchantSkillList{}, err
	}
	return Await(ctx, c, func(ExEnchantSkillList) bool { return true })
}
