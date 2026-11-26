package game

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/olympiad"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The day of the matches: Thursday 2026-11-19; the competition starts at
// 18:00 UTC.
var olyCompDay = time.Date(2026, 11, 19, 18, 0, 0, 0, time.UTC)

// twoNobleConfig lowers the waiting nobles a match needs to two (the demo).
func twoNobleConfig(o *config.Olympiad) { o.ClassedParticipants, o.NonClassedParticipants = 2, 2 }

// TestOlympiadMatch runs a non-classed and a classed match from the
// registration to the way home: the countdowns, the stadium, a kill, the
// points, the noblesse gate passes, the saved fight, and a spectator who
// watched through the manager.
func TestOlympiadMatch(t *testing.T) {
	for _, tc := range []struct {
		name    string
		classed bool
		join    string
		draws   []rnd.Draw
		reward  int64
		points  [2]int // winner, loser after the match
	}{
		// Two non-classed nobles: two draws pick Alice then Bob.
		{"non-classed", false, "OlympiadNoble 4", []rnd.Draw{rnd.Get(2, 0), rnd.Get(1, 0)}, 30, [2]int{21, 15}},
		// Two classed Duelists: a class drawn for the loop's test, the
		// branch's test and the match, then the two nobles; the next stadium's
		// test draws again.
		{"classed", true, "OlympiadNoble 5", []rnd.Draw{rnd.Get(1, 0), rnd.Get(1, 0), rnd.Get(1, 0), rnd.Get(2, 0), rnd.Get(1, 0), rnd.Get(1, 0)}, 50, [2]int{24, 12}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, store := newOlympiadEnv(t, olyEnvOptions{start: olyCompDay.Add(-time.Minute), configure: twoNobleConfig})
			sa, alice := e.enterNoble("alice", classDuelist, true)
			sb, bob := e.enterNoble("bob", classDuelist, true)
			sc, carol := e.enterNoble("carol", classHumanFighter, false)
			origin := e.position(alice)
			n, ps := e.olyManager(sa, alice)
			if page := onePage(t, ps); !strings.Contains(page.HTML, "noble main") {
				t.Fatalf("a noble's first page: %q", page.HTML)
			}

			// Before 18:00 the games are not in progress.
			ps = e.olyBypass(sa, alice, n, tc.join)
			if ids := sysMsgIDs(ps); len(ids) != 1 || ids[0] != int32(olympiad.MsgNotInProgress) {
				t.Errorf("before the competition: messages %v", ids)
			}

			// 18:00: everyone is told; registration opens.
			e.advance(time.Minute)
			if !hasSysMsg(sc.take(t), int32(olympiad.MsgCompetitionStarted), "", -1) {
				t.Error("no competition start message")
			}
			want := olympiad.MsgRegisteredNonClassed
			if tc.classed {
				want = olympiad.MsgRegisteredClassed
			}
			for _, x := range []struct {
				sess *fakeSession
				p    *Player
			}{{sa, alice}, {sb, bob}} {
				if ps := e.olyBypass(x.sess, x.p, n, tc.join); !hasSysMsg(ps, int32(want), "", -1) {
					t.Errorf("%s registers: %v", x.p.Name(), sysMsgIDs(ps))
				}
			}
			// A second registration is refused.
			if ps := e.olyBypass(sa, alice, n, tc.join); len(sysMsgIDs(ps)) != 1 || sysMsgIDs(ps)[0] == int32(want) {
				t.Errorf("registering again: %v", sysMsgIDs(ps))
			}
			page := onePage(t, e.olyBypass(sa, alice, n, "OlympiadNoble 2"))
			if wantList := map[bool]string{false: "waiting 0 0 2", true: "waiting 2 0 0"}[tc.classed]; !strings.Contains(page.HTML, wantList) {
				t.Errorf("waiting list page %q, want %q", page.HTML, wantList)
			}
			// Carol is no noble.
			n2, _ := e.olyManager(sc, carol)
			if page := onePage(t, e.bypass(sc, "npc_"+strconv.Itoa(int(n2.ID()))+"_Olympiad 1")); !strings.Contains(page.HTML, "arenas [") {
				t.Errorf("arena list %q", page.HTML)
			}

			// 18:01: the manager's round makes the match; the countdown starts.
			r := rnd.NewScripted(t, tc.draws...)
			e.onLoop(func() { e.srv.olympiad.rnd = r })
			e.advance(time.Minute)
			e.sync()
			r.Done()
			if !hasSysMsg(sa.take(t), int32(olympiad.MsgMovedToStadium), "", 120) || !hasSysMsg(sb.take(t), int32(olympiad.MsgMovedToStadium), "", 120) {
				t.Fatal("no 120 s countdown")
			}
			var registered bool
			e.onLoop(func() { registered = e.srv.olympiadRegistered(alice) || e.srv.olympiadRegistered(bob) })
			if registered {
				t.Error("still on the waiting list after the pairing")
			}
			// Mobius checks the waiting list before the games: a picked noble
			// is "not registered".
			if ps := e.olyBypass(sa, alice, n, "OlympiadNoble 1"); !hasSysMsg(ps, int32(olympiad.MsgNotRegistered), "", -1) {
				t.Errorf("cancelling in a game: %v", sysMsgIDs(ps))
			}

			// 120 s later both go to the first stadium.
			e.advance(120 * time.Second)
			for _, sess := range []*fakeSession{sa, sb} {
				ps := sess.take(t)
				if modes := only[testclient.ExOlympiadMode](ps); len(modes) != 1 || modes[0].Mode != 2 {
					t.Errorf("ExOlympiadMode at the stadium: %+v", modes)
				}
				e.appear(sess)
			}
			if pa, pb := e.position(alice), e.position(bob); pa.X != olyStadium0.X+900 || pb.X != olyStadium0.X-900 || pa.Y != olyStadium0.Y {
				t.Errorf("at the stadium: %v and %v", pa, pb)
			}
			var mode, start bool
			e.onLoop(func() { mode, start = alice.InOlympiadMode(), e.srv.olyState(alice).start })
			if !mode || start {
				t.Errorf("olympiad mode %v, started %v", mode, start)
			}
			// Before the start they cannot attack each other.
			var attackable bool
			e.onLoop(func() { attackable = bob.autoAttackableBy(alice) })
			if attackable {
				t.Error("attackable before the match started")
			}

			// Carol watches arena 1 through the manager.
			e.talk(sc, carol, n2)
			e.bypass(sc, "npc_"+strconv.Itoa(int(n2.ID()))+"_Olympiad 1")
			ps = e.bypass(sc, "npc_"+strconv.Itoa(int(n2.ID()))+"_Olympiad 3_0")
			if modes := only[testclient.ExOlympiadMode](ps); len(modes) != 1 || modes[0].Mode != 3 {
				t.Errorf("the spectator's mode: %+v (%v)", modes, testclient.Names(ps))
			}
			e.appear(sc)
			var observing bool
			e.onLoop(func() { observing = carol.InObserverMode() && carol.Invisible() })
			if !observing {
				t.Error("Carol is no invisible observer")
			}
			e.srv.HandleEx(sc, clientpackets.ExOpRequestOlympiadMatchList, nil)
			e.sync()
			if page := onePage(t, sc.take(t)); !strings.Contains(page.HTML, "list Standby Alice / Bob Initial State") {
				t.Errorf("the observer's arena list %q", page.HTML)
			}

			// 5 s, then the 60 s countdown; the fight starts.
			e.advance(5 * time.Second)
			if !hasSysMsg(sc.take(t), int32(olympiad.MsgMatchStartsIn), "", 60) {
				t.Error("the spectator did not get the start countdown")
			}
			e.advance(60 * time.Second)
			for _, sess := range []*fakeSession{sa, sc} {
				ps := sess.take(t)
				if !hasSysMsg(ps, int32(olympiad.MsgMatchStarted), "", -1) {
					t.Errorf("no 'fight!' in %v", testclient.Names(ps))
				}
				if len(only[testclient.ExOlympiadUserInfo](ps)) == 0 {
					t.Errorf("no fighter bar in %v", testclient.Names(ps))
				}
			}
			e.onLoop(func() { attackable = bob.autoAttackableBy(alice) })
			if !attackable {
				t.Error("not attackable after the start")
			}

			// Alice kills Bob; the next check ends the fight.
			e.onLoop(func() { bob.ReduceHP(bob.CurHP+bob.CurCP+1, alice) })
			e.advance(10 * time.Second)
			ps = sa.take(t)
			if !hasSysMsg(ps, int32(olympiad.MsgYouWin), "Alice", -1) || !hasSysMsg(ps, int32(olympiad.MsgEarnedPoints), "Alice", int64(tc.points[0]-18)) ||
				!hasSysMsg(ps, int32(olympiad.MsgLostPoints), "Bob", int64(18-tc.points[1])) || !hasSysMsg(ps, 53, "", tc.reward) ||
				!hasSysMsg(ps, int32(olympiad.MsgBackToTown), "", 40) {
				t.Errorf("the winner's messages: %v", sysMsgIDs(ps))
			}
			if !hasSysMsg(sc.take(t), int32(olympiad.MsgYouWin), "Alice", -1) {
				t.Error("the spectator did not see the result")
			}
			if got := e.countOf(alice, itemGatePass); got != tc.reward {
				t.Errorf("gate passes %d, want %d", got, tc.reward)
			}
			var a, b olympiad.Noble
			e.onLoop(func() {
				a, b = *e.srv.olympiad.nobles.Get(alice.ObjectID()), *e.srv.olympiad.nobles.Get(bob.ObjectID())
			})
			if a.Points != tc.points[0] || a.Won != 1 || a.Done != 1 || b.Points != tc.points[1] || b.Lost != 1 || b.Done != 1 {
				t.Errorf("nobles after the match: %+v %+v", a, b)
			}

			// 40 s later both go home, Bob alive and healed.
			e.advance(40 * time.Second)
			for _, sess := range []*fakeSession{sa, sb} {
				ps := sess.take(t)
				if len(only[testclient.ExOlympiadMatchEnd](ps)) != 1 {
					t.Errorf("no ExOlympiadMatchEnd in %v", testclient.Names(ps))
				}
				if modes := only[testclient.ExOlympiadMode](ps); len(modes) != 1 || modes[0].Mode != 0 {
					t.Errorf("ExOlympiadMode at the end: %+v", modes)
				}
				e.appear(sess)
			}
			if len(only[testclient.ExOlympiadMatchEnd](sc.take(t))) != 1 {
				t.Error("the spectator got no ExOlympiadMatchEnd")
			}
			var dead bool
			var hp, maxHP float64
			e.onLoop(func() { dead, hp, maxHP, mode = bob.IsDead(), bob.CurHP, bob.MaxHPValue(), bob.InOlympiadMode() })
			if dead || hp != maxHP || mode {
				t.Errorf("Bob after the match: dead %v, HP %v of %v, olympiad mode %v", dead, hp, maxHP, mode)
			}
			if pa := e.position(alice); pa.Distance2D(origin) > 100 {
				t.Errorf("Alice is back at %v, want near %v", pa, origin)
			}

			// The fight is stored, the state too.
			e.waitOlympiadWrites()
			fights, err := store.OlympiadFights(t.Context(), alice.ObjectID(), 1<<62)
			if err != nil || len(fights) != 1 || fights[0].Winner != 1 || fights[0].Classed != tc.classed || fights[0].CharTwoName != "Bob" {
				t.Errorf("stored fights %+v, %v", fights, err)
			}
			st, err := store.LoadOlympiad(t.Context())
			if err != nil || len(st.Nobles) != 2 {
				t.Errorf("stored state %+v, %v", st, err)
			}

			// Carol leaves the stadium.
			e.srv.HandleEx(sc, clientpackets.ExOpRequestOlympiadObserverEnd, nil)
			e.sync()
			if modes := only[testclient.ExOlympiadMode](sc.take(t)); len(modes) != 1 || modes[0].Mode != 0 {
				t.Errorf("leaving the observation: %+v", modes)
			}
			e.appear(sc)
			e.onLoop(func() { observing = carol.InObserverMode() || carol.Invisible() })
			if observing {
				t.Error("Carol still observes")
			}
		})
	}
}

