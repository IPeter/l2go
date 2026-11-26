package game

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/residence"
	"github.com/IPeter/l2go/internal/storage"
)

// The clan hall GM commands: Mobius AdminSiege's admin_clanhall* (a hall's
// page, give, take, doors, teleport) and AdminCHSiege's admin_chsiege_*
// (the contestable halls' sieges). Each takes the hall's ID first.

var hallGMCommands = []string{
	"clanhall", "clanhallset", "clanhalldel", "clanhallopendoors", "clanhallclosedoors", "clanhallteleportself",
}

var hallSiegeGMCommands = []string{
	"chsiege_siegablehall", "chsiege_startSiege", "chsiege_endsSiege", "chsiege_setSiegeDate", "chsiege_addAttacker",
	"chsiege_removeAttacker", "chsiege_clearAttackers", "chsiege_listAttackers", "chsiege_forwardSiege",
}

// registerHallGM registers the clan hall commands.
func (s *Server) registerHallGM() error {
	var errs []error
	for _, name := range hallGMCommands {
		errs = append(errs, s.RegisterGMCommand(name, storage.AccessLevelAdmin, func(p *Player, args []string) {
			s.gmClanHall(p, name, args)
		}))
	}
	for _, name := range hallSiegeGMCommands {
		errs = append(errs, s.RegisterGMCommand(name, storage.AccessLevelAdmin, func(p *Player, args []string) {
			s.gmHallSiege(p, name, args)
		}))
	}
	return errors.Join(errs...)
}

// gmHallArg returns the hall named by the first argument.
func (s *Server) gmHallArg(p *Player, args []string) *ClanHall {
	if len(args) == 0 {
		p.SendMessage("You have to specify the hall id at least")
		return nil
	}
	id, err := strconv.Atoi(args[0])
	h := s.ClanHallByID(int32(id)) //nolint:gosec // hall IDs
	if err != nil || h == nil {
		p.SendMessage("Could not find the clan hall (" + args[0] + ")")
		return nil
	}
	return h
}

// gmClanHall is AdminSiege's clan hall commands.
func (s *Server) gmClanHall(p *Player, cmd string, args []string) {
	h := s.gmHallArg(p, args)
	if h == nil {
		return
	}
	target, _ := p.Target.(*Player)
	switch cmd {
	case "clanhallset":
		switch {
		case target == nil || target.Clan() == nil:
			p.SendMessage("You must target a player with a clan.")
			return
		case h.ownerID > 0:
			p.SendMessage("This clan hall is not free.")
			return
		case target.Clan().HideoutID > 0:
			p.SendMessage("You have already a clan hall.")
			return
		}
		if h.siege == nil {
			if a := s.hallAuction(h.ID); a != nil {
				s.removeAuction(a)
				s.SaveJoint(nil, storage.Batch{DeletedAuctions: []int32{a.ID}})
			}
		}
		s.setHallOwner(h, target.Clan())
	case "clanhalldel":
		if h.ownerID == 0 {
			p.SendMessage("This clan hall is already free.")
			return
		}
		s.freeHall(h)
		if h.siege == nil && s.hallAuction(h.ID) == nil {
			if a := s.newNpcAuction(h); a != nil {
				s.halls.auctions = append(s.halls.auctions, a)
				s.startAuctionTask(a)
				s.SaveJoint(nil, storage.Batch{Auctions: []storage.Auction{a.document()}})
			}
		}
	case "clanhallopendoors":
		s.openCloseHallDoors(h, true)
	case "clanhallclosedoors":
		s.openCloseHallDoors(h, false)
	case "clanhallteleportself":
		if loc, ok := s.hallSpawnLoc(h.zone, data.ZoneSpawnNormal); ok {
			s.teleportAround(p, loc, 0)
		}
	}
	s.gmHallPage(p, h)
}

// gmHallPage is showClanHallPage or sendSiegableHallPage.
func (s *Server) gmHallPage(p *Player, h *ClanHall) {
	name := "admin/clanhall.htm"
	if h.siege != nil {
		name = "admin/siegablehall.htm"
	}
	owner := "None"
	if o := h.Owner(); o != nil {
		owner = o.Name
	} else if h.siege != nil {
		owner = "No Owner"
	}
	s.sendPage(p, s.pageFile(p, 0, name).replace("%clanhallName%", h.Name).replace("%clanhallId%", idText(h.ID)).
		replace("%clanhallOwner%", owner))
}

