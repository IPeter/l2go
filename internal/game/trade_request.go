package game

import "time"

// The "active requester" request slot of Plan 12 decision T7 as the trade
// uses it (Mobius Player._activeRequester and _requestExpireTime): a
// request waits 15 s for its answer, by the world clock. Loop-only.

// requestTimeout is how long a request waits for its answer (Mobius
// Player.REQUEST_TIMEOUT).
const requestTimeout = 15 * time.Second

// tradeRequests are the request slots of the players.
type tradeRequests struct {
	// requester is who asked the key player (Mobius _activeRequester).
	requester map[*Player]*Player
	// expires is when the key player's own request runs out (Mobius
	// _requestExpireTime).
	expires map[*Player]time.Time
}

func newTradeRequests() tradeRequests {
	return tradeRequests{requester: make(map[*Player]*Player), expires: make(map[*Player]time.Time)}
}

// activeRequester returns who asked p and waits for the answer (Mobius
// getActiveRequester): an expired request is forgotten unless p trades, and
// so is a requester that left the game.
func (s *Server) activeRequester(p *Player) *Player {
	r := s.trade.requests.requester[p]
	if r != nil && (!r.InGame() || (s.requestExpired(r) && !s.Trading(p))) {
		delete(s.trade.requests.requester, p)
		return nil
	}
	return r
}

// setActiveRequester sets (or, with nil, clears) who asked p.
func (s *Server) setActiveRequester(p, requester *Player) {
	if requester == nil {
		delete(s.trade.requests.requester, p)
		return
	}
	s.trade.requests.requester[p] = requester
}

// processingRequest reports whether p was asked and has not answered, or
// waits for the answer to its own request (Mobius isProcessingRequest).
func (s *Server) processingRequest(p *Player) bool {
	return s.activeRequester(p) != nil || s.Now().Before(s.trade.requests.expires[p])
}

// requestExpired reports whether p's own request ran out (Mobius
// isRequestExpired).
func (s *Server) requestExpired(p *Player) bool {
	return !s.Now().Before(s.trade.requests.expires[p])
}

// onTransactionRequest starts p's request to partner (Mobius
// onTransactionRequest): it runs out in 15 s, and p is partner's requester.
func (s *Server) onTransactionRequest(p, partner *Player) {
	s.trade.requests.expires[p] = s.Now().Add(requestTimeout)
	s.setActiveRequester(partner, p)
}

// onTransactionResponse ends p's own request (Mobius onTransactionResponse).
func (s *Server) onTransactionResponse(p *Player) {
	delete(s.trade.requests.expires, p)
}
