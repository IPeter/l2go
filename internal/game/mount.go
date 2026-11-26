package game

import (
	"errors"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// Riding striders and wyverns (Mobius Player.mount, setMount, dismount and
// the mount's feed: startFeed, PetFeedTask, setCurrentFeed, isHungry,
// storePetFood):
//
//   - mount.go: the rider's state, getting on and off, the speeds, the feed
//     and what a rider may not do (skills, equipment, items);
//   - pet_ride.go: action 38 (Player.mountPlayer) on a strider pet;
//   - mount_zone.go: the no-landing and siege zones;
//   - mount_gm.go: //ride_*, //unride*, /dismount and the Lua calls.
//
// A rider's state is not saved: a player who leaves on a mount comes back on
// foot, as in Mobius. Deviations: the level data of the feed follows the
// current mount (Mobius caches the first mount's), and a mount without food
// (//ride_*, mount(npcId, 0, false)) is never hungry.
func init() { registerFeature("mount", setupMounts) }

// The mount NPCs and skills (Mobius CategoryData STRIDER and WYVERN_GROUP,
// CommonSkill).
const (
	mountNpcStrider          = 12526 // wind strider, //ride_strider
	mountNpcWyvern           = 12621
	skillWyvernBreath        = 4289
	skillStriderSiegeAssault = 325
	skillWyvernAegis         = 327
)

// Mount rules (Mobius Player, PlayerStat, NoLandingZone, SiegeZone and
// Feature.ini).
const (
	mountFeedTick = 10 * time.Second
	// mountRange is how close a strider must be to be ridden, beyond both
	// collision radii (checkIfInRange(200, ...)).
	mountRange = 200
	// mountLevelGap: a mount this many levels above its rider halves the
	// rider's speeds.
	mountLevelGap = 10
	// mountMaxAltitude and mountMaxElevation: a non-GM may not get off above
	// z 10,000, nor 300 above the ground outside a no-landing zone.
	mountMaxAltitude  = 10000
	mountMaxElevation = 300
	// mountDismountDelay is NoLandingZone's default dismountDelay and
	// SiegeZone's DISMOUNT_DELAY, in seconds.
	mountDismountDelay = 5
	// mountAllowDuringSiege is AllowRideMountsDuringSiege.
	mountAllowDuringSiege = false
	// mountWaterInfoDelay is when a rider who got off in water gets its
	// UserInfo again.
	mountWaterInfoDelay = 1500 * time.Millisecond
)

// System messages of mounts (Mobius SystemMessageId).
const (
	mountMsgPetSummonedNoDestroy    serverpackets.SystemMessageID = 557  // "Your pet has been summoned and may not be destroyed."
	mountMsgHungryStrider           serverpackets.SystemMessageID = 1008 // "A hungry strider cannot be mounted or dismounted."
	mountMsgStriderDeadRider        serverpackets.SystemMessageID = 1009 // "A strider cannot be ridden when dead."
	mountMsgDeadStrider             serverpackets.SystemMessageID = 1010 // "A dead strider cannot be ridden."
	mountMsgStriderInBattle         serverpackets.SystemMessageID = 1011 // "A strider in battle cannot be ridden."
	mountMsgRiderInBattle           serverpackets.SystemMessageID = 1012 // "A strider cannot be ridden while in battle."
	mountMsgOnlyStanding            serverpackets.SystemMessageID = 1013 // "A strider can be ridden only when standing."
	mountMsgCannotDismountElevation serverpackets.SystemMessageID = 1158 // "You cannot dismount from this elevation."
	mountMsgOutOfFeed               serverpackets.SystemMessageID = 1248 // "You are out of feed. Mount status canceled."
	mountMsgNotAllowedToDismount    serverpackets.SystemMessageID = 1385 // "You are not allowed to dismount at this location."
	mountMsgFishing                 serverpackets.SystemMessageID = 1470 // "You cannot do that while fishing."
	mountMsgWyvernNoEntry           serverpackets.SystemMessageID = 1687 // "This area cannot be entered while mounted atop of a Wyvern. ..."
	mountMsgTooFarFromStrider       serverpackets.SystemMessageID = 1846 // "You are too far away from the Strider to mount it."
)

// sysMsgPetSummonedNoDestroy is RequestDestroyItem's refusal for a collar in
// use (items_use.go).
const sysMsgPetSummonedNoDestroy = mountMsgPetSummonedNoDestroy

// playerMount is what a player rides (Mobius _mountType, _mountNpcId,
// _mountLevel, _mountObjectID, isFlying and the feed fields).
type playerMount struct {
	typ   int32 // serverpackets.MountNone, MountStrider or MountWyvern
	npcID int
	level int
	// control is the collar of the ridden strider (Mobius _mountObjectID),
	// 0 for a mount without one.
	control int32
	// flying is set on a wyvern (Mobius setFlying).
	flying bool

	// The feed (Mobius _canFeed, _curFeed, _controlItemId, _mountFeedTask).
	canFeed     bool
	feed        int
	feedControl int32
	feedTimer   world.TimerID

	// landTimer is the dismount of a wyvern over a no-landing zone (Mobius
	// _dismountTask).
	landTimer world.TimerID
}

// setupMounts registers the rider's hooks, the GM ride commands and
// /dismount (mount_gm.go).
func setupMounts(s *Server) error {
	s.OnDie(func(victim, _ Creature) {
		if p, ok := victim.(*Player); ok && s.mounted(p) {
			s.mountStopFeed(p) // Player.doDie
		}
	})
	s.OnPlayerLeave(s.mountLeave)
	return errors.Join(s.registerMountCommands()...)
}

// mounted is Mobius isMounted.
func (s *Server) mounted(p *Player) bool { return p.mount.typ != serverpackets.MountNone }

// mountLeave is Player.stopAllTimers and deleteMe for a rider: the feed
// stops and goes to the strider's row; the no-landing timer stops.
func (s *Server) mountLeave(p *Player) {
	s.mountStopFeed(p)
	s.mountStorePetFood(p, p.mount.npcID)
	s.mountCancelLanding(p)
}

// ----- getting on and off -----

// setMount is Mobius setMount: a wyvern flies and gets Wyvern Breath; off a
// wyvern the breath goes. The rider's speeds and collision follow.
func (s *Server) setMount(p *Player, npcID, level int) {
	m := &p.mount
	typ := petMountType(npcID)
	switch typ {
	case serverpackets.MountNone:
		if m.flying {
			s.RemoveSkill(p, skillWyvernBreath, false, true)
			m.flying = false
			p.SendSkillList()
		}
	case serverpackets.MountWyvern:
		m.flying = true
		if sk, ok := s.Skill(skillWyvernBreath, 1); ok {
			s.AddSkill(p, sk, false)
		}
		p.SendSkillList()
	}
	m.typ, m.npcID, m.level = typ, npcID, level
	s.mountStats(p)
}

// mountNpc is Mobius mount(npcId, controlItemObjId, useFood): the weapon and
// shield come off (false when they cannot), the toggles end, the player
// rides the NPC at its own level and everyone sees it; with useFood the
// mount eats.
func (s *Server) mountNpc(p *Player, npcID int, control int32, useFood bool) bool {
	if !s.mountDisarm(p) {
		return false
	}
	p.skill.el.StopAllToggles(true)
	if !useFood {
		p.mount.canFeed = false
	}
	s.setMount(p, npcID, p.Level())
	p.mount.control = control
	s.world.Broadcast(p, ridePacket(p), true)
	p.BroadcastUserInfo()
	if useFood {
		s.mountStartFeed(p, npcID, nil)
	}
	return true
}

// dismount is Mobius dismount: refused for a non-GM above z 10,000 or 300
// above the ground outside a no-landing zone (unless water lies below);
// otherwise the feed gauge closes, the mount and its feed go, the skill list
// and Ride follow, the strider's feed goes to its row and everyone sees the
// rider on foot.
func (s *Server) dismount(p *Player) bool {
	pos := p.Position()
	below := pos
	below.Z -= mountMaxElevation
	if s.zoneAt(below, "WaterZone") == nil {
		if !p.IsGM() && !p.Zones.Inside(ZoneWater) && pos.Z > mountMaxAltitude {
			p.SendSystemMessage(mountMsgNotAllowedToDismount)
			p.SendActionFailed()
			return false
		}
		if !p.Zones.Inside(ZoneNoLanding) && s.geo.Height(pos.X, pos.Y, pos.Z)+mountMaxElevation < pos.Z {
			p.SendSystemMessage(mountMsgCannotDismountElevation)
			p.SendActionFailed()
			return false
		}
	} else {
		p.after(mountWaterInfoDelay, func() {
			if p.Zones.Inside(ZoneWater) {
				p.BroadcastUserInfo()
			}
		})
	}
	m := &p.mount
	p.Send(serverpackets.NewSetupGaugePacket(serverpackets.SetupGauge{Color: serverpackets.GaugeGreen}))
	petID := m.npcID
	s.setMount(p, 0, 0)
	s.mountStopFeed(p)
	s.mountCancelLanding(p)
	p.SendSkillList()
	s.world.Broadcast(p, ridePacket(p), true)
	m.control = 0
	s.mountStorePetFood(p, petID)
	p.BroadcastUserInfo()
	return true
}

// ridePacket is Ride of p's mount (Mobius Ride: the NPC ID offset even on
// foot).
func ridePacket(p *Player) []byte {
	pos := p.Position()
	m := &p.mount
	return serverpackets.NewRidePacket(serverpackets.Ride{
		ObjectID:   p.ObjectID(),
		Mounted:    m.typ != serverpackets.MountNone,
		MountType:  m.typ,
		MountNpcID: int32(m.npcID), //nolint:gosec // NPC IDs are small
		X:          pos.X, Y: pos.Y, Z: pos.Z,
	})
}

// mountDisarm is Mobius disarmWeapons and disarmShield: the weapon comes off
// unless it is a cursed or a force-equipped one (false then), the shield
// (or whatever the left hand holds) too.
func (s *Server) mountDisarm(p *Player) bool {
	if w := p.inv.Paperdoll(storage.PaperdollRHand); w != nil {
		if p.CursedWeaponEquipped() {
			return false
		}
		if wt := w.Template().Weapon; wt != nil && wt.ForceEquip {
			return false
		}
		s.mountTakeOff(p, w)
	}
	if sh := p.inv.Paperdoll(storage.PaperdollLHand); sh != nil {
		s.mountTakeOff(p, sh)
	}
	return true
}

// mountTakeOff takes an item off for disarm: InventoryUpdate, the attack
// stops, UserInfo, and "$s1 has been disarmed." (or the enchanted form).
func (s *Server) mountTakeOff(p *Player, it *inventory.Item) {
	off := p.inv.UnequipItem(it)
	s.sendInventoryUpdateNow(p, s.takeInventoryUpdates(p, nil))
	s.AbortAttack(p)
	s.refreshExpertisePenalty(p) // penalty.go, as any unequip
	p.BroadcastUserInfo()
	s.RequestSave(p)
	if len(off) == 0 {
		return
	}
	first := off[0]
	name := s.itemNameParam(first.ItemID())
	if first.Enchant() > 0 {
		p.SendSystemMessage(sysMsgEquipmentRemoved, serverpackets.NumberParam(first.Enchant()), name)
	} else {
		p.SendSystemMessage(sysMsgDisarmed, name)
	}
}

// mountStats puts the mount into p's stat set (PlayerStat.getBaseMoveSpeed
// and the run and walk penalties, getCollisionRadius and Height) and the
// wyvern's flight into its motion.
func (s *Server) mountStats(p *Player) {
	m := &p.mount
	motion := p.stats.Motion()
	motion.Flying = m.flying
	p.stats.SetMotion(motion)
	if m.typ == serverpackets.MountNone {
		p.stats.SetMount(nil)
		return
	}
	sm := &stats.Mount{}
	if s.worldData != nil {
		if t, ok := s.worldData.Npc(m.npcID); ok {
			sm.CollisionRadius = float64(float32(t.Collision.Radius))
			sm.CollisionHeight = float64(float32(t.Collision.Height))
		}
	}
	if _, l := s.mountPetData(p); l != nil {
		sm.OnRide = true
		sm.Walk, sm.Run, sm.SlowSwim, sm.FastSwim = l.WalkOnRide, l.RunOnRide, l.SlowSwimOnRide, l.FastSwimOnRide
	}
	if m.level-p.Level() >= mountLevelGap {
		sm.Penalties++
	}
	if s.mountHungry(p) {
		sm.Penalties++
	}
	p.stats.SetMount(sm)
}

// ----- the feed -----

// mountPetData is the pet data of p's mount and its level's stats (Mobius
// PetDataTable.getPetData and getPetLevelData), nil without.
func (s *Server) mountPetData(p *Player) (*data.PetData, *data.PetLevel) {
	m := &p.mount
	if m.npcID == 0 || s.summons.data == nil {
		return nil, nil
	}
	pd, ok := s.summons.data.Pet(m.npcID)
	if !ok {
		return nil, nil
	}
	l, _ := pd.Level(m.level)
	return pd, l
}

// mountMaxFeed is getMaxFeed.
func (s *Server) mountMaxFeed(p *Player) int {
	if _, l := s.mountPetData(p); l != nil {
		return l.MaxMeal
	}
	return 0
}

// mountFeedConsume is getFeedConsume: the battle meal while attacking.
func (s *Server) mountFeedConsume(p *Player) int {
	_, l := s.mountPetData(p)
	if l == nil {
		return 0
	}
	if s.AttackingNow(p) {
		return l.MealInBattle
	}
	return l.MealInNormal
}

// mountHungry is Player.isHungry: a feeding mount below its hungry limit.
func (s *Server) mountHungry(p *Player) bool {
	m := &p.mount
	if !m.canFeed {
		return false
	}
	pd, l := s.mountPetData(p)
	if pd == nil || l == nil {
		return false
	}
	return float32(m.feed) < float32(pd.HungryLimit)/100*float32(l.MaxMeal)
}

// mountGauge is the green feed gauge (SetupGauge 3, in consume units).
func (s *Server) mountGauge(p *Player) {
	consume := s.mountFeedConsume(p)
	if consume <= 0 {
		return
	}
	cur := p.mount.feed * 10000 / consume
	full := s.mountMaxFeed(p) * 10000 / consume
	p.Send(serverpackets.NewSetupGaugePacket(serverpackets.SetupGauge{
		Color:   serverpackets.GaugeGreen,
		Time:    int32(cur),  //nolint:gosec // Java int arithmetic
		MaxTime: int32(full), //nolint:gosec // as above
	}))
}

// mountSetFeed is setCurrentFeed: the feed, at most the maximum, and its
// gauge; a mount that turned hungry or full changes the rider's speeds.
func (s *Server) mountSetFeed(p *Player, n int) {
	p.mount.feed = min(n, s.mountMaxFeed(p))
	s.mountGauge(p)
	s.mountStats(p)
	p.RefreshStats()
}

// mountStartFeed is startFeed: a mount from a pet (sm) takes the pet's feed
// and collar, another one is full; the gauge shows it and, while the rider
// lives, it eats every 10 s.
func (s *Server) mountStartFeed(p *Player, npcID int, sm *Summon) {
	m := &p.mount
	m.canFeed = npcID > 0
	if !s.mounted(p) {
		return
	}
	switch {
	case sm != nil && sm.pet != nil:
		s.mountSetFeed(p, sm.pet.fed)
		m.feedControl = sm.pet.control
	case m.canFeed:
		s.mountSetFeed(p, s.mountMaxFeed(p))
	default:
		return
	}
	s.mountGauge(p)
	if !p.IsDead() {
		s.mountStopFeed(p)
		m.feedTimer = s.loop.Every(mountFeedTick, func() { s.mountFeedTick(p) })
	}
}

// mountStopFeed is stopFeed.
func (s *Server) mountStopFeed(p *Player) {
	if m := &p.mount; m.feedTimer != 0 {
		s.loop.Cancel(m.feedTimer)
		m.feedTimer = 0
	}
}

// mountFeedTick is Mobius PetFeedTask: the mount eats its meal; out of feed
// the rider gets off (a wyvern rider goes to town) and is told "You are out
// of feed. Mount status canceled."; a hungry mount eats the first of its
// foods the rider carries ("Your pet was hungry so it ate $s1.").
func (s *Server) mountFeedTick(p *Player) {
	m := &p.mount
	pd, _ := s.mountPetData(p)
	if !p.InGame() || !s.mounted(p) || pd == nil {
		s.mountStopFeed(p)
		return
	}
	if consume := s.mountFeedConsume(p); m.feed > consume {
		s.mountSetFeed(p, m.feed-consume)
	} else {
		s.mountSetFeed(p, 0)
		s.mountStopFeed(p)
		wyvern := m.flying && m.typ == serverpackets.MountWyvern
		s.dismount(p)
		if wyvern {
			s.teleportToTown(p)
		}
		p.SendSystemMessage(mountMsgOutOfFeed)
		return
	}
	var food *inventory.Item
	for _, id := range pd.Food {
		if food = p.inv.ByItemID(int32(id)); food != nil { //nolint:gosec // item IDs are small
			break
		}
	}
	if food != nil && s.mountHungry(p) {
		id := food.ItemID()
		s.mountEat(p, food)
		p.SendSystemMessage(sumMsgPetWasHungryAte, serverpackets.ItemNameParam(id))
	}
}

// mountEat is the PetFood handler for a player: a rider whose mount eats the
// food uses one up, shows the item's skill and the feed grows by the
// skill's feed; anyone else is told "$s1 cannot be used due to unsuitable
// terms." for each of the item's skills.
func (s *Server) mountEat(p *Player, it *inventory.Item) bool {
	id := it.ItemID()
	ate := false
	for _, ref := range normalSkills(it.Template()) {
		sk, ok := s.Skill(ref.ID, ref.Level)
		if !ok {
			continue
		}
		if pd, _ := s.mountPetData(p); s.mounted(p) && pd != nil && canEatFood(pd.Food, int(id)) && s.DestroyItem(p, it, 1, false) {
			s.world.Broadcast(p, castUsePacket(p, p, sk.ID, sk.Level, 0, 0), true)
			s.mountSetFeed(p, p.mount.feed+sk.Feed)
			ate = true
			continue
		}
		p.SendSystemMessage(sumMsgUnsuitableTerms, serverpackets.ItemNameParam(id))
	}
	return ate
}

// mountStorePetFood is storePetFood: the feed goes to the row of the
// strider pet the rider got on (once).
func (s *Server) mountStorePetFood(p *Player, petID int) {
	m := &p.mount
	if m.feedControl == 0 || petID == 0 {
		return
	}
	if o := s.summons.owners[p]; o != nil {
		if row := o.pets[m.feedControl]; row != nil {
			row.Fed = int32(m.feed) //nolint:gosec // a meal
			s.writePet(*row)
		}
	}
	m.feedControl = 0
}

// mountRevived is Player.doRevive for a rider: the mount eats again.
func (s *Server) mountRevived(p *Player) {
	if s.mounted(p) {
		s.mountStartFeed(p, p.mount.npcID, nil)
	}
}

// ----- what a rider may not do -----

// mountSkillAllowed is the mount part of checkUseMagicConditions and the
// skill list: on a strider only magic skills and Strider Siege Assault, on
// a wyvern only Wyvern Breath and Wyvern Aegis.
func mountSkillAllowed(p *Player, sk *skills.Skill) bool {
	switch p.mount.typ {
	case serverpackets.MountStrider:
		return sk.IsMagic() || sk.ID == skillStriderSiegeAssault
	case serverpackets.MountWyvern:
		return sk.ID == skillWyvernBreath || sk.ID == skillWyvernAegis
	}
	return true
}

// mountBlocksEquip is UseItem's check: a rider may not put on or take off a
// weapon or shield ("You do not meet the required condition to equip that
// item.").
func (s *Server) mountBlocksEquip(p *Player, it *inventory.Item) bool {
	if !s.mounted(p) {
		return false
	}
	switch it.Template().Part() {
	case data.BodyPartLRHand, data.BodyPartLHand, data.BodyPartRHand:
		p.SendSystemMessage(sysMsgEquipConditionNotMet)
		return true
	}
	return false
}

// collarInUse is the collar part of Mobius checkItemManipulation and
// RequestDestroyItem: the collar of the pet out or of the strider ridden.
func (s *Server) collarInUse(p *Player, objectID int32) bool {
	if objectID == 0 {
		return false
	}
	if sm := s.SummonOf(p); sm != nil && sm.pet != nil && sm.pet.control == objectID {
		return true
	}
	return p.mount.control == objectID
}

// mountEnchantEffect is the weapon glow of the info packets: none while
// mounted.
func (p *Player) mountEnchantEffect() uint8 {
	if p.mount.typ != serverpackets.MountNone {
		return 0
	}
	return p.enchantEffect()
}

// mountDropCursed is CursedWeapon.activate for a rider who may not get off:
// "You have failed to pick up $s1." and the weapon falls again.
func (s *Server) mountDropCursed(p *Player, it *inventory.Item) {
	p.SendSystemMessage(sysMsgFailedPickUpS1, s.itemNameParam(it.ItemID()))
	rt := s.cursed.rt[it.ItemID()]
	if it.Equipped() {
		p.inv.UnequipItem(it)
	}
	dropped, err := p.inv.DropItem(it)
	if err != nil {
		return
	}
	g, err := s.DropOnGround(dropped, p.Position(), GroundDrop{Dropper: p, PlayerDrop: true})
	if err != nil {
		s.log.Error("game: a cursed weapon cannot fall on the ground", append(p.logArgs(), "item", it.ItemID(), "err", err)...)
	}
	if rt != nil {
		rt.ground = g
		rt.setItem(dropped)
	}
	s.itemsChanged(p, true, nil)
}

// RidingStrider is ConditionPlayerIsRidingStrider's test
// (skills.PlayerMount).
func (a *skillPlayer) RidingStrider() bool {
	return a.p != nil && a.p.mount.typ == serverpackets.MountStrider
}

var _ skills.PlayerMount = (*skillPlayer)(nil)
