package testclient

import (
	"bytes"
	"reflect"
	"slices"
	"testing"
)

// The client packets of Milestones 6 and 7 as Mobius reads them,
// transcribed from their readImpl.

// mobiusRequestRestartPoint: d point.
func mobiusRequestRestartPoint(point int32) []byte { return msg(0x6d).i32(point).b }

// mobiusRequestMagicSkillUse: d skill, d ctrl (readInt() != 0), c shift
// (readByte() != 0).
func mobiusRequestMagicSkillUse(skillID int32, ctrl, shift bool) []byte {
	return msg(0x2f).i32(skillID).i32(b2i(ctrl)).u8(b2u8(shift)).b
}

// mobiusRequestAcquireSkill is RequestAcquireSkillInfo (0x6b) and
// RequestAcquireSkill (0x6c): d id, d level, d type.
func mobiusRequestAcquireSkill(op byte, skillID, level, typ int32) []byte {
	return msg(op).i32(skillID).i32(level).i32(typ).b
}

// mobiusRequestShortcutReg: d type, d slot (slot + 12 * page), d id, d
// level (read and ignored).
func mobiusRequestShortcutReg(typ, slot, id, level int32) []byte {
	return msg(0x33).i32(typ).i32(slot).i32(id).i32(level).b
}

// mobiusRequestShortcutDel: d slot + 12 * page.
func mobiusRequestShortcutDel(slot int32) []byte { return msg(0x35).i32(slot).b }

// mobiusRequestAutoSoulShot: the Ex header D0 05 00, d item, d type (1 on,
// 0 off).
func mobiusRequestAutoSoulShot(itemID int32, on bool) []byte {
	return msg(0xd0).u16(0x05).i32(itemID).i32(b2i(on)).b
}

