package testclient

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// The Mobius server packets of Milestone 7, transcribed from their
// writeImpl line by line.

// mobiusMagicSkillUse is MagicSkillUse.writeImpl: a set critical flag is
// followed by a zero short.
func mobiusMagicSkillUse(m MagicSkillUse) []byte {
	w := msg(0x48)
	w.i32(m.CasterID).i32(m.TargetID)
	w.i32(m.SkillID).i32(m.Level)
	w.i32(m.HitTime).i32(m.ReuseDelay)
	w.i32(m.X).i32(m.Y).i32(m.Z)
	if m.Critical {
		w.i32(1).u16(m.Reserved)
	} else {
		w.i32(0)
	}
	w.i32(m.TargetX).i32(m.TargetY).i32(m.TargetZ)
	return w.b
}

// mobiusMagicSkillLaunched is MagicSkillLaunched.writeImpl.
func mobiusMagicSkillLaunched(m MagicSkillLaunched) []byte {
	w := msg(0x76).i32(m.CasterID).i32(m.SkillID).i32(m.Level).i32(int32(len(m.Targets)))
	for _, id := range m.Targets {
		w.i32(id)
	}
	return w.b
}

// mobiusSetupGauge is SetupGauge.writeImpl: no object ID in Interlude.
func mobiusSetupGauge(g SetupGauge) []byte {
	return msg(0x6d).i32(g.Color).i32(g.Time).i32(g.MaxTime).b
}

// mobiusAcquireSkillList is AcquireSkillList.writeImpl: d type (ordinal),
// d count, then d id, d next level, d max level, d SP, d requirements.
func mobiusAcquireSkillList(l AcquireSkillList) []byte {
	w := msg(0x8a).i32(l.Type).i32(int32(len(l.Skills)))
	for _, s := range l.Skills {
		w.i32(s.ID).i32(s.NextLevel).i32(s.MaxLevel).i32(s.SPCost).i32(s.Requirements)
	}
	return w.b
}

// mobiusAcquireSkillInfo is AcquireSkillInfo.writeImpl: d id, d level, d
// SP, d type, d count, then per item d 99, d item, q count, d 50.
func mobiusAcquireSkillInfo(a AcquireSkillInfo) []byte {
	w := msg(0x8b).i32(a.SkillID).i32(a.Level).i32(a.SPCost).i32(a.Type).i32(int32(len(a.Requirements)))
	for _, r := range a.Requirements {
		w.i32(r.Type).i32(r.ItemID).i64(r.Count).i32(r.Unknown)
	}
	return w.b
}

// mobiusShortcutRegister is ShortcutRegister.writeImpl: d type, d slot,
// then d id for an item, action, macro or recipe, d id, d level, c 0 for a
// skill, and a d 1 for every type.
func mobiusShortcutRegister(typ, slot, id, level int32) []byte {
	w := msg(0x44).i32(typ).i32(slot)
	switch typ {
	case ShortcutItem, ShortcutAction, ShortcutMacro, ShortcutRecipe:
		w.i32(id)
	case ShortcutSkill:
		w.i32(id).i32(level).u8(0) // C5
	}
	return w.i32(1).b
}

// mobiusExAutoSoulShot is ExAutoSoulShot.writeImpl: the Ex header, d
// item, d type.
func mobiusExAutoSoulShot(itemID int32, on bool) []byte {
	return msg(0xfe).u16(0x12).i32(itemID).i32(b2i(on)).b
}

// mobiusPartySpelled is PartySpelled.writeImpl: d kind (2 servitor, 1 pet,
// 0 player), d object, d count, then d skill, h level, d seconds.
func mobiusPartySpelled(p PartySpelled) []byte {
	w := msg(0xee).i32(p.Kind).i32(p.ObjectID).i32(int32(len(p.Effects)))
	for _, e := range p.Effects {
		w.i32(e.SkillID).u16(e.Level).i32(e.Duration)
	}
	return w.b
}

