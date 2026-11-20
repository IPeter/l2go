package game

import (
	"math"
	"math/rand/v2"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/world"
)

// The item operations other features call (Mobius Player.addAdena,
// reduceAdena, addItem, destroyItem, destroyItemByItemId). Each sends what
// Mobius sends and asks for a save at the end of the tick. All are
// loop-only.

// adenaID is the item ID of adena (Mobius Inventory.ADENA_ID), untyped.
const adenaID = 57

// System messages of the item operations.
const (
	sysMsgS2S1Disappeared  serverpackets.SystemMessageID = 301 // "$s2 $s1 has disappeared."
	sysMsgS1Disappeared    serverpackets.SystemMessageID = 302 // "$s1 has disappeared."
	sysMsgIncorrectCount   serverpackets.SystemMessageID = 351 // "Incorrect item count."
	sysMsgAdenaDisappeared serverpackets.SystemMessageID = 672 // "$s1 adena disappeared."
)

// AddAdena gives p count adena (Mobius Player.addAdena): the count is cut
// to what fits under MaxAdena; with sendMessage p is told what it earned
// (or that it earned nothing) and when it reached the limit. A new adena
// stack sends ItemList, a grown one an InventoryUpdate. It returns how
// much was added.
func (s *Server) AddAdena(p *Player, count int64, sendMessage bool) int64 {
	asked := count
	count = min(count, s.items.cfg.MaxAdena-p.inv.Adena())
	if sendMessage {
		if count <= 0 && asked > 0 {
			p.SendSystemMessage(sysMsgFailedEarnAdena, serverpackets.NumberParam(itemCount32(asked)))
		} else {
			p.SendSystemMessage(sysMsgEarnedS1Adena, serverpackets.NumberParam(itemCount32(count)))
		}
	}
	if count > 0 {
		if _, err := p.inv.Create(adenaID, count); err != nil {
			s.log.Error("game: cannot add adena", append(p.logArgs(), "count", count, "err", err)...)
			return 0
		}
		// PlayerInventory.addItem sends the InventoryUpdate and the load;
		// a new stack also gets the whole list.
		s.itemsChanged(p, true, nil)
		if count == p.inv.Adena() {
			s.SendItemList(p, false)
		}
	}
	if sendMessage && p.inv.Adena() == s.items.cfg.MaxAdena {
		p.SendSystemMessage(sysMsgAdenaLimit)
	}
	return max(count, 0)
}

// ReduceAdena takes count adena from p (Mobius Player.reduceAdena): with
// too little adena it fails, telling p when sendMessage is set; otherwise
// the stack shrinks or goes, an InventoryUpdate follows and p is told how
// much disappeared. Taking 0 adena succeeds and changes nothing.
func (s *Server) ReduceAdena(p *Player, count int64, sendMessage bool) bool {
	if count > p.inv.Adena() {
		if sendMessage {
			p.SendSystemMessage(sysMsgNotEnoughAdena)
		}
		return false
	}
	if count <= 0 {
		return true
	}
	if _, err := p.inv.DestroyByItemID(adenaID, count); err != nil {
		return false
	}
	s.itemsChanged(p, false, nil)
	if sendMessage {
		p.SendSystemMessage(sysMsgAdenaDisappeared, serverpackets.NumberParam(itemCount32(count)))
	}
	return true
}

// AddItem gives p count items of the template (Mobius Player.addItem(process,
// itemId, count, enchantLevel, reference, sendMessage)): with sendMessage p
// is told what it obtained ("earned" with earned set, as for quest
// rewards); the items stack or come as new items, enchant (-1 for the
// template's) is set on the item returned, and InventoryUpdate and
// StatusUpdate with the load follow. A full inventory drops the new item
// at p's feet when AutoLootSlotLimit is on, as Mobius does. Items that take
// effect at once (herbs) need the skill engine and are not added. Adena
// goes through the same path; use AddAdena for its limit messages.
func (s *Server) AddItem(p *Player, itemID int32, count int64, enchant int32, sendMessage, earned bool) (*inventory.Item, error) {
	if count <= 0 {
		return nil, inventory.ErrInvalidCount
	}
	t, ok := s.data.ItemTemplate(int(itemID))
	if !ok {
		return nil, inventory.ErrUnknownItem
	}
	if t.ExImmediateEffect && t.Etc != nil {
		return nil, nil
	}
	if sendMessage {
		name := s.itemNameParam(itemID)
		switch {
		case count > 1 && earned:
			p.SendSystemMessage(sysMsgEarnedS2S1, name, serverpackets.NumberParam(itemCount32(count)))
		case count > 1:
			p.SendSystemMessage(sysMsgObtainedS2S1, name, serverpackets.NumberParam(itemCount32(count)))
		case earned:
			p.SendSystemMessage(sysMsgEarnedS1, name)
		default:
			p.SendSystemMessage(sysMsgObtainedS1, name)
		}
	}
	it, err := p.inv.Create(itemID, count)
	if err != nil {
		return nil, err
	}
	if enchant > -1 {
		it.SetEnchant(enchant)
	}
	s.itemsChanged(p, true, nil)
	if !p.IsGM() && !p.inv.ValidateCapacity(0) && t.Droppable && (!it.Stackable() || it.LastChange() != inventory.Modified) {
		s.dropAtFeet(p, it)
	}
	return it, nil
}

