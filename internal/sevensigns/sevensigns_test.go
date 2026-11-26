package sevensigns

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/calendar"
)

func TestContributionAndReward(t *testing.T) {
	if got := ContributionScore(2, 3, 4); got != 2*3+3*5+4*10 {
		t.Errorf("ContributionScore = %d", got)
	}
	if got := AncientAdenaReward(2, 3, 4); got != 2*3+3*5+4*10 {
		t.Errorf("AncientAdenaReward = %d", got)
	}
}

func TestJoinContributeAndScores(t *testing.T) {
	s := NewState(NewStatus(), nil)
	s.SetPlayerInfo(1, CabalDawn, SealGnosis)
	s.SetPlayerInfo(2, CabalDusk, SealStrife)
	s.SetPlayerInfo(3, CabalDusk, SealStrife)
	if s.TotalMembers(CabalDawn) != 1 || s.TotalMembers(CabalDusk) != 2 {
		t.Fatalf("members dawn %d dusk %d", s.TotalMembers(CabalDawn), s.TotalMembers(CabalDusk))
	}
	if s.DawnSealTotals[SealGnosis] != 1 || s.DuskSealTotals[SealStrife] != 2 {
		t.Errorf("seal totals %v %v", s.DawnSealTotals, s.DuskSealTotals)
	}
	if got := s.AddStoneContrib(1, 0, 0, 10, 1000000); got != 100 {
		t.Errorf("dawn contribution %d, want 100", got)
	}
	if got := s.AddStoneContrib(2, 10, 0, 0, 1000000); got != 30 {
		t.Errorf("dusk contribution %d, want 30", got)
	}
	if got := s.AddStoneContrib(2, 0, 0, 1, 35); got != -1 {
		t.Errorf("over the limit: %d, want -1", got)
	}
	if got := s.AddStoneContrib(99, 1, 0, 0, 1000000); got != -1 {
		t.Errorf("without a record: %d", got)
	}
	// 100 / 130 * 500 = 384.6 -> 385; 30 / 130 * 500 = 115.38 -> 115.
	if d, k := s.CurrentScore(CabalDawn), s.CurrentScore(CabalDusk); d != 385 || k != 115 {
		t.Errorf("scores dawn %d dusk %d, want 385 115", d, k)
	}
	if s.HighestCabal() != CabalDawn {
		t.Errorf("highest %v", s.HighestCabal())
	}
	s.AddFestivalScore(CabalDusk, 150)
	s.AddFestivalScore(CabalDusk, 150)
	s.AddFestivalScore(CabalDawn, 100) // takes 100 from dusk
	if s.DuskFestivalScore != 200 || s.DawnFestivalScore != 100 {
		t.Errorf("festival dawn %d dusk %d", s.DawnFestivalScore, s.DuskFestivalScore)
	}
	if s.HighestCabal() != CabalDawn { // 485 vs 315
		t.Errorf("highest after festival %v", s.HighestCabal())
	}
	p, _ := s.Player(1)
	if p.RedStones != 10 || p.AncientAdena != 100 || p.Contribution != 100 {
		t.Errorf("player %+v", p)
	}
	if got := s.TakeAncientAdenaReward(1); got != 100 {
		t.Errorf("reward %d", got)
	}
	if p, _ := s.Player(1); p.RedStones != 0 || p.AncientAdena != 0 || p.Contribution != 100 {
		t.Errorf("after the reward %+v", p)
	}
	s.ResetPlayers()
	s.ResetSeals()
	if p, _ := s.Player(2); p.Cabal != "" || p.Seal != SealNull || p.Contribution != 0 || p.BlueStones != 10 {
		t.Errorf("after the reset %+v", p)
	}
	if s.PlayerCabal(2) != CabalNull || s.TotalMembers(CabalDusk) != 0 {
		t.Error("the reset kept a cabal")
	}
}

func TestEmptyScoresTie(t *testing.T) {
	s := NewState(NewStatus(), nil)
	if s.CurrentScore(CabalDawn) != 0 || s.HighestCabal() != CabalNull {
		t.Errorf("empty: %d %v", s.CurrentScore(CabalDawn), s.HighestCabal())
	}
}

func TestNewSealOwner(t *testing.T) {
	for _, c := range []struct {
		prev, winner Cabal
		dawn, dusk   int
		want         Cabal
	}{
		{CabalNull, CabalNull, 100, 100, CabalNull},
		{CabalNull, CabalDawn, 35, 0, CabalDawn},
		{CabalNull, CabalDawn, 34, 100, CabalNull},
		{CabalNull, CabalDusk, 0, 35, CabalDusk},
		{CabalDawn, CabalNull, 10, 0, CabalDawn},
		{CabalDawn, CabalNull, 9, 0, CabalNull},
		{CabalDawn, CabalDawn, 10, 0, CabalDawn},
		{CabalDawn, CabalDusk, 10, 35, CabalDusk},
		{CabalDawn, CabalDusk, 10, 34, CabalDawn},
		{CabalDawn, CabalDusk, 9, 34, CabalNull},
		{CabalDusk, CabalDawn, 35, 0, CabalDawn},
		{CabalDusk, CabalDawn, 34, 10, CabalDusk},
		{CabalDusk, CabalDusk, 0, 9, CabalNull},
	} {
		if got := NewSealOwner(c.prev, c.winner, c.dawn, c.dusk); got != c.want {
			t.Errorf("NewSealOwner(%v, %v, %d, %d) = %v, want %v", c.prev, c.winner, c.dawn, c.dusk, got, c.want)
		}
	}
}

