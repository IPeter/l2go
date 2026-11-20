package effects

import (
	"fmt"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// escape is Mobius Escape (Scroll of Escape, Return): it sends the target
// to the nearest place of the escapeType (Creature.Escape). Without an
// escapeType it does nothing, as Mobius does.
type escape struct {
	skills.Base
	where skills.TeleportWhere
	set   bool
}

func init() {
	register(data.EffectEscape, func(def *data.SkillEffect) (skills.Effect, error) {
		e := &escape{Base: skills.NewBase(def)}
		if v, ok := def.Params.Get("escapeType"); ok {
			w, ok := skills.ParseTeleportWhere(v)
			if !ok {
				return nil, fmt.Errorf("unknown escapeType %q", v)
			}
			e.where, e.set = w, true
		}
		return e, nil
	})
}

func (*escape) Type() skills.EffectType { return skills.TypeTeleport }

func (*escape) Instant() bool { return true }

func (e *escape) OnStart(info *skills.BuffInfo) {
	if e.set {
		info.Effected().Escape(e.where)
	}
}
