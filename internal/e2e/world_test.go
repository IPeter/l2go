package e2e

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/IPeter/l2go/internal/testclient"
)

// Item, NPC and message IDs of the walk, from the Mobius data.
const (
	itemAdena         = 57
	itemDagger        = 10
	itemSquireShirt   = 1146
	itemSquirePants   = 1147
	itemSquireSword   = 2369
	itemTutorialGuide = 5588

	npcRoxxy        = 30006 // the Talking Island gatekeeper
	npcRoien        = 30008 // Grand Master Roien, at the Human Fighter start
	npcNewbieHelper = 30009 // at the Human Fighter start
	npcGuard        = 31032 // a Talking Island guard (scripts/npcs/talking_island_guard)

	// sysMsgAnotherLogin is "Another person has logged in with the same
	// account." (Mobius ANOTHER_PERSON_HAS_LOGGED_IN_WITH_THE_SAME_ACCOUNT).
	sysMsgAnotherLogin int32 = 421

	// week is DeleteCharAfterDays (7) in seconds, the delete timer of a
	// character just marked for deletion.
	week = 7 * 24 * 60 * 60

	// bAdena is the adena the in-process store gives B: enough for Roxxy's
	// Obelisk of Victory (470), not for the Village of Gludin (18000).
	bAdena     = 1000
	obeliskFee = 470
	gludinFee  = 18000
)

// Places of the walk, from the Mobius data. Regions are 2048 units wide and
// a player sees the 3x3 regions around its own.
var (
	// humanFighterStarts are the Human Fighter creation points
	// (HumanFighter.xml), all in one region.
	humanFighterStarts = []testclient.Position{
		{X: -71338, Y: 258271, Z: -3104},
		{X: -71417, Y: 258270, Z: -3104},
		{X: -71453, Y: 258305, Z: -3104},
		{X: -71467, Y: 258378, Z: -3104},
	}
	// newbieHelperSpawn and roienSpawn are where the NPCs of the start stand
	// (spawns/TalkingIsland/TalkingIslandNPCs.xml).
	newbieHelperSpawn = testclient.Position{X: -71424, Y: 258336, Z: -3104}
	roienSpawn        = testclient.Position{X: -71384, Y: 258304, Z: -3104}
	// helpersSpot is between them, in talking range of both.
	helpersSpot = testclient.Position{X: -71380, Y: 258340, Z: -3104}
	// guardSpawn is a guard on the road south-west of the start, and
	// roadSpot is 400 units before it, out of its talking range. roadSpot is
	// about 4,300 units from the start, two regions off in X and in Y, so
	// the players there and at the start do not see each other.
	guardSpawn = testclient.Position{X: -75001, Y: 255290, Z: -3208}
	roadSpot   = testclient.Position{X: -74695, Y: 255548, Z: -3208}
	// roxxySpot is next to the gatekeeper Roxxy at (-84108, 244604, -3728),
	// in the village.
	roxxySpot = testclient.Position{X: -84048, Y: 244544, Z: -3728}
	// obelisk is Roxxy's destination "Obelisk of Victory".
	obelisk = testclient.Position{X: -99586, Y: 237637, Z: -3568}
)

// inWorld walks A and B through the world, following the demo checklists
// of Milestone 3 (Plan 05; steps M3.n after its items) and Milestone 4
// (Plan 06; M4.n). Both are new Human Fighters, so they start a few steps
// apart at the Human Fighter creation points, by the Newbie Helper:
//
//   - A enters and sees the NPCs of the start; B enters next to it. They see
//     each other walk and chat; A targets the Newbie Helper and talks to it.
//   - A walks down the road to a guard, out of B's sight: chat out of range,
//     shout, trade, whisper; the guard's Lua page; GM commands refused; /loc.
//   - A restarts, is deleted and restored, and enters where it stood; a
//     second login of A's account kicks the first session.
//   - B, a GM with adena in the in-process stack, uses GM commands,
//     teleports to the gatekeeper Roxxy, follows her links and teleports
//     with her, first without enough adena, then with.
//   - B logs out and in again.
//
// Every step builds on the ones before it.
func (w *walk) inWorld(t *testing.T) {
	w.step(t, "M3.1 enter the world", w.enterWorld)
	w.step(t, "M4.1 NPCs stand at the start", w.npcsAtStart)
	w.step(t, "M3.4 a second player enters nearby", w.secondPlayer)
	w.step(t, "M3.3 the other player sees the walk", w.walkNearby)
	w.step(t, "M3.5 chat in range", w.chatInRange)
	w.step(t, "M4.2 target an NPC", w.targetHelper)
	w.step(t, "M4.2 talk to an NPC", w.talkToHelper)
	w.step(t, "M4.3 links and refused links", w.helperLinks)
	w.step(t, "M3.4 walking away hides the player", w.walkAway)
	w.step(t, "M3.5 chat out of range, shout, trade, whisper", w.chatFar)
	w.step(t, "M4.5 a Lua NPC answers with its page", w.luaGuard)
	w.step(t, "M3.10 GM commands are not for players, /loc", w.noGMForPlayers)
	w.step(t, "M3.6 restart returns to the character list", w.restart)
	w.step(t, "M3.8 delete a character and restore it", w.deleteRestore)
	w.step(t, "M3.6 enter again at the same position", w.enterAgain)
	w.step(t, "M3.9 a second login kicks the first session", w.doubleLogin)
	w.step(t, "M3.10 GM commands", w.gmCommands)
	w.step(t, "M4.3 gatekeeper dialog", w.gatekeeperDialog)
	w.step(t, "M4.4 gatekeeper teleport with and without adena", w.gatekeeperTeleport)
	w.step(t, "M3.7 log out and in again at the same position", w.logoutLogin)
}

