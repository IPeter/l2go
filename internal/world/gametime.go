package world

import (
	"sync/atomic"
	"time"
)

// Game time, from Mobius GameTimeTaskManager: 6 game days per real day.
const (
	// GameTick is one game tick (Mobius MILLIS_IN_TICK).
	GameTick = 100 * time.Millisecond
	// GameMinute is how long a game minute lasts in real time.
	GameMinute = 10 * time.Second
	// GameDay is how long a game day lasts in real time.
	GameDay = 4 * time.Hour

	ticksPerGameMinute = int64(GameMinute / GameTick) // 100
	ticksPerGameDay    = int64(GameDay / GameTick)    // 144000
	nightEndsHour      = 6
)

// GameTime is the clock the players see (Mobius GameTimeTaskManager). The
// game day starts at local midnight of the day the server started (midnight
// in the location of the clock's time) and lasts 4 real hours; it is night
// before 06:00.
type GameTime struct {
	loop  *Loop
	clock Clock
	ref   time.Time // the reference midnight
	night atomic.Bool
	hooks []func(night bool) // loop-only
}

func newGameTime(loop *Loop) *GameTime {
	clock := loop.Clock()
	now := clock.Now()
	y, m, d := now.Date()
	return &GameTime{loop: loop, clock: clock, ref: time.Date(y, m, d, 0, 0, 0, 0, now.Location())}
}

// Ticks returns the game ticks (100 ms) since the reference midnight (Mobius
// getGameTicks). Safe for concurrent use.
func (g *GameTime) Ticks() int64 { return gameTicks(g.ref, g.clock.Now()) }

// Minutes returns the minutes into the game day, 0–1439 (Mobius getGameTime,
// which CharSelected, ClientSetTime and NetPing send). Safe for concurrent
// use.
func (g *GameTime) Minutes() int32 { return gameMinutes(g.Ticks()) }

// Hour returns the game hour, 0–23. Safe for concurrent use.
func (g *GameTime) Hour() int32 { return g.Minutes() / 60 }

// Minute returns the minute of the game hour, 0–59. Safe for concurrent use.
func (g *GameTime) Minute() int32 { return g.Minutes() % 60 }

// IsNight reports whether it is night as of the last tick, the moment the
// day/night hooks saw. Safe for concurrent use.
func (g *GameTime) IsNight() bool { return g.night.Load() }

// OnDayNightChange adds fn to the functions called on the loop when night
// starts (true) or ends (false): night spawns, Shadow Sense, scripts. As in
// Mobius the state starts as day, so a server started at night calls them
// with true on its first tick. Loop-only.
func (g *GameTime) OnDayNightChange(fn func(night bool)) {
	g.hooks = append(g.hooks, fn)
}

// update is the world's tick hook: it notices day/night changes.
func (g *GameTime) update(now time.Time) {
	night := gameMinutes(gameTicks(g.ref, now))/60 < nightEndsHour
	if night == g.night.Load() {
		return
	}
	g.night.Store(night)
	for _, fn := range g.hooks {
		g.loop.call("day/night hook", func() { fn(night) })
	}
}

func gameTicks(ref, now time.Time) int64 { return int64(now.Sub(ref) / GameTick) }

func gameMinutes(ticks int64) int32 {
	t := ticks % ticksPerGameDay
	if t < 0 {
		t += ticksPerGameDay
	}
	return int32(t / ticksPerGameMinute)
}
