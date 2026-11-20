package effects_test

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/data"
	sp "github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/skills/effects"
	"github.com/IPeter/l2go/internal/skills/skilltest"
)

// waveC are the handlers of Wave C.
var waveC = []data.EffectName{
	data.EffectTriggerSkillByDamageReceived, data.EffectChangeHairStyle, data.EffectRunAway,
	data.EffectTargetMeProbability, data.EffectChangeHairColor, data.EffectGrow, data.EffectSpoil,
	data.EffectAddHate, data.EffectChangeFace, data.EffectGetAgro, data.EffectTargetMe,
	data.EffectDeleteHate, data.EffectDeleteHateOfMe, data.EffectSweeper, data.EffectConfuse,
	data.EffectElementSeed, data.EffectRandomizeHate, data.EffectBluff, data.EffectCpDamPercent,
	data.EffectGiveSp, data.EffectPassive, data.EffectResurrectionSpecial, data.EffectStaticDamage,
	data.EffectTeleport, data.EffectBackstab, data.EffectBetray, data.EffectCallSkill,
	data.EffectChameleonRest, data.EffectDistrust, data.EffectFakeDeath, data.EffectNoblesseBless,
	data.EffectPhysicalDamageHpLink, data.EffectPolearmSingleTarget, data.EffectProtectionBlessing,
	data.EffectRebalanceHP, data.EffectRecovery, data.EffectRelax, data.EffectSkillTurning,
	data.EffectTeleportToTarget,
}

func TestPortedHasWaveC(t *testing.T) {
	if len(waveC) != 39 {
		t.Fatalf("%d Wave C handlers, want 39", len(waveC))
	}
	got := effects.Ported()
	for _, n := range waveC {
		if !slices.Contains(got, n) {
			t.Errorf("%s is not ported", n)
		}
	}
}

// fixC is a test world for the Wave C handlers: an Env over a scripted
// source (add draws with r.Add before each action) and a catalog of the
// skills, which the Env finds (CallSkill, triggers).
type fixC struct {
	env   *skills.Env
	sched *skilltest.Sched
	world *skilltest.World
	r     *rnd.Scripted
	cat   *skills.Catalog
}

func newFixC(t *testing.T, tmpls ...data.SkillTemplate) *fixC {
	t.Helper()
	f := &fixC{r: rnd.NewScripted(t), world: &skilltest.World{}}
	f.env, f.sched = skilltest.NewEnv(t, f.r, f.world)
	f.cat = skilltest.Catalog(t, effects.NewRegistry(), tmpls...)
	f.env.Skills = f.cat
	return f
}

// skill returns level 1 of a skill, failing on a refused one.
func (f *fixC) skill(t *testing.T, id int) *skills.Skill {
	t.Helper()
	sk := skilltest.Skill(t, f.cat, id)
	if r := sk.Refusal(); r != "" {
		t.Fatalf("skill %d refused: %s", id, r)
	}
	return sk
}

// oneC is a fixture with skill 1 made of one GENERAL effect.
func oneC(t *testing.T, values map[string]string, name string, params map[string]string) (*fixC, *skills.Skill) {
	t.Helper()
	f := newFixC(t, skilltest.Template(1, values, effect(name, params)))
	return f, f.skill(t, 1)
}

// timed are the values of a continuous skill lasting 60 s.
var timed = map[string]string{"operateType": "A2", "abnormalTime": "60"}

func aiOf(c *skilltest.Creature) []skills.AIEvent { return c.AIEvents() }

func TestTriggerSkillByDamageReceived(t *testing.T) {
	f := newFixC(t,
		skilltest.Template(1, timed, effect("TriggerSkillByDamageReceived", map[string]string{
			"skillId": "2", "chance": "50", "minDamage": "10", "attackerType": "Playable", "targetType": "SELF",
		})),
		skilltest.Template(2, map[string]string{"targetType": "SELF"}, effect("HealPercent", map[string]string{"power": "10"})))
	p, m, a := skilltest.NewPlayer(f.env, 1), skilltest.NewMonster(f.env, 2), skilltest.NewPlayer(f.env, 3)
	f.skill(t, 1).ApplyTo(f.env, p, p)
	if p.Listeners() != 1 {
		t.Fatalf("%d listeners, want 1", p.Listeners())
	}
	// A monster draws the chance, then fails the attacker type.
	f.r.Add(rnd.Get(100, 10))
	p.NotifyDamageReceived(100, m, nil, false, false)
	// Too little damage and damage over time draw nothing.
	p.NotifyDamageReceived(5, a, nil, false, false)
	p.NotifyDamageReceived(100, a, nil, false, true)
	// A roll above the chance fails; at the chance it triggers.
	f.r.Add(rnd.Get(100, 51), rnd.Get(100, 50))
	p.NotifyDamageReceived(100, a, nil, false, false)
	p.NotifyDamageReceived(100, a, nil, false, false)
	if len(p.TriggerCasts) != 1 || p.TriggerCasts[0].Skill.ID != 2 || p.TriggerCasts[0].Target != skills.Creature(p) || p.TriggerCasts[0].IgnoreTargetType {
		t.Fatalf("trigger casts %+v", p.TriggerCasts)
	}
	p.EL.StopSkill(skills.FinishNormal, 1)
	if p.Listeners() != 0 {
		t.Fatalf("%d listeners after the end", p.Listeners())
	}

	t.Run("ONE needs the attacker to target the effected", func(t *testing.T) {
		f := newFixC(t,
			skilltest.Template(1, timed, effect("TriggerSkillByDamageReceived", map[string]string{"skillId": "2", "targetType": "ONE"})),
			skilltest.Template(2, map[string]string{"targetType": "ONE"}, effect("HealPercent", map[string]string{"power": "10"})))
		p, a := skilltest.NewPlayer(f.env, 1), skilltest.NewPlayer(f.env, 3)
		f.skill(t, 1).ApplyTo(f.env, p, p)
		p.NotifyDamageReceived(100, a, nil, false, false)
		a.Tgt = p
		f.r.Add(rnd.Get(100, 99))
		p.NotifyDamageReceived(100, a, nil, false, false)
		if len(p.TriggerCasts) != 1 || p.TriggerCasts[0].Target != skills.Creature(a) {
			t.Fatalf("trigger casts %+v", p.TriggerCasts)
		}
	})
}

