package game

import (
	"fmt"
	"math"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// fakeRand is a random source for the death tests: Get and Between answer
// from a script (0 or lo once it ends) and record their draws, so a test can
// check Mobius's draw order. The other methods fall back to a seeded PCG.
// It is swapped in on the loop only around the call under test (withRand),
// so the draws of other features stay out.
type fakeRand struct {
	rnd.Source
	mu      sync.Mutex
	answers []int
	calls   []string
}

func newFakeRand(answers ...int) *fakeRand {
	return &fakeRand{Source: rnd.New(1), answers: answers}
}

func (f *fakeRand) next(call string, def int) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, call)
	if len(f.answers) == 0 {
		return def
	}
	v := f.answers[0]
	f.answers = f.answers[1:]
	return v
}

func (f *fakeRand) Get(n int) int {
	if n <= 0 {
		return 0
	}
	return f.next(fmt.Sprintf("Get(%d)", n), 0)
}

func (f *fakeRand) Between(lo, hi int) int {
	if lo >= hi {
		return lo
	}
	return f.next(fmt.Sprintf("Between(%d, %d)", lo, hi), lo)
}

func (f *fakeRand) drawn() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.calls)
}

// withRand runs fn on the loop with f as the server's random source.
func (e *testEnv) withRand(f rnd.Source, fn func()) {
	e.t.Helper()
	e.onLoop(func() {
		old := e.srv.rnd
		e.srv.rnd = f
		defer func() { e.srv.rnd = old }()
		fn()
	})
}

// deathChar describes a character for enterDeathChar.
type deathChar struct {
	account, name string
	access        storage.AccessLevel
	pos           world.Position
	level         int32
	exp           int64
	karma, pk     int32
	hp            float64 // 0 keeps the full HP of level 1; use -1 for dead
	items         []storage.Item
}

// enterDeathChar creates a human fighter as c describes, enters the world
// with it and drops the packets.
func (e *testEnv) enterDeathChar(c deathChar) (*fakeSession, *Player) {
	e.t.Helper()
	ch := e.createCharacter(c.account, c.name, classHumanFighter, c.access)
	ch.X, ch.Y, ch.Z = tiStart.X, tiStart.Y, tiStart.Z
	if c.pos != (world.Position{}) {
		ch.X, ch.Y, ch.Z = c.pos.X, c.pos.Y, c.pos.Z
	}
	if c.level > 0 {
		ch.Level = c.level
	}
	ch.Exp, ch.Karma, ch.PKKills = c.exp, c.karma, c.pk
	if c.hp < 0 {
		ch.CurHP = 0
	} else if c.hp > 0 {
		ch.CurHP = c.hp
	}
	if len(c.items) > 0 {
		ids, err := e.ids.NextIDs(e.ctx, len(c.items))
		if err != nil {
			e.t.Fatalf("NextIDs: %v", err)
		}
		for i := range c.items {
			c.items[i].ObjectID = ids[i]
		}
		ch.Items = append(ch.Items, c.items...)
	}
	if err := e.store.Update(e.ctx, &ch); err != nil {
		e.t.Fatalf("Update: %v", err)
	}
	sess := newSession(c.account)
	p := e.enter(sess, 0)
	e.sync()
	sess.take(e.t)
	return sess, p
}

// deathGremlin spawns a Gremlin (a monster) out of everyone's sight, to be
// the killer.
func deathGremlin(e *testEnv) *Npc { return e.spawnNpc(npcTestGremlin, eastOf(tiStart, 6000)) }

// killWith kills p with killer on the loop, drawing from f.
func (e *testEnv) killWith(f rnd.Source, p *Player, killer Creature) {
	e.t.Helper()
	e.withRand(f, func() {
		if !e.srv.Kill(p, killer) {
			e.t.Errorf("Kill(%s) found the player dead", p.Name())
		}
	})
}

// playerState reads level, exp, exp before death and the dead flag.
func (e *testEnv) playerState(p *Player) (level int, exp, before int64, dead bool) {
	e.onLoop(func() { level, exp, before, dead = p.Level(), p.Exp(), p.ExpBeforeDeath(), p.IsDead() })
	return
}

