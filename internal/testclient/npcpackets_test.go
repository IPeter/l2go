package testclient

import (
	"errors"
	"strings"
	"testing"
)

// The Mobius server packets of Milestone 4, transcribed from their
// writeImpl line by line.

// mobiusNpcInfo is AbstractNpcInfo.NpcInfo.writeImpl.
func mobiusNpcInfo(n NpcInfo) []byte {
	w := msg(0x16)
	w.i32(n.ObjectID)
	w.i32(n.NpcTypeID) // displayId + 1000000
	w.i32(b2i(n.Attackable))
	w.i32(n.X).i32(n.Y).i32(n.Z)
	w.i32(n.Heading)
	w.i32(n.Unknown) // 0
	w.i32(n.MAtkSpd).i32(n.PAtkSpd)
	w.i32(n.RunSpd).i32(n.WalkSpd).i32(n.SwimRunSpd).i32(n.SwimWalkSpd)
	w.i32(n.FlyRunSpd).i32(n.FlyWalkSpd).i32(n.FlyRunSpd2).i32(n.FlyWalkSpd2)
	w.f64(n.MoveMultiplier).f64(n.AttackSpeedMultiplier)
	w.f64(n.CollisionRadius).f64(n.CollisionHeight)
	w.i32(n.RHand) // right hand weapon
	w.i32(n.Chest)
	w.i32(n.LHand)   // left hand weapon
	w.u8(n.ShowName) // name above char 1=true ... ??
	w.u8(b2u8(n.Running)).u8(b2u8(n.InCombat)).u8(b2u8(n.AlikeDead))
	w.u8(n.Summoned) // 2 = summoned
	w.str(n.Name)
	w.str(n.Title)
	w.i32(n.TitleColor) // 0 = client default
	w.i32(n.PvpFlag).i32(n.Karma)
	w.i32(n.AbnormalEffects)
	w.i32(n.ClanID).i32(n.ClanCrestID).i32(n.AllyID).i32(n.AllyCrestID)
	w.u8(n.Environment) // 1 water, 2 flying
	w.u8(n.Team)
	w.f64(n.CollisionRadius2).f64(n.CollisionHeight2)
	w.i32(n.EnchantEffect) // C4
	w.i32(b2i(n.Flying))   // C6
	return w.b
}

func mobiusServerObjectInfo(s ServerObjectInfo) []byte {
	w := msg(0x8c)
	w.i32(s.ObjectID)
	w.i32(s.NpcTypeID)
	w.str(s.Name)
	w.i32(b2i(s.Attackable))
	w.i32(s.X).i32(s.Y).i32(s.Z)
	w.i32(s.Heading)
	w.f64(s.MoveMultiplier)        // 1.0
	w.f64(s.AttackSpeedMultiplier) // 1.0
	w.f64(s.CollisionRadius).f64(s.CollisionHeight)
	w.i32(s.HP).i32(s.MaxHP)
	w.i32(s.ObjectType)     // 1
	w.i32(s.SpecialEffects) // 0
	return w.b
}

func mobiusMyTargetSelected(objectID int32, color uint16) []byte {
	return msg(0xa6).i32(objectID).u16(color).i32(0).b
}

func mobiusStatusUpdate(objectID int32, attrs ...StatusAttribute) []byte {
	w := msg(0x0e).i32(objectID).i32(int32(len(attrs)))
	for _, a := range attrs {
		w.i32(a.ID).i32(a.Value)
	}
	return w.b
}

func mobiusNpcHtmlMessage(objectID int32, html string) []byte {
	return msg(0x0f).i32(objectID).str(html).i32(0).b
}

func mobiusMoveToPawn(m MoveToPawn) []byte {
	return msg(0x60).i32(m.ObjectID).i32(m.TargetID).i32(m.Distance).
		i32(m.X).i32(m.Y).i32(m.Z).i32(m.TargetX).i32(m.TargetY).i32(m.TargetZ).b
}

// testGatekeeper is Roxxy, the Talking Island gatekeeper (NPC 30006), as
// Mobius shows her.
func testGatekeeper(objectID int32) NpcInfo {
	return NpcInfo{
		ObjectID: objectID, NpcTypeID: 1000000 + 30006,
		X: -84108, Y: 244604, Z: -3729, Heading: 8192,
		MAtkSpd: 333, PAtkSpd: 253, RunSpd: 120, WalkSpd: 50, SwimRunSpd: 120, SwimWalkSpd: 50,
		MoveMultiplier: 1, AttackSpeedMultiplier: 1, CollisionRadius: 8, CollisionHeight: 23,
		ShowName: 1, Name: "Roxxy", Title: "Gatekeeper",
		CollisionRadius2: 8, CollisionHeight2: 23,
	}
}

func TestNpcInfoRoundTrip(t *testing.T) {
	var n NpcInfo
	fillDistinct(&n)
	checkDecode(t, "NpcInfo", mobiusNpcInfo(n), n)

	g := testGatekeeper(0x60000001)
	checkDecode(t, "NpcInfo gatekeeper", mobiusNpcInfo(g), g)
	if g.NpcID() != 30006 || g.Position() != (Position{-84108, 244604, -3729}) {
		t.Errorf("NpcID %d, Position %v", g.NpcID(), g.Position())
	}

	var s ServerObjectInfo
	fillDistinct(&s)
	checkDecode(t, "ServerObjectInfo", mobiusServerObjectInfo(s), s)
	s.NpcTypeID = 1000000 + 31111
	if s.NpcID() != 31111 {
		t.Errorf("ServerObjectInfo NpcID %d", s.NpcID())
	}
}

