package game

import (
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/world"
)

// The shop feature (shop*.go, merchant_price.go): NPC merchants (Plan 08
// Phase 15). The bypasses "Buy <listId>", "Sell" and "BuyShadowItem" open
// the buy window, the sell window and the shadow item pages;
// RequestBuyItem and RequestSellItem trade with the merchant (shop_trade.go);
// limited products run out and restock (shop_stock.go, decision I8).
func init() { registerFeature("shop", setupShop) }

// siegeGuardsPriceRate is Mobius's RateSiegeGuardsPrice (Rates.ini, 1): it
// multiplies the price of the mercenary posting tickets, items 3960-4026.
const siegeGuardsPriceRate float32 = 1

// The mercenary posting tickets, whose price siegeGuardsPriceRate scales.
const (
	mercenaryTicketFirst = 3960
	mercenaryTicketLast  = 4026
)

// System messages of the shops (Mobius SystemMessageId).
const (
	shopMsgWeightLimit         serverpackets.SystemMessageID = 422  // "You have exceeded the weight limit."
	shopMsgTransactionComplete serverpackets.SystemMessageID = 880  // "The transaction is complete."
	shopMsgQuantityExceeded    serverpackets.SystemMessageID = 1036 // "You have exceeded the quantity that can be inputted."
)

// The shadow item pages of BuyShadowItem, by level (Mobius BuyShadowItem).
const (
	shadowItemLowLevel = "common/shadow_item-lowlevel.htm" // below 40
	shadowItemD        = "common/shadow_item_d.htm"        // 40-45
	shadowItemC        = "common/shadow_item_c.htm"        // 46-51
	shadowItemB        = "common/shadow_item_b.htm"        // 52 and up
)

// shopFeature is the shop feature's state on the Server (Server.shop).
// Loop-only.
type shopFeature struct {
	// priceZones holds, per town zone of a price config, the zone regions it
	// touches (merchant_price.go), worked out on first use.
	priceZones map[int]map[zoneRegion]bool
	// stock holds the limited products sold since their last restock
	// (shop_stock.go); a product that is not there is full.
	stock map[stockKey]*productStock
}

// setupShop registers the shop packets.
func setupShop(s *Server) error {
	s.shop.stock = make(map[stockKey]*productStock)
	return errors.Join(
		s.Register(clientpackets.OpRequestBuyItem, s.handleBuyItem),
		s.Register(clientpackets.OpRequestSellItem, s.handleSellItem),
	)
}

// shopCommands are the bypass commands of the shops (lower case), Mobius's
// bypass handlers Buy, Sell and BuyShadowItem.
var shopCommands = map[string]bool{"buy": true, "sell": true, "buyshadowitem": true}

// shopBypass runs a shop bypass command for n; word is the command's first
// word in lower case. Like Mobius's handlers, it does nothing for an NPC
// that is not a merchant.
func (s *Server) shopBypass(p *Player, n *Npc, word, command string) {
	if n == nil || !s.npcIs(n, "Merchant") {
		return
	}
	switch word {
	case "buy":
		// Mobius Buy: a StringTokenizer on spaces; a missing or bad list
		// number does nothing.
		fields := strings.Fields(command)
		if len(fields) < 2 {
			return
		}
		listID, err := strconv.Atoi(fields[1])
		if err != nil {
			s.log.Warn("game: bad Buy bypass", append(p.logArgs(), "bypass", command)...)
			return
		}
		s.showBuyWindow(p, n, listID)
	case "sell":
		s.showSellWindow(p)
	case "buyshadowitem":
		s.showShadowItemPage(p, n)
	}
}

// showBuyWindow opens buylist listID of merchant n for p (Mobius
// Merchant.showBuyWindow): the list must exist and name the merchant. The
// window shows p's adena and the prices with the merchant's tax; the
// inventory list requests of the next 1.5 s are ignored; ActionFailed
// follows.
func (s *Server) showBuyWindow(p *Player, n *Npc, listID int) {
	l := s.buyList(listID)
	if l == nil {
		s.log.Warn("game: buylist not found", "list", listID, "npc", n.Template.ID)
		p.SendActionFailed()
		return
	}
	if !l.AllowsNpc(n.Template.ID) {
		s.log.Warn("game: NPC not allowed in buylist", "list", listID, "npc", n.Template.ID)
		p.SendActionFailed()
		return
	}
	s.BlockInventory(p)
	p.Send(serverpackets.NewBuyListPacket(s.buyListPacket(p, l, merchantTaxRate(n))))
	p.SendActionFailed()
}

