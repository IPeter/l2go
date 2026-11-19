package game

import (
	"fmt"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// gmPair enters a GM, "Gm", and a player, "Player", at the same spot, and
// drops their packets.
func gmPair(t *testing.T, e *testEnv) (sg, sp *fakeSession, gm, pl *Player) {
	t.Helper()
	e.createCharacter("gm", "Gm", classHumanFighter, storage.AccessLevelAdmin)
	e.createCharacter("player", "Player", classHumanFighter, storage.AccessLevelPlayer)
	sg, sp = newSession("gm"), newSession("player")
	gm, pl = e.enter(sg, 0), e.enter(sp, 0)
	walkSettle(t, e, sg, sp)
	return sg, sp, gm, pl
}

// gmRun sends a "//" command line (without the slashes) for a session and
// waits for the loop.
func gmRun(e *testEnv, sess *fakeSession, line string) {
	e.srv.Handle(sess, clientpackets.OpSendBypassBuildCmd, stringBody(line))
	e.sync()
}

// gmSaid returns the free-text system messages of ps.
func gmSaid(ps []testclient.Packet) []string { return messages(ps) }

func TestGMCommandsAreRegistered(t *testing.T) {
	e := newEnv(t, nil)
	names := e.srv.GMCommands()
	for _, want := range []string{"teleport", "move_to", "loc", "gmspeed", "heal", "invis", "invisible", "vis", "visible", "online"} {
		if !slices.Contains(names, want) {
			t.Errorf("GM command %q is missing from %q", want, names)
		}
	}
}

// //teleport x y z (Mobius //move_to) teleports the GM there, then tells it
// so; bad coordinates get Mobius's answers.
func TestGMTeleport(t *testing.T) {
	e := newEnv(t, nil)
	sg, sp, gm, _ := gmPair(t, e)

	gmRun(e, sg, "teleport -12672 122776 -3116")
	ps := sg.take(t)
	checkNames(t, "the GM after //teleport", ps, "StopMove", "DeleteObject", "TeleportToLocation", "SystemMessage")
	if tp := ps[2].(testclient.TeleportToLocation); tp.ObjectID != gm.ObjectID() ||
		tp.Position() != (testclient.Position{X: -12672, Y: 122776, Z: -3116 + teleportZOffset}) {
		t.Errorf("TeleportToLocation %+v", tp)
	}
	if msgs := gmSaid(ps); !slices.Equal(msgs, []string{"You have been teleported to -12672 122776 -3116"}) {
		t.Errorf("messages %q", msgs)
	}
	checkNames(t, "the player left behind", sp.take(t), "StopMove", "DeleteObject", "TeleportToLocation")
	e.srv.Handle(sg, clientpackets.OpAppearing, nil)
	e.sync()
	sg.take(t)

	for _, tc := range []struct{ line, msg string }{
		{"teleport 1 2", "Wrong or no Coordinates given."},
		{"teleport a b c", "Usage: //teleport <x> <y> <z>"},
		{"move_to 1 2 x", "Usage: //move_to <x> <y> <z>"},
		{"teleport 999999 0 0", "Those coordinates are outside the world."},
	} {
		gmRun(e, sg, tc.line)
		if msgs := gmSaid(sg.take(t)); !slices.Equal(msgs, []string{tc.msg}) {
			t.Errorf("%q: messages %q, want %q", tc.line, msgs, tc.msg)
		}
	}

	gmRun(e, sg, "move_to -12000 122776 -3116")
	ps = sg.take(t)
	checkNames(t, "the GM after //move_to", ps, "StopMove", "TeleportToLocation", "SystemMessage")
	if tp := ps[1].(testclient.TeleportToLocation); tp.X != -12000 {
		t.Errorf("//move_to went to %v", tp.Position())
	}
}

// //loc tells the GM its position and heading.
func TestGMLoc(t *testing.T) {
	e := newEnv(t, nil)
	sg, _, gm, _ := gmPair(t, e)
	pos := e.position(gm)
	gmRun(e, sg, "loc")
	want := fmt.Sprintf("Current location: %d, %d, %d (heading %d)", pos.X, pos.Y, pos.Z, pos.Heading)
	if msgs := gmSaid(sg.take(t)); !slices.Equal(msgs, []string{want}) {
		t.Errorf("messages %q, want %q", msgs, want)
	}
}

// //gmspeed n multiplies the speed of the GM, or of its target, by 0 to 10
// (0 is normal again); the new speed reaches UserInfo and CharInfo. The
// answer formats the factor as Java does.
func TestGMSpeed(t *testing.T) {
	e := newEnv(t, nil)
	sg, sp, gm, pl := gmPair(t, e)
	speedOf := func(p *Player) float64 {
		var v float64
		e.onLoop(func() { v = p.Stats().MoveSpeed() })
		return v
	}

	gmRun(e, sg, "gmspeed 2")
	if msgs := gmSaid(sg.take(t)); !slices.Equal(msgs, []string{"[Gm] speed is [200.00]% fast."}) {
		t.Errorf("messages %q", msgs)
	}
	// 115 × 2 × DEX bonus 1.1; a GM has no run speed cap.
	if v := speedOf(gm); math.Abs(v-253) > 1e-9 {
		t.Errorf("GM speed %v, want 253", v)
	}
	e.advance(world.GameTick)
	ps := sg.take(t)
	checkNames(t, "the GM at the end of the tick", ps, "UserInfo")
	if ui := ps[0].(testclient.UserInfo); math.Abs(ui.MoveMultiplier-2.2) > 1e-9 || ui.RunSpd != 115 {
		t.Errorf("UserInfo multiplier %v run speed %d, want 2.2 and 115", ui.MoveMultiplier, ui.RunSpd)
	}
	checkNames(t, "the player", sp.take(t), "CharInfo")

	for _, line := range []string{"gmspeed", "gmspeed 11", "gmspeed -1", "gmspeed fast", "gmspeed NaN"} {
		gmRun(e, sg, line)
		if msgs := gmSaid(sg.take(t)); !slices.Equal(msgs, []string{"//gmspeed [0...10]"}) {
			t.Errorf("%q: messages %q", line, msgs)
		}
	}

	gmRun(e, sg, "gmspeed 0")
	if msgs := gmSaid(sg.take(t)); !slices.Equal(msgs, []string{"[Gm] speed is [0.00]% fast."}) {
		t.Errorf("messages %q", msgs)
	}
	if v := speedOf(gm); math.Abs(v-126.5) > 1e-9 {
		t.Errorf("GM speed after //gmspeed 0 %v, want 126.5", v)
	}

	// The target's speed.
	e.onLoop(func() { gm.Target = pl })
	gmRun(e, sg, "gmspeed 1.5")
	if msgs := gmSaid(sg.take(t)); !slices.Equal(msgs, []string{"[Player] speed is [150.00]% fast."}) {
		t.Errorf("messages %q", msgs)
	}
	if v := speedOf(pl); math.Abs(v-189.75) > 1e-9 {
		t.Errorf("player speed %v, want 115 × 1.5 × 1.1", v)
	}
	if v := speedOf(gm); math.Abs(v-126.5) > 1e-9 {
		t.Errorf("GM speed %v changed with its target's", v)
	}
}

// //heal fills HP, MP and CP: of the target, or the GM; of a player by name;
// of everyone the GM sees with a radius (Mobius leaves the GM out). A target
// that is no creature is invalid.
func TestGMHeal(t *testing.T) {
	e := newEnv(t, nil)
	sg, sp, gm, pl := gmPair(t, e)
	hurt := func(p *Player) { e.onLoop(func() { p.CurHP, p.CurMP, p.CurCP = 10, 5, 0 }) }
	healed := func(what string, ps []testclient.Packet) {
		t.Helper()
		checkNames(t, what, ps, "UserInfo", "StatusUpdate")
		ui := ps[0].(testclient.UserInfo)
		if ui.HP != 126 || ui.MP != 38 || ui.CP != 50 {
			t.Errorf("%s: UserInfo HP %d MP %d CP %d, want 126, 38, 50", what, ui.HP, ui.MP, ui.CP)
		}
		su := ps[1].(testclient.StatusUpdate)
		want := []testclient.StatusAttribute{
			{ID: testclient.StatusMaxHP, Value: 126}, {ID: testclient.StatusCurHP, Value: 126},
			{ID: testclient.StatusMaxMP, Value: 38}, {ID: testclient.StatusCurMP, Value: 38},
			{ID: testclient.StatusMaxCP, Value: 50}, {ID: testclient.StatusCurCP, Value: 50},
		}
		if !slices.Equal(su.Attributes, want) {
			t.Errorf("%s: StatusUpdate %+v, want %+v", what, su.Attributes, want)
		}
	}

	hurt(gm)
	gmRun(e, sg, "heal")
	healed("the GM after //heal", sg.take(t))
	checkNames(t, "the player", sp.take(t))

	hurt(pl)
	gmRun(e, sg, "heal PLAYER")
	healed("the player after //heal Player", sp.take(t))
	checkNames(t, "the GM", sg.take(t))

	hurt(gm)
	hurt(pl)
	gmRun(e, sg, "heal 300")
	if msgs := gmSaid(sg.take(t)); !slices.Equal(msgs, []string{"Healed within 300 unit radius."}) {
		t.Errorf("messages %q", msgs)
	}
	healed("the player after //heal 300", sp.take(t))
	var hp float64
	e.onLoop(func() { hp = gm.CurHP })
	if hp != 10 {
		t.Errorf("//heal 300 healed the GM to %v; Mobius leaves it out", hp)
	}

	box := &testNpc{}
	e.onLoop(func() {
		id, err := e.srv.World().IDs().Next()
		if err != nil {
			t.Errorf("IDs: %v", err)
			return
		}
		box.Base = world.NewBase(id, walkOffset(gm.Position(), 50, 0, 0))
		if err := e.srv.AddObject(box); err != nil {
			t.Errorf("AddObject: %v", err)
		}
		gm.Target = box
	})
	gmRun(e, sg, "heal")
	ps := sg.take(t)
	checkNames(t, "//heal on an object that is no creature", ps, "SystemMessage")
	if m := ps[0].(testclient.SystemMessage); m.ID != int32(gmSysMsgInvalidTarget) {
		t.Errorf("system message %d, want %d", m.ID, gmSysMsgInvalidTarget)
	}
}

// //invis hides the GM from players below GM level (DeleteObject) and
// shows other GMs its "Invisible" CharInfo; //vis shows it again.
func TestGMInvisible(t *testing.T) {
	e := newEnv(t, nil)
	sg, sp, gm, _ := gmPair(t, e)
	e.createCharacter("watcher", "Watcher", classHumanFighter, storage.AccessLevelAdmin)
	sw := newSession("watcher")
	e.enter(sw, 0)
	walkSettle(t, e, sg, sp, sw)

	gmRun(e, sg, "invis")
	ps := sp.take(t)
	checkNames(t, "the player when the GM goes invisible", ps, "DeleteObject")
	if d := ps[0].(testclient.DeleteObject); d.ObjectID != gm.ObjectID() {
		t.Errorf("DeleteObject of %d, want the GM", d.ObjectID)
	}
	ps = sw.take(t)
	checkNames(t, "another GM", ps, "CharInfo")
	if ci := ps[0].(testclient.CharInfoPacket); ci.Title != "Invisible" || ci.Invisible {
		t.Errorf("another GM sees title %q, invisible %v", ci.Title, ci.Invisible)
	}
	if msgs := gmSaid(sg.take(t)); !slices.Equal(msgs, []string{"Now, you cannot be seen."}) {
		t.Errorf("messages %q", msgs)
	}
	e.advance(world.GameTick)
	ps = sg.take(t)
	checkNames(t, "the GM at the end of the tick", ps, "UserInfo")
	if ui := ps[0].(testclient.UserInfo); ui.Title != "[Invisible]" {
		t.Errorf("UserInfo title %q", ui.Title)
	}
	checkNames(t, "the player at the end of the tick", sp.take(t))
	checkNames(t, "another GM at the end of the tick", sw.take(t), "CharInfo")

	gmRun(e, sg, "visible")
	checkNames(t, "the player when the GM shows again", sp.take(t), "CharInfo", "RelationChanged")
	checkNames(t, "another GM", sw.take(t), "CharInfo")
	if msgs := gmSaid(sg.take(t)); !slices.Equal(msgs, []string{"Now, you can be seen."}) {
		t.Errorf("messages %q", msgs)
	}
	e.advance(world.GameTick)
	checkNames(t, "the GM", sg.take(t), "UserInfo")
	checkNames(t, "the player", sp.take(t), "CharInfo")
	checkNames(t, "another GM", sw.take(t), "CharInfo")
}

// //online reports the players in the game, as Mobius AdminOnline.
func TestGMOnline(t *testing.T) {
	e := newEnv(t, nil)
	sg, _, _, _ := gmPair(t, e)
	gmRun(e, sg, "online")
	want := []string{
		"Online Player Report",
		"Total count: 2",
		"Total online: 2",
		"Total offline: 0",
		"Max connected: 2",
		"Unique IPs: 2",
		"In peace zone: 0",
		"Not in peace zone: 2",
		"In instances: 0",
		"In combat: 0",
	}
	if msgs := gmSaid(sg.take(t)); !slices.Equal(msgs, want) {
		t.Errorf("report %q, want %q", msgs, want)
	}
}

// Players below GM level get nothing from the GM commands.
func TestGMCommandsNeedAccess(t *testing.T) {
	e := newEnv(t, nil)
	_, sp, _, pl := gmPair(t, e)
	home := e.position(pl)
	for _, line := range []string{"teleport 0 0 0", "loc", "gmspeed 2", "heal", "invis", "online"} {
		gmRun(e, sp, line)
		if ps := sp.take(t); len(ps) != 0 {
			t.Errorf("%q: the player got %q", line, testclient.Names(ps))
		}
	}
	e.advance(time.Second)
	checkNames(t, "the player a second later", sp.take(t))
	var invisible bool
	var speed float64
	e.onLoop(func() { invisible, speed = pl.Invisible(), pl.Stats().MoveSpeed() })
	if invisible || math.Abs(speed-126.5) > 1e-9 || e.position(pl) != home {
		t.Errorf("a player's commands changed it: invisible %v speed %v", invisible, speed)
	}
}

// Java prints doubles with at least one decimal.
func TestGMJavaDouble(t *testing.T) {
	for v, want := range map[float64]string{200: "200.0", 0: "0.0", 150: "150.0", 123.4: "123.4", 1000: "1000.0"} {
		if got := gmJavaDouble(v); got != want {
			t.Errorf("gmJavaDouble(%v) = %q, want %q", v, got, want)
		}
	}
}
