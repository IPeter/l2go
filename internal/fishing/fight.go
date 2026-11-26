package fishing

import "github.com/IPeter/l2go/internal/data"

// The lure types of ExFishingStartCombat.
const (
	LureTypeNewbie = 0
	LureTypeNormal = 1
	LureTypeNight  = 2 // an upper-grade fish, or any night lure
)

// Combat is what ExFishingStartCombat shows when the fish bites.
type Combat struct {
	Time      int // seconds left
	HP        int
	Mode      int // 0 resting, 1 fighting
	LureType  int
	Deceptive int // 1: pumping and reeling work the other way round
}

// Regen is an ExFishingHpRegen: the fight's state after a second or a skill.
type Regen struct {
	Time    int
	HP      int
	Mode    int // 0 resting, 1 fighting
	GoodUse int // 0 none, 1 success, 2 failure
	Anim    int // 1 pumping, 2 reeling (Mobius's values)
	Penalty int
	// Deceptive turns the HP bar purple.
	Deceptive int
}

// Output is where a fight's effects go: the game sends the messages and
// packets and ends the fishing.
type Output interface {
	// Message sends a system message, with the number arg when the message
	// has $s1.
	Message(m Message, arg int)
	// Regen sends ExFishingHpRegen: to everyone around with broadcast, else
	// to the fisher only.
	Regen(r Regen, broadcast bool)
	// Over ends the fight (Fishing.doDie): the fish is landed with win,
	// otherwise it got away. The fight does nothing afterwards.
	Over(win bool)
}

// Fight is a fishing fight, Mobius mechanics/fishing/Fishing: every second
// (Tick) the fish may change between resting and fighting, and regains HP
// while it fights; pumping hurts a resting fish and reeling a fighting one,
// unless the fish is deceptive. The fish is landed at 0 HP and gets away at
// twice its HP or when the time runs out.
type Fight struct {
	time      int
	stop      int
	goodUse   int
	anim      int
	mode      int
	deceptive int

	itemID int
	maxHP  int
	hp     int
	regen  float64
	upper  bool
	lureID int

	over bool
}

// NewFight starts the fight with a fish that bit (Mobius Fishing's
// constructor): an upper-grade fight (a hard fish or a night lure) is
// deceptive when Rnd.get(100) >= 90; then the fish fights when Rnd.get(100)
// >= 80. The caller broadcasts ExFishingStartCombat with the Combat, sends
// PlaySound SF_S_01 and "You've got a bite!", and calls Tick every second.
func NewFight(r Rand, f *data.Fish, noob, upper bool, lureID int) (*Fight, Combat) {
	ft := &Fight{
		maxHP: f.HP, hp: f.HP, regen: f.HPRegen, itemID: f.ItemID,
		time: f.CombatDuration, upper: upper, lureID: lureID,
	}
	lureType := LureTypeNormal
	switch {
	case upper:
		if r.Get(100) >= 90 {
			ft.deceptive = 1
		}
		lureType = LureTypeNight
	case noob:
		lureType = LureTypeNewbie
	}
	if r.Get(100) >= 80 {
		ft.mode = 1
	}
	return ft, Combat{Time: ft.time, HP: ft.maxHP, Mode: ft.mode, LureType: lureType, Deceptive: ft.deceptive}
}

// ItemID is the fish's item, given when it is landed.
func (f *Fight) ItemID() int { return f.itemID }

// LureID is the lure the fish took, which the championship looks at.
func (f *Fight) LureID() int { return f.lureID }

// HP returns the fish's HP now.
func (f *Fight) HP() int { return f.hp }

// Mode returns 0 while the fish rests and 1 while it fights.
func (f *Fight) Mode() int { return f.mode }

// Deceptive reports a fish that reverses pumping and reeling.
func (f *Fight) Deceptive() bool { return f.deceptive == 1 }

// TimeLeft returns the seconds left.
func (f *Fight) TimeLeft() int { return f.time }

// IsOver reports a fight that ended.
func (f *Fight) IsOver() bool { return f.over }

