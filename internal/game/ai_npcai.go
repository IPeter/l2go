package game

import (
	"fmt"
	"iter"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/npcai"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/world"
)

// The game side of the full monster AI (Plan 13, Phase 10.4): the pure
// decisions of internal/npcai over the game's creatures.
//
//   - ai_npcai.go: the AI profiles of the templates (built at load), the
//     npcai interfaces over players, NPCs and the world (thin adapters), and
//     carrying out a Decision. Leaders, minions and their assist are the
//     minion feature's (minion.go).
//   - ai_cast.go: the CAST intention (Mobius thinkCast) and the friendly
//     skill drop of Monster.doCast.
//   - ai_fear.go: fear (Mobius notifyActionAfraid with FEAR_TICKS).
//   - ai_see.go: the on_creature_see scan (Mobius CreatureSeeTaskManager).
//
// ai_think.go calls the npcai steps at Mobius's points of thinkActive and
// thinkAttack; ai.go raises on_faction_call with the clan help.
func init() { registerFeature("ai_skills", setupNpcAI) }

// creatureSeeRange is how far an NPC without an aggro range sees for
// on_creature_see (Mobius Creature._seenCreatureRange, AltPartyRange).
const creatureSeeRange = 1500

// creatureSeePeriod is the period of the on_creature_see scan (Mobius
// CreatureSeeTaskManager: every second).
const creatureSeePeriod = time.Second

// aiFeatureX is the state of the full monster AI on the Server (in
// aiFeature). Loop-only.
type aiFeatureX struct {
	// thinker makes the decisions with the server's random source.
	thinker npcai.Thinker
	// profiles are the AI profiles by template ID.
	profiles map[int]*npcai.Profile

	// doCast replaces the casts of decisions in tests and benchmarks.
	doCast func(n *Npc, sk *skills.Skill)

	// see lists the NPCs whose template a script watches with
	// on_creature_see (ai_see.go); seeBatch is the scan's copy.
	see, seeBatch []*Npc
	seeEvents     []creatureSeeEvent
}

// npcAIx is the per-NPC state of the full monster AI (in npcAI).
type npcAIx struct {
	// brain is what npcai remembers between thinks (the chaos counter).
	brain npcai.State

	// coreAIDisabled is Mobius isCoreAIDisabled: scripts drive the NPC.
	coreAIDisabled bool

	// castSkill and castTarget are the CAST intention (ai_cast.go).
	castSkill  *skills.Skill
	castTarget Creature

	// fearTimer is the 5 s fear task (Mobius _fearTask), 0 for none.
	fearTimer world.TimerID

	// seeing is set while the NPC is in the scan list; seen holds the
	// creatures it saw (Mobius _seenCreatures).
	seeing bool
	seen   map[world.Object]struct{}
}

// aiSkillClass is what the full AI keeps of a template (in aiClass).
type aiSkillClass struct {
	profile *npcai.Profile

	// rank is RaidBoss or GrandBoss for those types, else normal (raid
	// minions are told apart by Npc.IsRaidMinion).
	rank npcai.Rank
}

// setupNpcAI builds the AI profiles and registers the scans.
func setupNpcAI(s *Server) error {
	f := &s.ai.x
	n := s.settings.NPCs
	cfg := npcai.Config{RaidChaosTime: n.RaidChaosTime, GrandChaosTime: n.GrandChaosTime, MinionChaosTime: n.MinionChaosTime}
	f.thinker = npcai.Thinker{Rnd: s.rnd, World: aiWorld{s: s}, Config: cfg}
	s.buildNpcProfiles()
	s.loop.Every(creatureSeePeriod, s.creatureSeeScan)
	s.OnForget(s.creatureSeeForget)
	return nil
}

// noSkills is the skill source of a server without skill data.
type noSkills struct{}

func (noSkills) SkillOrMax(int, int) (*skills.Skill, bool) { return nil, false }

// skillSource returns where profiles find skills.
func (s *Server) skillSource() npcai.SkillSource {
	if c := s.SkillCatalog(); c != nil {
		return c
	}
	return noSkills{}
}