// enterWorld: A selects its character and enters the world at a Human
// Fighter creation point in its starting gear (M3.1). The packets come in
// Mobius's order, and the character window shows Mobius's numbers (M3.2).
func (w *walk) enterWorld(t *testing.T) {
	ctx := w.stepCtx(t, 0)
	a := &w.a
	sel, res := enter(t, ctx, a)
	checkCharSelected(t, sel, *a)
	pos := testclient.Position{X: sel.X, Y: sel.Y, Z: sel.Z}
	if !slices.Contains(humanFighterStarts, pos) {
		t.Errorf("CharSelected at %v, want a Human Fighter creation point %v", pos, humanFighterStarts)
	}
	// A new character has full HP and MP (Mobius CharacterCreate).
	if sel.CurrentHP != float64(newFighter.MaxHP) || sel.CurrentMP != float64(newFighter.MaxMP) {
		t.Errorf("CharSelected HP %v MP %v, want full: %d and %d", sel.CurrentHP, sel.CurrentMP, newFighter.MaxHP, newFighter.MaxMP)
	}
	checkEnterSequence(t, res, *a, pos)

	u := res.UserInfo
	if got := numbersOf(u); got != newFighter {
		t.Errorf("character window:\n got %+v\nwant %+v", got, newFighter)
	}
	if u.HP != newFighter.MaxHP || u.MP != newFighter.MaxMP || u.Exp != 0 || u.SP != 0 || u.GM || u.PAtkSpd2 != u.PAtkSpd {
		t.Errorf("UserInfo HP %d MP %d exp %d SP %d GM %v attack speeds %d/%d; want full HP %d and MP %d, no exp, SP or GM, equal speeds",
			u.HP, u.MP, u.Exp, u.SP, u.GM, u.PAtkSpd, u.PAtkSpd2, newFighter.MaxHP, newFighter.MaxMP)
	}
	checkStartingItems(t, res.Items, u)
	if len(res.Shortcuts.Shortcuts)+len(res.Quests.Quests)+len(res.Skills.Skills) != 0 {
		t.Errorf("a new character has shortcuts %+v, quests %+v, skills %+v", res.Shortcuts, res.Quests, res.Skills)
	}
	if res.EtcStatus != (testclient.EtcStatusUpdate{}) {
		t.Errorf("EtcStatusUpdate %+v, want no status icons", res.EtcStatus)
	}
	want := testclient.ExStorageMaxCount{
		Inventory: 80, Warehouse: 100, ClanWarehouse: 200, PrivateSell: 3, PrivateBuy: 4, DwarfRecipe: 50, CommonRecipe: 50,
	}
	if limits, ok := testclient.Find[testclient.ExStorageMaxCount](res.Packets); !ok || limits != want {
		t.Errorf("ExStorageMaxCount %+v, want %+v", limits, want)
	}
	a.pos = pos
}

