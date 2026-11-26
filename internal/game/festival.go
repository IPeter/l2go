package game

import (
	"strings"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/sevensigns"
	"github.com/IPeter/l2go/internal/storage"
)

// The Festival of Darkness (Plan 14 M11l, Mobius SevenSignsFestival): the
// scores and bonuses (sevensigns.FestivalBook), saved with the Seven Signs,
// the reward of the best parties at the competition's end, and the record's
// second page. The festival cycles, the arenas and the guides are in
// festival_run.go.

// festivalClanWinMsg is CLAN_MEMBER_S1_WAS_AN_ACTIVE_MEMBER_OF_THE_HIGHEST_
// RANKED_PARTY_IN_THE_FESTIVAL_OF_DARKNESS_S2_POINTS_HAVE_BEEN_ADDED_TO_YOUR_
// CLAN_S_REPUTATION_SCORE.
const festivalClanWinMsg serverpackets.SystemMessageID = 1775

// festivalState is the festival on the Server. Loop-only.
type festivalState struct {
	book *sevensigns.FestivalBook
	run  festivalRun // festival_run.go
}

// restoreFestival is restoreFestivalData: the stored rows, cycle and
// bonuses; a new database starts with cycle 1's empty rows.
func (s *Server) restoreFestival(stored storage.SevenSigns) {
	f := &s.sevenSigns
	var bonus [sevensigns.FestivalCount]int
	cycle := 1
	if stored.HasStatus {
		cycle = int(stored.Status.FestivalCycle)
		for i, b := range stored.Status.AccumulatedBonus {
			if i < len(bonus) {
				bonus[i] = int(b)
			}
		}
	}
	rows := make([]sevensigns.FestivalScore, 0, len(stored.Festival))
	for _, r := range stored.Festival {
		rows = append(rows, sevensigns.FestivalScore{Festival: int(r.Festival), Cabal: sevensigns.CabalOf(r.Cabal), Cycle: int(r.Cycle),
			Date: r.Date, Score: int(r.Score), Members: r.Members})
	}
	f.festival.book = sevensigns.NewFestivalBook(cycle, f.state.Cycle, bonus, rows)
	if len(rows) == 0 {
		// seven_signs_festival.sql's rows for cycle 1.
		for id := range sevensigns.FestivalCount {
			for _, c := range []sevensigns.Cabal{sevensigns.CabalDawn, sevensigns.CabalDusk} {
				f.festival.book.SetRow(sevensigns.FestivalScore{Festival: id, Cabal: c, Cycle: 1})
			}
		}
	}
}

// festivalRows are the rows to store.
func (s *Server) festivalRows() []storage.SevenSignsFestivalScore {
	b := s.sevenSigns.festival.book
	if b == nil {
		return nil
	}
	var out []storage.SevenSignsFestivalScore
	for _, r := range b.Rows() {
		out = append(out, storage.SevenSignsFestivalScore{Festival: int32(r.Festival), Cabal: r.Cabal.ShortName(), Cycle: int32(r.Cycle), //nolint:gosec // small
			Date: r.Date, Score: int32(r.Score), Members: r.Members}) //nolint:gosec // scores
	}
	return out
}

// festivalRecordPage is the record's second page: each festival's level
// score and each cabal's best score of this cycle with its party.
func (s *Server) festivalRecordPage() []serverpackets.SSQFestival {
	b := s.sevenSigns.festival.book
	out := make([]serverpackets.SSQFestival, 0, sevensigns.FestivalCount)
	for id := range sevensigns.FestivalCount {
		e := serverpackets.SSQFestival{ID: int8(id + 1), LevelScore: int32(sevensigns.FestivalLevelScores[id])} //nolint:gosec // small
		if b != nil {
			dusk, dawn := b.HighestScoreData(sevensigns.CabalDusk, id), b.HighestScoreData(sevensigns.CabalDawn, id)
			e.DuskScore, e.DuskMembers = int32(dusk.Score), dusk.MemberNames() //nolint:gosec // scores
			e.DawnScore, e.DawnMembers = int32(dawn.Score), dawn.MemberNames() //nolint:gosec // scores
		} else {
			e.DuskMembers, e.DawnMembers = []string{""}, []string{""}
		}
		out = append(out, e)
	}
	return out
}

// rewardFestivalHighestRanked is rewardHighestRanked: the clan of every
// member of each festival's best party ever gains FestivalOfDarknessWin
// reputation, and its members online are told. Mobius finds an offline
// member's clan in the database; here the clan registry knows it.
func (s *Server) rewardFestivalHighestRanked() {
	b := s.sevenSigns.festival.book
	if b == nil {
		return
	}
	points := s.sevenSigns.cfg.FestivalOfDarknessWin
	for id := range sevensigns.FestivalCount {
		best, ok := b.OverallHighest(id)
		if !ok {
			continue
		}
		for _, name := range best.MemberNames() {
			cl := s.clanOfCharacterName(name)
			if cl == nil {
				continue
			}
			s.AddClanReputation(cl, points)
			broadcastToClan(cl, clanSysMsg(festivalClanWinMsg, serverpackets.TextParam(name), serverpackets.NumberParam(int32(points)))) //nolint:gosec // config value
		}
	}
}

// clanOfCharacterName returns the clan of a character by name, online or
// not, or nil.
func (s *Server) clanOfCharacterName(name string) *Clan {
	if name == "" {
		return nil
	}
	if p, ok := s.PlayerByName(name); ok {
		return p.Clan()
	}
	for _, cl := range s.Clans() {
		for _, m := range cl.Members() {
			if strings.EqualFold(m.Name(), name) {
				return cl
			}
		}
	}
	return nil
}

// resetFestival is resetFestivalData: a new cycle's empty scores, no bonus,
// no participants, and the players online lose their Blood Offerings.
func (s *Server) resetFestival() {
	f := &s.sevenSigns
	f.festival.book.Reset(f.state.Cycle)
	s.clearFestivalParties()
	for p := range s.Players() {
		if p.inv.Inventory == nil {
			continue
		}
		if it := p.inv.ByItemID(sevensigns.OfferingID); it != nil {
			s.DestroyItem(p, it, it.Count(), false)
		}
	}
	s.log.Info("game: festival of darkness reinitialized for the next competition")
}
