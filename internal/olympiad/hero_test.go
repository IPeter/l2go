package olympiad

import (
	"slices"
	"testing"
)

func TestHeroes(t *testing.T) {
	h := NewHeroes([]Hero{
		{CharID: 1, ClassID: 88, Name: "Old", Count: 2, Played: true, Claimed: true, Message: "hi"},
		{CharID: 2, ClassID: 93, Name: "Older", Count: 1},
	})
	if !h.IsHero(1) || h.IsUnclaimed(1) || h.IsHero(2) || h.ByClass(88) != 1 || h.ByClass(93) != 0 {
		t.Fatalf("loaded: hero %v, unclaimed %v", h.IsHero(1), h.IsUnclaimed(1))
	}
	former := h.ComputeNewHeroes([]Candidate{{2, 94, "Older"}, {3, 88, "New"}})
	if !slices.Equal(former, []int32{1}) {
		t.Errorf("former = %v", former)
	}
	if h.IsHero(1) || h.IsUnclaimed(1) || !h.IsUnclaimed(2) || !h.IsUnclaimed(3) || h.ByClass(88) != 3 {
		t.Errorf("after the period: 1 %v/%v, 2 %v, 3 %v", h.IsHero(1), h.IsUnclaimed(1), h.IsUnclaimed(2), h.IsUnclaimed(3))
	}
	if x, ok := h.Get(2); !ok || x.Count != 2 || x.ClassID != 93 {
		t.Errorf("re-elected hero %+v (the old row keeps its class, as Mobius)", x)
	}
	if x, ok := h.Get(3); !ok || x.Count != 1 || x.Name != "New" {
		t.Errorf("new hero %+v", x)
	}
	h.Claim(3, 88, "New")
	if !h.IsHero(3) || h.IsUnclaimed(3) || !slices.Equal(h.IDs(), []int32{3}) {
		t.Errorf("after the claim: %v %v", h.IsHero(3), h.IDs())
	}
	if !h.SetMessage(3, "words") || h.SetMessage(1, "x") {
		t.Error("SetMessage")
	}
	if x, _ := h.Get(3); x.Message != "words" {
		t.Errorf("message %q", x.Message)
	}
	// A character that was never a hero gets a row of its own (Mobius
	// claimHero with no row).
	h.Claim(9, 100, "Gm")
	if !h.IsHero(9) || len(h.Current()) != 3 || len(h.All()) != 4 {
		t.Errorf("claim without a row: %v, %d current", h.IsHero(9), len(h.Current()))
	}
	if !h.Revoke(9) || h.IsHero(9) || h.Revoke(9) {
		t.Error("Revoke")
	}
	h.Rename(3, "Renamed")
	if x, _ := h.Get(3); x.Name != "Renamed" {
		t.Errorf("Rename: %q", x.Name)
	}
}
