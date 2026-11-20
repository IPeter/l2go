package npcai

import (
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
)

// scene is a monster fighting a player 300 away, the player its most hated
// enemy (hate 10).
func scene(npc *fake, dist int32) (*fake, *fakeWorld) {
	target := player("target", dist, 0)
	npc.aggro = []hateEntry{{target, 10}}
	npc.most = target
	return target, &fakeWorld{all: []*fake{npc, target}}
}

// Snipe Cohort (21039): NPC Heal and NPC Flamestrike.
func TestUseSkillsSnipeCohort(t *testing.T) {
	d := repo(t)
	p := d.profile(t, 21039)

	t.Run("self heal below the threshold", func(t *testing.T) {
		npc := monster("cohort", 0, 0, p)
		npc.hp = 400 // 40 %: Get(100) < (100 - 40) / 3 = 20
		target, w := scene(npc, 300)
		th := newThinker(t, w, rnd.Get(100, 19))
		wantCast(t, th.UseSkills(npc, target), 4065, npc, target, true)
	})
	t.Run("self heal at the threshold", func(t *testing.T) {
		npc := monster("cohort", 0, 0, p)
		npc.hp = 400
		target, w := scene(npc, 300)
		// No self heal; no clan; Flamestrike: chance 3 < 10, Get(1) picks it.
		th := newThinker(t, w, rnd.Get(100, 20), rnd.Get(100, 3), rnd.Between(7, 15, 10), rnd.Get(1, 0))
		wantCast(t, th.UseSkills(npc, target), 4078, target, nil, true)
	})
	t.Run("skill chance fails", func(t *testing.T) {
		npc := monster("cohort", 0, 0, p)
		target, w := scene(npc, 300)
		th := newThinker(t, w, rnd.Get(100, 50), rnd.Get(100, 7), rnd.Between(7, 15, 7))
		if d := th.UseSkills(npc, target); d.Action != ActNone {
			t.Fatalf("decision %s, want none", describe(d))
		}
	})
	t.Run("out of cast range", func(t *testing.T) {
		npc := monster("cohort", 0, 0, p)
		target, w := scene(npc, 500) // Flamestrike reaches below 500 (3D)
		th := newThinker(t, w, rnd.Get(100, 50), rnd.Get(100, 0), rnd.Between(7, 15, 15), rnd.Get(1, 0))
		if d := th.UseSkills(npc, target); d.Action != ActNone {
			t.Fatalf("decision %s, want none", describe(d))
		}
	})
	t.Run("a moving fighter does not cast", func(t *testing.T) {
		npc := monster("cohort", 0, 0, p)
		npc.moving = true
		target, w := scene(npc, 300)
		if d := newThinker(t, w).UseSkills(npc, target); d.Action != ActNone {
			t.Fatalf("decision %s, want none", describe(d))
		}
	})
	t.Run("not enough MP", func(t *testing.T) {
		npc := monster("cohort", 0, 0, p)
		npc.hp, npc.mp = 100, 49 // NPC Heal 4 takes 49
		target, w := scene(npc, 300)
		// The self heal draw passes but nothing can be cast; Flamestrike
		// (53 MP) neither.
		th := newThinker(t, w, rnd.Get(100, 0), rnd.Get(100, 0), rnd.Between(7, 15, 15), rnd.Get(1, 0))
		if d := th.UseSkills(npc, target); d.Action != ActNone {
			t.Fatalf("decision %s, want none", describe(d))
		}
	})
}

func TestUseSkillsMovingMage(t *testing.T) {
	d := repo(t)
	p := d.synthetic(t, data.NpcTemplate{ID: 3, AI: data.NpcAI{Type: "MAGE"}, Skills: []data.NpcSkill{{ID: 4078, Level: 4}}})
	npc := monster("mage", 0, 0, p)
	npc.moving = true
	target, w := scene(npc, 300)
	th := newThinker(t, w, rnd.Get(100, 0), rnd.Between(7, 15, 7), rnd.Get(1, 0))
	wantCast(t, th.UseSkills(npc, target), 4078, target, nil, true)
}

