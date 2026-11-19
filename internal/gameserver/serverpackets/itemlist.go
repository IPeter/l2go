package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpItemList is the opcode of ItemList.
const OpItemList = 0x1b

// itemInfoSize is the size of one item block.
const itemInfoSize = 36

// ItemInfo is one item as the inventory packets send it (Mobius
// AbstractItemPacket.writeItem).
type ItemInfo struct {
	// Type1 is the template's type 1: 0 weapon or jewel, 1 armor or
	// shield, 4 other items.
	Type1    int16
	ObjectID int32
	ItemID   int32 // the display ID of the item template
	Count    int32
	// Type2 is the template's type 2: 0 weapon, 1 armor or shield, 2 jewel,
	// 3 quest item, 4 adena, 5 other.
	Type2       int16
	CustomType1 int16 // usually 0
	Equipped    bool
	// BodyPart is the body part mask of the template (Mobius
	// BodyPart.getMask), 0 for items that are not worn.
	BodyPart       int32
	Enchant        int16 // the enchant level; a pet's level for its collar
	CustomType2    int16 // usually 0; for a pet collar, whether the pet has a name
	AugmentationID int32 // 0 if not augmented
	Mana           int32 // the minutes left for a shadow item, -1 for others
}

// putItemInfo writes one item block (Mobius AbstractItemPacket.writeItem).
func putItemInfo(w *packets.Writer, it ItemInfo) {
	putInt16(w, it.Type1)
	w.WriteInt32(it.ObjectID)
	w.WriteInt32(it.ItemID)
	w.WriteInt32(it.Count)
	putInt16(w, it.Type2)
	putInt16(w, it.CustomType1)
	putBool16(w, it.Equipped)
	w.WriteInt32(it.BodyPart)
	putInt16(w, it.Enchant)
	putInt16(w, it.CustomType2)
	w.WriteInt32(it.AugmentationID)
	w.WriteInt32(it.Mana)
}

// ItemList is the player's whole inventory.
type ItemList struct {
	// ShowWindow opens the inventory window. EnterWorld sends the list
	// without it.
	ShowWindow bool
	Items      []ItemInfo
}

// NewItemListPacket builds ItemList (Mobius ItemList.writeImpl). An empty
// list is valid.
func NewItemListPacket(l ItemList) []byte {
	w := packets.NewWriter(5 + len(l.Items)*itemInfoSize)
	w.WriteUint8(OpItemList)
	putBool16(w, l.ShowWindow)
	putCount16(w, len(l.Items))
	for _, it := range l.Items {
		putItemInfo(w, it)
	}
	return w.Bytes()
}
