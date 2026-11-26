package game

import (
	"time"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/summon"
	"github.com/IPeter/l2go/internal/world"
)

// The summon's packets (Mobius PetInfo, AbstractNpcInfo.SummonInfo,
// PetStatusUpdate, PetDelete): the owner sees its summon through PetInfo,
// everyone else through SummonInfo.

// summonAbnormalDelay is Mobius Summon.updateAbnormalEffect's 50 ms task.
const summonAbnormalDelay = 50 * time.Millisecond

// maxPetLevel is Mobius ExperienceData.getMaxPetLevel: the table's
// maxPetLevel + 1, at most one above the players' MaxLevel.
func (s *Server) maxPetLevel() int {
	return min(s.data.Experience.MaxPetLevel+1, s.MaxLevel()+1)
}

// summonType is the packet value of the summon's kind.
func summonType(sm *Summon) serverpackets.SummonType {
	return serverpackets.SummonType(sm.kind.SummonType())
}

// summonExp returns the summon's exp and the exp of its level and the next
// one (Summon and Pet getExpForThisLevel, getExpForNextLevel).
func (s *Server) summonExp(sm *Summon) (exp, this, next int64) {
	if sm.pet != nil {
		c := s.petCurve(sm)
		return sm.pet.exp, c.ExpFor(sm.pet.level), c.ExpFor(sm.pet.level + 1)
	}
	level, maxPet := sm.level(), s.maxPetLevel()
	exp = sm.servitorExp(s)
	if level < maxPet {
		this = s.ExpForLevel(level)
	}
	if level < maxPet-1 {
		next = s.ExpForLevel(level + 1)
	}
	return exp, this, next
}

// summonFed returns the meal bar: a pet's meal, a servitor's life time left
// (Mobius PetInfo), in the units the client shows.
func summonFed(sm *Summon) (cur, maxFed int32) {
	if sm.pet != nil {
		return int32(sm.pet.fed), int32(sm.pet.maxMeal()) //nolint:gosec // small
	}
	return int32(sm.life.Remaining), int32(sm.life.Total) //nolint:gosec // milliseconds of an hour
}

// petInfoPacket builds the owner's PetInfo of sm with the appearance value
// (Mobius PetInfo(summon, value)).
func (s *Server) petInfoPacket(sm *Summon, value uint8) []byte {
	n := sm.npc
	v := n.statView()
	st := n.Stats
	pos := n.Position()
	motion := st.Motion()
	curFed, maxFed := summonFed(sm)
	exp, this, next := s.summonExp(sm)
	if this > exp {
		this = exp
	}
	name := sm.name
	if sm.pet == nil && n.Template.UsingServerSideName {
		name = n.Template.Name
	}
	var load, maxLoad int32
	if sm.pet != nil {
		load, maxLoad = s.petLoad(sm), s.petMaxLoad(sm)
	}
	ss, sps := s.summonShotsPerHit(sm)
	return serverpackets.NewPetInfoPacket(serverpackets.PetInfo{
		Type:                  summonType(sm),
		ObjectID:              int32(n.ID()),
		DisplayID:             int32(n.Template.DisplayID), //nolint:gosec // NPC IDs are small
		X:                     pos.X,
		Y:                     pos.Y,
		Z:                     pos.Z,
		Heading:               pos.Heading,
		MAtkSpd:               v.MAtkSpd,
		PAtkSpd:               v.PAtkSpd,
		RunSpd:                v.RunSpd,
		WalkSpd:               v.WalkSpd,
		SwimRunSpd:            v.SwimRunSpd,
		SwimWalkSpd:           v.SwimWalkSpd,
		FlyRunSpd:             v.FlyRunSpd,
		FlyWalkSpd:            v.FlyWalkSpd,
		MoveMultiplier:        v.MoveMultiplier,
		AttackSpeedMultiplier: v.AttackSpeedMultiplier,
		CollisionRadius:       float32(v.CollisionRadius),
		CollisionHeight:       float32(v.CollisionHeight),
		Weapon:                s.summonWeapon(sm),
		Armor:                 s.summonArmor(sm),
		ShowName:              sm.owner != nil,
		Running:               motion.Running,
		InCombat:              s.summonInCombat(sm),
		Dead:                  n.Dead,
		Summoned:              sm.showAnim,
		Appearance:            value,
		Name:                  name,
		Title:                 n.Title,
		PvPFlag:               int32(sm.owner.PvPFlag),
		Karma:                 sm.owner.char.Karma,
		CurFed:                curFed,
		MaxFed:                maxFed,
		CurHP:                 javaInt32(n.CurHP),
		MaxHP:                 n.MaxHP(),
		CurMP:                 javaInt32(n.CurMP),
		MaxMP:                 n.MaxMP(),
		SP:                    sm.sp(),
		Level:                 int32(sm.level()), //nolint:gosec // small
		Exp:                   exp,
		ExpThisLevel:          this,
		ExpNextLevel:          next,
		CurLoad:               load,
		MaxLoad:               maxLoad,
		PAtk:                  javaInt32(st.PAtk()),
		PDef:                  javaInt32(st.PDef()),
		MAtk:                  javaInt32(st.MAtk()),
		MDef:                  javaInt32(st.MDef()),
		Accuracy:              int32(st.Accuracy()), //nolint:gosec // small
		Evasion:               int32(st.Evasion()),  //nolint:gosec // small
		Critical:              int32(st.Crit()),     //nolint:gosec // small
		Speed:                 javaInt32(v.MoveSpeed),
		AbnormalEffects:       abnormalMask(n.skill),
		Mountable:             s.summonMountable(sm),
		InWater:               motion.InWater,
		Flying:                motion.Flying,
		SoulShotsPerHit:       ss,
		SpiritShotsPerHit:     sps,
	})
}

