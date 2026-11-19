package testclient

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// fillDistinct gives every number of the struct v points to a different
// value, sets the bools alternately and names the strings after their
// field, so that a field read from the wrong place shows up in a round
// trip. Slices, pointers and interfaces stay as they are.
func fillDistinct(v any) {
	n := 0
	flag := false
	var fill func(f reflect.Value, name string)
	fill = func(f reflect.Value, name string) {
		switch f.Kind() {
		case reflect.Struct:
			for i := range f.NumField() {
				fill(f.Field(i), f.Type().Field(i).Name)
			}
		case reflect.Array:
			for i := range f.Len() {
				fill(f.Index(i), fmt.Sprintf("%s%d", name, i))
			}
		case reflect.Bool:
			flag = !flag
			f.SetBool(flag)
		case reflect.String:
			n++
			f.SetString(fmt.Sprintf("%s-%d", name, n))
		case reflect.Int8, reflect.Uint8:
			n++
			setNumber(f, int64(n%100+1))
		case reflect.Int16, reflect.Uint16:
			n++
			setNumber(f, int64(n%30000+1))
		case reflect.Int32, reflect.Int64:
			n++
			f.SetInt(int64(n) * 1001)
		case reflect.Float64:
			n++
			f.SetFloat(float64(n) + 0.25)
		}
	}
	fill(reflect.ValueOf(v).Elem(), "")
}

func setNumber(f reflect.Value, v int64) {
	if f.CanInt() {
		f.SetInt(v)
		return
	}
	f.SetUint(uint64(v))
}

// fieldDiff lists the fields in which two values of the same struct type
// differ.
func fieldDiff(got, want any) []string {
	gv, wv := reflect.ValueOf(got), reflect.ValueOf(want)
	if !gv.IsValid() || !wv.IsValid() || gv.Type() != wv.Type() {
		return []string{fmt.Sprintf("got %T, want %T", got, want)}
	}
	if gv.Kind() != reflect.Struct {
		if !reflect.DeepEqual(got, want) {
			return []string{fmt.Sprintf("got %#v, want %#v", got, want)}
		}
		return nil
	}
	var diffs []string
	for i := range gv.NumField() {
		g, w := gv.Field(i).Interface(), wv.Field(i).Interface()
		if !reflect.DeepEqual(g, w) {
			diffs = append(diffs, fmt.Sprintf("%s = %#v, want %#v", gv.Type().Field(i).Name, g, w))
		}
	}
	return diffs
}

// checkDecode decodes body and compares it with want field by field.
func checkDecode(t *testing.T, name string, body []byte, want Packet) {
	t.Helper()
	got, err := DecodeServerPacket(body)
	if err != nil {
		t.Errorf("%s: %v", name, err)
		return
	}
	for _, d := range fieldDiff(got, want) {
		t.Errorf("%s: %s", name, d)
	}
	if got != nil && want != nil && got.PacketName() != want.PacketName() {
		t.Errorf("%s: PacketName %q, want %q", name, got.PacketName(), want.PacketName())
	}
}

// The Mobius server packets of Milestone 3, transcribed from their
// writeImpl line by line.

func mobiusCharSelected(p CharSelected) []byte {
	w := msg(0x15)
	w.str(p.Name)
	w.i32(p.ObjectID)
	w.str(p.Title)
	w.i32(p.SessionID)
	w.i32(p.ClanID)
	w.i32(p.Unknown) // ??
	w.i32(p.Sex)
	w.i32(p.Race)
	w.i32(p.ClassID)
	w.i32(p.Active) // active ??
	w.i32(p.X).i32(p.Y).i32(p.Z)
	w.f64(p.CurrentHP)
	w.f64(p.CurrentMP)
	w.i32(p.SP)
	w.i64(p.Exp)
	w.i32(p.Level)
	w.i32(p.Karma)
	w.i32(p.PkKills)
	w.i32(p.INT).i32(p.STR).i32(p.CON).i32(p.MEN).i32(p.DEX).i32(p.WIT)
	for _, v := range p.Reserved { // 30 zeros
		w.i32(v)
	}
	w.i32(p.Reserved2[0]).i32(p.Reserved2[1])
	w.i32(p.GameTime) // game time % (24 * 60)
	w.i32(p.Reserved3)
	w.i32(p.ClassID2)
	for _, v := range p.Reserved4 { // 12 zeros
		w.i32(v)
	}
	return w.b
}

