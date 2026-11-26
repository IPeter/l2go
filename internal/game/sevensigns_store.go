package game

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/IPeter/l2go/internal/sevensigns"
	"github.com/IPeter/l2go/internal/storage"
)

// Saving the Seven Signs (decision X4): every change hands a snapshot to
// a background writer, which writes it with storage.SevenSignsStore in one
// transaction. Mobius saves every 30 minutes and at period changes
// (AltSevenSignsLazyUpdate); here every change is saved, a period change
// with every player and the festival at once.

// sevenSignsWriteTimeout bounds one write.
const sevenSignsWriteTimeout = 30 * time.Second

// markSevenSignsPlayer records that a player's record changed.
func (s *Server) markSevenSignsPlayer(id int32) {
	f := &s.sevenSigns
	if f.dirty == nil {
		f.dirty = make(map[int32]bool)
	}
	f.dirty[id] = true
}

// saveSevenSigns hands the status, the changed players (all of them with
// all) and, with festival, the festival's rows to the writer. Loop-only.
func (s *Server) saveSevenSigns(all, festival bool) {
	f := &s.sevenSigns
	if f.state == nil {
		return
	}
	f.lastSave = s.Now()
	sv := storage.SevenSignsSave{Status: s.sevenSignsStatusDoc()}
	if all {
		for _, p := range f.state.Players() {
			sv.Players = append(sv.Players, sevenSignsPlayerDoc(p))
		}
	} else {
		for _, id := range sortedKeys(f.dirty) {
			if p, ok := f.state.Player(id); ok {
				sv.Players = append(sv.Players, sevenSignsPlayerDoc(p))
			}
		}
	}
	clear(f.dirty)
	if festival {
		sv.ReplaceFestival = true
		sv.Festival = s.festivalRows()
	}
	if f.store == nil || f.writer == nil {
		return
	}
	f.writer.save(f.store, sv)
}

// sevenSignsStatusDoc is the stored status.
func (s *Server) sevenSignsStatusDoc() storage.SevenSignsStatus {
	f := &s.sevenSigns
	st := f.state.Status
	doc := storage.SevenSignsStatus{
		Cycle: int32(st.Cycle), Period: int32(st.Period), PreviousWinner: int32(st.PreviousWinner), //nolint:gosec // small
		DawnStoneScore: st.DawnStoneScore, DuskStoneScore: st.DuskStoneScore,
		DawnFestivalScore: int32(st.DawnFestivalScore), DuskFestivalScore: int32(st.DuskFestivalScore), //nolint:gosec // scores
		AvariceOwner: int32(st.SealOwners[sevensigns.SealAvarice]), GnosisOwner: int32(st.SealOwners[sevensigns.SealGnosis]), //nolint:gosec // 0-2
		StrifeOwner:      int32(st.SealOwners[sevensigns.SealStrife]),                                                         //nolint:gosec // 0-2
		AvariceDawnScore: int32(st.DawnSealTotals[sevensigns.SealAvarice]), GnosisDawnScore: int32(st.DawnSealTotals[sevensigns.SealGnosis]), //nolint:gosec // counts
		StrifeDawnScore:  int32(st.DawnSealTotals[sevensigns.SealStrife]),                                                                   //nolint:gosec // counts
		AvariceDuskScore: int32(st.DuskSealTotals[sevensigns.SealAvarice]), GnosisDuskScore: int32(st.DuskSealTotals[sevensigns.SealGnosis]), //nolint:gosec // counts
		StrifeDuskScore:  int32(st.DuskSealTotals[sevensigns.SealStrife]),                                                                   //nolint:gosec // counts
		FestivalCycle:    int32(f.festival.book.Cycle),                                                                                           //nolint:gosec // small
		Date:             timeMs(f.lastSave),
		NextPeriodChange: timeMs(f.next),
	}
	for _, b := range f.festival.book.Bonus {
		doc.AccumulatedBonus = append(doc.AccumulatedBonus, int32(b)) //nolint:gosec // ancient adena counts
	}
	return doc
}

