package game

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/quest"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
)

// The quest scenarios run the hand-ported Q001 and Q401 of scripts/ on a
// server with fixture pages: every page of a quest says its name
// ("PAGE <name>") and offers every event link of the quest, so any click
// of the walkthrough is in the link cache.

const (
	q001 = "Q00001_LettersOfLove"
	q401 = "Q00401_PathOfTheWarrior"

	npcDarin   = 30048
	npcBaulro  = 30033
	npcAuron   = 30010
	npcSimplon = 30253

	itemDarinLetter   = 687
	itemRoxxyKerchief = 688
	itemDarinReceipt  = 1079
	itemBaulroPotion  = 1080
	itemNecklace      = 906

	itemAuronLetter     = 1138
	itemGuildMark       = 1139
	itemRustedSword1    = 1140
	itemRustedSword3    = 1142
	itemPoisonSpiderLeg = 1144
	itemMedallion       = 1145

	npcTrackerSkeleton = 20035
	npcVenomousSpider  = 20038

	classWarrior = 1
)

// questPages are the fixture pages: the NPCs' first pages with the quest
// link, and the quests' pages with every event link.
func questPages() map[string]string {
	pages := map[string]string{}
	first := `<html><body>%npcname% <a action="bypass -h npc_%objectId%_Script">Quest</a></body></html>`
	for _, page := range []string{"default/30048.htm", "teleporter/30006.htm", "trainer/30033.htm", "trainer/30010.htm", "merchant/30253.htm", "default/30598.htm"} {
		pages[page] = first
	}
	add := func(name string, events, names []string) {
		var links strings.Builder
		for _, ev := range events {
			links.WriteString(`<a action="bypass -h Script ` + name + ` ` + ev + `">` + ev + `</a>`)
		}
		for _, n := range names {
			pages["scripts/quests/"+name+"/"+n] = "<html><body>PAGE " + n + links.String() + "</body></html>"
		}
	}
	add(q001, []string{"30048-03.htm", "30048-04.htm", "30048-05.htm", "30048-06.htm"},
		[]string{"30006-01.htm", "30006-02.htm", "30006-03.htm", "30033-01.htm", "30033-02.htm",
			"30048-01.htm", "30048-02.htm", "30048-03.htm", "30048-04.htm", "30048-05.htm",
			"30048-06.htm", "30048-07.htm", "30048-08.htm", "30048-09.htm", "30048-10.htm"})
	add(q401, []string{"30010-05.htm", "30010-06.htm", "30010-10.htm", "30010-11.htm", "30253-02.htm"},
		[]string{"30010-01.htm", "30010-02.htm", "30010-02b.htm", "30010-03.htm", "30010-04.htm",
			"30010-05.htm", "30010-06.htm", "30010-07.htm", "30010-08.htm", "30010-09.htm",
			"30010-10.htm", "30010-11.htm", "30010-12.htm", "30010-13.htm", "30253-01.htm",
			"30253-02.htm", "30253-03.htm", "30253-03b.htm", "30253-04.htm", "30253-05.htm"})
	return pages
}

