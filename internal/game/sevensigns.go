package game

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/sevensigns"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The Seven Signs (Plan 14 M11l): Mobius's SevenSigns engine on the loop.
// The rules are internal/sevensigns; this file holds the state, the weekly
// period changes on the calendar (decision X5), the enter-world notices and
// the Seal of Strife's skills, the Record of the Seven Signs (SSQStatus) and
// the dungeons' access. The priests' dialogs are Lua
// (scripts/npcs/seven_signs, sevensigns_script.go), the seals' effects on
// castles are in sevensigns_castle.go, the Festival of Darkness in
// festival*.go and the Dimensional Rift in rift*.go.

func init() { registerFeature("sevensigns", setupSevenSigns) }

// sevenSignsLoadTimeout bounds reading the state at startup.
const sevenSignsLoadTimeout = 30 * time.Second

// Item handlers and player variables of the Seven Signs.
const (
	// handlerSevenSignsRecord is the Record of the Seven Signs' handler
	// (Mobius SevenSignsRecord): the record's first page.
	handlerSevenSignsRecord = "SevenSignsRecord"
	// varIn7sDungeon keeps Mobius's characters.isin7sdungeon: the player
	// went into a catacomb or necropolis through a dungeon gatekeeper.
	varIn7sDungeon = "isIn7sDungeon"
)

// sevenSignsFeature is the Seven Signs on the Server (Server.sevenSigns).
// Loop-only, except store and data, which New sets.
type sevenSignsFeature struct {
	store storage.SevenSignsStore
	data  *data.SevenSigns
	cfg   config.SevenSigns

	state *sevensigns.State
	// next is when the period changes next (Mobius _nextPeriodChange),
	// lastSave the last save (Mobius _lastSave).
	next, lastSave time.Time
	timer          world.TimerID
	dirty          map[int32]bool
	writer         *sevenSignsWriter

	festival festivalState // festival.go
	rift     riftState     // rift.go
	spawns   autoSpawnState
}

// setupSevenSigns loads the state, registers the packets, hooks and
// commands, and starts the period calendar once the loop runs.
func setupSevenSigns(s *Server) error {
	f := &s.sevenSigns
	f.cfg = s.gs.SevenSigns
	f.writer = &sevenSignsWriter{log: s.log}
	s.OnPlayerEnter(s.sevenSignsEnter)
	s.OnPlayerLeave(s.festivalLeave) // festival.go
	errs := []error{
		s.Register(clientpackets.OpRequestSSQStatus, s.handleRequestSSQStatus),
		s.registerSevenSignsGM(), // sevensigns_gm.go
		s.setupFestival(),        // festival.go
		s.setupRift(),            // rift.go
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}
	var stored storage.SevenSigns
	if f.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), sevenSignsLoadTimeout)
		defer cancel()
		var err error
		if stored, err = f.store.LoadSevenSigns(ctx); err != nil {
			return err
		}
	}
	status := sevensigns.NewStatus()
	// A new database's date is 1 (seven_signs_status.sql): never a missed
	// change.
	f.lastSave = time.UnixMilli(1)
	if stored.HasStatus {
		status = sevenSignsStatusOf(stored.Status)
		f.lastSave = time.UnixMilli(stored.Status.Date)
		f.next = msTime(stored.Status.NextPeriodChange)
	}
	players := make([]sevensigns.Player, 0, len(stored.Players))
	for _, p := range stored.Players {
		players = append(players, sevenSignsPlayerOf(p))
	}
	f.state = sevensigns.NewState(status, players)
	s.restoreFestival(stored) // festival.go
	s.loop.After(0, s.startSevenSigns)
	return nil
}

// startSevenSigns is the end of Mobius's SevenSigns constructor: the period
// changes at once when its change passed while the server was off (by the
// saved next change, decision X5, else Mobius's isNextPeriodChangeInPast),
// otherwise at its time; the NPCs spawn and the festival starts.
func (s *Server) startSevenSigns() {
	f := &s.sevenSigns
	now := s.Now()
	st := f.state
	var missed bool
	stored := f.next
	if !stored.IsZero() {
		missed = !stored.After(now)
	} else {
		missed = sevensigns.ChangeMissed(s.calendar, now, f.lastSave, st.Period)
	}
	f.next = now
	var delay time.Duration
	if !missed {
		if !stored.IsZero() {
			f.next = stored
		} else {
			f.next = sevensigns.NextPeriodChange(s.calendar, now, st.Period)
		}
		delay = f.next.Sub(now)
	}
	s.log.Info("game: seven signs loaded", "period", st.Period.Name(), "cycle", st.Cycle,
		"leader", st.HighestCabal().Name(), "next", f.next, "missed", missed)
	s.spawnSevenSignsNpcs()
	if !st.IsSealValidation() {
		s.startFestivalManager() // festival.go
	}
	s.cancelTimer(&f.timer)
	f.timer = s.loop.After(delay, s.sevenSignsPeriodChange)
}

