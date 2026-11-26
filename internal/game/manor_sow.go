package game

import (
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/manor"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/stats"
)

// Sowing and harvesting (Plan 14 M11k): the Seed and Harvester item
// handlers (Mobius handlers/items), the Sow and Harvesting effects
// (skills.ManorWorld), the seed state of a monster (Mobius Attackable
// setSeeded, takeHarvest, cleared when it leaves the world), a sown
// monster's death drops (adena and seal stones only) and its longer corpse.

var _ skills.ManorWorld = skillWorld{}

// The item handlers of the manor.
const (
	handlerSeed      = "Seed"
	handlerHarvester = "Harvester"
)

// System messages of sowing and harvesting.
const (
	manorMsgInvalidTarget      serverpackets.SystemMessageID = 109  // "Invalid target."
	manorMsgSeedNotHere        serverpackets.SystemMessageID = 872  // "This seed may not be sown here."
	manorMsgSown               serverpackets.SystemMessageID = 889  // "The seed was successfully sown."
	manorMsgNotSown            serverpackets.SystemMessageID = 890  // "The seed was not sown."
	manorMsgNotAuthorized      serverpackets.SystemMessageID = 891  // "You are not authorized to harvest."
	manorMsgHarvestFailed      serverpackets.SystemMessageID = 892  // "The harvest has failed."
	manorMsgHarvestNotSown     serverpackets.SystemMessageID = 893  // "The harvest failed because the seed was not sown."
	manorMsgHarvestedS3S2      serverpackets.SystemMessageID = 1137 // "$s1 harvested $s3 $s2(s)."
	manorMsgHarvestedS2        serverpackets.SystemMessageID = 1138 // "$s1 harvested $s2(s)."
	manorMsgUnavailableSeeding serverpackets.SystemMessageID = 1516 // "The target is unavailable for seeding."
)

// npcSeed is a monster's seed state (Mobius Attackable _seed, _seederObjId,
// _seeded and _harvestItem).
type npcSeed struct {
	seed   *manor.Seed
	seeder int32
	sown   bool
	// harvest is the crop count the harvest gives, 0 once it is taken.
	harvest int64
}

// manorSeedOf returns n's seed state, nil when no seed was chosen on it.
func (s *Server) manorSeedOf(n *Npc) *npcSeed { return s.manor.seeds[n] }

// manorSown reports whether n was sown (Attackable.isSeeded).
func (s *Server) manorSown(n *Npc) bool {
	st := s.manor.seeds[n]
	return st != nil && st.sown
}

// manorNpcGone clears a monster's seed when it leaves the world (Mobius
// clears it at the next spawn, Attackable.onSpawn).
func (s *Server) manorNpcGone(n *Npc) { delete(s.manor.seeds, n) }

// useSeed is the Seed item handler: on a living monster (not a raid boss or
// a chest) not sown yet, in the seed's castle's area, the seed is chosen for
// p and the item's Sowing skill is cast.
func (s *Server) useSeed(p *Player, it *inventory.Item) bool {
	f := &s.manor
	if !f.enabled {
		return false
	}
	n, ok := p.Target.(*Npc)
	if !ok || n == nil {
		p.SendSystemMessage(manorMsgInvalidTarget)
		return false
	}
	if n.class == nil || !n.class.monster || n.class.raid || s.npcIs(n, "Chest") {
		p.SendSystemMessage(manorMsgUnavailableSeeding)
		return false
	}
	if n.IsDead() {
		p.SendSystemMessage(manorMsgInvalidTarget)
		return false
	}
	if s.manorSown(n) {
		p.SendActionFailed()
		return false
	}
	seed := f.table.Seed(int(it.ItemID()))
	if seed == nil {
		return false
	}
	pos := p.Position()
	if s.worldData == nil || seed.CastleID != s.worldData.AreaCastle(int(pos.X), int(pos.Y)) {
		p.SendSystemMessage(manorMsgSeedNotHere)
		return false
	}
	// Attackable.setSeeded(seed, player): another seed or sower replaces one
	// that was not sown.
	f.seeds[n] = &npcSeed{seed: seed, seeder: p.ObjectID()}
	for _, ref := range normalSkills(it.Template()) {
		if sk, ok := s.Skill(ref.ID, ref.Level); ok {
			s.UseMagic(p, sk, false, false)
		}
	}
	return true
}

// useHarvester is the Harvester item handler: on a dead monster, the item's
// Harvesting skill is cast.
func (s *Server) useHarvester(p *Player, it *inventory.Item) bool {
	if !s.manor.enabled {
		return false
	}
	refs := normalSkills(it.Template())
	if len(refs) == 0 {
		s.log.Warn("game: the harvester has no skill", append(p.logArgs(), "item", it.ItemID())...)
		return false
	}
	n, ok := p.Target.(*Npc)
	if !ok || n == nil || n.class == nil || !n.class.monster || !n.IsDead() {
		p.SendSystemMessage(manorMsgInvalidTarget)
		p.SendActionFailed()
		return false
	}
	for _, ref := range refs {
		if sk, ok := s.Skill(ref.ID, ref.Level); ok {
			s.UseMagic(p, sk, false, false)
		}
	}
	return true
}

