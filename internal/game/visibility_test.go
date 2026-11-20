package game

import (
	"testing"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// twoPlayers enters Alice and Bob at the same creation point and drops the
// packets sent so far.
func twoPlayers(t *testing.T, e *testEnv) (sa, sb *fakeSession, pa, pb *Player) {
	t.Helper()
	e.createCharacter("alice", "Alice", classHumanFighter, storage.AccessLevelPlayer)
	e.createCharacter("bob", "Bob", classHumanFighter, storage.AccessLevelPlayer)
	sa, sb = newSession("alice"), newSession("bob")
	pa = e.enter(sa, 0)
	pb = e.enter(sb, 0)
	sa.take(t)
	sb.take(t)
	return sa, sb, pa, pb
}

func TestPlayersSeeEachOtherOnEnter(t *testing.T) {
	e := newEnv(t, nil)
	a := e.createCharacter("alice", "Alice", classHumanFighter, storage.AccessLevelPlayer)
	b := e.createCharacter("bob", "Bob", classHumanFighter, storage.AccessLevelPlayer)
	sa, sb := newSession("alice"), newSession("bob")
	e.enter(sa, 0)
	sa.take(t)

	e.enter(sb, 0)
	// Bob gets Alice's info in the spawn step, after QuestList.
	pb := sb.take(t)
	checkNames(t, "Bob's EnterWorld next to Alice", pb,
		"CharSelected", "UserInfo", "SendMacroList", "ItemList", "ShortcutInit", "HennaInfo", "QuestList",
		"CharInfo", "RelationChanged",
		"EtcStatusUpdate", "ExStorageMaxCount", "FriendList", "SystemMessage", "SkillCoolTime",
		"UserInfo", "ValidateLocation", "ActionFailed")
	ci := pb[7].(testclient.CharInfoPacket)
	if ci.ObjectID != a.ObjectID || ci.Name != "Alice" || ci.Position() != (testclient.Position{X: a.X, Y: a.Y, Z: a.Z}) {
		t.Errorf("Bob got CharInfo of %d %q at %v, want Alice", ci.ObjectID, ci.Name, ci.Position())
	}
	if !ci.Standing || !ci.Running || ci.Invisible || ci.RunSpd != 115 {
		t.Errorf("Alice's CharInfo: standing %v running %v invisible %v run speed %d", ci.Standing, ci.Running, ci.Invisible, ci.RunSpd)
	}
	if rel := pb[8].(testclient.RelationChanged); rel.ObjectID != a.ObjectID || rel.Relation != 0 || rel.AutoAttackable {
		t.Errorf("Bob got %+v, want Alice's relation 0, not auto-attackable", rel)
	}

	// Alice gets Bob's info when he spawns, and his CharInfo again from his
	// broadcastUserInfo; the relation is sent once.
	pa := sa.take(t)
	checkNames(t, "Alice while Bob enters", pa, "CharInfo", "RelationChanged", "CharInfo")
	for _, ci := range only[testclient.CharInfoPacket](pa) {
		if ci.ObjectID != b.ObjectID {
			t.Errorf("Alice got CharInfo of %d, want Bob (%d)", ci.ObjectID, b.ObjectID)
		}
	}

	// Bob leaves: Alice sees him stop and forgets him.
	e.srv.Logout(sb)
	pa = sa.take(t)
	checkNames(t, "Alice after Bob's logout", pa, "StopMove", "DeleteObject")
	if stop := pa[0].(testclient.StopMove); stop.ObjectID != b.ObjectID {
		t.Errorf("Alice got StopMove %d, want Bob (%d)", stop.ObjectID, b.ObjectID)
	}
	if del := pa[1].(testclient.DeleteObject); del.ObjectID != b.ObjectID {
		t.Errorf("Alice got DeleteObject %d, want Bob (%d)", del.ObjectID, b.ObjectID)
	}
}

func TestMovingOutOfSightAndBack(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, pa, pb := twoPlayers(t, e)
	home := e.position(pb)
	far := home
	far.X += 20 * world.RegionSize

	e.onLoop(func() {
		if err := e.srv.MoveObject(pb, far); err != nil {
			t.Errorf("MoveObject: %v", err)
		}
	})
	pas, pbs := sa.take(t), sb.take(t)
	checkNames(t, "Alice when Bob walks away", pas, "DeleteObject")
	checkNames(t, "Bob when he walks away", pbs, "DeleteObject")
	if pas[0].(testclient.DeleteObject).ObjectID != pb.ObjectID() || pbs[0].(testclient.DeleteObject).ObjectID != pa.ObjectID() {
		t.Errorf("wrong DeleteObjects: Alice %+v, Bob %+v", pas[0], pbs[0])
	}

	// Coming back, both get each other's info again, the relation included:
	// it was forgotten when they lost sight.
	e.onLoop(func() {
		if err := e.srv.MoveObject(pb, home); err != nil {
			t.Errorf("MoveObject: %v", err)
		}
	})
	checkNames(t, "Alice when Bob comes back", sa.take(t), "CharInfo", "RelationChanged")
	checkNames(t, "Bob when he comes back", sb.take(t), "CharInfo", "RelationChanged")
}

func TestNewViewerSeesAWalkingPlayer(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, _, pb := twoPlayers(t, e)
	home := e.position(pb)
	far := home
	far.X += 20 * world.RegionSize
	dest := world.Position{X: home.X + 300, Y: home.Y, Z: home.Z}

	e.onLoop(func() {
		if err := e.srv.MoveObject(pb, far); err != nil {
			t.Errorf("MoveObject: %v", err)
		}
	})
	sa.take(t)
	sb.take(t)
	e.onLoop(func() {
		pb.Move = &MoveState{Origin: home, Destination: dest, Start: e.srv.Now(), Speed: 126, Running: true}
		if err := e.srv.MoveObject(pb, home); err != nil {
			t.Errorf("MoveObject: %v", err)
		}
	})
	pas := sa.take(t)
	checkNames(t, "Alice when a walking Bob comes into sight", pas, "CharInfo", "RelationChanged", "MoveToLocation")
	if m := pas[2].(testclient.MoveToLocation); m.ObjectID != pb.ObjectID() || m.Destination() != (testclient.Position{X: dest.X, Y: dest.Y, Z: dest.Z}) {
		t.Errorf("MoveToLocation %+v, want Bob walking to %v", m, dest)
	}
}

func TestRefreshStatsCoalescesInfo(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, pa, _ := twoPlayers(t, e)

	// Nothing changed: nothing is sent.
	e.onLoop(func() { pa.RefreshStats() })
	e.advance(e.srv.Settings().TickInterval)
	if ps := sa.take(t); len(ps) != 0 {
		// The enter-world timers of 300 ms have not run yet.
		t.Errorf("RefreshStats without a change sent %q", testclient.Names(ps))
	}
	if ps := sb.take(t); len(ps) != 0 {
		t.Errorf("RefreshStats without a change sent %q to a viewer", testclient.Names(ps))
	}

	// A change sends one UserInfo and one CharInfo at the end of the tick,
	// however often it is reported.
	e.onLoop(func() {
		pa.Stats().SetGMSpeed(2)
		pa.RefreshStats()
		pa.RefreshStats()
		pa.BroadcastUserInfo()
	})
	if ps := sa.take(t); len(ps) != 0 {
		t.Errorf("sent %q before the end of the tick", testclient.Names(ps))
	}
	e.advance(e.srv.Settings().TickInterval)
	checkNames(t, "Alice after a stat change", sa.take(t), "UserInfo")
	checkNames(t, "Bob after Alice's stat change", sb.take(t), "CharInfo")
}
