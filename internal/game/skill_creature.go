package game

import (
	"math"
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The skills.Creature adapters of players (skillPlayer) and NPCs (skillNpc).
// skillBase holds what both carry: the known skills, the effect list, the
// trait arrays and the small states effects set; its methods are the
// skills.Creature methods, which look at the player or the NPC behind it.
// Each creature has one adapter for its whole life, made with it
// (newSkillPlayer, newSkillNpc), so the skill engine can keep and compare
// it.

// maxMCritRate caps the magic critical rate (PlayerConfig.MAX_MCRIT_RATE).
const maxMCritRate = 200

// skillBase is the adapter part players and NPCs share. Exactly one of p
// and n is set.
type skillBase struct {
	srv  *Server
	p    *Player
	n    *Npc
	self skills.Creature // the adapter itself

	known  knownSkills
	el     *skills.EffectList
	traits skills.Traits
	reuse  reuseStamps // skill_list.go
	cast   castState   // cast.go

	// shots are an NPC's charged shots by skills.ShotType; a player's are
	// on its weapon (K14, shot.go).
	shots       [4]bool
	listeners   []skillDamageListener
	immobilized bool
	locked      skills.Creature

	admin creatureAdmin // GM paralysis and visual effects (admin_gm.go)
}

// skillDamageListener is a damage listener of an effect
// (AddDamageListener).
type skillDamageListener struct {
	owner any
	fn    func(skills.DamageEvent)
}

// skillPlayer is a player as the skill engine sees it: skills.Creature and
// skills.Player.
type skillPlayer struct {
	skillBase

	// stored are the rows of character_skills of the active class: the
	// skill levels saved with the character, in the order they were added
	// (skill_persist.go).
	stored []storedSkill
	// saved freezes the effects to save when the player leaves, before they
	// end (skill_persist.go); nil while the player plays.
	saved []storage.SavedSkill
	// listTimer is the pending SkillList (sendSkillList's 300 ms task).
	listTimer world.TimerID
	// entered is Mobius _enteredWorld: set once EnterWorld has restored the
	// effects, so that restored effects send no "flow through you".
	entered bool

	charges                    int
	fakeDeath, recentFakeDeath bool
}

// skillNpc is an NPC as the skill engine sees it: skills.Creature, and
// skills.Attackable for attackables.
type skillNpc struct {
	skillBase

	instance data.InstanceType
	spoiler  int32
	fearTime int
}

var (
	_ skills.Creature   = (*skillPlayer)(nil)
	_ skills.Player     = (*skillPlayer)(nil)
	_ skills.Creature   = (*skillNpc)(nil)
	_ skills.Attackable = (*skillNpc)(nil)
)

// newSkillPlayer makes the adapter of p with an empty effect list. It
// touches no world state.
func (s *Server) newSkillPlayer(p *Player) *skillPlayer {
	a := &skillPlayer{}
	a.skillBase = skillBase{srv: s, p: p, self: a}
	a.el = skills.NewEffectList(s.skills.env, a)
	return a
}

// newSkillNpc makes the adapter of n with an empty effect list.
func (s *Server) newSkillNpc(n *Npc) *skillNpc {
	a := &skillNpc{}
	a.skillBase = skillBase{srv: s, n: n, self: a}
	a.instance, _ = data.ParseInstanceType(n.Template.Type)
	a.el = skills.NewEffectList(s.skills.env, a)
	return a
}

// SkillCreature returns the player as the skill engine sees it: the
// skills.Creature (and skills.Player) that effects, conditions and casts
// take. It is the same value for the player's whole session.
func (p *Player) SkillCreature() skills.Creature { return p.skill }

// Effects returns the player's effect list. Loop-only.
func (p *Player) Effects() *skills.EffectList { return p.skill.el }

// SkillCreature returns the NPC as the skill engine sees it.
func (n *Npc) SkillCreature() skills.Creature { return n.skill }

// Effects returns the NPC's effect list. Loop-only.
func (n *Npc) Effects() *skills.EffectList { return n.skill.el }

// effectHolder is an adapter that may be nil (a creature made without one).
type effectHolder interface {
	effectList() *skills.EffectList
	adminVisual() uint32
}

func (a *skillPlayer) effectList() *skills.EffectList {
	if a == nil {
		return nil
	}
	return a.el
}

func (a *skillNpc) effectList() *skills.EffectList {
	if a == nil {
		return nil
	}
	return a.el
}

// abnormalMask is getAbnormalVisualEffects: the normal visual mask of the
// effects and of the GM's visual effects (admin_gm.go), which UserInfo,
// CharInfo and NpcInfo show.
func abnormalMask(a effectHolder) int32 {
	el := a.effectList()
	if el == nil {
		return 0
	}
	return int32(el.Visual().Normal | a.adminVisual()) //nolint:gosec // a bit mask
}

// SkillCreature returns the skills.Creature of a player or an NPC, nil for
// anything else.
func (s *Server) SkillCreature(c Creature) skills.Creature { return s.skillCreatureOf(c) }

// skillCreatureOf returns the adapter of a world object, nil if it is not a
// creature with one.
func (s *Server) skillCreatureOf(o world.Object) skills.Creature {
	switch v := o.(type) {
	case *Player:
		if v != nil && v.skill != nil {
			return v.skill
		}
	case *Npc:
		if v != nil && v.skill != nil {
			return v.skill
		}
	}
	return nil
}

// statsRef is what stat conditions see as c (stats.Fighter.Ref): its
// adapter, so that the compiled skill conditions find a skills.Creature.
func (s *Server) statsRef(c Creature) any {
	if sc := s.skillCreatureOf(c); sc != nil {
		return sc
	}
	return c
}

// GameCreature returns the player or NPC behind a skills.Creature, nil for
// another value.
func (s *Server) GameCreature(c skills.Creature) Creature { return s.gameCreature(c) }

func (s *Server) gameCreature(c skills.Creature) Creature {
	switch v := c.(type) {
	case *skillPlayer:
		return v.p
	case *skillNpc:
		return v.n
	}
	return nil
}

// gc returns the player or NPC behind the adapter.
func (b *skillBase) gc() Creature {
	if b.p != nil {
		return b.p
	}
	return b.n
}

// skillContext is the stats.Context of a calculation against target with
// sk: nil stays an untyped nil, so conditions see "no target".
func skillContext(target skills.Creature, sk *skills.Skill) stats.Context {
	var ctx stats.Context
	if target != nil {
		ctx.Target = target
	}
	if sk != nil {
		ctx.Skill = sk
	}
	return ctx
}

// ObjectID returns the object ID.
func (b *skillBase) ObjectID() int32 { return int32(b.gc().ID()) } //nolint:gosec // object IDs fit

// Name returns the name.
func (b *skillBase) Name() string { return b.gc().Name() }

// InstanceType returns Player.
func (a *skillPlayer) InstanceType() data.InstanceType { return data.InstancePlayer }

// InstanceType returns the class of the template's type (Monster, Folk, ...).
func (a *skillNpc) InstanceType() data.InstanceType { return a.instance }

// NpcID returns the template ID of an NPC, 0 for a player.
func (b *skillBase) NpcID() int {
	if b.n != nil {
		return b.n.Template.ID
	}
	return 0
}

// Race returns the race.
func (b *skillBase) Race() data.Race {
	if b.n != nil {
		return b.n.Template.Race
	}
	return data.Race(b.p.char.Race) //nolint:gosec // races are small
}

// Level returns the level.
func (b *skillBase) Level() int {
	if b.n != nil {
		return b.n.Level()
	}
	return b.p.Level()
}

// IsRaid reports a raid boss.
func (b *skillBase) IsRaid() bool { return creatureRaid(b.gc()) }

// InCategory reports whether a player's class is in the category; NPCs are
// in none.
func (b *skillBase) InCategory(t data.CategoryType) bool {
	cats := b.srv.items.categories
	return b.p != nil && cats != nil && cats.InCategory(t, b.p.ClassID())
}

// Location returns the position.
func (b *skillBase) Location() skills.Location {
	pos := b.gc().Position()
	return skills.Location{X: int(pos.X), Y: int(pos.Y), Z: int(pos.Z)}
}

// Heading returns the heading.
func (b *skillBase) Heading() int { return int(b.gc().Position().Heading) }

// SetHeading turns the creature.
func (b *skillBase) SetHeading(h int) { setCreatureHeading(b.gc(), int32(h)) } //nolint:gosec // 0-65535

// CollisionRadius returns the template's collision radius.
func (b *skillBase) CollisionRadius() int { return attackCollision(b.gc()) }

// InsideZone reports a zone kind: a player's zones, or the zones at an NPC's
// position.
func (b *skillBase) InsideZone(z skills.ZoneKind) bool {
	var kind ZoneFlag
	switch z {
	case skills.ZonePvP:
		kind = ZonePvP
	case skills.ZonePeace:
		kind = ZonePeace
	case skills.ZoneSiege:
		kind = ZoneSiege
	default:
		return false
	}
	if b.p != nil {
		return b.p.Zones.Inside(kind)
	}
	return b.srv.npcInsideZone(b.n, kind)
}

// InZone reports whether a player is in the zone with the ID.
func (b *skillBase) InZone(id int) bool { return b.p != nil && slices.Contains(b.p.Zones.IDs(), id) }

// Target returns the current target.
func (b *skillBase) Target() skills.Creature {
	if b.p != nil {
		return b.srv.skillCreatureOf(b.p.Target)
	}
	return b.srv.skillCreatureOf(b.n.ai.target)
}

// IsDead reports death.
func (b *skillBase) IsDead() bool { return b.gc().IsDead() }

// IsAlikeDead reports death or fake death.
func (b *skillBase) IsAlikeDead() bool {
	if a, ok := b.self.(*skillPlayer); ok && a.fakeDeath {
		return true
	}
	return b.IsDead()
}

// IsInvul reports invulnerability: an Invincible effect, or a player
// waiting for Appearing after a teleport (Mobius isTeleporting) or made
// invulnerable by //invul (admin_gm.go), or an NPC a script made
// invulnerable (quest_world_ai.go).
func (b *skillBase) IsInvul() bool {
	if b.el.IsAffected(skills.FlagInvul) {
		return true
	}
	if b.n != nil {
		return b.n.script.invul
	}
	return b.p != nil && (b.p.Teleporting() || b.p.admin.invul)
}

// IsMoving reports a creature on its way somewhere.
func (b *skillBase) IsMoving() bool { return creatureMoving(b.gc()) }

// IsRunning reports running.
func (b *skillBase) IsRunning() bool {
	if b.p != nil {
		return b.p.Running()
	}
	return b.n.Stats.Motion().Running
}

// IsFlying reports flying.
func (b *skillBase) IsFlying() bool {
	if b.p != nil {
		return b.p.stats.Motion().Flying
	}
	return b.n.Stats.Motion().Flying
}

// CurrentHP returns the HP.
func (b *skillBase) CurrentHP() float64 { return b.gc().CurrentHP() }

// SetCurrentHP sets the HP through the status, with its packets.
func (b *skillBase) SetCurrentHP(hp float64) {
	if b.p != nil {
		b.p.SetCurrentHP(hp)
	} else {
		b.n.SetCurrentHP(hp)
	}
}

// CurrentMP returns the MP.
func (b *skillBase) CurrentMP() float64 {
	if b.p != nil {
		return b.p.CurMP
	}
	return b.n.CurMP
}

// SetCurrentMP sets the MP through the status.
func (b *skillBase) SetCurrentMP(mp float64) {
	if b.p != nil {
		b.p.SetCurrentMP(mp)
	} else {
		b.n.SetCurrentMP(mp)
	}
}

// CurrentCP returns a player's CP, 0 for NPCs.
func (b *skillBase) CurrentCP() float64 {
	if b.p != nil {
		return b.p.CurCP
	}
	return 0
}

// SetCurrentCP sets a player's CP.
func (b *skillBase) SetCurrentCP(cp float64) {
	if b.p != nil {
		b.p.SetCurrentCP(cp)
	}
}

// MaxHP returns the maximum HP.
func (b *skillBase) MaxHP() int { return int(b.gc().MaxHP()) }

// MaxMP returns the maximum MP.
func (b *skillBase) MaxMP() int {
	if b.p != nil {
		return b.p.stats.MaxMP()
	}
	return int(b.n.MaxMP())
}

// MaxCP returns a player's maximum CP, 0 for NPCs.
func (b *skillBase) MaxCP() int {
	if b.p != nil {
		return b.p.stats.MaxCP()
	}
	return 0
}

// MaxRecoverableHP returns where HP regeneration stops.
func (b *skillBase) MaxRecoverableHP() int {
	if b.p != nil {
		return b.p.stats.MaxRecoverableHP()
	}
	return b.n.Stats.MaxRecoverableHP()
}

// MaxRecoverableMP returns where MP regeneration stops.
func (b *skillBase) MaxRecoverableMP() int {
	if b.p != nil {
		return b.p.stats.MaxRecoverableMP()
	}
	return b.n.Stats.MaxRecoverableMP()
}

// MaxRecoverableCP returns where CP regeneration stops, 0 for NPCs.
func (b *skillBase) MaxRecoverableCP() int {
	if b.p != nil {
		return b.p.stats.MaxRecoverableCP()
	}
	return 0
}

// Calc runs the stat chain with target and sk as the context.
func (b *skillBase) Calc(st stats.Stat, init float64, target skills.Creature, sk *skills.Skill) float64 {
	ctx := skillContext(target, sk)
	if b.p != nil {
		return b.p.stats.CalcFor(st, init, ctx)
	}
	return b.n.Stats.CalcFor(st, init, ctx)
}

// AddModifiers adds stat modifiers and shows the new numbers.
func (b *skillBase) AddModifiers(mods ...stats.Modifier) error {
	var err error
	if b.p != nil {
		err = b.p.stats.AddModifiers(mods...)
	} else {
		err = b.n.Stats.AddModifiers(mods...)
	}
	if len(mods) > 0 {
		b.statsChanged()
	}
	return err
}

// RemoveOwner removes an owner's modifiers and shows the new numbers.
func (b *skillBase) RemoveOwner(o stats.Owner) int {
	var removed int
	if b.p != nil {
		removed = b.p.stats.RemoveOwner(o)
	} else {
		removed = b.n.Stats.RemoveOwner(o)
	}
	if removed > 0 {
		b.statsChanged()
	}
	return removed
}

// statsChanged shows changed numbers (Mobius broadcastModifiedStats): a
// player's UserInfo and CharInfo at the end of the tick when they changed,
// an NPC's NpcInfo when its packet numbers changed.
func (b *skillBase) statsChanged() {
	if b.p != nil {
		b.p.RefreshStats()
		return
	}
	n := b.n
	if !n.InWorld() || !n.cacheOK {
		return
	}
	old := n.cache.view
	if v := n.statView(); !v.Equal(old) {
		n.BroadcastInfo()
	}
}

// Traits returns the trait arrays.
func (b *skillBase) Traits() *skills.Traits { return &b.traits }

// baseStatOf maps a data base stat to the stat of its modifiers.
func baseStatOf(bs data.BaseStat) (stats.Stat, bool) {
	switch bs {
	case data.BaseSTR:
		return stats.STR, true
	case data.BaseINT:
		return stats.INT, true
	case data.BaseDEX:
		return stats.DEX, true
	case data.BaseWIT:
		return stats.WIT, true
	case data.BaseCON:
		return stats.CON, true
	case data.BaseMEN:
		return stats.MEN, true
	}
	return 0, false
}

// BaseStat returns STR, INT, ... (getSTR, ...); 0 for data.BaseNone.
func (b *skillBase) BaseStat(bs data.BaseStat) int {
	st, ok := baseStatOf(bs)
	if !ok {
		return 0
	}
	get := func(str, in, dex, wit, con, men func() int) int {
		switch st {
		case stats.STR:
			return str()
		case stats.INT:
			return in()
		case stats.DEX:
			return dex()
		case stats.WIT:
			return wit()
		case stats.CON:
			return con()
		}
		return men()
	}
	if b.p != nil {
		s := b.p.stats
		return get(s.STR, s.INT, s.DEX, s.WIT, s.CON, s.MEN)
	}
	s := b.n.Stats
	return get(s.STR, s.INT, s.DEX, s.WIT, s.CON, s.MEN)
}

// StatBonus returns a base stat's statBonus.xml multiplier, 1 for
// data.BaseNone.
func (b *skillBase) StatBonus(bs data.BaseStat) float64 {
	st, ok := baseStatOf(bs)
	if !ok {
		return 1
	}
	if b.p != nil {
		return b.p.stats.StatBonus(st)
	}
	return b.n.Stats.StatBonus(st)
}

// PAtk is getPAtk(target).
func (b *skillBase) PAtk(target skills.Creature) float64 {
	ctx := skillContext(target, nil)
	if b.p != nil {
		return b.p.stats.PAtkFor(ctx)
	}
	return b.n.Stats.PAtkFor(ctx)
}

// PDef is getPDef(target).
func (b *skillBase) PDef(target skills.Creature) float64 {
	ctx := skillContext(target, nil)
	if b.p != nil {
		return b.p.stats.PDefFor(ctx)
	}
	return b.n.Stats.PDefFor(ctx)
}

// MAtk is getMAtk(target, skill), capped at MaxMAtk.
func (b *skillBase) MAtk(target skills.Creature, sk *skills.Skill) float64 {
	ctx := skillContext(target, sk)
	limit := float64(b.srv.rules.Config().MaxMAtk)
	if b.p != nil {
		return math.Min(b.p.stats.CalcFor(stats.MAtk, float64(b.p.stats.Template().MAtk), ctx), limit)
	}
	return math.Min(b.n.Stats.CalcFor(stats.MAtk, float64(javaInt32(b.n.Stats.Template().MAtk)), ctx), limit)
}

// MDef is getMDef(target, skill).
func (b *skillBase) MDef(target skills.Creature, sk *skills.Skill) float64 {
	ctx := skillContext(target, sk)
	if b.p != nil {
		d := b.p.stats.Template().MDef
		base := d.REar + d.LEar + d.RFinger + d.RFinger + d.Neck // PlayerTemplateData counts rfinger twice
		return b.p.stats.CalcFor(stats.MDef, float64(base), ctx)
	}
	return b.n.Stats.CalcFor(stats.MDef, float64(javaInt32(b.n.Stats.Template().MDef)), ctx)
}

// PAtkSpd returns the attack speed.
func (b *skillBase) PAtkSpd() float64 {
	if b.p != nil {
		return b.p.stats.PAtkSpd()
	}
	return b.n.Stats.PAtkSpd()
}

// MAtkSpd returns the casting speed.
func (b *skillBase) MAtkSpd() int {
	if b.p != nil {
		return b.p.stats.MAtkSpd()
	}
	return b.n.Stats.MAtkSpd()
}

// MCriticalHit is getMCriticalHit(target, skill): the mCritRate chain on 1,
// cut, times 10, capped at MaxMCritRate.
func (b *skillBase) MCriticalHit(target skills.Creature, sk *skills.Skill) int {
	v := int(javaInt32(b.Calc(stats.MCritRate, 1, target, sk))) * 10
	return min(v, maxMCritRate)
}

// ShieldDef returns the shield defence.
func (b *skillBase) ShieldDef() int {
	if b.p != nil {
		return b.p.stats.ShieldDef()
	}
	return b.n.Stats.ShieldDef()
}

// RandomDamageMultiplier is getRandomDamageMultiplier, drawing from r.
func (b *skillBase) RandomDamageMultiplier(r stats.Rand) float64 {
	c := b.gc()
	f := b.srv.fighter(c, b.srv.activeWeaponItem(c))
	return b.srv.combat.RandomDamageMultiplier(r, &f)
}

// BreakState tells what a hit could break.
func (b *skillBase) BreakState() stats.BreakState {
	_, st := b.srv.breakTarget(b.gc())
	st.Casting = b.IsCastingNow()
	st.Invul = b.IsInvul()
	return st
}

// AttackType is getAttackType: the active weapon's type, else a player's
// fists or the NPC template's attack type.
func (b *skillBase) AttackType() data.WeaponType {
	if w := b.srv.activeWeaponItem(b.gc()); w != nil {
		return w.Weapon.Type
	}
	if b.p != nil {
		return data.WeaponFist
	}
	return b.n.Template.Stats.Attack.Type
}

// ActiveWeapon is getActiveWeaponItem: a player without a weapon holds its
// class fists.
func (b *skillBase) ActiveWeapon() (skills.Weapon, bool) {
	if w := b.srv.activeWeaponItem(b.gc()); w != nil {
		return skills.Weapon{Type: w.Weapon.Type, BodyPart: w.Part()}, true
	}
	if b.p != nil {
		return skills.Weapon{Type: data.WeaponFist, BodyPart: data.BodyPartRHand}, true
	}
	return skills.Weapon{}, false
}

// HasShield reports a shield in the left hand.
func (b *skillBase) HasShield() bool {
	if b.p != nil {
		return playerHasShield(b.p)
	}
	t, ok := b.srv.data.ItemTemplate(b.n.Template.Equipment.LHand)
	return ok && t.Armor != nil
}

// ChargedShot reports a charged shot.
func (b *skillBase) ChargedShot(t skills.ShotType) bool {
	if b.p != nil {
		return playerChargedShot(b.p, t)
	}
	return int(t) < len(b.shots) && b.shots[t]
}

// SetChargedShot charges or spends a shot: a player's on its weapon.
func (b *skillBase) SetChargedShot(t skills.ShotType, charged bool) {
	if b.p != nil {
		setPlayerChargedShot(b.p, t, charged)
		return
	}
	if int(t) < len(b.shots) {
		b.shots[t] = charged
	}
}

// Effects returns the effect list.
func (b *skillBase) Effects() *skills.EffectList { return b.el }

// SkillLevel returns the level of a known skill, 0 if unknown.
func (b *skillBase) SkillLevel(id int) int { return b.known.level(id) }

// AsPlayer returns the player.
func (a *skillPlayer) AsPlayer() skills.Player { return a }

// AsPlayer returns nil: NPCs are not players.
func (a *skillNpc) AsPlayer() skills.Player { return nil }

// AsAttackable returns nil for a player.
func (a *skillPlayer) AsAttackable() skills.Attackable { return nil }

// AsAttackable returns the NPC for an attackable, nil otherwise.
func (a *skillNpc) AsAttackable() skills.Attackable {
	if a.n.class != nil && a.n.class.attackable {
		return a
	}
	return nil
}

// SendMessage sends a system message to a player.
func (b *skillBase) SendMessage(m serverpackets.SystemMessage) {
	if b.p != nil {
		b.p.Send(serverpackets.NewSystemMessagePacket(m))
	}
}

// SendText sends a text to a player.
func (b *skillBase) SendText(text string) {
	if b.p != nil {
		b.p.SendMessage(text)
	}
}

// SendDamageMessage is Player.sendDamageMessage.
func (b *skillBase) SendDamageMessage(target skills.Creature, damage int, mcrit, pcrit, miss bool) {
	p := b.p
	if p == nil {
		return
	}
	if miss {
		p.SendSystemMessage(sysMsgYouMissed)
		return
	}
	if pcrit {
		p.SendSystemMessage(sysMsgCriticalHit)
	}
	if mcrit {
		p.SendSystemMessage(skills.MsgMagicCriticalHit)
	}
	if target == nil || target.ObjectID() != b.ObjectID() {
		p.SendSystemMessage(sysMsgYouHitForDamage, serverpackets.NumberParam(int32(damage))) //nolint:gosec // damage fits
	}
}

// ReduceCurrentHP takes damage: the effects that end on damage end, then
// the status takes the HP with hate, death and packets. A player takes no
// damage from a ranged skill whose caster cannot see it (Mobius
// reduceCurrentHp, geo_sight.go).
func (b *skillBase) ReduceCurrentHP(damage float64, attacker skills.Creature, sk *skills.Skill) {
	b.el.StopOnDamage(true)
	ga := b.srv.gameCreature(attacker)
	if b.srv.skillUnseen(b.p, ga, sk) {
		damage = 0
	}
	b.gc().ReduceHP(damage, ga)
}

// ReduceCurrentHPByDOT takes the damage of a damage-over-time effect; a
// toggle's damage does not wake.
func (b *skillBase) ReduceCurrentHPByDOT(damage float64, attacker skills.Creature, sk *skills.Skill) {
	b.el.StopOnDamage(sk == nil || !sk.IsToggle())
	if b.p != nil {
		b.p.reduceHP(damage, b.srv.gameCreature(attacker), true) // invulnerability does not stop it
		return
	}
	if b.n != nil {
		b.n.reduceHP(damage, b.srv.gameCreature(attacker), true)
		return
	}
	b.gc().ReduceHP(damage, b.srv.gameCreature(attacker))
}

// NotifyDamageReceived runs the damage listeners.
func (b *skillBase) NotifyDamageReceived(damage float64, attacker skills.Creature, sk *skills.Skill, crit, dot bool) {
	ev := skills.DamageEvent{Attacker: attacker, Target: b.self, Damage: damage, Skill: sk, Crit: crit, DOT: dot}
	for _, l := range slices.Clone(b.listeners) {
		l.fn(ev)
	}
}

// AddDamageListener adds a damage listener.
func (b *skillBase) AddDamageListener(owner any, fn func(skills.DamageEvent)) {
	b.listeners = append(b.listeners, skillDamageListener{owner: owner, fn: fn})
}

// RemoveDamageListeners removes the listeners of owner.
func (b *skillBase) RemoveDamageListeners(owner any) {
	b.listeners = slices.DeleteFunc(b.listeners, func(l skillDamageListener) bool { return l.owner == owner })
}

// BreakAttack interrupts a swing.
func (b *skillBase) BreakAttack() { b.srv.BreakAttack(b.gc()) }

// AbortAttack stops a swing.
func (b *skillBase) AbortAttack() { b.srv.AbortAttack(b.gc()) }

// BreakCast interrupts a cast (Phase 7.3, SetSkillCaster).
func (b *skillBase) BreakCast() {
	if c := b.srv.skills.caster; c != nil {
		c.BreakCast(b.self)
	}
}

// AbortCast stops a cast (Phase 7.3, SetSkillCaster).
func (b *skillBase) AbortCast() {
	if c := b.srv.skills.caster; c != nil {
		c.AbortCast(b.self)
	}
}

// IsCastingNow reports a cast in progress (Phase 7.3).
func (b *skillBase) IsCastingNow() bool {
	c := b.srv.skills.caster
	return c != nil && c.IsCastingNow(b.self)
}

// CanAbortCast reports a cast still before its interrupt time (Phase 7.3).
func (b *skillBase) CanAbortCast() bool {
	c := b.srv.skills.caster
	return c != nil && c.CanAbortCast(b.self)
}

// MakeTriggerCast casts a triggered skill (Phase 7.3).
func (b *skillBase) MakeTriggerCast(sk *skills.Skill, target skills.Creature, ignoreTargetType bool) {
	if c := b.srv.skills.caster; c != nil {
		c.TriggerCast(b.self, sk, target, ignoreTargetType)
	}
}

// StopMove stops a moving creature where it stands.
func (b *skillBase) StopMove() { b.srv.stopCreature(b.gc()) }

// DoDie kills the creature.
func (b *skillBase) DoDie(killer skills.Creature) { b.srv.Kill(b.gc(), b.srv.gameCreature(killer)) }

// Escape sends a player to the nearest town (castles, clan halls and siege
// flags do not exist yet, and Mobius falls back to the town without them).
// Guards are not sent anywhere yet.
func (b *skillBase) Escape(skills.TeleportWhere) {
	p := b.p
	if p == nil {
		return
	}
	if loc, ok := b.srv.restartLocation(p); ok {
		b.srv.teleportAround(p, loc, maxOffsetOnTeleport)
	}
}

// UpdateAbnormalEffect shows the visual masks: UserInfo and CharInfo, or
// NpcInfo.
func (b *skillBase) UpdateAbnormalEffect() {
	if b.p != nil {
		if b.p.InGame() {
			b.p.BroadcastUserInfo()
		}
		return
	}
	if b.n.InWorld() {
		b.n.BroadcastInfo()
	}
}

// IsLethalable reports whether lethal strikes work: on players, and on NPCs
// that are not undying, raid bosses or grand bosses (their constructors call
// setLethalable(false)), or made not lethalable by a script.
func (b *skillBase) IsLethalable() bool {
	return b.n == nil || (!b.n.Template.Status.Undying && (b.n.class == nil || !b.n.class.raid) && !b.n.script.notLethalable)
}

// SetTarget changes the target.
func (b *skillBase) SetTarget(t skills.Creature) {
	var o world.Object
	if gc := b.srv.gameCreature(t); gc != nil {
		o = gc
	}
	if b.p != nil {
		b.srv.SetTarget(b.p, o)
		return
	}
	b.n.ai.target = o
}

// SetImmobilized blocks or frees movement.
func (b *skillBase) SetImmobilized(v bool) { b.immobilized = v }

// IsMovementDisabled reports a creature that cannot move: dead,
// immobilized, stunned, rooted, sleeping or paralyzed (also by //para).
func (b *skillBase) IsMovementDisabled() bool {
	if b.immobilized || b.admin.paralyzed || b.IsAlikeDead() {
		return true
	}
	for _, f := range [...]skills.EffectFlag{skills.FlagStunned, skills.FlagRooted, skills.FlagSleep, skills.FlagParalyzed} {
		if b.el.IsAffected(f) {
			return true
		}
	}
	return false
}

// NotifyAI tells the AI what an effect did. The monster AI hears attacks and
// aggression; a player's intention follows attack and idle orders. The
// other events wait for the AI of Milestone 10 and Phase 7.4.
func (b *skillBase) NotifyAI(ev skills.AIEvent, other skills.Creature) {
	o := b.srv.gameCreature(other)
	switch {
	case b.n != nil && ev == skills.AIAttacked && o != nil:
		b.srv.NotifyAttacked(b.n, o)
	case b.n != nil && ev == skills.AIAttack && o != nil:
		b.srv.NotifyAggression(b.n, o, 1)
	case b.n != nil && (ev == skills.AIAfraidStart || ev == skills.AIAfraid):
		b.srv.npcAfraid(b.n, o, ev == skills.AIAfraidStart) // ai_fear.go
	case b.p != nil && ev == skills.AIAttack && o != nil:
		b.srv.AttackIntention(b.p, o)
	case b.p != nil && ev == skills.AIIdle:
		b.srv.IdleIntention(b.p)
	}
}

// Decay removes a dead NPC's corpse after delay.
func (b *skillBase) Decay(delay time.Duration) {
	n := b.n
	if n == nil {
		return
	}
	s := b.srv
	s.CancelNpcDecay(n)
	if delay <= 0 {
		s.EndNpcDecay(n)
		return
	}
	s.loop.After(delay, func() {
		if n.Dead {
			s.EndNpcDecay(n)
		}
	})
}

// Revive brings a dead creature back: a player gets percent of its lost exp
// back and revives (Mobius doRevive(percent)); an NPC's corpse stops
// decaying and it gets its HP back.
func (b *skillBase) Revive(percent float64) {
	s := b.srv
	if p := b.p; p != nil {
		if p.dead {
			s.RestoreExp(p, percent)
			s.RevivePlayer(p)
		}
		return
	}
	n := b.n
	if !n.Dead {
		return
	}
	s.CancelNpcDecay(n)
	n.Revive()
	n.SetCurrentHPMP(float64(n.MaxHP()), float64(n.MaxMP()))
}

// SetRunning makes the creature run.
func (b *skillBase) SetRunning() { b.setRunning(true) }

// SetWalking makes the creature walk.
func (b *skillBase) SetWalking() { b.setRunning(false) }

func (b *skillBase) setRunning(run bool) {
	if b.p != nil {
		b.srv.setRunning(b.p, run)
		return
	}
	b.srv.SetNpcRunning(b.n, run)
}

// SetLockedTarget locks a player's target.
func (b *skillBase) SetLockedTarget(t skills.Creature) {
	if b.p != nil {
		b.locked = t
	}
}

// TeleToLocation teleports the creature.
func (b *skillBase) TeleToLocation(loc skills.Location, randomOffset bool) {
	pos := world.Position{X: int32(loc.X), Y: int32(loc.Y), Z: int32(loc.Z)} //nolint:gosec // world coordinates
	if p := b.p; p != nil {
		offset := 0
		if randomOffset {
			offset = maxOffsetOnTeleport
		}
		b.srv.teleportAround(p, data.Location{X: loc.X, Y: loc.Y, Z: loc.Z}, offset)
		return
	}
	b.srv.TeleportNpc(b.n, pos)
}

// SetXYZ puts the creature at loc without a teleport.
func (b *skillBase) SetXYZ(loc skills.Location) {
	c := b.gc()
	pos := c.Position()
	pos.X, pos.Y, pos.Z = int32(loc.X), int32(loc.Y), int32(loc.Z) //nolint:gosec // world coordinates
	if err := b.srv.MoveObject(c, pos); err != nil {
		b.srv.log.Debug("game: skills: SetXYZ", "objectId", c.ID(), "err", err)
	}
}

// BroadcastValidateLocation shows the creature where it is.
func (b *skillBase) BroadcastValidateLocation() {
	c := b.gc()
	b.srv.world.Broadcast(c, validateLocation(c), true)
}

// BroadcastStartRotation does nothing yet: the rotation packets come with
// the effects that use them (Phase 7.4).
func (b *skillBase) BroadcastStartRotation(_, _, _ int) {}

// BroadcastStopRotation does nothing yet (Phase 7.4).
func (b *skillBase) BroadcastStopRotation(_, _ int) {}

// StartFakeDeath starts a player's fake death. Its packets come with the
// FakeDeath effect (Phase 7.4).
func (b *skillBase) StartFakeDeath() {
	if a, ok := b.self.(*skillPlayer); ok {
		a.fakeDeath = true
	}
}

// StopFakeDeath ends a player's fake death, with its effects when asked.
func (b *skillBase) StopFakeDeath(removeEffects bool) {
	a, ok := b.self.(*skillPlayer)
	if !ok || !a.fakeDeath {
		return
	}
	if removeEffects {
		b.el.StopEffectType(skills.TypeFakeDeath)
	}
	a.fakeDeath, a.recentFakeDeath = false, true
}

// BroadcastFakeDeathEnd does nothing yet (Phase 7.4).
func (b *skillBase) BroadcastFakeDeathEnd() {}

// SetCollisionGrown does nothing yet: the grown radius comes with the Grow
// effect (Phase 7.4).
func (b *skillBase) SetCollisionGrown(bool) {}

// npcInsideZone reports whether an NPC stands in a zone of the kind.
func (s *Server) npcInsideZone(n *Npc, kind ZoneFlag) bool {
	var typ string
	switch kind {
	case ZonePeace:
		typ = "PeaceZone"
	case ZonePvP:
		typ = "ArenaZone"
	case ZoneSiege:
		typ = "SiegeZone"
	default:
		return false
	}
	return s.zoneAt(n.Position(), typ) != nil
}

// The skills.Player methods.

// HasEnteredWorld reports that EnterWorld has finished.
func (a *skillPlayer) HasEnteredWorld() bool { return a.entered }

// IsGM reports a GM.
func (a *skillPlayer) IsGM() bool { return a.p.IsGM() }

// CanGiveDamage is true: the GM access levels of the config all deal
// damage.
func (a *skillPlayer) CanGiveDamage() bool { return true }

// IsMageClass reports a mystic class.
func (a *skillPlayer) IsMageClass() bool { return data.PlayerClass(a.p.ClassID()).IsMage() } //nolint:gosec // class IDs are small

// ClassID returns the active class.
func (a *skillPlayer) ClassID() int { return a.p.ClassID() }

// IsFemale reports a female character.
func (a *skillPlayer) IsFemale() bool { return a.p.Female() }

// IsSitting reports sitting.
func (a *skillPlayer) IsSitting() bool { return a.p.Sitting }

// Karma returns the karma.
func (a *skillPlayer) Karma() int { return int(a.p.char.Karma) }

// PkKills returns the PK count.
func (a *skillPlayer) PkKills() int { return int(a.p.char.PKKills) }

// PvpFlag returns the PvP flag.
func (a *skillPlayer) PvpFlag() int { return int(a.p.PvPFlag) }

// IsHero reports the hero status; until the Olympiad only //sethero gives
// it (admin_char.go).
func (a *skillPlayer) IsHero() bool { return a.p.IsHero() }

// InOlympiad is false: the Olympiad does not exist yet.
func (a *skillPlayer) InOlympiad() bool { return false }

// Charges returns the force charges.
func (a *skillPlayer) Charges() int { return a.charges }

// CanEscape is ConditionPlayerCanEscape: duels, events, the Olympiad and
// boss zones do not exist yet, so only flying keeps a player from it.
func (a *skillPlayer) CanEscape() bool { return !a.IsFlying() }

// CurrentLoad returns the weight carried.
func (a *skillPlayer) CurrentLoad() int { return int(a.p.inv.Load()) }

// MaxLoad returns the weight limit.
func (a *skillPlayer) MaxLoad() int { return int(a.p.stats.View().WeightLimit) }

// BonusWeightPenalty is getBonusWeightPenalty: the weightPenalty chain on 1,
// cut.
func (a *skillPlayer) BonusWeightPenalty() int {
	return int(javaInt32(a.p.stats.Calc(stats.WeightPenalty, 1)))
}

// DietMode reports the diet mode of //diet (admin_gm.go).
func (a *skillPlayer) DietMode() bool { return a.p.admin.diet }

// InventoryNonQuestSize returns the number of items in the inventory.
// Quest items have their own limit in Mobius; until quest items are told
// apart there, every item counts.
func (a *skillPlayer) InventoryNonQuestSize() int {
	if a.p.inv.Inventory == nil {
		return 0
	}
	return a.p.inv.Len()
}

// InventoryLimit returns the number of slots.
func (a *skillPlayer) InventoryLimit() int { return int(a.p.stats.View().InventoryLimit) }

// WornMask returns the item type bits of the worn items.
func (a *skillPlayer) WornMask() int {
	if a.p.inv.Inventory == nil {
		return 0
	}
	return a.p.inv.WearedMask()
}

// ArmorMasks returns the item type masks of the chest and legs items.
func (a *skillPlayer) ArmorMasks() (chest, legs int, fullArmor bool) {
	if a.p.inv.Inventory == nil {
		return 0, 0, false
	}
	if it := a.p.inv.Paperdoll(storage.PaperdollChest); it != nil {
		chest = it.Template().Mask()
		fullArmor = it.Template().Part() == data.BodyPartFullArmor
	}
	if it := a.p.inv.Paperdoll(storage.PaperdollLegs); it != nil {
		legs = it.Template().Mask()
	}
	return chest, legs, fullArmor
}

// PaperdollItem returns the item in a paperdoll slot.
func (a *skillPlayer) PaperdollItem(slot int) (itemID, enchant int, ok bool) {
	if a.p.inv.Inventory == nil || slot < 0 || slot >= storage.PaperdollTotalSlots {
		return 0, 0, false
	}
	it := a.p.inv.Paperdoll(storage.PaperdollSlot(slot)) //nolint:gosec // checked above
	if it == nil {
		return 0, 0, false
	}
	return int(it.ItemID()), int(it.Enchant()), true
}

// InObserverMode is false: observing does not exist yet.
func (a *skillPlayer) InObserverMode() bool { return false }

// SiegeState is 0: sieges do not exist yet.
func (a *skillPlayer) SiegeState() int { return 0 }

// SiegeSide is 0.
func (a *skillPlayer) SiegeSide() int { return 0 }

// ClanID returns the clan.
func (a *skillPlayer) ClanID() int { return int(a.p.char.ClanID) }

// AllyID returns the alliance of the clan.
func (a *skillPlayer) AllyID() int { return int(a.p.AllyID()) }

// IsInSiege is false.
func (a *skillPlayer) IsInSiege() bool { return false }

// IsFestivalParticipant is false.
func (a *skillPlayer) IsFestivalParticipant() bool { return false }

// CheckPvPSkill is Player.checkPvpSkill against another player
// (cast_player.go).
func (a *skillPlayer) CheckPvPSkill(target skills.Player, sk *skills.Skill) bool {
	t, ok := target.(*skillPlayer)
	if !ok {
		return true
	}
	return a.srv.checkPvPSkill(a.p, t.p, sk)
}

// SendEffectIcons sends AbnormalStatusUpdate, and PartySpelled to the
// party (party_window.go); the Olympiad does not exist yet.
func (a *skillPlayer) SendEffectIcons(u skills.IconUpdate) {
	if !a.p.InGame() {
		return
	}
	if u.SendAbnormal {
		a.p.Send(serverpackets.NewAbnormalStatusUpdatePacket(u.Icons))
	}
	a.srv.partySpelled(a.p, u.Party)
}

// SendShortBuff sends ShortBuffStatusUpdate.
func (a *skillPlayer) SendShortBuff(sb skills.ShortBuff) {
	if a.p.InGame() {
		a.p.Send(serverpackets.NewShortBuffStatusUpdatePacket(sb))
	}
}

// SendRegenMax sends ExRegenMax.
func (a *skillPlayer) SendRegenMax(duration, ticks int, amount float64) {
	if a.p.InGame() {
		a.p.Send(exRegenMaxPacket(duration, ticks, amount))
	}
}

// IncreaseCharges adds force charges up to max and shows them; the charge
// messages come with the effects that use them (Phase 7.4).
func (a *skillPlayer) IncreaseCharges(count, maxCharges int) {
	if a.charges >= maxCharges {
		return
	}
	a.charges = min(a.charges+count, maxCharges)
	if a.p.InGame() {
		a.p.Send(etcStatusPacket(a.p))
	}
}

// ReviveRequest does nothing yet: ConfirmDlg comes with Phase 7.4.
func (a *skillPlayer) ReviveRequest(skills.Creature, bool, int) {}

// AddItem gives the player an item (Player.addItem).
func (a *skillPlayer) AddItem(itemID, count, enchant int, _ skills.Creature, sendMessage bool) (skills.CreatedItem, bool) {
	if count <= 0 {
		return skills.CreatedItem{}, false
	}
	it, err := a.srv.AddItem(a.p, int32(itemID), int64(count), int32(max(enchant, 0)), sendMessage, false) //nolint:gosec // item IDs and counts are small
	if err != nil || it == nil {
		return skills.CreatedItem{}, false
	}
	return skills.CreatedItem{ObjectID: it.ObjectID(), ItemID: int(it.ItemID()), Stackable: it.Stackable(), Enchant: int(it.Enchant())}, true
}

// SendItemsUpdate sends the pending inventory changes.
func (a *skillPlayer) SendItemsUpdate([]int) { a.srv.ItemsChanged(a.p) }

// SetHairStyle changes the hair style.
func (a *skillPlayer) SetHairStyle(v int) { a.p.char.HairStyle = int32(v) } //nolint:gosec // small

// SetHairColor changes the hair color.
func (a *skillPlayer) SetHairColor(v int) { a.p.char.HairColor = int32(v) } //nolint:gosec // small

// SetFace changes the face.
func (a *skillPlayer) SetFace(v int) { a.p.char.Face = int32(v) } //nolint:gosec // small

// BroadcastUserInfo sends UserInfo and CharInfo.
func (a *skillPlayer) BroadcastUserInfo() { a.p.BroadcastUserInfo() }

// AddExpAndSp gives experience and SP.
func (a *skillPlayer) AddExpAndSp(exp, sp float64) {
	a.srv.AddExpAndSP(a.p, stats.Reward{Exp: int64(exp), SP: int64(sp), ShownExp: int64(exp), ShownSP: int64(sp)})
}

// ReduceDeathPenaltyBuffLevel lowers the death penalty by a level.
func (a *skillPlayer) ReduceDeathPenaltyBuffLevel() { a.srv.ReduceDeathPenalty(a.p) }

// AddElementSeed does nothing yet (Phase 7.4).
func (a *skillPlayer) AddElementSeed(data.SeedElement) {}

// IsFakeDeath reports fake death.
func (a *skillPlayer) IsFakeDeath() bool { return a.fakeDeath }

// SetFakeDeath sets the fake death flag.
func (a *skillPlayer) SetFakeDeath(v bool) { a.fakeDeath = v }

// SetRecentFakeDeath sets the recent fake death flag.
func (a *skillPlayer) SetRecentFakeDeath(v bool) { a.recentFakeDeath = v }

// SitDown sits the player down.
func (a *skillPlayer) SitDown(bool) {
	if !a.p.Sitting {
		a.srv.sitDown(a.p)
	}
}

// CheckInventorySlotsAndWeight reports whether the items fit: one slot per
// item and the weight limit (PlayerInventory.checkInventorySlotsAndWeight).
func (a *skillPlayer) CheckInventorySlotsAndWeight(itemIDs []int, _, _ bool) bool {
	weight, slots := 0, 0
	for _, id := range itemIDs {
		t, ok := a.srv.data.ItemTemplate(id)
		if !ok {
			continue
		}
		weight += t.Weight
		if !t.Stackable || a.p.inv.Inventory == nil || a.p.inv.Count(int32(id), -1, true) == 0 { //nolint:gosec // item IDs are small
			slots++
		}
	}
	return a.InventoryNonQuestSize()+slots <= a.InventoryLimit() && a.CurrentLoad()+weight <= a.MaxLoad()
}

// PartyDistributeItem hands the item to the player's party by its loot
// mode (Party.distributeItem, party_reward.go); without a party the player
// gets it.
func (a *skillPlayer) PartyDistributeItem(item skills.ItemHolder, spoil bool, from skills.Creature) {
	if item.Count <= 0 {
		return
	}
	if a.p.party.party != nil {
		a.srv.partyDistributeItem(a.p, int32(item.ID), item.Count, spoil, a.srv.gameCreature(from)) //nolint:gosec // item IDs are small
		return
	}
	a.AddItem(item.ID, int(min(item.Count, math.MaxInt32)), 0, from, true)
}

// The skills.Attackable methods.

// AddDamageHate adds damage and hate.
func (a *skillNpc) AddDamageHate(attacker skills.Creature, damage, aggro int64) {
	if c := a.srv.gameCreature(attacker); c != nil {
		a.n.AddDamageHate(c, damage, aggro)
	}
}

// ReduceHate lowers the hate for target, not below 0.
func (a *skillNpc) ReduceHate(target skills.Creature, amount int64) {
	if e := a.n.aggroOf(a.srv.gameCreature(target)); e != nil {
		e.Hate = max(e.Hate-amount, 0)
	}
}

// StopHating sets the hate for target to 0.
func (a *skillNpc) StopHating(target skills.Creature) {
	if c := a.srv.gameCreature(target); c != nil {
		a.n.StopHating(c)
	}
}

// ClearAggroList empties the hate list.
func (a *skillNpc) ClearAggroList() { a.n.ClearAggroList() }

// Hating returns the hate for target.
func (a *skillNpc) Hating(target skills.Creature) int64 {
	c := a.srv.gameCreature(target)
	if c == nil {
		return 0
	}
	return a.n.Hating(c)
}

// AggroList returns the creatures on the hate list.
func (a *skillNpc) AggroList() []skills.Creature {
	var out []skills.Creature
	for _, e := range a.n.ai.hate {
		if c := a.srv.skillCreatureOf(e.Attacker); c != nil {
			out = append(out, c)
		}
	}
	return out
}

// SetFearTime sets how long the AI runs away, in seconds (AttackableAI
// setFearTime; ai_fear.go).
func (a *skillNpc) SetFearTime(t int) { a.fearTime = t }

// ClanHelpRange returns the template's clan help range.
func (a *skillNpc) ClanHelpRange() (int, bool) {
	ai := &a.n.Template.AI
	return ai.ClanHelpRange, len(ai.Clans) > 0
}

// IsClan reports whether this NPC shares a clan with of.
func (a *skillNpc) IsClan(of skills.Attackable) bool {
	o, ok := of.(*skillNpc)
	if !ok {
		return false
	}
	mine := a.n.Template.AI.Clans
	return slices.ContainsFunc(o.n.Template.AI.Clans, func(c string) bool { return slices.Contains(mine, c) })
}

// IsSpoiled reports a spoiled monster.
func (a *skillNpc) IsSpoiled() bool { return a.spoiler != 0 }

// SetSpoiler spoils the monster for a player.
func (a *skillNpc) SetSpoiler(objectID int32) { a.spoiler = objectID }

// CheckSpoilOwner reports whether sweeper spoiled the monster; the
// sweeping rules come with Phase 7.4.
func (a *skillNpc) CheckSpoilOwner(sweeper skills.Creature, _ bool) bool {
	return sweeper != nil && sweeper.ObjectID() == a.spoiler
}

// SpoilLootItems returns nil: spoil loot comes with Phase 7.4.
func (a *skillNpc) SpoilLootItems() []int { return nil }

// TakeSweep returns nil: spoil loot comes with Phase 7.4.
func (a *skillNpc) TakeSweep() []skills.ItemHolder { return nil }

// StartDistrust does nothing yet (Phase 7.4).
func (a *skillNpc) StartDistrust(skills.Creature) {}

// StopDistrust does nothing yet (Phase 7.4).
func (a *skillNpc) StopDistrust() {}
