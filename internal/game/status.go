package game

import (
	"reflect"
	"strconv"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/world"
)

// The status feature ports Mobius CreatureStatus, PlayerStatus and NpcStatus:
// the current HP, MP and CP of players and NPCs, the dead flag, regeneration
// and the StatusUpdate packets that follow the values (status_player.go,
// status_npc.go). It also maps the config to the combat rules (combatConfig)
// and region activation (activationOf), and starts the regeneration of the
// attackables of a region that becomes active (Mobius switchAI).
//
// The API for combat, AI and death:
//
//   - Creature is a player or an NPC: IsDead, CurrentHP, MaxHP, ReduceHP.
//   - Player and Npc: SetCurrentHP, SetCurrentMP, SetCurrentHPMP (and the
//     player's SetCurrentCP, ReduceCP), ReduceHP, ReduceMP, StartRegen,
//     StopRegen, Regenerating, Revive, BroadcastStatus.
//   - Server.Kill runs a death once (Mobius doDie's guard) and the OnDie
//     hooks; ReduceHP calls it when the HP fall under 0.5.
//   - Server.OnRegionActivation adds a region activation hook.
func init() { registerFeature("status", setupStatus) }

// sysMsgHitYouForDamage is "$s1 hit you for $s2 damage." (Mobius
// S1_HIT_YOU_FOR_S2_DAMAGE).
const sysMsgHitYouForDamage serverpackets.SystemMessageID = 36

// Creature is a player or an NPC: what has HP, takes damage and dies. *Player
// and *Npc implement it.
type Creature interface {
	world.Object
	// Name is the name shown in system messages (Mobius getName).
	Name() string
	// IsDead reports whether the creature is dead.
	IsDead() bool
	// CurrentHP returns the current HP.
	CurrentHP() float64
	// MaxHP returns the maximum HP.
	MaxHP() int32
	// ReduceHP takes value HP away, as a hit of attacker (nil for none)
	// does, and kills the creature when its HP fall under 0.5 (Mobius
	// reduceCurrentHp through the status). Loop-only.
	ReduceHP(value float64, attacker Creature)
}

var (
	_ Creature = (*Player)(nil)
	_ Creature = (*Npc)(nil)
)

// DieHook runs when a creature dies (Mobius doDie after its guard), with the
// killer or nil.
type DieHook func(victim, killer Creature)

// statusFeature is the status feature's state on the Server (Server.status).
// Loop-only.
type statusFeature struct {
	die []DieHook
	// queue holds the players whose StatusUpdate is due at the end of the
	// tick (Player.BroadcastStatus).
	queue, spare []*Player
}

// setupStatus registers the status feature's hooks.
func setupStatus(s *Server) error {
	s.OnPlayerEnter(s.restoreStatus)
	s.OnPlayerLeave(func(p *Player) { p.StopRegen() })
	s.loop.OnTickEnd(s.flushStatus)
	s.world.OnRegionActivation(s.regionSwitched)
	return nil
}

// OnDie adds fn to the functions run when a creature dies, in the order
// added. They run after the HP were set to 0 (the HP bar or the StatusUpdate
// is on its way), the dead flag was set and regeneration stopped: rewards,
// stopping the attack, movement and AI, the Die packet and the decay are
// theirs (the rest of Mobius doDie). Call it from a feature's setup function.
func (s *Server) OnDie(fn DieHook) { s.status.die = append(s.status.die, fn) }

// OnRegionActivation adds fn to the hooks run when a world region becomes
// active or inactive (world.World.OnRegionActivation): the AI resets monsters
// there (Mobius switchAI). Call it from a feature's setup function.
func (s *Server) OnRegionActivation(fn world.RegionHook) { s.world.OnRegionActivation(fn) }

// Kill kills victim, as Mobius doDie does: it reports false when the victim
// is already dead; otherwise it sets the HP to 0 (with its packets), sets the
// dead flag, stops regeneration and runs the OnDie hooks. GM commands and
// scripts kill with it; ReduceHP calls it. Loop-only.
func (s *Server) Kill(victim, killer Creature) bool {
	switch v := victim.(type) {
	case *Player:
		if v.dead {
			return false
		}
		v.setCurrentHP(0, true)
		v.dead = true
		v.StopRegen()
	case *Npc:
		if v.Dead {
			return false
		}
		v.setCurrentHP(0, true)
		v.Dead = true
		v.StopRegen()
	default:
		return false
	}
	for _, fn := range s.status.die {
		s.safely("OnDie hook", func() { fn(victim, killer) })
	}
	return true
}

// Regeneration flags of a regenState (Mobius REGEN_FLAG_*).
const (
	regenHP uint8 = 1 << iota
	regenMP
	regenCP
)

// regenState is a creature's regeneration task (Mobius _regTask and
// _flagsRegenActive): a timer every stats.RegenPeriod, started when a value
// drops below its maximum and stopped when all are full again.
type regenState struct {
	timer world.TimerID
	flags uint8
}

// start starts the task unless it runs (Mobius startHpMpRegeneration; the
// caller checks the dead flag). The first tick comes one period later.
func (r *regenState) start(l *world.Loop, tick func()) {
	if r.timer == 0 {
		r.timer = l.Every(stats.RegenPeriod, tick)
	}
}

