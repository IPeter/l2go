package skills_test

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/data"
	sp "github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/skills/skilltest"
)

func ids(cs []skills.Creature) []int32 {
	out := make([]int32, len(cs))
	for i, c := range cs {
		out[i] = c.ObjectID()
	}
	return out
}

// targetSkill compiles a skill of a target type; negative skills have a
// negative effect point.
func targetSkill(t *testing.T, f *fixture, targetType string, negative bool, more map[string]string) *skills.Skill {
	t.Helper()
	v := map[string]string{"targetType": targetType}
	if negative {
		v["effectPoint"] = "-100"
	}
	for k, x := range more {
		v[k] = x
	}
	return oneSkill(t, f, v, nil)
}

func at(c *skilltest.Creature, x, y int) *skilltest.Creature {
	c.Loc = skills.Location{X: x, Y: y}
	return c
}

func TestTargetOneAndSelf(t *testing.T) {
	f := newFixture(t)
	one := targetSkill(t, f, "ONE", true, nil)
	p := skilltest.NewPlayer(f.env, 1)
	m := skilltest.NewMonster(f.env, 2)
	dead := skilltest.NewMonster(f.env, 3)
	dead.Dead = true
	for _, target := range []skills.Creature{nil, dead, p} {
		p.Reset()
		if got := one.Targets(f.env, p, false, target); got != nil {
			t.Errorf("target %v: got %v", target, ids(got))
		}
		if !slices.Equal(p.MessageIDs(), []sp.SystemMessageID{skills.MsgIncorrectTarget}) {
			t.Errorf("messages = %v, want incorrect target", p.MessageIDs())
		}
	}
	if got := ids(one.Targets(f.env, p, false, m)); !slices.Equal(got, []int32{2}) {
		t.Errorf("ONE = %v", got)
	}
	self := targetSkill(t, f, "SELF", false, nil)
	if got := ids(self.Targets(f.env, p, false, m)); !slices.Equal(got, []int32{1}) {
		t.Errorf("SELF = %v", got)
	}
	p.Tgt = m
	if got := one.FirstTarget(f.env, p); got == nil || got.ObjectID() != 2 {
		t.Errorf("FirstTarget = %v", got)
	}
}

func TestTargetTypeNotHandled(t *testing.T) {
	f := newFixture(t)
	sk := targetSkill(t, f, "HOLY", false, nil)
	p := skilltest.NewPlayer(f.env, 1)
	if got := sk.Targets(f.env, p, false, p); got != nil {
		t.Fatalf("HOLY = %v, want none until M11", ids(got))
	}
	if !slices.Equal(p.Texts, []string{skills.TextTargetTypeNotHandled}) {
		t.Fatalf("texts = %v", p.Texts)
	}
	if skills.TargetTypePorted(data.TargetHoly) || !skills.TargetTypePorted(data.TargetAura) {
		t.Fatalf("TargetTypePorted is wrong")
	}
}

func TestTargetAura(t *testing.T) {
	f := newFixture(t)
	aura := targetSkill(t, f, "AURA", true, map[string]string{"affectRange": "200"})
	caster := at(skilltest.NewPlayer(f.env, 1), 0, 0)
	near := at(skilltest.NewMonster(f.env, 2), 50, 0)
	far := at(skilltest.NewMonster(f.env, 3), 500, 0)
	dead := at(skilltest.NewMonster(f.env, 4), 60, 0)
	dead.Dead = true
	folk := at(skilltest.New(f.env, 5, data.InstanceFolk), 70, 0)
	mate := at(skilltest.NewPlayer(f.env, 6), 80, 0)
	stranger := at(skilltest.NewPlayer(f.env, 7), 90, 0)
	peaceful := at(skilltest.NewPlayer(f.env, 8), 100, 0)
	peaceful.Zones[skills.ZonePeace] = true
	hidden := at(skilltest.NewMonster(f.env, 9), 110, 0)
	clan := at(skilltest.NewPlayer(f.env, 10), 120, 0)
	caster.Clan, clan.Clan = 5, 5
	f.world.Add(caster, near, far, dead, folk, mate, stranger, peaceful, hidden, clan)
	f.world.Parties = [][]skills.Creature{{caster, mate}}
	f.world.Hidden = map[[2]int32]bool{{1, 9}: true}

	if got := ids(aura.Targets(f.env, caster, false, nil)); !slices.Equal(got, []int32{2, 7}) {
		t.Fatalf("AURA = %v, want [2 7]", got)
	}
	if got := ids(aura.Targets(f.env, caster, true, nil)); !slices.Equal(got, []int32{2}) {
		t.Fatalf("AURA, first only = %v", got)
	}
	// A positive aura skips monsters for a playable caster.
	good := targetSkill(t, f, "AURA", false, map[string]string{"affectRange": "200"})
	if got := ids(good.Targets(f.env, caster, false, nil)); !slices.Equal(got, []int32{7, 8}) {
		t.Fatalf("positive AURA = %v, want [7 8]", got)
	}
	// Arenas ignore clans.
	caster.Zones[skills.ZonePvP] = true
	if got := ids(aura.Targets(f.env, caster, false, nil)); !slices.Equal(got, []int32{2, 7, 10}) {
		t.Fatalf("AURA in an arena = %v, want [2 7 10]", got)
	}
}

