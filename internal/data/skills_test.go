package data

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

// Test templates in the form datagen writes them, from the Mobius skill
// files named on each.

// windStrike is skill 1177 of stats/skills/01100-01199.xml.
func windStrike() SkillTemplate {
	return SkillTemplate{
		ID: 1177, Name: "Wind Strike", Levels: 5,
		Tables: map[string][]string{
			"#effectPoints":     {"-92", "-106", "-121", "-143", "-162"},
			"#magicLevel":       {"1", "4", "7", "11", "14"},
			"#mpConsume":        {"7", "7", "8", "11", "12"},
			"#mpInitialConsume": {"2", "2", "2", "3", "3"},
			"#power":            {"12", "13", "15", "18", "21"},
		},
		Values: map[string]string{
			"icon": "icon.skill1177", "operateType": "A1", "targetType": "ONE", "castRange": "600",
			"effectPoint": "#effectPoints", "effectRange": "1100", "element": "2", "elementPower": "20",
			"hitTime": "4000", "isMagic": "1", "magicLevel": "#magicLevel", "mpConsume": "#mpConsume",
			"mpInitialConsume": "#mpInitialConsume", "power": "#power", "reuseDelay": "6000",
		},
		Effects: map[EffectScope][]EffectDef{ScopeGeneral: {{Name: "MagicalDamage"}}},
	}
}

// might is skill 1068 of stats/skills/01000-01099.xml: a buff with two
// enchant routes (time and cost).
func might() SkillTemplate {
	return SkillTemplate{
		ID: 1068, Name: "Might", Levels: 3,
		Tables: map[string][]string{
			"#abnormalLevels":        {"1", "2", "3"},
			"#effectPoints":          {"121", "204", "379"},
			"#magicLevel":            {"7", "20", "40"},
			"#mpConsume":             {"8", "16", "28"},
			"#mpInitialConsume":      {"2", "4", "7"},
			"#rate":                  {"1.08", "1.12", "1.15"},
			"#ench1AbnormalTimes":    strings.Fields("1240 1280 1320 1360 1400 1440 1480 1520 1560 1600 1640 1680 1720 1760 1800 1840 1880 1920 1960 2000 2040 2080 2120 2160 2200 2240 2280 2320 2360 2400"),
			"#ench2mpConsume":        strings.Fields("27 27 26 26 25 25 24 24 23 23 22 22 21 21 21 20 20 19 19 18 18 17 17 16 16 15 15 14 14 14"),
			"#ench2mpInitialConsume": strings.Fields("6 6 6 6 6 6 6 6 5 5 5 5 5 5 5 5 5 4 4 4 4 4 4 4 4 3 3 3 3 3"),
			"#enchMagicLvl":          strings.Fields("76 76 76 77 77 77 78 78 78 79 79 79 80 80 80 81 81 81 82 82 82 83 83 83 84 84 84 85 85 85"),
		},
		Values: map[string]string{
			"abnormalLevel": "#abnormalLevels", "abnormalTime": "1200", "abnormalType": "PA_UP", "castRange": "400",
			"effectPoint": "#effectPoints", "effectRange": "900", "hitTime": "4000", "icon": "icon.skill1068",
			"isMagic": "1", "magicLevel": "#magicLevel", "mpConsume": "#mpConsume", "mpInitialConsume": "#mpInitialConsume",
			"operateType": "A2", "reuseDelay": "6000", "targetType": "ONE",
		},
		Effects: map[EffectScope][]EffectDef{ScopeGeneral: {{
			Name:  "Buff",
			Funcs: []StatFuncDef{{Stat: StatPowerAttack, Op: OpMul, Order: 20, Value: "#rate"}},
		}}},
		Enchants: []EnchantRoute{
			{Levels: 30, Values: map[string]string{"abnormalTime": "#ench1AbnormalTimes", "magicLevel": "#enchMagicLvl"}},
			{Levels: 30, Values: map[string]string{"magicLevel": "#enchMagicLvl", "mpConsume": "#ench2mpConsume", "mpInitialConsume": "#ench2mpInitialConsume"}},
		},
	}
}

// weaponMastery is skill 141 of stats/skills/00100-00199.xml, a passive.
func weaponMastery() SkillTemplate {
	return SkillTemplate{
		ID: 141, Name: "Weapon Mastery", Levels: 3,
		Tables: map[string][]string{"#magicLevel": {"5", "10", "15"}, "#pAtk": {"2", "3", "4"}},
		Values: map[string]string{"icon": "icon.skill0141", "operateType": "P", "targetType": "SELF", "magicLevel": "#magicLevel"},
		Effects: map[EffectScope][]EffectDef{ScopeGeneral: {{
			Name: "Buff",
			Funcs: []StatFuncDef{
				{Stat: StatPowerAttack, Op: OpMul, Order: 20, Value: "1.085"},
				{Stat: StatPowerAttack, Op: OpAdd, Order: 30, Value: "#pAtk"},
			},
		}}},
	}
}

// powerStrike is skill 3 of stats/skills/00000-00099.xml, which needs a
// sword or a blunt weapon.
func powerStrike() SkillTemplate {
	return SkillTemplate{
		ID: 3, Name: "Power Strike", Levels: 9,
		Tables: map[string][]string{
			"#effectPoints": strings.Fields("-52 -54 -57 -64 -67 -70 -79 -83 -86"),
			"#magicLevel":   strings.Fields("3 4 5 8 9 10 13 14 15"),
			"#mpConsume":    strings.Fields("10 10 11 13 13 14 17 18 19"),
			"#power":        strings.Fields("25 27 30 39 42 46 60 65 70"),
		},
		Values: map[string]string{
			"castRange": "40", "coolTime": "720", "effectPoint": "#effectPoints", "effectRange": "400", "hitTime": "1080",
			"icon": "icon.skill0003", "magicLevel": "#magicLevel", "mpConsume": "#mpConsume", "nextActionAttack": "true",
			"operateType": "A1", "overHit": "true", "power": "#power", "reuseDelay": "13000", "targetType": "ONE",
		},
		Conditions: []SkillConditions{{MsgID: 113, AddName: true, Cond: ConditionDef{Kind: CondUsingKind, Value: "SWORD,BLUNT"}}},
		Effects:    map[EffectScope][]EffectDef{ScopeGeneral: {{Name: "PhysicalDamage"}}},
	}
}

// sonicFocus is skill 8 of stats/skills/00000-00099.xml: two condition
// blocks, the second using a table.
func sonicFocus() SkillTemplate {
	return SkillTemplate{
		ID: 8, Name: "Sonic Focus", Levels: 7,
		Tables: map[string][]string{
			"#magicLevel": strings.Fields("40 43 49 55 60 66 70"),
			"#maxCharges": strings.Fields("1 2 3 4 5 6 7"),
		},
		Values: map[string]string{
			"icon": "icon.skill0008", "operateType": "A1", "targetType": "SELF", "effectPoint": "200", "hitTime": "900",
			"hpConsume": "25", "magicLevel": "#magicLevel", "mpConsume": "5", "reuseDelay": "1000",
		},
		Conditions: []SkillConditions{
			{MsgID: 113, AddName: true, Cond: ConditionDef{Kind: CondUsingKind, Value: "DUAL,SWORD,BLUNT"}},
			{MsgID: 324, Cond: ConditionDef{Kind: CondNot, Conds: []ConditionDef{{Kind: CondPlayerCharges, Value: "#maxCharges"}}}},
		},
		Effects: map[EffectScope][]EffectDef{ScopeGeneral: {{Name: "FocusEnergy", Params: map[string]string{"charge": "#maxCharges"}}}},
	}
}

