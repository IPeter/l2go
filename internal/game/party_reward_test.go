package game

import (
	"math"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/social"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// partyBonusOf is Party.BONUS_EXP_SP for n valid members.
func partyBonusOf(n int) float64 {
	return [...]float64{1.0, 1.10, 1.20, 1.30, 1.40, 1.50, 2.0, 2.10, 2.20}[min(n, 9)-1]
}

// partyKill makes the leader deal all of a Wolf's damage and kill it, and
// returns each player's exp and SP gain.
func (f *partyFixture) partyKill(r rnd.Source) (exp []int64, sp []int32) {
	f.t.Helper()
	wolf := f.e.spawnNpc(npcWolf, eastOf(tiStart, 100))
	before := make([]int64, len(f.ps))
	beforeSP := make([]int32, len(f.ps))
	f.e.onLoop(func() {
		for i, p := range f.ps {
			before[i], beforeSP[i] = p.Exp(), p.SP()
		}
		f.e.srv.addDamageHate(wolf, f.ps[0], int64(wolf.CurHP), 1)
	})
	if r == nil {
		r = rnd.New(7)
	}
	f.e.withRand(r, func() { f.e.srv.Kill(wolf, f.ps[0]) })
	f.e.onLoop(func() {
		for i, p := range f.ps {
			exp = append(exp, p.Exp()-before[i])
			sp = append(sp, p.SP()-beforeSP[i])
		}
	})
	return exp, sp
}

// A party of 2 to 9 level 1 members kills a Wolf (141 exp, 4 SP): each gets
// round(141 × bonus × 1/n) exp and round(4 × bonus × 1/n) SP, Mobius's
// distributeXpAndSp with the bonus of n valid members.
func TestPartyExpShares(t *testing.T) {
	for n := 2; n <= 9; n++ {
		f := newPartyFixture(t, n, nil)
		members := make([]int, n)
		for i := range members {
			members[i] = i
		}
		f.form(0, members...)
		exp, sp := f.partyKill(nil)
		b := partyBonusOf(n)
		pre := 1 / float64(n)
		wantExp := int64(math.Floor(141*b*pre + 0.5))
		wantSP := int32(math.Floor(4*b*pre + 0.5))
		for i := range n {
			if exp[i] != wantExp || sp[i] != wantSP {
				t.Errorf("%d members: %s got %d exp, %d SP; want %d, %d", n, partyNames[i], exp[i], sp[i], wantExp, wantSP)
			}
		}
	}
}

// Shares follow level², and the cutoff (5 levels here) leaves a low member
// out: it gets nothing and does not count for the bonus. A member out of
// the party range is not rewarded and does not count either.
func TestPartyExpCutoff(t *testing.T) {
	f := newPartyFixture(t, 4, func(gs *config.GameServer) { gs.Party.XPCutoffLevel = 5 })
	f.form(0, 0, 1, 2, 3)
	f.e.onLoop(func() {
		f.e.setPlayerLevel(f.ps[0], 10)
		f.e.setPlayerLevel(f.ps[1], 4)
		f.e.setPlayerLevel(f.ps[2], 7)
		f.e.setPlayerLevel(f.ps[3], 10)
		if err := f.e.srv.MoveObject(f.ps[3], eastOf(tiStart, 3000)); err != nil {
			t.Errorf("MoveObject: %v", err)
		}
	})
	exp, sp := f.partyKill(nil)
	// Valid: Alice (10) and Carol (7), bonus 1.1; Bob (4) is 6 below.
	b := partyBonusOf(2)
	share := func(level int) (int64, int32) {
		pre := float64(level*level) / float64(100+49)
		return int64(math.Floor(141*b*pre + 0.5)), int32(math.Floor(4*b*pre + 0.5))
	}
	ea, sa := share(10)
	ec, sc := share(7)
	if want := []int64{ea, 0, ec, 0}; !slices.Equal(exp, want) {
		t.Errorf("exp %v, want %v", exp, want)
	}
	if want := []int32{sa, 0, sc, 0}; !slices.Equal(sp, want) {
		t.Errorf("SP %v, want %v", sp, want)
	}
}

// An attacker outside the party takes its share: the party's exp is cut by
// its part of the damage, twice as Mobius does (calculateExpAndSp and
// partyMul).
func TestPartyExpDamageShare(t *testing.T) {
	f := newPartyFixture(t, 3, nil)
	f.form(0, 0, 1)
	wolf := f.e.spawnNpc(npcWolf, eastOf(tiStart, 100))
	var before [3]int64
	f.e.onLoop(func() {
		for i, p := range f.ps {
			before[i] = p.Exp()
		}
		f.e.srv.addDamageHate(wolf, f.ps[0], 30, 1)
		f.e.srv.addDamageHate(wolf, f.ps[2], 10, 1)
	})
	f.e.withRand(rnd.New(3), func() { f.e.srv.Kill(wolf, f.ps[0]) })
	var got [3]int64
	f.e.onLoop(func() {
		for i, p := range f.ps {
			got[i] = p.Exp() - before[i]
		}
	})
	// The party: 141 × 30 / 40 = 105, × 0.75 = 78.75, × 1.1 / 2 = 43.3 → 43.
	party := int64(math.Floor(float64(105)*0.75*1.1*0.5 + 0.5))
	if want := [3]int64{party, party, 141 * 10 / 40}; got != want {
		t.Errorf("exp %v, want %v", got, want)
	}
}

// lootRand answers Get from a script (0 once it ends) and records the
// draws.
func lootRand(gets ...int) *dropRand { return &dropRand{Source: rnd.New(1), gets: gets} }

// The five loot modes for a kill's item, with spoil, the by-turn rotation,
// a member out of range, and the messages to the others.
func TestPartyLootModes(t *testing.T) {
	f := newPartyFixture(t, 3, nil)
	f.form(0, 0, 1, 2)
	var pt *Party
	f.e.onLoop(func() { pt = f.ps[0].Party() })
	// give hands Alice an item by mode and returns who got it and the
	// draws.
	give := func(mode social.LootMode, spoil bool, count int64, r *dropRand) (string, []string) {
		t.Helper()
		var before [3]int64
		f.e.onLoop(func() {
			pt.loot.Mode = mode
			for i, p := range f.ps {
				before[i] = p.inv.Count(itemVarnish, -1, true)
			}
		})
		f.e.withRand(r, func() { f.e.srv.partyDistributeItem(f.ps[0], itemVarnish, count, spoil, f.ps[0]) })
		who := ""
		f.e.onLoop(func() {
			for i, p := range f.ps {
				if p.inv.Count(itemVarnish, -1, true) != before[i] {
					who += partyNames[i]
				}
			}
		})
		return who, r.drawn()
	}
	for _, tc := range []struct {
		name  string
		mode  social.LootMode
		spoil bool
		gets  []int
		want  string
		draws []string
	}{
		{"finders keepers", social.FindersKeepers, false, nil, "Alice", nil},
		{"random", social.Random, false, []int{2}, "Carol", []string{"Get(3)"}},
		{"random, spoil", social.Random, true, nil, "Alice", nil},
		{"random including spoil", social.RandomIncludingSpoil, true, []int{1}, "Bob", []string{"Get(3)"}},
		{"by turn 1", social.ByTurn, false, nil, "Bob", nil},
		{"by turn 2", social.ByTurn, false, nil, "Carol", nil},
		{"by turn, spoil", social.ByTurn, true, nil, "Alice", nil},
		{"by turn 3", social.ByTurn, false, nil, "Alice", nil},
		{"by turn including spoil", social.ByTurnIncludingSpoil, true, nil, "Bob", nil},
	} {
		who, draws := give(tc.mode, tc.spoil, 1, lootRand(tc.gets...))
		if who != tc.want || !slices.Equal(draws, tc.draws) {
			t.Errorf("%s: %q got it with draws %q; want %q, %q", tc.name, who, draws, tc.want, tc.draws)
		}
	}
	for i := range 3 {
		f.take(i)
	}
	// The others learn of it: "$s1 has obtained $s3 $s2." for a stack,
	// with Sweeper for spoil.
	give(social.FindersKeepers, false, 2, lootRand())
	if got := messageTexts(f.take(1), partyMsgObtainedCount); len(got) != 1 || got[0][0] != "Alice" {
		t.Errorf("Bob learnt %q", got)
	}
	if got := messageTexts(f.take(0), partyMsgObtainedCount); len(got) != 0 {
		t.Errorf("Alice learnt %q about herself", got)
	}
	give(social.FindersKeepers, true, 1, lootRand())
	if got := messageTexts(f.take(2), partyMsgObtainedSweeper); len(got) != 1 {
		t.Errorf("Carol learnt %q", got)
	}

	// Carol out of range: only Alice and Bob draw.
	f.e.onLoop(func() {
		if err := f.e.srv.MoveObject(f.ps[2], eastOf(tiStart, 3000)); err != nil {
			t.Errorf("MoveObject: %v", err)
		}
	})
	if who, draws := give(social.Random, false, 1, lootRand(1)); who != "Bob" || !slices.Equal(draws, []string{"Get(2)"}) {
		t.Errorf("Carol away: %q with %q", who, draws)
	}
}

// Adena is split among the members in range, the remainder 1 each from a
// drawn member on.
func TestPartyAdenaSplit(t *testing.T) {
	f := newPartyFixture(t, 3, nil)
	f.form(int32(social.Random), 0, 1, 2)
	split := func(adena int64, r *dropRand) ([3]int64, []string) {
		t.Helper()
		var before, got [3]int64
		f.e.onLoop(func() {
			for i, p := range f.ps {
				before[i] = p.inv.Adena()
			}
		})
		f.e.withRand(r, func() { f.e.srv.partyDistributeItem(f.ps[0], adenaID, adena, false, f.ps[0]) })
		f.e.onLoop(func() {
			for i, p := range f.ps {
				got[i] = p.inv.Adena() - before[i]
			}
		})
		return got, r.drawn()
	}
	if got, draws := split(9, lootRand()); got != [3]int64{3, 3, 3} || draws != nil {
		t.Errorf("9 adena: %v, draws %q", got, draws)
	}
	if got, draws := split(10, lootRand(2)); got != [3]int64{3, 3, 4} || !slices.Equal(draws, []string{"Get(3)"}) {
		t.Errorf("10 adena: %v, draws %q", got, draws)
	}
	if got, _ := split(11, lootRand(2)); got != [3]int64{4, 3, 4} {
		t.Errorf("11 adena: %v", got)
	}
	f.e.onLoop(func() {
		if err := f.e.srv.MoveObject(f.ps[1], eastOf(tiStart, 3000)); err != nil {
			t.Errorf("MoveObject: %v", err)
		}
	})
	if got, draws := split(11, lootRand(0)); got != [3]int64{6, 0, 5} || !slices.Equal(draws, []string{"Get(2)"}) {
		t.Errorf("11 adena, Bob away: %v, draws %q", got, draws)
	}
}

// With auto-loot, a party's kill is shared: the Wolf's 25 adena split three
// ways and Varnish to a random member, the others told.
func TestPartyKillAutoLoot(t *testing.T) {
	f := newPartyFixture(t, 3, func(gs *config.GameServer) {
		gs.Combat = config.DefaultCombat()
		gs.Combat.AutoLoot = true
	})
	f.form(int32(social.Random), 0, 1, 2)
	r := wolfKillRand()
	r.gets = []int{2, 1} // the first loot draw, then the second
	var before [3]int64
	f.e.onLoop(func() {
		for i, p := range f.ps {
			before[i] = p.inv.Adena()
		}
	})
	exp, _ := f.partyKill(r)
	var adena [3]int64
	var varnish [3]int64
	f.e.onLoop(func() {
		for i, p := range f.ps {
			adena[i] = p.inv.Adena() - before[i]
			varnish[i] = p.inv.Count(itemVarnish, -1, true)
		}
	})
	if adena[0]+adena[1]+adena[2] != 25 || slices.Max(adena[:])-slices.Min(adena[:]) != 1 {
		t.Errorf("adena %v, want 25 split 8/8/9", adena)
	}
	if varnish[0]+varnish[1]+varnish[2] != 1 {
		t.Errorf("Varnish %v, want one", varnish)
	}
	want := int64(math.Floor(141*1.2/3 + 0.5))
	if !slices.Equal(exp, []int64{want, want, want}) {
		t.Errorf("exp %v, want %d each", exp, want)
	}
	told := 0
	for i := range 3 {
		told += len(messageTexts(f.take(i), partyMsgObtained))
	}
	if told != 2 {
		t.Errorf("%d members told of the Varnish, want 2", told)
	}
}

// A party member picks up an item protected for another member; the party
// shares what it picks up.
func TestPartyPickUp(t *testing.T) {
	f := newPartyFixture(t, 3, nil)
	f.form(int32(social.Random), 0, 1)
	drop := func(itemID int, count int64, owner *Player) *GroundItem {
		t.Helper()
		var g *GroundItem
		f.e.onLoop(func() {
			tmpl, _ := f.e.srv.data.ItemTemplate(itemID)
			id, _ := f.e.srv.items.pool.TryNext()
			var err error
			g, err = f.e.srv.DropOnGround(inventory.NewItem(id, tmpl, count), f.ps[0].Position(), GroundDrop{ProtectFor: owner})
			if err != nil {
				t.Errorf("DropOnGround: %v", err)
			}
		})
		return g
	}
	g := drop(itemVarnish, 1, f.ps[1])
	r := lootRand(1)
	f.e.withRand(r, func() { f.e.srv.doPickUp(f.ps[0], g) })
	var bob int64
	f.e.onLoop(func() { bob = f.ps[1].inv.Count(itemVarnish, -1, true) })
	if bob != 1 || !slices.Equal(r.drawn(), []string{"Get(2)"}) {
		t.Errorf("Bob has %d Varnish, draws %q", bob, r.drawn())
	}
	if got := messageTexts(f.take(0), partyMsgObtained); len(got) != 1 || got[0][0] != "Bob" {
		t.Errorf("Alice learnt %q", got)
	}

	// Carol, outside the party, may not take Bob's drop.
	g = drop(itemVarnish, 1, f.ps[1])
	f.e.onLoop(func() { f.e.srv.doPickUp(f.ps[2], g) })
	var on bool
	f.e.onLoop(func() { on = g.InWorld() })
	if !on {
		t.Errorf("Carol took Bob's protected drop")
	}

	g = drop(adenaID, 7, nil)
	var before [2]int64
	f.e.onLoop(func() { before = [2]int64{f.ps[0].inv.Adena(), f.ps[1].inv.Adena()} })
	f.e.withRand(lootRand(0), func() { f.e.srv.doPickUp(f.ps[0], g) })
	var got [2]int64
	f.e.onLoop(func() { got = [2]int64{f.ps[0].inv.Adena() - before[0], f.ps[1].inv.Adena() - before[1]} })
	if got != [2]int64{4, 3} {
		t.Errorf("7 adena picked up: %v, want 4/3", got)
	}
}

// The party target types reach the party; CallParty brings the members
// that may come, and CallPc asks with ConfirmDlg and teleports on a yes.
func TestPartySkillsAndSummons(t *testing.T) {
	f := newPartyFixture(t, 4, nil)
	f.form(0, 0, 1, 2)
	w := skillWorld{f.e.srv}
	f.e.onLoop(func() {
		if got := w.Party(f.ps[1].skill); len(got) != 3 || got[0] != f.ps[0].skill {
			t.Errorf("Party = %v", got)
		}
		if !w.SameParty(f.ps[0].skill, f.ps[2].skill) || w.SameParty(f.ps[0].skill, f.ps[3].skill) {
			t.Errorf("SameParty is wrong")
		}
		far := world.Position{X: tiStart.X + 2000, Y: tiStart.Y, Z: tiStart.Z}
		for _, p := range f.ps[1:] {
			if err := f.e.srv.MoveObject(p, far); err != nil {
				t.Errorf("MoveObject: %v", err)
			}
		}
		f.ps[2].dead = true
	})
	f.settle()
	f.e.onLoop(func() { w.CallParty(f.ps[0].skill) })
	if tp := only[testclient.TeleportToLocation](f.take(1)); len(tp) != 1 || math.Abs(float64(tp[0].X-tiStart.X)) > maxOffsetOnTeleport {
		t.Errorf("Bob's teleport %+v", tp)
	}
	if got := messageTexts(f.take(0), summonMsgDead); len(got) != 1 || got[0][0] != "Carol" {
		t.Errorf("Alice learnt %q", got)
	}
	for _, tp := range only[testclient.TeleportToLocation](f.take(3)) {
		if tp.ObjectID == f.ps[3].ObjectID() {
			t.Errorf("Dave, outside the party, was called")
		}
	}

	f.e.onLoop(func() { w.CallPc(f.ps[0].skill, f.ps[3].skill, 0, 0) })
	var dlg []testclient.Unknown
	for _, u := range only[testclient.Unknown](f.take(3)) {
		if u.Opcode == 0xed {
			dlg = append(dlg, u)
		}
	}
	if len(dlg) != 1 {
		t.Fatalf("Dave got %d ConfirmDlg", len(dlg))
	}
	answer := func(a, requester int32) {
		body := packets.NewWriter(12)
		body.WriteInt32(int32(summonMsgWishes))
		body.WriteInt32(a)
		body.WriteInt32(requester)
		f.send(3, clientpackets.OpDlgAnswer, body.Bytes())
	}
	answer(1, 12345) // not the summoner
	if n := len(only[testclient.TeleportToLocation](f.take(3))); n != 0 {
		t.Errorf("a wrong requester teleported Dave")
	}
	f.e.onLoop(func() { w.CallPc(f.ps[0].skill, f.ps[3].skill, 0, 0) })
	f.take(3)
	answer(1, f.ps[0].ObjectID())
	if n := len(only[testclient.TeleportToLocation](f.take(3))); n != 1 {
		t.Errorf("Dave's yes did not teleport him")
	}
	// With an item cost Dave cannot pay: only the message.
	f.e.onLoop(func() { w.CallPc(f.ps[0].skill, f.ps[3].skill, itemVarnish, 1) })
	if got := partyPackets(f.take(3)); !slices.Equal(got, []string{partySM(summonMsgItemRequired)}) {
		t.Errorf("Dave without the item got %q", got)
	}
	f.e.onLoop(func() {
		if f.ps[3].party.summon != nil {
			t.Errorf("a summon request without the item")
		}
	})
}