// summonInfoPacket builds the SummonInfo another player gets of sm (Mobius
// AbstractNpcInfo.SummonInfo(summon, viewer, value)).
func (s *Server) summonInfoPacket(sm *Summon, viewer *Player, value uint8) []byte {
	n := sm.npc
	v := n.statView()
	pos := n.Position()
	motion := n.Stats.Motion()
	title := ""
	if sm.owner != nil && sm.owner.InGame() {
		title = sm.owner.Name()
	}
	name := sm.name
	if name == "" {
		name = n.Template.Name
	}
	return serverpackets.NewSummonInfoPacket(serverpackets.SummonInfo{
		ObjectID:              int32(n.ID()),
		DisplayID:             int32(n.Template.DisplayID), //nolint:gosec // NPC IDs are small
		Attackable:            viewer != nil && s.summonAutoAttackable(sm, viewer),
		X:                     pos.X,
		Y:                     pos.Y,
		Z:                     pos.Z,
		Heading:               pos.Heading,
		MAtkSpd:               v.MAtkSpd,
		PAtkSpd:               v.PAtkSpd,
		RunSpd:                v.RunSpd,
		WalkSpd:               v.WalkSpd,
		SwimRunSpd:            v.SwimRunSpd,
		SwimWalkSpd:           v.SwimWalkSpd,
		FlyRunSpd:             v.FlyRunSpd,
		FlyWalkSpd:            v.FlyWalkSpd,
		MoveMultiplier:        v.MoveMultiplier,
		AttackSpeedMultiplier: v.AttackSpeedMultiplier,
		CollisionRadius:       float32(v.CollisionRadius),
		CollisionHeight:       float32(v.CollisionHeight),
		Weapon:                s.summonWeapon(sm),
		Armor:                 s.summonArmor(sm),
		InCombat:              s.summonInCombat(sm),
		Dead:                  n.Dead,
		Summoned:              sm.showAnim,
		Appearance:            value,
		Name:                  name,
		Title:                 title,
		PvPFlag:               int32(sm.owner.PvPFlag),
		Karma:                 sm.owner.char.Karma,
		AbnormalEffects:       abnormalMask(n.skill),
		Invisible:             n.Invisible(),
		ViewerGM:              viewer != nil && viewer.IsGM(),
		InWater:               motion.InWater,
		Flying:                motion.Flying,
	})
}

