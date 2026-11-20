package game

import (
	"errors"
	"fmt"
	"iter"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/skills/effects"
	"github.com/IPeter/l2go/internal/world"
)

// The skills of creatures (Plan 10 Phase 7.2): the skill engine of
// internal/skills on the world loop.
//
//   - New builds one skills.Env for the server (newSkillEnv): the Catalog of
//     Options.Skills with the effects of internal/skills/effects, the game's
//     random source, the loop's timers (loopScheduler), the world queries
//     (skillWorld) and the settings of the config (skillConfig).
//   - Every player and NPC carries a skills.Creature adapter with its effect
//     list and its known skills (skill_creature.go, skill_known.go):
//     Server.SkillCreature, Player.SkillCreature and Npc.SkillCreature return
//     it, and its Effects method the skills.EffectList.
//   - Known skills follow Mobius addSkill and removeSkill; passives become
//     stat modifiers owned by the skill ID (K8); NPCs get their template's
//     skills when they are made (skill_known.go).
//   - SkillList (coalesced, 300 ms), RequestSkillList, SkillCoolTime and the
//     reuse stamps are in skill_list.go; the icons, the short buff and
//     ExRegenMax in skill_effects.go.
//   - Loading, saving and the restore at EnterWorld (K12, K13) are in
//     skill_persist.go, with the death, teleport and leave hooks.
//
// Casting (Phase 7.3) plugs into the adapters with SetSkillCaster.

func init() { registerFeature("skills", setupSkills) }

// skillsFeature is the skill state on the Server (Server.skills). Loop-only,
// except env.Skills (the Catalog), which is read-only.
type skillsFeature struct {
	env     *skills.Env
	catalog *skills.Catalog // nil without skill data
	data    *data.Skills    // nil without skill data
	caster  SkillCaster
}

// newSkillEnv builds the skills.Env of the server, with the Catalog of d
// when d is not nil. It runs in New, before the features: the NPC feature
// spawns NPCs, which take their passive skills from the Catalog.
func (s *Server) newSkillEnv(d *data.Skills) error {
	env, err := skills.NewEnv(s.rnd, loopScheduler{s}, skillWorld{s}, skillConfig(s.gs), s.log)
	if err != nil {
		return fmt.Errorf("game: skills: %w", err)
	}
	s.skills.env, s.skills.data = env, d
	if d == nil {
		return nil
	}
	catalog, err := skills.NewCatalog(d, effects.NewRegistry())
	if err != nil {
		return fmt.Errorf("game: skills: %w", err)
	}
	catalog.Report().Log(s.log)
	s.skills.catalog = catalog
	env.Skills = catalog
	return nil
}

// skillConfig maps the config of a game server to the skill settings: the
// Mobius defaults, with the combat keys whose names match.
func skillConfig(gs config.GameServer) skills.Config {
	cfg := skills.DefaultConfig()
	c := gs.Combat
	if c == (config.Combat{}) {
		c = config.DefaultCombat()
	}
	cfg.ShieldBlocks = c.ShieldBlocks
	cfg.PerfectShieldBlockRate = c.PerfectShieldBlockRate
	cfg.CancelCastByHit = c.CancelByHit == config.CancelByHitCast || c.CancelByHit == config.CancelByHitAll
	cfg.CancelBowByHit = c.CancelByHit == config.CancelByHitBow || c.CancelByHit == config.CancelByHitAll
	return cfg
}

// setupSkills registers the skill handlers and hooks.
func setupSkills(s *Server) error {
	s.SetTraitBonus(func(a, t Creature) (float64, bool) {
		sa, st := s.SkillCreature(a), s.SkillCreature(t)
		if sa == nil || st == nil {
			return 0, false
		}
		return skills.AttackTraitBonus(sa, st), true
	})
	s.OnPlayerEnter(s.skillsEnter)
	s.OnPlayerLeave(s.skillsLeave)
	s.OnTeleport(func(p *Player, _ world.Position) { s.skillsTeleport(p) })
	s.OnDie(s.skillsDie)
	s.OnHit(s.skillsHit)
	s.OnAttacked(s.skillsAttacked)
	return errors.Join(
		s.Register(clientpackets.OpRequestSkillList, s.handleRequestSkillList),
	)
}

// SkillEnv returns the skill environment the server's creatures share: the
// Catalog (Env.Skills), the random source, the timers and the settings.
// Pass it to every skills function that takes an Env. Loop-only.
func (s *Server) SkillEnv() *skills.Env { return s.skills.env }

// SkillCatalog returns the compiled skills, nil when the server has no skill
// data (Options.Skills). Safe on any goroutine.
func (s *Server) SkillCatalog() *skills.Catalog { return s.skills.catalog }

// SkillData returns the skill data (trees, armor sets), nil when the server
// has none. Safe on any goroutine.
func (s *Server) SkillData() *data.Skills { return s.skills.data }

// Skill returns one level of a skill, false when it does not exist or the
// server has no skill data.
func (s *Server) Skill(id, level int) (*skills.Skill, bool) {
	if s.skills.catalog == nil {
		return nil, false
	}
	return s.skills.catalog.Skill(id, level)
}

