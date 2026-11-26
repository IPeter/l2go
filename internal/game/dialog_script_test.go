package game

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
)

// dialogScripts are the test scripts: TestNpc owns the Newbie Guide's first
// talk and answers every kind of reply; OtherNpc only talks at it.
var dialogScripts = map[string]string{
	"npcs/test/test.lua": `
local t = script.new("TestNpc")
t:on_first_talk(30598, function(npc, player)
  if player:level() >= 50 then
    return nil
  end
  return "first.htm", { name = player:name() }
end)
t:on_talk(30598, function(npc, player)
  return "<html><body>Talk with %playername%</body></html>"
end)
t:on_event(function(event, npc, player)
  if event == "inline" then
    return "<html><body>Inline for %playername% at %objectId%</body></html>"
  elseif event == "text" then
    return "Just text"
  elseif event == "say" then
    npc:say("Hello " .. player:name())
    return ""
  elseif event == "fail" then
    error("boom")
  elseif event == "shared" then
    return "data/html/default/SupportMagic.htm"
  elseif event == "page" then
    return "first.htm", { name = "again" }
  end
  return nil
end)
`,
	"npcs/test/first.htm": `<html><body>Hello %name% (%playername%) at %objectId%<br>` +
		`<a action="bypass -h Script TestNpc inline">Inline</a><br>` +
		`<a action="bypass -h Script TestNpc text">Text</a><br>` +
		`<a action="bypass -h Script TestNpc say">Say</a><br>` +
		`<a action="bypass -h Script TestNpc fail">Fail</a><br>` +
		`<a action="bypass -h Script TestNpc shared">Shared</a><br>` +
		`<a action="bypass -h Script TestNpc page">Page</a><br>` +
		`<a action="bypass -h npc_%objectId%_Chat 0">Again</a><br>` +
		`<a action="bypass Script">Quest</a></body></html>`,
	"npcs/other/other.lua": `
local o = script.new("OtherNpc")
o:on_talk(30598, function(npc, player)
  return "Other talks"
end)
`,
}

func TestScriptFirstTalkAndEvents(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: dialogPages, scripts: dialogScripts})
	sess, p := e.enterChar(testChar{account: "hero", name: "Hero", pos: tiStart})
	guide := e.spawnNpc(npcNewbieGuide, eastOf(tiStart, -50))
	gid := strconv.Itoa(int(guide.ID()))
	firstPage := `<html><body>Hello Hero (Hero) at ` + gid + `<br>` +
		`<a action="bypass -h Script TestNpc inline">Inline</a><br>` +
		`<a action="bypass -h Script TestNpc text">Text</a><br>` +
		`<a action="bypass -h Script TestNpc say">Say</a><br>` +
		`<a action="bypass -h Script TestNpc fail">Fail</a><br>` +
		`<a action="bypass -h Script TestNpc shared">Shared</a><br>` +
		`<a action="bypass -h Script TestNpc page">Page</a><br>` +
		`<a action="bypass -h npc_` + gid + `_Chat 0">Again</a><br>` +
		`<a action="bypass Script">Quest</a></body></html>`

	// The first talk answers with the script's page: the script's
	// placeholders, %objectId% and %playername% filled, then ActionFailed.
	ps := e.talk(sess, p, guide)
	checkNames(t, "first talk", ps, "NpcHtmlMessage", "ActionFailed")
	wantPage(t, "first talk", onePage(t, ps), guide.ID(), firstPage)

	// "Chat 0" runs the first talk again.
	ps = e.bypass(sess, "npc_"+gid+"_Chat 0")
	wantPage(t, "Chat 0", onePage(t, ps), guide.ID(), firstPage)

	// Inline HTML gets %playername% only, as in Mobius.
	ps = e.bypass(sess, "Script TestNpc inline")
	checkNames(t, "inline", ps, "NpcHtmlMessage", "ActionFailed")
	wantPage(t, "inline", onePage(t, ps), guide.ID(), "<html><body>Inline for Hero at %objectId%</body></html>")

	// The inline page had no links: back to the first page for the others.
	e.talk(sess, p, guide)
	ps = e.bypass(sess, "Script TestNpc text")
	checkNames(t, "text", ps, "SystemMessage")
	if msgs := messages(ps); !slices.Equal(msgs, []string{"Just text"}) {
		t.Errorf("text reply %q", msgs)
	}

	ps = e.bypass(sess, "Script TestNpc say")
	says := only[testclient.NpcSay](ps)
	if len(says) != 1 || says[0].Text != "Hello Hero" || says[0].ObjectID != int32(guide.ID()) {
		t.Errorf("npc:say gave %+v", says)
	}
	noPage(t, `a "" reply`, ps)

	// A failing handler shows players nothing.
	if ps := e.bypass(sess, "Script TestNpc fail"); len(ps) != 0 {
		t.Errorf("a failing handler answered %q", testclient.Names(ps))
	}

	// A "data/html/" page comes from the HTML folder.
	ps = e.bypass(sess, "Script TestNpc shared")
	wantPage(t, "shared page", onePage(t, ps), guide.ID(), `<html><body>Support at `+gid+`<br><a action="bypass -h npc_`+gid+`_Chat 0">Back</a></body></html>`)

	// Script events need the NPC within interaction range (2D).
	e.talk(sess, p, guide)
	e.onLoop(func() {
		if err := e.srv.MoveObject(p, eastOf(tiStart, -50+260)); err != nil {
			t.Errorf("MoveObject: %v", err)
		}
	})
	e.onLoop(func() { p.HTML.Set(HTMLScopeNpc, 0, `<a action="bypass -h Script TestNpc page">x</a>`) })
	noPage(t, "script event out of range", e.bypass(sess, "Script TestNpc page"))
}

