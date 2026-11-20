package skilltest

import (
	"slices"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// Reactions is the part of a fake Creature that the hate, trigger,
// appearance and utility effects use: what they read and what they did.
type Reactions struct {
	// NoMove makes IsMovementDisabled true (so does Immobilized).
	NoMove bool
	// Locked is the locked target (SetLockedTarget).
	Locked skills.Creature
	// Teleports and Placed record TeleToLocation and SetXYZ; Validates
	// counts BroadcastValidateLocation.
	Teleports []skills.Location
	Placed    []skills.Location
	Validates int
	// Rotations records BroadcastStartRotation as {heading, side, speed}
	// and BroadcastStopRotation as {heading, -1, speed}.
	Rotations [][3]int
	// FakeDeathCalls records "start" (StartFakeDeath), "stop" and
	// "stop+effects" (StopFakeDeath) and "end" (BroadcastFakeDeathEnd).
	FakeDeathCalls []string
	// Grown is the collision state SetCollisionGrown set.
	Grown bool
	// TriggerCasts records MakeTriggerCast.
	TriggerCasts []TriggerCast
	listeners    []damageListener

	// Player state.
	HairStyle, HairColor, Face int
	UserInfos                  int
	ExpSp                      [][2]float64
	PenaltyDrops               int
	Seeds                      []data.SeedElement
	FakeDead, RecentFakeDeath  bool
	SitDowns                   int
	// InventoryFull makes CheckInventorySlotsAndWeight false; InventoryChecks
	// records the item IDs it was asked about.
	InventoryFull   bool
	InventoryChecks [][]int
	PartyItems      []skills.ItemHolder

	// Attackable state.
	HateList []HateEntry
	FearTime int
	// Clans are the template's clans; ClanRange its clan help range.
	Clans     []string
	ClanRange int
	Spoiler   int32
	// NotSpoilOwner makes CheckSpoilOwner false; SpoilLoot is the loot
	// TakeSweep takes.
	NotSpoilOwner bool
	SpoilLoot     []skills.ItemHolder
	Distrusted    skills.Creature
	DistrustStops int
}

// TriggerCast is one recorded MakeTriggerCast.
type TriggerCast struct {
	Skill            *skills.Skill
	Target           skills.Creature
	IgnoreTargetType bool
}

// HateEntry is a creature on a fake attackable's hate list.
type HateEntry struct {
	Creature skills.Creature
	Damage   int64
	Hate     int64
}

type damageListener struct {
	owner any
	fn    func(skills.DamageEvent)
}

func (c *Creature) notifyDamage(ev skills.DamageEvent) {
	for _, l := range slices.Clone(c.listeners) {
		l.fn(ev)
	}
}

// Listeners returns the number of damage listeners.
func (c *Creature) Listeners() int { return len(c.listeners) }

// SetRunning makes the creature run.
func (c *Creature) SetRunning() { c.Running = true }

// SetWalking makes the creature walk.
func (c *Creature) SetWalking() { c.Running = false }

// IsMovementDisabled returns NoMove or Immobilized.
func (c *Creature) IsMovementDisabled() bool { return c.NoMove || c.Immobilized }

// SetLockedTarget sets Locked.
func (c *Creature) SetLockedTarget(t skills.Creature) { c.Locked = t }

// AsAttackable returns the creature for an attackable, else nil.
func (c *Creature) AsAttackable() skills.Attackable {
	if !c.Type.Is(data.InstanceAttackable) {
		return nil
	}
	return c
}

// TeleToLocation records the location and moves there.
func (c *Creature) TeleToLocation(loc skills.Location, _ bool) {
	c.Teleports = append(c.Teleports, loc)
	c.Loc = loc
}

// SetXYZ records the location and moves there.
func (c *Creature) SetXYZ(loc skills.Location) {
	c.Placed = append(c.Placed, loc)
	c.Loc = loc
}

// BroadcastValidateLocation counts the call.
func (c *Creature) BroadcastValidateLocation() { c.Validates++ }

// BroadcastStartRotation records the rotation.
func (c *Creature) BroadcastStartRotation(heading, side, speed int) {
	c.Rotations = append(c.Rotations, [3]int{heading, side, speed})
}

// BroadcastStopRotation records the rotation with side -1.
func (c *Creature) BroadcastStopRotation(heading, speed int) {
	c.Rotations = append(c.Rotations, [3]int{heading, -1, speed})
}

// StartFakeDeath records "start" and fakes death.
func (c *Creature) StartFakeDeath() {
	c.FakeDeathCalls = append(c.FakeDeathCalls, "start")
	c.FakeDead, c.AlikeDead = true, true
}

// StopFakeDeath records "stop" or "stop+effects" and ends fake death.
func (c *Creature) StopFakeDeath(removeEffects bool) {
	call := "stop"
	if removeEffects {
		call = "stop+effects"
	}
	c.FakeDeathCalls = append(c.FakeDeathCalls, call)
	c.FakeDead, c.AlikeDead = false, false
}

// BroadcastFakeDeathEnd records "end".
func (c *Creature) BroadcastFakeDeathEnd() { c.FakeDeathCalls = append(c.FakeDeathCalls, "end") }

// SetCollisionGrown sets Grown.
func (c *Creature) SetCollisionGrown(grown bool) { c.Grown = grown }

// MakeTriggerCast records the cast.
func (c *Creature) MakeTriggerCast(sk *skills.Skill, target skills.Creature, ignoreTargetType bool) {
	c.TriggerCasts = append(c.TriggerCasts, TriggerCast{Skill: sk, Target: target, IgnoreTargetType: ignoreTargetType})
}

// AddDamageListener adds a damage listener.
func (c *Creature) AddDamageListener(owner any, fn func(skills.DamageEvent)) {
	c.listeners = append(c.listeners, damageListener{owner: owner, fn: fn})
}

// RemoveDamageListeners removes the listeners of owner.
func (c *Creature) RemoveDamageListeners(owner any) {
	c.listeners = slices.DeleteFunc(c.listeners, func(l damageListener) bool { return l.owner == owner })
}

// SetHairStyle sets HairStyle.
func (c *Creature) SetHairStyle(v int) { c.HairStyle = v }

// SetHairColor sets HairColor.
func (c *Creature) SetHairColor(v int) { c.HairColor = v }

// SetFace sets Face.
func (c *Creature) SetFace(v int) { c.Face = v }

// BroadcastUserInfo counts the call.
func (c *Creature) BroadcastUserInfo() { c.UserInfos++ }

// AddExpAndSp records the experience and SP.
func (c *Creature) AddExpAndSp(exp, sp float64) { c.ExpSp = append(c.ExpSp, [2]float64{exp, sp}) }

// ReduceDeathPenaltyBuffLevel counts the call.
func (c *Creature) ReduceDeathPenaltyBuffLevel() { c.PenaltyDrops++ }

// AddElementSeed records the seed.
func (c *Creature) AddElementSeed(e data.SeedElement) { c.Seeds = append(c.Seeds, e) }

// IsFakeDeath returns FakeDead.
func (c *Creature) IsFakeDeath() bool { return c.FakeDead }

// SetFakeDeath sets FakeDead.
func (c *Creature) SetFakeDeath(v bool) { c.FakeDead = v }

// SetRecentFakeDeath sets RecentFakeDeath.
func (c *Creature) SetRecentFakeDeath(v bool) { c.RecentFakeDeath = v }

// SitDown counts the call and sits down.
func (c *Creature) SitDown(bool) {
	c.SitDowns++
	c.Sitting = true
}

// CheckInventorySlotsAndWeight records the item IDs; false with
// InventoryFull.
func (c *Creature) CheckInventorySlotsAndWeight(itemIDs []int, _, _ bool) bool {
	c.InventoryChecks = append(c.InventoryChecks, slices.Clone(itemIDs))
	return !c.InventoryFull
}

// PartyDistributeItem records the item.
func (c *Creature) PartyDistributeItem(item skills.ItemHolder, _ bool, _ skills.Creature) {
	c.PartyItems = append(c.PartyItems, item)
}

func (c *Creature) hateIndex(t skills.Creature) int {
	return slices.IndexFunc(c.HateList, func(e HateEntry) bool { return e.Creature.ObjectID() == t.ObjectID() })
}

// AddDamageHate adds to the hate list.
func (c *Creature) AddDamageHate(attacker skills.Creature, damage, aggro int64) {
	if i := c.hateIndex(attacker); i >= 0 {
		c.HateList[i].Damage += damage
		c.HateList[i].Hate += aggro
		return
	}
	c.HateList = append(c.HateList, HateEntry{Creature: attacker, Damage: damage, Hate: aggro})
}

// ReduceHate lowers the hate for target, not below 0.
func (c *Creature) ReduceHate(target skills.Creature, amount int64) {
	if i := c.hateIndex(target); i >= 0 {
		c.HateList[i].Hate = max(c.HateList[i].Hate-amount, 0)
	}
}

// StopHating sets the hate for target to 0.
func (c *Creature) StopHating(target skills.Creature) {
	if i := c.hateIndex(target); i >= 0 {
		c.HateList[i].Hate = 0
	}
}

// ClearAggroList empties the hate list.
func (c *Creature) ClearAggroList() { c.HateList = nil }

// Hating returns the hate for target.
func (c *Creature) Hating(target skills.Creature) int64 {
	if i := c.hateIndex(target); i >= 0 {
		return c.HateList[i].Hate
	}
	return 0
}

// AggroList returns the creatures on the hate list.
func (c *Creature) AggroList() []skills.Creature {
	out := make([]skills.Creature, len(c.HateList))
	for i, e := range c.HateList {
		out[i] = e.Creature
	}
	return out
}

// SetFearTime sets FearTime.
func (c *Creature) SetFearTime(t int) { c.FearTime = t }

// ClanHelpRange returns ClanRange, false without clans.
func (c *Creature) ClanHelpRange() (int, bool) { return c.ClanRange, len(c.Clans) > 0 }

// IsClan reports whether the creature shares a clan with of.
func (c *Creature) IsClan(of skills.Attackable) bool {
	o, ok := of.(*Creature)
	if !ok {
		return false
	}
	return slices.ContainsFunc(c.Clans, func(cl string) bool { return slices.Contains(o.Clans, cl) })
}

// IsSpoiled reports a spoiler.
func (c *Creature) IsSpoiled() bool { return c.Spoiler != 0 }

// SetSpoiler sets Spoiler.
func (c *Creature) SetSpoiler(id int32) { c.Spoiler = id }

// CheckSpoilOwner is false with NotSpoilOwner.
func (c *Creature) CheckSpoilOwner(skills.Creature, bool) bool { return !c.NotSpoilOwner }

// SpoilLootItems returns the item IDs of SpoilLoot.
func (c *Creature) SpoilLootItems() []int {
	ids := make([]int, len(c.SpoilLoot))
	for i, it := range c.SpoilLoot {
		ids[i] = it.ID
	}
	return ids
}

// TakeSweep takes SpoilLoot.
func (c *Creature) TakeSweep() []skills.ItemHolder {
	loot := c.SpoilLoot
	c.SpoilLoot = nil
	return loot
}

// StartDistrust sets Distrusted.
func (c *Creature) StartDistrust(target skills.Creature) { c.Distrusted = target }

// StopDistrust clears Distrusted and counts the call.
func (c *Creature) StopDistrust() {
	c.Distrusted = nil
	c.DistrustStops++
}

var _ skills.Attackable = (*Creature)(nil)
