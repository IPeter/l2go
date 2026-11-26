package game

import (
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/residence"
	"github.com/IPeter/l2go/internal/social"
)

// The auctioneer (Mobius Auctioneer): the list of running auctions, an
// auction's page and bidders, bidding and cancelling a bid, and an owner's
// sale of its hall. Its pages are Mobius's data/html/auction/. The town's
// castle must exist; during its siege the auctioneer is busy.

const auctionHTML = "auction/"

// Auction date formats (Auctioneer.DATE_FORMAT and DATE_FORMAT_SHORT).
const (
	auctionDateFormat      = "02/01/2006 15:04"
	auctionDateFormatShort = "06/01/02"
)

// auctionPageLimit is the auctions per list page.
const auctionPageLimit = 15

// The conditions of Auctioneer.validateCondition.
const (
	auctionCondAllFalse = iota
	auctionCondBusy
	auctionCondRegular
)

// auctioneerCondition is Auctioneer.validateCondition: the town's castle
// must exist; during its siege the auctioneer is busy.
func (s *Server) auctioneerCondition(n *Npc) int {
	c := s.npcCastle(n)
	switch {
	case c == nil || c.ID <= 0:
		return auctionCondAllFalse
	case c.InProgress():
		return auctionCondBusy
	}
	return auctionCondRegular
}

// showAuctioneerPage is Auctioneer.showChatWindow.
func (s *Server) showAuctioneerPage(p *Player, n *Npc) {
	name := "auction.htm"
	if s.auctioneerCondition(n) == auctionCondBusy {
		name = "auction-busy.htm"
	}
	s.sendPage(p, s.pageFile(p, n.ID(), auctionHTML+name).replaceObjectID(n.ID()).
		replace("%npcId%", strconv.Itoa(n.Template.ID)).replace("%npcname%", n.Template.Name))
}

// auctionDate formats a Unix-milliseconds date in the server's zone.
func (s *Server) auctionDate(ms int64, layout string) string {
	return s.calendar.In(msTime(ms)).Format(layout)
}

// auctionHallInfo fills the hall placeholders of an auction page.
func (s *Server) auctionHallInfo(pg *htmlPage, a *hallAuction) *htmlPage {
	h := s.ClanHallByID(a.ID)
	if h == nil {
		return pg
	}
	hours, minutes := auctionRemaining(s.Now(), a.EndDate)
	return pg.replace("%AGIT_NAME%", a.ItemName).replace("%OWNER_PLEDGE_NAME%", a.SellerClanName).
		replace("%OWNER_PLEDGE_MASTER%", a.SellerName).replace("%AGIT_SIZE%", strconv.Itoa(h.Grade()*10)).
		replace("%AGIT_LEASE%", strconv.Itoa(h.Lease())).replace("%AGIT_LOCATION%", h.Data.Location).
		replace("%AGIT_AUCTION_END%", s.auctionDate(a.EndDate, auctionDateFormat)).
		replace("%AGIT_AUCTION_REMAIN%", strconv.FormatInt(hours, 10)+" hours "+strconv.FormatInt(minutes, 10)+" minutes").
		replace("%AGIT_AUCTION_MINBID%", strconv.FormatInt(a.StartingBid, 10)).replace("%AGIT_AUCTION_DESC%", h.Data.Desc)
}

