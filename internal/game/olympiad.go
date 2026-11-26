package game

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/olympiad"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The Grand Olympiad and the heroes (Plan 14, M11h): the rules of
// internal/olympiad on the loop.
//
//   - olympiad.go: the feature, the calendar (Mobius Olympiad: the period,
//     the daily competition, the weekly points, the validation period), the
//     saved state (decision X4: one document, saved at every change);
//   - olympiad_register.go: registration, the waiting lists, the leave hook;
//   - olympiad_manager.go: the match making every 30 s (OlympiadManager);
//   - olympiad_game.go: a match from the countdown to the way back
//     (OlympiadGame, OlympiadGameTask);
//   - olympiad_observe.go: the spectators (observer mode, the arena list);
//   - olympiad_npc.go: the Grand Olympiad Managers' pages (Mobius
//     entity/actor/instance/OlympiadManager);
//   - olympiad_rules.go: what the Olympiad changes elsewhere (attacks, shots,
//     the stadium zones);
//   - olympiad_gm.go: //checkolympiad, //saveolymp, //endolympiad,
//     //givehero;
//   - hero.go, hero_diary.go: the heroes (Mobius Hero), their items,
//     skills, words, fights and diary.

func init() { registerFeature("olympiad", setupOlympiad) }

const (
	// olympiadLoadTimeout bounds reading the Olympiad at startup.
	olympiadLoadTimeout = 30 * time.Second
	// olympiadWriteTimeout bounds one write.
	olympiadWriteTimeout = 10 * time.Second
	// olympiadManagerPeriod is the manager's pause between two rounds
	// (Mobius wait(30000)), olympiadIdlePeriod its pause without nobles
	// (wait(60000)), olympiadStartGap the pause after each game it starts
	// (wait(1000)), olympiadBattleWait the competition end's wait for
	// running battles (Thread.sleep(60000)).
	olympiadManagerPeriod = 30 * time.Second
	olympiadIdlePeriod    = 60 * time.Second
	olympiadStartGap      = time.Second
	olympiadBattleWait    = 60 * time.Second
	// olympiadRegistrationClose is how long before the competition's end
	// registration closes (Mobius getMillisToCompEnd() - 600000).
	olympiadRegistrationClose = 10 * time.Minute
)

// The announcements of the Olympiad (Mobius broadcastToAllOnlinePlayers
// with a text).
const (
	olympiadValidationBegan = "Olympiad Validation Period has began"
	olympiadValidationEnded = "Olympiad Validation Period has ended"
)

// olympiadFeature is the Olympiad's state on the Server (Server.olympiad).
// Loop-only, except store, which New sets, and heroIDs.
type olympiadFeature struct {
	store    storage.OlympiadStore // nil keeps the Olympiad in memory only
	cfg      olympiad.Config
	rnd      olympiad.Rand
	stadiums []data.OlympiadStadium

	state     olympiad.State
	nobles    *olympiad.Nobles
	heroes    *olympiad.Heroes
	unclaimed map[int32]int64 // passes to settle by character
	queues    olympiad.Queues

	// compStart is the competition's start (Mobius _compStart), the end
	// CompPeriod later; inComp is _inCompPeriod.
	compStart time.Time
	inComp    bool
	// battleStarted and gameIsStarted are Mobius's static
	// OlympiadGame._battleStarted and _gameIsStarted, shared by every game.
	battleStarted, gameIsStarted bool

	games      [olympiad.Stadiums]*olyGame
	spectators [olympiad.Stadiums][]*Player
	players    map[*Player]*olyPlayer
	// managerOn: the manager runs (from a competition's start until its
	// games ended after the competition).
	managerOn bool

	endTimer, validationTimer, weeklyTimer    world.TimerID
	compStartTimer, compEndTimer, regEndTimer world.TimerID
	managerTimer, battleWaitTimer             world.TimerID

	// fights and diary are the hero pages' data of the current heroes
	// (Mobius HERO_FIGHTS and HERO_DIARY), loaded in the background.
	fights map[int32][]storage.OlympiadFight
	diary  map[int32][]storage.HeroDiaryEntry

	writer *olympiadWriter
	// heroIDs are the claimed heroes for reads off the loop (the inventory
	// load at character selection). Safe for concurrent use.
	heroIDs atomic.Pointer[map[int32]bool]
}

