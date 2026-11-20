package scripting

import (
	"context"
	"errors"
	"maps"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

func firstTalk(t testing.TB, e *Engine, npcID int, p PlayerRef, n NpcRef) (Reply, bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return e.FirstTalk(ctx, npcID, p, n)
}

func scriptEvent(t testing.TB, e *Engine, script, event string, p PlayerRef, n NpcRef) (Reply, bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return e.Event(ctx, script, event, p, n)
}

func talk(t testing.TB, e *Engine, script string, npcID int, p PlayerRef, n NpcRef) (Reply, bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return e.Talk(ctx, script, npcID, p, n)
}

// What a handler returns becomes a Reply by Mobius's showResult rules.
func TestReplyKinds(t *testing.T) {
	dir := writeScripts(t, map[string]string{"kinds.lua": `
local answers = {
  page = function() return "intro.htm" end,
  page_html = function() return "sub/intro.html" end,
  inline = function() return "<html><body>Hi %name%</body></html>", { name = "Hero" } end,
  text = function() return "Go away, %name%.", { name = "Hero" } end,
  none = function() return "" end,
  default = function() return nil end,
  html_ending_htm = function() return "<html><body>see x.htm" end,
  number = function() return 5 end,
  table = function() return {} end,
  parent_page = function() return "../secret.htm" end,
  rooted_page = function() return "/etc/x.htm" end,
  dot_page = function() return "./x.htm" end,
  backslash_page = function() return "a\\b.htm" end,
}
local s = script.new("Kinds")
s:on_first_talk(30733, function(npc, player)
  return answers[player:name()]()
end)`})
	e := newEngine(t, nil, Options{Dir: dir})

	tests := []struct {
		name    string
		ok      bool
		kind    ReplyKind
		content string
		err     string // in Reply.Err; "" for none
	}{
		{"page", true, ReplyPage, "intro.htm", ""},
		{"page_html", true, ReplyPage, "sub/intro.html", ""},
		{"inline", true, ReplyHTML, "<html><body>Hi Hero</body></html>", ""},
		{"text", true, ReplyText, "Go away, Hero.", ""},
		{"none", true, ReplyNone, "", ""},
		{"default", false, ReplyNone, "", ""},
		// Mobius checks the file suffix before the <html prefix.
		{"html_ending_htm", true, ReplyPage, "<html><body>see x.htm", ""},
		{"number", false, ReplyNone, "", "returned a number"},
		{"table", false, ReplyNone, "", "returned a table"},
		{"parent_page", false, ReplyNone, "", "page name"},
		{"rooted_page", false, ReplyNone, "", "page name"},
		{"dot_page", false, ReplyNone, "", "page name"},
		{"backslash_page", false, ReplyNone, "", "page name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reply, ok := firstTalk(t, e, 30733, &fakePlayer{name: tt.name}, guardNpc())
			if ok != tt.ok || reply.Kind != tt.kind || reply.Content != tt.content {
				t.Errorf("reply = %+v, %v; want %v %q, %v", reply, ok, tt.kind, tt.content, tt.ok)
			}
			if reply.Script != "Kinds" || reply.Dir != dir {
				t.Errorf("script, dir = %q, %q; want Kinds, %q", reply.Script, reply.Dir, dir)
			}
			switch {
			case tt.err == "" && reply.Err != nil:
				t.Errorf("Err = %v", reply.Err)
			case tt.err != "":
				he, isErr := errors.AsType[*Error](reply.Err)
				if !isErr || !strings.Contains(he.Msg, tt.err) || !strings.HasSuffix(he.File, "kinds.lua") || he.Line == 0 {
					t.Errorf("Err = %v, want kinds.lua:<line> ... %s", reply.Err, tt.err)
				}
			}
		})
	}
}