func checkDie(t *testing.T, ps []testclient.Packet, p *Player, fixed bool) {
	t.Helper()
	var dies []testclient.Die
	for _, d := range only[testclient.Die](ps) {
		if d.ObjectID == p.ObjectID() {
			dies = append(dies, d)
		}
	}
	if len(dies) != 1 {
		t.Fatalf("got %d Die of %s in %q, want 1", len(dies), p.Name(), testclient.Names(ps))
	}
	if d := dies[0]; d.ObjectID != p.ObjectID() || !d.CanTeleport || d.FixedRes != fixed || d.ToClanHall || d.ToCastle || d.ToSiegeHQ {
		t.Errorf("Die %+v, want object %d, to village, fixed %v", d, p.ObjectID(), fixed)
	}
}

// A death at level 5 costs nothing: Lucky (F12). No draw is made.
func TestDeathAtLevel5IsLucky(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{})
	exp := e.srv.ExpForLevel(5) + 100
	sess, p := e.enterDeathChar(deathChar{account: "alice", name: "Alice", level: 5, exp: exp})
	gremlin := deathGremlin(e)

	f := newFakeRand()
	e.killWith(f, p, gremlin)
	if d := f.drawn(); len(d) != 0 {
		t.Errorf("draws %q, want none", d)
	}
	level, got, before, dead := e.playerState(p)
	if level != 5 || got != exp || before != 0 || !dead {
		t.Errorf("after death: level %d exp %d before %d dead %v; want 5, %d, 0, dead", level, got, before, dead, exp)
	}
	checkDie(t, sess.take(t), p, false)
}

// A death at level 10 to a monster loses round(22972 × 8.875 %) = 2,039 exp;
// at the start of the level that drops the character to level 9, which is
// lucky again, so no Death Penalty roll follows (as in Mobius).
func TestDeathAtLevel10Delevels(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{})
	if e.srv.ExpForLevel(10) != 48229 || e.srv.ExpForLevel(11)-e.srv.ExpForLevel(10) != 22972 {
		t.Fatalf("level 10 exp %d, span %d", e.srv.ExpForLevel(10), e.srv.ExpForLevel(11)-e.srv.ExpForLevel(10))
	}
	sess, p := e.enterDeathChar(deathChar{account: "alice", name: "Alice", level: 10, exp: 48229})
	gremlin := deathGremlin(e)

	f := newFakeRand()
	e.killWith(f, p, gremlin)
	if d := f.drawn(); len(d) != 0 {
		t.Errorf("draws %q, want none (lucky after the delevel)", d)
	}
	level, exp, before, _ := e.playerState(p)
	if level != 9 || exp != 48229-2039 || before != 48229 {
		t.Errorf("after death: level %d exp %d before %d; want 9, %d, 48229", level, exp, before, 48229-2039)
	}
	e.advance(world.GameTick)
	ps := sess.take(t)
	checkDie(t, ps, p, false)
	uis := only[testclient.UserInfo](ps)
	if len(uis) == 0 || uis[len(uis)-1].Level != 9 || uis[len(uis)-1].Exp != 46190 {
		t.Errorf("UserInfo after the delevel: %+v", uis)
	}
	for _, m := range only[testclient.SystemMessage](ps) {
		if m.ID == int32(sysMsgExpDecreased) {
			t.Errorf("the death loss is silent in Mobius, got message %d", m.ID)
		}
	}
}

