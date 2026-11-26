package game

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/enchant"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
)

// The enchant feature (enchant*.go, crystallize.go; Plan 14, M11a): enchant
// scrolls (Mobius EnchantScrolls, RequestEnchantItem), crystallization
// (RequestCrystallizeItem) and the over-enchant protection. The rules are
// internal/enchant; the rates are data/enchant/enchant.json.
func init() { registerFeature("enchant", setupEnchant) }

// handlerEnchantScrolls is the item handler of the enchant scrolls.
const handlerEnchantScrolls = "EnchantScrolls"

// The firework of an announced enchant (Mobius CommonSkill.FIREWORK, skill
// 2024 level 1), with its times from the skill data when the server has
// none.
const (
	fireworkSkill      = 2024
	fireworkReuseDelay = 1000
)

// System messages of enchanting (Mobius SystemMessageId).
const (
	sysMsgEnchanted                serverpackets.SystemMessageID = 62   // "Your $s1 has been successfully enchanted."
	sysMsgEnchantedLevel           serverpackets.SystemMessageID = 63   // "Your +$S1 $S2 has been successfully enchanted."
	sysMsgEnchantCrystallized      serverpackets.SystemMessageID = 64   // "The enchantment has failed! Your $s1 has been crystallized."
	sysMsgEnchantCrystallizedLevel serverpackets.SystemMessageID = 65   // "The enchantment has failed! Your +$s1 $s2 has been crystallized."
	sysMsgSelectItemToEnchant      serverpackets.SystemMessageID = 303  // "Select item to enchant."
	sysMsgIncorrectItemCount2      serverpackets.SystemMessageID = 351  // "Incorrect item count."
	sysMsgInappropriateEnchant     serverpackets.SystemMessageID = 355  // "Inappropriate enchant conditions."
	sysMsgEnchantCancelled         serverpackets.SystemMessageID = 423  // "You have cancelled the enchanting process."
	sysMsgBlessedEnchantFailed     serverpackets.SystemMessageID = 1517 // "Failed in Blessed Enchant. The enchant value of the item became 0."
	sysMsgNoEnchantInStore         serverpackets.SystemMessageID = 1688 // "You cannot enchant while operating a Private Store or Private Workshop."
)

// safeEnchantFailedText is what a failed safe (ancient crystal) scroll says.
const safeEnchantFailedText = "Enchant failed. The enchant level for the corresponding item will be exactly retained."

// enchantFeature is the enchant feature's state on the Server
// (Server.enchant). Loop-only.
type enchantFeature struct {
	// table is the enchant data; nil without one, and then scrolls do
	// nothing and nothing is over-enchanted.
	table *enchant.Table
	// rnd is the source of the enchant rolls (the game's; tests replace it).
	rnd rnd.Source
	// players holds the active scroll and the flood limit of each player.
	players map[*Player]*enchantState
}

// enchantState is a player's enchanting state.
type enchantState struct {
	// scroll is the object ID of the scroll used last (Mobius
	// activeEnchantItemId), 0 for none.
	scroll int32
	// next is when the next enchant request is taken (flood protection).
	next time.Time
}

// setupEnchant loads the enchant data and registers the packets and hooks.
func setupEnchant(s *Server) error {
	f := &s.enchant
	f.rnd = s.rnd
	f.players = make(map[*Player]*enchantState)
	if dir := s.settings.DataDir; dir != "" {
		switch d, err := enchant.Load(dir); {
		case err == nil:
			s.UseEnchantData(d)
		case errors.Is(err, fs.ErrNotExist):
			s.log.Info("game: no enchant data, enchant scrolls do nothing", "file", filepath.Join(dir, enchant.File))
		default:
			return fmt.Errorf("enchant data: %w", err)
		}
	}
	s.OnPlayerEnter(s.overEnchantEnter) // enchant_protect.go
	s.OnPlayerLeave(func(p *Player) { delete(f.players, p) })
	return errors.Join(
		s.Register(clientpackets.OpRequestEnchantItem, s.handleEnchantItem),
		s.Register(clientpackets.OpRequestCrystallizeItem, s.handleCrystallizeItem), // crystallize.go
	)
}