func TestAppearance(t *testing.T) {
	for _, tt := range []struct {
		name string
		get  func(c *skilltest.Creature) int
	}{
		{"ChangeHairStyle", func(c *skilltest.Creature) int { return c.HairStyle }},
		{"ChangeHairColor", func(c *skilltest.Creature) int { return c.HairColor }},
		{"ChangeFace", func(c *skilltest.Creature) int { return c.Face }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f, sk := oneC(t, nil, tt.name, map[string]string{"value": "3"})
			a, b := skilltest.NewPlayer(f.env, 1), skilltest.NewPlayer(f.env, 2)
			sk.ApplyTo(f.env, a, b)
			// Mobius changes the caster.
			if tt.get(a) != 3 || a.UserInfos != 1 || tt.get(b) != 0 {
				t.Fatalf("caster %d (%d UserInfo), target %d", tt.get(a), a.UserInfos, tt.get(b))
			}
			b.AlikeDead = true
			sk.ApplyTo(f.env, b, b)
			sk.ApplyTo(f.env, b, skilltest.NewMonster(f.env, 3))
			if tt.get(b) != 0 {
				t.Fatalf("changed on a dead or NPC target")
			}
		})
	}
}

func TestRunAway(t *testing.T) {
	f, sk := oneC(t, nil, "RunAway", map[string]string{"power": "30", "time": "20"})
	a, m := skilltest.NewPlayer(f.env, 1), skilltest.NewMonster(f.env, 2)
	m.Casting, m.Abortable = true, true
	f.r.Add(rnd.Get(100, 31), rnd.Get(100, 30))
	sk.ApplyTo(f.env, a, m)
	if m.FearTime != 0 || len(m.AI) != 0 {
		t.Fatalf("ran away above the power")
	}
	sk.ApplyTo(f.env, a, m)
	if m.FearTime != 20 || !slices.Equal(m.Aborts, []string{"cast"}) || len(m.AI) != 1 ||
		m.AI[0] != (skilltest.AIEvent{Event: skills.AIAfraidStart, Other: a}) {
		t.Fatalf("fear %d, aborts %v, AI %v", m.FearTime, m.Aborts, m.AI)
	}
	sk.ApplyTo(f.env, a, skilltest.NewPlayer(f.env, 3)) // no draw for a player
}

// probabilityValues make Probability's rate 60 with chance 60 against a
// level 40 target with 30 INT: 40 + 60 − 40 + 30 − 30.
var probabilityValues = map[string]string{"magicLevel": "40"}

func probTarget(c *skilltest.Creature) *skilltest.Creature {
	c.Lvl = 40
	c.Base = map[data.BaseStat]int{data.BaseINT: 30}
	return c
}

func TestTargetMeProbability(t *testing.T) {
	f, sk := oneC(t, probabilityValues, "TargetMeProbability", map[string]string{"chance": "60"})
	a, b := skilltest.NewPlayer(f.env, 1), probTarget(skilltest.NewPlayer(f.env, 2))
	f.r.Add(rnd.Get(100, 60), rnd.Get(100, 59))
	sk.ApplyTo(f.env, a, b)
	if b.Tgt != nil {
		t.Fatalf("targeted on a failed roll")
	}
	sk.ApplyTo(f.env, a, b)
	if b.Tgt != skills.Creature(a) {
		t.Fatalf("target %v, want the caster", b.Tgt)
	}
	// An NPC caster clears the target, as in Mobius.
	m := skilltest.NewMonster(f.env, 3)
	f.r.Add(rnd.Get(100, 0))
	sk.ApplyTo(f.env, m, b)
	if b.Tgt != nil {
		t.Fatalf("target %v after an NPC's cast", b.Tgt)
	}
}

func TestGrow(t *testing.T) {
	f, sk := oneC(t, timed, "Grow", nil)
	m := skilltest.NewMonster(f.env, 2)
	sk.ApplyTo(f.env, skilltest.NewPlayer(f.env, 1), m)
	if !m.Grown || !sk.HasEffectType(skills.TypeBuff) {
		t.Fatalf("not grown")
	}
	f.sched.Advance(61 * time.Second)
	if m.Grown {
		t.Fatalf("still grown after the end")
	}
}

func TestSpoil(t *testing.T) {
	f, sk := oneC(t, nil, "Spoil", nil)
	a, m := skilltest.NewPlayer(f.env, 1), skilltest.NewMonster(f.env, 2)
	// MagicSuccess: 100 − 1 against Get(100).
	f.r.Add(rnd.Get(100, 98), rnd.Get(100, 98), rnd.Get(100, 98))
	sk.ApplyTo(f.env, a, m)
	sk.ApplyTo(f.env, a, m)
	sk.ApplyTo(f.env, a, skilltest.NewPlayer(f.env, 3))
	want := []sp.SystemMessageID{612, 357, 109}
	if m.Spoiler != 1 || !slices.Equal(a.MessageIDs(), want) || !slices.Equal(aiOf(m), []skills.AIEvent{skills.AIAttacked}) {
		t.Fatalf("spoiler %d, messages %v, AI %v", m.Spoiler, a.MessageIDs(), m.AI)
	}
	f.r.Add(rnd.Get(100, 99))
	sk.ApplyTo(f.env, a, skilltest.NewMonster(f.env, 4))
	if len(a.Messages) != 3 {
		t.Fatalf("a failed roll spoiled")
	}
}

