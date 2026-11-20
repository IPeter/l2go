package npcai

import (
	"encoding/json"
	"iter"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/skills/effects"
)

const repoData = "../../data"

// fake is a creature, and an NPC, with plain fields.
type fake struct {
	name                      string
	loc                       Point
	hp                        float64
	maxHP                     int32
	mp                        float64
	dead, moving, immobilized bool
	coll                      int
	class                     Class
	prof                      *Profile
	affected                  []int
	buff, effects             bool
	casting, muted, pmuted    bool
	rooted                    bool
	disabled                  []int
	leader                    *fake
	minions                   bool
	rank                      Rank
	guard, aggressive         bool
	aggroRange, reach         int
	attackBy                  []*fake
	aggro                     []hateEntry
	most                      *fake
}

type hateEntry struct {
	c *fake
	h int64
}

var _ Npc = (*fake)(nil)

func (f *fake) Loc() Point { return f.loc }

func (f *fake) CurrentHP() float64 { return f.hp }

func (f *fake) MaxHP() int32 { return f.maxHP }

func (f *fake) IsDead() bool { return f.dead }

func (f *fake) IsMoving() bool { return f.moving }

func (f *fake) TemplateCollision() int { return f.coll }

func (f *fake) AIClass() Class { return f.class }

func (f *fake) AIProfile() *Profile { return f.prof }

func (f *fake) AffectedBySkill(id int) bool { return slices.Contains(f.affected, id) }

func (f *fake) HasBuffEffect() bool { return f.buff }

func (f *fake) HasEffects() bool { return f.effects }

func (f *fake) IsImmobilized() bool { return f.immobilized }

func (f *fake) CurrentMP() float64 { return f.mp }

func (f *fake) IsCastingNow() bool { return f.casting }

func (f *fake) IsMuted() bool { return f.muted }

func (f *fake) IsPhysicalMuted() bool { return f.pmuted }

func (f *fake) IsSkillDisabled(sk *skills.Skill) bool { return slices.Contains(f.disabled, sk.ID) }

func (f *fake) IsMovementDisabled() bool { return f.rooted }

func (f *fake) HasMinions() bool { return f.minions }

func (f *fake) Rank() Rank { return f.rank }

func (f *fake) IsGuard() bool { return f.guard }

func (f *fake) IsAggressive() bool { return f.aggressive }

func (f *fake) AggroRange() int { return f.aggroRange }

func (f *fake) PhysicalAttackRange() int { return f.reach }

func (f *fake) Leader() Creature {
	if f.leader == nil {
		return nil
	}
	return f.leader
}

func (f *fake) MostHated() Creature {
	if f.most == nil {
		return nil
	}
	return f.most
}

func (f *fake) Hating(c Creature) int64 {
	for _, e := range f.aggro {
		if Creature(e.c) == c {
			return e.h
		}
	}
	return 0
}

func (f *fake) AttackBy() iter.Seq[Creature] {
	return func(yield func(Creature) bool) {
		for _, c := range f.attackBy {
			if !yield(c) {
				return
			}
		}
	}
}

func (f *fake) Aggro() iter.Seq2[Creature, int64] {
	return func(yield func(Creature, int64) bool) {
		if f.dead {
			return
		}
		for _, e := range f.aggro {
			if !yield(e.c, e.h) {
				return
			}
		}
	}
}

func (f *fake) String() string { return f.name }

// fakeWorld holds the creatures in their lookup order.
type fakeWorld struct {
	all      []*fake
	blind    [][2]*fake // from, to pairs without sight
	friendly []*fake    // creatures nobody may attack
}

var _ World = (*fakeWorld)(nil)

func (w *fakeWorld) CanSee(from, to Creature) bool {
	for _, b := range w.blind {
		if Creature(b[0]) == from && Creature(b[1]) == to {
			return false
		}
	}
	return true
}

func (w *fakeWorld) Visible(center Creature, of Of, radius int) iter.Seq[Creature] {
	return func(yield func(Creature) bool) {
		for _, c := range w.all {
			if Creature(c) == center || !of.Accepts(c.class) {
				continue
			}
			if radius >= 0 && dist3D(center, c) > float64(radius) {
				continue
			}
			if !yield(c) {
				return
			}
		}
	}
}