// sevenSignsPeriodChange is Mobius SevenSignsPeriodChange.run: the period
// that ended decides what happens, everything is saved, the losers leave
// the dungeons, the NPCs follow the new period, and the next change is
// scheduled.
func (s *Server) sevenSignsPeriodChange() {
	f := &s.sevenSigns
	f.timer = 0
	st := f.state
	ended := st.Period
	st.Period = ended.Next()
	switch ended {
	case sevensigns.PeriodRecruiting:
		s.startFestivalManager()
		s.broadcastSysMsg(sevensigns.MsgCompetitionBegunVisit)
	case sevensigns.PeriodCompetition:
		s.broadcastSysMsg(sevensigns.MsgCompetitionEnded)
		winner := st.HighestCabal()
		s.stopFestivalManager()
		s.rewardFestivalHighestRanked()
		owners := st.CalcNewSealOwners()
		for _, seal := range sevensigns.Seals {
			if msg := sevensigns.SealObtainedMsg(seal, owners[seal]); msg != 0 {
				s.broadcastSysMsg(msg)
			}
		}
		s.validateCastleTaxes(owners[sevensigns.SealStrife])
		switch winner {
		case sevensigns.CabalDawn:
			s.broadcastSysMsg(sevensigns.MsgDawnWon)
		case sevensigns.CabalDusk:
			s.broadcastSysMsg(sevensigns.MsgDuskWon)
		}
		st.PreviousWinner = winner
		s.resetCastleTickets()
	case sevensigns.PeriodResults:
		for _, seal := range sevensigns.Seals {
			s.log.Info("game: seven signs seal", "seal", seal.Name(false), "owner", st.SealOwner(seal).Name())
		}
		s.giveStrifeSkills()
		s.broadcastSysMsg(sevensigns.MsgValidationBegun)
		s.log.Info("game: seven signs competition won", "cabal", st.PreviousWinner.Name(), "score", st.CurrentScore(st.PreviousWinner))
	case sevensigns.PeriodSealValidation:
		s.broadcastSysMsg(sevensigns.MsgValidationEnded)
		s.removeStrifeSkills()
		st.ResetPlayers()
		st.ResetSeals()
		st.Cycle++
		s.resetFestival()
		st.DawnStoneScore, st.DuskStoneScore = 0, 0
		st.DawnFestivalScore, st.DuskFestivalScore = 0, 0
	}
	f.next = sevensigns.NextPeriodChange(s.calendar, f.next, st.Period)
	s.saveSevenSigns(true, true)
	s.teleLosingCabalFromDungeons()
	s.spawnSevenSignsNpcs()
	s.log.Info("game: seven signs period began", "period", st.Period.Name(), "next", f.next)
	f.timer = s.loop.After(f.next.Sub(s.Now()), s.sevenSignsPeriodChange)
}

// changeSevenSignsPeriodNow runs the next period change at once (//ssq_change).
func (s *Server) changeSevenSignsPeriodNow() {
	f := &s.sevenSigns
	if f.state == nil {
		return
	}
	s.cancelTimer(&f.timer)
	f.next = s.Now()
	s.sevenSignsPeriodChange()
}

// ----- players -----

// SevenSignsCabal returns the player's cabal (Mobius getPlayerCabal).
// Loop-only.
func (s *Server) SevenSignsCabal(p *Player) sevensigns.Cabal {
	if s.sevenSigns.state == nil {
		return sevensigns.CabalNull
	}
	return s.sevenSigns.state.PlayerCabal(p.ObjectID())
}

// in7sDungeon is Player.isIn7sDungeon.
func (p *Player) in7sDungeon() bool {
	v, _ := p.Var(varIn7sDungeon)
	return v == "1"
}

// setIn7sDungeon is Player.setIn7sDungeon, saved with the character.
func (p *Player) setIn7sDungeon(in bool) {
	if p.in7sDungeon() == in {
		return
	}
	v := "0"
	if in {
		v = "1"
	}
	p.SetVar(varIn7sDungeon, v)
}

// leaveDungeon sends p to town out of a catacomb or necropolis with Mobius's
// message.
func (s *Server) leaveDungeon(p *Player, msg string) {
	s.teleportToTown(p)
	p.setIn7sDungeon(false)
	p.SendMessage(msg)
}

// Messages of the dungeons' oust.
const (
	dungeonOustValidation = "You have been teleported to the nearest town due to the beginning of the Seal Validation period."
	dungeonOustNoCabal    = "You have been teleported to the nearest town because you have not signed for any cabal."
)

