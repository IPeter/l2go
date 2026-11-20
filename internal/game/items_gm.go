package game

import (
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/storage"
)

// The GM item commands (decision I15, Mobius AdminCreateItem) and the item
// methods scripts call on players (scripting.PlayerItems).

var _ scripting.PlayerItems = (*Player)(nil)

// registerItemGMCommands registers //create_item, //give_item_target and
// //delete_item.
func (s *Server) registerItemGMCommands() error {
	admin := storage.AccessLevelAdmin
	return errors.Join(
		s.RegisterGMCommand("create_item", admin, s.gmCreateItem),
		s.RegisterGMCommand("give_item_target", admin, s.gmGiveItemTarget),
		s.RegisterGMCommand("delete_item", admin, s.gmDeleteItem),
	)
}

// gmInt parses a number of a GM command as Java Integer.parseInt.
func gmInt(s string) (int32, bool) {
	n, err := strconv.ParseInt(s, 10, 32)
	return int32(n), err == nil
}

// gmCreateItem is //create_item <id> [count] [enchant]: the items go into
// the GM's inventory.
func (s *Server) gmCreateItem(p *Player, args []string) {
	if len(args) == 0 {
		return
	}
	id, ok := gmInt(args[0])
	count := int32(1)
	if ok && len(args) >= 2 {
		count, ok = gmInt(args[1])
	}
	enchant := int32(-1)
	if ok && len(args) >= 3 {
		enchant, ok = gmInt(args[2])
	}
	if !ok {
		p.SendMessage("Specify a valid number.")
		return
	}
	s.gmGiveItem(p, p, id, count, enchant, len(args) >= 3)
}

// gmGiveItemTarget is //give_item_target <id> [count]: the items go into
// the inventory of the GM's target, which must be a player.
func (s *Server) gmGiveItemTarget(p *Player, args []string) {
	target, ok := p.Target.(*Player)
	if !ok {
		p.SendMessage("Invalid target.")
		return
	}
	if len(args) == 0 || len(args) > 2 {
		return
	}
	id, ok := gmInt(args[0])
	count := int32(1)
	if ok && len(args) == 2 {
		count, ok = gmInt(args[1])
	}
	if !ok {
		p.SendMessage("Specify a valid number.")
		return
	}
	s.gmGiveItem(p, target, id, count, -1, false)
}

// gmGiveItem is AdminCreateItem.createItem: more than 10 of an item that
// does not stack is refused; the target is told and gets its ItemList.
func (s *Server) gmGiveItem(gm, target *Player, id, count, enchant int32, withEnchant bool) {
	t, ok := s.data.ItemTemplate(int(id))
	if !ok {
		gm.SendMessage("This item doesn't exist.")
		return
	}
	if count > 10 && !t.Stackable {
		gm.SendMessage("This item does not stack - Creation aborted.")
		return
	}
	if count <= 0 {
		gm.SendMessage("Specify a valid number.")
		return
	}
	it, err := target.inv.Create(id, int64(count))
	if err != nil {
		s.log.Warn("game: //create_item failed", append(gm.logArgs(), "item", id, "count", count, "err", err)...)
		gm.SendMessage("The item could not be created.")
		return
	}
	if withEnchant {
		if t.Enchantable && enchant > 0 {
			it.SetEnchant(enchant)
		} else {
			gm.SendMessage("This item is not enchantable, no enchant applied.")
		}
	}
	s.itemsChanged(target, true, nil)
	if gm != target {
		target.SendMessage(fmt.Sprintf("Admin spawned %d %s in your inventory.", count, t.Name))
	}
	s.SendItemList(target, false)
	gm.SendMessage(fmt.Sprintf("You have spawned %d %s(%d) in %s inventory.", count, t.Name, id, target.Name()))
}

