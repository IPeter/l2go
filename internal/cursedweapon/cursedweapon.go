// Package cursedweapon holds the rules of the cursed weapons (Plan 14 M11m):
// Demonic Sword Zariche and Blood Sword Akamanah, Mobius CursedWeapon and
// CursedWeaponsManager without the world. A weapon drops from a monster
// once in 100,000 kills, turns whoever picks it up into a red-named killer
// with karma 9,999,999, grows a skill level every few kills, loses a few
// minutes of its life with each kill, and ends after its life or vanishes
// when its holder dies.
//
// The package keeps the state and makes the rolls; internal/game moves the
// items, sends the packets and saves the state. Rolls take the random
// source with Mobius's exact expressions (decision X7). Times are Unix
// milliseconds, as Mobius stores them.
package cursedweapon

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
)

// Mobius's constants of the cursed weapons.
const (
	// HolderKarma is the karma of a holder (CursedWeapon.activate).
	HolderKarma = 9999999
	// SkillVoidBurst and SkillVoidFlow are the two skills every holder gets
	// besides the weapon's own (CommonSkill.VOID_BURST, VOID_FLOW, level 1).
	SkillVoidBurst = 3630
	SkillVoidFlow  = 3631
	// DropBound is the bound of the drop roll: Rnd.get(100000) < dropRate.
	DropBound = 100000
	// DisappearBound is the bound of the death roll: Rnd.get(100) <=
	// disapearChance vanishes the weapon.
	DisappearBound = 100
	// checkPeriodPerMinute is the check period per minute of durationLost
	// (RemoveTask runs every durationLost × 12,000 ms).
	checkPeriodPerMinute = 12 * time.Second
	// SocialActionID is the social action a new holder plays 300 ms after
	// the pickup (SocialAction 17).
	SocialActionID = 17
	// SocialActionDelay is that delay.
	SocialActionDelay = 300 * time.Millisecond
	// RedSkySeconds and the earthquake numbers are what everyone online gets
	// when a monster drops a weapon (ExRedSky(10), Earthquake(x, y, z, 14,
	// 3)).
	RedSkySeconds       = 10
	EarthquakeIntensity = 14
	EarthquakeDuration  = 3
)

// ExcludedNpcTypes are the NPC types (and their subtypes) that never drop a
// cursed weapon (CursedWeaponsManager.checkDrop).
var ExcludedNpcTypes = []string{"Defender", "RiftInvader", "FestivalMonster", "Guard", "GrandBoss", "FeedableBeast"}

// Excluded reports whether an NPC never drops a cursed weapon; isType
// answers whether the NPC is of a type or one of its subtypes (Java's
// instanceof).
func Excluded(isType func(typ string) bool) bool {
	return slices.ContainsFunc(ExcludedNpcTypes, isType)
}

// Weapon is one cursed weapon: its data and its state (Mobius CursedWeapon).
// The zero state is a weapon that does not exist in the world.
type Weapon struct {
	Def data.CursedWeapon
	// SkillMaxLevel is the highest level of the weapon's skill.
	SkillMaxLevel int

	// Activated is set while a player holds the weapon.
	Activated bool
	// Dropped is set once the weapon dropped (from a monster or a dying
	// holder). Mobius keeps it after a pickup too, until the weapon ends.
	Dropped bool
	// PlayerID is the holder (or the last holder of a dropped weapon), 0 for
	// none; PlayerKarma and PlayerPKKills are the holder's karma and PK count
	// from before, given back when the weapon goes.
	PlayerID      int32
	PlayerKarma   int32
	PlayerPKKills int32
	// NbKills is the holder's kills with the weapon.
	NbKills int32
	// EndTime is when the weapon ends, in Unix milliseconds; 0 while it is
	// not in the world.
	EndTime int64
}

// ItemID returns the weapon's item ID.
func (w *Weapon) ItemID() int32 { return int32(w.Def.ItemID) } //nolint:gosec // item IDs are small

// SkillID returns the weapon's skill ID.
func (w *Weapon) SkillID() int { return w.Def.SkillID }

// Active reports whether the weapon is in the world: held or dropped
// (Mobius isActive).
func (w *Weapon) Active() bool { return w.Activated || w.Dropped }

// SkillLevel is the level of the weapon's skill its holder gets
// (giveSkill): 1 + kills / stageKills, at most the skill's highest level.
func (w *Weapon) SkillLevel() int {
	level := 1 + int(w.NbKills)/w.Def.StageKills
	if w.SkillMaxLevel > 0 && level > w.SkillMaxLevel {
		level = w.SkillMaxLevel
	}
	return level
}