// mobiusShortBuffStatusUpdate is ShortBuffStatusUpdate.writeImpl.
func mobiusShortBuffStatusUpdate(s ShortBuffStatusUpdate) []byte {
	return msg(0xf4).i32(s.SkillID).i32(s.Level).i32(s.Duration).b
}

// testWindStrike is a Wind Strike cast on a monster.
func testWindStrike() MagicSkillUse {
	return MagicSkillUse{CasterID: 0x10000001, TargetID: 0x60000005, SkillID: 1177, Level: 2, HitTime: 1080, ReuseDelay: 4000,
		X: -71338, Y: 258271, Z: -3104, TargetX: -71300, TargetY: 258300, TargetZ: -3104}
}

func TestDecodeSkillPackets(t *testing.T) {
	critical := testWindStrike()
	critical.Critical = true
	info := AcquireSkillInfo{SkillID: 1177, Level: 2, SPCost: 50, Type: AcquireSkillClass,
		Requirements: []SkillRequirement{{Type: 99, ItemID: 1050, Count: 1, Unknown: 50}, {Type: 99, ItemID: 57, Count: 1 << 33, Unknown: 50}}}

	tests := []struct {
		name string
		body []byte
		want Packet
	}{
		{"MagicSkillUse", mobiusMagicSkillUse(testWindStrike()), testWindStrike()},
		{"MagicSkillUse critical", mobiusMagicSkillUse(critical), critical},
		{"MagicSkillLaunched", mobiusMagicSkillLaunched(MagicSkillLaunched{CasterID: 0x10000001, SkillID: 1177, Level: 2,
			Targets: []int32{0x60000005, 0x60000006}}),
			MagicSkillLaunched{CasterID: 0x10000001, SkillID: 1177, Level: 2, Targets: []int32{0x60000005, 0x60000006}}},
		{"MagicSkillLaunched without targets", mobiusMagicSkillLaunched(MagicSkillLaunched{CasterID: 1, SkillID: 2, Level: 3}),
			MagicSkillLaunched{CasterID: 1, SkillID: 2, Level: 3}},
		{"MagicSkillCanceled", msg(0x49).i32(0x10000001).b, MagicSkillCanceled{ObjectID: 0x10000001}},
		{"SetupGauge", mobiusSetupGauge(SetupGauge{Color: GaugeBlue, Time: 1080, MaxTime: 1080}),
			SetupGauge{Color: GaugeBlue, Time: 1080, MaxTime: 1080}},
		{"AcquireSkillList", mobiusAcquireSkillList(AcquireSkillList{Type: AcquireSkillClass, Skills: []LearnableSkill{
			{ID: 1177, NextLevel: 2, MaxLevel: 2, SPCost: 50}, {ID: 1184, NextLevel: 1, MaxLevel: 1, SPCost: 70}}}),
			AcquireSkillList{Type: AcquireSkillClass, Skills: []LearnableSkill{
				{ID: 1177, NextLevel: 2, MaxLevel: 2, SPCost: 50}, {ID: 1184, NextLevel: 1, MaxLevel: 1, SPCost: 70}}}},
		{"AcquireSkillInfo", mobiusAcquireSkillInfo(info), info},
		{"AcquireSkillInfo without items", mobiusAcquireSkillInfo(AcquireSkillInfo{SkillID: 3, Level: 1, Type: AcquireSkillFishing}),
			AcquireSkillInfo{SkillID: 3, Level: 1, Type: AcquireSkillFishing}},
		{"AcquireSkillDone", msg(0x8e).b, AcquireSkillDone{}},
		{"ShortcutRegister item", mobiusShortcutRegister(ShortcutItem, 3, 0x10000015, 0),
			ShortcutRegister{Shortcut{Type: ShortcutItem, Slot: 3, ID: 0x10000015, Level: 1}}},
		{"ShortcutRegister skill", mobiusShortcutRegister(ShortcutSkill, 13, 1177, 2),
			ShortcutRegister{Shortcut{Type: ShortcutSkill, Slot: 13, ID: 1177, Level: 2, Rest: []byte{0, 1, 0, 0, 0}}}},
		{"ShortcutRegister action", mobiusShortcutRegister(ShortcutAction, 120, 2, 0),
			ShortcutRegister{Shortcut{Type: ShortcutAction, Slot: 120, ID: 2, Level: 1}}},
		{"ShortcutRegister macro", mobiusShortcutRegister(ShortcutMacro, 0, 1000, 0),
			ShortcutRegister{Shortcut{Type: ShortcutMacro, Slot: 0, ID: 1000, Level: 1}}},
		{"ExAutoSoulShot on", mobiusExAutoSoulShot(1835, true), ExAutoSoulShot{ItemID: 1835, Enabled: true}},
		{"ExAutoSoulShot off", mobiusExAutoSoulShot(1835, false), ExAutoSoulShot{ItemID: 1835}},
		{"PartySpelled", mobiusPartySpelled(PartySpelled{Kind: SpelledPet, ObjectID: 0x60000009,
			Effects: []Effect{{SkillID: 1040, Level: 3, Duration: 1200}, {SkillID: 1068, Level: 1, Duration: -1}}}),
			PartySpelled{Kind: SpelledPet, ObjectID: 0x60000009,
				Effects: []Effect{{SkillID: 1040, Level: 3, Duration: 1200}, {SkillID: 1068, Level: 1, Duration: -1}}}},
		{"PartySpelled empty", mobiusPartySpelled(PartySpelled{ObjectID: 0x10000002}), PartySpelled{ObjectID: 0x10000002}},
		{"ShortBuffStatusUpdate", mobiusShortBuffStatusUpdate(ShortBuffStatusUpdate{SkillID: 2031, Level: 1, Duration: 15}),
			ShortBuffStatusUpdate{SkillID: 2031, Level: 1, Duration: 15}},
		{"ShortBuffStatusUpdate reset", mobiusShortBuffStatusUpdate(ShortBuffStatusUpdate{}), ShortBuffStatusUpdate{}},
	}
	for _, tt := range tests {
		checkDecode(t, tt.name, tt.body, tt.want)
	}

	l := AcquireSkillList{Skills: []LearnableSkill{{ID: 1177, NextLevel: 2}}}
	if s, ok := l.Find(1177); !ok || s.NextLevel != 2 {
		t.Errorf("Find(1177) = %+v, %v", s, ok)
	}
	if _, ok := l.Find(1); ok {
		t.Error("Find(1) found a skill")
	}
}

