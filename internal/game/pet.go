package game

import (
	"context"
	"errors"
	"math"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/summon"
	"github.com/IPeter/l2go/internal/world"
)

// Pets (Mobius Pet, PetStat, SummonPet, SummonItems): a collar item casts
// its summoning skill (2046 for the wolf), whose SummonPet effect brings out
// the pet of the collar. A pet has its own level, exp and meal, keyed by the
// collar's object ID (storage.Pet), carries the items of its owner's pet
// inventory (pet_items.go), eats (pet_feed.go) and comes back at login when
// its owner logged out with it (RestorePetOnReconnect).

// Pet rates (Rates.ini; Mobius defaults).
const (
	petXPRate      = 1.0 // PetXpRate
	sinEaterXPRate = 1.0 // SinEaterXpRate
	petFoodRate    = 1   // PetFoodRate
)

// System messages of pets (Mobius SystemMessageId).
const (
	sumMsgPetGainedExp         serverpackets.SystemMessageID = 1014 // "Your pet gained $s1 experience points."
	sumMsgPetKilledResurrect   serverpackets.SystemMessageID = 1519 // "Your pet has been killed! ..."
	sumMsgS1HasDisappeared     serverpackets.SystemMessageID = 302  // "$s1 has disappeared."
	sumMsgCannotSummonInCombat serverpackets.SystemMessageID = 578  // "You cannot summon during combat."
	sumMsgCannotSummonInTrade  serverpackets.SystemMessageID = 577  // "You cannot summon during a trade or while using the private shops."
)

// petState is a pet's own state (Mobius Pet and PetStat).
type petState struct {
	data    *data.PetData
	control int32 // the collar's object ID
	level   int
	exp     int64
	sp      int64
	fed     int
	// expBeforeDeath is the exp before the last death penalty.
	expBeforeDeath int64
	// respawned is set for a pet with a stored row.
	respawned bool
	// effects are the effects kept over an unsummon.
	effects []storage.SavedSkill
	// weightPenalty is the pet's weight penalty level (refreshOverloaded).
	weightPenalty int
	feedTimer     world.TimerID
	// baby is a baby pet's support state (pet_baby.go), nil for others.
	baby *babyState
}

// levelData returns the pet data of the pet's level.
func (ps *petState) levelData() *data.PetLevel {
	if ps.data == nil {
		return nil
	}
	l, _ := ps.data.Level(ps.level)
	return l
}

// maxMeal is the level's maximum meal (PetStat.getMaxFeed).
func (ps *petState) maxMeal() int {
	if l := ps.levelData(); l != nil {
		return l.MaxMeal
	}
	return 0
}

// setupPets registers the pet packets.
func (s *Server) setupPets() error {
	return errors.Join(
		s.Register(clientpackets.OpRequestPetUseItem, s.handleRequestPetUseItem),
		s.Register(clientpackets.OpRequestGiveItemToPet, s.handleRequestGiveItemToPet),
		s.Register(clientpackets.OpRequestGetItemFromPet, s.handleRequestGetItemFromPet),
		s.Register(clientpackets.OpRequestPetGetItem, s.handleRequestPetGetItem),
		s.Register(clientpackets.OpRequestChangePetName, s.handleRequestChangePetName),
	)
}

// petCurve is the pet's exp table (PetStat.getExpForLevel).
func (s *Server) petCurve(sm *Summon) summon.ExpCurve {
	pd := sm.pet.data
	return summon.ExpCurve{
		MinLevel: pd.MinLevel(),
		MaxLevel: s.maxPetLevel(),
		ExpFor: func(level int) int64 {
			if l, ok := pd.Level(level); ok {
				return l.Exp
			}
			return 0
		},
	}
}

