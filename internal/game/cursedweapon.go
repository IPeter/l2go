package game

import (
	"context"
	"errors"
	"time"

	"github.com/IPeter/l2go/internal/cursedweapon"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The cursed weapon feature (Plan 14 M11m): Demonic Sword Zariche and Blood
// Sword Akamanah, Mobius CursedWeaponsManager and CursedWeapon in the
// world. The rules and the state are in internal/cursedweapon; this file
// moves the items, changes the holder, sends the packets and saves.
//
//   - cursedweapon.go: the feature, the drop on a kill, the pickup, the
//     kills, the holder's death, the end of a weapon, the restore at
//     startup and at enter world, and the packets D0:22 and D0:23.
//   - cursedweapon_gm.go: //cw_info, //cw_info_menu, //cw_add, //cw_remove,
//     //cw_goto and //cw_reload (AdminCursedWeapons).
//   - cursedweapon_store.go: the cursed weapon part of a joint save.
//
// The state is saved at every transition (decision X4) in the joint save
// of the holder (storage.Batch.CursedWeapons), so a pickup, a kill, a death
// or the end of a weapon is stored with the holder's karma and items, or
// not at all. Mobius saves at the pickup and at each kill only; the death
// drop and //cw_add are saved here too.
//
// The rest of the game asks Player.CursedWeaponEquipped: the drop on death
// and the kill count (death.go, pvp.go), the karma loss on exp (levels.go),
// the info packets (info.go), skills (cast_player.go), equipment
// (items_use.go), parties (party_invite.go), chat (chat.go) and the Lua
// player:is_cursed_weapon_equipped().
func init() { registerFeature("cursedweapon", setupCursedWeapons) }

// System messages of the cursed weapons.
const (
	cwMsgDisappeared   serverpackets.SystemMessageID = 302  // "$s1 has disappeared."
	cwMsgMinutesLeft   serverpackets.SystemMessageID = 1814 // "$s1 has $s2 minute(s) of usage time remaining."
	cwMsgDroppedIn     serverpackets.SystemMessageID = 1815 // "$s2 was dropped in the $s1 region."
	cwMsgOwnerAppeared serverpackets.SystemMessageID = 1816 // "The owner of $s2 has appeared in the $s1 region."
	cwMsgOwnerLoggedIn serverpackets.SystemMessageID = 1817 // "$s2's owner has logged into the $s1 region."
)

// cwLoadTimeout bounds reading the cursed weapon state at startup.
const cwLoadTimeout = 30 * time.Second

// cwChatRefusal is Say2's answer to a holder's shout or trade chat.
const cwChatRefusal = "Shout and trade chatting cannot be used while possessing a cursed weapon."

// cwFormalWearItemID is Formal Wear, which a holder cannot put on (UseItem).
const cwFormalWearItemID = 6408

// cursedFeature is the feature's state on the Server (Server.cursed).
// Loop-only, except what setup makes.
type cursedFeature struct {
	set   *cursedweapon.Set
	store storage.CursedWeaponStore // nil keeps the state in memory only
	rt    map[int32]*cursedRuntime  // by item ID
}

// cursedRuntime is what Mobius CursedWeapon keeps besides its state: the
// life timer (RemoveTask), the online holder (_player) and the item
// (_item), on the ground or carried.
type cursedRuntime struct {
	timer  world.TimerID
	player *Player
	item   *inventory.Item
	ground *GroundItem // while it lies on the ground
	// itemObjectID is the object ID of the held copy, also while its holder
	// is offline (from the stored row), 0 when unknown.
	itemObjectID int32
}

// setItem remembers the weapon's item.
func (rt *cursedRuntime) setItem(it *inventory.Item) {
	rt.item = it
	if it != nil {
		rt.itemObjectID = it.ObjectID()
	}
}

// setupCursedWeapons makes the weapons of the data, restores the held ones
// from the store and deletes stray copies (Mobius CursedWeaponsManager.init:
// load, restore, controlPlayers), then registers the packets, hooks and GM
// commands.
func setupCursedWeapons(s *Server) error {
	f := &s.cursed
	f.set = cursedweapon.NewSet(s.data.CursedWeapons, func(skillID int) int {
		if c := s.SkillCatalog(); c != nil {
			return c.MaxLevel(skillID)
		}
		return 0
	})
	f.rt = make(map[int32]*cursedRuntime)
	for _, w := range f.set.Weapons() {
		f.rt[w.ItemID()] = &cursedRuntime{}
	}
	f.store, _ = s.chars.(storage.CursedWeaponStore)
	if err := s.restoreCursedWeapons(); err != nil {
		return err
	}
	s.OnPlayerEnter(s.cursedEnter)
	s.OnPlayerLeave(s.cursedLeave)
	return errors.Join(
		s.RegisterEx(clientpackets.ExOpRequestCursedWeaponList, s.handleCursedWeaponList),
		s.RegisterEx(clientpackets.ExOpRequestCursedWeaponLocation, s.handleCursedWeaponLocation),
		s.registerCursedWeaponGM(),
	)
}

// restoreCursedWeapons reads the held weapons (CursedWeaponsManager.restore
// and reActivate): a weapon whose life ended while the server was down ends
// now, the others get their life timer. Then every copy of a weapon that is
// not held is deleted and its owner gets the weapon's saved karma and PK
// count, 0 and 0 (controlPlayers). It runs before the loop starts, so it
// writes at once.
func (s *Server) restoreCursedWeapons() error {
	f := &s.cursed
	if f.store == nil || len(f.set.Weapons()) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), cwLoadTimeout)
	defer cancel()
	rows, err := f.store.LoadCursedWeapons(ctx)
	if err != nil {
		return err
	}
	ids := make([]int32, 0, len(f.set.Weapons()))
	for _, w := range f.set.Weapons() {
		ids = append(ids, w.ItemID())
	}
	copies, err := f.store.ItemsByTemplate(ctx, ids)
	if err != nil {
		return err
	}
	var b storage.Batch
	gone := make(map[int32]bool) // copies the batch deletes
	now := s.Now()
	for _, r := range rows {
		w := f.set.Get(r.ItemID)
		if w == nil {
			s.log.Warn("game: a stored cursed weapon is not in the data; left as it is", "item", r.ItemID)
			continue
		}
		w.Activate(r.CharID, r.PlayerKarma, r.PlayerPKKills)
		w.NbKills, w.EndTime = r.NbKills, r.EndTime
		if w.TimeLeft(now) > 0 {
			f.rt[r.ItemID].itemObjectID = r.ItemObjectID // the copy comes with its holder (cursedEnter)
			s.startCursedTimer(w)
			s.log.Info("cursed weapon restored", "item", r.ItemID, "holder", r.CharID, "kills", r.NbKills, "minutesLeft", w.MinutesLeft(now))
			continue
		}
		// endOfLife offline: the holder's copies go (Mobius deletes by owner
		// and item ID), its karma and PK count come back.
		s.log.Info("cursed weapon ended while the server was down", "item", r.ItemID, "holder", r.CharID)
		b = cursedOfflineEnd(b, w, 0)
		for _, c := range copies {
			if c.Owner == r.CharID && c.Item.ItemID == r.ItemID {
				b.Items = append(b.Items, storage.OwnerItemChanges{Owner: c.Owner, Destroyed: []int32{c.Item.ObjectID}})
				gone[c.Item.ObjectID] = true
			}
		}
		w.Reset()
	}
	// controlPlayers: a copy of a weapon nobody holds goes, and its owner
	// gets the weapon's karma and PK count, which are 0 for such a weapon.
	for _, c := range copies {
		w := f.set.Get(c.Item.ItemID)
		if w.Activated || gone[c.Item.ObjectID] {
			continue
		}
		s.log.Warn("game: a character owns a cursed weapon it should not; deleted", "character", c.Owner, "item", c.Item.ItemID)
		b.Items = append(b.Items, storage.OwnerItemChanges{Owner: c.Owner, Destroyed: []int32{c.Item.ObjectID}})
		b.Patches = append(b.Patches, cursedKarmaPatch(c.Owner, w))
		gone[c.Item.ObjectID] = true
		if !containsInt32(b.DeletedCursedWeapons, w.ItemID()) {
			b.DeletedCursedWeapons = append(b.DeletedCursedWeapons, w.ItemID())
		}
	}
	if b.Empty() {
		return nil
	}
	bw, ok := s.chars.(storage.BatchWriter)
	if !ok {
		return errNoBatchWriter
	}
	return bw.WriteBatch(ctx, b)
}