// olyPlayer is a player's Olympiad state (Mobius _olympiadGameId,
// _inOlympiadMode, _olympiadStart, _olympiadSide and the observer mode).
type olyPlayer struct {
	gameID int // the stadium, -1 for none
	inMode bool
	start  bool
	side   int
	// observer: watching a stadium; last is where to go back (Mobius
	// _lastLoc), hasLast whether it is set.
	observer bool
	last     world.Position
	hasLast  bool
}

// olympiadConfigOf converts the config section.
func olympiadConfigOf(c config.Olympiad) olympiad.Config {
	return olympiad.Config{
		Enabled:                   c.Enabled,
		StartHour:                 c.StartTime,
		StartMinute:               c.Min,
		CompPeriod:                time.Duration(c.CPeriod),
		Battle:                    time.Duration(c.Battle),
		WeeklyPeriod:              time.Duration(c.WPeriod),
		ValidationPeriod:          time.Duration(c.VPeriod),
		StartPoints:               c.StartPoints,
		WeeklyPoints:              c.WeeklyPoints,
		ClassedParticipants:       c.ClassedParticipants,
		NonClassedParticipants:    c.NonClassedParticipants,
		RegDisplay:                c.RegistrationDisplayNumber,
		BattleRewardItem:          c.BattleRewItem,
		ClassedReward:             c.ClassedRewItemCount,
		NonClassedReward:          c.NonClassedRewItemCount,
		CompRewardItem:            c.CompRewItem,
		GPPerPoint:                c.GPPerPoint,
		HeroPoints:                c.HeroPoints,
		MaxPoints:                 c.MaxPoints,
		ShowMonthlyWinners:        c.ShowMonthlyWinners,
		AnnounceGames:             c.AnnounceGames,
		RestrictedItems:           c.RestrictedItems,
		DisableBlessedSpiritShots: c.DisableBlessedSpiritShots,
		EnchantLimit:              c.EnchantLimit,
		LogFights:                 c.LogFights,
		WaitTime:                  c.WaitTime,
		CustomPeriod:              c.UseCustomPeriodSettings,
		Period:                    c.PeriodName(),
		PeriodMultiplier:          c.PeriodMultiplier,
		CompetitionDays:           c.CompetitionDays,
		ClanHeroPoints:            c.ClanHeroPoints,
	}
}

// setupOlympiad loads the stadiums and the saved Olympiad, starts its
// calendar (Mobius Olympiad and Hero constructors) and registers the
// packets, hooks and commands.
func setupOlympiad(s *Server) error {
	f := &s.olympiad
	f.cfg = olympiadConfigOf(s.gs.Olympiad)
	f.rnd = s.rnd
	f.players = make(map[*Player]*olyPlayer)
	f.unclaimed = make(map[int32]int64)
	f.fights = make(map[int32][]storage.OlympiadFight)
	f.diary = make(map[int32][]storage.HeroDiaryEntry)
	f.nobles = &olympiad.Nobles{}
	f.heroes = &olympiad.Heroes{}
	f.writer = &olympiadWriter{log: s.log}
	f.publishHeroes()
	if dir := s.settings.DataDir; dir != "" {
		switch list, err := data.LoadOlympiadStadiums(dir); {
		case err == nil:
			f.stadiums = list
		case errors.Is(err, fs.ErrNotExist):
			s.log.Info("game: no olympiad stadiums, no match can start", "file", filepath.Join(dir, data.OlympiadStadiumsFile))
		default:
			return fmt.Errorf("olympiad stadiums: %w", err)
		}
	}
	// olympiad_register.go, olympiad_game.go, olympiad_observe.go, hero.go
	// and olympiad_gm.go.
	s.OnPlayerLeave(s.olympiadLeave)
	s.OnHit(s.olympiadHit)
	s.SubclassHooks().LeaveOlympiad = s.olympiadLeaveForSubclass
	errs := []error{
		s.RegisterEx(clientpackets.ExOpRequestWriteHeroWords, s.handleWriteHeroWords),
		s.RegisterEx(clientpackets.ExOpRequestOlympiadObserverEnd, s.handleObserverEnd),
		s.RegisterEx(clientpackets.ExOpRequestOlympiadMatchList, s.handleOlympiadMatchList),
		s.registerOlympiadGM(),
	}
	if f.cfg.Enabled {
		errs = append(errs, s.loadOlympiad())
	} else {
		s.log.Info("game: the Olympiad is disabled")
	}
	return errors.Join(errs...)
}