// teleLosingCabalFromDungeons is teleLosingCabalFromDungeons, at a period
// change.
func (s *Server) teleLosingCabalFromDungeons() {
	st := s.sevenSigns.state
	winner := st.HighestCabal()
	msg := dungeonOustNoCabal
	if st.IsSealValidation() || st.IsResults() {
		msg = dungeonOustValidation
	}
	var out []*Player
	for p := range s.Players() {
		if p.state != stateInGame || p.IsGM() || !p.in7sDungeon() {
			continue
		}
		rec, has := st.Player(p.ObjectID())
		if sevensigns.MustLeaveDungeon(st.Period, rec.Cabal, has, winner) {
			out = append(out, p)
		}
	}
	for _, p := range out {
		s.leaveDungeon(p, msg)
	}
}

// sevenSignsEnter is what EnterWorld and Player.onPlayerEnter do for the
// Seven Signs: the Seal of Strife's skill, the period's notice, and the oust
// from a dungeon the player may no longer be in.
func (s *Server) sevenSignsEnter(p *Player) {
	st := s.sevenSigns.state
	if st == nil {
		return
	}
	cabal := st.PlayerCabal(p.ObjectID())
	if st.IsSealValidation() && st.SealOwner(sevensigns.SealStrife) != sevensigns.CabalNull {
		if id := sevensigns.StrifeSkill(st.SealOwner(sevensigns.SealStrife), cabal); id != 0 {
			s.addStrifeSkill(p, id)
		}
	} else {
		s.takeStrifeSkills(p)
	}
	p.SendSystemMessage(serverpackets.SystemMessageID(sevensigns.PeriodMsg(st.Period))) //nolint:gosec // message IDs
	if !p.IsGM() && p.in7sDungeon() && sevensigns.MustLeaveDungeonOnEnter(st.Period, cabal, st.HighestCabal()) {
		msg := dungeonOustNoCabal
		if st.IsSealValidation() || st.IsResults() {
			msg = dungeonOustValidation
		}
		s.leaveDungeon(p, msg)
	}
}

// addStrifeSkill adds a Seal of Strife skill, not stored, without packets
// (Player.addSkill).
func (s *Server) addStrifeSkill(p *Player, id int) {
	if c := s.skills.catalog; c != nil {
		if sk, ok := c.Skill(id, 1); ok {
			s.AddSkill(p, sk, false)
		}
	}
}

// takeStrifeSkills removes both Seal of Strife skills.
func (s *Server) takeStrifeSkills(p *Player) {
	s.RemoveSkill(p, sevensigns.VictorOfWarSkill, false, true)
	s.RemoveSkill(p, sevensigns.VanquishedOfWarSkill, false, true)
}

// giveStrifeSkills is giveCPMult: every player online with a cabal gets the
// Seal of Strife's skill.
func (s *Server) giveStrifeSkills() {
	st := s.sevenSigns.state
	owner := st.SealOwner(sevensigns.SealStrife)
	for p := range s.Players() {
		cabal := st.PlayerCabal(p.ObjectID())
		if cabal == sevensigns.CabalNull {
			continue
		}
		// Mobius gives the Vanquished skill also when nobody owns the seal.
		id := sevensigns.VanquishedOfWarSkill
		if cabal == owner {
			id = sevensigns.VictorOfWarSkill
		}
		s.addStrifeSkill(p, id)
	}
}

// removeStrifeSkills is removeCPMult.
func (s *Server) removeStrifeSkills() {
	for p := range s.Players() {
		s.takeStrifeSkills(p)
	}
}

// ----- the record -----

// handleRequestSSQStatus answers RequestSSQStatus (0xC7) with the page of
// the Record of the Seven Signs; the fourth page (the predictions) is not
// shown in the results and seal validation periods (Mobius
// RequestSSQStatus).
func (s *Server) handleRequestSSQStatus(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestSSQStatus(body)
	st := s.sevenSigns.state
	if err != nil || st == nil {
		return
	}
	if (st.IsSealValidation() || st.IsResults()) && req.Page == 4 {
		return
	}
	p.Send(s.ssqStatusPacket(p, req.Page))
}

// useSevenSignsRecord is the Record of the Seven Signs' item handler: the
// record's first page.
func (s *Server) useSevenSignsRecord(p *Player) bool {
	if s.sevenSigns.state == nil {
		return false
	}
	p.Send(s.ssqStatusPacket(p, 1))
	return true
}

