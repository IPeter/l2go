package game

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/manor"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The castle manor (Plan 14 M11k, Mobius CastleManorManager) on the loop.
// The rules are internal/manor; this file holds the manor's state (the mode
// and every castle's lists), its daily mode changes on the server's
// calendar, the GM page and the saving. The manor managers' and the
// chamberlains' windows and the manor's packets are in manor_npc.go, sowing
// and harvesting in manor_sow.go, the stored form in manor_store.go.
//
// Decisions X4 and X5: every change is saved at once through the saver, the
// mode changes with the treasuries and clan warehouse items they move in one
// transaction; the mode and its next change are stored (Mobius takes the
// mode from the clock at start), and the changes missed while the server
// was down are replayed at start, in order. A server started for the first
// time takes Mobius's mode for the hour.
func init() { registerFeature("residence_manor", setupManor) }

const (
	// manorLoadTimeout bounds reading the manor at startup.
	manorLoadTimeout = 30 * time.Second
	// manorMaxReplay bounds the missed mode changes replayed at start: a
	// year of them. A longer stop starts from the clock, as Mobius does.
	manorMaxReplay = 3 * 366
)

// System messages of the manor.
const (
	manorMsgUpdated       serverpackets.SystemMessageID = 884  // "The manor information has been updated."
	manorMsgNotSufficient serverpackets.SystemMessageID = 935  // "The amount is not sufficient and so the manor is not in operation."
	manorMsgMaintenance   serverpackets.SystemMessageID = 879  // "The manor system is currently under maintenance."
	manorMsgCannotSetUp   serverpackets.SystemMessageID = 1675 // "A manor cannot be set up between 6 a.m. and 8 p.m."
	manorMsgOnlySeedsOf   serverpackets.SystemMessageID = 1605 // "* Here, you can buy only seeds of $s1 Manor."
	manorMsgFailedTrading serverpackets.SystemMessageID = 1491 // "Failed in trading $s2 of crop $s1."
)

// manorFeature is the manor on the Server (Server.manor). Loop-only, except
// store, which New sets.
type manorFeature struct {
	store storage.ManorStore

	enabled  bool
	settings manor.Settings
	table    *manor.Table

	mode  manor.Mode
	next  time.Time // the next mode change, zero while disabled
	timer world.TimerID
	// stored reports that the mode came from the store (else from the clock).
	stored bool

	// castles are the lists of every castle, by castle ID.
	castles map[int32]*manor.Castle
	// seeds are the monsters a seed was chosen or sown on (manor_sow.go).
	seeds map[*Npc]*npcSeed
}

// setupManor builds the seed table, reads the stored manor, registers the
// packets and the GM command, and starts the mode changes once the loop runs.
// It runs after the castles (residence).
func setupManor(s *Server) error {
	f := &s.manor
	cfg := s.gs.Manor
	if cfg == (config.Manor{}) {
		cfg = config.DefaultManor() // a config made in code without the section
	}
	f.enabled = cfg.AllowManor
	f.settings = manor.Settings{RefreshHour: cfg.RefreshTime, RefreshMin: cfg.RefreshMin, ApproveHour: cfg.ApproveTime,
		ApproveMin: cfg.ApproveMin, MaintenanceMin: cfg.MaintenanceMin, DropRate: max(cfg.RateDropManor, 1)}
	var seeds []data.Seed
	if s.data != nil {
		seeds = s.data.Seeds
	}
	f.table = manor.NewTable(seeds, func(id int) (int64, bool) {
		t, ok := s.data.ItemTemplate(id)
		if !ok {
			return 0, false
		}
		return int64(t.Price), true
	}, f.settings.DropRate)
	f.castles = make(map[int32]*manor.Castle)
	f.seeds = make(map[*Npc]*npcSeed)
	for _, c := range s.Castles() {
		f.castles[c.ID] = &manor.Castle{}
	}
	err := errors.Join(
		s.RegisterEx(clientpackets.ExOpRequestProcureCropList, s.handleRequestProcureCropList),
		s.RegisterEx(clientpackets.ExOpRequestSetSeed, s.handleRequestSetSeed),
		s.RegisterEx(clientpackets.ExOpRequestSetCrop, s.handleRequestSetCrop),
		s.Register(clientpackets.OpRequestBuySeed, s.handleRequestBuySeed),
		s.RegisterGMCommand("manor", storage.AccessLevelAdmin, s.gmManor),
	)
	if err != nil {
		return err
	}
	if !f.enabled {
		f.mode = manor.Disabled
		s.log.Info("game: the manor system is deactivated")
		return nil
	}
	if err := s.loadManor(); err != nil {
		return err
	}
	s.log.Info("game: manor loaded", "seeds", f.table.Len(), "mode", f.mode.String(), "next", f.next)
	s.loop.After(0, s.startManor)
	return nil
}

