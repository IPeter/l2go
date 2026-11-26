package game

import (
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
)

// waitHeroPages waits until the background load of a hero's pages arrived.
func (e *testEnv) waitHeroPages(charID int32, fights int) {
	e.t.Helper()
	for range 500 {
		var n int
		e.onLoop(func() { n = len(e.srv.olympiad.fights[charID]) })
		if n >= fights {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	e.t.Fatalf("the hero pages of %d did not load", charID)
}

// A hero's words, diary and fights (the hero list's pages), and the hero
// status of the admin panel's class page.
func TestHeroPages(t *testing.T) {
	store := memstore.NewOlympiad()
	e0, _ := newOlympiadEnv(t, olyEnvOptions{store: memstore.NewOlympiad()})
	c := e0.createClanChar(clanChar{account: "xena", name: "Xena", level: 76, class: classHumanFighter})
	c.Noble, c.ClassID, c.BaseClassID = true, classDuelist, classDuelist
	if err := e0.store.Update(e0.ctx, &c); err != nil {
		t.Fatal(err)
	}
	e0.stop()
	lastMonth := time.Date(2026, 10, 20, 19, 30, 0, 0, time.UTC).UnixMilli()
	for _, f := range []storage.OlympiadFight{
		{CharOneID: c.ObjectID, CharTwoID: 99, CharOneName: "Xena", CharTwoName: "Yuri", CharOneClass: 88, CharTwoClass: 93, Winner: 1, Start: lastMonth, Time: 95000, Classed: true},
		{CharOneID: 98, CharTwoID: c.ObjectID, CharOneName: "Zoe", CharTwoName: "Xena", CharOneClass: 90, CharTwoClass: 88, Winner: 1, Start: lastMonth + 3600000, Time: 360000},
		// This month's fights are not shown (Mobius: start < the first of the month).
		{CharOneID: c.ObjectID, CharTwoID: 98, CharOneName: "Xena", CharTwoName: "Zoe", Winner: 0, Start: testStart.UnixMilli() - 1000, Time: 1},
	} {
		if err := store.AddOlympiadFight(t.Context(), f); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.SaveOlympiad(t.Context(), storage.Olympiad{Cycle: 2,
		OlympiadEnd: time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC).UnixMilli(), NextWeeklyChange: testStart.Add(time.Hour).UnixMilli(),
		Heroes: []storage.Hero{{CharID: c.ObjectID, ClassID: classDuelist, Name: "Xena", Count: 1, Played: true}}}); err != nil {
		t.Fatal(err)
	}
	e, _ := newOlympiadEnv(t, olyEnvOptions{prev: e0, store: store})
	sx, xena := e.clanLogin("xena")
	e.waitHeroPages(c.ObjectID, 2)

	// Unclaimed, Xena has no words page yet; the fights show.
	ps := e.bypass(sx, "_match?class=88&page=1")
	page := onePage(t, ps).HTML
	for _, want := range []string{"Xena 1 1 0", "2026-10-20 19:30", "vs Yuri (", "(01:35)", `<font color="00ff00">victory</font>`, `<font color="ff0000">loss</font>`, "vs Zoe ("} {
		if !strings.Contains(page, want) {
			t.Errorf("fights page lacks %q: %s", want, page)
		}
	}
	noPage(t, "the diary of an unclaimed hero", e.bypass(sx, "_diary?class=88&page=1"))

	// The admin panel's Hero On is the real status: claimed, kept by the
	// hero table; the words follow.
	sg, gm := e.enterClanChar(clanChar{account: "gm", name: "Gm", class: classHumanFighter, access: storage.AccessLevelAdmin})
	e.onLoop(func() { e.srv.SetTarget(gm, xena) })
	gmRun(e, sg, "class_menu hero on")
	var hero, claimed bool
	e.onLoop(func() { hero, claimed = xena.IsHero(), e.srv.olympiad.heroes.IsHero(xena.ObjectID()) })
	if !hero || !claimed {
		t.Fatalf("after Hero On: hero %v, claimed %v", hero, claimed)
	}
	e.srv.HandleEx(sx, clientpackets.ExOpRequestWriteHeroWords, stringBody("<b>Fear</b> me"))
	e.sync()
	page = onePage(t, e.bypass(sx, "_diary?class=88&page=1")).HTML
	if !strings.Contains(page, "Xena says Fear me") || !strings.Contains(page, "Gained Hero status") {
		t.Errorf("diary page %s", page)
	}

	// Hero Off takes it away.
	gmRun(e, sg, "class_menu hero off")
	e.onLoop(func() { hero, claimed = xena.IsHero(), e.srv.olympiad.heroes.IsHero(xena.ObjectID()) })
	if hero || claimed {
		t.Errorf("after Hero Off: hero %v, claimed %v", hero, claimed)
	}
	sx.take(t)
	e.onLoop(func() { xena.Send(e.srv.heroListPacket()) })
	if l := only[testclient.ExHeroList](sx.take(t)); len(l) != 1 || len(l[0].Heroes) != 0 {
		t.Errorf("hero list after Hero Off: %+v", l)
	}
}
