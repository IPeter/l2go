package game

import (
	"math"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/world"
)

// Doors in a fight (Mobius Door as a Creature, DoorStatus, DoorStat and
// DoorAI): a door has HP, takes physical and magic damage, regenerates, dies
// and is revived.
//
//   - Who may attack a door is Door.isAutoAttackable (autoAttackableBy,
//     door.go): an attackable door by any player, a door that shows its HP
//     during its castle's siege by everyone but the owner clan (and the clan
//     hall sieges, siege_zone.go). A click attacks such a door (DoorClick);
//     Ctrl+click attacks a door for GMs only (canBeAttacked is false for
//     doors). Skills hit a door only inside an active siege zone
//     (Player.useMagic's door rule, cast_player.go).
//   - Damage goes through the status: a wall outside instances takes damage
//     only from siege weapon servitors (Door.reduceCurrentHp), damage over
//     time never; under 0.5 HP the door dies.
//   - A dead door is broken: it keeps its open state but no longer blocks
//     walking or sight (DoorData.checkIfDoorsBetween skips dead doors), its
//     viewers see it at 0 HP, and during a siege they read "The castle gate
//     has been broken down.". The castles revive their doors at the siege's
//     start and end, at half HP after a mid victory (Castle.spawnDoor).
//   - Defence: the door calculators (Formulas.getStdDoorCalculators) have no
//     level or stat bonus; P.Def and M.Def follow the Seal of Strife
//     (FuncGatesPDefMod, FuncGatesMDefMod). The maximum HP is the template's
//     times the castle's door upgrade (DoorStat).
//   - DoorAI: an attacked door calls the Defenders around it.

// Door fight constants (Mobius).
const (
	// doorRegenPeriod is Formulas.getRegeneratePeriod for a door: 100 ×
	// HP_REGENERATE_PERIOD (3 s).
	doorRegenPeriod = 300 * time.Second
	// doorClickMaxZ is the height difference within which a click attacks
	// a door (DoorClick).
	doorClickMaxZ = 400
	// doorDefenderAggro is the aggression DoorAI gives the Defenders.
	doorDefenderAggro = 15
	// doorMinCollision is DoorData.insertCollisionData's smallest radius.
	doorMinCollision = 20
	// sysMsgCastleGateBroken is "The castle gate has been broken down.".
	sysMsgCastleGateBroken serverpackets.SystemMessageID = 288
)

// The Seal of Strife's multipliers of door defence (Feature.ini
// AltDawnGatesPdefMult, AltDuskGatesPdefMult, AltDawnGatesMdefMult,
// AltDuskGatesMdefMult; the config has no keys for them).
const (
	doorDawnDefMult = 1.1
	doorDuskDefMult = 0.8
)

// The Seven Signs cabals as sevenSignsSealOwner returns them.
const (
	doorCabalDusk = 1
	doorCabalDawn = 2
)

// doorCombat is a door's fight state (Door.combat). Loop-only.
type doorCombat struct {
	// dead is Creature._isDead: a broken door.
	dead bool
	// hpRatio is DoorStat._upgradeHpRatio; 0 means 1.
	hpRatio int32
	// regen is the regeneration task, 0 without.
	regen world.TimerID
	// skill is the door as the skill engine sees it, made on first use.
	skill *skillDoor
}

var _ Creature = (*Door)(nil)

// IsDead reports whether the door is broken (Creature).
func (d *Door) IsDead() bool { return d.combat.dead }

// CurrentHP returns the current HP (Creature).
func (d *Door) CurrentHP() float64 { return d.curHP }

// ReduceHP takes damage (Mobius Door.reduceCurrentHp through
// CreatureStatus.reduceHp): see reduceDoorHP. Loop-only.
func (d *Door) ReduceHP(value float64, attacker Creature) {
	if d.srv != nil {
		d.srv.reduceDoorHP(d, value, attacker)
	}
}

// HPRatio returns the door's upgrade (DoorStat.getUpgradeHpRatio): its
// maximum HP are the template's times it.
func (d *Door) HPRatio() int32 { return max(d.combat.hpRatio, 1) }

// setHPRatio is DoorStat.setUpgradeHpRatio: the maximum HP become the
// template's (cut to an int) times ratio. The current HP stay.
func (d *Door) setHPRatio(ratio int32) {
	d.combat.hpRatio = max(ratio, 1)
	v := int64(truncHP(d.Template.HP)) * int64(d.combat.hpRatio)
	d.maxHP = int32(min(v, math.MaxInt32))
}

