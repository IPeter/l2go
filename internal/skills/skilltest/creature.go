package skilltest

import (
	"time"

	"github.com/IPeter/l2go/internal/data"
	sp "github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/stats"
)

// Creature is a fake skills.Creature, and a skills.Player when its
// instance type is a player. Its stats are base values run through its
// modifiers; everything the engine does to it is recorded.
type Creature struct {
	ID    int32
	Nm    string
	Type  data.InstanceType
	Npc   int
	Rc    data.Race
	Lvl   int
	Raid  bool
	Cats  map[data.CategoryType]bool
	Loc   skills.Location
	Head  int
	Coll  int
	Zones map[skills.ZoneKind]bool
	Tgt   skills.Creature

	Dead, AlikeDead, Invul, Moving, Running, Flying bool

	HP           float64
	MP           float64
	CP           float64
	MaxHPV       int
	MaxMPV       int
	MaxCPV       int
	BasePAtk     float64
	BasePDef     float64
	BaseMAtk     float64
	BaseMDef     float64
	BasePAtkSpd  float64
	BaseMAtkSpd  float64
	MCrit        int
	ShieldDefV   int
	RandomDamage int
	Base         map[data.BaseStat]int
	Bonus        map[data.BaseStat]float64
	Mods         stats.Modifiers
	TraitsV      skills.Traits
	Break        stats.BreakState
	Attack       data.WeaponType
	Weapon       *skills.Weapon
	Shield       bool
	Shots        map[skills.ShotType]bool
	EL           *skills.EffectList
	Known        map[int]int

	// Player state.
	Entered    bool
	GM         bool
	NoDamage   bool
	Mage       bool
	Female     bool
	Sitting    bool
	ClassIDV   int
	KarmaV     int
	PkV        int
	PvpV       int
	ChargesV   int
	Hero       bool
	Olympiad   bool
	CanEscapeV bool
	Load       int
	MaxLoadV   int
	InvSize    int
	InvLimit   int
	Clan       int
	Ally       int
	Siege      int
	Side       int
	InSiegeV   bool
	Festival   bool
	Observer   bool
	Worn       int
	Chest      int
	Legs       int
	FullArmor  bool

	// Recorded calls.
	Messages      []sp.SystemMessage
	Texts         []string
	Damage        []float64
	DamageMsgs    []int
	Icons         []skills.IconUpdate
	ShortBuffs    []skills.ShortBuff
	RegenMax      [][3]float64
	Escapes       []skills.TeleportWhere
	VisualUpdates int
	Breaks        int
	Died          bool

	// Wave B state and recorded calls.
	NoLethal     bool // IsLethalable is false
	Casting      bool // IsCastingNow
	Abortable    bool // CanAbortCast
	Immobilized  bool
	DOTDamage    []float64
	Aborts       []string // "attack", "cast" and "move", in call order
	AI           []AIEvent
	Decays       []time.Duration
	Revives      []float64
	ReviveAsks   []ReviveAsk
	ChargeCalls  [][2]int // IncreaseCharges(count, max)
	Items        []skills.CreatedItem
	ItemMessages []bool  // the sendMessage of each AddItem
	ItemUpdates  [][]int // SendItemsUpdate's item IDs
	// ItemStackable lists the stackable item IDs; NextItemID numbers the
	// instances AddItem makes (from 1000).
	ItemStackable map[int]bool
	NextItemID    int32

	// Reactions holds the state and records of the AI, hate, appearance
	// and the other calls of the hate, trigger and utility effects.
	Reactions
}

// AIEvent is one recorded NotifyAI call.
type AIEvent struct {
	Event skills.AIEvent
	Other skills.Creature
}

// ReviveAsk is one recorded ReviveRequest.
type ReviveAsk struct {
	Reviver skills.Creature
	Pet     bool
	Power   int
}