// dropAtFeet drops a whole item of p's inventory next to p (Mobius
// Player.dropItem(process, item, reference, sendMessage)).
func (s *Server) dropAtFeet(p *Player, it *inventory.Item) {
	dropped, err := p.inv.DropItem(it)
	if err != nil {
		return
	}
	pos := p.Position()
	at := world.Position{X: pos.X + rand.Int32N(50) - 25, Y: pos.Y + rand.Int32N(50) - 25, Z: pos.Z + 20} //nolint:gosec // game randomness
	if _, err := s.DropOnGround(dropped, at, GroundDrop{Dropper: p, PlayerDrop: true}); err != nil {
		s.log.Error("game: a dropped item cannot go into the world", append(p.logArgs(), "itemObjectId", dropped.ObjectID(), "err", err)...)
	}
	s.itemsChanged(p, true, nil)
	p.SendSystemMessage(sysMsgDroppedS1, s.itemNameParam(dropped.ItemID()))
}

// sysMsgDroppedS1 is "You have dropped $s1.".
const sysMsgDroppedS1 serverpackets.SystemMessageID = 298

// DestroyItem destroys count of the item it of p's inventory (Mobius
// Player.destroyItem): the InventoryUpdate, ItemList when the last of a
// worn item went, StatusUpdate with the load and, with sendMessage, "...
// has disappeared.". Too many fails with "Incorrect item count.".
func (s *Server) DestroyItem(p *Player, it *inventory.Item, count int64, sendMessage bool) bool {
	wasEquipped := it != nil && it.Equipped()
	if _, err := p.inv.Destroy(it, count); err != nil {
		if sendMessage {
			p.SendSystemMessage(sysMsgIncorrectCount)
		}
		return false
	}
	s.forgetMana(it)
	s.itemsChanged(p, true, nil)
	if wasEquipped && it.Count() == 0 {
		s.SendItemList(p, false)
		p.BroadcastUserInfo()
	}
	if sendMessage {
		s.sendDisappeared(p, it.ItemID(), count)
	}
	return true
}

// DestroyItemByItemID takes count items of template itemID from p (Mobius
// Player.destroyItemByItemId; a negative count takes all): adena goes
// through ReduceAdena; without the item, or with too few, it fails with
// "Incorrect item count." when sendMessage is set. On success p gets an
// InventoryUpdate, a StatusUpdate of its load and, with sendMessage, the
// "disappeared" message.
func (s *Server) DestroyItemByItemID(p *Player, itemID int32, count int64, sendMessage bool) bool {
	if itemID == adenaID {
		return s.ReduceAdena(p, count, sendMessage)
	}
	it := p.inv.ByItemID(itemID)
	if it == nil {
		if sendMessage {
			p.SendSystemMessage(sysMsgIncorrectCount)
		}
		return false
	}
	have := it.Count()
	if !it.Stackable() {
		have = p.inv.Count(itemID, -1, true)
	}
	take := count
	if take < 0 {
		take = have
	}
	if take <= 0 || have < take {
		if sendMessage {
			p.SendSystemMessage(sysMsgIncorrectCount)
		}
		return false
	}
	last, err := p.inv.DestroyByItemID(itemID, take)
	if err != nil {
		if sendMessage {
			p.SendSystemMessage(sysMsgIncorrectCount)
		}
		return false
	}
	s.forgetMana(last)
	s.itemsChanged(p, true, nil)
	p.RefreshStats()
	if sendMessage {
		s.sendDisappeared(p, itemID, take)
	}
	return true
}

// sendDisappeared tells p that count items of the template disappeared.
func (s *Server) sendDisappeared(p *Player, itemID int32, count int64) {
	name := s.itemNameParam(itemID)
	if count > 1 {
		p.SendSystemMessage(sysMsgS2S1Disappeared, name, serverpackets.NumberParam(itemCount32(count)))
	} else {
		p.SendSystemMessage(sysMsgS1Disappeared, name)
	}
}

// itemNameParam is the item name parameter of a system message (Mobius
// SystemMessage.addItemName(int)): the client's name of the item, or the
// server's name as text when the client shows another item.
func (s *Server) itemNameParam(id int32) serverpackets.SystemMessageParam {
	if t, ok := s.data.ItemTemplate(int(id)); ok && t.DisplayID != int(id) {
		return serverpackets.TextParam(t.Name)
	}
	return serverpackets.ItemNameParam(id)
}

// itemCount32 cuts a count to the int range, as the packets carry counts.
func itemCount32(v int64) int32 {
	return int32(min(max(v, math.MinInt32), math.MaxInt32)) //nolint:gosec // clamped to the int32 range
}