// cursedOfflineEnd adds the end of a weapon whose holder is offline to b
// (endOfLife: DELETE FROM items, UPDATE characters SET karma, pkkills, and
// the row goes).
func cursedOfflineEnd(b storage.Batch, w *cursedweapon.Weapon, itemObjectID int32) storage.Batch {
	if itemObjectID != 0 {
		b.Items = append(b.Items, storage.OwnerItemChanges{Owner: w.PlayerID, Destroyed: []int32{itemObjectID}})
	}
	b.Patches = append(b.Patches, cursedKarmaPatch(w.PlayerID, w))
	b.DeletedCursedWeapons = append(b.DeletedCursedWeapons, w.ItemID())
	return b
}

// cursedKarmaPatch gives a character the karma and PK count a weapon kept.
func cursedKarmaPatch(charID int32, w *cursedweapon.Weapon) storage.CharacterPatch {
	return storage.CharacterPatch{ObjectID: charID, Fields: storage.PatchKarma | storage.PatchPKKills,
		Values: storage.PatchValues{Karma: w.PlayerKarma, PKKills: w.PlayerPKKills}}
}

func containsInt32(list []int32, v int32) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// ----- questions the rest of the game asks -----

// CursedWeaponEquipped reports whether p holds a cursed weapon (Mobius
// isCursedWeaponEquipped). It is also player:is_cursed_weapon_equipped() of
// the Lua scripts (scripting.PlayerCursedWeapon). Loop-only.
func (p *Player) CursedWeaponEquipped() bool { return p.CursedWeaponEquippedID() != 0 }

