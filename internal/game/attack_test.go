package game

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// attackTestKeltir is the Bearded Keltir (NPC 20481, npcs/20400-20499.xml)
// as the data has it, without its passive 4416 (F10 is not applied yet) and
// without random walks, so that it stands where the tests put it: 62 HP,
// P.Def 39.6, evasion 34 (stats.TestGoldenFighterVsKeltir).
func attackTestKeltir() data.NpcTemplate {
	return data.NpcTemplate{
		ID: attackKeltir, DisplayID: attackKeltir, Level: 1, Type: "Monster", Name: "Bearded Keltir",
		Race: data.RaceAnimal, Sex: "MALE",
		Acquire: data.NpcAcquire{Exp: 35, SP: 2},
		Stats: data.NpcStats{
			STR: 40, INT: 21, DEX: 30, WIT: 20, CON: 43, MEN: 10,
			Vitals: data.NpcVitals{HP: 39.74519, HPRegen: 2, MP: 40, MPRegen: 0.9},
			Attack: data.NpcAttack{
				Physical: 8.47458, Magical: 5.78704, Random: 30, Critical: 4, Accuracy: 4.75,
				Speed: 253, Type: data.WeaponSword, Range: 40, Distance: 80, Width: 120,
			},
			Defence: data.NpcDefence{Physical: 44.44444, Magical: 29.5916164000214},
			Speed:   data.NpcSpeed{Walk: 23, Run: 110, SwimWalk: 23, SwimRun: 110, FlyWalk: 23, FlyRun: 110},
			HitTime: 370,
		},
		Status: data.NpcStatus{
			Attackable: true, Targetable: true, Talkable: true, ShowName: true,
			RandomAnimation: true, CanMove: true, CanBeSown: true,
		},
		Collision: data.NpcCollision{Radius: 9.5, Height: 10},
		AI:        data.NpcAI{Type: "FIGHTER", AggroRange: 1000, ClanHelpRange: 300, MinSkillChance: 7, MaxSkillChance: 15},
	}
}

const (
	attackKeltir = 20481
	// attackKeltirDistance is how far west of the start the Keltir stands,
	// facing east: within a sword's reach (40 + 9 + 9), out of the fists'
	// (0 + 9 + 9).
	attackKeltirDistance = 50
)

// Sword swing draws (Mobius doAttackHitSimple): the hit roll, the crit
// roll, the two random damage factors of the Squire's Sword (randomDamage
// 10). 0 always hits; 999 never crits.
func attackSwordHit(factor int) []rnd.Draw {
	return []rnd.Draw{rnd.Get(1000, 0), rnd.Get(1000, 999), rnd.Between(-10, 10, factor), rnd.Between(-10, 10, factor)}
}