// secondPlayer brings in B: a second account with a new Human Fighter, which
// enters at the start next to A. Each sees the other (M3.4). In the
// in-process stack B becomes a GM with adena first, for the GM and
// gatekeeper steps.
func (w *walk) secondPlayer(t *testing.T) {
	ctx := w.stepCtx(t, 0)
	b := &w.b
	if list := w.login(t, ctx, b); len(list.Characters) != 0 {
		t.Fatalf("new account %s has characters: %+v", b.account, list.Characters)
	}
	res, err := b.conn.CreateCharacter(ctx, humanFighter(b.name))
	if err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}
	if !res.Created {
		t.Fatalf("%q: refused with %v", b.name, res.Reason)
	}
	info, ok := res.Characters.Find(b.name)
	if !ok {
		t.Fatalf("%q is not in the list: %+v", b.name, res.Characters.Characters)
	}
	b.id = info.ObjectID
	if w.f.makeGM != nil {
		w.f.makeGM(t, b.account)
	}
	if w.f.giveItems != nil {
		w.f.giveItems(t, b.account, itemAdena, bAdena)
	}

	sel, er := enter(t, ctx, b)
	checkCharSelected(t, sel, *b)
	b.pos = testclient.Position{X: sel.X, Y: sel.Y, Z: sel.Z}
	checkEnterSequence(t, er, *b, b.pos)
	if gm := w.f.makeGM != nil; er.UserInfo.GM != gm {
		t.Errorf("B's UserInfo shows GM %v, want %v", er.UserInfo.GM, gm)
	}

	// B saw A while entering; A sees B come.
	ci, ok := findCharInfo(er.Packets, w.a.id)
	if !ok {
		t.Fatalf("B entered without a CharInfo of A (%s)", strings.Join(er.Names(), ", "))
	}
	checkFighterCharInfo(t, "A as B sees it", ci, w.a)
	ci = await(t, ctx, w.a.conn, func(c testclient.CharInfoPacket) bool { return c.ObjectID == b.id })
	checkFighterCharInfo(t, "B as A sees it", ci, *b)
	if _, ok := w.a.conn.World().FindPlayer(b.name); !ok {
		t.Error("A's world has no B")
	}
}

// walkNearby: A walks to the NPCs of the start. The server echoes the move
// to A and shows it to B (M3.3, M3.4).
func (w *walk) walkNearby(t *testing.T) {
	a := &w.a
	d := walkTime(a.pos, helpersSpot)
	ctx := w.stepCtx(t, d)
	checkMove(t, "A's own MoveToLocation", moveTo(t, ctx, a, helpersSpot), a.pos, helpersSpot)
	checkMove(t, "A's MoveToLocation at B", awaitMove(t, ctx, w.b.conn, a.id, helpersSpot), a.pos, helpersSpot)
	w.let(t, d)
	a.pos = helpersSpot
}

// chatInRange: A's normal chat reaches A and B, who stands next to it; B's
// shout reaches both (M3.5).
func (w *walk) chatInRange(t *testing.T) {
	ctx := w.stepCtx(t, 0)
	a, b := &w.a, &w.b
	hello := "Hello, " + b.name
	if err := a.conn.Say(ctx, testclient.ChatGeneral, hello); err != nil {
		t.Fatalf("Say: %v", err)
	}
	checkSay(t, "A's normal chat at A", awaitSay(t, ctx, a.conn, testclient.ChatGeneral, hello), a.id, a.name)
	checkSay(t, "A's normal chat at B", awaitSay(t, ctx, b.conn, testclient.ChatGeneral, hello), a.id, a.name)

	shout := "Hello, everyone"
	if err := b.conn.Say(ctx, testclient.ChatShout, shout); err != nil {
		t.Fatalf("Say: %v", err)
	}
	checkSay(t, "B's shout at A", awaitSay(t, ctx, a.conn, testclient.ChatShout, shout), b.id, b.name)
	checkSay(t, "B's shout at B", awaitSay(t, ctx, b.conn, testclient.ChatShout, shout), b.id, b.name)
}

// walkAway: A walks down the road towards a guard, two regions away in X
// and in Y. B sees the walk start and then A disappear (DeleteObject), and
// A loses B the same way (M3.4).
func (w *walk) walkAway(t *testing.T) {
	a, b := &w.a, &w.b
	d := walkTime(a.pos, roadSpot)
	ctx := w.stepCtx(t, d)
	checkMove(t, "A's own MoveToLocation", moveTo(t, ctx, a, roadSpot), a.pos, roadSpot)
	checkMove(t, "A's MoveToLocation at B", awaitMove(t, ctx, b.conn, a.id, roadSpot), a.pos, roadSpot)

	// A arrives; on the way it leaves B's sight.
	w.let(t, d)
	await(t, ctx, b.conn, func(o testclient.DeleteObject) bool { return o.ObjectID == a.id })
	await(t, ctx, a.conn, func(o testclient.DeleteObject) bool { return o.ObjectID == b.id })
	if _, ok := b.conn.World().FindPlayer(a.name); ok {
		t.Error("B still sees A")
	}
	a.pos = roadSpot
}

