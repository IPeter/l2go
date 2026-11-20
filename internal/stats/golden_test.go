package stats

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
)

// The Phase 6.2 goldens, hand-computed from Mobius Creature.doAttack,
// Formulas.calcHitMiss, calcCrit, calcPhysDam and calcAttackTraitBonus.

// beardedKeltir is NPC 20481 (npcs/20400-20499.xml): level 1, attack type
// SWORD, no rhand, passive 4416 level 4 (Animals).
func beardedKeltir() *NpcTemplate {
	return &NpcTemplate{
		Level:       1,
		Base:        BaseStats{STR: 40, INT: 21, DEX: 30, WIT: 20, CON: 43, MEN: 10},
		HP:          39.74519,
		MP:          40,
		HPRegen:     2,
		MPRegen:     0.9,
		PAtk:        8.47458,
		MAtk:        5.78704,
		PDef:        44.44444,
		MDef:        29.5916164000214,
		CritRate:    4,
		PAtkSpd:     253,
		AttackRange: 40,
		RunSpeed:    110,
		WalkSpeed:   23,
	}
}

// orcArcher is NPC 20006 (npcs/20000-20099.xml): level 8, attack type BOW,
// rhand 14 (Bow, reuse_delay 1500), passive 4416 level 6 (Humanoids).
func orcArcher() *NpcTemplate {
	return &NpcTemplate{
		Level:       8,
		Base:        BaseStats{STR: 40, INT: 21, DEX: 30, WIT: 20, CON: 43, MEN: 10},
		HP:          124.92828,
		MP:          92.64,
		HPRegen:     2,
		MPRegen:     0.9,
		PAtk:        16.20165,
		MAtk:        11.06363,
		PDef:        57.56203,
		MDef:        38.3254529794081,
		CritRate:    8,
		PAtkSpd:     253,
		AttackRange: 1100,
		RunSpeed:    110,
		WalkSpeed:   45,
	}
}

// keltirSwordDefence is the Keltir's SWORD defence trait from 4416 level 4,
// a hand-made fixture of the F10 passive data: the table value 1 becomes
// (1 + 100) / 100 in Java floats (DefenceTrait), 1.00999999046. It has no
// FIST entry, so fists meet the neutral 1.
var keltirSwordDefence = float32(1+100) / 100

