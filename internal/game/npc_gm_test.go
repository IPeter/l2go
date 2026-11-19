package game

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
)

// npcGMEnv starts the NPC test world with a GM in it.
func npcGMEnv(t *testing.T) (*testEnv, *fakeSession, *Player) {
	t.Helper()
	e := newNpcEnv(t, nil)
	e.createCharacter("gm", "Gm", classHumanFighter, storage.AccessLevelAdmin)
	sg := newSession("gm")
	pg := e.enter(sg, 0)
	sg.take(t)
	return e, sg, pg
}

// npcGM runs a GM command line from sess and returns what sess got.
func npcGM(t *testing.T, e *testEnv, sess *fakeSession, line string) []testclient.Packet {
	t.Helper()
	e.srv.Handle(sess, clientpackets.OpSendBypassBuildCmd, stringBody(line))
	e.sync()
	return sess.take(t)
}

// npcSysMsgIDs returns the IDs of the system messages in ps.
func npcSysMsgIDs(ps []testclient.Packet) []int32 {
	var ids []int32
	for _, m := range only[testclient.SystemMessage](ps) {
		ids = append(ids, m.ID)
	}
	return ids
}

func TestGMSpawn(t *testing.T) {
	e, sg, pg := npcGMEnv(t)
	gpos := e.position(pg)
	before := len(npcsSpawned(e, npcTestGremlin))

	ps := npcGM(t, e, sg, "spawn 20001 2")
	if want := fmt.Sprintf("Created Gremlin on %d", pg.ObjectID()); !slices.Equal(messages(ps), []string{want}) {
		t.Errorf("messages %q, want %q", messages(ps), want)
	}
	if n := len(only[testclient.NpcInfo](ps)); n != 2 {
		t.Errorf("the GM saw %d new NPCs, want 2", n)
	}
	all := npcsSpawned(e, npcTestGremlin)
	if len(all) != before+2 {
		t.Fatalf("%d Gremlins, want %d", len(all), before+2)
	}
	spawned := all[before:]
	for _, n := range spawned {
		if pos := npcPosition(e, n); pos != gpos {
			t.Errorf("a GM spawn at %v, want the GM's place %v", pos, gpos)
		}
	}
	// A GM spawn never respawns.
	e.onLoop(func() {
		for _, n := range spawned {
			if !n.spawn.gm || n.spawn.respawn {
				t.Errorf("GM spawn slot %+v, want one without respawn", n.spawn)
			}
		}
		e.srv.DespawnNpc(spawned[0])
	})
	e.advance(time.Hour)
	if npcInWorld(e, spawned[0]) {
		t.Error("a despawned GM spawn came back")
	}

	// By name, at the target.
	roxxy := npcSpawnedOne(t, e, npcTestRoxxy)
	npcTestClick(e, sg, roxxy, false)
	sg.take(t)
	ps = npcGM(t, e, sg, "spawn lector")
	if want := fmt.Sprintf("Created Lector on %d", roxxy.ID()); !slices.Equal(messages(ps), []string{want}) {
		t.Errorf("messages %q, want %q", messages(ps), want)
	}
	lectors := npcsSpawned(e, npcTestLector)
	if len(lectors) != 2 {
		t.Fatalf("%d Lectors, want 2", len(lectors))
	}
	rpos, lpos := npcPosition(e, roxxy), npcPosition(e, lectors[1])
	if lpos.X != rpos.X || lpos.Y != rpos.Y || lpos.Z != rpos.Z || lpos.Heading != gpos.Heading {
		t.Errorf("the Lector spawned at %v, want Roxxy's place %v facing the GM's way %d", lpos, rpos, gpos.Heading)
	}

	for _, tc := range []struct {
		line    string
		wantMsg []string
		wantIDs []int32
	}{
		{"spawn", []string{"Usage: //spawn <npcId|name> [count]"}, []int32{testclient.SysMsgText}},
		{"spawn 20001 many", []string{"Usage: //spawn <npcId|name> [count]"}, []int32{testclient.SysMsgText}},
		{"spawn 99999", nil, []int32{int32(npcSysMsgTargetNotFound)}},
		{"spawn 24000", nil, []int32{int32(npcSysMsgTargetNotFound)}}, // a door
		{"spawn Nobody At All", nil, []int32{int32(npcSysMsgTargetNotFound)}},
	} {
		ps := npcGM(t, e, sg, tc.line)
		if !slices.Equal(messages(ps), tc.wantMsg) || !slices.Equal(npcSysMsgIDs(ps), tc.wantIDs) {
			t.Errorf("//%s: messages %q, system messages %v; want %q, %v", tc.line, messages(ps), npcSysMsgIDs(ps), tc.wantMsg, tc.wantIDs)
		}
	}
}

func TestGMDeleteFixedSpawn(t *testing.T) {
	e, sg, _ := npcGMEnv(t)
	lector := npcSpawnedOne(t, e, npcTestLector)
	if ps := npcGM(t, e, sg, "delete"); !slices.Equal(messages(ps), []string{"Incorrect target."}) {
		t.Errorf("//delete without a target: %q", messages(ps))
	}

	npcTestClick(e, sg, lector, false)
	sg.take(t)
	ps := npcGM(t, e, sg, "delete")
	if want := fmt.Sprintf("Deleted Lector from %d.", lector.ID()); !slices.Equal(messages(ps), []string{want}) {
		t.Errorf("messages %q, want %q", messages(ps), want)
	}
	if n := len(only[testclient.DeleteObject](ps)); n != 1 {
		t.Errorf("the GM got %d DeleteObject, want 1", n)
	}
	// Nothing respawns, not even with //respawn.
	e.advance(2 * time.Minute)
	if ps := npcGM(t, e, sg, "respawn"); !slices.Equal(messages(ps), []string{"Respawned 0 NPCs."}) {
		t.Errorf("//respawn: %q", messages(ps))
	}
	if npcInWorld(e, lector) || len(npcsSpawned(e, npcTestLector)) != 0 {
		t.Error("the deleted Lector came back")
	}
	e.onLoop(func() {
		if len(e.srv.npc.spawns) != 10 {
			t.Errorf("%d spawn slots, want 10", len(e.srv.npc.spawns))
		}
	})
}

