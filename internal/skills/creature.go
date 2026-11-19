package skills

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/stats"
)

// Creature is a player, NPC or summon as skills, effects, conditions and
// target types see it: the Mobius Creature methods they call.
// internal/game implements it for its players and NPCs. Methods are called
// on the world loop only.
//
// Stat reads take the other creature and the skill of the calculation, as
// Mobius calcStat does; the game passes them to stats in a stats.Context
// (Target holding the Creature, Skill the *Skill), where the compiled
// conditions of stat functions find them again.
type Creature interface {
	// ObjectID is the object ID.
	ObjectID() int32
	// Name is the name system messages show.
	Name() string
	// InstanceType is the Mobius class of the creature (Player, Monster,
	// Folk, ...); the kind checks use InstanceType.Is.
	InstanceType() data.InstanceType
	// NpcID is the template ID of an NPC or door, 0 for players.
	NpcID() int
	// Race is the race (the template's for NPCs).
	Race() data.Race
	// Level is the level.
	Level() int
	// IsRaid reports a raid boss or one of its minions (Mobius isRaid).
	IsRaid() bool
	// InCategory reports whether the creature is in a category of
	// CategoryData.xml (Mobius isInCategory).
	InCategory(c data.CategoryType) bool

	// Location is the current position.
	Location() Location
	// Heading is the client heading, 0-65535.
	Heading() int
	// SetHeading turns the creature (front area skills turn the caster).
	SetHeading(heading int)
	// CollisionRadius is the template's collision radius, as Mobius
	// CreatureTemplate.getCollisionRadius returns it: an int, and the male
	// radius for players.
	CollisionRadius() int
	// InsideZone reports whether the creature is in a zone of the kind.
	InsideZone(z ZoneKind) bool
	// InZone reports whether the creature is in the zone with the ID.
	InZone(id int) bool
	// Target is the creature's current target, nil if none or not a
	// creature.
	Target() Creature

	// IsDead reports death; IsAlikeDead also fake death.
	IsDead() bool
	IsAlikeDead() bool
	// IsInvul reports invulnerability (GM invul, Invincible effects).
	IsInvul() bool
	// IsMoving, IsRunning and IsFlying give the movement state.
	IsMoving() bool
	IsRunning() bool
	IsFlying() bool

	// Current and maximum HP, MP and CP (Mobius getCurrentHp, getMaxHp,
	// getMaxRecoverableHp, ...). The setters keep the value within 0 and
	// the maximum and send the status updates Mobius sends.
	CurrentHP() float64
	SetCurrentHP(hp float64)
	CurrentMP() float64
	SetCurrentMP(mp float64)
	CurrentCP() float64
	SetCurrentCP(cp float64)
	MaxHP() int
	MaxMP() int
	MaxCP() int
	MaxRecoverableHP() int
	MaxRecoverableMP() int
	MaxRecoverableCP() int

	// Calc runs stat for the base value init with target and sk as the
	// calculation's context (Mobius calcStat(stat, init, target, skill)).
	Calc(stat stats.Stat, init float64, target Creature, sk *Skill) float64
	// AddModifiers and RemoveOwner change the stat set (Mobius
	// addStatFuncs and removeStatsOwner) and send the updates the change
	// needs.
	AddModifiers(mods ...stats.Modifier) error
	RemoveOwner(owner stats.Owner) int
	// Traits are the attack and defence trait arrays.
	Traits() *Traits
	// BaseStat is the value of a base stat (getSTR, ...); StatBonus is its
	// statBonus.xml multiplier (BaseStat.calcBonus), 1 for data.BaseNone.
	BaseStat(b data.BaseStat) int
	StatBonus(b data.BaseStat) float64
	// PAtk, PDef, MAtk and MDef are the Mobius getters with their target
	// and skill (getPAtk(target), getMDef(target, skill), ...), caps and
	// raid multipliers included.
	PAtk(target Creature) float64
	PDef(target Creature) float64
	MAtk(target Creature, sk *Skill) float64
	MDef(target Creature, sk *Skill) float64
	// PAtkSpd is the attack speed (a double in Mobius), MAtkSpd the casting
	// speed (an int).
	PAtkSpd() float64
	MAtkSpd() int
	// MCriticalHit is getMCriticalHit(target, skill), capped at MaxMCritRate.
	MCriticalHit(target Creature, sk *Skill) int
	// ShieldDef is getShldDef.
	ShieldDef() int
	// RandomDamageMultiplier is getRandomDamageMultiplier, drawing from r
	// (see stats.CombatRules.RandomDamageMultiplier).
	RandomDamageMultiplier(r stats.Rand) float64
	// BreakState tells what a hit could break (calcAtkBreak).
	BreakState() stats.BreakState

	// AttackType is the active weapon's type, or the template's attack type
	// without a weapon (getAttackType).
	AttackType() data.WeaponType
	// ActiveWeapon is getActiveWeaponItem: false without a weapon. Players
	// without a weapon hold their class fists, as in Mobius.
	ActiveWeapon() (Weapon, bool)
	// HasShield reports whether the left hand holds armor other than a
	// sigil (the shield check of calcShldUse).
	HasShield() bool
	// ChargedShot and SetChargedShot read and change the charged shots of
	// the active weapon.
	ChargedShot(t ShotType) bool
	SetChargedShot(t ShotType, charged bool)

	// Effects is the creature's effect list.
	Effects() *EffectList
	// SkillLevel is the level of a known skill, 0 if unknown.
	SkillLevel(id int) int

	// AsPlayer returns the acting player (Mobius asPlayer: the player
	// itself, the owner of a summon), nil for NPCs.
	AsPlayer() Player

	// SendMessage and SendText send a system message or a text (Mobius
	// sendPacket(SystemMessage) and sendMessage); NPCs ignore them.
	SendMessage(m serverpackets.SystemMessage)
	SendText(text string)
	// SendDamageMessage tells the attacker the damage it dealt (Mobius
	// sendDamageMessage).
	SendDamageMessage(target Creature, damage int, mcrit, pcrit, miss bool)
	// ReduceCurrentHP takes damage from attacker, with all that follows:
	// hate, death, status updates (Mobius reduceCurrentHp).
	ReduceCurrentHP(damage float64, attacker Creature, sk *Skill)
	// NotifyDamageReceived runs the damage-received listeners and triggers.
	NotifyDamageReceived(damage float64, attacker Creature, sk *Skill, crit, dot bool)
	// BreakAttack and BreakCast interrupt an attack or a cast.
	BreakAttack()
	BreakCast()
	// DoDie kills the creature.
	DoDie(killer Creature)
	// Escape teleports the creature as the Escape effect does: a guard to
	// its spawn, anyone else to the nearest place of the kind (Mobius
	// MapRegionData.getTeleToLocation), out of instances and the Seven
	// Signs dungeons.
	Escape(where TeleportWhere)
	// UpdateAbnormalEffect broadcasts the abnormal visual masks (UserInfo,
	// CharInfo or NpcInfo).
	UpdateAbnormalEffect()
}