func newTestSkills(t *testing.T, d SkillsData) *Skills {
	t.Helper()
	s, err := NewSkills(d)
	if err != nil {
		t.Fatalf("NewSkills: %v", err)
	}
	return s
}

func mustSkill(t *testing.T, s *Skills, id, level int) *Skill {
	t.Helper()
	sk, ok := s.Skill(id, level)
	if !ok {
		t.Fatalf("Skill(%d, %d) not found", id, level)
	}
	return sk
}

// A table skill resolves each level from its tables (Wind Strike).
func TestSkillTableLevels(t *testing.T) {
	s := newTestSkills(t, SkillsData{Skills: []SkillTemplate{windStrike()}})
	for _, tt := range []struct {
		level, mp, mpInit, magicLevel, effectPoint int
		power                                      float64
	}{
		{1, 7, 2, 1, -92, 12},
		{3, 8, 2, 7, -121, 15},
		{5, 12, 3, 14, -162, 21},
	} {
		sk := mustSkill(t, s, 1177, tt.level)
		if sk.MPConsume != tt.mp || sk.MPInitialConsume != tt.mpInit || sk.MagicLevel != tt.magicLevel ||
			sk.EffectPoint != tt.effectPoint || sk.Power != tt.power {
			t.Errorf("level %d: mp %d+%d, magic level %d, effect point %d, power %v; want %d+%d, %d, %d, %v",
				tt.level, sk.MPConsume, sk.MPInitialConsume, sk.MagicLevel, sk.EffectPoint, sk.Power,
				tt.mp, tt.mpInit, tt.magicLevel, tt.effectPoint, tt.power)
		}
		if sk.ID != 1177 || sk.Level != tt.level || sk.DisplayID != 1177 || sk.DisplayLevel != tt.level || sk.Name != "Wind Strike" {
			t.Errorf("level %d: id %d level %d display %d/%d name %q", tt.level, sk.ID, sk.Level, sk.DisplayID, sk.DisplayLevel, sk.Name)
		}
		// Values without tables, and the Mobius defaults.
		if sk.OperateType != OperateA1 || sk.TargetType != TargetOne || !sk.IsMagic() || sk.CastRange != 600 ||
			sk.EffectRange != 1100 || sk.HitTime != 4000 || sk.ReuseDelay != 6000 || sk.Element != 2 ||
			sk.ElementPower != 20 || sk.Icon != "icon.skill1177" {
			t.Errorf("level %d: fixed values %+v", tt.level, sk)
		}
		if sk.PvPPower != sk.Power || sk.PvEPower != sk.Power || sk.MPPerChanneling != sk.MPConsume ||
			sk.ActivateRate != -1 || sk.MinChance != -1 || sk.AbnormalType != AbnormalNone || sk.Trait != TraitNone ||
			sk.BasicProperty != BaseNone || !sk.CanBeDispeled || !sk.StayOnSubclassChange ||
			sk.ChannelingTickInterval != 2000 || sk.ChannelingTickInitialDelay != 2000 || sk.CoolTime != 0 {
			t.Errorf("level %d: defaults %+v", tt.level, sk)
		}
		if e := sk.Effects[ScopeGeneral]; len(e) != 1 || e[0].Name != EffectMagicalDamage || len(e[0].Params) != 0 {
			t.Errorf("level %d: effects %+v", tt.level, sk.Effects)
		}
	}
	for _, level := range []int{0, 6, 101} {
		if _, ok := s.Skill(1177, level); ok {
			t.Errorf("Skill(1177, %d) exists", level)
		}
	}
	if n := len(s.Skills[0].AllLevels()); n != 5 {
		t.Errorf("AllLevels has %d levels, want 5", n)
	}
}

// Enchant levels take the last normal level and the route's values; a
// table reference of the route takes the enchant level's entry (Might).
func TestSkillEnchantLevels(t *testing.T) {
	s := newTestSkills(t, SkillsData{Skills: []SkillTemplate{might()}})
	for _, tt := range []struct {
		level, abnormalTime, magicLevel, mp, mpInit int
	}{
		{1, 1200, 7, 8, 2},
		{3, 1200, 40, 28, 7},
		{101, 1240, 76, 28, 7}, // route 1 (time), +1
		{130, 2400, 85, 28, 7}, // route 1, +30
		{141, 1200, 76, 27, 6}, // route 2 (cost), +1
		{155, 1200, 80, 21, 5}, // route 2, +15
		{170, 1200, 85, 14, 3}, // route 2, +30
	} {
		sk := mustSkill(t, s, 1068, tt.level)
		if sk.AbnormalTime != tt.abnormalTime || sk.MagicLevel != tt.magicLevel || sk.MPConsume != tt.mp || sk.MPInitialConsume != tt.mpInit {
			t.Errorf("level %d: abnormal time %d, magic level %d, mp %d+%d; want %d, %d, %d+%d", tt.level,
				sk.AbnormalTime, sk.MagicLevel, sk.MPConsume, sk.MPInitialConsume, tt.abnormalTime, tt.magicLevel, tt.mp, tt.mpInit)
		}
		if sk.Level != tt.level || sk.DisplayLevel != tt.level || sk.AbnormalType != AbnormalPaUp || sk.OperateType != OperateA2 {
			t.Errorf("level %d: level %d display %d, abnormal %v, operate %v", tt.level, sk.Level, sk.DisplayLevel, sk.AbnormalType, sk.OperateType)
		}
		// The routes have no effects of their own: those of the last
		// normal level apply.
		wantRate := 1.15
		if tt.level == 1 {
			wantRate = 1.08
		}
		e := sk.Effects[ScopeGeneral]
		if len(e) != 1 || e[0].Name != EffectBuff || len(e[0].Funcs) != 1 ||
			e[0].Funcs[0] != (StatFunc{Stat: StatPowerAttack, Op: OpMul, Order: 20, Value: wantRate}) {
			t.Errorf("level %d: effects %+v, want Buff mul pAtk %v", tt.level, e, wantRate)
		}
		if sk.AbnormalLevel != map[bool]int{true: 1, false: 3}[tt.level == 1] {
			t.Errorf("level %d: abnormal level %d", tt.level, sk.AbnormalLevel)
		}
	}
	for _, level := range []int{4, 100, 131, 140, 171} {
		if _, ok := s.Skill(1068, level); ok {
			t.Errorf("Skill(1068, %d) exists", level)
		}
	}
	if n := len(s.Skills[0].AllLevels()); n != 63 {
		t.Errorf("AllLevels has %d levels, want 3+30+30", n)
	}
	if EnchantLevel(1) != 101 || EnchantLevel(2) != 141 || EnchantLevel(3) != 0 {
		t.Errorf("EnchantLevel = %d, %d, %d", EnchantLevel(1), EnchantLevel(2), EnchantLevel(3))
	}
}

