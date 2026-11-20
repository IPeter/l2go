package effects

import (
	"math"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// headingPerDegree is the client heading units per degree
// (LocationUtil.convertHeadingToDegree).
const headingPerDegree = 182.044444444

// teleportToTarget is Mobius TeleportToTarget (Shadow Step): the caster
// stops what it does and appears 25 behind the target, at a place the
// geodata allows; everyone is shown its new place.
type teleportToTarget struct{ skills.Base }

func init() {
	register(data.EffectTeleportToTarget, func(def *data.SkillEffect) (skills.Effect, error) {
		return &teleportToTarget{Base: skills.NewBase(def)}, nil
	})
}

func (*teleportToTarget) Type() skills.EffectType { return skills.TypeTeleportToTarget }

func (*teleportToTarget) Instant() bool { return true }

func (*teleportToTarget) CanStart(info *skills.BuffInfo) bool {
	env := info.Env()
	return info.Effected() != nil && (env.World == nil || env.World.CanSee(info.Effected(), info.Effector()))
}

func (*teleportToTarget) OnStart(info *skills.BuffInfo) {
	env, effector, effected := info.Env(), info.Effector(), info.Effected()
	if effected == nil {
		return
	}
	tl := effected.Location()
	ph := float64(effected.Heading()) / headingPerDegree
	ph += 180
	if ph > 360 {
		ph -= 360
	}
	ph = math.Pi * ph / 180
	loc := skills.Location{
		X: int(javaInt(float64(tl.X) + float64(25*math.Cos(ph)))),
		Y: int(javaInt(float64(tl.Y) + float64(25*math.Sin(ph)))),
		Z: tl.Z,
	}
	if env.World != nil {
		loc = env.World.ValidLocation(effector.Location(), loc)
	}
	effector.NotifyAI(skills.AIIdle, nil)
	effector.AbortAttack()
	effector.AbortCast()
	effector.SetXYZ(loc)
	effector.BroadcastValidateLocation()
}
