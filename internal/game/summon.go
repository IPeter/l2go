package game

import (
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/summon"
	"github.com/IPeter/l2go/internal/world"
)

// The summon feature (Plan 14 M11f): servitors, pets and baby pets, one
// summon type on the NPC creature base (decision: a Summon is an Npc whose
// summon field is set), with the rules of internal/summon.
//
//   - summon.go: the Summon, summoning into the world, unsummoning, death and
//     decay, following its owner through teleports, the owner leaving.
//   - summon_info.go: PetInfo to the owner, SummonInfo to the others, the
//     status packets.
//   - summon_ai.go: Mobius SummonAI on the NPC movement and attack code.
//   - summon_actions.go: the summon actions of RequestActionUse.
//   - summon_world.go: skills.SummonWorld, the summoning effects' side.
//   - servitor.go: servitors (the Summon effect, life and upkeep, restore).
//   - pet*.go: pets (collars, feeding, levels, the pet inventory, restore).
//   - cubic*.go: cubics.
//   - summon_store.go: the background writes of storage.SummonStore.
//
// The API other features use (loop-only): SummonOf, Unsummon (a subclass
// switch sends the summon away), actingPlayer (a summon acts for its owner).
func init() { registerFeature("summon", setupSummons) }

// Summon is a player's servitor or pet: an NPC of a Servitor, Pet or BabyPet
// template that follows and fights for its owner (Mobius Summon). It lives
// on the loop.
type Summon struct {
	npc   *Npc
	owner *Player
	kind  summon.Kind
	// name is the name the summon shows: a pet's given name, else "".
	name string

	// follow and prevFollow are Mobius _follow and _previousFollowStatus.
	follow, prevFollow bool
	// restore is Mobius _restoreSummon: set while the owner logs out, so the
	// summon comes back with it.
	restore bool
	// showAnim is Mobius isShowSummonAnimation.
	showAnim bool
	// gone is set once the summon left its owner for good.
	gone bool

	ai summonAI

	// The servitor's part (servitor.go).
	refSkill  int
	expMult   float32
	life      summon.Life
	lifeTimer world.TimerID

	// The pet's part (pet.go), nil for a servitor.
	pet *petState

	decayTimer world.TimerID
	// abnormalTimer coalesces SummonInfo refreshes (updateAbnormalEffect's
	// 50 ms task).
	abnormalTimer world.TimerID
}

// summonFeature is the summon feature's state on the Server (Server.summons).
// Loop-only, except data and store, set by New.
type summonFeature struct {
	data  *data.Summons
	store storage.SummonStore
	// writer writes the store in the background.
	writer *bossWriter

	// byOwner holds each player's summon; list the same summons in the order
	// they came out, for the ticks.
	byOwner map[*Player]*Summon
	list    []*Summon
	// owners holds what a player keeps of its summons while it plays: the
	// collar in use, the pet inventory, the cubics, the saved state.
	owners map[*Player]*summonOwner
	// ticks counts the summon ticks (summonTickPeriod).
	ticks int
}

// summonOwner is what the summon feature keeps per player.
type summonOwner struct {
	// saved is the player's stored summon state, loaded at character select.
	saved storage.OwnerSummons
	// restorePet is the stored pet to bring back with the player, if any.
	restorePet *storage.Pet
	// collar is the collar a SummonItems use attached for SummonPet (Mobius
	// PetItemHolder).
	collar int32
	// teleporting is set between a teleport's start and its end.
	teleporting bool
	// pets caches the stored pets of the player's collars by collar object
	// ID (loaded at character select or on first use).
	pets map[int32]*storage.Pet
	// cubics are the player's cubics (cubic.go).
	cubics    []*cubic
	maxCubics int
	// petLoading is set while a pet's stored state loads.
	petLoading bool
}

// summonTickPeriod is the period of the summon tick: the avoid task (Mobius
// SummonAI's 100 ms task); every 5th tick the attack follow (500 ms), every
// 10th the normal follow (1,000 ms).
const summonTickPeriod = 100 * time.Millisecond

