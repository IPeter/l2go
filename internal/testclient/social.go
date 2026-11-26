package testclient

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// The Milestone 9 flows that take more than one request, often on two
// connections: forming a party, a whole trade, a purchase from a store,
// founding a clan at a village master and taking a player into it.

// System messages of the social flows (Mobius SystemMessageId).
const (
	// SysMsgYouPurchased is "You have purchased $s2 from $s1." (a single
	// item that does not stack).
	SysMsgYouPurchased int32 = 559
	// SysMsgYouPurchasedCount is "You have purchased $s3 $s2(s) from $s1.".
	SysMsgYouPurchasedCount int32 = 561
	// SysMsgTradeSuccessful is "Your trade is successful.".
	SysMsgTradeSuccessful int32 = 123
	// SysMsgFriendLoggedIn is "$s1 (Friend) has logged in.".
	SysMsgFriendLoggedIn int32 = 503
)

// FormParty makes leader invite every member with the loot mode loot (a
// Loot constant), one after the other: each member waits for the
// invitation, accepts it and gets the party window, and the leader gets the
// JoinParty that answers. It returns the leader's party window once it
// shows every member.
func FormParty(ctx context.Context, leader *GameConn, loot int32, members ...*GameConn) (PartyState, error) {
	leaderName := leader.world.Me().Name
	for _, m := range members {
		name := m.world.Me().Name
		if err := leader.RequestJoinParty(ctx, name, loot); err != nil {
			return PartyState{}, err
		}
		if _, err := Await(ctx, m, func(a AskJoinParty) bool { return a.Requestor == leaderName }); err != nil {
			return PartyState{}, fmt.Errorf("%s's invitation: %w", name, err)
		}
		if _, err := m.AcceptPartyInvite(ctx); err != nil {
			return PartyState{}, fmt.Errorf("%s accepts: %w", name, err)
		}
		j, err := Await[JoinParty](ctx, leader, nil)
		if err != nil {
			return PartyState{}, fmt.Errorf("the answer of %s: %w", name, err)
		}
		if j.Answer != PartyAnswerAccepted {
			return PartyState{}, fmt.Errorf("%s answered %d", name, j.Answer)
		}
	}
	var party PartyState
	err := pollWorld(ctx, func() bool {
		p, ok := leader.world.Party()
		party = p
		if !ok {
			return false
		}
		for _, m := range members {
			if !p.HasMember(m.world.Me().Name) {
				return false
			}
		}
		return true
	})
	return party, err
}

// TradeOffer is an inventory item to put into a trade.
type TradeOffer struct {
	ObjectID int32
	Count    int32
}

// TradeResult is how a trade ended: the TradeDone each side got.
type TradeResult struct {
	Requester, Partner TradeDone
}

// Trade runs a whole trade between two characters that stand within 150
// units of each other: from asks to (TradeRequest), to accepts and both
// windows open; from puts give into the trade and to puts take, each item
// shown to the other side (TradeOtherAdd); from confirms, to sees it
// (TradeOtherDone) and confirms, and both windows close (TradeDone). Mobius
// closes the windows with TradeDoneFinished also when the exchange failed
// (the inventories did not fit): then "Your trade is successful." does not
// come (SysMsgTradeSuccessful). A refused step ends with the context.
func Trade(ctx context.Context, from, to *GameConn, give, take []TradeOffer) (TradeResult, error) {
	var res TradeResult
	fromID, toID := from.world.Me().ObjectID, to.world.Me().ObjectID
	if err := from.RequestTrade(ctx, toID); err != nil {
		return res, err
	}
	if _, err := Await(ctx, to, func(r SendTradeRequest) bool { return r.SenderID == fromID }); err != nil {
		return res, fmt.Errorf("the trade request: %w", err)
	}
	if _, err := to.AcceptTrade(ctx); err != nil {
		return res, fmt.Errorf("accepting the trade: %w", err)
	}
	if _, err := Await(ctx, from, func(s TradeStart) bool { return s.PartnerID == toID }); err != nil {
		return res, fmt.Errorf("the requester's trade window: %w", err)
	}
	if err := addTradeItems(ctx, from, to, give); err != nil {
		return res, err
	}
	if err := addTradeItems(ctx, to, from, take); err != nil {
		return res, err
	}
	if err := from.ConfirmTrade(ctx); err != nil {
		return res, err
	}
	if _, err := Await[TradeOtherDone](ctx, to, nil); err != nil {
		return res, fmt.Errorf("the requester's confirmation: %w", err)
	}
	if err := to.ConfirmTrade(ctx); err != nil {
		return res, err
	}
	var err error
	if res.Partner, err = Await[TradeDone](ctx, to, nil); err != nil {
		return res, fmt.Errorf("the partner's TradeDone: %w", err)
	}
	if res.Requester, err = Await[TradeDone](ctx, from, nil); err != nil {
		return res, fmt.Errorf("the requester's TradeDone: %w", err)
	}
	return res, nil
}

