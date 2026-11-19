package data

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// loadRepoSkills loads the committed skill data (see worldRepoData).
func loadRepoSkills(t *testing.T) *Skills {
	t.Helper()
	if _, err := os.Stat(filepath.Join(worldRepoData, SkillsFile)); err != nil {
		t.Skipf("no skill data in %s: %v", worldRepoData, err)
	}
	s, err := LoadSkills(worldRepoData)
	if err != nil {
		t.Fatalf("LoadSkills(%s): %v", worldRepoData, err)
	}
	return s
}

// Spot checks of the committed skill data against the Mobius XML it was
// generated from (the expected values are copied from the files named in
// each check).
func TestRepoSkillsSize(t *testing.T) {
	s := loadRepoSkills(t)
	levels, enchant := 0, 0
	for i := range s.Skills {
		levels += s.Skills[i].Levels
		enchant += len(s.Skills[i].AllLevels()) - s.Skills[i].Levels
	}
	// Lower bounds, so a datapack update does not break the test; the
	// Interlude datapack of 2026-11 has 2694 skills (not custom) with 15311
	// levels and 14550 enchant levels, 95 trees and 51 armor sets.
	for name, n := range map[string][2]int{
		"skills":         {len(s.Skills), 2600},
		"levels":         {levels, 15000},
		"enchant levels": {enchant, 14000},
		"skill trees":    {len(s.Trees), 90},
		"armor sets":     {len(s.ArmorSets), 50},
	} {
		if n[0] < n[1] {
			t.Errorf("%d %s, want at least %d", n[0], name, n[1])
		}
	}
	if _, ok := s.Template(10000); ok {
		t.Error("custom skill 10000 is in the data")
	}
}

// stats/skills/01100-01199.xml, skill 1177.
func TestRepoWindStrike(t *testing.T) {
	s := loadRepoSkills(t)
	for _, tt := range []struct {
		level, mp, mpInit, magicLevel, effectPoint int
		power                                      float64
	}{
		{1, 7, 2, 1, -92, 12},
		{2, 7, 2, 4, -106, 13},
		{5, 12, 3, 14, -162, 21},
	} {
		sk := mustSkill(t, s, 1177, tt.level)
		if sk.MPConsume != tt.mp || sk.MPInitialConsume != tt.mpInit || sk.MagicLevel != tt.magicLevel ||
			sk.EffectPoint != tt.effectPoint || sk.Power != tt.power {
			t.Errorf("Wind Strike %d: mp %d+%d, magic level %d, effect point %d, power %v", tt.level,
				sk.MPConsume, sk.MPInitialConsume, sk.MagicLevel, sk.EffectPoint, sk.Power)
		}
		if sk.Name != "Wind Strike" || sk.OperateType != OperateA1 || sk.TargetType != TargetOne || !sk.IsMagic() ||
			sk.CastRange != 600 || sk.EffectRange != 1100 || sk.HitTime != 4000 || sk.ReuseDelay != 6000 ||
			sk.Element != 2 || sk.ElementPower != 20 || sk.Icon != "icon.skill1177" {
			t.Errorf("Wind Strike %d = %+v", tt.level, sk)
		}
		if e := sk.Effects[ScopeGeneral]; len(e) != 1 || e[0].Name != EffectMagicalDamage {
			t.Errorf("Wind Strike %d effects = %+v", tt.level, sk.Effects)
		}
	}
	if s.MaxLevel(1177) != 5 {
		t.Errorf("Wind Strike max level %d", s.MaxLevel(1177))
	}
}

// stats/skills/00000-00099.xml, skill 3.
func TestRepoPowerStrike(t *testing.T) {
	s := loadRepoSkills(t)
	sk := mustSkill(t, s, 3, 9)
	if sk.Power != 70 || sk.MPConsume != 19 || sk.MagicLevel != 15 || sk.EffectPoint != -86 || sk.CastRange != 40 ||
		sk.CoolTime != 720 || sk.HitTime != 1080 || sk.ReuseDelay != 13000 || !sk.NextActionAttack || !sk.OverHit ||
		!sk.IsPhysical() || sk.TargetType != TargetOne {
		t.Errorf("Power Strike 9 = %+v", sk)
	}
	if len(sk.Conditions) != 1 || sk.Conditions[0].MsgID != 113 || !sk.Conditions[0].AddName ||
		sk.Conditions[0].Cond.Kind != CondUsingKind || sk.Conditions[0].Cond.Int != WeaponSword.Mask()|WeaponBlunt.Mask() {
		t.Errorf("Power Strike conditions = %+v", sk.Conditions)
	}
	if e := sk.Effects[ScopeGeneral]; len(e) != 1 || e[0].Name != EffectPhysicalDamage {
		t.Errorf("Power Strike effects = %+v", sk.Effects)
	}
}

