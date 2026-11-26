package game

import (
	"fmt"
	"slices"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/enchant"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
)

// The over-enchant protection (Mobius OverEnchantProtection): equipment
// above the maximum level the enchant data allows (+16) is destroyed when a
// player who is no GM enters the world or puts it on, and the player is
// punished (OverEnchantPunishment); the GM enchant commands cap the level.

// overEnchantScreenTime is how long "You have over-enchanted items!" stays
// on the screen, in milliseconds.
const overEnchantScreenTime = 6000

// overEnchanted reports whether it is equipment above its maximum level
// for a player who is no GM, with the protection on.
func (s *Server) overEnchanted(p *Player, it *inventory.Item) bool {
	t := s.enchant.table
	return t != nil && s.gs.Enchant.OverEnchantProtection && !p.IsGM() &&
		it.Equipable() && t.OverEnchanted(it.Template(), int(it.Enchant()))
}

// overEnchantEnter is the OnPlayerEnter hook (Mobius EnterWorld): every
// over-enchanted item is destroyed, then the player is punished once. Not
// in Mobius: the InventoryUpdate of the destroyed items, so that the client
// does not keep showing them.
func (s *Server) overEnchantEnter(p *Player) {
	punish := false
	for _, it := range slices.Clone(p.inv.Items()) {
		if s.overEnchanted(p, it) {
			s.destroyOverEnchanted(p, it)
			punish = true
		}
	}
	if !punish {
		return
	}
	s.itemsChanged(p, true, nil)
	s.punishOverEnchant(p)
}

// overEnchantedOnUse is the over-enchant check of Mobius UseItem for an item
// being put on or taken off: the item is destroyed and the player punished.
// It reports whether it did that (the use then stops).
func (s *Server) overEnchantedOnUse(p *Player, it *inventory.Item) bool {
	if !s.overEnchanted(p, it) {
		return false
	}
	s.destroyOverEnchanted(p, it)
	s.itemsChanged(p, true, nil)
	p.BroadcastUserInfo()
	s.punishOverEnchant(p)
	return true
}

// destroyOverEnchanted destroys an over-enchanted item (worn ones come off
// first) and logs it.
func (s *Server) destroyOverEnchanted(p *Player, it *inventory.Item) {
	s.log.Info("over-enchanted item removed", append(p.logArgs(), "item", it.ItemID(), "itemObjectId", it.ObjectID(), "enchant", it.Enchant())...)
	if _, err := p.inv.Destroy(it, it.Count()); err != nil {
		s.log.Error("game: over-enchanted item not destroyed", append(p.logArgs(), "itemObjectId", it.ObjectID(), "err", err)...)
		return
	}
	s.forgetMana(it)
}

// punishOverEnchant tells the player and punishes it as
// OverEnchantPunishment says (Mobius PunishmentManager with
// IllegalPlayerActionTask). NONE does nothing; BROADCAST tells the GMs
// online 5 s later; KICK kicks 5 s later; KICKBAN and JAIL kick too, until
// there are bans and a jail.
func (s *Server) punishOverEnchant(p *Player) {
	kind := s.gs.Enchant.Punishment()
	if kind == config.PunishNone {
		return
	}
	p.SendMessage("[Server]: You have over-enchanted items!")
	p.SendMessage("[Server]: Respect our server rules.")
	p.Send(serverpackets.NewExShowScreenMessagePacket(serverpackets.ScreenText("You have over-enchanted items!", overEnchantScreenTime)))
	reason := p.Name() + " has over-enchanted items."
	if kind == config.PunishBroadcast {
		s.log.Warn("game: illegal action", append(p.logArgs(), "action", reason, "punishment", kind)...)
		s.loop.After(illegalActionKickDelay, func() {
			for gm := range s.Players() {
				if gm.IsGM() {
					gm.SendMessage(reason)
				}
			}
		})
		return
	}
	s.illegalAction(p, reason) // items.go: KICK
}

// capGMEnchant caps the level //seteh gives the item of a player who is no
// GM at its derived maximum, telling the GM (Mobius AdminEnchant.setEnchant
// with OverEnchantProtection).
func (s *Server) capGMEnchant(gm, target *Player, t *data.Item, level int32) int32 {
	tbl := s.enchant.table
	if tbl == nil || !s.gs.Enchant.OverEnchantProtection || target.IsGM() {
		return level
	}
	maxLevel := int32(tbl.MaxEnchant(t)) //nolint:gosec // small
	if level <= maxLevel {
		return level
	}
	kind := "armor"
	switch {
	case t.Weapon != nil:
		kind = "weapon"
	case t.Type2 == enchant.Type2Accessory:
		kind = "accessory"
	}
	gmSysMessage(gm, fmt.Sprintf("Maximum enchantment for %s items is %d.", kind, maxLevel))
	return maxLevel
}

// createItemEnchant is the enchant level //create_item gives a new item
// (Mobius AdminCreateItem.createItem): only an enchantable item gets one
// above 0, capped at its derived maximum with OverEnchantProtection. ok is
// false when no level is given.
func (s *Server) createItemEnchant(t *data.Item, level int32) (int32, bool) {
	if !s.enchantableTemplate(t) || level <= 0 {
		return 0, false
	}
	if tbl := s.enchant.table; tbl != nil && s.gs.Enchant.OverEnchantProtection {
		level = min(level, int32(tbl.MaxEnchant(t))) //nolint:gosec // small
	}
	return level, true
}
