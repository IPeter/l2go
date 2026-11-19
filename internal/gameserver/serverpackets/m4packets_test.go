package serverpackets

import (
	"encoding/binary"
	"math"
	"strings"
	"testing"
	"unicode/utf16"
)

// The golden tests of the M4 packets (NPCs, targets, dialogs). Each expected
// packet is built field by field from the Mobius writeImpl, listed above the
// test.

// m4Golden builds an expected packet like the Mobius WriteBuffer: c byte,
// h short, d int, f double, s string.
type m4Golden struct{ b []byte }

func (g *m4Golden) c(v byte) *m4Golden { g.b = append(g.b, v); return g }

func (g *m4Golden) h(v int16) *m4Golden {
	g.b = binary.LittleEndian.AppendUint16(g.b, uint16(v))
	return g
}

func (g *m4Golden) d(v int32) *m4Golden {
	g.b = binary.LittleEndian.AppendUint32(g.b, uint32(v))
	return g
}

func (g *m4Golden) f(v float64) *m4Golden {
	g.b = binary.LittleEndian.AppendUint64(g.b, math.Float64bits(v))
	return g
}

func (g *m4Golden) s(v string) *m4Golden {
	for _, u := range utf16.Encode([]rune(v)) {
		g.b = binary.LittleEndian.AppendUint16(g.b, u)
	}
	g.b = append(g.b, 0, 0)
	return g
}

// m4Gremlin is a monster with a value of its own in every NpcInfo field, so
// that a field written in the wrong place shows.
func m4Gremlin() NpcInfo {
	return NpcInfo{
		ObjectID: 0x60000001, DisplayID: 20001, Attackable: true,
		X: -84023, Y: 244581, Z: -3730, Heading: 11823,
		MAtkSpd: 333, PAtkSpd: 278,
		RunSpd: 120, WalkSpd: 60, SwimRunSpd: 50, SwimWalkSpd: 30,
		FlyRunSpd: 140, FlyWalkSpd: 70,
		MoveMultiplier: 1.25, AttackSpeedMultiplier: 1.1,
		CollisionRadius: 10.5, CollisionHeight: 13.2,
		RHand: 2369, Chest: 1146, LHand: 626,
		Running: true, InCombat: false, Dead: true, Summoned: true,
		Name: "Gremlin", Title: "Wanderer",
		AbnormalEffects: 0x0004,
		ClanID:          0x10000001, ClanCrestID: 3, AllyID: 0x10000002, AllyCrestID: 4,
		InWater: false, Flying: true,
		Team: 2, EnchantEffect: 7,
	}
}

// Mobius AbstractNpcInfo.NpcInfo.writeImpl:
//
//	c 0x16
//	d objectId, displayId + 1000000, isAttackable
//	d x, y, z, heading, 0
//	d mAtkSpd, pAtkSpd
//	d runSpd, walkSpd, swimRunSpd, swimWalkSpd, flyRunSpd, flyWalkSpd, flyRunSpd, flyWalkSpd
//	f moveMultiplier, attackSpeedMultiplier, collisionRadius, collisionHeight
//	d rhand, chest, lhand
//	c 1 (name shown), isRunning, isInCombat, isAlikeDead, summoned ? 2 : 0
//	s name, title
//	d 0 (title colour), 0 (PvP flag), 0 (karma), abnormal effects
//	d clanId, clanCrest, allyId, allyCrest
//	c water ? 1 : flying ? 2 : 0, team
//	f collisionRadius, collisionHeight
//	d enchantEffect, isFlying
func TestNpcInfoPacket(t *testing.T) {
	var g m4Golden
	g.c(0x16)
	g.d(0x60000001).d(1020001).d(1)
	g.d(-84023).d(244581).d(-3730).d(11823).d(0)
	g.d(333).d(278)
	g.d(120).d(60).d(50).d(30).d(140).d(70).d(140).d(70)
	g.f(1.25)
	g.f(1.10000002384185791015625) // the Java float 1.1f, widened
	g.f(10.5)
	g.f(13.19999980926513671875) // the Java float 13.2f, widened
	g.d(2369).d(1146).d(626)
	g.c(1).c(1).c(0).c(1).c(2)
	g.s("Gremlin").s("Wanderer")
	g.d(0).d(0).d(0).d(0x0004)
	g.d(0x10000001).d(3).d(0x10000002).d(4)
	g.c(2).c(2) // flying, red team
	g.f(10.5).f(13.19999980926513671875)
	g.d(7).d(1)
	checkPacket(t, "NpcInfo", NewNpcInfoPacket(m4Gremlin()), g.b)
}