func mobiusUserInfo(u UserInfo) []byte {
	w := msg(0x04)
	w.i32(u.X).i32(u.Y).i32(u.Z)
	w.i32(u.VehicleID)
	w.i32(u.ObjectID)
	w.str(u.Name)
	w.i32(u.Race)
	w.i32(u.Sex)
	w.i32(u.BaseClassID)
	w.i32(u.Level)
	w.i64(u.Exp)
	w.i32(u.STR).i32(u.DEX).i32(u.CON).i32(u.INT).i32(u.WIT).i32(u.MEN)
	w.i32(u.MaxHP).i32(u.HP).i32(u.MaxMP).i32(u.MP)
	w.i32(u.SP)
	w.i32(u.CurLoad).i32(u.MaxLoad)
	w.i32(u.WeaponFlag) // 20 no weapon, 40 weapon equipped
	// UNDER REAR LEAR NECK RFINGER LFINGER HEAD RHAND LHAND GLOVES CHEST
	// LEGS FEET CLOAK RHAND HAIR FACE: object IDs, then item IDs.
	for _, v := range u.PaperdollObjectIDs {
		w.i32(v)
	}
	for _, v := range u.PaperdollItemIDs {
		w.i32(v)
	}
	for _, v := range u.Reserved1 { // c6 new h's: 14 shorts
		w.u16(v)
	}
	w.i32(u.AugmentationID)
	for _, v := range u.Reserved2 { // 12 shorts
		w.u16(v)
	}
	w.i32(u.AugmentationID2)
	for _, v := range u.Reserved3 { // 4 shorts
		w.u16(v)
	}
	w.i32(u.PAtk).i32(u.PAtkSpd).i32(u.PDef).i32(u.Evasion).i32(u.Accuracy).i32(u.Critical)
	w.i32(u.MAtk).i32(u.CastSpd).i32(u.PAtkSpd2).i32(u.MDef)
	w.i32(u.PvpFlag).i32(u.Karma)
	w.i32(u.RunSpd).i32(u.WalkSpd).i32(u.SwimRunSpd).i32(u.SwimWalkSpd)
	w.i32(u.FlyRunSpd).i32(u.FlyWalkSpd).i32(u.FlyRunSpd2).i32(u.FlyWalkSpd2)
	w.f64(u.MoveMultiplier).f64(u.AttackSpeedMultiplier)
	w.f64(u.CollisionRadius).f64(u.CollisionHeight)
	w.i32(u.HairStyle).i32(u.HairColor).i32(u.Face)
	w.i32(b2i(u.GM)) // builder level
	w.str(u.Title)
	w.i32(u.ClanID).i32(u.ClanCrestID).i32(u.AllyID).i32(u.AllyCrestID)
	w.i32(u.Relation)
	w.u8(u.MountType).u8(u.PrivateStoreType).u8(b2u8(u.DwarvenCraft))
	w.i32(u.PkKills).i32(u.PvpKills)
	w.u16(uint16(len(u.Cubics)))
	for _, id := range u.Cubics {
		w.u16(id)
	}
	w.u8(b2u8(u.PartyMatchRoom))
	w.i32(u.AbnormalEffects)
	w.u8(b2u8(u.InWater))
	w.i32(u.ClanPrivileges)
	w.u16(u.RecomLeft).u16(u.RecomHave)
	w.i32(u.MountNpcID)
	w.u16(u.InventoryLimit)
	w.i32(u.ClassID)
	w.i32(u.SpecialEffects) // special effects? circles around player...
	w.i32(u.MaxCP).i32(u.CP)
	w.u8(u.EnchantEffect).u8(u.Team)
	w.i32(u.ClanCrestLargeID)
	w.u8(b2u8(u.Noble)).u8(b2u8(u.Hero))
	w.u8(b2u8(u.Fishing)).i32(u.FishX).i32(u.FishY).i32(u.FishZ)
	w.i32(u.NameColor)
	w.u8(b2u8(u.Running)) // new c5
	w.i32(u.PledgeClass).i32(u.PledgeType)
	w.i32(u.TitleColor)
	w.i32(u.CursedWeaponLevel)
	return w.b
}

