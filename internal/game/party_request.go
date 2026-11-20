package game

import (
	"time"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
)

// The two request slots of Plan 12 decision T7.
//
//   - The active requester (Mobius Player._activeRequester and
//     _requestExpireTime): party, trade and friend invitations. Its state is
//     in trade_request.go; the Player methods below are the same slot as
//     Mobius names it.
//   - The request partner (Mobius RequestPartner): clan and alliance
//     invitations, below. Each side forgets the request 15 s after it was
//     made (by the world clock), and an answer clears both sides.
//
// Leaving the game clears both slots of the player (the trade's and the
// party's leave hooks).

// ActiveRequester returns who asked p and still waits for p's answer
// (Mobius getActiveRequester): an expired request is forgotten unless p
// trades. A requester that left the game counts as none. Loop-only.
func (p *Player) ActiveRequester() *Player {
	r := p.srv.activeRequester(p)
	if r != nil && !r.InGame() {
		p.srv.setActiveRequester(p, nil)
		return nil
	}
	return r
}

// SetActiveRequester sets, or with nil clears, who asked p (Mobius
// setActiveRequester). Loop-only.
func (p *Player) SetActiveRequester(r *Player) { p.srv.setActiveRequester(p, r) }

// IsProcessingRequest reports whether p was asked and has not answered,
// or waits for the answer to its own request (Mobius
// isProcessingRequest). Loop-only.
func (p *Player) IsProcessingRequest() bool {
	return p.ActiveRequester() != nil || !p.srv.requestExpired(p)
}

// IsRequestExpired reports whether p's own request ran out (Mobius
// isRequestExpired). Loop-only.
func (p *Player) IsRequestExpired() bool { return p.srv.requestExpired(p) }

// OnTransactionRequest starts p's request to partner (Mobius
// onTransactionRequest): it runs out in 15 s, and p becomes partner's
// active requester. Loop-only.
func (p *Player) OnTransactionRequest(partner *Player) { p.srv.onTransactionRequest(p, partner) }

// OnTransactionResponse ends p's own request (Mobius
// onTransactionResponse). Loop-only.
func (p *Player) OnTransactionResponse() { p.srv.onTransactionResponse(p) }

// requestPartner is one side of a Mobius RequestPartner.
type requestPartner struct {
	partner *Player
	// packet is what the requester asked (Mobius getRequestPacket): the
	// feature's own request value.
	packet any
	// requestor is set on the side that asked.
	requestor bool
	expires   time.Time
}

// RequestPartner returns p's partner in a clan or alliance request and the
// request, nil when there is none or it ran out (Mobius
// RequestPartner.getPartner and getRequestPacket). Loop-only.
func (s *Server) RequestPartner(p *Player) (*Player, any) {
	r := &p.party.partner
	if r.partner == nil {
		return nil, nil
	}
	if !s.Now().Before(r.expires) || !r.partner.InGame() {
		*r = requestPartner{}
		return nil, nil
	}
	return r.partner, r.packet
}

// ProcessingPartnerRequest reports whether p is in a clan or alliance
// request (Mobius RequestPartner.isProcessingRequest). Loop-only.
func (s *Server) ProcessingPartnerRequest(p *Player) bool {
	partner, _ := s.RequestPartner(p)
	return partner != nil
}

// IsPartnerRequestor reports whether p asked in its open request partner
// slot. Loop-only.
func (s *Server) IsPartnerRequestor(p *Player) bool {
	partner, _ := s.RequestPartner(p)
	return partner != nil && p.party.partner.requestor
}

// SetRequest opens a request from p to partner with the feature's request
// value (Mobius RequestPartner.setRequest): it refuses a missing partner
// ("You have invited the wrong target."), a busy partner ("$s1 is busy.
// Please try again later.") and a busy p ("Waiting for another reply.").
// Both sides forget it in 15 s. Loop-only.
func (s *Server) SetRequest(p, partner *Player, packet any) bool {
	switch {
	case partner == nil:
		p.SendSystemMessage(partyMsgWrongTarget)
		return false
	case s.ProcessingPartnerRequest(partner):
		p.SendSystemMessage(partyMsgBusy, serverpackets.TextParam(partner.Name()))
		return false
	case s.ProcessingPartnerRequest(p):
		p.SendSystemMessage(partyMsgWaitingReply)
		return false
	}
	expires := s.Now().Add(requestTimeout)
	p.party.partner = requestPartner{partner: partner, packet: packet, requestor: true, expires: expires}
	partner.party.partner = requestPartner{partner: p, packet: packet, expires: expires}
	return true
}

// OnRequestResponse clears p's request partner slot and its partner's
// (Mobius RequestPartner.onRequestResponse), after an answer. Loop-only.
func (s *Server) OnRequestResponse(p *Player) {
	if q := p.party.partner.partner; q != nil {
		q.party.partner = requestPartner{}
	}
	p.party.partner = requestPartner{}
}

// clearRequestPartner forgets p's request partner when p leaves, on both
// sides when the partner still waits for p.
func (s *Server) clearRequestPartner(p *Player) {
	if q := p.party.partner.partner; q != nil && q.party.partner.partner == p {
		q.party.partner = requestPartner{}
	}
	p.party.partner = requestPartner{}
}