// New returns a creature with 1000 HP, MP and CP, level 1 and an effect
// list on env.
func New(env *skills.Env, id int32, t data.InstanceType) *Creature {
	c := &Creature{
		ID: id, Nm: "c" + itoa(int(id)), Type: t, Lvl: 1,
		HP: 1000, MP: 1000, CP: 1000, MaxHPV: 1000, MaxMPV: 1000, MaxCPV: 1000,
		BasePAtk: 100, BasePDef: 100, BaseMAtk: 100, BaseMDef: 100,
		BasePAtkSpd: 300, BaseMAtkSpd: 333,
		Attack: data.WeaponFist,
		Shots:  make(map[skills.ShotType]bool),
		Known:  make(map[int]int),
		Zones:  make(map[skills.ZoneKind]bool),
	}
	c.EL = skills.NewEffectList(env, c)
	return c
}

// NewPlayer returns a player that has entered the world.
func NewPlayer(env *skills.Env, id int32) *Creature {
	c := New(env, id, data.InstancePlayer)
	c.Entered = true
	return c
}

// NewMonster returns a monster.
func NewMonster(env *skills.Env, id int32) *Creature { return New(env, id, data.InstanceMonster) }

// MessageIDs returns the IDs of the recorded system messages.
func (c *Creature) MessageIDs() []sp.SystemMessageID {
	ids := make([]sp.SystemMessageID, len(c.Messages))
	for i, m := range c.Messages {
		ids[i] = m.ID
	}
	return ids
}

// Reset forgets the recorded messages and texts.
func (c *Creature) Reset() { c.Messages, c.Texts = nil, nil }

// ObjectID returns the ID.
func (c *Creature) ObjectID() int32 { return c.ID }

// Name returns the name.
func (c *Creature) Name() string { return c.Nm }

// InstanceType returns the kind.
func (c *Creature) InstanceType() data.InstanceType { return c.Type }

// NpcID returns the NPC template ID.
func (c *Creature) NpcID() int { return c.Npc }

// Race returns the race.
func (c *Creature) Race() data.Race { return c.Rc }

// Level returns the level.
func (c *Creature) Level() int { return c.Lvl }

// IsRaid reports a raid.
func (c *Creature) IsRaid() bool { return c.Raid }

// InCategory reports a category.
func (c *Creature) InCategory(t data.CategoryType) bool { return c.Cats[t] }

// Location returns the position.
func (c *Creature) Location() skills.Location { return c.Loc }

// Heading returns the heading.
func (c *Creature) Heading() int { return c.Head }

// SetHeading turns the creature.
func (c *Creature) SetHeading(h int) { c.Head = h }

// CollisionRadius returns the collision radius.
func (c *Creature) CollisionRadius() int { return c.Coll }

// InsideZone reports a zone kind.
func (c *Creature) InsideZone(z skills.ZoneKind) bool { return c.Zones[z] }

// InZone reports false.
func (c *Creature) InZone(int) bool { return false }

// Target returns the target.
func (c *Creature) Target() skills.Creature { return c.Tgt }

// IsDead reports death.
func (c *Creature) IsDead() bool { return c.Dead }

// IsAlikeDead reports death or fake death.
func (c *Creature) IsAlikeDead() bool { return c.Dead || c.AlikeDead }

// IsInvul reports invulnerability.
func (c *Creature) IsInvul() bool { return c.Invul }

// IsMoving reports moving.
func (c *Creature) IsMoving() bool { return c.Moving }

// IsRunning reports running.
func (c *Creature) IsRunning() bool { return c.Running }

// IsFlying reports flying.
func (c *Creature) IsFlying() bool { return c.Flying }

func clampTo(v float64, hi int) float64 { return max(0, min(v, float64(hi))) }

// CurrentHP returns the HP.
func (c *Creature) CurrentHP() float64 { return c.HP }

// SetCurrentHP sets the HP within 0 and the maximum.
func (c *Creature) SetCurrentHP(hp float64) { c.HP = clampTo(hp, c.MaxHP()) }