func TestAddHate(t *testing.T) {
	f := newFixC(t,
		skilltest.Template(1, nil, effect("AddHate", map[string]string{"power": "500.7"})),
		skilltest.Template(2, nil, effect("AddHate", map[string]string{"power": "-200"})))
	a, m := skilltest.NewPlayer(f.env, 1), skilltest.NewMonster(f.env, 2)
	f.skill(t, 1).ApplyTo(f.env, a, m)
	f.skill(t, 2).ApplyTo(f.env, a, m)
	if m.Hating(a) != 300 {
		t.Fatalf("hate %d, want 300", m.Hating(a))
	}
	f.skill(t, 1).ApplyTo(f.env, a, skilltest.NewPlayer(f.env, 3))
}

func TestGetAgro(t *testing.T) {
	t.Run("monsters and their clan", func(t *testing.T) {
		f, sk := oneC(t, nil, "GetAgro", map[string]string{"chance": "60"})
		a, m := skilltest.NewPlayer(f.env, 1), skilltest.NewMonster(f.env, 2)
		m.Clans, m.ClanRange = []string{"orc"}, 300
		near, stuck, other, far := skilltest.NewMonster(f.env, 3), skilltest.NewMonster(f.env, 4), skilltest.NewMonster(f.env, 5), skilltest.NewMonster(f.env, 6)
		near.Clans, stuck.Clans, other.Clans, far.Clans = []string{"orc"}, []string{"orc"}, []string{"elf"}, []string{"orc"}
		stuck.NoMove = true
		far.Loc = skills.Location{X: 301}
		f.world.Add(a, m, near, stuck, other, far)
		f.r.Add(rnd.Get(100, 60), rnd.Get(100, 59))
		sk.ApplyTo(f.env, a, m)
		if len(m.AI) != 0 {
			t.Fatalf("aggro on a failed roll")
		}
		sk.ApplyTo(f.env, a, m)
		if !slices.Equal(aiOf(m), []skills.AIEvent{skills.AIAttack}) || m.AI[0].Other != skills.Creature(a) {
			t.Fatalf("target AI %v", m.AI)
		}
		if len(near.HateList) != 1 || near.HateList[0] != (skilltest.HateEntry{Creature: a, Damage: 1, Hate: 200}) || !near.Running ||
			!slices.Equal(aiOf(near), []skills.AIEvent{skills.AIAttack}) {
			t.Fatalf("clan member: hate %v, running %v, AI %v", near.HateList, near.Running, near.AI)
		}
		for _, c := range []*skilltest.Creature{stuck, other, far} {
			if len(c.HateList) != 0 || len(c.AI) != 0 {
				t.Errorf("%s helped", c.Nm)
			}
		}
	})
	t.Run("players", func(t *testing.T) {
		f, sk := oneC(t, nil, "GetAgro", map[string]string{"affectPlayers": "true", "range": "500"})
		a, b := skilltest.NewPlayer(f.env, 1), skilltest.NewPlayer(f.env, 2)
		stranger, friend, clanmate := skilltest.NewPlayer(f.env, 3), skilltest.NewPlayer(f.env, 4), skilltest.NewPlayer(f.env, 5)
		a.Clan, clanmate.Clan = 7, 7
		f.world.Parties = [][]skills.Creature{{a, friend}}
		f.world.Add(a, b, stranger, friend, clanmate)
		f.r.Add(rnd.Get(100, 99))
		sk.ApplyTo(f.env, a, b)
		for _, c := range []*skilltest.Creature{b, stranger} {
			if !slices.Equal(aiOf(c), []skills.AIEvent{skills.AIAttack}) {
				t.Errorf("%s: AI %v", c.Nm, c.AI)
			}
		}
		for _, c := range []*skilltest.Creature{a, friend, clanmate} {
			if len(c.AI) != 0 {
				t.Errorf("%s attacks", c.Nm)
			}
		}
	})
}

func TestTargetMe(t *testing.T) {
	f, sk := oneC(t, timed, "TargetMe", nil)
	a, b, m := skilltest.NewPlayer(f.env, 1), skilltest.NewPlayer(f.env, 2), skilltest.NewMonster(f.env, 3)
	sk.ApplyTo(f.env, a, b)
	sk.ApplyTo(f.env, a, m)
	if b.Locked != skills.Creature(a) || !slices.Equal(aiOf(b), []skills.AIEvent{skills.AIAttack}) {
		t.Fatalf("player: locked %v, AI %v", b.Locked, b.AI)
	}
	if !slices.Equal(aiOf(m), []skills.AIEvent{skills.AIAttack}) {
		t.Fatalf("monster: AI %v", m.AI)
	}
	b.EL.StopSkill(skills.FinishNormal, 1)
	if b.Locked != nil {
		t.Fatalf("still locked")
	}
}

func hated(f *fixC, cs ...*skilltest.Creature) *skilltest.Creature {
	m := probTarget(skilltest.NewMonster(f.env, 9))
	for i, c := range cs {
		m.AddDamageHate(c, 0, int64(100*(i+1)))
	}
	m.Running = true
	return m
}