// From level 10 a monster kill rolls the Death Penalty: Between(1, 100) at
// most 20 raises it, with EtcStatusUpdate and "Your Death Penalty is now
// level 1."; 21 does not.
func TestDeathPenaltyRoll(t *testing.T) {
	for _, tc := range []struct {
		roll int
		want int32
	}{{20, 1}, {21, 0}} {
		t.Run(fmt.Sprint(tc.roll), func(t *testing.T) {
			e := newDialogEnv(t, dialogEnvOptions{})
			sess, p := e.enterDeathChar(deathChar{account: "alice", name: "Alice", level: 10, exp: 60000})
			gremlin := deathGremlin(e)

			f := newFakeRand(tc.roll)
			e.killWith(f, p, gremlin)
			if d := f.drawn(); !slices.Equal(d, []string{"Between(1, 100)"}) {
				t.Errorf("draws %q", d)
			}
			var penalty int32
			e.onLoop(func() { penalty = p.DeathPenaltyLevel() })
			if penalty != tc.want {
				t.Errorf("death penalty level %d, want %d", penalty, tc.want)
			}
			level, exp, _, _ := e.playerState(p)
			if level != 10 || exp != 60000-2039 {
				t.Errorf("level %d exp %d", level, exp)
			}
			ps := sess.take(t)
			etc := only[testclient.EtcStatusUpdate](ps)
			msgs := only[testclient.SystemMessage](ps)
			gotMsg := slices.ContainsFunc(msgs, func(m testclient.SystemMessage) bool {
				return m.ID == int32(sysMsgDeathPenaltyLevel) && len(m.Params) == 1 && m.Params[0].Number == 1
			})
			if tc.want == 1 && (len(etc) != 1 || etc[0].DeathPenalty != 1 || !gotMsg) {
				t.Errorf("EtcStatusUpdate %+v, message %v", etc, gotMsg)
			}
			if tc.want == 0 && (len(etc) != 0 || gotMsg) {
				t.Errorf("no penalty, yet EtcStatusUpdate %+v, message %v", etc, gotMsg)
			}
		})
	}
}

// "To Village" revives with 65 % HP before the teleport to a restart point
// of the region, 50 units around it at most. A death in the Talking Island
// village restarts at one of its points.
func TestRestartToVillage(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{})
	sess, p := e.enterDeathChar(deathChar{account: "alice", name: "Alice", pos: tiVillage})
	gremlin := deathGremlin(e)
	e.killWith(newFakeRand(), p, gremlin)
	sess.take(t)

	region, ok := e.srv.WorldData().MapRegion("talking_island_town")
	if !ok || len(region.RespawnPoints) < 2 {
		t.Fatalf("talking_island_town: %+v", region)
	}
	f := newFakeRand(1, 30, -20)
	e.withRand(f, func() {
		e.srv.handleRestartPoint(p, int32Body(clientpackets.RestartPointVillage))
	})
	want := []string{fmt.Sprintf("Get(%d)", len(region.RespawnPoints)), "Between(-50, 50)", "Between(-50, 50)"}
	if d := f.drawn(); !slices.Equal(d, want) {
		t.Errorf("draws %q, want %q", d, want)
	}
	ps := sess.take(t)
	names := testclient.Names(ps)
	ri, ti := slices.Index(names, "Revive"), slices.Index(names, "TeleportToLocation")
	if ri < 0 || ti < 0 || ri > ti {
		t.Fatalf("packets %q: want Revive before TeleportToLocation", names)
	}
	pt := region.RespawnPoints[1]
	tp := ps[ti].(testclient.TeleportToLocation)
	if wantPos := (testclient.Position{X: int32(pt.X) + 30, Y: int32(pt.Y) - 20, Z: int32(pt.Z) + teleportZOffset}); tp.Position() != wantPos {
		t.Errorf("teleported to %+v, want %+v", tp.Position(), wantPos)
	}
	var hp, maxHP float64
	var dead bool
	e.onLoop(func() { hp, maxHP, dead = p.CurrentHP(), float64(p.MaxHP()), p.IsDead() })
	if dead || hp != maxHP*0.65 {
		t.Errorf("after the restart: dead %v, HP %v of %v; want 65 %%", dead, hp, maxHP)
	}
	e.srv.Handle(sess, clientpackets.OpAppearing, nil)
	e.sync()
	if pos := e.position(p); math.Abs(float64(pos.X-int32(pt.X))) > 50 || math.Abs(float64(pos.Y-int32(pt.Y))) > 50 {
		t.Errorf("standing at %v, not within 50 of %+v", pos, pt)
	}
}

