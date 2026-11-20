package npcai

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/data"
)

func TestJavaHashMapOrder(t *testing.T) {
	seq := func(from, n, step int) []int {
		out := make([]int, n)
		for i := range out {
			out[i] = from + i*step
		}
		return out
	}
	for _, tt := range []struct {
		name string
		keys []int
		want []int
	}{
		{"empty", nil, []int{}},
		// Capacity 16: by key mod 16, then by insertion.
		{"Catherok", []int{4072, 4085, 4279, 4408, 4410, 4411, 4412, 4413, 4416},
			[]int{4416, 4085, 4279, 4072, 4408, 4410, 4411, 4412, 4413}},
		{"a repeated key keeps its first place", []int{4078, 4065, 4078}, []int{4065, 4078}},
		// 13 keys pass the threshold of 12: capacity 32.
		{"resize", append([]int{33, 17, 1}, seq(100, 10, 1)...),
			append(append([]int{33, 1}, seq(100, 10, 1)...), 17)},
		// The 9th key of one bucket doubles a table below 64 (treeifyBin).
		{"treeify resize", seq(0, 9, 16), []int{0, 32, 64, 96, 128, 16, 48, 80, 112}},
		// Negative and large keys spread their high bits.
		{"spread", []int{1 << 16, 1}, []int{1 << 16, 1}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := javaHashMapOrder(tt.keys); !slices.Equal(got, tt.want) {
				t.Errorf("order %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScopesOf(t *testing.T) {
	d := repo(t)
	// NPC Heal; NPC Flamestrike (range 500); Shock (an aura debuff without
	// a cast range); Decrease Atk. Spd.; NPC Chant of Life; NPC MP Drain;
	// Boom Attack; Sleep (continuous comes first); Resurrection.
	for _, tt := range []struct {
		id, level int
		want      []Scope
	}{
		{4065, 4, []Scope{ScopeGeneral, ScopeHeal}},
		{4078, 4, []Scope{ScopeGeneral, ScopeAttack, ScopeUniversal, ScopeLongRange}},
		{4072, 4, []Scope{ScopeGeneral, ScopeDebuff, ScopeCOT, ScopeShortRange}},
		{4038, 4, []Scope{ScopeGeneral, ScopeDebuff, ScopeCOT, ScopeLongRange}},
		{4097, 5, []Scope{ScopeGeneral, ScopeBuff}},
		{4039, 5, []Scope{ScopeGeneral, ScopeAttack, ScopeUniversal, ScopeLongRange}},
		{4139, 6, []Scope{ScopeSuicide}},
		{4046, 4, []Scope{ScopeGeneral, ScopeDebuff, ScopeCOT, ScopeLongRange}},
		{1016, 1, []Scope{ScopeGeneral, ScopeRes}},
	} {
		sk := d.skill(t, tt.id, tt.level)
		if got := ScopesOf(sk); !slices.Equal(got, tt.want) {
			t.Errorf("%v: scopes %v, want %v", sk, got, tt.want)
		}
	}
}

func TestProfileRealTemplates(t *testing.T) {
	d := repo(t)
	for _, tt := range []struct {
		npc   int
		lists map[Scope][]int
	}{
		{21035, map[Scope][]int{ // Catherok: Shock
			ScopeGeneral: {4072}, ScopeDebuff: {4072}, ScopeCOT: {4072}, ScopeShortRange: {4072},
		}},
		{21039, map[Scope][]int{ // Snipe Cohort: heal and Flamestrike, heal first (bucket 1)
			ScopeGeneral: {4065, 4078}, ScopeHeal: {4065}, ScopeAttack: {4078}, ScopeUniversal: {4078}, ScopeLongRange: {4078},
		}},
		{21040, map[Scope][]int{ // Soldier of Darkness: heal and a debuff
			ScopeGeneral: {4065, 4038}, ScopeHeal: {4065}, ScopeDebuff: {4038}, ScopeCOT: {4038}, ScopeLongRange: {4038},
		}},
		{21059, map[Scope][]int{ // Beast Guardian: Chant of Life and MP Drain
			ScopeGeneral: {4097, 4039}, ScopeBuff: {4097}, ScopeAttack: {4039}, ScopeUniversal: {4039}, ScopeLongRange: {4039},
		}},
	} {
		p := d.profile(t, tt.npc)
		for s := range NumScopes {
			if got := ids(p.Skills(s)); !slices.Equal(got, tt.lists[s]) && (len(got) > 0 || len(tt.lists[s]) > 0) {
				t.Errorf("NPC %d %v: %v, want %v", tt.npc, s, got, tt.lists[s])
			}
		}
		if p.MinSkillChance != 7 || p.MaxSkillChance != 15 {
			t.Errorf("NPC %d: skill chances %d-%d, want 7-15", tt.npc, p.MinSkillChance, p.MaxSkillChance)
		}
	}
	if p := d.profile(t, 21040); !slices.Equal(p.Clans, []string{"UNDEAD"}) || p.AIType != AIFighter {
		t.Errorf("Soldier of Darkness: clans %v, AI type %d", p.Clans, p.AIType)
	}
}

// Every template of the data builds without a warning.
func TestProfileAllTemplates(t *testing.T) {
	d := repo(t)
	withSkills := 0
	for id, tmpl := range d.npcs {
		p, warnings := NewProfile(tmpl, d.cat)
		if len(warnings) > 0 {
			t.Errorf("NPC %d: %v", id, warnings)
		}
		if len(p.Skills(ScopeGeneral)) > 0 || len(p.Skills(ScopeSuicide)) > 0 {
			withSkills++
		}
	}
	if withSkills < 1000 {
		t.Errorf("%d templates use skills, want over 1000", withSkills)
	}
	t.Logf("%d of %d templates have active skills", withSkills, len(d.npcs))
}

func TestProfileMissingSkill(t *testing.T) {
	d := repo(t)
	tmpl := data.NpcTemplate{ID: 1, AI: data.NpcAI{Type: "MAGE"}, Skills: []data.NpcSkill{{ID: 999999, Level: 1}, {ID: 4065, Level: 99}}}
	p, warnings := NewProfile(&tmpl, d.cat)
	if len(warnings) != 1 {
		t.Fatalf("warnings %v, want one for skill 999999", warnings)
	}
	// A level above the highest takes the highest (SkillData.getSkill).
	if got := p.Skills(ScopeHeal); len(got) != 1 || got[0].Level != 12 {
		t.Errorf("heal list %v, want NPC Heal level 12", got)
	}
	if p.AIType != AIMage {
		t.Errorf("AI type %d, want MAGE", p.AIType)
	}
}

func TestIsClan(t *testing.T) {
	for _, tt := range []struct {
		mine, theirs []string
		want         bool
	}{
		{nil, []string{"ORC"}, false},
		{[]string{"ORC"}, nil, false},
		{[]string{"ORC"}, []string{"ELF", "ORC"}, true},
		{[]string{"ORC"}, []string{"ELF"}, false},
		{[]string{"ALL"}, []string{"ELF"}, true},
		// Not symmetric: only the first one's ALL counts.
		{[]string{"ELF"}, []string{"ALL"}, false},
	} {
		p := &Profile{Clans: tt.mine}
		if got := p.IsClan(tt.theirs); got != tt.want {
			t.Errorf("%v isClan %v = %v, want %v", tt.mine, tt.theirs, got, tt.want)
		}
	}
}