// An enchant route's own blocks replace the skill's and take the enchant
// level's table entries; the others are the skill's at its last level,
// except that route 2 gets no <startEffects> (Mobius DocumentSkill). An
// effect name may come from a table.
func TestEnchantRouteBlocks(t *testing.T) {
	addPAtk := func(v string) []StatFuncDef {
		return []StatFuncDef{{Stat: StatPowerAttack, Op: OpAdd, Order: 30, Value: v}}
	}
	tpl := SkillTemplate{
		ID: 9001, Name: "Test", Levels: 2,
		Tables: map[string][]string{"#power": {"10", "20"}, "#ench": {"100", "200"}, "#effect": {"Buff", "Stun"}},
		Values: map[string]string{"operateType": "A2", "power": "#power"},
		Effects: map[EffectScope][]EffectDef{
			ScopeGeneral: {{Name: "Buff", Funcs: addPAtk("#power")}},
			ScopeStart:   {{Name: "MagicalDamage"}},
			ScopeSelf:    {{Name: "#effect"}},
		},
		Enchants: []EnchantRoute{
			{Levels: 2, Values: map[string]string{"power": "#ench"}, Effects: map[EffectScope][]EffectDef{
				ScopeGeneral: {{Name: "Buff", Funcs: addPAtk("#ench")}},
			}},
			{Levels: 2, Effects: map[EffectScope][]EffectDef{ScopeSelf: {}}},
		},
	}
	s := newTestSkills(t, SkillsData{Skills: []SkillTemplate{tpl}})
	type want struct {
		power   float64
		general float64     // the Buff's pAtk, 0 for none
		start   bool        // MagicalDamage at start
		self    *EffectName // nil for none
	}
	buff, stun := EffectBuff, EffectStun
	for level, w := range map[int]want{
		1:   {10, 10, true, &buff},
		2:   {20, 20, true, &stun},
		101: {100, 100, true, &stun},
		102: {200, 200, true, &stun},
		141: {20, 20, false, nil},
		142: {20, 20, false, nil},
	} {
		sk := mustSkill(t, s, 9001, level)
		if sk.Power != w.power {
			t.Errorf("level %d: power %v, want %v", level, sk.Power, w.power)
		}
		g := sk.Effects[ScopeGeneral]
		if len(g) != 1 || g[0].Funcs[0].Value != w.general {
			t.Errorf("level %d: general effects %+v, want Buff +%v", level, g, w.general)
		}
		if st := sk.Effects[ScopeStart]; (len(st) == 1 && st[0].Name == EffectMagicalDamage) != w.start {
			t.Errorf("level %d: start effects %+v, want MagicalDamage: %v", level, st, w.start)
		}
		self := sk.Effects[ScopeSelf]
		switch {
		case w.self == nil && len(self) != 0:
			t.Errorf("level %d: self effects %+v, want none", level, self)
		case w.self != nil && (len(self) != 1 || self[0].Name != *w.self):
			t.Errorf("level %d: self effects %+v, want %v", level, self, *w.self)
		}
	}
}

// A passive skill has its <effects> as passive effects (Weapon Mastery).
func TestPassiveEffects(t *testing.T) {
	s := newTestSkills(t, SkillsData{Skills: []SkillTemplate{weaponMastery()}})
	for level, add := range map[int]float64{1: 2, 3: 4} {
		sk := mustSkill(t, s, 141, level)
		if !sk.IsPassive() || sk.IsToggle() || len(sk.Effects[ScopeGeneral]) != 0 {
			t.Errorf("level %d: passive %v, general effects %+v", level, sk.IsPassive(), sk.Effects[ScopeGeneral])
		}
		want := []StatFunc{
			{Stat: StatPowerAttack, Op: OpMul, Order: 20, Value: 1.085},
			{Stat: StatPowerAttack, Op: OpAdd, Order: 30, Value: add},
		}
		e := sk.Effects[ScopePassive]
		if len(e) != 1 || e[0].Name != EffectBuff || !slices.Equal(e[0].Funcs, want) {
			t.Errorf("level %d: passive effects %+v, want Buff %+v", level, e, want)
		}
		if !sk.HasEffect(EffectBuff) || sk.HasEffect(EffectStun) {
			t.Errorf("level %d: HasEffect", level)
		}
	}
}

// Conditions resolve into typed trees, with their messages; a table in a
// condition takes the level's entry (Power Strike, Sonic Focus).
func TestSkillConditions(t *testing.T) {
	s := newTestSkills(t, SkillsData{Skills: []SkillTemplate{powerStrike(), sonicFocus()}})
	ps := mustSkill(t, s, 3, 7)
	if len(ps.Conditions) != 1 {
		t.Fatalf("Power Strike conditions = %+v", ps.Conditions)
	}
	c := ps.Conditions[0]
	if c.MsgID != 113 || !c.AddName || c.Msg != "" || c.Cond.Kind != CondUsingKind || c.Cond.Int != WeaponSword.Mask()|WeaponBlunt.Mask() {
		t.Errorf("Power Strike condition = %+v, want using sword|blunt (%#x) with message 113 and the name", c, WeaponSword.Mask()|WeaponBlunt.Mask())
	}
	if ps.Power != 60 || ps.MPConsume != 17 || ps.MagicLevel != 13 || ps.EffectPoint != -79 || !ps.NextActionAttack || !ps.OverHit ||
		ps.CoolTime != 720 || ps.HitTime != 1080 || ps.ReuseDelay != 13000 || ps.CastRange != 40 {
		t.Errorf("Power Strike level 7 = %+v", ps)
	}

	for level := 1; level <= 7; level++ {
		sf := mustSkill(t, s, 8, level)
		if len(sf.Conditions) != 2 {
			t.Fatalf("Sonic Focus level %d conditions = %+v", level, sf.Conditions)
		}
		first, second := sf.Conditions[0], sf.Conditions[1]
		if first.Cond.Int != WeaponDual.Mask()|WeaponSword.Mask()|WeaponBlunt.Mask() || first.MsgID != 113 {
			t.Errorf("level %d: first condition %+v", level, first)
		}
		want := Condition{Kind: CondNot, Conds: []Condition{{Kind: CondPlayerCharges, Int: level}}}
		if second.MsgID != 324 || second.AddName || !equalConditions(second.Cond, want) {
			t.Errorf("level %d: second condition %+v, want %+v", level, second, want)
		}
		if v, ok := sf.Effects[ScopeGeneral][0].Params.Get("charge"); !ok || v != string(rune('0'+level)) {
			t.Errorf("level %d: charge parameter %q, %v", level, v, ok)
		}
	}
}

func equalConditions(a, b Condition) bool {
	if a.Kind != b.Kind || a.Bool != b.Bool || a.Int != b.Int || !slices.Equal(a.Ints, b.Ints) ||
		!slices.Equal(a.Races, b.Races) || !slices.Equal(a.Categories, b.Categories) ||
		!slices.Equal(a.NpcTypes, b.NpcTypes) || a.Abnormal != b.Abnormal || a.Element != b.Element ||
		len(a.Conds) != len(b.Conds) {
		return false
	}
	for i := range a.Conds {
		if !equalConditions(a.Conds[i], b.Conds[i]) {
			return false
		}
	}
	return true
}

