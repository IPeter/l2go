package game

import (
	"errors"
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/world"
)

// The death feature (Plan 09 phase 6.7, decisions F12 and F13): player
// death (the rest of Mobius Player.doDie after the status's Kill), the
// restart window and RequestRestartPoint (restart.go), the PvP flag and
// karma (pvp.go), exp and levels (levels.go) and their GM commands
// (death_gm.go).
func init() { registerFeature("death", setupDeath) }

// maxDeathPenaltyLevel is the highest Death Penalty level (skill 5076).
const maxDeathPenaltyLevel = 15

// typeWeapon is the Type2 of weapons (Mobius ItemTemplate.TYPE2_WEAPON).
const typeWeapon = 0

// System messages of death.
const (
	sysMsgDeathPenaltyLevel  serverpackets.SystemMessageID = 1916 // "Your Death Penalty is now level $s1."
	sysMsgDeathPenaltyLifted serverpackets.SystemMessageID = 1917 // "Your Death Penalty has been lifted."
)

// deathPenaltyOwner owns the modifiers of the Death Penalty (skill 5076).
var deathPenaltyOwner = stats.Owner{Kind: stats.OwnerSkill, ID: data.SkillDeathPenalty}

// deathFeature is the death feature's state on the Server (Server.death).
// Loop-only, except the modifiers, which setup fills.
type deathFeature struct {
	// flagged are the players in the PvP flag scan, in the order they were
	// added (Mobius PvpFlagTaskManager.PLAYERS).
	flagged []*Player
	// flagLasts is when each player's PvP flag ends (Mobius pvpFlagLasts).
	flagLasts map[*Player]time.Time
	// karmaHooks run after a karma change (OnKarmaChanged).
	karmaHooks []KarmaHook
	// penaltyMods are skill 5076's modifiers per level 1-15.
	penaltyMods [maxDeathPenaltyLevel + 1][]stats.Modifier
}

// setupDeath registers the death feature.
func setupDeath(s *Server) error {
	s.death.flagLasts = make(map[*Player]time.Time)
	s.death.penaltyMods = s.deathPenaltyModifiers()
	s.OnDie(s.onDeath)
	s.OnPlayerEnter(s.deathEnter)
	s.OnPlayerLeave(s.pvpLeave)
	s.OnPlayerSwing(s.UpdatePvPStatus) // Mobius onHitTimer: updatePvPStatus(target)
	s.loop.Every(pvpFlagScan, s.pvpFlagTick)
	return errors.Join(
		s.Register(clientpackets.OpRequestRestartPoint, s.handleRestartPoint),
		s.registerDeathGMCommands(),
	)
}

// deathPenaltyModifiers converts skill 5076's levels into modifiers; none
// without passive skill data.
func (s *Server) deathPenaltyModifiers() [maxDeathPenaltyLevel + 1][]stats.Modifier {
	var mods [maxDeathPenaltyLevel + 1][]stats.Modifier
	if s.items.passives == nil {
		return mods
	}
	for level := 1; level <= maxDeathPenaltyLevel; level++ {
		l, ok := s.items.passives.Level(data.SkillDeathPenalty, level)
		if !ok {
			s.log.Warn("game: the passive skill data has no death penalty level", "level", level)
			continue
		}
		for _, fn := range l.Funcs {
			st, ok1 := stats.ParseStat(fn.Stat.String())
			op, ok2 := stats.ParseOp(fn.Op.String())
			if !ok1 || !ok2 {
				s.log.Warn("game: death penalty function not supported", "stat", fn.Stat.String(), "op", fn.Op.String())
				continue
			}
			mods[level] = append(mods[level], stats.Modifier{Stat: st, Op: op, Order: fn.Order, Value: fn.Value, Owner: deathPenaltyOwner})
		}
	}
	return mods
}

// applyDeathPenalty puts the modifiers of p's Death Penalty level on its
// stat set (Mobius restoreDeathPenaltyBuffLevel and the skill swap of
// increaseDeathPenaltyBuffLevel). It touches no world state.
func (s *Server) applyDeathPenalty(p *Player) {
	p.stats.RemoveOwner(deathPenaltyOwner)
	level := p.char.DeathPenaltyLevel
	if level <= 0 || level > maxDeathPenaltyLevel {
		return
	}
	if mods := s.death.penaltyMods[level]; len(mods) > 0 {
		if err := p.stats.AddModifiers(mods...); err != nil {
			s.log.Warn("game: death penalty modifiers refused", append(p.logArgs(), "err", err)...)
		}
	}
}

// DeathPenaltyLevel returns p's Death Penalty level, 0-15.
func (p *Player) DeathPenaltyLevel() int32 { return p.char.DeathPenaltyLevel }

// ExpBeforeDeath returns the exp p had before its last death, 0 after a
// revive that restored it.
func (p *Player) ExpBeforeDeath() int64 { return p.char.ExpBeforeDeath }

