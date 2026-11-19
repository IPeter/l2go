package scripting_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// scriptsDir is the scripts folder shipped with the server.
var scriptsDir = filepath.Join("..", "..", "scripts")

// Every script in scripts/ loads, so a broken script fails go test.
func TestEveryScriptLoads(t *testing.T) {
	n, err := scripting.Check(context.Background(), scripting.Options{Dir: scriptsDir})
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatalf("no scripts found in %s", scriptsDir)
	}
}

// The default rule: names containing a Mobius ForbiddenNames word are
// refused with incorrect_name, ignoring case.
func TestDefaultCharacterCreateScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "events/character_create.lua")
	tests := []struct {
		name   string
		reject bool
	}{
		{"Frostwind", false},
		{"Hero", false},
		{"Elf", false},
		{"Annou", true},
		{"announcer", true},
		{"MyAnnouncer", true},
		{"AMMOU", true},
		{"xAmnOux", true},
		{"anmou1", true},
		{"anou", true},
		{"Kamou", true},
		{"Ano", false},
		{"Amo", false},
		{"", false},
	}
	for _, tt := range tests {
		res := h.CharacterCreate(scripting.CharacterCreate{Account: "acc", Name: tt.name})
		if res.Rejected != tt.reject {
			t.Errorf("%q: rejected = %v, want %v", tt.name, res.Rejected, tt.reject)
		}
		if tt.reject && res.Reason != scripting.ReasonIncorrectName {
			t.Errorf("%q: reason = %q, want %q", tt.name, res.Reason, scripting.ReasonIncorrectName)
		}
	}
}

func TestDefaultCharacterCreateScriptLogs(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "events/character_create.lua")
	h.ClearLogs()
	h.CharacterCreate(scripting.CharacterCreate{Account: "acc1", Name: "Announcer"})
	for _, l := range h.Logs() {
		if l.Msg == "character name refused" {
			if l.Attrs["account"] != "acc1" || l.Attrs["name"] != "Announcer" || l.Attrs["word"] != "annou" {
				t.Errorf("log attrs = %v", l.Attrs)
			}
			return
		}
	}
	t.Errorf("no 'character name refused' log in %v", h.Logs())
}

const guardScript = "TalkingIslandGuard"

// guardLinks returns the events of the guard script's links in page.
func guardLinks(page string) []string {
	link := regexp.MustCompile(`bypass -h Script ` + guardScript + ` ([^"]+)"`)
	var events []string
	for _, m := range link.FindAllStringSubmatch(page, -1) {
		events = append(events, m[1])
	}
	return events
}

// The example NPC script: newcomers get the welcome page, everyone else
// the guards' usual dialog, and every link on the script's pages works.
func TestTalkingIslandGuardScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "npcs/talking_island_guard/guard.lua")
	newcomer := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 5}
	veteran := &scripttest.Player{ObjectID: 0x10000002, Name: "Veteran", Level: 20}

	for _, id := range []int{30733, 31032} {
		guard := &scripttest.Npc{ObjectID: 0x60000001, ID: id, Name: "Guard"}
		reply, ok := h.FirstTalk(newcomer, guard)
		if !ok || reply.Kind != scripting.ReplyPage || reply.Content != "welcome.htm" {
			t.Errorf("guard %d: reply = %+v, %v; want welcome.htm", id, reply, ok)
		}
		if reply.Vars["name"] != "Hero" || reply.Vars["level"] != "5" {
			t.Errorf("guard %d: vars = %v", id, reply.Vars)
		}
		if reply, ok := h.FirstTalk(veteran, guard); ok {
			t.Errorf("guard %d: veteran got %+v, want the default dialog", id, reply)
		}
	}

	// The page is next to the script; with the vars, no placeholder is
	// left, and each of its links has an answer.
	guard := &scripttest.Npc{ObjectID: 0x60000001, ID: 30733, Name: "Guard"}
	reply, _ := h.FirstTalk(newcomer, guard)
	data, err := os.ReadFile(filepath.Join(reply.Dir, filepath.FromSlash(reply.Content)))
	if err != nil {
		t.Fatal(err)
	}
	page := scripting.ReplaceVars(string(data), reply.Vars)
	if strings.Contains(page, "%") {
		t.Errorf("unfilled placeholder in:\n%s", page)
	}
	links := guardLinks(page)
	if !slices.Contains(links, "directions") || !slices.Contains(links, "salute") {
		t.Errorf("welcome.htm links = %v", links)
	}
	for _, event := range links {
		if _, ok := h.Event(guardScript, event, newcomer, guard); !ok {
			t.Errorf("link %q has no answer", event)
		}
	}

	reply, ok := h.Event(guardScript, "directions", newcomer, guard)
	if !ok || reply.Kind != scripting.ReplyHTML || !strings.Contains(reply.Content, "Hero") {
		t.Errorf("directions = %+v, %v", reply, ok)
	}
	if links := guardLinks(reply.Content); !slices.Equal(links, []string{"welcome"}) {
		t.Errorf("directions links = %v, want [welcome]", links)
	}
	if reply, ok := h.Event(guardScript, "welcome", newcomer, guard); !ok || reply.Content != "welcome.htm" {
		t.Errorf("welcome = %+v, %v", reply, ok)
	}

	h.ClearSaid()
	reply, ok = h.Event(guardScript, "salute", newcomer, guard)
	if !ok || reply.Kind != scripting.ReplyNone {
		t.Errorf("salute = %+v, %v; want ReplyNone", reply, ok)
	}
	want := []scripttest.Speech{{NpcID: 30733, ObjectID: 0x60000001, Text: "Stay on the roads, Hero!"}}
	if said := h.Said(); !slices.Equal(said, want) {
		t.Errorf("said = %+v, want %+v", said, want)
	}

	if reply, ok := h.Event(guardScript, "nonsense", newcomer, guard); ok {
		t.Errorf("an unknown event answered %+v", reply)
	}
}