// m4NpcInfoBytes encodes n as Mobius does. The values the builder works out
// itself are given by the test.
func m4NpcInfoBytes(n NpcInfo, summon byte, title string, effects int32, env byte) []byte {
	b8 := func(v bool) byte {
		if v {
			return 1
		}
		return 0
	}
	var g m4Golden
	g.c(0x16)
	g.d(n.ObjectID).d(n.DisplayID + 1000000).d(int32(b8(n.Attackable)))
	g.d(n.X).d(n.Y).d(n.Z).d(n.Heading).d(0)
	g.d(n.MAtkSpd).d(n.PAtkSpd)
	g.d(n.RunSpd).d(n.WalkSpd).d(n.SwimRunSpd).d(n.SwimWalkSpd)
	g.d(n.FlyRunSpd).d(n.FlyWalkSpd).d(n.FlyRunSpd).d(n.FlyWalkSpd)
	g.f(n.MoveMultiplier).f(float64(n.AttackSpeedMultiplier))
	g.f(float64(n.CollisionRadius)).f(float64(n.CollisionHeight))
	g.d(n.RHand).d(n.Chest).d(n.LHand)
	g.c(1).c(b8(n.Running)).c(b8(n.InCombat)).c(b8(n.Dead)).c(summon)
	g.s(n.Name).s(title)
	g.d(0).d(0).d(0).d(effects)
	g.d(n.ClanID).d(n.ClanCrestID).d(n.AllyID).d(n.AllyCrestID)
	g.c(env).c(n.Team)
	g.f(float64(n.CollisionRadius)).f(float64(n.CollisionHeight))
	g.d(n.EnchantEffect).d(int32(b8(n.Flying)))
	return g.b
}

// The values the builder works out: the summon code, the environment
// (water before flying), and the title and stealth effect of an invisible
// NPC.
func TestNpcInfoDerivedValues(t *testing.T) {
	const stealth = 0x00100000 // Mobius AbnormalVisualEffect.STEALTH
	tests := []struct {
		name    string
		edit    func(*NpcInfo)
		summon  byte
		title   string
		effects int32
		env     byte
	}{
		{"as is", func(*NpcInfo) {}, 2, "Wanderer", 0x0004, 2},
		{"not summoned", func(n *NpcInfo) { n.Summoned = false }, 0, "Wanderer", 0x0004, 2},
		{"on land", func(n *NpcInfo) { n.Flying = false }, 2, "Wanderer", 0x0004, 0},
		{"swimming", func(n *NpcInfo) { n.Flying, n.InWater = false, true }, 2, "Wanderer", 0x0004, 1},
		{"water before flying", func(n *NpcInfo) { n.InWater = true }, 2, "Wanderer", 0x0004, 1},
		{"invisible", func(n *NpcInfo) { n.Invisible = true }, 2, "Invisible", 0x0004 | stealth, 2},
		{"invisible, stealth set", func(n *NpcInfo) { n.Invisible, n.AbnormalEffects = true, stealth }, 2, "Invisible", stealth, 2},
		{"town NPC, client-side name", func(n *NpcInfo) {
			n.DisplayID, n.Attackable, n.Name, n.Title = 30001, false, "", ""
		}, 2, "", 0x0004, 2},
		{"non-ASCII title", func(n *NpcInfo) { n.Title = "Kapuőr" }, 2, "Kapuőr", 0x0004, 2},
	}
	for _, tt := range tests {
		n := m4Gremlin()
		tt.edit(&n)
		want := m4NpcInfoBytes(n, tt.summon, tt.title, tt.effects, tt.env)
		checkPacket(t, "NpcInfo "+tt.name, NewNpcInfoPacket(n), want)
	}
}

