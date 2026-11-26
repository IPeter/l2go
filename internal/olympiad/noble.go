package olympiad

import (
	"cmp"
	"math"
	"slices"
)

// Noble is a noble of the running period (a row of Mobius olympiad_nobles).
type Noble struct {
	CharID int32
	// ClassID is the class the noble registered with first.
	ClassID int
	Name    string
	Points  int
	// Done, Won, Lost and Drawn count the matches.
	Done, Won, Lost, Drawn int
}

// Nobles are the nobles of the running period by character, and those of
// the last period's end (Mobius olympiad_nobles and olympiad_nobles_eom).
// The zero value is empty and ready to use.
type Nobles struct {
	byID map[int32]*Noble
	// order keeps the nobles in the order they came, for stable output.
	order []int32
	eom   []Noble
}

// NewNobles returns the table of the stored nobles: points are clamped as
// Mobius load does, to [0, MaxPoints × matches done + WeeklyPoints × 4].
func NewNobles(c Config, nobles, eom []Noble) *Nobles {
	t := &Nobles{}
	for _, n := range nobles {
		n.Points = c.clampPoints(n.Points, n.Done)
		t.put(n)
	}
	t.eom = slices.Clone(eom)
	return t
}

// clampPoints is Math.clamp(points, 0, MaxPoints × done + WeeklyPoints × 4).
func (c Config) clampPoints(points, done int) int {
	hi := c.MaxPoints*done + c.WeeklyPoints*4
	return max(0, min(points, hi))
}

func (t *Nobles) put(n Noble) *Noble {
	if t.byID == nil {
		t.byID = make(map[int32]*Noble)
	}
	if old, ok := t.byID[n.CharID]; ok {
		*old = n
		return old
	}
	p := &n
	t.byID[n.CharID] = p
	t.order = append(t.order, n.CharID)
	return p
}

// Get returns a noble, nil if the character is none.
func (t *Nobles) Get(charID int32) *Noble {
	if t == nil {
		return nil
	}
	return t.byID[charID]
}

// Len is Mobius getNobleCount.
func (t *Nobles) Len() int {
	if t == nil {
		return 0
	}
	return len(t.byID)
}

// Add makes a character a noble of the period with StartPoints (Mobius
// registerNoble for a character not in NOBLES) and returns it; a noble
// already there is returned as it is.
func (t *Nobles) Add(c Config, charID int32, classID int, name string) *Noble {
	if n := t.Get(charID); n != nil {
		return n
	}
	return t.put(Noble{CharID: charID, ClassID: classID, Name: name, Points: c.StartPoints})
}

// All returns the nobles in the order they came.
func (t *Nobles) All() []Noble {
	if t == nil {
		return nil
	}
	out := make([]Noble, 0, len(t.order))
	for _, id := range t.order {
		out = append(out, *t.byID[id])
	}
	return out
}

// MonthEnd returns the nobles of the last period's end.
func (t *Nobles) MonthEnd() []Noble {
	if t == nil {
		return nil
	}
	return slices.Clone(t.eom)
}

// Points is Mobius getNoblePoints: 0 for a character that is no noble.
func (t *Nobles) Points(charID int32) int {
	if n := t.Get(charID); n != nil {
		return n.Points
	}
	return 0
}

// LastPoints is Mobius getLastNobleOlympiadPoints: the character's points
// at the last period's end, 0 if it had none.
func (t *Nobles) LastPoints(charID int32) int {
	if t == nil {
		return 0
	}
	for _, n := range t.eom {
		if n.CharID == charID {
			return n.Points
		}
	}
	return 0
}

// AddWeeklyPoints is Mobius addWeeklyPoints: every noble gets WeeklyPoints,
// clamped to [0, MaxPoints × done + WeeklyPoints × 4] (which takes points
// from a noble with few matches: 18 becomes 12 without a match, as in
// Mobius). Nothing happens in the validation period.
func (t *Nobles) AddWeeklyPoints(c Config, p Period) {
	if p == Validation || t == nil {
		return
	}
	for _, id := range t.order {
		n := t.byID[id]
		n.Points = c.clampPoints(n.Points+c.WeeklyPoints, n.Done)
	}
}

// CopyToMonthEnd is Mobius updateMonthlyData: the month-end table becomes
// a copy of the nobles.
func (t *Nobles) CopyToMonthEnd() { t.eom = t.All() }

// Clear is Mobius deleteNobles: the nobles of the period are gone (the
// month-end table stays).
func (t *Nobles) Clear() {
	t.byID = nil
	t.order = nil
}

// heroOrder is the ORDER BY of Mobius's hero and ranking queries: points,
// then matches, then wins, all descending. SQL leaves ties in no order; the
// character ID decides here.
func heroOrder(a, b Noble) int {
	return cmp.Or(cmp.Compare(b.Points, a.Points), cmp.Compare(b.Done, a.Done), cmp.Compare(b.Won, a.Won), cmp.Compare(a.CharID, b.CharID))
}