// CursedWeaponEquippedID returns the item ID of the cursed weapon p holds,
// 0 for none (Mobius getCursedWeaponEquippedId). Loop-only.
func (p *Player) CursedWeaponEquippedID() int32 {
	if p == nil || p.srv == nil {
		return 0
	}
	if w := p.srv.cursed.set.HeldBy(p.ObjectID()); w != nil {
		return w.ItemID()
	}
	return 0
}

// cursedWeaponLevel is the weapon level UserInfo and CharInfo show, 0
// without a cursed weapon.
func (p *Player) cursedWeaponLevel() int32 {
	if p == nil || p.srv == nil {
		return 0
	}
	if w := p.srv.cursed.set.HeldBy(p.ObjectID()); w != nil {
		return w.Level()
	}
	return 0
}

// IsCursedWeapon reports whether an item is a cursed weapon (Mobius
// CursedWeaponsManager.isCursed). Loop-only.
func (s *Server) IsCursedWeapon(itemID int32) bool { return s.cursed.set.IsCursed(itemID) }

// cursedBlocksEquip is UseItem's rule for a holder: no formal wear, and no
// weapon or shield, the cursed weapon itself included, goes on or off.
func cursedBlocksEquip(p *Player, it *inventory.Item) bool {
	if !p.CursedWeaponEquipped() {
		return false
	}
	if it.ItemID() == cwFormalWearItemID {
		return true
	}
	switch it.Template().Part() {
	case data.BodyPartLRHand, data.BodyPartLHand, data.BodyPartRHand:
		return true
	}
	return false
}

