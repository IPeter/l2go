package game

import (
	"math"
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/stats"
)

// adenaID is the item ID of adena (Mobius Inventory.ADENA_ID).
const adenaID = 57

// inventoryUpdateDelay is how long Mobius collects inventory changes before
// it sends them in one InventoryUpdate (Player.sendInventoryUpdate).
const inventoryUpdateDelay = 100 * time.Millisecond

// System messages of the item operations.
const (
	sysMsgS2S1Disappeared  serverpackets.SystemMessageID = 301 // "$s2 $s1 has disappeared."
	sysMsgS1Disappeared    serverpackets.SystemMessageID = 302 // "$s1 has disappeared."
	sysMsgIncorrectCount   serverpackets.SystemMessageID = 351 // "Incorrect item count."
	sysMsgAdenaDisappeared serverpackets.SystemMessageID = 672 // "$s1 adena disappeared."
)

// The minimal item operations of Milestone 4: taking a fee. Items change
// fully in Milestone 5.

// itemByItemID returns the first item of the template id (Mobius
// getItemByItemId), or nil.
func (inv *Inventory) itemByItemID(id int32) *Item {
	for _, it := range inv.items {
		if it.ItemID == id {
			return it
		}
	}
	return nil
}

// countByItemID returns how many of template id the inventory holds: the
// stack of a stackable item, or the number of items otherwise (Mobius
// getInventoryItemCount).
func (inv *Inventory) countByItemID(id int32) int64 {
	var n int64
	for _, it := range inv.items {
		if it.ItemID != id {
			continue
		}
		if it.Template != nil && it.Template.Stackable {
			return it.Count
		}
		n++
	}
	return n
}

// removeItem takes it out of the inventory, and off the paperdoll if worn.
// It reports the slot the item was worn in, or -1.
func (inv *Inventory) removeItem(it *Item) int {
	if i := slices.Index(inv.items, it); i >= 0 {
		inv.items = slices.Delete(inv.items, i, i+1)
	}
	for slot, worn := range inv.paperdoll {
		if worn == it {
			inv.paperdoll[slot] = nil
			return slot
		}
	}
	return -1
}

// reduceAdena takes count adena from p (Mobius Player.reduceAdena): with too
// little adena it fails, telling p when sendMessage is set; otherwise the
// stack shrinks or goes, an InventoryUpdate follows and p is told how much
// disappeared. Taking 0 adena succeeds and changes nothing. The character
// is saved in the background. Loop-only.
func (s *Server) reduceAdena(p *Player, count int64, sendMessage bool) bool {
	adena := p.inv.itemByItemID(adenaID)
	var have int64
	if adena != nil {
		have = adena.Count
	}
	if count > have {
		if sendMessage {
			p.SendSystemMessage(sysMsgNotEnoughAdena)
		}
		return false
	}
	if count <= 0 {
		return true
	}
	change := s.takeFromStack(p, adena, count)
	s.queueInventoryUpdate(p, change)
	if sendMessage {
		p.SendSystemMessage(sysMsgAdenaDisappeared, serverpackets.NumberParam(itemCount32(count)))
	}
	s.Save(p, nil)
	return true
}