func TestDeleteHate(t *testing.T) {
	f, sk := oneC(t, probabilityValues, "DeleteHate", map[string]string{"chance": "60"})
	a, b := skilltest.NewPlayer(f.env, 1), skilltest.NewPlayer(f.env, 2)
	m := hated(f, a, b)
	f.r.Add(rnd.Get(100, 59))
	sk.ApplyTo(f.env, a, m)
	if len(m.HateList) != 0 || m.Running || !slices.Equal(aiOf(m), []skills.AIEvent{skills.AIActive}) || !sk.HasEffectType(skills.TypeHate) {
		t.Fatalf("hate %v, running %v, AI %v", m.HateList, m.Running, m.AI)
	}
}

func TestDeleteHateOfMe(t *testing.T) {
	f, sk := oneC(t, probabilityValues, "DeleteHateOfMe", map[string]string{"chance": "60"})
	a, b := skilltest.NewPlayer(f.env, 1), skilltest.NewPlayer(f.env, 2)
	m := hated(f, a, b)
	f.r.Add(rnd.Get(100, 60), rnd.Get(100, 59))
	sk.ApplyTo(f.env, a, m)
	if m.Hating(a) != 100 {
		t.Fatalf("hate dropped on a failed roll")
	}
	sk.ApplyTo(f.env, a, m)
	if m.Hating(a) != 0 || m.Hating(b) != 200 || m.Running || !slices.Equal(aiOf(m), []skills.AIEvent{skills.AIActive}) {
		t.Fatalf("hate %v, running %v, AI %v", m.HateList, m.Running, m.AI)
	}
}

func TestSweeper(t *testing.T) {
	loot := []skills.ItemHolder{{ID: 57, Count: 10}, {ID: 1864, Count: 2}}
	newCorpse := func(f *fixC) *skilltest.Creature {
		m := skilltest.NewMonster(f.env, 2)
		m.Dead = true
		m.SpoilLoot = slices.Clone(loot)
		return m
	}
	t.Run("alone", func(t *testing.T) {
		f, sk := oneC(t, nil, "Sweeper", nil)
		a := skilltest.NewPlayer(f.env, 1)
		m := newCorpse(f)
		sk.ApplyTo(f.env, a, m)
		if len(a.Items) != 2 || a.Items[0].ItemID != 57 || a.Items[1].ItemID != 1864 || !slices.Equal(a.ItemMessages, []bool{true, true}) {
			t.Fatalf("items %+v", a.Items)
		}
		if len(a.InventoryChecks) != 1 || !slices.Equal(a.InventoryChecks[0], []int{57, 1864}) || m.SpoilLoot != nil {
			t.Fatalf("checks %v, loot left %v", a.InventoryChecks, m.SpoilLoot)
		}
	})
	t.Run("in a party", func(t *testing.T) {
		f, sk := oneC(t, nil, "Sweeper", nil)
		a := skilltest.NewPlayer(f.env, 1)
		f.world.Parties = [][]skills.Creature{{a, skilltest.NewPlayer(f.env, 3)}}
		sk.ApplyTo(f.env, a, newCorpse(f))
		if !slices.Equal(a.PartyItems, loot) || len(a.Items) != 0 {
			t.Fatalf("party items %v, own %v", a.PartyItems, a.Items)
		}
	})
	t.Run("refused", func(t *testing.T) {
		f, sk := oneC(t, nil, "Sweeper", nil)
		a := skilltest.NewPlayer(f.env, 1)
		notOwner, full := newCorpse(f), newCorpse(f)
		notOwner.NotSpoilOwner = true
		sk.ApplyTo(f.env, a, notOwner)
		a.InventoryFull = true
		sk.ApplyTo(f.env, a, full)
		if len(a.Items) != 0 || len(notOwner.SpoilLoot) != 2 || len(full.SpoilLoot) != 2 {
			t.Fatalf("swept: %v", a.Items)
		}
	})
}

func TestConfuse(t *testing.T) {
	f, sk := oneC(t, probabilityValues, "Confuse", map[string]string{"chance": "60"})
	a, m := skilltest.NewPlayer(f.env, 1), probTarget(skilltest.NewMonster(f.env, 2))
	c3, c4 := skilltest.NewMonster(f.env, 3), skilltest.NewMonster(f.env, 4)
	f.world.Add(m, c3, c4)
	// The roll, then a reservoir draw per creature m sees: c3 is kept.
	f.r.Add(rnd.Get(100, 0), rnd.Get(1, 0), rnd.Get(2, 1))
	sk.ApplyTo(f.env, a, m)
	if m.Tgt != skills.Creature(c3) || !slices.Equal(aiOf(m), []skills.AIEvent{skills.AIConfused, skills.AIAttack}) || m.AI[1].Other != skills.Creature(c3) {
		t.Fatalf("target %v, AI %v", m.Tgt, m.AI)
	}
}

func TestElementSeed(t *testing.T) {
	f, sk := oneC(t, timed, "ElementSeed", map[string]string{"element": "WIND"})
	p, m := skilltest.NewPlayer(f.env, 1), skilltest.NewMonster(f.env, 2)
	sk.ApplyTo(f.env, p, p)
	sk.ApplyTo(f.env, p, m)
	if !slices.Equal(p.Seeds, []data.SeedElement{data.SeedWind}) {
		t.Fatalf("seeds %v", p.Seeds)
	}
}