// Handlers get (npc, player), in Mobius's order, with working methods.
func TestFirstTalkArgs(t *testing.T) {
	e := newEngine(t, map[string]string{"args.lua": `
local s = script.new("Args")
s:on_first_talk({ 30733, 31032 }, function(npc, player)
  return table.concat({ npc:id(), npc:object_id(), npc:name(), player:object_id(),
    player:name(), player:level(), player:class_id(), player:race(),
    tostring(npc), tostring(s) }, "|")
end)`}, Options{})
	p := &fakePlayer{id: 0x10000002, name: "Elfie", level: 12, classID: 25, race: 1}
	for _, id := range []int{30733, 31032} {
		n := &fakeNpc{id: 0x60000005, npcID: id, name: "Guard"}
		reply, ok := firstTalk(t, e, id, p, n)
		want := strings.Join([]string{
			strconv.Itoa(id), "1610612741", "Guard", "268435458",
			"Elfie", "12", "25", "1", "npc Guard (" + strconv.Itoa(id) + ", object 1610612741)", "script Args",
		}, "|")
		if !ok || reply.Kind != ReplyText || reply.Content != want {
			t.Errorf("NPC %d: reply = %+v, %v; want %q", id, reply, ok, want)
		}
	}
	if !e.HasFirstTalk(30733) || !e.HasFirstTalk(31032) || e.HasFirstTalk(30734) {
		t.Error("HasFirstTalk is wrong")
	}
	if reply, ok := firstTalk(t, e, 30734, p, guardNpc()); ok || reply.Err != nil || reply.Script != "" {
		t.Errorf("an NPC without a script: %+v, %v", reply, ok)
	}
}

// Only one script may own the first talk of an NPC; the load error names
// both places.
func TestFirstTalkTwice(t *testing.T) {
	le := loadErr(t, map[string]string{
		"npcs/a.lua": "local a = script.new(\"A\")\na:on_first_talk({ 30733, 30734 }, function() end)\n",
		"npcs/b.lua": "local b = script.new(\"B\")\n\nb:on_first_talk(30734, function() end)\n",
	}, Options{})
	if len(le.Errs) != 1 {
		t.Fatalf("errors = %v, want 1", le.Errs)
	}
	e, ok := errors.AsType[*Error](le.Errs[0])
	if !ok || !strings.HasSuffix(e.File, filepath.Join("npcs", "b.lua")) || e.Line != 3 {
		t.Fatalf("error = %v, want npcs/b.lua:3", le.Errs[0])
	}
	for _, want := range []string{"NPC 30734", "already has a first-talk handler", "script A", filepath.Join("npcs", "a.lua") + ":2"} {
		if !strings.Contains(e.Msg, want) {
			t.Errorf("message %q does not contain %q", e.Msg, want)
		}
	}
}

func TestScriptAPIErrors(t *testing.T) {
	tests := map[string]struct{ src, want string }{
		"bad name":     {`script.new("bad name")`, "invalid script name"},
		"empty name":   {`script.new("")`, "invalid script name"},
		"long name":    {`script.new("` + strings.Repeat("a", 65) + `")`, "invalid script name"},
		"duplicate":    {"script.new(\"Same\")\nscript.new(\"Same\")", `a script named "Same" already exists`},
		"ids type":     {`script.new("S"):on_first_talk("x", function() end)`, "NPC ID or list of NPC IDs expected"},
		"empty ids":    {`script.new("S"):on_talk({}, function() end)`, "empty"},
		"zero id":      {`script.new("S"):on_talk(0, function() end)`, "invalid NPC ID 0"},
		"fraction id":  {`script.new("S"):on_talk(1.5, function() end)`, "invalid NPC ID 1.5"},
		"huge id":      {`script.new("S"):on_talk(4294967296, function() end)`, "invalid NPC ID"},
		"listed twice": {`script.new("S"):on_first_talk({ 1, 1 }, function() end)`, "listed twice"},
		"no function":  {`script.new("S"):on_event()`, "function expected"},
		"event twice":  {"local s = script.new(\"S\")\ns:on_event(function() end)\ns:on_event(function() end)", "already has an event handler"},
		"talk twice":   {"local s = script.new(\"S\")\ns:on_talk(5, function() end)\ns:on_talk({ 6, 5 }, function() end)", "NPC 5 already has a talk handler"},
		"dot call":     {`local s = script.new("S"); s.on_event(function() end)`, "must be called on a script"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			le := loadErr(t, map[string]string{"s.lua": tt.src}, Options{})
			if !strings.Contains(le.Error(), tt.want) || !strings.Contains(le.Error(), "s.lua:") {
				t.Errorf("error = %v, want s.lua:<line> ... %s", le, tt.want)
			}
		})
	}
}