// petStatsTemplate is the stat template of a pet at a level (Mobius
// PetStat): the NPC template with the level's HP, MP, attack, defence and
// regeneration, and the pet's level.
func petStatsTemplate(t *data.NpcTemplate, l *data.PetLevel, level int) *stats.NpcTemplate {
	st := npcStatsTemplate(t)
	st.Level = level
	if l != nil {
		st.HP, st.MP = l.HP, l.MP
		st.PAtk, st.PDef, st.MAtk, st.MDef = l.PAtk, l.PDef, l.MAtk, l.MDef
		st.HPRegen, st.MPRegen = l.HPRegen, l.MPRegen
	}
	return st
}

// petSetLevel gives the pet a level (PetStat.setLevel): its stats follow,
// the feed task starts over and the collar shows the level.
func (s *Server) petSetLevel(sm *Summon, level int) {
	ps := sm.pet
	ps.level = level
	sm.npc.Stats.SetTemplate(petStatsTemplate(sm.npc.Template, ps.levelData(), level))
	sm.npc.cacheOK = false
	if sm.npc.InWorld() {
		s.petStartFeed(sm)
	}
	s.petCollarLevel(sm)
}

// petCollarLevel sets the collar's enchant level to the pet's level (Mobius
// item.setEnchantLevel(pet.getLevel())).
func (s *Server) petCollarLevel(sm *Summon) {
	it := sm.owner.inv.ByObjectID(sm.pet.control)
	if it == nil || int(it.Enchant()) == sm.pet.level {
		return
	}
	it.SetEnchant(int32(sm.pet.level)) //nolint:gosec // small
	s.ItemsChanged(sm.owner)
}

// summonPetFromCollar is Mobius SummonPet.onStart: the pet of the collar
// the player used comes out, its stored row loaded first when it is not in
// the cache.
func (s *Server) summonPetFromCollar(p *Player) {
	o := s.summonOwnerOf(p)
	control := o.collar
	o.collar = 0
	if s.SummonOf(p) != nil || s.mounted(p) {
		p.SendSystemMessage(sumMsgYouAlreadyHaveAPet)
		return
	}
	it := p.inv.ByObjectID(control)
	if it == nil || s.summons.data == nil {
		return
	}
	pd, ok := s.summons.data.PetByItem(int(it.ItemID()))
	if !ok || s.worldData == nil {
		return
	}
	t, ok := s.worldData.Npc(pd.NpcID)
	if !ok {
		return
	}
	if stored, ok := o.pets[control]; ok || s.summons.store == nil {
		s.spawnPet(p, control, pd, t, stored)
		return
	}
	if o.petLoading {
		return
	}
	o.petLoading = true
	store := s.summons.store
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), saveTimeout)
		defer cancel()
		pet, err := store.LoadPet(ctx, control)
		s.loop.Post(func() {
			o.petLoading = false
			var stored *storage.Pet
			switch {
			case err == nil:
				stored = &pet
			case !errors.Is(err, storage.ErrNotFound):
				s.log.Warn("game: cannot load a pet", append(p.logArgs(), "collar", control, "err", err)...)
				return
			}
			if p.InGame() && s.SummonOf(p) == nil && p.inv.ByObjectID(control) != nil {
				if o.pets == nil {
					o.pets = make(map[int32]*storage.Pet)
				}
				o.pets[control] = stored
				s.spawnPet(p, control, pd, t, stored)
			}
		})
	}()
}

