package skills_test

import (
	"slices"
	"testing"

	sp "github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/skills/skilltest"
)

// PARTY_OTHER: another living member only; the force buffs reach only
// fighters (426) or mystics (427); anyone else is an incorrect target.
func TestTargetPartyOther(t *testing.T) {
	f := newFixture(t)
	other := targetSkill(t, f, "PARTY_OTHER", false, nil)
	caster := skilltest.NewPlayer(f.env, 1)
	fighter := skilltest.NewPlayer(f.env, 2)
	mystic := skilltest.NewPlayer(f.env, 3)
	mystic.Mage = true
	dead := skilltest.NewPlayer(f.env, 4)
	dead.Dead = true
	stranger := skilltest.NewPlayer(f.env, 5)
	f.world.Parties = [][]skills.Creature{{caster, fighter, mystic, dead}}

	if got := ids(other.Targets(f.env, caster, false, fighter)); !slices.Equal(got, []int32{2}) {
		t.Errorf("PARTY_OTHER on a member = %v", got)
	}
	if got := other.Targets(f.env, caster, false, dead); got != nil {
		t.Errorf("PARTY_OTHER on a dead member = %v", ids(got))
	}
	for _, target := range []skills.Creature{caster, stranger, nil} {
		caster.Reset()
		if got := other.Targets(f.env, caster, false, target); got != nil {
			t.Errorf("PARTY_OTHER on %v = %v", target, ids(got))
		}
		if !slices.Equal(caster.MessageIDs(), []sp.SystemMessageID{skills.MsgIncorrectTarget}) {
			t.Errorf("messages = %v, want incorrect target", caster.MessageIDs())
		}
	}
	for _, tt := range []struct {
		id     int
		target *skilltest.Creature
		want   []int32
	}{{426, fighter, []int32{2}}, {426, mystic, nil}, {427, mystic, []int32{3}}, {427, fighter, nil}} {
		sk := skilltest.Skill(t, f.catalog(t, skilltest.Template(tt.id, map[string]string{"targetType": "PARTY_OTHER"}, nil)), tt.id)
		if got := ids(sk.Targets(f.env, caster, false, tt.target)); !slices.Equal(got, tt.want) {
			t.Errorf("skill %d on %d = %v, want %v", tt.id, tt.target.ObjectID(), got, tt.want)
		}
	}
}

// TARGET_PARTY: the target's party members and summons within the affect
// range of the target, or the target alone outside a party.
func TestTargetTargetParty(t *testing.T) {
	f := newFixture(t)
	sk := targetSkill(t, f, "TARGET_PARTY", false, map[string]string{"affectRange": "500"})
	caster := skilltest.NewPlayer(f.env, 1)
	target := at(skilltest.NewPlayer(f.env, 2), 1000, 0)
	near := at(skilltest.NewPlayer(f.env, 3), 1200, 0)
	far := at(skilltest.NewPlayer(f.env, 4), 3000, 0)
	pet := at(skilltest.NewPlayer(f.env, 5), 1100, 0)
	alone := skilltest.NewPlayer(f.env, 6)
	monster := skilltest.NewMonster(f.env, 7)
	f.world.Parties = [][]skills.Creature{{near, target, far}}
	f.world.Summons = map[int32]skills.Creature{3: pet}

	if got := ids(sk.Targets(f.env, caster, false, target)); !slices.Equal(got, []int32{3, 5, 2}) {
		t.Errorf("TARGET_PARTY = %v, want [3 5 2]", got)
	}
	if got := ids(sk.Targets(f.env, caster, false, alone)); !slices.Equal(got, []int32{6}) {
		t.Errorf("TARGET_PARTY on a player alone = %v", got)
	}
	if got := sk.Targets(f.env, caster, false, monster); got != nil {
		t.Errorf("TARGET_PARTY on a monster = %v", ids(got))
	}
	caster.Reset()
	if got := sk.Targets(f.env, caster, false, caster); got != nil || !slices.Contains(caster.MessageIDs(), skills.MsgIncorrectTarget) {
		t.Errorf("TARGET_PARTY on the caster = %v", ids(got))
	}
}