// System messages of summons (Mobius SystemMessageId).
const (
	sumMsgTargetCannotBeFound     serverpackets.SystemMessageID = 50   // "Your target cannot be found."
	sumMsgInvalidTarget           serverpackets.SystemMessageID = 109  // "Invalid target."
	sumMsgYouAlreadyHaveAPet      serverpackets.SystemMessageID = 543  // "You already have a pet."
	sumMsgPetCannotGoBackInBattle serverpackets.SystemMessageID = 579  // "A pet cannot be sent back during battle."
	sumMsgMultiplePets            serverpackets.SystemMessageID = 580  // "You may not use multiple pets or servitors at the same time."
	sumMsgDeadPetCannotBeSentBack serverpackets.SystemMessageID = 589  // "A dead pet cannot be sent back."
	sumMsgHungryPetNoRestore      serverpackets.SystemMessageID = 594  // "You may not restore a hungry pet."
	sumMsgYourPetHitForDamage     serverpackets.SystemMessageID = 1015 // "Your pet hit for $s1 damage."
	sumMsgPetReceivedDamage       serverpackets.SystemMessageID = 1016 // "Your pet received $s2 damage caused by $s1."
	sumMsgPetCritical             serverpackets.SystemMessageID = 1017 // "Pet's critical hit!"
	sumMsgSummonedMonsterCritical serverpackets.SystemMessageID = 1028 // "Summoned monster's critical hit!"
	sumMsgServitorPassedAway      serverpackets.SystemMessageID = 1520 // "Servitor passed away."
	sumMsgServitorVanished        serverpackets.SystemMessageID = 1521 // "Your servitor has vanished! You'll need to summon a new one."
	sumMsgServitorInBattle        serverpackets.SystemMessageID = 1724 // "A servitor whom is engaged in battle cannot be de-activated."
	sumMsgPetUnresponsive         serverpackets.SystemMessageID = 1864 // "Your pet/servitor is unresponsive and will not obey any orders."
	sumMsgPetInDistress           serverpackets.SystemMessageID = 1865 // "Your pet/servitor is currently in a state of distress."
	sumMsgPetTooHighLevel         serverpackets.SystemMessageID = 1918 // "Your pet is too high level to control."
)

// setupSummons registers the summon feature's handlers and hooks.
func setupSummons(s *Server) error {
	f := &s.summons
	f.byOwner = make(map[*Player]*Summon)
	f.owners = make(map[*Player]*summonOwner)
	f.writer = newBossWriter(s.log)
	s.loop.Every(summonTickPeriod, s.summonTick)
	s.OnPlayerEnter(s.summonEnter)
	s.OnPlayerLeave(s.summonLeave)
	s.OnTeleport(func(p *Player, _ world.Position) {
		if o := f.owners[p]; o != nil {
			o.teleporting = true
		}
	})
	s.OnPlayerMoved(s.summonOwnerMoved)
	s.OnDie(s.summonDied)
	s.OnForget(s.summonForget)
	s.OnNpcAttackEvent(s.summonAttackEvent)
	s.OnAttacked(s.summonAttacked)
	s.OnSkillFinished(s.summonSkillFinished)
	if err := s.registerSummonActions(); err != nil { // summon_actions.go
		return err
	}
	if err := s.setupPets(); err != nil { // pet.go
		return err
	}
	return s.setupCubics() // cubic.go
}

// SummonOf returns p's summon, nil without one. Loop-only.
func (s *Server) SummonOf(p *Player) *Summon {
	if p == nil {
		return nil
	}
	return s.summons.byOwner[p]
}

// Npc returns the summon's NPC.
func (sm *Summon) Npc() *Npc { return sm.npc }

// Owner returns the summon's owner.
func (sm *Summon) Owner() *Player { return sm.owner }