// Scripts and handlers can only be created while the scripts load.
func TestScriptOnlyWhileLoading(t *testing.T) {
	e := newEngine(t, map[string]string{"late.lua": `
local s = script.new("Late")
s:on_event(function(event, npc, player)
  if event == "new" then script.new("Later") end
  if event == "register" then s:on_talk(1, function() end) end
  return "fine"
end)`}, Options{})
	for _, event := range []string{"new", "register"} {
		reply, ok := scriptEvent(t, e, "Late", event, hero(), guardNpc())
		if ok || reply.Err == nil || !strings.Contains(reply.Err.Error(), "while the scripts load") {
			t.Errorf("%s: reply = %+v, %v; want a loading error", event, reply, ok)
		}
	}
	if e.HasScript("Later") {
		t.Error("a script was created after loading")
	}
}

func TestTalkDispatch(t *testing.T) {
	dir := writeScripts(t, map[string]string{
		"quests/a.lua": `local a = script.new("QuestA")
a:on_talk({ 30001, 30002 }, function(npc, player) return "a-" .. npc:id() .. ".htm" end)`,
		"quests/b.lua": `local b = script.new("QuestB")
b:on_talk(30001, function(npc, player) return "b.htm" end)`,
	})
	e := newEngine(t, nil, Options{Dir: dir})

	if got := e.TalkScripts(30001); !slices.Equal(got, []string{"QuestA", "QuestB"}) {
		t.Errorf("TalkScripts(30001) = %v", got)
	}
	if got := e.TalkScripts(30002); !slices.Equal(got, []string{"QuestA"}) {
		t.Errorf("TalkScripts(30002) = %v", got)
	}
	if got := e.TalkScripts(1); got != nil {
		t.Errorf("TalkScripts(1) = %v", got)
	}
	n := &fakeNpc{id: 0x60000010, npcID: 30001, name: "Lector"}
	if reply, ok := talk(t, e, "QuestA", 30001, hero(), n); !ok || reply.Content != "a-30001.htm" || reply.Dir != filepath.Join(dir, "quests") {
		t.Errorf("QuestA: %+v, %v", reply, ok)
	}
	if reply, ok := talk(t, e, "QuestB", 30001, hero(), n); !ok || reply.Content != "b.htm" {
		t.Errorf("QuestB: %+v, %v", reply, ok)
	}
	if reply, ok := talk(t, e, "QuestB", 30002, hero(), n); ok || reply.Err != nil {
		t.Errorf("QuestB at an NPC it does not talk at: %+v, %v", reply, ok)
	}
	if _, ok := talk(t, e, "Missing", 30001, hero(), n); ok {
		t.Error("a missing script answered")
	}
	if !e.HasScript("QuestA") || e.HasScript("Missing") {
		t.Error("HasScript is wrong")
	}
	if e.HasFirstTalk(30001) {
		t.Error("on_talk counts as first talk")
	}
}

