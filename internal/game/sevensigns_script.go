package game

import (
	"strings"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/sevensigns"
)

// The sevensigns module of the scripts (scripting.SevenSigns) and the
// priests' links: the "SevenSigns ..." links of a SignsPriest's pages
// (npc_<id>_SevenSigns <n> ...) go to the SignsPriest script
// (scripts/npcs/seven_signs), as Mobius SignsPriest.onBypassFeedback runs
// them; the Dawn and Dusk priests' "Chat" shows their first page.

var _ scripting.SevenSigns = (*ScriptOutput)(nil)

// signsPriestScript is the Lua script of the priests.
const signsPriestScript = "SignsPriest"

// signsPriestCommand runs a priest's link; false for a link it does not
// own.
func (s *Server) signsPriestCommand(p *Player, n *Npc, command string) bool {
	priest := s.npcIs(n, "DawnPriest") || s.npcIs(n, "DuskPriest")
	switch {
	case priest && strings.HasPrefix(command, "Chat"):
		if s.scripts != nil && s.scripts.HasFirstTalk(n.Template.ID) {
			s.scriptFirstTalk(p, n)
		} else {
			s.showFirstPage(p, n)
		}
		return true
	case !strings.HasPrefix(command, "SevenSigns"):
		return false
	}
	// Mobius: only the NPC the player talked to last.
	if p.LastNpc != n.ID() {
		return true
	}
	if s.scripts == nil || !s.scripts.HasScript(signsPriestScript) {
		s.showNotYet(p, n)
		return true
	}
	reply, ok := s.scripts.Event(s.background(), signsPriestScript, command, p, n.ScriptRef())
	if ok {
		s.showScriptReply(p, n, reply)
	} else {
		s.showScriptError(p, reply.Err)
	}
	return true
}

func (o *ScriptOutput) ssState() (*Server, *sevensigns.State) {
	s := o.srv.Load()
	if s == nil || s.sevenSigns.state == nil {
		return nil, nil
	}
	return s, s.sevenSigns.state
}

// SevenSignsPeriod is getCurrentPeriod.
func (o *ScriptOutput) SevenSignsPeriod() int {
	if _, st := o.ssState(); st != nil {
		return int(st.Period)
	}
	return int(sevensigns.PeriodCompetition)
}

// SevenSignsCycle is getCurrentCycle.
func (o *ScriptOutput) SevenSignsCycle() int {
	if _, st := o.ssState(); st != nil {
		return st.Cycle
	}
	return 1
}

// SevenSignsSealOwner is getSealOwner.
func (o *ScriptOutput) SevenSignsSealOwner(seal int) int {
	if _, st := o.ssState(); st != nil {
		return int(st.SealOwner(sevensigns.Seal(seal)))
	}
	return 0
}

// SevenSignsHighestCabal is getCabalHighestScore.
func (o *ScriptOutput) SevenSignsHighestCabal() int {
	if _, st := o.ssState(); st != nil {
		return int(st.HighestCabal())
	}
	return 0
}

// SevenSignsPlayerCabal is getPlayerCabal.
func (o *ScriptOutput) SevenSignsPlayerCabal(p scripting.PlayerRef) int {
	if _, st := o.ssState(); st != nil && p != nil {
		return int(st.PlayerCabal(p.ObjectID()))
	}
	return 0
}

// SevenSignsPlayerSeal is getPlayerSeal.
func (o *ScriptOutput) SevenSignsPlayerSeal(p scripting.PlayerRef) int {
	if _, st := o.ssState(); st != nil && p != nil {
		return int(st.PlayerSeal(p.ObjectID()))
	}
	return 0
}

// SevenSignsContribution is getPlayerContribScore.
func (o *ScriptOutput) SevenSignsContribution(p scripting.PlayerRef) int {
	if _, st := o.ssState(); st != nil && p != nil {
		return st.PlayerContribScore(p.ObjectID())
	}
	return 0
}

// SevenSignsJoin is setPlayerInfo, saved at once (decision X4).
func (o *ScriptOutput) SevenSignsJoin(p scripting.PlayerRef, cabal, seal int) {
	s, st := o.ssState()
	if st == nil || p == nil {
		return
	}
	st.SetPlayerInfo(p.ObjectID(), sevensigns.Cabal(cabal), sevensigns.Seal(seal))
	s.markSevenSignsPlayer(p.ObjectID())
	s.saveSevenSigns(false, false)
}

// SevenSignsContribute is addPlayerStoneContrib, saved at once.
func (o *ScriptOutput) SevenSignsContribute(p scripting.PlayerRef, blue, green, red int) int {
	s, st := o.ssState()
	if st == nil || p == nil {
		return -1
	}
	score := st.AddStoneContrib(p.ObjectID(), blue, green, red, s.sevenSigns.cfg.MaxPlayerContrib)
	if score >= 0 {
		s.markSevenSignsPlayer(p.ObjectID())
		s.saveSevenSigns(false, false)
	}
	return score
}