// petStatusUpdatePacket builds PetStatusUpdate of sm.
func (s *Server) petStatusUpdatePacket(sm *Summon) []byte {
	n := sm.npc
	pos := n.Position()
	curFed, maxFed := summonFed(sm)
	exp, this, next := s.summonExp(sm)
	return serverpackets.NewPetStatusUpdatePacket(serverpackets.PetStatusUpdate{
		Type:         summonType(sm),
		ObjectID:     int32(n.ID()),
		X:            pos.X,
		Y:            pos.Y,
		Z:            pos.Z,
		Title:        n.Title,
		CurFed:       curFed,
		MaxFed:       maxFed,
		CurHP:        javaInt32(n.CurHP),
		MaxHP:        n.MaxHP(),
		CurMP:        javaInt32(n.CurMP),
		MaxMP:        n.MaxMP(),
		Level:        int32(sm.level()), //nolint:gosec // small
		Exp:          exp,
		ExpThisLevel: this,
		ExpNextLevel: next,
	})
}

// petDeletePacket builds PetDelete of sm.
func petDeletePacket(sm *Summon) []byte {
	return serverpackets.NewPetDeletePacket(serverpackets.PetDelete{Type: summonType(sm), ObjectID: int32(sm.npc.ID())})
}

// summonInCombat is Mobius Summon.isInCombat: as its owner.
func (s *Server) summonInCombat(sm *Summon) bool { return sm.owner != nil && s.InCombat(sm.owner) }

// summonInfoFor is the InfoProvider side of a summon (Mobius Summon.sendInfo):
// the owner gets PetInfo, then the effect icons and a pet's PetItemList;
// anyone else SummonInfo; MoveToLocation follows while it walks.
func (s *Server) summonInfoFor(sm *Summon, viewer *Player) [][]byte {
	var out [][]byte
	if viewer == sm.owner {
		out = append(out, s.petInfoPacket(sm, serverpackets.SummonAppearanceTeleported))
		if sm.pet != nil {
			out = append(out, s.petItemListPacket(sm.owner)) // pet_items.go
		}
		s.summonIconsLater(sm)
	} else {
		out = append(out, s.summonInfoPacket(sm, viewer, serverpackets.SummonAppearanceTeleported))
	}
	if m := sm.npc.Move; m != nil {
		out = append(out, moveToLocationPacket(sm.npc, m.Destination))
	}
	return out
}

// summonIconsLater sends the owner its summon's effect icons (PartySpelled)
// after PetInfo wiped them (Summon.sendInfo's updateEffectIcons).
func (s *Server) summonIconsLater(sm *Summon) {
	if sm.npc.skill != nil {
		sm.npc.skill.el.UpdateIcons(true)
	}
}

// summonFlush is the coalesced refresh of a summon (Mobius
// Summon.updateAbnormalEffect's SummonInfo to everyone, and PetInfo to the
// owner): every player that sees it gets its view.
func (s *Server) summonFlush(sm *Summon, u world.Update) {
	n := sm.npc
	if u&world.UpdateCharInfo == 0 || !n.InWorld() {
		return
	}
	for v := range s.world.VisiblePlayers(n) {
		vp, ok := v.(*Player)
		if !ok || !s.world.CanSee(v, n) {
			continue
		}
		if vp == sm.owner {
			vp.Send(s.petInfoPacket(sm, serverpackets.SummonAppearanceDefault))
			continue
		}
		vp.Send(s.summonInfoPacket(sm, vp, serverpackets.SummonAppearanceDefault))
	}
}

// summonBroadcastStatus is Mobius Summon.updateAndBroadcastStatus(value):
// the owner gets PetInfo and PetStatusUpdate, the others SummonInfo, and
// the owner the effect icons again.
func (s *Server) summonBroadcastStatus(sm *Summon, value uint8) {
	n, owner := sm.npc, sm.owner
	if owner == nil || !n.InWorld() {
		return
	}
	if sm.pet != nil {
		s.petRefreshOverloaded(sm) // pet_items.go
	}
	if owner.InGame() {
		owner.Send(s.petInfoPacket(sm, value))
		owner.Send(s.petStatusUpdatePacket(sm))
	}
	for v := range s.world.VisiblePlayers(n) {
		vp, ok := v.(*Player)
		if !ok || vp == owner || !s.world.CanSee(v, n) {
			continue
		}
		vp.Send(s.summonInfoPacket(sm, vp, value))
	}
	s.summonIconsLater(sm)
}