func TestCalcNewSealOwnersAndPage4(t *testing.T) {
	s := NewState(NewStatus(), nil)
	// Three dawn members: two on strife (67 %), one on avarice (33 %).
	s.SetPlayerInfo(1, CabalDawn, SealStrife)
	s.SetPlayerInfo(2, CabalDawn, SealStrife)
	s.SetPlayerInfo(3, CabalDawn, SealAvarice)
	s.SetPlayerInfo(4, CabalDusk, SealGnosis)
	s.AddStoneContrib(1, 0, 0, 5, 1000000)
	winner, pred := s.Page4()
	if winner != CabalDawn || pred[2].Owner != CabalDawn || pred[2].Msg != MsgNotOwnedAnd35Percent ||
		pred[0].Owner != CabalNull || pred[0].Msg != MsgNotOwnedAndBelow35 {
		t.Errorf("page 4: %v %+v", winner, pred)
	}
	owners := s.CalcNewSealOwners()
	if owners[SealAvarice] != CabalNull || owners[SealGnosis] != CabalNull || owners[SealStrife] != CabalDawn {
		t.Errorf("owners %v", owners)
	}
	if SealObtainedMsg(SealStrife, CabalDawn) != 1214 || SealObtainedMsg(SealGnosis, CabalDusk) != 1216 || SealObtainedMsg(SealAvarice, CabalNull) != 0 {
		t.Error("SealObtainedMsg")
	}
	votes := s.Page3()
	if votes[2].DawnPercent != 67 || votes[2].DuskPercent != 0 || votes[1].DuskPercent != 100 || votes[2].Owner != CabalDawn {
		t.Errorf("page 3: %+v", votes)
	}
	p1 := s.Page1(1)
	if p1.Cabal != CabalDawn || p1.Seal != SealStrife || p1.StonesTurnedIn != 5 || p1.AncientAdenaToCollect != 50 ||
		p1.DawnStone != 500 || p1.DawnTotal != 500 || p1.DawnPercent != 100 || p1.PeriodMsg != MsgQuestEventPeriod || p1.PeriodEndMsg != MsgUntilNextMonday {
		t.Errorf("page 1: %+v", p1)
	}
}