// stats/skills/01000-01099.xml, skill 1068: a buff with stat modifiers and
// two enchant routes.
func TestRepoMight(t *testing.T) {
	s := loadRepoSkills(t)
	sk := mustSkill(t, s, 1068, 3)
	if sk.AbnormalType != AbnormalPaUp || sk.AbnormalLevel != 3 || sk.AbnormalTime != 1200 || sk.OperateType != OperateA2 ||
		sk.MPConsume != 28 || sk.MPInitialConsume != 7 || sk.EffectPoint != 379 || sk.MagicLevel != 40 {
		t.Errorf("Might 3 = %+v", sk)
	}
	want := []StatFunc{{Stat: StatPowerAttack, Op: OpMul, Order: 20, Value: 1.15}}
	if e := sk.Effects[ScopeGeneral]; len(e) != 1 || e[0].Name != EffectBuff || !slices.Equal(e[0].Funcs, want) {
		t.Errorf("Might 3 effects = %+v, want Buff %+v", sk.Effects, want)
	}
	if e := mustSkill(t, s, 1068, 1).Effects[ScopeGeneral]; len(e) != 1 || e[0].Funcs[0].Value != 1.08 {
		t.Errorf("Might 1 effects = %+v, want pAtk x1.08", e)
	}
	if sk := mustSkill(t, s, 1068, 101); sk.AbnormalTime != 1240 || sk.MagicLevel != 76 || sk.MPConsume != 28 {
		t.Errorf("Might 101: abnormal time %d, magic level %d, mp %d; want 1240, 76, 28", sk.AbnormalTime, sk.MagicLevel, sk.MPConsume)
	}
	if sk := mustSkill(t, s, 1068, 170); sk.AbnormalTime != 1200 || sk.MagicLevel != 85 || sk.MPConsume != 14 || sk.MPInitialConsume != 3 {
		t.Errorf("Might 170: abnormal time %d, magic level %d, mp %d+%d; want 1200, 85, 14+3", sk.AbnormalTime, sk.MagicLevel, sk.MPConsume, sk.MPInitialConsume)
	}
}

// stats/skills/00100-00199.xml, skill 141, and 00200-00299.xml, skill 227:
// passive masteries.
func TestRepoMasteries(t *testing.T) {
	s := loadRepoSkills(t)
	wm := mustSkill(t, s, 141, 2)
	want := []StatFunc{
		{Stat: StatPowerAttack, Op: OpMul, Order: 20, Value: 1.085},
		{Stat: StatPowerAttack, Op: OpAdd, Order: 30, Value: 3},
	}
	if e := wm.Effects[ScopePassive]; !wm.IsPassive() || len(e) != 1 || e[0].Name != EffectBuff || !slices.Equal(e[0].Funcs, want) {
		t.Errorf("Weapon Mastery 2 = passive %v, effects %+v; want Buff %+v", wm.IsPassive(), wm.Effects, want)
	}

	lam := mustSkill(t, s, 227, 50)
	f := lam.Effects[ScopePassive][0].Funcs
	if lam.MagicLevel != 74 || len(f) != 2 || f[0].Stat != StatPowerDefence || f[0].Value != 81.3 || f[1].Stat != StatEvasionRate || f[1].Value != 6 {
		t.Fatalf("Light Armor Mastery 50: magic level %d, functions %+v", lam.MagicLevel, f)
	}
	for _, fn := range f {
		if fn.Op != OpAdd || fn.Order != 30 || fn.Cond == nil || fn.Cond.Kind != CondUsingKind || fn.Cond.Int != ArmorLight.Mask() {
			t.Errorf("Light Armor Mastery function %+v (condition %+v), want add when wearing light armor", fn, fn.Cond)
		}
	}
}

// stats/skills/00000-00099.xml, skill 8: two condition blocks, the second
// with a table; 05000-05099.xml, skill 5008: the effect name is a table.
func TestRepoTableConditionsAndEffects(t *testing.T) {
	s := loadRepoSkills(t)
	sf := mustSkill(t, s, 8, 4)
	if len(sf.Conditions) != 2 || sf.Conditions[1].MsgID != 324 ||
		!equalConditions(sf.Conditions[1].Cond, Condition{Kind: CondNot, Conds: []Condition{{Kind: CondPlayerCharges, Int: 4}}}) {
		t.Errorf("Sonic Focus 4 conditions = %+v", sf.Conditions)
	}
	for level, want := range map[int]EffectName{1: EffectBuff, 4: EffectBuff, 5: EffectStun} {
		if e := mustSkill(t, s, 5008, level).Effects[ScopeGeneral]; len(e) == 0 || e[0].Name != want {
			t.Errorf("skill 5008 level %d effects = %+v, want %v first", level, e, want)
		}
	}
}

