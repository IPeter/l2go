package data

import (
	"errors"
	"path/filepath"
	"testing"
)

// TestAugmentRepoData loads the committed augmentation files and spot
// checks them against the Mobius XML and Java tables.
func TestAugmentRepoData(t *testing.T) {
	a, err := LoadAugment(filepath.Join("..", "..", "data"))
	if err != nil {
		t.Fatal(err)
	}
	if ls, ok := a.LifeStone(8723); !ok || ls != (LifeStone{ItemID: 8723, PlayerLevel: 46}) {
		t.Errorf("LifeStone(8723) = %+v, %v", ls, ok)
	}
	if ls, ok := a.LifeStone(8752); !ok || ls != (LifeStone{ItemID: 8752, Grade: LifeStoneGradeHigh, Level: 9, PlayerLevel: 76}) {
		t.Errorf("LifeStone(8752) = %+v, %v", ls, ok)
	}
	// options/00000-00099.xml: <option id="1"><stats><add stat="pDef" val="15.45"/>.
	if o, ok := a.Option(1); !ok || len(o.Funcs) != 1 || o.Funcs[0] != (AugmentFunc{Stat: StatPowerDefence, Op: OpAdd, Order: 30, Value: 15.45}) {
		t.Errorf("Option(1) = %+v, %v", o, ok)
	}
	// options/14500-14599.xml: o_s_46_2s_8 casts Charm (3081) on 1 % of hits.
	if o, ok := a.Option(14568); !ok || len(o.Triggers) != 1 || o.Triggers[0] != (AugmentTrigger{Type: AugmentTriggerAttack, ID: 3081, Level: 1, Chance: 1}) {
		t.Errorf("Option(14568) = %+v, %v", o, ok)
	}
	// retailchances.xml starts with the warrior's first stone, variation 1.
	g := a.Chances[0]
	if g.Weapon != AugmentWeaponWarrior || g.Stone != 8723 || g.Variation != 1 || g.Category != 100 || len(g.Augments) != 91 ||
		g.Augments[0] != (AugmentChance{ID: 1, Chance: 1.1}) || g.Augments[90] != (AugmentChance{ID: 91, Chance: 1}) {
		t.Errorf("first chance group %+v", g)
	}
	var mage, warrior int
	for _, g := range a.Chances {
		if g.Weapon == AugmentWeaponMage {
			mage++
		} else {
			warrior++
		}
	}
	if mage != 200 || warrior != 200 {
		t.Errorf("%d mage and %d warrior groups, want 200 each (40 stones × 5 categories)", mage, warrior)
	}

	// Every skill an option names is in the skill data.
	sk, err := LoadSkills(filepath.Join("..", "..", "data"))
	if err != nil {
		t.Fatal(err)
	}
	missing := 0
	for _, o := range a.Options {
		refs := []SkillRef{}
		for _, r := range []*SkillRef{o.ActiveSkill, o.PassiveSkill} {
			if r != nil {
				refs = append(refs, *r)
			}
		}
		for _, tr := range o.Triggers {
			refs = append(refs, SkillRef{ID: tr.ID, Level: tr.Level})
		}
		for _, r := range refs {
			if _, ok := sk.Skill(r.ID, r.Level); !ok {
				missing++
				if missing <= 5 {
					t.Errorf("option %d: skill %d level %d is not in the skill data", o.ID, r.ID, r.Level)
				}
			}
		}
	}
}

func TestNewAugmentRejects(t *testing.T) {
	good := func() AugmentData {
		return AugmentData{
			Refine: AugmentRefine{
				Source:     []string{"AbstractRefinePacket.java"},
				LifeStones: []LifeStone{{ItemID: 8723, PlayerLevel: 46}, {ItemID: 8724, Level: 1, PlayerLevel: 49}},
				Gemstones: []AugmentGemstone{
					{Grade: CrystalC, ItemID: 2130, Count: 20}, {Grade: CrystalB, ItemID: 2130, Count: 30},
					{Grade: CrystalA, ItemID: 2131, Count: 20}, {Grade: CrystalS, ItemID: 2131, Count: 25},
				},
				CancelFees: []AugmentCancelFee{
					{Grade: CrystalC, Below: 1720, Fee: 95000}, {Grade: CrystalC, Fee: 210000},
					{Grade: CrystalB, Fee: 270000}, {Grade: CrystalA, Fee: 420000}, {Grade: CrystalS, Fee: 480000},
				},
				Npcs: []int{30298, 30678},
			},
			Options: []AugmentOption{
				{ID: 1, Funcs: []AugmentFunc{{Stat: StatPowerDefence, Op: OpAdd, Order: 30, Value: 1}}},
				{ID: 2, ActiveSkill: &SkillRef{ID: 3203, Level: 1}, Triggers: []AugmentTrigger{{Type: AugmentTriggerCritical, ID: 3109, Level: 1, Chance: 5}}},
			},
			Chances: []AugmentChanceGroup{{Weapon: AugmentWeaponWarrior, Stone: 8723, Variation: 1, Category: 100, Augments: []AugmentChance{{ID: 1, Chance: 50}, {ID: 2, Chance: 50}}}},
		}
	}
	if _, err := NewAugment(good()); err != nil {
		t.Fatalf("good data: %v", err)
	}
	bad := map[string]func(d *AugmentData){
		"no source":           func(d *AugmentData) { d.Refine.Source = nil },
		"duplicate stone":     func(d *AugmentData) { d.Refine.LifeStones[1].ItemID = 8723 },
		"stone level 10":      func(d *AugmentData) { d.Refine.LifeStones[1].Level = 10 },
		"stone grade 4":       func(d *AugmentData) { d.Refine.LifeStones[1].Grade = 4 },
		"no S gemstone":       func(d *AugmentData) { d.Refine.Gemstones = d.Refine.Gemstones[:3] },
		"D gemstone":          func(d *AugmentData) { d.Refine.Gemstones[0].Grade = CrystalD },
		"open fee not last":   func(d *AugmentData) { d.Refine.CancelFees[0].Below = 0 },
		"no fees for S":       func(d *AugmentData) { d.Refine.CancelFees = d.Refine.CancelFees[:4] },
		"NPCs unsorted":       func(d *AugmentData) { d.Refine.Npcs = []int{30678, 30298} },
		"unsorted options":    func(d *AugmentData) { d.Options[0].ID = 3 },
		"option too big":      func(d *AugmentData) { d.Options[1].ID = 70000 },
		"bad trigger":         func(d *AugmentData) { d.Options[1].Triggers[0].Type = "always" },
		"skill level 0":       func(d *AugmentData) { d.Options[1].ActiveSkill.Level = 0 },
		"func order negative": func(d *AugmentData) { d.Options[0].Funcs[0].Order = -1 },
		"unknown weapon kind": func(d *AugmentData) { d.Chances[0].Weapon = "archer" },
		"not a life stone":    func(d *AugmentData) { d.Chances[0].Stone = 1 },
		"unknown option":      func(d *AugmentData) { d.Chances[0].Augments[0].ID = 99 },
		"empty group":         func(d *AugmentData) { d.Chances[0].Augments = nil },
	}
	for name, change := range bad {
		d := good()
		change(&d)
		if _, err := NewAugment(d); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: NewAugment = %v, want ErrInvalid", name, err)
		}
	}
}