// Kind returns whether the summon is a servitor, a pet or a baby pet.
func (sm *Summon) Kind() summon.Kind { return sm.kind }

// level returns the summon's level: a pet's own, else the template's.
func (sm *Summon) level() int {
	if sm.pet != nil {
		return sm.pet.level
	}
	return sm.npc.Template.Level
}

// summonOwnerOf returns the player state of p, made on first use.
func (s *Server) summonOwnerOf(p *Player) *summonOwner {
	f := &s.summons
	o := f.owners[p]
	if o == nil {
		o = &summonOwner{maxCubics: 1}
		o.saved.OwnerID = p.ObjectID()
		f.owners[p] = o
	}
	return o
}

// actingPlayer is Mobius asPlayer: a player itself, a summon's owner, nil for
// any other creature.
func (s *Server) actingPlayer(c Creature) *Player {
	switch v := c.(type) {
	case *Player:
		return v
	case *Npc:
		if v != nil && v.summon != nil {
			return v.summon.owner
		}
	}
	return nil
}

// summonOfNpc returns the summon an NPC is, nil for a plain NPC.
func summonOfNpc(c Creature) *Summon {
	if n, ok := c.(*Npc); ok && n != nil {
		return n.summon
	}
	return nil
}

// newSummon makes the summon NPC of template t for owner, out of the world
// (Mobius new Servitor / new Pet).
func (s *Server) newSummon(owner *Player, t *data.NpcTemplate, kind summon.Kind) (*Summon, error) {
	n, err := s.newNpc(t)
	if err != nil {
		return nil, err
	}
	sm := &Summon{npc: n, owner: owner, kind: kind, follow: true, showAnim: true}
	sm.ai.startFollow = true
	n.summon = sm
	n.Title = owner.Name()
	return sm, nil
}

// summonSpawnPoint is where a servitor appears (Mobius Summon's
// constructor): a valid point within 100 of its owner.
func (s *Server) summonSpawnPoint(owner *Player) world.Position {
	pos := owner.Position()
	to := pos
	to.X += int32(s.rnd.Between(-summon.SpawnOffset, summon.SpawnOffset)) //nolint:gosec // small
	to.Y += int32(s.rnd.Between(-summon.SpawnOffset, summon.SpawnOffset)) //nolint:gosec // small
	p := s.geo.ValidLocation(geoPoint(pos), geoPoint(to))
	return world.Position{X: p.X, Y: p.Y, Z: p.Z, Heading: pos.Heading}
}

// spawnSummon puts sm into the world at pos as its owner's summon (Mobius
// setPet, spawnMe and Summon.onSpawn): the players around see it appear with
// its summon animation, it follows its owner, and the owner gets PetInfo and
// PetStatusUpdate.
func (s *Server) spawnSummon(sm *Summon, pos world.Position) error {
	f := &s.summons
	n, owner := sm.npc, sm.owner
	if err := n.SetPosition(pos); err != nil {
		return err
	}
	n.Summoned = sm.showAnim
	f.byOwner[owner] = sm
	f.list = append(f.list, sm)
	if err := s.AddObject(n); err != nil {
		delete(f.byOwner, owner)
		f.list = f.list[:len(f.list)-1]
		return err
	}
	n.StartRegen()
	s.summonSetFollowStatus(sm, true)
	s.summonBroadcastStatus(sm, 0)
	sm.showAnim, n.Summoned = false, false
	s.rechargeSummonShots(sm) // pet_shots.go
	return nil
}

// Unsummon sends p's summon away (Mobius Summon.unSummon): a living summon
// in the world stops, its owner gets PetDelete, its state is stored, its
// effects end and it leaves the world. A dead one stays. A subclass switch
// and the Unsummon effect use it. Loop-only.
func (s *Server) Unsummon(p *Player) { s.unsummon(p) }

func (s *Server) unsummon(p *Player) {
	sm := s.SummonOf(p)
	if sm == nil || sm.npc.Dead || !sm.npc.InWorld() {
		return
	}
	s.unsummonNow(sm)
}