// SkillCaster is what casting (Plan 10 Phase 7.3) gives the creature
// adapters: the cast state the skill engine asks about and the ways it
// interrupts a cast. Without one, nothing ever casts.
type SkillCaster interface {
	// IsCastingNow and CanAbortCast are Mobius isCastingNow and
	// canAbortCast.
	IsCastingNow(c skills.Creature) bool
	CanAbortCast(c skills.Creature) bool
	// AbortCast stops c's cast (Mobius abortCast); BreakCast interrupts it
	// as damage does (Mobius breakCast).
	AbortCast(c skills.Creature)
	BreakCast(c skills.Creature)
	// TriggerCast casts a triggered skill at once (Mobius makeTriggerCast).
	TriggerCast(c skills.Creature, sk *skills.Skill, target skills.Creature, ignoreTargetType bool)
}

// SetSkillCaster plugs casting into the creature adapters. Call it from a
// feature's setup.
func (s *Server) SetSkillCaster(c SkillCaster) { s.skills.caster = c }

// loopScheduler is the world loop as the skill engine's skills.Scheduler.
type loopScheduler struct{ s *Server }

// After runs fn on the loop once d has passed.
func (l loopScheduler) After(d time.Duration, fn func()) skills.TimerID {
	return skills.TimerID(l.s.loop.After(d, fn))
}

// Every runs fn on the loop every d, the first time once d has passed.
func (l loopScheduler) Every(d time.Duration, fn func()) skills.TimerID {
	return skills.TimerID(l.s.loop.Every(d, fn))
}

// Cancel stops a timer.
func (l loopScheduler) Cancel(id skills.TimerID) bool { return l.s.loop.Cancel(world.TimerID(id)) }

// GameTicks returns the 100 ms ticks of the loop clock.
func (l loopScheduler) GameTicks() int64 { return gameTick(l.s.Now()) }

// skillWorld answers the world queries of the skill engine (skills.World).
// Parties, summons and geodata line of sight come with their milestones.
type skillWorld struct{ s *Server }

// VisibleInRange returns the creatures within rng of c (3D), c excluded.
func (w skillWorld) VisibleInRange(c skills.Creature, rng int) []skills.Creature {
	o := w.s.gameCreature(c)
	if o == nil {
		return nil
	}
	return w.s.skillCreatures(w.s.world.VisibleInRange(o, int32(min(rng, 1<<30)))) //nolint:gosec // clamped
}

// Visible returns every creature around c, c excluded.
func (w skillWorld) Visible(c skills.Creature) []skills.Creature {
	o := w.s.gameCreature(c)
	if o == nil {
		return nil
	}
	return w.s.skillCreatures(w.s.world.Visible(o))
}

// CanSee is the geodata line of sight between two creatures.
func (w skillWorld) CanSee(from, to skills.Creature) bool {
	a, b := w.s.gameCreature(from), w.s.gameCreature(to)
	if a == nil || b == nil {
		return false
	}
	return w.s.canSee(a, b)
}

// Party returns the members of c's party in party order, nil when c is not
// a player in a party (party.go).
func (w skillWorld) Party(c skills.Creature) []skills.Creature {
	p, ok := w.s.gameCreature(c).(*Player)
	if !ok || p.party.party == nil {
		return nil
	}
	out := make([]skills.Creature, 0, len(p.party.party.members))
	for _, m := range p.party.party.members {
		out = append(out, m.skill)
	}
	return out
}

// SameParty reports whether a and b are players of the same party.
func (w skillWorld) SameParty(a, b skills.Creature) bool {
	pa, okA := w.s.gameCreature(a).(*Player)
	pb, okB := w.s.gameCreature(b).(*Player)
	return okA && okB && w.s.SameParty(pa, pb)
}

// Summon returns nil: summons come with Milestone 11.
func (skillWorld) Summon(skills.Creature) skills.Creature { return nil }

// OldCorpse reports a corpse that decays within the time corpse skills
// leave (Mobius Attackable.isOldCorpse with
// CorpseConsumeSkillAllowedTimeBeforeDecay, 2 s), telling caster why.
func (w skillWorld) OldCorpse(corpse, caster skills.Creature) bool {
	const allowedBeforeDecay = 2 * time.Second
	n, ok := w.s.gameCreature(corpse).(*Npc)
	if !ok {
		return false
	}
	at, ok := w.s.NpcDecaysAt(n)
	if !ok || at.Sub(w.s.Now()) >= allowedBeforeDecay {
		return false
	}
	if caster != nil {
		skills.Send(caster, sysMsgCorpseTooOld)
	}
	return true
}

// sysMsgCorpseTooOld is "Corpse too old. The skill cannot be used." (Mobius
// CORPSE_TOO_OLD_SKILL_NOT_USED).
const sysMsgCorpseTooOld = 1186

// IsNight reports game-time night.
func (w skillWorld) IsNight() bool { return w.s.world.GameTime().IsNight() }

// ValidLocation returns where a creature going from from to to can stand.
func (w skillWorld) ValidLocation(from, to skills.Location) skills.Location {
	p := w.s.geo.ValidLocation(geoPointOf(from), geoPointOf(to))
	return skills.Location{X: int(p.X), Y: int(p.Y), Z: int(p.Z)}
}

// skillCreatures returns the adapters of the creatures among objs.
func (s *Server) skillCreatures(objs iter.Seq[world.Object]) []skills.Creature {
	var out []skills.Creature
	for o := range objs {
		if c := s.skillCreatureOf(o); c != nil {
			out = append(out, c)
		}
	}
	return out
}

func geoPointOf(l skills.Location) geo.Point {
	return geo.Point{X: int32(l.X), Y: int32(l.Y), Z: int32(l.Z)} //nolint:gosec // world coordinates
}
