package data

import (
	"strings"
	"testing"
)

func TestCheckCursedWeapons(t *testing.T) {
	good := CursedWeapon{ItemID: 8190, Name: "Demonic Sword Zariche", SkillID: 3603, DropRate: 1, Duration: 300,
		DurationLost: 3, DisappearChance: 50, StageKills: 10}
	if err := CheckCursedWeapons([]CursedWeapon{good}); err != nil {
		t.Fatalf("a good weapon: %v", err)
	}
	for _, c := range []struct {
		name   string
		mutate func(w *CursedWeapon)
		want   string
	}{
		{"no item", func(w *CursedWeapon) { w.ItemID = 0 }, "no item ID"},
		{"no skill", func(w *CursedWeapon) { w.SkillID = 0 }, "no skill ID"},
		{"negative rate", func(w *CursedWeapon) { w.DropRate = -1 }, "negative drop rate"},
		{"zero stage kills", func(w *CursedWeapon) { w.StageKills = 0 }, "must be positive"},
		{"zero duration lost", func(w *CursedWeapon) { w.DurationLost = 0 }, "must be positive"},
	} {
		w := good
		c.mutate(&w)
		if err := CheckCursedWeapons([]CursedWeapon{w}); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: error %v, want %q", c.name, err, c.want)
		}
	}
	if err := CheckCursedWeapons([]CursedWeapon{good, good}); err == nil || !strings.Contains(err.Error(), "item listed twice") {
		t.Errorf("twice: error %v", err)
	}
}

// The committed data has Mobius's two weapons, both items.
func TestCommittedCursedWeapons(t *testing.T) {
	d, err := Load("../../data")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.CursedWeapons) != 2 {
		t.Fatalf("%d cursed weapons, want 2", len(d.CursedWeapons))
	}
	for _, c := range []struct {
		item, skill int
		name        string
	}{{8190, 3603, "Demonic Sword Zariche"}, {8689, 3629, "Blood Sword Akamanah"}} {
		w, ok := d.CursedWeapon(c.item)
		if !ok || w.SkillID != c.skill || w.Name != c.name || w.DropRate != 1 || w.Duration != 300 ||
			w.DurationLost != 3 || w.DisappearChance != 50 || w.StageKills != 10 {
			t.Errorf("CursedWeapon(%d) = %+v, %v", c.item, w, ok)
		}
	}
}