// buyList returns the buylist with the given ID, or nil.
func (s *Server) buyList(id int) *data.BuyList {
	if s.economy == nil {
		return nil
	}
	l, _ := s.economy.BuyList(id)
	return l
}

// buyListPacket builds the BuyList of l for p. A limited product that is
// sold out is left out, so that the count the packet announces is the
// number of entries it holds (Mobius announces every product).
func (s *Server) buyListPacket(p *Player, l *data.BuyList, taxRate float64) serverpackets.BuyList {
	out := serverpackets.BuyList{
		ListID:               int32(l.ID), //nolint:gosec // list IDs fit in int32
		Adena:                itemCount32(p.inv.Adena()),
		TaxRate:              taxRate,
		SiegeGuardsPriceRate: siegeGuardsPriceRate,
		Products:             make([]serverpackets.BuyListProduct, 0, len(l.Products)),
	}
	for i := range l.Products {
		pr := &l.Products[i]
		t, ok := s.data.ItemTemplate(pr.ItemID)
		if !ok {
			continue
		}
		var count int64
		if pr.LimitedStock() {
			if count = s.stockLeft(l.ID, pr); count <= 0 {
				continue
			}
		}
		//nolint:gosec // item IDs, list types, masks, prices and stocks fit in their fields
		out.Products = append(out.Products, serverpackets.BuyListProduct{
			ItemID:       int32(pr.ItemID),
			Type1:        int16(t.Type1),
			Type2:        int16(t.Type2),
			BodyPart:     int32(t.BodyPartMask),
			Price:        int32(pr.Price),
			LimitedStock: pr.LimitedStock(),
			Count:        int32(count),
		})
	}
	return out
}

// productPrice is the price of one product before tax (Mobius
// RequestBuyItem): the list's price, the mercenary posting tickets scaled by
// siegeGuardsPriceRate with Java's int-times-float rounding.
func productPrice(pr *data.BuyListProduct) int64 {
	price := int64(pr.Price)
	if pr.ItemID >= mercenaryTicketFirst && pr.ItemID <= mercenaryTicketLast {
		price = int64(float32(price) * siegeGuardsPriceRate)
	}
	return price
}

// taxedPrice is the price of one product with the tax, cut to an int before
// it is multiplied by the count (Mobius RequestBuyItem).
func taxedPrice(price int64, taxRate float64) int64 {
	return int64(float64(price) * (1 + taxRate))
}

// showSellWindow opens the sell window for p (Mobius Sell): the items it
// carries unworn that merchants buy, at half their reference price; then
// ActionFailed.
func (s *Server) showSellWindow(p *Player) {
	p.Send(serverpackets.NewSellListPacket(s.sellListPacket(p)))
	p.SendActionFailed()
}

// sellListPacket builds the SellList of p (Mobius SellList): the unworn
// sellable items in inventory order, with their template IDs. Pets do not
// exist yet, so no control item is left out.
func (s *Server) sellListPacket(p *Player) serverpackets.SellList {
	out := serverpackets.SellList{Adena: itemCount32(p.inv.Adena())}
	for _, it := range p.inv.Items() {
		t := it.Template()
		if it.Equipped() || !t.Sellable || augmentLocked(it) {
			continue
		}
		info := ItemInfo(it)
		info.ItemID = it.ItemID()
		out.Items = append(out.Items, serverpackets.SellListItem{
			Item:           info,
			ReferencePrice: int32(min(t.Price, math.MaxInt32)), //nolint:gosec // clamped
		})
	}
	return out
}

// showShadowItemPage shows the shadow item page for p's level (Mobius
// BuyShadowItem), bound to merchant n.
func (s *Server) showShadowItemPage(p *Player, n *Npc) {
	var page string
	switch level := p.Level(); {
	case level < 40:
		page = shadowItemLowLevel
	case level < 46:
		page = shadowItemD
	case level < 52:
		page = shadowItemC
	default:
		page = shadowItemB
	}
	s.sendPage(p, s.pageFile(p, n.ID(), page).replaceObjectID(n.ID()))
}

// shopMerchant returns the merchant p trades with: obj when it is a
// merchant (Mobius instanceof Merchant) in p's instance within interaction
// range (3D), else nil.
func (s *Server) shopMerchant(p *Player, obj world.Object) *Npc {
	n, ok := obj.(*Npc)
	if !ok || n == nil || !n.InWorld() || !s.npcIs(n, "Merchant") {
		return nil
	}
	if p.InstanceID() != n.InstanceID() || !p.Position().InRange3D(n.Position(), s.npcInteractionRange()) {
		return nil
	}
	return n
}