// Mobius ServerObjectInfo.writeImpl:
//
//	c 0x8c
//	d objectId, displayId + 1000000
//	s name
//	d isAttackable, x, y, z, heading
//	f 1.0, 1.0, collisionRadius, collisionHeight
//	d attackable ? (int) currentHp : 0, attackable ? maxHp : 0
//	d 1 (object type), 0 (special effects)
func TestServerObjectInfoPacket(t *testing.T) {
	o := ServerObjectInfo{
		ObjectID: 0x60000010, DisplayID: 31688, Name: "Statue",
		Attackable: true, X: 1, Y: -2, Z: 3, Heading: 4,
		CollisionRadius: 8.5, CollisionHeight: 22.25,
		CurHP: 99.9, MaxHP: 120,
	}
	var g m4Golden
	g.c(0x8c).d(0x60000010).d(1031688).s("Statue")
	g.d(1).d(1).d(-2).d(3).d(4)
	g.f(1).f(1).f(8.5).f(22.25)
	g.d(99).d(120) // the current HP cut to an int
	g.d(1).d(0)
	checkPacket(t, "attackable", NewServerObjectInfoPacket(o), g.b)

	o.Attackable = false
	g = m4Golden{}
	g.c(0x8c).d(0x60000010).d(1031688).s("Statue")
	g.d(0).d(1).d(-2).d(3).d(4)
	g.f(1).f(1).f(8.5).f(22.25)
	g.d(0).d(0) // no HP for an object that cannot be attacked
	g.d(1).d(0)
	checkPacket(t, "not attackable", NewServerObjectInfoPacket(o), g.b)
}

// Mobius MyTargetSelected.writeImpl: c 0xa6; d objectId; h color; d 0.
func TestMyTargetSelectedPacket(t *testing.T) {
	checkPacket(t, "weaker target",
		NewMyTargetSelectedPacket(MyTargetSelected{ObjectID: 0x60000001, Color: 5}),
		unhex(t, "a6 01000060 0500 00000000"))
	checkPacket(t, "stronger target",
		NewMyTargetSelectedPacket(MyTargetSelected{ObjectID: 0x60000001, Color: -12}),
		unhex(t, "a6 01000060 f4ff 00000000"))
	checkPacket(t, "not attackable",
		NewMyTargetSelectedPacket(MyTargetSelected{ObjectID: 7}),
		unhex(t, "a6 07000000 0000 00000000"))
}

// Mobius TargetSelected.writeImpl: c 0x29; d objectId, targetId, x, y, z, 0.
func TestTargetSelectedPacket(t *testing.T) {
	var g m4Golden
	g.c(0x29).d(0x10000001).d(0x60000001).d(-84023).d(244581).d(-3730).d(0)
	got := NewTargetSelectedPacket(TargetSelected{
		ObjectID: 0x10000001, TargetID: 0x60000001, X: -84023, Y: 244581, Z: -3730,
	})
	checkPacket(t, "TargetSelected", got, g.b)
}

// Mobius TargetUnselected.writeImpl: c 0x2a; d objectId, x, y, z, 0.
func TestTargetUnselectedPacket(t *testing.T) {
	var g m4Golden
	g.c(0x2a).d(0x10000001).d(-84023).d(244581).d(-3730).d(0)
	got := NewTargetUnselectedPacket(TargetUnselected{ObjectID: 0x10000001, X: -84023, Y: 244581, Z: -3730})
	checkPacket(t, "TargetUnselected", got, g.b)
}

// Mobius StatusUpdate.writeImpl: c 0x0e; d objectId; d count; per value
// d id, d value.
func TestStatusUpdatePacket(t *testing.T) {
	hpBar := StatusUpdate{
		ObjectID: 0x60000001,
		Values: []StatusValue{
			{Attr: StatusMaxHP, Value: 39},
			{Attr: StatusCurHP, Value: 38},
		},
	}
	checkPacket(t, "HP bar", NewStatusUpdatePacket(hpBar), unhex(t, `
		0e
		01000060
		02000000
		0a000000 27000000
		09000000 26000000`))
	checkPacket(t, "no values", NewStatusUpdatePacket(StatusUpdate{ObjectID: 5}),
		unhex(t, "0e 05000000 00000000"))
}