// A minion heals its leader when Get(100) > the leader's HP %.
func TestUseSkillsLeaderHeal(t *testing.T) {
	d := repo(t)
	p := d.profile(t, 21039)
	setup := func(leaderX int32) (*fake, *fake, *fake, *fakeWorld) {
		leader := monster("leader", leaderX, 0, p)
		leader.hp = 400
		npc := monster("cohort", 0, 0, p)
		npc.leader = leader
		target, w := scene(npc, 300)
		w.all = append(w.all, leader)
		return npc, leader, target, w
	}
	t.Run("heals above the threshold", func(t *testing.T) {
		npc, leader, target, w := setup(200)
		th := newThinker(t, w, rnd.Get(100, 41))
		wantCast(t, th.UseSkills(npc, target), 4065, leader, target, true)
	})
	t.Run("not without sight", func(t *testing.T) {
		npc, leader, target, w := setup(200)
		w.blind = [][2]*fake{{npc, leader}}
		// On to the self heal (none at full HP) and the long range skill.
		th := newThinker(t, w, rnd.Get(100, 99), rnd.Get(100, 99), rnd.Get(100, 99), rnd.Between(7, 15, 15))
		if d := th.UseSkills(npc, target); d.Action != ActNone {
			t.Fatalf("decision %s, want none", describe(d))
		}
	})
	t.Run("moves to a leader out of reach", func(t *testing.T) {
		npc, leader, target, w := setup(2000)
		th := newThinker(t, w, rnd.Get(100, 41))
		d := th.UseSkills(npc, target)
		// Reach: cast range 600 + both radii (14 + 14).
		if d.Action != ActMoveToPawn || d.Target != Creature(leader) || d.Range != 628 {
			t.Fatalf("decision %s range %d, want moveToPawn to the leader at 628", describe(d), d.Range)
		}
	})
	t.Run("at the threshold", func(t *testing.T) {
		npc, _, target, w := setup(200)
		// 40 is not above 40 %: on to the self heal (0 at full HP) and the
		// long range skill.
		th := newThinker(t, w, rnd.Get(100, 40), rnd.Get(100, 0), rnd.Get(100, 99), rnd.Between(7, 15, 15))
		if d := th.UseSkills(npc, target); d.Action != ActNone {
			t.Fatalf("decision %s, want none", describe(d))
		}
	})
	t.Run("a dead leader draws nothing", func(t *testing.T) {
		npc, leader, target, w := setup(200)
		leader.dead = true
		th := newThinker(t, w, rnd.Get(100, 0), rnd.Get(100, 99), rnd.Between(7, 15, 15))
		if d := th.UseSkills(npc, target); d.Action != ActNone {
			t.Fatalf("decision %s, want none", describe(d))
		}
	})
}

// Soldier of Darkness (21040): NPC Heal goes to a dead clan member (a Mobius
// quirk), and its Decrease Atk. Spd. is shut by its Skill03 gate.
func TestUseSkillsSoldierOfDarkness(t *testing.T) {
	d := repo(t)
	p := d.profile(t, 21040)
	setup := func() (*fake, *fake, *fake, *fakeWorld) {
		npc := monster("soldier", 0, 0, p)
		target, w := scene(npc, 300)
		alive := monster("alive", 50, 0, p)
		alive.hp = 100
		corpse := monster("corpse", 100, 0, p)
		corpse.dead, corpse.hp = true, 0
		outsider := monster("outsider", 120, 0, d.profile(t, 21039)) // no clan
		outsider.dead, outsider.hp = true, 0
		w.all = append(w.all, alive, outsider, corpse)
		return npc, target, corpse, w
	}
	t.Run("heals the dead", func(t *testing.T) {
		npc, target, corpse, w := setup()
		// Full HP: no self heal. The living member and the outsider draw
		// nothing; the corpse draws (100 − 0) / 10 = 10.
		th := newThinker(t, w, rnd.Get(100, 0), rnd.Get(100, 9))
		wantCast(t, th.UseSkills(npc, target), 4065, corpse, target, true)
	})
	t.Run("the gate shuts the debuff", func(t *testing.T) {
		npc, target, _, w := setup()
		th := newThinker(t, w, rnd.Get(100, 0), rnd.Get(100, 10),
			rnd.Get(100, 0), rnd.Between(7, 15, 7), rnd.Get(1, 0))
		if d := th.UseSkills(npc, target); d.Action != ActNone {
			t.Fatalf("decision %s, want none", describe(d))
		}
	})
}

