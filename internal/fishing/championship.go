package fishing

import (
	"strconv"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/calendar"
)

// The reward states of a championship fisher (the rewarded column of Mobius
// fishing_championship).
const (
	RewardNone  = 0 // a fisher of the running week
	RewardDue   = 1 // a winner of last week who has not claimed the prize
	RewardTaken = 2 // a winner who claimed it
)

// ChampionshipRanks is the number of fishers ranked (and rewarded).
const ChampionshipRanks = 5

// ChampionshipConfig is the championship part of Mobius General.ini.
type ChampionshipConfig struct {
	// Enabled is AltFishChampionshipEnabled (true).
	Enabled bool
	// RewardItem is AltFishChampionshipRewardItemId (57, adena).
	RewardItem int
	// Rewards are AltFishChampionshipReward1-5 (800,000 / 500,000 /
	// 300,000 / 200,000 / 100,000).
	Rewards [ChampionshipRanks]int
}

// DefaultChampionshipConfig returns the Mobius defaults.
func DefaultChampionshipConfig() ChampionshipConfig {
	return ChampionshipConfig{Enabled: true, RewardItem: 57, Rewards: [ChampionshipRanks]int{800000, 500000, 300000, 200000, 100000}}
}

// The end of a championship week: Tuesday 19:00 (setEndOfChamp).
const (
	ChampionshipEndDay  = time.Tuesday
	ChampionshipEndHour = 19
)

// Fisher is a fisher of the championship, Mobius FishingChampionshipManager
// .Fisher.
type Fisher struct {
	Name   string
	Length float64
	Reward int
}

// ChampionshipState is what the store keeps (decision X4): the end of the
// running week (Mobius global variable fishChampionshipEnd; zero for none)
// and the fishers, last week's winners first, as Mobius's shutdown writes
// them.
type ChampionshipState struct {
	End     time.Time
	Fishers []Fisher
}

// Championship is Mobius FishingChampionshipManager: the five biggest
// fish of the running week (each fisher once), and last week's five winners,
// who claim their prize from a fisherman.
type Championship struct {
	cfg ChampionshipConfig
	end time.Time

	tmp, win []*Fisher

	names, lengths       []string // the ranking of the running week, as last refreshed
	winNames, winLengths []string
	minLength            float64
	needRefresh          bool
}

// NewChampionship restores the championship from st as Mobius's constructor
// does (restoreData, refreshWinResult, recalculateMinLength). When the
// stored end has passed (or there is none) the week ends now (finished),
// and the caller saves the new state.
func NewChampionship(cfg ChampionshipConfig, st ChampionshipState, now time.Time, cal calendar.Calendar) (c *Championship, finished bool) {
	c = &Championship{cfg: cfg, end: st.End, needRefresh: true}
	for i := range st.Fishers {
		f := st.Fishers[i]
		switch {
		case f.Reward == RewardNone:
			c.tmp = append(c.tmp, &f)
		case f.Reward > RewardNone:
			c.win = append(c.win, &f)
		}
	}
	c.refreshWinResult()
	c.recalculateMinLength()
	if !c.end.After(now) {
		c.end = now
		c.Finish(cal)
		return c, true
	}
	return c, false
}

// Config returns the championship's settings.
func (c *Championship) Config() ChampionshipConfig { return c.cfg }

// End returns when the running week ends.
func (c *Championship) End() time.Time { return c.end }

// Finish ends the week (finishChamp.run): this week's fishers become the
// winners, waiting for their prize, and the next week ends at the Tuesday
// 19:00 that setEndOfChamp finds from the old end. The caller saves the
// state and schedules the next Finish at End.
func (c *Championship) Finish(cal calendar.Calendar) {
	c.win = c.win[:0]
	for _, f := range c.tmp {
		f.Reward = RewardDue
		c.win = append(c.win, f)
	}
	c.tmp = nil
	c.refreshWinResult()
	c.end = EndOfWeek(cal, c.end)
}

// EndOfWeek is Mobius setEndOfChamp from end: minute and second to 0, six
// days later, Tuesday of that week (Java's set(DAY_OF_WEEK) within the
// calendar's week), 19:00; the milliseconds stay.
func EndOfWeek(cal calendar.Calendar, end time.Time) time.Time {
	t := cal.In(end)
	t = cal.SetTime(t, t.Hour(), 0, 0)
	t = cal.AddDays(t, 6)
	t = cal.SetDayOfWeek(t, ChampionshipEndDay)
	return cal.SetTime(t, ChampionshipEndHour, 0, 0)
}

// Catch is a fish caught for the championship.
type Catch struct {
	// Length is the fish's length, shown with JavaDouble ("You caught a
	// fish $s1 in length.").
	Length float64
	// Registered is set when it entered the ranking ("Because of the size
	// of fish caught, you will be registered in the ranking.").
	Registered bool
}