// deathEnter is the OnPlayerEnter hook of the feature: a player entering
// dead gets its restart window (Mobius EnterWorld sends Die right before
// onPlayerEnter). The status marks it dead in its own hook.
func (s *Server) deathEnter(p *Player) {
	if p.dead || p.CurHP < 0.5 {
		p.Send(diePacket(p))
	}
}

// diePacket is Die for a dead player (Mobius Die): "To Village", and
// "Fixed" for GMs (their access level allows a fixed resurrection; there
// are no self-resurrection items yet). Clans come later.
func diePacket(p *Player) []byte {
	return serverpackets.NewDiePacket(serverpackets.Die{
		ObjectID:    p.ObjectID(),
		CanTeleport: true,
		FixedRes:    p.IsGM(),
	})
}

// killerKind is what killed a player, for the death penalties.
func killerKind(killer Creature) stats.KillerKind {
	switch k := killer.(type) {
	case *Player:
		return stats.KillerPlayable
	case *Npc:
		switch {
		case k.class != nil && k.class.raid:
			return stats.KillerRaid
		case k.IsMonster():
			return stats.KillerMonster
		}
	}
	return stats.KillerOther
}

// isLucky is Mobius Player.isLucky: level 9 or less with the Lucky skill
// (194). Every starting class learns it at level 1 and nothing removes it
// before M7, so up to level 9 every player is lucky.
func isLucky(p *Player) bool { return p.Level() <= 9 }

// onDeath is the OnDie hook of the feature: the death of a player, in the
// order of Mobius Playable.doDie and Player.doDie after the HP went to 0.
//
//  1. The target is dropped and the walk stops (StopMove).
//  2. A player killer gets its PvP or PK count and karma.
//  3. Die goes to the player and the players around (the AI's
//     notifyActionDeath).
//  4. With a killer: the exp before death is cleared, karma drops roll,
//     and outside PvP zones (or in a siege zone) a non-lucky player loses
//     exp with Delevel on.
//  5. With a killer: the Death Penalty roll.
//  6. Karma goes down to karma − karma/4, or 0 below 200.
func (s *Server) onDeath(victim, killer Creature) {
	p, ok := victim.(*Player)
	if !ok || p.state != stateInGame {
		return
	}
	if p.Target != nil {
		s.SetTarget(p, nil)
	}
	s.StopPlayer(p)
	if kp, ok := killer.(*Player); ok && kp != nil {
		s.onKillUpdatePvPKarma(kp, p)
	}
	if p.InWorld() {
		s.world.Broadcast(p, diePacket(p), true)
	}

	if killer != nil {
		p.char.ExpBeforeDeath = 0
		s.dropOnDeath(p, killer)
		pvp, siege := p.Zones.Inside(ZonePvP), p.Zones.Inside(ZoneSiege)
		if s.combatSettings().Delevel && !isLucky(p) && (siege || !pvp) {
			s.deathExpPenalty(p, killer)
		}
		s.deathPenaltyRoll(p, killer)
	}
	if p.char.Karma > 0 {
		s.SetKarma(p, int32(stats.KarmaAfterDeath(int(p.char.Karma)))) //nolint:gosec // karma fits
	}
}

// deathExpPenalty is Mobius calculateDeathExpPenalty: the loss of
// stats.CombatRules.DeathExpLoss for p's level span and experienceLoss.xml
// percent, after keeping the exp before death; RemoveExp may lower the
// level. Nothing is sent but the UserInfo of a delevel.
func (s *Server) deathExpPenalty(p *Player, killer Creature) {
	level := p.Level()
	maxLevel := s.MaxLevel()
	var span int64
	if level < maxLevel {
		span = s.ExpForLevel(level+1) - s.ExpForLevel(level)
	} else {
		span = s.ExpForLevel(maxLevel) - s.ExpForLevel(maxLevel-1)
	}
	percent, _ := s.data.ExperienceLoss.At(level)
	lost := s.combat.DeathExpLoss(p.stats, stats.DeathLoss{
		Span:    span,
		Percent: percent,
		Killer:  killerKind(killer),
		Karma:   p.char.Karma > 0,
	})
	p.char.ExpBeforeDeath = p.char.Exp
	s.RemoveExp(p, lost)
}

// deathPenaltyRoll is Mobius calculateDeathPenaltyBuffLevel: no roll for a
// lucky player, in a PvP or siege zone, or for a GM; otherwise
// stats.CombatRules.DeathPenaltyRoll raises the level.
func (s *Server) deathPenaltyRoll(p *Player, killer Creature) {
	if isLucky(p) || p.Zones.Inside(ZonePvP) || p.Zones.Inside(ZoneSiege) || p.IsGM() {
		return
	}
	if s.combat.DeathPenaltyRoll(s.rnd, p.stats, killerKind(killer), p.char.Karma > 0) {
		s.IncreaseDeathPenalty(p)
	}
}