// Each condition kind parses its value as Mobius DocumentBase does.
func TestConditionKinds(t *testing.T) {
	tpl := &SkillTemplate{Tables: map[string][]string{
		"#range": {"1;20", "21;40"}, "#lvl": {"10", "20"}, "#skill": {"1234,5", "1234,6"},
	}}
	ti := &tableIndex{tpl, 1}
	leaf := func(k ConditionKind, v string) ConditionDef { return ConditionDef{Kind: k, Value: v} }
	tests := []struct {
		def  ConditionDef
		want Condition
	}{
		{leaf(CondPlayerLevel, "#lvl"), Condition{Kind: CondPlayerLevel, Int: 20}},
		{leaf(CondPlayerLevelrange, "#range"), Condition{Kind: CondPlayerLevelrange, Ints: []int{21, 40}}},
		{leaf(CondTargetLevelrange, "5;9"), Condition{Kind: CondTargetLevelrange, Ints: []int{5, 9}}},
		{leaf(CondPlayerActiveSkillIDLvl, "#skill"), Condition{Kind: CondPlayerActiveSkillIDLvl, Ints: []int{1234, 6}}},
		{leaf(CondTargetActiveEffectIDLvl, "4001,#lvl"), Condition{Kind: CondTargetActiveEffectIDLvl, Ints: []int{4001, 20}}},
		{leaf(CondPlayerElementseed, "WIND,2"), Condition{Kind: CondPlayerElementseed, Element: SeedWind, Int: 2}},
		{leaf(CondTargetNpctype, "Attackable,Chest"), Condition{Kind: CondTargetNpctype, NpcTypes: []InstanceType{InstanceAttackable, InstanceChest}}},
		{leaf(CondPlayerRaces, "HUMAN,ELF"), Condition{Kind: CondPlayerRaces, Races: []Race{RaceHuman, RaceElf}}},
		{leaf(CondTargetRace, "UNDEAD"), Condition{Kind: CondTargetRace, Races: []Race{RaceUndead}}},
		{leaf(CondPlayerInsidezoneid, "10501, 10502,10503"), Condition{Kind: CondPlayerInsidezoneid, Ints: []int{10501, 10502, 10503}}},
		{leaf(CondPlayerCheckabnormal, "PA_UP,3"), Condition{Kind: CondPlayerCheckabnormal, Abnormal: AbnormalPaUp, Int: 3}},
		{leaf(CondPlayerCheckabnormal, "PA_UP"), Condition{Kind: CondPlayerCheckabnormal, Abnormal: AbnormalPaUp, Int: -1}},
		{leaf(CondUsingSlot, "lrhand"), Condition{Kind: CondUsingSlot, Int: BodyPartLRHand.Mask()}},
		{leaf(CondUsingKind, "LIGHT, BOW"), Condition{Kind: CondUsingKind, Int: ArmorLight.Mask() | WeaponBow.Mask()}},
		{leaf(CondTargetUsing, "SWORD"), Condition{Kind: CondTargetUsing, Int: WeaponSword.Mask()}},
		{leaf(CondUsingSlotitem, "57;7"), Condition{Kind: CondUsingSlotitem, Ints: []int{57, 7, 0}}},
		{leaf(CondUsingSkill, "8"), Condition{Kind: CondUsingSkill, Int: 8}},
		{leaf(CondGameChance, "0x10"), Condition{Kind: CondGameChance, Int: 16}},
		{leaf(CondGameNight, "TRUE"), Condition{Kind: CondGameNight, Bool: true}},
		{leaf(CondPlayerNpcidradius, "1;2,100,true"), Condition{Kind: CondPlayerNpcidradius, Ints: []int{1, 2}, Int: 100, Bool: true}},
		{leaf(CondPlayerCategorytype, "FIGHTER_GROUP"), Condition{Kind: CondPlayerCategorytype, Categories: []CategoryType{CategoryFighterGroup}}},
		{leaf(CondPlayerCantakecastle, "true"), Condition{Kind: CondPlayerCantakecastle}},
		{leaf(CondPlayerHP, "50"), Condition{Kind: CondPlayerHP, Int: 50}},
		{ConditionDef{Kind: CondAnd, Conds: []ConditionDef{leaf(CondPlayerInvsize, "10"), leaf(CondPlayerWeight, "80")}},
			Condition{Kind: CondAnd, Conds: []Condition{{Kind: CondPlayerInvsize, Int: 10}, {Kind: CondPlayerWeight, Int: 80}}}},
		{ConditionDef{Kind: CondOr, Conds: []ConditionDef{leaf(CondPlayerFront, "true"), {Kind: CondNot, Conds: []ConditionDef{leaf(CondPlayerBehind, "false")}}}},
			Condition{Kind: CondOr, Conds: []Condition{{Kind: CondPlayerFront, Bool: true}, {Kind: CondNot, Conds: []Condition{{Kind: CondPlayerBehind}}}}}},
	}
	for _, tt := range tests {
		got, err := tt.def.resolve(ti)
		if err != nil {
			t.Errorf("%s %q: %v", tt.def.Kind, tt.def.Value, err)
			continue
		}
		if !equalConditions(got, tt.want) {
			t.Errorf("%s %q = %+v, want %+v", tt.def.Kind, tt.def.Value, got, tt.want)
		}
	}

	bad := []struct {
		def    ConditionDef
		noTabs bool // resolved as the condition of a stat function
		want   string
	}{
		{leaf(CondPlayerHP, "#lvl"), false, "Mobius takes no table here"},
		{leaf(CondPlayerLevel, "#lvl"), true, "Mobius takes no table here"},
		{leaf(CondPlayerLevel, "#missing"), false, "table #missing does not exist"},
		{leaf(CondUsingKind, "LASER"), false, `unknown weapon or armor type "LASER"`},
		{leaf(CondPlayerResting, "1"), false, `"1" is not true or false`},
		{leaf(CondPlayerLevelrange, "1;2;3"), false, "want min;max"},
		{leaf(CondPlayerRaces, "HUMAN, ELF"), false, `unknown race " ELF"`},
		{leaf(CondPlayerNpcidradius, "1,100"), false, "want ids,radius"},
		{leaf(CondTargetNpctype, "Folkx"), false, `unknown instance type "Folkx"`},
		{leaf(CondPlayerClanhall, ""), false, "no values"},
		{ConditionDef{Kind: CondNot, Conds: []ConditionDef{leaf(CondPlayerHP, "1"), leaf(CondPlayerMP, "1")}}, false, "not: 2 operands"},
		{ConditionDef{Kind: CondAnd}, false, "and: 0 operands"},
		{ConditionDef{Kind: CondAnd, Value: "x", Conds: []ConditionDef{leaf(CondPlayerHP, "1")}}, false, "a logic condition has no value"},
		{ConditionDef{Kind: CondPlayerHP, Value: "1", Conds: []ConditionDef{leaf(CondPlayerHP, "1")}}, false, "a leaf condition has operands"},
		{ConditionDef{Kind: ConditionKind(NumConditionKinds)}, false, "unknown condition"},
	}
	for _, tt := range bad {
		var idx *tableIndex
		if !tt.noTabs {
			idx = ti
		}
		_, err := tt.def.resolve(idx)
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s %q: error %v, want one with %q", tt.def.Kind, tt.def.Value, err, tt.want)
		}
	}
}

