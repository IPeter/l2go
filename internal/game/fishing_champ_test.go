package game

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/fishing"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
)

// The week of the championship: Tuesday 2026-11-24 19:00 UTC ends it.
var fishWeekEnd = time.Date(2026, 11, 24, 19, 0, 0, 0, time.UTC)

// fishPage returns the text of the only page in ps.
func fishPage(t *testing.T, ps []testclient.Packet) string {
	t.Helper()
	return onePage(t, ps).HTML
}

// klufe spawns the fisherman next to p and talks to him.
func (e *testEnv) klufe(sess *fakeSession, p *Player) *Npc {
	e.t.Helper()
	n := e.spawnNpc(fishTestFisherman, eastOf(e.position(p), 30))
	e.talk(sess, p, n)
	return n
}

// TestFishingChampionshipWeek runs a championship week on the fake clock
// and across restarts: the ranking, the end at Tuesday 19:00, the prizes,
// a claim kept over a restart, and a missed end replayed at start.
func TestFishingChampionshipWeek(t *testing.T) {
	store := memstore.NewFishing()
	seed := storage.FishingChampionship{End: fishWeekEnd.UnixMilli(), Fishers: []storage.ChampionshipFisher{
		{Name: "Bobby", Length: 70, Rewarded: 0},
		{Name: "Angler", Length: 80.25, Rewarded: 0},
	}}
	if err := store.SaveFishingChampionship(t.Context(), seed); err != nil {
		t.Fatal(err)
	}

	// Ten minutes before the end.
	e, _ := newFishingEnv(t, fishEnvOptions{store: store, start: fishWeekEnd.Add(-10 * time.Minute)})
	sess, p := e.enterFisher(fisher{account: "angler", level: 10})
	n := e.klufe(sess, p)
	npc := strconv.Itoa(int(n.ID()))

	page := fishPage(t, e.bypass(sess, "npc_"+npc+"_FishingChampionship"))
	wantTable := "null" + fishRow(1, "None", "0") + fishRow(2, "None", "0") + fishRow(3, "None", "0") + fishRow(4, "None", "0") + fishRow(5, "None", "0")
	if want := "winners " + wantTable + " Adena 800000 100000 in 9 npc " + npc; !strings.Contains(page, want) {
		t.Errorf("before the end:\n got %q\nwant %q", page, want)
	}
	// The running ranking: refreshed on the first look, then shown.
	ranking := func() string {
		t.Helper()
		e.srv.HandleEx(sess, clientpackets.ExOpRequestExFishRanking, nil)
		e.sync()
		return fishPage(t, sess.take(t))
	}
	if got := ranking(); !strings.Contains(got, "being updated") {
		t.Errorf("first ranking = %q, want the update page", got)
	}
	if got, want := ranking(), "ranking null"+fishRow(1, "Angler", "80.25")+fishRow(2, "Bobby", "70.0"); !strings.Contains(got, want) {
		t.Errorf("ranking:\n got %q\nwant %q", got, want)
	}

	// Tuesday 19:00: the week ends; the next ends a week later.
	e.advance(10 * time.Minute)
	var end time.Time
	e.onLoop(func() { end = e.srv.fishing.champ.End() })
	if want := fishWeekEnd.AddDate(0, 0, 7); !end.Equal(want) {
		t.Errorf("next end %v, want %v", end, want)
	}
	e.talk(sess, p, n)
	page = fishPage(t, e.bypass(sess, "npc_"+npc+"_FishingChampionship"))
	// A second after the end (the enter took one): 7 days less a second.
	if want := "winners null" + fishRow(1, "Angler", "80.25") + fishRow(2, "Bobby", "70.0"); !strings.Contains(page, want) || !strings.Contains(page, " in 10079 npc ") {
		t.Errorf("after the end:\n got %q\nwant %q and 10079 minutes", page, want)
	}

	// Angler claims the first prize, once.
	ps := e.bypass(sess, "npc_"+npc+"_FishingReward")
	if got := fishPage(t, ps); !strings.Contains(got, "here is your prize") {
		t.Errorf("reward page = %q", got)
	}
	if got := e.countOf(p, adenaID); got != 800000 {
		t.Errorf("adena %d, want 800000", got)
	}
	e.bypass(sess, "npc_"+npc+"_FishingChampionship") // the page with the button again
	noPage(t, "a second claim", e.bypass(sess, "npc_"+npc+"_FishingReward"))
	if got := e.countOf(p, adenaID); got != 800000 {
		t.Errorf("adena after a second claim %d, want 800000", got)
	}
	e.stop()
	got, err := store.LoadFishingChampionship(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	want := storage.FishingChampionship{End: fishWeekEnd.AddDate(0, 0, 7).UnixMilli(), Fishers: []storage.ChampionshipFisher{
		{Name: "Angler", Length: 80.25, Rewarded: fishing.RewardTaken},
		{Name: "Bobby", Length: 70, Rewarded: fishing.RewardDue},
	}}
	if got.End != want.End || !slices.Equal(got.Fishers, want.Fishers) {
		t.Fatalf("stored %+v, want %+v", got, want)
	}

	// A restart in the next week: Bobby claims the second prize; Angler's
	// claim stays done.
	e2, _ := newFishingEnv(t, fishEnvOptions{store: store, start: fishWeekEnd.Add(48 * time.Hour)})
	sess2, bobby := e2.enterFisher(fisher{account: "bobby"})
	n2 := e2.klufe(sess2, bobby)
	if got := fishPage(t, e2.bypass(sess2, fmt.Sprintf("npc_%d_FishingReward", n2.ID()))); !strings.Contains(got, "here is your prize") {
		t.Errorf("Bobby's reward page = %q", got)
	}
	if got := e2.countOf(bobby, adenaID); got != 500000 {
		t.Errorf("Bobby's adena %d, want 500000", got)
	}
	sess3, angler := e2.enterFisher(fisher{account: "angler"})
	e2.talk(sess3, angler, n2)
	noPage(t, "Angler after a restart", e2.bypass(sess3, fmt.Sprintf("npc_%d_FishingReward", n2.ID())))
	// Someone who did not win gets the "no prize" page.
	sess4, stranger := e2.enterFisher(fisher{account: "stranger"})
	e2.talk(sess4, stranger, n2)
	if got := onePage(t, e2.bypass(sess4, fmt.Sprintf("npc_%d_FishingReward", n2.ID()))); !strings.Contains(got.HTML, "no prize for you") || got.ObjectID != int32(n2.ID()) {
		t.Errorf("a stranger's reward page = %+v, want the fisherman's no-prize page", got)
	}
	// A fish of the new week.
	r := e2.fishScript(rnd.Between(60, 89, 61), rnd.Between(0, 1000, 0))
	e2.onLoop(func() { e2.srv.championshipNewFish(stranger, fishTestLure) })
	r.Done()
	e2.stop()

	// The server was down over the next Tuesday: the week ends at start.
	late := fishWeekEnd.AddDate(0, 0, 7).Add(15 * time.Hour) // Wednesday 10:00
	e3, _ := newFishingEnv(t, fishEnvOptions{store: store, start: late})
	var winners []fishing.Fisher
	e3.onLoop(func() {
		end = e3.srv.fishing.champ.End()
		_, winners = e3.srv.fishing.champ.Ranking()
	})
	if want := fishWeekEnd.AddDate(0, 0, 14); !end.Equal(want) {
		t.Errorf("end after a missed week %v, want %v", end, want)
	}
	if len(winners) != 1 || winners[0].Name != "Stranger" || winners[0].Reward != fishing.RewardDue {
		t.Errorf("winners after a missed week %+v, want Stranger", winners)
	}
}

// fishRow is a row of the championship tables.
func fishRow(rank int, name, length string) string {
	return "<tr><td width=70 align=center>" + strconv.Itoa(rank) + "</td><td width=110 align=center>" + name +
		"</td><td width=80 align=center>" + length + "</td></tr>"
}

// TestFishingChampionshipGM checks //fishchamp_end.
func TestFishingChampionshipGM(t *testing.T) {
	e, store := newFishingEnv(t, fishEnvOptions{})
	e.createCharacter("gm", "Gm", classHumanFighter, storage.AccessLevelAdmin)
	sess := newSession("gm")
	p := e.enter(sess, 0)
	r := e.fishScript(rnd.Between(60, 89, 88), rnd.Between(0, 1000, 1))
	e.onLoop(func() { e.srv.championshipNewFish(p, fishTestLure) })
	r.Done()
	sess.take(t)
	gmRun(e, sess, "fishchamp_end")
	var winners []fishing.Fisher
	e.onLoop(func() { _, winners = e.srv.fishing.champ.Ranking() })
	if len(winners) != 1 || winners[0].Name != "Gm" || winners[0].Length != 88.001 {
		t.Errorf("winners after //fishchamp_end %+v", winners)
	}
	e.stop()
	got, err := store.LoadFishingChampionship(t.Context())
	if err != nil || len(got.Fishers) != 1 || got.Fishers[0].Rewarded != fishing.RewardDue {
		t.Errorf("stored %+v, %v", got, err)
	}
}

// TestFishermanSkills learns Fishing at a fisherman: the list of fishing
// skills, the info with its adena, and the skill.
func TestFishermanSkills(t *testing.T) {
	e, _ := newFishingEnv(t, fishEnvOptions{})
	sess, p := e.enterFisher(fisher{account: "pupil"})
	e.onLoop(func() {
		if _, err := e.srv.AddItem(p, adenaID, 2000, -1, false, false); err != nil {
			t.Error(err)
		}
	})
	n := e.klufe(sess, p)
	list := func(ps []testclient.Packet) []string {
		t.Helper()
		var out []string
		for _, l := range only[testclient.AcquireSkillList](ps) {
			if l.Type != 1 {
				t.Errorf("AcquireSkillList type %d, want 1 (fishing)", l.Type)
			}
			for _, s := range l.Skills {
				out = append(out, fmt.Sprintf("%d-%d", s.ID, s.NextLevel))
			}
		}
		slices.Sort(out)
		return out
	}
	if got, want := list(e.bypass(sess, fmt.Sprintf("npc_%d_FishSkillList", n.ID()))), []string{"1312-1", "1313-1", "1314-1", "1315-1"}; !slices.Equal(got, want) {
		t.Errorf("fishing skills %v, want %v", got, want)
	}
	acquire := func(op byte, id, level int32) []testclient.Packet {
		e.srv.Handle(sess, op, ints(id, level, clientpackets.AcquireSkillFishing))
		e.sync()
		return sess.take(t)
	}
	infos := only[testclient.AcquireSkillInfo](acquire(clientpackets.OpRequestAcquireSkillInfo, 1312, 1))
	if len(infos) != 1 || infos[0].Type != 1 || len(infos[0].Requirements) != 1 || infos[0].Requirements[0].ItemID != adenaID || infos[0].Requirements[0].Count != 1000 {
		t.Errorf("AcquireSkillInfo = %+v", infos)
	}
	ps := acquire(clientpackets.OpRequestAcquireSkill, 1312, 1)
	if got, want := list(ps), []string{"1313-1", "1314-1", "1315-1"}; !slices.Equal(got, want) {
		t.Errorf("list after learning %v, want %v", got, want)
	}
	var level int
	e.onLoop(func() { level = p.SkillLevel(fishing.SkillFishing) })
	if level != 1 || e.countOf(p, adenaID) != 1000 {
		t.Errorf("Fishing level %d, adena %d; want 1, 1000", level, e.countOf(p, adenaID))
	}
}