// newAttackEnv starts a server on npcTestWorld with a Bearded Keltir 50
// west of the start, facing it. The swings draw from src; the rest of the
// game from a fixed seed. The Keltir never swings back: the monster AI is
// not this feature's, and its swings would draw from src too.
func newAttackEnv(t *testing.T, src rnd.Source, configure func(gs *config.GameServer)) *testEnv {
	t.Helper()
	start := npcTestStart(t)
	wd := npcTestWorld(start)
	wd.Npcs = append(wd.Npcs, attackTestKeltir())
	// The Keltir alone: the other monsters of the test world wander.
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
	if configure != nil {
		configure(gs)
	}
	clock := world.NewFakeClock(testStart)
	ids := memstore.NewIDs()
	store := newRecordingStore()
	srv, err := New(Options{
		Config: cfg, ServerID: gs.ID, Data: testGameData(t), World: w,
		Characters: store, IDs: ids, Clock: clock, CheckLoopOwner: true,
		Rand: rnd.New(1),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	srv.attack.rnd = src
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
	return &testEnv{t: t, ctx: ctx, srv: srv, store: store, ids: ids, clock: clock, data: testGameData(t), stop: stop}
}

// attackEnter enters a level 1 Human Fighter (unarmed when asked) and lets
// 5 s pass, so that the regeneration tasks of the region activation are
// over, then drops the packets.
func attackEnter(t *testing.T, e *testEnv, account, name string, unarmed bool) (*fakeSession, *Player) {
	t.Helper()
	c := e.createCharacter(account, name, classHumanFighter, storage.AccessLevelPlayer)
	if unarmed {
		for i := range c.Items {
			if c.Items[i].ItemID == itemSquireSword {
				c.Items[i].Slot = storage.PaperdollNone
			}
		}
		if err := e.store.Update(e.ctx, &c); err != nil {
			t.Fatalf("Update: %v", err)
		}
	}
	sess := newSession(account)
	p := e.enter(sess, 0)
	e.advance(5 * time.Second)
	sess.take(t)
	return sess, p
}

// attackClick sends Action (attack false) or AttackRequest (attack true) on
// o and waits for the loop.
func attackClick(e *testEnv, sess *fakeSession, o world.Object, attack bool) {
	op := byte(clientpackets.OpAction)
	if attack {
		op = clientpackets.OpAttackRequest
	}
	e.srv.Handle(sess, op, npcActionBody(int32(o.ID()), false))
	e.sync()
}

// attackOwn drops the packets the monster AI sends about the NPC with the
// given object ID (its stance, run and walks), which are not this feature's.
func attackOwn(ps []testclient.Packet, npc int32) []testclient.Packet {
	return slices.DeleteFunc(slices.Clone(ps), func(p testclient.Packet) bool {
		switch v := p.(type) {
		case testclient.AutoAttackStart:
			return v.ObjectID == npc
		case testclient.AutoAttackStop:
			return v.ObjectID == npc
		case testclient.ChangeMoveType:
			return v.ObjectID == npc
		case testclient.MoveToPawn:
			return v.ObjectID == npc
		case testclient.MoveToLocation:
			return v.ObjectID == npc
		case testclient.StopMove:
			return v.ObjectID == npc
		case testclient.NpcInfo:
			return v.ObjectID == npc
		case testclient.SocialAction:
			return v.ObjectID == npc
		}
		return false
	})
}

// attackDescribe writes the packets as short lines for comparisons: the
// object IDs of the player and the Keltir become "me" and "keltir".
func attackDescribe(ps []testclient.Packet, me, keltir int32) []string {
	who := func(id int32) string {
		switch id {
		case me:
			return "me"
		case keltir:
			return "keltir"
		}
		return fmt.Sprint(id)
	}
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		switch v := p.(type) {
		case testclient.Attack:
			hits := make([]string, len(v.Hits))
			for i, h := range v.Hits {
				hits[i] = fmt.Sprintf("%s %d", who(h.TargetID), h.Damage)
				if h.Miss() {
					hits[i] += " miss"
				}
				if h.Critical() {
					hits[i] += " crit"
				}
			}
			out = append(out, fmt.Sprintf("Attack %s [%s]", who(v.AttackerID), strings.Join(hits, ", ")))
		case testclient.SystemMessage:
			params := make([]string, len(v.Params))
			for i, prm := range v.Params {
				if prm.Text != "" {
					params[i] = prm.Text
				} else {
					params[i] = fmt.Sprint(prm.Number)
				}
			}
			out = append(out, fmt.Sprintf("SystemMessage %d %v", v.ID, params))
		case testclient.StatusUpdate:
			hp, _ := v.Value(testclient.StatusCurHP)
			out = append(out, fmt.Sprintf("StatusUpdate %s HP %d", who(v.ObjectID), hp))
		case testclient.AutoAttackStart:
			out = append(out, "AutoAttackStart "+who(v.ObjectID))
		case testclient.AutoAttackStop:
			out = append(out, "AutoAttackStop "+who(v.ObjectID))
		case testclient.MoveToPawn:
			out = append(out, fmt.Sprintf("MoveToPawn %s→%s %d", who(v.ObjectID), who(v.TargetID), v.Distance))
		default:
			out = append(out, p.PacketName())
		}
	}
	return out
}

// checkLines fails unless got equals want.
func checkLines(t *testing.T, what string, got []string, want ...string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("%s:\n got %q\nwant %q", what, got, want)
	}
}