// gmHallSiege is AdminCHSiege.onCommand.
func (s *Server) gmHallSiege(p *Player, cmd string, args []string) {
	h := s.gmHallArg(p, args)
	if h == nil {
		return
	}
	hs := h.siege
	if hs == nil {
		p.SendMessage("The given hall does not have any attached siege!")
		return
	}
	target, _ := p.Target.(*Player)
	clanArg := func() *Clan {
		if len(args) > 1 {
			return s.ClanByName(args[1])
		}
		if target != nil {
			return target.Clan()
		}
		return nil
	}
	switch cmd {
	case "chsiege_startSiege":
		if hs.status == residence.HallRunning {
			p.SendMessage("The requested clan hall is alredy in siege!")
			break
		}
		if owner := h.Owner(); owner != nil {
			s.freeHall(h)
			if !hs.reg.IsAttacker(owner.ID) {
				hs.reg.AddAttacker(owner.ID)
			}
		}
		s.hallStartSiege(h)
	case "chsiege_endsSiege":
		if hs.status != residence.HallRunning {
			p.SendMessage("The requested clan hall is not in siege!")
			break
		}
		s.hallEndSiege(h)
	case "chsiege_setSiegeDate":
		s.gmHallSiegeDate(p, h, args[1:])
	case "chsiege_addAttacker":
		cl := clanArg()
		switch {
		case hs.status == residence.HallRunning:
			p.SendMessage("The clan hall is in siege, cannot add attackers now.")
		case cl == nil:
			p.SendMessage("Your target does not have any clan!")
		case hs.reg.IsAttacker(cl.ID):
			p.SendMessage("Your target's clan is alredy participating!")
		default:
			hs.reg.AddAttacker(cl.ID)
			h.save()
		}
	case "chsiege_removeAttacker":
		cl := clanArg()
		switch {
		case hs.status == residence.HallRunning:
			p.SendMessage("The clan hall is in siege, cannot remove attackers now.")
		case cl == nil || !hs.reg.IsAttacker(cl.ID):
			p.SendMessage("Your target's clan is not participating!")
		default:
			hs.reg.Remove(cl.ID)
			h.save()
		}
	case "chsiege_clearAttackers":
		if hs.status == residence.HallRunning {
			p.SendMessage("The requested hall is in siege right now, cannot clear attacker list!")
			break
		}
		hs.reg.Clear()
		h.save()
	case "chsiege_listAttackers":
		s.showHallSiegeInfo(p, h)
	case "chsiege_forwardSiege":
		s.cancelHallTask(h)
		switch hs.status {
		case residence.HallRegistering:
			s.hallPrepareOwner(h)
		case residence.HallWaitingBattle:
			s.hallStartSiege(h)
		case residence.HallRunning:
			s.hallEndSiege(h)
		}
	}
	s.gmHallPage(p, h)
}

// gmHallSiegeDate is admin_chsiege_setSiegeDate DD-MM-YYYY;HH:MM: a date in
// the future while the hall registers.
func (s *Server) gmHallSiegeDate(p *Player, h *ClanHall, args []string) {
	if h.siege.status != residence.HallRegistering {
		p.SendMessage("Cannot change siege date while hall is in siege")
		return
	}
	if len(args) == 0 {
		p.SendMessage("The date format is incorrect. Try again.")
		return
	}
	day, clock, ok := strings.Cut(args[0], ";")
	if !ok {
		p.SendMessage("You have to specify this format DD-MM-YYYY;HH:MM")
		return
	}
	t, err := time.ParseInLocation("2-1-2006 15:04", day+" "+clock, s.calendar.Location())
	if err != nil {
		p.SendMessage("Wrong day/month/year gave!")
		return
	}
	if !t.After(s.Now()) {
		p.SendMessage("The given time is in the past!")
		return
	}
	h.siege.nextSiege = t
	p.SendMessage(h.Name + " siege: " + t.String())
	s.scheduleHallPrepare(h)
	h.save()
}