// auctioneerCommand is Auctioneer.onBypassFeedback; it reports false for
// the commands of Npc (Chat, Quest, ...).
func (s *Server) auctioneerCommand(p *Player, n *Npc, command string) bool {
	switch s.auctioneerCondition(n) {
	case auctionCondAllFalse:
		p.SendMessage("Wrong conditions.")
		return true
	case auctionCondBusy:
		s.sendPage(p, s.pageFile(p, n.ID(), auctionHTML+"auction-busy.htm").replaceObjectID(n.ID()))
		return true
	}
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return false
	}
	val := ""
	if len(fields) > 1 {
		val = fields[1]
	}
	rest := fields[min(2, len(fields)):]
	cl := p.Clan()
	obj := strconv.Itoa(int(n.ID()))
	page := func(name string) *htmlPage { return s.pageFile(p, n.ID(), auctionHTML+name) }
	notAuthorized := func() { s.sendPage(p, page("not_authorized.htm").replaceObjectID(n.ID())) }
	switch strings.ToLower(fields[0]) {
	case "auction":
		s.auctionSale(p, n, val, rest)
	case "confirmauction":
		if cl == nil {
			return true
		}
		if a := s.halls.pending[cl.HideoutID]; a != nil {
			s.confirmAuction(a)
			delete(s.halls.pending, cl.HideoutID)
		} else {
			p.SendMessage("Invalid auction")
		}
	case "bidding":
		id, err := strconv.Atoi(val)
		if err != nil {
			if val != "" {
				p.SendMessage("Invalid auction!")
			}
			return true
		}
		pg := page("AgitAuctionInfo.htm")
		if a := s.hallAuction(int32(id)); a != nil { //nolint:gosec // hall IDs
			s.auctionHallInfo(pg, a).replace("%AGIT_AUCTION_COUNT%", strconv.Itoa(len(a.Bidders))).
				replace("%AGIT_LINK_BACK%", "bypass -h npc_"+obj+"_list").
				replace("%AGIT_LINK_BIDLIST%", "bypass -h npc_"+obj+"_bidlist "+idText(a.ID)).
				replace("%AGIT_LINK_RE%", "bypass -h npc_"+obj+"_bid1 "+idText(a.ID))
		} else {
			s.log.Warn("game: auctioneer: no auction", "auction", id)
		}
		s.sendPage(p, pg)
	case "bid":
		id, err := strconv.Atoi(val)
		if err != nil {
			if val != "" {
				p.SendMessage("Invalid auction!")
			}
			return true
		}
		var bid int64
		if len(rest) > 0 {
			if bid, err = strconv.ParseInt(rest[0], 10, 64); err != nil {
				p.SendMessage("Invalid bid!")
				return true
			}
		}
		a := s.hallAuction(int32(id)) //nolint:gosec // hall IDs
		if a == nil || cl == nil {
			p.SendMessage("Invalid bid!")
			return true
		}
		s.placeBid(p, a, residence.ClampBid(bid))
	case "bid1":
		if cl == nil || cl.Level < 2 {
			p.SendSystemMessage(serverpackets.SystemMessageID(residence.MsgAuctionNeedsLevel2))
			return true
		}
		if val == "" {
			return true
		}
		id, err := strconv.Atoi(val)
		if err != nil {
			p.SendMessage("Invalid auction!")
			return true
		}
		if b := s.clanBidAuction(cl); (b != nil && b.ID != int32(id)) || cl.HideoutID > 0 { //nolint:gosec // hall IDs
			p.SendSystemMessage(serverpackets.SystemMessageID(residence.MsgAlreadyBid))
			return true
		}
		a := s.hallAuction(int32(id)) //nolint:gosec // hall IDs
		if a == nil {
			p.SendMessage("Invalid auction!")
			return true
		}
		minimum := a.StartingBid
		if h, ok := a.Highest(); ok && h.Bid != 0 {
			minimum = h.Bid
		}
		s.sendPage(p, page("AgitBid1.htm").replace("%AGIT_LINK_BACK%", "bypass -h npc_"+obj+"_bidding "+val).
			replace("%PLEDGE_ADENA%", strconv.FormatInt(s.ClanWarehouse(cl).Adena(), 10)).
			replace("%AGIT_AUCTION_MINBID%", strconv.FormatInt(minimum, 10)).
			replace("npc_%objectId%_bid", "npc_"+obj+"_bid "+val))
	case "list":
		s.auctionList(p, n, val)
	case "bidlist":
		var a *hallAuction
		if val == "" {
			a = s.clanBidAuction(cl)
		} else if id, err := strconv.Atoi(val); err == nil {
			a = s.hallAuction(int32(id)) //nolint:gosec // hall IDs
		}
		if a == nil {
			return true
		}
		var b strings.Builder
		for _, bd := range a.Bidders {
			t := s.calendar.In(msTime(bd.Time))
			b.WriteString("<tr><td>" + bd.ClanName + "</td><td>" + bd.Name + "</td><td>" + strconv.Itoa(t.Year()) + "/" +
				strconv.Itoa(int(t.Month())) + "/" + strconv.Itoa(t.Day()) + "</td><td>" + strconv.FormatInt(bd.Bid, 10) + "</td></tr>")
		}
		s.sendPage(p, page("AgitBidderList.htm").replace("%AGIT_LIST%", b.String()).
			replace("%AGIT_LINK_BACK%", "bypass -h npc_"+obj+"_selectedItems").replace("%x%", val).replaceObjectID(n.ID()))
	case "selecteditems":
		s.auctionSelectedItems(p, n)
	case "cancelbid":
		a := s.clanBidAuction(cl)
		if a == nil {
			return true
		}
		bd, _ := a.Bidder(cl.ID)
		s.sendPage(p, page("AgitBidCancel.htm").replace("%AGIT_BID%", strconv.FormatInt(bd.Bid, 10)).
			replace("%AGIT_BID_REMAIN%", strconv.FormatInt(int64(float64(bd.Bid)*0.9), 10)).
			replace("%AGIT_LINK_BACK%", "bypass -h npc_"+obj+"_selectedItems").replaceObjectID(n.ID()))
	case "docancelbid":
		if a := s.clanBidAuction(cl); a != nil {
			s.cancelBid(a, cl)
			p.SendSystemMessage(serverpackets.SystemMessageID(residence.MsgBidCanceled))
		}
	case "cancelauction":
		if !p.HasClanAccess(social.AccessHallAuction) {
			notAuthorized()
			return true
		}
		h := s.ClanHallByOwner(cl)
		if h == nil {
			return true
		}
		s.sendPage(p, page("AgitSaleCancel.htm").replace("%AGIT_DEPOSIT%", strconv.Itoa(h.Lease())).
			replace("%AGIT_LINK_BACK%", "bypass -h npc_"+obj+"_selectedItems").replaceObjectID(n.ID()))
	case "docancelauction":
		if cl == nil {
			return true
		}
		if a := s.hallAuction(cl.HideoutID); a != nil {
			s.cancelAuction(a)
			p.SendMessage("Your auction has been canceled")
		}
	case "sale2":
		h := s.ClanHallByOwner(cl)
		if h == nil {
			return true
		}
		s.sendPage(p, page("AgitSale2.htm").replace("%AGIT_LAST_PRICE%", strconv.Itoa(h.Lease())).
			replace("%AGIT_LINK_BACK%", "bypass -h npc_"+obj+"_sale").replaceObjectID(n.ID()))
	case "sale":
		if !p.HasClanAccess(social.AccessHallAuction) {
			notAuthorized()
			return true
		}
		h := s.ClanHallByOwner(cl)
		if h == nil {
			return true
		}
		s.sendPage(p, page("AgitSale1.htm").replace("%AGIT_DEPOSIT%", strconv.Itoa(h.Lease())).
			replace("%AGIT_PLEDGE_ADENA%", strconv.FormatInt(s.ClanWarehouse(cl).Adena(), 10)).
			replace("%AGIT_LINK_BACK%", "bypass -h npc_"+obj+"_selectedItems").replaceObjectID(n.ID()))
	case "rebid":
		if !p.HasClanAccess(social.AccessHallAuction) {
			notAuthorized()
			return true
		}
		pg := page("AgitBid2.htm")
		if a := s.clanBidAuction(cl); a != nil {
			bd, _ := a.Bidder(cl.ID)
			pg.replace("%AGIT_AUCTION_BID%", strconv.FormatInt(bd.Bid, 10)).
				replace("%AGIT_AUCTION_MINBID%", strconv.FormatInt(a.StartingBid, 10)).
				replace("%AGIT_AUCTION_END%", s.auctionDate(a.EndDate, auctionDateFormat)).
				replace("%AGIT_LINK_BACK%", "bypass -h npc_"+obj+"_selectedItems").
				replace("npc_%objectId%_bid1", "npc_"+obj+"_bid1 "+idText(a.ID))
		}
		s.sendPage(p, pg)
	case "location":
		pos := p.Position()
		town := ""
		if s.worldData != nil {
			town = s.worldData.ClosestTownName(int(pos.X), int(pos.Y))
		}
		s.sendPage(p, page("location.htm").replace("%location%", town).replace("%LOCATION%", s.auctionPictureName(p)).
			replace("%AGIT_LINK_BACK%", "bypass -h npc_"+obj+"_start"))
	case "start":
		s.showAuctioneerPage(p, n)
	default:
		return false
	}
	return true
}