// collisionRadius is DoorData.insertCollisionData's radius: the smaller
// side of the first edge, at least 20.
func (d *Door) collisionRadius() int {
	n := d.Template.Nodes
	dx, dy := n[0].X-n[1].X, n[0].Y-n[1].Y
	return max(min(absInt(dx), absInt(dy)), doorMinCollision)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// isEnemy is Door.isEnemy, DoorStatusUpdate's flag: a door that shows its HP
// in an active siege zone.
func (d *Door) isEnemy() bool {
	return d.srv != nil && d.Template.ShowHP && d.srv.siegeAt(d.Position()) != nil
}

// ----- status -----

// reduceDoorHP is Door.reduceCurrentHp and CreatureStatus.reduceHp: a wall
// outside instances takes damage only from a siege weapon servitor; a dead
// door takes none; the HP go down to 0 and under 0.5 the door dies.
// Doors are never invulnerable, and every GM access level deals damage.
func (s *Server) reduceDoorHP(d *Door, value float64, attacker Creature) {
	if d.Template.Wall && d.InstanceID() == 0 && !siegeWeaponServitor(attacker) {
		return
	}
	if d.combat.dead {
		return
	}
	if value > 0 {
		s.setDoorHP(d, max(d.curHP-value, 0), true)
	}
	if d.curHP < 0.5 {
		s.doorDie(d, attacker)
	}
}

// siegeWeaponServitor reports a servitor of the SIEGE_WEAPON race (the
// siege golems and cannons), the only attackers that hurt walls.
func siegeWeaponServitor(c Creature) bool {
	n, ok := c.(*Npc)
	return ok && n.summon != nil && n.summon.pet == nil && n.Template.Race == data.RaceSiegeWeapon
}

// setDoorHP is CreatureStatus.setCurrentHp for a door: nothing on a dead
// door; at most the maximum, which stops the regeneration, below it the
// regeneration starts; viewers see a change (broadcastStatusUpdate) when
// broadcast is set. It reports whether the HP changed, comparing the old
// value cut to an int with the new one, as Mobius does.
func (s *Server) setDoorHP(d *Door, hp float64, broadcast bool) bool {
	if d.combat.dead {
		return false
	}
	old := float64(javaInt32(d.curHP))
	if maxHP := float64(d.maxHP); hp >= maxHP {
		d.curHP = maxHP
		s.stopDoorRegen(d)
	} else {
		d.curHP = max(hp, 0)
		s.startDoorRegen(d)
	}
	changed := old != d.curHP
	if changed && broadcast {
		s.broadcastDoor(d)
	}
	return changed
}

// startDoorRegen starts the regeneration task unless it runs or the door is
// dead (startHpMpRegeneration).
func (s *Server) startDoorRegen(d *Door) {
	if d.combat.regen != 0 || d.combat.dead {
		return
	}
	d.combat.regen = s.loop.Every(doorRegenPeriod, func() { s.doorRegenerate(d) })
}

// stopDoorRegen stops the regeneration task.
func (s *Server) stopDoorRegen(d *Door) {
	if d.combat.regen != 0 {
		s.loop.Cancel(d.combat.regen)
		d.combat.regen = 0
	}
}

// doorRegenerate is one tick of CreatureStatus.doRegeneration: a door's base
// HP regeneration is 0, so it gets Math.max(1, 0) × HPRegenMultiplier.
func (s *Server) doorRegenerate(d *Door) {
	if d.combat.dead || d.curHP >= float64(d.maxHP) {
		s.stopDoorRegen(d)
		return
	}
	s.setDoorHP(d, d.curHP+s.combat.Config().HPRegenMultiplier, true)
}

// doorDie is Door.doDie: the HP go to 0, the door is dead (broken) and
// passable, its viewers see it, and during a siege of its castle (or clan
// hall) they read "The castle gate has been broken down.".
func (s *Server) doorDie(d *Door, _ Creature) {
	if d.combat.dead {
		return
	}
	s.setDoorHP(d, 0, true)
	d.combat.dead = true
	s.stopDoorRegen(d)
	s.broadcastDoor(d)
	s.GeoBlockersChanged() // pathing_door.go: a broken door no longer blocks
	if s.siegeAt(d.Position()) != nil {
		s.world.Broadcast(d, clanSysMsg(sysMsgCastleGateBroken), false)
	}
}

// reviveDoor is Creature.doRevive for a door: alive again with at least
// RespawnRestoreHP of its HP, and Revive to its viewers. It blocks again
// when closed.
func (s *Server) reviveDoor(d *Door) {
	if !d.combat.dead {
		return
	}
	d.combat.dead = false
	if pct := s.combatSettings().RespawnRestoreHP / 100; pct > 0 && d.curHP < float64(d.maxHP)*pct {
		s.setDoorHP(d, float64(d.maxHP)*pct, true)
	}
	s.world.Broadcast(d, serverpackets.NewRevivePacket(int32(d.ID())), false) //nolint:gosec // object IDs fit
	s.GeoBlockersChanged()
}

// ----- AI and clicks -----

// doorAttacked is DoorAI.notifyActionAttacked: every Defender that sees the
// door and has it within its clan help range turns on the attacker.
func (s *Server) doorAttacked(d *Door, attacker Creature) {
	if attacker == nil || !d.InWorld() {
		return
	}
	var guards []*Npc
	for o := range s.world.Visible(d) {
		if n, ok := o.(*Npc); ok && s.npcIs(n, "Defender") && inRadius3D(d.Position(), n.Position(), n.Template.AI.ClanHelpRange) {
			guards = append(guards, n)
		}
	}
	for _, g := range guards {
		s.NotifyAggression(g, attacker, doorDefenderAggro)
	}
}

// doorAttackClick is the attack part of DoorClick, for a click on the
// targeted door: a door p may attack is attacked when the height difference
// is under 400. It reports whether the door was one p may attack (the click
// is then handled, whether p attacks or not).
func (s *Server) doorAttackClick(p *Player, d *Door) bool {
	if !d.autoAttackableBy(p) {
		return false
	}
	if absInt(int(p.Position().Z)-int(d.Position().Z)) < doorClickMaxZ {
		s.playerIntendAttack(p, d)
	}
	return true
}

// doorSkillTarget is Player.useMagic's door rule: skills reach a door only in
// an active siege zone (Mobius door.isInsideZone(SIEGE)), and UNLOCKABLE
// skills any door.
func (s *Server) doorSkillTarget(d *Door, unlockable bool) bool {
	return unlockable || s.siegeAt(d.Position()) != nil
}

// ----- stats -----

// doorStats is a door's stat set for the fight formulas (stats.Combatant):
// the door calculators of Mobius, Formulas.getStdDoorCalculators. The base
// stats (STR, DEX, ...) of a door are 0; nothing adds modifiers, as doors
// take no continuous effects (calcEffectSuccess).
type doorStats struct {
	s *Server
	d *Door
}

var _ stats.Combatant = doorStats{}

// combatStats returns the door's stat set.
func (d *Door) combatStats() doorStats { return doorStats{s: d.srv, d: d} }

// gateDefMult is FuncGatesPDefMod and FuncGatesMDefMod: the defence
// multiplier for the Seal of Strife's owner.
func gateDefMult(sealOwner int) float64 {
	switch sealOwner {
	case doorCabalDawn:
		return doorDawnDefMult
	case doorCabalDusk:
		return doorDuskDefMult
	}
	return 1
}

// Level returns the door's level.
func (st doorStats) Level() int { return st.d.Template.Level }

// CalcFor runs the door calculators: P.Def and M.Def times the Seal of
// Strife's multiplier, FuncAtkAccuracy and FuncAtkEvasion (non-player
// branch) with DEX 0; every other stat is its base value.
func (st doorStats) CalcFor(stat stats.Stat, init float64, _ stats.Context) float64 {
	level := st.d.Template.Level
	switch stat {
	case stats.PDef, stats.MDef:
		owner := 0
		if st.s != nil {
			owner = st.s.sevenSignsSealOwner()
		}
		return init * gateDefMult(owner)
	case stats.AccCombat:
		return init + float64(level)
	case stats.REvas:
		v := init + float64(level)
		if level > 69 {
			v += float64((level - 69) + 2)
		}
		return v
	}
	return init
}

// Accuracy is getAccuracy; doors never attack.
func (st doorStats) Accuracy() int { return int(st.CalcFor(stats.AccCombat, 0, stats.Context{})) }

// EvasionFor is getEvasionRate, capped at MaxEvasion.
func (st doorStats) EvasionFor(ctx stats.Context) int {
	v := int(math.Round(st.CalcFor(stats.REvas, 0, ctx)))
	if st.s != nil {
		v = min(v, st.s.rules.Config().MaxEvasion)
	}
	return v
}

// CritFor is 0: doors never attack.
func (doorStats) CritFor(stats.Context) int { return 0 }

// PAtkFor is 0: doors never attack.
func (doorStats) PAtkFor(stats.Context) float64 { return 0 }

// PDefFor is getPDef: the template's P.Def through the calculator.
func (st doorStats) PDefFor(ctx stats.Context) float64 {
	return st.CalcFor(stats.PDef, float64(st.d.Template.PDef), ctx)
}

// MDefFor is getMDef: the template's M.Def through the calculator.
func (st doorStats) MDefFor(ctx stats.Context) float64 {
	return st.CalcFor(stats.MDef, float64(st.d.Template.MDef), ctx)
}

// ShieldDef is 0: no sDef modifiers.
func (doorStats) ShieldDef() int { return 0 }

// PAtkSpd is CreatureTemplate's default attack speed; doors never attack.
func (doorStats) PAtkSpd() float64 { return 300 }

// StatBonus is BaseStat.calcBonus at the door's base stats, all 0.
func (st doorStats) StatBonus(s stats.Stat) float64 {
	switch s {
	case stats.STR, stats.INT, stats.DEX, stats.WIT, stats.CON, stats.MEN:
		if st.s != nil {
			return st.s.rules.Bonus(s, 0)
		}
	}
	return 1
}