// Sow implements skills.ManorWorld (Mobius Sow.onStart): on a living monster
// that can be sown, not sown yet, whose seed p chose, one seed is used up;
// with SowSucceeds the monster is sown for p (its harvest is set) with a
// sound and "The seed was successfully sown.", else "The seed was not
// sown.", to p's party or to p; then the monster's AI goes idle.
func (w skillWorld) Sow(effector, effected skills.Creature) {
	s := w.s
	p, ok1 := s.gameCreature(effector).(*Player)
	n, ok2 := s.gameCreature(effected).(*Npc)
	if !ok1 || !ok2 || p == nil || n == nil {
		return
	}
	st := s.manorSeedOf(n)
	if n.IsDead() || !n.Template.Status.CanBeSown || st == nil || st.sown || st.seeder != p.ObjectID() {
		return
	}
	if !s.DestroyItemByItemID(p, int32(st.seed.SeedID), 1, false) { //nolint:gosec // item IDs
		return
	}
	msg := manorMsgNotSown
	if manor.SowSucceeds(s.rnd, st.seed, p.Level(), n.Level()) {
		p.Send(serverpackets.NewPlaySoundPacket(serverpackets.PlaySound{Sound: "ItemSound.quest_itemget"}))
		st.sown = true
		st.harvest = manor.HarvestCount(st.seed, n.Level(), npcSkillIDs(n), s.manor.settings.DropRate)
		msg = manorMsgSown
	}
	if pt := p.Party(); pt != nil {
		pt.Broadcast(clanSysMsg(msg))
	} else {
		p.SendSystemMessage(msg)
	}
	s.npcSetIdle(n)
}

// npcSkillIDs returns the IDs of the skills of n's template.
func npcSkillIDs(n *Npc) []int {
	ids := make([]int, 0, len(n.Template.Skills))
	for _, sk := range n.Template.Skills {
		if !slices.Contains(ids, sk.ID) {
			ids = append(ids, sk.ID)
		}
	}
	return ids
}

// Harvest implements skills.ManorWorld (Mobius Harvesting.onStart): only the
// sower harvests ("You are not authorized to harvest."), only a sown
// monster ("The harvest failed because the seed was not sown."); with
// HarvestSucceeds p takes the crops (once), with "You have obtained ..." and
// "... harvested ..." to the rest of the party, else "The harvest has
// failed.".
func (w skillWorld) Harvest(effector, effected skills.Creature) {
	s := w.s
	p, ok1 := s.gameCreature(effector).(*Player)
	n, ok2 := s.gameCreature(effected).(*Npc)
	if !ok1 || !ok2 || p == nil || n == nil {
		return
	}
	st := s.manorSeedOf(n)
	switch {
	case st == nil || st.seeder != p.ObjectID():
		p.SendSystemMessage(manorMsgNotAuthorized)
	case !st.sown:
		p.SendSystemMessage(manorMsgHarvestNotSown)
	case !manor.HarvestSucceeds(s.rnd, p.Level(), n.Level()):
		p.SendSystemMessage(manorMsgHarvestFailed)
	case st.harvest > 0:
		count, cropID := st.harvest, int32(st.seed.CropID) //nolint:gosec // item IDs
		st.harvest = 0
		if _, err := s.AddItem(p, cropID, count, -1, true, false); err != nil {
			s.log.Error("game: cannot give a harvest", append(p.logArgs(), "item", cropID, "count", count, "err", err)...)
			return
		}
		if pt := p.Party(); pt != nil {
			name := serverpackets.TextParam(p.Name())
			if count == 1 {
				pt.BroadcastExcept(p, clanSysMsg(manorMsgHarvestedS2, name, s.itemNameParam(cropID)))
			} else {
				pt.BroadcastExcept(p, clanSysMsg(manorMsgHarvestedS3S2, name, serverpackets.NumberParam(itemCount32(count)), s.itemNameParam(cropID)))
			}
		}
	}
}

// manorFilterDrops is the sown part of NpcTemplate.calculateDrops: a sown
// monster drops only adena and seal stones.
func (s *Server) manorFilterDrops(n *Npc, drops []stats.ItemDrop) []stats.ItemDrop {
	if !s.manorSown(n) {
		return drops
	}
	return slices.DeleteFunc(drops, func(d stats.ItemDrop) bool { return !manor.KeptWhenSeeded(d.ItemID) })
}

// manorCorpseExtension is the longer corpse of a sown monster
// (DecayTaskManager: SpoiledCorpseExtendTime for a spoiled or seeded one).
func (s *Server) manorCorpseExtension(n *Npc) time.Duration {
	if !s.manorSown(n) {
		return 0
	}
	return time.Duration(s.settings.NPCs.SpoiledCorpseExtendTime)
}
