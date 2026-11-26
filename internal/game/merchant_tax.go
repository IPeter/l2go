package game

import (
	"math"
	"strconv"

	"github.com/IPeter/l2go/internal/data"
)

// Castle tax in the shops (Plan 14 M11i, decision I7): a price config of a
// castle adds the castle's tax percent to its base tax
// (MerchantPriceConfig.getTotalTax), and after a purchase the castle the
// merchant belongs to (Npc.getCastle) gets subtotal × its price config's
// castle tax rate in its treasury (RequestBuyItem).

// priceConfigTaxRate is MerchantPriceConfig.getTotalTaxRate.
func (s *Server) priceConfigTaxRate(pc *data.PriceConfig) float64 {
	if pc.CastleID > 0 {
		if c := s.CastleByID(int32(pc.CastleID)); c != nil { //nolint:gosec // castle IDs are small
			return float64(c.TaxPercent()+pc.BaseTax) / 100.0
		}
	}
	return pc.BaseTaxRate()
}

// merchantTreasury puts the castle's part of a purchase into the treasury of
// the merchant's castle: (long) (subTotal × castle tax rate).
func (s *Server) merchantTreasury(n *Npc, subTotal int64) {
	pc := n.priceConfig
	if pc == nil || pc.CastleID <= 0 {
		return
	}
	cfgCastle := s.CastleByID(int32(pc.CastleID)) //nolint:gosec // castle IDs are small
	if cfgCastle == nil || cfgCastle.taxPercent == 0 {
		return
	}
	if c := s.npcCastle(n); c != nil {
		c.AddToTreasury(int64(float64(subTotal) * cfgCastle.TaxRate()))
	}
}

// npcCastle is Npc.getCastle: the castle its town pays tax to (the town
// zone's taxById), else the castle whose siege zone is nearest.
func (s *Server) npcCastle(n *Npc) *Castle {
	pos := n.Position()
	if z := s.zoneAt(pos, "TownZone"); z != nil {
		if v, ok := z.Param("taxById"); ok {
			if id, err := strconv.Atoi(v); err == nil {
				if c := s.CastleByID(int32(id)); c != nil { //nolint:gosec // castle IDs are small
					return c
				}
			}
		}
	}
	if c := s.CastleAt(pos); c != nil {
		return c
	}
	var best *Castle
	dist := math.MaxFloat64
	for _, c := range s.residence.castles {
		if c.siegeZone == nil {
			continue
		}
		minX, minY, maxX, maxY := c.siegeZone.Shape.Bounds()
		dx := float64(pos.X) - float64(minX+maxX)/2
		dy := float64(pos.Y) - float64(minY+maxY)/2
		if d := math.Hypot(dx, dy); d < dist {
			best, dist = c, d
		}
	}
	return best
}
