package game

import (
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// Monsters of the data/ world in the NPC death tests.
const (
	npcWolf          = 20120 // Wolf, level 4: 141 exp, 4 SP
	npcBeardedKeltir = 20481 // Bearded Keltir, level 1: 35 exp, 2 SP
	itemVarnish      = 1865
)

// dropRand is a random source for the drop tests: Float64, Between and Get
// answer from their scripts (0.99, lo and 0 once a script ends) and record
// every draw, so a test can check Mobius's draw order. The other methods
// fall back to a seeded PCG.
type dropRand struct {
	rnd.Source
	mu       sync.Mutex
	floats   []float64
	betweens []int
	gets     []int
	calls    []string
}

func (r *dropRand) Float64() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, "Float64")
	if len(r.floats) == 0 {
		return 0.99
	}
	v := r.floats[0]
	r.floats = r.floats[1:]
	return v
}

func (r *dropRand) Between(lo, hi int) int {
	if lo >= hi {
		return lo
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, fmt.Sprintf("Between(%d, %d)", lo, hi))
	if len(r.betweens) == 0 {
		return lo
	}
	v := r.betweens[0]
	r.betweens = r.betweens[1:]
	return v
}

func (r *dropRand) Get(n int) int {
	if n <= 0 {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, fmt.Sprintf("Get(%d)", n))
	if len(r.gets) == 0 {
		return 0
	}
	v := r.gets[0]
	r.gets = r.gets[1:]
	return v
}

func (r *dropRand) drawn() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.calls)
}

// wolfKillRand scripts a Wolf kill by a level 1 player (no level gap):
// adena 25 drops from the first group, Varnish from the third; the fourth
// group's first item puts the adena aside (the occurrence limit of 2 is
// reached) and, as nothing else drops, it comes back last. Varnish falls at
// (-60, +50), the adena at (0, -70) from the corpse. The third group stops
// at its drop, Varnish, its second item.
func wolfKillRand() *dropRand {
	return &dropRand{
		Source: rnd.New(1),
		// adena: gap, drop; 13: gap, drop; 1786: gap, drop; 1865: gap, drop.
		floats:   []float64{0.5, 0.1, 0.99, 0.99, 0.99, 0.99, 0.99, 0.001},
		betweens: []int{25},
		// The two cursed weapon rolls (no drop), then x and y of each drop.
		gets: []int{50000, 50000, 10, 120, 70, 0},
	}
}

// wolfKillDraws are the draws of wolfKillRand's kill: the two cursed weapon
// rolls (cursedweapon.go), two Float64 (level gap, drop) per item rolled in
// the nine groups (one item each in the first two and the last, two of the
// third's four, three or two in the others), the adena amount, then x and y
// of each drop.
func wolfKillDraws() []string {
	want := []string{"Get(100000)", "Get(100000)"}
	f := func(n int) {
		for range n {
			want = append(want, "Float64")
		}
	}
	f(2)
	want = append(want, "Between(19, 29)")
	f(2 + 2*2 + 3*2 + 3*2 + 3*2 + 2*2 + 3*2 + 2)
	return append(want, "Get(141)", "Get(141)", "Get(141)", "Get(141)")
}

// killNpc kills n by p on the loop with r as the random source: p's damage
// is n's HP (Kill after the hate a hit adds), so that only the death draws.
func (e *testEnv) killNpc(r rnd.Source, n *Npc, p *Player) {
	e.t.Helper()
	e.onLoop(func() {
		e.srv.addDamageHate(n, p, int64(n.CurHP), 1)
	})
	e.withRand(r, func() {
		if !e.srv.Kill(n, p) {
			e.t.Errorf("Kill(%s) found the NPC dead", n.Name())
		}
	})
}