func TestRandomizeHate(t *testing.T) {
	f, sk := oneC(t, probabilityValues, "RandomizeHate", map[string]string{"chance": "60"})
	a, b, c := skilltest.NewPlayer(f.env, 1), skilltest.NewPlayer(f.env, 2), skilltest.NewPlayer(f.env, 3)
	m := hated(f, a, b, c)
	f.r.Add(rnd.Get(100, 0), rnd.Get(2, 1))
	sk.ApplyTo(f.env, a, m)
	if m.Hating(a) != 0 || m.Hating(b) != 200 || m.Hating(c) != 400 {
		t.Fatalf("hate %v", m.HateList)
	}
	// Alone on the list: nothing to move, no draw.
	m2 := hated(f, a)
	f.r.Add(rnd.Get(100, 0))
	sk.ApplyTo(f.env, a, m2)
	if m2.Hating(a) != 100 {
		t.Fatalf("hate %v", m2.HateList)
	}
}

func TestBluff(t *testing.T) {
	f, sk := oneC(t, probabilityValues, "Bluff", map[string]string{"chance": "60"})
	a, m := skilltest.NewPlayer(f.env, 1), probTarget(skilltest.NewMonster(f.env, 2))
	a.Head, m.Head = 5000, 100
	f.r.Add(rnd.Get(100, 0))
	sk.ApplyTo(f.env, a, m)
	if !slices.Equal(m.Rotations, [][3]int{{100, 1, 65535}, {5000, -1, 65535}}) || m.Head != 5000 {
		t.Fatalf("rotations %v, heading %d", m.Rotations, m.Head)
	}
	hq, raid := probTarget(skilltest.NewMonster(f.env, 3)), probTarget(skilltest.NewMonster(f.env, 4))
	hq.Npc, raid.Raid = 35062, true
	f.r.Add(rnd.Get(100, 0), rnd.Get(100, 0))
	sk.ApplyTo(f.env, a, hq)
	sk.ApplyTo(f.env, a, raid)
	if len(hq.Rotations)+len(raid.Rotations) != 0 {
		t.Fatalf("a headquarters or raid turned")
	}
}

func TestCpDamPercent(t *testing.T) {
	f, sk := oneC(t, nil, "CpDamPercent", map[string]string{"power": "90"})
	a, b := skilltest.NewPlayer(f.env, 1), skilltest.NewPlayer(f.env, 2)
	b.CP = 999
	b.FakeDead = true
	f.env.Config.FakeDeathDamageStand = true // off in Player.ini
	sk.ApplyTo(f.env, a, b)
	// int(999 × 90 / 100) = 899.
	if b.CP != 100 || !slices.Equal(b.FakeDeathCalls, []string{"stop+effects"}) {
		t.Fatalf("CP %v, fake death %v", b.CP, b.FakeDeathCalls)
	}
	// A casting target may be broken: AtkBreak draws.
	b.Break.Casting = true
	f.r.Add(rnd.Get(100, 0))
	sk.ApplyTo(f.env, a, b)
	if b.CP != 10 || b.Breaks != 1 {
		t.Fatalf("CP %v, breaks %d", b.CP, b.Breaks)
	}
	m := skilltest.NewMonster(f.env, 3)
	sk.ApplyTo(f.env, a, m)
	if m.CP != 1000 {
		t.Fatalf("an NPC lost CP")
	}
}

func TestGiveSp(t *testing.T) {
	f, sk := oneC(t, nil, "GiveSp", map[string]string{"sp": "500"})
	a, b := skilltest.NewPlayer(f.env, 1), skilltest.NewPlayer(f.env, 2)
	sk.ApplyTo(f.env, a, b)
	sk.ApplyTo(f.env, a, skilltest.NewMonster(f.env, 3))
	if !slices.Equal(a.ExpSp, [][2]float64{{0, 500}}) || len(b.ExpSp) != 0 {
		t.Fatalf("caster %v, target %v", a.ExpSp, b.ExpSp)
	}
}

func TestPassive(t *testing.T) {
	f, sk := oneC(t, timed, "Passive", nil)
	a, m, b := skilltest.NewPlayer(f.env, 1), skilltest.NewMonster(f.env, 2), skilltest.NewPlayer(f.env, 3)
	m.Tgt, m.Running = a, true
	sk.ApplyTo(f.env, a, m)
	sk.ApplyTo(f.env, a, b)
	if !m.EL.IsAffected(skills.FlagPassive) || m.Tgt != nil || m.Running || !slices.Equal(aiOf(m), []skills.AIEvent{skills.AIActive}) {
		t.Fatalf("flag %v, target %v, running %v, AI %v", m.EL.IsAffected(skills.FlagPassive), m.Tgt, m.Running, m.AI)
	}
	if b.EL.IsAffected(skills.FlagPassive) {
		t.Fatalf("a player turned passive")
	}
}

func TestResurrectionSpecial(t *testing.T) {
	f, sk := oneC(t, timed, "ResurrectionSpecial", map[string]string{"power": "70"})
	a, b, c := skilltest.NewPlayer(f.env, 1), skilltest.NewPlayer(f.env, 2), skilltest.NewPlayer(f.env, 3)
	c.Olympiad = true
	sk.ApplyTo(f.env, a, b)
	sk.ApplyTo(f.env, a, c)
	if len(b.ReviveAsks) != 0 || !b.EL.IsAffected(skills.FlagResurrectionSpecial) {
		t.Fatalf("asked before the end")
	}
	f.sched.Advance(61 * time.Second)
	if !slices.Equal(b.ReviveAsks, []skilltest.ReviveAsk{{Reviver: a, Pet: false, Power: 70}}) || len(c.ReviveAsks) != 0 {
		t.Fatalf("asks %v, in the Olympiad %v", b.ReviveAsks, c.ReviveAsks)
	}
}