func TestEncodeCombatRequests(t *testing.T) {
	tests := []struct {
		name string
		got  []byte
		want []byte
	}{
		{"RequestRestartPoint", encodeInt(OpRequestRestartPoint, RestartVillage), []byte{0x6d, 0, 0, 0, 0}},
		{"RequestRestartPoint fixed", encodeInt(OpRequestRestartPoint, RestartFixed), mobiusRequestRestartPoint(4)},
		{"RequestMagicSkillUse", encodeRequestMagicSkillUse(1177, false, false), []byte{0x2f, 0x99, 0x04, 0, 0, 0, 0, 0, 0, 0}},
		{"RequestMagicSkillUse ctrl shift", encodeRequestMagicSkillUse(1177, true, true), mobiusRequestMagicSkillUse(1177, true, true)},
		{"RequestAcquireSkillInfo", encodeInts(OpRequestAcquireSkillInfo, 1177, 2, AcquireSkillClass),
			mobiusRequestAcquireSkill(0x6b, 1177, 2, AcquireSkillClass)},
		{"RequestAcquireSkill", encodeInts(OpRequestAcquireSkill, 1312, 1, AcquireSkillFishing),
			mobiusRequestAcquireSkill(0x6c, 1312, 1, AcquireSkillFishing)},
		{"RequestShortcutReg", encodeInts(OpRequestShortcutReg, ShortcutSkill, 13, 1177, 2), mobiusRequestShortcutReg(2, 13, 1177, 2)},
		{"RequestShortcutDel", encodeInt(OpRequestShortcutDel, 13), mobiusRequestShortcutDel(13)},
		{"RequestAutoSoulShot", encodeRequestAutoSoulShot(1835, true), []byte{0xd0, 0x05, 0x00, 0x2b, 0x07, 0, 0, 1, 0, 0, 0}},
		{"RequestAutoSoulShot off", encodeRequestAutoSoulShot(1835, false), mobiusRequestAutoSoulShot(1835, false)},
	}
	for _, tt := range tests {
		if !bytes.Equal(tt.got, tt.want) {
			t.Errorf("%s:\n got % x\nwant % x", tt.name, tt.got, tt.want)
		}
	}

	names := []struct {
		raw  []byte
		want string
	}{
		{encodeInt(OpRequestRestartPoint, 0), "RequestRestartPoint"},
		{encodeRequestMagicSkillUse(1, false, false), "RequestMagicSkillUse"},
		{encodeInts(OpRequestAcquireSkillInfo, 1, 1, 0), "RequestAcquireSkillInfo"},
		{encodeInts(OpRequestAcquireSkill, 1, 1, 0), "RequestAcquireSkill"},
		{encodeInts(OpRequestShortcutReg, 1, 1, 1, 1), "RequestShortcutReg"},
		{encodeInt(OpRequestShortcutDel, 1), "RequestShortcutDel"},
		{encodeRequestAutoSoulShot(1, true), "RequestAutoSoulShot"},
	}
	for _, tt := range names {
		if got := clientPacketName(tt.raw); got != tt.want {
			t.Errorf("clientPacketName(% x) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

// TestCombatFlow fights a monster to its death against a fake server, then
// dies and restarts in the village, in Mobius's order of packets.
func TestCombatFlow(t *testing.T) {
	pos := Position{-71338, 258271, -3104}
	town := Position{-84318, 244579, -3730}
	monster := testMonster(0x60000005, Position{-71300, 258300, -3104})
	const loot int32 = 0x10000050
	first := Attack{AttackerID: testMeID, Hits: []Hit{
		{TargetID: monster.ObjectID, Damage: 30, Flags: HitCritical},
		{TargetID: monster.ObjectID, Flags: HitMiss},
	}, X: pos.X, Y: pos.Y, Z: pos.Z, TargetX: monster.X, TargetY: monster.Y, TargetZ: monster.Z}
	second := first
	second.Hits = []Hit{{TargetID: monster.ObjectID, Damage: 70, Flags: HitSoulshot | 1}}

	addr := startFakeGame(t, func(f *fakeGame) {
		f.enterQuick(pos, testItems(), mobiusNpcInfo(monster))

		// A click targets the monster, then AttackRequest starts the fight.
		f.expect(mobiusAction(0x04, monster.ObjectID, pos, false))
		f.send(mobiusMyTargetSelected(monster.ObjectID, 0))
		f.send(mobiusStatusUpdate(monster.ObjectID, StatusAttribute{StatusCurHP, 100}, StatusAttribute{StatusMaxHP, 100}))
		f.expect(mobiusAction(0x0a, monster.ObjectID, pos, false))
		f.send(msg(0x2b).i32(testMeID).b)
		f.send(mobiusAttack(first))
		f.send(mobiusStatusUpdate(monster.ObjectID, StatusAttribute{StatusCurHP, 70}))

		// The second swing kills it, and it drops adena.
		f.send(mobiusAttack(second))
		f.send(mobiusStatusUpdate(monster.ObjectID, StatusAttribute{StatusCurHP, 0}))
		f.send(mobiusDie(Die{ObjectID: monster.ObjectID, Sweepable: true}))
		f.send(msg(0x64).i32(SysMsgEarnedExpAndSP).i32(2).i32(SMParamInt).i32(42).i32(SMParamInt).i32(3).b)
		f.send(mobiusDropItem(DropItem{DropperID: monster.ObjectID, ObjectID: loot, ItemID: 57,
			X: monster.X, Y: monster.Y, Z: monster.Z, Stackable: true, Count: 25, Unknown: 1}))
		f.send(msg(0x2c).i32(testMeID).b)

		// Then the character dies and restarts in the village: Mobius
		// revives it, teleports it, and sends UserInfo after Appearing.
		f.send(mobiusStatusUpdate(testMeID, StatusAttribute{StatusCurHP, 0}))
		f.send(mobiusDie(Die{ObjectID: testMeID, CanTeleport: true}))
		f.expect(mobiusRequestRestartPoint(RestartVillage))
		f.send(msg(0x07).i32(testMeID).b)
		f.send(msg(0x28).i32(testMeID).i32(town.X).i32(town.Y).i32(town.Z).i32(0).i32(0).b)
		f.expect([]byte{0x30})
		f.send(mobiusUserInfo(testUserInfoAt(town)))
		f.drain()
	})

	ctx := testContext(t)
	c := clientEnterQuick(t, ctx, addr)
	w := c.World()

	a, err := c.Attack(ctx, monster.ObjectID)
	if err != nil {
		t.Fatalf("Attack: %v", err)
	}
	if !reflect.DeepEqual(a, first) {
		t.Errorf("Attack\n got %+v\nwant %+v", a, first)
	}
	if h, ok := a.HitOn(monster.ObjectID); !ok || !h.Critical() || h.Damage != 30 {
		t.Errorf("first hit %+v, %v", h, ok)
	}
	if me := w.Me(); !me.InCombat || me.TargetID != monster.ObjectID {
		t.Errorf("after the first swing: in combat %v, target %d", me.InCombat, me.TargetID)
	}

	d, err := Await(ctx, c, func(d Die) bool { return d.ObjectID == monster.ObjectID })
	if err != nil || !d.Sweepable {
		t.Fatalf("the monster's Die = %+v, %v", d, err)
	}
	if o, _ := w.Object(monster.ObjectID); !o.Dead || o.HP != 0 || o.MaxHP != 100 {
		t.Errorf("dead monster %+v", o)
	}
	drop, err := Await(ctx, c, func(d DropItem) bool { return d.DropperID == monster.ObjectID })
	if err != nil || drop.ObjectID != loot {
		t.Fatalf("loot = %+v, %v", drop, err)
	}
	if g, ok := w.FindGroundItem(57); !ok || g.ObjectID != loot || g.Count != 25 || g.DropperID != monster.ObjectID {
		t.Errorf("FindGroundItem(57) = %+v, %v", g, ok)
	}
	if !slices.ContainsFunc(w.SystemMessages(), func(m SystemMessage) bool { return m.ID == SysMsgEarnedExpAndSP }) {
		t.Error("the experience message is missing")
	}
	if _, err := Await(ctx, c, func(s AutoAttackStop) bool { return s.ObjectID == testMeID }); err != nil {
		t.Fatal(err)
	}
	if w.Me().InCombat {
		t.Error("still in combat after AutoAttackStop")
	}

	dead, err := Await(ctx, c, func(d Die) bool { return d.ObjectID == testMeID })
	if err != nil || !dead.CanTeleport {
		t.Fatalf("own Die = %+v, %v", dead, err)
	}
	if me := w.Me(); !me.Dead || me.HP != 0 || !me.Death.CanTeleport {
		t.Errorf("dead character %+v", me)
	}

	res, err := c.RestartPoint(ctx, RestartVillage)
	if err != nil {
		t.Fatalf("RestartPoint: %v", err)
	}
	if res.Revive.ObjectID != testMeID || !res.Teleported || res.Teleport.Position() != town || res.UserInfo.Position() != town {
		t.Errorf("RestartPoint = %+v", res)
	}
	if me := w.Me(); me.Dead || me.Pos != town || me.HP != 120 {
		t.Errorf("after the restart: dead %v, at %v, HP %d", me.Dead, me.Pos, me.HP)
	}
}

// TestAttackTargeted attacks the current target: no click goes first.
func TestAttackTargeted(t *testing.T) {
	pos := Position{100, 200, -300}
	monster := testMonster(0x60000005, Position{120, 200, -300})
	swing := Attack{AttackerID: testMeID, Hits: []Hit{{TargetID: monster.ObjectID, Damage: 9}},
		X: pos.X, Y: pos.Y, Z: pos.Z, TargetX: monster.X, TargetY: monster.Y, TargetZ: monster.Z}
	other := swing
	other.AttackerID = testOtherID

	addr := startFakeGame(t, func(f *fakeGame) {
		f.enterQuick(pos, nil, mobiusNpcInfo(monster), mobiusMyTargetSelected(monster.ObjectID, 0))
		f.expect(mobiusAction(0x0a, monster.ObjectID, pos, false))
		f.send(mobiusAttack(other)) // someone else's swing is skipped
		f.send(mobiusAttack(swing))
		f.drain()
	})
	ctx := testContext(t)
	c := clientEnterQuick(t, ctx, addr)
	a, err := c.Attack(ctx, monster.ObjectID)
	if err != nil || !reflect.DeepEqual(a, swing) {
		t.Fatalf("Attack = %+v, %v", a, err)
	}
}

// TestRestartPointOrders gets up with the teleport before the revival, and
// in place with the fixed restart point.
func TestRestartPointOrders(t *testing.T) {
	pos := Position{1, 2, 3}
	town := Position{-84318, 244579, -3730}
	tests := []struct {
		name   string
		point  int32
		answer func(f *fakeGame)
		check  func(t *testing.T, res RestartResult)
	}{
		{
			"teleport first", RestartVillage,
			func(f *fakeGame) {
				f.send(msg(0x28).i32(testMeID).i32(town.X).i32(town.Y).i32(town.Z).i32(0).i32(5).b)
				f.send(msg(0x07).i32(testOtherID).b) // another player's revival
				f.send(msg(0x07).i32(testMeID).b)
				f.expect([]byte{0x30})
				f.send(mobiusUserInfo(testUserInfoAt(town)))
			},
			func(t *testing.T, res RestartResult) {
				if !res.Teleported || res.Teleport.Heading != 5 || res.Revive.ObjectID != testMeID || res.UserInfo.Position() != town {
					t.Errorf("RestartPoint = %+v", res)
				}
			},
		},
		{
			"fixed", RestartFixed,
			func(f *fakeGame) {
				f.send(mobiusStatusUpdate(testMeID, StatusAttribute{StatusCurHP, 126}))
				f.send(msg(0x07).i32(testMeID).b)
			},
			func(t *testing.T, res RestartResult) {
				if res.Teleported || res.Revive.ObjectID != testMeID {
					t.Errorf("RestartPoint = %+v", res)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := startFakeGame(t, func(f *fakeGame) {
				f.enterQuick(pos, nil, mobiusDie(Die{ObjectID: testMeID, CanTeleport: true, FixedRes: true}))
				f.expect(mobiusRequestRestartPoint(tt.point))
				tt.answer(f)
				f.drain()
			})
			ctx := testContext(t)
			c := clientEnterQuick(t, ctx, addr)
			if !c.World().Me().Dead {
				t.Fatal("not dead after Die")
			}
			res, err := c.RestartPoint(ctx, tt.point)
			if err != nil {
				t.Fatalf("RestartPoint: %v", err)
			}
			tt.check(t, res)
			if c.World().Me().Dead {
				t.Error("still dead after RestartPoint")
			}
		})
	}
}

// TestSkillFlow learns a skill, casts it, puts it on the shortcut bar,
// turns on automatic soulshots and gets buff icons, against a fake server.
func TestSkillFlow(t *testing.T) {
	pos := Position{-71338, 258271, -3104}
	monster := testMonster(0x60000005, Position{-71300, 258300, -3104})
	const pet int32 = 0x60000009
	shots := Item{Type1: 4, ObjectID: 0x10000017, ItemID: 1835, Count: 100, Type2: 5, Mana: -1}
	itemShortcut := msg(0x45).i32(1).i32(ShortcutItem).i32(3).i32(shots.ObjectID).i32(1).i32(-1).i32(0).i32(0).u16(0).u16(0).b
	info := AcquireSkillInfo{SkillID: 1177, Level: 2, SPCost: 50, Type: AcquireSkillClass,
		Requirements: []SkillRequirement{{Type: 99, ItemID: 1050, Count: 1, Unknown: 50}}}
	cast := MagicSkillUse{CasterID: testMeID, TargetID: monster.ObjectID, SkillID: 1177, Level: 2, HitTime: 1080, ReuseDelay: 4000,
		X: pos.X, Y: pos.Y, Z: pos.Z, TargetX: monster.X, TargetY: monster.Y, TargetZ: monster.Z}
	launched := MagicSkillLaunched{CasterID: testMeID, SkillID: 1177, Level: 2, Targets: []int32{monster.ObjectID}}

	addr := startFakeGame(t, func(f *fakeGame) {
		f.enterQuick(pos, append(testItems(), shots), mobiusNpcInfo(monster), itemShortcut)

		// What the next level of Wind Strike costs, then learning it: the
		// message, the skill list and the trainer's next list.
		f.expect(mobiusRequestAcquireSkill(0x6b, 1177, 2, AcquireSkillClass))
		f.send(mobiusAcquireSkillInfo(info))
		f.expect(mobiusRequestAcquireSkill(0x6c, 1177, 2, AcquireSkillClass))
		f.send(msg(0x64).i32(SysMsgLearnedSkill).i32(1).i32(SMParamSkillName).i32(1177).i32(2).b)
		f.send(mobiusSkillList([]Skill{{Level: 2, ID: 1177}}))
		f.send(mobiusAcquireSkillList(AcquireSkillList{Type: AcquireSkillClass,
			Skills: []LearnableSkill{{ID: 1184, NextLevel: 1, MaxLevel: 1, SPCost: 70}}}))

		// Casting it on the monster.
		f.expect(mobiusRequestMagicSkillUse(1177, false, false))
		f.send(mobiusMagicSkillUse(cast))
		f.send(mobiusSetupGauge(SetupGauge{Color: GaugeBlue, Time: 1080, MaxTime: 1080}))
		f.send(mobiusStatusUpdate(testMeID, StatusAttribute{StatusCurMP, 20}))
		f.send(mobiusMagicSkillLaunched(launched))

		// A shortcut for it, then deleting it: Mobius does not answer the
		// delete.
		f.expect(mobiusRequestShortcutReg(ShortcutSkill, 13, 1177, 2))
		f.send(mobiusShortcutRegister(ShortcutSkill, 13, 1177, 2))
		f.expect(mobiusRequestShortcutDel(13))

		// Automatic soulshots.
		f.expect(mobiusRequestAutoSoulShot(1835, true))
		f.send(mobiusExAutoSoulShot(1835, true))
		f.send(mobiusSMItem(SysMsgAutoShotOn, 1835))

		// Buff icons.
		f.send(msg(0x7f).u16(1).i32(1040).u16(3).i32(1200).b)
		f.send(mobiusShortBuffStatusUpdate(ShortBuffStatusUpdate{SkillID: 2031, Level: 1, Duration: 15}))
		f.send(mobiusPartySpelled(PartySpelled{Kind: SpelledPet, ObjectID: pet,
			Effects: []Effect{{SkillID: 1040, Level: 3, Duration: 1200}}}))
		f.drain()
	})

	ctx := testContext(t)
	c := clientEnterQuick(t, ctx, addr)
	w := c.World()
	if s, ok := w.Shortcut(3); !ok || s.Type != ShortcutItem || s.ID != shots.ObjectID {
		t.Errorf("Shortcut(3) after entering = %+v, %v", s, ok)
	}

	// Learn.
	got, err := c.AcquireSkillInfo(ctx, 1177, 2, AcquireSkillClass)
	if err != nil || !reflect.DeepEqual(got, info) {
		t.Fatalf("AcquireSkillInfo = %+v, %v", got, err)
	}
	list, err := c.AcquireSkill(ctx, 1177, 2, AcquireSkillClass)
	if err != nil || len(list.Skills) != 1 {
		t.Fatalf("AcquireSkill = %+v, %v", list, err)
	}
	if lvl := w.SkillLevel(1177); lvl != 2 {
		t.Errorf("Wind Strike level %d, want 2", lvl)
	}
	next, err := Await[AcquireSkillList](ctx, c, nil)
	if _, ok := next.Find(1184); err != nil || !ok {
		t.Errorf("the next AcquireSkillList = %+v, %v", next, err)
	}

	// Cast.
	m, err := c.CastSkill(ctx, 1177, false, false)
	if err != nil || m != cast {
		t.Fatalf("CastSkill = %+v, %v", m, err)
	}
	l, err := Await(ctx, c, func(l MagicSkillLaunched) bool { return l.CasterID == testMeID })
	if err != nil || !reflect.DeepEqual(l, launched) {
		t.Fatalf("MagicSkillLaunched = %+v, %v", l, err)
	}
	if mp := w.Me().MP; mp != 20 {
		t.Errorf("MP %d after the cast, want 20", mp)
	}

	// Shortcuts.
	reg, err := c.RegisterShortcut(ctx, Shortcut{Type: ShortcutSkill, Slot: 13, ID: 1177, Level: 2})
	want := Shortcut{Type: ShortcutSkill, Slot: 13, ID: 1177, Level: 2, Rest: []byte{0, 1, 0, 0, 0}}
	if err != nil || !reflect.DeepEqual(reg.Shortcut, want) {
		t.Fatalf("RegisterShortcut = %+v, %v", reg, err)
	}
	if s, ok := w.Shortcut(13); !ok || s.ID != 1177 {
		t.Errorf("Shortcut(13) = %+v, %v", s, ok)
	}
	if err := c.DeleteShortcut(ctx, 13); err != nil {
		t.Fatal(err)
	}
	if _, ok := w.Shortcut(13); ok {
		t.Error("the deleted shortcut is still there")
	}
	if n := len(w.Shortcuts()); n != 1 {
		t.Errorf("%d shortcuts, want the item one", n)
	}

	// Soulshots.
	ss, err := c.AutoSoulShot(ctx, 1835, true)
	if err != nil || !ss.Enabled || ss.ItemID != 1835 {
		t.Fatalf("AutoSoulShot = %+v, %v", ss, err)
	}
	if got := w.AutoSoulShots(); !slices.Equal(got, []int32{1835}) {
		t.Errorf("AutoSoulShots() = %v", got)
	}

	// Buffs.
	ps, err := Await[PartySpelled](ctx, c, nil)
	if err != nil || ps.ObjectID != pet || ps.Kind != SpelledPet {
		t.Fatalf("PartySpelled = %+v, %v", ps, err)
	}
	if e, ok := w.Effect(1040); !ok || e.Level != 3 || e.Duration != 1200 {
		t.Errorf("Effect(1040) = %+v, %v", e, ok)
	}
	if s, ok := w.ShortBuff(); !ok || s.SkillID != 2031 {
		t.Errorf("ShortBuff() = %+v, %v", s, ok)
	}
	if p, ok := w.PartySpelled(pet); !ok || len(p.Effects) != 1 {
		t.Errorf("PartySpelled(pet) = %+v, %v", p, ok)
	}
	if !slices.ContainsFunc(w.SystemMessages(), func(m SystemMessage) bool { return m.ID == SysMsgAutoShotOn }) {
		t.Error("the automatic use message is missing")
	}
}