// A Wolf killed by a level 1 player with scripted draws: the drops are
// exact, protected for the killer and fall in Mobius's packet order
// (StatusUpdate, DropItem, the exp, Die); the level-up to 2 fills HP, MP and
// CP with SocialAction 15 and its message.
func TestNpcKillWolfDropsAndLevelUp(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{})
	sess, p := e.enterDeathChar(deathChar{account: "alice", name: "Alice"})
	wolf := e.spawnNpc(npcWolf, eastOf(tiStart, 100))
	e.onLoop(func() {
		e.srv.SetTarget(p, wolf)
		p.SetCurrentHP(10)
		p.SetCurrentMP(1)
		p.SetCurrentCP(1)
	})
	e.sync()
	sess.take(t)

	r := wolfKillRand()
	killed := e.clock.Now()
	e.killNpc(r, wolf, p)
	e.advance(100 * time.Millisecond) // the UserInfo goes at the end of the tick
	ps := sess.take(t)
	if got, want := r.drawn(), wolfKillDraws(); !slices.Equal(got, want) {
		t.Errorf("draws\n got %q\nwant %q", got, want)
	}

	pos := wolf.Position()
	drops := only[testclient.DropItem](ps)
	want := []testclient.DropItem{
		{DropperID: int32(wolf.ID()), ItemID: itemVarnish, X: pos.X - 60, Y: pos.Y + 50, Z: pos.Z + 20, Stackable: true, Count: 1, Unknown: 1},
		{DropperID: int32(wolf.ID()), ItemID: adenaID, X: pos.X, Y: pos.Y - 70, Z: pos.Z + 20, Stackable: true, Count: 25, Unknown: 1},
	}
	if len(drops) != len(want) {
		t.Fatalf("got %d DropItem in %q, want %d", len(drops), testclient.Names(ps), len(want))
	}
	for i := range want {
		want[i].ObjectID = drops[i].ObjectID
		if drops[i] != want[i] {
			t.Errorf("drop %d: %+v, want %+v", i, drops[i], want[i])
		}
	}
	e.onLoop(func() {
		for _, d := range drops {
			o, ok := e.srv.world.Find(world.ObjectID(d.ObjectID))
			g, _ := o.(*GroundItem)
			if !ok || g == nil {
				t.Errorf("drop %d is not on the ground", d.ObjectID)
				continue
			}
			if g.protectedFor != p.ID() || !g.protectedUntil.Equal(killed.Add(15*time.Second)) {
				t.Errorf("drop %d protected for %d until %v, want the killer for 15 s", d.ObjectID, g.protectedFor, g.protectedUntil)
			}
		}
	})

	// Mobius order: the HP bar at 0, the drops, the exp, the second HP bar,
	// Die.
	idx := func(match func(testclient.Packet) bool) int { return slices.IndexFunc(ps, match) }
	firstBar := idx(func(q testclient.Packet) bool {
		s, ok := q.(testclient.StatusUpdate)
		return ok && s.ObjectID == int32(wolf.ID())
	})
	firstDrop := idx(func(q testclient.Packet) bool { _, ok := q.(testclient.DropItem); return ok })
	earned := idx(func(q testclient.Packet) bool {
		m, ok := q.(testclient.SystemMessage)
		return ok && m.ID == int32(sysMsgEarnedExpAndSP)
	})
	die := idx(func(q testclient.Packet) bool {
		d, ok := q.(testclient.Die)
		return ok && d.ObjectID == int32(wolf.ID())
	})
	if firstBar < 0 || firstDrop < firstBar || earned < firstDrop || die < earned {
		t.Errorf("packet order %q: HP bar %d, drop %d, exp %d, Die %d", testclient.Names(ps), firstBar, firstDrop, earned, die)
	}
	bars := 0
	for _, s := range only[testclient.StatusUpdate](ps) {
		if s.ObjectID == int32(wolf.ID()) {
			bars++
		}
	}
	if bars != 2 {
		t.Errorf("got %d HP bars of the wolf, want 2 (Kill and doDie)", bars)
	}
	if d := ps[die].(testclient.Die); !d.CanTeleport || d.Sweepable || d.FixedRes {
		t.Errorf("Die %+v, want only CanTeleport", d)
	}
	if m := ps[earned].(testclient.SystemMessage); len(m.Params) != 2 || m.Params[0].Number != 141 || m.Params[1].Number != 4 {
		t.Errorf("earned message %+v, want 141 exp and 4 SP", m)
	}

	// Level 1 → 2: full HP, MP and CP, SocialAction 15, "Your level has
	// increased!".
	var level int
	var exp int64
	var sp int32
	var hp, mp, cp, maxHP, maxMP, maxCP float64
	e.onLoop(func() {
		level, exp, sp, hp, mp, cp = p.Level(), p.Exp(), p.SP(), p.CurHP, p.CurMP, p.CurCP
		v := p.stats.View()
		maxHP, maxMP, maxCP = float64(v.MaxHP), float64(v.MaxMP), float64(v.MaxCP)
	})
	if level != 2 || exp != 141 || sp != 4 {
		t.Errorf("level %d, exp %d, SP %d; want 2, 141, 4", level, exp, sp)
	}
	if hp != maxHP || mp != maxMP || cp != maxCP {
		t.Errorf("HP/MP/CP %v/%v/%v after the level-up, want the level 2 maxima %v/%v/%v", hp, mp, cp, maxHP, maxMP, maxCP)
	}
	social := false
	for _, s := range only[testclient.SocialAction](ps) {
		social = social || (s.ObjectID == p.ObjectID() && s.ActionID == 15)
	}
	if !social || !slices.Contains(sysMsgIDsOf(ps), int32(sysMsgLevelIncreased)) {
		t.Errorf("level-up packets missing in %q", testclient.Names(ps))
	}
	if uis := only[testclient.UserInfo](ps); len(uis) == 0 || uis[len(uis)-1].Level != 2 {
		t.Errorf("no UserInfo of level 2 in %q", testclient.Names(ps))
	}
}