// TestSkillPacketSizes checks the test builders against body sizes
// counted by hand from the Mobius writeImpl.
func TestSkillPacketSizes(t *testing.T) {
	critical := MagicSkillUse{Critical: true}
	tests := []struct {
		name string
		body []byte
		size int
	}{
		// Nine d, the critical d, three d.
		{"MagicSkillUse", mobiusMagicSkillUse(MagicSkillUse{}), 52},
		{"MagicSkillUse critical", mobiusMagicSkillUse(critical), 54},
		{"MagicSkillLaunched of 1", mobiusMagicSkillLaunched(MagicSkillLaunched{Targets: []int32{1}}), 20},
		{"MagicSkillCanceled", msg(0x49).i32(0).b, 4},
		{"SetupGauge", mobiusSetupGauge(SetupGauge{}), 12},
		{"AcquireSkillList of 1", mobiusAcquireSkillList(AcquireSkillList{Skills: []LearnableSkill{{}}}), 8 + 20},
		{"AcquireSkillInfo of 1", mobiusAcquireSkillInfo(AcquireSkillInfo{Requirements: []SkillRequirement{{}}}), 20 + 20},
		{"ShortcutRegister item", mobiusShortcutRegister(ShortcutItem, 0, 0, 0), 16},
		{"ShortcutRegister skill", mobiusShortcutRegister(ShortcutSkill, 0, 0, 0), 21},
		{"ExAutoSoulShot", mobiusExAutoSoulShot(0, false), 2 + 8},
		{"PartySpelled of 1", mobiusPartySpelled(PartySpelled{Effects: []Effect{{}}}), 12 + 10},
		{"ShortBuffStatusUpdate", mobiusShortBuffStatusUpdate(ShortBuffStatusUpdate{}), 12},
	}
	for _, tt := range tests {
		if got := len(tt.body) - 1; got != tt.size {
			t.Errorf("%s body is %d bytes, Mobius writes %d", tt.name, got, tt.size)
		}
	}
}

