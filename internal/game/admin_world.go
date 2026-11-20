package game

import (
	"errors"
	"strconv"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
)

// The admin panel's world and server tools: the GM shop (Mobius AdminShop
// admin_buy; //gmshop, its menu page, is in admin.go). The gmstore pages
// below html/admin/gmstore open with //html and offer "admin_buy <listId>"
// buttons.
func init() { registerFeature("admin_world", setupAdminWorld) }

// setupAdminWorld registers the GM commands of the world and server tools.
func setupAdminWorld(s *Server) error {
	admin := storage.AccessLevelAdmin
	return errors.Join(
		s.RegisterGMCommand("buy", admin, s.gmBuy),
	)
}

// gmBuy is //buy <listId> (Mobius AdminShop.handleBuyRequest): the buy
// window of that buylist opens with the GM's adena and no tax, bound to no
// merchant, then ActionFailed. Buying goes through RequestBuyItem, which
// lets GMs buy from any list without a merchant, untaxed, and take the
// GM-only free products (OnlyGMItemsFree). Mobius only logs a list that
// does not exist; the GM is also told here, since some of the gmstore
// buttons name lists that Mobius does not have either.
func (s *Server) gmBuy(p *Player, args []string) {
	if len(args) == 0 {
		p.SendMessage("Please specify buylist.")
		return
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		s.log.Warn("game: admin buylist failed", append(p.logArgs(), "list", args[0])...)
		id = -1
	}
	if l := s.buyList(id); l != nil {
		p.Send(serverpackets.NewBuyListPacket(s.buyListPacket(p, l, 0)))
	} else {
		s.log.Warn("game: no buylist with that id", append(p.logArgs(), "list", args[0])...)
		p.SendMessage("Buylist " + args[0] + " does not exist.")
	}
	p.SendActionFailed()
}