// The attribute IDs are those of Mobius StatusUpdate; Exp and SP are the
// two IDs it leaves unnamed.
func TestStatusAttrIDs(t *testing.T) {
	tests := []struct {
		mobius string
		attr   StatusAttr
		id     int32
	}{
		{"LEVEL", StatusLevel, 0x01},
		{"exp", StatusExp, 0x02},
		{"STR", StatusSTR, 0x03},
		{"DEX", StatusDEX, 0x04},
		{"CON", StatusCON, 0x05},
		{"INT", StatusINT, 0x06},
		{"WIT", StatusWIT, 0x07},
		{"MEN", StatusMEN, 0x08},
		{"CUR_HP", StatusCurHP, 0x09},
		{"MAX_HP", StatusMaxHP, 0x0a},
		{"CUR_MP", StatusCurMP, 0x0b},
		{"MAX_MP", StatusMaxMP, 0x0c},
		{"sp", StatusSP, 0x0d},
		{"CUR_LOAD", StatusCurLoad, 0x0e},
		{"MAX_LOAD", StatusMaxLoad, 0x0f},
		{"P_ATK", StatusPAtk, 0x11},
		{"ATK_SPD", StatusAtkSpd, 0x12},
		{"P_DEF", StatusPDef, 0x13},
		{"EVASION", StatusEvasion, 0x14},
		{"ACCURACY", StatusAccuracy, 0x15},
		{"CRITICAL", StatusCritical, 0x16},
		{"M_ATK", StatusMAtk, 0x17},
		{"CAST_SPD", StatusCastSpd, 0x18},
		{"M_DEF", StatusMDef, 0x19},
		{"PVP_FLAG", StatusPvPFlag, 0x1a},
		{"KARMA", StatusKarma, 0x1b},
		{"CUR_CP", StatusCurCP, 0x21},
		{"MAX_CP", StatusMaxCP, 0x22},
	}
	for _, tt := range tests {
		if int32(tt.attr) != tt.id {
			t.Errorf("%s = %#x, want %#x", tt.mobius, int32(tt.attr), tt.id)
		}
	}
}

// Mobius NpcHtmlMessage.writeImpl: c 0x0f; d npcObjId; s html; d itemId.
func TestNpcHtmlMessagePacket(t *testing.T) {
	const page = `<html><body>Gatekeeper Roxxy:<br><a action="bypass -h npc_1610612737_Chat 1">Teleport</a></body></html>`
	var g m4Golden
	g.c(0x0f).d(1610612737).s(page).d(0)
	checkPacket(t, "NPC page",
		NewNpcHtmlMessagePacket(NpcHtmlMessage{NpcObjectID: 1610612737, HTML: page}), g.b)

	const errPage = "<html><body>Bypass error</body></html>"
	g = m4Golden{}
	g.c(0x0f).d(0).s(errPage).d(1)
	checkPacket(t, "item page",
		NewNpcHtmlMessagePacket(NpcHtmlMessage{HTML: errPage, ItemID: 1}), g.b)

	checkPacket(t, "no page",
		NewNpcHtmlMessagePacket(NpcHtmlMessage{NpcObjectID: 3}),
		unhex(t, "0f 03000000 0000 00000000"))
}

// m4Pair is a character outside the Basic Multilingual Plane: two UTF-16
// code units.
const m4Pair = "\U0001D11E"