func TestGMDeleteTerritoryGroup(t *testing.T) {
	e, sg, _ := npcGMEnv(t)
	gremlins := npcsSpawned(e, npcTestGremlin)
	// One of them is dead, waiting for its respawn: its spawn goes too.
	e.onLoop(func() { e.srv.DespawnNpc(gremlins[4]) })
	npcTestClick(e, sg, gremlins[0], false)
	sg.take(t)
	ps := npcGM(t, e, sg, "delete")
	if want := fmt.Sprintf("Deleted Gremlin group from %d.", gremlins[0].ID()); !slices.Equal(messages(ps), []string{want}) {
		t.Errorf("messages %q, want %q", messages(ps), want)
	}
	e.advance(5 * time.Minute)
	if n := len(npcsSpawned(e, npcTestGremlin)); n != 0 {
		t.Errorf("%d Gremlins after deleting their group", n)
	}
	e.onLoop(func() {
		if len(e.srv.npc.byTerritory) != 0 || len(e.srv.npc.pending) != 0 {
			t.Errorf("territory slots %v and pending respawns %v left", e.srv.npc.byTerritory, e.srv.npc.pending)
		}
	})
}

func TestGMDeleteInRange(t *testing.T) {
	e, sg, _ := npcGMEnv(t)
	// Roxxy (100 away) and the FriendlyMob (300 away) are within 400;
	// Lector (1000 away) and the territory are not.
	ps := npcGM(t, e, sg, "delete 400")
	msgs := messages(ps)
	slices.Sort(msgs)
	if len(msgs) != 2 || !strings.HasPrefix(msgs[0], "Deleted Friendly from") || !strings.HasPrefix(msgs[1], "Deleted Roxxy from") {
		t.Errorf("messages %q, want Friendly and Roxxy deleted", msgs)
	}
	if len(npcsSpawned(e, npcTestLector)) != 1 || len(npcsSpawned(e, npcTestGremlin)) != 5 {
		t.Error("//delete 400 deleted NPCs out of range")
	}
}

func TestGMRespawn(t *testing.T) {
	e, sg, _ := npcGMEnv(t)
	roxxy := npcSpawnedOne(t, e, npcTestRoxxy)
	gremlin := npcsSpawned(e, npcTestGremlin)[0]
	e.onLoop(func() {
		e.srv.DespawnNpc(roxxy)
		e.srv.DespawnNpc(gremlin)
	})
	if ps := npcGM(t, e, sg, "respawn"); !slices.Equal(messages(ps), []string{"Respawned 2 NPCs."}) {
		t.Errorf("//respawn: %q", messages(ps))
	}
	if !npcInWorld(e, roxxy) || !npcInWorld(e, gremlin) {
		t.Fatal("//respawn did not bring both back")
	}
	e.onLoop(func() {
		if len(e.srv.npc.pending) != 0 {
			t.Errorf("pending respawns %v left", e.srv.npc.pending)
		}
	})

	// With a target, its spawn spawns again now, as the same object.
	npcTestClick(e, sg, roxxy, false)
	sg.take(t)
	ps := npcGM(t, e, sg, "respawn")
	if !slices.Equal(messages(ps), []string{"Respawned Roxxy."}) {
		t.Errorf("//respawn on Roxxy: %q", messages(ps))
	}
	checkNames(t, "the GM's packets", targetDrop(ps, "SystemMessage"), "DeleteObject", "TargetUnselected", "NpcInfo")
	if !npcInWorld(e, roxxy) {
		t.Error("Roxxy is not in the world after //respawn")
	}
}

func TestGMNpcInfo(t *testing.T) {
	e, sg, _ := npcGMEnv(t)
	if ps := npcGM(t, e, sg, "npcinfo"); !slices.Equal(messages(ps), []string{"Incorrect target."}) {
		t.Errorf("//npcinfo without a target: %q", messages(ps))
	}
	gremlin := npcsSpawned(e, npcTestGremlin)[0]
	npcTestClick(e, sg, gremlin, false)
	sg.take(t)
	pages := only[testclient.NpcHtmlMessage](npcGM(t, e, sg, "npcinfo"))
	if len(pages) != 1 {
		t.Fatalf("got %d pages, want 1", len(pages))
	}
	for _, want := range []string{"Gremlin", "20001", "Monster", "Random", npcTestTerritory, "20-40 sec", "158/158"} {
		if !strings.Contains(pages[0].HTML, want) {
			t.Errorf("the page lacks %q:\n%s", want, pages[0].HTML)
		}
	}
}

func TestGMCommandsNeedAGM(t *testing.T) {
	e := newNpcEnv(t, nil)
	sa, _, _, _ := twoPlayers(t, e)
	for _, line := range []string{"spawn 20001", "delete 1000", "respawn", "npcinfo"} {
		if ps := npcGM(t, e, sa, line); len(ps) != 0 {
			t.Errorf("a player's //%s got %q", line, testclient.Names(ps))
		}
	}
	if n := len(npcsSpawned(e, npcTestGremlin)); n != 5 {
		t.Errorf("%d Gremlins, want 5", n)
	}
}