func TestDecodeNpcPackets(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want Packet
	}{
		{
			// MyTargetSelected.writeImpl: d object, h colour (level
			// difference, here -3), d 0.
			"MyTargetSelected",
			mobiusMyTargetSelected(0x60000001, 0xfffd),
			MyTargetSelected{ObjectID: 0x60000001, Color: -3},
		},
		{"TargetSelected", msg(0x29).i32(0x10000002).i32(0x60000001).i32(1).i32(2).i32(3).i32(0).b,
			TargetSelected{ObjectID: 0x10000002, TargetID: 0x60000001, X: 1, Y: 2, Z: 3}},
		{"TargetUnselected", msg(0x2a).i32(0x10000002).i32(1).i32(2).i32(3).i32(0).b,
			TargetUnselected{ObjectID: 0x10000002, X: 1, Y: 2, Z: 3}},
		{"StatusUpdate", mobiusStatusUpdate(0x60000001, StatusAttribute{StatusCurHP, 80}, StatusAttribute{StatusMaxHP, 100}),
			StatusUpdate{ObjectID: 0x60000001, Attributes: []StatusAttribute{{StatusCurHP, 80}, {StatusMaxHP, 100}}}},
		{"StatusUpdate empty", mobiusStatusUpdate(0x60000001), StatusUpdate{ObjectID: 0x60000001}},
		{"NpcHtmlMessage", mobiusNpcHtmlMessage(0x60000001, "<html><body>Hello %playername%</body></html>"),
			NpcHtmlMessage{ObjectID: 0x60000001, HTML: "<html><body>Hello %playername%</body></html>"}},
		{"NpcHtmlMessage item page", msg(0x0f).i32(0).str("<html></html>").i32(57).b,
			NpcHtmlMessage{HTML: "<html></html>", ItemID: 57}},
		{"MoveToPawn", mobiusMoveToPawn(MoveToPawn{ObjectID: 0x10000001, TargetID: 0x60000001, Distance: 100, X: 1, Y: 2, Z: 3, TargetX: 4, TargetY: 5, TargetZ: 6}),
			MoveToPawn{ObjectID: 0x10000001, TargetID: 0x60000001, Distance: 100, X: 1, Y: 2, Z: 3, TargetX: 4, TargetY: 5, TargetZ: 6}},
		{"SocialAction", msg(0x2d).i32(0x10000001).i32(2).b, SocialAction{ObjectID: 0x10000001, ActionID: 2}},
		{"NpcSay", msg(0x02).i32(0x60000001).i32(int32(ChatShout)).i32(1000000 + 30006).str("Welcome!").b,
			NpcSay{ObjectID: 0x60000001, Type: ChatShout, NpcTypeID: 1000000 + 30006, Text: "Welcome!"}},
		{"ShowMiniMap", msg(0x9d).i32(1665).u8(2).b, ShowMiniMap{MapID: 1665, Period: 2}},
	}
	for _, tt := range tests {
		checkDecode(t, tt.name, tt.body, tt.want)
	}

	su := StatusUpdate{Attributes: []StatusAttribute{{StatusCurHP, 80}, {StatusMaxHP, 100}, {StatusCurHP, 70}}}
	if v, ok := su.Value(StatusCurHP); !ok || v != 70 {
		t.Errorf("Value(cur HP) = %d, %v; want the last one, 70", v, ok)
	}
	if _, ok := su.Value(StatusCurMP); ok {
		t.Error("Value(cur MP) found a value that is not there")
	}
	if id := (NpcSay{NpcTypeID: 1000000 + 30006}).NpcID(); id != 30006 {
		t.Errorf("NpcSay NpcID %d", id)
	}
}

func TestDecodeNpcPacketErrors(t *testing.T) {
	g := mobiusNpcInfo(testGatekeeper(1))
	flying := append([]byte(nil), g...)
	flying[len(flying)-4] = 2 // the last word is the flying flag
	// ServerObjectInfo: opcode, d object, d type, S "" (2 bytes), then the
	// attackable word.
	attackable := mobiusServerObjectInfo(ServerObjectInfo{})
	attackable[1+4+4+2] = 5

	tests := []struct {
		name   string
		body   []byte
		substr string
	}{
		{"NpcInfo flying flag 2", flying, "NpcInfo: flying flag is 2"},
		{"NpcInfo extra byte", append(append([]byte(nil), g...), 0), "NpcInfo: 1 bytes after the last field"},
		{"NpcInfo short", g[:len(g)-1], "NpcInfo: packets: short packet"},
		{"StatusUpdate negative count", msg(0x0e).i32(1).i32(-2).b, "StatusUpdate: attribute count -2"},
		{"StatusUpdate count too high", msg(0x0e).i32(1).i32(1).b, "StatusUpdate: packets: short packet"},
		{"NpcHtmlMessage without item", msg(0x0f).i32(1).str("x").b, "NpcHtmlMessage: packets: short packet"},
		{"ShowMiniMap without period", msg(0x9d).i32(1).b, "ShowMiniMap: packets: short packet"},
		{"ServerObjectInfo attackable 5", attackable, "ServerObjectInfo: attackable flag is 5"},
	}
	for _, tt := range tests {
		_, err := DecodeServerPacket(tt.body)
		if err == nil || !strings.Contains(err.Error(), tt.substr) {
			t.Errorf("%s: got %v, want an error containing %q", tt.name, err, tt.substr)
			continue
		}
		if !errors.Is(err, ErrMalformed) {
			t.Errorf("%s: %v does not wrap ErrMalformed", tt.name, err)
		}
	}
}