func TestEventDispatch(t *testing.T) {
	e := newEngine(t, map[string]string{"events.lua": `
local s = script.new("Events")
s:on_event(function(event, npc, player)
  if event:sub(-4) == ".htm" then return event end
  if npc == nil then return "no npc: " .. event end
  return event .. " at " .. npc:id() .. " for " .. player:name()
end)
script.new("Quiet")`}, Options{})

	tests := []struct {
		event   string
		npc     NpcRef
		kind    ReplyKind
		content string
	}{
		{"buy 5", guardNpc(), ReplyText, "buy 5 at 30733 for Hero"},
		{"buy 5", nil, ReplyText, "no npc: buy 5"},
		{"30598-1.htm", guardNpc(), ReplyPage, "30598-1.htm"},
	}
	for _, tt := range tests {
		reply, ok := scriptEvent(t, e, "Events", tt.event, hero(), tt.npc)
		if !ok || reply.Kind != tt.kind || reply.Content != tt.content || reply.Script != "Events" {
			t.Errorf("%q: reply = %+v, %v; want %v %q", tt.event, reply, ok, tt.kind, tt.content)
		}
	}
	if reply, ok := scriptEvent(t, e, "Quiet", "x", hero(), guardNpc()); ok || reply.Script != "Quiet" || reply.Err != nil {
		t.Errorf("a script without on_event: %+v, %v", reply, ok)
	}
	if _, ok := scriptEvent(t, e, "Missing", "x", hero(), guardNpc()); ok {
		t.Error("a missing script answered")
	}
}

func TestReplyVars(t *testing.T) {
	e := newEngine(t, map[string]string{"vars.lua": `
local s = script.new("Vars")
s:on_event(function(event, npc, player)
  if event == "page" then return "p.htm", { name = player:name(), level = player:level(), ok = true } end
  if event == "html" then return "<html>%name%/%level%/%missing%</html>", { name = "Hero", ["%level%"] = 5 } end
  if event == "one_pass" then return "Hi %name%", { name = "%level%", level = "5" } end
  if event == "gsub" then return ("<html>x</html>"):gsub("x", "y") end
  if event == "replace" then return html.replace("<html>%a% and %b%</html>", { a = 1, b = "two" }) end
  if event == "bad_key" then return "p.htm", { "x" } end
  if event == "bad_value" then return "p.htm", { name = {} } end
end)`}, Options{})

	reply, ok := scriptEvent(t, e, "Vars", "page", hero(), guardNpc())
	if want := map[string]string{"name": "Hero", "level": "5", "ok": "true"}; !ok || reply.Kind != ReplyPage || !maps.Equal(reply.Vars, want) {
		t.Errorf("page: %+v, %v", reply, ok)
	}
	for event, want := range map[string]string{
		"html":     "<html>Hero/5/%missing%</html>",
		"one_pass": "Hi %level%",
		"gsub":     "<html>y</html>",
		"replace":  "<html>1 and two</html>",
	} {
		if reply, ok := scriptEvent(t, e, "Vars", event, hero(), guardNpc()); !ok || reply.Content != want {
			t.Errorf("%s: %+v, %v; want %q", event, reply, ok, want)
		}
	}
	for event, want := range map[string]string{"bad_key": "names must be strings", "bad_value": "has a table value"} {
		if reply, ok := scriptEvent(t, e, "Vars", event, hero(), guardNpc()); ok || reply.Err == nil || !strings.Contains(reply.Err.Error(), want) {
			t.Errorf("%s: %+v, %v; want an error with %q", event, reply, ok, want)
		}
	}
}

func TestReplaceVars(t *testing.T) {
	tests := []struct {
		text string
		vars map[string]string
		want string
	}{
		{"Hi %name%!", map[string]string{"name": "Bob"}, "Hi Bob!"},
		{"%a%%b%", map[string]string{"a": "%b%", "b": "B"}, "%b%B"},
		{"%name%", map[string]string{"%name%": "x"}, "x"},
		{"%name% %other%", map[string]string{"name": "x"}, "x %other%"},
		{"100%%", map[string]string{"": "x", "%": "y"}, "100%%"},
		{"no placeholders", map[string]string{"a": "b"}, "no placeholders"},
		{"%a%", nil, "%a%"},
	}
	for _, tt := range tests {
		if got := ReplaceVars(tt.text, tt.vars); got != tt.want {
			t.Errorf("ReplaceVars(%q, %v) = %q, want %q", tt.text, tt.vars, got, tt.want)
		}
	}
}