// Player is what skills ask of a player besides Creature: Mobius Player
// methods. A Creature's AsPlayer returns it.
type Player interface {
	// HasEnteredWorld reports that EnterWorld has finished.
	HasEnteredWorld() bool
	// IsGM reports a GM; CanGiveDamage is false for GMs whose access level
	// forbids damage.
	IsGM() bool
	CanGiveDamage() bool
	// IsMageClass reports a mystic class; ClassID is the active class.
	IsMageClass() bool
	ClassID() int
	// IsFemale is the appearance's sex.
	IsFemale() bool
	// IsSitting reports sitting.
	IsSitting() bool
	// Karma, PkKills and PvpFlag are the PvP state.
	Karma() int
	PkKills() int
	PvpFlag() int
	// IsHero and InOlympiad report hero status and Olympiad mode.
	IsHero() bool
	InOlympiad() bool
	// Charges are the force charges (Focus Energy).
	Charges() int
	// CanEscape is ConditionPlayerCanEscape's test: not on an event, in a
	// duel, control blocked, flying, in the Olympiad, or (non-GM) in a
	// grand boss zone.
	CanEscape() bool
	// CurrentLoad, MaxLoad, BonusWeightPenalty and DietMode are the weight
	// state the weight condition reads.
	CurrentLoad() int
	MaxLoad() int
	BonusWeightPenalty() int
	DietMode() bool
	// InventoryNonQuestSize is the number of non-quest items, and
	// InventoryLimit the number of slots.
	InventoryNonQuestSize() int
	InventoryLimit() int
	// WornMask is Inventory.getWearedMask: the item type bits of the worn
	// items.
	WornMask() int
	// ArmorMasks returns the item type masks of the worn chest and legs
	// items (0 for an empty slot) and whether the chest is a full armor.
	ArmorMasks() (chest, legs int, fullArmor bool)
	// PaperdollItem returns the item ID and enchant level in a paperdoll
	// slot (Mobius Inventory.PAPERDOLL_*); false for an empty slot.
	PaperdollItem(slot int) (itemID, enchant int, ok bool)
	// InObserverMode, SiegeState, SiegeSide, ClanID, AllyID and IsInSiege
	// are what the area skills check before hitting another player.
	InObserverMode() bool
	SiegeState() int
	SiegeSide() int
	ClanID() int
	AllyID() int
	IsInSiege() bool
	// IsFestivalParticipant reports a Festival of Darkness participant.
	IsFestivalParticipant() bool
	// CheckPvPSkill is Player.checkPvpSkill: whether the player may use the
	// skill on target outside PvP zones.
	CheckPvPSkill(target Player, sk *Skill) bool
	// SendEffectIcons sends the buff icons: AbnormalStatusUpdate when
	// u.SendAbnormal, PartySpelled to the party when the player is in one.
	SendEffectIcons(u IconUpdate)
	// SendShortBuff sends ShortBuffStatusUpdate; a zero ShortBuff resets it.
	SendShortBuff(sb ShortBuff)
	// SendRegenMax sends ExRegenMax (the potion HP regeneration bar).
	SendRegenMax(duration, ticks int, amount float64)
}