// Mobius AbstractHtmlPacket.setHtml: a page over 8,192 Java chars is cut
// and not wrapped; a page without "<html" that does not start with ..\L2 is
// wrapped in <html><body>.
func TestPrepareHTML(t *testing.T) {
	long := strings.Repeat("a", MaxHTMLLength+10)
	atLimit := strings.Repeat("b", MaxHTMLLength)
	tests := []struct {
		name, in, want string
	}{
		{"page", "<html><body>Hi</body></html>", "<html><body>Hi</body></html>"},
		{"text", "Hi", "<html><body>Hi</body></html>"},
		{"empty", "", "<html><body></body></html>"},
		{"<html anywhere", "<title>x</title><html>y</html>", "<title>x</title><html>y</html>"},
		{"<HTML is not found", "<HTML><BODY>Hi</BODY></HTML>", "<html><body><HTML><BODY>Hi</BODY></HTML></body></html>"},
		{"client page", `..\L2text\help.htm`, `..\L2text\help.htm`},
		{"too long: cut, not wrapped", long, long[:MaxHTMLLength]},
		{"at the limit: kept and wrapped", atLimit, "<html><body>" + atLimit + "</body></html>"},
		{"counted in UTF-16 units, not bytes",
			"<html>" + strings.Repeat("é", MaxHTMLLength),
			"<html>" + strings.Repeat("é", MaxHTMLLength-len("<html>"))},
		{"pairs count twice",
			strings.Repeat(m4Pair, MaxHTMLLength/2+1),
			strings.Repeat(m4Pair, MaxHTMLLength/2)},
		{"a pair across the limit is left out",
			strings.Repeat("a", MaxHTMLLength-1) + m4Pair + "a",
			strings.Repeat("a", MaxHTMLLength-1)},
	}
	for _, tt := range tests {
		if got := PrepareHTML(tt.in); got != tt.want {
			t.Errorf("%s: PrepareHTML: got %d bytes %.40q..., want %d bytes %.40q...",
				tt.name, len(got), got, len(tt.want), tt.want)
		}
	}
}

// Mobius MoveToPawn.writeImpl: c 0x60; d objectId, targetId, distance; d x,
// y, z; d target x, y, z.
func TestMoveToPawnPacket(t *testing.T) {
	var g m4Golden
	g.c(0x60).d(0x10000001).d(0x60000001).d(36)
	g.d(-84100).d(244600).d(-3728)
	g.d(-84023).d(244581).d(-3730)
	got := NewMoveToPawnPacket(MoveToPawn{
		ObjectID: 0x10000001, TargetID: 0x60000001, Distance: 36,
		X: -84100, Y: 244600, Z: -3728,
		TargetX: -84023, TargetY: 244581, TargetZ: -3730,
	})
	checkPacket(t, "MoveToPawn", got, g.b)
}

// Mobius SocialAction.writeImpl: c 0x2d; d objectId, actionId.
func TestSocialActionPacket(t *testing.T) {
	checkPacket(t, "greeting",
		NewSocialActionPacket(SocialAction{ObjectID: 0x10000001, ActionID: 2}),
		unhex(t, "2d 01000010 02000000"))
	checkPacket(t, "level up",
		NewSocialActionPacket(SocialAction{ObjectID: 0x10000001, ActionID: SocialActionLevelUp}),
		unhex(t, "2d 01000010 0f000000"))
}

// Mobius NpcSay.writeImpl: c 0x02; d objectId; d chat type; d npcId +
// 1000000; s text.
func TestNpcSayPacket(t *testing.T) {
	var g m4Golden
	g.c(0x02).d(0x60000001).d(1).d(1030001).s("Hey, you!")
	checkPacket(t, "shout",
		NewNpcSayPacket(NpcSay{ObjectID: 0x60000001, Type: ChatShout, NpcID: 30001, Text: "Hey, you!"}), g.b)

	g = m4Golden{}
	g.c(0x02).d(0x60000002).d(0).d(1030006).s("Üdv, vándor!")
	checkPacket(t, "general",
		NewNpcSayPacket(NpcSay{ObjectID: 0x60000002, Type: ChatGeneral, NpcID: 30006, Text: "Üdv, vándor!"}), g.b)
}

// Mobius ShowMiniMap.writeImpl: c 0x9d; d mapId; c Seven Signs period.
// RequestShowMiniMap answers with map 1665.
func TestShowMiniMapPacket(t *testing.T) {
	checkPacket(t, "ShowMiniMap",
		NewShowMiniMapPacket(ShowMiniMap{MapID: MiniMapID, Period: SevenSignsCompetition}),
		unhex(t, "9d 81060000 01"))
	checkPacket(t, "seal validation",
		NewShowMiniMapPacket(ShowMiniMap{MapID: MiniMapID, Period: SevenSignsSealValidation}),
		unhex(t, "9d 81060000 03"))
}
