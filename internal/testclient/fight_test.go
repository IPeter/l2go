package testclient

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// mobiusExRegenMax is ExRegenMax.writeImpl: the Ex header, d 1, d count, d
// time, f hpRegen (already multiplied by 0.90 in the constructor).
func mobiusExRegenMax(r ExRegenMax) []byte {
	return msg(0xfe).u16(0x01).i32(r.One).i32(r.Duration).i32(r.Ticks).f64(r.Amount).b
}

// mobiusExUseSharedGroupItem is ExUseSharedGroupItem.writeImpl: the Ex
// header, d item, d group, d seconds left, d total seconds.
func mobiusExUseSharedGroupItem(u ExUseSharedGroupItem) []byte {
	return msg(0xfe).u16(0x49).i32(u.ItemID).i32(u.GroupID).i32(u.Remaining).i32(u.Total).b
}

func TestDecodeItemSkillPackets(t *testing.T) {
	// Healing Potion (1061, skill 2032): 15 s, a tick every 2 s (666 ms
	// ticks scaled), 24 HP per tick sent as 21.6.
	regen := ExRegenMax{One: 1, Duration: 15, Ticks: 3, Amount: 24 * 0.90}
	shared := ExUseSharedGroupItem{ItemID: 1061, GroupID: 2, Remaining: 9, Total: 10}
	checkDecode(t, "ExRegenMax", mobiusExRegenMax(regen), regen)
	checkDecode(t, "ExUseSharedGroupItem", mobiusExUseSharedGroupItem(shared), shared)

	// Sizes counted from the writeImpl: the sub-opcode, three d and an f;
	// the sub-opcode and four d.
	if n := len(mobiusExRegenMax(ExRegenMax{})) - 1; n != 2+12+8 {
		t.Errorf("ExRegenMax body is %d bytes", n)
	}
	if n := len(mobiusExUseSharedGroupItem(ExUseSharedGroupItem{})) - 1; n != 2+16 {
		t.Errorf("ExUseSharedGroupItem body is %d bytes", n)
	}

	errs := []struct {
		name, substr string
		body         []byte
	}{
		{"ExRegenMax without the amount", "ExRegenMax: packets: short packet", msg(0xfe).u16(0x01).i32(1).i32(15).i32(3).b},
		{"ExUseSharedGroupItem extra", "ExUseSharedGroupItem: 4 bytes after the last field", append(mobiusExUseSharedGroupItem(shared), 0, 0, 0, 0)},
	}
	for _, tt := range errs {
		_, err := DecodeServerPacket(tt.body)
		if err == nil || !strings.Contains(err.Error(), tt.substr) || !errors.Is(err, ErrMalformed) {
			t.Errorf("%s: got %v, want ErrMalformed with %q", tt.name, err, tt.substr)
		}
	}
	for raw, want := range map[string][]byte{"ExRegenMax": mobiusExRegenMax(regen), "ExUseSharedGroupItem": mobiusExUseSharedGroupItem(shared)} {
		if got := serverPacketName(want); got != raw {
			t.Errorf("serverPacketName = %q, want %q", got, raw)
		}
	}
}

func TestPickMonster(t *testing.T) {
	const keltir int32 = 20481
	me := Position{}
	mob := func(id, npcID, x int32) Object {
		return Object{ObjectID: id, Kind: KindNpc, NpcID: npcID, Attackable: true, Motion: Motion{Pos: Position{X: x}}}
	}
	dead := mob(1, keltir, 10)
	dead.Dead = true
	guard := mob(2, keltir, 20)
	guard.Attackable = false
	player := mob(7, 0, 5)
	player.Kind = KindPlayer
	objs := []Object{
		dead, guard, player,
		mob(3, 20432, 30), // an Elpy, nearer than any keltir
		mob(5, keltir, 400),
		mob(4, keltir, -400),
		mob(6, keltir, 900),
	}
	o, d, ok := PickMonster(me, objs, keltir)
	if !ok || o.ObjectID != 4 || d != 400 {
		t.Errorf("PickMonster(keltir) = %d at %v, %v; want object 4 (the nearest living keltir, lower ID on a tie)", o.ObjectID, d, ok)
	}
	if o, _, ok := PickMonster(me, objs, 0); !ok || o.ObjectID != 3 {
		t.Errorf("PickMonster(any) = %d, %v; want the Elpy", o.ObjectID, ok)
	}
	if _, _, ok := PickMonster(me, objs[:3], 0); ok {
		t.Error("a dead monster, a guard or a player was picked")
	}
}

func TestSkillListBypass(t *testing.T) {
	if got := SkillListBypass(268441600); got != "npc_268441600_SkillList" {
		t.Errorf("SkillListBypass = %q", got)
	}
}