// Weapon is what skills read of an active weapon.
type Weapon struct {
	Type     data.WeaponType
	BodyPart data.BodyPart
}

// The kind checks of Mobius (isPlayer, isPlayable, ...), by instance type.

func isPlayer(c Creature) bool     { return c != nil && c.InstanceType().Is(data.InstancePlayer) }
func isPlayable(c Creature) bool   { return c != nil && c.InstanceType().Is(data.InstancePlayable) }
func isAttackable(c Creature) bool { return c != nil && c.InstanceType().Is(data.InstanceAttackable) }
func isNpc(c Creature) bool        { return c != nil && c.InstanceType().Is(data.InstanceNpc) }
func isSummon(c Creature) bool     { return c != nil && c.InstanceType().Is(data.InstanceSummon) }
func isServitor(c Creature) bool   { return c != nil && c.InstanceType().Is(data.InstanceServitor) }
func isPet(c Creature) bool        { return c != nil && c.InstanceType().Is(data.InstancePet) }
func isDoor(c Creature) bool       { return c != nil && c.InstanceType().Is(data.InstanceDoor) }

// IsPlayer reports a player (Mobius isPlayer).
func IsPlayer(c Creature) bool { return isPlayer(c) }

// IsPlayable reports a player, summon or pet (Mobius isPlayable).
func IsPlayable(c Creature) bool { return isPlayable(c) }

// IsAttackable reports a monster or another attackable NPC (Mobius
// isAttackable).
func IsAttackable(c Creature) bool { return isAttackable(c) }