// buildNpcProfiles builds the AI profile of every NPC template once, as
// Mobius NpcData does at load, and logs what it could not read.
func (s *Server) buildNpcProfiles() {
	f := &s.ai.x
	f.profiles = make(map[int]*npcai.Profile)
	if s.worldData == nil {
		return
	}
	var warnings []string
	for i := range s.worldData.Npcs {
		t := &s.worldData.Npcs[i]
		p, w := npcai.NewProfile(t, s.skillSource())
		f.profiles[t.ID] = p
		warnings = append(warnings, w...)
	}
	if s.SkillCatalog() == nil || len(warnings) == 0 {
		return
	}
	for _, w := range warnings {
		s.log.Debug("game: NPC AI profile", "warning", w)
	}
	shown := warnings[:min(len(warnings), 5)]
	s.log.Warn("game: NPC AI profiles left out what they could not read",
		"warnings", len(warnings), "first", fmt.Sprint(shown))
}

// npcProfile returns the AI profile of t, built on first use for a template
// the world data does not have (tests, scripts' templates).
func (s *Server) npcProfile(t *data.NpcTemplate) *npcai.Profile {
	f := &s.ai.x
	if p := f.profiles[t.ID]; p != nil {
		return p
	}
	p, _ := npcai.NewProfile(t, s.skillSource())
	if f.profiles == nil {
		f.profiles = make(map[int]*npcai.Profile)
	}
	f.profiles[t.ID] = p
	return p
}

// aiSkillClassOf works out the full AI's part of a template.
func (s *Server) aiSkillClassOf(t *data.NpcTemplate) aiSkillClass {
	c := aiSkillClass{profile: s.npcProfile(t)}
	if s.worldData != nil {
		switch {
		case s.worldData.NpcTypeIs(t.Type, "GrandBoss"):
			c.rank = npcai.RankGrandBoss
		case s.worldData.NpcTypeIs(t.Type, "RaidBoss"):
			c.rank = npcai.RankRaidBoss
		}
	}
	return c
}

// npcThinker returns the decision maker.
func (s *Server) npcThinker() *npcai.Thinker { return &s.ai.x.thinker }

// SetNpcCoreAIDisabled is Mobius disableCoreAI: the NPC's own AI stops
// choosing targets and attacking; scripts drive it. Loop-only.
func (s *Server) SetNpcCoreAIDisabled(n *Npc, disabled bool) { n.ai.x.coreAIDisabled = disabled }

// NpcCoreAIDisabled is Mobius isCoreAIDisabled. Loop-only.
func (n *Npc) NpcCoreAIDisabled() bool { return n.ai.x.coreAIDisabled }

// ----- the adapters -----

// aiNpc is an NPC as npcai sees it (npcai.Npc). It is the same struct, so
// conversions cost nothing and the same NPC always compares equal.
type aiNpc Npc

// aiPlayer is a player as npcai sees it (npcai.Creature).
type aiPlayer Player

var (
	_ npcai.Npc      = (*aiNpc)(nil)
	_ npcai.Creature = (*aiPlayer)(nil)
	_ npcai.World    = aiWorld{}
)

// aiNpcOf returns n as npcai sees it.
func aiNpcOf(n *Npc) *aiNpc { return (*aiNpc)(n) }

// aiView returns a creature as npcai sees it, nil for nil.
func aiView(c Creature) npcai.Creature {
	switch v := c.(type) {
	case *Npc:
		if v != nil {
			return (*aiNpc)(v)
		}
	case *Player:
		if v != nil {
			return (*aiPlayer)(v)
		}
	}
	return nil
}

// aiObjectView returns a world object as npcai sees it, nil for anything
// but a player or an NPC.
func aiObjectView(o world.Object) npcai.Creature {
	switch v := o.(type) {
	case *Npc:
		return (*aiNpc)(v)
	case *Player:
		return (*aiPlayer)(v)
	}
	return nil
}

// gameOf returns the player or NPC behind an npcai creature.
func gameOf(c npcai.Creature) Creature {
	switch v := c.(type) {
	case *aiNpc:
		return (*Npc)(v)
	case *aiPlayer:
		return (*Player)(v)
	}
	return nil
}

func (a *aiNpc) n() *Npc { return (*Npc)(a) }

func (a *aiNpc) Loc() npcai.Point {
	pos := a.n().Position()
	return npcai.Point{X: pos.X, Y: pos.Y, Z: pos.Z}
}

func (a *aiNpc) CurrentHP() float64     { return a.CurHP }
func (a *aiNpc) MaxHP() int32           { return a.n().MaxHP() }
func (a *aiNpc) IsDead() bool           { return a.Dead }
func (a *aiNpc) IsMoving() bool         { return a.Move != nil }
func (a *aiNpc) TemplateCollision() int { return int(npcCollision(a.n())) }

func (a *aiNpc) AIClass() npcai.Class {
	if a.class != nil && a.class.attackable {
		return npcai.ClassAttackable
	}
	return npcai.ClassNpc
}

