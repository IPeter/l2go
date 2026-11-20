package npcai

import (
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
)

func TestRingPoint(t *testing.T) {
	center := Point{X: 1000, Y: 2000, Z: -300}
	for _, tt := range []struct {
		name     string
		min, rad int
		draws    []rnd.Draw
		want     Point
	}{
		// x 300 > 230: past the center (+100); y sqrt(350² − 300²) = 180:
		// before it (−150).
		{"follow", 30, 200, []rnd.Draw{rnd.Between(60, 400, 300), rnd.Between(300, 400, 350)}, Point{X: 1100, Y: 1850, Z: -300}},
		// x 60, distance 60: y 0; both before the center.
		{"innermost", 30, 200, []rnd.Draw{rnd.Between(60, 400, 60), rnd.Between(60, 400, 60)}, Point{X: 970, Y: 2030, Z: -300}},
		// x 400 is the top: Between(400, 400) does not draw.
		{"outermost", 30, 200, []rnd.Draw{rnd.Between(60, 400, 400)}, Point{X: 1200, Y: 2030, Z: -300}},
		{"raid", 30, 500, []rnd.Draw{rnd.Between(60, 1000, 600), rnd.Between(600, 1000, 1000)}, Point{X: 1100, Y: 2300, Z: -300}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := RingPoint(rnd.NewScripted(t, tt.draws...), center, tt.min, tt.rad); got != tt.want {
				t.Errorf("point %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSpawnPoint(t *testing.T) {
	// Collision 21.5 → 21 + 30 = 51 inside, 200 outside.
	r := rnd.NewScripted(t, rnd.Between(102, 400, 102), rnd.Between(102, 400, 102))
	if got := SpawnPoint(r, Point{X: 0, Y: 0, Z: 7}, 21.5); got != (Point{X: -51, Y: 51, Z: 7}) {
		t.Errorf("point %+v", got)
	}
}

func TestFollow(t *testing.T) {
	d := repo(t)
	guardian := d.profile(t, 21059) // Beast Guardian: NPC Chant of Life
	setup := func(dist int32, rank Rank) (*fake, *fake) {
		leader := monster("leader", 0, 0, guardian)
		npc := monster("minion", dist, 0, guardian)
		npc.leader, npc.rank = leader, rank
		return npc, leader
	}
	t.Run("follows past 200", func(t *testing.T) {
		npc, leader := setup(201, RankNormal)
		th := newThinker(t, nil, rnd.Between(60, 400, 300), rnd.Between(300, 400, 350))
		dd := th.Follow(npc, leader)
		if dd.Action != ActMoveTo || dd.To != (Point{X: 100, Y: -150}) {
			t.Fatalf("decision %s to %+v", describe(dd), dd.To)
		}
	})
	t.Run("a raid minion lets its leader go 500", func(t *testing.T) {
		npc, leader := setup(500, RankRaidMinion)
		if dd := newThinker(t, nil, rnd.Get(RandomWalkRate, 1)).Follow(npc, leader); dd.Action != ActNone {
			t.Fatalf("decision %s, want none", describe(dd))
		}
	})
	t.Run("buffs when hurt", func(t *testing.T) {
		npc, leader := setup(200, RankNormal)
		npc.hp = 900
		dd := newThinker(t, nil, rnd.Get(RandomWalkRate, 0)).Follow(npc, leader)
		wantCast(t, dd, 4097, npc, nil, false)
	})
	t.Run("Chant of Life waits for a wound", func(t *testing.T) {
		npc, leader := setup(200, RankNormal)
		if dd := newThinker(t, nil, rnd.Get(RandomWalkRate, 0)).Follow(npc, leader); dd.Action != ActNone {
			t.Fatalf("decision %s, want none", describe(dd))
		}
	})
}

func TestBuff(t *testing.T) {
	d := repo(t)
	// Might (1068, bucket 12) and Chant of Life (4097, bucket 1): Chant
	// comes first.
	p := d.synthetic(t, data.NpcTemplate{ID: 8, Skills: []data.NpcSkill{{ID: 1068, Level: 1}, {ID: 4097, Level: 5}}})
	if got := ids(p.Skills(ScopeBuff)); len(got) != 2 || got[0] != 4097 || got[1] != 1068 {
		t.Fatalf("buffs %v, want [4097 1068]", got)
	}
	might := d.skill(t, 1068, 1)
	for _, tt := range []struct {
		name string
		set  func(*fake)
		want int // 0: none
	}{
		{"hurt", func(f *fake) { f.hp = 999 }, 4097},
		{"full HP skips Chant of Life", func(*fake) {}, 1068},
		{"has Might", func(f *fake) { f.affected = []int{1068} }, 0},
		{"MP equal to the cost", func(f *fake) { f.mp = float64(might.MPConsume) }, 0},
		{"hurt, but has Chant of Life", func(f *fake) { f.hp, f.affected = 10, []int{4097} }, 1068},
	} {
		npc := monster("npc", 0, 0, p)
		tt.set(npc)
		dd := Buff(npc)
		switch {
		case tt.want == 0 && dd.Action != ActNone:
			t.Errorf("%s: %s, want none", tt.name, describe(dd))
		case tt.want != 0 && (dd.Action != ActCast || dd.Skill.ID != tt.want || dd.Target != Creature(npc) || dd.StopMove || dd.Restore != nil):
			t.Errorf("%s: %s, want a cast of %d on itself", tt.name, describe(dd), tt.want)
		}
	}
}
