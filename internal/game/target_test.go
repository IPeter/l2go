package game

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// targetDrop returns ps without the packets of the given names, for checks
// that other features' packets (dialog pages) must not disturb.
func targetDrop(ps []testclient.Packet, names ...string) []testclient.Packet {
	return slices.DeleteFunc(slices.Clone(ps), func(p testclient.Packet) bool { return slices.Contains(names, p.PacketName()) })
}

// targetCancelBody encodes RequestTargetCanceld.
func targetCancelBody(unselect uint16) []byte {
	w := packets.NewWriter(2)
	w.WriteUint16(unselect)
	return w.Bytes()
}

func TestTargetNpc(t *testing.T) {
	e := newNpcEnv(t, nil)
	sa, sb, pa, _ := twoPlayers(t, e)
	roxxy := npcSpawnedOne(t, e, npcTestRoxxy)
	rpos := npcPosition(e, roxxy)

	npcTestClick(e, sa, roxxy, false)
	ps := sa.take(t)
	checkNames(t, "Alice targets Roxxy", ps, "ValidateLocation", "MyTargetSelected", "StatusUpdate", "ActionFailed")
	if v := ps[0].(testclient.ValidateLocation); v.ObjectID != int32(roxxy.ID()) || v.Position() != (testclient.Position{X: rpos.X, Y: rpos.Y, Z: rpos.Z}) || v.Heading != rpos.Heading {
		t.Errorf("ValidateLocation %+v, want Roxxy at %v", v, rpos)
	}
	if m := ps[1].(testclient.MyTargetSelected); m.ObjectID != int32(roxxy.ID()) || m.Color != 0 {
		t.Errorf("MyTargetSelected %+v, want Roxxy with colour 0 (not attackable)", m)
	}
	su := ps[2].(testclient.StatusUpdate)
	if want := []testclient.StatusAttribute{{ID: testclient.StatusMaxHP, Value: 158}, {ID: testclient.StatusCurHP, Value: 158}}; su.ObjectID != int32(roxxy.ID()) || !slices.Equal(su.Attributes, want) {
		t.Errorf("StatusUpdate %+v, want Roxxy's HP bar %v", su, want)
	}
	apos := e.position(pa)
	pbs := sb.take(t)
	checkNames(t, "Bob sees Alice target", pbs, "TargetSelected")
	if ts := pbs[0].(testclient.TargetSelected); ts.ObjectID != pa.ObjectID() || ts.TargetID != int32(roxxy.ID()) || ts.X != apos.X || ts.Y != apos.Y || ts.Z != apos.Z {
		t.Errorf("TargetSelected %+v, want Alice at %v targeting Roxxy", ts, apos)
	}
	e.onLoop(func() {
		if pa.Target != world.Object(roxxy) || pa.LastNpc != roxxy.ID() {
			t.Errorf("Alice's target %v, last NPC %v; want Roxxy", pa.Target, pa.LastNpc)
		}
	})
}

func TestTargetMonsterColour(t *testing.T) {
	e := newNpcEnv(t, nil)
	sa, _, _, _ := twoPlayers(t, e)
	gremlin := npcsSpawned(e, npcTestGremlin)[0]
	npcTestClick(e, sa, gremlin, false)
	ms := only[testclient.MyTargetSelected](sa.take(t))
	// Alice (level 1) can attack the level 5 Gremlin: 1 - 5.
	if len(ms) != 1 || ms[0].Color != -4 {
		t.Errorf("MyTargetSelected %+v, want colour -4", ms)
	}
}