func TestStaticDamage(t *testing.T) {
	f, sk := oneC(t, nil, "StaticDamage", map[string]string{"power": "100"})
	a, m := skilltest.NewPlayer(f.env, 1), skilltest.NewMonster(f.env, 2)
	sk.ApplyTo(f.env, a, m)
	sk.ApplyTo(f.env, m, a)
	if !slices.Equal(m.Damage, []float64{100}) || !slices.Equal(a.DamageMsgs, []int{100}) || !slices.Equal(a.Damage, []float64{100}) || len(m.DamageMsgs) != 0 {
		t.Fatalf("monster %v %v, player %v %v", m.Damage, m.DamageMsgs, a.Damage, a.DamageMsgs)
	}
	a.AlikeDead = true
	sk.ApplyTo(f.env, a, m)
	if len(m.Damage) != 1 {
		t.Fatalf("a dead caster dealt damage")
	}
}

func TestTeleport(t *testing.T) {
	f, sk := oneC(t, nil, "Teleport", map[string]string{"x": "-80826", "y": "149775", "z": "-3043"})
	p := skilltest.NewPlayer(f.env, 1)
	sk.ApplyTo(f.env, p, p)
	if !slices.Equal(p.Teleports, []skills.Location{{X: -80826, Y: 149775, Z: -3043}}) || !sk.HasEffectType(skills.TypeTeleport) {
		t.Fatalf("teleports %v", p.Teleports)
	}
}

func TestBackstab(t *testing.T) {
	values := map[string]string{"power": "100", "blowChance": "20", "targetType": "ONE", "effectPoint": "-100"}
	f, sk := oneC(t, values, "Backstab", nil)
	a, m := skilltest.NewPlayer(f.env, 1), skilltest.NewMonster(f.env, 2)
	a.Loc = skills.Location{X: -100} // behind a monster facing heading 0
	// The evasion, the blow chance (20 × 2 behind), the skill critical, the
	// counter-attack.
	f.r.Add(rnd.Get(100, 50), rnd.Get(100, 39), rnd.Get(1000, 500), rnd.Get(100, 50))
	sk.ApplyTo(f.env, a, m)
	// 77 × (100 + 100) / 100 × 1.2 from behind, in doubles: 184.79999999999998.
	baseMod, prox := 77.0*200/100, 1.2
	want := baseMod * prox
	if len(m.Damage) != 1 || m.Damage[0] != want || !slices.Equal(a.DamageMsgs, []int{184}) {
		t.Fatalf("damage %v, messages %v, want %v", m.Damage, a.DamageMsgs, want)
	}
	// In front of the target the blow never lands, without a draw.
	a.Loc = skills.Location{X: 100}
	sk.ApplyTo(f.env, a, m)
	// A missed blow chance deals nothing.
	a.Loc = skills.Location{X: -100}
	f.r.Add(rnd.Get(100, 50), rnd.Get(100, 40))
	sk.ApplyTo(f.env, a, m)
	if len(m.Damage) != 1 {
		t.Fatalf("damage %v", m.Damage)
	}
}

func TestBetray(t *testing.T) {
	f, sk := oneC(t, timed, "Betray", nil)
	a, s := skilltest.NewPlayer(f.env, 1), skilltest.New(f.env, 2, data.InstanceServitor)
	sk.ApplyTo(f.env, a, s)
	if !s.EL.IsAffected(skills.FlagBetrayed) || !slices.Equal(aiOf(s), []skills.AIEvent{skills.AIAttack}) || !sk.HasEffectType(skills.TypeDebuff) {
		t.Fatalf("flag %v, AI %v", s.EL.IsAffected(skills.FlagBetrayed), s.AI)
	}
	s.EL.StopSkill(skills.FinishNormal, 1)
	if !slices.Equal(aiOf(s), []skills.AIEvent{skills.AIAttack, skills.AIIdle}) {
		t.Fatalf("AI %v", s.AI)
	}
	m := skilltest.NewMonster(f.env, 3)
	sk.ApplyTo(f.env, m, skilltest.New(f.env, 4, data.InstanceServitor))
	sk.ApplyTo(f.env, a, m)
	if m.EL.IsAffected(skills.FlagBetrayed) {
		t.Fatalf("betrayed a monster")
	}
}

func TestCallSkill(t *testing.T) {
	f := newFixC(t,
		skilltest.Template(1, nil, effect("CallSkill", map[string]string{"skillId": "2"})),
		skilltest.Template(2, nil, effect("CallSkill", map[string]string{"skillId": "2"})),
		skilltest.Template(3, nil, effect("CallSkill", map[string]string{"skillId": "9"})))
	a, m := skilltest.NewPlayer(f.env, 1), skilltest.NewMonster(f.env, 2)
	f.skill(t, 1).ApplyTo(f.env, a, m)
	f.skill(t, 2).ApplyTo(f.env, a, m) // calls itself: nothing
	f.skill(t, 3).ApplyTo(f.env, a, m) // no such skill: nothing
	if len(a.TriggerCasts) != 1 || a.TriggerCasts[0] != (skilltest.TriggerCast{Skill: f.skill(t, 2), Target: m, IgnoreTargetType: true}) {
		t.Fatalf("trigger casts %+v", a.TriggerCasts)
	}
}

// tick is the period of an effect with ticks.
func tick(ticks int) time.Duration { return time.Duration(ticks*666) * time.Millisecond }

func perTick(ticks int) float64 { return float64(float32(ticks*666) / 1000) }

