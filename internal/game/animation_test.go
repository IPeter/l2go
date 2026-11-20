package game

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/testclient"
)

// The NPCs of the animation test, next to the start.
const (
	animCorpse = 30300 // Folk with the CORPSE AI
	animChest  = 30301 // a treasure chest
	animWalker = 30302 // Folk that walks around
)

// animationWorld is npcTestWorld with random animations on for Roxxy, a
// corpse, a chest and a walking NPC (and off for Lector).
func animationWorld(start data.Location) data.WorldData {
	wd := npcTestWorld(start)
	for i := range wd.Npcs {
		if wd.Npcs[i].ID == npcTestRoxxy {
			wd.Npcs[i].Status.RandomAnimation = true
		}
	}
	corpse := npcTestTemplate(animCorpse, "Folk", "Corpse", 1)
	corpse.AI.Type = "CORPSE"
	chest := npcTestTemplate(animChest, "Chest", "Chest", 1)
	walker := npcTestTemplate(animWalker, "Folk", "Walker", 1)
	for _, t := range []*data.NpcTemplate{&corpse, &chest, &walker} {
		t.Status.RandomAnimation = true
		wd.Npcs = append(wd.Npcs, *t)
	}
	wd.NpcTypes = append(wd.NpcTypes, data.MobiusClass{Name: "Chest", Parent: "Monster", Usable: true})
	at := func(id, dx int) data.SpawnNpc {
		return data.SpawnNpc{NpcID: id, X: start.X + dx, Y: start.Y + 200, Z: start.Z, Count: 1, RespawnDelay: 60}
	}
	wd.Spawns = append(wd.Spawns, data.Spawn{
		File: "TalkingIsland/TestAnimation.xml",
		Npcs: []data.SpawnNpc{at(animCorpse, 0), at(animChest, 100), at(animWalker, 200)},
	})
	return wd
}

// Idle NPCs in active regions play social action 2 or 3 now and then, at
// least 6 s apart (Mobius's waits are 5 to 60 s); corpses, chests, NPCs
// whose template turns animations off and walking NPCs never do.
func TestRandomAnimations(t *testing.T) {
	start := npcTestStart(t)
	e := newAIEnv(t, animationWorld(start), newAIRand(), nil, false)
	roxxy := npcSpawnedOne(t, e, npcTestRoxxy)
	walker := npcSpawnedOne(t, e, animWalker)
	sess, _ := statusPlayer(t, e, "alice", "Alice")

	var scheduled []int
	e.onLoop(func() {
		for _, n := range e.srv.animation.pending {
			scheduled = append(scheduled, n.Template.ID)
		}
	})
	ids := map[int]bool{}
	for _, id := range scheduled {
		ids[id] = true
	}
	if len(scheduled) != 2 || !ids[npcTestRoxxy] || !ids[animWalker] {
		t.Fatalf("scheduled %v; want Roxxy (%d) and the walker (%d)", scheduled, npcTestRoxxy, animWalker)
	}

	// The walker walks all the time.
	keepWalking := func() {
		e.onLoop(func() {
			if walker.Move == nil {
				pos := walker.Position()
				pos.X += 2000
				e.srv.NpcMoveTo(walker, pos)
			}
		})
	}
	var at []time.Duration
	for elapsed := time.Second; elapsed <= 5*time.Minute; elapsed += time.Second {
		keepWalking()
		e.advance(time.Second)
		for _, s := range only[testclient.SocialAction](sess.take(t)) {
			switch s.ObjectID {
			case int32(roxxy.ID()):
				if s.ActionID != 2 && s.ActionID != 3 {
					t.Errorf("Roxxy played social action %d, want 2 or 3", s.ActionID)
				}
				at = append(at, elapsed)
			default:
				t.Errorf("NPC %d played social action %d", s.ObjectID, s.ActionID)
			}
		}
	}
	if len(at) < 3 || len(at) > 50 {
		t.Fatalf("Roxxy played %d animations in 5 minutes (at %v); want one every 5-60 s", len(at), at)
	}
	// A wait of 5 s ends 6 s later (Mobius plays when the time has passed),
	// which the 6 s rule refuses; so a gap is more than 6 s and at most 6 s
	// plus the longest wait.
	for i := 1; i < len(at); i++ {
		if gap := at[i] - at[i-1]; gap <= targetSocialInterval || gap > 67*time.Second {
			t.Errorf("animations %v apart (at %v); want more than 6 s and at most 67 s", gap, at)
		}
	}
}

// A region that turns off drops its NPCs from the animations; turning on
// again schedules them anew.
func TestRandomAnimationsFollowRegions(t *testing.T) {
	start := npcTestStart(t)
	e := newAIEnv(t, animationWorld(start), newAIRand(), nil, false)
	roxxy := npcSpawnedOne(t, e, npcTestRoxxy)
	scheduled := func() bool {
		var ok bool
		e.onLoop(func() { _, ok = e.srv.animation.due[roxxy] })
		return ok
	}
	if scheduled() {
		t.Fatal("Roxxy is scheduled before any player came")
	}
	sess, _ := statusPlayer(t, e, "alice", "Alice")
	if !scheduled() {
		t.Fatal("Roxxy is not scheduled once Alice is there")
	}
	e.srv.Logout(sess)
	e.sync()
	e.advance(2 * time.Minute)
	if scheduled() {
		t.Error("Roxxy is still scheduled after her region turned off")
	}
}