// Every leaf condition kind Mobius knows has a parser.
func TestCondParsersCoverEveryKind(t *testing.T) {
	for k := range ConditionKind(NumConditionKinds) {
		if _, ok := condParsers[k]; !ok && !k.IsLogic() {
			t.Errorf("no parser for condition %s", k)
		}
	}
}

// A stat function's condition applies it to one armor kind (Light Armor
// Mastery, stats/skills/00200-00299.xml, with two of its 50 levels).
func TestStatFuncCondition(t *testing.T) {
	light := ConditionDef{Kind: CondUsingKind, Value: "LIGHT"}
	tpl := SkillTemplate{
		ID: 227, Name: "Light Armor Mastery", Levels: 2,
		Tables: map[string][]string{"#pDef": {"4.2", "5.3"}, "#rEvas": {"3", "3"}},
		Values: map[string]string{"operateType": "P", "targetType": "SELF"},
		Effects: map[EffectScope][]EffectDef{ScopeGeneral: {{Name: "Buff", Funcs: []StatFuncDef{
			{Stat: StatPowerDefence, Op: OpAdd, Order: 30, Value: "#pDef", Cond: &light},
			{Stat: StatEvasionRate, Op: OpAdd, Order: 30, Value: "#rEvas", Cond: &light},
		}}}},
	}
	s := newTestSkills(t, SkillsData{Skills: []SkillTemplate{tpl}})
	f := mustSkill(t, s, 227, 2).Effects[ScopePassive][0].Funcs
	if len(f) != 2 || f[0].Value != 5.3 || f[1].Value != 3 || f[0].Stat != StatPowerDefence || f[1].Stat != StatEvasionRate {
		t.Fatalf("functions = %+v", f)
	}
	for _, fn := range f {
		if fn.Cond == nil || fn.Cond.Kind != CondUsingKind || fn.Cond.Int != ArmorLight.Mask() {
			t.Errorf("condition of %s = %+v, want using light armor", fn.Stat, fn.Cond)
		}
	}
}

// RestorationRandom gets the item groups of its level (Jade Treasure Box,
// stats/skills/02200-02299.xml, cut down); chances keep float32 precision.
func TestRestorationItems(t *testing.T) {
	tpl := SkillTemplate{
		ID: 2200, Name: "Item - Jade Treasure Box", Levels: 3,
		Values: map[string]string{"operateType": "A1", "targetType": "SELF"},
		Effects: map[EffectScope][]EffectDef{ScopeGeneral: {{Name: "RestorationRandom", Items: &RestorationGroups{ByLevel: map[int][]RestorationGroup{
			1: {{Chance: 28.5, Items: []RestorationItem{{ID: 6908, Count: 24}}}, {Chance: 26.0572, Items: []RestorationItem{{ID: 7609, Count: 26}}}},
			3: {{Chance: 34.5, Items: []RestorationItem{{ID: 6908, Count: 24, MinEnchant: 1, MaxEnchant: 2}}}},
		}}}}},
	}
	s := newTestSkills(t, SkillsData{Skills: []SkillTemplate{tpl}})
	items := func(level int) []RestorationGroup { return mustSkill(t, s, 2200, level).Effects[ScopeGeneral][0].Items }
	if g := items(1); len(g) != 2 || g[0].Chance != 28.5 || g[1].Chance != float64(float32(26.0572)) || g[1].Items[0] != (RestorationItem{ID: 7609, Count: 26}) {
		t.Errorf("level 1 groups = %+v", g)
	}
	if g := items(2); g != nil {
		t.Errorf("level 2 groups = %+v, want none", g)
	}
	if g := items(3); len(g) != 1 || g[0].Items[0].MaxEnchant != 2 {
		t.Errorf("level 3 groups = %+v", g)
	}
}

func TestEffectParams(t *testing.T) {
	ps := EffectParams{{"chance", "50"}, {"enabled", "TRUE"}, {"power", "1.5"}, {"slot", "buff"}}
	if v, ok := ps.Get("slot"); !ok || v != "buff" {
		t.Errorf("Get(slot) = %q, %v", v, ok)
	}
	if _, ok := ps.Get("missing"); ok {
		t.Error("Get(missing) found")
	}
	if n, err := ps.Int("chance", 0); err != nil || n != 50 {
		t.Errorf("Int(chance) = %d, %v", n, err)
	}
	if n, err := ps.Int("missing", 7); err != nil || n != 7 {
		t.Errorf("Int(missing) = %d, %v", n, err)
	}
	if _, err := ps.Int("power", 0); err == nil {
		t.Error("Int(power) of 1.5 succeeded")
	}
	if f, err := ps.Float("power", 0); err != nil || f != 1.5 {
		t.Errorf("Float(power) = %v, %v", f, err)
	}
	if b, err := ps.Bool("enabled", false); err != nil || !b {
		t.Errorf("Bool(enabled) = %v, %v", b, err)
	}
	if ps.String("slot", "x") != "buff" || ps.String("missing", "x") != "x" {
		t.Error("String")
	}
}

func TestSkillOrMax(t *testing.T) {
	s := newTestSkills(t, SkillsData{Skills: []SkillTemplate{windStrike()}})
	if sk, ok := s.SkillOrMax(1177, 9); !ok || sk.Level != 5 {
		t.Errorf("SkillOrMax(1177, 9) = %v, %v; want level 5", sk, ok)
	}
	if sk, ok := s.SkillOrMax(1177, 2); !ok || sk.Level != 2 {
		t.Errorf("SkillOrMax(1177, 2) = %v, %v", sk, ok)
	}
	if _, ok := s.SkillOrMax(1177, 0); ok {
		t.Error("SkillOrMax(1177, 0) found")
	}
	if _, ok := s.SkillOrMax(1, 1); ok {
		t.Error("SkillOrMax of a missing skill found")
	}
	if s.MaxLevel(1177) != 5 || s.MaxLevel(1) != 0 {
		t.Errorf("MaxLevel = %d, %d", s.MaxLevel(1177), s.MaxLevel(1))
	}
	if tpl, ok := s.Template(1177); !ok || tpl.Name != "Wind Strike" {
		t.Errorf("Template(1177) = %v, %v", tpl, ok)
	}
}

func TestOperateTypes(t *testing.T) {
	for _, tt := range []struct {
		o                                                     SkillOperateType
		active, passive, toggle, continuous, self, channeling bool
	}{
		{OperateA1, true, false, false, false, false, false},
		{OperateA2, true, false, false, true, false, false},
		{OperateA3, true, false, false, false, true, false},
		{OperateCA5, true, false, false, false, false, true},
		{OperateDA2, true, false, false, true, false, false},
		{OperateP, false, true, false, false, false, false},
		{OperateT, false, false, true, false, false, false},
	} {
		if tt.o.IsActive() != tt.active || tt.o.IsPassive() != tt.passive || tt.o.IsToggle() != tt.toggle ||
			tt.o.IsContinuous() != tt.continuous || tt.o.IsSelfContinuous() != tt.self || tt.o.IsChanneling() != tt.channeling {
			t.Errorf("%s: classification wrong", tt.o)
		}
	}
}