// unsummonNow is Mobius unSummon for a living summon in the world.
func (s *Server) unsummonNow(sm *Summon) {
	n, owner := sm.npc, sm.owner
	n.StopRegen()
	s.summonStopFollow(sm)
	owner.Send(petDeletePacket(sm))
	if sm.pet != nil {
		s.petUnsummonNotice(sm) // pet_items.go: the items left in the pet inventory
	}
	s.AbortAttack(n)
	s.AbortCast(n)
	s.summonStoreMe(sm)
	s.summonStoreEffects(sm, true)
	s.summonStopTasks(sm)
	if n.skill != nil {
		n.skill.el.StopAll()
	}
	n.ai.target = nil
	s.summonGone(sm)
}

// summonStopTasks stops the summon's own timers: the life task, the feed
// task, the decay and the coalesced info.
func (s *Server) summonStopTasks(sm *Summon) {
	for _, id := range []*world.TimerID{&sm.lifeTimer, &sm.decayTimer, &sm.abnormalTimer} {
		if *id != 0 {
			s.loop.Cancel(*id)
			*id = 0
		}
	}
	if sm.pet != nil {
		s.petStopFeed(sm)  // pet_feed.go
		s.babyStopTask(sm) // pet_baby.go
	}
	sm.ai.intention = summonIdle
	sm.ai.attackTarget, sm.ai.castTarget, sm.ai.lastAttack = nil, nil, nil
}

// summonGone takes the summon out of the world and away from its owner for
// good (Mobius decayMe and owner.setPet(null)).
func (s *Server) summonGone(sm *Summon) {
	if sm.gone {
		return
	}
	sm.gone = true
	f := &s.summons
	n := sm.npc
	if f.byOwner[sm.owner] == sm {
		delete(f.byOwner, sm.owner)
	}
	for i, x := range f.list {
		if x == sm {
			f.list = append(f.list[:i], f.list[i+1:]...)
			break
		}
	}
	s.npcWatchersGone(n)
	wasIn := n.InWorld()
	if wasIn {
		n.Move = nil
		s.RemoveObject(n)
	}
	s.npcSkillsGone(n, true)
	if wasIn {
		s.releaseNpcID(n)
	}
}

// npcWatchersGone drops the players' targets on n before it leaves (the
// players that target it lose it with the visibility anyway).
func (s *Server) npcWatchersGone(n *Npc) {
	for _, p := range append([]*Player(nil), s.npc.watchers[n.ID()]...) {
		if p.Target == world.Object(n) {
			s.SetTarget(p, nil)
		}
	}
}

// summonDied is the OnDie hook of summons (Mobius Summon.doDie, Servitor
// and Pet doDie): the summon's effects are stored, it stops and shows Die,
// the monsters that hated it hate its owner as much, the killer hates the
// owner by 500 more, and the corpse decays (deleteMe). A servitor's life
// task stops and it is not restored; a pet loses exp and stops eating.
func (s *Server) summonDied(victim, killer Creature) {
	sm := summonOfNpc(victim)
	if sm == nil {
		if p, ok := victim.(*Player); ok {
			s.cubicsOwnerDied(p) // cubic.go: death clears the cubics
		}
		return
	}
	n, owner := sm.npc, sm.owner
	if sm.pet != nil {
		s.petDeathPenalty(sm) // pet.go
	}
	s.summonStoreEffects(sm, false)
	s.stopNpc(n, nil)
	n.BroadcastStatus()
	if n.InWorld() {
		s.world.Broadcast(n, npcDiePacket(n), false)
	}
	sm.ai.intention = summonIdle
	sm.ai.attackTarget, sm.ai.castTarget = nil, nil
	s.summonStopFollow(sm)
	if owner != nil {
		for o := range s.world.Visible(n) {
			m, ok := o.(*Npc)
			if !ok || m.Dead || m.class == nil || !m.class.attackable {
				continue
			}
			if e := m.aggroOf(n); e != nil {
				m.AddDamageHate(owner, e.Damage, e.Hate)
			}
		}
		if m, ok := killer.(*Npc); ok && m.class != nil && m.class.attackable && !m.Dead {
			m.AddDamageHate(owner, 0, 500)
		}
	}
	if sm.lifeTimer != 0 {
		s.loop.Cancel(sm.lifeTimer)
		sm.lifeTimer = 0
	}
	switch {
	case sm.pet != nil:
		s.petDied(sm) // pet.go
	default:
		s.servitorDied(sm) // servitor.go
	}
	s.summonStartDecay(sm)
	s.summonBroadcastStatus(sm, 1)
}