func mobiusCharInfo(c CharInfoPacket) []byte {
	w := msg(0x03)
	w.i32(c.X).i32(c.Y).i32(c.Z)
	w.i32(c.VehicleID)
	w.i32(c.ObjectID)
	w.str(c.Name)
	w.i32(c.Race).i32(c.Sex).i32(c.BaseClassID)
	// UNDER HEAD RHAND LHAND GLOVES CHEST LEGS FEET CLOAK RHAND HAIR HAIR2
	for _, v := range c.ItemIDs {
		w.i32(v)
	}
	for _, v := range c.Reserved1 { // 4 shorts
		w.u16(v)
	}
	w.i32(c.AugmentationID)
	for _, v := range c.Reserved2 { // 12 shorts
		w.u16(v)
	}
	w.i32(c.AugmentationID2)
	for _, v := range c.Reserved3 { // 4 shorts
		w.u16(v)
	}
	w.i32(c.PvpFlag).i32(c.Karma)
	w.i32(c.MAtkSpd).i32(c.PAtkSpd)
	w.i32(c.PvpFlag2).i32(c.Karma2)
	w.i32(c.RunSpd).i32(c.WalkSpd).i32(c.SwimRunSpd).i32(c.SwimWalkSpd)
	w.i32(c.FlyRunSpd).i32(c.FlyWalkSpd).i32(c.FlyRunSpd2).i32(c.FlyWalkSpd2)
	w.f64(c.MoveMultiplier).f64(c.AttackSpeedMultiplier)
	w.f64(c.CollisionRadius).f64(c.CollisionHeight)
	w.i32(c.HairStyle).i32(c.HairColor).i32(c.Face)
	w.str(c.Title)
	w.i32(c.ClanID).i32(c.ClanCrestID).i32(c.AllyID).i32(c.AllyCrestID)
	w.i32(c.Relation) // found nothing??
	w.u8(b2u8(c.Standing)).u8(b2u8(c.Running)).u8(b2u8(c.InCombat))
	w.u8(b2u8(c.AlikeDead)).u8(b2u8(c.Invisible))
	w.u8(c.MountType).u8(c.PrivateStoreType)
	w.u16(uint16(len(c.Cubics)))
	for _, id := range c.Cubics {
		w.u16(id)
	}
	w.u8(b2u8(c.PartyMatchRoom))
	w.i32(c.AbnormalEffects)
	w.u8(c.RecomLeft)
	w.u16(c.RecomHave)
	w.i32(c.ClassID)
	w.i32(c.MaxCP).i32(c.CP)
	w.u8(c.EnchantEffect).u8(c.Team)
	w.i32(c.ClanCrestLargeID)
	w.u8(b2u8(c.Noble)).u8(b2u8(c.Hero))
	w.u8(b2u8(c.Fishing)).i32(c.FishX).i32(c.FishY).i32(c.FishZ)
	w.i32(c.NameColor)
	w.i32(c.Heading)
	w.i32(c.PledgeClass).i32(c.PledgeType)
	w.i32(c.TitleColor)
	w.i32(c.CursedWeaponLevel)
	return w.b
}

func mobiusRelationChanged(r RelationChanged) []byte {
	return msg(0xce).i32(r.ObjectID).i32(r.Relation).i32(b2i(r.AutoAttackable)).i32(r.Karma).i32(r.PvpFlag).b
}

// writeMobiusItem is AbstractItemPacket.writeItem.
func writeMobiusItem(w *wire, it Item) {
	w.u16(it.Type1)
	w.i32(it.ObjectID)
	w.i32(it.ItemID)
	w.i32(it.Count)
	w.u16(it.Type2)
	w.u16(it.CustomType1)
	w.u16(uint16(b2i(it.Equipped)))
	w.i32(it.BodyPart)
	w.u16(it.Enchant)
	w.u16(it.CustomType2)
	w.i32(it.AugmentationID)
	w.i32(it.Mana)
}

func mobiusItemList(l ItemList) []byte {
	w := msg(0x1b).u16(uint16(b2i(l.ShowWindow))).u16(uint16(len(l.Items)))
	for _, it := range l.Items {
		writeMobiusItem(w, it)
	}
	return w.b
}

func mobiusInventoryUpdate(u InventoryUpdate) []byte {
	w := msg(0x27).u16(uint16(len(u.Items)))
	for _, ch := range u.Items {
		w.u16(ch.Change)
		writeMobiusItem(w, ch.Item)
	}
	return w.b
}

func mobiusSkillList(skills []Skill) []byte {
	w := msg(0x58).i32(int32(len(skills)))
	for _, s := range skills {
		w.i32(b2i(s.Passive)).i32(s.Level).i32(s.ID).u8(b2u8(s.Disabled))
	}
	return w.b
}

