package data

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewHennas(t *testing.T) {
	good := func() []Henna {
		return []Henna{
			{DyeID: 1, DyeItemID: 4445, STR: 1, CON: -3, WearCount: 10, WearFee: 37000, CancelCount: 5, CancelFee: 7400, Classes: []int{1, 11}},
			{DyeID: 5, DyeItemID: 4449, Classes: []int{}},
		}
	}
	h, err := NewHennas(good())
	if err != nil {
		t.Fatal(err)
	}
	if d := h.Henna(1); d == nil || d.DyeItemID != 4445 || !d.AllowedClass(11) || d.AllowedClass(0) {
		t.Errorf("Henna(1) = %+v", d)
	}
	if h.Henna(2) != nil || (*Hennas)(nil).Henna(1) != nil || (*Hennas)(nil).ForClass(1) != nil {
		t.Error("a missing dye is found")
	}
	if got := h.ForClass(1); len(got) != 1 || got[0].DyeID != 1 {
		t.Errorf("ForClass(1) = %+v", got)
	}

	tests := []struct {
		name   string
		change func(l []Henna)
		want   string
	}{
		{"zero dye", func(l []Henna) { l[0].DyeID = 0 }, "dye ID not positive"},
		{"unsorted", func(l []Henna) { l[1].DyeID = 1 }, "not sorted by dye ID"},
		{"no item", func(l []Henna) { l[1].DyeItemID = 0 }, "dye item ID not positive"},
		{"negative fee", func(l []Henna) { l[0].CancelFee = -1 }, "a negative count or fee"},
		{"classes unsorted", func(l []Henna) { l[0].Classes = []int{11, 1} }, "classes not sorted"},
		{"unknown class", func(l []Henna) { l[0].Classes = []int{1, 999} }, "unknown class 999"},
	}
	for _, tt := range tests {
		l := good()
		tt.change(l)
		_, err := NewHennas(l)
		if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: error %v, want %q", tt.name, err, tt.want)
		}
	}
}

// The committed file loads: 180 dyes.
func TestLoadHennas(t *testing.T) {
	h, err := LoadHennas(filepath.Join("..", "..", "data"))
	if err != nil {
		t.Fatal(err)
	}
	if len(h.List) != 180 {
		t.Errorf("%d dyes, want 180", len(h.List))
	}
}
