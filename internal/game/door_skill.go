package game

import (
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/stats"
)

// skillDoor is a door as the skill engine sees it (skills.Creature with the
// Door instance type): the target of damage skills during a siege. A door
// never moves, attacks or casts (Door overrides moveToLocation, doAttack and
// doCast with nothing), takes no continuous effects (calcEffectSuccess), no
// damage over time and no heals (the heal effects skip doors), so most
// methods do nothing. Each door has one, made on first use
// (Door.skillCreature).
type skillDoor struct {
	d      *Door
	el     *skills.EffectList
	traits skills.Traits
}

var _ skills.Creature = (*skillDoor)(nil)

// skillCreature returns the door's adapter.
func (d *Door) skillCreature() *skillDoor {
	if d.combat.skill == nil {
		a := &skillDoor{d: d}
		var env *skills.Env
		if d.srv != nil {
			env = d.srv.skills.env
		}
		a.el = skills.NewEffectList(env, a)
		d.combat.skill = a
	}
	return d.combat.skill
}

// ObjectID returns the door's object ID.
func (a *skillDoor) ObjectID() int32 { return int32(a.d.ID()) } //nolint:gosec // object IDs fit

// Name returns the door's name.
func (a *skillDoor) Name() string { return a.d.Name() }

// InstanceType returns Door.
func (a *skillDoor) InstanceType() data.InstanceType { return data.InstanceDoor }

// NpcID returns the door ID.
func (a *skillDoor) NpcID() int { return a.d.Template.ID }

// Race is CreatureTemplate's default race.
func (a *skillDoor) Race() data.Race { return data.RaceNone }

// Level returns the door's level.
func (a *skillDoor) Level() int { return a.d.Level() }

// IsRaid is false.
func (a *skillDoor) IsRaid() bool { return false }

// InCategory is false: doors are in no category.
func (a *skillDoor) InCategory(data.CategoryType) bool { return false }

// Location returns the door's position.
func (a *skillDoor) Location() skills.Location {
	pos := a.d.Position()
	return skills.Location{X: int(pos.X), Y: int(pos.Y), Z: int(pos.Z)}
}

// Heading returns the door's heading.
func (a *skillDoor) Heading() int { return int(a.d.Position().Heading) }

// SetHeading does nothing: doors do not turn.
func (a *skillDoor) SetHeading(int) {}

// CollisionRadius returns the door's collision radius.
func (a *skillDoor) CollisionRadius() int { return a.d.collisionRadius() }

// InsideZone reports the zone flags a door carries: an active siege zone
// gives PvP and siege (SiegeZone.onEnter), an arena PvP, a peace zone peace.
func (a *skillDoor) InsideZone(z skills.ZoneKind) bool {
	s := a.d.srv
	if s == nil {
		return false
	}
	pos := a.d.Position()
	switch z {
	case skills.ZoneSiege:
		return s.siegeAt(pos) != nil
	case skills.ZonePvP:
		return s.siegeAt(pos) != nil || s.zoneAt(pos, "ArenaZone") != nil
	case skills.ZonePeace:
		return s.zoneAt(pos, "PeaceZone") != nil
	}
	return false
}

// InZone is false.
func (a *skillDoor) InZone(int) bool { return false }

// Target is nil: doors target nothing.
func (a *skillDoor) Target() skills.Creature { return nil }

// IsDead reports a broken door.
func (a *skillDoor) IsDead() bool { return a.d.combat.dead }

// IsAlikeDead reports a broken door.
func (a *skillDoor) IsAlikeDead() bool { return a.d.combat.dead }

// IsInvul is false (the Door constructor's setInvul(false)).
func (a *skillDoor) IsInvul() bool { return false }

// IsMoving is false.
func (a *skillDoor) IsMoving() bool { return false }

// IsRunning is false.
func (a *skillDoor) IsRunning() bool { return false }

// IsFlying is false.
func (a *skillDoor) IsFlying() bool { return false }

// CurrentHP returns the door's HP.
func (a *skillDoor) CurrentHP() float64 { return a.d.curHP }

// SetCurrentHP sets the HP through the door's status, with its packets.
func (a *skillDoor) SetCurrentHP(hp float64) {
	if s := a.d.srv; s != nil {
		s.setDoorHP(a.d, hp, true)
	}
}

// CurrentMP is 0.
func (a *skillDoor) CurrentMP() float64 { return 0 }

// SetCurrentMP does nothing.
func (a *skillDoor) SetCurrentMP(float64) {}

// CurrentCP is 0.
func (a *skillDoor) CurrentCP() float64 { return 0 }

