package data

import (
	"strings"
	"testing"
)

func TestCheckSeeds(t *testing.T) {
	good := Seed{CastleID: 1, SeedID: 5016, CropID: 5073, MatureID: 5103, Reward1: 1864, Reward2: 1878, Level: 10, LimitSeeds: 8100, LimitCrops: 9000}
	if err := CheckSeeds([]Seed{good}); err != nil {
		t.Fatalf("a good seed: %v", err)
	}
	for _, c := range []struct {
		name   string
		mutate func(s *Seed)
		want   string
	}{
		{"no castle", func(s *Seed) { s.CastleID = 0 }, "must be positive"},
		{"no mature crop", func(s *Seed) { s.MatureID = 0 }, "must be positive"},
		{"negative limit", func(s *Seed) { s.LimitCrops = -1 }, "negative level or limit"},
	} {
		s := good
		c.mutate(&s)
		if err := CheckSeeds([]Seed{s}); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: error %v, want %q", c.name, err, c.want)
		}
	}
	if err := CheckSeeds([]Seed{good, good}); err == nil || !strings.Contains(err.Error(), "seed listed twice") {
		t.Errorf("twice: error %v", err)
	}
}

// The committed data has Mobius's 258 seeds in 9 castles.
func TestCommittedSeeds(t *testing.T) {
	d, err := Load("../../data")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Seeds) != 258 {
		t.Fatalf("%d seeds, want 258", len(d.Seeds))
	}
	if s := d.Seeds[0]; s != (Seed{CastleID: 1, SeedID: 5016, CropID: 5073, MatureID: 5103, Reward1: 1864, Reward2: 1878, Level: 10, LimitSeeds: 8100, LimitCrops: 9000}) {
		t.Errorf("first seed %+v", s)
	}
	castles := map[int]bool{}
	for _, s := range d.Seeds {
		castles[s.CastleID] = true
	}
	if len(castles) != 9 {
		t.Errorf("%d castles", len(castles))
	}
}
