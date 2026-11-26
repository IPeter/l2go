package game

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/IPeter/l2go/internal/fishing"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
)

// The weekly fishing championship (Mobius FishingChampionshipManager) on
// the loop: restored at start (a missed Tuesday 19:00 ends the week at
// once), ended by a loop timer at the end the calendar gives
// (fishing.EndOfWeek), and saved whole, in the background, whenever it
// changes (decision X4): a week's end, a ranked fish, a claimed prize.

const (
	// fishingLoadTimeout bounds reading the championship at startup.
	fishingLoadTimeout = 30 * time.Second
	// fishingWriteTimeout bounds one write of it.
	fishingWriteTimeout = 10 * time.Second
	// fishingRefreshDelay is how long the running ranking stays as shown
	// (Mobius needRefresh, 60 s).
	fishingRefreshDelay = 60 * time.Second
)

// The championship pages (data/html/fisherman/championship/).
const (
	fishPageWinners     = "fisherman/championship/fish_event001.htm"
	fishPageRanking     = "fisherman/championship/fish_event002.htm"
	fishPageRefreshing  = "fisherman/championship/fish_event003.htm"
	fishPageReward      = "fisherman/championship/fish_event_reward001.htm"
	fishPageNoReward    = "fisherman/championship/no_fish_event_reward001.htm"
	fishPageNoChampion  = "fisherman/championship/no_fish_event001.htm"
	fishChampionshipLog = "FishingChampionshipManager : new event period start."
)

// setupFishingChampionship restores the championship (Mobius's constructor)
// and schedules the end of the week.
func (s *Server) setupFishingChampionship() error {
	f := &s.fishing
	f.writer = &fishingWriter{log: s.log}
	var stored storage.FishingChampionship
	if f.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), fishingLoadTimeout)
		defer cancel()
		var err error
		if stored, err = f.store.LoadFishingChampionship(ctx); err != nil {
			return err
		}
	}
	champ, finished := fishing.NewChampionship(f.cfg.champ, championshipFromStore(stored), s.Now(), s.calendar)
	f.champ = champ
	if finished {
		s.log.Info("game: " + fishChampionshipLog)
		s.saveChampionship()
	}
	s.scheduleChampionshipEnd()
	return nil
}

// scheduleChampionshipEnd arms the timer of the week's end, replacing one
// still pending (a GM ended the week early).
func (s *Server) scheduleChampionshipEnd() {
	f := &s.fishing
	if f.champTimer != 0 {
		s.loop.Cancel(f.champTimer)
	}
	f.champTimer = s.loop.After(f.champ.End().Sub(s.Now()), s.finishChampionship)
}

// finishChampionship is Mobius finishChamp.run: the week's fishers become
// the winners, the next end is set, saved and scheduled.
func (s *Server) finishChampionship() {
	f := &s.fishing
	f.champ.Finish(s.calendar)
	s.saveChampionship()
	s.log.Info("game: "+fishChampionshipLog, "end", f.champ.End())
	s.scheduleChampionshipEnd()
}

// championshipNewFish is Mobius newFish for a landed fish: "You caught a
// fish $s1 in length." and, when it enters the ranking, "Because of the
// size of fish caught, you will be registered in the ranking.".
func (s *Server) championshipNewFish(p *Player, lureID int) {
	c, ok := s.fishing.champ.NewFish(s.fishing.rnd, p.Name(), lureID)
	if !ok {
		return
	}
	p.SendSystemMessage(serverpackets.SystemMessageID(fishing.MsgCaughtFishLength), serverpackets.TextParam(fishing.JavaDouble(c.Length)))
	if c.Registered {
		sendFishingMessage(p, fishing.MsgRegisteredInRanking, 0)
		s.saveChampionship()
	}
}

// prizePage fills the prize placeholders of a championship page.
func (s *Server) prizePage(pg *htmlPage) *htmlPage {
	cfg := s.fishing.champ.Config()
	name := ""
	if t, ok := s.data.ItemTemplate(cfg.RewardItem); ok {
		name = t.Name
	}
	pg.replace("%prizeItem%", name)
	for i, key := range []string{"%prizeFirst%", "%prizeTwo%", "%prizeThree%", "%prizeFour%", "%prizeFive%"} {
		pg.replace(key, strconv.Itoa(cfg.Rewards[i]))
	}
	return pg
}

// showChampScreen is Mobius showChampScreen: last week's winners, the
// prizes and the minutes until the week ends.
func (s *Server) showChampScreen(p *Player, n *Npc) {
	champ := s.fishing.champ
	pg := s.pageFile(p, 0, fishPageWinners).replace("%TABLE%", champ.WinnerTable())
	s.prizePage(pg)
	pg.replace("%refresh%", strconv.FormatInt(champ.MinutesLeft(s.Now()), 10))
	pg.replace("%objectId%", strconv.Itoa(int(n.ID())))
	s.sendPage(p, pg)
}