// olyState returns p's Olympiad state, made on first use.
func (s *Server) olyState(p *Player) *olyPlayer {
	st := s.olympiad.players[p]
	if st == nil {
		st = &olyPlayer{gameID: -1, side: -1}
		s.olympiad.players[p] = st
	}
	return st
}

// olyOf returns p's Olympiad state, nil when it has none.
func (s *Server) olyOf(p *Player) *olyPlayer {
	if s == nil || p == nil || s.olympiad.players == nil {
		return nil
	}
	return s.olympiad.players[p]
}

// InOlympiadMode reports whether p fights in a stadium (Mobius
// isInOlympiadMode). Loop-only.
func (p *Player) InOlympiadMode() bool {
	st := p.srv.olyOf(p)
	return st != nil && st.inMode
}

// olympiadGameID is Mobius getOlympiadGameId: the stadium p fights in or
// watches, -1 for none.
func (p *Player) olympiadGameID() int {
	if st := p.srv.olyOf(p); st != nil {
		return st.gameID
	}
	return -1
}

// InObserverMode reports whether p watches an Olympiad stadium (Mobius
// inObserverMode). Loop-only.
func (p *Player) InObserverMode() bool {
	st := p.srv.olyOf(p)
	return st != nil && st.observer
}

// loadOlympiad is Mobius Olympiad.load and init and Hero.init: the saved
// state means a running competition period, a validation period, or the
// next one (a missed period end only sets a new end: Mobius chooses no
// heroes for it).
func (s *Server) loadOlympiad() error {
	f := &s.olympiad
	var stored storage.Olympiad
	if f.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), olympiadLoadTimeout)
		defer cancel()
		var err error
		if stored, err = f.store.LoadOlympiad(ctx); err != nil {
			return err
		}
	}
	f.state = olympiadStateFromStore(stored)
	f.heroes = olympiad.NewHeroes(heroesFromStore(stored.Heroes))
	for _, u := range stored.Unclaimed {
		f.unclaimed[u.CharID] = u.Passes
	}
	f.nobles = olympiad.NewNobles(f.cfg, noblesFromStore(stored.Nobles), noblesFromStore(stored.NoblesEOM))
	now := s.Now()
	switch f.state.RestartAt(now) {
	case olympiad.RestartNewPeriod:
		if f.state.Cycle == 0 {
			f.state.Cycle = 1
		}
		f.state.Period = olympiad.Competition
		s.setNewOlympiadEnd()
	case olympiad.RestartCompetition:
		s.scheduleWeeklyChange()
	case olympiad.RestartValidation:
		s.loadNoblesRank()
		s.scheduleValidationEnd()
	case olympiad.RestartNextCycle:
		f.state.Cycle++
		f.state.Period = olympiad.Competition
		f.nobles.Clear()
		s.setNewOlympiadEnd()
	}
	if f.state.Period == olympiad.Competition {
		s.olympiadInit()
	}
	f.publishHeroes()
	s.loadHeroPages(f.heroes.Current())
	s.saveOlympiad()
	s.log.Info("game: olympiad loaded", "cycle", f.state.Cycle, "period", f.state.Period, "end", f.state.OlympiadEnd,
		"nobles", f.nobles.Len(), "heroes", len(f.heroes.Current()))
	return nil
}

// olympiadInit is Mobius Olympiad.init: in the competition period, new
// waiting lists, today's competition start, the period's end scheduled and
// the competition's start (updateCompStatus).
func (s *Server) olympiadInit() {
	f := &s.olympiad
	if f.state.Period == olympiad.Validation {
		return
	}
	f.queues = olympiad.Queues{}
	f.compStart = f.cfg.InitialCompStart(s.calendar, s.Now())
	s.cancelTimer(&f.endTimer)
	f.endTimer = s.loop.After(f.state.OlympiadEnd.Sub(s.Now()), s.olympiadEnd)
	s.updateCompStatus()
}

// cancelTimer cancels a timer that may be pending and clears it.
func (s *Server) cancelTimer(id *world.TimerID) {
	if *id != 0 {
		s.loop.Cancel(*id)
		*id = 0
	}
}

// broadcastOlympiadMessage sends a system message to everyone online.
func (s *Server) broadcastOlympiadMessage(m olympiad.Message, params ...serverpackets.SystemMessageParam) {
	s.world.BroadcastAll(serverpackets.NewSystemMessagePacket(serverpackets.SystemMessage{ID: serverpackets.SystemMessageID(m), Params: params}))
}

