package game

import (
	"fmt"
	"maps"
	"reflect"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/world"
)

// The NPC death feature (Plan 09 phase 6.6, decisions F8 and F9): the rest
// of Mobius Creature.doDie, Npc.doDie and Attackable.doDie after the
// status's Kill (npc_death.go), the kill rewards (npc_reward.go) and the
// corpse decay (npc_decay.go). The order of a kill is Mobius's:
//
//  1. Kill: HP 0 (the HP bar of the players targeting it), the dead flag,
//     regeneration stops.
//  2. The rewards of an attackable: drops, then exp and SP with their
//     level-ups (Attackable.calculateRewards).
//  3. The walk stops and, for a kill by a player, the clan comes for the
//     killer (the AI's npcDeathBeforeDie).
//  4. The HP bar again (broadcastStatusUpdate), Die to the players around
//     (notifyActionDeath), and the AI turns IDLE (npcDeathAfterDie).
//  5. The weapons shown go back to the template's and the decay timer
//     starts (Npc.doDie): after the corpse time the NPC leaves the world and
//     its spawn schedules the respawn (Npc.onDecay, Spawn.decreaseCount).
//
// The API: Server.CancelNpcDecay and Server.EndNpcDecay (Mobius
// DecayTaskManager.cancel and Npc.endDecayTask).
func init() { registerFeature("npcdeath", setupNpcDeath) }

// npcDeathFeature is the NPC death feature's state on the Server
// (Server.kill). Loop-only, except the drop rules, which setup makes.
type npcDeathFeature struct {
	dropRules *stats.DropRules
	// drops holds the sorted drop list of each template, nil for none,
	// worked out on the first kill.
	drops map[int]*stats.DropList
	// decay holds when each corpse decays; decayOrder keeps the corpses in
	// the order they died, for the scan (Mobius DECAY_SCHEDULES).
	decay      map[*Npc]time.Time
	decayOrder []*Npc
}

// setupNpcDeath registers the death hook and the decay scan.
func setupNpcDeath(s *Server) error {
	rules, err := stats.NewDropRules(dropConfig(s.gs))
	if err != nil {
		return fmt.Errorf("drop rules: %w", err)
	}
	f := &s.kill
	f.dropRules = rules
	f.drops = make(map[int]*stats.DropList)
	f.decay = make(map[*Npc]time.Time)
	s.OnDie(s.npcDied)
	s.loop.Every(decayScanPeriod, s.decayScan)
	return nil
}

// dropConfig maps the rates of a game server's config to the drop rules
// (Rates.ini): the death, herb and per-item rates, the occurrence limit and
// the level gaps. Raid rates keep Mobius's defaults. A rates section missing
// from a config made in code takes its defaults.
func dropConfig(gs config.GameServer) stats.DropConfig {
	r := gs.Rates
	if reflect.ValueOf(r).IsZero() {
		r = config.DefaultRates()
	}
	cfg := stats.DefaultDropConfig()
	cfg.DeathChance, cfg.DeathAmount = r.DeathDropChance, r.DeathDropAmount
	cfg.HerbChance, cfg.HerbAmount = r.HerbDropChance, r.HerbDropAmount
	cfg.ChanceByItem = maps.Clone(map[int]float64(r.DropChanceByItem))
	cfg.AmountByItem = maps.Clone(map[int]float64(r.DropAmountByItem))
	cfg.MaxOccurrencesNormal = r.DropMaxOccurrencesNormal
	cfg.AdenaMinLevelDifference = r.DropAdenaMinLevelDifference
	cfg.AdenaMaxLevelDifference = r.DropAdenaMaxLevelDifference
	cfg.AdenaMinLevelGapChance = r.DropAdenaMinLevelGapChance
	cfg.ItemMinLevelDifference = r.DropItemMinLevelDifference
	cfg.ItemMaxLevelDifference = r.DropItemMaxLevelDifference
	cfg.ItemMinLevelGapChance = r.DropItemMinLevelGapChance
	return cfg
}

// npcDied is the OnDie hook for NPCs: the kill sequence after Kill, in the
// order of the feature comment.
func (s *Server) npcDied(victim, killer Creature) {
	n, ok := victim.(*Npc)
	if !ok {
		return
	}
	attackable := n.class != nil && n.class.attackable
	if attackable {
		s.npcRewards(n, killer)
		s.npcDeathBeforeDie(n, killer)
	} else {
		s.stopNpc(n, nil)
	}
	n.BroadcastStatus()
	if n.InWorld() {
		s.world.Broadcast(n, npcDiePacket(n), false)
	}
	if attackable {
		s.npcDeathAfterDie(n)
	}
	// Npc.doDie: the weapons shown go back to the template's.
	n.RHand, n.LHand = int32(n.Template.Equipment.RHand), int32(n.Template.Equipment.LHand) //nolint:gosec // item IDs are small
	s.startDecay(n)
	// raid.go: Attackable.doDie's minion hooks, then RaidBoss and GrandBoss
	// doDie (the success message, raid points, the raid spawner).
	s.bossNpcDied(n, killer)
}

// npcDiePacket is Die for an NPC (Mobius Die): an NPC can always be
// revived; spoil, which makes it sweepable, comes with M7.
func npcDiePacket(n *Npc) []byte {
	return serverpackets.NewDiePacket(serverpackets.Die{ObjectID: int32(n.ID()), CanTeleport: true})
}