func TestSecondClickTalksInRange(t *testing.T) {
	var talks []npcTalk
	e := newNpcEnv(t, nil, npcRecordTalks(&talks))
	sa, sb, pa, _ := twoPlayers(t, e)
	roxxy := npcSpawnedOne(t, e, npcTestRoxxy)
	npcTestClick(e, sa, roxxy, false)
	sa.take(t)
	sb.take(t)

	npcTestClick(e, sa, roxxy, false)
	ps := sa.take(t)
	checkNames(t, "Alice talks to Roxxy", ps, "MoveToPawn", "ActionFailed")
	apos, rpos := e.position(pa), npcPosition(e, roxxy)
	want := testclient.MoveToPawn{
		ObjectID: pa.ObjectID(), TargetID: int32(roxxy.ID()), Distance: targetTalkTurnDistance,
		X: apos.X, Y: apos.Y, Z: apos.Z, TargetX: rpos.X, TargetY: rpos.Y, TargetZ: rpos.Z,
	}
	if got := ps[0].(testclient.MoveToPawn); got != want {
		t.Errorf("MoveToPawn %+v, want %+v", got, want)
	}
	if got := npcTalks(e, &talks); !slices.Equal(got, []npcTalk{{"Alice", npcTestRoxxy}}) {
		t.Errorf("interaction hooks ran %v, want once for Alice and Roxxy", got)
	}
	// Turning to the NPC is Alice's own business.
	if ps := sb.take(t); len(ps) != 0 {
		t.Errorf("Bob got %q", testclient.Names(ps))
	}

	// Sitting, Alice cannot talk.
	e.onLoop(func() { pa.Sitting = true })
	npcTestClick(e, sa, roxxy, false)
	checkNames(t, "Alice clicks Roxxy while sitting", sa.take(t), "ActionFailed")
	if got := npcTalks(e, &talks); len(got) != 1 {
		t.Errorf("interaction hooks ran %d times, want once", len(got))
	}
}

func TestTalkPlaysARandomAnimation(t *testing.T) {
	e := newNpcEnv(t, nil)
	sa, sb, _, _ := twoPlayers(t, e)
	roxxy := npcSpawnedOne(t, e, npcTestRoxxy)
	e.onLoop(func() { roxxy.randomAnimation = true })
	npcTestClick(e, sa, roxxy, false)
	sa.take(t)
	sb.take(t)

	npcTestClick(e, sa, roxxy, false)
	social := only[testclient.SocialAction](sa.take(t))
	if len(social) != 1 || social[0].ObjectID != int32(roxxy.ID()) || social[0].ActionID < 0 || social[0].ActionID >= targetTalkAnimations {
		t.Errorf("Alice got %+v, want one of Roxxy's 8 animations", social)
	}
	if n := len(only[testclient.SocialAction](sb.take(t))); n != 1 {
		t.Errorf("Bob got %d animations, want 1", n)
	}
	// At most one every 6 seconds.
	npcTestClick(e, sa, roxxy, false)
	if n := len(only[testclient.SocialAction](sa.take(t))); n != 0 {
		t.Errorf("a second talk at once played %d animations", n)
	}
	e.advance(targetSocialInterval + time.Millisecond)
	sa.take(t)
	npcTestClick(e, sa, roxxy, false)
	if n := len(only[testclient.SocialAction](sa.take(t))); n != 1 {
		t.Errorf("a talk 6 seconds later played %d animations, want 1", n)
	}
}

func TestSecondClickOnMonsterDoesNotTalk(t *testing.T) {
	var talks []npcTalk
	e := newNpcEnv(t, nil, npcRecordTalks(&talks))
	sa, _, _, _ := twoPlayers(t, e)
	gremlin := npcsSpawned(e, npcTestGremlin)[0]
	npcTestClick(e, sa, gremlin, false)
	sa.take(t)
	npcTestClick(e, sa, gremlin, false)
	checkNames(t, "Alice clicks her Gremlin target", sa.take(t), "ActionFailed")
	if got := npcTalks(e, &talks); len(got) != 0 {
		t.Errorf("interaction hooks ran %v for a monster", got)
	}
}

func TestAttackRequest(t *testing.T) {
	e := newNpcEnv(t, nil)
	sa, _, pa, _ := twoPlayers(t, e)
	gremlin := npcsSpawned(e, npcTestGremlin)[0]
	attack := func() {
		e.srv.Handle(sa, clientpackets.OpAttackRequest, npcActionBody(int32(gremlin.ID()), false))
		e.sync()
	}
	// On an object that is not the target, Mobius clicks it.
	attack()
	checkNames(t, "AttackRequest on a new object", sa.take(t), "ValidateLocation", "MyTargetSelected", "StatusUpdate", "ActionFailed")
	// On the target it attacks, which comes with combat.
	attack()
	checkNames(t, "AttackRequest on the target", sa.take(t), "ActionFailed")
	e.onLoop(func() {
		if pa.Target != world.Object(gremlin) {
			t.Errorf("Alice's target %v, want the Gremlin", pa.Target)
		}
	})
	// An unknown object.
	e.srv.Handle(sa, clientpackets.OpAttackRequest, npcActionBody(12345, false))
	e.sync()
	checkNames(t, "AttackRequest on nothing", sa.take(t), "ActionFailed")
}