// Catherok (21035): Shock, 10 % within 150.
func TestUseSkillsCatherok(t *testing.T) {
	d := repo(t)
	p := d.profile(t, 21035)
	t.Run("stuns", func(t *testing.T) {
		npc := monster("catherok", 0, 0, p)
		target, w := scene(npc, 150)
		th := newThinker(t, w, rnd.Get(100, 6), rnd.Between(7, 15, 7), rnd.Get(1, 0), rnd.Get(GateScale, 999))
		wantCast(t, th.UseSkills(npc, target), 4072, target, nil, true)
	})
	t.Run("the gate fails", func(t *testing.T) {
		npc := monster("catherok", 0, 0, p)
		target, w := scene(npc, 100)
		th := newThinker(t, w, rnd.Get(100, 6), rnd.Between(7, 15, 7), rnd.Get(1, 0), rnd.Get(GateScale, 1000))
		if d := th.UseSkills(npc, target); d.Action != ActNone {
			t.Fatalf("decision %s, want none", describe(d))
		}
	})
	t.Run("the chance fails", func(t *testing.T) {
		npc := monster("catherok", 0, 0, p)
		target, w := scene(npc, 100)
		th := newThinker(t, w, rnd.Get(100, 7), rnd.Between(7, 15, 7))
		if d := th.UseSkills(npc, target); d.Action != ActNone {
			t.Fatalf("decision %s, want none", describe(d))
		}
	})
	t.Run("beyond 150 after the chance draws", func(t *testing.T) {
		npc := monster("catherok", 0, 0, p)
		target, w := scene(npc, 151)
		th := newThinker(t, w, rnd.Get(100, 0), rnd.Between(7, 15, 15))
		if d := th.UseSkills(npc, target); d.Action != ActNone {
			t.Fatalf("decision %s, want none", describe(d))
		}
	})
}

func TestUseSkillsResurrectsLeader(t *testing.T) {
	d := repo(t)
	p := d.synthetic(t, data.NpcTemplate{ID: 4, Skills: []data.NpcSkill{{ID: 1016, Level: 1}}})
	leader := monster("leader", 200, 0, p)
	leader.dead = true
	npc := monster("priest", 0, 0, p)
	npc.leader = leader
	target, w := scene(npc, 300)
	w.all = append(w.all, leader)
	// No draw for a dead leader; no heal list. Then the long range check:
	// none (Resurrection is in RES only).
	th := newThinker(t, w)
	wantCast(t, th.UseSkills(npc, target), 1016, leader, target, true)

	leader.loc.X = 5000
	d2 := newThinker(t, w).UseSkills(npc, target)
	if d2.Action != ActMoveToPawn || d2.Target != Creature(leader) {
		t.Fatalf("decision %s, want moveToPawn to the leader", describe(d2))
	}
}

func TestSuicide(t *testing.T) {
	d := repo(t)
	p := d.synthetic(t, data.NpcTemplate{ID: 5, Skills: []data.NpcSkill{{ID: 4139, Level: 6}}})
	t.Run("blows up", func(t *testing.T) {
		npc := monster("bomb", 0, 0, p)
		npc.hp = 299.9 // 29 % as an int
		target, w := scene(npc, 100)
		th := newThinker(t, w, rnd.Get(1, 0), rnd.Get(100, 0), rnd.Between(7, 15, 7))
		wantCast(t, th.Suicide(npc, target), 4139, target, nil, true)
	})
	t.Run("not below 30 %", func(t *testing.T) {
		npc := monster("bomb", 0, 0, p)
		npc.hp = 300
		target, w := scene(npc, 100)
		if d := newThinker(t, w).Suicide(npc, target); d.Action != ActNone {
			t.Fatalf("decision %s, want none", describe(d))
		}
	})
	t.Run("out of its affect range", func(t *testing.T) {
		npc := monster("bomb", 0, 0, p)
		npc.hp = 100
		target, w := scene(npc, 174) // 150 + 14 + 9 = 173
		if d := newThinker(t, w, rnd.Get(1, 0)).Suicide(npc, target); d.Action != ActNone {
			t.Fatalf("decision %s, want none", describe(d))
		}
	})
	t.Run("the chance fails", func(t *testing.T) {
		npc := monster("bomb", 0, 0, p)
		npc.hp = 100
		target, w := scene(npc, 173)
		th := newThinker(t, w, rnd.Get(1, 0), rnd.Get(100, 7), rnd.Between(7, 15, 7))
		if d := th.Suicide(npc, target); d.Action != ActNone {
			t.Fatalf("decision %s, want none", describe(d))
		}
	})
}

