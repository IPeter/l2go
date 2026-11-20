package stats

import "testing"

// TestFistsItemID follows Player.findFistsWeaponItem's class ranges.
func TestFistsItemID(t *testing.T) {
	tests := []struct{ class, want int }{
		{0, 246}, {9, 246}, // Human Fighter line
		{10, 251}, {17, 251}, // Human Mystic line
		{18, 244}, {24, 244}, // Elven Fighter line
		{25, 249}, {30, 249}, // Elven Mystic line
		{31, 245}, {37, 245}, // Dark Fighter line
		{38, 250}, {43, 250}, // Dark Mystic line
		{44, 248}, {48, 248}, // Orc Fighter line
		{49, 252}, {52, 252}, // Orc Mystic line
		{53, 247}, {57, 247}, // Dwarven Fighter line
		{88, 246}, {-1, 246}, // outside the ranges: the default
	}
	for _, tt := range tests {
		if got := FistsItemID(tt.class); got != tt.want {
			t.Errorf("FistsItemID(%d) = %d, want %d", tt.class, got, tt.want)
		}
	}
}

func TestActiveWeapon(t *testing.T) {
	sword := Weapon{ItemID: 2369, Type: "SWORD"}
	bow := Weapon{ItemID: 14, Type: "BOW", ReuseDelay: 1500}
	if got := PlayerWeapon(44, nil); got != (Weapon{ItemID: 248, Type: "FIST"}) {
		t.Errorf("PlayerWeapon(orc fighter, none) = %+v, want the orc fists", got)
	}
	if got := PlayerWeapon(0, &sword); got != sword {
		t.Errorf("PlayerWeapon(with a sword) = %+v", got)
	}
	if AttackType(nil, "SWORD") != "SWORD" || AttackType(&bow, "SWORD") != "BOW" {
		t.Error("AttackType: the weapon's type, else the template's")
	}

	tests := []struct {
		name   string
		w      *Weapon
		tmpl   string
		player bool
		want   Fighter
	}{
		{"unarmed player: fists, dual", &Weapon{ItemID: 246, Type: "FIST"}, "FIST", true, Fighter{Armed: true, Attack: ShapeDual}},
		{"player with a sword", &sword, "FIST", true, Fighter{Armed: true, Attack: ShapeSimple}},
		{"player with a bow", &bow, "FIST", true, Fighter{Armed: true, Attack: ShapeBow, ReuseDelay: 1500}},
		{"NPC without rhand: template SWORD", nil, "SWORD", false, Fighter{Attack: ShapeSimple}},
		{"NPC without rhand: template FIST is simple", nil, "FIST", false, Fighter{Attack: ShapeSimple}},
		{"NPC without rhand: template BOW swings as a bow, no reuse", nil, "BOW", false, Fighter{Attack: ShapeBow}},
		{"Orc Archer: rhand bow 14", &bow, "BOW", false, Fighter{Armed: true, Attack: ShapeBow, ReuseDelay: 1500}},
	}
	for _, tt := range tests {
		f := Fighter{ReuseDelay: 99}
		f.Arm(tt.w, tt.tmpl, tt.player)
		if f.Armed != tt.want.Armed || f.Attack != tt.want.Attack || f.ReuseDelay != tt.want.ReuseDelay {
			t.Errorf("%s: armed %v, %v, reuse %d; want %v, %v, %d", tt.name,
				f.Armed, f.Attack, f.ReuseDelay, tt.want.Armed, tt.want.Attack, tt.want.ReuseDelay)
		}
	}
}

func TestAttackCountMax(t *testing.T) {
	if got := AttackCountMax(&fake{}); got != 1 {
		t.Errorf("AttackCountMax without modifiers = %d, want 1", got)
	}
	f := &fake{calc: map[Stat]func(float64, Context) float64{AtkCountMax: plus(2.5)}}
	if got := AttackCountMax(f); got != 3 {
		t.Errorf("AttackCountMax with +2.5 = %d, want (int) 3.5 = 3", got)
	}
}
