package social

import (
	"cmp"
	"slices"
	"sort"
)

// MaxReputation caps a clan's reputation both ways (Clan.setReputationScore:
// ±100,000,000).
const MaxReputation = 100_000_000

// ReputationChange is Clan.setReputationScore for score + delta: the new
// score, capped at ±MaxReputation, and whether clan skills switch off (the
// score drops from 0 or more to below 0) or back on (from below 0 to 0 or
// more). Mobius compares the uncapped value; the cap never changes its
// sign.
func ReputationChange(score, delta int) (newScore int, skillsOff, skillsOn bool) {
	v := score + delta
	skillsOff = score >= 0 && v < 0
	skillsOn = score < 0 && v >= 0
	return min(max(v, -MaxReputation), MaxReputation), skillsOff, skillsOn
}

// WarKillReputation is the reputation part of Player.doDie for a kill in a
// mutual war (Player.java 5143–5160), in Mobius order: the killer's clan
// gains perKill if the victim's clan's score is above 0; then the victim's
// clan loses perKill if the killer's clan's score, after that gain, is
// above 0. It returns the deltas to apply with ReputationChange.
func WarKillReputation(perKill, killerScore, victimScore int) (killerDelta, victimDelta int) {
	if victimScore > 0 {
		killerDelta = perKill
		killerScore, _, _ = ReputationChange(killerScore, perKill)
	}
	if killerScore > 0 {
		victimDelta = -perKill
	}
	return killerDelta, victimDelta
}

// ClanStanding is a clan as the clan ranking sees it.
type ClanStanding struct {
	Level, Reputation int
}

// ClanRank is ClanTable.getClanRank for clans[i]: 0 below clan level 3,
// else 1 plus the number of other clans (any level) above it, where a clan
// is above when its level is higher, or equal with a reputation at least as
// high (ties rank each other down, as in Mobius).
func ClanRank(clans []ClanStanding, i int) int {
	c := clans[i]
	if c.Level < 3 {
		return 0
	}
	rank := 1
	for j, o := range clans {
		if j != i && (c.Level < o.Level || (c.Level == o.Level && c.Reputation <= o.Reputation)) {
			rank++
		}
	}
	return rank
}

// ClanRanks is ClanRank for every clan, in O(n log n) for the 20-minute
// ranking of all clans.
func ClanRanks(clans []ClanStanding) []int {
	sorted := slices.Clone(clans)
	slices.SortFunc(sorted, func(a, b ClanStanding) int {
		if c := cmp.Compare(a.Level, b.Level); c != 0 {
			return c
		}
		return cmp.Compare(a.Reputation, b.Reputation)
	})
	n := len(sorted)
	ranks := make([]int, len(clans))
	for i, c := range clans {
		if c.Level < 3 {
			continue
		}
		// The first clan not below c: same level with reputation >= c's, or
		// a higher level. Everyone from there on is above c, c included.
		k := sort.Search(n, func(j int) bool {
			s := sorted[j]
			return s.Level > c.Level || (s.Level == c.Level && s.Reputation >= c.Reputation)
		})
		ranks[i] = 1 + (n - k - 1)
	}
	return ranks
}
