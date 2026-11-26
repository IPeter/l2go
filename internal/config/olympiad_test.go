package config

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// The Mobius Olympiad.ini values (and Feature.ini HeroPoints).
func TestDefaultOlympiad(t *testing.T) {
	o := Default().GameServers[0].Olympiad
	if !reflect.DeepEqual(o, DefaultOlympiad()) {
		t.Errorf("Default olympiad = %+v", o)
	}
	switch {
	case !o.Enabled, o.StartTime != 18, o.Min != 0, time.Duration(o.CPeriod) != 6*time.Hour,
		time.Duration(o.Battle) != 6*time.Minute, time.Duration(o.WPeriod) != 7*24*time.Hour,
		time.Duration(o.VPeriod) != 24*time.Hour, o.StartPoints != 18, o.WeeklyPoints != 3,
		o.ClassedParticipants != 5, o.NonClassedParticipants != 9, o.BattleRewItem != 6651,
		o.ClassedRewItemCount != 50, o.NonClassedRewItemCount != 30, o.CompRewItem != 6651,
		o.GPPerPoint != 1000, o.HeroPoints != 100, o.MaxPoints != 10, !o.ShowMonthlyWinners,
		!o.AnnounceGames, !o.DisableBlessedSpiritShots, o.EnchantLimit != -1, o.WaitTime != 120,
		o.UseCustomPeriodSettings, o.PeriodName() != OlympiadPeriodMonth, o.PeriodMultiplier != 1,
		len(o.CompetitionDays) != 7, o.ClanHeroPoints != 1000:
		t.Errorf("DefaultOlympiad = %+v", o)
	}
}

// A section left out keeps its defaults; the keys given replace them.
func TestParseOlympiad(t *testing.T) {
	cfg, err := Parse([]byte(`{"gameservers": [{"id": 1, "olympiad": {"_comment": "demo",
		"classedParticipants": 2, "nonClassedParticipants": 2, "battle": "1m0s", "period": "week",
		"announceGames": false, "restrictedItems": [6611], "clanHeroPoints": 0}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := DefaultOlympiad()
	want.Comment = "demo"
	want.ClassedParticipants, want.NonClassedParticipants = 2, 2
	want.Battle = Duration(time.Minute)
	want.Period = "week"
	want.AnnounceGames = false
	want.RestrictedItems = []int{6611}
	want.ClanHeroPoints = 0
	if got := cfg.GameServers[0].Olympiad; !reflect.DeepEqual(got, want) {
		t.Errorf("Olympiad = %+v, want %+v", got, want)
	}
	if got := cfg.GameServers[0].Olympiad.PeriodName(); got != OlympiadPeriodWeek {
		t.Errorf("PeriodName = %q", got)
	}
	cfg, err = Parse([]byte(`{"gameservers": [{"id": 1}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.GameServers[0].Olympiad; !reflect.DeepEqual(got, DefaultOlympiad()) {
		t.Errorf("left out: Olympiad = %+v, want the defaults", got)
	}
	if _, err := Parse([]byte(`{"gameservers": [{"id": 1, "olympiad": {"nope": 1}}]}`)); err == nil {
		t.Error("an unknown olympiad key parsed")
	}
}

func TestValidateOlympiad(t *testing.T) {
	gs := Default().GameServers[0]
	if errs := gs.validateOlympiad(); len(errs) != 0 {
		t.Errorf("defaults: %v", errs)
	}
	gs.Olympiad.StartTime = 24
	gs.Olympiad.Min = -1
	gs.Olympiad.CPeriod = 0
	gs.Olympiad.MaxPoints = -2
	gs.Olympiad.ClassedParticipants = 1
	gs.Olympiad.BattleRewItem = 0
	gs.Olympiad.RestrictedItems = []int{0}
	gs.Olympiad.WaitTime = 0
	gs.Olympiad.Period = "YEAR"
	gs.Olympiad.PeriodMultiplier = 0
	gs.Olympiad.CompetitionDays = []int{8}
	errs := gs.validateOlympiad()
	wants := []string{
		"olympiad.startTime: 24", "olympiad.min: -1", "olympiad.cPeriod", "olympiad.maxPoints: -2",
		"olympiad.classedParticipants", "olympiad.battleRewItem: 0", "olympiad.restrictedItems: 0",
		"olympiad.waitTime: 0", `olympiad.period: "YEAR"`, "olympiad.periodMultiplier: 0", "olympiad.competitionDays: 8",
	}
	if len(errs) != len(wants) {
		t.Fatalf("validateOlympiad = %v, want %d errors", errs, len(wants))
	}
	for i, want := range wants {
		if !strings.Contains(errs[i].Error(), want) {
			t.Errorf("error %d = %v, want %q", i, errs[i], want)
		}
	}
	cfg := Default()
	cfg.GameServers[0].Olympiad.WaitTime = 0
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "gameservers[0].olympiad.waitTime") {
		t.Errorf("Validate = %v, want the olympiad error with its path", err)
	}
}