// stats/skills/02200-02299.xml, skill 2200: RestorationRandom by level.
func TestRepoRestorationRandom(t *testing.T) {
	s := loadRepoSkills(t)
	g := mustSkill(t, s, 2200, 3).Effects[ScopeGeneral][0].Items
	want := []RestorationGroup{
		{Chance: 34.5, Items: []RestorationItem{{ID: 6908, Count: 24}}},
		{Chance: float64(float32(31.5428)), Items: []RestorationItem{{ID: 7609, Count: 26}}},
	}
	if len(g) != len(want) {
		t.Fatalf("Jade Treasure Box 3 groups = %+v", g)
	}
	for i := range want {
		if g[i].Chance != want[i].Chance || !slices.Equal(g[i].Items, want[i].Items) {
			t.Errorf("group %d = %+v, want %+v", i, g[i], want[i])
		}
	}
}

// stats/players/skillTrees: the Sorcerer's tree holds the Human Mystic
// (StartingClass/HumanMystic.xml) and Human Wizard (1stClass/
// HumanWizard.xml) trees and its own (2ndClass/Sorcerer.xml).
func TestRepoSorcererTree(t *testing.T) {
	s := loadRepoSkills(t)
	tree, ok := s.SkillTree(int(ClassSorcerer))
	if !ok || len(tree) < 500 || len(tree) > 61+122+367 {
		t.Fatalf("Sorcerer tree: %d entries, %v; want 500 to 550", len(tree), ok)
	}
	find := func(id, level int) *SkillLearn {
		for i := range tree {
			if tree[i].SkillID == id && tree[i].Level == level {
				return &tree[i]
			}
		}
		return nil
	}
	if l := find(1177, 1); l == nil || l.GetLevel != 1 || !l.AutoGet {
		t.Errorf("Wind Strike 1 (Human Mystic) = %+v", l)
	}
	if l := find(1184, 5); l == nil || l.GetLevel != 20 || l.SP != 1400 || !l.LearnedByNpc {
		t.Errorf("Ice Bolt 5 (Human Wizard) = %+v", l)
	}
	if l := find(249, 10); l == nil || l.GetLevel != 40 || l.SP != 11000 {
		t.Errorf("Weapon Mastery 10 (Sorcerer) = %+v", l)
	}
	// The root class's entries come first.
	mystic, _ := s.SkillTree(int(ClassMage))
	if len(mystic) == 0 || tree[0].Ref() != mystic[0].Ref() {
		t.Errorf("the Sorcerer tree starts with %+v, want the Human Mystic tree's first entry", tree[0])
	}
	if _, ok := s.SkillTree(int(ClassWizard)); !ok {
		t.Error("no Human Wizard tree")
	}
}

// stats/players/skillTrees/pledgeSkillTree.xml and fishingSkillTree.xml.
func TestRepoOtherTrees(t *testing.T) {
	s := loadRepoSkills(t)
	var vitality *SkillLearn
	for _, l := range s.ClanSkills() {
		if l.SkillID == 370 && l.Level == 1 {
			vitality = &l
		}
	}
	if vitality == nil || vitality.GetLevel != 5 || vitality.SP != 500 || vitality.SocialClass == nil ||
		*vitality.SocialClass != SocialHeir || !slices.Equal(vitality.Items, []SkillItem{{ID: 8166, Count: 1}}) {
		t.Errorf("Clan Vitality 1 = %+v", vitality)
	}
	dwarf := false
	for _, l := range s.OtherSkillTree(TreeFishing) {
		if slices.Contains(l.Races, RaceDwarf) {
			dwarf = true
		}
	}
	if !dwarf {
		t.Error("no fishing skill for dwarves only")
	}
}

// stats/armorsets/c_grade.xml, the sets of chests 397 and 354.
func TestRepoArmorSets(t *testing.T) {
	s := loadRepoSkills(t)
	mithril, ok := s.ArmorSet(397)
	if !ok || !slices.Equal(mithril.Legs, []int{2387}) || !slices.Equal(mithril.Feet, []int{62}) ||
		!slices.Equal(mithril.Skills, []SkillRef{{ID: 3006, Level: 1}, {ID: 3508, Level: 1}}) ||
		!slices.Equal(mithril.Enchant6Skills, []SkillRef{{ID: 3615, Level: 1}}) || len(mithril.Shield) != 0 {
		t.Errorf("set of chest 397 = %+v, %v", mithril, ok)
	}
	chain, ok := s.ArmorSet(354)
	if !ok || !chain.ContainsAll(354, 381, 2413, 0, 0) || !chain.ContainsShield(2495) ||
		!slices.Equal(chain.ShieldSkills, []SkillRef{{ID: 3545, Level: 1}}) {
		t.Errorf("set of chest 354 = %+v, %v", chain, ok)
	}
}
