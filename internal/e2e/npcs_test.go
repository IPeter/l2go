package e2e

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/IPeter/l2go/internal/testclient"
)

// talkRange is how close a player must be to talk to an NPC (Mobius
// Npc.INTERACTION_DISTANCE, the default of npcs.interactionRange).
const talkRange = 250

// npcsAtStart: A sees the NPCs of the Human Fighter start, the Newbie
// Helper and Roien, as NpcInfo at their spawn points (M4.1).
func (w *walk) npcsAtStart(t *testing.T) {
	ctx := w.stepCtx(t, 0)
	for _, n := range []struct {
		id   int32
		name string
		at   testclient.Position
	}{
		{npcNewbieHelper, "the Newbie Helper", newbieHelperSpawn},
		{npcRoien, "Roien", roienSpawn},
	} {
		o := findNpc(t, ctx, w.a.conn, n.id)
		if o.Kind != testclient.KindNpc || o.Attackable {
			t.Errorf("%s is shown as %v, attackable %v; want an NpcInfo, not attackable", n.name, o.Kind, o.Attackable)
		}
		if d := dist2D(o.Pos, n.at); d > posTolerance {
			t.Errorf("%s stands at %v, %.0f units from its spawn point %v", n.name, o.Pos, d, n.at)
		}
	}
}

// targetHelper: A clicks the Newbie Helper once. MyTargetSelected makes it
// A's target, and B, next to A, sees A select it (TargetSelected) (M4.2).
func (w *walk) targetHelper(t *testing.T) {
	ctx := w.stepCtx(t, 0)
	a := &w.a
	helper := findNpc(t, ctx, a.conn, npcNewbieHelper)
	if _, err := a.conn.Target(ctx, helper.ObjectID); err != nil {
		t.Fatalf("Target: %v", err)
	}
	if got := a.conn.World().Me().TargetID; got != helper.ObjectID {
		t.Errorf("A's target is %d, want the Newbie Helper %d", got, helper.ObjectID)
	}
	ts := await(t, ctx, w.b.conn, func(s testclient.TargetSelected) bool { return s.ObjectID == a.id })
	if ts.TargetID != helper.ObjectID {
		t.Errorf("B sees A target %d, want the Newbie Helper %d", ts.TargetID, helper.ObjectID)
	}
}

// talkToHelper: A clicks the targeted Newbie Helper again, and its dialog
// opens: Mobius's default/30009.htm, with a Quest link (M4.2). A is a new
// character, so the tutorial runs: Q00999_T0Tutorial owns the helper's first
// talk and, as in Mobius, shows the default page and then the helper's
// tutorial page (30530-01.htm, the first mission).
func (w *walk) talkToHelper(t *testing.T) {
	w.needHTML(t)
	ctx := w.stepCtx(t, 0)
	c := w.a.conn
	helper := findNpc(t, ctx, c, npcNewbieHelper)
	page, err := c.Talk(ctx, helper.ObjectID)
	if err != nil {
		t.Fatalf("Talk: %v", err)
	}
	checkPage(t, page, helper.ObjectID, "Newbie Helper:", "Cedric's Training Hall")
	requireLink(t, page, "Script")
	tutorial := await[testclient.NpcHtmlMessage](t, ctx, c, nil)
	checkPage(t, tutorial, helper.ObjectID, "Newbie Helper:", "Are you ready for a mission?")
}

// helperLinks: links the last page did not offer do nothing: the Quest link
// of the default page, which the tutorial page replaced, and a Chat link no
// page offered (M4.3, decision N5).
func (w *walk) helperLinks(t *testing.T) {
	w.needHTML(t)
	ctx := w.stepCtx(t, 0)
	c := w.a.conn
	helper := findNpc(t, ctx, c, npcNewbieHelper)
	bypass(t, ctx, c, "Script", fmt.Sprintf("npc_%d_Chat 1", helper.ObjectID))
	none[testclient.NpcHtmlMessage](t, ping(t, ctx, c), "links the last page did not offer", nil)
}