// With autoLoot, the Wolf's adena and Varnish go into the killer's
// inventory: no item falls, no position is drawn.
func TestNpcKillAutoLoot(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{configure: func(gs *config.GameServer) {
		gs.Combat = config.DefaultCombat()
		gs.Combat.AutoLoot = true
	}})
	sess, p := e.enterDeathChar(deathChar{account: "alice", name: "Alice"})
	wolf := e.spawnNpc(npcWolf, eastOf(tiStart, 100))
	r := wolfKillRand()
	e.killNpc(r, wolf, p)
	e.sync()
	ps := sess.take(t)
	if n := len(only[testclient.DropItem](ps)); n != 0 {
		t.Errorf("got %d DropItem with autoLoot", n)
	}
	if got := r.drawn(); slices.Contains(got, "Get(141)") {
		t.Errorf("drop positions drawn with autoLoot: %q", got)
	}
	var adena, varnish int64
	e.onLoop(func() {
		adena = p.inv.Adena()
		if it := p.inv.ByItemID(itemVarnish); it != nil {
			varnish = it.Count()
		}
	})
	if adena != 25 || varnish != 1 {
		t.Errorf("adena %d, Varnish %d; want 25 and 1", adena, varnish)
	}
}

// talkingIslandMonsters returns the templates of the monsters of
// TalkingIsland/TalkingIslandMonsters.xml, by ID.
func talkingIslandMonsters(t *testing.T, e *testEnv) []int {
	t.Helper()
	w, err := chatTestWorld()
	if err != nil {
		t.Fatalf("load the world data: %v", err)
	}
	var ids []int
	for _, b := range w.Spawns {
		if b.File != "TalkingIsland/TalkingIslandMonsters.xml" {
			continue
		}
		for _, n := range b.Npcs {
			if !slices.Contains(ids, n.NpcID) {
				ids = append(ids, n.NpcID)
			}
		}
	}
	slices.Sort(ids)
	var out []int
	e.onLoop(func() {
		for _, id := range ids {
			if tm, ok := e.srv.worldData.Npc(id); ok && e.srv.npcClassOf(tm).monster {
				out = append(out, id)
			}
		}
	})
	if len(out) < 10 {
		t.Fatalf("only %d Talking Island monsters: %v", len(out), out)
	}
	return out
}

// setPlayerLevel puts p at the start of level on the loop.
func (e *testEnv) setPlayerLevel(p *Player, level int) {
	e.srv.AddLevel(p, level-p.Level())
	p.char.Exp = e.srv.ExpForLevel(level)
}

// Every Talking Island monster killed alone at levels 1-10 gives its
// template exp and SP at x1, and nothing 11 or more levels apart
// (MonsterExpMaxLevelDifference).
func TestNpcKillExpOfTalkingIslandMonsters(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{})
	sess, p := e.enterDeathChar(deathChar{account: "alice", name: "Alice"})
	cutOff := 0
	for _, id := range talkingIslandMonsters(t, e) {
		for level := 1; level <= 10; level++ {
			n := e.spawnNpc(id, eastOf(tiStart, 100))
			var gotExp, gotSP int64
			var wantExp, wantSP int64
			e.onLoop(func() {
				e.setPlayerLevel(p, level)
				exp, sp := p.Exp(), p.SP()
				diff := level - n.Level()
				if diff < 11 && diff > -11 {
					wantExp, wantSP = int64(n.Template.Acquire.Exp), int64(n.Template.Acquire.SP)
				} else {
					cutOff++
				}
				n.ReduceHP(n.CurHP+1, p)
				gotExp, gotSP = p.Exp()-exp, int64(p.SP()-sp)
			})
			if gotExp != wantExp || gotSP != wantSP {
				t.Errorf("%s (%d, level %d) at level %d: %d exp, %d SP; want %d, %d",
					n.Name(), id, n.Level(), level, gotExp, gotSP, wantExp, wantSP)
			}
			sess.take(t)
		}
	}
	if cutOff == 0 {
		t.Errorf("no kill hit the 11-level cut-off")
	}
}