// SetCurrentCP does nothing.
func (a *skillDoor) SetCurrentCP(float64) {}

// MaxHP returns the door's maximum HP.
func (a *skillDoor) MaxHP() int { return int(a.d.maxHP) }

// MaxMP is 0.
func (a *skillDoor) MaxMP() int { return 0 }

// MaxCP is 0.
func (a *skillDoor) MaxCP() int { return 0 }

// MaxRecoverableHP is the maximum HP: doors have no modifiers.
func (a *skillDoor) MaxRecoverableHP() int { return int(a.d.maxHP) }

// MaxRecoverableMP is 0.
func (a *skillDoor) MaxRecoverableMP() int { return 0 }

// MaxRecoverableCP is 0.
func (a *skillDoor) MaxRecoverableCP() int { return 0 }

// Calc runs the door calculators (doorStats).
func (a *skillDoor) Calc(st stats.Stat, init float64, target skills.Creature, sk *skills.Skill) float64 {
	return a.d.combatStats().CalcFor(st, init, skillContext(target, sk))
}

// AddModifiers adds nothing: doors carry no stat modifiers.
func (a *skillDoor) AddModifiers(...stats.Modifier) error { return nil }

// RemoveOwner removes nothing.
func (a *skillDoor) RemoveOwner(stats.Owner) int { return 0 }

// Traits returns the trait arrays, all at 1.
func (a *skillDoor) Traits() *skills.Traits { return &a.traits }

// BaseStat is 0: a door's STR, DEX, ... are 0.
func (a *skillDoor) BaseStat(data.BaseStat) int { return 0 }

// StatBonus is the statBonus.xml multiplier at the door's base stats (0).
func (a *skillDoor) StatBonus(bs data.BaseStat) float64 {
	st, ok := baseStatOf(bs)
	if !ok {
		return 1
	}
	return a.d.combatStats().StatBonus(st)
}

// PAtk is 0: doors never attack.
func (a *skillDoor) PAtk(skills.Creature) float64 { return 0 }

// PDef is getPDef(target) of the door calculators.
func (a *skillDoor) PDef(target skills.Creature) float64 {
	return a.d.combatStats().PDefFor(skillContext(target, nil))
}

// MAtk is 0: doors never cast.
func (a *skillDoor) MAtk(skills.Creature, *skills.Skill) float64 { return 0 }

// MDef is getMDef(target, skill) of the door calculators.
func (a *skillDoor) MDef(target skills.Creature, sk *skills.Skill) float64 {
	return a.d.combatStats().MDefFor(skillContext(target, sk))
}

// PAtkSpd is CreatureTemplate's default attack speed.
func (a *skillDoor) PAtkSpd() float64 { return 300 }

// MAtkSpd is CreatureTemplate's default casting speed.
func (a *skillDoor) MAtkSpd() int { return 333 }

// MCriticalHit is 0.
func (a *skillDoor) MCriticalHit(skills.Creature, *skills.Skill) int { return 0 }

// ShieldDef is 0.
func (a *skillDoor) ShieldDef() int { return 0 }

// RandomDamageMultiplier is 1: doors never attack.
func (a *skillDoor) RandomDamageMultiplier(stats.Rand) float64 { return 1 }

// BreakState is empty: a door neither attacks nor casts.
func (a *skillDoor) BreakState() stats.BreakState { return stats.BreakState{} }

// AttackType is CreatureTemplate's default, the fists.
func (a *skillDoor) AttackType() data.WeaponType { return data.WeaponFist }

// ActiveWeapon is none (Door.getActiveWeaponItem).
func (a *skillDoor) ActiveWeapon() (skills.Weapon, bool) { return skills.Weapon{}, false }

// HasShield is false.
func (a *skillDoor) HasShield() bool { return false }

// ChargedShot is false.
func (a *skillDoor) ChargedShot(skills.ShotType) bool { return false }

// SetChargedShot does nothing.
func (a *skillDoor) SetChargedShot(skills.ShotType, bool) {}

// Effects returns the door's effect list, which stays empty.
func (a *skillDoor) Effects() *skills.EffectList { return a.el }

// SkillLevel is 0: doors know no skills.
func (a *skillDoor) SkillLevel(int) int { return 0 }

// AsPlayer is nil.
func (a *skillDoor) AsPlayer() skills.Player { return nil }

// SendMessage does nothing.
func (a *skillDoor) SendMessage(serverpackets.SystemMessage) {}

// SendText does nothing.
func (a *skillDoor) SendText(string) {}

// SendDamageMessage does nothing.
func (a *skillDoor) SendDamageMessage(skills.Creature, int, bool, bool, bool) {}