// attackStep advances the clock by d and returns what the player got,
// described.
func attackStep(t *testing.T, e *testEnv, sess *fakeSession, d time.Duration, me, keltir int32) []string {
	t.Helper()
	e.advance(d)
	return attackDescribe(attackOwn(sess.take(t), keltir), me, keltir)
}

// Done when: a level 1 Human Fighter kills a Bearded Keltir with its
// Squire's Sword, every hit 76 × 6.48 / 39.6 × 1.1 × 1.1 = 15. The packets
// are exact up to the HP bar at 0, Mobius's order: the swing (Attack), its
// hit half a swing later (timeAtk 1,199 at speed 417: the hit at 599, the
// next swing at 1,199), the stance on the first hit, the Keltir's
// regeneration 3 s after its first wound (+2 HP). Death, rewards and drops
// that follow belong to the death features; here Server.Kill must have run
// with the player as the killer. The stance ends 15-16 s after the last
// blow.
func TestAttackKillsBeardedKeltir(t *testing.T) {
	var draws []rnd.Draw
	for range 5 {
		draws = append(draws, attackSwordHit(10)...)
	}
	src := rnd.NewScripted(t, draws...)
	e := newAttackEnv(t, src, nil)
	var kills []string
	e.onLoop(func() {
		e.srv.OnDie(func(victim, killer Creature) {
			if killer != nil {
				kills = append(kills, victim.Name()+" by "+killer.Name())
			}
		})
	})
	sess, p := attackEnter(t, e, "alice", "Alice", false)
	keltir := npcSpawnedOne(t, e, attackKeltir)
	me, kid := p.ObjectID(), int32(keltir.ID())

	attackClick(e, sess, keltir, false)
	checkLines(t, "the first click", attackDescribe(attackOwn(sess.take(t), kid), me, kid),
		"ValidateLocation", "MyTargetSelected", "StatusUpdate keltir HP 62", "ActionFailed")

	attackClick(e, sess, keltir, false)
	checkLines(t, "the second click", attackDescribe(attackOwn(sess.take(t), kid), me, kid),
		"Attack me [keltir 15]", "ActionFailed")

	ms := time.Millisecond
	hit := func(hp int) []string {
		return []string{"SystemMessage 35 [15]", fmt.Sprintf("StatusUpdate keltir HP %d", hp)}
	}
	swing := []string{"Attack me [keltir 15]"}
	// at is the time since the second click, in ms.
	steps := []struct {
		at   int
		want []string
	}{
		{598, nil},
		{599, append(hit(47), "AutoAttackStart me")},
		{1198, nil},
		{1199, swing},
		{1798, hit(32)},
		{2398, swing},
		{2997, hit(17)},
		{3597, swing},
		{3599, []string{"StatusUpdate keltir HP 19"}}, // regeneration, 3 s after the first wound
		{4196, hit(4)},
		{4796, swing},
		{5394, nil},
		{5395, hit(0)}, // dead
	}
	last := 0
	for i, st := range steps {
		got := attackStep(t, e, sess, time.Duration(st.at-last)*ms, me, kid)
		last = st.at
		if i == len(steps)-1 {
			// The death features send more after the HP bar.
			got = got[:min(len(got), len(st.want))]
		}
		checkLines(t, fmt.Sprintf("at %d ms", st.at), got, st.want...)
	}
	src.Done()
	e.onLoop(func() {
		if !keltir.IsDead() || !slices.Equal(kills, []string{"Bearded Keltir by Alice"}) {
			t.Errorf("Keltir dead %v, kills %q", keltir.IsDead(), kills)
		}
	})
	sess.take(t)

	// The ready time at 5,995 finds the target dead: no swing, the player
	// is idle.
	if got := attackStep(t, e, sess, 600*ms, me, kid); slices.ContainsFunc(got, func(s string) bool { return strings.HasPrefix(s, "Attack ") }) {
		t.Errorf("after the kill: %q", got)
	}
	e.onLoop(func() {
		if e.srv.AttackTarget(p) != nil || !e.srv.InAttackStance(p) {
			t.Errorf("attack target %v, stance %v; want none and in stance", e.srv.AttackTarget(p), e.srv.InAttackStance(p))
		}
	})

	// The last blow was at 5,395: the stance ends between 20,395 and
	// 21,395, at the second's scan after it.
	e.advance(20395*ms - 5995*ms)
	if got := attackStep(t, e, sess, 0, me, kid); slices.Contains(got, "AutoAttackStop me") {
		t.Fatalf("the stance ended at 15 s: %q", got)
	}
	e.advance(time.Second)
	got := attackDescribe(attackOwn(sess.take(t), kid), me, kid)
	if !slices.Contains(got, "AutoAttackStop me") {
		t.Fatalf("no AutoAttackStop within 16 s: %q", got)
	}
	e.onLoop(func() {
		if e.srv.InAttackStance(p) {
			t.Error("still in stance after AutoAttackStop")
		}
	})
}