// TestMagicSkillUseCriticalLayout writes the tail of MagicSkillUse byte by
// byte: the critical flag, its short, then the target's position.
func TestMagicSkillUseCriticalLayout(t *testing.T) {
	head := msg(0x48).i32(1).i32(2).i32(3).i32(4).i32(5).i32(6).i32(7).i32(8).i32(9).b
	tests := []struct {
		name     string
		tail     []byte
		critical bool
	}{
		{"plain", []byte{0, 0, 0, 0, 10, 0, 0, 0, 11, 0, 0, 0, 12, 0, 0, 0}, false},
		{"critical", []byte{1, 0, 0, 0, 0, 0, 10, 0, 0, 0, 11, 0, 0, 0, 12, 0, 0, 0}, true},
	}
	for _, tt := range tests {
		body := append(append([]byte(nil), head...), tt.tail...)
		want := MagicSkillUse{CasterID: 1, TargetID: 2, SkillID: 3, Level: 4, HitTime: 5, ReuseDelay: 6,
			X: 7, Y: 8, Z: 9, Critical: tt.critical, TargetX: 10, TargetY: 11, TargetZ: 12}
		checkDecode(t, "MagicSkillUse "+tt.name, body, want)
		if !bytes.Equal(mobiusMagicSkillUse(want), body) {
			t.Errorf("%s: test builder differs:\n got % x\nwant % x", tt.name, mobiusMagicSkillUse(want), body)
		}
	}
}

// TestAcquireSkillInfoLayout checks that the item count is a long.
func TestAcquireSkillInfoLayout(t *testing.T) {
	body := []byte{
		0x8b,
		0x99, 0x04, 0x00, 0x00, // skill 1177
		0x02, 0x00, 0x00, 0x00, // level 2
		0x32, 0x00, 0x00, 0x00, // SP 50
		0x00, 0x00, 0x00, 0x00, // class skill
		0x01, 0x00, 0x00, 0x00, // one item
		0x63, 0x00, 0x00, 0x00, // 99
		0x1a, 0x04, 0x00, 0x00, // item 1050
		0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // count 2 as a long
		0x32, 0x00, 0x00, 0x00, // 50
	}
	want := AcquireSkillInfo{SkillID: 1177, Level: 2, SPCost: 50, Type: AcquireSkillClass,
		Requirements: []SkillRequirement{{Type: 99, ItemID: 1050, Count: 2, Unknown: 50}}}
	checkDecode(t, "AcquireSkillInfo", body, want)
	if !bytes.Equal(mobiusAcquireSkillInfo(want), body) {
		t.Errorf("test builder differs:\n got % x\nwant % x", mobiusAcquireSkillInfo(want), body)
	}
}