// spawnPet is Mobius Pet.spawnPet, Pet.restore and the rest of
// SummonPet.onStart: a stored pet gets its row back (its exp raised to its
// level's), a new one starts at its template's level (the owner's for the
// Sin Eater) with full HP, MP and meal and is stored; a level-synced pet
// takes its owner's level; it appears 50 and 100 from its owner with its
// summon animation, starts eating, follows, and the owner gets its items.
func (s *Server) spawnPet(p *Player, control int32, pd *data.PetData, t *data.NpcTemplate, stored *storage.Pet) *Summon {
	kind := summon.Pet
	if t.Type == "BabyPet" {
		kind = summon.BabyPet
	}
	sm, err := s.newSummon(p, t, kind)
	if err != nil {
		s.log.Warn("game: cannot make a pet", append(p.logArgs(), "npc", t.ID, "err", err)...)
		return nil
	}
	ps := &petState{data: pd, control: control}
	sm.pet = ps
	level := t.Level
	if t.DisplayID == summon.SinEaterID {
		level = p.Level()
	}
	if stored != nil {
		level = int(stored.Level)
	}
	level = max(level, pd.MinLevel())
	s.petSetLevel(sm, level)
	n := sm.npc
	if stored != nil {
		ps.respawned = true
		sm.name = stored.Name
		ps.exp = stored.Exp
		if l := ps.levelData(); l != nil && ps.exp < l.Exp {
			ps.exp = l.Exp
		}
		ps.sp = stored.SP
		n.CurHP = min(stored.CurHP, float64(n.MaxHP()))
		n.CurMP = min(stored.CurMP, float64(n.MaxMP()))
		ps.fed = summon.ClampFed(int(stored.Fed), ps.maxMeal())
		ps.effects = append([]storage.SavedSkill(nil), stored.Effects...)
	}
	if pd.SyncLevel && ps.level != p.Level() {
		lvl := min(pd.MaxLevel(), p.Level())
		s.petSetLevel(sm, lvl)
		ps.exp = s.petCurve(sm).ExpFor(lvl)
	}
	if !ps.respawned {
		n.CurHP, n.CurMP = float64(n.MaxHP()), float64(n.MaxMP())
		ps.exp = s.petCurve(sm).ExpFor(ps.level)
		ps.fed = ps.maxMeal()
	}
	dead := stored != nil && stored.CurHP < 1
	if !ps.respawned {
		s.petStoreMe(sm)
	}
	s.petCollarLevel(sm)
	s.petWornStats(sm) // pet_items.go
	pos := p.Position()
	pos.X += summon.PetSpawnDX
	pos.Y += summon.PetSpawnDY
	if err := s.spawnSummon(sm, pos); err != nil {
		s.log.Warn("game: cannot spawn a pet", append(p.logArgs(), "npc", t.ID, "err", err)...)
		s.releaseNpcID(n)
		return nil
	}
	s.SetNpcRunning(n, true)
	s.petRestoreEffects(sm)
	if dead {
		n.CurHP = 0
		n.Dead = true
		n.StopRegen()
		s.summonStartDecay(sm)
	} else {
		s.petStartFeed(sm)
		if kind == summon.BabyPet {
			s.babyPetSpawned(sm) // pet_baby.go
		}
	}
	s.petHungerSpeed(sm)
	p.Send(s.petItemListPacket(p))
	s.summonBroadcastStatus(sm, 1)
	return sm
}

// restorePet is CharSummonTable.restorePet at login: the pet the owner
// logged out with comes back from its collar.
func (s *Server) restorePet(p *Player) {
	o := s.summons.owners[p]
	if o == nil || o.restorePet == nil || s.SummonOf(p) != nil || !restorePetOnReconnect {
		return
	}
	stored := o.restorePet
	o.restorePet = nil
	it := p.inv.ByObjectID(stored.ControlID)
	if it == nil || s.summons.data == nil || s.worldData == nil {
		return
	}
	pd, ok := s.summons.data.PetByItem(int(it.ItemID()))
	if !ok {
		return
	}
	t, ok := s.worldData.Npc(pd.NpcID)
	if !ok {
		return
	}
	s.spawnPet(p, stored.ControlID, pd, t, stored)
}

