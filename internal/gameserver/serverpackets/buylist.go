package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpBuyList is the opcode of BuyList.
const OpBuyList = 0x11

// The mercenary posting tickets, items 3960 to 4026: Mobius multiplies their
// price by RatesConfig.RATE_SIEGE_GUARDS_PRICE.
const (
	mercenaryTicketFirst = 3960
	mercenaryTicketLast  = 4026
)

// type1Other is the template type 1 of the items that are not worn (Mobius
// ItemTemplate.TYPE1_ITEM_QUESTITEM_ADENA). BuyList sends no body part for
// them.
const type1Other = 4

// buyListProductSize is the size of one BuyList entry.
const buyListProductSize = 32

// BuyListProduct is one product of a merchant's buylist.
type BuyListProduct struct {
	ItemID   int32 // the item template ID (Mobius Product.getItemId)
	Type1    int16 // the template's type 1, as in ItemInfo
	Type2    int16 // the template's type 2, as in ItemInfo
	BodyPart int32 // the template's body part mask; not sent when Type1 is 4
	// Price is the price before tax (Mobius Product.getPrice: the buylist
	// price, or the template's reference price when the buylist has none).
	Price int32
	// LimitedStock marks a product with a limited stock, of which Count are
	// left. A limited product with none left is not listed. Count is not sent
	// for other products (Mobius sends 0).
	LimitedStock bool
	Count        int32
}

// BuyList is the buy window of a merchant.
type BuyList struct {
	ListID int32
	Adena  int32 // the adena the player carries
	// TaxRate is the merchant's tax rate (Mobius
	// MerchantPriceConfig.getTotalTaxRate), 0 for none. The prices shown
	// include it.
	TaxRate float64
	// SiegeGuardsPriceRate multiplies the price of the mercenary posting
	// tickets, items 3960 to 4026: Mobius RatesConfig.RATE_SIEGE_GUARDS_PRICE
	// (RateSiegeGuardsPrice), a Java float that is 1 by default. At 0 the
	// tickets show as free.
	SiegeGuardsPriceRate float32
	Products             []BuyListProduct
}

// NewBuyListPacket builds BuyList (Mobius BuyList.writeImpl). As in Mobius,
// the product count counts every product while a limited product with none
// left is not written, so the client is told of more entries than follow.
// Leaving such products out of Products keeps the two equal.
func NewBuyListPacket(l BuyList) []byte {
	w := packets.NewWriter(11 + len(l.Products)*buyListProductSize)
	w.WriteUint8(OpBuyList)
	w.WriteInt32(l.Adena)
	w.WriteInt32(l.ListID)
	putCount16(w, len(l.Products))
	for _, p := range l.Products {
		if p.LimitedStock && p.Count <= 0 {
			continue
		}
		var count, bodyPart int32
		if p.LimitedStock {
			count = p.Count
		}
		if p.Type1 != type1Other {
			bodyPart = p.BodyPart
		}
		putInt16(w, p.Type1)
		w.WriteInt32(0) // object ID
		w.WriteInt32(p.ItemID)
		w.WriteInt32(count)
		putInt16(w, p.Type2)
		w.WriteUint16(0) // equipped
		w.WriteInt32(bodyPart)
		w.WriteUint16(0) // enchant level
		w.WriteUint16(0)
		w.WriteUint16(0)
		w.WriteInt32(buyListPrice(p, l.TaxRate, l.SiegeGuardsPriceRate))
	}
	return w.Bytes()
}

// buyListPrice is the price BuyList shows for p, with the tax, cut to an
// int like the Java cast. For a mercenary ticket Mobius first multiplies the
// price by the siege guards rate in float precision (int times float).
func buyListPrice(p BuyListProduct, taxRate float64, siegeGuardsRate float32) int32 {
	if p.ItemID >= mercenaryTicketFirst && p.ItemID <= mercenaryTicketLast {
		// The conversion rounds the product to float precision explicitly.
		ticket := float32(float32(p.Price) * siegeGuardsRate)
		return truncToInt32(float64(ticket) * (1 + taxRate))
	}
	return truncToInt32(float64(p.Price) * (1 + taxRate))
}