// TestGoldenFighterVsKeltir: a level 1 Human Fighter with the Squire's
// Sword, shirt and pants, then unarmed, against a Bearded Keltir.
//
//   - Fighter with the sword: P.Atk 6 × 1.2 × 0.9 = 6.48, accuracy 34, crit
//     88, random damage 10, speed 417 (hit at 599). Unarmed: the fists are
//     the active weapon (random damage 0, dual), P.Atk 4.32, crit 44, speed
//     330 (hits at 378 and 757).
//   - Keltir: evasion 34, P.Def 44 × 0.9 = 39.6. Hit chance in front
//     (80 + 0) × 10 = 800, behind (int)(800 × 1.1) = 880.
//   - Sword trait: 1 − (1.00999999046 − 1) = 0.99000000954; the fighter has
//     no attack traits, so the Animals weakness gives 1.
func TestGoldenFighterVsKeltir(t *testing.T) {
	r := testRules(t, DefaultConfig())
	c := testCombat(t, DefaultCombatConfig())
	keltirNpc, err := NewNpc(r, beardedKeltir())
	if err != nil {
		t.Fatal(err)
	}
	keltir := Fighter{Stats: keltirNpc, Ref: "keltir", Attackable: true}
	keltir.Arm(nil, "SWORD", false)

	armed := newTestPlayer(t, r, humanFighter(), 1, fighterGear...)
	sword := Fighter{Stats: armed, Ref: "fighter", Playable: true}
	sword.Arm(&Weapon{ItemID: 2369, Type: "SWORD"}, "FIST", true)
	unarmed := Fighter{Stats: newTestPlayer(t, r, humanFighter(), 1), Ref: "fighter", Playable: true}
	fists := PlayerWeapon(0, nil)
	unarmed.Arm(&fists, "FIST", true)

	swordTrait := AttackTraitBonus(WeaponTraitBonus(keltirSwordDefence), 1)
	if swordTrait != 0.9900000095367432 {
		t.Fatalf("sword trait bonus = %v, want 0.9900000095367432", swordTrait)
	}
	fistTrait := AttackTraitBonus(WeaponTraitBonus(1), 1)
	ms := func(v int) time.Duration { return time.Duration(v) * time.Millisecond }

	tests := []struct {
		name     string
		attacker Fighter
		trait    float64
		pos      Position
		draws    []rnd.Draw
		hits     []Hit
		at       []time.Duration
	}{
		{
			// 76 × 6.48 / 39.6 = 12.4364, × 0.99 = 12.312, × 1.1 × 1.1 =
			// 14.898. Without the trait it would be 15.
			name: "sword, hit, both random factors +10 %", attacker: sword, trait: swordTrait, pos: PosFront,
			draws: []rnd.Draw{rnd.Get(1000, 800), rnd.Get(1000, 88), rnd.Between(-10, 10, 10), rnd.Between(-10, 10, 10)},
			hits:  []Hit{{Damage: 14}}, at: []time.Duration{ms(599)},
		},
		{
			// 2 × 12.312 = 24.624, × 0.9 × 0.9 = 19.946. Without the trait
			// it would be 20.
			name: "sword, critical (88 > 87), both factors −10 %", attacker: sword, trait: swordTrait, pos: PosFront,
			draws: []rnd.Draw{rnd.Get(1000, 0), rnd.Get(1000, 87), rnd.Between(-10, 10, -10), rnd.Between(-10, 10, -10)},
			hits:  []Hit{{Crit: true, Damage: 19}}, at: []time.Duration{ms(599)},
		},
		{
			name: "sword, miss (800 < 801)", attacker: sword, trait: swordTrait, pos: PosFront,
			draws: []rnd.Draw{rnd.Get(1000, 801)},
			hits:  []Hit{{Miss: true}}, at: []time.Duration{ms(599)},
		},
		{
			// Behind: chance 880, damage × 1.2: 12.312 × 1.2 = 14.774.
			name: "sword from behind: 880 hits, × 1.2", attacker: sword, trait: swordTrait, pos: PosBack,
			draws: []rnd.Draw{rnd.Get(1000, 880), rnd.Get(1000, 999), rnd.Between(-10, 10, 0), rnd.Between(-10, 10, 0)},
			hits:  []Hit{{Damage: 14}}, at: []time.Duration{ms(599)},
		},
		{
			// Both miss rolls first, then crit per hit; no random draws (r =
			// 0). 76 × 4.32 / 39.6 = 8.29 → 8 → 4; critical 16.58 → 16 → 8.
			name: "unarmed: two half hits, the second critical", attacker: unarmed, trait: fistTrait, pos: PosFront,
			draws: []rnd.Draw{rnd.Get(1000, 0), rnd.Get(1000, 800), rnd.Get(1000, 44), rnd.Get(1000, 43)},
			hits:  []Hit{{Damage: 4}, {Crit: true, Damage: 8}}, at: []time.Duration{ms(378), ms(757)},
		},
		{
			name: "unarmed: both miss", attacker: unarmed, trait: fistTrait, pos: PosFront,
			draws: []rnd.Draw{rnd.Get(1000, 801), rnd.Get(1000, 999)},
			hits:  []Hit{{Miss: true}, {Miss: true}}, at: []time.Duration{ms(378), ms(757)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := rnd.NewScripted(t, tt.draws...)
			s := strikeOf(tt.attacker, keltir, tt.pos)
			s.SetTrait(tt.trait)
			res := c.AutoAttack(src, s)
			src.Done()
			if res.Timing.NumHits != len(tt.hits) {
				t.Fatalf("%d hits, want %d", res.Timing.NumHits, len(tt.hits))
			}
			for i, want := range tt.hits {
				if res.Hits[i] != want {
					t.Errorf("hit %d = %+v, want %+v", i, res.Hits[i], want)
				}
				if res.Timing.Hits[i] != tt.at[i] {
					t.Errorf("hit %d lands at %v, want %v", i, res.Timing.Hits[i], tt.at[i])
				}
			}
		})
	}

	// The Keltir's hate from a 14 damage hit: 1400 / 8 = 175.
	if got := DamageHate(14, keltirNpc.Level(), HateRatio(0)); got != 175 {
		t.Errorf("Keltir hate for 14 damage = %d, want 175", got)
	}
	// Regeneration per 3 s: hpRegen 2, mpRegen max(1, 0.9).
	if got := c.NpcRegen(keltirNpc, false); got != (Regen{HP: 2, MP: 1}) {
		t.Errorf("Keltir regen = %+v, want HP 2, MP 1", got)
	}
}