// petStoreMe is Mobius Pet.storeMe: the pet's row (name, level, HP, MP,
// exp, SP, meal, whether it comes back at login), and the owner's restore
// entry; the collar shows its level.
func (s *Server) petStoreMe(sm *Summon) {
	ps := sm.pet
	if ps.control == 0 {
		return
	}
	restore := sm.restore && restorePetOnReconnect
	row := storage.Pet{
		ControlID: ps.control,
		OwnerID:   sm.owner.ObjectID(),
		Name:      sm.name,
		Level:     int32(ps.level), //nolint:gosec // small
		CurHP:     sm.npc.CurHP,
		CurMP:     sm.npc.CurMP,
		Exp:       ps.exp,
		SP:        ps.sp,
		Fed:       int32(ps.fed), //nolint:gosec // small
		Restore:   restore,
		Effects:   ps.effects,
	}
	ps.respawned = true
	o := s.summonOwnerOf(sm.owner)
	if o.pets == nil {
		o.pets = make(map[int32]*storage.Pet)
	}
	cp := row
	o.pets[ps.control] = &cp
	s.writePet(row)
	switch {
	case restore && o.saved.RestorePet != ps.control:
		o.saved.RestorePet = ps.control
		s.writeOwnerSummons(sm.owner)
	case !restore && o.saved.RestorePet == ps.control:
		o.saved.RestorePet = 0
		s.writeOwnerSummons(sm.owner)
	}
}

// petStoreEffects is Mobius Pet.storeEffect: the pet's effects are kept
// with its row.
func (s *Server) petStoreEffects(sm *Summon, store bool) {
	if !summonStoreSkillCooltime {
		return
	}
	sm.pet.effects = nil
	if store {
		sm.pet.effects = s.summonEffectsToSave(sm)
	}
}

// petRestoreEffects is Mobius Pet.restoreEffects: the kept effects come
// back and are used up.
func (s *Server) petRestoreEffects(sm *Summon) {
	if !summonStoreSkillCooltime || len(sm.pet.effects) == 0 {
		return
	}
	s.applySummonEffects(sm, sm.pet.effects)
	sm.pet.effects = nil
}

// petDeathPenalty is Mobius Pet.deathPenalty, outside PvP zones: the pet
// loses (−0.07 × level + 6.5) % of its level's exp.
func (s *Server) petDeathPenalty(sm *Summon) {
	owner := sm.owner
	if owner == nil || owner.Zones.Inside(ZonePvP) {
		return
	}
	ps := sm.pet
	c := s.petCurve(sm)
	lost := c.DeathPenalty(ps.level)
	ps.expBeforeDeath = ps.exp
	s.petAddExp(sm, -lost)
}

// petDied is the rest of Pet.doDie: it stops eating and its owner is told
// to resurrect it.
func (s *Server) petDied(sm *Summon) {
	s.petStopFeed(sm)
	s.babyStopTask(sm)
	s.AbortCast(sm.npc)
	sm.owner.SendSystemMessage(sumMsgPetKilledResurrect)
}

// petRevived is Pet.doRevive(power): percent of the lost exp comes back,
// it eats again and runs unless hungry.
func (s *Server) petRevived(sm *Summon, percent float64) {
	ps := sm.pet
	if back := summon.RestoredExp(ps.expBeforeDeath, ps.exp, percent); ps.expBeforeDeath > 0 {
		s.petAddExp(sm, back)
		ps.expBeforeDeath = 0
	}
	s.petStartFeed(sm)
	s.babyStartTask(sm)
	if !s.petHungry(sm) {
		s.SetNpcRunning(sm.npc, true)
	}
}

// petDeleteMe is Pet.deleteMe: the pet's items go back to its owner, the
// collar is destroyed ("$s1 has disappeared.") and the pet's row deleted.
func (s *Server) petDeleteMe(sm *Summon) {
	owner := sm.owner
	ps := sm.pet
	s.petItemsToOwner(owner) // pet_items.go
	if it := owner.inv.ByObjectID(ps.control); it != nil {
		id := it.ItemID()
		if s.DestroyItem(owner, it, 1, false) {
			owner.SendSystemMessage(sumMsgS1HasDisappeared, serverpackets.ItemNameParam(id))
		}
	}
	o := s.summonOwnerOf(owner)
	delete(o.pets, ps.control)
	if o.saved.RestorePet == ps.control {
		o.saved.RestorePet = 0
		s.writeOwnerSummons(owner)
	}
	s.deletePet(ps.control)
}