// A player with karma restarts at a chaotic point when its region has one.
func TestRestartChaotic(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{})
	_, p := e.enterDeathChar(deathChar{account: "alice", name: "Alice", karma: 100, pos: tiVillage})
	var loc data.Location
	var ok bool
	f := newFakeRand()
	e.withRand(f, func() { loc, ok = e.srv.restartLocation(p) })
	region, _ := e.srv.WorldData().MapRegion("talking_island_town")
	points := region.RespawnPoints
	if len(region.ChaoticRespawnPoints) > 0 {
		points = region.ChaoticRespawnPoints
	}
	if !ok || loc != points[0] {
		t.Errorf("restart location %+v %v, want %+v", loc, ok, points[0])
	}
	if d := f.drawn(); !slices.Equal(d, []string{fmt.Sprintf("Get(%d)", len(points))}) {
		t.Errorf("draws %q", d)
	}
}

// Fixed (4) is for GMs: it revives where they lie and gives back the lost
// exp; a player's request is ignored, as are the clan points.
func TestRestartFixed(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{})
	ss, pl := e.enterDeathChar(deathChar{account: "alice", name: "Alice"})
	sg, gm := e.enterDeathChar(deathChar{account: "gm", name: "Gm", access: storage.AccessLevelAdmin, level: 10, exp: 60000, pos: eastOf(tiStart, 3000)})
	gremlin := deathGremlin(e)
	e.killWith(newFakeRand(), pl, gremlin)
	e.killWith(newFakeRand(), gm, gremlin)
	checkDie(t, sg.take(t), gm, true)
	ss.take(t)

	for _, point := range []int32{clientpackets.RestartPointFixed, clientpackets.RestartPointClanHall, clientpackets.RestartPointCastle, clientpackets.RestartPointSiegeHQ, clientpackets.RestartPointAgathion} {
		e.srv.Handle(ss, clientpackets.OpRequestRestartPoint, int32Body(point))
		e.sync()
		if _, _, _, dead := e.playerState(pl); !dead {
			t.Fatalf("point %d revived a player", point)
		}
	}
	if ps := ss.take(t); len(only[testclient.Revive](ps)) != 0 {
		t.Errorf("player got %q", testclient.Names(ps))
	}

	if _, exp, _, _ := e.playerState(gm); exp != 60000-2039 {
		t.Fatalf("GM exp after death %d", exp)
	}
	e.srv.Handle(sg, clientpackets.OpRequestRestartPoint, int32Body(clientpackets.RestartPointFixed))
	e.sync()
	level, exp, before, dead := e.playerState(gm)
	if dead || level != 10 || exp != 60000 || before != 0 {
		t.Errorf("GM after fixed: dead %v level %d exp %d before %d", dead, level, exp, before)
	}
	ps := sg.take(t)
	if len(only[testclient.Revive](ps)) != 1 || len(only[testclient.TeleportToLocation](ps)) != 0 {
		t.Errorf("GM got %q", testclient.Names(ps))
	}
}

// Entering the world with 0 HP shows the restart window.
func TestEnterWorldDead(t *testing.T) {
	e := newEnv(t, nil)
	c := e.createCharacter("alice", "Alice", classHumanFighter, storage.AccessLevelPlayer)
	c.CurHP = 0
	if err := e.store.Update(e.ctx, &c); err != nil {
		t.Fatal(err)
	}
	sess := newSession("alice")
	p := e.enter(sess, 0)
	ps := sess.take(t)
	checkDie(t, ps, p, false)
	names := testclient.Names(ps)
	if di, wi := slices.Index(names, "Die"), slices.Index(names, "SystemMessage"); di < wi {
		t.Errorf("Die before the welcome message: %q", names)
	}
	if _, _, _, dead := e.playerState(p); !dead {
		t.Error("the player is not dead")
	}
}

// pvpPair enters Alice and Bob next to each other outside the village.
func pvpPair(t *testing.T, e *testEnv) (sa, sb *fakeSession, a, b *Player) {
	t.Helper()
	sa, a = e.enterDeathChar(deathChar{account: "alice", name: "Alice"})
	sb, b = e.enterDeathChar(deathChar{account: "bob", name: "Bob", pos: eastOf(tiStart, 50)})
	e.sync()
	sa.take(t)
	sb.take(t)
	return sa, sb, a, b
}