func (a *aiNpc) AIProfile() *npcai.Profile {
	if a.srv == nil {
		return nil
	}
	return a.srv.aiClassOf(a.n()).sk.profile
}

func (a *aiNpc) AffectedBySkill(id int) bool {
	return a.skill != nil && a.skill.el.IsAffectedBySkill(id)
}

func (a *aiNpc) HasBuffEffect() bool {
	return a.skill != nil && a.skill.el.FirstEffect(skills.TypeBuff) != nil
}

func (a *aiNpc) HasEffects() bool { return a.skill != nil && !a.skill.el.IsEmpty() }

func (a *aiNpc) IsImmobilized() bool { return a.skill != nil && a.skill.immobilized }

func (a *aiNpc) CurrentMP() float64 { return a.CurMP }

func (a *aiNpc) IsCastingNow() bool { return a.srv.CastingNow(a.n()) }

func (a *aiNpc) IsMuted() bool { return a.skill != nil && a.skill.el.IsAffected(skills.FlagMuted) }

func (a *aiNpc) IsPhysicalMuted() bool {
	return a.skill != nil && a.skill.el.IsAffected(skills.FlagPhysicalMuted)
}

func (a *aiNpc) IsSkillDisabled(sk *skills.Skill) bool { return a.srv.SkillDisabled(a.n(), sk) }

func (a *aiNpc) IsMovementDisabled() bool { return npcMovementDisabled(a.n()) }

func (a *aiNpc) Leader() npcai.Creature {
	if l := a.n().Leader(); l != nil {
		return (*aiNpc)(l)
	}
	return nil
}

func (a *aiNpc) HasMinions() bool { return a.n().HasMinions() }

func (a *aiNpc) Rank() npcai.Rank {
	n := a.n()
	if r := a.srv.aiClassOf(n).sk.rank; r != npcai.RankNormal {
		return r
	}
	if n.IsRaidMinion() {
		return npcai.RankRaidMinion
	}
	return npcai.RankNormal
}

func (a *aiNpc) IsGuard() bool      { return a.srv.aiClassOf(a.n()).guard }
func (a *aiNpc) IsAggressive() bool { return a.srv.aiClassOf(a.n()).aggressive }
func (a *aiNpc) AggroRange() int    { return a.srv.npcAggroRange(a.n()) }

func (a *aiNpc) PhysicalAttackRange() int { return a.Stats.PhysicalAttackRange() }

func (a *aiNpc) AttackBy() iter.Seq[npcai.Creature] {
	return func(yield func(npcai.Creature) bool) {
		for _, c := range a.ai.attackBy {
			if !yield(aiView(c)) {
				return
			}
		}
	}
}

func (a *aiNpc) Aggro() iter.Seq2[npcai.Creature, int64] {
	return func(yield func(npcai.Creature, int64) bool) {
		if a.Dead {
			return
		}
		for i := 0; i < len(a.ai.hate); i++ {
			e := &a.ai.hate[i]
			if !yield(aiView(e.Attacker), e.Hate) {
				return
			}
		}
	}
}

func (a *aiNpc) MostHated() npcai.Creature { return aiView(a.n().MostHated()) }

func (a *aiNpc) Hating(c npcai.Creature) int64 { return a.n().Hating(gameOf(c)) }

func (a *aiPlayer) p() *Player { return (*Player)(a) }

func (a *aiPlayer) Loc() npcai.Point {
	pos := a.p().Position()
	return npcai.Point{X: pos.X, Y: pos.Y, Z: pos.Z}
}

func (a *aiPlayer) CurrentHP() float64        { return a.p().CurrentHP() }
func (a *aiPlayer) MaxHP() int32              { return a.p().MaxHP() }
func (a *aiPlayer) IsDead() bool              { return a.p().IsDead() }
func (a *aiPlayer) IsMoving() bool            { return a.Move != nil }
func (a *aiPlayer) TemplateCollision() int    { return int(walkCollision(a.p())) }
func (a *aiPlayer) AIClass() npcai.Class      { return npcai.ClassPlayer }
func (a *aiPlayer) AIProfile() *npcai.Profile { return nil }

func (a *aiPlayer) AffectedBySkill(id int) bool {
	return a.skill != nil && a.skill.el.IsAffectedBySkill(id)
}

func (a *aiPlayer) HasBuffEffect() bool {
	return a.skill != nil && a.skill.el.FirstEffect(skills.TypeBuff) != nil
}

