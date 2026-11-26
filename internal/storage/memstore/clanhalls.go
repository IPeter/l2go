package memstore

import (
	"context"
	"maps"
	"slices"

	"github.com/IPeter/l2go/internal/storage"
)

var _ storage.ClanHallStore = (*Characters)(nil)

// LoadClanHalls implements storage.ClanHallStore.
func (s *Characters) LoadClanHalls(ctx context.Context) ([]storage.ClanHall, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.clanHalls) == 0 {
		return nil, nil
	}
	out := make([]storage.ClanHall, 0, len(s.clanHalls))
	for _, id := range slices.Sorted(maps.Keys(s.clanHalls)) {
		out = append(out, cloneClanHall(s.clanHalls[id]))
	}
	return out, nil
}

// LoadAuctions implements storage.ClanHallStore.
func (s *Characters) LoadAuctions(ctx context.Context) ([]storage.Auction, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.auctions) == 0 {
		return nil, nil
	}
	out := make([]storage.Auction, 0, len(s.auctions))
	for _, id := range slices.Sorted(maps.Keys(s.auctions)) {
		a := s.auctions[id]
		a.Bids = slices.Clone(a.Bids)
		out = append(out, a)
	}
	return out, nil
}

// cloneClanHall copies a hall's lists and map, so the store never shares them.
func cloneClanHall(h storage.ClanHall) storage.ClanHall {
	h.Functions = slices.Clone(h.Functions)
	h.Attackers = slices.Clone(h.Attackers)
	h.Vars = maps.Clone(h.Vars)
	return h
}

// applyClanHalls applies the clan hall part of a batch. The caller holds
// s.mu.
func (s *Characters) applyClanHalls(b *storage.Batch) {
	if len(b.ClanHalls) > 0 && s.clanHalls == nil {
		s.clanHalls = make(map[int32]storage.ClanHall)
	}
	for _, h := range b.ClanHalls {
		s.clanHalls[h.ID] = cloneClanHall(h)
	}
	for _, id := range b.DeletedAuctions {
		delete(s.auctions, id)
	}
	if len(b.Auctions) > 0 && s.auctions == nil {
		s.auctions = make(map[int32]storage.Auction)
	}
	for _, a := range b.Auctions {
		a.Bids = slices.Clone(a.Bids)
		s.auctions[a.ID] = a
	}
}