// Unarmed, a player swings its class fists twice per swing (Mobius sends
// player FIST down the dual path): at speed 330 the swing lasts 1,515 ms,
// the hits land at 378 and 757 for half damage, (int)(76 × 4.32 / 39.6) / 2
// = 4. Out of reach (0 + 9 + 9 < 50) it first runs up to the Keltir: the
// attack follow walks it 5 short of the Keltir (MoveToPawn, offset 10), and
// on arrival it swings.
func TestAttackUnarmedFollowsAndHitsTwice(t *testing.T) {
	src := rnd.NewScripted(t,
		rnd.Get(1000, 0), rnd.Get(1000, 0), // both hits land
		rnd.Get(1000, 999), rnd.Get(1000, 999), // no crits, no random damage (r = 0)
		rnd.Get(1000, 0), rnd.Get(1000, 801), // the second swing: a hit and a miss
		rnd.Get(1000, 999),
	)
	e := newAttackEnv(t, src, nil)
	sess, p := attackEnter(t, e, "alice", "Alice", true)
	keltir := npcSpawnedOne(t, e, attackKeltir)
	me, kid := p.ObjectID(), int32(keltir.ID())
	attackClick(e, sess, keltir, false)
	sess.take(t)

	attackClick(e, sess, keltir, false)
	checkLines(t, "the second click", attackDescribe(attackOwn(sess.take(t), kid), me, kid),
		"MoveToPawn me→keltir 10", "ActionFailed")

	// The run: 45 units at 126 (less the collision of 9) take a few ticks.
	var got []string
	for range 20 {
		got = attackStep(t, e, sess, 100*time.Millisecond, me, kid)
		if len(got) > 0 {
			break
		}
	}
	checkLines(t, "the arrival", got, "MoveToLocation", "StopMove", "Attack me [keltir 4, keltir 4]")
	if pos, kpos := e.position(p), npcPosition(e, keltir); pos.X-kpos.X != 5 || pos.Y != kpos.Y {
		t.Errorf("the player stopped at %v, the Keltir stands at %v; want 5 east of it", pos, kpos)
	}

	ms := time.Millisecond
	checkLines(t, "377 ms", attackStep(t, e, sess, 377*ms, me, kid))
	checkLines(t, "378 ms", attackStep(t, e, sess, 1*ms, me, kid),
		"SystemMessage 35 [4]", "StatusUpdate keltir HP 58", "AutoAttackStart me")
	checkLines(t, "757 ms", attackStep(t, e, sess, 379*ms, me, kid),
		"SystemMessage 35 [4]", "StatusUpdate keltir HP 54")
	checkLines(t, "1,514 ms", attackStep(t, e, sess, 757*ms, me, kid))
	checkLines(t, "1,515 ms", attackStep(t, e, sess, 1*ms, me, kid), "Attack me [keltir 4, keltir 0 miss]")
	checkLines(t, "1,893 ms", attackStep(t, e, sess, 378*ms, me, kid), "SystemMessage 35 [4]", "StatusUpdate keltir HP 50")
	checkLines(t, "2,272 ms", attackStep(t, e, sess, 379*ms, me, kid), "SystemMessage 43 []")
	src.Done()
}

