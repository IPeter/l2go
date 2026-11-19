package scripting

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func chat(t testing.TB, e *Engine, msg ChatMessage) Result {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return e.Invoke(ctx, msg.Event())
}

// say is a general chat message from hero.
func say(text string) ChatMessage {
	return ChatMessage{Player: hero(), Type: 0, Text: text}
}

func TestChatWithoutHandlers(t *testing.T) {
	e := newEngine(t, nil, Options{})
	if res := chat(t, e, say("hello")); res.Dropped || res.Text != "hello" || res.Err != nil {
		t.Errorf("result = %+v, want the text unchanged", res)
	}
}

func TestChatReplaceAndDrop(t *testing.T) {
	logs := &logBuffer{}
	e := newEngine(t, map[string]string{
		"a.lua": `events.on_chat(function(ev)
  if ev.text == "drop me" then return false end
  if ev.text == "empty" then return "" end
  if ev.text == "keep" then return nil end
  return (ev.text:gsub("darn", "d**n"))
end)`,
		"b.lua": `events.on_chat(function(ev)
  log.info("b saw", "text", ev.text)
  if ev.text:find("d**n", 1, true) then return ev.text .. "!" end
end)`,
	}, Options{Logger: logs.logger()})

	tests := []struct {
		text string
		want string
		drop bool
	}{
		{"hello", "hello", false},
		{"keep", "keep", false},
		{"darn it", "d**n it!", false}, // b sees a's replacement
		{"drop me", "", true},
		{"empty", "", true},
	}
	for _, tt := range tests {
		res := chat(t, e, say(tt.text))
		if res.Err != nil || res.Dropped != tt.drop || (!tt.drop && res.Text != tt.want) {
			t.Errorf("%q: result = %+v, want text %q dropped %v", tt.text, res, tt.want, tt.drop)
		}
	}
	for _, rec := range logs.records(t) {
		if rec["msg"] == "b saw" && (rec["text"] == "drop me" || rec["text"] == "empty") {
			t.Errorf("a handler ran after the message was dropped: %v", rec)
		}
	}
}

func TestChatEventTable(t *testing.T) {
	e := newEngine(t, map[string]string{"check.lua": `
events.on_chat(function(ev)
  assert(ev.player:name() == "Hero", "player name")
  assert(ev.player:object_id() == 268435457, "object id")
  assert(ev.player:level() == 5, "level")
  if ev.type == "whisper" then
    assert(ev.target == "Friend", "target")
  else
    assert(ev.target == nil, "no target")
  end
  return ev.type
end)`}, Options{})
	types := map[int]string{
		0: "general", 1: "shout", 2: "whisper", 3: "party", 4: "clan", 8: "trade",
		9: "alliance", 17: "hero_voice", 21: "mpcc_room", 22: "unknown", -1: "unknown",
	}
	for id, want := range types {
		msg := ChatMessage{Player: hero(), Type: id, Text: "x"}
		if id == 2 {
			msg.Target = "Friend"
		}
		if res := chat(t, e, msg); res.Err != nil || res.Text != want {
			t.Errorf("type %d: result = %+v, want %q", id, res, want)
		}
	}
}

func TestChatNoPlayer(t *testing.T) {
	e := newEngine(t, map[string]string{"a.lua": `
events.on_chat(function(ev)
  assert(ev.player == nil)
  return "system: " .. ev.text
end)`}, Options{})
	if res := chat(t, e, ChatMessage{Text: "x"}); res.Err != nil || res.Text != "system: x" {
		t.Errorf("result = %+v", res)
	}
}

func TestChatBadReturn(t *testing.T) {
	for name, ret := range map[string]string{"true": "true", "number": "5", "table": "{}"} {
		t.Run(name, func(t *testing.T) {
			e := newEngine(t, map[string]string{"a.lua": `events.on_chat(function(ev) return ` + ret + ` end)`}, Options{})
			res := chat(t, e, say("hello"))
			if res.Err == nil || !strings.Contains(res.Err.Error(), "returned") {
				t.Errorf("Err = %v, want a bad return error", res.Err)
			}
			if res.Dropped || res.Text != "hello" {
				t.Errorf("result = %+v, want the text unchanged", res)
			}
		})
	}
}

// A failing handler is skipped; the message goes on with the text so far.
func TestChatHandlerErrorKeepsText(t *testing.T) {
	e := newEngine(t, map[string]string{
		"a.lua": `events.on_chat(function(ev) return "y" end)`,
		"b.lua": "events.on_chat(function(ev)\n  error('filter broke')\nend)",
		"c.lua": `events.on_chat(function(ev) return ev.text .. "!" end)`,
	}, Options{})
	res := chat(t, e, say("x"))
	if res.Dropped || res.Text != "y!" {
		t.Errorf("result = %+v, want y!", res)
	}
	if he, ok := errors.AsType[*Error](res.Err); !ok || !strings.HasSuffix(he.File, "b.lua") || he.Line != 2 {
		t.Errorf("Err = %v, want b.lua:2", res.Err)
	}
}

func TestChatTimeout(t *testing.T) {
	e := newEngine(t, map[string]string{
		"a.lua": `events.on_chat(function(ev) return "changed" end)`,
		"b.lua": `events.on_chat(function(ev) while true do end end)`,
	}, Options{Timeout: 20 * time.Millisecond})
	res := chat(t, e, say("x"))
	if !errors.Is(res.Err, ErrTimeout) {
		t.Errorf("Err = %v, want ErrTimeout", res.Err)
	}
	if res.Dropped || res.Text != "changed" {
		t.Errorf("result = %+v, want the text so far", res)
	}
}

func TestChatCannotReject(t *testing.T) {
	e := newEngine(t, map[string]string{"a.lua": `events.on_chat(function(ev) return events.reject("incorrect_name") end)`}, Options{})
	res := chat(t, e, say("x"))
	if res.Rejected || res.Err == nil || !strings.Contains(res.Err.Error(), "return false to drop") {
		t.Errorf("result = %+v, want a 'cannot be rejected' error", res)
	}
}

// A player kept from an earlier event cannot be used.
func TestChatPlayerExpires(t *testing.T) {
	e := newEngine(t, map[string]string{"keep.lua": `
local saved
events.on_chat(function(ev)
  if ev.text == "save" then saved = ev.player; return end
  if ev.text == "show" then return tostring(saved) end
  if ev.text == "use" then return saved:name() end
end)`}, Options{})
	if res := chat(t, e, say("save")); res.Err != nil {
		t.Fatal(res.Err)
	}
	if res := chat(t, e, say("show")); res.Text != "player (expired)" {
		t.Errorf("tostring = %q", res.Text)
	}
	if res := chat(t, e, say("use")); res.Err == nil || !strings.Contains(res.Err.Error(), "earlier handler") {
		t.Errorf("Err = %v, want an expired player error", res.Err)
	}
}

func TestChatTypeName(t *testing.T) {
	if len(chatTypes) != 22 {
		t.Errorf("%d chat types, want Mobius's 22", len(chatTypes))
	}
	for id, want := range map[int]string{0: "general", 1: "shout", 2: "whisper", 8: "trade", 17: "hero_voice", 22: "unknown", -5: "unknown"} {
		if got := chatTypeName(id); got != want {
			t.Errorf("chatTypeName(%d) = %q, want %q", id, got, want)
		}
	}
}