// cursedSkillAllowed is Player.checkUseMagicConditions for a holder: only
// Void Burst and Void Flow.
func cursedSkillAllowed(p *Player, skillID int) bool {
	return !p.CursedWeaponEquipped() || skillID == cursedweapon.SkillVoidBurst || skillID == cursedweapon.SkillVoidFlow
}

// ----- the drop on a kill -----

// cursedCheckDrop is CursedWeaponsManager.checkDrop for a monster p killed
// (Attackable.doItemDrop, before the drop list): no drop from the excluded
// NPC types; otherwise each weapon not in the world rolls Rnd.get(100000) <
// 1 in turn, and the first that rolls drops by the corpse.
func (s *Server) cursedCheckDrop(n *Npc, p *Player) {
	f := &s.cursed
	if len(f.set.Weapons()) == 0 {
		return
	}
	excluded := cursedweapon.Excluded(func(typ string) bool {
		return s.worldData != nil && s.worldData.NpcTypeIs(n.Template.Type, typ)
	})
	w := f.set.CheckDrop(s.rnd, excluded)
	if w == nil {
		return
	}
	s.cursedDropFromMonster(w, n, p)
}

// cursedDropFromMonster is CursedWeapon.dropIt from a monster and the rest
// of checkDrop: the weapon falls by the corpse for good (no auto-destroy),
// the sky turns red and the earth shakes for everyone online, everyone is
// told the region, and the life starts.
func (s *Server) cursedDropFromMonster(w *cursedweapon.Weapon, n *Npc, p *Player) {
	rt := s.cursed.rt[w.ItemID()]
	t, ok := s.data.ItemTemplate(int(w.ItemID()))
	if !ok || s.items.pool == nil {
		s.log.Error("game: a cursed weapon cannot drop", "item", w.ItemID())
		return
	}
	id, ok := s.items.pool.TryNext()
	if !ok {
		s.log.Error("game: no object ID left; a cursed weapon does not drop", "item", w.ItemID())
		return
	}
	it := inventory.NewItem(id, t, 1)
	pos := n.Position()
	x := pos.X + int32(s.rnd.Get(2*npcDropRange+1)) - npcDropRange //nolint:gosec // below 141
	y := pos.Y + int32(s.rnd.Get(2*npcDropRange+1)) - npcDropRange //nolint:gosec // below 141
	g, err := s.DropOnGround(it, world.Position{X: x, Y: y, Z: pos.Z + npcDropHeight}, GroundDrop{Dropper: n, ProtectFor: p})
	if err != nil {
		s.log.Error("game: a cursed weapon cannot go into the world", "item", w.ItemID(), "err", err)
		return
	}
	s.loop.Cancel(g.timer) // setDropTime(0): ItemsAutoDestroy leaves it
	g.timer = 0
	rt.ground = g
	rt.setItem(it)

	pp := p.Position()
	s.world.BroadcastAll(serverpackets.NewExRedSkyPacket(cursedweapon.RedSkySeconds))
	s.world.BroadcastAll(serverpackets.NewEarthquakePacket(serverpackets.Earthquake{X: pp.X, Y: pp.Y, Z: pp.Z,
		Intensity: cursedweapon.EarthquakeIntensity, Duration: cursedweapon.EarthquakeDuration}))
	s.cursedAnnounce(cwMsgDroppedIn, serverpackets.ZoneNameParam(pp.X, pp.Y, pp.Z), s.itemNameParam(w.ItemID()))

	w.DroppedByMonster(s.Now())
	s.startCursedTimer(w)
	s.log.Info("cursed weapon dropped", append(p.logArgs(), "item", w.ItemID(), "npc", n.Template.ID)...)
}

// startCursedTimer starts the life check (RemoveTask at a fixed rate of
// durationLost × 12 s): the weapon ends once its end time passed.
func (s *Server) startCursedTimer(w *cursedweapon.Weapon) {
	rt := s.cursed.rt[w.ItemID()]
	s.loop.Cancel(rt.timer)
	rt.timer = s.loop.Every(w.CheckPeriod(), func() {
		if w.Expired(s.Now()) {
			s.cursedEndOfLife(w)
		}
	})
}

