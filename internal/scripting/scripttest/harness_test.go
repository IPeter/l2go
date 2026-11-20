package scripttest

import (
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/scripting"
)

func TestLoadSource(t *testing.T) {
	h := LoadSource(t, map[string]string{
		"lib/words.lua": `return { "bad" }`,
		"events/a.lua": `local words = require("lib.words")
events.on_character_create(function(ev)
  log.warn("checking", "name", ev.name)
  for _, w in ipairs(words) do
    if ev.name == w then return events.reject("name_exists") end
  end
end)`,
	})
	if res := h.CharacterCreate(scripting.CharacterCreate{Name: "bad"}); !res.Rejected || res.Reason != scripting.ReasonNameExists {
		t.Errorf("bad: %+v", res)
	}
	if res := h.CharacterCreate(scripting.CharacterCreate{Name: "good"}); res.Rejected {
		t.Errorf("good: %+v", res)
	}

	var names []any
	for _, l := range h.Logs() {
		if l.Msg == "checking" {
			if l.Level != slog.LevelWarn {
				t.Errorf("level = %v, want WARN", l.Level)
			}
			names = append(names, l.Attrs["name"])
		}
	}
	if len(names) != 2 || names[0] != "bad" || names[1] != "good" {
		t.Errorf("logged names = %v", names)
	}
	h.ClearLogs()
	if len(h.Logs()) != 0 {
		t.Error("ClearLogs kept records")
	}
}

func TestLoadOnlyGivenFiles(t *testing.T) {
	dir := WriteScripts(t, map[string]string{
		"events/one.lua": `events.on_character_create(function() return events.reject("name_exists") end)`,
		"events/two.lua": `events.on_character_create(function() return events.reject("incorrect_name") end)`,
	})
	h := Load(t, dir, "events/two.lua")
	if res := h.CharacterCreate(scripting.CharacterCreate{Name: "x"}); res.Reason != scripting.ReasonIncorrectName {
		t.Errorf("result = %+v, want only events/two.lua loaded", res)
	}
}

func TestFireAllowErrors(t *testing.T) {
	h := LoadSource(t, map[string]string{
		"a.lua": `events.on_character_create(function() error("boom") end)`,
	})
	res := h.FireAllowErrors(scripting.CharacterCreate{Name: "x"}.Event())
	if res.Err == nil || res.Rejected {
		t.Errorf("result = %+v, want a handler error", res)
	}
}

func TestReload(t *testing.T) {
	h := LoadSource(t, map[string]string{"a.lua": ""})
	WriteScript(t, h.Dir(), "a.lua", `events.on_character_create(function() return events.reject("creation_not_allowed") end)`)
	if err := h.Reload(); err != nil {
		t.Fatal(err)
	}
	if res := h.CharacterCreate(scripting.CharacterCreate{Name: "x"}); res.Reason != scripting.ReasonCreationNotAllowed {
		t.Errorf("result = %+v after reload", res)
	}

	WriteScript(t, h.Dir(), "a.lua", `events.on_character_create(`)
	if err := h.Reload(); err == nil {
		t.Error("Reload of a broken script succeeded")
	}
	if res := h.CharacterCreate(scripting.CharacterCreate{Name: "x"}); res.Reason != scripting.ReasonCreationNotAllowed {
		t.Errorf("result = %+v after a failed reload, want the old script", res)
	}
}

func TestChat(t *testing.T) {
	h := LoadSource(t, map[string]string{"chat.lua": `
events.on_chat(function(ev)
  if ev.type == "trade" and ev.player:level() < 10 then return false end
  return ev.player:name() .. ": " .. ev.text
end)`})
	p := &Player{ObjectID: 1, Name: "Hero", Level: 5}
	if res := h.Chat(scripting.ChatMessage{Player: p.Ref(), Type: 0, Text: "hi"}); res.Dropped || res.Text != "Hero: hi" {
		t.Errorf("general = %+v", res)
	}
	if res := h.Chat(scripting.ChatMessage{Player: p.Ref(), Type: 8, Text: "WTS"}); !res.Dropped {
		t.Errorf("trade = %+v, want dropped", res)
	}
	p.Level = 10 // the ref reads the fields at call time
	if res := h.Chat(scripting.ChatMessage{Player: p.Ref(), Type: 8, Text: "WTS"}); res.Dropped {
		t.Errorf("trade at level 10 = %+v", res)
	}
}

