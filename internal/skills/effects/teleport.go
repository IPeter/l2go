package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// teleport is Mobius Teleport: the target is teleported to x, y, z, with a
// random offset.
type teleport struct {
	skills.Base
	loc skills.Location
}

func init() {
	register(data.EffectTeleport, func(def *data.SkillEffect) (skills.Effect, error) {
		var loc skills.Location
		for _, p := range []struct {
			name string
			to   *int
		}{{"x", &loc.X}, {"y", &loc.Y}, {"z", &loc.Z}} {
			v, err := def.Params.Int(p.name, 0)
			if err != nil {
				return nil, err
			}
			*p.to = v
		}
		return &teleport{Base: skills.NewBase(def), loc: loc}, nil
	})
}

func (*teleport) Type() skills.EffectType { return skills.TypeTeleport }

func (*teleport) Instant() bool { return true }

func (e *teleport) OnStart(info *skills.BuffInfo) {
	info.Effected().TeleToLocation(e.loc, true)
}