// setNewOlympiadEnd is Mobius setNewOlympiadEnd: "Period $s1 of the Grand
// Olympiad Games has started!" to everyone, the period's end and the next
// weekly points, and the weekly schedule.
func (s *Server) setNewOlympiadEnd() {
	f := &s.olympiad
	s.broadcastOlympiadMessage(olympiad.MsgPeriodStarted, serverpackets.NumberParam(int32(f.state.Cycle))) //nolint:gosec // small
	f.state.OlympiadEnd, f.state.NextWeeklyChange = f.cfg.NewOlympiadEnd(s.calendar, s.Now(), f.state.NextWeeklyChange)
	s.scheduleWeeklyChange()
}

// scheduleWeeklyChange is Mobius scheduleWeeklyChange: the weekly points at
// the next weekly change (at once when it passed), then every WeeklyPeriod.
func (s *Server) scheduleWeeklyChange() {
	f := &s.olympiad
	s.cancelTimer(&f.weeklyTimer)
	f.weeklyTimer = s.loop.After(f.state.UntilWeeklyChange(s.Now()), s.weeklyChange)
}

// weeklyChange is the weekly task: the points, and the next change a week
// after now.
func (s *Server) weeklyChange() {
	f := &s.olympiad
	f.nobles.AddWeeklyPoints(f.cfg, f.state.Period)
	s.log.Info("game: olympiad weekly points added to the nobles")
	f.state.NextWeeklyChange = s.Now().Add(f.cfg.WeeklyPeriod)
	f.weeklyTimer = s.loop.After(f.cfg.WeeklyPeriod, s.weeklyChange)
	s.saveOlympiad()
}

// updateCompStatus is Mobius updateCompStatus: the competition starts at
// its time (at once while it runs).
func (s *Server) updateCompStatus() {
	f := &s.olympiad
	delay, start := f.cfg.CompBegin(s.calendar, s.Now(), f.compStart)
	f.compStart = start
	s.cancelTimer(&f.compStartTimer)
	f.compStartTimer = s.loop.After(delay, s.competitionStart)
}

// untilCompEnd is Mobius getMillisToCompEnd.
func (s *Server) untilCompEnd() time.Duration {
	f := &s.olympiad
	return f.compStart.Add(f.cfg.CompPeriod).Sub(s.Now())
}

// competitionStart is the start of the daily competition: registration
// opens, everyone is told, the manager runs, and the end and the close of
// registration are scheduled.
func (s *Server) competitionStart() {
	f := &s.olympiad
	f.compStartTimer = 0
	if f.state.Period != olympiad.Competition {
		return
	}
	f.inComp = true
	s.broadcastOlympiadMessage(olympiad.MsgCompetitionStarted)
	s.log.Info("game: olympiad competition started")
	s.startOlympiadManager() // olympiad_manager.go
	if regEnd := s.untilCompEnd() - olympiadRegistrationClose; regEnd > 0 {
		s.cancelTimer(&f.regEndTimer)
		f.regEndTimer = s.loop.After(regEnd, func() {
			f.regEndTimer = 0
			s.broadcastOlympiadMessage(olympiad.MsgRegistrationEnded)
		})
	}
	s.cancelTimer(&f.compEndTimer)
	f.compEndTimer = s.loop.After(s.untilCompEnd(), s.competitionEnd)
}

// competitionEnd is the end of the daily competition: registration closes,
// everyone is told; once the battles ended, the state is saved and the next
// competition scheduled (init).
func (s *Server) competitionEnd() {
	f := &s.olympiad
	f.compEndTimer = 0
	if f.state.Period != olympiad.Competition {
		return
	}
	f.inComp = false
	s.broadcastOlympiadMessage(olympiad.MsgCompetitionEnded)
	s.log.Info("game: olympiad competition ended")
	s.awaitBattles()
}

// awaitBattles waits a minute at a time while a battle runs (Mobius's
// while (_battleStarted) sleep), then saves and schedules the next
// competition.
func (s *Server) awaitBattles() {
	f := &s.olympiad
	f.battleWaitTimer = 0
	if f.battleStarted {
		f.battleWaitTimer = s.loop.After(olympiadBattleWait, s.awaitBattles)
		return
	}
	s.saveOlympiad()
	s.olympiadInit()
}