// The plan's reference numbers: a Bearded Keltir gives "You have earned 35
// experience and 2 SP." at level 1; at level 12 (11 above it) nothing.
func TestNpcKillKeltirMessage(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{})
	sess, p := e.enterDeathChar(deathChar{account: "alice", name: "Alice"})
	k := e.spawnNpc(npcBeardedKeltir, eastOf(tiStart, 100))
	e.onLoop(func() { k.ReduceHP(k.CurHP+1, p) })
	e.sync()
	var msg *testclient.SystemMessage
	for _, m := range only[testclient.SystemMessage](sess.take(t)) {
		if m.ID == int32(sysMsgEarnedExpAndSP) {
			msg = &m
		}
	}
	if msg == nil || len(msg.Params) != 2 || msg.Params[0].Number != 35 || msg.Params[1].Number != 2 {
		t.Errorf("earned message %+v, want 35 and 2", msg)
	}

	k2 := e.spawnNpc(npcBeardedKeltir, eastOf(tiStart, 100))
	var exp int64
	e.onLoop(func() {
		e.setPlayerLevel(p, 12)
		exp = p.Exp()
		k2.ReduceHP(k2.CurHP+1, p)
		exp = p.Exp() - exp
	})
	if exp != 0 {
		t.Errorf("a level 12 player got %d exp for a level 1 Keltir", exp)
	}
}

// Damage shares: two players split the exp by their damage (integer
// division of the template exp), and the drops go to the one that did the
// most damage. An attacker outside the party range gets nothing and does
// not count.
func TestNpcKillDamageShare(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{})
	_, a := e.enterDeathChar(deathChar{account: "alice", name: "Alice"})
	_, b := e.enterDeathChar(deathChar{account: "bob", name: "Bob", pos: eastOf(tiStart, 50)})
	_, c := e.enterDeathChar(deathChar{account: "carol", name: "Carol", pos: eastOf(tiStart, 2000)})
	wolf := e.spawnNpc(npcWolf, eastOf(tiStart, 100))
	r := wolfKillRand()
	e.onLoop(func() {
		e.srv.addDamageHate(wolf, a, 10, 1)
		e.srv.addDamageHate(wolf, b, 30, 1)
		e.srv.addDamageHate(wolf, c, 1000, 1)
	})
	e.withRand(r, func() { e.srv.Kill(wolf, a) })
	var expA, expB, expC int64
	var spA, spB int32
	var protectedFor world.ObjectID
	e.onLoop(func() {
		expA, expB, expC, spA, spB = a.Exp(), b.Exp(), c.Exp(), a.SP(), b.SP()
		for o := range e.srv.world.Visible(wolf) {
			if g, ok := o.(*GroundItem); ok {
				protectedFor = g.protectedFor
			}
		}
	})
	// 141 × 10 / 40 = 35, 141 × 30 / 40 = 105; SP 4 × 10 / 40 = 1, 4 × 30 / 40 = 3.
	if expA != 35 || expB != 105 || expC != 0 || spA != 1 || spB != 3 {
		t.Errorf("exp %d/%d/%d, SP %d/%d; want 35/105/0, 1/3", expA, expB, expC, spA, spB)
	}
	if protectedFor != b.ID() {
		t.Errorf("drops protected for %d, want Bob %d", protectedFor, b.ID())
	}
}