// UseEnchantData makes d the enchant data, with the EnchantBlackList of the
// config. Scrolls whose item is not an enchant scroll are left out with a
// warning, as Mobius does. Loop-only (or before Run).
func (s *Server) UseEnchantData(d enchant.Data) {
	t, skipped := enchant.NewTable(d, s.data, s.gs.Enchant.EnchantBlackList)
	if len(skipped) > 0 {
		s.log.Warn("game: enchant scrolls left out: their item is no enchant scroll", "items", skipped)
	}
	s.enchant.table = t
	s.log.Info("enchant data loaded", "scrolls", len(t.Scrolls()), "maxWeapon", t.MaxWeapon(), "maxArmor", t.MaxArmor(), "maxAccessory", t.MaxAccessory())
}

// enchantStateOf returns p's enchanting state, made on first use.
func (s *Server) enchantStateOf(p *Player) *enchantState {
	st := s.enchant.players[p]
	if st == nil {
		st = &enchantState{}
		s.enchant.players[p] = st
	}
	return st
}

// enchantableTemplate is ItemTemplate.isEnchantable: the template allows
// it and it is not in EnchantBlackList.
func (s *Server) enchantableTemplate(t *data.Item) bool {
	if s.enchant.table != nil {
		return s.enchant.table.Enchantable(t)
	}
	return t.Enchantable
}

// enchantTarget is it as the enchant rules see it.
func enchantTarget(p *Player, it *inventory.Item) enchant.Target {
	return enchant.Target{
		Template: it.Template(),
		Enchant:  int(it.Enchant()),
		Carried:  it.Container() == &p.inv.Container && (it.Loc() == storage.LocInventory || it.Equipped()),
	}
}

// useEnchantScroll is Mobius EnchantScrolls.onItemUse: the scroll becomes
// the active one, and the client opens the enchant window
// (ChooseInventoryItem) with "Select item to enchant.". A rider gets
// "Inappropriate enchant conditions." (mount.go); a cast in progress
// refuses quietly. Its "another enchantment" check never fires in
// Interlude (isEnchanting is never set).
func (s *Server) useEnchantScroll(p *Player, it *inventory.Item) bool {
	if s.mounted(p) {
		p.SendSystemMessage(sysMsgInappropriateEnchant)
		return false
	}
	if s.CastingNow(p) {
		return false
	}
	s.enchantStateOf(p).scroll = it.ObjectID()
	p.Send(serverpackets.NewChooseInventoryItemPacket(it.ItemID()))
	p.SendSystemMessage(sysMsgSelectItemToEnchant)
	return true
}

// cancelEnchant is the check of Mobius's warehouse requests: with an active
// scroll, "You have cancelled the enchanting process." and EnchantResult 0
// end it, and the request stops. It reports whether it cancelled.
func (s *Server) cancelEnchant(p *Player) bool {
	st := s.enchant.players[p]
	if st == nil || st.scroll == 0 {
		return false
	}
	st.scroll = 0
	p.SendSystemMessage(sysMsgEnchantCancelled)
	p.Send(serverpackets.NewEnchantResultPacket(0))
	return true
}

// handleEnchantItem handles RequestEnchantItem (0x58) in Mobius's order:
// the flood limit, the store and trade check, the item and the scroll, the
// scroll's checks and DisableOverEnchanting; then one scroll is used up,
// the roll decides, and the InventoryUpdate of everything that changed goes
// at once, followed by UserInfo and CharInfo.
func (s *Server) handleEnchantItem(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestEnchantItem(body)
	if err != nil {
		s.log.Debug("game: bad RequestEnchantItem", append(p.logArgs(), "err", err)...)
		return
	}
	st := s.enchantStateOf(p)
	if s.flood(&st.next, time.Duration(s.gs.Enchant.FloodEnchantItem)) || req.ObjectID == 0 {
		return
	}
	fail := func(msg serverpackets.SystemMessageID) {
		st.scroll = 0
		p.Send(serverpackets.NewEnchantResultPacket(0))
		p.SendSystemMessage(msg)
	}
	if s.ProcessingTransaction(p) || p.InStoreMode() { // trade.go, store.go
		fail(sysMsgNoEnchantInStore)
		return
	}
	it := p.inv.ByObjectID(req.ObjectID)
	var scroll *inventory.Item
	if st.scroll != 0 {
		scroll = p.inv.ByObjectID(st.scroll)
	}
	if it == nil || scroll == nil {
		fail(sysMsgEnchantCancelled)
		return
	}
	t := s.enchant.table
	info := t.Scroll(int(scroll.ItemID()))
	if info == nil {
		return // Mobius keeps the scroll active
	}
	target := enchantTarget(p, it)
	if !t.Valid(info, target) || (s.gs.Enchant.DisableOverEnchanting && target.Enchant == info.MaxEnchant) {
		fail(sysMsgInappropriateEnchant)
		return
	}
	if _, err := p.inv.Destroy(scroll, 1); err != nil {
		fail(sysMsgIncorrectItemCount2)
		s.illegalAction(p, "enchant with a scroll it does not have")
		return
	}
	iu := s.takeInventoryUpdates(p, nil) // the scroll
	switch t.Roll(info, target, s.enchant.rnd) {
	case enchant.ResultError:
		fail(sysMsgInappropriateEnchant)
	case enchant.ResultSuccess:
		iu = s.enchantSucceeded(p, it, info, target, iu)
	case enchant.ResultFailure:
		var ok bool
		if iu, ok = s.enchantFailed(p, it, info, iu); !ok {
			st.scroll = 0
			return
		}
	}
	s.flushInventoryUpdate(p)
	s.sendInventoryUpdateNow(p, iu)
	p.BroadcastUserInfo()
	st.scroll = 0
	s.refreshOverloaded(p)
	s.RequestSave(p)
}

