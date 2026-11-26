package storage

import "context"

// Clan hall state (Plan 14 M11j, decision X4): one document per hall in the
// clanhalls collection (what Mobius keeps in clanhall, siegable_clanhall,
// clanhall_functions, clanhall_siege_attackers and the hall scripts' tables
// siegable_hall_flagwar_attackers(_members) and rainbowsprings_attacker_list),
// and one per auction in the auctions collection (Mobius auction and
// auction_bid). The owner is the clan whose Clan.HideoutID names the hall
// (Mobius clanhall.ownerId). Both are written with a Batch, in the same
// transaction as the clans and clan warehouse items a transition changes
// (a sale, a lease, a siege's end). Times are Unix milliseconds, 0 for none.
type ClanHall struct {
	ID int32
	// PaidUntil and Paid are an auctionable hall's lease (clanhall.paidUntil
	// and paid).
	PaidUntil int64
	Paid      bool
	Functions []ClanHallFunction
	// NextSiege is a contestable hall's next siege; Attackers the clans
	// registered for it (clanhall_siege_attackers).
	NextSiege int64
	Attackers []int32
	// Vars are the values the hall's siege script keeps (its own tables in
	// Mobius), nil for none.
	Vars map[string]string
}

// ClanHallFunction is a clanhall_functions row: a function the owner rents,
// its level, the fee taken every Rate milliseconds and when it is due next.
type ClanHallFunction struct {
	Type    int32
	Level   int32
	Lease   int32
	Rate    int64
	EndTime int64
}

// Auction is a clan hall auction (an auction row with its auction_bid rows):
// the hall (ID and itemId are the hall's ID), the seller (0 and "NPC" for an
// NPC auction), the bids and the end.
type Auction struct {
	ID             int32
	SellerID       int32
	SellerName     string
	SellerClanName string
	ItemName       string
	StartingBid    int64
	CurrentBid     int64
	EndDate        int64
	Bids           []AuctionBid
}

// AuctionBid is an auction_bid row: the bidding clan, its leader's name, its
// name, the bid and when it was made.
type AuctionBid struct {
	ClanID     int32
	BidderName string
	ClanName   string
	MaxBid     int64
	TimeBid    int64
}

// ClanHallStore reads the clan halls and auctions. Both stores implement it
// on their character store, next to BatchWriter, which writes them
// (Batch.ClanHalls, Batch.Auctions and Batch.DeletedAuctions).
type ClanHallStore interface {
	// LoadClanHalls returns the stored halls sorted by ID, nil for none.
	LoadClanHalls(ctx context.Context) ([]ClanHall, error)
	// LoadAuctions returns the stored auctions sorted by ID, nil for none.
	LoadAuctions(ctx context.Context) ([]Auction, error)
}