// ----- drops (decision F8) -----

// npcDropRange is how far from the corpse a drop falls in x and y (Mobius
// Npc.RANDOM_ITEM_DROP_LIMIT); it falls 20 above the corpse.
const (
	npcDropRange  = 70
	npcDropHeight = 20
)

// npcDropItems is Mobius Attackable.doItemDrop for the death drop list:
// nothing unless a player gets the drops; the list's drops
// (stats.DropRules.CalculateDrops) go into the player's inventory with
// autoLoot (herbs with autoLootHerbs, everything from a flying monster) and
// otherwise fall around the corpse (npcDropItem). Spoil, quest and event
// drops come later.
func (s *Server) npcDropItems(n *Npc, c Creature) {
	p, ok := c.(*Player)
	if !ok || p == nil {
		return
	}
	list := s.dropListOf(n.Template)
	if list == nil {
		return
	}
	raid := n.class != nil && n.class.raid
	drops := s.kill.dropRules.CalculateDrops(s.rnd, list, stats.DropKill{
		VictimLevel: n.Level(),
		KillerLevel: p.Level(),
		Raid:        raid,
		Player:      p.stats,
	})
	combat := s.combatSettings()
	for _, d := range drops {
		t, ok := s.data.ItemTemplate(d.ItemID)
		if !ok || d.Count <= 0 {
			continue
		}
		herb := t.ExImmediateEffect
		autoLoot := n.Template.Status.Flying || (!herb && !raid && combat.AutoLoot) || (herb && combat.AutoLootHerbs)
		if autoLoot && p.InGame() {
			s.autoLoot(p, n, t, d.Count)
			continue
		}
		s.npcDropItem(n, p, t, d.Count)
	}
}

// autoLoot is Mobius Player.doAutoLoot: in a party everything but herbs is
// shared by the loot mode around the monster (party_reward.go); otherwise
// adena goes through AddAdena, other items through AddItem, with their
// messages.
func (s *Server) autoLoot(p *Player, n *Npc, t *data.Item, count int) {
	if p.InParty() && !t.ExImmediateEffect {
		s.partyDistributeItem(p, int32(t.ID), int64(count), false, n) //nolint:gosec // item IDs are small
		return
	}
	if t.ID == adenaID {
		s.AddAdena(p, int64(count), true)
		return
	}
	if _, err := s.AddItem(p, int32(t.ID), int64(count), -1, true, false); err != nil { //nolint:gosec // item IDs are small
		s.log.Warn("game: cannot loot an item", append(p.logArgs(), "item", t.ID, "count", count, "err", err)...)
	}
}

// npcDropItem is Mobius Npc.dropItem: a new item at a random point within
// 70 of the corpse in x and y, 20 above it, protected for the player for
// 15 s. A stack falls as one item; a count of items that do not stack falls
// as that many items with MultipleItemDrop, one each, or as one otherwise.
func (s *Server) npcDropItem(n *Npc, p *Player, t *data.Item, count int) {
	if s.items.pool == nil {
		s.log.Error("game: no object ID pool; a monster drop is lost", "npc", n.Template.ID, "item", t.ID)
		return
	}
	pos := n.Position()
	for range count {
		id, ok := s.items.pool.TryNext()
		if !ok {
			s.log.Error("game: no object ID left; a monster drop is lost", "npc", n.Template.ID, "item", t.ID)
			return
		}
		it := inventory.NewItem(id, t, int64(count))
		x := pos.X + int32(s.rnd.Get(2*npcDropRange+1)) - npcDropRange //nolint:gosec // below 141
		y := pos.Y + int32(s.rnd.Get(2*npcDropRange+1)) - npcDropRange //nolint:gosec // below 141
		at := world.Position{X: x, Y: y, Z: pos.Z + npcDropHeight}
		if _, err := s.DropOnGround(it, at, GroundDrop{Dropper: n, ProtectFor: p}); err != nil {
			s.log.Error("game: a monster drop cannot go into the world", "npc", n.Template.ID, "item", t.ID, "err", err)
			return
		}
		if it.Stackable() || !s.gs.Items.MultipleItemDrop {
			return
		}
	}
}

// dropListOf returns the sorted death drop list of a template, nil when it
// has none. Items the item data does not know are left out, as Mobius
// NpcData does; herbs are the items with ex_immediate_effect.
func (s *Server) dropListOf(t *data.NpcTemplate) *stats.DropList {
	f := &s.kill
	if l, ok := f.drops[t.ID]; ok {
		return l
	}
	convert := func(items []data.NpcDropItem) []stats.DropItem {
		var out []stats.DropItem
		for _, d := range items {
			it, ok := s.data.ItemTemplate(d.ItemID)
			if !ok {
				continue
			}
			out = append(out, stats.DropItem{ItemID: d.ItemID, Min: d.Min, Max: d.Max, Chance: d.Chance, Herb: it.ExImmediateEffect})
		}
		return out
	}
	l := &stats.DropList{Items: convert(t.Drops.Items)}
	for _, g := range t.Drops.Groups {
		l.Groups = append(l.Groups, stats.DropGroup{Chance: g.Chance, Items: convert(g.Items)})
	}
	if len(l.Items) == 0 && len(l.Groups) == 0 {
		l = nil
	} else {
		l.Sort()
	}
	f.drops[t.ID] = l
	return l
}