func flagOf(e *testEnv, p *Player) int32 {
	var v int32
	e.onLoop(func() { v = p.PvPFlag })
	return v
}

// lastRelation returns the last RelationChanged about o in ps.
func lastRelation(t *testing.T, ps []testclient.Packet, o *Player) testclient.RelationChanged {
	t.Helper()
	var last *testclient.RelationChanged
	for _, r := range only[testclient.RelationChanged](ps) {
		if r.ObjectID == o.ObjectID() {
			last = &r
		}
	}
	if last == nil {
		t.Fatalf("no RelationChanged about %s in %q", o.Name(), testclient.Names(ps))
	}
	return *last
}

// Hitting a karma-free player flags the attacker for 120 s: flag 1 at once,
// blinking (2) for the last 20 s, cleared by the 1 s scan after 120 s. The
// players around see each change in RelationChanged.
func TestPvPFlagTimings(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{})
	sa, sb, a, b := pvpPair(t, e)
	e.advance(500 * time.Millisecond) // the flag starts between two scans
	e.onLoop(func() { e.srv.UpdatePvPStatus(a, b) })
	if f := flagOf(e, a); f != 1 {
		t.Fatalf("flag %d at 0 s, want 1", f)
	}
	if r := lastRelation(t, sb.take(t), a); r.PvpFlag != 1 {
		t.Errorf("Bob sees %+v", r)
	}
	e.advance(world.GameTick)
	elapsed := world.GameTick
	if uis := only[testclient.UserInfo](sa.take(t)); len(uis) == 0 || uis[len(uis)-1].PvpFlag != 1 {
		t.Errorf("Alice's UserInfo %+v", uis)
	}

	// The scans run every second from the server's start, 0.5 s off the
	// flag's start.
	for _, st := range []struct {
		at   time.Duration // since the flag started
		want int32
	}{
		{99500 * time.Millisecond, 1},  // the scan 100 s after the start: 20.5 s left
		{100500 * time.Millisecond, 2}, // 19.5 s left: blinking
		{119500 * time.Millisecond, 2}, // 0.5 s left
		{120500 * time.Millisecond, 0}, // over
	} {
		e.advance(st.at - elapsed)
		elapsed = st.at
		if f := flagOf(e, a); f != st.want {
			t.Fatalf("flag %d at %v, want %d", f, st.at, st.want)
		}
	}
	if r := lastRelation(t, sb.take(t), a); r.PvpFlag != 0 {
		t.Errorf("Bob last sees %+v", r)
	}
}

// Hitting a flagged player flags for 60 s only.
func TestPvPFlagAgainstFlagged(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{})
	_, _, a, b := pvpPair(t, e)
	var left time.Duration
	e.onLoop(func() {
		e.srv.UpdatePvPStatus(b, a)
		e.srv.UpdatePvPStatus(a, b)
		left = e.srv.pvpFlagTimeLeft(a)
	})
	if left != 60*time.Second {
		t.Errorf("flag time %v, want 60 s", left)
	}
}

