package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Little Wing: Mimyu hides from a player without business with her, moving
// in turn to her three places (npc:teleport). The quest stays behind pets.
func TestFixQ420MimyuTeleports(t *testing.T) {
	q := b3aStart(t, "Q00420_LittleWing")
	b3cRequires(t, q.h, q.name, "pets")
	w := q.h.World()
	mimyu := &scripttest.Npc{ObjectID: 0x60000001, ID: 30747, Name: "Mimyu"}
	cooper := &scripttest.Npc{ObjectID: 0x60000002, ID: 30829, Name: "Cooper"}

	stranger := &scripttest.Player{ObjectID: 0x10000001, Name: "Stranger", Level: 40}
	want := []scripting.Location{
		{X: 108940, Y: 41615, Z: -4643},
		{X: 110395, Y: 41625, Z: -4642},
		{X: 109816, Y: 40854, Z: -4640},
	}
	for i, loc := range want {
		q.talk(stranger, mimyu, "")
		if mimyu.X != loc.X || mimyu.Y != loc.Y || mimyu.Z != loc.Z {
			t.Errorf("talk %d: Mimyu at %d, %d, %d, want %+v", i+1, mimyu.X, mimyu.Y, mimyu.Z, loc)
		}
	}
	if got := fixActions(w, "npc_teleport"); len(got) != 3 {
		t.Errorf("teleports %+v, want 3", got)
	}

	// A started player before cond 4 sends her away too.
	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 35}
	q.talk(hero, cooper, "30829-01.htm")
	q.event(hero, cooper, "30829-02.htm", "30829-02.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	w.ClearRecords()
	q.talk(hero, mimyu, "")
	if got := fixActions(w, "npc_teleport"); len(got) != 1 || got[0].Loc.X != 108940 {
		t.Errorf("started: teleports %+v", got)
	}
}