// enchantSucceeded is the SUCCESS case of RequestEnchantItem: the message;
// the level rises when the rate group's own chance is above 0 (a worn item
// adds its new stats, and the ItemList follows, since the client keeps the
// old level of a worn item otherwise); EnchantResult with the level; +7 and
// +15 weapons and +6 armor are announced to the players around with a
// firework; armor reaching +4 while worn gets its +4 skill. It returns iu
// with the item's change.
func (s *Server) enchantSucceeded(p *Player, it *inventory.Item, info *enchant.ScrollInfo, target enchant.Target, iu []serverpackets.InventoryItem) []serverpackets.InventoryItem {
	name := s.itemNameParam(it.ItemID())
	if it.Enchant() == 0 {
		p.SendSystemMessage(sysMsgEnchanted, name)
	} else {
		p.SendSystemMessage(sysMsgEnchantedLevel, serverpackets.NumberParam(it.Enchant()), name)
	}
	if s.enchant.table.Gains(info, target) {
		it.SetEnchant(it.Enchant() + 1)
		iu = append(iu, s.takeInventoryUpdates(p, nil)...)
		if it.Equipped() {
			s.refreshWornItemStats(p, it)
			s.SendItemList(p, false)
		}
	}
	p.Send(serverpackets.NewEnchantResultPacket(it.Enchant()))
	level := int(it.Enchant())
	tmpl := it.Template()
	if enchant.Announced(tmpl, level) && p.InWorld() {
		text := p.Name() + " has successfully enchanted a +" + strconv.Itoa(level) + " " + tmpl.Name + "."
		s.world.Broadcast(p, serverpackets.NewSystemMessageTextPacket(text), false)
		s.world.Broadcast(p, s.fireworkPacket(p), true)
	}
	if tmpl.Armor != nil && level == 4 && it.Equipped() {
		if sk := s.itemSkillOf(tmpl, data.ItemSkillOnEnchant4); sk != nil { // item_skill_worn.go
			s.AddSkill(p, sk, false)
			p.SendSkillList()
		}
	}
	return iu
}

// enchantFailed is the FAILURE case of RequestEnchantItem. A safe scroll
// keeps the level. Otherwise a worn item comes off first, its own
// InventoryUpdate sent at once; a blessed scroll resets the item to +0; a
// normal scroll destroys it and gives the crystals of its grade (Mobius
// addItem: a delayed InventoryUpdate and the load, then "You have earned
// ..."). EnchantResult 0 ends each case. It returns iu with the item's
// changes, and false when the item could not be destroyed (the player is
// punished and nothing more is sent).
func (s *Server) enchantFailed(p *Player, it *inventory.Item, info *enchant.ScrollInfo, iu []serverpackets.InventoryItem) ([]serverpackets.InventoryItem, bool) {
	if info.Kind == enchant.KindSafe {
		p.Send(serverpackets.NewEnchantResultPacket(0))
		p.SendMessage(safeEnchantFailedText)
		return iu, true
	}
	name := s.itemNameParam(it.ItemID())
	if it.Equipped() {
		if it.Enchant() > 0 {
			p.SendSystemMessage(sysMsgEquipmentRemoved, serverpackets.NumberParam(it.Enchant()), name)
		} else {
			p.SendSystemMessage(sysMsgDisarmed, name)
		}
		p.inv.UnequipSlot(it.Slot())
		unequipped := s.takeInventoryUpdates(p, nil)
		s.flushInventoryUpdate(p)
		s.sendInventoryUpdateNow(p, unequipped)
	}
	if info.Kind == enchant.KindBlessed {
		p.SendSystemMessage(sysMsgBlessedEnchantFailed)
		it.SetEnchant(0)
		iu = append(iu, s.takeInventoryUpdates(p, nil)...)
		p.Send(serverpackets.NewEnchantResultPacket(0))
		return iu, true
	}
	level := it.Enchant()
	if level > 0 {
		p.SendSystemMessage(sysMsgEnchantCrystallizedLevel, serverpackets.NumberParam(level), name)
	} else {
		p.SendSystemMessage(sysMsgEnchantCrystallized, name)
	}
	crystalID, count := enchant.FailureCrystals(it.Template(), int(level))
	if _, err := p.inv.Destroy(it, it.Count()); err != nil {
		s.illegalAction(p, "unable to delete the item on an enchant failure")
		p.Send(serverpackets.NewEnchantResultPacket(0))
		return iu, false
	}
	s.forgetMana(it)
	iu = append(iu, s.takeInventoryUpdates(p, nil)...)
	if crystalID != 0 {
		iu = s.giveCrystals(p, int32(crystalID), count, iu) //nolint:gosec // crystal item IDs are small
	}
	p.Send(serverpackets.NewEnchantResultPacket(0))
	return iu, true
}

