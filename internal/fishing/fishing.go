// Package fishing holds the rules of fishing (Plan 14, M11g), ported from
// Mobius without the world: internal/game adapts them to the loop, the
// packets and the store.
//
//   - Casting: where the bait lands (PlaceBait, from the Fishing effect).
//   - Hooking: the fish a lure catches, by the player's fishing level and
//     the lure's grade and group tables (Tables.Hook, Player.startFishing),
//     and the bite checks (Bites, CheckDelay; LookingForFishTask).
//   - The fight: a one-second loop in which the fish rests or fights, and
//     pumping and reeling hurt it or let it regain HP (Fight,
//     mechanics/fishing/Fishing.java); the damage of the two skills (Damage).
//   - The weekly championship of the biggest fish (Championship,
//     FishingChampionshipManager).
//
// Every roll draws from the Rand it is given, with Mobius's expression and
// in Mobius's order (decision X7).
package fishing

import (
	"cmp"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// Rand is the random source of the rules: Mobius Rnd.get(n) and Rnd.get(lo,
// hi). rnd.Source is one.
type Rand interface {
	// Get is a number in [0, n).
	Get(n int) int
	// Between is a number in [lo, hi], both included.
	Between(lo, hi int) int
}

// Message is a system message of fishing (Mobius SystemMessageId).
type Message int32

// The system messages fishing sends.
const (
	MsgCannotDoThatWhileFishing2 Message = 1470 // "You cannot do that while fishing." (dropping, mounting)
	MsgCannotDoThatWhileFishing3 Message = 1471 // "You cannot do that while fishing." (items, actions)
	MsgOnlyFishingSkills         Message = 1448 // "Only fishing skills may be used at this time."
	MsgBite                      Message = 1449 // "You've got a bite!"
	MsgSpitTheHook               Message = 1450 // "That fish is more determined than you are - it spit the hook!"
	MsgBaitStolen                Message = 1451 // "Your bait was stolen by that fish!"
	MsgBaitsLost                 Message = 1452 // "Baits have been lost because the fish got away."
	MsgNoFishingPole             Message = 1453 // "You do not have a fishing pole equipped."
	MsgNoBait                    Message = 1454 // "You must put bait on your hook before you can fish."
	MsgUnderWater                Message = 1455 // "You cannot fish while under water."
	MsgOnBoat                    Message = 1456 // "You cannot fish while riding as a passenger of a boat - it's against the rules."
	MsgCannotFishHere            Message = 1457 // "You can't fish here."
	MsgCancelled                 Message = 1458 // "Your attempt at fishing has been cancelled."
	MsgNotEnoughBait             Message = 1459 // "You do not have enough bait."
	MsgStopFishing               Message = 1460 // "You reel your line in and stop fishing."
	MsgStartFishing              Message = 1461 // "You cast your line and start to fish."
	MsgPumpingOnlyWhileFishing   Message = 1462 // "You may only use the Pumping skill while you are fishing."
	MsgReelingOnlyWhileFishing   Message = 1463 // "You may only use the Reeling skill while you are fishing."
	MsgResisted                  Message = 1464 // "The fish has resisted your attempt to bring it in."
	MsgPumpingSuccess            Message = 1465 // "Your pumping is successful, causing $s1 damage."
	MsgPumpingFailed             Message = 1466 // "You failed to do anything with the fish and it regains $s1 HP."
	MsgReelingSuccess            Message = 1467 // "You reel that fish in closer and cause $s1 damage."
	MsgReelingFailed             Message = 1468 // "You failed to reel that fish in further and it regains $s1 HP."
	MsgCaughtSomething           Message = 1469 // "You caught something!"
	MsgWrongShotGrade            Message = 1479 // "That is the wrong grade of soulshot for that fishing pole."
	MsgInStoreOrCrafting         Message = 1638 // "You cannot fish while using a recipe book, private manufacture or private store."
	MsgSmellyCatch               Message = 1655 // "You caught something smelly and scary, maybe you should throw it back!?"
	MsgMasteryPenalty            Message = 1670 // "Due to your Reeling and/or Pumping skill being three or more levels higher than your Fishing skill, a 50 damage penalty will be applied."
	MsgReelingPenalty            Message = 1671 // "Your reeling was successful! (Mastery Penalty:$s1 )"
	MsgPumpingPenalty            Message = 1672 // "Your pumping was successful! (Mastery Penalty:$s1 )"
	MsgCaughtFishLength          Message = 1847 // "You caught a fish $s1 in length."
	MsgRegisteredInRanking       Message = 1848 // "Because of the size of fish caught, you will be registered in the ranking."
)

// Fishing skills and items Mobius names by ID.
const (
	SkillFishing          = 1312 // the cast
	SkillPumping          = 1313 // CommonSkill.PUMPING
	SkillReeling          = 1314 // CommonSkill.REELING
	SkillFishingExpertise = 1315 // the fishing level
	SkillFishermansPotion = 2274 // its power replaces the fishing level
	// FirstFishingShot and LastFishingShot are the fishing shots, the only
	// items a fishing player may use (Mobius UseItem).
	FirstFishingShot = 6535
	LastFishingShot  = 6540
	// MaxFishLevel is the highest fish level (getRandomFishLvl).
	MaxFishLevel = 27
)

// Tables are the fishing data with the fish of each grade in the order
// Mobius FishData iterates them.
type Tables struct {
	data    *data.Fishing
	byGrade [data.FishGradeHard + 1][]*data.Fish
}

// NewTables orders the fish of d as Mobius's three HashMaps of fish IDs
// (one per grade) hold them.
func NewTables(d *data.Fishing) *Tables {
	t := &Tables{data: d}
	for g := range t.byGrade {
		var fish []*data.Fish
		var ids []int
		for i := range d.Fishes {
			if f := &d.Fishes[i]; f.Grade == g {
				fish = append(fish, f)
				ids = append(ids, f.FishID)
			}
		}
		for _, i := range javaHashMapOrder(ids) {
			t.byGrade[g] = append(t.byGrade[g], fish[i])
		}
	}
	return t
}

// Data returns the fishing data.
func (t *Tables) Data() *data.Fishing { return t.data }

// Fishes is Mobius FishData.getFish: the fish of a grade with the level and
// group, in Mobius's order; nil for none (or an unknown grade).
func (t *Tables) Fishes(level, group, grade int) []*data.Fish {
	if grade < 0 || grade >= len(t.byGrade) {
		return nil
	}
	var out []*data.Fish
	for _, f := range t.byGrade[grade] {
		if f.Level == level && f.Group == group {
			out = append(out, f)
		}
	}
	return out
}

// Hook is the fish choice of Mobius Player.startFishing: the fish level
// (FishLevel), the lure's grade and group (LureGroup), then one of the
// matching fish (Rnd.get(size)). False when no fish matches ("Error - Fish
// are not defined").
func (t *Tables) Hook(r Rand, lure *data.Lure, expertise int) (*data.Fish, bool) {
	level := FishLevel(r, expertise)
	group := LureGroup(r, lure)
	list := t.Fishes(level, group, lure.Grade)
	if len(list) == 0 {
		return nil, false
	}
	return list[r.Get(len(list))], true
}

// FishLevel is Mobius getRandomFishLvl for a fishing level (Fishing
// Expertise's level, or the power of a Fisherman's Potion): 1 without one;
// otherwise check = Rnd.get(100) gives the level itself up to 50, one less
// (at least 1) up to 85, else one more (at most 27).
func FishLevel(r Rand, expertise int) int {
	if expertise <= 0 {
		return 1
	}
	switch check := r.Get(100); {
	case check <= 50:
		return expertise
	case check <= 85:
		return max(expertise-1, 1)
	default:
		return min(expertise+1, MaxFishLevel)
	}
}

// LureGroup is Mobius getRandomFishGroup: check = Rnd.get(100), drawn for
// every lure, picks the group from the lure's table.
func LureGroup(r Rand, lure *data.Lure) int {
	check := r.Get(100)
	for _, g := range lure.Groups {
		if check <= g.UpTo {
			return g.Group
		}
	}
	return data.FishGroupSwift // Mobius's default type; the data never gets here
}

// ShownGroup is the group ExFishingStart shows: -1 when the lure does not
// fit the time of day (a night lure by day, any other at night), and then
// the fish never bites.
func ShownGroup(f *data.Fish, lure *data.Lure, night bool) int {
	if night != lure.Night {
		return -1
	}
	return f.Group
}

// CheckDelay is the period of the bite checks in milliseconds
// (startLookingForFishTask): the fish's guts check time times the lure's
// factor.
func CheckDelay(f *data.Fish, lure *data.Lure) int { return f.GutsCheckTime * lure.CheckDelay }

// FirstCheck is when the first bite check comes, in milliseconds after the
// cast; WaitLimit is how long the bait waits for a bite before the fishing
// ends (LookingForFishTask: startCombatTime seconds after the first 10).
const FirstCheck = 10000

// WaitLimit returns, in milliseconds after the cast, when the bait stops
// waiting for a bite.
func WaitLimit(f *data.Fish) int { return f.StartCombatTime*1000 + FirstCheck }

// Bites is a bite check of LookingForFishTask: the fish bites when its guts
// are above Rnd.get(100).
func Bites(r Rand, f *data.Fish) bool { return f.Guts > float64(r.Get(100)) }

// SmellyCatch is the win of Mobius Fishing.doDie: with a fishing monster for
// the player's level, Rnd.get(100) <= its probability pulls up the monster
// instead of the fish.
func SmellyCatch(r Rand, m *data.FishingMonster) bool { return r.Get(100) <= m.Probability }

// Damage is the damage of Pumping and Reeling (Mobius Pumping.onStart):
// (rod damage + fishing expertise + the skill's power) × rod level × 0.1,
// doubled by a charged fishing shot, cut to an int. When the player's
// Fishing Expertise is more than two levels below the skill's, 5 % of it is
// the penalty and comes off (penalized reports it).
func Damage(rod *data.FishingRod, expertise, power float64, shot bool, fishingLevel, skillLevel int) (dmg, pen int, penalized bool) {
	ss := 1
	if shot {
		ss = 2
	}
	gradeBonus := float64(rod.Level) * 0.1
	dmg = javaInt((rod.Damage + expertise + power) * gradeBonus * float64(ss))
	if fishingLevel < skillLevel-2 {
		pen = javaInt(float64(dmg) * 0.05)
		dmg -= pen
		penalized = true
	}
	return dmg, pen, penalized
}

// degreesToRadians is Java's Math.DEGREES_TO_RADIANS (Math.toRadians).
const degreesToRadians = 0.017453292519943295

// HeadingDegrees is Mobius LocationUtil.convertHeadingToDegree.
func HeadingDegrees(heading int) float64 { return float64(heading) / 182.044444444 }

// The bait distances of the Fishing effect.
const (
	MinBaitDistance = 90
	MaxBaitDistance = 250
)

// Spot answers where the bait may land.
type Spot interface {
	// WaterZ returns the Z of the bait at (x, y): the top of a water zone
	// where a fishing zone and a water zone both hold (x, y) at their tops
	// (ZoneManager.getZones(x, y)); false where they do not meet.
	WaterZ(x, y int) (z int, ok bool)
	// Clear reports whether the fisher sees the bait at (x, y, z) and no
	// ground rises above it (computeBaitZ's geodata checks).
	Clear(x, y, z int) bool
}

// PlaceBait is where the Fishing effect throws the bait from (x, y) facing
// heading: at Rnd.get(90, 250) units ahead, else at the first distance from
// 250 down to 90 where Spot allows it. False means "You can't fish here.".
func PlaceBait(r Rand, x, y, heading int, spot Spot) (bx, by, bz int, ok bool) {
	distance := r.Between(MinBaitDistance, MaxBaitDistance)
	radian := HeadingDegrees(heading) * degreesToRadians // Math.toRadians
	sin, cos := math.Sin(radian), math.Cos(radian)
	at := func(d int) (int, int, int, bool) {
		bx := int(float64(x) + cos*float64(d))
		by := int(float64(y) + sin*float64(d))
		z, ok := spot.WaterZ(bx, by)
		if !ok || !spot.Clear(bx, by, z) {
			return bx, by, 0, false
		}
		return bx, by, z, true
	}
	if bx, by, bz, ok = at(distance); ok {
		return bx, by, bz, true
	}
	for d := MaxBaitDistance; d >= MinBaitDistance; d-- {
		if bx, by, bz, ok = at(d); ok {
			return bx, by, bz, true
		}
	}
	return 0, 0, 0, false
}

// javaInt is Java's (int) cast of a double.
func javaInt(x float64) int {
	switch {
	case x != x:
		return 0
	case x >= math.MaxInt32:
		return math.MaxInt32
	case x <= math.MinInt32:
		return math.MinInt32
	}
	return int(x)
}

// javaHashMapOrder returns the indexes of keys (small Integer keys, put in
// order) in the iteration order of a java.util.HashMap with default
// settings: by bucket, then by insertion within a bucket. The table starts
// with 16 buckets and doubles when it holds more than 3/4 of its size, or
// when a bucket reaches 9 entries while the table is under 64 buckets.
func javaHashMapOrder(keys []int) []int {
	hashes := make([]uint32, len(keys))
	for i, k := range keys {
		h := uint32(int32(k)) //nolint:gosec // Java int bits
		hashes[i] = h ^ h>>16
	}
	capacity := uint32(16)
	bucket := func(i int) uint32 { return hashes[i] & (capacity - 1) }
	for i := range keys {
		n := 0
		for j := range i + 1 {
			if bucket(j) == bucket(i) {
				n++
			}
		}
		if n >= 9 && capacity < 64 {
			capacity *= 2
		}
		if uint32(i+1) > capacity*3/4 { //nolint:gosec // i is small
			capacity *= 2
		}
	}
	order := make([]int, len(keys))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return cmp.Compare(bucket(a), bucket(b)) })
	return order
}

// JavaDouble formats x as Java's String.valueOf(double): the shortest
// decimal that reads back as x, with ".0" for whole numbers, and the
// computerized scientific form ("1.0E7") outside 10^-3 to 10^7.
func JavaDouble(x float64) string {
	switch {
	case math.IsNaN(x):
		return "NaN"
	case math.IsInf(x, 1):
		return "Infinity"
	case math.IsInf(x, -1):
		return "-Infinity"
	case x == 0:
		if math.Signbit(x) {
			return "-0.0"
		}
		return "0.0"
	}
	if a := math.Abs(x); a >= 1e-3 && a < 1e7 {
		s := strconv.FormatFloat(x, 'f', -1, 64)
		if !strings.Contains(s, ".") {
			s += ".0"
		}
		return s
	}
	s := strconv.FormatFloat(x, 'e', -1, 64) // like 1.2345e+07
	mant, exp, _ := strings.Cut(s, "e")
	if !strings.Contains(mant, ".") {
		mant += ".0"
	}
	e, _ := strconv.Atoi(exp)
	return mant + "E" + strconv.Itoa(e)
}