func TestNextPeriodChange(t *testing.T) {
	cal := calendar.New(time.UTC)
	at := func(s string) time.Time {
		t.Helper()
		v, err := time.Parse("2006-01-02 15:04:05.000", s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	// 2026-11-23 is a Monday.
	for _, c := range []struct {
		from   string
		period Period
		want   string
	}{
		{"2026-11-23 17:59:12.345", PeriodCompetition, "2026-11-23 18:00:12.345"},
		{"2026-11-23 18:00:00.000", PeriodCompetition, "2026-11-30 18:00:00.000"},
		{"2026-11-23 18:15:07.000", PeriodSealValidation, "2026-11-30 18:00:07.000"},
		{"2026-11-22 23:00:00.000", PeriodCompetition, "2026-11-23 18:00:00.000"}, // Sunday
		{"2026-11-28 09:00:00.000", PeriodCompetition, "2026-11-30 18:00:00.000"}, // Saturday
		{"2026-11-24 09:00:00.000", PeriodCompetition, "2026-11-30 18:00:00.000"}, // Tuesday
		{"2026-11-23 18:00:00.000", PeriodRecruiting, "2026-11-23 18:15:00.000"},
		{"2026-11-23 18:15:00.000", PeriodResults, "2026-11-23 18:30:00.000"},
	} {
		if got := NextPeriodChange(cal, at(c.from), c.period); !got.Equal(at(c.want)) {
			t.Errorf("NextPeriodChange(%s, %v) = %s, want %s", c.from, c.period, got, c.want)
		}
	}
}

func TestChangeMissed(t *testing.T) {
	cal := calendar.New(time.UTC)
	mon := time.Date(2026, 11, 23, 18, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		now, save time.Time
		period    Period
		want      bool
	}{
		{mon.Add(time.Hour), mon.Add(-time.Hour), PeriodCompetition, true},
		{mon.Add(time.Hour), mon.Add(30 * time.Minute), PeriodSealValidation, false},
		{mon.Add(-time.Hour), mon.Add(-2 * time.Hour), PeriodCompetition, false},           // last change a week ago
		{mon.Add(-time.Hour), mon.Add(-8 * 24 * time.Hour), PeriodCompetition, true},       // saved before last week's
		{mon.Add(20 * time.Minute), mon.Add(time.Minute), PeriodRecruiting, true},          // 16 minutes later
		{mon.Add(10 * time.Minute), mon.Add(time.Minute), PeriodRecruiting, true},          // Mobius: a minor period always counts as missed
		{mon.Add(10 * 24 * time.Hour), time.UnixMilli(1), PeriodCompetition, false},        // a new database
		{time.Date(2026, 11, 22, 20, 0, 0, 0, time.UTC), mon, PeriodCompetition, false},    // Sunday: this week's Monday ahead
		{time.Date(2026, 11, 22, 20, 0, 0, 0, time.UTC), mon.AddDate(0, 0, -8), PeriodCompetition, true},
	} {
		if got := ChangeMissed(cal, c.now, c.save, c.period); got != c.want {
			t.Errorf("ChangeMissed(now %s, saved %s, %v) = %v, want %v", c.now, c.save, c.period, got, c.want)
		}
	}
}

func TestDungeonAccess(t *testing.T) {
	for _, c := range []struct {
		open, validation          bool
		winner, cabal, sealOwner  Cabal
		ok                        bool
		msg                       int
	}{
		{true, true, CabalDawn, CabalNull, CabalNull, true, 0},
		{false, false, CabalNull, CabalDusk, CabalNull, true, 0},
		{false, false, CabalNull, CabalNull, CabalNull, false, 0},
		{false, true, CabalDawn, CabalDawn, CabalDawn, true, 0},
		{false, true, CabalDawn, CabalDawn, CabalNull, false, MsgOnlyDawn},
		{false, true, CabalDawn, CabalDusk, CabalDawn, false, MsgOnlyDawn},
		{false, true, CabalDusk, CabalDusk, CabalDusk, true, 0},
		{false, true, CabalDusk, CabalNull, CabalDusk, false, MsgOnlyDusk},
		{false, true, CabalNull, CabalDawn, CabalNull, true, 0},
		{false, true, CabalNull, CabalNull, CabalNull, false, 0},
	} {
		ok, msg := DungeonAccess(c.open, c.validation, c.winner, c.cabal, c.sealOwner)
		if ok != c.ok || msg != c.msg {
			t.Errorf("DungeonAccess(%+v) = %v %d", c, ok, msg)
		}
	}
}

func TestMustLeaveDungeon(t *testing.T) {
	if !MustLeaveDungeon(PeriodSealValidation, "dusk", true, CabalDawn) || MustLeaveDungeon(PeriodSealValidation, "dawn", true, CabalDawn) {
		t.Error("validation")
	}
	if !MustLeaveDungeon(PeriodSealValidation, "dawn", true, CabalNull) {
		t.Error("a tie sends everyone out (the short name No Cabal matches nobody)")
	}
	if !MustLeaveDungeon(PeriodRecruiting, "", false, CabalNull) || MustLeaveDungeon(PeriodRecruiting, "", true, CabalNull) {
		t.Error("recruiting")
	}
	if !MustLeaveDungeonOnEnter(PeriodCompetition, CabalNull, CabalDawn) || MustLeaveDungeonOnEnter(PeriodCompetition, CabalDusk, CabalDawn) ||
		!MustLeaveDungeonOnEnter(PeriodResults, CabalDusk, CabalDawn) {
		t.Error("on enter")
	}
}

func TestStrifeSkillAndTickets(t *testing.T) {
	if StrifeSkill(CabalDawn, CabalDawn) != VictorOfWarSkill || StrifeSkill(CabalDawn, CabalDusk) != VanquishedOfWarSkill ||
		StrifeSkill(CabalDawn, CabalNull) != 0 || StrifeSkill(CabalNull, CabalDawn) != 0 {
		t.Error("StrifeSkill")
	}
	if !DawnTicket(6115) || DawnTicket(6175) || !RookieTicket(6175) || RookieTicket(6115) {
		t.Error("tickets")
	}
	if MaxTax(CabalDawn) != 25 || MaxTax(CabalDusk) != 5 || MaxTax(CabalNull) != 15 {
		t.Error("MaxTax")
	}
}

func TestNames(t *testing.T) {
	if CabalDawn.ShortName() != "dawn" || CabalNull.ShortName() != "No Cabal" || CabalDusk.Name() != "Revolutionaries of Dusk" {
		t.Error("cabal names")
	}
	if SealStrife.Name(false) != "Seal of Strife" || SealGnosis.Name(true) != "Gnosis" || CabalOf("DAWN") != CabalDawn || CabalOf("") != CabalNull {
		t.Error("seal names")
	}
	if PeriodSealValidation.Next() != PeriodRecruiting || PeriodCompetition.Next() != PeriodResults {
		t.Error("Next")
	}
}