// CurrentMP returns the MP.
func (c *Creature) CurrentMP() float64 { return c.MP }

// SetCurrentMP sets the MP within 0 and the maximum.
func (c *Creature) SetCurrentMP(mp float64) { c.MP = clampTo(mp, c.MaxMPV) }

// CurrentCP returns the CP.
func (c *Creature) CurrentCP() float64 { return c.CP }

// SetCurrentCP sets the CP within 0 and the maximum.
func (c *Creature) SetCurrentCP(cp float64) { c.CP = clampTo(cp, c.MaxCPV) }

// MaxHP returns MaxHPV through the maxHp modifiers, truncated.
func (c *Creature) MaxHP() int { return int(c.Calc(stats.MaxHP, float64(c.MaxHPV), nil, nil)) }

// MaxMP returns the maximum MP.
func (c *Creature) MaxMP() int { return c.MaxMPV }

// MaxCP returns the maximum CP.
func (c *Creature) MaxCP() int { return c.MaxCPV }

// MaxRecoverableHP returns the maximum HP.
func (c *Creature) MaxRecoverableHP() int { return c.MaxHP() }

// MaxRecoverableMP returns the maximum MP.
func (c *Creature) MaxRecoverableMP() int { return c.MaxMPV }

// MaxRecoverableCP returns the maximum CP.
func (c *Creature) MaxRecoverableCP() int { return c.MaxCPV }

// Traits returns the trait arrays.
func (c *Creature) Traits() *skills.Traits { return &c.TraitsV }

// BreakState returns what a hit could break.
func (c *Creature) BreakState() stats.BreakState { return c.Break }

// Calc runs the creature's modifiers of st over init.
func (c *Creature) Calc(st stats.Stat, init float64, target skills.Creature, sk *skills.Skill) float64 {
	var ctx stats.Context
	if target != nil {
		ctx.Target = target
	}
	if sk != nil {
		ctx.Skill = sk
	}
	v := init
	for _, m := range c.Mods.Of(st) {
		if m.Cond != nil && !m.Cond.Applies(ctx) {
			continue
		}
		switch m.Op {
		case stats.OpSet:
			v = m.Value
		case stats.OpAdd, stats.OpEnchant, stats.OpEnchantHP:
			v += m.Value
		case stats.OpSub:
			v -= m.Value
		case stats.OpMul:
			v *= m.Value
		case stats.OpDiv:
			if m.Value != 0 {
				v /= m.Value
			}
		}
	}
	return v
}

// AddModifiers adds stat modifiers.
func (c *Creature) AddModifiers(mods ...stats.Modifier) error { return c.Mods.Add(mods...) }

// RemoveOwner removes an owner's modifiers.
func (c *Creature) RemoveOwner(o stats.Owner) int { return c.Mods.RemoveOwner(o) }

// BaseStat returns a base stat (0 unless set).
func (c *Creature) BaseStat(b data.BaseStat) int { return c.Base[b] }

// StatBonus returns a base stat's bonus (1 unless set).
func (c *Creature) StatBonus(b data.BaseStat) float64 {
	if v, ok := c.Bonus[b]; ok {
		return v
	}
	return 1
}

// PAtk is the base P.Atk through the modifiers.
func (c *Creature) PAtk(t skills.Creature) float64 { return c.Calc(stats.PAtk, c.BasePAtk, t, nil) }

// PDef is the base P.Def through the modifiers.
func (c *Creature) PDef(t skills.Creature) float64 { return c.Calc(stats.PDef, c.BasePDef, t, nil) }

// MAtk is the base M.Atk through the modifiers.
func (c *Creature) MAtk(t skills.Creature, sk *skills.Skill) float64 {
	return c.Calc(stats.MAtk, c.BaseMAtk, t, sk)
}

// MDef is the base M.Def through the modifiers.
func (c *Creature) MDef(t skills.Creature, sk *skills.Skill) float64 {
	return c.Calc(stats.MDef, c.BaseMDef, t, sk)
}

