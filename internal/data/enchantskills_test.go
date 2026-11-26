package data

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestNewEnchantSkillTree(t *testing.T) {
	good := EnchantSkill{ID: 1, Level: 101, BaseLevel: 37, MinSkillLevel: 37, SP: 306000, Exp: 3060000, Rates: [5]int{82, 92, 97, 97, 97}}
	tests := []struct {
		name string
		edit func(e *EnchantSkill)
		want string
	}{
		{"skill", func(e *EnchantSkill) { e.ID = 0 }, "invalid skill ID"},
		{"level", func(e *EnchantSkill) { e.Level = 37 }, "above 100"},
		{"base", func(e *EnchantSkill) { e.BaseLevel = 0 }, "base level 0"},
		{"cost", func(e *EnchantSkill) { e.SP = -1 }, "negative cost"},
		{"rate", func(e *EnchantSkill) { e.Rates[4] = 101 }, "rate 101"},
	}
	for _, tt := range tests {
		e := good
		tt.edit(&e)
		if _, err := NewEnchantSkillTree([]EnchantSkill{e}); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: error %v, want %q", tt.name, err, tt.want)
		}
	}
	if _, err := NewEnchantSkillTree([]EnchantSkill{good, good}); err == nil || !strings.Contains(err.Error(), "listed twice") {
		t.Errorf("twice: %v", err)
	}
	var none *EnchantSkillTree
	if none.Find(1, 101) != nil {
		t.Error("a nil tree found something")
	}
}

// The committed data: 14,551 enchant levels; spot checks against the XML.
func TestEnchantSkillTreeData(t *testing.T) {
	tree, err := LoadEnchantSkillTree(filepath.Join("..", "..", "data"))
	if err != nil {
		t.Fatal(err)
	}
	if n := len(tree.Skills); n != 14551 {
		t.Errorf("%d enchant levels, want 14551", n)
	}
	// <skill id="1" level="130" ... baseLevel="37" minSkillLevel="129" sp="2597000" exp="25970000" rate76="0" ... rate80="1" />
	if e := tree.Find(1, 130); e == nil || e.MinSkillLevel != 129 || e.SP != 2597000 || e.Exp != 25970000 || e.Rate(76) != 0 || e.Rate(80) != 1 {
		t.Errorf("Triple Slash +30 = %+v", e)
	}
	if e := tree.Find(1, 141); e == nil || e.MinSkillLevel != 37 || e.Rate(77) != 92 {
		t.Errorf("Triple Slash cost +1 = %+v", e)
	}
}