// A move click during a swing gets ActionFailed and waits for the swing's
// ready time; then the player walks there instead of swinging again
// (Mobius PlayerAI's saved MOVE_TO).
func TestAttackMoveDuringSwingWaits(t *testing.T) {
	src := rnd.NewScripted(t, attackSwordHit(0)...)
	e := newAttackEnv(t, src, nil)
	sess, p := attackEnter(t, e, "alice", "Alice", false)
	keltir := npcSpawnedOne(t, e, attackKeltir)
	me, kid := p.ObjectID(), int32(keltir.ID())
	attackClick(e, sess, keltir, false)
	attackClick(e, sess, keltir, false)
	sess.take(t)

	e.advance(100 * time.Millisecond)
	from := e.position(p)
	walkClick(e, sess, from, walkOffset(from, 300, 0, 0))
	checkLines(t, "the click during the swing", attackDescribe(attackOwn(sess.take(t), kid), me, kid), "ActionFailed")
	if walking, _ := walkState(e, p); walking {
		t.Fatal("the player walks during its swing")
	}
	// The hit (12) at 599, then the ready time at 1,199 starts the walk.
	checkLines(t, "the hit", attackStep(t, e, sess, 499*time.Millisecond, me, kid),
		"SystemMessage 35 [12]", "StatusUpdate keltir HP 50", "AutoAttackStart me")
	checkLines(t, "the ready time", attackStep(t, e, sess, 600*time.Millisecond, me, kid), "MoveToLocation")
	if walking, _ := walkState(e, p); !walking {
		t.Fatal("the saved click did not start a walk")
	}
	e.onLoop(func() {
		if e.srv.AttackTarget(p) != nil {
			t.Error("the walk kept the attack")
		}
	})
	src.Done()
}

// Every refused attack ends with ActionFailed, so that the client does not
// wait.
func TestAttackRefusalsSendActionFailed(t *testing.T) {
	e := newAttackEnv(t, rnd.NewScripted(t), nil)
	sa, sb, pa, pb := attackTwoPlayers(t, e)
	keltir := npcSpawnedOne(t, e, attackKeltir)
	aid, kid := pa.ObjectID(), int32(keltir.ID())
	failed := func(what string, want ...string) {
		t.Helper()
		checkLines(t, what, attackDescribe(attackOwn(sa.take(t), kid), aid, kid), want...)
	}

	// An AttackRequest on nothing.
	e.srv.Handle(sa, clientpackets.OpAttackRequest, npcActionBody(0x7fff0000, false))
	e.sync()
	failed("an unknown object", "ActionFailed")

	// A second click while sitting (Mobius REST).
	attackClick(e, sa, keltir, false)
	sa.take(t)
	e.onLoop(func() { pa.Sitting = true })
	attackClick(e, sa, keltir, false)
	failed("sitting", "ActionFailed", "ActionFailed")
	e.onLoop(func() { pa.Sitting = false })

	// A dead Keltir is not attacked.
	e.onLoop(func() { e.srv.Kill(keltir, nil) })
	sa.take(t)
	attackClick(e, sa, keltir, false)
	failed("a dead monster", "ActionFailed")

	// A plain second click on a player that is not fair game.
	attackClick(e, sa, pb, false)
	sa.take(t)
	attackClick(e, sa, pb, false)
	failed("a karma-free player", "ActionFailed", "ActionFailed")

	// Ctrl+click on a player in a peace zone (F13).
	e.onLoop(func() { pb.Zones.Enter(ZonePeace) })
	attackClick(e, sa, pb, true)
	failed("a player in a peace zone", "SystemMessage 85 []", "ActionFailed")
	sb.take(t)
}

// attackTwoPlayers enters Alice and Bob with their swords at the start and
// lets 5 s pass.
func attackTwoPlayers(t *testing.T, e *testEnv) (sa, sb *fakeSession, pa, pb *Player) {
	t.Helper()
	e.createCharacter("alice", "Alice", classHumanFighter, storage.AccessLevelPlayer)
	e.createCharacter("bob", "Bob", classHumanFighter, storage.AccessLevelPlayer)
	sa, sb = newSession("alice"), newSession("bob")
	pa = e.enter(sa, 0)
	pb = e.enter(sb, 0)
	e.advance(5 * time.Second)
	sa.take(t)
	sb.take(t)
	return sa, sb, pa, pb
}

