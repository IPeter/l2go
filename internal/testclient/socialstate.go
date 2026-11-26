package testclient

import (
	"cmp"
	"slices"
)

// The social part of the World model (Milestone 9): the party window, the
// open trade, the friend list and friend messages, and the invitations and
// requests the character has not answered yet. A long-running client (a
// bot) reads them from the World instead of waiting for their packets, which
// another wait may have skipped.

// PartyState is the party window.
type PartyState struct {
	LeaderID int32
	Loot     int32
	// Members are the other members: Mobius shows each member the party
	// without itself.
	Members []PartyMember
}

// Member returns the member objectID.
func (p PartyState) Member(objectID int32) (PartyMember, bool) {
	for _, m := range p.Members {
		if m.ObjectID == objectID {
			return m, true
		}
	}
	return PartyMember{}, false
}

// HasMember reports whether the member name is in the window.
func (p PartyState) HasMember(name string) bool {
	return slices.ContainsFunc(p.Members, func(m PartyMember) bool { return m.Name == name })
}

// TradeState is the open trade window.
type TradeState struct {
	PartnerID int32
	// Own and Partner are the items each side put in (TradeOwnAdd and
	// TradeOtherAdd).
	Own, Partner []TradeItem
	// PartnerConfirmed is set by TradeOtherDone and cleared when the
	// partner adds an item, which takes the confirmation back.
	PartnerConfirmed bool
}

// socialState is the social part of a World; the World's lock guards it.
type socialState struct {
	party   PartyState
	inParty bool
	trade   TradeState
	trading bool

	partyInvite     AskJoinParty
	hasPartyInvite  bool
	tradeRequest    SendTradeRequest
	hasTradeRequest bool
	friendInvite    FriendAddRequest
	hasFriendInvite bool
	clanInvite      AskJoinPledge
	hasClanInvite   bool

	friends    map[string]Friend
	friendSays []L2FriendSay
}

// Party returns the party window; ok is false outside a party.
func (w *World) Party() (PartyState, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	p := w.social.party
	p.Members = slices.Clone(p.Members)
	return p, w.social.inParty
}

// Trade returns the open trade window; ok is false without one.
func (w *World) Trade() (TradeState, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	t := w.social.trade
	t.Own, t.Partner = slices.Clone(t.Own), slices.Clone(t.Partner)
	return t, w.social.trading
}

// TakePartyInvite returns the last party invitation (AskJoinParty) that came
// since the last call, and forgets it.
func (w *World) TakePartyInvite() (AskJoinParty, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	v, ok := w.social.partyInvite, w.social.hasPartyInvite
	w.social.partyInvite, w.social.hasPartyInvite = AskJoinParty{}, false
	return v, ok
}

// TakeTradeRequest returns the last trade request (SendTradeRequest) that
// came since the last call, and forgets it.
func (w *World) TakeTradeRequest() (SendTradeRequest, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	v, ok := w.social.tradeRequest, w.social.hasTradeRequest
	w.social.tradeRequest, w.social.hasTradeRequest = SendTradeRequest{}, false
	return v, ok
}

// TakeFriendInvite returns the last friend invitation (FriendAddRequest)
// that came since the last call, and forgets it.
func (w *World) TakeFriendInvite() (FriendAddRequest, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	v, ok := w.social.friendInvite, w.social.hasFriendInvite
	w.social.friendInvite, w.social.hasFriendInvite = FriendAddRequest{}, false
	return v, ok
}

// TakeClanInvite returns the last clan invitation (AskJoinPledge) that came
// since the last call, and forgets it.
func (w *World) TakeClanInvite() (AskJoinPledge, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	v, ok := w.social.clanInvite, w.social.hasClanInvite
	w.social.clanInvite, w.social.hasClanInvite = AskJoinPledge{}, false
	return v, ok
}

// Friends returns the friend list, sorted by name: the FriendList of the
// enter-world sequence with the L2Friend and FriendStatusPacket changes
// after it.
func (w *World) Friends() []Friend {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]Friend, 0, len(w.social.friends))
	for _, f := range w.social.friends {
		out = append(out, f)
	}
	slices.SortFunc(out, func(a, b Friend) int { return cmp.Compare(a.Name, b.Name) })
	return out
}

// FriendMessages returns the friend messages received (L2FriendSay), oldest
// first.
func (w *World) FriendMessages() []L2FriendSay {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.social.friendSays)
}

