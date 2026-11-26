package config

import (
	"reflect"
	"strings"
	"testing"
)

// The Mobius Feature.ini Seven Signs and General.ini Dimension Rift values.
func TestDefaultSevenSigns(t *testing.T) {
	s := Default().GameServers[0].SevenSigns
	if !reflect.DeepEqual(s, DefaultSevenSigns()) {
		t.Errorf("Default sevenSigns = %+v", s)
	}
	switch {
	case s.OpenCatacombs, s.OpenNecropolis, !s.CastleForDawn, !s.CastleForDusk, s.FestivalMinPlayers != 5,
		s.MaxPlayerContrib != 1000000, s.FestivalManagerStart != 120000, s.FestivalLength != 1080000,
		s.FestivalCycleLength != 2280000, s.FestivalFirstSpawn != 120000, s.FestivalFirstSwarm != 300000,
		s.FestivalSecondSpawn != 540000, s.FestivalSecondSwarm != 720000, s.FestivalChestSpawn != 900000,
		s.DawnGatesPDefMult != 1.1, s.DuskGatesPDefMult != 0.8, s.DawnGatesMDefMult != 1.1, s.DuskGatesMDefMult != 0.8,
		s.StrictSevenSigns, s.DawnTicketQuantity != 300, s.DawnTicketPrice != 1000, s.DawnTicketBundle != 10,
		s.ManorsAgreementID != 6388, s.JoinDawnFee != 50000, s.FestivalOfDarknessWin != 200:
		t.Errorf("DefaultSevenSigns = %+v", s)
	}
	r := s.Rift
	switch {
	case r.MinPartySize != 2, r.MaxJumps != 4, r.SpawnDelay != 10000, r.AutoJumpsDelayMin != 480, r.AutoJumpsDelayMax != 600,
		r.BossRoomTimeMultiply != 1.5, r.RecruitCost != 18, r.SoldierCost != 21, r.OfficerCost != 24, r.CaptainCost != 27,
		r.CommanderCost != 30, r.HeroCost != 33:
		t.Errorf("DefaultRift = %+v", r)
	}
}

// A section left out keeps its defaults; the keys given replace them.
func TestParseSevenSigns(t *testing.T) {
	cfg, err := Parse([]byte(`{"gameservers": [{"id": 1, "sevenSigns": {"_comment": "demo",
		"openNecropolis": true, "festivalMinPlayers": 2, "joinDawnFee": 0, "rift": {"minPartySize": 3, "heroCost": 1}}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := DefaultSevenSigns()
	want.Comment = "demo"
	want.OpenNecropolis = true
	want.FestivalMinPlayers = 2
	want.JoinDawnFee = 0
	want.Rift.MinPartySize, want.Rift.HeroCost = 3, 1
	if got := cfg.GameServers[0].SevenSigns; !reflect.DeepEqual(got, want) {
		t.Errorf("SevenSigns = %+v, want %+v", got, want)
	}
	cfg, err = Parse([]byte(`{"gameservers": [{"id": 1}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.GameServers[0].SevenSigns; !reflect.DeepEqual(got, DefaultSevenSigns()) {
		t.Errorf("left out: SevenSigns = %+v, want the defaults", got)
	}
	if _, err := Parse([]byte(`{"gameservers": [{"id": 1, "sevenSigns": {"nope": 1}}]}`)); err == nil {
		t.Error("an unknown sevenSigns key parsed")
	}
	if _, err := Parse([]byte(`{"gameservers": [{"id": 1, "sevenSigns": {"rift": {"nope": 1}}}]}`)); err == nil {
		t.Error("an unknown rift key parsed")
	}
}

func TestValidateSevenSigns(t *testing.T) {
	gs := Default().GameServers[0]
	if errs := gs.validateSevenSigns(); len(errs) != 0 {
		t.Errorf("defaults: %v", errs)
	}
	gs.SevenSigns.FestivalMinPlayers = 0
	gs.SevenSigns.DawnTicketBundle = 0
	gs.SevenSigns.FestivalLength = 3000000
	gs.SevenSigns.DuskGatesMDefMult = 0
	gs.SevenSigns.Rift.MinPartySize = 10
	gs.SevenSigns.Rift.MaxJumps = 0
	gs.SevenSigns.Rift.AutoJumpsDelayMax = 100
	gs.SevenSigns.Rift.BossRoomTimeMultiply = 0
	errs := gs.validateSevenSigns()
	wants := []string{
		"sevenSigns.festivalMinPlayers: 0", "sevenSigns.dawnTicketBundle: 0", "sevenSigns.festivalLength: 3000000",
		"sevenSigns.duskGatesMDefMult: 0", "sevenSigns.rift.minPartySize: 10", "sevenSigns.rift.maxJumps: 0",
		"sevenSigns.rift.autoJumpsDelayMax: 100", "sevenSigns.rift.bossRoomTimeMultiply: 0",
	}
	if len(errs) != len(wants) {
		t.Fatalf("validateSevenSigns = %v, want %d errors", errs, len(wants))
	}
	for i, want := range wants {
		if !strings.Contains(errs[i].Error(), want) {
			t.Errorf("error %d = %v, want %q", i, errs[i], want)
		}
	}
	cfg := Default()
	cfg.GameServers[0].SevenSigns.Rift.MinPartySize = 1
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "gameservers[0].sevenSigns.rift.minPartySize") {
		t.Errorf("Validate = %v, want the sevenSigns error with its path", err)
	}
}