// shippedQuests returns the shipped quest scripts as test script files,
// plus extra ones.
func shippedQuests(t *testing.T, extra map[string]string) map[string]string {
	t.Helper()
	scripts := map[string]string{}
	for _, name := range []string{q001, q401} {
		rel := "quests/" + name + "/" + name + ".lua"
		src, err := os.ReadFile(filepath.Join("..", "..", "scripts", filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		scripts[rel] = string(src)
	}
	for k, v := range extra {
		scripts[k] = v
	}
	return scripts
}

// newQuestEnv starts a server with the shipped quests.
func newQuestEnv(t *testing.T, opts dialogEnvOptions) *testEnv {
	t.Helper()
	opts.pages = questPages()
	opts.scripts = shippedQuests(t, opts.scripts)
	return newDialogEnv(t, opts)
}

// questTalk talks to n and opens its quest link; it returns the packets of
// the quest link.
func (e *testEnv) questTalk(sess *fakeSession, p *Player, n *Npc) []testclient.Packet {
	e.t.Helper()
	e.talk(sess, p, n)
	return e.bypass(sess, "npc_"+strconv.Itoa(int(n.ID()))+"_Script")
}

// wantQuestPage fails unless ps hold the fixture page name.
func wantQuestPage(t *testing.T, what string, ps []testclient.Packet, name string) {
	t.Helper()
	pg := onePage(t, ps)
	if !strings.Contains(pg.HTML, "PAGE "+name+"<") {
		t.Fatalf("%s: page %q, want %s", what, pg.HTML, name)
	}
}

// questState reads p's state of a quest on the loop: its state, step and
// whether it exists.
func (e *testEnv) questState(p *Player, name string) (quest.StateID, int32, bool) {
	var (
		state quest.StateID
		cond  int32
		ok    bool
	)
	e.onLoop(func() {
		if st := e.srv.QuestState(p, name); st != nil {
			state, cond, ok = st.State(), st.Cond(), true
		}
	})
	return state, cond, ok
}

// itemCount reads how many items of a template p carries.
func (e *testEnv) itemCount(p *Player, itemID int32) int64 {
	var n int64
	e.onLoop(func() { n = p.inv.Count(itemID, -1, true) })
	return n
}

// storedChar waits for p's pending saves and returns the stored character.
func (e *testEnv) storedChar(p *Player) storage.Character {
	e.t.Helper()
	if err := e.srv.saver.wait(e.ctx, p.ObjectID()); err != nil {
		e.t.Fatal(err)
	}
	c, err := e.store.Get(e.ctx, p.ObjectID())
	if err != nil {
		e.t.Fatal(err)
	}
	return c
}

// questNames returns the names of the quest packets of ps, in order.
func questNames(ps []testclient.Packet) []string {
	var out []string
	for _, n := range testclient.Names(ps) {
		switch n {
		case "QuestList", "ExShowQuestMark", "PlaySound":
			out = append(out, n)
		}
	}
	return out
}

func lastQuestList(t *testing.T, ps []testclient.Packet) []testclient.Quest {
	t.Helper()
	lists := only[testclient.QuestList](ps)
	if len(lists) == 0 {
		t.Fatalf("no QuestList in %q", testclient.Names(ps))
	}
	return lists[len(lists)-1].Quests
}

func sounds(ps []testclient.Packet) []string {
	var out []string
	for _, s := range only[testclient.PlaySound](ps) {
		out = append(out, s.Sound)
	}
	return out
}

// TestQuestLettersOfLove walks Q001 with a relog in the middle.
func TestQuestLettersOfLove(t *testing.T) {
	e := newQuestEnv(t, dialogEnvOptions{})
	sess, p := e.enterChar(testChar{account: "lover", name: "Lover", pos: tiStart, level: 2})
	darin := e.spawnNpc(npcDarin, eastOf(tiStart, -50))
	roxxy := e.spawnNpc(npcRoxxy, eastOf(tiStart, -60))
	baulro := e.spawnNpc(npcBaulro, eastOf(tiStart, -70))

	// Darin offers the quest: the only quest of the window answers at once.
	wantQuestPage(t, "offer", e.questTalk(sess, p, darin), "30048-02.htm")
	if _, _, ok := e.questState(p, q001); !ok {
		t.Fatal("talking did not create the state")
	}

	// Accepting: the cond's QuestList and quest mark, the state's QuestList
	// and the accept sound, in Mobius's order; the letter.
	ps := e.bypass(sess, "Script "+q001+" 30048-06.htm")
	wantQuestPage(t, "accept", ps, "30048-06.htm")
	if got, want := questNames(ps), []string{"QuestList", "ExShowQuestMark", "QuestList", "PlaySound"}; !slices.Equal(got, want) {
		t.Errorf("accept packets %q, want %q", got, want)
	}
	if m := only[testclient.ExShowQuestMark](ps); len(m) != 1 || m[0].QuestID != 1 {
		t.Errorf("quest mark %+v", m)
	}
	if got := lastQuestList(t, ps); !slices.Equal(got, []testclient.Quest{{ID: 1, State: 1}}) {
		t.Errorf("quest list %+v", got)
	}
	if got := sounds(ps); !slices.Equal(got, []string{testclient.SoundQuestAccept}) {
		t.Errorf("sounds %q", got)
	}
	if e.itemCount(p, itemDarinLetter) != 1 {
		t.Fatal("no letter")
	}

	// Roxxy takes the letter and gives her kerchief.
	ps = e.questTalk(sess, p, roxxy)
	wantQuestPage(t, "Roxxy", ps, "30006-01.htm")
	if got := sounds(ps); !slices.Equal(got, []string{testclient.SoundQuestMiddle}) {
		t.Errorf("Roxxy sounds %q", got)
	}
	if e.itemCount(p, itemDarinLetter) != 0 || e.itemCount(p, itemRoxxyKerchief) != 1 {
		t.Fatal("Roxxy did not swap the letter for the kerchief")
	}

	// Relog: the step and the quest items stay.
	e.srv.Logout(sess)
	sess = newSession("lover")
	p = e.enter(sess, 0)
	ps = sess.take(t)
	if got := lastQuestList(t, ps); !slices.Equal(got, []testclient.Quest{{ID: 1, State: 2}}) {
		t.Errorf("quest list after the relog %+v", got)
	}
	if e.itemCount(p, itemRoxxyKerchief) != 1 {
		t.Fatal("the kerchief is gone after the relog")
	}
	if st, cond, _ := e.questState(p, q001); st != quest.Started || cond != 2 {
		t.Fatalf("after the relog: state %v cond %d", st, cond)
	}

	wantQuestPage(t, "Darin receipt", e.questTalk(sess, p, darin), "30048-08.htm")
	wantQuestPage(t, "Baulro", e.questTalk(sess, p, baulro), "30033-01.htm")
	ps = e.questTalk(sess, p, darin)
	wantQuestPage(t, "reward", ps, "30048-10.htm")
	if got := lastQuestList(t, ps); len(got) != 0 {
		t.Errorf("quest list after the end %+v", got)
	}
	if got := sounds(ps); !slices.Contains(got, testclient.SoundQuestFinish) {
		t.Errorf("no finish sound in %q", got)
	}
	if e.itemCount(p, itemNecklace) != 1 || e.itemCount(p, itemBaulroPotion) != 0 || e.itemCount(p, itemDarinReceipt) != 0 {
		t.Fatal("wrong items after the reward")
	}
	if st, _, ok := e.questState(p, q001); !ok || st != quest.Completed {
		t.Fatalf("state %v %v, want completed", st, ok)
	}

	// The completed quest answers with Mobius's text and is saved as such.
	pg := onePage(t, e.questTalk(sess, p, darin))
	if !strings.Contains(pg.HTML, "This quest has already been completed.") {
		t.Errorf("completed page %q", pg.HTML)
	}
	e.srv.Logout(sess)
	c := e.storedChar(p)
	if len(c.Quests) != 1 || c.Quests[0].Name != q001 || !maps.Equal(c.Quests[0].Vars, map[string]string{quest.VarState: "Completed"}) {
		t.Errorf("stored quests %+v", c.Quests)
	}
}

// TestQuestAbort aborts a started quest from the quest log: the state and
// the quest items are gone; a completed quest cannot be aborted.
func TestQuestAbort(t *testing.T) {
	e := newQuestEnv(t, dialogEnvOptions{})
	sess, p := e.enterChar(testChar{account: "quitter", name: "Quitter", pos: tiStart, level: 2})
	darin := e.spawnNpc(npcDarin, eastOf(tiStart, -50))
	e.questTalk(sess, p, darin)
	e.bypass(sess, "Script "+q001+" 30048-06.htm")

	e.srv.Handle(sess, clientpackets.OpRequestQuestAbort, int32Body(1))
	e.sync()
	ps := sess.take(t)
	if got := lastQuestList(t, ps); len(got) != 0 {
		t.Errorf("quest list after the abort %+v", got)
	}
	if _, _, ok := e.questState(p, q001); ok {
		t.Error("the state is still there")
	}
	if e.itemCount(p, itemDarinLetter) != 0 {
		t.Error("the letter is still there")
	}

	// The deletion reaches the store.
	before := e.store.saveCount(p.ObjectID())
	e.advance(defaultQuestSaveDelay)
	c := e.storedChar(p)
	if e.store.saveCount(p.ObjectID()) == before {
		t.Error("no save after the quest save delay")
	}
	if len(c.Quests) != 0 {
		t.Errorf("stored quests after the abort %+v", c.Quests)
	}

	// Completed quests stay.
	e.onLoop(func() {
		st := e.srv.questsOf(p).book.New(q001)
		st.SetState(quest.Completed)
	})
	e.srv.Handle(sess, clientpackets.OpRequestQuestAbort, int32Body(1))
	e.sync()
	if st, _, ok := e.questState(p, q001); !ok || st != quest.Completed {
		t.Error("a completed quest was aborted")
	}
}

// classChangeScript is a stand-in for the village masters: it changes a
// level 20 fighter with the Medallion of Warrior into a Warrior.
const classChangeScript = `
local m = script.new("ClassChangeTest")
m:on_event(function(event, npc, player)
  if event == "warrior" then
    if player:class_id() ~= ClassId.FIGHTER or player:level() < 20 or not m:has_quest_items(player, 1145) then
      return "<html><body>No</body></html>"
    end
    m:take_items(player, 1145, 1)
    player:set_class_id(ClassId.WARRIOR)
    player:broadcast_user_info()
    return "<html><body>Warrior</body></html>"
  end
end)
`

// TestQuestPathOfTheWarrior walks Q401 with the kills on the fake clock,
// then changes the class.
func TestQuestPathOfTheWarrior(t *testing.T) {
	src := rnd.NewScripted(t)
	e := newQuestEnv(t, dialogEnvOptions{rand: src, scripts: map[string]string{"village_master/test/test.lua": classChangeScript}})
	sess, p := e.enterChar(testChar{account: "warrior", name: "Warrior", pos: tiStart, level: 19})
	auron := e.spawnNpc(npcAuron, eastOf(tiStart, -50))
	simplon := e.spawnNpc(npcSimplon, eastOf(tiStart, -60))

	wantQuestPage(t, "offer", e.questTalk(sess, p, auron), "30010-01.htm")
	wantQuestPage(t, "check", e.bypass(sess, "Script "+q401+" 30010-05.htm"), "30010-05.htm")
	wantQuestPage(t, "accept", e.bypass(sess, "Script "+q401+" 30010-06.htm"), "30010-06.htm")
	if e.itemCount(p, itemAuronLetter) != 1 {
		t.Fatal("no letter")
	}
	wantQuestPage(t, "Simplon", e.questTalk(sess, p, simplon), "30253-01.htm")
	wantQuestPage(t, "guild mark", e.bypass(sess, "Script "+q401+" 30253-02.htm"), "30253-02.htm")
	if _, cond, _ := e.questState(p, q401); cond != 2 || e.itemCount(p, itemGuildMark) != 1 {
		t.Fatalf("cond %d after Simplon", cond)
	}

	kill := func(npcID int) []testclient.Packet {
		t.Helper()
		n := e.spawnNpc(npcID, eastOf(tiStart, 40))
		sess.take(t)
		e.onLoop(func() { e.srv.Kill(n, p) })
		e.advance(defaultQuestKillDelay - time.Millisecond)
		if ps := sess.take(t); len(sounds(ps)) != 0 {
			t.Fatal("the kill handler ran before the kill delay")
		}
		e.advance(time.Millisecond)
		return sess.take(t)
	}

	// A miss gives nothing (rnd(10) = 4 is not below 4).
	src.Add(rnd.Get(10, 4))
	if got := sounds(kill(npcTrackerSkeleton)); len(got) != 0 {
		t.Fatalf("a miss: %q", got)
	}
	// Ten Rusted Bronze Swords from Tracker Skeletons, 40% each: forced.
	for i := 1; i <= 10; i++ {
		src.Add(rnd.Get(10, 0))
		ps := kill(npcTrackerSkeleton)
		want := testclient.SoundQuestItemGet
		if i == 10 {
			want = testclient.SoundQuestMiddle
		}
		if got := sounds(ps); !slices.Equal(got, []string{want}) {
			t.Fatalf("kill %d: sounds %q", i, got)
		}
	}
	// A full count moves to cond 3; no more draws happen then.
	kill(npcTrackerSkeleton)
	if _, cond, _ := e.questState(p, q401); cond != 3 || e.itemCount(p, itemRustedSword1) != 10 {
		t.Fatalf("cond %d swords %d", cond, e.itemCount(p, itemRustedSword1))
	}

	wantQuestPage(t, "swords", e.questTalk(sess, p, simplon), "30253-04.htm")
	wantQuestPage(t, "Auron again", e.questTalk(sess, p, auron), "30010-09.htm")
	wantQuestPage(t, "sword 3", e.bypass(sess, "Script "+q401+" 30010-11.htm"), "30010-11.htm")

	// Spider legs only with the Rusted Bronze Sword in hand.
	if got := sounds(kill(npcVenomousSpider)); len(got) != 0 {
		t.Fatalf("a leg without the sword: %q", got)
	}
	e.onLoop(func() { e.srv.useEquippableItem(p, p.inv.ByItemID(itemRustedSword3)) })
	for i := 1; i <= 20; i++ {
		kill(npcVenomousSpider)
	}
	if _, cond, _ := e.questState(p, q401); cond != 6 || e.itemCount(p, itemPoisonSpiderLeg) != 20 {
		t.Fatalf("cond %d legs %d", cond, e.itemCount(p, itemPoisonSpiderLeg))
	}

	ps := e.questTalk(sess, p, auron)
	wantQuestPage(t, "medallion", ps, "30010-13.htm")
	if acts := only[testclient.SocialAction](ps); len(acts) != 1 || acts[0].ActionID != 3 {
		t.Errorf("social actions %+v", acts)
	}
	if e.itemCount(p, itemMedallion) != 1 || e.itemCount(p, itemRustedSword3) != 0 || e.itemCount(p, itemPoisonSpiderLeg) != 0 {
		t.Fatal("wrong items after the reward")
	}
	if _, _, ok := e.questState(p, q401); ok {
		t.Fatal("Q401 is repeatable: its state must be gone")
	}

	// Level 20, then the class change.
	e.onLoop(func() { e.srv.AddLevel(p, 1) })
	e.onLoop(func() {
		p.HTML.Set(HTMLScopeNpc, auron.ID(), `<a action="bypass -h Script ClassChangeTest warrior">x</a>`)
	})
	ps = e.bypass(sess, "Script ClassChangeTest warrior")
	if pg := onePage(t, ps); pg.HTML != "<html><body>Warrior</body></html>" {
		t.Fatalf("class change page %q", pg.HTML)
	}
	if uses := only[testclient.MagicSkillUse](ps); len(uses) != 1 || uses[0].SkillID != classChangeSkill {
		t.Errorf("class change effect %+v", uses)
	}
	if msgs := only[testclient.SystemMessage](ps); !slices.ContainsFunc(msgs, func(m testclient.SystemMessage) bool { return m.ID == int32(sysMsgClassTransfer) }) {
		t.Errorf("no class transfer message in %+v", msgs)
	}
	// The skill list task runs 100 ms later; Mobius sendSkillList sends it
	// 300 ms after that.
	e.advance(classChangeSkillListDelay + enterDelay)
	ps = sess.take(t)
	if uis := only[testclient.UserInfo](ps); len(uis) == 0 || uis[len(uis)-1].ClassID != classWarrior {
		t.Errorf("UserInfo after the class change %+v", uis)
	}
	if !slices.Contains(testclient.Names(ps), "SkillList") {
		t.Errorf("no SkillList 100 ms later: %q", testclient.Names(ps))
	}
	var class int
	e.onLoop(func() { class = p.ClassID() })
	if class != classWarrior || e.itemCount(p, itemMedallion) != 0 {
		t.Errorf("class %d, medallions %d", class, e.itemCount(p, itemMedallion))
	}
}

// tutorialScript is a stand-in for Q255's first steps: enter world shows
// question mark 1, which opens a tutorial page.
const tutorialScript = `
local q = script.new("Q00255_Tutorial")
q:on_event(function(event, npc, player)
  local st = q:get_quest_state(player, false)
  if st == nil then
    return nil
  end
  if event == "UC" then
    st:show_question_mark(1)
    st:play_tutorial_voice("tutorial_voice_006")
    q:play_sound(player, "ItemSound.quest_tutorial")
  elseif event == "QM1" then
    st:show_tutorial_html("tutorial_04.htm")
  elseif event == "CE8" then
    st:on_tutorial_client_event(0)
  elseif event == "close" then
    st:close_tutorial_html()
  end
  return ""
end)
`

// TestTutorialQuestionMark: a new character's tutorial shows its first
// question mark on entering, and a click opens the tutorial page.
func TestTutorialQuestionMark(t *testing.T) {
	e := newQuestEnv(t, dialogEnvOptions{scripts: map[string]string{
		"quests/Q00255_Tutorial/Q00255_Tutorial.lua": tutorialScript,
		"quests/Q00255_Tutorial/tutorial_04.htm":     "<html><body>Tutorial 4</body></html>",
	}})
	var start []storage.Quest
	e.onLoop(func() { start = e.srv.NewCharacterQuests() })
	ch := e.createCharacter("newbie", "Newbie", classHumanFighter, storage.AccessLevelPlayer)
	ch.Quests = start
	if err := e.store.Update(e.ctx, &ch); err != nil {
		t.Fatal(err)
	}
	sess := newSession("newbie")
	p := e.enter(sess, 0)
	ps := sess.take(t)
	names := testclient.Names(ps)
	mark, list := slices.Index(names, "TutorialShowQuestionMark"), slices.Index(names, "QuestList")
	if mark < 0 || list < mark {
		t.Fatalf("enter world packets %q: want the question mark before QuestList", names)
	}
	if m := only[testclient.TutorialShowQuestionMark](ps); m[0].Mark != 1 {
		t.Errorf("question mark %+v", m)
	}
	voices := only[testclient.PlaySound](ps)
	if len(voices) != 2 || voices[0].Type != 2 || voices[0].Sound != "tutorial_voice_006" || voices[1].Sound != testclient.SoundTutorial {
		t.Errorf("sounds %+v", voices)
	}
	// Q255 is a script, not a listed quest.
	if got := lastQuestList(t, ps); len(got) != 0 {
		t.Errorf("quest list %+v", got)
	}

	e.srv.Handle(sess, clientpackets.OpRequestTutorialQuestionMark, int32Body(1))
	e.sync()
	ps = sess.take(t)
	if h := only[testclient.TutorialShowHtml](ps); len(h) != 1 || h[0].HTML != "<html><body>Tutorial 4</body></html>" {
		t.Fatalf("tutorial page %+v", h)
	}
	e.srv.Handle(sess, clientpackets.OpRequestTutorialClientEvent, int32Body(8))
	e.sync()
	if ev := only[testclient.TutorialEnableClientEvent](sess.take(t)); len(ev) != 1 || ev[0].Event != 0 {
		t.Errorf("client event %+v", ev)
	}
	e.srv.Handle(sess, clientpackets.OpRequestTutorialLinkHtml, stringBody("close"))
	e.sync()
	if got := testclient.Names(sess.take(t)); !slices.Equal(got, []string{"TutorialCloseHtml", "TutorialCloseHtml"}) {
		t.Errorf("close: %q", got)
	}

	// The tutorial state is saved with the character.
	e.srv.Logout(sess)
	c := e.storedChar(p)
	if len(c.Quests) != 1 || c.Quests[0].Name != tutorialQuest {
		t.Errorf("stored quests %+v", c.Quests)
	}
}

// questWindowScripts are two quests at the Newbie Guide and a plain script
// that also talks there.
var questWindowScripts = map[string]string{
	"quests/QB/QB.lua": `
local q = quest.new(7, "QB", "Quest Seven")
q:add_start_npc(30598)
q:add_talk_id(30598)
q:on_talk(function(npc, player)
  local st = q:get_quest_state(player, true)
  if st:is_created() then
    st:start_quest()
  end
  return "<html><body>Seven</body></html>"
end)
`,
	"quests/QA/QA.lua": `
local q = quest.new(3, "QA", "Quest Three")
q:add_start_npc(30598)
q:add_talk_id(30598)
q:on_talk(function(npc, player)
  return "<html><body>Three</body></html>"
end)
`,
	"quests/QR/QR.lua": `
local q = quest.new(9, "QR", "Needs Clan Halls")
q:requires("clan_halls")
q:add_start_npc(30598)
q:add_talk_id(30598)
q:on_talk(function(npc, player) return "<html><body>Clans</body></html>" end)
`,
	"npcs/other/other.lua": `
local o = script.new("OtherNpc")
o:on_talk(30598, function(npc, player) return "Other talks" end)
`,
}

// TestQuestWindow: the choose window lists the quests by ID with their
// state, leaves out plain scripts and quests whose features are missing,
// and refuses a full inventory.
func TestQuestWindow(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: questPages(), scripts: questWindowScripts})
	sess, p := e.enterChar(testChar{account: "chooser", name: "Chooser", pos: tiStart})
	guide := e.spawnNpc(npcNewbieGuide, eastOf(tiStart, -50))
	gid := strconv.Itoa(int(guide.ID()))
	choice := func(seven string) string {
		return `<html><body>` +
			`<a action="bypass npc_` + gid + `_Script QA">[Quest Three]</a><br>` +
			`<a action="bypass npc_` + gid + `_Script QB">[Quest Seven` + seven + `]</a><br>` +
			`</body></html>`
	}
	wantPage(t, "choice", onePage(t, e.questTalk(sess, p, guide)), guide.ID(), choice(""))
	ps := e.bypass(sess, "npc_"+gid+"_Script QB")
	wantPage(t, "talk", onePage(t, ps), guide.ID(), "<html><body>Seven</body></html>")
	wantPage(t, "in progress", onePage(t, e.questTalk(sess, p, guide)), guide.ID(), choice(" (In Progress)"))

	// Over 80% weight: Mobius's message, no talk.
	e.onLoop(func() { p.items.weightPenalty = 3 })
	ps = e.bypass(sess, "npc_"+gid+"_Script QA")
	if msgs := only[testclient.SystemMessage](ps); len(msgs) != 1 || msgs[0].ID != int32(sysMsgQuestInventoryFull) {
		t.Errorf("full inventory: %q", testclient.Names(ps))
	}
	noPage(t, "full inventory", ps)
}