// olympiadEnd is Mobius OlympiadEndTask: the period ends, the heroes are
// chosen, the month-end table is kept, the passes settled, and the
// validation period runs.
func (s *Server) olympiadEnd() {
	f := &s.olympiad
	f.endTimer = 0
	s.broadcastOlympiadMessage(olympiad.MsgPeriodEnded, serverpackets.NumberParam(int32(f.state.Cycle))) //nolint:gosec // small
	s.announceToAll(olympiadValidationBegan, false)
	s.cancelTimer(&f.weeklyTimer)
	f.state.Period = olympiad.Validation
	s.computeNewHeroes(f.nobles.HeroesToBe()) // hero.go
	f.nobles.CopyToMonthEnd()
	f.state.ValidationEnd = s.Now().Add(f.cfg.ValidationPeriod)
	s.loadNoblesRank()
	s.scheduleValidationEnd()
	s.saveOlympiad()
	s.log.Info("game: olympiad period ended", "cycle", f.state.Cycle, "heroes", len(f.heroes.Current()))
}

// endOlympiadNow is Mobius manualSelectHeroes (//endolympiad): the period
// ends at once.
func (s *Server) endOlympiadNow() {
	f := &s.olympiad
	s.cancelTimer(&f.endTimer)
	f.endTimer = s.loop.After(0, s.olympiadEnd)
}

// scheduleValidationEnd schedules the validation period's end.
func (s *Server) scheduleValidationEnd() {
	f := &s.olympiad
	s.cancelTimer(&f.validationTimer)
	f.validationTimer = s.loop.After(f.state.UntilValidationEnd(s.Now()), s.validationEnd)
}

// validationEnd is Mobius ValidationEndTask: the next cycle starts with no
// nobles.
func (s *Server) validationEnd() {
	f := &s.olympiad
	f.validationTimer = 0
	s.announceToAll(olympiadValidationEnded, false)
	f.state.Period = olympiad.Competition
	f.state.Cycle++
	f.nobles.Clear()
	s.setNewOlympiadEnd()
	s.olympiadInit()
	s.saveOlympiad()
}

// loadNoblesRank is Mobius loadNoblesRank: the ranks of the last period's
// end, and the passes each ranked noble earns, kept until it settles them.
func (s *Server) loadNoblesRank() {
	f := &s.olympiad
	passes := f.nobles.SettlePasses(f.cfg, f.state.Period, f.nobles.Ranks(), f.heroes.IsHero)
	for id, n := range passes {
		f.unclaimed[id] = int64(n)
	}
}

// saveOlympiad hands the Olympiad's state to the writer (decision X4).
func (s *Server) saveOlympiad() {
	f := &s.olympiad
	if f.store == nil || f.writer == nil {
		return
	}
	f.writer.saveState(f.store, s.olympiadToStore())
}

// saveOlympiadAtShutdown writes the Olympiad when the server stops (Mobius
// Shutdown: saveOlympiadStatus and Hero.shutdown) and waits for it.
func (s *Server) saveOlympiadAtShutdown() {
	f := &s.olympiad
	if f.writer == nil || !f.cfg.Enabled {
		return
	}
	s.saveOlympiad()
	ctx, cancel := context.WithTimeout(s.background(), shutdownTimeout)
	defer cancel()
	if err := f.writer.wait(ctx); err != nil {
		s.log.Error("game: the olympiad is not saved at shutdown", "err", err)
	}
}

// olympiadToStore converts the state for the store.
func (s *Server) olympiadToStore() storage.Olympiad {
	f := &s.olympiad
	ms := func(t time.Time) int64 {
		if t.IsZero() {
			return 0
		}
		return t.UnixMilli()
	}
	o := storage.Olympiad{
		Cycle: int32(f.state.Cycle), Period: int32(f.state.Period), //nolint:gosec // small
		OlympiadEnd: ms(f.state.OlympiadEnd), ValidationEnd: ms(f.state.ValidationEnd), NextWeeklyChange: ms(f.state.NextWeeklyChange),
		Nobles: noblesToStore(f.nobles.All()), NoblesEOM: noblesToStore(f.nobles.MonthEnd()),
	}
	for _, h := range f.heroes.All() {
		o.Heroes = append(o.Heroes, storage.Hero{
			CharID: h.CharID, ClassID: int32(h.ClassID), Name: h.Name, Count: int32(h.Count), //nolint:gosec // small
			Played: h.Played, Claimed: h.Claimed, Message: h.Message,
		})
	}
	for _, id := range slices.Sorted(maps.Keys(f.unclaimed)) {
		o.Unclaimed = append(o.Unclaimed, storage.UnclaimedPasses{CharID: id, Passes: f.unclaimed[id]})
	}
	return o
}