// Ctrl+click on a karma-free player outside peace zones attacks it: its CP
// take the damage first, it hears "Alice hit you for N damage." and both
// enter their stance. Then neither may restart or log out until the stance
// ends (F14); a lost connection leaves the player in the world 15 s more.
func TestAttackPvPAndLeavingInCombat(t *testing.T) {
	src := rnd.NewScripted(t, attackSwordHit(0)...)
	e := newAttackEnv(t, src, nil)
	sa, sb, pa, pb := attackTwoPlayers(t, e)
	aid, bid := pa.ObjectID(), pb.ObjectID()
	attackClick(e, sa, pb, false)
	sa.take(t)
	attackClick(e, sa, pb, true)
	got := attackDescribe(sa.take(t), aid, 0)
	if len(got) == 0 || !strings.HasPrefix(got[0], fmt.Sprintf("Attack me [%d ", bid)) {
		t.Fatalf("the forced attack: %q", got)
	}
	attacks := only[testclient.Attack](sb.take(t))
	if len(attacks) != 1 || len(attacks[0].Hits) != 1 {
		t.Fatalf("Bob saw %d swings", len(attacks))
	}
	damage := attacks[0].Hits[0].Damage
	if damage <= 0 {
		t.Fatalf("damage %d", damage)
	}

	e.advance(599 * time.Millisecond)
	bobGot := sb.take(t)
	if starts := only[testclient.AutoAttackStart](bobGot); len(starts) != 2 || starts[0].ObjectID != bid || starts[1].ObjectID != aid {
		t.Errorf("Bob's stances %+v, want his own, then Alice's", starts)
	}
	msgs := only[testclient.SystemMessage](bobGot)
	if len(msgs) != 1 || msgs[0].ID != int32(sysMsgHitYouForDamage) || msgs[0].Params[1].Number != int64(damage) {
		t.Errorf("Bob's messages %+v, want Alice hit you for %d", msgs, damage)
	}
	var cp float64
	e.onLoop(func() {
		cp = pb.CurCP
		e.srv.IdleIntention(pa) // no second swing
	})
	if cp != 50-float64(damage) {
		t.Errorf("Bob's CP %v, want 50 − %d", cp, damage)
	}
	src.Done()
	sa.take(t)

	// In combat: restart and logout are refused.
	e.srv.Restart(sa)
	checkLines(t, "restart in combat", attackDescribe(sa.take(t), aid, 0),
		"SystemMessage 102 []", "RestartResponse", "ActionFailed")
	e.srv.Logout(sa)
	checkLines(t, "logout in combat", attackDescribe(sa.take(t), aid, 0), "SystemMessage 101 []", "ActionFailed")
	if _, _, closed := sa.closedWith(t); closed {
		t.Fatal("the logout closed the connection")
	}

	// Bob loses his connection: he stays 15 s.
	e.srv.Disconnected(sb)
	if !e.inWorld(pb) {
		t.Fatal("Bob left the world in combat")
	}
	e.advance(attackStanceTime - time.Millisecond)
	if !e.inWorld(pb) {
		t.Fatal("Bob left before 15 s")
	}
	e.advance(time.Millisecond)
	if e.inWorld(pb) {
		t.Fatal("Bob is still in the world after 15 s")
	}

	// The stance from the hit at 599 ms ends by 16.6 s: Alice may log out.
	e.advance(2 * time.Second)
	sa.take(t)
	e.srv.Logout(sa)
	if _, _, closed := sa.closedWith(t); !closed {
		t.Fatalf("the logout after the stance: %q", testclient.Names(sa.take(t)))
	}
}