// ReduceCurrentHP takes a skill's damage through the door's status.
func (a *skillDoor) ReduceCurrentHP(damage float64, attacker skills.Creature, _ *skills.Skill) {
	if s := a.d.srv; s != nil {
		s.reduceDoorHP(a.d, damage, s.gameCreature(attacker))
	}
}

// NotifyDamageReceived does nothing: doors have no damage listeners.
func (a *skillDoor) NotifyDamageReceived(float64, skills.Creature, *skills.Skill, bool, bool) {}

// BreakAttack does nothing.
func (a *skillDoor) BreakAttack() {}

// BreakCast does nothing.
func (a *skillDoor) BreakCast() {}

// DoDie breaks the door.
func (a *skillDoor) DoDie(killer skills.Creature) {
	if s := a.d.srv; s != nil {
		s.doorDie(a.d, s.gameCreature(killer))
	}
}

// Escape does nothing: doors do not move.
func (a *skillDoor) Escape(skills.TeleportWhere) {}

// UpdateAbnormalEffect does nothing (Door.updateAbnormalEffect).
func (a *skillDoor) UpdateAbnormalEffect() {}

// ReduceCurrentHPByDOT does nothing: doors take no damage over time.
func (a *skillDoor) ReduceCurrentHPByDOT(float64, skills.Creature, *skills.Skill) {}

// IsLethalable is false (the Door constructor's setLethalable(false)).
func (a *skillDoor) IsLethalable() bool { return false }

// AbortAttack does nothing.
func (a *skillDoor) AbortAttack() {}

// AbortCast does nothing.
func (a *skillDoor) AbortCast() {}

// StopMove does nothing.
func (a *skillDoor) StopMove() {}

// IsCastingNow is false.
func (a *skillDoor) IsCastingNow() bool { return false }

// CanAbortCast is false.
func (a *skillDoor) CanAbortCast() bool { return false }

// SetTarget does nothing.
func (a *skillDoor) SetTarget(skills.Creature) {}

// SetImmobilized does nothing.
func (a *skillDoor) SetImmobilized(bool) {}

// IsMovementDisabled is true: doors never move.
func (a *skillDoor) IsMovementDisabled() bool { return true }

// SetLockedTarget does nothing.
func (a *skillDoor) SetLockedTarget(skills.Creature) {}

// AsAttackable is nil: a door is no Attackable.
func (a *skillDoor) AsAttackable() skills.Attackable { return nil }

// NotifyAI hears attacks only (DoorAI.notifyActionAttacked).
func (a *skillDoor) NotifyAI(ev skills.AIEvent, other skills.Creature) {
	if s := a.d.srv; s != nil && ev == skills.AIAttacked {
		s.doorAttacked(a.d, s.gameCreature(other))
	}
}

// Decay does nothing: doors leave no corpse.
func (a *skillDoor) Decay(time.Duration) {}

// Revive brings a broken door back (doRevive).
func (a *skillDoor) Revive(float64) {
	if s := a.d.srv; s != nil {
		s.reviveDoor(a.d)
	}
}

// SetRunning does nothing.
func (a *skillDoor) SetRunning() {}

// SetWalking does nothing.
func (a *skillDoor) SetWalking() {}

// TeleToLocation does nothing: doors do not move.
func (a *skillDoor) TeleToLocation(skills.Location, bool) {}

// SetXYZ does nothing.
func (a *skillDoor) SetXYZ(skills.Location) {}

// BroadcastValidateLocation does nothing.
func (a *skillDoor) BroadcastValidateLocation() {}

// BroadcastStartRotation does nothing.
func (a *skillDoor) BroadcastStartRotation(int, int, int) {}

// BroadcastStopRotation does nothing.
func (a *skillDoor) BroadcastStopRotation(int, int) {}

// StartFakeDeath does nothing.
func (a *skillDoor) StartFakeDeath() {}

// StopFakeDeath does nothing.
func (a *skillDoor) StopFakeDeath(bool) {}

// BroadcastFakeDeathEnd does nothing.
func (a *skillDoor) BroadcastFakeDeathEnd() {}

// SetCollisionGrown does nothing.
func (a *skillDoor) SetCollisionGrown(bool) {}

// MakeTriggerCast does nothing: doors never cast.
func (a *skillDoor) MakeTriggerCast(*skills.Skill, skills.Creature, bool) {}

// AddDamageListener does nothing: no effect that listens lands on a door.
func (a *skillDoor) AddDamageListener(any, func(skills.DamageEvent)) {}

// RemoveDamageListeners does nothing.
func (a *skillDoor) RemoveDamageListeners(any) {}
