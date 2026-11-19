package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpSellList is the opcode of SellList.
const OpSellList = 0x10

// sellListItemSize is the size of one SellList entry.
const sellListItemSize = 32

// SellListItem is an item the player can sell to a merchant.
type SellListItem struct {
	// Item is the inventory item. Mobius sends the template ID as the item
	// ID (the display ID differs only for custom items) and leaves out
	// Equipped, AugmentationID and Mana.
	Item ItemInfo
	// ReferencePrice is the template's reference price (data.Item.Price).
	// The packet carries half of it, cut to an int: what the merchant pays.
	ReferencePrice int32
}

// SellList is the sell window of a merchant.
type SellList struct {
	Adena int32 // the adena the player carries
	// Items are the items to offer. Mobius lists the inventory items that
	// are not equipped, can be sold, and are not the control item of the
	// summoned pet.
	Items []SellListItem
}

// NewSellListPacket builds SellList (Mobius SellList.writeImpl).
func NewSellListPacket(l SellList) []byte {
	w := packets.NewWriter(11 + len(l.Items)*sellListItemSize)
	w.WriteUint8(OpSellList)
	w.WriteInt32(l.Adena)
	w.WriteInt32(0) // always 0 in Mobius
	putCount16(w, len(l.Items))
	for _, s := range l.Items {
		it := s.Item
		putInt16(w, it.Type1)
		w.WriteInt32(it.ObjectID)
		w.WriteInt32(it.ItemID)
		w.WriteInt32(it.Count)
		putInt16(w, it.Type2)
		putInt16(w, it.CustomType1)
		w.WriteInt32(it.BodyPart)
		putInt16(w, it.Enchant)
		putInt16(w, it.CustomType2)
		w.WriteUint16(0)
		w.WriteInt32(s.ReferencePrice / 2)
	}
	return w.Bytes()
}
