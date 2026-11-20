package game

import (
	"cmp"
	"context"
	"maps"
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/storage"
)

// Raid points (Mobius RaidBossPointsManager): each member of the party
// that kills a raid or grand boss wins level / 2 + Rnd(−5, 5) points on that
// boss. Every day at 00:10 a task runs; on Mondays it ranks the characters
// by their points, hands the ranking to the OnRaidPointsReset hooks (the
// clans' reputation, Milestone 9) and deletes every character's points.

// raidResetHour and raidResetMinute are when the daily reset task runs
// (TimeUtil.getNextTime(0, 10)); raidResetPeriod is its period.
const (
	raidResetHour   = 0
	raidResetMinute = 10
	raidResetPeriod = 24 * time.Hour
)

// setupRaidPoints loads the stored points and starts the reset task.
func (s *Server) setupRaidPoints(rows []storage.RaidPoints) {
	f := &s.bosses
	for _, r := range rows {
		m := f.points[r.CharID]
		if m == nil {
			m = make(map[int32]int32)
			f.points[r.CharID] = m
		}
		m[r.BossID] = r.Points
	}
	now := s.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), raidResetHour, raidResetMinute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	f.resetTimer = s.loop.After(next.Sub(now), func() {
		s.resetRaidPoints()
		f.resetTimer = s.loop.Every(raidResetPeriod, s.resetRaidPoints)
	})
}

// OnRaidPointsReset registers fn to run at the weekly reset of the raid
// points with the ranking of the characters (object ID → rank, from 1),
// before the points are deleted (Mobius rewards clan reputation for the
// first 100 there). Call it from a feature's setup function.
func (s *Server) OnRaidPointsReset(fn func(ranking map[int32]int)) {
	s.bosses.resetHooks = append(s.bosses.resetHooks, fn)
}

// resetRaidPoints is Mobius resetRaidPoints: on Mondays only, the hooks get
// the ranking, then every character's points are deleted.
func (s *Server) resetRaidPoints() {
	if s.Now().Weekday() != time.Monday {
		return
	}
	ranking := s.RaidRanking()
	for _, fn := range s.bosses.resetHooks {
		s.safely("OnRaidPointsReset hook", func() { fn(ranking) })
	}
	clear(s.bosses.points)
	if store := s.bosses.store; store != nil {
		s.bosses.writer.enqueue("raid points reset", store.ClearRaidPoints)
	}
	s.log.Info("raid points reset")
}

// AddRaidPoints adds points on a boss to p's raid points (Mobius addPoints)
// and stores them. Loop-only.
func (s *Server) AddRaidPoints(p *Player, bossID, points int) {
	f := &s.bosses
	id := p.ObjectID()
	m := f.points[id]
	if m == nil {
		m = make(map[int32]int32)
		f.points[id] = m
	}
	boss := int32(bossID)    //nolint:gosec // NPC IDs are small
	m[boss] += int32(points) //nolint:gosec // raid points are small
	row := storage.RaidPoints{CharID: id, BossID: boss, Points: m[boss]}
	if store := f.store; store != nil {
		f.writer.enqueue("raid points", func(ctx context.Context) error {
			return store.SaveRaidPoints(ctx, []storage.RaidPoints{row})
		})
	}
}

// RaidPoints returns a character's points per boss, nil for none (Mobius
// getList). Loop-only.
func (s *Server) RaidPoints(charID int32) map[int32]int32 {
	return maps.Clone(s.bosses.points[charID])
}

// RaidPointsTotal returns the sum of a character's raid points (Mobius
// getPointsByOwnerId). Loop-only.
func (s *Server) RaidPointsTotal(charID int32) int {
	total := 0
	for _, v := range s.bosses.points[charID] {
		total += int(v)
	}
	return total
}

// RaidRanking returns the rank of every character with points, from 1 for
// the most points (Mobius getRankList); characters with the same points
// rank by object ID. Loop-only.
func (s *Server) RaidRanking() map[int32]int {
	type entry struct {
		id    int32
		total int
	}
	var list []entry
	for id := range s.bosses.points {
		if t := s.RaidPointsTotal(id); t != 0 {
			list = append(list, entry{id, t})
		}
	}
	slices.SortFunc(list, func(a, b entry) int { return cmp.Or(cmp.Compare(b.total, a.total), cmp.Compare(a.id, b.id)) })
	out := make(map[int32]int, len(list))
	for i, e := range list {
		out[e.id] = i + 1
	}
	return out
}