// summonStartDecay schedules the decay of a dead summon's corpse after its
// template's corpse time (Mobius DecayTaskManager.add).
func (s *Server) summonStartDecay(sm *Summon) {
	if sm.decayTimer != 0 {
		s.loop.Cancel(sm.decayTimer)
	}
	sm.decayTimer = s.loop.After(s.corpseTime(sm.npc), func() {
		sm.decayTimer = 0
		if sm.npc.Dead && !sm.gone {
			s.summonDeleteMe(sm)
		}
	})
}

// summonDeleteMe is Mobius Summon.deleteMe (onDecay): the owner gets
// PetDelete and the summon is gone; a pet gives its items back and its
// collar is destroyed (Pet.deleteMe).
func (s *Server) summonDeleteMe(sm *Summon) {
	owner := sm.owner
	if sm.pet != nil {
		s.petDeleteMe(sm) // pet.go: the items go back, the collar goes
	}
	if owner != nil && owner.InGame() {
		owner.Send(petDeletePacket(sm))
	}
	s.summonStopTasks(sm)
	s.summonGone(sm)
}

// summonRevived brings a dead summon back (a resurrection skill): its decay
// stops, and a pet gets percent of its lost exp back and eats again (Mobius
// Pet.doRevive).
func (s *Server) summonRevived(sm *Summon, percent float64) {
	n := sm.npc
	if !n.Dead {
		return
	}
	if sm.decayTimer != 0 {
		s.loop.Cancel(sm.decayTimer)
		sm.decayTimer = 0
	}
	n.Revive()
	n.SetCurrentHPMP(float64(n.MaxHP())*percent/100, n.CurMP)
	if n.CurHP < 1 {
		n.SetCurrentHP(1)
	}
	if sm.pet != nil {
		s.petRevived(sm, percent)
	}
	s.world.Broadcast(n, serverpackets.NewRevivePacket(int32(n.ID())), false)
	s.summonSetActive(sm)
	s.summonBroadcastStatus(sm, 1)
}

// summonEnter is the OnPlayerEnter hook: the summons stored to come back
// with the player do (Mobius EnterWorld: CharSummonTable.restorePet and
// restoreServitor).
func (s *Server) summonEnter(p *Player) {
	s.restoreServitor(p) // servitor.go
	s.restorePet(p)      // pet.go
}

// summonLeave is the OnPlayerLeave hook (Mobius Player.deleteMe): a summon
// is kept to come back at the next login and unsummoned; a dead one goes;
// the cubics stop.
func (s *Server) summonLeave(p *Player) {
	f := &s.summons
	if sm := f.byOwner[p]; sm != nil {
		sm.restore = true
		if !sm.npc.Dead && sm.npc.InWorld() {
			s.unsummonNow(sm)
		} else {
			s.summonStoreMe(sm)
			s.summonStopTasks(sm)
			s.summonGone(sm)
		}
	}
	s.cubicsStop(p, false) // cubic.go
	delete(f.owners, p)
}

