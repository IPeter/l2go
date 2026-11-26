package olympiad

import "slices"

// The heroes (Mobius Hero): every character that was ever a hero, with how
// often; the heroes of the running cycle are the played ones. A new hero is
// unclaimed until it claims the status at a Monument of Heroes.

// Hero is a row of Mobius heroes.
type Hero struct {
	CharID  int32
	ClassID int
	Name    string
	// Count is how many periods the character was a hero.
	Count int
	// Played: a hero of the running cycle. Claimed: it took the status.
	Played, Claimed bool
	// Message is the hero's words (RequestWriteHeroWords).
	Message string
}

// Heroes is the hero table. The zero value is empty and ready to use.
type Heroes struct {
	all []*Hero
}

// NewHeroes returns the table of the stored heroes.
func NewHeroes(list []Hero) *Heroes {
	h := &Heroes{}
	for _, x := range list {
		h.all = append(h.all, &x)
	}
	return h
}

func (h *Heroes) find(charID int32) *Hero {
	if h == nil {
		return nil
	}
	for _, x := range h.all {
		if x.CharID == charID {
			return x
		}
	}
	return nil
}

// All returns every hero ever, in the table's order.
func (h *Heroes) All() []Hero {
	if h == nil {
		return nil
	}
	out := make([]Hero, len(h.all))
	for i, x := range h.all {
		out[i] = *x
	}
	return out
}

// Current returns the heroes of the running cycle (Mobius HEROES).
func (h *Heroes) Current() []Hero {
	var out []Hero
	for _, x := range h.All() {
		if x.Played {
			out = append(out, x)
		}
	}
	return out
}

// Get returns a current hero, false if the character is none.
func (h *Heroes) Get(charID int32) (Hero, bool) {
	x := h.find(charID)
	if x == nil || !x.Played {
		return Hero{}, false
	}
	return *x, true
}

// IsHero is Mobius isHero: a current hero that claimed the status.
func (h *Heroes) IsHero(charID int32) bool {
	x := h.find(charID)
	return x != nil && x.Played && x.Claimed
}

// IsUnclaimed is Mobius isUnclaimedHero: a current hero that has not
// claimed the status yet.
func (h *Heroes) IsUnclaimed(charID int32) bool {
	x := h.find(charID)
	return x != nil && x.Played && !x.Claimed
}

// ByClass is Mobius getHeroByClass: the current hero of a class, 0 for
// none.
func (h *Heroes) ByClass(classID int) int32 {
	for _, x := range h.Current() {
		if x.ClassID == classID {
			return x.CharID
		}
	}
	return 0
}

// ComputeNewHeroes is Mobius computeNewHeroes for the table: no one is a
// current hero any more, then each candidate is: an old hero counts one
// more period, a new one starts at 1; both unclaimed. It returns the heroes
// that lost the status, whose hero items and skills go (Mobius does that
// for those online, and deletes the hero items of everyone).
func (h *Heroes) ComputeNewHeroes(candidates []Candidate) (former []int32) {
	for _, x := range h.all {
		if x.Played {
			former = append(former, x.CharID)
		}
		x.Played = false
	}
	for _, c := range candidates {
		if x := h.find(c.CharID); x != nil {
			x.Count++
			x.Played, x.Claimed = true, false
			continue
		}
		h.all = append(h.all, &Hero{CharID: c.CharID, ClassID: c.ClassID, Name: c.Name, Count: 1, Played: true})
	}
	return former
}

// Claim is the table part of Mobius claimHero: the character becomes a
// claimed current hero with empty words. A character that was no hero
// gets a row of its own, as Mobius puts an empty one (class and name as
// given).
func (h *Heroes) Claim(charID int32, classID int, name string) {
	x := h.find(charID)
	if x == nil {
		x = &Hero{CharID: charID, ClassID: classID, Name: name}
		h.all = append(h.all, x)
	}
	x.Played, x.Claimed = true, true
	x.Message = ""
}

// Revoke takes the hero status from a current hero (not Mobius: the admin
// page's Hero off button): it stays an all-time hero.
func (h *Heroes) Revoke(charID int32) bool {
	x := h.find(charID)
	if x == nil || !x.Played {
		return false
	}
	x.Played, x.Claimed = false, false
	return true
}

// SetMessage is Mobius setHeroMessage for a current hero.
func (h *Heroes) SetMessage(charID int32, msg string) bool {
	x := h.find(charID)
	if x == nil || !x.Played {
		return false
	}
	x.Message = msg
	return true
}

// Rename follows a character's new name.
func (h *Heroes) Rename(charID int32, name string) {
	if x := h.find(charID); x != nil {
		x.Name = name
	}
}

// IDs returns the character IDs of the claimed current heroes, sorted.
func (h *Heroes) IDs() []int32 {
	var out []int32
	for _, x := range h.Current() {
		if x.Claimed {
			out = append(out, x.CharID)
		}
	}
	slices.Sort(out)
	return out
}

// HeroItems are the items only heroes may wear, deleted when a cycle ends
// (Mobius Hero.DELETE_ITEMS): Wings of Destiny Circlet and the eleven
// Infinity weapons.
var HeroItems = []int{6842, 6611, 6612, 6613, 6614, 6615, 6616, 6617, 6618, 6619, 6620, 6621}

// The hero diary actions (Mobius Hero.ACTION_*).
const (
	DiaryRaidKilled  = 1
	DiaryHeroGained  = 2
	DiaryCastleTaken = 3
)

// HeroSocialAction is the hero animation of a claim (SocialAction 20016).
const HeroSocialAction = 20016