// giveCrystals adds count crystals of the item (Mobius
// PlayerInventory.addItem with the player: a delayed InventoryUpdate and
// StatusUpdate with the load), then says "You have earned $s2 $s1(s).". It
// returns iu with the crystals' change.
func (s *Server) giveCrystals(p *Player, crystalID int32, count int, iu []serverpackets.InventoryItem) []serverpackets.InventoryItem {
	if _, err := p.inv.Create(crystalID, int64(count)); err != nil {
		s.log.Error("game: crystals not given", append(p.logArgs(), "item", crystalID, "count", count, "err", err)...)
		return iu
	}
	added := s.takeInventoryUpdates(p, nil)
	s.queueInventoryUpdate(p, added...)
	p.Send(loadStatusUpdate(p))
	p.SendSystemMessage(sysMsgEarnedS2S1, s.itemNameParam(crystalID), serverpackets.NumberParam(int32(count))) //nolint:gosec // crystal counts are far below 2^31
	return append(iu, added...)
}

// refreshWornItemStats gives a worn item's stats again after its enchant
// level changed (Mobius FuncEnchant reads the level each time).
func (s *Server) refreshWornItemStats(p *Player, it *inventory.Item) {
	err := p.stats.Equip(stats.Slot(it.Slot()), stats.ItemOwner(it.ObjectID()), inventory.StatsItem(it.Template(), int(it.Enchant()))) //nolint:gosec // a worn item's slot is 0-17
	if err != nil {
		s.log.Warn("game: equipped item has no usable stats", append(p.logArgs(), "item", it.ItemID(), "err", err)...)
	}
}

// fireworkPacket is the MagicSkillUse of the firework on p (skill 2024,
// with its hit time and reuse delay).
func (s *Server) fireworkPacket(p *Player) []byte {
	hit, reuse := 0, fireworkReuseDelay
	if sk, ok := s.Skill(fireworkSkill, 1); ok {
		hit, reuse = sk.HitTime, sk.ReuseDelay
	}
	pos := p.Position()
	return serverpackets.NewMagicSkillUsePacket(serverpackets.MagicSkillUse{
		CasterID: p.ObjectID(), TargetID: p.ObjectID(),
		SkillID: fireworkSkill, SkillLevel: 1,
		HitTime:    time.Duration(hit) * time.Millisecond,
		ReuseDelay: time.Duration(reuse) * time.Millisecond,
		X:          pos.X, Y: pos.Y, Z: pos.Z,
		TargetX: pos.X, TargetY: pos.Y, TargetZ: pos.Z,
	})
}

// flushInventoryUpdate sends the InventoryUpdate waiting for its 100 ms at
// once (Mobius Player.flushInventoryUpdate).
func (s *Server) flushInventoryUpdate(p *Player) {
	st := &p.items
	if st.iuTimer != 0 {
		s.loop.Cancel(st.iuTimer)
		st.iuTimer = 0
	}
	items := st.iu
	st.iu = nil
	if len(items) > 0 && p.InGame() {
		p.Send(serverpackets.NewInventoryUpdatePacket(serverpackets.InventoryUpdate{Items: items}))
	}
}