func mobiusValidateLocation(v ValidateLocation) []byte {
	return msg(0x61).i32(v.ObjectID).i32(v.X).i32(v.Y).i32(v.Z).i32(v.Heading).b
}

func mobiusMoveToLocation(m MoveToLocation) []byte {
	return msg(0x01).i32(m.ObjectID).i32(m.DestX).i32(m.DestY).i32(m.DestZ).i32(m.X).i32(m.Y).i32(m.Z).b
}

func mobiusCreatureSay(s CreatureSay) []byte {
	return msg(0x4a).i32(s.ObjectID).i32(int32(s.Type)).str(s.Name).str(s.Text).b
}

func mobiusSystemMessage(id int32, texts ...string) []byte {
	w := msg(0x64).i32(id).i32(int32(len(texts)))
	for _, s := range texts {
		w.i32(0).str(s)
	}
	return w.b
}

// testItems is the starting inventory of a Human Fighter, with adena.
func testItems() []Item {
	return []Item{
		{Type1: 1, ObjectID: 0x10000011, ItemID: 1146, Count: 1, Type2: 1, Equipped: true, BodyPart: 0x0400, Mana: -1},
		{Type1: 1, ObjectID: 0x10000012, ItemID: 1147, Count: 1, Type2: 1, Equipped: true, BodyPart: 0x0800, Mana: -1},
		{Type1: 0, ObjectID: 0x10000013, ItemID: 2369, Count: 1, Type2: 0, Equipped: true, BodyPart: 0x4000, Mana: -1},
		{Type1: 4, ObjectID: 0x10000014, ItemID: 57, Count: 5000, Type2: 4, Mana: -1},
	}
}

func TestBigWorldPacketsRoundTrip(t *testing.T) {
	var sel CharSelected
	fillDistinct(&sel)
	var ui UserInfo
	fillDistinct(&ui)
	ui.Cubics = []uint16{3, 9}
	var ci CharInfoPacket
	fillDistinct(&ci)
	ci.Cubics = []uint16{4}

	checkDecode(t, "CharSelected", mobiusCharSelected(sel), sel)
	checkDecode(t, "UserInfo", mobiusUserInfo(ui), ui)
	checkDecode(t, "CharInfo", mobiusCharInfo(ci), ci)
}

// TestPacketSizes checks the test builders against body sizes counted by
// hand from the Mobius writeImpl, with empty strings and no cubics, and
// that such zero packets decode to the zero value.
func TestPacketSizes(t *testing.T) {
	tests := []struct {
		body []byte
		zero Packet
		size int
	}{
		{mobiusCharSelected(CharSelected{}), CharSelected{}, 300},
		{mobiusUserInfo(UserInfo{}), UserInfo{}, 547},
		{mobiusCharInfo(CharInfoPacket{}), CharInfoPacket{}, 326},
		{mobiusNpcInfo(NpcInfo{}), NpcInfo{}, 183},
		{mobiusServerObjectInfo(ServerObjectInfo{}), ServerObjectInfo{}, 78},
	}
	for _, tt := range tests {
		name := tt.zero.PacketName()
		if got := len(tt.body) - 1; got != tt.size {
			t.Errorf("%s body is %d bytes, Mobius writes %d", name, got, tt.size)
		}
		checkDecode(t, name, tt.body, tt.zero)
	}
}

