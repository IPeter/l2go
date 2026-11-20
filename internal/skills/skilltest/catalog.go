package skilltest

import (
	"io"
	"log/slog"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/stats"
)

// NewEnv returns an Env with the Mobius defaults over r, a fresh Sched and
// w (which may be nil), logging nowhere.
func NewEnv(tb testing.TB, r stats.Rand, w *World) (*skills.Env, *Sched) {
	tb.Helper()
	sched := NewSched()
	var world skills.World
	if w != nil {
		world = w
	}
	env, err := skills.NewEnv(r, sched, world, skills.DefaultConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		tb.Fatalf("NewEnv: %v", err)
	}
	return env, sched
}

// Template returns a one-level skill template with the values (operateType
// A1 when not given) and effects.
func Template(id int, values map[string]string, effects map[data.EffectScope][]data.EffectDef) data.SkillTemplate {
	v := map[string]string{"operateType": "A1"}
	for k, x := range values {
		v[k] = x
	}
	return data.SkillTemplate{ID: id, Name: "skill" + itoa(id), Levels: 1, Values: v, Effects: effects}
}

// General returns the effects map of a skill with GENERAL effects.
func General(defs ...data.EffectDef) map[data.EffectScope][]data.EffectDef {
	return map[data.EffectScope][]data.EffectDef{data.ScopeGeneral: defs}
}

// Catalog compiles the templates with reg.
func Catalog(tb testing.TB, reg *skills.Registry, tmpls ...data.SkillTemplate) *skills.Catalog {
	tb.Helper()
	slices.SortFunc(tmpls, func(a, b data.SkillTemplate) int { return a.ID - b.ID })
	d, err := data.NewSkills(data.SkillsData{Skills: tmpls})
	if err != nil {
		tb.Fatalf("NewSkills: %v", err)
	}
	c, err := skills.NewCatalog(d, reg)
	if err != nil {
		tb.Fatalf("NewCatalog: %v", err)
	}
	return c
}

// Skill returns level 1 of a skill of the catalog.
func Skill(tb testing.TB, c *skills.Catalog, id int) *skills.Skill {
	tb.Helper()
	sk, ok := c.Skill(id, 1)
	if !ok {
		tb.Fatalf("skill %d level 1 is missing", id)
	}
	return sk
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}