func TestTargetCancel(t *testing.T) {
	e := newNpcEnv(t, nil)
	sa, sb, pa, _ := twoPlayers(t, e)
	roxxy := npcSpawnedOne(t, e, npcTestRoxxy)
	npcTestClick(e, sa, roxxy, false)
	sa.take(t)
	sb.take(t)

	e.srv.Handle(sa, clientpackets.OpRequestTargetCanceld, targetCancelBody(0))
	e.sync()
	apos := e.position(pa)
	want := testclient.TargetUnselected{ObjectID: pa.ObjectID(), X: apos.X, Y: apos.Y, Z: apos.Z}
	for who, sess := range map[string]*fakeSession{"Alice": sa, "Bob": sb} {
		ps := sess.take(t)
		checkNames(t, who+" after the cancel", ps, "TargetUnselected")
		if got := ps[0].(testclient.TargetUnselected); got != want {
			t.Errorf("%s got %+v, want %+v", who, got, want)
		}
	}
	e.onLoop(func() {
		if pa.Target != nil || len(e.srv.npc.watchers) != 0 {
			t.Errorf("Alice's target %v, HP bar watchers %v; want none", pa.Target, e.srv.npc.watchers)
		}
	})
	// Without a target nothing happens.
	e.srv.Handle(sa, clientpackets.OpRequestTargetCanceld, targetCancelBody(1))
	e.sync()
	if ps := sa.take(t); len(ps) != 0 {
		t.Errorf("a cancel without a target sent %q", testclient.Names(ps))
	}
}

func TestTargetDroppedOutOfSight(t *testing.T) {
	e := newNpcEnv(t, nil)
	sa, _, pa, _ := twoPlayers(t, e)
	roxxy := npcSpawnedOne(t, e, npcTestRoxxy)
	npcTestClick(e, sa, roxxy, false)
	sa.take(t)

	e.onLoop(func() {
		far := pa.Position()
		far.X += 20 * world.RegionSize
		if err := e.srv.MoveObject(pa, far); err != nil {
			t.Errorf("MoveObject: %v", err)
		}
	})
	ps := sa.take(t)
	if n := len(only[testclient.TargetUnselected](ps)); n != 1 {
		t.Errorf("Alice got %d TargetUnselected walking away, want 1 (%q)", n, testclient.Names(ps))
	}
	e.onLoop(func() {
		if pa.Target != nil {
			t.Errorf("Alice still targets %v", pa.Target)
		}
	})
}

func TestTargetDroppedWhenTheNpcLeaves(t *testing.T) {
	e := newNpcEnv(t, nil)
	sa, _, pa, _ := twoPlayers(t, e)
	lector := npcSpawnedOne(t, e, npcTestLector)
	npcTestClick(e, sa, lector, false)
	sa.take(t)

	e.onLoop(func() { e.srv.DespawnNpc(lector) })
	ps := sa.take(t)
	checkNames(t, "Alice when her target leaves", ps, "DeleteObject", "TargetUnselected")
	if d := ps[0].(testclient.DeleteObject); d.ObjectID != int32(lector.ID()) {
		t.Errorf("DeleteObject %+v, want Lector", d)
	}
	e.onLoop(func() {
		if pa.Target != nil {
			t.Errorf("Alice still targets %v", pa.Target)
		}
	})
}