// NewFish is Mobius newFish: a landed fish is Rnd.get(60, 89) +
// Rnd.get(0, 1000)/1000 long, plus Rnd.get(0, 3000)/1000 with a prize lure
// (8484-8486). While fewer than five fishers are ranked, a new one enters
// and a ranked one keeps the longer fish; afterwards a fish longer than the
// shortest ranked one replaces it (or the fisher's own shorter fish). False
// when the championship is off (nothing is drawn then).
func (c *Championship) NewFish(r Rand, name string, lureID int) (Catch, bool) {
	if !c.cfg.Enabled {
		return Catch{}, false
	}
	length := float64(r.Between(60, 89)) + float64(r.Between(0, 1000))/1000.
	if lureID >= 8484 && lureID <= 8486 {
		length += float64(r.Between(0, 3000)) / 1000.
	}
	out := Catch{Length: length}
	if len(c.tmp) >= ChampionshipRanks && !(c.minLength < length) {
		return out, true
	}
	for _, f := range c.tmp {
		if strings.EqualFold(f.Name, name) {
			if f.Length < length {
				f.Length = length
				out.Registered = true
				c.recalculateMinLength()
			}
			return out, true
		}
	}
	if len(c.tmp) >= ChampionshipRanks {
		var shortest *Fisher
		minLen := 99999.
		for _, f := range c.tmp {
			if f.Length < minLen {
				shortest, minLen = f, f.Length
			}
		}
		for i, f := range c.tmp {
			if f == shortest {
				c.tmp = append(c.tmp[:i:i], c.tmp[i+1:]...)
				break
			}
		}
	}
	c.tmp = append(c.tmp, &Fisher{Name: name, Length: length})
	out.Registered = true
	c.recalculateMinLength()
	return out, true
}

// recalculateMinLength is Mobius recalculateMinLength.
func (c *Championship) recalculateMinLength() {
	minLen := 99999.
	for _, f := range c.tmp {
		if f.Length < minLen {
			minLen = f.Length
		}
	}
	c.minLength = minLen
}

// IsWinner reports a name among last week's winners (Mobius isWinner, an
// exact comparison).
func (c *Championship) IsWinner(name string) bool {
	for _, n := range c.winNames {
		if n == name {
			return true
		}
	}
	return false
}

// Reward is Mobius getReward: each winner entry of the name (ignoring case)
// that has not claimed gets the prize of its rank and is marked claimed.
// It returns the counts of the reward item to give, in order; an entry
// without a prize gives none but is claimed all the same.
func (c *Championship) Reward(name string) []int {
	var counts []int
	for _, f := range c.win {
		if !strings.EqualFold(f.Name, name) || f.Reward == RewardTaken {
			continue
		}
		count := 0
		for x, n := range c.winNames {
			if strings.EqualFold(n, name) && x < ChampionshipRanks {
				count = c.cfg.Rewards[x]
			}
		}
		f.Reward = RewardTaken
		if count > 0 {
			counts = append(counts, count)
		}
	}
	return counts
}

// MidResult is the start of Mobius showMidResult: when the ranking needs a
// refresh it is refreshed and stale reports true (the fisherman shows
// fish_event003 and the caller calls Refreshed after 60 s); otherwise table
// is the running week's ranking for fish_event002.
func (c *Championship) MidResult() (table string, stale bool) {
	if c.needRefresh {
		c.refreshResult()
		return "", true
	}
	return rankingTable(c.names, c.lengths, "0"), false
}

// Refreshed is Mobius needRefresh.run: the next MidResult refreshes again.
func (c *Championship) Refreshed() { c.needRefresh = true }

// WinnerTable is the table of showChampScreen: last week's winners.
func (c *Championship) WinnerTable() string { return rankingTable(c.winNames, c.winLengths, "0") }

// MinutesLeft is Mobius getTimeRemaining: whole minutes until the end.
func (c *Championship) MinutesLeft(now time.Time) int64 {
	return c.end.Sub(now).Milliseconds() / 60000
}

// State returns what the store keeps: the end and the fishers, the winners
// first (Mobius shutdown).
func (c *Championship) State() ChampionshipState {
	st := ChampionshipState{End: c.end}
	for _, f := range c.win {
		st.Fishers = append(st.Fishers, *f)
	}
	for _, f := range c.tmp {
		st.Fishers = append(st.Fishers, Fisher{Name: f.Name, Length: f.Length})
	}
	return st
}

// Ranking returns the running week's fishers in their current order and
// the winners, for tests and GM pages.
func (c *Championship) Ranking() (running, winners []Fisher) {
	for _, f := range c.tmp {
		running = append(running, *f)
	}
	for _, f := range c.win {
		winners = append(winners, *f)
	}
	return running, winners
}

// refreshResult is Mobius refreshResult: the running week's fishers are
// bubble-sorted longest first, in place, and their names and lengths noted.
func (c *Championship) refreshResult() {
	c.needRefresh = false
	c.names, c.lengths = sortedRanking(c.tmp)
}

// refreshWinResult is Mobius refreshWinResult for the winners.
func (c *Championship) refreshWinResult() {
	c.winNames, c.winLengths = sortedRanking(c.win)
}

// sortedRanking bubble-sorts fishers longest first in place, as Mobius
// does, and returns their names and lengths (String.valueOf).
func sortedRanking(fishers []*Fisher) (names, lengths []string) {
	for range fishers {
		for y := 0; y <= len(fishers)-2; y++ {
			if fishers[y].Length < fishers[y+1].Length {
				fishers[y], fishers[y+1] = fishers[y+1], fishers[y]
			}
		}
	}
	for _, f := range fishers {
		names = append(names, f.Name)
		lengths = append(lengths, JavaDouble(f.Length))
	}
	return names, lengths
}

// rankingTable builds the rows of a ranking page. Mobius starts the text
// from a null String, so the table begins with "null"; kept for parity.
func rankingTable(names, lengths []string, none string) string {
	var b strings.Builder
	b.WriteString("null")
	for x := 1; x <= ChampionshipRanks; x++ {
		name, length := "None", none
		if len(names) >= x {
			name = names[x-1]
		}
		if len(lengths) >= x {
			length = lengths[x-1]
		}
		b.WriteString("<tr><td width=70 align=center>" + strconv.Itoa(x) + "</td>")
		b.WriteString("<td width=110 align=center>" + name + "</td>")
		b.WriteString("<td width=80 align=center>" + length + "</td></tr>")
	}
	return b.String()
}