// A noble that leaves after reaching the stadium loses: the other gets the
// points but no reward, and goes home at once.
func TestOlympiadDisconnect(t *testing.T) {
	// 18:01: the competition runs (it starts 10 ms after the server).
	e, _ := newOlympiadEnv(t, olyEnvOptions{start: olyCompDay.Add(time.Minute), configure: twoNobleConfig})
	e.advance(time.Second)
	sa, alice := e.enterNoble("alice", classDuelist, true)
	sb, bob := e.enterNoble("bob", classAdventurer, true)
	e.onLoop(func() {
		e.srv.olympiad.rnd = rnd.NewScripted(t, rnd.Get(2, 0), rnd.Get(1, 0))
		e.srv.registerNoble(alice, false)
		e.srv.registerNoble(bob, false)
	})
	// The round pairs them; two minutes later they are in the stadium.
	e.advance(time.Minute)
	e.advance(2 * time.Minute)
	e.appear(sa)
	e.appear(sb)
	bobID := bob.ObjectID()
	e.srv.Logout(sb)
	e.sync()
	e.advance(65 * time.Second) // the fight would start
	ps := sa.take(t)
	if !hasSysMsg(ps, int32(olympiad.MsgYouWin), "Alice", -1) || hasSysMsg(ps, int32(olympiad.MsgBackToTown), "", 40) {
		t.Errorf("the winner of a crash: %v", sysMsgIDs(ps))
	}
	if len(only[testclient.ExOlympiadMatchEnd](ps)) != 1 {
		t.Errorf("no way home at once: %v", testclient.Names(ps))
	}
	var a, b olympiad.Noble
	e.onLoop(func() { a, b = *e.srv.olympiad.nobles.Get(alice.ObjectID()), *e.srv.olympiad.nobles.Get(bobID) })
	if a.Points != 21 || a.Won != 1 || b.Points != 15 || b.Lost != 1 || b.Done != 1 {
		t.Errorf("nobles after a crash: %+v %+v", a, b)
	}
	if got := e.countOf(alice, itemGatePass); got != 0 {
		t.Errorf("a crash gave %d gate passes", got)
	}
}