func TestTargetArea(t *testing.T) {
	f := newFixture(t, rnd.Get(2, 1))
	area := targetSkill(t, f, "AREA", true, map[string]string{"castRange": "600", "affectRange": "100", "affectLimit": "1-2"})
	caster := at(skilltest.NewPlayer(f.env, 1), 0, 0)
	target := at(skilltest.NewMonster(f.env, 2), 300, 0)
	a := at(skilltest.NewMonster(f.env, 3), 350, 0)
	b := at(skilltest.NewMonster(f.env, 4), 320, 0)
	out := at(skilltest.NewMonster(f.env, 5), 500, 0)
	f.world.Add(caster, target, a, b, out)
	// The limit is 1 + Get(2) = 2, the target included.
	if got := ids(area.Targets(f.env, caster, false, target)); !slices.Equal(got, []int32{2, 3}) {
		t.Fatalf("AREA = %v, want [2 3]", got)
	}
	if got := ids(area.Targets(f.env, caster, true, target)); !slices.Equal(got, []int32{2}) {
		t.Fatalf("AREA, first only = %v", got)
	}
	if got := area.Targets(f.env, caster, false, caster); got != nil || !slices.Contains(caster.MessageIDs(), skills.MsgIncorrectTarget) {
		t.Fatalf("AREA on the caster = %v", ids(got))
	}
}

func TestTargetFrontAndBehind(t *testing.T) {
	f := newFixture(t)
	front := targetSkill(t, f, "FRONT_AURA", true, map[string]string{"affectRange": "200"})
	behind := targetSkill(t, f, "BEHIND_AURA", true, map[string]string{"affectRange": "200"})
	frontArea := targetSkill(t, f, "FRONT_AREA", true, map[string]string{"castRange": "100", "affectRange": "200"})
	caster := at(skilltest.NewPlayer(f.env, 1), 0, 0) // heading 0: facing +x
	ahead := at(skilltest.NewMonster(f.env, 2), 50, 0)
	back := at(skilltest.NewMonster(f.env, 3), -50, 0)
	f.world.Add(caster, ahead, back)
	if got := ids(front.Targets(f.env, caster, false, nil)); !slices.Equal(got, []int32{2}) {
		t.Errorf("FRONT_AURA = %v", got)
	}
	if got := ids(behind.Targets(f.env, caster, false, nil)); !slices.Equal(got, []int32{3}) {
		t.Errorf("BEHIND_AURA = %v", got)
	}
	// FRONT_AREA turns the caster to the target first.
	caster.Head = 32768
	if got := ids(frontArea.Targets(f.env, caster, false, ahead)); !slices.Equal(got, []int32{2}) {
		t.Errorf("FRONT_AREA = %v", got)
	}
	if caster.Head != 0 {
		t.Errorf("caster heading %d, want 0 (towards the target)", caster.Head)
	}
}