// ----- the pickup -----

// cursedPickUp is Player.doPickupItem for a cursed weapon, which goes to no
// party member and is not announced: it goes into p's inventory and
// activates. It reports whether it was one.
func (s *Server) cursedPickUp(p *Player, it *inventory.Item) bool {
	if !s.cursed.set.IsCursed(it.ItemID()) {
		return false
	}
	s.cursed.rt[it.ItemID()].ground = nil
	s.addItemObject(p, it)
	s.cursedActivate(p, it)
	return true
}

// cursedActivate is CursedWeaponsManager.activate for a cursed weapon that
// came into p's inventory (Player.addItem): a holder of the other weapon
// gets a stage of kills and the new one ends; anyone else becomes its
// holder.
func (s *Server) cursedActivate(p *Player, it *inventory.Item) {
	f := &s.cursed
	w := f.set.Get(it.ItemID())
	if w == nil {
		return
	}
	if held := f.set.HeldBy(p.ObjectID()); held != nil {
		levelUp := held.AddBonusStage()
		s.cursedKillsChanged(held, p, levelUp)
		rt := f.rt[w.ItemID()]
		rt.player = p
		rt.setItem(it)
		s.cursedEndOfLife(w)
		return
	}
	s.cursedActivateWeapon(w, p, it)
}

// cursedActivateWeapon is CursedWeapon.activate: p keeps its karma and PK
// count in the weapon, gets karma 9,999,999 and no PKs, leaves its party,
// gets the skills, wears the weapon, heals fully, plays the social action
// and is announced to everyone. A rider gets off first; one who may not
// drops the weapon again (mount.go).
func (s *Server) cursedActivateWeapon(w *cursedweapon.Weapon, p *Player, it *inventory.Item) {
	if s.mounted(p) && !s.dismount(p) {
		s.mountDropCursed(p, it)
		return
	}
	rt := s.cursed.rt[w.ItemID()]
	w.Activate(p.ObjectID(), p.char.Karma, p.char.PKKills)
	rt.player, rt.ground = p, nil
	rt.setItem(it)

	s.SetKarma(p, cursedweapon.HolderKarma)
	p.char.PKKills = 0
	if pt := p.party.party; pt != nil {
		s.removePartyMember(pt, p, partyLeaveExpelled)
	}
	s.cursedGiveSkill(w, p)

	if !it.Equipped() {
		p.inv.Equip(it, inventory.Wearer{GM: p.IsGM()})
		s.refreshExpertisePenalty(p) // penalty.go, as any equip
	}
	p.SendSystemMessage(sysMsgEquippedS1, s.itemNameParam(w.ItemID()))
	v := p.stats.View()
	p.SetCurrentHPMP(float64(v.MaxHP), float64(v.MaxMP))
	p.SetCurrentCP(float64(v.MaxCP))
	p.inv.TakeUpdates() // the item list below shows them
	s.SendItemList(p, false)
	p.BroadcastUserInfo()
	p.after(cursedweapon.SocialActionDelay, func() {
		s.world.Broadcast(p, serverpackets.NewSocialActionPacket(serverpackets.SocialAction{
			ObjectID: p.ObjectID(), ActionID: cursedweapon.SocialActionID}), true)
	})
	pos := p.Position()
	s.cursedAnnounce(cwMsgOwnerAppeared, serverpackets.ZoneNameParam(pos.X, pos.Y, pos.Z), s.itemNameParam(w.ItemID()))
	s.cursedSave(w, p)
	s.log.Info("cursed weapon activated", append(p.logArgs(), "item", w.ItemID(), "minutesLeft", w.MinutesLeft(s.Now()))...)
}

// ----- the kills -----