// TestFightUntilDead counts the character's swings until the target dies,
// and ends with ErrDied or ErrTargetLost.
func TestFightUntilDead(t *testing.T) {
	pos := Position{-71338, 258271, -3104}
	monster := testMonster(0x60000005, Position{-71300, 258300, -3104})
	swing := func(attacker int32) []byte {
		return mobiusAttack(Attack{AttackerID: attacker, Hits: []Hit{{TargetID: monster.ObjectID, Damage: 10}},
			X: pos.X, Y: pos.Y, Z: pos.Z, TargetX: monster.X, TargetY: monster.Y, TargetZ: monster.Z})
	}
	tests := []struct {
		name   string
		fight  [][]byte
		swings int
		err    error
	}{
		{"kill", [][]byte{swing(testMeID), swing(testOtherID), swing(testMeID), swing(testMeID), mobiusDie(Die{ObjectID: monster.ObjectID})}, 3, nil},
		{"death", [][]byte{swing(testMeID), mobiusDie(Die{ObjectID: testMeID, CanTeleport: true})}, 1, ErrDied},
		{"lost", [][]byte{swing(testMeID), msg(0x12).i32(monster.ObjectID).i32(0).b}, 1, ErrTargetLost},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := startFakeGame(t, func(f *fakeGame) {
				f.enterQuick(pos, nil, mobiusNpcInfo(monster))
				f.expect([]byte{OpRequestSkillCoolTime})
				for _, b := range tt.fight {
					f.send(b)
				}
				f.drain()
			})
			ctx := testContext(t)
			c := clientEnterQuick(t, ctx, addr)
			if err := c.RequestSkillCoolTime(ctx); err != nil {
				t.Fatal(err)
			}
			n, err := c.FightUntilDead(ctx, monster.ObjectID)
			if n != tt.swings || !errors.Is(err, tt.err) {
				t.Errorf("FightUntilDead = %d, %v; want %d, %v", n, err, tt.swings, tt.err)
			}
		})
	}
}

// TestFightUntilDeadAlreadyOver returns at once for a dead target or a
// dead character.
func TestFightUntilDeadAlreadyOver(t *testing.T) {
	pos := Position{1, 2, 3}
	monster := testMonster(0x60000005, Position{10, 2, 3})
	dead := monster
	dead.AlikeDead = true
	addr := startFakeGame(t, func(f *fakeGame) {
		f.enterQuick(pos, nil, mobiusNpcInfo(dead))
		f.drain()
	})
	ctx := testContext(t)
	c := clientEnterQuick(t, ctx, addr)
	if n, err := c.FightUntilDead(ctx, monster.ObjectID); n != 0 || err != nil {
		t.Errorf("FightUntilDead on a corpse = %d, %v", n, err)
	}
	c.World().apply(Die{ObjectID: testMeID})
	if _, err := c.FightUntilDead(ctx, monster.ObjectID); !errors.Is(err, ErrDied) {
		t.Errorf("FightUntilDead of a dead character = %v, want ErrDied", err)
	}
}