func TestHTMLReplaceInLua(t *testing.T) {
	newEngine(t, map[string]string{"h.lua": `
assert(html.replace("Hello, %name%!", { name = "Hero" }) == "Hello, Hero!")
assert(html.replace("%n% items", { n = 3 }) == "3 items")
assert(not pcall(html.replace, "x", 5), "vars must be a table")
assert(not pcall(html.replace, "x", { name = {} }), "values must be strings or numbers")`}, Options{})
}

func TestNpcHandlerError(t *testing.T) {
	logs := &logBuffer{}
	e := newEngine(t, map[string]string{"err.lua": `local s = script.new("Err")
s:on_first_talk(30733, function(npc, player)
  error("boom")
end)`}, Options{Logger: logs.logger()})
	reply, ok := firstTalk(t, e, 30733, hero(), guardNpc())
	he, isErr := errors.AsType[*Error](reply.Err)
	if ok || !isErr || !strings.HasSuffix(he.File, "err.lua") || he.Line != 3 || !strings.Contains(he.Msg, "boom") {
		t.Errorf("reply = %+v, %v; want err.lua:3 boom", reply, ok)
	}
	rec, logged := logs.find(t, "script handler failed")
	if !logged || rec["script"] != "Err" {
		t.Errorf("log = %v, %v", rec, logged)
	}
}

// A Go panic in a ref method (the game layer's code) fails the handler,
// not the server.
func TestNpcHandlerGoPanic(t *testing.T) {
	e := newEngine(t, map[string]string{"p.lua": `local s = script.new("P")
s:on_first_talk(30733, function(npc, player) return player:name() end)`}, Options{})
	p := hero()
	p.onCall = func() { panic("game layer bug") }
	if reply, ok := firstTalk(t, e, 30733, p, guardNpc()); ok || reply.Err == nil || !strings.Contains(reply.Err.Error(), "game layer bug") {
		t.Errorf("reply = %+v, %v; want the panic as an error", reply, ok)
	}
	if reply, ok := firstTalk(t, e, 30733, hero(), guardNpc()); !ok || reply.Content != "Hero" {
		t.Errorf("after the panic: %+v, %v", reply, ok)
	}
}

func TestNpcHandlerTimeout(t *testing.T) {
	e := newEngine(t, map[string]string{"loop.lua": `local s = script.new("Loop")
s:on_first_talk(30733, function(npc, player)
  if player:name() == "loop" then while true do end end
  return "ok.htm"
end)`}, Options{Timeout: 20 * time.Millisecond})
	begin := time.Now()
	reply, ok := firstTalk(t, e, 30733, &fakePlayer{name: "loop"}, guardNpc())
	if ok || !errors.Is(reply.Err, ErrTimeout) {
		t.Errorf("reply = %+v, %v; want ErrTimeout", reply, ok)
	}
	if d := time.Since(begin); d > 2*time.Second {
		t.Errorf("the endless loop ran for %v", d)
	}
	if reply, ok := firstTalk(t, e, 30733, hero(), guardNpc()); !ok || reply.Content != "ok.htm" {
		t.Errorf("after the timeout: %+v, %v", reply, ok)
	}
}

func TestNpcSay(t *testing.T) {
	src := map[string]string{"say.lua": `local s = script.new("Talker")
s:on_first_talk(30733, function(npc, player)
  npc:say("Hello, " .. player:name())
  npc:say("")
  return ""
end)`}
	out := &fakeOutput{}
	e := newEngine(t, src, Options{Output: out})
	reply, ok := firstTalk(t, e, 30733, hero(), guardNpc())
	if !ok || reply.Kind != ReplyNone {
		t.Errorf("reply = %+v, %v; want ReplyNone", reply, ok)
	}
	if len(out.said) != 1 || out.said[0].text != "Hello, Hero" || out.said[0].npc.ID() != 30733 {
		t.Errorf("said = %+v", out.said)
	}

	// Without an Output, npc:say does nothing.
	quiet := newEngine(t, src, Options{})
	if reply, ok := firstTalk(t, quiet, 30733, hero(), guardNpc()); !ok || reply.Err != nil {
		t.Errorf("without an output: %+v, %v", reply, ok)
	}
}

