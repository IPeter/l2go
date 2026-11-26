package sevensigns

import (
	"sort"
	"strings"
)

// The Festival of Darkness (Mobius SevenSignsFestival): five festivals by
// level range for each cabal, the best score of each per cycle, the score
// the best one brings its cabal, and the ancient adena bonus the offerings
// build up.

// FestivalCount is FESTIVAL_COUNT.
const FestivalCount = 5

// Festival items (Mobius SevenSignsFestival).
const (
	// OfferingID is the Blood Offering (FESTIVAL_OFFERING_ID); each is worth
	// OfferingValue points.
	OfferingID    = 5901
	OfferingValue = 5
)

// FestivalLevelScores are FESTIVAL_LEVEL_SCORES: what the best party of a
// festival brings its cabal (500 in all).
var FestivalLevelScores = [FestivalCount]int{60, 70, 100, 120, 150}

// FestivalMaxLevel is getMaxLevelForFestival: 31, 42, 53, 64, or the last
// level below the maximum (maxLevel - 1) for the fifth.
func FestivalMaxLevel(id, maxLevel int) int {
	switch id {
	case 0:
		return 31
	case 1:
		return 42
	case 2:
		return 53
	case 3:
		return 64
	}
	return maxLevel - 1
}

// FestivalName is getFestivalName.
func FestivalName(id int) string {
	switch id {
	case 0:
		return "Level 31 or lower"
	case 1:
		return "Level 42 or lower"
	case 2:
		return "Level 53 or lower"
	case 3:
		return "Level 64 or lower"
	}
	return "No Level Limit"
}

// IsFestivalArcher is isFestivalArcher.
func IsFestivalArcher(npcID int) bool {
	if npcID < 18009 || npcID > 18108 {
		return false
	}
	id := npcID % 10
	return id == 4 || id == 9
}

// IsFestivalChest is Mobius isFestivalChest, which says true for every NPC
// that is NOT a chest (18109-18118): kept as Mobius has it.
func IsFestivalChest(npcID int) bool { return npcID < 18109 || npcID > 18118 }

// FestivalScore is a festival's best score of a cycle for a cabal (a
// Mobius seven_signs_festival row).
type FestivalScore struct {
	Festival int
	Cabal    Cabal
	Cycle    int
	// Date is Unix milliseconds, 0 for none.
	Date    int64
	Score   int
	Members string
}

// MemberNames splits Members as Java's split(","): an empty list is one
// empty name.
func (f FestivalScore) MemberNames() []string {
	names := strings.Split(f.Members, ",")
	for len(names) > 1 && names[len(names)-1] == "" {
		names = names[:len(names)-1]
	}
	return names
}

// FestivalBook is the festival's scores of every cycle and the accumulated
// bonuses (Mobius _festivalData and _accumulatedBonuses).
type FestivalBook struct {
	// Cycle is _festivalCycle, SignsCycle _signsCycle: the Seven Signs
	// cycle whose scores count.
	Cycle, SignsCycle int
	Bonus             [FestivalCount]int
	rows              map[int]map[int]*FestivalScore // cycle -> festival (+5 for Dawn)
}

// NewFestivalBook returns a book with the rows.
func NewFestivalBook(cycle, signsCycle int, bonus [FestivalCount]int, rows []FestivalScore) *FestivalBook {
	b := &FestivalBook{Cycle: cycle, SignsCycle: signsCycle, Bonus: bonus, rows: make(map[int]map[int]*FestivalScore)}
	for i := range rows {
		r := rows[i]
		b.put(&r)
	}
	return b
}

func offsetOf(c Cabal, id int) int {
	if c == CabalDawn {
		return id + FestivalCount
	}
	return id
}

// SetRow stores a row (replacing the row of its cycle, cabal and festival).
func (b *FestivalBook) SetRow(r FestivalScore) { b.put(&r) }

func (b *FestivalBook) put(r *FestivalScore) {
	m := b.rows[r.Cycle]
	if m == nil {
		m = make(map[int]*FestivalScore)
		b.rows[r.Cycle] = m
	}
	m[offsetOf(r.Cabal, r.Festival)] = r
}