// Level is the weapon level UserInfo and CharInfo show (getLevel): kills /
// stageKills, or the skill's highest level once the kills pass
// stageKills × that level.
func (w *Weapon) Level() int32 {
	if int(w.NbKills) > w.Def.StageKills*w.SkillMaxLevel {
		return int32(w.SkillMaxLevel) //nolint:gosec // skill levels are small
	}
	return w.NbKills / int32(w.Def.StageKills) //nolint:gosec // stage kills are small
}

// CheckPeriod is how often the weapon's life is checked (RemoveTask, at a
// fixed rate of durationLost × 12,000 ms).
func (w *Weapon) CheckPeriod() time.Duration {
	return time.Duration(w.Def.DurationLost) * checkPeriodPerMinute
}

// Life is the weapon's life from its drop (duration minutes).
func (w *Weapon) Life() time.Duration {
	return time.Duration(w.Def.Duration) * time.Minute
}

// KillCost is what each kill takes off the life (durationLost minutes).
func (w *Weapon) KillCost() time.Duration {
	return time.Duration(w.Def.DurationLost) * time.Minute
}

// TimeLeft is the life left at now, in milliseconds (getTimeLeft).
func (w *Weapon) TimeLeft(now time.Time) int64 { return w.EndTime - now.UnixMilli() }

// MinutesLeft is the life left in whole minutes, as Mobius shows it (Java
// long division, then an int cast).
func (w *Weapon) MinutesLeft(now time.Time) int32 {
	return int32(w.TimeLeft(now) / 60000) //nolint:gosec // lives are minutes
}

// Expired reports whether the life is over at now (RemoveTask:
// currentTimeMillis >= endTime).
func (w *Weapon) Expired(now time.Time) bool { return now.UnixMilli() >= w.EndTime }

// RollDrop is the drop roll of one kill (CursedWeapon.checkDrop):
// Rnd.get(100000) < dropRate.
func (w *Weapon) RollDrop(r rnd.Source) bool { return r.Get(DropBound) < w.Def.DropRate }

// RollDisappear is the roll when the holder dies (CursedWeapon.dropIt):
// Rnd.get(100) <= disapearChance vanishes the weapon (51 % with Mobius's
// 50, a quirk Plan 14 keeps).
func (w *Weapon) RollDisappear(r rnd.Source) bool {
	return r.Get(DisappearBound) <= w.Def.DisappearChance
}

// DroppedByMonster is the state after a monster dropped the weapon
// (checkDrop): on the ground, its life starting at now.
func (w *Weapon) DroppedByMonster(now time.Time) {
	w.Activated = false
	w.Dropped = true
	w.EndTime = now.Add(w.Life()).UnixMilli()
}

// StartLife sets the life to start at now, for a weapon a GM hands out
// (//cw_add: setEndTime(now + duration)).
func (w *Weapon) StartLife(now time.Time) { w.EndTime = now.Add(w.Life()).UnixMilli() }

// Activate is the state after a player picked the weapon up (activate): the
// holder's karma and PK count are kept to give back later.
func (w *Weapon) Activate(playerID, karma, pkKills int32) {
	w.Activated = true
	w.PlayerID = playerID
	w.PlayerKarma = karma
	w.PlayerPKKills = pkKills
}

// DroppedByHolder is the state after the dying holder dropped the weapon
// (dropIt from the player).
func (w *Weapon) DroppedByHolder() {
	w.Activated = false
	w.Dropped = true
}

// AddKill counts a kill of the holder (increaseKills): one more kill and
// durationLost minutes less life. levelUp reports whether an online holder
// gets the next skill level (kills a multiple of stageKills, up to the
// skill's highest level).
func (w *Weapon) AddKill() (levelUp bool) {
	w.NbKills++
	w.EndTime -= w.KillCost().Milliseconds()
	kills := int(w.NbKills)
	last := w.SkillMaxLevel - 1
	if last < 0 {
		last = 0
	}
	return kills%w.Def.StageKills == 0 && kills <= w.Def.StageKills*last
}

