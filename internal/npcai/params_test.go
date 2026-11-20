package npcai

import (
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
)

// gated returns a template with NPC Heal (4065) in the given slot and the
// parameters.
func gated(slot string, params map[string]string) data.NpcTemplate {
	return data.NpcTemplate{
		ID:     2,
		Skills: []data.NpcSkill{{ID: 4065, Level: 4}},
		Parameters: data.NpcParameters{
			Params: params,
			Skills: map[string]data.NpcSkill{"Skill0" + slot + "_ID": {ID: 4065, Level: 4}},
		},
	}
}

func TestAllowsCast(t *testing.T) {
	d := repo(t)
	heal := d.skill(t, 4065, 4)
	other := d.skill(t, 4078, 4)
	for _, tt := range []struct {
		name   string
		slot   string
		params map[string]string
		dist   int32   // caster to target, on X
		hp     float64 // caster's and target's HP of 1000
		draws  []rnd.Draw
		want   bool
	}{
		{"no probability, no hint", "1", map[string]string{"Skill01_Type": "3"}, 100, 1000, nil, false},
		{"attack splash", "1", map[string]string{"Skill01_AttackSplash": "1"}, 100, 1000, nil, true},
		{"main attack", "2", map[string]string{"Skill02_MainAttack": "1"}, 100, 1000, nil, true},
		{"target hint", "3", map[string]string{"Skill03_Target": "4"}, 100, 1000, nil, true},
		{"Probablity passes", "1", map[string]string{"Skill01_Probablity": "1000"}, 100, 1000, []rnd.Draw{rnd.Get(GateScale, 999)}, true},
		{"Probablity fails", "1", map[string]string{"Skill01_Probablity": "1000"}, 100, 1000, []rnd.Draw{rnd.Get(GateScale, 1000)}, false},
		{"Probability spelling", "4", map[string]string{"Skill04_Probability": "50"}, 100, 1000, []rnd.Draw{rnd.Get(GateScale, 49)}, true},
		{"Prob spelling", "5", map[string]string{"Skill05_Prob": "50"}, 100, 1000, []rnd.Draw{rnd.Get(GateScale, 50)}, false},
		{"Probablity first", "6", map[string]string{"Skill06_Probablity": "10", "Skill06_Prob": "9000"}, 100, 1000, []rnd.Draw{rnd.Get(GateScale, 10)}, false},
		{"distance alone is a hint", "1", map[string]string{"Skill01_Check_Dist": "1", "Skill01_Dist_Max": "150"}, 150, 1000, nil, true},
		{"too far", "1", map[string]string{"Skill01_Check_Dist": "1", "Skill01_Dist_Max": "150"}, 151, 1000, nil, false},
		{"too near", "1", map[string]string{"Skill01_Check_Dist": "1", "Skill01_Dist_Min": "700"}, 699, 1000, nil, false},
		{"far enough", "1", map[string]string{"Skill01_Check_Dist": "1", "Skill01_Dist_Min": "700"}, 700, 1000, nil, true},
		{"drawn before the distance", "1", map[string]string{"Skill01_Probablity": "1000", "Skill01_Check_Dist": "1", "Skill01_Dist_Max": "150"}, 200, 1000, []rnd.Draw{rnd.Get(GateScale, 0)}, false},
		{"target above HighHP", "1", map[string]string{"Skill01_HighHP": "50"}, 100, 501, nil, true},
		{"target at HighHP", "1", map[string]string{"Skill01_HighHP": "50"}, 100, 500, nil, false},
		{"caster's HP", "1", map[string]string{"Skill01_HighHP": "50", "Skill01_HPTarget": "1"}, 100, 400, nil, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := d.synthetic(t, gated(tt.slot, tt.params))
			caster := monster("caster", 0, 0, p)
			target := player("target", tt.dist, 0)
			target.hp = tt.hp
			if tt.params["Skill01_HPTarget"] == "1" {
				caster.hp, target.hp = tt.hp, 1000
			}
			r := rnd.NewScripted(t, tt.draws...)
			if got := p.AllowsCast(r, caster, heal, target); got != tt.want {
				t.Errorf("AllowsCast = %v, want %v", got, tt.want)
			}
			// A skill in no slot is always allowed, without a draw.
			if !p.AllowsCast(r, caster, other, target) {
				t.Error("a skill in no slot is refused")
			}
		})
	}
}

func TestAllowsCastHighHPWithoutTarget(t *testing.T) {
	d := repo(t)
	p := d.synthetic(t, gated("1", map[string]string{"Skill01_HighHP": "50"}))
	if p.AllowsCast(rnd.NewScripted(t), monster("caster", 0, 0, p), d.skill(t, 4065, 4), nil) {
		t.Error("HighHP without a target is allowed")
	}
}

func TestGateBadNumber(t *testing.T) {
	d := repo(t)
	tmpl := gated("1", map[string]string{"Skill01_Probablity": "ten"})
	_, warnings := NewProfile(&tmpl, d.cat)
	if len(warnings) != 1 {
		t.Errorf("warnings %v, want one", warnings)
	}
}

// Catherok's Shock: 10 % within 150.
func TestGatesCatherok(t *testing.T) {
	d := repo(t)
	p := d.profile(t, 21035)
	g := p.gateOf(4072)
	if g == nil || g.probability != 1000 || !g.checkDist || g.distMax != 150 || !g.attackHint {
		t.Fatalf("Catherok's Shock gate %+v", g)
	}
	// Soldier of Darkness's Decrease Atk. Spd. has only a Type: never used.
	p = d.profile(t, 21040)
	if p.AllowsCast(rnd.NewScripted(t), monster("soldier", 0, 0, p), d.skill(t, 4038, 4), player("p", 100, 0)) {
		t.Error("Soldier of Darkness may use Decrease Atk. Spd.")
	}
}