// addTradeItems puts items of owner into the open trade, each shown to the
// partner.
func addTradeItems(ctx context.Context, owner, partner *GameConn, items []TradeOffer) error {
	for _, it := range items {
		if _, err := owner.AddTradeItem(ctx, it.ObjectID, it.Count); err != nil {
			return fmt.Errorf("adding item %d: %w", it.ObjectID, err)
		}
		if _, err := Await(ctx, partner, func(a TradeOtherAdd) bool { return a.Item.ObjectID == it.ObjectID }); err != nil {
			return fmt.Errorf("the partner's view of item %d: %w", it.ObjectID, err)
		}
	}
	return nil
}

// Purchase buys items from the sell store of storeID (RequestPrivateStoreBuy)
// and returns the purchase message of the last item ("You have purchased
// ..."). Mobius sells only to a buyer within 250 units of the store, so call
// ViewSellStore first, which walks there. A refused purchase ends with the
// context.
func (c *GameConn) Purchase(ctx context.Context, storeID int32, items []StorePurchase) (SystemMessage, error) {
	if err := c.BuyFromStore(ctx, storeID, items); err != nil {
		return SystemMessage{}, err
	}
	var last SystemMessage
	for range items {
		m, err := Await(ctx, c, func(m SystemMessage) bool {
			return m.ID == SysMsgYouPurchased || m.ID == SysMsgYouPurchasedCount
		})
		if err != nil {
			return last, err
		}
		last = m
	}
	return last, nil
}

// clanCreatePages are the links from a village master's first page to the
// clan name page, as the shipped pages have them (the ClanMaster script,
// page 9000-02.htm).
var clanCreatePages = []string{"ClanMaster", "9000-02.htm"}

// CreateClan founds the clan name at the village master villageMaster: it
// talks to the NPC, follows its pages to the clan name page (the ClanMaster
// links, as the shipped pages have them) and sends the page's create_clan
// link with the name filled in, as the client does when the player types it.
// It returns the PledgeShowMemberListAll of the new clan. A refusal (level,
// name, penalty) comes as a system message, so it ends with the context.
func (c *GameConn) CreateClan(ctx context.Context, villageMaster int32, name string) (PledgeShowMemberListAll, error) {
	page, err := c.Talk(ctx, villageMaster)
	if err != nil {
		return PledgeShowMemberListAll{}, err
	}
	for _, part := range clanCreatePages {
		if _, ok := PageBypass(page.HTML, "create_clan"); ok {
			break
		}
		link, ok := PageBypass(page.HTML, part)
		if !ok {
			return PledgeShowMemberListAll{}, fmt.Errorf("the page of %d has no link to %s", villageMaster, part)
		}
		if err := c.Bypass(ctx, link); err != nil {
			return PledgeShowMemberListAll{}, err
		}
		if page, err = Await[NpcHtmlMessage](ctx, c, nil); err != nil {
			return PledgeShowMemberListAll{}, err
		}
	}
	link, ok := PageBypass(page.HTML, "create_clan")
	if !ok {
		return PledgeShowMemberListAll{}, fmt.Errorf("the clan name page of %d has no create_clan link", villageMaster)
	}
	if err := c.Bypass(ctx, strings.Replace(link, "$name", name, 1)); err != nil {
		return PledgeShowMemberListAll{}, err
	}
	return Await(ctx, c, func(p PledgeShowMemberListAll) bool { return p.ClanName == name })
}

// JoinClan takes member into leader's clan or its unit pledgeType (a Pledge
// constant): leader invites (RequestJoinPledge), member waits for the
// invitation (AskJoinPledge) and accepts it, and gets JoinPledge.
func JoinClan(ctx context.Context, leader, member *GameConn, pledgeType int32) (JoinPledge, error) {
	if err := leader.InviteToClan(ctx, member.world.Me().ObjectID, pledgeType); err != nil {
		return JoinPledge{}, err
	}
	leaderID := leader.world.Me().ObjectID
	if _, err := Await(ctx, member, func(a AskJoinPledge) bool { return a.RequestorID == leaderID }); err != nil {
		return JoinPledge{}, fmt.Errorf("the clan invitation: %w", err)
	}
	return member.AcceptClanInvite(ctx)
}

// bypassRe matches the action of a bypass link: action="bypass -h cmd" or
// action="bypass cmd".
var bypassRe = regexp.MustCompile(`(?i)action\s*=\s*"\s*bypass(?:\s+-h)?\s+([^"]+?)\s*"`)

// PageBypass returns the first bypass command of a dialog page that
// contains part, as RequestBypassToServer sends it (without "bypass -h").
func PageBypass(html, part string) (string, bool) {
	for _, m := range bypassRe.FindAllStringSubmatch(html, -1) {
		if strings.Contains(m[1], part) {
			return m[1], true
		}
	}
	return "", false
}