func TestChaosAfter30Thinks(t *testing.T) {
	boss := monster("boss", 0, 0, &Profile{})
	boss.rank, boss.hp = RankRaidBoss, 500
	p1 := player("p1", 100, 0)
	boss.aggro, boss.most = []hateEntry{{p1, 10}}, p1
	w := &fakeWorld{all: []*fake{boss, p1}}
	th := newThinker(t, w)
	var st State
	for i := range 30 {
		if d, stop := th.Chaos(&st, boss); d.Action != ActNone || stop {
			t.Fatalf("think %d: %s, stop %v", i+1, describe(d), stop)
		}
	}
	src := th.Rnd.(*rnd.Scripted)
	src.Add(rnd.Get(100, 51)) // 51 > 100 − 50
	if d, stop := th.Chaos(&st, boss); d.Action != ActNone || stop || st.ChaosTime != 31 {
		t.Fatalf("think 31: %s, stop %v, count %d", describe(d), stop, st.ChaosTime)
	}
	src.Add(rnd.Get(100, 50), rnd.Get(1, 0))
	d, stop := th.Chaos(&st, boss)
	if d.Action != ActRetarget || d.Target != Creature(p1) || !stop || st.ChaosTime != 0 {
		t.Fatalf("think 32: %s, stop %v, count %d", describe(d), stop, st.ChaosTime)
	}
}

func TestChaosRates(t *testing.T) {
	for _, tt := range []struct {
		name    string
		rank    Rank
		hp      float64
		minions bool
		roll    int
		change  bool
	}{
		{"raid at its rate", RankRaidBoss, 500, false, 50, true},
		{"raid above its rate", RankRaidBoss, 500, false, 51, false},
		{"raid with minions", RankRaidBoss, 500, true, 0, true},
		{"raid with minions above", RankRaidBoss, 500, true, 1, false},
		{"grand boss, low rate", RankGrandBoss, 900, false, 10, true},
		{"grand boss, low rate above 10", RankGrandBoss, 900, false, 11, false},
		{"grand boss at its rate", RankGrandBoss, 200, false, 40, true},
		{"grand boss above its rate", RankGrandBoss, 200, false, 41, false},
		{"raid minion", RankRaidMinion, 250, false, 50, true},
		{"raid minion above", RankRaidMinion, 250, false, 51, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			npc := monster("npc", 0, 0, &Profile{})
			npc.rank, npc.hp, npc.minions = tt.rank, tt.hp, tt.minions
			p1 := player("p1", 100, 0)
			npc.aggro, npc.most = []hateEntry{{p1, 10}}, p1
			draws := []rnd.Draw{rnd.Get(100, tt.roll)}
			if tt.change {
				draws = append(draws, rnd.Get(1, 0))
			}
			th := newThinker(t, &fakeWorld{all: []*fake{npc, p1}}, draws...)
			st := State{ChaosTime: 30}
			d, stop := th.Chaos(&st, npc)
			if got := d.Action == ActRetarget; got != tt.change || stop != tt.change {
				t.Fatalf("%s, stop %v; want change %v", describe(d), stop, tt.change)
			}
		})
	}
	t.Run("a monster never", func(t *testing.T) {
		npc := monster("npc", 0, 0, &Profile{})
		st := State{ChaosTime: 100}
		if d, stop := newThinker(t, nil).Chaos(&st, npc); d.Action != ActNone || stop || st.ChaosTime != 100 {
			t.Fatalf("%s, stop %v, count %d", describe(d), stop, st.ChaosTime)
		}
	})
	t.Run("a raid past the minion count", func(t *testing.T) {
		npc := monster("npc", 0, 0, &Profile{})
		npc.rank, npc.hp = RankRaidBoss, 500
		th := newThinker(t, nil, rnd.Get(100, 1)) // the raid rate allows 1, the minion rate (0) does not
		th.Config.MinionChaosTime = 10
		st := State{ChaosTime: 10}
		if d, stop := th.Chaos(&st, npc); d.Action != ActNone || stop {
			t.Fatalf("%s, stop %v", describe(d), stop)
		}
	})
	t.Run("nobody to change to", func(t *testing.T) {
		npc := monster("npc", 0, 0, &Profile{})
		npc.rank, npc.hp = RankRaidBoss, 0
		p1 := player("p1", 100, 0)
		p1.dead = true
		npc.aggro = []hateEntry{{p1, 10}}
		th := newThinker(t, &fakeWorld{all: []*fake{npc, p1}}, rnd.Get(100, 99))
		st := State{ChaosTime: 30}
		if d, stop := th.Chaos(&st, npc); d.Action != ActNone || !stop || st.ChaosTime != 31 {
			t.Fatalf("%s, stop %v, count %d", describe(d), stop, st.ChaosTime)
		}
	})
}

