package game

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/scripting"
)

// The monster flags of the AI scripts on the game: lethal strikes, Silent
// Move, the world region and the template's minion lists.
func TestScriptAIMonsterFlags(t *testing.T) {
	start := npcTestStart(t)
	e := newAIEnv(t, aiTestWorld(start), newAIRand(), nil, false)
	_, p := statusPlayer(t, e, "alice", "Alice")
	k := aiSpawn(e, aiKeltir, start, 200, 0)
	o := scriptOutputOf(e)
	ref := k.ScriptRef()

	var lethal, notLethal, active, hidesSeen, hidesPlain bool
	var minions []scripting.TemplateMinion
	e.onLoop(func() {
		lethal = k.skill.IsLethalable()
		o.NpcSetLethalable(ref, false)
		notLethal = !k.skill.IsLethalable()
		o.NpcSetLethalable(ref, true)
		lethal = lethal && k.skill.IsLethalable()

		active = o.NpcInActiveRegion(ref)

		// Without Silent Move on the player nothing hides; a monster that
		// sees through it is never fooled.
		hidesPlain = silentMoveHides(k, p)
		o.NpcSetSeeThroughSilentMove(ref, true)
		hidesSeen = silentMoveHides(k, p)

		k.Template.Parameters.Minions = map[string][]data.NpcMinion{"Privates": {{NpcID: 20768, Count: 1}, {NpcID: 20769, Count: 2}}}
		minions = ref.(scripting.NpcTemplateMinions).TemplateMinions("Privates")
	})
	if !lethal || !notLethal {
		t.Errorf("lethalable (before and after) %v, after set_lethalable(false) not %v; want true, true", lethal, notLethal)
	}
	if !active {
		t.Errorf("a keltir next to a player is in an inactive region")
	}
	if hidesPlain || hidesSeen {
		t.Errorf("hides: plain %v, seeing through %v; want false, false", hidesPlain, hidesSeen)
	}
	want := []scripting.TemplateMinion{{ID: 20768, Count: 1}, {ID: 20769, Count: 2}}
	if !slices.Equal(minions, want) {
		t.Errorf("minions = %+v, want %+v", minions, want)
	}
}