// PAtkSpd is the base attack speed through the modifiers.
func (c *Creature) PAtkSpd() float64 { return c.Calc(stats.PAtkSpd, c.BasePAtkSpd, nil, nil) }

// MAtkSpd is the base casting speed through the modifiers, truncated.
func (c *Creature) MAtkSpd() int { return int(c.Calc(stats.MAtkSpd, c.BaseMAtkSpd, nil, nil)) }

// MCriticalHit returns MCrit.
func (c *Creature) MCriticalHit(skills.Creature, *skills.Skill) int { return c.MCrit }

// ShieldDef returns ShieldDefV.
func (c *Creature) ShieldDef() int { return c.ShieldDefV }

// RandomDamageMultiplier draws Between(-RandomDamage, RandomDamage).
func (c *Creature) RandomDamageMultiplier(r stats.Rand) float64 {
	return 1 + float64(r.Between(-c.RandomDamage, c.RandomDamage))/100
}

// AttackType returns Attack.
func (c *Creature) AttackType() data.WeaponType { return c.Attack }

// ActiveWeapon returns the weapon, if any.
func (c *Creature) ActiveWeapon() (skills.Weapon, bool) {
	if c.Weapon == nil {
		return skills.Weapon{}, false
	}
	return *c.Weapon, true
}

// HasShield returns Shield.
func (c *Creature) HasShield() bool { return c.Shield }

// ChargedShot reports a charged shot.
func (c *Creature) ChargedShot(t skills.ShotType) bool { return c.Shots[t] }

// SetChargedShot charges or spends a shot.
func (c *Creature) SetChargedShot(t skills.ShotType, v bool) { c.Shots[t] = v }

// Effects returns the effect list.
func (c *Creature) Effects() *skills.EffectList { return c.EL }

// SkillLevel returns a known skill's level.
func (c *Creature) SkillLevel(id int) int { return c.Known[id] }

// AsPlayer returns the creature itself for a player, else nil.
func (c *Creature) AsPlayer() skills.Player {
	if !c.Type.Is(data.InstancePlayer) {
		return nil
	}
	return c
}

// SendMessage records a system message.
func (c *Creature) SendMessage(m sp.SystemMessage) { c.Messages = append(c.Messages, m) }

// SendText records a text.
func (c *Creature) SendText(t string) { c.Texts = append(c.Texts, t) }

// SendDamageMessage records the damage.
func (c *Creature) SendDamageMessage(_ skills.Creature, dmg int, _, _, _ bool) {
	c.DamageMsgs = append(c.DamageMsgs, dmg)
}

// ReduceCurrentHP takes damage; the creature dies at 0 HP.
func (c *Creature) ReduceCurrentHP(dmg float64, _ skills.Creature, _ *skills.Skill) {
	c.Damage = append(c.Damage, dmg)
	c.HP = max(0, c.HP-dmg)
	if c.HP == 0 {
		c.Dead = true
	}
}

// NotifyDamageReceived runs the damage listeners (AddDamageListener).
func (c *Creature) NotifyDamageReceived(dmg float64, attacker skills.Creature, sk *skills.Skill, crit, dot bool) {
	c.notifyDamage(skills.DamageEvent{Attacker: attacker, Target: c, Damage: dmg, Skill: sk, Crit: crit, DOT: dot})
}

// BreakAttack does nothing.
func (c *Creature) BreakAttack() {}

// BreakCast counts the breaks.
func (c *Creature) BreakCast() { c.Breaks++ }

// DoDie kills the creature.
func (c *Creature) DoDie(skills.Creature) { c.Dead, c.Died = true, true }

// Escape records the destination.
func (c *Creature) Escape(w skills.TeleportWhere) { c.Escapes = append(c.Escapes, w) }

// UpdateAbnormalEffect counts the broadcasts.
func (c *Creature) UpdateAbnormalEffect() { c.VisualUpdates++ }