// petAddExp is PlayableStat.addExp with PetStat.addLevel: the pet's exp and
// level follow the curve; a new level shows the level-up animation, fills
// HP and MP and refreshes the pet's windows.
func (s *Server) petAddExp(sm *Summon, amount int64) {
	ps := sm.pet
	n := sm.npc
	exp, level := s.petCurve(sm).AddExp(ps.exp, ps.level, amount)
	ps.exp = exp
	if level == ps.level {
		return
	}
	up := level > ps.level
	s.petSetLevel(sm, level)
	if up {
		n.SetCurrentHPMP(float64(n.MaxHP()), float64(n.MaxMP()))
	}
	if n.InWorld() {
		s.world.Broadcast(n, serverpackets.NewStatusUpdatePacket(serverpackets.StatusUpdate{
			ObjectID: int32(n.ID()),
			Values: []serverpackets.StatusValue{
				{Attr: serverpackets.StatusLevel, Value: int32(level)}, //nolint:gosec // small
				{Attr: serverpackets.StatusMaxHP, Value: n.MaxHP()},
				{Attr: serverpackets.StatusMaxMP, Value: n.MaxMP()},
			},
		}), false)
		if up {
			s.world.Broadcast(n, serverpackets.NewSocialActionPacket(serverpackets.SocialAction{
				ObjectID: int32(n.ID()), ActionID: serverpackets.SocialActionLevelUp,
			}), false)
		}
	}
	s.summonBroadcastStatus(sm, 1)
}

// petAddExpAndSP is Pet.addExpAndSp and PetStat.addExpAndSp: the pet's
// rate, nothing for a starving pet, "Your pet gained $s1 experience
// points.".
func (s *Server) petAddExpAndSP(sm *Summon, exp, sp float64) {
	ps := sm.pet
	if summon.Uncontrollable(ps.fed) {
		return
	}
	rate := petXPRate
	if sm.npc.Template.ID == summon.SinEaterID {
		rate = sinEaterXPRate
	}
	gain := summon.PetExpGain(exp, rate)
	s.petAddExp(sm, gain)
	ps.sp += javaRound(sp)
	sm.owner.SendSystemMessage(sumMsgPetGainedExp, serverpackets.LongNumberParam(gain))
	s.summonBroadcastStatus(sm, 1)
}

// petExpShare is the pet part of PlayerStat.addExpAndSp: a pet within the
// party range takes its share of the owner's exp and SP (get_exp_type), when
// alive.
func (s *Server) petExpShare(p *Player, r stats.Reward) stats.Reward {
	sm := s.SummonOf(p)
	if sm == nil || sm.pet == nil || !sm.npc.InWorld() {
		return r
	}
	if p.Position().Distance3D(sm.npc.Position()) >= float64(s.combatSettings().PartyRange) {
		return r
	}
	l := sm.pet.levelData()
	if l == nil {
		return r
	}
	ownerExp, ownerSP, petExp, petSP := summon.OwnerShare(float64(r.Exp), float64(r.SP), l.OwnerExpTaken)
	if !sm.npc.Dead {
		s.petAddExpAndSP(sm, petExp, petSP)
	}
	r.Exp, r.SP = javaRound(ownerExp), javaRound(ownerSP)
	r.ShownExp, r.ShownSP = int64(ownerExp), int64(ownerSP)
	return r
}

// petHungry is Pet.isHungry.
func (s *Server) petHungry(sm *Summon) bool {
	return summon.Hungry(sm.pet.fed, sm.pet.maxMeal(), sm.pet.data.HungryLimit)
}

// petMountType is the mount a pet is (Mobius MountType.findByNpcId).
func petMountType(npcID int) int32 {
	switch npcID {
	case 12526, 12527, 12528, 16038, 16039, 16040, 16068:
		return serverpackets.MountStrider
	case 12621:
		return serverpackets.MountWyvern
	}
	return serverpackets.MountNone
}

// petInventoryItems returns the items of p's pet inventory.
func petInventoryItems(p *Player) []*inventory.Item {
	if p.items.pet == nil {
		return nil
	}
	return p.items.pet.Items()
}

var _ = math.MaxInt32