func TestTargetPlayer(t *testing.T) {
	e := newNpcEnv(t, nil)
	sa, sb, pa, pb := twoPlayers(t, e)
	npcTestClick(e, sa, pb, false)
	ps := sa.take(t)
	checkNames(t, "Alice targets Bob", ps, "ValidateLocation", "MyTargetSelected", "StatusUpdate", "ActionFailed")
	if m := ps[1].(testclient.MyTargetSelected); m.ObjectID != pb.ObjectID() || m.Color != 0 {
		t.Errorf("MyTargetSelected %+v, want Bob, colour 0", m)
	}
	pbs := sb.take(t)
	checkNames(t, "Bob sees Alice target him", pbs, "TargetSelected")
	if ts := pbs[0].(testclient.TargetSelected); ts.ObjectID != pa.ObjectID() || ts.TargetID != pb.ObjectID() {
		t.Errorf("TargetSelected %+v, want Alice targeting Bob", ts)
	}
	// A second click follows or attacks, which come later.
	npcTestClick(e, sa, pb, false)
	checkNames(t, "Alice clicks Bob again", sa.take(t), "ActionFailed")

	// Bob leaves: Alice drops him.
	e.srv.Logout(sb)
	if n := len(only[testclient.TargetUnselected](sa.take(t))); n != 1 {
		t.Errorf("Alice got %d TargetUnselected when Bob left, want 1", n)
	}
}

func TestShiftClick(t *testing.T) {
	var talks []npcTalk
	e := newNpcEnv(t, nil, npcRecordTalks(&talks))
	e.createCharacter("gm", "Gm", classHumanFighter, storage.AccessLevelAdmin)
	e.createCharacter("alice", "Alice", classHumanFighter, storage.AccessLevelPlayer)
	sg, sa := newSession("gm"), newSession("alice")
	pg := e.enter(sg, 0)
	e.enter(sa, 0)
	sg.take(t)
	sa.take(t)
	roxxy := npcSpawnedOne(t, e, npcTestRoxxy)

	// A GM gets the info page with the target.
	npcTestClick(e, sg, roxxy, true)
	ps := sg.take(t)
	checkNames(t, "the GM's Shift-click", ps, "ValidateLocation", "MyTargetSelected", "StatusUpdate", "NpcHtmlMessage", "ActionFailed")
	page := ps[3].(testclient.NpcHtmlMessage)
	for _, want := range []string{"Roxxy", "30006", "Teleporter", "Fixed", "TalkingIsland/TestNPCs.xml", "60 sec", "admin_delete"} {
		if !strings.Contains(page.HTML, want) {
			t.Errorf("the info page lacks %q:\n%s", want, page.HTML)
		}
	}
	if page.ObjectID != 0 {
		t.Errorf("the info page belongs to object %d, want none", page.ObjectID)
	}
	e.onLoop(func() {
		if _, ok := pg.HTML.Validate("admin_delete"); !ok {
			t.Error("the page's admin_delete button is not in the GM's link cache")
		}
	})
	if ts := only[testclient.TargetSelected](sa.take(t)); len(ts) != 1 || ts[0].ObjectID != pg.ObjectID() {
		t.Errorf("Alice got %+v, want the GM's TargetSelected", ts)
	}

	// Another player's Shift-click is a plain click that never talks.
	npcTestClick(e, sa, roxxy, true)
	checkNames(t, "a player's Shift-click", sa.take(t), "ValidateLocation", "MyTargetSelected", "StatusUpdate", "ActionFailed")
	npcTestClick(e, sa, roxxy, true)
	checkNames(t, "a player's Shift-click on the target", sa.take(t), "ActionFailed")
	if got := npcTalks(e, &talks); len(got) != 0 {
		t.Errorf("interaction hooks ran %v", got)
	}
}

func TestClickChecks(t *testing.T) {
	e := newNpcEnv(t, nil)
	sa, _, _, _ := twoPlayers(t, e)
	roxxy := npcSpawnedOne(t, e, npcTestRoxxy)

	// Nothing with that ID.
	e.srv.Handle(sa, clientpackets.OpAction, npcActionBody(0x7000_0000, false))
	e.sync()
	checkNames(t, "a click on nothing", sa.take(t), "ActionFailed")

	// A bad action ID.
	body := npcActionBody(int32(roxxy.ID()), false)
	body[len(body)-1] = 7
	e.srv.Handle(sa, clientpackets.OpAction, body)
	e.sync()
	checkNames(t, "a click with action 7", sa.take(t), "ActionFailed")

	// A template that cannot be targeted.
	e.onLoop(func() {
		tmpl := *roxxy.Template
		tmpl.Status.Targetable = false
		roxxy.Template = &tmpl
	})
	npcTestClick(e, sa, roxxy, false)
	checkNames(t, "a click on an untargetable NPC", sa.take(t), "ActionFailed")
}