func TestSkillValueNames(t *testing.T) {
	names := SkillValueNames()
	if len(names) != len(skillFields) || !slices.IsSorted(names) {
		t.Errorf("SkillValueNames = %v", names)
	}
	for _, n := range []string{"operateType", "targetType", "mpConsume", "hitTime", "abnormalType", "power", "magicLevel"} {
		if !slices.Contains(names, n) {
			t.Errorf("SkillValueNames lacks %s", n)
		}
	}
}

// NewSkills refuses templates that do not resolve, naming the skill and
// the problem.
func TestNewSkillsErrors(t *testing.T) {
	tests := []struct {
		name   string
		change func(d *SkillsData)
		want   string
	}{
		{"missing table", func(d *SkillsData) { d.Skills[0].Values["power"] = "#nope" }, "skill 1177 (\"Wind Strike\"): level 1: value power: table #nope does not exist"},
		{"short table", func(d *SkillsData) { d.Skills[0].Tables["#power"] = []string{"1", "2", "3", "4"} }, "level 5: value power: table #power has 4 values, entry 5 is needed"},
		{"unknown value", func(d *SkillsData) { d.Skills[0].Values["fooBar"] = "1" }, `unknown value "fooBar"`},
		{"no operate type", func(d *SkillsData) { delete(d.Skills[0].Values, "operateType") }, "no operateType"},
		{"bad target type", func(d *SkillsData) { d.Skills[0].Values["targetType"] = "SOMEWHERE" }, `unknown target type "SOMEWHERE"`},
		{"bad abnormal type", func(d *SkillsData) { d.Skills[0].Values["abnormalType"] = "XX" }, `unknown abnormal type "XX"`},
		{"bad int", func(d *SkillsData) { d.Skills[0].Values["mpConsume"] = "x" }, `mpConsume: "x" is not an int`},
		{"bad bool", func(d *SkillsData) { d.Skills[0].Values["nextActionAttack"] = "1" }, `"1" is not true or false`},
		{"bad element", func(d *SkillsData) { d.Skills[0].Values["element"] = "300" }, `"300" is not a byte`},
		{"bad affect limit", func(d *SkillsData) { d.Skills[0].Values["affectLimit"] = "5" }, "want min-max"},
		{"unknown effect", func(d *SkillsData) { d.Skills[0].Effects[ScopeGeneral][0].Name = "Nope" }, `unknown effect "Nope"`},
		{"items on another effect", func(d *SkillsData) {
			d.Skills[0].Effects[ScopeGeneral][0].Items = &RestorationGroups{All: []RestorationGroup{{Chance: 1, Items: []RestorationItem{{ID: 1, Count: 1}}}}}
		}, "items on an effect other than RestorationRandom"},
		{"passive block", func(d *SkillsData) { d.Skills[0].Effects[ScopePassive] = []EffectDef{} }, "passive effects are the GENERAL block"},
		{"table in a condition", func(d *SkillsData) {
			d.Skills[0].Conditions = []SkillConditions{{Cond: ConditionDef{Kind: CondPlayerHP, Value: "#power"}}}
		}, `conditions 1: player.hp "#power": table #power: Mobius takes no table here`},
		{"table in a function condition", func(d *SkillsData) {
			d.Skills[0].Effects[ScopeGeneral][0].Funcs = []StatFuncDef{{Stat: StatPowerAttack, Op: OpAdd, Order: 30, Value: "1",
				Cond: &ConditionDef{Kind: CondPlayerLevel, Value: "#power"}}}
		}, `function 1 (add pAtk): condition: player.level "#power": table #power: Mobius takes no table here`},
		{"bad function value", func(d *SkillsData) {
			d.Skills[0].Effects[ScopeGeneral][0].Funcs = []StatFuncDef{{Stat: StatPowerAttack, Op: OpAdd, Order: 30, Value: "#icon"}}
			d.Skills[0].Tables["#icon"] = []string{"a", "b", "c", "d", "e"}
		}, `"a" is not a number`},
		{"three routes", func(d *SkillsData) { d.Skills[0].Enchants = make([]EnchantRoute, 3) }, "3 enchant routes, at most 2"},
		{"duplicate", func(d *SkillsData) { d.Skills = append(d.Skills, windStrike()) }, "duplicate skill ID"},
		{"unsorted", func(d *SkillsData) { d.Skills = append(d.Skills, weaponMastery()) }, "skills are not sorted by ID"},
		{"no levels", func(d *SkillsData) { d.Skills[0].Levels = 0 }, "ID and levels must be positive"},
		{"no skills", func(d *SkillsData) { d.Skills = nil }, "no skills"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := SkillsData{Skills: []SkillTemplate{windStrike()}}
			tt.change(&d)
			_, err := NewSkills(d)
			if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %v, want one wrapping ErrInvalid with %q", err, tt.want)
			}
		})
	}
}

// testTrees are the cut-down Human mystic trees of the datagen sample
// (cmd/datagen/testdata/mobius/.../skillTrees): Human Wizard (11) teaches
// Wind Strike 2 again, which replaces the Human Mystic (10) entry in place.
func testTrees() []SkillTree {
	learn := func(id, level, getLevel, sp int) SkillLearn {
		return SkillLearn{Name: "x", SkillID: id, Level: level, GetLevel: getLevel, SP: sp, LearnedByNpc: true}
	}
	might := learn(1068, 1, 20, 1400)
	might.Items = []SkillItem{{ID: 5588, Count: 1}}
	mastery := SkillLearn{Name: "Weapon Mastery", SkillID: 141, Level: 2, GetLevel: 40, AutoGet: true}
	heir := SocialHeir
	return []SkillTree{
		{Type: TreeClass, ClassID: 10, ParentClassID: -1, Skills: []SkillLearn{learn(1177, 1, 1, 0), learn(1177, 2, 7, 150), learn(141, 1, 5, 50)}},
		{Type: TreeClass, ClassID: 11, ParentClassID: 10, Skills: []SkillLearn{learn(1177, 3, 20, 1400), learn(1177, 2, 20, 999), might}},
		{Type: TreeClass, ClassID: 12, ParentClassID: 11, Skills: []SkillLearn{learn(1177, 4, 40, 5000), learn(1068, 2, 40, 5000), mastery}},
		{Type: TreePledge, ClassID: -1, ParentClassID: -1, Skills: []SkillLearn{
			{Name: "Clan", SkillID: 141, Level: 3, GetLevel: 5, SP: 500, SocialClass: &heir, Items: []SkillItem{{ID: 57, Count: 1000}}},
			{Name: "Residence", SkillID: 1177, Level: 5, GetLevel: 5, ResidenceSkill: true, ResidenceIDs: []int{21, 34}},
		}},
		{Type: TreeFishing, ClassID: -1, ParentClassID: -1, Skills: []SkillLearn{{Name: "Fish", SkillID: 1068, Level: 3, GetLevel: 10, Races: []Race{RaceDwarf}}}},
	}
}

func refs(ls []SkillLearn) []SkillRef {
	out := make([]SkillRef, len(ls))
	for i := range ls {
		out[i] = ls[i].Ref()
	}
	return out
}