// summonStatusUpdate sends the owner PetStatusUpdate (Summon's
// broadcastStatusUpdate adds updateAndBroadcastStatus(1); the HP bar of the
// players targeting it is the NPC's).
func (s *Server) summonStatusUpdate(sm *Summon) {
	if sm.owner != nil && sm.owner.InGame() && sm.npc.InWorld() {
		sm.owner.Send(s.petStatusUpdatePacket(sm))
	}
}

// summonAbnormalChanged coalesces the info refresh after an effect changed
// the summon's look (updateAbnormalEffect's 50 ms task).
func (s *Server) summonAbnormalChanged(sm *Summon) {
	if sm.abnormalTimer != 0 {
		return
	}
	sm.abnormalTimer = s.loop.After(summonAbnormalDelay, func() {
		sm.abnormalTimer = 0
		if sm.npc.InWorld() {
			s.summonFlush(sm, world.UpdateCharInfo)
		}
	})
}

// summonWeapon and summonArmor are the item IDs a pet wears (Pet.getWeapon,
// getArmor); servitors show none.
func (s *Server) summonWeapon(sm *Summon) int32 {
	if sm.pet == nil {
		return 0
	}
	return s.petWornItemID(sm.owner, petSlotWeapon)
}

func (s *Server) summonArmor(sm *Summon) int32 {
	if sm.pet == nil {
		return 0
	}
	return s.petWornItemID(sm.owner, petSlotArmor)
}

// summonMountable is Mobius isMountable: striders and wyverns.
func (s *Server) summonMountable(sm *Summon) bool {
	return sm.pet != nil && petMountType(sm.npc.Template.ID) != serverpackets.MountNone
}

// summonShotsPerHit is getSoulShotsPerHit and getSpiritShotsPerHit: a pet's
// level data, a servitor's template shots (at least 1).
func (s *Server) summonShotsPerHit(sm *Summon) (ss, sps int32) {
	if sm.pet != nil {
		if l := sm.pet.levelData(); l != nil {
			return int32(l.SoulshotCount), int32(l.SpiritshotCount) //nolint:gosec // small
		}
	}
	ss, sps = 1, 1
	if v := sm.npc.Template.Shots.Soul; v > 0 {
		ss = int32(v) //nolint:gosec // small
	}
	if v := sm.npc.Template.Shots.Spirit; v > 0 {
		sps = int32(v) //nolint:gosec // small
	}
	return ss, sps
}

// sp returns the summon's SP (a pet's own).
func (sm *Summon) sp() int32 {
	if sm.pet != nil {
		return int32(min(sm.pet.sp, 1<<31-1)) //nolint:gosec // clamped
	}
	return 0
}

// servitorExp is the exp a servitor shows (Mobius Summon effect: the exp of
// level % maxPetLevel, of maxPetLevel − 1 at and above it).
func (sm *Summon) servitorExp(s *Server) int64 {
	maxPet := s.maxPetLevel()
	level := sm.level()
	if level >= maxPet {
		return s.ExpForLevel(maxPet - 1)
	}
	return s.ExpForLevel(level % maxPet)
}

// summonSpelled sends the owner, and its party, PartySpelled of its summon's
// effects (Mobius EffectList.updateEffectIcons for a summon).
func (s *Server) summonSpelled(owner *Player, u skills.IconUpdate) {
	sm := s.SummonOf(owner)
	if sm == nil {
		return
	}
	typ := serverpackets.PartySpelledServitor
	if sm.kind.IsPet() {
		typ = serverpackets.PartySpelledPet
	}
	pkt := serverpackets.NewPartySpelledPacket(serverpackets.PartySpelled{Type: typ, ObjectID: int32(sm.npc.ID()), Effects: u.Party})
	if pt := owner.party.party; pt != nil {
		pt.Broadcast(pkt)
		return
	}
	owner.Send(pkt)
}

var _ = summon.Servitor