// The corpse goes 7 to 8 s after the death (the 1 s decay scan), and the
// spawn brings the NPC back after its respawn delay (30 ± 10 s for the
// Gremlins of npcTestWorld), counted from the decay.
func TestNpcCorpseDecayAndRespawn(t *testing.T) {
	e := newNpcEnv(t, nil)
	sess, p := statusPlayer(t, e, "alice", "Alice")
	var g *Npc
	e.onLoop(func() {
		for _, sp := range e.srv.npc.spawns {
			if sp.tmpl.ID == npcTestGremlin && sp.npc != nil && sp.npc.InWorld() {
				g = sp.npc
				break
			}
		}
	})
	if g == nil {
		t.Fatal("no Gremlin in the world")
	}
	e.advance(500 * time.Millisecond) // between two decay scans
	died := e.clock.Now()
	e.onLoop(func() { g.ReduceHP(g.CurHP+1, p) })
	state := func() (dead, in bool) {
		e.onLoop(func() { dead, in = g.Dead, g.InWorld() })
		return
	}
	if dead, in := state(); !dead || !in {
		t.Fatalf("after the kill: dead %v, in the world %v", dead, in)
	}
	e.advance(7 * time.Second)
	if _, in := state(); !in {
		t.Fatalf("the corpse went before 7 s")
	}
	sess.take(t)
	e.advance(time.Second)
	if _, in := state(); in {
		t.Fatalf("the corpse is still there after 8 s")
	}
	decayed := e.clock.Now()
	gone := false
	for _, d := range only[testclient.DeleteObject](sess.take(t)) {
		gone = gone || d.ObjectID == int32(g.ID())
	}
	if !gone {
		t.Errorf("no DeleteObject of the corpse")
	}
	if d := decayed.Sub(died); d < 7*time.Second || d > 8*time.Second {
		t.Errorf("the corpse went after %v, want 7-8 s", d)
	}

	var back time.Duration
	for i := 0; i < 45 && back == 0; i++ {
		e.advance(time.Second)
		if dead, in := state(); in {
			if dead {
				t.Fatal("the Gremlin came back dead")
			}
			back = e.clock.Now().Sub(decayed)
		}
	}
	if back < 20*time.Second || back > 41*time.Second {
		t.Errorf("the Gremlin came back %v after the decay, want 20-40 s (+ the tick)", back)
	}
	var hp float64
	var maxHP int32
	e.onLoop(func() { hp, maxHP = g.CurHP, g.MaxHP() })
	if hp != float64(maxHP) {
		t.Errorf("respawned with %v of %d HP", hp, maxHP)
	}
}

// Killing to level 10 with rates.xp 10, each time the best Talking Island
// monster within reach, follows the exp curve: every kill adds the template
// exp × 10, the level is the experience table's for the exp, the corpse
// goes within 8 s, and nothing stays dead, dying or waiting for decay.
func TestNpcKillToLevel10(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{configure: func(gs *config.GameServer) {
		gs.Rates = config.DefaultRates()
		gs.Rates.XP = 10
	}})
	sess, p := e.enterDeathChar(deathChar{account: "alice", name: "Alice"})
	monsters := talkingIslandMonsters(t, e)
	var wantExp int64
	for kills := 0; ; kills++ {
		if kills > 200 {
			t.Fatal("level 10 not reached in 200 kills")
		}
		var level int
		e.onLoop(func() { level = p.Level() })
		if level >= 10 {
			t.Logf("level %d after %d kills", level, kills)
			break
		}
		// The monster with the most exp at most 3 levels above the player.
		var best int
		var bestExp float64
		e.onLoop(func() {
			for _, id := range monsters {
				tm, _ := e.srv.worldData.Npc(id)
				if tm.Level <= level+3 && tm.Acquire.Exp > bestExp {
					best, bestExp = id, tm.Acquire.Exp
				}
			}
		})
		n := e.spawnNpc(best, eastOf(tiStart, 100))
		e.onLoop(func() { n.ReduceHP(n.CurHP+1, p) })
		wantExp += int64(bestExp * 10)
		var exp int64
		var dead, playerDead bool
		e.onLoop(func() {
			exp, level, dead, playerDead = p.Exp(), p.Level(), n.Dead, p.IsDead()
		})
		if exp != wantExp || !dead || playerDead {
			t.Fatalf("kill %d (%d): exp %d (want %d), monster dead %v, player dead %v", kills, best, exp, wantExp, dead, playerDead)
		}
		var wantLevel int
		e.onLoop(func() { wantLevel = e.srv.levelForExp(exp) })
		if level != wantLevel {
			t.Fatalf("kill %d: level %d with %d exp, want %d", kills, level, exp, wantLevel)
		}
		e.advance(8 * time.Second)
		var in bool
		var pending int
		e.onLoop(func() { in, pending = n.InWorld(), len(e.srv.kill.decay) })
		if in || pending != 0 {
			t.Fatalf("kill %d: corpse in the world %v, %d decays pending after 8 s", kills, in, pending)
		}
		sess.take(t)
	}
	var exp int64
	e.onLoop(func() { exp = p.Exp() })
	if exp < e.srv.ExpForLevel(10) || exp >= e.srv.ExpForLevel(11) {
		t.Errorf("exp %d is not within level 10", exp)
	}
}