// TestCastOnAndLearn targets a monster and casts on it, sees an
// interrupted cast, opens a trainer's list twice (the second time it has
// nothing left) and learns a skill, against a fake server.
func TestCastOnAndLearn(t *testing.T) {
	pos := Position{-84729, 245001, -3720}
	monster := testMonster(0x60000005, Position{-84700, 245001, -3720})
	const trainer int32 = 0x60000007
	cast := MagicSkillUse{CasterID: testMeID, TargetID: monster.ObjectID, SkillID: 1177, Level: 1, HitTime: 1080, ReuseDelay: 4000,
		X: pos.X, Y: pos.Y, Z: pos.Z, TargetX: monster.X, TargetY: monster.Y, TargetZ: monster.Z}
	launched := MagicSkillLaunched{CasterID: testMeID, SkillID: 1177, Level: 1, Targets: []int32{monster.ObjectID}}
	list := AcquireSkillList{Type: AcquireSkillClass, Skills: []LearnableSkill{{ID: 1177, NextLevel: 2, MaxLevel: 2, SPCost: 240}}}
	info := AcquireSkillInfo{SkillID: 1177, Level: 2, SPCost: 240, Type: AcquireSkillClass}

	addr := startFakeGame(t, func(f *fakeGame) {
		f.enterQuick(pos, nil, mobiusNpcInfo(monster))
		f.expect(mobiusAction(0x04, monster.ObjectID, pos, false))
		f.send(mobiusMyTargetSelected(monster.ObjectID, 0))
		f.expect(mobiusRequestMagicSkillUse(1177, false, false))
		f.send(mobiusMagicSkillUse(cast))
		f.send(mobiusSetupGauge(SetupGauge{Time: 1080, MaxTime: 1080}))
		f.send(mobiusMagicSkillLaunched(MagicSkillLaunched{CasterID: testOtherID, SkillID: 1177, Level: 1}))
		f.send(mobiusMagicSkillLaunched(launched))

		// The second cast is interrupted.
		f.expect(mobiusRequestMagicSkillUse(1177, false, false))
		f.send(mobiusMagicSkillUse(cast))
		f.send(msg(0x49).i32(testMeID).b)

		f.expect(msg(0x21).str("npc_1610612743_SkillList").b)
		f.send(mobiusAcquireSkillList(list))
		f.expect(mobiusRequestAcquireSkill(0x6b, 1177, 2, AcquireSkillClass))
		f.send(mobiusAcquireSkillInfo(info))
		f.expect(mobiusRequestAcquireSkill(0x6c, 1177, 2, AcquireSkillClass))
		f.send(mobiusSkillList([]Skill{{Level: 2, ID: 1177}}))
		f.send(msg(0x8e).b)
		f.expect(msg(0x21).str("npc_1610612743_SkillList").b)
		f.send(msg(0x8e).b)
		f.drain()
	})
	ctx := testContext(t)
	c := clientEnterQuick(t, ctx, addr)

	got, err := c.CastOn(ctx, monster.ObjectID, 1177)
	if err != nil || got.Use != cast || !reflect.DeepEqual(got.Launched, launched) {
		t.Fatalf("CastOn = %+v, %v", got, err)
	}
	if _, err := c.CastOn(ctx, monster.ObjectID, 1177); err == nil || !strings.Contains(err.Error(), "canceled") {
		t.Errorf("an interrupted CastOn = %v, want a cancel error", err)
	}

	l, err := c.LearnableSkills(ctx, trainer)
	if err != nil || !reflect.DeepEqual(l, list) {
		t.Fatalf("LearnableSkills = %+v, %v", l, err)
	}
	gotInfo, skills, err := c.LearnSkill(ctx, 1177, 2, AcquireSkillClass)
	if err != nil || !reflect.DeepEqual(gotInfo, info) || skillLevel(skills.Skills, 1177) != 2 {
		t.Fatalf("LearnSkill = %+v, %+v, %v", gotInfo, skills, err)
	}
	if _, err := c.LearnableSkills(ctx, trainer); !errors.Is(err, ErrNothingToLearn) {
		t.Errorf("LearnableSkills after the last skill = %v, want ErrNothingToLearn", err)
	}
}

// TestGMHelpers sets the level and the SP of the character with GM
// commands: the character targets itself once, and a level it has already
// sends nothing.
func TestGMHelpers(t *testing.T) {
	pos := Position{-71338, 258271, -3104}
	leveled := testUserInfoAt(pos)
	leveled.Level, leveled.SP = 7, 0
	rich := leveled
	rich.SP = 1000
	addr := startFakeGame(t, func(f *fakeGame) {
		f.enterQuick(pos, nil)
		f.expect(mobiusAction(0x04, testMeID, pos, false))
		f.send(mobiusMyTargetSelected(testMeID, 0))
		f.expect(msg(0x5b).str("set_level 7").b)
		f.send(mobiusUserInfo(testUserInfoAt(pos))) // a level-up on the way
		f.send(mobiusUserInfo(leveled))
		f.expect(msg(0x5b).str("add_exp_sp 0 1000").b)
		f.send(mobiusUserInfo(rich))
		f.expect(mobiusRequestRestartPoint(RestartFixed))
		f.drain()
	})
	ctx := testContext(t)
	c := clientEnterQuick(t, ctx, addr)
	u, err := c.LevelTo(ctx, 7)
	if err != nil || u.Level != 7 {
		t.Fatalf("LevelTo(7) = level %d, %v", u.Level, err)
	}
	if u, err := c.LevelTo(ctx, 7); err != nil || u.Level != 7 {
		t.Errorf("LevelTo at the level = %d, %v", u.Level, err)
	}
	if u, err := c.AddSP(ctx, 1000); err != nil || u.SP != 1000 {
		t.Errorf("AddSP = %d, %v", u.SP, err)
	}
	if err := c.RequestRestartPoint(ctx, RestartFixed); err != nil {
		t.Fatal(err)
	}
}

// TestAwaitEffect waits for a buff icon.
func TestAwaitEffect(t *testing.T) {
	pos := Position{1, 2, 3}
	addr := startFakeGame(t, func(f *fakeGame) {
		f.enterQuick(pos, nil)
		f.expect([]byte{OpRequestSkillCoolTime})
		f.send(msg(0x7f).u16(2).i32(1068).u16(1).i32(1200).i32(1040).u16(1).i32(1200).b)
		f.drain()
	})
	ctx := testContext(t)
	c := clientEnterQuick(t, ctx, addr)
	if err := c.RequestSkillCoolTime(ctx); err != nil {
		t.Fatal(err)
	}
	e, err := c.AwaitEffect(ctx, 1040)
	if err != nil || e.Level != 1 || e.Duration != 1200 {
		t.Errorf("AwaitEffect(1040) = %+v, %v", e, err)
	}
}
