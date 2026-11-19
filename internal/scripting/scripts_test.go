package scripting_test

import (
	"context"
	"path/filepath"
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
