package game

import (
	"slices"

	"github.com/IPeter/l2go/internal/storage"
)

// hallKey and auctionKey are the saver's keys of a clan hall and an auction
// document (persist.go).
func hallKey(id int32) saveKey    { return saveKey(int64(id) | 11<<40) }
func auctionKey(id int32) saveKey { return saveKey(int64(id) | 12<<40) }

// hallTouches adds the keys of the halls and auctions a batch writes.
func hallTouches(b *storage.Batch, add func(saveKey)) {
	for _, h := range b.ClanHalls {
		add(hallKey(h.ID))
	}
	for _, id := range b.DeletedAuctions {
		add(auctionKey(id))
	}
	for _, a := range b.Auctions {
		add(auctionKey(a.ID))
	}
}

// mergeClanHalls adds the halls and auctions of b, written after a, to a: a
// hall of b replaces the one of a with its ID; a deleted auction drops a
// stored one and a stored one cancels a delete.
func mergeClanHalls(a, b *storage.Batch) {
	for _, h := range b.ClanHalls {
		if i := slices.IndexFunc(a.ClanHalls, func(x storage.ClanHall) bool { return x.ID == h.ID }); i >= 0 {
			a.ClanHalls[i] = h
		} else {
			a.ClanHalls = append(a.ClanHalls, h)
		}
	}
	for _, id := range b.DeletedAuctions {
		a.Auctions = slices.DeleteFunc(a.Auctions, func(x storage.Auction) bool { return x.ID == id })
		if !slices.Contains(a.DeletedAuctions, id) {
			a.DeletedAuctions = append(a.DeletedAuctions, id)
		}
	}
	for _, au := range b.Auctions {
		a.DeletedAuctions = slices.DeleteFunc(a.DeletedAuctions, func(id int32) bool { return id == au.ID })
		if i := slices.IndexFunc(a.Auctions, func(x storage.Auction) bool { return x.ID == au.ID }); i >= 0 {
			a.Auctions[i] = au
		} else {
			a.Auctions = append(a.Auctions, au)
		}
	}
}