// applySocial handles the social packets; the caller holds w.mu.
func (w *World) applySocial(p Packet) {
	s := &w.social
	switch v := p.(type) {
	case AskJoinParty:
		s.partyInvite, s.hasPartyInvite = v, true
	case PartySmallWindowAll:
		s.party = PartyState{LeaderID: v.LeaderID, Loot: v.Loot, Members: slices.Clone(v.Members)}
		s.inParty = true
	case PartySmallWindowAdd:
		if !s.inParty {
			s.party, s.inParty = PartyState{}, true
		}
		s.party.LeaderID, s.party.Loot = v.LeaderID, v.Loot
		s.setPartyMember(v.Member)
	case PartySmallWindowUpdate:
		if i := s.partyMemberIndex(v.Member.ObjectID); i >= 0 {
			m := v.Member
			m.Unknown, m.Race = s.party.Members[i].Unknown, s.party.Members[i].Race
			s.party.Members[i] = m
		}
	case PartySmallWindowDelete:
		if i := s.partyMemberIndex(v.ObjectID); i >= 0 {
			s.party.Members = slices.Delete(s.party.Members, i, i+1)
		}
		// The last member of a dispersed party gets no
		// PartySmallWindowDeleteAll (Mobius Party.removePartyMember), only
		// the Delete of the one who left: its window is empty.
		if len(s.party.Members) == 0 {
			s.party, s.inParty = PartyState{}, false
		}
	case PartySmallWindowDeleteAll:
		s.party, s.inParty = PartyState{}, false
	case SendTradeRequest:
		s.tradeRequest, s.hasTradeRequest = v, true
	case TradeStart:
		s.trade, s.trading = TradeState{PartnerID: v.PartnerID}, true
	case TradeOwnAdd:
		if s.trading {
			s.trade.Own = append(s.trade.Own, v.Item)
		}
	case TradeOtherAdd:
		if s.trading {
			s.trade.Partner = append(s.trade.Partner, v.Item)
			s.trade.PartnerConfirmed = false
		}
	case TradeOtherDone:
		if s.trading {
			s.trade.PartnerConfirmed = true
		}
	case TradeDone:
		s.trade, s.trading = TradeState{}, false
	default:
		w.applyFriendsAndClan(p)
	}
}

// applyFriendsAndClan handles the friend, clan and title packets; the
// caller holds w.mu.
func (w *World) applyFriendsAndClan(p Packet) {
	s := &w.social
	switch v := p.(type) {
	case FriendAddRequest:
		s.friendInvite, s.hasFriendInvite = v, true
	case FriendList:
		s.friends = make(map[string]Friend, len(v.Friends))
		for _, f := range v.Friends {
			s.friends[f.Name] = f
		}
	case L2Friend:
		if s.friends == nil {
			s.friends = make(map[string]Friend)
		}
		switch v.Action {
		case FriendAdded, FriendModified:
			f := Friend{ObjectID: v.ObjectID, Name: v.Name, Online: v.Online}
			if v.Online {
				f.OnlineObjectID = v.ObjectID
			}
			s.friends[v.Name] = f
		case FriendRemoved:
			delete(s.friends, v.Name)
		}
	case FriendStatus:
		if f, ok := s.friends[v.Name]; ok {
			f.Online, f.OnlineObjectID = v.Online, 0
			if v.Online {
				f.OnlineObjectID = v.ObjectID
			}
			s.friends[v.Name] = f
		}
	case L2FriendSay:
		s.friendSays = appendLog(s.friendSays, v)
	case AskJoinPledge:
		s.clanInvite, s.hasClanInvite = v, true
	case NicknameChanged:
		if o, ok := w.objects[v.ObjectID]; ok {
			o.Title = v.Title
		}
	}
}

// partyMemberIndex returns the index of the member objectID, or -1.
func (s *socialState) partyMemberIndex(objectID int32) int {
	return slices.IndexFunc(s.party.Members, func(m PartyMember) bool { return m.ObjectID == objectID })
}

// setPartyMember adds m to the party window or replaces its line.
func (s *socialState) setPartyMember(m PartyMember) {
	if i := s.partyMemberIndex(m.ObjectID); i >= 0 {
		s.party.Members[i] = m
		return
	}
	s.party.Members = append(s.party.Members, m)
}