// loadManor reads the stored lists and mode; without a stored mode the
// manor takes Mobius's mode for the hour.
func (s *Server) loadManor() error {
	f := &s.manor
	var (
		st     storage.ManorState
		ok     bool
		stored []storage.ManorCastle
	)
	if f.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), manorLoadTimeout)
		defer cancel()
		var err error
		if st, ok, stored, err = f.store.LoadManor(ctx); err != nil {
			return fmt.Errorf("game: load the manor: %w", err)
		}
	}
	for _, doc := range stored {
		mc := f.castles[doc.CastleID]
		if mc == nil {
			s.log.Warn("game: manor lists of an unknown castle, ignored", "castle", doc.CastleID)
			continue
		}
		c, unknown := manorCastleOf(doc, f.table)
		if len(unknown) > 0 {
			s.log.Warn("game: unknown seeds or crops in the manor, left out", "castle", doc.CastleID, "items", unknown)
		}
		*mc = c
	}
	if mode := manor.Mode(st.Mode); ok && mode >= manor.Modifiable && mode <= manor.Approved && st.NextChange > 0 {
		f.mode, f.next, f.stored = mode, time.UnixMilli(st.NextChange), true
		return nil
	}
	now := s.Now()
	t := s.calendar.In(now)
	f.mode = manor.InitialMode(t.Hour(), t.Minute(), f.settings)
	f.next = manor.NextChange(f.mode, now, s.calendar, f.settings)
	return nil
}

// startManor replays the mode changes missed while the server was down, in
// order, saves the mode the first time and schedules the next change.
func (s *Server) startManor() {
	f := &s.manor
	now := s.Now()
	n := 0
	for ; n < manorMaxReplay && !f.next.After(now); n++ {
		s.manorChangeMode(f.next)
	}
	if n == manorMaxReplay {
		t := s.calendar.In(now)
		f.mode = manor.InitialMode(t.Hour(), t.Minute(), f.settings)
		f.next = manor.NextChange(f.mode, now, s.calendar, f.settings)
		s.log.Warn("game: too many missed manor changes, the manor starts from the clock", "mode", f.mode.String())
	}
	if n > 0 {
		s.log.Info("game: missed manor changes replayed", "changes", n, "mode", f.mode.String(), "next", f.next)
	}
	if !f.stored || n == manorMaxReplay {
		s.saveManorState()
	}
	s.scheduleManorChange()
}

// scheduleManorChange arms the timer of the next mode change, replacing one
// still pending.
func (s *Server) scheduleManorChange() {
	f := &s.manor
	if f.timer != 0 {
		s.loop.Cancel(f.timer)
		f.timer = 0
	}
	if f.mode == manor.Disabled || f.next.IsZero() {
		return
	}
	f.timer = s.loop.After(f.next.Sub(s.Now()), func() {
		f.timer = 0
		s.manorChangeMode(f.next)
		s.scheduleManorChange()
	})
}

// ManorMode returns the manor's mode. Loop-only.
func (s *Server) ManorMode() manor.Mode { return s.manor.mode }

// ManorNextChange returns when the manor's mode changes next, zero while it
// is disabled. Loop-only.
func (s *Server) ManorNextChange() time.Time { return s.manor.next }

// manorChangeMode is CastleManorManager.changeMode at the time at (the
// change's scheduled time): the castles with an owner are processed, the
// next change is set, and everything that changed is saved in one
// transaction.
func (s *Server) manorChangeMode(at time.Time) {
	f := &s.manor
	var b storage.Batch
	switch f.mode {
	case manor.Approved:
		f.mode = manor.Maintenance
		for _, c := range s.Castles() {
			owner := c.Owner()
			if owner == nil {
				continue
			}
			s.manorMaintenance(c, owner, &b)
		}
		b.ManorCastles = s.manorDocs()
	case manor.Maintenance:
		for _, c := range s.Castles() {
			if owner := c.Owner(); owner != nil {
				manorTellLeader(owner, manorMsgUpdated)
			}
		}
		f.mode = manor.Modifiable
	case manor.Modifiable:
		f.mode = manor.Approved
		for _, c := range s.Castles() {
			owner := c.Owner()
			if owner == nil {
				continue
			}
			wh := s.ClanWarehouse(owner)
			treasury, cleared := f.table.Approve(f.castles[c.ID], c.treasury, func(slots int) bool { return wh.ValidateCapacity(int64(slots)) })
			if cleared {
				manorTellLeader(owner, manorMsgNotSufficient)
			}
			if treasury != c.treasury {
				c.treasury = treasury
				b.Castles = append(b.Castles, c.document())
			}
		}
		b.ManorCastles = s.manorDocs()
	default:
		return
	}
	f.next = manor.NextChange(f.mode, at, s.calendar, f.settings)
	b.ManorState = s.manorStateDoc()
	s.SaveJoint(nil, b)
}