// HeroClasses are the classes that get a hero, in Mobius's order
// (Olympiad.HERO_IDS): the third classes 88-118 and 131-134.
var HeroClasses = []int{
	88, 89, 90, 91, 92, 93, 94, 95, 96, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110,
	111, 112, 113, 114, 115, 116, 117, 118, 131, 132, 133, 134,
}

// Candidate is a hero to be (Mobius HEROS_TO_BE).
type Candidate struct {
	CharID  int32
	ClassID int
	Name    string
}

// HeroesToBe is Mobius sortHerosToBe: per class of HeroClasses the noble
// with the most points, then matches, then wins, among those with at least
// 9 matches and a win (OLYMPIAD_GET_HEROS); of the two soulhound classes
// (132, 133) only the better one.
func (t *Nobles) HeroesToBe() []Candidate {
	var out, soulHounds []Candidate
	var hounds []Noble
	for _, class := range HeroClasses {
		var best *Noble
		for _, n := range t.All() {
			if n.ClassID != class || n.Done < 9 || n.Won <= 0 {
				continue
			}
			if best == nil || heroOrder(n, *best) < 0 {
				b := n
				best = &b
			}
		}
		if best == nil {
			continue
		}
		c := Candidate{CharID: best.CharID, ClassID: class, Name: best.Name}
		if class == 132 || class == 133 {
			soulHounds = append(soulHounds, c)
			hounds = append(hounds, *best)
			continue
		}
		out = append(out, c)
	}
	switch len(hounds) {
	case 1:
		out = append(out, soulHounds[0])
	case 2:
		// Points, then matches, then wins; the second on a full tie.
		a, b := hounds[0], hounds[1]
		win := soulHounds[1]
		switch {
		case a.Points > b.Points:
			win = soulHounds[0]
		case b.Points > a.Points:
		case a.Done > b.Done:
			win = soulHounds[0]
		case b.Done > a.Done:
		case a.Won > b.Won:
			win = soulHounds[0]
		}
		out = append(out, win)
	}
	return out
}

// ClassLeaders is Mobius getClassLeaderBoard: the names of the first ten
// nobles of a class with at least 9 matches, from the last period's end
// when monthly is set (OlympiadShowMonthlyWinners), else from the running
// period. Class 132 counts 133 too.
func (t *Nobles) ClassLeaders(classID int, monthly bool) []string {
	list := t.All()
	if monthly {
		list = t.MonthEnd()
	}
	var picked []Noble
	for _, n := range list {
		if (n.ClassID == classID || (classID == 132 && n.ClassID == 133)) && n.Done >= 9 {
			picked = append(picked, n)
		}
	}
	slices.SortFunc(picked, heroOrder)
	var names []string
	for i, n := range picked {
		if i == 10 {
			break
		}
		names = append(names, n.Name)
	}
	return names
}

// Ranks is Mobius loadNoblesRank's table: the nobles of the last period's
// end with at least 9 matches, ranked 1 (the top 1 %) to 5 (the rest
// below 50 %).
func (t *Nobles) Ranks() map[int32]int {
	var classified []Noble
	for _, n := range t.MonthEnd() {
		if n.Done >= 9 {
			classified = append(classified, n)
		}
	}
	slices.SortFunc(classified, heroOrder)
	size := float64(len(classified))
	rank1 := int(math.Round(size * 0.01))
	rank2 := int(math.Round(size * 0.10))
	rank3 := int(math.Round(size * 0.25))
	rank4 := int(math.Round(size * 0.50))
	if rank1 == 0 {
		rank1 = 1
		rank2++
		rank3++
		rank4++
	}
	ranks := make(map[int32]int, len(classified))
	for i, n := range classified {
		place := i + 1
		switch {
		case place <= rank1:
			ranks[n.CharID] = 1
		case place <= rank2:
			ranks[n.CharID] = 2
		case place <= rank3:
			ranks[n.CharID] = 3
		case place <= rank4:
			ranks[n.CharID] = 4
		default:
			ranks[n.CharID] = 5
		}
	}
	return ranks
}

// SettlePasses is the rest of Mobius loadNoblesRank, getNoblessePasses for
// every noble in the validation period: a ranked noble with points loses
// them and earns HeroPoints × GPPerPoint if it is a hero (a claimed one),
// nothing otherwise (Mobius gives the rank no points). It returns the
// passes earned by character, to keep for the noble until claimed.
func (t *Nobles) SettlePasses(c Config, p Period, ranks map[int32]int, isHero func(charID int32) bool) map[int32]int {
	out := map[int32]int{}
	if p != Validation || len(ranks) == 0 || t == nil {
		return out
	}
	for _, id := range t.order {
		n := t.byID[id]
		if _, ok := ranks[id]; !ok || n.Points == 0 {
			continue
		}
		points := 0
		if isHero(id) {
			points = c.HeroPoints
		}
		points *= c.GPPerPoint
		n.Points = 0
		if points > 0 {
			out[id] = points
		}
	}
	return out
}