// ssqStatusPacket builds SSQStatus's page for p (Mobius SSQStatus).
func (s *Server) ssqStatusPacket(p *Player, page int) []byte {
	st := s.sevenSigns.state
	pkt := serverpackets.SSQStatus{Page: int8(page), Period: int8(st.Period)} //nolint:gosec // pages and periods are small
	switch page {
	case 1:
		r := st.Page1(p.ObjectID())
		pkt.Cycle = int32(r.Cycle)                                    //nolint:gosec // small
		pkt.PeriodMsg, pkt.PeriodEndMsg = int32(r.PeriodMsg), int32(r.PeriodEndMsg) //nolint:gosec // message IDs
		pkt.Cabal, pkt.Seal = int8(r.Cabal), int8(r.Seal)             //nolint:gosec // 0-3
		pkt.StonesTurnedIn, pkt.AncientAdenaToCollect = int32(r.StonesTurnedIn), int32(r.AncientAdenaToCollect) //nolint:gosec // counts
		pkt.DuskStoneScore, pkt.DuskFestivalScore, pkt.DuskTotal = int32(r.DuskStone), int32(r.DuskFestival), int32(r.DuskTotal) //nolint:gosec // scores
		pkt.DuskPercent = int8(r.DuskPercent)                                                                                    //nolint:gosec // 0-100
		pkt.DawnStoneScore, pkt.DawnFestivalScore, pkt.DawnTotal = int32(r.DawnStone), int32(r.DawnFestival), int32(r.DawnTotal) //nolint:gosec // scores
		pkt.DawnPercent = int8(r.DawnPercent)                                                                                    //nolint:gosec // 0-100
	case 2:
		pkt.Festivals = s.festivalRecordPage() // festival.go
	case 3:
		for _, v := range st.Page3() {
			pkt.Seals = append(pkt.Seals, serverpackets.SSQSealVote{Seal: int8(v.Seal), Owner: int8(v.Owner), //nolint:gosec // small
				DuskPercent: int8(v.DuskPercent), DawnPercent: int8(v.DawnPercent)}) //nolint:gosec // 0-100
		}
	case 4:
		winner, preds := st.Page4()
		pkt.Winner = int8(winner) //nolint:gosec // 0-2
		for _, v := range preds {
			pkt.Predictions = append(pkt.Predictions, serverpackets.SSQSealPrediction{Seal: int8(v.Seal), Owner: int8(v.Owner), Msg: int16(v.Msg)}) //nolint:gosec // small
		}
	}
	return serverpackets.NewSSQStatusPacket(pkt)
}

// ----- dungeons -----

// dungeonGatekeeperCommand runs a dungeon gatekeeper's links (Mobius
// DungeonGatekeeper.onBypassFeedback): "necro <id>" and "cata <id>" enter a
// necropolis or a catacomb when the seals allow it, "exit <id>" leaves,
// "goto <id>" moves; the rest goes to the teleporter's handlers.
func (s *Server) dungeonGatekeeperCommand(p *Player, n *Npc, command string) bool {
	st := s.sevenSigns.state
	tokens := strings.FieldsFunc(command, func(r rune) bool { return r == ' ' })
	if len(tokens) == 0 {
		return false
	}
	index := -1
	if len(tokens) > 1 && bypassIsNumeric(tokens[1]) {
		if v, err := strconv.Atoi(tokens[1]); err == nil {
			index = v
		}
	}
	var open bool
	var seal sevensigns.Seal
	var refusal string
	switch word := tokens[0]; {
	case strings.HasPrefix(word, "necro"):
		open, seal, refusal = s.sevenSigns.cfg.OpenNecropolis, sevensigns.SealAvarice, "necro_no.htm"
	case strings.HasPrefix(word, "cata"):
		open, seal, refusal = s.sevenSigns.cfg.OpenCatacombs, sevensigns.SealGnosis, "cata_no.htm"
	case strings.HasPrefix(word, "exit"):
		p.SendActionFailed()
		s.doTeleport(p, n, data.TeleportOther, index)
		p.setIn7sDungeon(false)
		return true
	case strings.HasPrefix(word, "goto"):
		p.SendActionFailed()
		s.doTeleport(p, n, data.TeleportOther, index)
		return true
	default:
		return false
	}
	p.SendActionFailed()
	if st == nil {
		return true
	}
	ok, msg := sevensigns.DungeonAccess(open, st.IsSealValidation(), st.HighestCabal(), st.PlayerCabal(p.ObjectID()), st.SealOwner(seal))
	if msg != 0 {
		p.SendSystemMessage(serverpackets.SystemMessageID(msg)) //nolint:gosec // message IDs
	}
	if !ok {
		s.sendPage(p, s.pageFile(p, n.ID(), sevenSignsHTML+refusal))
		return true
	}
	// Mobius marks the player as in the dungeon whether or not the
	// teleport ran.
	s.doTeleport(p, n, data.TeleportOther, index)
	p.setIn7sDungeon(true)
	return true
}