// The fake quest state follows Mobius's __compltdStateFlags arithmetic.
func TestQuestStateCondFlags(t *testing.T) {
	w := NewWorld()
	p := &Player{ObjectID: 1}
	st := w.NewQuestState(p.Ref(), "Q1")
	st.SetState(scripting.StateStarted)
	steps := []struct {
		cond  string
		flags string // "" for unset
	}{
		{"1", ""},
		{"2", ""},
		{"4", "-2147483637"}, // 0x80000001 | 0b11 | 0b1000: step 3 skipped
		{"5", "-2147483621"}, // | 0b10000
	}
	for i, s := range steps {
		st.Set("cond", s.cond)
		got, ok := st.Get("__compltdStateFlags")
		if (s.flags == "" && ok) || (s.flags != "" && got != s.flags) {
			t.Errorf("step %d cond %s: flags %q (set %v), want %q", i, s.cond, got, ok, s.flags)
		}
	}
	// Back below 3 always drops the flags.
	st.Set("cond", "2")
	if got, ok := st.Get("__compltdStateFlags"); ok {
		t.Errorf("back to 2: flags %q, want unset", got)
	}
	// 1 → 4 skips steps 2 and 3; back to 3 keeps step 1 done, step 2
	// skipped, and the skip mark.
	st.Set("cond", "1")
	st.Set("cond", "4")
	if got, _ := st.Get("__compltdStateFlags"); got != "-2147483639" {
		t.Errorf("1, 4: flags %q, want -2147483639", got)
	}
	st.Set("cond", "3")
	if got, _ := st.Get("__compltdStateFlags"); got != "-2147483647" {
		t.Errorf("1, 4, 3: flags %q, want -2147483647", got)
	}
	if st.Cond() != 3 {
		t.Errorf("cond = %d", st.Cond())
	}
	if w.QuestLists[1] == 0 || len(w.QuestMarks) == 0 {
		t.Error("no QuestList or quest mark recorded")
	}
}

// Timers run in due order, then start order, each at its time.
func TestWorldTimers(t *testing.T) {
	w := NewWorld()
	var got []string
	w.After(2*time.Second, func() { got = append(got, "b@"+w.Now().Sub(Start).String()) })
	w.After(time.Second, func() { got = append(got, "a@"+w.Now().Sub(Start).String()) })
	w.After(2*time.Second, func() { got = append(got, "c@"+w.Now().Sub(Start).String()) })
	stop := w.Every(1500*time.Millisecond, func() { got = append(got, "r@"+w.Now().Sub(Start).String()) })
	w.Advance(3 * time.Second)
	stop()
	w.Advance(3 * time.Second)
	want := []string{"a@1s", "r@1.5s", "b@2s", "c@2s", "r@3s"}
	if !slices.Equal(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
	if w.Pending() != 0 {
		t.Errorf("pending = %d", w.Pending())
	}
}

func TestNpcHelpers(t *testing.T) {
	h := LoadSource(t, map[string]string{"npcs/shop.lua": `
local s = script.new("Shop")
s:on_first_talk(30001, function(npc, player)
  npc:say("Welcome, " .. player:name())
  return "shop.htm", { name = player:name() }
end)
s:on_talk(30001, function(npc, player) return "<html><body>Quest talk</body></html>" end)
s:on_event(function(event, npc, player)
  if npc == nil then return "far away" end
  return ""
end)`})
	hero := &Player{ObjectID: 0x10000001, Name: "Hero", Level: 5}
	lector := &Npc{ObjectID: 0x60000001, ID: 30001, Name: "Lector"}

	reply, ok := h.FirstTalk(hero, lector)
	if !ok || reply.Kind != scripting.ReplyPage || reply.Content != "shop.htm" || reply.Vars["name"] != "Hero" || reply.Script != "Shop" {
		t.Errorf("first talk = %+v, %v", reply, ok)
	}
	want := []Speech{{NpcID: 30001, ObjectID: 0x60000001, Text: "Welcome, Hero"}}
	if said := h.Said(); !slices.Equal(said, want) {
		t.Errorf("said = %+v, want %+v", said, want)
	}
	h.ClearSaid()
	if len(h.Said()) != 0 {
		t.Error("ClearSaid kept speech")
	}

	if reply, ok := h.Talk("Shop", hero, lector); !ok || reply.Kind != scripting.ReplyHTML {
		t.Errorf("talk = %+v, %v", reply, ok)
	}
	if reply, ok := h.Event("Shop", "buy", hero, lector); !ok || reply.Kind != scripting.ReplyNone {
		t.Errorf("event = %+v, %v", reply, ok)
	}
	if reply, ok := h.Event("Shop", "buy", hero, nil); !ok || reply.Content != "far away" {
		t.Errorf("event without an NPC = %+v, %v", reply, ok)
	}
	if _, ok := h.FirstTalk(hero, &Npc{ID: 30002}); ok {
		t.Error("an NPC without a script answered")
	}
}