// luaGuard: A clicks the guard down the road. The first click targets it.
// The guard stands out of talking range, so the second click walks A up to
// it (MoveToPawn). Once A is close, the walk ends with StopMove (on arrival,
// or when A comes within reach and talks) and the dialog opens, so the
// client knows where A stands (M4.2). The script in
// scripts/npcs/talking_island_guard owns the guards' first talk: its page
// greets A by name and level, its links lead back into the script (an
// inline page, then the page again), and one makes the guard speak (M4.5).
// The script brings its own page, so this step needs no data/html.
func (w *walk) luaGuard(t *testing.T) {
	ctx := w.stepCtx(t, 0)
	a := &w.a
	c := a.conn
	guard := findNpc(t, ctx, c, npcGuard)
	if d := dist2D(guard.Pos, guardSpawn); d > posTolerance {
		t.Errorf("the guard stands at %v, %.0f units from its spawn point %v", guard.Pos, d, guardSpawn)
	}
	far := dist2D(guard.Pos, a.pos)
	if far <= talkRange || far > 2*talkRange {
		t.Fatalf("the nearest guard stands %.0f units from A; want it just out of talking range (%d)", far, talkRange)
	}
	if _, err := c.Target(ctx, guard.ObjectID); err != nil {
		t.Fatalf("Target: %v", err)
	}
	if err := c.Action(ctx, guard.ObjectID, false); err != nil {
		t.Fatalf("Action: %v", err)
	}
	page, before := until[testclient.NpcHtmlMessage](t, ctx, c, nil)
	walkUp := func(m testclient.MoveToPawn) bool { return m.ObjectID == a.id && m.TargetID == guard.ObjectID }
	if !slices.ContainsFunc(testclient.FindAll[testclient.MoveToPawn](before), walkUp) {
		t.Errorf("A talked to the guard %.0f units away without walking up to it (MoveToPawn); it got %s",
			far, strings.Join(testclient.Names(before), ", "))
	}
	stopped := func(m testclient.StopMove) bool { return m.ObjectID == a.id }
	if !slices.ContainsFunc(testclient.FindAll[testclient.StopMove](before), stopped) {
		t.Errorf("A's walk up to the guard did not end with StopMove before the dialog; it got %s",
			strings.Join(testclient.Names(before), ", "))
	}
	me := c.World().Me()
	if d := dist2D(me.Pos, guard.Pos); d > talkRange {
		t.Errorf("A talks to the guard from %.0f units, want at most %d", d, talkRange)
	}
	a.pos = me.Pos

	welcome := "Welcome to Talking Island, " + a.name + "!"
	checkPage(t, page, guard.ObjectID, welcome, "At level 1 the fields")

	page = follow(t, ctx, c, requireLink(t, page, "Script TalkingIslandGuard directions"))
	checkPage(t, page, guard.ObjectID, "The village is small, "+a.name+", you will find your way.")
	page = follow(t, ctx, c, requireLink(t, page, "Script TalkingIslandGuard welcome"))
	checkPage(t, page, guard.ObjectID, welcome)

	bypass(t, ctx, c, requireLink(t, page, "Script TalkingIslandGuard salute"))
	say := await(t, ctx, c, func(s testclient.NpcSay) bool { return s.ObjectID == guard.ObjectID })
	if want := "Stay on the roads, " + a.name + "!"; say.Text != want || say.NpcID() != npcGuard {
		t.Errorf("the guard says %q as NPC %d, want %q as NPC %d", say.Text, say.NpcID(), want, npcGuard)
	}
}

// gatekeeperDialog: B, next to Roxxy after the GM teleport, opens her
// dialog: Mobius's teleporter/30006.htm, with her object ID in the links.
// Its Chat link opens 30006-1.htm, links that page did not offer do
// nothing, and its Chat 0 link leads back (M4.2, M4.3).
func (w *walk) gatekeeperDialog(t *testing.T) {
	w.needStore(t)
	w.needHTML(t)
	ctx := w.stepCtx(t, 0)
	c := w.b.conn
	roxxy := findNpc(t, ctx, c, npcRoxxy)
	id := roxxy.ObjectID
	if _, err := c.Target(ctx, id); err != nil {
		t.Fatalf("Target: %v", err)
	}
	page, err := c.Talk(ctx, id)
	if err != nil {
		t.Fatalf("Talk: %v", err)
	}
	checkPage(t, page, id, "Gatekeeper Roxxy:")
	showTeleports := requireLink(t, page, fmt.Sprintf("npc_%d_showTeleports", id))

	page = follow(t, ctx, c, requireLink(t, page, fmt.Sprintf("npc_%d_Chat 1", id)))
	checkPage(t, page, id, "The Ivory Tower is the foremost temple of all Mystics.")
	back := requireLink(t, page, fmt.Sprintf("npc_%d_Chat 0", id))

	bypass(t, ctx, c, showTeleports, fmt.Sprintf("npc_%d_Chat 2", id))
	none[testclient.NpcHtmlMessage](t, ping(t, ctx, c), "links the last page did not offer", nil)

	page = follow(t, ctx, c, back)
	checkPage(t, page, id, "Gatekeeper Roxxy:")
	requireLink(t, page, showTeleports)
}