// The rules of registration in the game: a noble on a subclass or without
// its third class gets its pages; ten minutes before the end registration
// is closed; the competition ends at midnight and the next starts the next
// day.
func TestOlympiadRegistrationWindow(t *testing.T) {
	e, _ := newOlympiadEnv(t, olyEnvOptions{start: olyCompDay.Add(5*time.Hour + 51*time.Minute)})
	e.advance(time.Second) // the running competition starts 10 ms after the server
	sa, alice := e.enterNoble("alice", classDuelist, true)
	sb, bob := e.enterNoble("bob", classHumanFighter, true)
	n, _ := e.olyManager(sa, alice)
	if page := onePage(t, e.olyBypass(sb, bob, n, "OlympiadNoble 3")); !strings.Contains(page.HTML, "not a third class noble") {
		t.Errorf("a noble of a first class: %q", page.HTML)
	}
	if page := onePage(t, e.olyBypass(sa, alice, n, "OlympiadNoble 3")); !strings.Contains(page.HTML, "points 0") {
		t.Errorf("points of a new noble: %q", page.HTML)
	}
	if page := onePage(t, e.olyBypass(sa, alice, n, "OlympiadDesc 2a")); !strings.Contains(page.HTML, "desc 2a") {
		t.Errorf("OlympiadDesc 2a: %q", page.HTML)
	}
	// 23:51: the competition runs, but registration closed at 23:50.
	ps := e.olyBypass(sa, alice, n, "OlympiadNoble 4")
	if ids := sysMsgIDs(ps); len(ids) != 1 || ids[0] != int32(olympiad.MsgNoRegistrationLast10) {
		t.Errorf("registering at 23:51: %v", ids)
	}
	// Midnight: the competition ends.
	e.advance(9 * time.Minute)
	if !hasSysMsg(sa.take(t), int32(olympiad.MsgCompetitionEnded), "", -1) {
		t.Error("no competition end message")
	}
	var inComp bool
	var next time.Time
	e.onLoop(func() { inComp, next = e.srv.olympiad.inComp, e.srv.olympiad.compStart })
	// The next start keeps the seconds of the moment it was set, as Java's
	// Calendar does.
	if inComp || !next.Truncate(time.Minute).Equal(olyCompDay.Add(24*time.Hour)) {
		t.Errorf("after midnight: in competition %v, next start %v", inComp, next)
	}
}