// stop stops the task and clears the flags (Mobius stopHpMpRegeneration).
func (r *regenState) stop(l *world.Loop) {
	if r.timer != 0 {
		l.Cancel(r.timer)
		r.timer = 0
		r.flags = 0
	}
}

// javaInt32 is Java's (int) cast of a double: toward zero, NaN to 0, and
// saturating.
func javaInt32(v float64) int32 {
	switch {
	case v != v:
		return 0
	case v >= 2147483647:
		return 2147483647
	case v <= -2147483648:
		return -2147483648
	}
	return int32(v)
}

// flushStatus sends the StatusUpdate of every player that asked for one
// during the tick (Mobius sends it 50 ms after the first request). It runs
// after the coalesced UserInfo, as Mobius sends UserInfo at once.
func (s *Server) flushStatus(time.Time) {
	f := &s.status
	if len(f.queue) == 0 {
		return
	}
	queue := f.queue
	f.queue = f.spare[:0]
	for _, p := range queue {
		p.statusQueued = false
		if p.state == stateInGame {
			p.Send(vitalsStatusUpdate(p))
		}
	}
	clear(queue)
	f.spare = queue[:0]
}

// regionSwitched is the region activation hook of the status: the
// attackables of a region that becomes active start regenerating (Mobius
// switchAI(true)); a full one stops again at its first tick.
func (s *Server) regionSwitched(_ world.RegionKey, active bool, objects []world.Object) {
	if !active {
		return
	}
	for _, o := range objects {
		if n, ok := o.(*Npc); ok && n.class != nil && n.class.attackable {
			n.StartRegen()
		}
	}
}

// regenZone returns what p's zones add to its regeneration: the Mother Tree
// bonuses (Mobius calcHpRegen and calcMpRegen read the first MotherTreeZone
// the player is in). Sieges, festivals and residences do not exist yet.
func (s *Server) regenZone(p *Player) stats.RegenZone {
	var z stats.RegenZone
	if !p.Zones.Inside(ZoneMotherTree) || s.worldData == nil {
		return z
	}
	for _, id := range p.Zones.IDs() {
		dz, ok := s.worldData.Zone(id)
		if !ok || !s.worldData.ZoneTypeIs(dz.Type, "MotherTreeZone") {
			continue
		}
		z.HPBonus = zoneIntParam(dz.Params, "HpRegenBonus")
		z.MPBonus = zoneIntParam(dz.Params, "MpRegenBonus")
		break
	}
	return z
}

// zoneIntParam returns the last value of the int parameter name, as Mobius
// setParameter keeps it; 0 when it is missing or no number.
func zoneIntParam(params []data.ZoneParam, name string) float64 {
	v := 0
	for _, prm := range params {
		if prm.Name == name {
			if n, err := strconv.Atoi(prm.Value); err == nil {
				v = n
			}
		}
	}
	return float64(v)
}

// combatConfig maps the config of a game server to the combat rules: the
// combat section, the rates and Mobius's defaults for what the config has no
// key for (raid regeneration, exp and SP bonus caps). The percentages
// become factors, as Mobius divides HpRegenMultiplier and friends by 100. A
// section missing from a config made in code takes its defaults.
func combatConfig(gs config.GameServer) stats.CombatConfig {
	c, r := gs.Combat, gs.Rates
	if c == (config.Combat{}) {
		c = config.DefaultCombat()
	}
	if reflect.ValueOf(r).IsZero() {
		r = config.DefaultRates()
	}
	cfg := stats.DefaultCombatConfig()
	cfg.RandomizeAutoAttackDamage = c.RandomizeAutoAttackDamage
	cfg.ShieldBlocks = c.ShieldBlocks
	cfg.PerfectShieldBlockRate = c.PerfectShieldBlockRate
	cfg.DistanceBowDamage = c.DistanceBowDamage
	cfg.CancelCastByHit = c.CancelByHit == config.CancelByHitCast || c.CancelByHit == config.CancelByHitAll
	cfg.CancelBowByHit = c.CancelByHit == config.CancelByHitBow || c.CancelByHit == config.CancelByHitAll
	cfg.HPRegenMultiplier = c.HPRegenMultiplier / 100
	cfg.MPRegenMultiplier = c.MPRegenMultiplier / 100
	cfg.CPRegenMultiplier = c.CPRegenMultiplier / 100
	cfg.DeathPenaltyChance = c.DeathPenaltyChance
	cfg.RateXP, cfg.RateSP = r.XP, r.SP
	cfg.MonsterExpMaxLevelDifference = r.MonsterExpMaxLevelDifference
	cfg.RateKarmaLost = r.KarmaLost
	cfg.RateKarmaExpLost = r.KarmaExpLost
	return cfg
}

// activationOf maps the world config to region activation (decision F6). A
// world section missing from a config made in code takes its defaults.
func activationOf(w config.World) world.Activation {
	if w == (config.World{}) {
		w = config.DefaultWorld()
	}
	return world.Activation{
		AlwaysOn: w.GridsAlwaysOn,
		TurnOn:   time.Duration(w.GridTurnOnTime),
		TurnOff:  time.Duration(w.GridTurnOffTime),
	}
}