// chatFar: A is out of B's normal chat range (1,250 units), so its normal
// chat reaches A alone, while its shout reaches B in the same map region.
// Trade chat needs level 20 (Mobius ChatTrade), so B's is refused. A
// whisper reaches its target wherever it is, and one to a player who is not
// online gets "That player is not online." (M3.5).
func (w *walk) chatFar(t *testing.T) {
	ctx := w.stepCtx(t, 0)
	a, b := &w.a, &w.b
	far := "Can you hear me, " + b.name + "?"
	if err := a.conn.Say(ctx, testclient.ChatGeneral, far); err != nil {
		t.Fatalf("Say: %v", err)
	}
	checkSay(t, "A's normal chat at A", awaitSay(t, ctx, a.conn, testclient.ChatGeneral, far), a.id, a.name)

	// The server handles A's packets in order, so the shout reaches B after
	// the normal chat would have.
	shout := "Can you hear this?"
	if err := a.conn.Say(ctx, testclient.ChatShout, shout); err != nil {
		t.Fatalf("Say: %v", err)
	}
	m, before := until(t, ctx, b.conn, func(m testclient.CreatureSay) bool {
		return m.Type == testclient.ChatShout && m.Text == shout
	})
	checkSay(t, "A's shout at B", m, a.id, a.name)
	none(t, before, fmt.Sprintf("B, %.0f units from A's normal chat", dist2D(a.pos, b.pos)),
		func(m testclient.CreatureSay) bool { return m.Text == far })

	trade := "Selling a Dagger"
	if err := b.conn.Say(ctx, testclient.ChatTrade, trade); err != nil {
		t.Fatalf("Say: %v", err)
	}
	refusal := awaitText(t, ctx, b.conn, "Trade chat")
	if want := "Players can use Trade chat after Lv. 20."; !slices.Equal(refusal.Texts(), []string{want}) {
		t.Errorf("trade chat at level 1: %q, want %q", refusal.Texts(), want)
	}

	// B's trade chat has been handled, so it would have reached A before
	// anything A asks for now.
	psst := "Psst, " + b.name
	if err := a.conn.Whisper(ctx, b.name, psst); err != nil {
		t.Fatalf("Whisper: %v", err)
	}
	checkSay(t, "A's whisper at B", awaitSay(t, ctx, b.conn, testclient.ChatWhisper, psst), a.id, a.name)
	m, before = until(t, ctx, a.conn, func(m testclient.CreatureSay) bool {
		return m.Type == testclient.ChatWhisper && m.Text == psst
	})
	checkSay(t, "A's whisper at A", m, a.id, "->"+b.name)
	none(t, before, "A, from B's trade chat below level 20", func(m testclient.CreatureSay) bool { return m.Text == trade })

	// Nobody has this name: A's own plus a letter.
	if err := a.conn.Whisper(ctx, a.name+"x", "Anyone there?"); err != nil {
		t.Fatalf("Whisper: %v", err)
	}
	if msg, _ := until[testclient.SystemMessage](t, ctx, a.conn, nil); msg.ID != testclient.SysMsgPlayerNotOnline {
		t.Errorf("whisper to a player who is not online: system message %d %+v, want %d (That player is not online.)",
			msg.ID, msg.Params, testclient.SysMsgPlayerNotOnline)
	}
}

// noGMForPlayers: A is not a GM, so the server ignores its GM commands, as
// Mobius does. The user command /loc tells A where it stands, in the
// message of its map region (M3.10, M4 map regions).
func (w *walk) noGMForPlayers(t *testing.T) {
	ctx := w.stepCtx(t, 0)
	a := &w.a
	for _, cmd := range []string{fmt.Sprintf("teleport %d %d %d", roxxySpot.X, roxxySpot.Y, roxxySpot.Z), "loc"} {
		if err := a.conn.GMCommand(ctx, cmd); err != nil {
			t.Fatalf("GMCommand(%q): %v", cmd, err)
		}
	}
	before := ping(t, ctx, a.conn)
	none[testclient.TeleportToLocation](t, before, "a player's //teleport", nil)
	none(t, before, "a player's GM commands", func(m testclient.SystemMessage) bool {
		_, isLoc := messageLocation(m) // //loc, or "You have been teleported to x y z"
		return isLoc
	})

	if err := a.conn.UserCommand(ctx, testclient.UserCmdLoc); err != nil {
		t.Fatalf("UserCommand: %v", err)
	}
	awaitLocation(t, ctx, a.conn, a.pos)
}