// olympiadStateFromStore converts the stored calendar.
func olympiadStateFromStore(o storage.Olympiad) olympiad.State {
	at := func(ms int64) time.Time {
		if ms == 0 {
			return time.Time{}
		}
		return time.UnixMilli(ms)
	}
	return olympiad.State{
		Cycle: int(o.Cycle), Period: olympiad.Period(o.Period),
		OlympiadEnd: at(o.OlympiadEnd), ValidationEnd: at(o.ValidationEnd), NextWeeklyChange: at(o.NextWeeklyChange),
	}
}

func noblesFromStore(list []storage.OlympiadNoble) []olympiad.Noble {
	out := make([]olympiad.Noble, len(list))
	for i, n := range list {
		out[i] = olympiad.Noble{CharID: n.CharID, ClassID: int(n.ClassID), Name: n.Name, Points: int(n.Points),
			Done: int(n.Done), Won: int(n.Won), Lost: int(n.Lost), Drawn: int(n.Drawn)}
	}
	return out
}

func noblesToStore(list []olympiad.Noble) []storage.OlympiadNoble {
	out := make([]storage.OlympiadNoble, len(list))
	for i, n := range list {
		out[i] = storage.OlympiadNoble{CharID: n.CharID, ClassID: int32(n.ClassID), Name: n.Name, Points: int32(n.Points), //nolint:gosec // small
			Done: int32(n.Done), Won: int32(n.Won), Lost: int32(n.Lost), Drawn: int32(n.Drawn)} //nolint:gosec // small
	}
	return out
}

func heroesFromStore(list []storage.Hero) []olympiad.Hero {
	out := make([]olympiad.Hero, len(list))
	for i, h := range list {
		out[i] = olympiad.Hero{CharID: h.CharID, ClassID: int(h.ClassID), Name: h.Name, Count: int(h.Count),
			Played: h.Played, Claimed: h.Claimed, Message: h.Message}
	}
	return out
}

// olympiadWriter writes the Olympiad in the background: a state replaces
// the last one waiting, a fight or a diary entry is appended, in order.
// Safe for concurrent use.
type olympiadWriter struct {
	log interface {
		Error(msg string, args ...any)
	}

	mu      sync.Mutex
	state   *storage.Olympiad
	store   storage.OlympiadStore
	appends []func(ctx context.Context, store storage.OlympiadStore) error
	running bool
	idle    []chan struct{}
}

// saveState hands a state to write; it never blocks.
func (w *olympiadWriter) saveState(store storage.OlympiadStore, o storage.Olympiad) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.state, w.store = &o, store
	w.start()
}

// add hands an append to write; it never blocks.
func (w *olympiadWriter) add(store storage.OlympiadStore, fn func(ctx context.Context, store storage.OlympiadStore) error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.store = store
	w.appends = append(w.appends, fn)
	w.start()
}

// start runs the writer if it does not run. Called with mu held.
func (w *olympiadWriter) start() {
	if !w.running {
		w.running = true
		go w.run()
	}
}

// run writes until nothing waits.
func (w *olympiadWriter) run() {
	for {
		w.mu.Lock()
		state, appends, store := w.state, w.appends, w.store
		w.state, w.appends = nil, nil
		if state == nil && len(appends) == 0 {
			w.running = false
			for _, ch := range w.idle {
				close(ch)
			}
			w.idle = nil
			w.mu.Unlock()
			return
		}
		w.mu.Unlock()
		for _, fn := range appends {
			ctx, cancel := context.WithTimeout(context.Background(), olympiadWriteTimeout)
			if err := fn(ctx, store); err != nil {
				w.log.Error("game: cannot write an olympiad record", "err", err)
			}
			cancel()
		}
		if state != nil {
			ctx, cancel := context.WithTimeout(context.Background(), olympiadWriteTimeout)
			if err := store.SaveOlympiad(ctx, *state); err != nil {
				w.log.Error("game: cannot save the olympiad", "err", err)
			}
			cancel()
		}
	}
}

// wait returns once everything handed over so far is written, or with
// ctx's error.
func (w *olympiadWriter) wait(ctx context.Context) error {
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
