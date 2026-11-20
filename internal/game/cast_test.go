package game

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// castKeltirHP is the Keltir's HP in the cast tests: enough to take a few
// skills without dying.
const castKeltirHP = 2000

// newCastEnv starts a server with the skill data on npcTestWorld, with a
// Bearded Keltir of castKeltirHP HP 50 west of the start that never swings
// back (newAttackEnv). The game draws from a fixed seed.
func newCastEnv(t *testing.T) *testEnv {
	t.Helper()
	start := npcTestStart(t)
	wd := npcTestWorld(start)
	keltir := attackTestKeltir()
	keltir.Stats.Vitals.HP = castKeltirHP
	wd.Npcs = append(wd.Npcs, keltir)
	wd.Spawns = []data.Spawn{{
		File: "TalkingIsland/TestKeltir.xml",
		Npcs: []data.SpawnNpc{{NpcID: attackKeltir, X: start.X - attackKeltirDistance, Y: start.Y, Z: start.Z, Heading: 0, Count: 1, RespawnDelay: 60}},
	}}
	w, err := data.NewWorld(wd)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval, gs.World.SpawnProtection = 0, 0
	clock := world.NewFakeClock(testStart)
	ids := memstore.NewIDs()
	store := newRecordingStore()
	d := testGameData(t)
	srv, err := New(Options{
		Config: cfg, ServerID: gs.ID, Data: d, World: w,
		Characters: store, IDs: ids, Clock: clock, CheckLoopOwner: true,
		Passives: testPassives(t), Categories: testCategories(t), Skills: testSkills(t),
		Rand: rnd.New(1),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	srv.ai.doAttack = func(*Npc, Creature) {}
	runCtx, stopRun := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Run(runCtx) }()
	var once sync.Once
	stop := func() {
		once.Do(func() {
			stopRun()
			if err := <-done; err != nil {
				t.Errorf("Run: %v", err)
			}
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(func() {
		stop()
		cancel()
	})
	return &testEnv{t: t, ctx: ctx, srv: srv, store: store, ids: ids, clock: clock, data: d, stop: stop}
}

// castEnter enters a level 1 character of the class at the test start with
// the skills, lets 5 s pass and drops the packets.
func castEnter(t *testing.T, e *testEnv, account string, class int, known ...storage.CharacterSkill) (*fakeSession, *Player) {
	t.Helper()
	return castEnterLevel(t, e, account, class, 1, known...)
}

// castEnterLevel is castEnter at a level.
func castEnterLevel(t *testing.T, e *testEnv, account string, class int, level int32, known ...storage.CharacterSkill) (*fakeSession, *Player) {
	t.Helper()
	c := e.createCharacter(account, strings.ToUpper(account[:1])+account[1:], class, storage.AccessLevelPlayer)
	c.Level = level
	start := npcTestStart(t)
	c.X, c.Y, c.Z = int32(start.X), int32(start.Y), int32(start.Z)
	c.Skills = known
	if err := e.store.Update(e.ctx, &c); err != nil {
		t.Fatalf("Update: %v", err)
	}
	sess := newSession(account)
	p := e.enter(sess, 0)
	e.advance(5 * time.Second)
	sess.take(t)
	return sess, p
}

// castBody encodes RequestMagicSkillUse.
func castBody(id int32, ctrl, shift bool) []byte {
	w := packets.NewWriter(9)
	w.WriteInt32(id)
	if ctrl {
		w.WriteInt32(1)
	} else {
		w.WriteInt32(0)
	}
	if shift {
		w.WriteUint8(1)
	} else {
		w.WriteUint8(0)
	}
	return w.Bytes()
}

// castUse sends RequestMagicSkillUse and waits for the loop.
func castUse(e *testEnv, sess *fakeSession, id int32, ctrl, shift bool) {
	e.srv.Handle(sess, clientpackets.OpRequestMagicSkillUse, castBody(id, ctrl, shift))
	e.sync()
}

// castDescribe writes the packets as short lines like attackDescribe, with
// the cast packets spelled out. A StatusUpdate of the player with one value
// is the cast's own ("me 11=MP"); the full ones of the status feature are
// left out.
func castDescribe(ps []testclient.Packet, me, keltir int32) []string {
	who := func(id int32) string {
		switch id {
		case me:
			return "me"
		case keltir:
			return "keltir"
		}
		return strconv.Itoa(int(id))
	}
	var out []string
	for _, p := range attackOwn(ps, keltir) {
		switch v := p.(type) {
		case testclient.StatusUpdate:
			if v.ObjectID != me {
				hp, _ := v.Value(testclient.StatusCurHP)
				out = append(out, fmt.Sprintf("StatusUpdate %s HP %d", who(v.ObjectID), hp))
				continue
			}
			if len(v.Attributes) == 1 {
				out = append(out, fmt.Sprintf("StatusUpdate me %d=%d", v.Attributes[0].ID, v.Attributes[0].Value))
			}
		case testclient.MagicSkillUse:
			out = append(out, fmt.Sprintf("MagicSkillUse %s>%s %d/%d %d %d", who(v.CasterID), who(v.TargetID), v.SkillID, v.Level, v.HitTime, v.ReuseDelay))
		case testclient.MagicSkillLaunched:
			ts := make([]string, len(v.Targets))
			for i, id := range v.Targets {
				ts[i] = who(id)
			}
			out = append(out, fmt.Sprintf("MagicSkillLaunched %s %d/%d [%s]", who(v.CasterID), v.SkillID, v.Level, strings.Join(ts, " ")))
		case testclient.MagicSkillCanceled:
			out = append(out, "MagicSkillCanceled "+who(v.ObjectID))
		case testclient.SetupGauge:
			out = append(out, fmt.Sprintf("SetupGauge %d %d %d", v.Color, v.Time, v.MaxTime))
		case testclient.SystemMessage:
			params := make([]string, len(v.Params))
			for i, prm := range v.Params {
				switch {
				case prm.Text != "":
					params[i] = prm.Text
				case len(prm.Numbers) == 2:
					params[i] = fmt.Sprintf("skill %d/%d", prm.Numbers[0], prm.Numbers[1])
				default:
					params[i] = strconv.FormatInt(prm.Number, 10)
				}
			}
			out = append(out, fmt.Sprintf("SystemMessage %d [%s]", v.ID, strings.Join(params, ", ")))
		default:
			out = append(out, attackDescribe([]testclient.Packet{p}, me, keltir)...)
		}
	}
	return out
}

// castStep advances the clock by d and returns what the player got.
func castStep(t *testing.T, e *testEnv, sess *fakeSession, d time.Duration, me, keltir int32) []string {
	t.Helper()
	e.advance(d)
	return castDescribe(sess.take(t), me, keltir)
}

// castDamage reads "SystemMessage 35 [N]" and the Keltir's HP bar after it.
func castDamage(t *testing.T, lines []string) (damage int, rest []string) {
	t.Helper()
	for i, l := range lines {
		if n, ok := strings.CutPrefix(l, "SystemMessage 35 ["); ok {
			v, err := strconv.Atoi(strings.TrimSuffix(n, "]"))
			if err != nil {
				t.Fatalf("damage message %q", l)
			}
			return v, slices.Delete(slices.Clone(lines), i, i+1)
		}
	}
	t.Fatalf("no damage message in %q", lines)
	return 0, nil
}

// The Wind Strike timeline of a level 1 Human Mystic on a Keltir 50 away,
// to the millisecond as Mobius beginCast and MagicUseTask run it: at t0 the
// 2 MP of the start (StatusUpdate), MagicSkillUse with the cast time T =
// (int)(4000 / M.Atk.Spd × 333) and the reuse (int)(6000f × (333f /
// M.Atk.Spd)), MagicSkillLaunched, "You use Wind Strike.", the blue gauge;
// nothing at the launch (T − 400); at T the 7 MP of the hit, the damage and
// the stance of the negative skill. The reuse stamp runs from t0.
func TestCastWindStrikeTimeline(t *testing.T) {
	e := newCastEnv(t)
	sess, p := castEnter(t, e, "mage", classHumanMystic, storage.CharacterSkill{ID: 1177, Level: 1})
	keltir := npcSpawnedOne(t, e, attackKeltir)
	me, kid := p.ObjectID(), int32(keltir.ID())
	attackClick(e, sess, keltir, false)
	sess.take(t)

	sk := e.skill(1177, 1)
	var castTime, reuse, mp0, maxHP int
	e.onLoop(func() {
		spd := p.stats.MAtkSpd()
		castTime = int(4000.0 / float64(spd) * 333)
		reuse = int(float32(6000) * (float32(333) / float32(spd)))
		if got := skills.CastTime(p.skill, sk); got != castTime {
			t.Errorf("CastTime %d, want %d", got, castTime)
		}
		mp0, maxHP = int(p.CurMP), int(keltir.MaxHP())
	})
	if castTime <= 550 {
		t.Fatalf("cast time %d: the test needs a long cast", castTime)
	}

	castUse(e, sess, 1177, false, false)
	checkLines(t, "t0", castDescribe(sess.take(t), me, kid),
		fmt.Sprintf("StatusUpdate me 11=%d", mp0-2),
		fmt.Sprintf("MagicSkillUse me>keltir 1177/1 %d %d", castTime, reuse),
		"MagicSkillLaunched me 1177/1 [keltir]",
		"SystemMessage 46 [skill 1177/1]",
		fmt.Sprintf("SetupGauge 0 %d %d", castTime, castTime),
	)
	var casting, abortable bool
	var left time.Duration
	e.onLoop(func() {
		casting, abortable, left = e.srv.CastingNow(p), e.srv.CanAbortCast(p), e.srv.SkillReuseLeft(p, sk)
	})
	if !casting || !abortable || left != time.Duration(reuse)*time.Millisecond {
		t.Fatalf("after t0: casting %v, abortable %v, reuse left %v", casting, abortable, left)
	}

	ms := time.Millisecond
	launch := time.Duration(castTime)*ms - castLaunchLead
	checkLines(t, "before the launch", castStep(t, e, sess, launch-ms, me, kid))
	checkLines(t, "at the launch", castStep(t, e, sess, ms, me, kid))
	checkLines(t, "before the hit", castStep(t, e, sess, castLaunchLead-ms, me, kid))
	var mp1 int
	e.onLoop(func() {
		mp1 = int(p.CurMP)
		casting = e.srv.CastingNow(p)
	})
	if !casting {
		t.Fatal("the cast ended before its hit")
	}
	got := castStep(t, e, sess, ms, me, kid)
	dmg, rest := castDamage(t, got)
	checkLines(t, "at the hit", rest,
		fmt.Sprintf("StatusUpdate me 11=%d", mp1-7),
		fmt.Sprintf("StatusUpdate keltir HP %d", maxHP-dmg),
		"AutoAttackStart me",
	)
	e.onLoop(func() {
		casting, left = e.srv.CastingNow(p), e.srv.SkillReuseLeft(p, sk)
	})
	if casting || left != time.Duration(reuse-castTime)*ms {
		t.Errorf("after the hit: casting %v, reuse left %v, want %d ms", casting, left, reuse-castTime)
	}

	// Under reuse: Mobius tells the seconds left.
	castUse(e, sess, 1177, false, false)
	want := fmt.Sprintf("There are %d second(s) remaining in Wind Strike's re-use time.", (reuse-castTime)/1000)
	checkLines(t, "under reuse", castDescribe(sess.take(t), me, kid), "SystemMessage 1983 ["+want+"]")
}

// Power Strike of a level 1 Human Fighter with its Squire's Sword: physical,
// so T = (int)(1800 / P.Atk.Spd × 300) and the reuse (int)(13000 × 333 /
// P.Atk.Spd); no MP at the start, 10 at the hit; nextActionAttack swings
// 333 ms after the cast.
func TestCastPowerStrikeTimeline(t *testing.T) {
	e := newCastEnv(t)
	sess, p := castEnter(t, e, "fighter", classHumanFighter, storage.CharacterSkill{ID: 3, Level: 1})
	keltir := npcSpawnedOne(t, e, attackKeltir)
	me, kid := p.ObjectID(), int32(keltir.ID())
	attackClick(e, sess, keltir, false)
	sess.take(t)

	var castTime, reuse, maxHP int
	e.onLoop(func() {
		spd := p.stats.PAtkSpd()
		castTime = int(1800 / spd * 300)
		reuse = int(float64(13000) * (333 / spd))
		maxHP = int(keltir.MaxHP())
	})
	castUse(e, sess, 3, false, false)
	checkLines(t, "t0", castDescribe(sess.take(t), me, kid),
		fmt.Sprintf("MagicSkillUse me>keltir 3/1 %d %d", castTime, reuse),
		"MagicSkillLaunched me 3/1 [keltir]",
		"SystemMessage 46 [skill 3/1]",
		fmt.Sprintf("SetupGauge 0 %d %d", castTime, castTime),
	)
	ms := time.Millisecond
	checkLines(t, "before the hit", castStep(t, e, sess, time.Duration(castTime-1)*ms, me, kid))
	var mp1 int
	e.onLoop(func() { mp1 = int(p.CurMP) })
	got := castStep(t, e, sess, ms, me, kid)
	dmg, rest := castDamage(t, got)
	checkLines(t, "at the hit", rest,
		fmt.Sprintf("StatusUpdate me 11=%d", mp1-10),
		fmt.Sprintf("StatusUpdate keltir HP %d", maxHP-dmg),
		"AutoAttackStart me",
	)
	checkLines(t, "before the attack", castStep(t, e, sess, castNextAttackDelay-ms, me, kid))
	got = castStep(t, e, sess, ms, me, kid)
	if len(got) == 0 || !strings.HasPrefix(got[0], "Attack me [keltir") {
		t.Fatalf("333 ms after the cast: %q, want the attack", got)
	}
}

// Out of reach (40 + both collision radii) the player first walks up to the
// target (MoveToPawn with the cast range), then casts on arrival.
func TestCastWalksIntoRange(t *testing.T) {
	e := newCastEnv(t)
	sess, p := castEnter(t, e, "fighter", classHumanFighter, storage.CharacterSkill{ID: 3, Level: 1})
	keltir := npcSpawnedOne(t, e, attackKeltir)
	me, kid := p.ObjectID(), int32(keltir.ID())
	attackClick(e, sess, keltir, false)
	e.onLoop(func() {
		pos := p.Position()
		pos.X += 150
		if err := e.srv.MoveObject(p, pos); err != nil {
			t.Errorf("MoveObject: %v", err)
		}
	})
	sess.take(t)
	castUse(e, sess, 3, false, false)
	got := castDescribe(sess.take(t), me, kid)
	if !slices.Contains(got, "MoveToPawn me→keltir 40") || slices.ContainsFunc(got, func(l string) bool { return strings.HasPrefix(l, "MagicSkillUse") }) {
		t.Fatalf("out of reach: %q, want the walk and no cast", got)
	}
	for range 30 {
		got = castStep(t, e, sess, 100*time.Millisecond, me, kid)
		if slices.ContainsFunc(got, func(l string) bool { return strings.HasPrefix(l, "MagicSkillUse me>keltir 3/1 ") }) {
			return
		}
	}
	t.Fatal("no cast within 3 s of walking")
}

// Damage breaks a magic cast before its interrupt time (calcAtkBreak in
// onHitTimer): MagicSkillCanceled, ActionFailed, "Your casting has been
// interrupted."; the hit never comes.
func TestCastDamageInterrupts(t *testing.T) {
	e := newCastEnv(t)
	sess, p := castEnter(t, e, "mage", classHumanMystic, storage.CharacterSkill{ID: 1177, Level: 1})
	keltir := npcSpawnedOne(t, e, attackKeltir)
	me, kid := p.ObjectID(), int32(keltir.ID())
	attackClick(e, sess, keltir, false)
	castUse(e, sess, 1177, false, false)
	sess.take(t)
	e.advance(time.Second)
	sess.take(t)
	e.onLoop(func() {
		e.srv.attack.rnd = rnd.NewScripted(t, rnd.Get(100, 0)) // the break roll
		e.srv.onHitTimer(keltir, p, stats.Hit{Damage: 5}, false)
	})
	got := castDescribe(sess.take(t), me, kid)
	i := slices.Index(got, "MagicSkillCanceled me")
	if i < 0 || !slices.Equal(got[i:i+3], []string{"MagicSkillCanceled me", "ActionFailed", "SystemMessage 27 []"}) {
		t.Fatalf("after the hit: %q", got)
	}
	var casting bool
	e.onLoop(func() { casting = e.srv.CastingNow(p) })
	if casting {
		t.Fatal("still casting")
	}
	for _, l := range castStep(t, e, sess, 5*time.Second, me, kid) {
		if strings.HasPrefix(l, "SystemMessage 35") || strings.HasPrefix(l, "StatusUpdate me 11=") {
			t.Fatalf("the broken cast hit: %q", l)
		}
	}
}

// Esc aborts a cast that can still be aborted, and drops the target only
// the second time.
func TestCastEscAborts(t *testing.T) {
	e := newCastEnv(t)
	sess, p := castEnter(t, e, "mage", classHumanMystic, storage.CharacterSkill{ID: 1177, Level: 1})
	keltir := npcSpawnedOne(t, e, attackKeltir)
	me, kid := p.ObjectID(), int32(keltir.ID())
	attackClick(e, sess, keltir, false)
	castUse(e, sess, 1177, false, false)
	sess.take(t)
	esc := func() []string {
		e.srv.Handle(sess, clientpackets.OpRequestTargetCanceld, []byte{0, 0})
		e.sync()
		return castDescribe(sess.take(t), me, kid)
	}
	checkLines(t, "the first Esc", esc(), "MagicSkillCanceled me", "ActionFailed")
	checkLines(t, "the second Esc", esc(), "TargetUnselected")
}

// The refusals send what Mobius sends.
func TestCastRefusals(t *testing.T) {
	e := newCastEnv(t)
	sess, p := castEnter(t, e, "mage", classHumanMystic,
		storage.CharacterSkill{ID: 1177, Level: 1}, storage.CharacterSkill{ID: 1011, Level: 1})
	keltir := npcSpawnedOne(t, e, attackKeltir)
	me, kid := p.ObjectID(), int32(keltir.ID())
	use := func(id int32, ctrl bool) []string {
		castUse(e, sess, id, ctrl, false)
		return castDescribe(sess.take(t), me, kid)
	}

	checkLines(t, "an unknown skill", use(3, false), "ActionFailed")
	checkLines(t, "no target", use(1177, false), "ActionFailed")
	attackClick(e, sess, keltir, false)
	sess.take(t)
	checkLines(t, "a good skill on a monster without Ctrl", use(1011, false), "ActionFailed")

	e.onLoop(func() { p.SetCurrentMP(8) })
	sess.take(t)
	checkLines(t, "not enough MP", use(1177, false), "SystemMessage 24 []", "ActionFailed")

	e.onLoop(func() { e.srv.sitDown(p) })
	e.advance(3 * time.Second)
	sess.take(t)
	checkLines(t, "sitting", use(1177, false), "SystemMessage 31 []", "ActionFailed")
}

// During a cast another skill is queued (ActionFailed) and cast when the
// first one finishes.
func TestCastQueue(t *testing.T) {
	e := newCastEnv(t)
	sess, p := castEnter(t, e, "mage", classHumanMystic,
		storage.CharacterSkill{ID: 1177, Level: 1}, storage.CharacterSkill{ID: 1011, Level: 1})
	keltir := npcSpawnedOne(t, e, attackKeltir)
	me, kid := p.ObjectID(), int32(keltir.ID())
	attackClick(e, sess, keltir, false)
	castUse(e, sess, 1177, false, false)
	sess.take(t)
	var castTime int
	e.onLoop(func() { castTime = skills.CastTime(p.skill, e.skill(1177, 1)) })
	castUse(e, sess, 1011, true, false)
	checkLines(t, "the queued skill", castDescribe(sess.take(t), me, kid), "ActionFailed")
	got := castStep(t, e, sess, time.Duration(castTime)*time.Millisecond, me, kid)
	if !slices.ContainsFunc(got, func(l string) bool { return strings.HasPrefix(l, "MagicSkillUse me>keltir 1011/1 ") }) {
		t.Fatalf("after the cast: %q, want Heal", got)
	}
}

// During a cast a move click gets ActionFailed and waits; the same skill
// gets ActionFailed. After the cast the walk starts.
func TestCastQueueAndMoves(t *testing.T) {
	e := newCastEnv(t)
	sess, p := castEnter(t, e, "mage", classHumanMystic,
		storage.CharacterSkill{ID: 1177, Level: 1}, storage.CharacterSkill{ID: 1011, Level: 1})
	keltir := npcSpawnedOne(t, e, attackKeltir)
	me, kid := p.ObjectID(), int32(keltir.ID())
	attackClick(e, sess, keltir, false)
	castUse(e, sess, 1177, false, false)
	sess.take(t)
	var castTime int
	e.onLoop(func() { castTime = skills.CastTime(p.skill, e.skill(1177, 1)) })

	castUse(e, sess, 1177, false, false)
	checkLines(t, "the same skill", castDescribe(sess.take(t), me, kid), "ActionFailed")
	pos := e.position(p)
	dest := world.Position{X: pos.X + 200, Y: pos.Y, Z: pos.Z}
	e.srv.Handle(sess, clientpackets.OpMoveToLocation, walkRequest(dest, pos, false))
	e.sync()
	checkLines(t, "a move", castDescribe(sess.take(t), me, kid), "ActionFailed")

	got := castStep(t, e, sess, time.Duration(castTime)*time.Millisecond, me, kid)
	if !slices.Contains(got, "MoveToLocation") {
		t.Fatalf("after the cast: %q, want the walk", got)
	}
	var moving bool
	e.onLoop(func() { moving = p.Move != nil })
	if !moving {
		t.Fatal("the player does not walk after the cast")
	}
}

// A toggle goes on with its MP and no cast packets, and goes off with
// ActionFailed when used again; its MP drain (a tick every 5 × 666 ms of
// 0.8 × 3.33 × 19 / 7.5 MP at level 20) ends it when the MP run out.
func TestCastToggle(t *testing.T) {
	e := newCastEnv(t)
	sess, p := castEnterLevel(t, e, "fighter", classHumanFighter, 20, storage.CharacterSkill{ID: 312, Level: 1})
	me := p.ObjectID()
	isOn := func() bool {
		var on bool
		e.onLoop(func() { on = p.Effects().IsAffectedBySkill(312) })
		return on
	}
	var mp0 int
	e.onLoop(func() { mp0 = int(p.CurMP) })
	castUse(e, sess, 312, false, false)
	got := castDescribe(sess.take(t), me, 0)
	checkLines(t, "on", got, fmt.Sprintf("StatusUpdate me 11=%d", mp0-4), "SystemMessage 46 [skill 312/1]",
		"SystemMessage 110 [skill 312/1]") // "The effects of S1 flow through you."
	if !isOn() {
		t.Fatal("Vicious Stance is not on")
	}
	castUse(e, sess, 312, false, false)
	got = castDescribe(sess.take(t), me, 0)
	if !slices.Contains(got, "ActionFailed") || isOn() {
		t.Fatalf("off: %q, on %v", got, isOn())
	}

	castUse(e, sess, 312, false, false)
	if !isOn() {
		t.Fatal("Vicious Stance is not on again")
	}
	e.onLoop(func() { p.SetCurrentMP(1) })
	e.advance(time.Duration(5*666) * time.Millisecond)
	if isOn() {
		t.Error("Vicious Stance outlasted the MP")
	}
}

// A potion is cast in the simultaneous slot during a cast: at once (no hit
// time, no gauge), and the cast goes on to its hit.
func TestCastSimultaneous(t *testing.T) {
	e := newCastEnv(t)
	sess, p := castEnter(t, e, "mage", classHumanMystic, storage.CharacterSkill{ID: 1177, Level: 1})
	keltir := npcSpawnedOne(t, e, attackKeltir)
	me, kid := p.ObjectID(), int32(keltir.ID())
	attackClick(e, sess, keltir, false)
	castUse(e, sess, 1177, false, false)
	sess.take(t)
	potion := e.skill(2031, 1)
	var castTime int
	var casting, sim, healing bool
	e.onLoop(func() {
		castTime = skills.CastTime(p.skill, e.skill(1177, 1))
		e.srv.DoSimultaneousCast(p, potion)
		casting, sim = e.srv.CastingNow(p), e.srv.CastingSimultaneouslyNow(p)
		healing = p.Effects().IsAffectedBySkill(2031)
	})
	if !casting || sim || !healing {
		t.Fatalf("after the potion: casting %v, simultaneous %v, healing %v", casting, sim, healing)
	}
	got := castDescribe(sess.take(t), me, kid)
	if len(got) < 3 || got[0] != "MagicSkillUse me>me 2031/1 0 10000" || got[1] != "MagicSkillLaunched me 2031/1 [me]" ||
		got[2] != "SystemMessage 46 [skill 2031/1]" {
		t.Fatalf("potion packets %q", got)
	}
	got = castStep(t, e, sess, time.Duration(castTime)*time.Millisecond, me, kid)
	if !slices.ContainsFunc(got, func(l string) bool { return strings.HasPrefix(l, "SystemMessage 35 ") }) {
		t.Fatalf("the cast did not hit after the potion: %q", got)
	}
}

// An NPC casts at a player at once when the skill has no hit time, as the
// newbie buffer does (NpcCast): the players around see MagicSkillUse and
// the buff lands.
func TestCastNpcBuff(t *testing.T) {
	e := newCastEnv(t)
	sess, p := castEnter(t, e, "newbie", classHumanFighter)
	keltir := npcSpawnedOne(t, e, attackKeltir)
	me, kid := p.ObjectID(), int32(keltir.ID())
	sk := e.skill(4322, 1) // Wind Walk of the newbie guide
	var buffed bool
	e.onLoop(func() {
		e.srv.NpcCast(keltir, p, sk)
		buffed = p.Effects().IsAffectedBySkill(4322)
	})
	if !buffed {
		t.Fatal("no Wind Walk")
	}
	got := castDescribe(sess.take(t), me, kid)
	if !slices.Contains(got, "MagicSkillUse keltir>me 4322/1 0 0") || !slices.Contains(got, "MagicSkillLaunched keltir 4322/1 [me]") {
		t.Fatalf("packets %q", got)
	}
}