func TestChameleonRest(t *testing.T) {
	f := newFixC(t, skilltest.Template(1, map[string]string{"operateType": "T"},
		skilltest.General(data.EffectDef{Name: "ChameleonRest", Ticks: "5", Params: map[string]string{"power": "2"}})))
	sk := f.skill(t, 1)
	p := skilltest.NewPlayer(f.env, 1)
	sk.ApplyTo(f.env, p, p)
	if p.SitDowns != 1 || !p.EL.IsAffected(skills.FlagSilentMove) || !p.EL.IsAffected(skills.FlagRelaxing) {
		t.Fatalf("sat %d, flags %b", p.SitDowns, p.EL.Flags())
	}
	f.sched.Advance(tick(5))
	if want := 1000 - 2*perTick(5); p.MP != want {
		t.Fatalf("MP %v, want %v", p.MP, want)
	}
	p.MP = 1
	f.sched.Advance(tick(5))
	if p.EL.BySkillID(1) != nil || !slices.Contains(p.MessageIDs(), 140) {
		t.Fatalf("toggle kept going without MP: %v", p.MessageIDs())
	}
	m := skilltest.NewMonster(f.env, 2)
	sk.ApplyTo(f.env, m, m)
	if !slices.Equal(aiOf(m), []skills.AIEvent{skills.AIRest}) {
		t.Fatalf("NPC AI %v", m.AI)
	}
}

func TestRelax(t *testing.T) {
	f := newFixC(t, skilltest.Template(1, map[string]string{"operateType": "T"},
		skilltest.General(data.EffectDef{Name: "Relax", Ticks: "3", Params: map[string]string{"power": "1"}})))
	sk := f.skill(t, 1)
	p := skilltest.NewPlayer(f.env, 1)
	p.HP = 500
	sk.ApplyTo(f.env, p, p)
	f.sched.Advance(tick(3))
	if want := 1000 - perTick(3); p.MP != want || !p.EL.IsAffected(skills.FlagRelaxing) {
		t.Fatalf("MP %v, want %v", p.MP, want)
	}
	p.HP = 1000
	f.sched.Advance(tick(3))
	if p.EL.BySkillID(1) != nil || !slices.Contains(p.MessageIDs(), 175) {
		t.Fatalf("toggle kept going at full HP: %v", p.MessageIDs())
	}
	// Standing up ends it.
	p.HP = 500
	sk.ApplyTo(f.env, p, p)
	p.Sitting = false
	f.sched.Advance(tick(3))
	if p.EL.BySkillID(1) != nil {
		t.Fatalf("toggle kept going standing")
	}
}

func TestFakeDeath(t *testing.T) {
	f := newFixC(t, skilltest.Template(1, map[string]string{"operateType": "T"},
		skilltest.General(data.EffectDef{Name: "FakeDeath", Ticks: "5", Params: map[string]string{"power": "10"}})))
	sk := f.skill(t, 1)
	p := skilltest.NewPlayer(f.env, 1)
	sk.ApplyTo(f.env, p, p)
	if !p.FakeDead || !sk.HasEffectType(skills.TypeFakeDeath) {
		t.Fatalf("not faking")
	}
	f.sched.Advance(tick(5))
	if want := 1000 - 10*perTick(5); p.MP != want {
		t.Fatalf("MP %v, want %v", p.MP, want)
	}
	p.EL.StopSkill(skills.FinishRemoved, 1)
	if p.FakeDead || !p.RecentFakeDeath || !slices.Equal(p.FakeDeathCalls, []string{"start", "end"}) {
		t.Fatalf("fake %v, recent %v, calls %v", p.FakeDead, p.RecentFakeDeath, p.FakeDeathCalls)
	}
}

func TestDistrust(t *testing.T) {
	f, sk := oneC(t, timed, "Distrust", nil)
	a, m := skilltest.NewPlayer(f.env, 1), skilltest.NewMonster(f.env, 2)
	other, chest, raid, far := skilltest.NewMonster(f.env, 3), skilltest.New(f.env, 4, data.InstanceChest), skilltest.NewMonster(f.env, 5), skilltest.NewMonster(f.env, 6)
	raid.Raid = true
	far.Loc = skills.Location{X: 1101}
	f.world.Add(a, m, other, chest, raid, far)
	f.r.Add(rnd.Get(1, 0))
	sk.ApplyTo(f.env, a, m)
	if m.Distrusted != skills.Creature(other) {
		t.Fatalf("distrusts %v", m.Distrusted)
	}
	m.Tgt, m.Running = other, true
	m.EL.StopSkill(skills.FinishNormal, 1)
	if m.Distrusted != nil || m.DistrustStops != 1 || m.Tgt != nil || m.Running || !slices.Equal(aiOf(m), []skills.AIEvent{skills.AIActive}) {
		t.Fatalf("after the end: %v, target %v, AI %v", m.Distrusted, m.Tgt, m.AI)
	}
	sk.ApplyTo(f.env, a, raid)
	if !slices.Equal(a.MessageIDs(), []sp.SystemMessageID{109}) {
		t.Fatalf("messages %v", a.MessageIDs())
	}
}

func TestBlessingsAndPolearm(t *testing.T) {
	for _, tt := range []struct {
		name   string
		flag   skills.EffectFlag
		player bool // the player gets it
		npc    bool // the monster gets it
	}{
		{"NoblesseBless", skills.FlagNoblessBlessing, true, false},
		{"ProtectionBlessing", skills.FlagProtectionBlessing, true, false},
		{"PolearmSingleTarget", skills.FlagPolearmSingleTarget, true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f, sk := oneC(t, timed, tt.name, nil)
			p, m := skilltest.NewPlayer(f.env, 1), skilltest.NewMonster(f.env, 2)
			sk.ApplyTo(f.env, p, p)
			sk.ApplyTo(f.env, p, m)
			if p.EL.IsAffected(tt.flag) != tt.player || m.EL.IsAffected(tt.flag) != tt.npc {
				t.Fatalf("player %v, monster %v", p.EL.IsAffected(tt.flag), m.EL.IsAffected(tt.flag))
			}
		})
	}
	f, sk := oneC(t, timed, "NoblesseBless", nil)
	if !sk.HasEffectType(skills.TypeNoblesseBlessing) {
		t.Fatalf("type")
	}
	_ = f
}

