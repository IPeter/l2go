package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpMultiSellList is the opcode of MultiSellList.
const OpMultiSellList = 0xd0

// MultiSellPageSize is the number of entries on one MultiSellList page
// (Mobius MultisellData.PAGE_SIZE).
const MultiSellPageSize = 40

// Sizes of the MultiSellList parts.
const (
	multiSellHeaderSize     = 21
	multiSellEntrySize      = 17
	multiSellProductSize    = 22
	multiSellIngredientSize = 18
)

// multiSellNoType2 is the type 2 Mobius sends for an item without a
// template.
const multiSellNoType2 = 0xffff

// MultiSellItem is a product or an ingredient of a multisell entry (Mobius
// multisell.Ingredient).
type MultiSellItem struct {
	// ItemID is the display ID of the item template, or the ID of an item
	// without a template (an ID below 1 for a special ingredient, or an
	// unknown one). The packet carries it as a 16-bit short: larger IDs are
	// cut to their low 16 bits, as in Mobius.
	ItemID int32
	// NoTemplate marks an item without a template: Mobius sends type 2 65535
	// for it, and body part 0 for a product.
	NoTemplate bool
	BodyPart   int32 // the template's body part mask; sent for products only
	Type2      int16 // the template's type 2, as in ItemInfo
	Count      int32
	// Enchant is the enchant level: that of the player's item the entry
	// keeps (Mobius Ingredient.getItemInfo), else the level the list gives.
	Enchant int16
	// AugmentationID is the augmentation of the player's item the entry
	// keeps, 0 otherwise.
	AugmentationID int32
}

// MultiSellEntry is one exchange of a multisell list: the player gives the
// ingredients for the products.
type MultiSellEntry struct {
	// EntryID is the ID MultiSellChoose sends back (Mobius PreparedEntry:
	// the entry's ID times 100,000, plus the enchant level it keeps).
	EntryID     int32
	Products    []MultiSellItem
	Ingredients []MultiSellItem
}

// MultiSellList is a multisell list as the player sees it, with every entry.
type MultiSellList struct {
	ListID  int32
	Entries []MultiSellEntry
}

// NewMultiSellListPacket builds the page of l that starts at entry index
// (Mobius MultiSellList.writeImpl): at most MultiSellPageSize entries, the
// page number counted from 1, and the finished flag set on the page with the
// last entry. An index outside the list is taken as its nearest end; Mobius
// is never given one.
func NewMultiSellListPacket(l MultiSellList, index int) []byte {
	index = min(max(index, 0), len(l.Entries))
	page := l.Entries[index:]
	finished := len(page) <= MultiSellPageSize
	if !finished {
		page = page[:MultiSellPageSize]
	}

	size := multiSellHeaderSize
	for _, e := range page {
		size += multiSellEntrySize + len(e.Products)*multiSellProductSize + len(e.Ingredients)*multiSellIngredientSize
	}
	w := packets.NewWriter(size)
	w.WriteUint8(OpMultiSellList)
	w.WriteInt32(l.ListID)
	putCount32(w, 1+index/MultiSellPageSize)
	putBool32(w, finished)
	w.WriteInt32(MultiSellPageSize)
	putCount32(w, len(page))
	for _, e := range page {
		putMultiSellEntry(w, e)
	}
	return w.Bytes()
}

// NewMultiSellListPages builds every page of l in order, as Mobius
// MultisellData.separateAndSend sends them: one page per MultiSellPageSize
// entries, and a single empty page for an empty list.
func NewMultiSellListPages(l MultiSellList) [][]byte {
	pages := make([][]byte, 0, max(1, (len(l.Entries)+MultiSellPageSize-1)/MultiSellPageSize))
	index := 0
	for {
		pages = append(pages, NewMultiSellListPacket(l, index))
		index += MultiSellPageSize
		if index >= len(l.Entries) {
			return pages
		}
	}
}

// putMultiSellEntry writes one entry of MultiSellList.
func putMultiSellEntry(w *packets.Writer, e MultiSellEntry) {
	w.WriteInt32(e.EntryID)
	w.WriteInt32(0)
	w.WriteInt32(0)
	w.WriteUint8(1)
	putCount16(w, len(e.Products))
	putCount16(w, len(e.Ingredients))
	for _, p := range e.Products {
		putMultiSellItemID(w, p.ItemID)
		if p.NoTemplate {
			w.WriteInt32(0)
			w.WriteUint16(multiSellNoType2)
		} else {
			w.WriteInt32(p.BodyPart)
			putInt16(w, p.Type2)
		}
		putMultiSellItemTail(w, p)
	}
	for _, i := range e.Ingredients {
		putMultiSellItemID(w, i.ItemID)
		if i.NoTemplate {
			w.WriteUint16(multiSellNoType2)
		} else {
			putInt16(w, i.Type2)
		}
		putMultiSellItemTail(w, i)
	}
}

// putMultiSellItemID writes an item ID as the short Mobius writes.
func putMultiSellItemID(w *packets.Writer, id int32) {
	putInt16(w, int16(id)) //nolint:gosec // cut to 16 bits, as in Mobius
}

// putMultiSellItemTail writes the fields after the type 2: the count, the
// enchant level, the augmentation and the mana (always 0).
func putMultiSellItemTail(w *packets.Writer, it MultiSellItem) {
	w.WriteInt32(it.Count)
	putInt16(w, it.Enchant)
	w.WriteInt32(it.AugmentationID)
	w.WriteInt32(0)
}