// cursedKilled counts a player the holder killed (onKillUpdatePvPKarma:
// the kill counts for the weapon and nothing else). It reports whether
// killer is a holder.
func (s *Server) cursedKilled(killer *Player) bool {
	w := s.cursed.set.HeldBy(killer.ObjectID())
	if w == nil {
		return false
	}
	levelUp := w.AddKill()
	s.cursedKillsChanged(w, killer, levelUp)
	return true
}

// cursedKillsChanged is the rest of increaseKills: an online holder's PK
// count is the kill count (UserInfo) and the skill steps up at a stage;
// the weapon is saved.
func (s *Server) cursedKillsChanged(w *cursedweapon.Weapon, p *Player, levelUp bool) {
	if p != nil && p.InGame() {
		p.char.PKKills = w.NbKills
		p.UpdateUserInfo()
		if levelUp {
			s.cursedGiveSkill(w, p)
		}
	}
	s.cursedSave(w, p)
}

// cursedKillerOf reports whether the creature that killed someone holds a
// cursed weapon: its victims drop nothing and lose no exp (Player.doDie).
func cursedKillerOf(killer Creature) bool {
	kp, ok := killer.(*Player)
	return ok && kp != nil && kp.CursedWeaponEquipped()
}

// ----- the holder's death -----

// cursedHolderDied is CursedWeaponsManager.drop for a holder that died with
// a killer (Player.doDie): Rnd.get(100) <= 50 ends the weapon; otherwise it
// falls at the holder's feet, the holder gets its karma and PK count back
// and loses the skills, and everyone is told the region. It reports whether
// p was a holder.
func (s *Server) cursedHolderDied(p *Player) bool {
	w := s.cursed.set.HeldBy(p.ObjectID())
	if w == nil {
		return false
	}
	if w.RollDisappear(s.rnd) {
		s.cursedEndOfLife(w)
		return true
	}
	rt := s.cursed.rt[w.ItemID()]
	it := p.inv.ByItemID(w.ItemID())
	w.DroppedByHolder()
	if it != nil {
		if it.Equipped() {
			p.inv.UnequipItem(it)
		}
		if dropped, err := p.inv.DropItem(it); err == nil {
			pos := p.Position()
			dx := int32(s.rnd.Get(50)) - 25 //nolint:gosec // below 50
			dy := int32(s.rnd.Get(50)) - 25 //nolint:gosec // below 50
			g, err := s.DropOnGround(dropped, world.Position{X: pos.X + dx, Y: pos.Y + dy, Z: pos.Z + 20}, GroundDrop{Dropper: p, PlayerDrop: true})
			if err != nil {
				s.log.Error("game: a cursed weapon cannot fall on the ground", append(p.logArgs(), "item", w.ItemID(), "err", err)...)
			}
			rt.ground = g
			rt.setItem(dropped)
			s.itemsChanged(p, true, nil)
			p.SendSystemMessage(sysMsgDroppedS1, s.itemNameParam(w.ItemID()))
		}
	}
	rt.player = nil
	s.SetKarma(p, w.PlayerKarma)
	p.char.PKKills = w.PlayerPKKills
	s.cursedRemoveSkill(w, p)
	s.AbortAttack(p)
	pos := p.Position()
	s.cursedAnnounce(cwMsgDroppedIn, serverpackets.ZoneNameParam(pos.X, pos.Y, pos.Z), s.itemNameParam(w.ItemID()))
	p.BroadcastUserInfo()
	s.cursedSave(w, p)
	return true
}

// ----- the end of a weapon -----