// auctionSale is the "auction <days> <bid>" command: a sale of the clan's
// hall is prepared (AgitSale3.htm) and waits for confirmAuction.
func (s *Server) auctionSale(p *Player, n *Npc, val string, rest []string) {
	if val == "" {
		return
	}
	days, err := strconv.Atoi(val)
	if err != nil {
		p.SendMessage("Invalid auction duration!")
		return
	}
	var bid int64
	if len(rest) > 0 {
		if bid, err = strconv.ParseInt(rest[0], 10, 64); err != nil {
			p.SendMessage("Invalid bid!")
			return
		}
	}
	cl := p.Clan()
	h := s.ClanHallByOwner(cl)
	if h == nil || h.siege != nil {
		p.SendMessage("Invalid bid!")
		return
	}
	a := &residence.Auction{ID: h.ID, EndDate: s.Now().UnixMilli() + int64(days)*86_400_000, ItemName: h.Name,
		SellerID: cl.LeaderID, SellerName: cl.LeaderName(), SellerClanName: cl.Name, StartingBid: residence.ClampBid(bid)}
	s.halls.pending[h.ID] = a
	s.sendPage(p, s.pageFile(p, n.ID(), auctionHTML+"AgitSale3.htm").replace("%x%", val).
		replace("%AGIT_AUCTION_END%", s.auctionDate(a.EndDate, auctionDateFormat)).
		replace("%AGIT_AUCTION_MINBID%", strconv.FormatInt(a.StartingBid, 10)).replace("%AGIT_AUCTION_MIN%", strconv.FormatInt(a.StartingBid, 10)).
		replace("%AGIT_AUCTION_DESC%", h.Data.Desc).replace("%AGIT_LINK_BACK%", "bypass -h npc_"+strconv.Itoa(int(n.ID()))+"_sale2").
		replaceObjectID(n.ID()))
}