// A 2nd class tree holds the trees of its parents from the root, each in
// Mobius's HashMap order, a level taught again replacing the earlier entry
// in place. (Buckets of 16, hash ID*1021+level spread with h^(h>>>16):
// Wind Strike 1-4 fall in 4, 5, 10, 11, Weapon Mastery 1-2 in 8, 9, Might
// 1-2 in 13, 14.)
func TestSkillTreeInheritance(t *testing.T) {
	s := newTestSkills(t, SkillsData{Skills: []SkillTemplate{weaponMastery(), might(), windStrike()}, Trees: testTrees()})
	r := func(id, level int) SkillRef { return SkillRef{id, level} }
	for class, want := range map[int][]SkillRef{
		10: {r(1177, 1), r(1177, 2), r(141, 1)},
		11: {r(1177, 1), r(1177, 2), r(141, 1), r(1177, 3), r(1068, 1)},
		12: {r(1177, 1), r(1177, 2), r(141, 1), r(1177, 3), r(1068, 1), r(141, 2), r(1177, 4), r(1068, 2)},
	} {
		tree, ok := s.SkillTree(class)
		if !ok || !slices.Equal(refs(tree), want) {
			t.Errorf("SkillTree(%d) = %v, %v; want %v", class, refs(tree), ok, want)
		}
	}
	tree, _ := s.SkillTree(12)
	if tree[1].GetLevel != 20 || tree[1].SP != 999 {
		t.Errorf("Wind Strike 2 in the Sorcerer tree = %+v, want the Human Wizard entry", tree[1])
	}
	if tree[4].Items[0] != (SkillItem{ID: 5588, Count: 1}) || !tree[5].AutoGet {
		t.Errorf("Sorcerer tree entries = %+v", tree)
	}
	if _, ok := s.SkillTree(13); ok {
		t.Error("SkillTree(13) exists")
	}

	if len(s.OtherSkillTree(TreeFishing)) != 1 || s.OtherSkillTree(TreeClass) != nil || s.OtherSkillTree(TreeHero) != nil {
		t.Error("OtherSkillTree")
	}
	clan := s.ClanSkills()
	if len(clan) != 1 || clan[0].SocialClass == nil || *clan[0].SocialClass != SocialHeir {
		t.Errorf("ClanSkills = %+v", clan)
	}
	if res := s.ResidenceSkills(34); len(res) != 1 || res[0].SkillID != 1177 {
		t.Errorf("ResidenceSkills(34) = %+v", res)
	}
	if res := s.ResidenceSkills(1); len(res) != 0 {
		t.Errorf("ResidenceSkills(1) = %+v", res)
	}
}

// javaIntHashMapOrder follows a resize: 13 keys of skill 1 (hash
// 1021+level, below 2^16 so not spread) grow the table to 32 buckets, which
// puts level 17 (bucket 14) before levels 1 and 2 (buckets 30, 31), unlike
// 16 buckets (14, 14, 15).
func TestJavaIntHashMapOrder(t *testing.T) {
	levels := []int{1, 17, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	var skills []SkillLearn
	for _, l := range levels {
		skills = append(skills, SkillLearn{SkillID: 1, Level: l})
	}
	got, err := javaIntHashMapOrder(skills)
	if err != nil {
		t.Fatal(err)
	}
	want := []int{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 1, 0, 2}
	if !slices.Equal(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
	got, err = javaIntHashMapOrder(skills[:3])
	if err != nil || !slices.Equal(got, []int{0, 1, 2}) {
		t.Errorf("order of 3 = %v, %v; want buckets 14, 14, 15 in insertion order", got, err)
	}
}

func TestSkillTreeErrors(t *testing.T) {
	tests := []struct {
		name   string
		change func(ts []SkillTree) []SkillTree
		want   string
	}{
		{"unknown type", func(ts []SkillTree) []SkillTree { ts[3].Type = "transferSkillTree"; return ts }, "unknown type"},
		{"two trees of a class", func(ts []SkillTree) []SkillTree { ts[1].ClassID = 10; return ts }, "the class has two trees"},
		{"cycle", func(ts []SkillTree) []SkillTree { ts[0].ParentClassID = 12; return ts }, "the parent classes form a cycle"},
		{"missing skill level", func(ts []SkillTree) []SkillTree { ts[0].Skills[0].Level = 9; return ts }, "skill 1177 level 9: the skill level does not exist"},
		{"item count", func(ts []SkillTree) []SkillTree { ts[1].Skills[2].Items[0].Count = 0; return ts }, "item 5588 count 0"},
		{"class on another tree", func(ts []SkillTree) []SkillTree { ts[4].ClassID = 10; return ts }, "only class trees name classes"},
		{"taught twice", func(ts []SkillTree) []SkillTree {
			ts[0].Skills = append(ts[0].Skills, ts[0].Skills[0])
			return ts
		}, "taught twice by the tree"},
		{"residence IDs", func(ts []SkillTree) []SkillTree { ts[3].Skills[1].ResidenceSkill = false; return ts }, "residence IDs on a skill that is not a residence skill"},
		{"bad parent", func(ts []SkillTree) []SkillTree { ts[2].ParentClassID = 12; return ts }, "parent class 12 is not another player class"},
		{"prerequisite", func(ts []SkillTree) []SkillTree {
			ts[0].Skills[0].PreRequisites = []SkillRef{{ID: 4, Level: 1}}
			return ts
		}, "prerequisite skill 4 level 1 does not exist"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := SkillsData{Skills: []SkillTemplate{weaponMastery(), might(), windStrike()}, Trees: tt.change(testTrees())}
			_, err := NewSkills(d)
			if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %v, want one with %q", err, tt.want)
			}
		})
	}
}

func testArmorSet() ArmorSet {
	return ArmorSet{
		Chest: 397, Legs: []int{2387}, Feet: []int{62}, Shield: []int{20},
		Skills: []SkillRef{{141, 1}}, ShieldSkills: []SkillRef{{141, 2}}, Enchant6Skills: []SkillRef{{141, 3}}, STR: 4, CON: -1,
	}
}

func TestArmorSets(t *testing.T) {
	s := newTestSkills(t, SkillsData{Skills: []SkillTemplate{weaponMastery()}, ArmorSets: []ArmorSet{testArmorSet()}})
	set, ok := s.ArmorSet(397)
	if !ok || set.STR != 4 || set.CON != -1 {
		t.Fatalf("ArmorSet(397) = %+v, %v", set, ok)
	}
	if _, ok := s.ArmorSet(2387); ok {
		t.Error("ArmorSet of a legs item found")
	}
	if !set.ContainsAll(397, 2387, 0, 0, 62) || set.ContainsAll(397, 2387, 0, 0, 63) || set.ContainsAll(398, 2387, 0, 0, 62) {
		t.Error("ContainsAll: the set needs its chest, legs and feet, and no head or gloves")
	}
	if !set.ContainsShield(20) || set.ContainsShield(21) {
		t.Error("ContainsShield")
	}
	if !slices.Equal(set.ItemIDs(), []int{397, 2387, 62, 20}) {
		t.Errorf("ItemIDs = %v", set.ItemIDs())
	}

	for name, tt := range map[string]struct {
		sets []ArmorSet
		want string
	}{
		"duplicate chest": {[]ArmorSet{testArmorSet(), testArmorSet()}, "two sets of the same chest"},
		"missing skill":   {[]ArmorSet{{Chest: 1, Skills: []SkillRef{{141, 4}}}}, "skill 141 level 4 does not exist"},
		"no skills":       {[]ArmorSet{{Chest: 1}}, "armor set of chest 1: no skills"},
		"bad item":        {[]ArmorSet{{Chest: 1, Legs: []int{-2}, Skills: []SkillRef{{141, 1}}}}, "item IDs must be positive"},
	} {
		_, err := NewSkills(SkillsData{Skills: []SkillTemplate{weaponMastery()}, ArmorSets: tt.sets})
		if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: error %v, want one with %q", name, err, tt.want)
		}
	}
}