// cursedEndOfLife is CursedWeapon.endOfLife: an online holder gets its
// karma and PK count back, loses the skills and the weapon; an offline
// holder's copy is deleted and its karma and PK count written; a copy in
// the inventory of a holder of the other weapon goes; one on the ground
// leaves the world. Everyone is told it disappeared, and the weapon no
// longer exists.
func (s *Server) cursedEndOfLife(w *cursedweapon.Weapon) {
	rt := s.cursed.rt[w.ItemID()]
	var save *Player
	offline := false
	switch {
	case w.Activated:
		if p, ok := s.Player(world.ObjectID(w.PlayerID)); ok && p.InGame() {
			s.log.Info("cursed weapon removed online", append(p.logArgs(), "item", w.ItemID())...)
			s.AbortAttack(p)
			s.SetKarma(p, w.PlayerKarma)
			p.char.PKKills = w.PlayerPKKills
			s.cursedRemoveSkill(w, p)
			p.inv.UnequipBodyPart(data.BodyPartLRHand)
			if it := p.inv.ByItemID(w.ItemID()); it != nil {
				s.DestroyItem(p, it, 1, false)
			}
			s.SendItemList(p, true)
			p.BroadcastUserInfo()
			save = p
		} else {
			s.log.Info("cursed weapon removed offline", "item", w.ItemID(), "holder", w.PlayerID)
			s.SaveJoint(nil, cursedOfflineEnd(storage.Batch{}, w, rt.itemObjectID))
			offline = true
		}
	case rt.player != nil && rt.player.inv.ByItemID(w.ItemID()) != nil:
		p := rt.player
		s.DestroyItem(p, p.inv.ByItemID(w.ItemID()), 1, false)
		s.SendItemList(p, true)
		p.BroadcastUserInfo()
		save = p
	case rt.ground != nil:
		s.removeGroundItem(rt.ground)
		s.log.Info("cursed weapon removed from the world", "item", w.ItemID())
	}

	s.cursedAnnounce(cwMsgDisappeared, s.itemNameParam(w.ItemID()))
	s.loop.Cancel(rt.timer)
	*rt = cursedRuntime{}
	w.Reset()
	if !offline {
		s.cursedSave(w, save) // the row goes (removeFromDb), with the holder
	}
}

// ----- enter world and leave -----

// cursedEnter restores a holder entering the world (checkPlayer and
// cursedOnLogin: the skills, everyone is told the region, the holder its
// minutes left) and deletes the copies of a player who holds none
// (EnterWorld's "remove demonic weapon").
func (s *Server) cursedEnter(p *Player) {
	f := &s.cursed
	if w := f.set.HeldBy(p.ObjectID()); w != nil {
		rt := f.rt[w.ItemID()]
		it := p.inv.ByItemID(w.ItemID())
		if it == nil {
			s.log.Warn("game: a cursed weapon holder enters without the weapon; it ends", append(p.logArgs(), "item", w.ItemID())...)
			s.cursedEndOfLife(w)
		} else {
			rt.player, rt.ground = p, nil
			rt.setItem(it)
			s.cursedGiveSkill(w, p)
			pos := p.Position()
			s.cursedAnnounce(cwMsgOwnerLoggedIn, serverpackets.ZoneNameParam(pos.X, pos.Y, pos.Z), s.itemNameParam(w.ItemID()))
			p.SendSystemMessage(cwMsgMinutesLeft, s.itemNameParam(w.ItemID()), serverpackets.NumberParam(w.MinutesLeft(s.Now())))
		}
	}
	if p.CursedWeaponEquipped() {
		return
	}
	for _, w := range f.set.Weapons() {
		if it := p.inv.ByItemID(w.ItemID()); it != nil {
			s.log.Warn("game: a stray cursed weapon is deleted", append(p.logArgs(), "item", w.ItemID())...)
			s.DestroyItem(p, it, it.Count(), true)
		}
	}
}

// cursedLeave forgets a holder that leaves (Player.deleteMe: setPlayer(null)).
func (s *Server) cursedLeave(p *Player) {
	for _, rt := range s.cursed.rt {
		if rt.player == p {
			rt.player = nil
		}
	}
}

// ----- packets -----

// handleCursedWeaponList answers RequestCursedWeaponList (D0:22) with
// ExCursedWeaponList: every cursed weapon's item ID.
func (s *Server) handleCursedWeaponList(p *Player, body []byte) {
	if _, err := clientpackets.ParseRequestCursedWeaponList(body); err != nil {
		return
	}
	ws := s.cursed.set.Weapons()
	ids := make([]int32, len(ws))
	for i, w := range ws {
		ids[i] = w.ItemID()
	}
	p.Send(serverpackets.NewExCursedWeaponListPacket(ids))
}