// HasEnteredWorld returns Entered.
func (c *Creature) HasEnteredWorld() bool { return c.Entered }

// IsGM returns GM.
func (c *Creature) IsGM() bool { return c.GM }

// CanGiveDamage is false with NoDamage.
func (c *Creature) CanGiveDamage() bool { return !c.NoDamage }

// IsMageClass returns Mage.
func (c *Creature) IsMageClass() bool { return c.Mage }

// ClassID returns ClassIDV.
func (c *Creature) ClassID() int { return c.ClassIDV }

// IsFemale returns Female.
func (c *Creature) IsFemale() bool { return c.Female }

// IsSitting returns Sitting.
func (c *Creature) IsSitting() bool { return c.Sitting }

// Karma returns KarmaV.
func (c *Creature) Karma() int { return c.KarmaV }

// PkKills returns PkV.
func (c *Creature) PkKills() int { return c.PkV }

// PvpFlag returns PvpV.
func (c *Creature) PvpFlag() int { return c.PvpV }

// IsHero returns Hero.
func (c *Creature) IsHero() bool { return c.Hero }

// InOlympiad returns Olympiad.
func (c *Creature) InOlympiad() bool { return c.Olympiad }

// Charges returns ChargesV.
func (c *Creature) Charges() int { return c.ChargesV }

// CanEscape returns CanEscapeV.
func (c *Creature) CanEscape() bool { return c.CanEscapeV }

// CurrentLoad returns Load.
func (c *Creature) CurrentLoad() int { return c.Load }

// MaxLoad returns MaxLoadV.
func (c *Creature) MaxLoad() int { return c.MaxLoadV }

// BonusWeightPenalty returns 0.
func (c *Creature) BonusWeightPenalty() int { return 0 }

// DietMode returns false.
func (c *Creature) DietMode() bool { return false }

// InventoryNonQuestSize returns InvSize.
func (c *Creature) InventoryNonQuestSize() int { return c.InvSize }

// InventoryLimit returns InvLimit.
func (c *Creature) InventoryLimit() int { return c.InvLimit }

// WornMask returns Worn.
func (c *Creature) WornMask() int { return c.Worn }

// ArmorMasks returns Chest, Legs and FullArmor.
func (c *Creature) ArmorMasks() (chest, legs int, full bool) { return c.Chest, c.Legs, c.FullArmor }

// PaperdollItem returns an empty slot.
func (c *Creature) PaperdollItem(int) (itemID, enchant int, ok bool) { return 0, 0, false }

// InObserverMode returns Observer.
func (c *Creature) InObserverMode() bool { return c.Observer }

// SiegeState returns Siege.
func (c *Creature) SiegeState() int { return c.Siege }

// SiegeSide returns Side.
func (c *Creature) SiegeSide() int { return c.Side }

// ClanID returns Clan.
func (c *Creature) ClanID() int { return c.Clan }

// AllyID returns Ally.
func (c *Creature) AllyID() int { return c.Ally }

// IsInSiege returns InSiegeV.
func (c *Creature) IsInSiege() bool { return c.InSiegeV }

// IsFestivalParticipant returns Festival.
func (c *Creature) IsFestivalParticipant() bool { return c.Festival }

// CheckPvPSkill allows everything.
func (c *Creature) CheckPvPSkill(skills.Player, *skills.Skill) bool { return true }

// SendEffectIcons records the icons.
func (c *Creature) SendEffectIcons(u skills.IconUpdate) { c.Icons = append(c.Icons, u) }

// SendShortBuff records the short buff.
func (c *Creature) SendShortBuff(sb skills.ShortBuff) { c.ShortBuffs = append(c.ShortBuffs, sb) }

// SendRegenMax records duration, ticks and amount.
func (c *Creature) SendRegenMax(d, ticks int, amount float64) {
	c.RegenMax = append(c.RegenMax, [3]float64{float64(d), float64(ticks), amount})
}