// restart: A leaves the world for the character screen (M3.6). The list
// shows the character as saved, with the maximum HP and MP the stats give.
func (w *walk) restart(t *testing.T) {
	ctx := w.stepCtx(t, 0)
	a := &w.a
	me := a.conn.World().Me()
	a.hp, a.mp = me.HP, me.MP
	list, err := a.conn.Restart(ctx)
	if err != nil {
		t.Fatalf("Restart: %v", err)
	}
	info, ok := list.Find(a.name)
	if !ok || len(list.Characters) != 1 {
		t.Fatalf("list after the restart: %+v, want only %q", list.Characters, a.name)
	}
	if info.ObjectID != a.id || info.Level != 1 || info.ClassID != 0 || info.DeleteTimer != 0 {
		t.Errorf("listed character %+v", info)
	}
	if info.MaxHP != float64(newFighter.MaxHP) || info.MaxMP != float64(newFighter.MaxMP) ||
		int32(info.CurrentHP) != a.hp || int32(info.CurrentMP) != a.mp {
		t.Errorf("listed HP %v of %v, MP %v of %v; want %d of %d and %d of %d as saved",
			info.CurrentHP, info.MaxHP, info.CurrentMP, info.MaxMP, a.hp, newFighter.MaxHP, a.mp, newFighter.MaxMP)
	}
	// The character screen has no world packets left to read.
	if _, err := a.conn.Drain(); err != nil {
		t.Errorf("Drain: %v", err)
	}
}

// deleteRestore: A's character is marked for deletion, and the list shows
// the time left: 7 days (DeleteCharAfterDays). Restoring it clears the
// timer (M3.8).
func (w *walk) deleteRestore(t *testing.T) {
	ctx := w.stepCtx(t, 0)
	a := &w.a
	res, err := a.conn.DeleteCharacter(ctx, 0)
	if err != nil {
		t.Fatalf("DeleteCharacter: %v", err)
	}
	if !res.Deleted {
		t.Fatalf("CharacterDelete answered %+v, want CharDeleteSuccess", res)
	}
	info, ok := res.Characters.Find(a.name)
	if !ok {
		t.Fatalf("%q left the list at once: %+v", a.name, res.Characters.Characters)
	}
	if info.DeleteTimer <= week-300 || info.DeleteTimer > week {
		t.Errorf("delete timer %d s, want about 7 days (%d s)", info.DeleteTimer, week)
	}
	list, err := a.conn.RestoreCharacter(ctx, 0)
	if err != nil {
		t.Fatalf("RestoreCharacter: %v", err)
	}
	if info, ok := list.Find(a.name); !ok || info.DeleteTimer != 0 {
		t.Errorf("list after the restore: %+v, want %q without a delete timer", list.Characters, a.name)
	}
}

// enterAgain: A enters again from the character screen and stands where it
// left the world, with the HP and MP it had (M3.6).
func (w *walk) enterAgain(t *testing.T) {
	ctx := w.stepCtx(t, 0)
	a := &w.a
	sel, res := enter(t, ctx, a)
	checkCharSelected(t, sel, *a)
	checkPosition(t, "CharSelected", testclient.Position{X: sel.X, Y: sel.Y, Z: sel.Z}, a.pos)
	checkEnterSequence(t, res, *a, a.pos)
	u := res.UserInfo
	if u.HP != a.hp || u.MP != a.mp {
		t.Errorf("UserInfo HP %d MP %d, want %d and %d as A left", u.HP, u.MP, a.hp, a.mp)
	}
	if got := numbersOf(u); got != newFighter {
		t.Errorf("character window:\n got %+v\nwant %+v", got, newFighter)
	}
	a.pos = u.Position()
}

// doubleLogin: A's account logs in again, as from a second machine, while A
// is in the world. Selecting the character there kicks the first session
// with "Another person has logged in with the same account." and the
// server closes it. The new session enters where A stood (M3.9).
func (w *walk) doubleLogin(t *testing.T) {
	ctx := w.stepCtx(t, 0)
	a := &w.a
	first := a.conn
	me := first.World().Me()
	if list := w.login(t, ctx, a); len(list.Characters) != 1 || list.Characters[0].Name != a.name {
		t.Fatalf("the second login lists %+v, want only %q", list.Characters, a.name)
	}
	sel, res := enter(t, ctx, a)
	checkCharSelected(t, sel, *a)

	seen := waitClosed(t, ctx, first)
	if !slices.ContainsFunc(seen, func(p testclient.Packet) bool {
		m, ok := p.(testclient.SystemMessage)
		return ok && m.ID == sysMsgAnotherLogin
	}) {
		t.Errorf("the first session closed without the system message %d (Another person has logged in with the same account.); it got %s",
			sysMsgAnotherLogin, strings.Join(testclient.Names(seen), ", "))
	}
	checkEnterSequence(t, res, *a, a.pos)
	if u := res.UserInfo; u.HP != me.HP || u.MP != me.MP {
		t.Errorf("UserInfo HP %d MP %d, want %d and %d as the first session left them", u.HP, u.MP, me.HP, me.MP)
	}
}