// Player values work only during the run that handed them out. Npc values
// are handles: without a world to look them up in, they work again only
// when a later run hands out the same NPC.
func TestRefsExpire(t *testing.T) {
	e := newEngine(t, map[string]string{"keep.lua": `
local kept_player, kept_npc
local s = script.new("Keeper")
s:on_first_talk(30733, function(npc, player)
  kept_player, kept_npc = player, npc
  return ""
end)
s:on_event(function(event)
  if event == "player" then return kept_player:name() end
  if event == "npc" then return kept_npc:name() end
  if event == "tostring" then return tostring(kept_player) .. "/" .. tostring(kept_npc) end
end)`}, Options{})
	if _, ok := firstTalk(t, e, 30733, hero(), guardNpc()); !ok {
		t.Fatal("first talk failed")
	}
	other := &fakeNpc{id: 0x60000002, npcID: 30734, name: "Other"}
	if reply, ok := scriptEvent(t, e, "Keeper", "player", hero(), other); ok || reply.Err == nil || !strings.Contains(reply.Err.Error(), "earlier handler") {
		t.Errorf("player: reply = %+v, %v; want an expired value error", reply, ok)
	}
	if reply, ok := scriptEvent(t, e, "Keeper", "npc", hero(), other); ok || reply.Err == nil || !strings.Contains(reply.Err.Error(), "npc:valid()") {
		t.Errorf("npc: reply = %+v, %v; want a gone NPC error", reply, ok)
	}
	if reply, _ := scriptEvent(t, e, "Keeper", "tostring", hero(), other); reply.Content != "player (expired)/npc 30733 (object 1610612737, gone)" {
		t.Errorf("tostring = %q", reply.Content)
	}
	// The same NPC handed out again: the kept handle is that NPC.
	if reply, _ := scriptEvent(t, e, "Keeper", "npc", hero(), guardNpc()); reply.Content != "Guard" {
		t.Errorf("npc again = %+v", reply)
	}
}

func TestRejectInNpcHandler(t *testing.T) {
	e := newEngine(t, map[string]string{"r.lua": `local s = script.new("R")
s:on_first_talk(30733, function() return events.reject("incorrect_name") end)`}, Options{})
	if reply, ok := firstTalk(t, e, 30733, hero(), guardNpc()); ok || reply.Err == nil || !strings.Contains(reply.Err.Error(), "outside an event handler") {
		t.Errorf("reply = %+v, %v", reply, ok)
	}
}

// Page names are relative to the folder of the file that called
// script.new, also when a lib/ helper made the call.
func TestScriptDir(t *testing.T) {
	dir := writeScripts(t, map[string]string{
		"lib/maker.lua": `return function(name)
  local s = script.new(name)
  s:on_event(function() return "page.htm" end)
  return s
end`,
		"quests/q1.lua":     `require("lib.maker")("Q1")`,
		"npcs/deep/npc.lua": `script.new("Deep"):on_event(function() return "page.htm" end)`,
	})
	e := newEngine(t, nil, Options{Dir: dir})
	for script, want := range map[string]string{"Q1": filepath.Join(dir, "quests"), "Deep": filepath.Join(dir, "npcs", "deep")} {
		if reply, ok := scriptEvent(t, e, script, "x", hero(), nil); !ok || reply.Dir != want {
			t.Errorf("%s: Dir = %q, want %q", script, reply.Dir, want)
		}
	}
}

func TestReplyKindString(t *testing.T) {
	for kind, want := range map[ReplyKind]string{ReplyNone: "none", ReplyPage: "page", ReplyHTML: "html", ReplyText: "text", 9: "ReplyKind(9)"} {
		if got := kind.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", int(kind), got, want)
		}
	}
}