// TestMergeItemChangesKeepsQuests: coalesced saves keep the quest changes
// of both, per quest the newer.
func TestMergeItemChangesKeepsQuests(t *testing.T) {
	doc := func(name, cond string) storage.Quest {
		return storage.Quest{Name: name, Vars: map[string]string{"cond": cond}}
	}
	a := storage.ItemChanges{Destroyed: []int32{1}, Quests: storage.QuestChanges{Upserts: []storage.Quest{doc("A", "1"), doc("B", "1")}}}
	b := storage.ItemChanges{Quests: storage.QuestChanges{Upserts: []storage.Quest{doc("A", "2")}, Deleted: []string{"B"}}}
	got := mergeItemChanges(a, b)
	if !slices.Equal(got.Destroyed, []int32{1}) || len(got.Quests.Upserts) != 1 || got.Quests.Upserts[0].Vars["cond"] != "2" ||
		!slices.Equal(got.Quests.Deleted, []string{"B"}) {
		t.Errorf("merged %+v", got)
	}
	c := storage.ItemChanges{Upserts: []storage.Item{{ObjectID: 2}}, Quests: storage.QuestChanges{Upserts: []storage.Quest{doc("C", "1")}}}
	got = mergeItemChanges(got, c)
	if len(got.Upserts) != 1 || len(got.Quests.Upserts) != 2 {
		t.Errorf("merged with items %+v", got)
	}
}

// TestQuestSaveFailedMarksDirty: quest changes whose write failed for good
// are saved again later.
func TestQuestSaveFailedMarksDirty(t *testing.T) {
	e := newQuestEnv(t, dialogEnvOptions{})
	_, p := e.enterChar(testChar{account: "unlucky", name: "Unlucky", pos: tiStart})
	var changes storage.QuestChanges
	e.onLoop(func() {
		e.srv.questsOf(p).book.New(q001).SetState(quest.Started)
		changes = p.takeSaveChanges().Quests
	})
	if len(changes.Upserts) != 1 {
		t.Fatalf("changes %+v", changes)
	}
	e.srv.questSaveDone(p, changes, nil)(errInjectedSave)
	e.sync()
	var dirty bool
	e.onLoop(func() { dirty = p.quests.book.Dirty() })
	if !dirty {
		t.Error("the quest is not dirty after a failed save")
	}
}