// gmCommands: B is a GM. //loc tells where it stands, an unknown command
// gets Mobius's answer, and //teleport takes it to the gatekeeper Roxxy
// through the loading screen: TeleportToLocation, Appearing, UserInfo
// (M3.10).
func (w *walk) gmCommands(t *testing.T) {
	w.needStore(t)
	ctx := w.stepCtx(t, 0)
	b := &w.b
	c := b.conn
	gm := func(cmd string) {
		t.Helper()
		if err := c.GMCommand(ctx, cmd); err != nil {
			t.Fatalf("GMCommand(%q): %v", cmd, err)
		}
	}

	gm("loc")
	awaitLocation(t, ctx, c, b.pos)

	gm("nosuchcommand")
	m := awaitText(t, ctx, c, "nosuchcommand")
	if want := "The command 'nosuchcommand' does not exist!"; !slices.Equal(m.Texts(), []string{want}) {
		t.Errorf("unknown GM command: %q, want %q", m.Texts(), want)
	}

	gm(fmt.Sprintf("teleport %d %d %d", roxxySpot.X, roxxySpot.Y, roxxySpot.Z))
	tr, err := c.FinishTeleport(ctx)
	if err != nil {
		t.Fatalf("FinishTeleport: %v", err)
	}
	checkPosition(t, "TeleportToLocation", tr.Teleport.Position(), roxxySpot)
	checkPosition(t, "UserInfo after the teleport", tr.UserInfo.Position(), tr.Teleport.Position())
	b.pos = tr.UserInfo.Position()

	gm("loc")
	awaitLocation(t, ctx, c, b.pos)
}

// logoutLogin: B logs out (LeaveWorld, then the server closes the
// connection), logs in again and stands where it left, with the same HP and
// MP (M3.7). Then it logs out for good.
func (w *walk) logoutLogin(t *testing.T) {
	ctx := w.stepCtx(t, 0)
	b := &w.b
	me := b.conn.World().Me()
	if err := b.conn.Logout(ctx); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	list := w.login(t, ctx, b)
	if info, ok := list.Find(b.name); !ok || len(list.Characters) != 1 || info.DeleteTimer != 0 {
		t.Fatalf("list after logging in again: %+v, want only %q", list.Characters, b.name)
	}
	sel, res := enter(t, ctx, b)
	checkCharSelected(t, sel, *b)
	checkEnterSequence(t, res, *b, b.pos)
	if u := res.UserInfo; u.HP != me.HP || u.MP != me.MP {
		t.Errorf("UserInfo HP %d MP %d, want %d and %d as B left", u.HP, u.MP, me.HP, me.MP)
	}
	if err := b.conn.Logout(ctx); err != nil {
		t.Fatalf("Logout: %v", err)
	}
}

// enterOrder is the order of Mobius's EnterWorld.runImpl for what L2Go
// sends (decision E1), ending with the SkillList that comes 300 ms later.
// Other packets come in between: the players and NPCs around, and what the
// zones and the enter hooks send.
var enterOrder = []string{
	"UserInfo", "SendMacroList", "ItemList", "ShortcutInit", "HennaInfo", "QuestList",
	"EtcStatusUpdate", "ExStorageMaxCount", "FriendList", "SystemMessage", "SkillCoolTime",
	"UserInfo", "ValidateLocation", "ActionFailed", "SkillList",
}