// AddBonusStage is what a holder that picks up the other weapon gets
// (CursedWeaponsManager.activate, a "TEMP fix" in Mobius): the kills are set
// to one under a stage and a kill is counted, which also takes durationLost
// minutes off the life.
func (w *Weapon) AddBonusStage() (levelUp bool) {
	w.NbKills = int32(w.Def.StageKills - 1) //nolint:gosec // stage kills are small
	return w.AddKill()
}

// Reset is the state after the weapon ended (endOfLife): it no longer
// exists in the world.
func (w *Weapon) Reset() {
	w.Activated = false
	w.Dropped = false
	w.EndTime = 0
	w.PlayerID = 0
	w.PlayerKarma = 0
	w.PlayerPKKills = 0
	w.NbKills = 0
}

// Set is the cursed weapons of a server (Mobius CursedWeaponsManager's
// map), in the iteration order of Mobius's HashMap: drops are rolled and
// lists are sent in that order.
type Set struct {
	weapons []*Weapon
}

// NewSet makes the weapons of the data with no state. maxLevel returns the
// highest level of a skill (SkillData.getMaxLevel), 0 when it is unknown.
func NewSet(defs []data.CursedWeapon, maxLevel func(skillID int) int) *Set {
	ids := make([]int32, len(defs))
	for i, d := range defs {
		ids[i] = int32(d.ItemID) //nolint:gosec // item IDs are small
	}
	s := &Set{}
	for _, i := range HashMapOrder(ids) {
		w := &Weapon{Def: defs[i]}
		if maxLevel != nil {
			w.SkillMaxLevel = maxLevel(defs[i].SkillID)
		}
		s.weapons = append(s.weapons, w)
	}
	return s
}

// Weapons returns the weapons in iteration order. The slice must not be
// modified.
func (s *Set) Weapons() []*Weapon {
	if s == nil {
		return nil
	}
	return s.weapons
}

// Get returns the weapon of an item, nil if the item is not a cursed weapon.
func (s *Set) Get(itemID int32) *Weapon {
	for _, w := range s.Weapons() {
		if w.ItemID() == itemID {
			return w
		}
	}
	return nil
}

// IsCursed reports whether an item is a cursed weapon (isCursed).
func (s *Set) IsCursed(itemID int32) bool { return s.Get(itemID) != nil }

// HeldBy returns the weapon a player holds, nil for none
// (checkOwnsWeaponId).
func (s *Set) HeldBy(playerID int32) *Weapon {
	if playerID == 0 {
		return nil
	}
	for _, w := range s.Weapons() {
		if w.Activated && w.PlayerID == playerID {
			return w
		}
	}
	return nil
}

// CheckDrop rolls the drop of a kill (CursedWeaponsManager.checkDrop):
// nothing from an excluded NPC; otherwise each weapon not in the world
// rolls in turn until one drops. It returns that weapon, nil for none; the
// caller drops the item and then calls DroppedByMonster.
func (s *Set) CheckDrop(r rnd.Source, excluded bool) *Weapon {
	if excluded {
		return nil
	}
	for _, w := range s.Weapons() {
		if w.Active() {
			continue
		}
		if w.RollDrop(r) {
			return w
		}
	}
	return nil
}

// Find returns the weapon a GM names (AdminCursedWeapons): an item ID, or
// a part of the name in any case with underscores for spaces. ok is false
// for a parameter that names none.
func (s *Set) Find(param string) (w *Weapon, ok bool) {
	if id, err := strconv.ParseInt(param, 10, 32); err == nil {
		w = s.Get(int32(id))
		return w, w != nil
	}
	want := strings.ToLower(strings.ReplaceAll(param, "_", " "))
	for _, w := range s.Weapons() {
		if strings.Contains(strings.ToLower(w.Def.Name), want) {
			return w, true
		}
	}
	return nil, false
}

// HashMapOrder returns the indexes of keys in the iteration order of a Java
// HashMap<Integer, V> filled with them in order: by bucket of the final
// table (the default 16, doubled while the size passes three quarters of
// it), then in insertion order within a bucket.
func HashMapOrder(keys []int32) []int {
	capacity := 16
	for len(keys) > capacity*3/4 {
		capacity *= 2
	}
	mask := uint32(capacity) //nolint:gosec // a power of two
	mask--
	bucket := func(k int32) int {
		h := uint32(k) //nolint:gosec // Integer.hashCode is the value's bits
		spread := h >> 16
		spread ^= h
		return int(spread & mask)
	}
	order := make([]int, len(keys))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return cmp.Compare(bucket(keys[a]), bucket(keys[b])) })
	return order
}