func TestDecodeWorldPackets(t *testing.T) {
	items := testItems()
	tests := []struct {
		name string
		body []byte
		want Packet
	}{
		{"RelationChanged", mobiusRelationChanged(RelationChanged{ObjectID: 0x10000002, Relation: 0x0480, AutoAttackable: true, Karma: 5, PvpFlag: 1}),
			RelationChanged{ObjectID: 0x10000002, Relation: 0x0480, AutoAttackable: true, Karma: 5, PvpFlag: 1}},
		{"ItemList", mobiusItemList(ItemList{Items: items}), ItemList{Items: items}},
		{"ItemList shown", mobiusItemList(ItemList{ShowWindow: true}), ItemList{ShowWindow: true}},
		{"InventoryUpdate",
			mobiusInventoryUpdate(InventoryUpdate{Items: []ItemChange{{Change: ItemModified, Item: items[3]}, {Change: ItemRemoved, Item: items[0]}}}),
			InventoryUpdate{Items: []ItemChange{{Change: ItemModified, Item: items[3]}, {Change: ItemRemoved, Item: items[0]}}}},
		{"ShortcutInit empty", msg(0x45).i32(0).b, ShortcutInit{}},
		{
			// ShortcutInit.writeImpl per type: item d id, d 1, d -1, d 0,
			// d 0, h 0, h 0; skill d id, d level, c 0, d 1; action,
			// macro and recipe d id, d 1.
			"ShortcutInit",
			msg(0x45).i32(4).
				i32(1).i32(3).i32(0x10000011).i32(1).i32(-1).i32(0).i32(0).u16(0).u16(0).
				i32(2).i32(13).i32(3).i32(1).u8(0).i32(1).
				i32(3).i32(14).i32(0).i32(1).
				i32(4).i32(15).i32(1000).i32(1).b,
			ShortcutInit{Shortcuts: []Shortcut{
				{Type: ShortcutItem, Slot: 3, ID: 0x10000011, Level: 1, Rest: []byte{0xff, 0xff, 0xff, 0xff, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
				{Type: ShortcutSkill, Slot: 13, ID: 3, Level: 1, Rest: []byte{0, 1, 0, 0, 0}},
				{Type: ShortcutAction, Slot: 14, ID: 0, Level: 1},
				{Type: ShortcutMacro, Slot: 15, ID: 1000, Level: 1},
			}},
		},
		{"QuestList empty", msg(0x80).u16(0).b, QuestList{}},
		{"QuestList", msg(0x80).u16(2).i32(1).i32(3).i32(255).i32(-2147483647).b,
			QuestList{Quests: []Quest{{ID: 1, State: 3}, {ID: 255, State: -2147483647}}}},
		{"SkillList", mobiusSkillList([]Skill{{Level: 1, ID: 3}, {Passive: true, Level: 2, ID: 194, Disabled: true}}),
			SkillList{Skills: []Skill{{Level: 1, ID: 3}, {Passive: true, Level: 2, ID: 194, Disabled: true}}}},
		{"SendMacroList empty", msg(0xe7).i32(1).u8(0).u8(0).u8(0).b, SendMacroList{Revision: 1}},
		{
			// SendMacroList.writeImpl: d rev, c 0, c count, c 1, d id,
			// S name, S descr, S acronym, c icon, c count, then per
			// command c index, c type, d d1, c d2, S command.
			"SendMacroList",
			msg(0xe7).i32(2).u8(0).u8(1).u8(1).i32(1000).str("Buff").str("all buffs").str("BF").u8(3).u8(2).
				u8(1).u8(1).i32(1204).u8(0).str("").
				u8(2).u8(3).i32(0).u8(5).str("/sit").b,
			SendMacroList{Revision: 2, Count: 1, Macro: &Macro{
				ID: 1000, Name: "Buff", Description: "all buffs", Acronym: "BF", Icon: 3,
				Commands: []MacroCommand{
					{Index: 1, Type: 1, D1: 1204},
					{Index: 2, Type: 3, D2: 5, Command: "/sit"},
				},
			}},
		},
		{
			// HennaInfo.writeImpl: c INT, c STR, c CON, c MEN, c DEX, c WIT,
			// d 3, d count, then d dye, d 1.
			"HennaInfo",
			msg(0xe4).u8(1).u8(0xff).u8(2).u8(0).u8(0xfe).u8(3).i32(3).i32(1).i32(4).i32(1).b,
			HennaInfo{INT: 1, STR: -1, CON: 2, MEN: 0, DEX: -2, WIT: 3, Slots: 3, Hennas: []Henna{{DyeID: 4, Unknown: 1}}},
		},
		{"FriendList", msg(0xfa).i32(2).i32(0x10000005).str("Pal").i32(1).i32(0x10000005).i32(0x10000006).str("Away").i32(0).i32(0).b,
			FriendList{Friends: []Friend{
				{ObjectID: 0x10000005, Name: "Pal", Online: true, OnlineObjectID: 0x10000005},
				{ObjectID: 0x10000006, Name: "Away"},
			}}},
		{"SkillCoolTime", msg(0xc1).i32(1).i32(1177).i32(5).i32(30).i32(12).b,
			SkillCoolTime{Skills: []SkillCooldown{{SkillID: 1177, Level: 5, Reuse: 30, Remaining: 12}}}},
		{"AbnormalStatusUpdate", msg(0x7f).u16(1).i32(1040).u16(3).i32(1200).b,
			AbnormalStatusUpdate{Effects: []Effect{{SkillID: 1040, Level: 3, Duration: 1200}}}},
		{"ExStorageMaxCount", msg(0xfe).u16(0x2e).i32(80).i32(100).i32(150).i32(4).i32(3).i32(0).i32(30).i32(2).b,
			ExStorageMaxCount{Inventory: 80, Warehouse: 100, ClanWarehouse: 150, PrivateSell: 4, PrivateBuy: 3, CommonRecipe: 30, InventoryExtra: 2}},
		{"EtcStatusUpdate", msg(0xf3).i32(2).i32(1).i32(1).i32(0).i32(1).i32(0).i32(4).b,
			EtcStatusUpdate{Charges: 2, WeightPenalty: 1, ChatBlocked: true, GradePenalty: true, DeathPenalty: 4}},
		{"ValidateLocation", mobiusValidateLocation(ValidateLocation{ObjectID: 0x10000001, X: -71338, Y: 258271, Z: -3104, Heading: 32768}),
			ValidateLocation{ObjectID: 0x10000001, X: -71338, Y: 258271, Z: -3104, Heading: 32768}},
		{"ActionFailed", msg(0x25).b, ActionFailed{}},
		{"StopMove", msg(0x47).i32(0x10000001).i32(10).i32(20).i32(-30).i32(16384).b,
			StopMove{ObjectID: 0x10000001, X: 10, Y: 20, Z: -30, Heading: 16384}},
		{"DeleteObject", msg(0x12).i32(0x10000002).i32(1).b, DeleteObject{ObjectID: 0x10000002, Disappear: 1}},
		{"TeleportToLocation", msg(0x28).i32(0x10000001).i32(1000).i32(2000).i32(-3000).i32(0).i32(100).b,
			TeleportToLocation{ObjectID: 0x10000001, X: 1000, Y: 2000, Z: -3000, Heading: 100}},
		{"ChangeMoveType", msg(0x2e).i32(0x10000001).i32(1).i32(0).b, ChangeMoveType{ObjectID: 0x10000001, Running: true}},
		{"ChangeWaitType", msg(0x2f).i32(0x10000001).i32(WaitSitting).i32(1).i32(2).i32(3).b,
			ChangeWaitType{ObjectID: 0x10000001, WaitType: WaitSitting, X: 1, Y: 2, Z: 3}},
		{"StartRotation", msg(0x62).i32(0x10000001).i32(16000).i32(1).i32(0).b,
			StartRotation{ObjectID: 0x10000001, Degree: 16000, Side: 1}},
		{"StopRotation", msg(0x63).i32(0x10000001).i32(16000).i32(0).u8(0).b,
			StopRotation{ObjectID: 0x10000001, Degree: 16000}},
		{"CreatureSay", mobiusCreatureSay(CreatureSay{ObjectID: 0x10000001, Type: ChatShout, Name: "Hero", Text: "hello world"}),
			CreatureSay{ObjectID: 0x10000001, Type: ChatShout, Name: "Hero", Text: "hello world"}},
		{"CreatureSay boat", msg(0x4a).i32(0).i32(int32(ChatBoat)).i32(801).i32(1478).b,
			CreatureSay{Type: ChatBoat, NameID: 801, MessageID: 1478}},
		{"SystemMessage", mobiusSystemMessage(SysMsgS1IsNotCurrentlyLoggedIn, "Nobody"),
			SystemMessage{ID: SysMsgS1IsNotCurrentlyLoggedIn, Params: []SMParam{{Type: SMParamText, Text: "Nobody"}}}},
		{"SystemMessage without parameters", mobiusSystemMessage(SysMsgWelcome), SystemMessage{ID: SysMsgWelcome}},
		{
			// SystemMessage.writeImpl: d id, d count, then d type and the
			// value: S for text and player names, q for longs, d d for
			// skills, d d d for zones, d for the others.
			"SystemMessage parameter types",
			msg(0x64).i32(SysMsgLocationTalkingIsland).i32(7).
				i32(SMParamInt).i32(-84318).
				i32(SMParamLong).i64(1 << 40).
				i32(SMParamSkillName).i32(1177).i32(5).
				i32(SMParamZoneName).i32(10).i32(20).i32(-30).
				i32(SMParamPlayerName).str("Hero").
				i32(SMParamNpcName).i32(1000000 + 30001).
				i32(SMParamItemName).i32(57).b,
			SystemMessage{ID: SysMsgLocationTalkingIsland, Params: []SMParam{
				{Type: SMParamInt, Number: -84318},
				{Type: SMParamLong, Number: 1 << 40},
				{Type: SMParamSkillName, Numbers: []int32{1177, 5}},
				{Type: SMParamZoneName, Numbers: []int32{10, 20, -30}},
				{Type: SMParamPlayerName, Text: "Hero"},
				{Type: SMParamNpcName, Number: 1000000 + 30001},
				{Type: SMParamItemName, Number: 57},
			}},
		},
		{"RestartResponse ok", msg(0x5f).i32(1).b, RestartResponse{OK: true}},
		{"RestartResponse refused", msg(0x5f).i32(0).b, RestartResponse{}},
		{"LeaveWorld", msg(0x7e).b, LeaveWorld{}},
		{"ServerClose", msg(0x26).b, ServerClose{}},
		{"CharDeleteSuccess", msg(0x23).b, CharDeleteSuccess{}},
		{"CharDeleteFail", msg(0x24).i32(2).b, CharDeleteFail{Reason: DeleteFailClanMember}},
	}
	for _, tt := range tests {
		checkDecode(t, tt.name, tt.body, tt.want)
	}
}

// TestMoveToLocationLayout writes MoveToLocation byte by byte: Mobius
// writes the destination before the current position.
func TestMoveToLocationLayout(t *testing.T) {
	body := []byte{
		0x01,
		0x01, 0x00, 0x00, 0x10, // object 0x10000001
		0x10, 0x27, 0x00, 0x00, // destination x 10000
		0x20, 0x4e, 0x00, 0x00, // destination y 20000
		0x18, 0xfc, 0xff, 0xff, // destination z -1000
		0xe8, 0x03, 0x00, 0x00, // x 1000
		0xd0, 0x07, 0x00, 0x00, // y 2000
		0x9c, 0xff, 0xff, 0xff, // z -100
	}
	want := MoveToLocation{ObjectID: 0x10000001, DestX: 10000, DestY: 20000, DestZ: -1000, X: 1000, Y: 2000, Z: -100}
	checkDecode(t, "MoveToLocation", body, want)
	if !bytes.Equal(mobiusMoveToLocation(want), body) {
		t.Errorf("test builder differs:\n got % x\nwant % x", mobiusMoveToLocation(want), body)
	}
	if want.Origin() != (Position{1000, 2000, -100}) || want.Destination() != (Position{10000, 20000, -1000}) {
		t.Errorf("Origin %v, Destination %v", want.Origin(), want.Destination())
	}
}

// TestExStorageMaxCountLayout checks the Ex header: 0xfe and a
// little-endian sub-opcode.
func TestExStorageMaxCountLayout(t *testing.T) {
	body := []byte{0xfe, 0x2e, 0x00}
	for _, v := range []byte{80, 100, 150, 4, 3, 0, 30, 2} {
		body = append(body, v, 0, 0, 0)
	}
	checkDecode(t, "ExStorageMaxCount", body, ExStorageMaxCount{
		Inventory: 80, Warehouse: 100, ClanWarehouse: 150, PrivateSell: 4, PrivateBuy: 3, CommonRecipe: 30, InventoryExtra: 2,
	})
}

func TestDecodeUnknownPackets(t *testing.T) {
	tests := []struct {
		name     string
		body     []byte
		wantName string
		wantEx   uint16
		isEx     bool
	}{
		{"unknown opcode", []byte{0xf8, 1, 2}, "Unknown(0xf8)", 0, false},
		{"named, not parsed", []byte{0x1c}, "SunRise", 0, false},
		{"named Ex", []byte{0xfe, 0x32, 0x00, 1, 0, 0, 0}, "ExSetCompassZoneCode", 0x32, true},
		{"unknown Ex", []byte{0xfe, 0x99, 0x00}, "UnknownEx(0x0099)", 0x99, true},
	}
	for _, tt := range tests {
		p, err := DecodeServerPacket(tt.body)
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		u, ok := p.(Unknown)
		if !ok {
			t.Errorf("%s: got %T, want Unknown", tt.name, p)
			continue
		}
		if u.Opcode != tt.body[0] || !bytes.Equal(u.Body, tt.body[1:]) {
			t.Errorf("%s: Unknown{0x%02x, % x}", tt.name, u.Opcode, u.Body)
		}
		if u.PacketName() != tt.wantName {
			t.Errorf("%s: PacketName %q, want %q", tt.name, u.PacketName(), tt.wantName)
		}
		if sub, isEx := u.ExOpcode(); isEx != tt.isEx || sub != tt.wantEx {
			t.Errorf("%s: ExOpcode = 0x%04x, %v", tt.name, sub, isEx)
		}
	}
}

func TestDecodeWorldPacketErrors(t *testing.T) {
	var ui UserInfo
	runningFlag := mobiusUserInfo(ui)
	// The running flag sits before the last four words.
	runningFlag[len(runningFlag)-17] = 2
	// The equipped short of the first item: opcode, h, h, then h, d, d, d,
	// h, h before it.
	equipped := mobiusItemList(ItemList{Items: []Item{{Equipped: true}}})
	equipped[1+2+2+2+4+4+4+2+2] = 2

	tests := []struct {
		name   string
		body   []byte
		substr string
	}{
		{"UserInfo running flag 2", runningFlag, "UserInfo: running flag is 2"},
		{"UserInfo short", mobiusUserInfo(ui)[:100], "UserInfo: packets: short packet"},
		{"QuestList extra byte", msg(0x80).u16(0).u8(0).b, "QuestList: 1 bytes after the last field"},
		{"ItemList count too high", msg(0x1b).u16(0).u16(2).b, "ItemList: packets: short packet"},
		{"ItemList equipped 2", equipped, "ItemList: equipped flag is 2"},
		{"SkillList negative count", msg(0x58).i32(-1).b, "SkillList: skill count -1"},
		{"ShortcutInit type 6", msg(0x45).i32(1).i32(6).i32(0).b, "ShortcutInit: shortcut type 6"},
		{"SystemMessage parameter type 8", msg(0x64).i32(1).i32(1).i32(8).i32(0).b, "SystemMessage: parameter type 8"},
		{"RestartResponse result 2", msg(0x5f).i32(2).b, "RestartResponse: result is 2"},
		{"RelationChanged flag 3", msg(0xce).i32(1).i32(0).i32(3).i32(0).i32(0).b, "auto-attackable flag is 3"},
		{"CreatureSay without text", msg(0x4a).i32(1).i32(0).str("Hero").b, "CreatureSay: packets: short packet"},
		{"CreatureSay with two texts", msg(0x4a).i32(1).i32(0).str("Hero").str("a").str("b").b, "CreatureSay: 4 bytes after the last field"},
		{"Ex without a sub-opcode", []byte{0xfe, 0x2e}, "Ex packet: packets: short packet"},
		{"ExStorageMaxCount short", msg(0xfe).u16(0x2e).i32(1).b, "ExStorageMaxCount: packets: short packet"},
		{"LeaveWorld with a byte", msg(0x7e).u8(0).b, "LeaveWorld: 1 bytes after the last field"},
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

func TestSystemMessageTexts(t *testing.T) {
	m := SystemMessage{ID: SysMsgText, Params: []SMParam{
		{Type: SMParamText, Text: "a"}, {Type: SMParamInt, Number: 1}, {Type: SMParamPlayerName, Text: "Hero"},
	}}
	if got := m.Texts(); !reflect.DeepEqual(got, []string{"a", "Hero"}) {
		t.Errorf("Texts() = %q", got)
	}
}

// TestCharacterScreenPacketsArePackets checks that the packets of the
// character screen also come out of DecodeServerPacket.
func TestCharacterScreenPacketsArePackets(t *testing.T) {
	hero := testChar("Hero", 0x10000001)
	checkDecode(t, "CharSelectionInfo", mobiusCharSelectionInfo([]CharInfo{hero}), CharacterList{Characters: []CharInfo{hero}})
	checkDecode(t, "NewCharacterSuccess", mobiusNewCharacterSuccess(testTemplates()), NewCharacterSuccess{Templates: testTemplates()})
	checkDecode(t, "LoginFail", msg(0x14).i32(1).b, GameLoginFail{Reason: GameLoginFailSystemErrorLoginLater})
	checkDecode(t, "KeyPacket", mobiusKeyPacket(1, testKeyHalf, true, 1),
		KeyPacket{Result: 1, KeyHalf: testKeyHalf, Encryption: true, ServerID: 1, Unknown: 1})
}
