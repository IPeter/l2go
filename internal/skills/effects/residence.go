package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// The castle handlers (Plan 14 M11i): TakeCastle (Seal of Ruler) and
// HeadquarterCreate (Build Headquarters). The world's skills.ResidenceWorld
// does the work; without one they do nothing.

// takeCastle is Mobius TakeCastle.
type takeCastle struct{ skills.Base }

// headquarterCreate is Mobius HeadquarterCreate.
type headquarterCreate struct {
	skills.Base
	advanced bool
}

func init() {
	register(data.EffectTakeCastle, func(def *data.SkillEffect) (skills.Effect, error) {
		return &takeCastle{Base: skills.NewBase(def)}, nil
	})
	register(data.EffectHeadquarterCreate, func(def *data.SkillEffect) (skills.Effect, error) {
		adv, err := def.Params.Bool("isAdvanced", false)
		if err != nil {
			return nil, err
		}
		return &headquarterCreate{Base: skills.NewBase(def), advanced: adv}, nil
	})
}

func residenceWorld(info *skills.BuffInfo) skills.ResidenceWorld {
	rw, _ := info.Env().World.(skills.ResidenceWorld)
	return rw
}

func (*takeCastle) Instant() bool { return true }

// OnStart is TakeCastle.onStart: a player effector engraves the ruler.
func (*takeCastle) OnStart(info *skills.BuffInfo) {
	if !isPlayer(info.Effector()) {
		return
	}
	if rw := residenceWorld(info); rw != nil {
		rw.TakeCastle(info.Effector(), info.Effected())
	}
}

func (*headquarterCreate) Instant() bool { return true }

// OnStart is HeadquarterCreate.onStart.
func (e *headquarterCreate) OnStart(info *skills.BuffInfo) {
	if !isPlayer(info.Effector()) {
		return
	}
	if rw := residenceWorld(info); rw != nil {
		rw.CreateHeadquarters(info.Effector(), e.advanced)
	}
}
