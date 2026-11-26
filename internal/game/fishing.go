package game

import (
	"errors"
	"time"

	"github.com/IPeter/l2go/internal/augment"
	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/fishing"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// Fishing (Plan 14, M11g): the rules of internal/fishing on the loop.
//
//   - The Fishing skill (1312) throws the bait (castFishing, Mobius
//     handlers/skill/effects/Fishing): rod and lure worn, not under water,
//     not on a boat or in a store; the bait lands where a fishing zone and a
//     water zone meet, at the water's top, in sight; one lure is used.
//   - The player stands still and waits (Player.startFishing): the lure and
//     the fishing level pick the fish; a lure of the wrong time of day never
//     gets a bite. Bite checks start after 10 s (LookingForFishTask).
//   - A bite starts the fight (mechanics/fishing/Fishing): one tick a
//     second; Pumping (1313) and Reeling (1314) hurt the fish
//     (fishingPull). A landed fish goes into the inventory and the weekly
//     championship (fishing_champ.go); at a player's level the catch may be
//     a fishing monster instead.
//   - While fishing, only fishing skills and fishing shots may be used, and
//     the player can neither walk, drop items nor play social actions.
//   - The fishermen teach the fishing skills and show the championship
//     (fishing_npc.go); fishing shots charge the rod (fishing_shot.go).

func init() { registerFeature("fishing", setupFishing) }

// fishingFeature is the fishing state on the Server (Server.fishing).
// Loop-only, except data and store, which New sets.
type fishingFeature struct {
	data   *data.Fishing        // nil: no fishing data
	store  storage.FishingStore // nil keeps the championship in memory only
	tables *fishing.Tables
	cfg    fishingConfig
	// rnd is the game's random source (Server.Rand); tests may give
	// fishing its own.
	rnd rnd.Source

	// states are the players fishing now (Player._fishing).
	states map[*Player]*fishState
	// skillTree is the fishing skill tree in the order Mobius iterates it.
	skillTree []data.SkillLearn

	champ      *fishing.Championship
	champTimer world.TimerID
	writer     *fishingWriter
}

// fishingConfig are the fishing settings of Mobius General.ini, from the
// config's fishing section (fishingConfigOf).
type fishingConfig struct {
	// allow is AllowFishing (true); GMs fish anyway.
	allow bool
	champ fishing.ChampionshipConfig
}

// fishingConfigOf returns the settings of the config's fishing section.
func fishingConfigOf(c config.Fishing) fishingConfig {
	return fishingConfig{
		allow: c.AllowFishing,
		champ: fishing.ChampionshipConfig{
			Enabled:    c.ChampionshipEnabled,
			RewardItem: c.ChampionshipItem,
			Rewards:    c.ChampionshipRewards(),
		},
	}
}

// fishState is a player's fishing: the bait, the fish and the fight.
type fishState struct {
	x, y, z int32 // where the bait is (Player._fishX, _fishY, _fishZ)
	lure    *data.Lure
	lureID  int32
	fish    *data.Fish
	// group is the fish group ExFishingStart showed: -1 never bites.
	group int
	// look is the bite check timer, until its deadline.
	look     world.TimerID
	deadline time.Time
	fight    *fishing.Fight
	tick     world.TimerID
}

// setupFishing builds the fish tables and the fishing skill tree, restores
// the championship and registers the hooks.
func setupFishing(s *Server) error {
	f := &s.fishing
	f.cfg = fishingConfigOf(s.gs.Fishing)
	f.rnd = s.rnd
	f.states = make(map[*Player]*fishState)
	if f.data != nil {
		f.tables = fishing.NewTables(f.data)
	}
	if d := s.skills.data; d != nil {
		file := d.OtherSkillTree(data.TreeFishing)
		var order chmOrder
		byHash := make(map[int]int, len(file))
		for i := range file {
			h := file[i].SkillID*1021 + file[i].Level // SkillData.getSkillHashCode
			byHash[h] = i
			order.put(h)
		}
		for _, h := range order.keys() {
			f.skillTree = append(f.skillTree, file[byHash[h]])
		}
	}
	s.OnPlayerLeave(s.fishingLeave)
	// A fishing player cannot augment (AbstractRefinePacket.isValid).
	s.OnAugmentCheck(func(p *Player, st *augment.Player) { st.Fishing = s.Fishing(p) })
	return errors.Join(
		s.setupFishingChampionship(), // fishing_champ.go
		s.RegisterEx(clientpackets.ExOpRequestExFishRanking, s.handleRequestExFishRanking),
		s.registerFishingGM(), // fishing_gm.go
	)
}

// Fishing reports whether p is fishing (Mobius isFishing). Loop-only.
func (s *Server) Fishing(p *Player) bool { return s.fishing.states[p] != nil }

// fishingSpot returns where p's bait is for UserInfo and CharInfo.
func (p *Player) fishingSpot() (on bool, x, y, z int32) {
	if p.srv == nil {
		return false, 0, 0, 0
	}
	st := p.srv.fishing.states[p]
	if st == nil {
		return false, 0, 0, 0
	}
	return true, st.x, st.y, st.z
}

// sendFishingMessage sends a fishing system message, with its number when
// it has $s1.
func sendFishingMessage(p *Player, m fishing.Message, arg int) {
	switch m {
	case fishing.MsgPumpingSuccess, fishing.MsgPumpingFailed, fishing.MsgReelingSuccess, fishing.MsgReelingFailed,
		fishing.MsgReelingPenalty, fishing.MsgPumpingPenalty:
		p.SendSystemMessage(serverpackets.SystemMessageID(m), serverpackets.NumberParam(int32(arg))) //nolint:gosec // damage fits in int32
	default:
		p.SendSystemMessage(serverpackets.SystemMessageID(m))
	}
}

// ----- casting (Mobius handlers/skill/effects/Fishing) -----

// StartFishing implements skills.Fisher for the Fishing effect.
func (w skillWorld) StartFishing(c skills.Creature) {
	if p, ok := w.s.gameCreature(c).(*Player); ok {
		w.s.castFishing(p)
	}
}

// Pump implements skills.Fisher for the Pumping effect.
func (w skillWorld) Pump(c skills.Creature, sk *skills.Skill, power float64) {
	if p, ok := w.s.gameCreature(c).(*Player); ok {
		w.s.fishingPull(p, sk, power, false)
	}
}

// Reel implements skills.Fisher for the Reeling effect.
func (w skillWorld) Reel(c skills.Creature, sk *skills.Skill, power float64) {
	if p, ok := w.s.gameCreature(c).(*Player); ok {
		w.s.fishingPull(p, sk, power, true)
	}
}

var _ skills.Fisher = skillWorld{}

// castFishing is Mobius Fishing.onStart: casting while fishing cancels it;
// otherwise the rod, the lure, the boat, the store and the water are
// checked in Mobius's order, the bait is placed (fishing.PlaceBait), one
// lure is used and the fishing starts.
func (s *Server) castFishing(p *Player) {
	f := &s.fishing
	if !f.cfg.allow && !p.IsGM() {
		p.SendMessage("Fishing is disabled!")
		return
	}
	if st := f.states[p]; st != nil {
		if st.fight != nil {
			st.fight.Abort(fishOutput{s, p, st})
		} else {
			s.endFishing(p, false)
		}
		sendFishingMessage(p, fishing.MsgCancelled, 0)
		return
	}
	rod := p.inv.Paperdoll(storage.PaperdollRHand)
	if rod == nil || rod.Template().Weapon == nil || rod.Template().Weapon.Type != data.WeaponFishingrod {
		sendFishingMessage(p, fishing.MsgNoFishingPole, 0)
		return
	}
	lure := p.inv.Paperdoll(storage.PaperdollLHand)
	if lure == nil || lure.Template().Etc == nil || lure.Template().Etc.Type != data.EtcLure {
		sendFishingMessage(p, fishing.MsgNoBait, 0)
		return
	}
	if p.vehicleID() != 0 {
		sendFishingMessage(p, fishing.MsgOnBoat, 0)
		return
	}
	if p.InStoreMode() { // no isCrafting: without AltGameCreation it never outlasts a packet (craft.go)
		sendFishingMessage(p, fishing.MsgInStoreOrCrafting, 0)
		return
	}
	if p.Zones.Inside(ZoneWater) {
		sendFishingMessage(p, fishing.MsgUnderWater, 0)
		return
	}
	var lureData *data.Lure
	if f.data != nil {
		lureData, _ = f.data.Lure(int(lure.ItemID()))
	}
	if lureData == nil || f.tables == nil {
		s.log.Warn("game: no fishing data for the lure", append(p.logArgs(), "lure", lure.ItemID())...)
		sendFishingMessage(p, fishing.MsgNoBait, 0)
		return
	}
	pos := p.Position()
	bx, by, bz, ok := fishing.PlaceBait(f.rnd, int(pos.X), int(pos.Y), int(pos.Heading), fishSpot{s: s, p: p})
	if !ok {
		sendFishingMessage(p, fishing.MsgCannotFishHere, 0)
		return
	}
	lureID := lure.ItemID()
	if !s.DestroyItem(p, lure, 1, false) {
		sendFishingMessage(p, fishing.MsgNotEnoughBait, 0)
		return
	}
	s.startFishing(p, lureID, lureData, int32(bx), int32(by), int32(bz)) //nolint:gosec // world coordinates
}

// fishSpot is the world as fishing.PlaceBait sees it around a player.
type fishSpot struct {
	s *Server
	p *Player
}

// WaterZ is the zone part of Mobius Fishing.onStart: the zones holding (x,
// y) at their tops (ZoneManager.getZones(x, y)), in ID order, the last
// fishing and water zones seen until both are found; the bait floats at the
// water zone's top.
func (f fishSpot) WaterZ(x, y int) (int, bool) {
	ix := f.s.dialog.zones
	if ix == nil {
		return 0, false
	}
	var fishZone, water *data.Zone
	for _, i := range ix.cells[zoneCellOf(world.Position{X: int32(x), Y: int32(y)})] { //nolint:gosec // world coordinates
		z := ix.rules[i].zone
		if !z.Shape.Contains(x, y, z.Shape.HighZ()) {
			continue
		}
		switch {
		case f.s.worldData.ZoneTypeIs(z.Type, "FishingZone"):
			fishZone = z
		case f.s.worldData.ZoneTypeIs(z.Type, "WaterZone"):
			water = z
		}
		if fishZone != nil && water != nil {
			break
		}
	}
	if fishZone == nil || water == nil {
		return 0, false
	}
	return water.Shape.HighZ(), true
}

// Clear is computeBaitZ's geodata part: the player sees the bait, and where
// there is geodata no ground lies above the bait, neither at its height nor
// at the player's.
func (f fishSpot) Clear(x, y, z int) bool {
	g := f.s.geo
	pos := f.p.Position()
	bx, by, bz := int32(x), int32(y), int32(z) //nolint:gosec // world coordinates
	if !g.CanSeeTarget(geoPoint(pos), geo.Point{X: bx, Y: by, Z: bz}) {
		return false
	}
	if g.HasGeo(bx, by) && (g.Height(bx, by, bz) > bz || g.Height(bx, by, pos.Z) > bz) {
		return false
	}
	return true
}

// fishingLevel is the fishing level of getRandomFishLvl: Fishing
// Expertise's level, or the power of an active Fisherman's Potion.
func (s *Server) fishingLevel(p *Player) int {
	level := p.SkillLevel(fishing.SkillFishingExpertise)
	if p.skill != nil {
		if info := p.skill.Effects().BySkillID(fishing.SkillFishermansPotion); info != nil {
			level = int(info.Skill().Power)
		}
	}
	return level
}

// startFishing is Mobius Player.startFishing: p stops and may not move, the
// fish is hooked (fishing.Tables.Hook), "You cast your line and start to
// fish.", ExFishingStart to everyone around (the fish group, -1 when the
// lure does not fit the time of day), PlaySound SF_P_01, and the bite
// checks.
func (s *Server) startFishing(p *Player, lureID int32, lure *data.Lure, x, y, z int32) {
	f := &s.fishing
	s.stopWalk(p, s.motionOf(p), nil)
	if p.skill != nil {
		p.skill.SetImmobilized(true)
	}
	st := &fishState{x: x, y: y, z: z, lure: lure, lureID: lureID}
	f.states[p] = st
	fish, ok := f.tables.Hook(f.rnd, lure, s.fishingLevel(p))
	if !ok {
		p.SendMessage("Error - Fish are not defined")
		s.endFishing(p, false)
		return
	}
	st.fish = fish
	sendFishingMessage(p, fishing.MsgStartFishing, 0)
	st.group = fishing.ShownGroup(fish, lure, s.world.GameTime().IsNight())
	s.world.Broadcast(p, serverpackets.NewExFishingStartPacket(serverpackets.ExFishingStart{
		ObjectID: p.ObjectID(), FishType: int32(st.group), X: x, Y: y, Z: z, Night: lure.Night, //nolint:gosec // a group
	}), true)
	p.Send(serverpackets.NewPlaySoundPacket(serverpackets.PlaySound{Type: 1, Sound: "SF_P_01"}))
	s.startLookingForFish(p, st)
}

// startLookingForFish is startLookingForFishTask: the first bite check
// after 10 s, then one every CheckDelay, until the fish's wait limit.
func (s *Server) startLookingForFish(p *Player, st *fishState) {
	if p.dead {
		return
	}
	st.deadline = s.Now().Add(time.Duration(fishing.WaitLimit(st.fish)) * time.Millisecond)
	every := time.Duration(fishing.CheckDelay(st.fish, st.lure)) * time.Millisecond
	st.look = s.loop.After(fishing.FirstCheck*time.Millisecond, func() {
		st.look = 0
		if s.lookForFish(p, st) {
			st.look = s.loop.Every(every, func() { s.lookForFish(p, st) })
		}
	})
}

// lookForFish is LookingForFishTask.run: past the deadline the fishing
// ends; a fish of group -1 never bites; otherwise a bite (fishing.Bites)
// starts the fight. It reports whether the checks go on.
func (s *Server) lookForFish(p *Player, st *fishState) bool {
	if s.fishing.states[p] != st || st.fight != nil {
		return false
	}
	if !s.Now().Before(st.deadline) {
		s.endFishing(p, false)
		return false
	}
	if st.group == -1 {
		return true
	}
	if fishing.Bites(s.fishing.rnd, st.fish) {
		s.stopLookingForFish(st)
		s.startFishCombat(p, st)
		return false
	}
	return true
}

// stopLookingForFish is stopLookingForFishTask.
func (s *Server) stopLookingForFish(st *fishState) {
	if st.look != 0 {
		s.loop.Cancel(st.look)
		st.look = 0
	}
}

// startFishCombat is Player.startFishCombat and the Fishing constructor:
// every night lure fights like an upper-grade fish; ExFishingStartCombat to
// everyone around, PlaySound SF_S_01, "You've got a bite!", and the fight's
// one-second ticks.
func (s *Server) startFishCombat(p *Player, st *fishState) {
	noob := st.fish.Grade == data.FishGradeEasy
	upper := st.fish.Grade == data.FishGradeHard || st.lure.Night
	fight, c := fishing.NewFight(s.fishing.rnd, st.fish, noob, upper, int(st.lureID))
	st.fight = fight
	s.world.Broadcast(p, serverpackets.NewExFishingStartCombatPacket(serverpackets.ExFishingStartCombat{
		ObjectID: p.ObjectID(), Time: int32(c.Time), HP: int32(c.HP), //nolint:gosec // seconds and fish HP
		Mode: uint8(c.Mode), LureType: uint8(c.LureType), Deceptive: uint8(c.Deceptive), //nolint:gosec // 0-2
	}), true)
	p.Send(serverpackets.NewPlaySoundPacket(serverpackets.PlaySound{Type: 1, Sound: "SF_S_01"}))
	sendFishingMessage(p, fishing.MsgBite, 0)
	st.tick = s.loop.Every(time.Second, func() {
		if s.fishing.states[p] == st && !fight.IsOver() {
			fight.Tick(s.fishing.rnd, fishOutput{s, p, st})
		}
	})
}

// fishOutput sends what a fight does to its player (fishing.Output).
type fishOutput struct {
	s  *Server
	p  *Player
	st *fishState
}

// Message sends a fishing system message.
func (o fishOutput) Message(m fishing.Message, arg int) { sendFishingMessage(o.p, m, arg) }

// Regen sends ExFishingHpRegen to the fisher, or to everyone around.
func (o fishOutput) Regen(r fishing.Regen, broadcast bool) {
	pkt := serverpackets.NewExFishingHpRegenPacket(serverpackets.ExFishingHpRegen{
		ObjectID: o.p.ObjectID(), Time: int32(r.Time), HP: int32(r.HP), //nolint:gosec // seconds and fish HP
		Mode: uint8(r.Mode), GoodUse: uint8(r.GoodUse), Anim: uint8(r.Anim), //nolint:gosec // 0-2
		Penalty: int32(r.Penalty), BarColor: uint8(r.Deceptive), //nolint:gosec // small
	})
	if broadcast {
		o.s.world.Broadcast(o.p, pkt, true)
		return
	}
	o.p.Send(pkt)
}

// Over is Fishing.doDie.
func (o fishOutput) Over(win bool) { o.s.fishDie(o.p, o.st, win) }

// fishDie is Mobius Fishing.doDie: the ticks stop; a landed fish is, with a
// fishing monster for p's level (none below level 6, as in Mobius), either
// that monster ("You caught something smelly and scary...", spawned at p
// with p as its target) or the fish ("You caught something!", into the
// inventory and the championship). Then the fishing ends.
func (s *Server) fishDie(p *Player, st *fishState, win bool) {
	if st.tick != 0 {
		s.loop.Cancel(st.tick)
		st.tick = 0
	}
	if s.fishing.states[p] != st {
		return
	}
	if win && s.fishing.data != nil {
		if m, ok := s.fishing.data.Monster(p.Level()); ok {
			if fishing.SmellyCatch(s.fishing.rnd, m) {
				sendFishingMessage(p, fishing.MsgSmellyCatch, 0)
				s.spawnFishingMonster(p, m.NpcID)
			} else {
				sendFishingMessage(p, fishing.MsgCaughtSomething, 0)
				if _, err := s.AddItem(p, int32(st.fight.ItemID()), 1, -1, true, false); err != nil { //nolint:gosec // item IDs are small
					s.log.Warn("game: cannot give a caught fish", append(p.logArgs(), "item", st.fight.ItemID(), "err", err)...)
				}
				s.championshipNewFish(p, st.fight.LureID()) // fishing_champ.go
			}
		}
	}
	s.endFishing(p, win)
}

// spawnFishingMonster is Quest.addSpawn(npcId, player) with the player as
// the monster's target.
func (s *Server) spawnFishingMonster(p *Player, npcID int) {
	n, err := s.SpawnNpc(npcID, p.Position())
	if err != nil {
		s.log.Warn("game: cannot spawn a fishing monster", append(p.logArgs(), "npc", npcID, "err", err)...)
		return
	}
	if n.skill != nil {
		n.skill.SetTarget(s.skillCreatureOf(p))
	}
}

// endFishing is Mobius Player.endFishing: without a fight "Baits have been
// lost because the fish got away."; ExFishingEnd to everyone around, "You
// reel your line in and stop fishing.", p may move again and the bite
// checks stop.
func (s *Server) endFishing(p *Player, win bool) {
	st := s.fishing.states[p]
	if st == nil {
		return
	}
	delete(s.fishing.states, p)
	if st.fight == nil {
		sendFishingMessage(p, fishing.MsgBaitsLost, 0)
	}
	if st.tick != 0 {
		s.loop.Cancel(st.tick)
		st.tick = 0
	}
	s.world.Broadcast(p, serverpackets.NewExFishingEndPacket(serverpackets.ExFishingEnd{ObjectID: p.ObjectID(), Win: win}), true)
	sendFishingMessage(p, fishing.MsgStopFishing, 0)
	if p.skill != nil {
		p.skill.SetImmobilized(false)
	}
	s.stopLookingForFish(st)
}

// fishingLeave drops the fishing of a player who leaves; nobody is left to
// tell (Mobius cancels the bite checks in deleteMe).
func (s *Server) fishingLeave(p *Player) {
	st := s.fishing.states[p]
	if st == nil {
		return
	}
	delete(s.fishing.states, p)
	s.stopLookingForFish(st)
	if st.tick != 0 {
		s.loop.Cancel(st.tick)
	}
	if p.skill != nil {
		p.skill.SetImmobilized(false)
	}
}

// ----- the fight's skills (Mobius Pumping and Reeling) -----

// fishingPull is Mobius Pumping.onStart and Reeling.onStart: without a fish
// to fight a message and ActionFailed; otherwise the damage of
// fishing.Damage with the rod, Fishing Expertise's stat, the skill's power
// and a charged fishing shot (used up), with the mastery penalty message,
// goes to the fight.
func (s *Server) fishingPull(p *Player, sk *skills.Skill, power float64, reel bool) {
	st := s.fishing.states[p]
	if st == nil || st.fight == nil || st.fight.IsOver() {
		if reel {
			sendFishingMessage(p, fishing.MsgReelingOnlyWhileFishing, 0)
		} else {
			sendFishingMessage(p, fishing.MsgPumpingOnlyWhileFishing, 0)
		}
		p.SendActionFailed()
		return
	}
	w := activeWeaponInstance(p)
	if w == nil || w.Template().Weapon == nil || s.fishing.data == nil {
		return
	}
	rod, ok := s.fishing.data.Rod(int(w.ItemID()))
	if !ok {
		return
	}
	shot := playerChargedShot(p, skills.FishSoulshot)
	expertise := 1.0
	if p.skill != nil {
		expertise = p.skill.Calc(stats.FishingExpertise, 1, nil, nil)
	}
	dmg, pen, penalized := fishing.Damage(rod, expertise, power, shot, p.SkillLevel(fishing.SkillFishingExpertise), sk.Level)
	if penalized {
		sendFishingMessage(p, fishing.MsgMasteryPenalty, 0)
	}
	if shot {
		setPlayerChargedShot(p, skills.FishSoulshot, false)
	}
	out := fishOutput{s, p, st}
	if reel {
		st.fight.Reel(s.fishing.rnd, dmg, pen, out)
	} else {
		st.fight.Pump(s.fishing.rnd, dmg, pen, out)
	}
}

// ----- what a fishing player may not do -----

// fishingSkillRefused is the fishing part of Player.checkUseMagicConditions:
// while fishing only fishing skills ("Only fishing skills may be used at
// this time."); Pumping and Reeling only with a fish to fight (their
// message and ActionFailed), refused here so no reuse starts.
func (s *Server) fishingSkillRefused(p *Player, sk *skills.Skill) bool {
	st := s.fishing.states[p]
	if st != nil && !sk.HasEffectType(skills.TypeFishing, skills.TypeFishingStart) {
		sendFishingMessage(p, fishing.MsgOnlyFishingSkills, 0)
		return true
	}
	if (st == nil || st.fight == nil) && sk.HasEffectType(skills.TypeFishing) {
		if sk.ID == fishing.SkillPumping {
			sendFishingMessage(p, fishing.MsgPumpingOnlyWhileFishing, 0)
		} else {
			sendFishingMessage(p, fishing.MsgReelingOnlyWhileFishing, 0)
		}
		p.SendActionFailed()
		return true
	}
	return false
}

// fishingItemRefused is UseItem's check: a fishing player may use only the
// fishing shots ("You cannot do that while fishing.").
func (s *Server) fishingItemRefused(p *Player, itemID int32) bool {
	if s.Fishing(p) && (itemID < fishing.FirstFishingShot || itemID > fishing.LastFishingShot) {
		sendFishingMessage(p, fishing.MsgCannotDoThatWhileFishing3, 0)
		return true
	}
	return false
}

// fishingRefusesDrop is RequestDropItem's check: a fishing player is told
// "You cannot do that while fishing." (1470).
func (s *Server) fishingRefusesDrop(p *Player) bool {
	return s.fishingRefuses(p, fishing.MsgCannotDoThatWhileFishing2)
}

// fishingRefusesSocial is RequestSocialAction's check (1471).
func (s *Server) fishingRefusesSocial(p *Player) bool {
	return s.fishingRefuses(p, fishing.MsgCannotDoThatWhileFishing3)
}

// fishingRefuses tells a fishing player m and reports true.
func (s *Server) fishingRefuses(p *Player, m fishing.Message) bool {
	if s.Fishing(p) {
		sendFishingMessage(p, m, 0)
		return true
	}
	return false
}

// immobilized reports a player who may not walk (Mobius isImmobilized, set
// while fishing).
func (p *Player) immobilized() bool { return p.skill != nil && p.skill.immobilized }