// checkEnterSequence checks the packets of EnterWorld for pl standing at
// pos: Mobius's order, the welcome message, and UserInfo and
// ValidateLocation for pl at pos.
func checkEnterSequence(t *testing.T, res testclient.EnterWorldResult, pl player, pos testclient.Position) {
	t.Helper()
	names := res.Names()
	if len(names) == 0 || names[0] != "UserInfo" || !subsequence(enterOrder, names) {
		t.Errorf("enter-world packets out of Mobius's order:\n got %s\nwant in this order %s",
			strings.Join(names, ", "), strings.Join(enterOrder, ", "))
	}
	welcome := func(m testclient.SystemMessage) bool { return m.ID == testclient.SysMsgWelcome }
	if !slices.ContainsFunc(testclient.FindAll[testclient.SystemMessage](res.Packets), welcome) {
		t.Error(`no "Welcome to the World of Lineage II." while entering`)
	}
	u := res.UserInfo
	if u.ObjectID != pl.id || u.Name != pl.name {
		t.Errorf("UserInfo of %d %q, want %d %q", u.ObjectID, u.Name, pl.id, pl.name)
	}
	checkPosition(t, "UserInfo", u.Position(), pos)
	checkPosition(t, "ValidateLocation", res.Location.Position(), pos)
}

// checkCharSelected checks CharSelected for pl, a level 1 male Human
// Fighter.
func checkCharSelected(t *testing.T, sel testclient.CharSelected, pl player) {
	t.Helper()
	switch {
	case sel.Name != pl.name || sel.ObjectID != pl.id:
		t.Errorf("CharSelected for %d %q, want %d %q", sel.ObjectID, sel.Name, pl.id, pl.name)
	case sel.SessionID != pl.keys.PlayOK1:
		t.Errorf("CharSelected session ID %d, want PlayOK1 %d", sel.SessionID, pl.keys.PlayOK1)
	case sel.Level != 1 || sel.ClassID != 0 || sel.ClassID2 != 0 || sel.Race != 0 || sel.Sex != 0:
		t.Errorf("CharSelected level %d class %d/%d race %d sex %d, want a level 1 male Human Fighter",
			sel.Level, sel.ClassID, sel.ClassID2, sel.Race, sel.Sex)
	case sel.STR != 40 || sel.DEX != 30 || sel.CON != 43 || sel.INT != 21 || sel.WIT != 11 || sel.MEN != 25:
		t.Errorf("CharSelected STR %d DEX %d CON %d INT %d WIT %d MEN %d, want the template's 40 30 43 21 11 25",
			sel.STR, sel.DEX, sel.CON, sel.INT, sel.WIT, sel.MEN)
	}
}

// fighterNumbers are the numbers of UserInfo that the character window
// shows, and the other values that come from the stat engine.
type fighterNumbers struct {
	Level, Race, Sex, ClassID, BaseClassID int32
	STR, DEX, CON, INT, WIT, MEN           int32
	MaxHP, MaxMP, MaxCP, MaxLoad           int32
	InventoryLimit                         uint16
	PAtk, PAtkSpd, PDef, Evasion           int32
	Accuracy, Critical, MAtk, CastSpd      int32
	MDef                                   int32
	RunSpd, WalkSpd                        int32
	SwimRunSpd, SwimWalkSpd                int32
	CollisionRadius, CollisionHeight       float64
	WeaponFlag                             int32
	Running                                bool
	NameColor, TitleColor                  int32
}

// newFighter is what Mobius shows for a new male Human Fighter in Squire's
// Shirt, Pants and Sword (M3.2): the template's stats, HP, MP and CP with
// the CON and MEN bonuses, the sword's P.Atk and the armor's P.Def. The
// stat engine's hand-checked tests have the same numbers.
var newFighter = fighterNumbers{
	Level: 1,
	STR:   40, DEX: 30, CON: 43, INT: 21, WIT: 11, MEN: 25,
	MaxHP: 126, MaxMP: 38, MaxCP: 50, MaxLoad: 109020,
	InventoryLimit: 80,
	PAtk:           6, PAtkSpd: 417, PDef: 75, Evasion: 34,
	Accuracy: 34, Critical: 88, MAtk: 2, CastSpd: 213,
	MDef:   47,
	RunSpd: runSpeed, WalkSpd: 80,
	SwimRunSpd: 50, SwimWalkSpd: 50,
	CollisionRadius: 9, CollisionHeight: 23,
	WeaponFlag: 40,
	Running:    true,
	NameColor:  0xFFFFFF, TitleColor: 0xECF9A2,
}