// Killing an unflagged, karma-free player is a PK: 720 karma for the first
// (calculateKarmaGain), a red name; dying with karma keeps karma − karma/4.
// Killing a flagged player counts as PvP.
func TestPKAndPvPKills(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{})
	sa, sb, a, b := pvpPair(t, e)

	e.killWith(newFakeRand(), b, a)
	var karma, pk, pvp int32
	e.onLoop(func() { karma, pk, pvp = a.Karma(), a.PKKills(), a.PvPKills() })
	if karma != 720 || pk != 1 || pvp != 0 {
		t.Fatalf("after the PK: karma %d, PK %d, PvP %d; want 720, 1, 0", karma, pk, pvp)
	}
	ps := sa.take(t)
	sus := only[testclient.StatusUpdate](ps)
	if !slices.ContainsFunc(sus, func(s testclient.StatusUpdate) bool {
		v, ok := s.Value(int32(0x1b))
		return ok && v == 720
	}) {
		t.Errorf("no karma StatusUpdate in %q", testclient.Names(ps))
	}
	if r := lastRelation(t, sb.take(t), a); r.Karma != 720 {
		t.Errorf("Bob sees %+v", r)
	}

	// A flagged Bob, revived, is a PvP kill for Alice (who has no karma).
	e.onLoop(func() {
		e.srv.RevivePlayer(b)
		e.srv.SetKarma(a, 0)
		e.srv.UpdatePvPStatus(b, a)
	})
	e.killWith(newFakeRand(), b, a)
	e.onLoop(func() { karma, pk, pvp = a.Karma(), a.PKKills(), a.PvPKills() })
	if karma != 0 || pk != 1 || pvp != 1 {
		t.Errorf("after the PvP kill: karma %d, PK %d, PvP %d; want 0, 1, 1", karma, pk, pvp)
	}

	// Alice with 720 karma dies to a monster: 540 left.
	e.onLoop(func() { e.srv.SetKarma(a, 720) })
	e.killWith(newFakeRand(), a, deathGremlin(e))
	e.onLoop(func() { karma = a.Karma() })
	if karma != 540 {
		t.Errorf("karma after death %d, want 540", karma)
	}
}

// Exp burns karma: exp / RateKarmaLost / karmaLoss(level) / 30, with
// "Your Karma has been changed to $s1.".
func TestKarmaBurnsWithExp(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{})
	sess, p := e.enterDeathChar(deathChar{account: "alice", name: "Alice", karma: 720})
	var karma int32
	var level int
	e.onLoop(func() {
		e.srv.AddExp(p, 10000)
		karma, level = p.Karma(), p.Level()
	})
	// Mobius reads the modifier of the level the exp led to.
	mod, _ := e.data.KarmaLoss.At(level)
	want := int32(720 - e.srv.Combat().KarmaLost(10000, mod))
	if karma != want || want >= 720 {
		t.Errorf("karma %d, want %d", karma, want)
	}
	msgs := only[testclient.SystemMessage](sess.take(t))
	if !slices.ContainsFunc(msgs, func(m testclient.SystemMessage) bool {
		return m.ID == int32(sysMsgKarmaChanged) && len(m.Params) == 1 && m.Params[0].Number == int64(want)
	}) {
		t.Errorf("messages %+v", msgs)
	}
}

// Karma hooks run on every change, with the old and the new karma.
func TestOnKarmaChanged(t *testing.T) {
	type change struct{ old, karma int32 }
	var got []change
	e := newDialogEnv(t, dialogEnvOptions{})
	e.onLoop(func() {
		e.srv.OnKarmaChanged(func(_ *Player, old, karma int32) { got = append(got, change{old, karma}) })
	})
	_, p := e.enterDeathChar(deathChar{account: "alice", name: "Alice"})
	e.onLoop(func() {
		e.srv.SetKarma(p, 100)
		e.srv.SetKarma(p, -5)
	})
	if want := []change{{0, 100}, {100, 0}}; !slices.Equal(got, want) {
		t.Errorf("hooks saw %v, want %v", got, want)
	}
}