// Rows returns copies of every row, sorted by festival, cabal (Dawn first,
// as "dawn" < "dusk") and cycle.
func (b *FestivalBook) Rows() []FestivalScore {
	var out []FestivalScore
	for _, m := range b.rows {
		for _, r := range m {
			out = append(out, *r)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, c := out[i], out[j]
		if a.Festival != c.Festival {
			return a.Festival < c.Festival
		}
		if a.Cabal != c.Cabal {
			return a.Cabal > c.Cabal
		}
		return a.Cycle < c.Cycle
	})
	return out
}

// HighestScoreData is getHighestScoreData: the best score of this cycle,
// an empty one when there is none.
func (b *FestivalBook) HighestScoreData(c Cabal, id int) FestivalScore {
	if r, ok := b.rows[b.SignsCycle][offsetOf(c, id)]; ok {
		return *r
	}
	return FestivalScore{Festival: id, Cabal: c, Cycle: b.SignsCycle}
}

// OverallHighest is getOverallHighestScoreData: the best score of the
// festival in any cycle and for either cabal (the first found on a tie, in
// the order of Rows), false when no score is above 0.
func (b *FestivalBook) OverallHighest(id int) (FestivalScore, bool) {
	var best FestivalScore
	found := false
	for _, r := range b.Rows() {
		if r.Festival == id && r.Score > best.Score {
			best, found = r, true
		}
	}
	return best, found
}

// Reset is resetFestivalData: a new cycle with empty scores for both
// cabals, no bonus.
func (b *FestivalBook) Reset(signsCycle int) {
	b.Cycle = 0
	b.SignsCycle = signsCycle
	b.Bonus = [FestivalCount]int{}
	for id := range FestivalCount {
		for _, c := range []Cabal{CabalDusk, CabalDawn} {
			b.put(&FestivalScore{Festival: id, Cabal: c, Cycle: signsCycle})
		}
	}
}

// SetFinalScore is setFinalScore: a party's offerings score for its
// cabal's festival. A score above the cabal's best of this cycle becomes the
// best (with members and the date), unless the cabal already led the other;
// it brings the cabal the festival's points when it also beats the other
// cabal's best. It reports whether the score became the best, and the
// points to add to the cabal (0 for none).
func (b *FestivalBook) SetFinalScore(c Cabal, id, score int, members []string, now int64) (best bool, points int) {
	if id < 0 || id >= FestivalCount {
		return false, 0
	}
	other := CabalDusk
	if c == CabalDusk {
		other = CabalDawn
	}
	mine := b.HighestScoreData(c, id).Score
	theirs := b.HighestScoreData(other, id).Score
	if score <= mine || mine > theirs {
		return false, 0
	}
	b.put(&FestivalScore{Festival: id, Cabal: c, Cycle: b.SignsCycle, Date: now, Score: score, Members: strings.Join(members, ",")})
	if score > theirs {
		points = FestivalLevelScores[id]
	}
	return true, points
}

// AddBonus is addAccumulatedBonus: offered seal stones of a kind raise a
// festival's bonus by their ancient adena value.
func (b *FestivalBook) AddBonus(id, stoneID, count int) {
	if id < 0 || id >= FestivalCount {
		return
	}
	each := 0
	switch stoneID {
	case StoneBlueID:
		each = StoneBlueValue
	case StoneGreenID:
		each = StoneGreenValue
	case StoneRedID:
		each = StoneRedValue
	}
	b.Bonus[id] += count * each
}

// TotalBonus is getTotalAccumulatedBonus.
func (b *FestivalBook) TotalBonus() int {
	n := 0
	for _, v := range b.Bonus {
		n += v
	}
	return n
}

// DistribBonus is distribAccumulatedBonus: a member of the winning cabal
// whose name is in a best party of this cycle (a substring test, as Mobius
// does) takes its share of that festival's bonus (bonus / party size); the
// bonus shrinks by it. The first matching festival in Mobius's map order
// (Dusk 0-4, then Dawn 0-4) counts.
func (b *FestivalBook) DistribBonus(name string, playerCabal, winner Cabal) int {
	if playerCabal != winner {
		return 0
	}
	m := b.rows[b.SignsCycle]
	for off := range 2 * FestivalCount {
		r, ok := m[off]
		if !ok || !strings.Contains(r.Members, name) {
			continue
		}
		n := len(r.MemberNames())
		share := b.Bonus[r.Festival] / n
		b.Bonus[r.Festival] -= share
		return share
	}
	return 0
}