// auctionList is the "list [page]" command: 15 auctions a page with the page
// links.
func (s *Server) auctionList(p *Player, n *Npc, val string) {
	auctions := s.halls.auctions
	limit, start := auctionPageLimit, 1
	if val != "" {
		pg, err := strconv.Atoi(val)
		if err != nil {
			return
		}
		start = limit*(pg-1) + 1
		limit *= pg
	}
	obj := strconv.Itoa(int(n.ID()))
	pages := (len(auctions) + auctionPageLimit - 1) / auctionPageLimit
	var b strings.Builder
	b.WriteString("<table width=280 border=0><tr>")
	for j := 1; j <= pages; j++ {
		b.WriteString(`<td><center><a action="bypass -h npc_` + obj + `_list ` + strconv.Itoa(j) + `"> Page ` + strconv.Itoa(j) + ` </a></center></td>`)
	}
	b.WriteString("</tr></table><table width=280 border=0>")
	i := 1
	for _, a := range auctions {
		if i > limit {
			break
		}
		if i < start {
			i++
			continue
		}
		i++
		loc := ""
		if h := s.ClanHallByID(a.ID); h != nil {
			loc = h.Data.Location
		}
		b.WriteString("<tr><td>" + loc + `</td><td><a action="bypass -h npc_` + obj + `_bidding ` + idText(a.ID) + `">` + a.ItemName +
			"</a></td><td>" + s.auctionDate(a.EndDate, auctionDateFormatShort) + "</td><td>" + strconv.FormatInt(a.StartingBid, 10) + "</td></tr>")
	}
	b.WriteString("</table>")
	s.sendPage(p, s.pageFile(p, n.ID(), auctionHTML+"AgitAuctionList.htm").
		replace("%AGIT_LINK_BACK%", "bypass -h npc_"+obj+"_start").replace("%itemsField%", b.String()))
}

