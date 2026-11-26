package config

import (
	"reflect"
	"strings"
	"testing"
)

// The Mobius Player.ini Augmenting values.
func TestDefaultAugment(t *testing.T) {
	want := Augment{
		RetailLikeAugmentation: true,
		NoGradeChance:          []int{55, 35, 7, 3},
		MidGradeChance:         []int{55, 35, 7, 3},
		HighGradeChance:        []int{55, 35, 7, 3},
		TopGradeChance:         []int{55, 35, 7, 3},
		BlackList:              []int{6656, 6657, 6658, 6659, 6660, 6661, 6662, 8191},
		AllowDestroy:           true,
	}
	if got := Default().GameServers[0].Augment; !reflect.DeepEqual(got, want) {
		t.Errorf("Augment = %+v, want %+v", got, want)
	}
	// The defaults are copies.
	a := DefaultAugment()
	a.BlackList[0], a.NoGradeChance[0] = 1, 1
	if DefaultAugmentBlackList[0] != 6656 || DefaultAugmentGradeChance[0] != 55 || DefaultAugment().MidGradeChance[0] != 55 {
		t.Error("DefaultAugment shares its lists")
	}
}

// Every key is read, a section left out keeps its defaults, and explicit
// false is kept.
func TestParseAugment(t *testing.T) {
	cfg, err := Parse([]byte(`{"gameservers": [{"id": 1, "augment": {"_comment": "c",
		"retailLikeAugmentation": true,
		"retailLikeAugmentationNoGradeChance": [40, 40, 10, 10],
		"retailLikeAugmentationMidGradeChance": [50, 30, 10, 10],
		"retailLikeAugmentationHighGradeChance": [60, 20, 10, 10],
		"retailLikeAugmentationTopGradeChance": [70, 10, 10, 10],
		"augmentationBlackList": [57],
		"altAllowAugmentPvPItems": true, "altAllowAugmentTrade": false, "altAllowAugmentDestroy": false}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := Augment{Comment: "c", RetailLikeAugmentation: true,
		NoGradeChance: []int{40, 40, 10, 10}, MidGradeChance: []int{50, 30, 10, 10},
		HighGradeChance: []int{60, 20, 10, 10}, TopGradeChance: []int{70, 10, 10, 10},
		BlackList: []int{57}, AllowPvPItems: true}
	if got := cfg.GameServers[0].Augment; !reflect.DeepEqual(got, want) {
		t.Errorf("Augment = %+v, want %+v", got, want)
	}
	if errs := cfg.GameServers[0].validateAugment(); len(errs) != 0 {
		t.Errorf("validateAugment: %v", errs)
	}
	cfg, err = Parse([]byte(`{"gameservers": [{"id": 1, "augment": {"altAllowAugmentDestroy": false}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	want = DefaultAugment()
	want.AllowDestroy = false
	if got := cfg.GameServers[0].Augment; !reflect.DeepEqual(got, want) {
		t.Errorf("one key: Augment = %+v, want %+v", got, want)
	}
	cfg, err = Parse([]byte(`{"gameservers": [{"id": 1}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.GameServers[0].Augment; !reflect.DeepEqual(got, DefaultAugment()) {
		t.Errorf("left out: Augment = %+v, want the defaults", got)
	}
	if _, err := Parse([]byte(`{"gameservers": [{"id": 1, "augment": {"augmentationNGSkillChance": 15}}]}`)); err == nil {
		t.Error("augmentationNGSkillChance parsed; the classic augmentation is not supported")
	}
}

func TestValidateAugment(t *testing.T) {
	gs := Default().GameServers[0]
	if errs := gs.validateAugment(); len(errs) != 0 {
		t.Errorf("defaults: %v", errs)
	}
	gs.Augment = Augment{
		NoGradeChance:   []int{55, 35, 7},
		MidGradeChance:  []int{55, 35, 7, 4},
		HighGradeChance: []int{110, -10, 0, 0},
		TopGradeChance:  []int{55, 35, 7, 3},
		BlackList:       []int{0},
		AllowTrade:      true,
	}
	errs := gs.validateAugment()
	wants := []string{
		"augment.retailLikeAugmentation: false",
		"augment.retailLikeAugmentationNoGradeChance: 3 numbers",
		"augment.retailLikeAugmentationMidGradeChance: [55 35 7 4] adds up to 101",
		"augment.retailLikeAugmentationHighGradeChance: -10 is negative",
		"augment.augmentationBlackList: 0",
		"augment.altAllowAugmentTrade: true",
	}
	if len(errs) != len(wants) {
		t.Fatalf("validateAugment = %v, want %d errors", errs, len(wants))
	}
	for i, want := range wants {
		if !strings.Contains(errs[i].Error(), want) {
			t.Errorf("error %d = %v, want %q", i, errs[i], want)
		}
	}
	cfg := Default()
	cfg.GameServers[0].Augment.RetailLikeAugmentation = false
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "gameservers[0].augment.retailLikeAugmentation") {
		t.Errorf("Validate = %v, want the augment error with its path", err)
	}
}