// sevenSignsStatusOf converts a stored status.
func sevenSignsStatusOf(d storage.SevenSignsStatus) sevensigns.Status {
	st := sevensigns.Status{Cycle: int(d.Cycle), Period: sevensigns.Period(d.Period), PreviousWinner: sevensigns.Cabal(d.PreviousWinner),
		DawnStoneScore: d.DawnStoneScore, DuskStoneScore: d.DuskStoneScore,
		DawnFestivalScore: int(d.DawnFestivalScore), DuskFestivalScore: int(d.DuskFestivalScore)}
	st.SealOwners[sevensigns.SealAvarice] = sevensigns.Cabal(d.AvariceOwner)
	st.SealOwners[sevensigns.SealGnosis] = sevensigns.Cabal(d.GnosisOwner)
	st.SealOwners[sevensigns.SealStrife] = sevensigns.Cabal(d.StrifeOwner)
	st.DawnSealTotals[sevensigns.SealAvarice] = int(d.AvariceDawnScore)
	st.DawnSealTotals[sevensigns.SealGnosis] = int(d.GnosisDawnScore)
	st.DawnSealTotals[sevensigns.SealStrife] = int(d.StrifeDawnScore)
	st.DuskSealTotals[sevensigns.SealAvarice] = int(d.AvariceDuskScore)
	st.DuskSealTotals[sevensigns.SealGnosis] = int(d.GnosisDuskScore)
	st.DuskSealTotals[sevensigns.SealStrife] = int(d.StrifeDuskScore)
	return st
}

func sevenSignsPlayerDoc(p sevensigns.Player) storage.SevenSignsPlayer {
	return storage.SevenSignsPlayer{CharID: p.CharID, Cabal: p.Cabal, Seal: int32(p.Seal), //nolint:gosec // 0-3
		RedStones: int32(p.RedStones), GreenStones: int32(p.GreenStones), BlueStones: int32(p.BlueStones), //nolint:gosec // counts
		AncientAdena: int64(p.AncientAdena), Contribution: int64(p.Contribution)}
}

func sevenSignsPlayerOf(d storage.SevenSignsPlayer) sevensigns.Player {
	return sevensigns.Player{CharID: d.CharID, Cabal: d.Cabal, Seal: sevensigns.Seal(d.Seal),
		RedStones: int(d.RedStones), GreenStones: int(d.GreenStones), BlueStones: int(d.BlueStones),
		AncientAdena: int(d.AncientAdena), Contribution: int(d.Contribution)}
}

// sevenSignsWriter writes the Seven Signs in the background: the writes
// waiting merge (the last status and festival win, players by ID). Safe for
// concurrent use.
type sevenSignsWriter struct {
	log interface {
		Error(msg string, args ...any)
	}

	mu      sync.Mutex
	store   storage.SevenSignsStore
	pending *storage.SevenSignsSave
	running bool
	idle    []chan struct{}
}

// save hands a write over; it never blocks.
func (w *sevenSignsWriter) save(store storage.SevenSignsStore, sv storage.SevenSignsSave) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.store = store
	if w.pending == nil {
		w.pending = &sv
	} else {
		p := w.pending
		p.Status = sv.Status
		for _, pl := range sv.Players {
			i := slices.IndexFunc(p.Players, func(o storage.SevenSignsPlayer) bool { return o.CharID == pl.CharID })
			if i >= 0 {
				p.Players[i] = pl
			} else {
				p.Players = append(p.Players, pl)
			}
		}
		if sv.ReplaceFestival {
			p.ReplaceFestival, p.Festival = true, sv.Festival
		}
	}
	if !w.running {
		w.running = true
		go w.run()
	}
}

func (w *sevenSignsWriter) run() {
	for {
		w.mu.Lock()
		sv, store := w.pending, w.store
		w.pending = nil
		if sv == nil {
			w.running = false
			for _, ch := range w.idle {
				close(ch)
			}
			w.idle = nil
			w.mu.Unlock()
			return
		}
		w.mu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), sevenSignsWriteTimeout)
		if err := store.SaveSevenSigns(ctx, *sv); err != nil {
			w.log.Error("game: cannot save the seven signs", "err", err)
		}
		cancel()
	}
}

// wait returns once everything handed over so far is written, or with
// ctx's error.
func (w *sevenSignsWriter) wait(ctx context.Context) error {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return nil
	}
	ch := make(chan struct{})
	w.idle = append(w.idle, ch)
	w.mu.Unlock()
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// saveSevenSignsAtShutdown waits for the last writes when the server stops
// (Mobius Shutdown saves the Seven Signs and the festival).
func (s *Server) saveSevenSignsAtShutdown() {
	f := &s.sevenSigns
	if f.writer == nil || f.state == nil {
		return
	}
	s.saveSevenSigns(false, true)
	ctx, cancel := context.WithTimeout(s.background(), shutdownTimeout)
	defer cancel()
	if err := f.writer.wait(ctx); err != nil {
		s.log.Error("game: the seven signs are not saved at shutdown", "err", err)
	}
}
