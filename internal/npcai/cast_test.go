package npcai

import (
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
)

// castProfile is a template with the skills and no gates.
func castProfile(t *testing.T, sk ...data.NpcSkill) *Profile {
	t.Helper()
	return repo(t).synthetic(t, data.NpcTemplate{ID: 6, Skills: sk})
}

// cast() by effect type, with real skills.
func TestCastDebuff(t *testing.T) {
	d := repo(t)
	p := castProfile(t, data.NpcSkill{ID: 4038, Level: 4})
	sk := d.skill(t, 4038, 4) // Decrease Atk. Spd.: continuous debuff, ONE, range 600

	t.Run("in range", func(t *testing.T) {
		npc := monster("npc", 0, 0, p)
		target, w := scene(npc, 623) // 623 − 9 <= 600 + 14
		wantCast(t, newThinker(t, w).cast(npc, target, sk), 4038, target, nil, true)
	})
	t.Run("already affected: the last resort casts anyway", func(t *testing.T) {
		npc := monster("npc", 0, 0, p)
		target, w := scene(npc, 300)
		target.affected = []int{4038}
		wantCast(t, newThinker(t, w).cast(npc, target, sk), 4038, target, nil, true)
	})
	t.Run("out of range: an attacker in range, cast at the target", func(t *testing.T) {
		npc := monster("npc", 0, 0, p)
		target, w := scene(npc, 1000)
		other := player("other", 200, 0)
		npc.attackBy = []*fake{target, other}
		w.all = append(w.all, other)
		wantCast(t, newThinker(t, w).cast(npc, target, sk), 4038, target, nil, true)
	})
	t.Run("out of range and nobody else", func(t *testing.T) {
		npc := monster("npc", 0, 0, p)
		target, w := scene(npc, 1000)
		npc.aggro = nil // skillTargetReconsider finds nobody either
		if dd := newThinker(t, w).cast(npc, target, sk); dd.Action != ActNone {
			t.Fatalf("decision %s, want none", describe(dd))
		}
	})
}

func TestCastAuraDebuff(t *testing.T) {
	d := repo(t)
	p := castProfile(t, data.NpcSkill{ID: 4072, Level: 4})
	sk := d.skill(t, 4072, 4) // Shock: continuous debuff, AURA 150
	t.Run("canAOE: someone around has an effect", func(t *testing.T) {
		npc := monster("npc", 0, 0, p)
		target, w := scene(npc, 100)
		target.effects = true
		wantCast(t, newThinker(t, w).cast(npc, target, sk), 4072, target, nil, true)
	})
	t.Run("no AOE: single-target rules, then the aura", func(t *testing.T) {
		npc := monster("npc", 0, 0, p)
		target, w := scene(npc, 100)
		// Cast range -1 + 14 is short of 100 − 9; the last resort finds the
		// target under the aura.
		wantCast(t, newThinker(t, w).cast(npc, target, sk), 4072, target, nil, true)
	})
	t.Run("attackables count only for chaos NPCs", func(t *testing.T) {
		npc := monster("npc", 0, 0, p)
		target, w := scene(npc, 1000)
		buddy := monster("buddy", 50, 0, p)
		buddy.effects = true
		w.all = append(w.all, buddy)
		npc.aggro = nil
		if dd := newThinker(t, w).cast(npc, target, sk); dd.Action != ActNone {
			t.Fatalf("decision %s, want none", describe(dd))
		}
	})
}

func TestCastDamage(t *testing.T) {
	d := repo(t)
	p := castProfile(t, data.NpcSkill{ID: 4078, Level: 4})
	sk := d.skill(t, 4078, 4) // Flamestrike: AREA, range 500
	t.Run("in range", func(t *testing.T) {
		npc := monster("npc", 0, 0, p)
		target, w := scene(npc, 523)
		wantCast(t, newThinker(t, w).cast(npc, target, sk), 4078, target, nil, true)
	})
	t.Run("out of sight", func(t *testing.T) {
		npc := monster("npc", 0, 0, p)
		target, w := scene(npc, 300)
		w.blind = [][2]*fake{{npc, target}}
		other := player("other", 0, 300)
		npc.aggro = append(npc.aggro, hateEntry{other, 1})
		w.all = append(w.all, other)
		wantCast(t, newThinker(t, w).cast(npc, target, sk), 4078, other, target, true)
	})
	t.Run("out of range: another hated creature in range", func(t *testing.T) {
		npc := monster("npc", 0, 0, p)
		target, w := scene(npc, 1000)
		other := player("other", 0, 300)
		npc.aggro = append(npc.aggro, hateEntry{other, 1})
		w.all = append(w.all, other)
		wantCast(t, newThinker(t, w).cast(npc, target, sk), 4078, other, target, true)
	})
	t.Run("out of range: a visible player", func(t *testing.T) {
		npc := monster("npc", 0, 0, p)
		target, w := scene(npc, 1000)
		npc.aggro = nil
		mob := monster("mob", 0, 100, p)
		other := player("other", 0, 300)
		w.all = append(w.all, mob, other)
		wantCast(t, newThinker(t, w).cast(npc, target, sk), 4078, other, target, true)
	})
	t.Run("a guard looks no further than its enemies", func(t *testing.T) {
		npc := monster("npc", 0, 0, p)
		npc.guard = true
		target, w := scene(npc, 1000)
		npc.aggro = nil
		w.all = append(w.all, player("other", 0, 300))
		if dd := newThinker(t, w).cast(npc, target, sk); dd.Action != ActNone {
			t.Fatalf("decision %s, want none", describe(dd))
		}
	})
}