func TestScriptFirstTalkDefaults(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: dialogPages, scripts: dialogScripts})
	sess, p := e.enterChar(testChar{account: "veteran", name: "Veteran", pos: tiStart, level: 50})
	guide := e.spawnNpc(npcNewbieGuide, eastOf(tiStart, -50))
	// nil from the first talk: the NPC's own page.
	pg := onePage(t, e.talk(sess, p, guide))
	if !strings.HasPrefix(pg.HTML, "<html><body>Newbie Guide (") {
		t.Errorf("default page %q", pg.HTML)
	}
}

func TestScriptErrorsShowToGMs(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: dialogPages, scripts: dialogScripts})
	sess, gm := e.enterChar(testChar{account: "gm", name: "Gm", access: storage.AccessLevelAdmin, pos: tiStart})
	guide := e.spawnNpc(npcNewbieGuide, eastOf(tiStart, -50))
	e.talk(sess, gm, guide)
	pg := onePage(t, e.bypass(sess, "Script TestNpc fail"))
	if pg.ObjectID != 0 || !strings.HasPrefix(pg.HTML, "<html><body><title>Script error</title>") || !strings.Contains(pg.HTML, "boom") {
		t.Errorf("GM error page %+v", pg)
	}
}

// The ported Mobius scripts: their first talks and every link of their
// pages have answers.
func TestPortedNpcScripts(t *testing.T) {
	const scriptsDir = "../../scripts"
	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 40}

	t.Run("SymbolMaker", func(t *testing.T) {
		h := scripttest.Load(t, scriptsDir, "npcs/symbol_maker/symbol_maker.lua")
		for _, id := range []int{31046, 31047, 31048, 31049, 31050, 31051, 31052, 31053, 31264, 31308, 31953} {
			npc := &scripttest.Npc{ObjectID: 0x60000001, ID: id, Name: "Symbol Maker"}
			if reply, ok := h.FirstTalk(hero, npc); !ok || reply.Kind != scripting.ReplyPage || reply.Content != "symbol_maker.htm" {
				t.Errorf("first talk at %d: %+v, %v", id, reply, ok)
			}
		}
		npc := &scripttest.Npc{ObjectID: 0x60000001, ID: 31046, Name: "Marsden"}
		for _, event := range []string{"symbol_maker.htm", "symbol_maker-1.htm", "symbol_maker-2.htm", "symbol_maker-3.htm"} {
			if reply, ok := h.Event("SymbolMaker", event, hero, npc); !ok || reply.Content != event {
				t.Errorf("event %s: %+v, %v", event, reply, ok)
			}
		}
		// Draw and Remove open the symbol windows and show no page (henna_test.go
		// runs them in a game server).
		for _, event := range []string{"Draw", "Remove"} {
			if reply, ok := h.Event("SymbolMaker", event, hero, npc); ok {
				t.Errorf("event %s: %+v; want no page", event, reply)
			}
		}
		if !slices.Equal(hero.HennaWindows, []string{"equip", "remove"}) {
			t.Errorf("symbol windows %q, want equip, remove", hero.HennaWindows)
		}
		if _, ok := h.Event("SymbolMaker", "nonsense", hero, npc); ok {
			t.Error("an unknown event answered")
		}
	})

	t.Run("Asamah", func(t *testing.T) {
		h := scripttest.Load(t, scriptsDir, "npcs/asamah/asamah.lua")
		npc := &scripttest.Npc{ObjectID: 0x60000002, ID: 32115, Name: "Asamah"}
		if reply, ok := h.FirstTalk(hero, npc); !ok || reply.Content != "32115-02.htm" {
			t.Errorf("first talk: %+v, %v", reply, ok)
		}
		for _, event := range []string{"32115-03.htm", "32115-04.htm"} {
			if reply, ok := h.Event("Asamah", event, hero, npc); !ok || reply.Content != event {
				t.Errorf("event %s: %+v, %v", event, reply, ok)
			}
		}
		if _, ok := h.Event("Asamah", "32115-01.htm", hero, npc); ok {
			t.Error("the event 32115-01.htm answered; Mobius allows only the explanations")
		}
	})

	t.Run("Kier", func(t *testing.T) {
		h := scripttest.Load(t, scriptsDir, "npcs/kier/kier.lua")
		npc := &scripttest.Npc{ObjectID: 0x60000003, ID: 32022, Name: "Kierre"}
		if reply, ok := h.FirstTalk(hero, npc); !ok || reply.Content != "32022-02.html" {
			t.Errorf("first talk: %+v, %v", reply, ok)
		}
	})
}