// destroyItemByItemID takes count items of template itemID from p (Mobius
// Player.destroyItemByItemId; a negative count takes all): adena goes
// through reduceAdena; without the item, or with too few, it fails with
// "Incorrect item count." when sendMessage is set. On success p gets an
// InventoryUpdate, a StatusUpdate of its load and, with sendMessage, the
// "disappeared" message. The character is saved in the background.
// Loop-only.
func (s *Server) destroyItemByItemID(p *Player, itemID int32, count int64, sendMessage bool) bool {
	if itemID == adenaID {
		return s.reduceAdena(p, count, sendMessage)
	}
	it := p.inv.itemByItemID(itemID)
	have := p.inv.countByItemID(itemID)
	take := count
	if take < 0 {
		take = have
	}
	if it == nil || take <= 0 || have < take {
		if sendMessage {
			p.SendSystemMessage(sysMsgIncorrectCount)
		}
		return false
	}

	var changes []serverpackets.InventoryItem
	if it.Template != nil && it.Template.Stackable {
		changes = append(changes, s.takeFromStack(p, it, take))
	} else {
		// One item per count.
		for left := take; left > 0; left-- {
			next := p.inv.itemByItemID(itemID)
			if next == nil {
				break
			}
			changes = append(changes, s.takeFromStack(p, next, next.Count))
		}
	}
	s.queueInventoryUpdate(p, changes...)
	p.Send(serverpackets.NewStatusUpdatePacket(serverpackets.StatusUpdate{
		ObjectID: p.ObjectID(),
		Values:   []serverpackets.StatusValue{{Attr: serverpackets.StatusCurLoad, Value: p.inv.Weight()}},
	}))
	if sendMessage {
		name := s.itemNameParam(itemID)
		if take > 1 {
			p.SendSystemMessage(sysMsgS2S1Disappeared, name, serverpackets.NumberParam(itemCount32(take)))
		} else {
			p.SendSystemMessage(sysMsgS1Disappeared, name)
		}
	}
	s.Save(p, nil)
	return true
}

// takeFromStack takes count of the item it (count <= it.Count): the stack
// shrinks, or the item goes when nothing is left (and comes off the
// paperdoll, changing the stats). It returns the change for the
// InventoryUpdate.
func (s *Server) takeFromStack(p *Player, it *Item, count int64) serverpackets.InventoryItem {
	if it.Count > count {
		it.Count -= count
		return s.inventoryChange(it, serverpackets.ItemModified)
	}
	it.Count = 0
	if slot := p.inv.removeItem(it); slot >= 0 {
		p.stats.Unequip(stats.Slot(slot))
		p.BroadcastUserInfo()
	}
	return s.inventoryChange(it, serverpackets.ItemRemoved)
}

// inventoryChange is one entry of an InventoryUpdate.
func (s *Server) inventoryChange(it *Item, change serverpackets.ItemChange) serverpackets.InventoryItem {
	info := serverpackets.ItemInfo{ObjectID: it.ObjectID, ItemID: it.ItemID, Count: itemCount32(it.Count), Mana: -1}
	if it.Template != nil {
		info = ItemInfo(it)
		info.Equipped = info.Equipped && change != serverpackets.ItemRemoved
	}
	return serverpackets.InventoryItem{Change: change, Item: info}
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

// queueInventoryUpdate sends changes to p in an InventoryUpdate 100 ms after
// the last change, merged with the changes queued meanwhile (a later change
// of an item replaces an earlier one), as Mobius sendInventoryUpdate does.
func (s *Server) queueInventoryUpdate(p *Player, changes ...serverpackets.InventoryItem) {
	if len(changes) == 0 {
		return
	}
	st := s.dialogState(p)
	for _, c := range changes {
		i := slices.IndexFunc(st.invItems, func(q serverpackets.InventoryItem) bool { return q.Item.ObjectID == c.Item.ObjectID })
		if i >= 0 {
			st.invItems[i] = c
		} else {
			st.invItems = append(st.invItems, c)
		}
	}
	if st.invTimer != 0 {
		s.loop.Cancel(st.invTimer)
	}
	st.invTimer = s.loop.After(inventoryUpdateDelay, func() {
		st.invTimer = 0
		items := st.invItems
		st.invItems = nil
		if p.InGame() && len(items) > 0 {
			p.Send(serverpackets.NewInventoryUpdatePacket(serverpackets.InventoryUpdate{Items: items}))
		}
	})
}

// itemCount32 cuts a count to the int range, as the packets carry counts.
func itemCount32(v int64) int32 {
	return int32(min(max(v, math.MinInt32), math.MaxInt32)) //nolint:gosec // clamped to the int32 range
}