func TestCastHeal(t *testing.T) {
	d := repo(t)
	p := castProfile(t, data.NpcSkill{ID: 4065, Level: 4})
	sk := d.skill(t, 4065, 4)
	t.Run("leader out of reach: moves, then heals", func(t *testing.T) {
		npc := monster("npc", 0, 0, p)
		leader := monster("leader", 2000, 0, p)
		leader.hp = 400
		npc.leader = leader
		target, w := scene(npc, 300)
		w.all = append(w.all, leader)
		dd := newThinker(t, w, rnd.Get(100, 41)).cast(npc, target, sk)
		wantCast(t, dd, 4065, leader, target, true)
		if dd.Pawn != Creature(leader) || dd.PawnRange != 628 {
			t.Fatalf("pawn %v at %d, want the leader at 628", dd.Pawn, dd.PawnRange)
		}
	})
	t.Run("leader out of sight: the move stays", func(t *testing.T) {
		npc := monster("npc", 0, 0, p)
		leader := monster("leader", 2000, 0, p)
		leader.hp = 400
		npc.leader = leader
		target, w := scene(npc, 300)
		w.all = append(w.all, leader)
		w.blind = [][2]*fake{{npc, leader}}
		// Self heal at full HP: no. Clan: none. The last resort casts at
		// the target (a monster then drops it: DropsCast).
		dd := newThinker(t, w, rnd.Get(100, 41), rnd.Get(100, 0)).cast(npc, target, sk)
		wantCast(t, dd, 4065, target, nil, true)
		if dd.Pawn != Creature(leader) || !DropsCast(dd.Skill, dd.Target) {
			t.Fatalf("pawn %v, drops %v", dd.Pawn, DropsCast(dd.Skill, dd.Target))
		}
	})
	t.Run("self heal", func(t *testing.T) {
		npc := monster("npc", 0, 0, p)
		npc.hp = 700 // (100 − 70) / 3 = 10
		target, w := scene(npc, 300)
		wantCast(t, newThinker(t, w, rnd.Get(100, 9)).cast(npc, target, sk), 4065, npc, target, true)
	})
	t.Run("a hurt clan member", func(t *testing.T) {
		clan := data.NpcTemplate{ID: 7, Skills: []data.NpcSkill{{ID: 4065, Level: 4}}, AI: data.NpcAI{Clans: []string{"ORC"}}}
		cp := d.synthetic(t, clan)
		npc := monster("npc", 0, 0, cp)
		hurt := monster("hurt", 100, 0, cp)
		hurt.hp = 500 // (100 − 50) / 10 = 5
		target, w := scene(npc, 300)
		w.all = append(w.all, hurt)
		dd := newThinker(t, w, rnd.Get(100, 99), rnd.Get(100, 4)).cast(npc, target, sk)
		wantCast(t, dd, 4065, hurt, target, true)
	})
}

func TestCastBuff(t *testing.T) {
	d := repo(t)
	p := castProfile(t, data.NpcSkill{ID: 4097, Level: 5})
	sk := d.skill(t, 4097, 5) // NPC Chant of Life: continuous, SELF
	npc := monster("npc", 0, 0, p)
	target, w := scene(npc, 300)
	wantCast(t, newThinker(t, w).cast(npc, target, sk), 4097, npc, target, true)
	npc.affected = []int{4097}
	if dd := newThinker(t, w).cast(npc, target, sk); dd.Action != ActNone {
		t.Fatalf("decision %s, want none for a self buff it has", describe(dd))
	}
}

func TestCastConditions(t *testing.T) {
	d := repo(t)
	p := castProfile(t, data.NpcSkill{ID: 4078, Level: 4})
	sk := d.skill(t, 4078, 4) // magic
	for _, tt := range []struct {
		name string
		set  func(*fake)
		want bool
	}{
		{"ready", func(*fake) {}, true},
		{"casting", func(f *fake) { f.casting = true }, false},
		{"MP equal to the cost", func(f *fake) { f.mp = float64(sk.MPConsume) }, false},
		{"MP above the cost", func(f *fake) { f.mp = float64(sk.MPConsume) + 0.5 }, true},
		{"on reuse", func(f *fake) { f.disabled = []int{4078} }, false},
		{"muted", func(f *fake) { f.muted = true }, false},
		{"physically muted", func(f *fake) { f.pmuted = true }, false},
	} {
		npc := monster("npc", 0, 0, p)
		tt.set(npc)
		if got := canCast(npc, sk); got != tt.want {
			t.Errorf("%s: canCast %v, want %v", tt.name, got, tt.want)
		}
	}
	// A physical skill ignores a magic mute.
	stun := d.skill(t, 4072, 4)
	npc := monster("npc", 0, 0, p)
	npc.muted = true
	if !canCast(npc, stun) {
		t.Error("a muted NPC cannot use a physical skill")
	}
}

func TestDropsCast(t *testing.T) {
	d := repo(t)
	heal, flame := d.skill(t, 4065, 4), d.skill(t, 4078, 4)
	pl, mob := player("p", 0, 0), monster("m", 0, 0, nil)
	for _, tt := range []struct {
		name string
		drop bool
		got  bool
	}{
		{"a heal at a player", true, DropsCast(heal, pl)},
		{"a heal at a monster", false, DropsCast(heal, mob)},
		{"a heal without a target", false, DropsCast(heal, nil)},
		{"an attack at a player", false, DropsCast(flame, pl)},
	} {
		if tt.got != tt.drop {
			t.Errorf("%s: drops %v, want %v", tt.name, tt.got, tt.drop)
		}
	}
}
