package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Helpers of the quests ported once the Lua API had the NPC AI, skill and
// item calls (fix_*). Every top-level name starts with fix: other batches
// share this package. The b3a helpers (talk, event, kill, want) do the
// quest steps.

// fixActions returns the recorded actions of a kind, in order.
func fixActions(w *scripttest.World, kind string) []scripttest.Action {
	var out []scripttest.Action
	for _, a := range w.Actions {
		if a.Kind == kind {
			out = append(out, a)
		}
	}
	return out
}

// fixWantAction fails unless an action of the kind was done by object on
// target (0: any target) with value (0: any value).
func fixWantAction(t *testing.T, what string, w *scripttest.World, kind string, object, target int32, value int) {
	t.Helper()
	for _, a := range fixActions(w, kind) {
		if a.Object == object && (target == 0 || a.Target == target) && (value == 0 || a.Value == value) {
			return
		}
	}
	t.Errorf("%s: no %s by %#x on %#x (value %d) in %+v", what, kind, object, target, value, w.Actions)
}

// fixSays reports whether an NPC said text.
func fixSays(w *scripttest.World, npc int32, text string) bool {
	for _, a := range fixActions(w, "say") {
		if a.Object == npc && a.Text == text {
			return true
		}
	}
	return false
}

// fixSpawnedOf returns the NPCs of a template spawned so far.
func fixSpawnedOf(w *scripttest.World, npcID int) []*scripttest.Npc {
	var out []*scripttest.Npc
	for _, n := range w.Spawned {
		if n.ID == npcID {
			out = append(out, n)
		}
	}
	return out
}