// SevenSignsTakeReward is getAncientAdenaReward(id, true), saved at once.
func (o *ScriptOutput) SevenSignsTakeReward(p scripting.PlayerRef) int {
	s, st := o.ssState()
	if st == nil || p == nil {
		return 0
	}
	n := st.TakeAncientAdenaReward(p.ObjectID())
	s.markSevenSignsPlayer(p.ObjectID())
	s.saveSevenSigns(false, false)
	return n
}

// SevenSignsConfig returns the dialogs' settings.
func (o *ScriptOutput) SevenSignsConfig() scripting.SevenSignsConfig {
	s := o.srv.Load()
	if s == nil {
		return scripting.SevenSignsConfig{}
	}
	c := s.sevenSigns.cfg
	return scripting.SevenSignsConfig{MaxPlayerContrib: c.MaxPlayerContrib, JoinDawnFee: c.JoinDawnFee, ManorsAgreementID: c.ManorsAgreementID,
		DawnTicketQuantity: c.DawnTicketQuantity, DawnTicketPrice: c.DawnTicketPrice, DawnTicketBundle: c.DawnTicketBundle,
		CastleForDawn: c.CastleForDawn, CastleForDusk: c.CastleForDusk}
}

// SevenSignsDestroyItem is Player.destroyItemByItemId.
func (o *ScriptOutput) SevenSignsDestroyItem(p scripting.PlayerRef, itemID int, count int64, sendMessage bool) bool {
	s, pl, ok := o.player(p)
	return ok && count > 0 && s.DestroyItemByItemID(pl, int32(itemID), count, sendMessage) //nolint:gosec // item IDs
}

// SevenSignsAddItem is Player.addItem (Mobius addAdena and addAncientAdena
// with the "You have earned" message).
func (o *ScriptOutput) SevenSignsAddItem(p scripting.PlayerRef, itemID int, count int64, sendMessage bool) bool {
	s, pl, ok := o.player(p)
	if !ok || count <= 0 {
		return false
	}
	if itemID == int(adenaID) {
		return s.AddAdena(pl, count, sendMessage) > 0
	}
	_, err := s.AddItem(pl, int32(itemID), count, 0, sendMessage, true) //nolint:gosec // item IDs
	return err == nil
}

// SevenSignsReduceAdena is Player.reduceAdena with its message.
func (o *ScriptOutput) SevenSignsReduceAdena(p scripting.PlayerRef, count int64) bool {
	s, pl, ok := o.player(p)
	return ok && s.ReduceAdena(pl, count, true)
}

// SevenSignsSendDisappeared sends "$s2 $s1 has disappeared.".
func (o *ScriptOutput) SevenSignsSendDisappeared(p scripting.PlayerRef, itemID int, count int64) {
	if s, pl, ok := o.player(p); ok {
		pl.SendSystemMessage(sysMsgS2S1Disappeared, s.itemNameParam(int32(itemID)), serverpackets.NumberParam(itemCount32(count))) //nolint:gosec // item IDs
	}
}

// SevenSignsHasRoom is validateCapacity.
func (o *ScriptOutput) SevenSignsHasRoom(p scripting.PlayerRef, slots int) bool {
	_, pl, ok := o.player(p)
	return ok && pl.inv.Inventory != nil && pl.inv.ValidateCapacity(int64(slots))
}

// SevenSignsClanCastle is the castle of the player's clan.
func (o *ScriptOutput) SevenSignsClanCastle(p scripting.PlayerRef) int {
	_, pl, ok := o.player(p)
	if !ok {
		return 0
	}
	if cl := pl.Clan(); cl != nil {
		return int(cl.CastleID)
	}
	return 0
}

// SevenSignsTicketCount is Castle.getTicketBuyCount.
func (o *ScriptOutput) SevenSignsTicketCount(castleID int) int {
	s := o.srv.Load()
	if s == nil {
		return 0
	}
	if c := s.CastleByID(int32(castleID)); c != nil { //nolint:gosec // castle IDs
		return c.ticketBuyCount
	}
	return 0
}

// SevenSignsSetTicketCount is Castle.setTicketBuyCount, saved.
func (o *ScriptOutput) SevenSignsSetTicketCount(castleID, n int) {
	s := o.srv.Load()
	if s == nil {
		return
	}
	if c := s.CastleByID(int32(castleID)); c != nil { //nolint:gosec // castle IDs
		c.ticketBuyCount = n
		c.save()
	}
}