// TestGoldenOrcArcherVsFighter: an Orc Archer shoots a level 1 Human
// Fighter in starting gear.
//
//   - Archer: P.Atk 16 × 1.2 × 0.97 = 18.624; accuracy sqrt(30) × 6 + 8 =
//     40.86 → 41; crit 8 × 1.1 × 10 = 88; speed round(253 × 1.1) = 278. Its
//     rhand bow is the active weapon: random damage 0 (NPC weapons add no
//     stats), reuse 1500 from the bow item.
//   - Fighter: evasion 34, P.Def 75.6, no shield, no defence traits.
//   - Hit chance (80 + 2 × (41 − 34)) × 10 = 940. Damage 76 × 18.624 / 75.6
//     = 18.72 → 18, critical 37.45 → 37. No random draws.
//   - Timing: (int)(500000 / 278) = 1798, the hit at 1798; reuse
//     (int)(1500 × 333 / 278) = 1796; the attack ends at 899 + 898 = 1797;
//     the next shot at 3594.
func TestGoldenOrcArcherVsFighter(t *testing.T) {
	r := testRules(t, DefaultConfig())
	c := testCombat(t, DefaultCombatConfig())
	n, err := NewNpc(r, orcArcher())
	if err != nil {
		t.Fatal(err)
	}
	archer := Fighter{Stats: n, Ref: "archer", Attackable: true}
	archer.Arm(&Weapon{ItemID: 14, Type: "BOW", ReuseDelay: 1500}, "BOW", false)
	fighter := Fighter{Stats: newTestPlayer(t, r, humanFighter(), 1, fighterGear...), Ref: "fighter", Playable: true}
	if n.Accuracy() != 41 || n.Crit() != 88 || n.PAtkSpd() != 278 {
		t.Fatalf("archer accuracy %d, crit %d, speed %v; want 41, 88, 278", n.Accuracy(), n.Crit(), n.PAtkSpd())
	}

	ms := func(v int) time.Duration { return time.Duration(v) * time.Millisecond }
	wantTiming := SwingTiming{
		Interval: ms(1798), Reuse: ms(1796), Hits: [2]time.Duration{ms(1798)}, NumHits: 1,
		End: ms(1797), Ready: ms(3594), Gauge: ms(3594), BowLockTicks: 35,
	}
	tests := []struct {
		name  string
		draws []rnd.Draw
		hit   Hit
	}{
		{"hit (940 < 940 is no miss)", []rnd.Draw{rnd.Get(1000, 940), rnd.Get(1000, 88)}, Hit{Damage: 18}},
		{"critical (88 > 87)", []rnd.Draw{rnd.Get(1000, 0), rnd.Get(1000, 87)}, Hit{Crit: true, Damage: 37}},
		{"miss (940 < 941)", []rnd.Draw{rnd.Get(1000, 941)}, Hit{Miss: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := rnd.NewScripted(t, tt.draws...)
			s := strikeOf(archer, fighter, PosFront)
			s.SetTrait(AttackTraitBonus(WeaponTraitBonus(1)))
			res := c.AutoAttack(src, s)
			src.Done()
			if res.Timing != wantTiming {
				t.Errorf("timing\n got %+v\nwant %+v", res.Timing, wantTiming)
			}
			if res.Hits[0] != tt.hit {
				t.Errorf("hit = %+v, want %+v", res.Hits[0], tt.hit)
			}
		})
	}

	// The archer hates humans × 1.1 (SetHateRaceRatio 10): a 10 damage hit
	// gives 1000 / 15 = 66, × 1.1 = 72.
	if got := DamageHate(10, n.Level(), HateRatio(10)); got != 72 {
		t.Errorf("archer hate = %d, want 72", got)
	}
}

// TestNpcPassiveMaxHP checks F10's NPC side with a hand-made fixture of
// 4408 (HP ×2): a passive skill's modifiers, owned by the skill, change the
// NPC's maximum HP and the HP bar NpcInfo shows, and leave with the skill.
// The data (data/skills/passives.json) comes from datagen.
func TestNpcPassiveMaxHP(t *testing.T) {
	r := testRules(t, DefaultConfig())
	n, err := NewNpc(r, beardedKeltir())
	if err != nil {
		t.Fatal(err)
	}
	if n.MaxHP() != 62 || n.View().MaxHP != 62 {
		t.Fatalf("Keltir max HP = %d (view %d), want 39.745 × 1.58 = 62", n.MaxHP(), n.View().MaxHP)
	}
	owner := Owner{Kind: OwnerSkill, ID: 4408}
	if err := n.AddModifiers(NewModifier(MaxHP, OpMul, 2, owner)); err != nil {
		t.Fatal(err)
	}
	// 62.797 × 2 = 125.59: the CON formula runs first (order 1), the Mul at
	// order 20.
	if n.MaxHP() != 125 || n.View().MaxHP != 125 || n.MaxRecoverableHP() != 125 {
		t.Errorf("max HP with 4408 = %d (view %d, recoverable %d), want 125", n.MaxHP(), n.View().MaxHP, n.MaxRecoverableHP())
	}
	if n.RemoveOwner(owner) != 1 || n.MaxHP() != 62 {
		t.Errorf("removing the passive left max HP %d, want 62", n.MaxHP())
	}
}