// The calendar of the Olympiad on the fake clock and across restarts: the
// weekly points, the period's end with its heroes (and the old hero's loss
// of status and items), the claim, hero items and skills, the validation
// period with the passes, the next cycle, and a missed period end.
func TestOlympiadPeriodAndHeroes(t *testing.T) {
	store := memstore.NewOlympiad()
	// The characters first, on a server with an empty Olympiad.
	e0, _ := newOlympiadEnv(t, olyEnvOptions{store: memstore.NewOlympiad()})
	var ids [4]int32
	for i, acc := range []string{"xena", "yuri", "zoe", "walt"} {
		items := []storage.Item(nil)
		if acc == "walt" {
			items = []storage.Item{{ItemID: itemInfinityBlade, Count: 1, Slot: storage.PaperdollRHand, Mana: storage.NoMana}}
		}
		c := e0.createClanChar(clanChar{account: acc, name: strings.ToUpper(acc[:1]) + acc[1:], level: 76, class: classHumanFighter, items: items})
		// Noble Duelists.
		c.Noble, c.ClassID, c.BaseClassID = true, classDuelist, classDuelist
		if err := e0.store.Update(e0.ctx, &c); err != nil {
			t.Fatal(err)
		}
		ids[i] = c.ObjectID
	}
	e0.stop()
	end := time.Date(2026, 11, 30, 12, 0, 0, 0, time.UTC)
	if err := store.SaveOlympiad(t.Context(), storage.Olympiad{
		Cycle: 3, OlympiadEnd: end.UnixMilli(), NextWeeklyChange: end.Add(-30 * time.Second).UnixMilli(),
		Nobles: []storage.OlympiadNoble{
			{CharID: ids[0], ClassID: classDuelist, Name: "Xena", Points: 40, Done: 10, Won: 5, Lost: 5},
			{CharID: ids[1], ClassID: classDuelist, Name: "Yuri", Points: 30, Done: 9, Won: 2, Lost: 7},
			{CharID: ids[2], ClassID: classAdventurer, Name: "Zoe", Points: 30, Done: 2, Won: 2},
		},
		Heroes: []storage.Hero{{CharID: ids[3], ClassID: 90, Name: "Walt", Count: 1, Played: true, Claimed: true}},
	}); err != nil {
		t.Fatal(err)
	}

	e1, _ := newOlympiadEnv(t, olyEnvOptions{prev: e0, store: store, start: end.Add(-time.Minute)})
	sw, walt := e1.clanLogin("walt")
	sx, xena := e1.clanLogin("xena")
	var waltHero, waltSkill, bladeOn bool
	e1.onLoop(func() {
		waltHero, waltSkill = walt.IsHero(), walt.SkillLevel(heroSkillID) > 0
		it := walt.inv.Paperdoll(storage.PaperdollRHand)
		bladeOn = it != nil && it.ItemID() == itemInfinityBlade
	})
	if !waltHero || !waltSkill || !bladeOn {
		t.Errorf("the old hero at login: hero %v, hero skill %v, blade worn %v", waltHero, waltSkill, bladeOn)
	}

	// The weekly points, clamped: Zoe (2 matches) may have 32 at most.
	e1.advance(30 * time.Second)
	var pts [3]int
	e1.onLoop(func() {
		for i := range pts {
			pts[i] = e1.srv.olympiad.nobles.Points(ids[i])
		}
	})
	if pts != [3]int{43, 33, 32} {
		t.Errorf("after the weekly points: %v, want 43 33 32", pts)
	}

	// 12:00: the period ends. Xena is the hero to be; Walt loses the status
	// and his blade.
	e1.advance(30 * time.Second)
	ps := sx.take(t)
	if !hasSysMsg(ps, int32(olympiad.MsgPeriodEnded), "", 3) {
		t.Errorf("no period end message: %v", sysMsgIDs(ps))
	}
	var period olympiad.Period
	var xenaUnclaimed, xenaHero bool
	e1.onLoop(func() {
		period = e1.srv.olympiad.state.Period
		xenaUnclaimed, xenaHero = e1.srv.olympiad.heroes.IsUnclaimed(ids[0]), xena.IsHero()
		waltHero, waltSkill = walt.IsHero(), walt.SkillLevel(heroSkillID) > 0
	})
	if period != olympiad.Validation || !xenaUnclaimed || xenaHero {
		t.Errorf("after the end: period %v, Xena unclaimed %v hero %v", period, xenaUnclaimed, xenaHero)
	}
	if waltHero || waltSkill || e1.countOf(walt, itemInfinityBlade) != 0 {
		t.Errorf("Walt after the end: hero %v, skill %v, blades %d", waltHero, waltSkill, e1.countOf(walt, itemInfinityBlade))
	}
	sw.take(t)

	// Xena claims (//givehero by a GM), takes a blade and wears it.
	sg, gm := e1.enterClanChar(clanChar{account: "gm", name: "Gm", class: classHumanFighter, access: storage.AccessLevelAdmin})
	e1.onLoop(func() { e1.srv.SetTarget(gm, xena) })
	gmRun(e1, sg, "givehero")
	e1.onLoop(func() { xenaHero = xena.IsHero() && xena.SkillLevel(heroSkillID) > 0 })
	if !xenaHero {
		t.Fatal("Xena is no hero after //givehero")
	}
	sg.take(t)
	gmRun(e1, sg, "givehero")
	if says := gmSaid(sg.take(t)); len(says) != 1 || says[0] != "This player has already claimed the hero status." {
		t.Errorf("//givehero again: %q", says)
	}
	var blade int32
	e1.onLoop(func() {
		it, err := e1.srv.AddItem(xena, itemInfinityBlade, 1, -1, false, false)
		if err != nil {
			t.Error(err)
			return
		}
		blade = it.ObjectID()
	})
	sx.take(t)
	useItemOn(e1, sx, blade)
	e1.onLoop(func() {
		it := xena.inv.Paperdoll(storage.PaperdollRHand)
		bladeOn = it != nil && it.ItemID() == itemInfinityBlade
	})
	if !bladeOn {
		t.Error("the hero cannot wear the Infinity Blade")
	}
	e1.waitOlympiadWrites()
	e1.stop()
	st, err := store.LoadOlympiad(t.Context())
	if err != nil || st.Period != 1 || len(st.NoblesEOM) != 3 || len(st.Heroes) != 2 {
		t.Fatalf("stored after the end: %+v, %v", st, err)
	}
	for _, h := range st.Heroes {
		if (h.CharID == ids[0]) != (h.Played && h.Claimed) || (h.CharID == ids[3] && h.Played) {
			t.Errorf("stored hero %+v", h)
		}
	}

	// The period's end took every ranked noble's points (Mobius
	// getNoblessePasses) before anyone could claim: nobody earned passes.
	for _, n := range st.Nobles {
		if n.Points != 0 && n.Done >= 9 {
			t.Errorf("a ranked noble kept its points: %+v", n)
		}
	}
	if len(st.Unclaimed) != 0 {
		t.Errorf("unclaimed passes %+v", st.Unclaimed)
	}
	// Passes a hero earned (100 hero points × 1000, as a restart in the
	// validation period after the claim would give) are settled × 1000
	// again, as Mobius does.
	st.Unclaimed = []storage.UnclaimedPasses{{CharID: ids[0], Passes: 100_000}}
	if err := store.SaveOlympiad(t.Context(), st); err != nil {
		t.Fatal(err)
	}

	// A restart in the validation period: Xena is still a hero.
	e2, _ := newOlympiadEnv(t, olyEnvOptions{prev: e1, store: store, start: end.Add(23 * time.Hour)})
	sx2, xena2 := e2.clanLogin("xena")
	e2.onLoop(func() { xenaHero = xena2.IsHero() && xena2.SkillLevel(heroSkillID) > 0 })
	if !xenaHero {
		t.Error("Xena is no hero after the restart")
	}
	n, _ := e2.olyManager(sx2, xena2)
	if page := onePage(t, e2.olyBypass(sx2, xena2, n, "OlympiadNoble 6")); !strings.Contains(page.HTML, "settle") {
		t.Errorf("settle page %q", page.HTML)
	}
	if page := onePage(t, e2.olyBypass(sx2, xena2, n, "OlympiadNoble 9")); !strings.Contains(page.HTML, "last points 43") {
		t.Errorf("last period's points %q", page.HTML)
	}
	e2.talk(sx2, xena2, n)
	e2.bypass(sx2, "npc_"+strconv.Itoa(int(n.ID()))+"_OlympiadNoble 6")
	e2.bypass(sx2, "npc_"+strconv.Itoa(int(n.ID()))+"_OlympiadNoble 10")
	if got := e2.countOf(xena2, itemGatePass); got != 100_000_000 {
		t.Errorf("settled passes %d", got)
	}
	// The hero list.
	e2.talk(sx2, xena2, n)
	ps = e2.bypass(sx2, "npc_"+strconv.Itoa(int(n.ID()))+"_Olympiad 4")
	if l := only[testclient.ExHeroList](ps); len(l) != 1 || len(l[0].Heroes) != 1 || l[0].Heroes[0].Name != "Xena" || l[0].Heroes[0].Count != 1 {
		t.Errorf("hero list %+v", l)
	}
	// The validation ends an hour later: cycle 4 without nobles.
	e2.advance(time.Hour)
	var state olympiad.State
	var nobles int
	e2.onLoop(func() { state, nobles = e2.srv.olympiad.state, e2.srv.olympiad.nobles.Len() })
	if state.Cycle != 4 || state.Period != olympiad.Competition || nobles != 0 ||
		!state.OlympiadEnd.Equal(time.Date(2027, 1, 1, 12, 0, 0, 0, time.UTC)) {
		t.Errorf("after the validation: %+v, %d nobles", state, nobles)
	}
	e2.stop()

	// The server was down over the next period's end: Mobius only sets a new
	// end; the heroes stay.
	e3, _ := newOlympiadEnv(t, olyEnvOptions{prev: e2, store: store, start: time.Date(2027, 1, 1, 13, 0, 0, 0, time.UTC)})
	var heroID bool
	e3.onLoop(func() { state, heroID = e3.srv.olympiad.state, e3.srv.olympiad.heroes.IsHero(ids[0]) })
	if state.Cycle != 4 || !state.OlympiadEnd.Equal(time.Date(2027, 2, 1, 12, 0, 0, 0, time.UTC)) || !heroID {
		t.Errorf("after a missed end: %+v, Xena hero %v", state, heroID)
	}
}

