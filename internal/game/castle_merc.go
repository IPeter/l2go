package game

import (
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// Mercenary posting tickets (Mobius MercTicketManager and the MercTicket item
// handler): the lord of a castle drops a ticket in the castle outside a
// siege; it stays on the ground and hires the mercenary, who stands there
// for 3 seconds and defends the castle at its next siege, within the limits
// of the Seven Signs (the seal validation period, the Seal of Strife's
// owner: sevensigns_castle.go).

// handlerMercTicket is the item handler of the tickets.
const handlerMercTicket = "MercTicket"

// Messages of the tickets.
const (
	mercMsgPlace       serverpackets.SystemMessageID = 651
	mercMsgNoAuthority serverpackets.SystemMessageID = 653
	mercMsgNotHere     serverpackets.SystemMessageID = 655
	mercMsgNoMore      serverpackets.SystemMessageID = 656
	mercMsgTooClose    serverpackets.SystemMessageID = 657
)

const (
	// mercPreviewDespawn is how long the placed mercenary stands, in
	// milliseconds (spawnMercenary).
	mercPreviewDespawn = 3000
	// mercMinDistance is isTooCloseToAnotherTicket's distance.
	mercMinDistance = 25
	// sealOfStrifeDusk is SevenSigns.CABAL_DUSK.
	sealOfStrifeDusk = 1
)

// groundTicket is a ticket lying in a castle (DROPPED_TICKETS).
type groundTicket struct {
	castle *Castle
	ticket *data.MercTicket
	item   *GroundItem
}

// sevenSignsSealOwner (the Seal of Strife's owner) and
// sevenSignsSealValidation are in sevensigns_castle.go.

// dawnTicket and rookieTicket are SevenSigns.checkIsDawnPostingTicket and
// checkIsRookiePostingTicket.
func dawnTicket(id int) bool {
	return (id > 6114 && id < 6175) || (id > 6801 && id < 6812) || (id > 7997 && id < 8008) || (id > 7940 && id < 7951) ||
		(id > 6294 && id < 6307) || (id > 6831 && id < 6834) || (id > 8027 && id < 8030) || (id > 7970 && id < 7973)
}

func rookieTicket(id int) bool {
	return (id > 6174 && id < 6295) || (id > 6811 && id < 6832) || (id > 7950 && id < 7971) || (id > 8007 && id < 8028)
}

// useMercTicket is the MercTicket item handler.
func (s *Server) useMercTicket(p *Player, it *inventory.Item) bool {
	f := &s.residence
	if f.data == nil {
		return false
	}
	id := int(it.ItemID())
	t, ok := f.data.Ticket(id)
	c := s.CastleAt(p.Position())
	switch {
	case !ok || c == nil || int(c.ID) != t.CastleID:
		p.SendSystemMessage(mercMsgNotHere)
		return false
	case c.ownerID == 0 || p.ClanID() != c.ownerID || !p.IsClanLeader():
		p.SendSystemMessage(mercMsgNoAuthority)
		return false
	case c.siege.inProgress, !s.sevenSignsSealValidation():
		p.SendSystemMessage(mercMsgNoMore)
		return false
	}
	switch s.sevenSignsSealOwner() {
	case 0:
		if dawnTicket(id) {
			p.SendSystemMessage(mercMsgNoMore)
			return false
		}
	case sealOfStrifeDusk:
		if !rookieTicket(id) {
			p.SendSystemMessage(mercMsgNoMore)
			return false
		}
	}
	pos := p.Position()
	switch {
	case s.ticketsOf(c) >= c.Data.MaxMercenaries, s.ticketsOfType(id) >= t.MaxPerType:
		p.SendSystemMessage(mercMsgNoMore)
		return false
	case s.ticketTooClose(pos):
		p.SendSystemMessage(mercMsgTooClose)
		return false
	}
	// MercTicketManager.addTicket.
	if n := s.spawnSiegeNpc(t.NpcID, world.Position{X: pos.X, Y: pos.Y, Z: pos.Z + 20}, pos.Heading, ""); n != nil {
		s.loop.After(mercPreviewDespawn*time.Millisecond, func() {
			if n.InWorld() {
				s.DespawnNpc(n)
			}
		})
	}
	c.hired = append(c.hired, storage.HiredGuard{NpcID: int32(t.NpcID), X: pos.X, Y: pos.Y, Z: pos.Z, Heading: pos.Heading, TicketItem: int32(id)}) //nolint:gosec // IDs
	s.dropTicket(c, t, pos)
	s.DestroyItem(p, it, 1, false)
	p.SendSystemMessage(mercMsgPlace, serverpackets.ItemNameParam(int32(id))) //nolint:gosec // item ID
	c.save()
	return true
}

// dropTicket puts a ticket on the ground (dropMe with no auto-destroy).
func (s *Server) dropTicket(c *Castle, t *data.MercTicket, pos world.Position) {
	tmpl, ok := s.data.ItemTemplate(t.ItemID)
	if !ok || s.items.pool == nil {
		return
	}
	oid, ok := s.items.pool.TryNext()
	if !ok {
		return
	}
	g, err := s.DropOnGround(inventory.NewItem(oid, tmpl, 1), pos, GroundDrop{PlayerDrop: true})
	if err != nil {
		s.log.Warn("game: a mercenary ticket cannot go on the ground", "castle", c.Name, "err", err)
		return
	}
	s.residence.tickets = append(s.residence.tickets, &groundTicket{castle: c, ticket: t, item: g})
}

func (s *Server) ticketsOf(c *Castle) int {
	n := 0
	for _, g := range s.residence.tickets {
		if g.castle == c && g.item.InWorld() {
			n++
		}
	}
	return n
}

func (s *Server) ticketsOfType(itemID int) int {
	n := 0
	for _, g := range s.residence.tickets {
		if g.ticket.ItemID == itemID && g.item.InWorld() {
			n++
		}
	}
	return n
}

func (s *Server) ticketTooClose(pos world.Position) bool {
	for _, g := range s.residence.tickets {
		if g.item.InWorld() && g.item.Position().DistanceSq3D(pos) < mercMinDistance*mercMinDistance {
			return true
		}
	}
	return false
}

// restoreMercTickets is MercTicketManager.load: the hired mercenaries of
// castles not under siege show their tickets again, up to each castle's
// limit.
func (s *Server) restoreMercTickets() {
	f := &s.residence
	if f.data == nil {
		return
	}
	for _, c := range f.castles {
		if c.siege.inProgress {
			continue
		}
		for i, h := range c.hired {
			if i >= c.Data.MaxMercenaries {
				break
			}
			if t, ok := f.data.Ticket(int(h.TicketItem)); ok {
				s.dropTicket(c, t, world.Position{X: h.X, Y: h.Y, Z: h.Z})
			}
		}
	}
}

// deleteMercTickets is MercTicketManager.deleteTickets: a castle's tickets
// leave the ground at its siege's start.
func (s *Server) deleteMercTickets(c *Castle) {
	kept := s.residence.tickets[:0]
	for _, g := range s.residence.tickets {
		if g.castle == c {
			if g.item.InWorld() {
				s.removeGroundItem(g.item)
			}
			continue
		}
		kept = append(kept, g)
	}
	s.residence.tickets = kept
}