// A PK with 6 kills drops items on death at the karma rates: one roll for
// the drop, one per droppable item, and the place around the player.
func TestKarmaDrop(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{})
	sess, p := e.enterDeathChar(deathChar{
		account: "alice", name: "Alice", karma: 2000, pk: 6,
		items: []storage.Item{{ItemID: itemLongSword, Count: 1, Slot: storage.PaperdollNone, Mana: storage.NoMana}},
	})
	// The drop roll (0 < 40); a miss (99) for each droppable starting item,
	// then a hit (10 < 50) for the sword and its place (0, 0 around).
	var swordID int32
	answers, want := []int{0}, []string{"Get(100)"}
	e.onLoop(func() {
		swordID = p.inv.ByItemID(itemLongSword).ObjectID()
		pvp := e.srv.pvpConfig()
		for _, it := range p.inv.Items() {
			if it.ItemID() == itemLongSword || !karmaDroppable(it, pvp.ListOfNonDroppableItems, pvp.ListOfPetItems) {
				continue
			}
			answers, want = append(answers, 99), append(want, "Get(100)")
		}
	})
	answers, want = append(answers, 10, 25, 25), append(want, "Get(100)", "Get(50)", "Get(50)")
	pos := e.position(p)

	f := newFakeRand(answers...)
	e.killWith(f, p, deathGremlin(e))
	if d := f.drawn(); !slices.Equal(d, want) {
		t.Errorf("draws %q, want %q", d, want)
	}
	var onGround world.Object
	var carried bool
	e.onLoop(func() {
		onGround, _ = e.srv.World().Find(world.ObjectID(swordID))
		carried = p.inv.ByItemID(itemLongSword) != nil
	})
	if carried || onGround == nil {
		t.Fatalf("the sword: carried %v, on the ground %v", carried, onGround)
	}
	if gp := onGround.Position(); gp.X != pos.X || gp.Y != pos.Y || gp.Z != pos.Z+20 {
		t.Errorf("the sword lies at %v, want %v + 20 high", gp, pos)
	}
	if msgs := only[testclient.SystemMessage](sess.take(t)); !slices.ContainsFunc(msgs, func(m testclient.SystemMessage) bool { return m.ID == int32(sysMsgDroppedS1) }) {
		t.Errorf("no \"You have dropped\" in %+v", msgs)
	}
}

// Below 6 PKs nothing drops and nothing is drawn.
func TestKarmaDropNeedsPKs(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{})
	_, p := e.enterDeathChar(deathChar{account: "alice", name: "Alice", karma: 2000, pk: 5})
	f := newFakeRand()
	e.killWith(f, p, deathGremlin(e))
	if d := f.drawn(); len(d) != 0 {
		t.Errorf("draws %q", d)
	}
}

// A level-up fills HP/MP/CP, plays SocialAction 15 and sends "Your level
// has increased!" before "You have earned ...".
func TestLevelUp(t *testing.T) {
	e := newEnv(t, nil)
	sess, p := e.enterDeathChar(deathChar{account: "alice", name: "Alice", hp: 10})
	e.onLoop(func() {
		e.srv.AddExpAndSP(p, stats.Reward{Exp: 68, SP: 2, ShownExp: 68, ShownSP: 2})
	})
	e.advance(world.GameTick)
	ps := sess.take(t)
	var level int
	var hp, maxHP float64
	e.onLoop(func() { level, hp, maxHP = p.Level(), p.CurrentHP(), float64(p.MaxHP()) })
	if level != 2 || hp != maxHP {
		t.Errorf("level %d, HP %v of %v", level, hp, maxHP)
	}
	soc := only[testclient.SocialAction](ps)
	if len(soc) != 1 || soc[0].ActionID != 15 {
		t.Errorf("SocialAction %+v", soc)
	}
	var ids []int32
	for _, m := range only[testclient.SystemMessage](ps) {
		ids = append(ids, m.ID)
	}
	if want := []int32{int32(sysMsgLevelIncreased), int32(sysMsgEarnedExpAndSP)}; !slices.Equal(ids, want) {
		t.Errorf("messages %v, want %v", ids, want)
	}
	if uis := only[testclient.UserInfo](ps); len(uis) == 0 || uis[len(uis)-1].Level != 2 || uis[len(uis)-1].Exp != 68 || uis[len(uis)-1].SP != 2 {
		t.Errorf("UserInfo %+v", uis)
	}
}

