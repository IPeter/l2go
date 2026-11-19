package testclient

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

// testSelected is the CharSelected of the test character "Hero".
func testSelected() CharSelected {
	return CharSelected{
		Name: "Hero", ObjectID: testMeID, SessionID: 0x0c, Active: 1,
		X: -71338, Y: 258271, Z: -3104, CurrentHP: 120, CurrentMP: 30, Level: 1,
		INT: 21, STR: 40, CON: 43, MEN: 25, DEX: 30, WIT: 11, GameTime: 600,
	}
}

// fakeAuth runs the server side of the handshake and AuthLogin.
func (f *fakeGame) fakeAuth(chars []CharInfo) {
	f.expect(msg(0x00).i32(746).b)
	f.sendKey(1, true)
	f.expect(testAuthLogin())
	f.send(mobiusCharSelectionInfo(chars))
}

// fakeSelect runs fakeAuth and the server side of CharacterSelect for slot
// 0.
func (f *fakeGame) fakeSelect(chars []CharInfo) {
	f.fakeAuth(chars)
	f.expect(mobiusCharacterSelect(0))
	f.send(mobiusCharSelected(testSelected()))
}

// clientAuth runs the client side of fakeAuth.
func clientAuth(t *testing.T, ctx context.Context, addr string, opts ...DialOption) *GameConn {
	t.Helper()
	c, err := DialGame(ctx, addr, InterludeProtocol, opts...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	if _, err := c.AuthLogin(ctx, "player1", testGameKeys); err != nil {
		t.Fatal(err)
	}
	return c
}

// clientSelect runs the client side of fakeSelect.
func clientSelect(t *testing.T, ctx context.Context, addr string, opts ...DialOption) *GameConn {
	t.Helper()
	c := clientAuth(t, ctx, addr, opts...)
	sel, err := c.SelectCharacter(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	if sel != testSelected() {
		t.Errorf("SelectCharacter = %+v", sel)
	}
	return c
}

// enterWorldPackets is an enter-world sequence in Mobius order
// (EnterWorld.runImpl) for "Hero" at start, with the gatekeeper and another
// player nearby.
func enterWorldPackets(start, other Position) [][]byte {
	ui := testUserInfoAt(start)
	return [][]byte{
		mobiusUserInfo(ui),
		msg(0xe7).i32(1).u8(0).u8(0).u8(0).b, // SendMacroList without macros
		mobiusItemList(ItemList{Items: testItems()}),
		msg(0x45).i32(0).b,                                            // ShortcutInit
		msg(0xe4).u8(0).u8(0).u8(0).u8(0).u8(0).u8(0).i32(3).i32(0).b, // HennaInfo
		msg(0x80).u16(0).b,                                            // QuestList
		mobiusNpcInfo(testGatekeeper(testNpcID)),
		mobiusCharInfo(testOtherPlayer(other)),
		mobiusRelationChanged(RelationChanged{ObjectID: testOtherID}),
		{0xf8, 0x01, 0x00},                                           // a packet the client does not parse
		msg(0xf3).i32(0).i32(0).i32(0).i32(0).i32(0).i32(0).i32(0).b, // EtcStatusUpdate
		msg(0xfe).u16(0x2e).i32(80).i32(100).i32(150).i32(4).i32(3).i32(0).i32(30).i32(0).b, // ExStorageMaxCount
		msg(0xfa).i32(0).b, // FriendList
		mobiusSystemMessage(SysMsgWelcome),
		msg(0xc1).i32(0).b, // SkillCoolTime
		mobiusUserInfo(ui), // broadcastUserInfo
		mobiusValidateLocation(ValidateLocation{ObjectID: testMeID, X: start.X, Y: start.Y, Z: start.Z}),
		msg(0x25).b, // ActionFailed
		mobiusSkillList([]Skill{{Level: 1, ID: 3}}), // 300 ms later in Mobius
	}
}

var enterWorldNames = []string{
	"UserInfo", "SendMacroList", "ItemList", "ShortcutInit", "HennaInfo", "QuestList",
	"NpcInfo", "CharInfo", "RelationChanged", "Unknown(0xf8)", "EtcStatusUpdate", "ExStorageMaxCount",
	"FriendList", "SystemMessage", "SkillCoolTime", "UserInfo", "ValidateLocation", "ActionFailed", "SkillList",
}

// recordNames returns "direction name" for each record.
func recordNames(recs []Record) []string {
	names := make([]string, len(recs))
	for i, r := range recs {
		names[i] = r.Dir.String() + " " + r.Name
	}
	return names
}

// checkRecords checks what every record must hold.
func checkRecords(t *testing.T, recs []Record) {
	t.Helper()
	for i, r := range recs {
		if r.Seq != i || len(r.Raw) == 0 || r.Raw[0] != r.Opcode || r.Time.IsZero() {
			t.Errorf("record %d: Seq %d, opcode 0x%02x, raw % x", i, r.Seq, r.Opcode, r.Raw)
		}
		switch r.Dir {
		case FromServer:
			if (r.Packet == nil) == (r.Err == nil) {
				t.Errorf("record %d (%s): packet %v, error %v", i, r.Name, r.Packet, r.Err)
			}
		case FromClient:
			if r.Packet != nil || r.Err != nil {
				t.Errorf("record %d (%s): a client packet with a decoded packet", i, r.Name)
			}
		default:
			t.Errorf("record %d: direction %v", i, r.Dir)
		}
	}
}

// TestInWorldFlow walks a character through the world against a fake
// server with encryption on: entering, walking, chat, a GM teleport,
// sitting, targeting and talking to an NPC, dialog links, a social action,
// a refused attack, another player leaving, restart, delete and restore.
func TestInWorldFlow(t *testing.T) {
	hero := testChar("Hero", testMeID)
	deleting := hero
	deleting.DeleteTimer = 7 * 24 * 3600
	start := Position{-71338, 258271, -3104}
	otherStart := Position{-71000, 258000, -3100}
	walkTo := Position{-71000, 258300, -3104}
	otherTo := Position{-70500, 258000, -3100}
	tele := Position{1000, 2000, -3000}
	npc := testGatekeeper(testNpcID)
	bypass := fmt.Sprintf("npc_%d_Chat 1", testNpcID)
	page1 := `<html><body>Roxxy: <a action="bypass -h ` + bypass + `">Chat</a></body></html>`
	page2 := "<html><body>Page 2</body></html>"
	page3 := "<html><body>Page 3</body></html>"
	rec := NewRecorder()

	addr := startFakeGame(t, func(f *fakeGame) {
		f.fakeSelect([]CharInfo{hero})
		f.expect(mobiusEnterWorld())
		for _, b := range enterWorldPackets(start, otherStart) {
			f.send(b)
		}

		// MoveTo: the other player's move comes first and is skipped.
		f.expect(mobiusClientMove(walkTo, start, 1))
		f.send(mobiusMoveToLocation(MoveToLocation{ObjectID: testOtherID,
			DestX: otherTo.X, DestY: otherTo.Y, DestZ: otherTo.Z, X: otherStart.X, Y: otherStart.Y, Z: otherStart.Z}))
		f.send(mobiusMoveToLocation(MoveToLocation{ObjectID: testMeID,
			DestX: walkTo.X, DestY: walkTo.Y, DestZ: walkTo.Z, X: start.X, Y: start.Y, Z: start.Z}))

		// Chat and a whisper to an offline player.
		f.expect(mobiusSay2("hello", ChatGeneral, ""))
		f.send(mobiusCreatureSay(CreatureSay{ObjectID: testMeID, Type: ChatGeneral, Name: "Hero", Text: "hello"}))
		f.expect(mobiusSay2("psst", ChatWhisper, "Nobody"))
		f.send(mobiusSystemMessage(SysMsgS1IsNotCurrentlyLoggedIn, "Nobody"))

		// GM teleport: TeleportToLocation, Appearing, UserInfo.
		f.expect(msg(0x5b).str("teleport 1000 2000 -3000").b)
		f.send(msg(0x28).i32(testMeID).i32(tele.X).i32(tele.Y).i32(tele.Z).i32(0).i32(100).b)
		f.expect([]byte{0x30})
		f.send(mobiusUserInfo(testUserInfoAt(tele)))

		// Sit down.
		f.expect([]byte{0x45, 0, 0, 0, 0, 0, 0, 0, 0, 0})
		f.send(msg(0x2f).i32(testMeID).i32(WaitSitting).i32(tele.X).i32(tele.Y).i32(tele.Z).b)

		// First click: the target and its HP bar.
		f.expect(mobiusAction(0x04, testNpcID, tele, false))
		f.send(mobiusMyTargetSelected(testNpcID, 0))
		f.send(mobiusStatusUpdate(testNpcID, StatusAttribute{StatusCurHP, 80}, StatusAttribute{StatusMaxHP, 100}))

		// Second click: MoveToPawn, the page, ActionFailed.
		f.expect(mobiusAction(0x04, testNpcID, tele, false))
		f.send(mobiusMoveToPawn(MoveToPawn{ObjectID: testMeID, TargetID: testNpcID, Distance: 100,
			X: tele.X, Y: tele.Y, Z: tele.Z, TargetX: npc.X, TargetY: npc.Y, TargetZ: npc.Z}))
		f.send(mobiusNpcHtmlMessage(testNpcID, page1))
		f.send(msg(0x25).b)

		// Dialog links.
		f.expect(msg(0x21).str(bypass).b)
		f.send(mobiusNpcHtmlMessage(testNpcID, page2))
		f.expect(msg(0x20).str("default/30006-1.htm").b)
		f.send(mobiusNpcHtmlMessage(testNpcID, page3))

		// Social action.
		f.expect(msg(0x1b).i32(2).b)
		f.send(msg(0x2d).i32(testMeID).i32(2).b)

		// Attacking the NPC is refused.
		f.expect(mobiusAction(0x0a, testNpcID, tele, false))
		f.send(msg(0x25).b)

		// Cancel the target; the other player leaves and the NPC speaks.
		f.expect([]byte{0x37, 0, 0})
		f.send(msg(0x2a).i32(testMeID).i32(tele.X).i32(tele.Y).i32(tele.Z).i32(0).b)
		f.send(msg(0x12).i32(testOtherID).i32(1).b)
		f.send(msg(0x02).i32(testNpcID).i32(0).i32(1000000 + 30006).str("Welcome!").b)

		// Restart, then delete and restore the character.
		f.expect([]byte{0x46})
		f.send(msg(0x5f).i32(1).b)
		f.send(mobiusCharSelectionInfo([]CharInfo{hero}))
		f.expect(msg(0x0c).i32(0).b)
		f.send(msg(0x23).b)
		f.send(mobiusCharSelectionInfo([]CharInfo{deleting}))
		f.expect(msg(0x62).i32(0).b)
		f.send(mobiusCharSelectionInfo([]CharInfo{hero}))
		f.drain()
	})

	ctx := testContext(t)
	c := clientSelect(t, ctx, addr, WithRecorder(rec))
	w := c.World()
	if me := w.Me(); me.ObjectID != testMeID || me.Name != "Hero" || me.Pos != start {
		t.Errorf("after CharSelected: Me = %+v", me)
	}

	res, err := c.EnterWorld(ctx)
	if err != nil {
		t.Fatalf("EnterWorld: %v", err)
	}
	if !slices.Equal(res.Names(), enterWorldNames) {
		t.Errorf("EnterWorld packets\n got %v\nwant %v", res.Names(), enterWorldNames)
	}
	if res.UserInfo.ObjectID != testMeID || res.UserInfo.MaxHP != 126 || len(res.Items.Items) != 4 ||
		len(res.Skills.Skills) != 1 || res.Location.Position() != start || res.Quests.Quests != nil {
		t.Errorf("EnterWorld result: UserInfo %d/%d, %d items, %d skills, location %v",
			res.UserInfo.ObjectID, res.UserInfo.MaxHP, len(res.Items.Items), len(res.Skills.Skills), res.Location)
	}
	if u, ok := Find[Unknown](res.Packets); !ok || u.Opcode != 0xf8 {
		t.Errorf("the unknown packet is missing: %+v", u)
	}
	if g, ok := w.FindNpc(30006); !ok || g.ObjectID != testNpcID || g.Name != "Roxxy" {
		t.Errorf("FindNpc(30006) = %+v, %v", g, ok)
	}
	if p, ok := w.FindPlayer("Pal"); !ok || p.Pos != otherStart {
		t.Errorf("FindPlayer(Pal) = %+v, %v", p, ok)
	}
	if n := w.ItemCount(57); n != 5000 {
		t.Errorf("adena %d", n)
	}

	// Walking.
	mv, err := c.MoveTo(ctx, walkTo.X, walkTo.Y, walkTo.Z)
	if err != nil {
		t.Fatalf("MoveTo: %v", err)
	}
	if mv.ObjectID != testMeID || mv.Origin() != start || mv.Destination() != walkTo {
		t.Errorf("MoveTo = %+v", mv)
	}
	if p, _ := w.FindPlayer("Pal"); !p.Moving || p.Dest != otherTo {
		t.Errorf("the other player's move was not tracked: %+v", p.Motion)
	}
	if me := w.Me(); !me.Moving || me.Dest != walkTo {
		t.Errorf("own move: %+v", me.Motion)
	}

	// Chat.
	if err := c.Say(ctx, ChatGeneral, "hello"); err != nil {
		t.Fatal(err)
	}
	if s, err := Await(ctx, c, func(s CreatureSay) bool { return s.ObjectID == testMeID }); err != nil || s.Text != "hello" {
		t.Errorf("chat echo = %+v, %v", s, err)
	}
	if err := c.Whisper(ctx, "Nobody", "psst"); err != nil {
		t.Fatal(err)
	}
	sm, err := Await(ctx, c, func(m SystemMessage) bool { return m.ID == SysMsgS1IsNotCurrentlyLoggedIn })
	if err != nil || !slices.Equal(sm.Texts(), []string{"Nobody"}) {
		t.Errorf("whisper answer = %+v, %v", sm, err)
	}
	if len(w.Chat()) != 1 || len(w.SystemMessages()) != 2 {
		t.Errorf("logs: %d chat lines, %d system messages", len(w.Chat()), len(w.SystemMessages()))
	}

	// GM teleport.
	if err := c.GMCommand(ctx, "//teleport 1000 2000 -3000"); err != nil {
		t.Fatal(err)
	}
	tr, err := c.FinishTeleport(ctx)
	if err != nil {
		t.Fatalf("FinishTeleport: %v", err)
	}
	if tr.Teleport.Position() != tele || tr.Teleport.Heading != 100 || tr.UserInfo.Position() != tele {
		t.Errorf("FinishTeleport = %+v", tr.Teleport)
	}
	if me := w.Me(); me.Pos != tele || me.Moving {
		t.Errorf("after the teleport: %+v", me.Motion)
	}

	// Sitting.
	if err := c.RequestActionUse(ctx, ActionSitStand, false, false); err != nil {
		t.Fatal(err)
	}
	if _, err := Await(ctx, c, func(cw ChangeWaitType) bool { return cw.ObjectID == testMeID }); err != nil {
		t.Fatal(err)
	}
	if !w.Me().Sitting {
		t.Error("not sitting after ChangeWaitType")
	}

	// Targeting and the HP bar.
	sel, err := c.Target(ctx, testNpcID)
	if err != nil || sel.ObjectID != testNpcID || sel.Color != 0 {
		t.Fatalf("Target = %+v, %v", sel, err)
	}
	if _, err := Await[StatusUpdate](ctx, c, nil); err != nil {
		t.Fatal(err)
	}
	if g, _ := w.Object(testNpcID); g.HP != 80 || g.MaxHP != 100 || w.Me().TargetID != testNpcID {
		t.Errorf("target HP %d/%d, target %d", g.HP, g.MaxHP, w.Me().TargetID)
	}

	// Talking.
	html, err := c.Talk(ctx, testNpcID)
	if err != nil || html.HTML != page1 || html.ObjectID != testNpcID {
		t.Fatalf("Talk = %+v, %v", html, err)
	}
	if _, err := Expect[ActionFailed](ctx, c); err != nil {
		t.Errorf("after the page: %v", err)
	}
	if h, ok := w.HTML(); !ok || h.HTML != page1 {
		t.Errorf("HTML() = %+v, %v", h, ok)
	}
	if me := w.Me(); !me.Moving || me.Dest != npc.Position() {
		t.Errorf("MoveToPawn not tracked: %+v", me.Motion)
	}

	// Dialog links.
	if err := c.Bypass(ctx, bypass); err != nil {
		t.Fatal(err)
	}
	if p, err := Await[NpcHtmlMessage](ctx, c, nil); err != nil || p.HTML != page2 {
		t.Errorf("bypass page = %+v, %v", p, err)
	}
	if err := c.LinkHtml(ctx, "default/30006-1.htm"); err != nil {
		t.Fatal(err)
	}
	if p, err := Await[NpcHtmlMessage](ctx, c, nil); err != nil || p.HTML != page3 {
		t.Errorf("link page = %+v, %v", p, err)
	}

	// Social action.
	if err := c.RequestSocialAction(ctx, 2); err != nil {
		t.Fatal(err)
	}
	if sa, err := Expect[SocialAction](ctx, c); err != nil || sa != (SocialAction{ObjectID: testMeID, ActionID: 2}) {
		t.Errorf("SocialAction = %+v, %v", sa, err)
	}

	// A refused attack.
	if err := c.AttackRequest(ctx, testNpcID, false); err != nil {
		t.Errorf("AttackRequest: %v", err)
	}

	// Cancel the target; the other player leaves; the NPC speaks.
	if err := c.CancelTarget(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := Expect[TargetUnselected](ctx, c); err != nil {
		t.Errorf("after CancelTarget: %v", err)
	}
	ns, err := Await[NpcSay](ctx, c, nil)
	if err != nil || ns.Text != "Welcome!" || ns.NpcID() != 30006 {
		t.Errorf("NpcSay = %+v, %v", ns, err)
	}
	if w.Me().TargetID != 0 {
		t.Error("the target survived TargetUnselected")
	}
	if _, ok := w.FindPlayer("Pal"); ok {
		t.Error("the other player survived DeleteObject")
	}

	// Back to the character screen.
	list, err := c.Restart(ctx)
	if err != nil || len(list.Characters) != 1 || list.Characters[0].Name != "Hero" {
		t.Fatalf("Restart = %+v, %v", list, err)
	}
	if me := w.Me(); me.ObjectID != 0 || len(w.Objects()) != 0 {
		t.Errorf("the world survived the restart: %+v, %d objects", me.Motion, len(w.Objects()))
	}
	del, err := c.DeleteCharacter(ctx, 0)
	if err != nil || !del.Deleted || del.Failed || len(del.Characters.Characters) != 1 ||
		del.Characters.Characters[0].DeleteTimer != 7*24*3600 {
		t.Fatalf("DeleteCharacter = %+v, %v", del, err)
	}
	list, err = c.RestoreCharacter(ctx, 0)
	if err != nil || len(list.Characters) != 1 || list.Characters[0].DeleteTimer != 0 {
		t.Fatalf("RestoreCharacter = %+v, %v", list, err)
	}

	// The recorder kept both directions in order.
	recs := rec.Records()
	checkRecords(t, recs)
	names := recordNames(recs)
	wantStart := []string{
		"client ProtocolVersion", "server KeyPacket", "client AuthLogin", "server CharSelectionInfo",
		"client CharacterSelect", "server CharSelected", "client EnterWorld", "server UserInfo",
	}
	if len(names) < len(wantStart) || !slices.Equal(names[:len(wantStart)], wantStart) {
		t.Errorf("records start %v", names)
	}
	wantEnd := []string{
		"client CharacterDelete", "server CharDeleteSuccess", "server CharSelectionInfo",
		"client CharacterRestore", "server CharSelectionInfo",
	}
	if len(names) < len(wantEnd) || !slices.Equal(names[len(names)-len(wantEnd):], wantEnd) {
		t.Errorf("records end %v", names)
	}
	for _, r := range recs {
		if r.Name == "ExStorageMaxCount" && r.ExOpcode != ExOpStorageMaxCount {
			t.Errorf("ExStorageMaxCount record has Ex opcode 0x%04x", r.ExOpcode)
		}
		if r.Name == "Unknown(0xf8)" {
			if _, ok := r.Packet.(Unknown); !ok {
				t.Errorf("the unknown packet was recorded as %T", r.Packet)
			}
		}
	}
	if n := len(rec.ServerPackets()); n != int(c.PacketsReceived()) {
		t.Errorf("recorded %d server packets, read %d", n, c.PacketsReceived())
	}
	if n := rec.Len() - len(rec.ServerPackets()); n != int(c.PacketsSent()) {
		t.Errorf("recorded %d client packets, sent %d", n, c.PacketsSent())
	}
}

func TestLogout(t *testing.T) {
	start := Position{1, 2, 3}
	rec := NewRecorder()
	addr := startFakeGame(t, func(f *fakeGame) {
		f.fakeSelect([]CharInfo{testChar("Hero", testMeID)})
		f.expect(mobiusEnterWorld())
		f.send(mobiusUserInfo(testUserInfoAt(start)))
		f.send(mobiusValidateLocation(ValidateLocation{ObjectID: testMeID, X: 1, Y: 2, Z: 3}))
		f.send(msg(0x25).b)
		f.send(mobiusSkillList(nil))
		f.expect([]byte{0x09})
		f.send(chatLine("bye"))
		f.send(msg(0x7e).b)
		// The script ends and the connection closes.
	})
	ctx := testContext(t)
	c := clientSelect(t, ctx, addr, WithRecorder(rec))
	if _, err := c.EnterWorld(ctx); err != nil {
		t.Fatalf("EnterWorld: %v", err)
	}
	if err := c.Logout(ctx); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	want := []string{
		"client ProtocolVersion", "server KeyPacket", "client AuthLogin", "server CharSelectionInfo",
		"client CharacterSelect", "server CharSelected", "client EnterWorld", "server UserInfo",
		"server ValidateLocation", "server ActionFailed", "server SkillList", "client Logout",
		"server CreatureSay", "server LeaveWorld",
	}
	recs := rec.Records()
	checkRecords(t, recs)
	if got := recordNames(recs); !slices.Equal(got, want) {
		t.Errorf("records\n got %v\nwant %v", got, want)
	}
	if c.PacketsSent() != 5 || c.PacketsReceived() != 9 {
		t.Errorf("sent %d, received %d; want 5 and 9", c.PacketsSent(), c.PacketsReceived())
	}
	if _, err := c.Next(ctx); !errors.Is(err, ErrServerClosed) {
		t.Errorf("Next after the logout = %v, want ErrServerClosed", err)
	}
}

func TestLogoutPacketAfterLeaveWorld(t *testing.T) {
	addr := startFakeGame(t, func(f *fakeGame) {
		f.fakeSelect([]CharInfo{testChar("Hero", testMeID)})
		f.expect(mobiusEnterWorld())
		f.send(mobiusValidateLocation(ValidateLocation{ObjectID: testMeID}))
		f.send(msg(0x25).b)
		f.send(mobiusSkillList(nil))
		f.expect([]byte{0x09})
		f.send(msg(0x7e).b)
		f.send(msg(0x25).b)
		f.drain()
	})
	ctx := testContext(t)
	c := clientSelect(t, ctx, addr)
	if _, err := c.EnterWorld(ctx); err != nil {
		t.Fatal(err)
	}
	err := c.Logout(ctx)
	var upe *UnexpectedPacketError
	if !errors.As(err, &upe) || upe.Name != "ActionFailed" || upe.Want != "connection close" {
		t.Errorf("Logout = %v, want ActionFailed as an unexpected packet", err)
	}
}

func TestEnterWorldTimeout(t *testing.T) {
	addr := startFakeGame(t, func(f *fakeGame) {
		f.fakeSelect([]CharInfo{testChar("Hero", testMeID)})
		f.expect(mobiusEnterWorld())
		f.send(mobiusUserInfo(testUserInfoAt(Position{})))
		f.send(mobiusValidateLocation(ValidateLocation{ObjectID: testMeID}))
		f.send(msg(0x25).b)
		// No SkillList.
		f.drain()
	})
	ctx := testContext(t)
	c := clientSelect(t, ctx, addr)
	short, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	res, err := c.EnterWorld(short)
	var we *WaitError
	if !errors.As(err, &we) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("EnterWorld = %v, want a *WaitError wrapping context.DeadlineExceeded", err)
	}
	want := []string{"UserInfo", "ValidateLocation", "ActionFailed"}
	if !slices.Equal(we.Seen, want) || !slices.Equal(res.Names(), want) {
		t.Errorf("seen %v, result %v; want %v", we.Seen, res.Names(), want)
	}
}

func TestEnterWorldUntil(t *testing.T) {
	addr := startFakeGame(t, func(f *fakeGame) {
		f.fakeSelect([]CharInfo{testChar("Hero", testMeID)})
		f.expect(mobiusEnterWorld())
		f.send(mobiusUserInfo(testUserInfoAt(Position{})))
		f.send(msg(0x25).b)
		f.send(chatLine("after"))
		f.drain()
	})
	ctx := testContext(t)
	c := clientSelect(t, ctx, addr)
	res, err := c.EnterWorldUntil(ctx, func(seen []Packet) bool {
		_, ok := Find[ActionFailed](seen)
		return ok
	})
	if err != nil || !slices.Equal(res.Names(), []string{"UserInfo", "ActionFailed"}) {
		t.Fatalf("EnterWorldUntil = %v, %v", res.Names(), err)
	}
	// What came after the end condition stays queued.
	if s, err := Expect[CreatureSay](ctx, c); err != nil || s.Text != "after" {
		t.Errorf("Expect = %+v, %v", s, err)
	}
}

func TestEnterWorldLeaveWorld(t *testing.T) {
	addr := startFakeGame(t, func(f *fakeGame) {
		f.fakeSelect([]CharInfo{testChar("Hero", testMeID)})
		f.expect(mobiusEnterWorld())
		f.send(mobiusUserInfo(testUserInfoAt(Position{})))
		f.send(msg(0x7e).b)
		f.drain()
	})
	ctx := testContext(t)
	c := clientSelect(t, ctx, addr)
	_, err := c.EnterWorld(ctx)
	if err == nil || !strings.Contains(err.Error(), "server sent LeaveWorld while entering the world (packets: UserInfo, LeaveWorld)") {
		t.Errorf("EnterWorld = %v", err)
	}
}

func TestSelectCharacterUnexpected(t *testing.T) {
	addr := startFakeGame(t, func(f *fakeGame) {
		f.fakeAuth([]CharInfo{testChar("Hero", testMeID)})
		f.expect(mobiusCharacterSelect(3))
		f.send(msg(0x26).b) // ServerClose
		f.drain()
	})
	ctx := testContext(t)
	c := clientAuth(t, ctx, addr)
	_, err := c.SelectCharacter(ctx, 3)
	var upe *UnexpectedPacketError
	if !errors.As(err, &upe) || upe.Name != "ServerClose" || upe.Want != "CharSelected" {
		t.Errorf("SelectCharacter = %v, want ServerClose as an unexpected packet", err)
	}
}

func TestRestartRefused(t *testing.T) {
	addr := startFakeGame(t, func(f *fakeGame) {
		f.fakeSelect([]CharInfo{testChar("Hero", testMeID)})
		f.expect(mobiusEnterWorld())
		f.send(mobiusValidateLocation(ValidateLocation{ObjectID: testMeID}))
		f.send(msg(0x25).b)
		f.send(mobiusSkillList(nil))
		f.expect([]byte{0x46})
		f.send(mobiusSystemMessage(1999))
		f.send(msg(0x5f).i32(0).b)
		f.send(msg(0x25).b)
		f.drain()
	})
	ctx := testContext(t)
	c := clientSelect(t, ctx, addr)
	if _, err := c.EnterWorld(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Restart(ctx); !errors.Is(err, ErrRestartRefused) {
		t.Errorf("Restart = %v, want ErrRestartRefused", err)
	}
	// The world model is still there.
	if w := c.World(); w.Me().ObjectID != testMeID {
		t.Errorf("Me after a refused restart: %+v", w.Me())
	}
}

func TestDeleteCharacterAnswers(t *testing.T) {
	hero := testChar("Hero", testMeID)
	tests := []struct {
		name   string
		answer [][]byte
		check  func(t *testing.T, res DeleteResult, err error)
	}{
		{
			"clan member",
			[][]byte{msg(0x24).i32(2).b, mobiusCharSelectionInfo([]CharInfo{hero})},
			func(t *testing.T, res DeleteResult, err error) {
				if err != nil || res.Deleted || !res.Failed || res.Reason != DeleteFailClanMember || len(res.Characters.Characters) != 1 {
					t.Errorf("got %+v, %v", res, err)
				}
			},
		},
		{
			// Mobius sends only the list when marking the character fails.
			"list only",
			[][]byte{mobiusCharSelectionInfo([]CharInfo{hero})},
			func(t *testing.T, res DeleteResult, err error) {
				if err != nil || res.Deleted || res.Failed || len(res.Characters.Characters) != 1 {
					t.Errorf("got %+v, %v", res, err)
				}
			},
		},
		{
			"no list after the answer",
			[][]byte{msg(0x23).b, msg(0x25).b},
			func(t *testing.T, _ DeleteResult, err error) {
				var upe *UnexpectedPacketError
				if !errors.As(err, &upe) || upe.Name != "ActionFailed" || !strings.Contains(upe.Want, "after the delete answer") {
					t.Errorf("got %v", err)
				}
			},
		},
		{
			"unexpected answer",
			[][]byte{msg(0x25).b},
			func(t *testing.T, _ DeleteResult, err error) {
				var upe *UnexpectedPacketError
				if !errors.As(err, &upe) || upe.Name != "ActionFailed" || !strings.Contains(upe.Want, "CharDeleteSuccess") {
					t.Errorf("got %v", err)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := startFakeGame(t, func(f *fakeGame) {
				f.fakeAuth([]CharInfo{hero})
				f.expect(msg(0x0c).i32(0).b)
				for _, b := range tt.answer {
					f.send(b)
				}
				f.drain()
			})
			ctx := testContext(t)
			c := clientAuth(t, ctx, addr)
			res, err := c.DeleteCharacter(ctx, 0)
			tt.check(t, res, err)
		})
	}
}

func TestSendOnlyRequests(t *testing.T) {
	pos := Position{10, 20, 30}
	addr := startFakeGame(t, func(f *fakeGame) {
		f.fakeSelect([]CharInfo{testChar("Hero", testMeID)})
		f.expect(mobiusEnterWorld())
		f.send(mobiusUserInfo(testUserInfoAt(pos)))
		f.send(mobiusValidateLocation(ValidateLocation{ObjectID: testMeID, X: pos.X, Y: pos.Y, Z: pos.Z}))
		f.send(msg(0x25).b)
		f.send(mobiusSkillList(nil))
		f.expect(msg(0x48).i32(11).i32(21).i32(31).i32(500).i32(0).b)
		f.expect(msg(0x36).i32(11).i32(21).i32(31).i32(500).b)
		f.expect(mobiusClientMove(Position{1, 2, 3}, pos, 0))
		f.expect(msg(0x4a).i32(16000).i32(1).b)
		f.expect(msg(0x4b).i32(16000).i32(0).b)
		f.expect([]byte{0x1c, 0, 0, 0, 0})
		f.expect([]byte{0x1d, 1, 0, 0, 0})
		f.expect(mobiusAction(0x04, testOtherID, pos, true))
		f.expect([]byte{0xaa, 0, 0, 0, 0})
		f.expect(mobiusSay2("all", ChatShout, ""))
		f.expect([]byte{0x63})
		f.expect([]byte{0x9d})
		f.expect([]byte{0x3f})
		f.expect([]byte{0xd0, 0x08, 0x00})
		f.expect([]byte{0xcd})
		f.expect([]byte{0x30})
		f.send(mobiusUserInfo(testUserInfoAt(pos)))
		f.drain()
	})
	ctx := testContext(t)
	c := clientSelect(t, ctx, addr)
	if _, err := c.EnterWorld(ctx); err != nil {
		t.Fatal(err)
	}
	near := Position{11, 21, 31}
	steps := []func() error{
		func() error { return c.ValidatePosition(ctx, near, 500) },
		func() error { return c.CannotMoveAnymore(ctx, near, 500) },
		func() error {
			return c.SendMove(ctx, MoveRequest{Target: Position{1, 2, 3}, Origin: pos, Keyboard: true})
		},
		func() error { return c.StartRotating(ctx, 16000, 1) },
		func() error { return c.FinishRotating(ctx, 16000) },
		func() error { return c.ChangeMoveType2(ctx, false) },
		func() error { return c.ChangeWaitType2(ctx, true) },
		func() error { return c.Action(ctx, testOtherID, true) },
		func() error { return c.UserCommand(ctx, UserCmdLoc) },
		func() error { return c.Say(ctx, ChatShout, "all") },
		func() error { return c.RequestQuestList(ctx) },
		func() error { return c.RequestSkillCoolTime(ctx) },
		func() error { return c.RequestSkillList(ctx) },
		func() error { return c.RequestManorList(ctx) },
		func() error { return c.RequestShowMiniMap(ctx) },
	}
	for i, step := range steps {
		if err := step(); err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}
	if u, err := c.Appearing(ctx); err != nil || u.ObjectID != testMeID {
		t.Errorf("Appearing = %+v, %v", u.ObjectID, err)
	}
}