// Tick is one second of the fight (Fishing.run): at twice its HP the fish
// steals the bait, with no time left it spits the hook; otherwise the
// fish's turn (aiTask).
func (f *Fight) Tick(r Rand, out Output) {
	if f.over {
		return
	}
	switch {
	case f.hp >= f.maxHP*2:
		out.Message(MsgBaitStolen, 0)
		f.die(false, out)
	case f.time <= 0:
		out.Message(MsgSpitTheHook, 0)
		f.die(false, out)
	default:
		f.turn(r, out)
	}
}

// turn is Fishing.aiTask: a second less; a fighting fish regains its regen;
// every other second Rnd.get(100) >= 70 switches the mode and, for an
// upper-grade fish, Rnd.get(100) >= 90 the deception. ExFishingHpRegen goes
// to the fisher (to everyone while an animation shows).
func (f *Fight) turn(r Rand, out Output) {
	f.time--
	if f.mode == 1 {
		f.hp += int(f.regen)
	}
	if f.stop == 0 {
		f.stop = 1
		if r.Get(100) >= 70 {
			f.mode = 1 - f.mode
		}
		if f.upper && r.Get(100) >= 90 {
			f.deceptive = 1 - f.deceptive
		}
	} else {
		f.stop--
	}
	out.Regen(Regen{Time: f.time, HP: f.hp, Mode: f.mode, Anim: f.anim, Deceptive: f.deceptive}, f.anim != 0)
}

// Pump is Fishing.usePumping with the skill's damage and penalty: the fish
// resists when Rnd.get(100) > 90; otherwise pumping a resting fish (or a
// fighting deceptive one) hurts it, and pumping a fighting fish (or a
// resting deceptive one) lets it regain the damage.
func (f *Fight) Pump(r Rand, dmg, pen int, out Output) {
	f.use(r, dmg, pen, out, 1, 0, MsgPumpingSuccess, MsgPumpingPenalty, MsgPumpingFailed)
}

// Reel is Fishing.useReeling: as Pump, with the modes the other way round.
func (f *Fight) Reel(r Rand, dmg, pen int, out Output) {
	f.use(r, dmg, pen, out, 2, 1, MsgReelingSuccess, MsgReelingPenalty, MsgReelingFailed)
}

func (f *Fight) use(r Rand, dmg, pen int, out Output, anim, goodMode int, success, penalty, failed Message) {
	if f.over {
		return
	}
	f.anim = anim
	if r.Get(100) > 90 {
		out.Message(MsgResisted, 0)
		f.goodUse = 0
		f.changeHP(0, pen, out)
		return
	}
	if (f.mode == goodMode) == (f.deceptive == 0) {
		out.Message(success, dmg)
		if pen > 0 {
			out.Message(penalty, pen)
		}
		f.goodUse = 1
		f.changeHP(dmg, pen, out)
		return
	}
	out.Message(failed, dmg)
	f.goodUse = 2
	f.changeHP(-dmg, pen, out)
}

// changeHP is Fishing.changeHp: the fish loses hp (never below 0), everyone
// around sees ExFishingHpRegen, the animation ends; above twice its HP the
// fish gets away, at 0 it is landed.
func (f *Fight) changeHP(hp, pen int, out Output) {
	f.hp = max(f.hp-hp, 0)
	out.Regen(Regen{Time: f.time, HP: f.hp, Mode: f.mode, GoodUse: f.goodUse, Anim: f.anim, Penalty: pen, Deceptive: f.deceptive}, true)
	f.anim = 0
	switch {
	case f.hp > f.maxHP*2:
		f.hp = f.maxHP * 2
		f.die(false, out)
	case f.hp == 0:
		f.die(true, out)
	}
}

// Abort ends the fight as lost (doDie(false)), when the player casts again
// or leaves.
func (f *Fight) Abort(out Output) {
	if !f.over {
		f.die(false, out)
	}
}

func (f *Fight) die(win bool, out Output) {
	f.over = true
	out.Over(win)
}