// The GM commands of death, levels and karma.
func TestDeathGMCommands(t *testing.T) {
	e := newEnv(t, nil)
	sg, sp, gm, pl := gmPair(t, e)
	e.onLoop(func() { gm.Target = pl })

	gmRun(e, sg, "set_level 10")
	if level, exp, _, _ := e.playerState(pl); level != 10 || exp != 48229 {
		t.Errorf("//set_level 10: level %d exp %d", level, exp)
	}
	if msgs := gmSaid(sg.take(t)); !slices.Equal(msgs, []string{"Added 48229 exp."}) {
		t.Errorf("GM messages %q", msgs)
	}
	gmRun(e, sg, "set_level 5")
	if level, exp, _, _ := e.playerState(pl); level != 5 || exp != e.srv.ExpForLevel(5) {
		t.Errorf("//set_level 5: level %d exp %d", level, exp)
	}
	gmRun(e, sg, "add_level 2")
	if level, exp, _, _ := e.playerState(pl); level != 7 || exp != e.srv.ExpForLevel(7) {
		t.Errorf("//add_level 2: level %d exp %d", level, exp)
	}
	gmRun(e, sg, "add_exp_sp 100 50")
	var spv int32
	e.onLoop(func() { spv = pl.SP() })
	if _, exp, _, _ := e.playerState(pl); exp != e.srv.ExpForLevel(7)+100 || spv != 50 {
		t.Errorf("//add_exp_sp: exp %d sp %d", exp, spv)
	}
	gmRun(e, sg, "remove_exp_sp 100 20")
	e.onLoop(func() { spv = pl.SP() })
	if _, exp, _, _ := e.playerState(pl); exp != e.srv.ExpForLevel(7) || spv != 30 {
		t.Errorf("//remove_exp_sp: exp %d sp %d", exp, spv)
	}
	sp.take(t)
	sg.take(t)

	gmRun(e, sg, "setkarma 100")
	var karma int32
	e.onLoop(func() { karma = pl.Karma() })
	if karma != 100 {
		t.Errorf("//setkarma 100: karma %d", karma)
	}
	if msgs := gmSaid(sg.take(t)); !slices.Equal(msgs, []string{"Successfully Changed karma for Player from (0) to (100)."}) {
		t.Errorf("GM messages %q", msgs)
	}
	gmRun(e, sg, "setkarma 0")

	gmRun(e, sg, "set_pvp_flag")
	if f := flagOf(e, pl); f != 1 {
		t.Errorf("//set_pvp_flag: %d", f)
	}
	gmRun(e, sg, "set_pvp_flag")
	if f := flagOf(e, pl); f != 0 {
		t.Errorf("//set_pvp_flag again: %d", f)
	}

	// Level 20 is not lucky: //kill takes exp, //res gives it back.
	gmRun(e, sg, "set_level 20")
	sp.take(t)
	gmRun(e, sg, "kill")
	if _, exp, before, dead := e.playerState(pl); !dead || before != e.srv.ExpForLevel(20) || exp >= before {
		t.Errorf("//kill: dead %v exp %d before %d", dead, exp, before)
	}
	checkDie(t, sp.take(t), pl, false)
	gmRun(e, sg, "res")
	if level, exp, before, dead := e.playerState(pl); dead || level != 20 || exp != e.srv.ExpForLevel(20) || before != 0 {
		t.Errorf("//res: dead %v level %d exp %d before %d", dead, level, exp, before)
	}
	if ps := sp.take(t); len(only[testclient.Revive](ps)) != 1 {
		t.Errorf("//res sent %q", testclient.Names(ps))
	}

	e.onLoop(func() { gm.Target = nil })
	gmRun(e, sg, "set_level 3")
	if ps := sg.take(t); !slices.ContainsFunc(only[testclient.SystemMessage](ps), func(m testclient.SystemMessage) bool { return m.ID == int32(gmSysMsgIncorrectTarget) }) {
		t.Errorf("//set_level without a target: %q", testclient.Names(ps))
	}
}

// Karma, kills, the death penalty level and the exp before death are saved.
func TestDeathStatePersists(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{})
	_, p := e.enterDeathChar(deathChar{account: "alice", name: "Alice", level: 10, exp: 60000, karma: 300, pk: 2})
	e.killWith(newFakeRand(1), p, deathGremlin(e)) // the penalty roll succeeds
	var c storage.Character
	e.onLoop(func() {
		p.char.PvPKills = 3
		c = p.Snapshot()
	})
	if c.Karma != 225 || c.PKKills != 2 || c.PvPKills != 3 || c.DeathPenaltyLevel != 1 || c.ExpBeforeDeath != 60000 || c.Exp >= 60000 {
		t.Errorf("saved karma %d PK %d PvP %d penalty %d before %d exp %d", c.Karma, c.PKKills, c.PvPKills, c.DeathPenaltyLevel, c.ExpBeforeDeath, c.Exp)
	}
}