// TestWalkUpToNpcThenTalk needs the movement feature's MovePlayerTo.
func TestWalkUpToNpcThenTalk(t *testing.T) {
	var talks []npcTalk
	e := newNpcEnv(t, nil, npcRecordTalks(&talks))
	sa, sb, pa, _ := twoPlayers(t, e)
	lector := npcSpawnedOne(t, e, npcTestLector)
	npcTestClick(e, sa, lector, false)
	sa.take(t)
	sb.take(t)

	// 1000 units away: the second click walks Alice up to Lector.
	npcTestClick(e, sa, lector, false)
	pawns := only[testclient.MoveToPawn](sa.take(t))
	if len(pawns) != 1 || pawns[0].ObjectID != pa.ObjectID() || pawns[0].TargetID != int32(lector.ID()) || pawns[0].Distance != targetApproachOffset {
		t.Fatalf("Alice got %+v, want MoveToPawn to Lector at distance %d", pawns, targetApproachOffset)
	}
	if n := len(only[testclient.MoveToPawn](sb.take(t))); n != 1 {
		t.Errorf("Bob got %d MoveToPawn, want 1", n)
	}
	if got := npcTalks(e, &talks); len(got) != 0 {
		t.Fatalf("talked %v before walking up", got)
	}
	for range 200 {
		e.advance(100 * time.Millisecond)
		if len(npcTalks(e, &talks)) > 0 {
			break
		}
	}
	if got := npcTalks(e, &talks); !slices.Equal(got, []npcTalk{{"Alice", npcTestLector}}) {
		t.Fatalf("interaction hooks ran %v, want once for Alice and Lector after the walk", got)
	}
	if d := e.position(pa).Distance3D(npcPosition(e, lector)); d > float64(e.srv.npcInteractionRange()) {
		t.Errorf("Alice talked from %.0f away", d)
	}
}

func TestWalkUpToNpcGivesUpWithTheTarget(t *testing.T) {
	var talks []npcTalk
	e := newNpcEnv(t, nil, npcRecordTalks(&talks))
	sa, _, _, _ := twoPlayers(t, e)
	lector := npcSpawnedOne(t, e, npcTestLector)
	npcTestClick(e, sa, lector, false)
	npcTestClick(e, sa, lector, false)
	e.srv.Handle(sa, clientpackets.OpRequestTargetCanceld, targetCancelBody(1))
	e.sync()
	e.advance(20 * time.Second)
	if got := npcTalks(e, &talks); len(got) != 0 {
		t.Errorf("talked %v after dropping the target", got)
	}
	e.onLoop(func() {
		if len(e.srv.npc.approaches) != 0 {
			t.Errorf("walks up to NPCs left: %v", e.srv.npc.approaches)
		}
	})
}

func TestTargetDroppedOnTeleport(t *testing.T) {
	e := newNpcEnv(t, nil)
	sa, sb, pa, _ := twoPlayers(t, e)
	roxxy := npcSpawnedOne(t, e, npcTestRoxxy)
	npcTestClick(e, sa, roxxy, false)
	sa.take(t)
	sb.take(t)
	e.onLoop(func() {
		dest := pa.Position()
		dest.X += 30 * world.RegionSize
		if err := e.srv.Teleport(pa, dest); err != nil {
			t.Errorf("Teleport: %v", err)
		}
	})
	// Mobius drops the target before the player leaves its place.
	if n := len(only[testclient.TargetUnselected](sb.take(t))); n != 1 {
		t.Errorf("Bob got %d TargetUnselected, want 1", n)
	}
	e.onLoop(func() {
		if pa.Target != nil {
			t.Errorf("Alice still targets %v after teleporting", pa.Target)
		}
	})
}
