package game

import (
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/residence"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The clan hall auctions on the loop (Mobius ClanHallAuction and
// ClanHallAuctionManager): a free hall is sold by an NPC auction that runs
// a week at a time; an owner may sell its hall. Bids come from the clan
// warehouse; when an auction ends the highest bidder gets the hall, the
// others get their bid back less 10 %, and a seller gets the price less
// 10 % and the lease. Every change is saved at once with the warehouses
// and clans it touched, in one transaction.

// hallAuction is a running auction with its end task.
type hallAuction struct {
	residence.Auction
	timer world.TimerID
}

func newHallAuction(st storage.Auction) *hallAuction {
	a := &hallAuction{Auction: residence.Auction{ID: st.ID, SellerID: st.SellerID, SellerName: st.SellerName,
		SellerClanName: st.SellerClanName, ItemName: st.ItemName, StartingBid: st.StartingBid, CurrentBid: st.CurrentBid, EndDate: st.EndDate}}
	for _, b := range st.Bids {
		a.Bidders = append(a.Bidders, residence.Bidder{ClanID: b.ClanID, Name: b.BidderName, ClanName: b.ClanName, Bid: b.MaxBid, Time: b.TimeBid})
	}
	return a
}

// document returns the auction's stored state.
func (a *hallAuction) document() storage.Auction {
	st := storage.Auction{ID: a.ID, SellerID: a.SellerID, SellerName: a.SellerName, SellerClanName: a.SellerClanName,
		ItemName: a.ItemName, StartingBid: a.StartingBid, CurrentBid: a.CurrentBid, EndDate: a.EndDate}
	for _, b := range a.Bidders {
		st.Bids = append(st.Bids, storage.AuctionBid{ClanID: b.ClanID, BidderName: b.Name, ClanName: b.ClanName, MaxBid: b.Bid, TimeBid: b.Time})
	}
	return st
}

// hallAuction returns the auction with the ID (ClanHallAuctionManager
// getAuction), or nil.
func (s *Server) hallAuction(id int32) *hallAuction {
	for _, a := range s.halls.auctions {
		if a.ID == id {
			return a
		}
	}
	return nil
}

// HallAuctions returns the running auctions in list order. Loop-only.
func (s *Server) HallAuctions() []*hallAuction { return s.halls.auctions }

// clanBidAuction is Clan.getAuctionBiddedAt: the auction the clan bid in,
// nil for none. Mobius stores it with the clan; here it is the auction that
// holds the clan's bid.
func (s *Server) clanBidAuction(cl *Clan) *hallAuction {
	if cl == nil {
		return nil
	}
	for _, a := range s.halls.auctions {
		if _, ok := a.Bidder(cl.ID); ok {
			return a
		}
	}
	return nil
}

// newNpcAuction is ClanHallAuctionManager.initNPC: the hall's NPC auction
// of the data (its end date is past, so it runs a week from now).
func (s *Server) newNpcAuction(h *ClanHall) *hallAuction {
	d, ok := s.halls.data.Auction(int(h.ID))
	if !ok {
		s.log.Warn("game: no NPC auction for the clan hall", "hall", h.ID)
		return nil
	}
	return &hallAuction{Auction: residence.Auction{ID: h.ID, SellerName: "NPC", SellerClanName: "NPC Clan", ItemName: d.ItemName,
		StartingBid: d.StartingBid, EndDate: d.EndDate}}
}

// startAuctionTask is ClanHallAuction.startAutoTask: a past end date moves a
// week ahead (it reports that it did, for the caller to save), and the end
// task is armed.
func (s *Server) startAuctionTask(a *hallAuction) bool {
	wait, end := residence.AuctionWait(s.Now().UnixMilli(), a.EndDate)
	changed := end != a.EndDate
	a.EndDate = end
	if a.timer != 0 {
		s.loop.Cancel(a.timer)
	}
	a.timer = s.loop.After(wait, func() {
		a.timer = 0
		s.endAuction(a)
	})
	return changed
}

// removeAuction takes an auction off the list and stops its task.
func (s *Server) removeAuction(a *hallAuction) {
	if a.timer != 0 {
		s.loop.Cancel(a.timer)
		a.timer = 0
	}
	s.halls.auctions = slices.DeleteFunc(s.halls.auctions, func(x *hallAuction) bool { return x == a })
}

// refundToClan is ClanHallAuction.returnItem: adena into a clan's warehouse,
// with the 10 % fee when penalty is set; the warehouse's changes join b.
func (s *Server) refundToClan(cl *Clan, amount int64, penalty bool, b *storage.Batch) {
	if cl == nil || amount <= 0 {
		return
	}
	wh := s.ClanWarehouse(cl)
	n := residence.Refund(amount, penalty, wh.Adena())
	if n <= 0 {
		return
	}
	if _, err := wh.Create(adenaID, n); err != nil {
		s.log.Error("game: auction refund failed", "clan", cl.Name, "adena", n, "err", err)
		return
	}
	s.addWarehouseChanges(cl, b)
}

// addWarehouseChanges moves a clan warehouse's pending item changes into b.
func (s *Server) addWarehouseChanges(cl *Clan, b *storage.Batch) {
	wh := s.ClanWarehouse(cl)
	ch := wh.Tracker().Take()
	wh.TakeUpdates()
	if noItemChanges(ch) {
		return
	}
	for i := range b.Items {
		if b.Items[i].Owner == cl.ID {
			b.Items[i] = ownerItemChanges(cl.ID, mergeItemChanges(itemChangesOf(b.Items[i]), ch))
			return
		}
	}
	b.Items = append(b.Items, ownerItemChanges(cl.ID, ch))
}

// endAuction is ClanHallAuction.endAuction: an NPC auction without bids
// runs another week; a clan's sale without bids is dropped (Mobius keeps
// its row; we delete it); else the seller is paid, the losers refunded and
// the highest bidder gets the hall, all in one transaction.
func (s *Server) endAuction(a *hallAuction) {
	h := s.ClanHallByID(a.ID)
	switch a.End() {
	case residence.AuctionRestart:
		s.startAuctionTask(a)
		s.SaveJoint(nil, storage.Batch{Auctions: []storage.Auction{a.document()}})
		return
	case residence.AuctionDropped:
		s.removeAuction(a)
		s.SaveJoint(nil, storage.Batch{DeletedAuctions: []int32{a.ID}})
		return
	}
	var b storage.Batch
	high, _ := a.Highest()
	if a.SellerID > 0 && h != nil {
		seller := s.ClanByName(a.SellerClanName)
		s.refundToClan(seller, high.Bid, true, &b)
		s.refundToClan(seller, int64(h.Data.Lease), false, &b)
	}
	s.removeAuction(a)
	b.DeletedAuctions = []int32{a.ID}
	winner := s.Clan(high.ClanID)
	for _, bd := range a.Bidders {
		if bd.ClanID == high.ClanID {
			continue
		}
		cl := s.Clan(bd.ClanID)
		if cl == nil {
			continue
		}
		if cl.HideoutID == 0 {
			s.refundToClan(cl, bd.Bid, true, &b)
		} else if pl, ok := s.PlayerByName(bd.Name); ok {
			pl.SendMessage("Congratulation you have won ClanHall!")
		}
	}
	if h == nil || winner == nil {
		s.SaveJoint(nil, b)
		return
	}
	for _, cl := range s.setHallOwnerState(h, winner) {
		b.Clans = append(b.Clans, cl.document())
	}
	b.ClanHalls = append(b.ClanHalls, h.document())
	s.SaveJoint(nil, b)
	s.afterHallOwner(h)
}

// setHallOwnerState is setHallOwner without the save: it returns the clans
// it changed.
func (s *Server) setHallOwnerState(h *ClanHall, cl *Clan) []*Clan {
	var changed []*Clan
	if h.ownerID != 0 {
		if old := s.freeHallState(h); old != nil {
			changed = append(changed, old)
		}
	}
	if old := s.ClanHallByOwner(cl); old != nil && old != h {
		s.freeHallState(old)
		old.save()
	}
	h.ownerID = cl.ID
	cl.HideoutID = h.ID
	changed = append(changed, cl)
	s.broadcastClanInfo(cl)
	if h.siege == nil {
		h.paidUntil = s.Now().UnixMilli()
	}
	return changed
}

// afterHallOwner starts an auctionable hall's lease after a new owner got
// it (initialyzeTask(true): due at once).
func (s *Server) afterHallOwner(h *ClanHall) {
	if h.siege == nil {
		s.scheduleLease(h, residence.LeaseFirstWait(s.Now().UnixMilli(), h.paidUntil, h.paid, true))
	}
}

// placeBid is ClanHallAuction.setBid for p's clan: the bid is taken from
// the clan warehouse and saved with the auction in one transaction; the
// clan that lost the lead hears of it.
func (s *Server) placeBid(p *Player, a *hallAuction, bid int64) {
	cl := p.Clan()
	if cl == nil {
		return
	}
	wh := s.ClanWarehouse(cl)
	res := a.PlaceBid(residence.BidRequest{ClanID: cl.ID, ClanName: cl.Name, LeaderName: cl.LeaderName(), PlayerName: p.Name(),
		Bid: bid, Warehouse: wh.Adena(), Now: s.Now().UnixMilli()})
	if res.Placed {
		var b storage.Batch
		if res.Take > 0 {
			if _, err := wh.DestroyByItemID(adenaID, res.Take); err != nil {
				s.log.Error("game: auction bid not taken", append(p.logArgs(), "err", err)...)
			}
			s.addWarehouseChanges(cl, &b)
		}
		b.Auctions = []storage.Auction{a.document()}
		s.SaveJoint(nil, b)
		if res.Outbid != 0 {
			if out := s.Clan(res.Outbid); out != nil {
				if pl, ok := s.PlayerByName(out.LeaderName()); ok {
					pl.SendMessage("You have been out bidded")
				}
			}
		}
	}
	if res.Msg != 0 {
		p.SendSystemMessage(serverpackets.SystemMessageID(res.Msg)) //nolint:gosec // message IDs
	}
}

// cancelBid is ClanHallAuction.cancelBid: the clan's bid comes back less
// 10 %, saved with the auction.
func (s *Server) cancelBid(a *hallAuction, cl *Clan) {
	bd, ok := a.RemoveBidder(cl.ID)
	if !ok {
		return
	}
	var b storage.Batch
	s.refundToClan(cl, bd.Bid, true, &b)
	b.Auctions = []storage.Auction{a.document()}
	s.SaveJoint(nil, b)
}

// cancelAuction is ClanHallAuction.cancelAuction: a seller withdraws the
// sale; every bidder gets its bid back less 10 % (or, owning a hall, hears
// that it won, as Mobius says).
func (s *Server) cancelAuction(a *hallAuction) {
	s.removeAuction(a)
	b := storage.Batch{DeletedAuctions: []int32{a.ID}}
	for _, bd := range a.Bidders {
		cl := s.Clan(bd.ClanID)
		if cl == nil {
			continue
		}
		if cl.HideoutID == 0 {
			s.refundToClan(cl, bd.Bid, true, &b)
		} else if pl, ok := s.PlayerByName(bd.Name); ok {
			pl.SendMessage("Congratulation you have won ClanHall!")
		}
	}
	s.SaveJoint(nil, b)
}

// confirmAuction is ClanHallAuction.confirmAuction: a prepared sale starts.
// Deviation: its end task starts at once (Mobius arms it only after a
// restart).
func (s *Server) confirmAuction(a *residence.Auction) {
	ha := &hallAuction{Auction: *a}
	if old := s.hallAuction(a.ID); old != nil {
		s.removeAuction(old)
	}
	s.halls.auctions = append(s.halls.auctions, ha)
	s.startAuctionTask(ha)
	s.SaveJoint(nil, storage.Batch{Auctions: []storage.Auction{ha.document()}})
}

// auctionRemaining is Auctioneer's "%d hours %d minutes" until an end.
func auctionRemaining(now time.Time, end int64) (hours, minutes int64) {
	left := end - now.UnixMilli()
	return left / 3600000, (left / 60000) % 60
}
