package game

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/testclient"
)

// The first click of a player that entered the world ends its spawn
// protection, with a message before the click's answer outside a peace
// zone (Mobius Action: onActionRequest before onAction). Inside one, a walk
// ends it silently. Later actions say nothing.
func TestSpawnProtection(t *testing.T) {
	e := newNpcEnv(t, func(gs *config.GameServer) { gs.World.SpawnProtection = config.Duration(time.Minute) })
	sa, sb, pa, pb := twoPlayers(t, e)
	walkSettle(t, e, sa, sb)
	roxxy := npcSpawnedOne(t, e, npcTestRoxxy)
	protected := func(p *Player) bool {
		var ok bool
		e.onLoop(func() { ok = e.srv.SpawnProtected(p) })
		return ok
	}
	if !protected(pa) || !protected(pb) {
		t.Fatal("players are not protected after entering the world")
	}

	npcTestClick(e, sa, roxxy, false)
	ps := sa.take(t)
	checkNames(t, "Alice's first click", ps, "SystemMessage", "ValidateLocation", "MyTargetSelected", "StatusUpdate", "ActionFailed")
	if m := ps[0].(testclient.SystemMessage); !slices.Equal(m.Texts(), []string{spawnProtectionEnded}) {
		t.Errorf("SystemMessage %+v, want %q", m, spawnProtectionEnded)
	}
	if protected(pa) {
		t.Error("Alice is still protected after a click")
	}
	npcTestClick(e, sa, roxxy, false)
	if n := len(only[testclient.SystemMessage](sa.take(t))); n != 0 {
		t.Errorf("Alice's second click got %d SystemMessage, want none", n)
	}

	e.onLoop(func() { pb.Zones.Enter(ZonePeace) })
	home := e.position(pb)
	walkClick(e, sb, home, walkOffset(home, 100, 0, 0))
	if n := len(only[testclient.SystemMessage](sb.take(t))); n != 0 {
		t.Errorf("Bob's walk in a peace zone got %d SystemMessage, want none", n)
	}
	if protected(pb) {
		t.Error("Bob is still protected after a walk")
	}
}

// Without the setting nobody is protected.
func TestSpawnProtectionOff(t *testing.T) {
	e := newNpcEnv(t, nil)
	_, _, pa, _ := twoPlayers(t, e)
	var ok bool
	e.onLoop(func() { ok = e.srv.SpawnProtected(pa) })
	if ok {
		t.Error("a player is protected with spawnProtection 0")
	}
}