// showMidResult is Mobius showMidResult: the running week's ranking, which
// is refreshed at most once a minute ("being updated" page meanwhile).
func (s *Server) showMidResult(p *Player) {
	champ := s.fishing.champ
	table, stale := champ.MidResult()
	if stale {
		s.sendPage(p, s.pageFile(p, 0, fishPageRefreshing))
		s.loop.After(fishingRefreshDelay, champ.Refreshed)
		return
	}
	s.sendPage(p, s.prizePage(s.pageFile(p, 0, fishPageRanking).replace("%TABLE%", table)))
}

// handleRequestExFishRanking handles RequestExFishRanking (D0 1F): the
// running ranking when the championship is on.
func (s *Server) handleRequestExFishRanking(p *Player, _ []byte) {
	if s.fishing.champ.Config().Enabled {
		s.showMidResult(p)
	}
}

// claimFishingPrize is the fisherman's FishingReward: a winner gets the
// prize of each unclaimed rank (Mobius getReward), with its page; anyone
// else the "no reward" page.
func (s *Server) claimFishingPrize(p *Player, n *Npc) {
	champ := s.fishing.champ
	if !champ.IsWinner(p.Name()) {
		s.sendPage(p, s.pageFile(p, n.ID(), fishPageNoReward))
		return
	}
	counts := champ.Reward(p.Name())
	for _, count := range counts {
		if _, err := s.AddItem(p, int32(champ.Config().RewardItem), int64(count), -1, true, false); err != nil { //nolint:gosec // an item ID
			s.log.Warn("game: cannot give a fishing prize", append(p.logArgs(), "count", count, "err", err)...)
		}
		s.sendPage(p, s.pageFile(p, 0, fishPageReward))
	}
	s.saveChampionship()
	s.RequestSave(p)
}

// saveChampionship hands the championship's state to the writer.
func (s *Server) saveChampionship() {
	f := &s.fishing
	if f.store == nil || f.champ == nil {
		return
	}
	f.writer.save(f.store, championshipToStore(f.champ.State()))
}

// saveFishingAtShutdown writes the championship when the server stops
// (Mobius Shutdown: FishingChampionshipManager.shutdown) and waits for it.
func (s *Server) saveFishingAtShutdown() {
	f := &s.fishing
	if f.writer == nil {
		return
	}
	s.saveChampionship()
	ctx, cancel := context.WithTimeout(s.background(), shutdownTimeout)
	defer cancel()
	if err := f.writer.wait(ctx); err != nil {
		s.log.Error("game: the fishing championship is not saved at shutdown", "err", err)
	}
}

// championshipFromStore converts the stored championship.
func championshipFromStore(c storage.FishingChampionship) fishing.ChampionshipState {
	var st fishing.ChampionshipState
	if c.End != 0 {
		st.End = time.UnixMilli(c.End)
	}
	for _, f := range c.Fishers {
		st.Fishers = append(st.Fishers, fishing.Fisher{Name: f.Name, Length: f.Length, Reward: int(f.Rewarded)})
	}
	return st
}

// championshipToStore converts the championship for the store.
func championshipToStore(st fishing.ChampionshipState) storage.FishingChampionship {
	var c storage.FishingChampionship
	if !st.End.IsZero() {
		c.End = st.End.UnixMilli()
	}
	for _, f := range st.Fishers {
		c.Fishers = append(c.Fishers, storage.ChampionshipFisher{Name: f.Name, Length: f.Length, Rewarded: int32(f.Reward)}) //nolint:gosec // 0-2
	}
	return c
}

// fishingWriter writes the championship in the background: a save replaces
// the whole state, so only the latest one waiting is written. Safe for
// concurrent use.
type fishingWriter struct {
	log interface {
		Error(msg string, args ...any)
	}

	mu      sync.Mutex
	next    *storage.FishingChampionship
	store   storage.FishingStore
	running bool
	idle    []chan struct{}
}

// save hands a state to write; it never blocks.
func (w *fishingWriter) save(store storage.FishingStore, c storage.FishingChampionship) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.next, w.store = &c, store
	if !w.running {
		w.running = true
		go w.run()
	}
}

// run writes until nothing waits.
func (w *fishingWriter) run() {
	for {
		w.mu.Lock()
		c, store := w.next, w.store
		w.next = nil
		if c == nil {
			w.running = false
			for _, ch := range w.idle {
				close(ch)
			}
			w.idle = nil
			w.mu.Unlock()
			return
		}
		w.mu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), fishingWriteTimeout)
		if err := store.SaveFishingChampionship(ctx, *c); err != nil {
			w.log.Error("game: cannot save the fishing championship", "err", err)
		}
		cancel()
	}
}

// wait returns once every state handed over so far is written, or with
// ctx's error.
func (w *fishingWriter) wait(ctx context.Context) error {
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