func TestTargetParty(t *testing.T) {
	f := newFixture(t)
	party := targetSkill(t, f, "PARTY", false, map[string]string{"affectRange": "1000"})
	notMe := targetSkill(t, f, "PARTY_NOTME", false, nil)
	member := targetSkill(t, f, "PARTY_MEMBER", false, nil)
	caster := skilltest.NewPlayer(f.env, 1)
	pet := at(skilltest.New(f.env, 2, data.InstanceServitor), 10, 0)
	near := at(skilltest.NewPlayer(f.env, 3), 100, 0)
	far := at(skilltest.NewPlayer(f.env, 4), 2000, 0)
	dead := at(skilltest.NewPlayer(f.env, 5), 100, 0)
	dead.Dead = true
	nearPet := at(skilltest.New(f.env, 6, data.InstanceServitor), 110, 0)
	stranger := skilltest.NewPlayer(f.env, 7)
	f.world.Parties = [][]skills.Creature{{caster, near, far, dead}}
	f.world.Summons = map[int32]skills.Creature{1: pet, 3: nearPet}

	if got := ids(party.Targets(f.env, caster, false, nil)); !slices.Equal(got, []int32{1, 2, 3, 6}) {
		t.Errorf("PARTY = %v, want [1 2 3 6]", got)
	}
	if got := ids(notMe.Targets(f.env, caster, false, nil)); !slices.Equal(got, []int32{3, 6}) {
		t.Errorf("PARTY_NOTME = %v, want [3 6]", got)
	}
	for _, tt := range []struct {
		target skills.Creature
		want   []int32
	}{{caster, []int32{1}}, {far, []int32{4}}, {pet, []int32{2}}, {dead, nil}, {stranger, nil}} {
		if got := ids(member.Targets(f.env, caster, false, tt.target)); !slices.Equal(got, tt.want) {
			t.Errorf("PARTY_MEMBER on %d = %v, want %v", tt.target.ObjectID(), got, tt.want)
		}
	}
	if !slices.Contains(caster.MessageIDs(), skills.MsgS1CannotBeUsedUnsuitable) {
		t.Errorf("no unsuitable message for a stranger")
	}
	// Without a party: the caster and its summon.
	f.world.Parties = nil
	if got := ids(party.Targets(f.env, caster, false, nil)); !slices.Equal(got, []int32{1, 2}) {
		t.Errorf("PARTY alone = %v", got)
	}
}

func TestTargetCorpses(t *testing.T) {
	f := newFixture(t)
	pcBody := targetSkill(t, f, "PC_BODY", false, nil)
	corpseMob := targetSkill(t, f, "CORPSE_MOB", false, nil)
	auraCorpse := targetSkill(t, f, "AURA_CORPSE_MOB", false, map[string]string{"affectRange": "200"})
	areaCorpse := targetSkill(t, f, "AREA_CORPSE_MOB", true, map[string]string{"affectRange": "200"})
	caster := skilltest.NewPlayer(f.env, 1)
	deadPlayer := skilltest.NewPlayer(f.env, 2)
	deadPlayer.Dead = true
	corpse := at(skilltest.NewMonster(f.env, 3), 50, 0)
	corpse.Dead = true
	alive := at(skilltest.NewMonster(f.env, 4), 60, 0)
	f.world.Add(caster, deadPlayer, corpse, alive)

	if got := ids(pcBody.Targets(f.env, caster, false, deadPlayer)); !slices.Equal(got, []int32{2}) {
		t.Errorf("PC_BODY = %v", got)
	}
	if got := pcBody.Targets(f.env, caster, false, alive); got != nil {
		t.Errorf("PC_BODY on a monster = %v", ids(got))
	}
	if got := ids(corpseMob.Targets(f.env, caster, false, corpse)); !slices.Equal(got, []int32{3}) {
		t.Errorf("CORPSE_MOB = %v", got)
	}
	if got := corpseMob.Targets(f.env, caster, false, alive); got != nil {
		t.Errorf("CORPSE_MOB on the living = %v", ids(got))
	}
	if got := ids(auraCorpse.Targets(f.env, caster, false, nil)); !slices.Equal(got, []int32{3}) {
		t.Errorf("AURA_CORPSE_MOB = %v", got)
	}
	if got := ids(areaCorpse.Targets(f.env, caster, false, corpse)); !slices.Equal(got, []int32{3, 4}) {
		t.Errorf("AREA_CORPSE_MOB = %v", got)
	}
}
