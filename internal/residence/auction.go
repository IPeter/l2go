package residence

import (
	"math"
	"slices"
	"time"
)

// Clan hall auctions (Mobius ClanHallAuction): the bids are paid from the
// clan warehouse, a lost or cancelled bid comes back less a 10 % fee, and
// the highest bid wins the hall when the auction ends.

// AuctionLength is how long an NPC auction runs again when it ends without a
// bid (startAutoTask: 7 days).
const AuctionLength = 7 * 24 * time.Hour

// Bidder is a clan's bid (Mobius Bidder, an auction_bid row): the clan, the
// name the bid was made under (the bidding player's on the first bid, the
// leader's on a raise, as Mobius stores them), the clan's name, the bid and
// when it was made (Unix milliseconds).
type Bidder struct {
	ClanID   int32
	Name     string
	ClanName string
	Bid      int64
	Time     int64
}

// Auction is a clan hall auction (Mobius ClanHallAuction). ID and the hall
// are the same. A seller ID of 0 is an NPC auction.
type Auction struct {
	ID             int32
	SellerID       int32
	SellerName     string
	SellerClanName string
	ItemName       string
	StartingBid    int64
	CurrentBid     int64
	EndDate        int64
	// Bidders are the bids in the order they were made.
	Bidders []Bidder
}

// Highest is Mobius loadBid's highest bidder: the highest bid, the first
// of equal bids; false without bids.
func (a *Auction) Highest() (Bidder, bool) {
	best := -1
	for i, b := range a.Bidders {
		if best < 0 || b.Bid > a.Bidders[best].Bid {
			best = i
		}
	}
	if best < 0 {
		return Bidder{}, false
	}
	return a.Bidders[best], true
}

// Bidder returns a clan's bid.
func (a *Auction) Bidder(clanID int32) (Bidder, bool) {
	i := slices.IndexFunc(a.Bidders, func(b Bidder) bool { return b.ClanID == clanID })
	if i < 0 {
		return Bidder{}, false
	}
	return a.Bidders[i], true
}

// MinimumBid is what the bid page asks at least (Auctioneer bid1): the
// highest bid, or the starting bid without one.
func (a *Auction) MinimumBid() int64 {
	if h, ok := a.Highest(); ok && h.Bid != 0 {
		return h.Bid
	}
	return a.StartingBid
}

// BidRequest is a bid of ClanHallAuction.setBid.
type BidRequest struct {
	ClanID     int32
	ClanName   string
	LeaderName string // the clan leader's name
	PlayerName string // the bidding player's name
	Bid        int64
	Warehouse  int64 // the clan warehouse's adena
	Now        int64
}

// BidResult is what setBid did.
type BidResult struct {
	// Msg is the system message for the bidder (MsgBidPlaced, MsgBidTooLow,
	// MsgNotEnoughHallAdena), 0 for none.
	Msg int
	// Take is the adena taken from the warehouse; Placed reports the bid was
	// made.
	Take   int64
	Placed bool
	// Outbid is the clan that lost the lead to a new bidder (its leader hears
	// "You have been out bidded" when online), 0 for none.
	Outbid int32
}

// PlaceBid is ClanHallAuction.setBid. A bid must beat the highest bid, or
// reach the starting bid without one. The clan that leads already pays only
// the difference (Mobius compares the leader's name); any other clan pays
// the whole bid, even one that had a lower bid (Mobius quirk: the earlier
// bid is overwritten, not refunded).
func (a *Auction) PlaceBid(r BidRequest) BidResult {
	high, has := a.Highest()
	required := r.Bid
	if has && high.ClanID == r.ClanID {
		required = r.Bid - high.Bid
	}
	valid := (has && r.Bid > high.Bid) || (!has && r.Bid >= a.StartingBid)
	if valid {
		if r.Warehouse < required {
			return BidResult{Msg: MsgNotEnoughHallAdena}
		}
		res := BidResult{Msg: MsgBidPlaced, Take: required, Placed: true}
		if i := slices.IndexFunc(a.Bidders, func(b Bidder) bool { return b.ClanID == r.ClanID }); i >= 0 {
			a.Bidders[i].Name, a.Bidders[i].Bid, a.Bidders[i].Time = r.LeaderName, r.Bid, r.Now
		} else {
			if has {
				res.Outbid = high.ClanID
			}
			a.Bidders = append(a.Bidders, Bidder{ClanID: r.ClanID, Name: r.PlayerName, ClanName: r.ClanName, Bid: r.Bid, Time: r.Now})
		}
		return res
	}
	if r.Bid < a.StartingBid || (has && r.Bid <= high.Bid) {
		return BidResult{Msg: MsgBidTooLow}
	}
	return BidResult{}
}

// RemoveBidder drops a clan's bid and returns it (cancelBid).
func (a *Auction) RemoveBidder(clanID int32) (Bidder, bool) {
	i := slices.IndexFunc(a.Bidders, func(b Bidder) bool { return b.ClanID == clanID })
	if i < 0 {
		return Bidder{}, false
	}
	b := a.Bidders[i]
	a.Bidders = slices.Delete(a.Bidders, i, i+1)
	return b, true
}

// Refund is ClanHallAuction.returnItem's amount: with the penalty 90 %
// (Java's int *= 0.9), capped so the warehouse stays at most MaxAdena.
func Refund(amount int64, penalty bool, warehouse int64) int64 {
	if penalty {
		amount = int64(float64(amount) * 0.9)
	}
	return max(min(amount, MaxAdena-warehouse), 0)
}

// AuctionEnd is what ClanHallAuction.endAuction does.
type AuctionEnd int

// The ends.
const (
	// AuctionRestart: an NPC auction without bids runs another week.
	AuctionRestart AuctionEnd = iota
	// AuctionDropped: a clan's sale without bids is dropped.
	AuctionDropped
	// AuctionSold: the highest bidder wins the hall.
	AuctionSold
)

// End is the branch of endAuction for the auction's bids.
func (a *Auction) End() AuctionEnd {
	if len(a.Bidders) > 0 {
		return AuctionSold
	}
	if a.SellerID == 0 {
		return AuctionRestart
	}
	return AuctionDropped
}

// AuctionWait is startAutoTask's delay: until the end, or 0 for an end in
// the past (which then moves AuctionLength ahead, startAutoTask).
func AuctionWait(now, end int64) (wait time.Duration, newEnd int64) {
	if end <= now {
		return 0, now + AuctionLength.Milliseconds()
	}
	return time.Duration(end-now) * time.Millisecond, end
}

// ClampBid is Auctioneer's Math.min(bid, MAX_ADENA) for a parsed bid.
func ClampBid(bid int64) int64 { return min(bid, math.MaxInt32) }