func (w *fakeWorld) AutoAttackable(target Creature, _ Npc) bool {
	if slices.ContainsFunc(w.friendly, func(f *fake) bool { return Creature(f) == target }) {
		return false
	}
	k := target.AIClass()
	return k == ClassPlayer || k == ClassSummon
}

func newThinker(t *testing.T, w *fakeWorld, draws ...rnd.Draw) *Thinker {
	t.Helper()
	if w == nil {
		w = &fakeWorld{}
	}
	return &Thinker{Rnd: rnd.NewScripted(t, draws...), World: w, Config: DefaultConfig()}
}

// player returns a player at x, y.
func player(name string, x, y int32) *fake {
	return &fake{name: name, loc: Point{X: x, Y: y}, hp: 1000, maxHP: 1000, coll: 9, class: ClassPlayer}
}

// monster returns a monster at x, y with the profile, at full HP and with
// plenty of MP.
func monster(name string, x, y int32, p *Profile) *fake {
	return &fake{name: name, loc: Point{X: x, Y: y}, hp: 1000, maxHP: 1000, mp: 1000, coll: 14, class: ClassAttackable, prof: p, reach: 40}
}

// realData is the repository's skills and NPC templates.
type realData struct {
	cat  *skills.Catalog
	npcs map[int]*data.NpcTemplate
}

var loadReal = sync.OnceValues(func() (*realData, error) {
	d, err := data.LoadSkills(repoData)
	if err != nil {
		return nil, err
	}
	cat, err := skills.NewCatalog(d, effects.NewRegistry())
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(filepath.Join(repoData, data.NpcsFile))
	if err != nil {
		return nil, err
	}
	var list []data.NpcTemplate
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, err
	}
	npcs := make(map[int]*data.NpcTemplate, len(list))
	for i := range list {
		npcs[list[i].ID] = &list[i]
	}
	return &realData{cat: cat, npcs: npcs}, nil
})

func repo(t *testing.T) *realData {
	t.Helper()
	d, err := loadReal()
	if err != nil {
		t.Fatalf("loading the data: %v", err)
	}
	return d
}

// profile builds the profile of a real template.
func (d *realData) profile(t *testing.T, id int) *Profile {
	t.Helper()
	tmpl, ok := d.npcs[id]
	if !ok {
		t.Fatalf("NPC %d is missing", id)
	}
	p, warnings := NewProfile(tmpl, d.cat)
	if len(warnings) > 0 {
		t.Fatalf("NPC %d: %v", id, warnings)
	}
	return p
}

// skill returns a real skill level.
func (d *realData) skill(t *testing.T, id, level int) *skills.Skill {
	t.Helper()
	sk, ok := d.cat.Skill(id, level)
	if !ok {
		t.Fatalf("skill %d level %d is missing", id, level)
	}
	return sk
}

// synthetic builds a profile from a template with the skills and
// parameters.
func (d *realData) synthetic(t *testing.T, tmpl data.NpcTemplate) *Profile {
	t.Helper()
	if tmpl.AI.Type == "" {
		tmpl.AI.Type = "FIGHTER"
	}
	if tmpl.AI.MinSkillChance == 0 && tmpl.AI.MaxSkillChance == 0 {
		tmpl.AI.MinSkillChance, tmpl.AI.MaxSkillChance = 7, 15
	}
	p, warnings := NewProfile(&tmpl, d.cat)
	if len(warnings) > 0 {
		t.Fatalf("synthetic NPC: %v", warnings)
	}
	return p
}

func ids(list []*skills.Skill) []int {
	out := make([]int, len(list))
	for i, sk := range list {
		out[i] = sk.ID
	}
	return out
}

// wantCast checks a cast decision.
func wantCast(t *testing.T, d Decision, skillID int, target, restore Creature, stop bool) {
	t.Helper()
	if d.Action != ActCast || d.Skill == nil || d.Skill.ID != skillID || d.Target != target || d.Restore != restore || d.StopMove != stop {
		t.Fatalf("decision %s, want cast of %d at %v (restore %v, stop %v)", describe(d), skillID, target, restore, stop)
	}
}

func describe(d Decision) string {
	s := d.Action.String()
	if d.Skill != nil {
		s += " skill " + d.Skill.String()
	}
	if d.Target != nil {
		s += " at " + d.Target.(*fake).name
	}
	if d.Restore != nil {
		s += " restore " + d.Restore.(*fake).name
	}
	if d.StopMove {
		s += " stop"
	}
	return s
}
