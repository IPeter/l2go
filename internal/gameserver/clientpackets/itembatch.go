package clientpackets

import "github.com/IPeter/l2go/internal/packets"

// MaxItemsInPacket is the largest number of entries in the item list of a
// shop or warehouse packet: Mobius PlayerConfig.MAX_ITEM_IN_PACKET, the
// largest inventory size of the default Player.ini (80, 100 for dwarves, 250
// for GMs).
const MaxItemsInPacket = 250

// itemBatchFits reports whether Mobius reads an item list of n entries of
// size bytes each: n is 1 to MaxItemsInPacket and the entries fill the rest
// of the data exactly (the BATCH_LENGTH check of the shop and warehouse
// packets). Mobius drops a list that fails it.
func itemBatchFits(r *packets.Reader, n int32, size int) bool {
	return n > 0 && n <= MaxItemsInPacket && int(n)*size == r.Len()
}