// TestShortcutRegisterLayout checks the short item form against the
// longer ShortcutInit entry: an item shortcut ends after its ID and the 1.
func TestShortcutRegisterLayout(t *testing.T) {
	body := []byte{
		0x44,
		0x01, 0x00, 0x00, 0x00, // item
		0x0f, 0x00, 0x00, 0x00, // slot 3 of page 1
		0x15, 0x00, 0x00, 0x10, // object 0x10000015
		0x01, 0x00, 0x00, 0x00, // 1
	}
	if !bytes.Equal(mobiusShortcutRegister(ShortcutItem, 15, 0x10000015, 0), body) {
		t.Errorf("test builder differs:\n got % x\nwant % x", mobiusShortcutRegister(ShortcutItem, 15, 0x10000015, 0), body)
	}
	checkDecode(t, "ShortcutRegister", body, ShortcutRegister{Shortcut{Type: ShortcutItem, Slot: 15, ID: 0x10000015, Level: 1}})
}

func TestDecodeSkillPacketErrors(t *testing.T) {
	critical := mobiusMagicSkillUse(MagicSkillUse{Critical: true})
	tests := []struct {
		name   string
		body   []byte
		substr string
	}{
		{"MagicSkillUse critical 2", msg(0x48).i32(0).i32(0).i32(0).i32(0).i32(0).i32(0).i32(0).i32(0).i32(0).
			i32(2).i32(0).i32(0).i32(0).b, "MagicSkillUse: critical flag is 2"},
		{"MagicSkillUse critical without its short", append(critical[:1+40:1+40], make([]byte, 12)...),
			"MagicSkillUse: packets: short packet"},
		{"MagicSkillLaunched negative count", msg(0x76).i32(1).i32(2).i32(3).i32(-1).b, "MagicSkillLaunched: target count -1"},
		{"MagicSkillLaunched target missing", msg(0x76).i32(1).i32(2).i32(3).i32(2).i32(9).b, "MagicSkillLaunched: packets: short packet"},
		{"MagicSkillCanceled extra", msg(0x49).i32(1).u16(0).b, "MagicSkillCanceled: 2 bytes after the last field"},
		{"SetupGauge short", msg(0x6d).i32(0).i32(1).b, "SetupGauge: packets: short packet"},
		{"AcquireSkillList negative count", msg(0x8a).i32(0).i32(-3).b, "AcquireSkillList: skill count -3"},
		{"AcquireSkillInfo item cut", msg(0x8b).i32(1).i32(1).i32(0).i32(0).i32(1).i32(99).i32(57).i32(1).b,
			"AcquireSkillInfo: packets: short packet"},
		{"AcquireSkillDone with a byte", msg(0x8e).u8(0).b, "AcquireSkillDone: 1 bytes after the last field"},
		{"ShortcutRegister type 0", msg(0x44).i32(0).i32(1).i32(1).b, "ShortcutRegister: shortcut type 0"},
		{"ShortcutRegister type 7", msg(0x44).i32(7).i32(1).i32(1).i32(1).b, "ShortcutRegister: shortcut type 7"},
		{"ShortcutRegister skill without the 1", mobiusShortcutRegister(ShortcutSkill, 1, 2, 3)[:18], "ShortcutRegister: packets: short packet"},
		{"ExAutoSoulShot enabled 2", msg(0xfe).u16(0x12).i32(1835).i32(2).b, "ExAutoSoulShot: enabled flag is 2"},
		{"PartySpelled negative count", msg(0xee).i32(0).i32(1).i32(-1).b, "PartySpelled: effect count -1"},
		{"PartySpelled effect cut", msg(0xee).i32(0).i32(1).i32(1).i32(1040).u16(3).b, "PartySpelled: packets: short packet"},
		{"ShortBuffStatusUpdate extra", msg(0xf4).i32(0).i32(0).i32(0).i32(0).b, "ShortBuffStatusUpdate: 4 bytes after the last field"},
	}
	for _, tt := range tests {
		_, err := DecodeServerPacket(tt.body)
		if err == nil || !strings.Contains(err.Error(), tt.substr) {
			t.Errorf("%s: got %v, want an error containing %q", tt.name, err, tt.substr)
			continue
		}
		if !errors.Is(err, ErrMalformed) {
			t.Errorf("%s: %v does not wrap ErrMalformed", tt.name, err)
		}
	}
}