func TestPhysicalDamageHpLink(t *testing.T) {
	values := map[string]string{"power": "100", "targetType": "ONE", "effectPoint": "-100"}
	f, sk := oneC(t, values, "PhysicalDamageHpLink", nil)
	a, m := skilltest.NewPlayer(f.env, 1), skilltest.NewMonster(f.env, 2)
	a.Loc = skills.Location{X: -100}
	m.HP = 250
	// The evasion, then the counter-attack.
	f.r.Add(rnd.Get(100, 50), rnd.Get(100, 50))
	sk.ApplyTo(f.env, a, m)
	// Power 100 × (2 − 2 × 250 / 1000) = 150; (100 + 150) × 76 × 1.2 / 100.
	if !slices.Equal(m.Damage, []float64{228}) || !sk.HasEffectType(skills.TypePhysicalAttackHPLink) {
		t.Fatalf("damage %v", m.Damage)
	}
	a.NoMove = true
	f.r.Add(rnd.Get(100, 50))
	sk.ApplyTo(f.env, a, m)
	if len(m.Damage) != 1 || !slices.Equal(a.MessageIDs(), []sp.SystemMessageID{skills.MsgS1CannotBeUsedUnsuitable}) {
		t.Fatalf("damage %v, messages %v", m.Damage, a.MessageIDs())
	}
}

func TestRebalanceHP(t *testing.T) {
	f, sk := oneC(t, map[string]string{"targetType": "PARTY", "affectRange": "1000"}, "RebalanceHP", nil)
	a, b := skilltest.NewPlayer(f.env, 1), skilltest.NewPlayer(f.env, 2)
	s, far := skilltest.New(f.env, 3, data.InstanceServitor), skilltest.NewPlayer(f.env, 4)
	far.Loc = skills.Location{X: 2000}
	a.HP, b.HP, s.HP, far.HP = 900, 100, 200, 50
	f.world.Parties = [][]skills.Creature{{a, b, far}}
	f.world.Summons = map[int32]skills.Creature{b.ID: s}
	sk.ApplyTo(f.env, a, a)
	// 1200 of 3000 HP: 40 % each.
	if a.HP != 400 || b.HP != 400 || s.HP != 400 || far.HP != 50 {
		t.Fatalf("HP %v %v %v %v", a.HP, b.HP, s.HP, far.HP)
	}
	f.world.Parties = nil
	a.HP = 900
	sk.ApplyTo(f.env, a, a)
	if a.HP != 900 {
		t.Fatalf("rebalanced without a party")
	}
}

func TestRecovery(t *testing.T) {
	f, sk := oneC(t, nil, "Recovery", nil)
	p := skilltest.NewPlayer(f.env, 1)
	sk.ApplyTo(f.env, p, p)
	sk.ApplyTo(f.env, p, skilltest.NewMonster(f.env, 2))
	if p.PenaltyDrops != 1 {
		t.Fatalf("%d penalty drops", p.PenaltyDrops)
	}
}

func TestSkillTurning(t *testing.T) {
	f, sk := oneC(t, probabilityValues, "SkillTurning", map[string]string{"chance": "60"})
	a, m, raid := skilltest.NewPlayer(f.env, 1), probTarget(skilltest.NewMonster(f.env, 2)), probTarget(skilltest.NewMonster(f.env, 3))
	raid.Raid = true
	f.r.Add(rnd.Get(100, 60), rnd.Get(100, 59), rnd.Get(100, 0))
	sk.ApplyTo(f.env, a, m)
	sk.ApplyTo(f.env, a, m)
	sk.ApplyTo(f.env, a, raid)
	if m.Breaks != 1 || raid.Breaks != 0 {
		t.Fatalf("breaks %d, raid %d", m.Breaks, raid.Breaks)
	}
}

func TestTeleportToTarget(t *testing.T) {
	f, sk := oneC(t, nil, "TeleportToTarget", nil)
	a, m := skilltest.NewPlayer(f.env, 1), skilltest.NewMonster(f.env, 2)
	m.Loc = skills.Location{X: 1000, Y: 2000, Z: 30}
	sk.ApplyTo(f.env, a, m)
	// 25 behind a target facing heading 0.
	if a.Loc != (skills.Location{X: 975, Y: 2000, Z: 30}) || a.Validates != 1 ||
		!slices.Equal(a.Aborts, []string{"attack", "cast"}) || !slices.Equal(aiOf(a), []skills.AIEvent{skills.AIIdle}) {
		t.Fatalf("at %v, validates %d, aborts %v, AI %v", a.Loc, a.Validates, a.Aborts, a.AI)
	}
}

func TestMobiusDataWaveC(t *testing.T) {
	d, err := data.LoadSkills(repoData)
	if err != nil {
		t.Fatalf("LoadSkills: %v", err)
	}
	cat, err := skills.NewCatalog(d, effects.NewRegistry())
	if err != nil {
		t.Fatal(err)
	}
	r := cat.Report()
	for _, n := range waveC {
		if r.EffectSkills[n] == 0 || !r.Ported[n] {
			t.Errorf("%s: used by %d skills, ported %v", n, r.EffectSkills[n], r.Ported[n])
		}
	}
	for _, p := range r.Problems {
		t.Errorf("problem: %s", p)
	}
}