func numbersOf(u testclient.UserInfo) fighterNumbers {
	return fighterNumbers{
		Level: u.Level, Race: u.Race, Sex: u.Sex, ClassID: u.ClassID, BaseClassID: u.BaseClassID,
		STR: u.STR, DEX: u.DEX, CON: u.CON, INT: u.INT, WIT: u.WIT, MEN: u.MEN,
		MaxHP: u.MaxHP, MaxMP: u.MaxMP, MaxCP: u.MaxCP, MaxLoad: u.MaxLoad,
		InventoryLimit: u.InventoryLimit,
		PAtk:           u.PAtk, PAtkSpd: u.PAtkSpd, PDef: u.PDef, Evasion: u.Evasion,
		Accuracy: u.Accuracy, Critical: u.Critical, MAtk: u.MAtk, CastSpd: u.CastSpd,
		MDef:   u.MDef,
		RunSpd: u.RunSpd, WalkSpd: u.WalkSpd,
		SwimRunSpd: u.SwimRunSpd, SwimWalkSpd: u.SwimWalkSpd,
		CollisionRadius: u.CollisionRadius, CollisionHeight: u.CollisionHeight,
		WeaponFlag: u.WeaponFlag,
		Running:    u.Running,
		NameColor:  u.NameColor, TitleColor: u.TitleColor,
	}
}

// checkStartingItems checks the inventory of a new Human Fighter (data/
// players/initial_equipment.json): Squire's Shirt, Pants and Sword worn,
// the Dagger and the Tutorial Guide carried, and UserInfo's paperdoll
// showing the worn items.
func checkStartingItems(t *testing.T, items testclient.ItemList, u testclient.UserInfo) {
	t.Helper()
	if items.ShowWindow {
		t.Error("the enter-world ItemList opens the inventory window")
	}
	worn := make(map[int32]int32) // item ID -> object ID
	var carried []int32
	for _, it := range items.Items {
		if it.Equipped {
			worn[it.ItemID] = it.ObjectID
		} else {
			carried = append(carried, it.ItemID)
		}
	}
	slices.Sort(carried)
	if len(worn) != 3 || worn[itemSquireShirt] == 0 || worn[itemSquirePants] == 0 || worn[itemSquireSword] == 0 ||
		!slices.Equal(carried, []int32{itemDagger, itemTutorialGuide}) {
		t.Errorf("ItemList %+v, want Squire's Shirt, Pants and Sword worn, the Dagger and the Tutorial Guide carried", items.Items)
	}
	var want [testclient.NumEquipSlots]int32
	want[testclient.SlotChest] = itemSquireShirt
	want[testclient.SlotLegs] = itemSquirePants
	want[testclient.SlotRHand] = itemSquireSword
	want[testclient.SlotRHand2] = itemSquireSword // Mobius writes the right hand twice
	if u.PaperdollItemIDs != want {
		t.Errorf("UserInfo paperdoll item IDs %v, want %v", u.PaperdollItemIDs, want)
	}
	for slot, id := range want {
		if u.PaperdollObjectIDs[slot] != worn[id] {
			t.Errorf("UserInfo paperdoll slot %d shows object %#x, want %#x", slot, u.PaperdollObjectIDs[slot], worn[id])
		}
	}
}

// checkFighterCharInfo checks how another player sees pl, a new Human
// Fighter made with humanFighter: its name, place, look and starting gear.
func checkFighterCharInfo(t *testing.T, what string, c testclient.CharInfoPacket, pl player) {
	t.Helper()
	if c.Name != pl.name {
		t.Errorf("%s: name %q, want %q", what, c.Name, pl.name)
	}
	checkPosition(t, what, c.Position(), pl.pos)
	if c.Race != 0 || c.Sex != 0 || c.ClassID != 0 || c.BaseClassID != 0 ||
		c.HairStyle != 1 || c.HairColor != 2 || c.Face != 1 {
		t.Errorf("%s: race %d sex %d class %d/%d hair %d/%d face %d, want a male Human Fighter with hair 1/2 and face 1",
			what, c.Race, c.Sex, c.ClassID, c.BaseClassID, c.HairStyle, c.HairColor, c.Face)
	}
	if !c.Standing || !c.Running || c.Invisible ||
		c.RunSpd != newFighter.RunSpd || c.WalkSpd != newFighter.WalkSpd || c.MaxCP != newFighter.MaxCP {
		t.Errorf("%s: standing %v running %v invisible %v speeds %d/%d max CP %d; want visible, standing, running at %d/%d with %d CP",
			what, c.Standing, c.Running, c.Invisible, c.RunSpd, c.WalkSpd, c.MaxCP,
			newFighter.RunSpd, newFighter.WalkSpd, newFighter.MaxCP)
	}
	var want [testclient.NumVisibleSlots]int32
	want[testclient.VisibleChest] = itemSquireShirt
	want[testclient.VisibleLegs] = itemSquirePants
	want[testclient.VisibleRHand] = itemSquireSword
	want[testclient.VisibleRHand2] = itemSquireSword
	if c.ItemIDs != want {
		t.Errorf("%s: item IDs %v, want %v", what, c.ItemIDs, want)
	}
}