// same reports whether a and b are the same creature.
func same(a, b Creature) bool {
	return a != nil && b != nil && a.ObjectID() == b.ObjectID()
}

// Traits are a creature's trait arrays (Mobius CreatureStat): an attack and
// a defence multiplier per trait, kept as Java floats, with the number of
// effects that set each, and the invulnerabilities. The zero value is ready
// to use: every multiplier starts at 1.
type Traits struct {
	ready        bool
	attack       [data.NumTraitTypes]float32
	attackCount  [data.NumTraitTypes]int
	defence      [data.NumTraitTypes]float32
	defenceCount [data.NumTraitTypes]int
	invul        [data.NumTraitTypes]int
}

func (t *Traits) init() {
	if t.ready {
		return
	}
	for i := range t.attack {
		t.attack[i], t.defence[i] = 1, 1
	}
	t.ready = true
}

func traitIndex(tt data.TraitType) (int, bool) {
	return int(tt), tt.Valid()
}

// Attack returns the attack multiplier of a trait (getAttackTrait).
func (t *Traits) Attack(tt data.TraitType) float32 {
	i, ok := traitIndex(tt)
	if !ok || !t.ready {
		return 1
	}
	return t.attack[i]
}

// HasAttack reports whether an effect set the attack trait.
func (t *Traits) HasAttack(tt data.TraitType) bool {
	i, ok := traitIndex(tt)
	return ok && t.attackCount[i] > 0
}

// Defence returns the defence multiplier of a trait (getDefenceTrait).
func (t *Traits) Defence(tt data.TraitType) float32 {
	i, ok := traitIndex(tt)
	if !ok || !t.ready {
		return 1
	}
	return t.defence[i]
}

// HasDefence reports whether an effect set the defence trait.
func (t *Traits) HasDefence(tt data.TraitType) bool {
	i, ok := traitIndex(tt)
	return ok && t.defenceCount[i] > 0
}

// Invul reports invulnerability to a trait (isTraitInvul).
func (t *Traits) Invul(tt data.TraitType) bool {
	i, ok := traitIndex(tt)
	return ok && t.invul[i] > 0
}

// AddAttack multiplies the attack trait by v and counts it (AttackTrait's
// onStart).
func (t *Traits) AddAttack(tt data.TraitType, v float32) {
	i, ok := traitIndex(tt)
	if !ok {
		return
	}
	t.init()
	t.attack[i] *= v
	t.attackCount[i]++
}

// RemoveAttack undoes AddAttack, if the trait is counted (AttackTrait's
// onExit).
func (t *Traits) RemoveAttack(tt data.TraitType, v float32) {
	i, ok := traitIndex(tt)
	if !ok || t.attackCount[i] == 0 {
		return
	}
	t.attack[i] /= v
	t.attackCount[i]--
}

// AddDefence multiplies the defence trait by v and counts it
// (DefenceTrait's onStart for values under 2).
func (t *Traits) AddDefence(tt data.TraitType, v float32) {
	i, ok := traitIndex(tt)
	if !ok {
		return
	}
	t.init()
	t.defence[i] *= v
	t.defenceCount[i]++
}

// RemoveDefence undoes AddDefence, if the trait is counted.
func (t *Traits) RemoveDefence(tt data.TraitType, v float32) {
	i, ok := traitIndex(tt)
	if !ok || t.defenceCount[i] == 0 {
		return
	}
	t.defence[i] /= v
	t.defenceCount[i]--
}

// AddInvul counts an invulnerability to the trait (DefenceTrait values of 2
// and more).
func (t *Traits) AddInvul(tt data.TraitType) {
	if i, ok := traitIndex(tt); ok {
		t.invul[i]++
	}
}

// RemoveInvul undoes AddInvul, if counted.
func (t *Traits) RemoveInvul(tt data.TraitType) {
	if i, ok := traitIndex(tt); ok && t.invul[i] > 0 {
		t.invul[i]--
	}
}