// summonOwnerMoved is the OnPlayerMoved hook: after a teleport the summon
// comes to its owner and follows again (Mobius Player.onTeleported).
func (s *Server) summonOwnerMoved(p *Player, _, to world.Position) {
	o := s.summons.owners[p]
	if o == nil || !o.teleporting {
		return
	}
	o.teleporting = false
	sm := s.SummonOf(p)
	if sm == nil || !sm.npc.InWorld() {
		return
	}
	s.summonSetFollowStatus(sm, false)
	s.teleportSummon(sm, to)
	sm.ai.startFollow = true
	s.summonSetFollowStatus(sm, true)
	s.summonBroadcastStatus(sm, 0)
}

// teleportSummon puts sm at pos (Mobius teleToLocation without a random
// offset): it stops, leaves the world and comes back there; the owner gets
// TeleportToLocation of it (Summon.onTeleported).
func (s *Server) teleportSummon(sm *Summon, pos world.Position) {
	n := sm.npc
	s.stopNpc(n, nil)
	s.AbortAttack(n)
	n.ai.target = nil
	s.RemoveObject(n)
	if err := n.SetPosition(pos); err != nil {
		s.log.Warn("game: cannot teleport a summon", "npc", n.Template.ID, "pos", pos, "err", err)
	}
	if err := s.AddObject(n); err != nil {
		s.log.Warn("game: cannot teleport a summon", "npc", n.Template.ID, "pos", pos, "err", err)
		return
	}
	sm.owner.Send(serverpackets.NewTeleportToLocationPacket(serverpackets.TeleportToLocation{
		ObjectID: int32(n.ID()), X: pos.X, Y: pos.Y, Z: pos.Z, Heading: pos.Heading,
	}))
}

// servitorExpPenalty is the servitor part of Attackable.calculateRewards: a
// player with a servitor gets its exp times the servitor's expMultiplier.
func (s *Server) servitorExpPenalty(p *Player, r stats.Reward) stats.Reward {
	sm := s.SummonOf(p)
	if sm == nil || sm.kind != summon.Servitor {
		return r
	}
	mult := float64(sm.expMult)
	r.Exp = javaRound(float64(r.Exp) * mult)
	r.ShownExp = javaRound(float64(r.ShownExp) * mult)
	return r
}

// summonAutoAttackable is Mobius Summon.isAutoAttackable: as its owner.
func (s *Server) summonAutoAttackable(sm *Summon, attacker *Player) bool {
	return sm.owner != nil && attacker != nil && attacker != sm.owner && sm.owner.autoAttackableBy(attacker)
}

// summonDamageMessage is Mobius Summon.sendDamageMessage: the owner hears of
// its summon's hits on others.
func (s *Server) summonDamageMessage(sm *Summon, target Creature, damage int, crit, miss bool) {
	owner := sm.owner
	if miss || owner == nil || target == nil || target == Creature(owner) {
		return
	}
	if crit {
		if sm.kind == summon.Servitor {
			owner.SendSystemMessage(sumMsgSummonedMonsterCritical)
		} else {
			owner.SendSystemMessage(sumMsgPetCritical)
		}
	}
	owner.SendSystemMessage(sumMsgYourPetHitForDamage, serverpackets.NumberParam(int32(damage))) //nolint:gosec // hit damage fits
}

// summonAttacked is the OnAttacked hook: a summon's owner hears of the
// damage it took (Summon.reduceCurrentHp), and its AI may step aside
// (SummonAI.notifyActionAttacked); a summon's hit is told to its owner.
func (s *Server) summonAttacked(target, attacker Creature, damage int) {
	if sm := summonOfNpc(attacker); sm != nil {
		s.summonDamageMessage(sm, target, damage, false, false)
	}
	sm := summonOfNpc(target)
	if sm == nil {
		return
	}
	if sm.owner != nil && attacker != nil {
		sm.owner.SendSystemMessage(sumMsgPetReceivedDamage, serverpackets.TextParam(attacker.Name()), serverpackets.NumberParam(int32(damage))) //nolint:gosec // damage fits
	}
	s.summonAvoidAttack(sm, attacker) // summon_ai.go
}
