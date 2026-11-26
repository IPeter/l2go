package skills_test

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/skills/skilltest"
)

// CLAN: the caster, its summon and the living clan members in the game
// within the affect range, each after its summon; the caster alone in the
// Olympiad or for the first target. An NPC gets the NPCs of its clan within
// the cast range.
func TestTargetClan(t *testing.T) {
	f := newFixture(t)
	sk := targetSkill(t, f, "CLAN", false, map[string]string{"affectRange": "500", "castRange": "300"})
	caster := skilltest.NewPlayer(f.env, 1)
	near := at(skilltest.NewPlayer(f.env, 2), 200, 0)
	far := at(skilltest.NewPlayer(f.env, 3), 2000, 0)
	dead := at(skilltest.NewPlayer(f.env, 4), 100, 0)
	dead.Dead = true
	stranger := at(skilltest.NewPlayer(f.env, 5), 50, 0)
	pet := at(skilltest.NewPlayer(f.env, 6), 250, 0)
	for _, c := range []*skilltest.Creature{caster, near, far, dead} {
		c.Clan = 7
	}
	f.world.Add(caster, near, far, dead, stranger)
	f.world.Summons = map[int32]skills.Creature{2: pet}

	if got := ids(sk.Targets(f.env, caster, false, nil)); !slices.Equal(got, []int32{1, 6, 2}) {
		t.Errorf("CLAN = %v, want [1 6 2]", got)
	}
	if got := ids(sk.Targets(f.env, caster, true, nil)); !slices.Equal(got, []int32{1}) {
		t.Errorf("CLAN first = %v", got)
	}
	caster.Olympiad = true
	if got := ids(sk.Targets(f.env, caster, false, nil)); !slices.Equal(got, []int32{1}) {
		t.Errorf("CLAN in the Olympiad = %v", got)
	}
	if got := ids(sk.Targets(f.env, stranger, false, nil)); !slices.Equal(got, []int32{5}) {
		t.Errorf("CLAN without a clan = %v", got)
	}

	// NPCs: the caster and its clan within the cast range.
	mob := skilltest.NewMonster(f.env, 10)
	friend := at(skilltest.NewMonster(f.env, 11), 100, 0)
	foe := at(skilltest.NewMonster(f.env, 12), 100, 0)
	distant := at(skilltest.NewMonster(f.env, 13), 1000, 0)
	f.world.Add(mob, friend, foe, distant)
	f.world.NpcClans = map[int32][]string{10: {"orc"}, 11: {"orc"}, 12: {"elf"}, 13: {"orc"}}
	if got := ids(sk.Targets(f.env, mob, false, nil)); !slices.Equal(got, []int32{10, 11}) {
		t.Errorf("CLAN of an NPC = %v, want [10 11]", got)
	}
	if got := ids(sk.Targets(f.env, foe, false, nil)); !slices.Equal(got, []int32{12}) {
		t.Errorf("CLAN of a lone NPC = %v", got)
	}
}

// CLAN_MEMBER: an NPC's first clan friend within the cast range, else
// itself; nothing for a player.
func TestTargetClanMember(t *testing.T) {
	f := newFixture(t)
	sk := targetSkill(t, f, "CLAN_MEMBER", false, map[string]string{"castRange": "300"})
	mob := skilltest.NewMonster(f.env, 10)
	friend := at(skilltest.NewMonster(f.env, 11), 100, 0)
	plain := skilltest.NewMonster(f.env, 12)
	f.world.Add(mob, friend, plain)
	f.world.NpcClans = map[int32][]string{10: {"ALL"}, 11: {"orc"}}
	if got := ids(sk.Targets(f.env, mob, false, nil)); !slices.Equal(got, []int32{11}) {
		t.Errorf("CLAN_MEMBER = %v, want [11]", got)
	}
	if got := ids(sk.Targets(f.env, plain, false, nil)); !slices.Equal(got, []int32{12}) {
		t.Errorf("CLAN_MEMBER without clans = %v", got)
	}
	if got := sk.Targets(f.env, skilltest.NewPlayer(f.env, 1), false, nil); got != nil {
		t.Errorf("CLAN_MEMBER of a player = %v", ids(got))
	}
}

// CORPSE_CLAN: the dead clan members within the affect range, not in a
// siege zone without a siege, up to the affect limit.
func TestTargetCorpseClan(t *testing.T) {
	f := newFixture(t)
	sk := targetSkill(t, f, "CORPSE_CLAN", false, map[string]string{"affectRange": "500", "affectLimit": "2-0"})
	caster := skilltest.NewPlayer(f.env, 1)
	alive := at(skilltest.NewPlayer(f.env, 2), 100, 0)
	var corpses []*skilltest.Creature
	for i := range 4 {
		c := at(skilltest.NewPlayer(f.env, int32(3+i)), 100+10*i, 0)
		c.Dead = true
		corpses = append(corpses, c)
	}
	corpses[0].Zones = map[skills.ZoneKind]bool{skills.ZoneSiege: true}
	for _, c := range append([]*skilltest.Creature{caster, alive}, corpses...) {
		c.Clan = 7
		f.world.Add(c)
	}
	if got := ids(sk.Targets(f.env, caster, false, nil)); !slices.Equal(got, []int32{4, 5}) {
		t.Errorf("CORPSE_CLAN = %v, want [4 5]", got)
	}
	if got := ids(sk.Targets(f.env, caster, true, nil)); !slices.Equal(got, []int32{4}) {
		t.Errorf("CORPSE_CLAN first = %v", got)
	}
}

// PARTY_CLAN: the caster and the living players within the affect range in
// its clan or its party.
func TestTargetPartyClan(t *testing.T) {
	f := newFixture(t)
	sk := targetSkill(t, f, "PARTY_CLAN", false, map[string]string{"affectRange": "500"})
	caster := skilltest.NewPlayer(f.env, 1)
	clanmate := at(skilltest.NewPlayer(f.env, 2), 100, 0)
	partner := at(skilltest.NewPlayer(f.env, 3), 150, 0)
	stranger := at(skilltest.NewPlayer(f.env, 4), 200, 0)
	farMate := at(skilltest.NewPlayer(f.env, 5), 900, 0)
	caster.Clan, clanmate.Clan, farMate.Clan = 7, 7, 7
	f.world.Add(caster, clanmate, partner, stranger, farMate)
	f.world.Parties = [][]skills.Creature{{caster, partner}}
	if got := ids(sk.Targets(f.env, caster, false, nil)); !slices.Equal(got, []int32{1, 2, 3}) {
		t.Errorf("PARTY_CLAN = %v, want [1 2 3]", got)
	}
	if got := ids(sk.Targets(f.env, stranger, false, nil)); !slices.Equal(got, []int32{4}) {
		t.Errorf("PARTY_CLAN alone = %v", got)
	}
}
