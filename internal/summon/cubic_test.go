package summon

import "testing"

// fixedRand answers Get with the next value of its list, recording the
// bounds.
type fixedRand struct {
	values []int
	bounds []int
}

func (r *fixedRand) Get(n int) int {
	r.bounds = append(r.bounds, n)
	v := r.values[0]
	r.values = r.values[1:]
	return v
}

func TestPickCubicSkill(t *testing.T) {
	for _, tt := range []struct {
		name   string
		mode   string
		rates  []int
		draws  []int
		want   int
		bounds []int
	}{
		{"random", SelectRandom, []int{100, 100, 100}, []int{2}, 2, []int{3}},
		{"ordered first", SelectOrdered, []int{100, 50}, nil, 0, nil},
		{"ordered second", SelectOrdered, []int{30, 50}, []int{40, 10}, 1, []int{100, 100}},
		{"ordered none", SelectOrdered, []int{30, 50}, []int{40, 60}, -1, []int{100, 100}},
		{"weighted", SelectWeighted, []int{20, 0, 60}, []int{30}, 2, []int{80}},
		{"weighted without rates", SelectWeighted, []int{0, 0}, []int{1}, 1, []int{2}},
	} {
		r := &fixedRand{values: tt.draws}
		if got := PickCubicSkill(r, tt.mode, tt.rates); got != tt.want {
			t.Errorf("%s: %d, want %d", tt.name, got, tt.want)
		}
		if len(r.bounds) != len(tt.bounds) {
			t.Errorf("%s: draws %v, want %v", tt.name, r.bounds, tt.bounds)
		}
	}
	if PickCubicSkill(&fixedRand{}, SelectRandom, nil) != -1 {
		t.Error("no skills")
	}
}