func (a *aiPlayer) HasEffects() bool    { return a.skill != nil && !a.skill.el.IsEmpty() }
func (a *aiPlayer) IsImmobilized() bool { return a.skill != nil && a.skill.immobilized }

// aiWorld is the world as npcai sees it.
type aiWorld struct{ s *Server }

// CanSee is GeoEngine.canSeeTarget.
func (w aiWorld) CanSee(from, to npcai.Creature) bool {
	a, b := gameOf(from), gameOf(to)
	if a == nil || b == nil {
		return false
	}
	return w.s.canSee(a, b)
}

// Visible walks the creatures around center in the world's region order,
// lazily.
func (w aiWorld) Visible(center npcai.Creature, of npcai.Of, radius int) iter.Seq[npcai.Creature] {
	return func(yield func(npcai.Creature) bool) {
		c := gameOf(center)
		if c == nil {
			return
		}
		var seq iter.Seq[world.Object]
		if radius < 0 {
			seq = w.s.world.Visible(c)
		} else {
			seq = w.s.world.VisibleInRange(c, int32(radius)) //nolint:gosec // ranges are small template values
		}
		for o := range seq {
			v := aiObjectView(o)
			if v == nil || !of.Accepts(v.AIClass()) {
				continue
			}
			if !yield(v) {
				return
			}
		}
	}
}

// AutoAttackable is target.isAutoAttackable(by) (Mobius
// Player.isAutoAttackable): only players in the game, never NPCs (they leave
// each other alone by default); not an invisible GM; friendly mobs never,
// monsters always, guards a player with karma, other NPCs a player with
// karma or a PvP flag.
func (w aiWorld) AutoAttackable(target npcai.Creature, by npcai.Npc) bool {
	n, _ := gameOf(by).(*Npc)
	return w.s.npcAutoAttacks(n, gameOf(target))
}

// npcAutoAttacks is target.isAutoAttackable(n) for the AI's checkTarget
// (see aiWorld.AutoAttackable).
func (s *Server) npcAutoAttacks(n *Npc, target Creature) bool {
	p, ok := target.(*Player)
	if !ok || p.state != stateInGame || (p.Invisible() && p.IsGM()) {
		return false
	}
	if n == nil {
		return true
	}
	c := s.aiClassOf(n)
	switch {
	case c.friendly:
		return false
	case c.monster:
		return true
	case c.guard:
		return p.Karma() > 0
	}
	return p.Karma() > 0 || p.PvPFlag > 0
}

// ----- carrying out decisions -----

// npcDo carries out a decision of npcai for n: the move Mobius started on
// the way first (Decision.Pawn), then the action.
func (s *Server) npcDo(n *Npc, d npcai.Decision) {
	if d.Pawn != nil {
		if p := gameOf(d.Pawn); p != nil {
			s.NpcMoveToPawn(n, p, int32(d.PawnRange)) //nolint:gosec // ranges are small
		}
	}
	switch d.Action {
	case npcai.ActCast:
		s.npcDecisionCast(n, d)
	case npcai.ActAttack:
		s.npcDoAttack(n, gameOf(d.Target))
	case npcai.ActMoveToPawn:
		if t := gameOf(d.Target); t != nil {
			s.NpcMoveToPawn(n, t, int32(d.Range)) //nolint:gosec // ranges are small
		}
	case npcai.ActMoveTo:
		s.NpcMoveTo(n, world.Position{X: d.To.X, Y: d.To.Y, Z: d.To.Z})
	case npcai.ActRetarget:
		t := gameOf(d.Target)
		if t == nil {
			return
		}
		if d.AddHate {
			s.addDamageHate(n, t, 0, d.Hate)
		}
		n.ai.attackTarget, n.ai.target = t, t
	}
}

// npcDecisionCast is a cast of the AI (Mobius clientStopMoving(null);
// npc.setTarget(target); npc.doCast(skill); npc.setTarget(restore)). The
// one-argument doCast the AI uses is not Monster's override, so no friendly
// skill is dropped here (see DoCastAt).
func (s *Server) npcDecisionCast(n *Npc, d npcai.Decision) {
	if d.Skill == nil {
		return
	}
	if d.StopMove {
		s.npcClientStopMoving(n, nil)
	}
	if t := gameOf(d.Target); t != nil {
		n.ai.target = t
	}
	if fn := s.ai.x.doCast; fn != nil {
		fn(n, d.Skill)
	} else {
		s.DoCast(n, d.Skill)
	}
	if d.Restore != nil {
		if t := gameOf(d.Restore); t != nil {
			n.ai.target = t
		}
	}
}