// The GM commands: //checkolympiad, //saveolymp and //endolympiad.
func TestOlympiadGM(t *testing.T) {
	e, store := newOlympiadEnv(t, olyEnvOptions{start: time.Date(2026, 11, 19, 13, 0, 0, 0, time.UTC)})
	sg, gm := e.enterClanChar(clanChar{account: "gm", name: "Gm", class: classHumanFighter, access: storage.AccessLevelAdmin})
	gmRun(e, sg, "checkolympiad")
	page := onePage(t, sg.take(t))
	for _, want := range []string{"Olympiad Period Ends:", "01/12/2026 12:00:00", "Next Weekly Change:", "26/11/2026 13:00:00", "11 days, 23 hours"} {
		if !strings.Contains(page.HTML, want) {
			t.Errorf("//checkolympiad page lacks %q: %s", want, page.HTML)
		}
	}
	gmRun(e, sg, "saveolymp")
	if says := gmSaid(sg.take(t)); len(says) != 1 || says[0] != "olympiad system saved." {
		t.Errorf("//saveolymp: %q", says)
	}
	e.onLoop(func() { e.srv.olympiad.nobles.Add(e.srv.olympiad.cfg, gm.ObjectID(), classDuelist, "Gm").Done = 9 })
	gmRun(e, sg, "endolympiad")
	e.sync()
	if says := gmSaid(sg.take(t)); len(says) == 0 || says[0] != "Heroes formed." {
		t.Errorf("//endolympiad: %q", says)
	}
	var period olympiad.Period
	e.onLoop(func() { period = e.srv.olympiad.state.Period })
	if period != olympiad.Validation {
		t.Errorf("period after //endolympiad: %v", period)
	}
	e.onLoop(func() { e.srv.SetTarget(gm, gm) })
	gmRun(e, sg, "givehero")
	if says := gmSaid(sg.take(t)); len(says) != 1 || says[0] != "This player cannot claim the hero status." {
		t.Errorf("//givehero for no hero: %q", says)
	}
	e.waitOlympiadWrites()
	if st, err := store.LoadOlympiad(t.Context()); err != nil || st.Period != 1 || st.Cycle != 1 {
		t.Errorf("stored after //endolympiad: %+v, %v", st, err)
	}
}

// An olympiad stadium sends anyone but a fighter, an observer or a GM to
// town.
func TestOlympiadStadiumZone(t *testing.T) {
	e, _ := newOlympiadEnv(t, olyEnvOptions{})
	sa, alice := e.enterNoble("alice", classDuelist, false)
	e.onLoop(func() {
		if err := e.srv.Teleport(alice, world.Position{X: olyStadium0.X, Y: olyStadium0.Y, Z: olyStadium0.Z}); err != nil {
			t.Error(err)
		}
	})
	e.appear(sa)
	e.advance(time.Second)
	e.appear(sa)
	if pos := e.position(alice); pos.Distance2D(olyStadium0) < 1000 {
		t.Errorf("an intruder stays in the stadium at %v", pos)
	}
}
