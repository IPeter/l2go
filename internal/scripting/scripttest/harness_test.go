package scripttest

import (
	"context"
	"log/slog"
	"testing"

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

func TestReloadThroughHarness(t *testing.T) {
	h := LoadSource(t, map[string]string{"a.lua": ""})
	WriteScript(t, h.Dir(), "a.lua", `events.on_character_create(function() return events.reject("creation_not_allowed") end)`)
	if err := h.Engine().Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	if res := h.CharacterCreate(scripting.CharacterCreate{Name: "x"}); res.Reason != scripting.ReasonCreationNotAllowed {
		t.Errorf("result = %+v after reload", res)
	}
}