// ReduceCurrentHPByDOT records the damage and takes it like ReduceCurrentHP.
func (c *Creature) ReduceCurrentHPByDOT(dmg float64, attacker skills.Creature, sk *skills.Skill) {
	c.DOTDamage = append(c.DOTDamage, dmg)
	c.ReduceCurrentHP(dmg, attacker, sk)
}

// IsLethalable is false with NoLethal.
func (c *Creature) IsLethalable() bool { return !c.NoLethal }

// AbortAttack records "attack".
func (c *Creature) AbortAttack() { c.Aborts = append(c.Aborts, "attack") }

// AbortCast records "cast" and ends the cast.
func (c *Creature) AbortCast() {
	c.Aborts = append(c.Aborts, "cast")
	c.Casting = false
}

// StopMove records "move" and stops moving.
func (c *Creature) StopMove() {
	c.Aborts = append(c.Aborts, "move")
	c.Moving = false
}

// IsCastingNow returns Casting.
func (c *Creature) IsCastingNow() bool { return c.Casting }

// CanAbortCast returns Abortable.
func (c *Creature) CanAbortCast() bool { return c.Abortable }

// SetTarget sets the target.
func (c *Creature) SetTarget(t skills.Creature) { c.Tgt = t }

// SetImmobilized sets Immobilized.
func (c *Creature) SetImmobilized(v bool) { c.Immobilized = v }

// NotifyAI records the event.
func (c *Creature) NotifyAI(ev skills.AIEvent, other skills.Creature) {
	c.AI = append(c.AI, AIEvent{Event: ev, Other: other})
}

// AIEvents returns the recorded AI events without their creatures.
func (c *Creature) AIEvents() []skills.AIEvent {
	evs := make([]skills.AIEvent, len(c.AI))
	for i, e := range c.AI {
		evs[i] = e.Event
	}
	return evs
}

// Decay records the delay.
func (c *Creature) Decay(d time.Duration) { c.Decays = append(c.Decays, d) }

// Revive records the percent and revives the creature.
func (c *Creature) Revive(percent float64) {
	c.Revives = append(c.Revives, percent)
	c.Dead, c.AlikeDead = false, false
}

// IncreaseCharges records the call and adds the charges up to max.
func (c *Creature) IncreaseCharges(count, maxCharges int) {
	c.ChargeCalls = append(c.ChargeCalls, [2]int{count, maxCharges})
	if c.ChargesV < maxCharges {
		c.ChargesV = min(c.ChargesV+count, maxCharges)
	}
}

// ReviveRequest records the request.
func (c *Creature) ReviveRequest(reviver skills.Creature, pet bool, power int) {
	c.ReviveAsks = append(c.ReviveAsks, ReviveAsk{Reviver: reviver, Pet: pet, Power: power})
}

// AddItem records the item. A stackable item ID (ItemStackable) adds to its
// earlier instance, if any.
func (c *Creature) AddItem(itemID, count, enchant int, _ skills.Creature, sendMessage bool) (skills.CreatedItem, bool) {
	if count <= 0 {
		return skills.CreatedItem{}, false
	}
	c.ItemMessages = append(c.ItemMessages, sendMessage)
	stackable := c.ItemStackable[itemID]
	if stackable {
		for _, it := range c.Items {
			if it.ItemID == itemID {
				return it, true
			}
		}
	}
	if c.NextItemID == 0 {
		c.NextItemID = 1000
	}
	it := skills.CreatedItem{ObjectID: c.NextItemID, ItemID: itemID, Stackable: stackable, Enchant: max(enchant, 0)}
	c.NextItemID++
	c.Items = append(c.Items, it)
	return it, true
}

// SendItemsUpdate records the item IDs.
func (c *Creature) SendItemsUpdate(itemIDs []int) {
	c.ItemUpdates = append(c.ItemUpdates, append([]int(nil), itemIDs...))
}

var (
	_ skills.Creature = (*Creature)(nil)
	_ skills.Player   = (*Creature)(nil)
)