// The checks into the item and NPC data report missing references.
func TestCheckReferences(t *testing.T) {
	s := newTestSkills(t, SkillsData{
		Skills:    []SkillTemplate{weaponMastery(), might(), windStrike()},
		Trees:     testTrees(),
		ArmorSets: []ArmorSet{testArmorSet()},
	})
	items := []Item{{ID: 20}, {ID: 57}, {ID: 62}, {ID: 397}, {ID: 2387}, {ID: 5588}}
	if err := s.CheckItems(items); err != nil {
		t.Errorf("CheckItems: %v", err)
	}
	if err := s.CheckItems(items[1:]); err == nil || !strings.Contains(err.Error(), "armor set of chest 397: item 20 does not exist") {
		t.Errorf("CheckItems without the shield: %v", err)
	}
	if err := s.CheckItems(items[:5]); err == nil || !strings.Contains(err.Error(), "classSkillTree skill 1068 level 1: item 5588 does not exist") {
		t.Errorf("CheckItems without the tree item: %v", err)
	}

	withSkills := []Item{
		{ID: 1, Name: "ok", Skills: []ItemSkill{{ID: 141, Level: 3}, {ID: 1177, Level: 9}}}, // 9 takes level 5, as Mobius
		{ID: 726, Name: "Mana Drug", Skills: []ItemSkill{{ID: 10000, Level: 1}}},
	}
	if err := s.CheckItemSkills(withSkills); !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), `item 726 ("Mana Drug"): skill 10000 level 1 does not exist`) ||
		strings.Contains(err.Error(), "item 1 ") {
		t.Errorf("CheckItemSkills: %v", err)
	}

	npcs := []NpcTemplate{
		{ID: 1, Name: "ok", Skills: []NpcSkill{{ID: 1068, Level: 3}}, AI: NpcAI{PrimarySkillID: 1177}},
		{ID: 2, Name: "bad", Skills: []NpcSkill{{ID: 4416, Level: 1}}, AI: NpcAI{LongRangeSkillID: 5},
			Parameters: NpcParameters{Skills: map[string]NpcSkill{"Magic": {ID: 141, Level: 0}}}},
	}
	err := s.CheckNpcSkills(npcs)
	for _, want := range []string{"NPC 2 (\"bad\"): skill 4416 level 1 does not exist", "parameter Magic: skill 141 level 0 does not exist", "AI skill 5 does not exist"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("CheckNpcSkills: %v, want %q", err, want)
		}
	}
	if err != nil && strings.Contains(err.Error(), "NPC 1 ") {
		t.Errorf("CheckNpcSkills reports the valid NPC: %v", err)
	}
}

// LoadSkillsFS reads the files in either form and refuses a missing one.
func TestLoadSkillsFS(t *testing.T) {
	d := SkillsData{
		Skills:    []SkillTemplate{weaponMastery(), might(), windStrike()},
		Trees:     testTrees(),
		ArmorSets: []ArmorSet{testArmorSet()},
	}
	m := fstest.MapFS{}
	for name, v := range map[string]any{SkillsFile: d.Skills, SkillTreesFile: d.Trees, ArmorSetsFile: d.ArmorSets} {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		m[name] = &fstest.MapFile{Data: b}
	}
	// The skills file gzipped, as datagen stores big files.
	var z bytes.Buffer
	zw := gzip.NewWriter(&z)
	if _, err := zw.Write(m[SkillsFile].Data); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	m[SkillsFile+".gz"] = &fstest.MapFile{Data: z.Bytes()}
	delete(m, SkillsFile)

	s, err := LoadSkillsFS(m)
	if err != nil {
		t.Fatalf("LoadSkillsFS: %v", err)
	}
	if sk := mustSkill(t, s, 1068, 141); sk.MPConsume != 27 {
		t.Errorf("Might 141 after a JSON round trip: mp %d, want 27", sk.MPConsume)
	}
	if tree, ok := s.SkillTree(12); !ok || len(tree) != 8 {
		t.Errorf("SkillTree(12) after a JSON round trip: %d entries, %v", len(tree), ok)
	}

	delete(m, ArmorSetsFile)
	if _, err := LoadSkillsFS(m); err == nil || !strings.Contains(err.Error(), ArmorSetsFile) {
		t.Errorf("LoadSkillsFS without %s: %v", ArmorSetsFile, err)
	}
	if _, err := LoadSkills(t.TempDir()); err == nil || !strings.Contains(err.Error(), "data folder") {
		t.Errorf("LoadSkills of an empty folder: %v", err)
	}
}

// The generated enums of the skill data parse and print their Mobius names.
func TestSkillEnums(t *testing.T) {
	if NumAbnormalTypes <= 256 {
		t.Errorf("NumAbnormalTypes = %d; the uint16 enum is for more than 256", NumAbnormalTypes)
	}
	if a, ok := ParseAbnormalType("PA_UP"); !ok || a != AbnormalPaUp || a.String() != "PA_UP" {
		t.Errorf("ParseAbnormalType(PA_UP) = %v, %v", a, ok)
	}
	var e EffectName
	if err := e.UnmarshalText([]byte("MagicalDamage")); err != nil || e != EffectMagicalDamage {
		t.Errorf("UnmarshalText(MagicalDamage) = %v, %v", e, err)
	}
	if err := e.UnmarshalText([]byte("Nope")); err == nil {
		t.Error("UnmarshalText(Nope) succeeded")
	}
	if !InstanceChest.Is(InstanceAttackable) || InstanceAttackable.Is(InstanceChest) {
		t.Error("InstanceType.Is: a chest is an attackable, not the other way")
	}
	if p, ok := InstanceAttackable.Parent(); !ok || p != InstanceNpc {
		t.Errorf("Attackable.Parent() = %v, %v", p, ok)
	}
	if VisualTurnFlee.Mask() != 0x80000000 {
		t.Errorf("TURN_FLEE mask %#x", VisualTurnFlee.Mask())
	}
	if TraitSword.Kind() != 1 || TraitPoison.Kind() != 3 || TraitBugWeakness.Kind() != 2 {
		t.Error("TraitType.Kind")
	}
	if k, ok := ParseConditionKind("player.levelrange"); !ok || k != CondPlayerLevelrange {
		t.Errorf("ParseConditionKind(player.levelrange) = %v, %v", k, ok)
	}
	if s, ok := ParseEffectScope("GENERAL"); !ok || s != ScopeGeneral {
		t.Errorf("ParseEffectScope = %v, %v", s, ok)
	}
}
