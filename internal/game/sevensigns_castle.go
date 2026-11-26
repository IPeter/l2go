package game

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/sevensigns"
)

// The seals' effects on castles (Plan 14 M11l): the Seal of Strife limits
// the tax (CastleManager.validateTaxes), changes the gates' defence
// (FuncGatesPDefMod, FuncGatesMDefMod), the mercenary tickets a lord may
// place (MercTicket) and, when Dawn owns it, sends the clans of a siege to
// the castle zone's other spawn instead of town (MapRegionData
// .getTeleToLocation). The castle NPC scripts read the seals and the period
// through castle_lua.go.

// sevenSignsSealOwner is the Seal of Strife's owner (SevenSigns.getSealOwner):
// 0 none, 1 Dusk, 2 Dawn.
func (s *Server) sevenSignsSealOwner() int { return s.sevenSignsSealOwnerOf(int(sevensigns.SealStrife)) }

// sevenSignsSealOwnerOf is SevenSigns.getSealOwner(seal).
func (s *Server) sevenSignsSealOwnerOf(seal int) int {
	st := s.sevenSigns.state
	if st == nil {
		return 0
	}
	return int(st.SealOwner(sevensigns.Seal(seal)))
}

// sevenSignsPeriod is SevenSigns.getCurrentPeriod.
func (s *Server) sevenSignsPeriod() int {
	st := s.sevenSigns.state
	if st == nil {
		return int(sevensigns.PeriodCompetition)
	}
	return int(st.Period)
}

// sevenSignsSealValidation is SevenSigns.isSealValidationPeriod, when
// mercenary tickets may be placed.
func (s *Server) sevenSignsSealValidation() bool {
	return s.sevenSignsPeriod() == int(sevensigns.PeriodSealValidation)
}

// SevenSignsGateDefence returns the multipliers of the castle gates' and
// walls' P.Def and M.Def by the Seal of Strife's owner (Mobius
// FuncGatesPDefMod and FuncGatesMDefMod, in every period): the config's
// Dawn or Dusk values (1.1 and 0.8 by default), 1 without an owner. The door
// combat multiplies a door's defence by them. Loop-only.
func (s *Server) SevenSignsGateDefence() (pdef, mdef float64) {
	cfg := &s.sevenSigns.cfg
	switch sevensigns.Cabal(s.sevenSignsSealOwner()) {
	case sevensigns.CabalDawn:
		return cfg.DawnGatesPDefMult, cfg.DawnGatesMDefMult
	case sevensigns.CabalDusk:
		return cfg.DuskGatesPDefMult, cfg.DuskGatesMDefMult
	}
	return 1, 1
}

// dawnSiegeRespawn is MapRegionData.getTeleToLocation's siege rule: while
// Dawn owns the Seal of Strife, a player without karma in a castle's siege
// zone whose clan attacks or defends it goes to the castle zone's other
// spawn.
func (s *Server) dawnSiegeRespawn(p *Player) (data.Location, bool) {
	if p.Karma() > 0 || s.sevenSignsSealOwner() != int(sevensigns.CabalDawn) {
		return data.Location{}, false
	}
	c := s.CastleAt(p.Position())
	cl := p.ClanID()
	if c == nil || !c.siege.inProgress || cl == 0 || (!c.siege.reg.IsDefender(cl) && !c.siege.reg.IsAttacker(cl)) {
		return data.Location{}, false
	}
	return c.spawnLoc(data.ZoneSpawnOther)
}

// validateCastleTaxes is CastleManager.validateTaxes: no castle's tax stays
// above the Seal of Strife owner's limit.
func (s *Server) validateCastleTaxes(strifeOwner sevensigns.Cabal) {
	limit := sevensigns.MaxTax(strifeOwner)
	for _, c := range s.Castles() {
		if c.taxPercent > limit {
			c.SetTaxPercent(limit)
		}
	}
}

// resetCastleTickets is the competition's end: every castle's Dawn ticket
// sale starts again (Castle.setTicketBuyCount(0)).
func (s *Server) resetCastleTickets() {
	for _, c := range s.Castles() {
		if c.ticketBuyCount != 0 {
			c.ticketBuyCount = 0
			c.save()
		}
	}
}