// gmDeleteItem is //delete_item <objectId> [count]: count of the item (all
// of it without a count) is destroyed from the online player who owns it.
func (s *Server) gmDeleteItem(p *Player, args []string) {
	if len(args) == 0 || len(args) > 2 {
		return
	}
	objectID, ok := gmInt(args[0])
	count := int32(0)
	if ok && len(args) == 2 {
		count, ok = gmInt(args[1])
	}
	if !ok {
		p.SendMessage("Specify a valid number.")
		return
	}
	for owner := range s.Players() {
		it := owner.inv.ByObjectID(objectID)
		if it == nil {
			continue
		}
		n := int64(count)
		if n <= 0 || n > it.Count() {
			n = it.Count()
		}
		if _, err := owner.inv.Destroy(it, n); err == nil {
			s.forgetMana(it)
			s.itemsChanged(owner, true, nil)
			owner.RefreshStats()
		}
		p.SendMessage("Item deleted.")
		return
	}
	p.SendMessage("Item doesn't have owner.")
}

// Adena returns how much adena the player carries (scripting.PlayerItems).
// Loop-only.
func (p *Player) Adena() int64 { return p.inv.Adena() }

// ItemCount returns how many items of the template the player carries,
// worn ones included: the stack of a stackable item, else the number of
// items (Mobius getQuestItemsCount) (scripting.PlayerItems). Loop-only.
func (p *Player) ItemCount(itemID int) int64 {
	if itemID <= 0 || itemID > math.MaxInt32 {
		return 0
	}
	return p.inv.Count(int32(itemID), -1, true)
}

// GiveItems gives the player count items of the template with the quest
// reward messages (Mobius Quest.giveItems): adena "You have earned $s1
// adena.", others "You have earned $s2 $s1(s)." or "You have earned
// $s1.". It reports whether the items were given (scripting.PlayerItems).
// Loop-only.
func (p *Player) GiveItems(itemID int, count int64) bool {
	s := p.srv
	if count <= 0 || itemID <= 0 || itemID > math.MaxInt32 || p.state != stateInGame {
		return false
	}
	id := int32(itemID)
	if _, ok := s.data.ItemTemplate(itemID); !ok {
		return false
	}
	it, err := p.inv.Create(id, count)
	if err != nil {
		s.log.Warn("game: cannot give items", append(p.logArgs(), "item", itemID, "count", count, "err", err)...)
		return false
	}
	s.itemsChanged(p, true, nil)
	name := s.itemNameParam(it.ItemID())
	switch {
	case id == adenaID:
		p.SendSystemMessage(sysMsgEarnedS1Adena, numberParam(count))
	case count > 1:
		p.SendSystemMessage(sysMsgEarnedS2S1, name, numberParam(count))
	default:
		p.SendSystemMessage(sysMsgEarnedS1, name)
	}
	return true
}

// TakeItems takes count items of the template from the player, all of them
// for a negative count (Mobius Quest.takeItems): worn items come off
// first, then each goes with "... has disappeared.". It reports whether
// the player had count of them; with fewer it takes what there is, as
// Mobius does (scripting.PlayerItems). Loop-only.
func (p *Player) TakeItems(itemID int, count int64) bool {
	s := p.srv
	if itemID <= 0 || itemID > math.MaxInt32 || count == 0 || p.state != stateInGame {
		return false
	}
	id := int32(itemID)
	enough := count < 0 || p.inv.Count(id, -1, true) >= count
	var taken int64
	for _, it := range p.inv.AllByItemID(id) {
		take := it.Count()
		if count >= 0 {
			take = min(take, count-taken)
		}
		if take <= 0 {
			break
		}
		if it.Equipped() {
			p.inv.UnequipBodyPart(it.Template().Part())
			s.sendInventoryUpdateNow(p, s.takeInventoryUpdates(p, nil))
			p.BroadcastUserInfo()
		}
		if s.DestroyItemByItemID(p, id, take, true) {
			taken += take
		}
	}
	return enough
}

// numberParam is a count as a system message number.
func numberParam(n int64) serverpackets.SystemMessageParam {
	return serverpackets.NumberParam(itemCount32(n))
}