// auctionSelectedItems is the "selectedItems" command: the clan's bid, its
// sale, or its hall; a clan without either, or a player without a clan,
// gets a message.
func (s *Server) auctionSelectedItems(p *Player, n *Npc) {
	cl := p.Clan()
	obj := strconv.Itoa(int(n.ID()))
	page := func(name string) *htmlPage { return s.pageFile(p, n.ID(), auctionHTML+name) }
	if cl == nil {
		p.SendSystemMessage(serverpackets.SystemMessageID(residence.MsgCannotParticipate))
		return
	}
	if a := s.clanBidAuction(cl); cl.HideoutID == 0 && a != nil {
		bd, _ := a.Bidder(cl.ID)
		pg := s.auctionHallInfo(page("AgitBidInfo.htm"), a).replace("%AGIT_AUCTION_MYBID%", strconv.FormatInt(bd.Bid, 10)).
			replaceObjectID(n.ID()).replace("%AGIT_LINK_BACK%", "bypass -h npc_"+obj+"_start")
		s.sendPage(p, pg)
		return
	}
	if a := s.hallAuction(cl.HideoutID); a != nil && cl.HideoutID != 0 {
		pg := s.auctionHallInfo(page("AgitSaleInfo.htm"), a).replace("%AGIT_OWNER_PLEDGE_NAME%", a.SellerClanName).
			replace("%AGIT_AUCTION_BIDCOUNT%", strconv.Itoa(len(a.Bidders))).
			replace("%AGIT_LINK_BACK%", "bypass -h npc_"+obj+"_start").replace("%id%", idText(a.ID)).replaceObjectID(n.ID())
		s.sendPage(p, pg)
		return
	}
	if cl.HideoutID != 0 {
		pg := page("AgitInfo.htm")
		if h := s.ClanHallByID(cl.HideoutID); h != nil {
			pg.replace("%AGIT_NAME%", h.Name).replace("%AGIT_OWNER_PLEDGE_NAME%", cl.Name).replace("%OWNER_PLEDGE_MASTER%", cl.LeaderName()).
				replace("%AGIT_SIZE%", strconv.Itoa(h.Grade()*10)).replace("%AGIT_LEASE%", strconv.Itoa(h.Lease())).
				replace("%AGIT_LOCATION%", h.Data.Location).replace("%AGIT_LINK_BACK%", "bypass -h npc_"+obj+"_start").replaceObjectID(n.ID())
		}
		s.sendPage(p, pg)
		return
	}
	p.SendSystemMessage(serverpackets.SystemMessageID(residence.MsgNoOfferings))
}

// auctionPictureName is Auctioneer.getPictureName: the town picture of the
// player's map region.
func (s *Server) auctionPictureName(p *Player) string {
	if s.worldData == nil {
		return "ADEN"
	}
	pos := p.Position()
	switch s.worldData.MapRegionLocID(int(pos.X), int(pos.Y)) {
	case 911:
		return "GLUDIN"
	case 912:
		return "GLUDIO"
	case 916:
		return "DION"
	case 918:
		return "GIRAN"
	case 1537:
		return "RUNE"
	case 1538:
		return "GODARD"
	case 1714:
		return "SCHUTTGART"
	}
	return "ADEN"
}