func TestTargetReconsider(t *testing.T) {
	setup := func() (*fake, *fake, *fake, *fake, *fakeWorld) {
		npc := monster("npc", 0, 0, &Profile{})
		npc.aggressive, npc.aggroRange = true, 300
		p1, p2, p3 := player("p1", 500, 0), player("p2", 600, 0), player("p3", 200, 0)
		npc.aggro = []hateEntry{{p1, 5}, {p2, 50}}
		return npc, p1, p2, p3, &fakeWorld{all: []*fake{npc, p1, p2, p3}}
	}
	t.Run("random", func(t *testing.T) {
		npc, _, _, p3, w := setup()
		if got := newThinker(t, w, rnd.Get(3, 2)).TargetReconsider(npc, true); got != Creature(p3) {
			t.Fatalf("got %v, want p3", got)
		}
	})
	t.Run("the most hated is never returned", func(t *testing.T) {
		npc, _, _, _, w := setup()
		if got := newThinker(t, w).TargetReconsider(npc, false); got != nil {
			t.Fatalf("got %v, want nil (Mobius quirk)", got)
		}
	})
	t.Run("aggro range without attackers", func(t *testing.T) {
		npc, p1, p2, p3, w := setup()
		p1.dead, p2.dead = true, true
		if got := newThinker(t, w).TargetReconsider(npc, false); got != Creature(p3) {
			t.Fatalf("got %v, want p3", got)
		}
	})
	t.Run("a rooted NPC needs its target in reach and sight", func(t *testing.T) {
		npc, _, _, p3, w := setup()
		npc.rooted = true
		npc.aggro = nil
		p3.loc.X = 63 // reach 40 + 14 + 9, excluded
		if got := newThinker(t, w).TargetReconsider(npc, false); got != nil {
			t.Fatalf("got %v, want nil", got)
		}
		p3.loc.X = 62
		if got := newThinker(t, w).TargetReconsider(npc, false); got != Creature(p3) {
			t.Fatalf("got %v, want p3", got)
		}
		w.blind = [][2]*fake{{npc, p3}}
		if got := newThinker(t, w).TargetReconsider(npc, false); got != nil {
			t.Fatalf("got %v, want nil without sight", got)
		}
	})
}

// Catherok rooted: one Get(100) picks the skill kinds.
func TestMovementDisabled(t *testing.T) {
	d := repo(t)
	p := d.profile(t, 21035)
	setup := func(dist int32) (*fake, *fake, *fakeWorld) {
		npc := monster("catherok", 0, 0, p)
		npc.rooted = true
		target, w := scene(npc, dist)
		return npc, target, w
	}
	t.Run("debuff below 30", func(t *testing.T) {
		npc, target, w := setup(100)
		th := newThinker(t, w, rnd.Get(100, 29), rnd.Get(GateScale, 500))
		wantCast(t, th.MovementDisabled(npc, target), 4072, target, nil, true)
	})
	t.Run("COT below 20 when the target is immobilized", func(t *testing.T) {
		npc, target, w := setup(100)
		target.immobilized = true
		th := newThinker(t, w, rnd.Get(100, 10), rnd.Get(GateScale, 0))
		wantCast(t, th.MovementDisabled(npc, target), 4072, target, nil, true)
	})
	t.Run("the gate fails in each list", func(t *testing.T) {
		npc, target, w := setup(100)
		// Below 15: IMMOBILIZE is empty; Shock in COT and in DEBUFF draws
		// its gate each time; then the target is out of reach.
		th := newThinker(t, w, rnd.Get(100, 14), rnd.Get(GateScale, 1000), rnd.Get(GateScale, 1000))
		if d := th.MovementDisabled(npc, target); d.Action != ActNone {
			t.Fatalf("decision %s, want none", describe(d))
		}
	})
	t.Run("attacks within reach", func(t *testing.T) {
		npc, target, w := setup(63) // 40 + 14 + 9
		th := newThinker(t, w, rnd.Get(100, 30))
		if d := th.MovementDisabled(npc, target); d.Action != ActAttack || d.Target != Creature(target) {
			t.Fatalf("decision %s, want an attack", describe(d))
		}
	})
	t.Run("reconsiders a near target", func(t *testing.T) {
		npc, target, w := setup(70) // 70 − 14 = 56 <= 63
		th := newThinker(t, w, rnd.Get(100, 30))
		d := th.MovementDisabled(npc, target)
		if d.Action != ActRetarget || d.Target != Creature(target) || !d.AddHate || d.Hate != 10 {
			t.Fatalf("decision %s (hate %v %d), want retarget adding 10", describe(d), d.AddHate, d.Hate)
		}
	})
	t.Run("nothing to do", func(t *testing.T) {
		npc, target, w := setup(100)
		th := newThinker(t, w, rnd.Get(100, 30))
		if d := th.MovementDisabled(npc, target); d.Action != ActNone {
			t.Fatalf("decision %s, want none", describe(d))
		}
	})
}