// IncreaseDeathPenalty raises p's Death Penalty level by one, up to 15
// (Mobius increaseDeathPenaltyBuffLevel): skill 5076 of the new level
// replaces the old one, p gets EtcStatusUpdate and "Your Death Penalty is
// now level $s1.". Loop-only.
func (s *Server) IncreaseDeathPenalty(p *Player) {
	if p.char.DeathPenaltyLevel >= maxDeathPenaltyLevel {
		return
	}
	p.char.DeathPenaltyLevel++
	s.applyDeathPenalty(p)
	p.Send(etcStatusPacket(p))
	p.SendSystemMessage(sysMsgDeathPenaltyLevel, serverpackets.NumberParam(p.char.DeathPenaltyLevel))
}

// ReduceDeathPenalty lowers p's Death Penalty level by one (Mobius
// reduceDeathPenaltyBuffLevel, for the NPCs and items that lift it):
// EtcStatusUpdate and "Your Death Penalty is now level $s1.", or "Your
// Death Penalty has been lifted." at 0. Loop-only.
func (s *Server) ReduceDeathPenalty(p *Player) {
	if p.char.DeathPenaltyLevel <= 0 {
		return
	}
	p.char.DeathPenaltyLevel--
	s.applyDeathPenalty(p)
	p.Send(etcStatusPacket(p))
	if p.char.DeathPenaltyLevel > 0 {
		p.SendSystemMessage(sysMsgDeathPenaltyLevel, serverpackets.NumberParam(p.char.DeathPenaltyLevel))
	} else {
		p.SendSystemMessage(sysMsgDeathPenaltyLifted)
	}
}

// dropOnDeath is Mobius Player.onDieDropItem without the player drop rates
// (PlayerRateDrop and friends are 0 by default): outside PvP zones (or
// killed by no player), not a GM unless CanGMDropEquipment, a player with
// karma and at least MinimumPkRequiredToDrop PKs rolls KarmaRateDrop; then
// every droppable item rolls its rate (weapon, equipment, inventory) until
// KarmaDropLimit items fell next to the player.
func (s *Server) dropOnDeath(p *Player, killer Creature) {
	cfg := s.pvpConfig()
	_, byPlayer := killer.(*Player)
	if p.Zones.Inside(ZonePvP) && byPlayer {
		return
	}
	if p.IsGM() && !cfg.CanGMDropEquipment {
		return
	}
	if p.char.Karma <= 0 || int(p.char.PKKills) < cfg.MinimumPkRequiredToDrop {
		return
	}
	if cfg.KarmaRateDrop <= 0 || s.rnd.Get(100) >= cfg.KarmaRateDrop {
		return
	}
	dropped := 0
	for _, it := range slices.Clone(p.inv.Items()) {
		if !karmaDroppable(it, cfg.ListOfNonDroppableItems, cfg.ListOfPetItems) {
			continue
		}
		chance := cfg.KarmaRateDropItem
		if it.Equipped() {
			chance = cfg.KarmaRateDropEquip
			if it.Template().Type2 == typeWeapon {
				chance = cfg.KarmaRateDropEquipWeapon
			}
		}
		if s.rnd.Get(100) >= chance {
			continue
		}
		if it.Equipped() {
			p.inv.UnequipSlot(it.Slot())
		}
		s.dropItemOnDeath(p, it)
		s.SendItemList(p, false)
		s.log.Warn("character has karma and dropped an item", append(p.logArgs(), "item", it.ItemID(), "count", it.Count())...)
		if dropped++; dropped >= cfg.KarmaDropLimit {
			break
		}
	}
}

// karmaDroppable reports whether a player with karma can lose it on death:
// not a shadow item, adena, a quest item, an item that cannot be dropped
// or one of the configured lists.
func karmaDroppable(it *inventory.Item, never, pet []int) bool {
	t := it.Template()
	id := int(it.ItemID())
	switch {
	case it.Shadow(), !t.Droppable, it.ItemID() == inventory.AdenaID, t.Type2 == typeQuest:
		return false
	case slices.Contains(never, id), slices.Contains(pet, id):
		return false
	}
	return true
}

// dropItemOnDeath drops a whole item next to p (Mobius Player.dropItem(DEATH,
// item, killer, true)): up to 25 units away in x and y, 20 higher, with the
// InventoryUpdate, the load and "You have dropped $s1.".
func (s *Server) dropItemOnDeath(p *Player, it *inventory.Item) {
	dropped, err := p.inv.DropItem(it)
	if err != nil {
		return
	}
	s.forgetMana(dropped)
	pos := p.Position()
	dx := int32(s.rnd.Get(50)) - 25 //nolint:gosec // below 50
	dy := int32(s.rnd.Get(50)) - 25 //nolint:gosec // below 50
	at := world.Position{X: pos.X + dx, Y: pos.Y + dy, Z: pos.Z + 20}
	if _, err := s.DropOnGround(dropped, at, GroundDrop{Dropper: p, PlayerDrop: true}); err != nil {
		s.log.Error("game: an item dropped on death cannot go into the world", append(p.logArgs(), "itemObjectId", dropped.ObjectID(), "err", err)...)
	}
	s.itemsChanged(p, true, nil)
	p.SendSystemMessage(sysMsgDroppedS1, s.itemNameParam(dropped.ItemID()))
}