// manorMaintenance is the maintenance of one owned castle: the mature crops
// go into the owner's clan warehouse and the unspent money into the
// treasury, both added to b.
func (s *Server) manorMaintenance(c *Castle, owner *Clan, b *storage.Batch) {
	f := &s.manor
	crops, treasury := f.table.Maintenance(f.castles[c.ID], c.treasury, s.rnd)
	if len(crops) > 0 {
		wh := s.ClanWarehouse(owner)
		for _, h := range crops {
			if _, err := wh.Create(int32(h.MatureID), h.Count); err != nil { //nolint:gosec // item IDs
				s.log.Error("game: cannot put mature crops into a clan warehouse", "clan", owner.Name, "item", h.MatureID, "count", h.Count, "err", err)
			}
		}
		wh.TakeUpdates()
		if ch := wh.Tracker().Take(); !noItemChanges(ch) {
			b.Items = append(b.Items, ownerItemChanges(owner.ID, ch))
		}
	}
	if treasury != c.treasury {
		c.treasury = treasury
		b.Castles = append(b.Castles, c.document())
	}
}

// manorTellLeader sends a message to the clan's leader when it plays.
func manorTellLeader(c *Clan, id serverpackets.SystemMessageID) {
	if l := c.Leader(); l != nil && l.Online() {
		l.Player().SendSystemMessage(id)
	}
}

// manorDocs returns every castle's stored lists, in castle order (Mobius
// storeMe writes them all).
func (s *Server) manorDocs() []storage.ManorCastle {
	out := make([]storage.ManorCastle, 0, len(s.manor.castles))
	for _, c := range s.Castles() {
		if mc := s.manor.castles[c.ID]; mc != nil {
			out = append(out, manorCastleDoc(c.ID, mc))
		}
	}
	return out
}

// manorStateDoc returns the stored mode.
func (s *Server) manorStateDoc() *storage.ManorState {
	f := &s.manor
	st := storage.ManorState{Mode: int32(f.mode)} //nolint:gosec // 0-3
	if !f.next.IsZero() {
		st.NextChange = f.next.UnixMilli()
	}
	return &st
}

// saveManorState saves the mode alone.
func (s *Server) saveManorState() {
	s.SaveJoint(nil, storage.Batch{ManorState: s.manorStateDoc()})
}

// saveManorCastle saves a castle's lists.
func (s *Server) saveManorCastle(id int32) {
	if mc := s.manor.castles[id]; mc != nil {
		s.SaveJoint(nil, storage.Batch{ManorCastles: []storage.ManorCastle{manorCastleDoc(id, mc)}})
	}
}

// manorCastle returns a castle's lists, nil for an unknown castle.
func (s *Server) manorCastle(id int32) *manor.Castle { return s.manor.castles[id] }

// manorCastleRemoved is resetManorData: a castle that lost its owner sells
// and buys nothing (Castle.updateOwnerInDB). Loop-only.
func (s *Server) manorCastleRemoved(c *Castle) {
	mc := s.manor.castles[c.ID]
	if !s.manor.enabled || mc == nil {
		return
	}
	mc.Reset()
	s.saveManorCastle(c.ID)
}

// ----- the GM page -----

// gmManor is AdminManor: the mode, its next change and each castle's cost of
// the running and the next period.
func (s *Server) gmManor(p *Player, _ []string) {
	f := &s.manor
	change := "Disabled"
	if f.mode != manor.Disabled && !f.next.IsZero() {
		change = s.calendar.In(f.next).Format("02/01 15:04:05")
	}
	var b strings.Builder
	for _, c := range s.Castles() {
		mc := f.castles[c.ID]
		if mc == nil {
			continue
		}
		b.WriteString("<tr><td>Name:</td><td><font color=008000>" + c.Name + "</font></td></tr>")
		b.WriteString("<tr><td>Current period cost:</td><td><font color=FF9900>" + formatAdena(f.table.Cost(mc, false)) + " Adena</font></td></tr>")
		b.WriteString("<tr><td>Next period cost:</td><td><font color=FF9900>" + formatAdena(f.table.Cost(mc, true)) + " Adena</font></td></tr>")
		b.WriteString("<tr><td><font color=808080>--------------------------</font></td><td><font color=808080>--------------------------</font></td></tr>")
	}
	pg := s.pageFile(p, 0, "admin/manor.htm").replace("%status%", f.mode.String()).replace("%change%", change)
	s.sendPage(p, pg.replace("%castleInfo%", b.String()))
}

// formatAdena is Mobius FormatUtil.formatAdena: the amount with commas.
func formatAdena(v int64) string {
	digits := strconv.FormatInt(v, 10)
	neg := strings.HasPrefix(digits, "-")
	digits = strings.TrimPrefix(digits, "-")
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return b.String()
}