// A bow needs arrows: without them the swing is refused with "You have run
// out of arrows." and the attack ends. With arrows the bow equips them,
// costs 1 MP and an arrow per shot, shows the red gauge (timeAtk + reuse)
// and "You carefully nock an arrow...", hits at the end of the swing
// (timeAtk) and shoots again at timeAtk + reuse (Mobius doAttackHitByBow).
func TestAttackBow(t *testing.T) {
	src := rnd.NewScripted(t,
		rnd.Get(1000, 0), rnd.Get(1000, 999), rnd.Between(-5, 5, 0), rnd.Between(-5, 5, 0),
		rnd.Get(1000, 0), rnd.Get(1000, 999), rnd.Between(-5, 5, 0), rnd.Between(-5, 5, 0),
	)
	e := newAttackEnv(t, src, nil)
	sess, p := attackEnter(t, e, "alice", "Alice", false)
	keltir := npcSpawnedOne(t, e, attackKeltir)
	me, kid := p.ObjectID(), int32(keltir.ID())
	var timing stats.SwingTiming
	e.onLoop(func() {
		bow, err := e.srv.AddItem(p, 14, 1, 0, false, false)
		if err != nil {
			t.Fatalf("AddItem: %v", err)
		}
		e.srv.useEquippableItem(p, bow)
		timing = stats.SwingTimingOf(p.stats.PAtkSpd(), stats.ShapeBow, 1500)
	})
	e.advance(time.Second)
	attackClick(e, sess, keltir, false)
	sess.take(t)

	attackClick(e, sess, keltir, false)
	checkLines(t, "no arrows", attackDescribe(attackOwn(sess.take(t), kid), me, kid),
		"ActionFailed", "SystemMessage 112 []", "ActionFailed")

	e.onLoop(func() {
		if _, err := e.srv.AddItem(p, 17, 10, 0, false, false); err != nil {
			t.Fatalf("AddItem: %v", err)
		}
	})
	e.advance(time.Second)
	sess.take(t)
	attackClick(e, sess, keltir, false)
	e.advance(world.DefaultTickInterval) // the coalesced updates
	ps := attackOwn(sess.take(t), kid)
	got := attackDescribe(ps, me, kid)
	var order []string
	for _, s := range got {
		if s == "ItemList" || s == "SetupGauge" || s == "ActionFailed" || strings.HasPrefix(s, "Attack ") || strings.HasPrefix(s, "SystemMessage") {
			order = append(order, s)
		}
	}
	// 76 × P.Atk 23 × 1.2 × 0.9 / 39.6 = 47.67.
	checkLines(t, "the first shot", order, "ItemList", "SystemMessage 41 []", "SetupGauge", "Attack me [keltir 47]", "ActionFailed")
	gauge, _ := testclient.Find[testclient.SetupGauge](ps)
	if want := int32(timing.Gauge / time.Millisecond); gauge.Color != 1 || gauge.Time != want || gauge.MaxTime != want {
		t.Errorf("SetupGauge %+v, want red, %d ms", gauge, want)
	}
	var arrows int64
	var mp float64
	e.onLoop(func() {
		if it := p.inv.Paperdoll(storage.PaperdollLHand); it != nil {
			arrows = it.Count()
		}
		mp = p.CurMP
	})
	if arrows != 9 || mp != 37 {
		t.Errorf("after a shot: %d arrows, %v MP; want 9 and 37", arrows, mp)
	}

	// The hit at timeAtk, the next shot at timeAtk + reuse.
	tick := world.DefaultTickInterval
	e.advance(timing.Interval - tick - time.Millisecond)
	if got := attackDescribe(attackOwn(sess.take(t), kid), me, kid); slices.ContainsFunc(got, func(s string) bool { return strings.HasPrefix(s, "SystemMessage 35") }) {
		t.Fatalf("the arrow hit early: %q", got)
	}
	e.advance(time.Millisecond)
	if got := attackDescribe(attackOwn(sess.take(t), kid), me, kid); !slices.Contains(got, "SystemMessage 35 [47]") {
		t.Fatalf("the arrow at timeAtk: %q", got)
	}
	e.advance(timing.Reuse - time.Millisecond)
	if got := attackDescribe(attackOwn(sess.take(t), kid), me, kid); slices.ContainsFunc(got, func(s string) bool { return strings.HasPrefix(s, "Attack ") }) {
		t.Fatalf("the second shot came early: %q", got)
	}
	e.advance(time.Millisecond)
	if got := attackDescribe(attackOwn(sess.take(t), kid), me, kid); !slices.Contains(got, "Attack me [keltir 47]") {
		t.Fatalf("no second shot at timeAtk + reuse: %q", got)
	}
	src.Done()
}