// gatekeeperTeleport: Roxxy's teleport list shows every destination with
// its fee (the in-process stack makes teleports cost adena from level 1).
// B's 1000 adena are not enough for the Village of Gludin: "You do not have
// enough adena." and B stays. The Obelisk of Victory takes its fee and
// moves B through the loading screen. From there Roxxy's links do nothing
// (M4.4, decisions N5 and N7).
func (w *walk) gatekeeperTeleport(t *testing.T) {
	w.needStore(t)
	w.needHTML(t)
	ctx := w.stepCtx(t, 0)
	b := &w.b
	c := b.conn
	if got := c.World().ItemCount(itemAdena); got != bAdena {
		t.Fatalf("B has %d adena, want the %d the store gave it", got, bAdena)
	}
	roxxy := findNpc(t, ctx, c, npcRoxxy)
	page := follow(t, ctx, c, fmt.Sprintf("npc_%d_showTeleports", roxxy.ObjectID))
	checkPage(t, page, roxxy.ObjectID)
	links := pageLinks(page.HTML)
	gludin := destination(t, links, "The Village of Gludin", gludinFee)
	toObelisk := destination(t, links, "Obelisk of Victory", obeliskFee)

	bypass(t, ctx, c, gludin.bypass)
	msg, before := until[testclient.SystemMessage](t, ctx, c, nil)
	if msg.ID != testclient.SysMsgNotEnoughAdena {
		t.Errorf("teleport without enough adena: system message %d %+v, want %d (You do not have enough adena.)",
			msg.ID, msg.Params, testclient.SysMsgNotEnoughAdena)
	}
	none(t, before, "a teleport without enough adena", func(p testclient.TeleportToLocation) bool { return p.ObjectID == b.id })
	if got := c.World().ItemCount(itemAdena); got != bAdena {
		t.Errorf("B has %d adena after a teleport it could not pay, want %d", got, bAdena)
	}

	bypass(t, ctx, c, toObelisk.bypass)
	tr, err := c.FinishTeleport(ctx)
	if err != nil {
		t.Fatalf("FinishTeleport: %v", err)
	}
	if d := dist2D(tr.Teleport.Position(), obelisk); d > 300 {
		t.Errorf("teleported to %v, %.0f units from the Obelisk of Victory %v", tr.Teleport.Position(), d, obelisk)
	}
	checkPosition(t, "UserInfo after the teleport", tr.UserInfo.Position(), tr.Teleport.Position())
	b.pos = tr.UserInfo.Position()
	poll(t, ctx, fmt.Sprintf("%d adena left after the fee", bAdena-obeliskFee), func() (bool, error) {
		return c.World().ItemCount(itemAdena) == bAdena-obeliskFee, nil
	})

	bypass(t, ctx, c, toObelisk.bypass)
	none(t, ping(t, ctx, c), "a gatekeeper link from far away", func(p testclient.TeleportToLocation) bool { return p.ObjectID == b.id })
	if got := c.World().ItemCount(itemAdena); got != bAdena-obeliskFee {
		t.Errorf("B has %d adena after a gatekeeper link from far away, want %d", got, bAdena-obeliskFee)
	}
}

// destination returns the link of a teleport list to the destination name,
// whose text must show the fee.
func destination(t *testing.T, links []pageLink, name string, fee int64) pageLink {
	t.Helper()
	for _, l := range links {
		if strings.HasPrefix(l.text, name) {
			if !strings.Contains(l.text, strconv.FormatInt(fee, 10)) {
				t.Errorf("the teleport list shows %q, want the fee %d", l.text, fee)
			}
			return l
		}
	}
	t.Fatalf("the teleport list has no %q: %+v", name, links)
	return pageLink{}
}