// handleCursedWeaponLocation answers RequestCursedWeaponLocation (D0:23):
// ExCursedWeaponLocation with each weapon in the world and where it is,
// nothing when none is.
func (s *Server) handleCursedWeaponLocation(p *Player, body []byte) {
	if _, err := clientpackets.ParseRequestCursedWeaponLocation(body); err != nil {
		return
	}
	var list []serverpackets.CursedWeaponLocation
	for _, w := range s.cursed.set.Weapons() {
		if !w.Active() {
			continue
		}
		if pos, ok := s.cursedPosition(w); ok {
			list = append(list, serverpackets.CursedWeaponLocation{ItemID: w.ItemID(), Activated: w.Activated, X: pos.X, Y: pos.Y, Z: pos.Z})
		}
	}
	if len(list) > 0 {
		p.Send(serverpackets.NewExCursedWeaponLocationPacket(list))
	}
}

// cursedPosition is CursedWeapon.getWorldPosition: the online holder's
// position, or the item's on the ground.
func (s *Server) cursedPosition(w *cursedweapon.Weapon) (world.Position, bool) {
	rt := s.cursed.rt[w.ItemID()]
	switch {
	case w.Activated && rt.player != nil:
		return rt.player.Position(), true
	case w.Dropped && rt.ground != nil && rt.ground.InWorld():
		return rt.ground.Position(), true
	}
	return world.Position{}, false
}

// ----- helpers -----

// cursedGiveSkill gives the holder the weapon's skill at its level and Void
// Burst and Void Flow, not stored with the character (giveSkill).
func (s *Server) cursedGiveSkill(w *cursedweapon.Weapon, p *Player) {
	for _, sk := range []struct{ id, level int }{
		{w.SkillID(), w.SkillLevel()},
		{cursedweapon.SkillVoidBurst, 1},
		{cursedweapon.SkillVoidFlow, 1},
	} {
		if skill, ok := s.Skill(sk.id, sk.level); ok {
			s.AddSkill(p, skill, false)
		}
	}
	p.SendSkillList()
}

// cursedRemoveSkill takes the three skills away (removeSkill), with their
// shortcuts.
func (s *Server) cursedRemoveSkill(w *cursedweapon.Weapon, p *Player) {
	for _, id := range []int{w.SkillID(), cursedweapon.SkillVoidBurst, cursedweapon.SkillVoidFlow} {
		s.RemoveSkill(p, id, true, false)
	}
	p.SendSkillList()
}

// cursedAnnounce sends a system message to everyone online
// (CursedWeaponsManager.announce).
func (s *Server) cursedAnnounce(id serverpackets.SystemMessageID, params ...serverpackets.SystemMessageParam) {
	s.world.BroadcastAll(serverpackets.NewSystemMessagePacket(serverpackets.SystemMessage{ID: id, Params: params}))
}

// cursedSave saves the weapon at once (decision X4): its row while it is
// held, none otherwise, in one joint save with p (the holder, or nil).
func (s *Server) cursedSave(w *cursedweapon.Weapon, p *Player) {
	var b storage.Batch
	if w.Activated {
		row := storage.CursedWeapon{ItemID: w.ItemID(), CharID: w.PlayerID, PlayerKarma: w.PlayerKarma,
			PlayerPKKills: w.PlayerPKKills, NbKills: w.NbKills, EndTime: w.EndTime}
		row.ItemObjectID = s.cursed.rt[w.ItemID()].itemObjectID
		b.CursedWeapons = []storage.CursedWeapon{row}
	} else {
		b.DeletedCursedWeapons = []int32{w.ItemID()}
	}
	var players []*Player
	if p != nil && p.state == stateInGame {
		players = []*Player{p}
	}
	s.SaveJoint(players, b)
}
