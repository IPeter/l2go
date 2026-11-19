package skills

import (
	"slices"

	"github.com/IPeter/l2go/internal/data"
	sp "github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/stats"
)

// ConditionContext is what a condition tests, the arguments of Mobius
// Condition.test: the caster (Mobius effector), the target (effected, may
// be nil), the skill and the item (both may be nil). For the condition of a
// stat function the caster is the creature whose stat is computed.
type ConditionContext struct {
	Env    *Env
	Caster Creature
	Target Creature
	Skill  *Skill
	Item   *data.Item
}

// Cond is a compiled condition (Plan 10 K9).
type Cond func(c ConditionContext) bool

// never is the condition of a kind without a port: it fails closed.
func never(ConditionContext) bool { return false }

// CompileCondition compiles a condition tree of the data. Kinds without a
// port compile to a condition that always fails; they are returned in
// unported, sorted, each once.
func CompileCondition(c *data.Condition) (cond Cond, unported []data.ConditionKind) {
	set := make(map[data.ConditionKind]bool)
	cond = compileCondition(c, set)
	for k := range set {
		unported = append(unported, k)
	}
	slices.Sort(unported)
	return cond, unported
}

// ConditionPorted reports whether a condition kind has a port.
func ConditionPorted(k data.ConditionKind) bool {
	if k.IsLogic() {
		return true
	}
	_, ok := leafConditions[k]
	return ok
}

// compileCondition compiles c, adding the kinds without a port to unported.
func compileCondition(c *data.Condition, unported map[data.ConditionKind]bool) Cond {
	switch c.Kind {
	case data.CondAnd, data.CondOr:
		subs := make([]Cond, len(c.Conds))
		for i := range c.Conds {
			subs[i] = compileCondition(&c.Conds[i], unported)
		}
		if c.Kind == data.CondAnd {
			return func(cc ConditionContext) bool {
				for _, s := range subs {
					if !s(cc) {
						return false
					}
				}
				return true
			}
		}
		return func(cc ConditionContext) bool {
			for _, s := range subs {
				if s(cc) {
					return true
				}
			}
			return false
		}
	case data.CondNot:
		if len(c.Conds) != 1 {
			unported[c.Kind] = true
			return never
		}
		sub := compileCondition(&c.Conds[0], unported)
		return func(cc ConditionContext) bool { return !sub(cc) }
	}
	build, ok := leafConditions[c.Kind]
	if !ok {
		unported[c.Kind] = true
		return never
	}
	return build(c)
}

// leafConditions builds the leaf kinds with a port, from the condition
// classes of Mobius mechanics/conditions. Their tests are those classes'
// testImpl, with Mobius's null checks.
var leafConditions = map[data.ConditionKind]func(c *data.Condition) Cond{
	// ConditionPlayerRace: a player of one of the races.
	data.CondPlayerRaces: func(c *data.Condition) Cond {
		races := slices.Clone(c.Races)
		return func(cc ConditionContext) bool {
			return isPlayer(cc.Caster) && slices.Contains(races, cc.Caster.Race())
		}
	},
	// ConditionPlayerLevel: level at least the value.
	data.CondPlayerLevel: func(c *data.Condition) Cond {
		lvl := c.Int
		return func(cc ConditionContext) bool { return cc.Caster != nil && cc.Caster.Level() >= lvl }
	},
	// ConditionPlayerLevelRange.
	data.CondPlayerLevelrange: func(c *data.Condition) Cond {
		lo, hi := rangeOf(c)
		return func(cc ConditionContext) bool {
			if cc.Caster == nil {
				return false
			}
			l := cc.Caster.Level()
			return l >= lo && l <= hi
		}
	},
	data.CondPlayerResting:  playerState(stateResting),
	data.CondPlayerMoving:   playerState(stateMoving),
	data.CondPlayerRunning:  playerState(stateRunning),
	data.CondPlayerStanding: playerState(stateStanding),
	data.CondPlayerFlying:   playerState(stateFlying),
	data.CondPlayerBehind:   playerState(stateBehind),
	data.CondPlayerFront:    playerState(stateFront),
	data.CondPlayerChaotic:  playerState(stateChaotic),
	data.CondPlayerOlympiad: playerState(stateOlympiad),
	// ConditionPlayerIsHero.
	data.CondPlayerIshero: func(c *data.Condition) Cond {
		want := c.Bool
		return func(cc ConditionContext) bool {
			p := asPlayer(cc.Caster)
			return p != nil && p.IsHero() == want
		}
	},
	// ConditionPlayerIsPvpFlagged.
	data.CondPlayerIspvpflagged: func(c *data.Condition) Cond {
		want := c.Bool
		return func(cc ConditionContext) bool {
			p := asPlayer(cc.Caster)
			return p != nil && (p.PvpFlag() > 0) == want
		}
	},
	// ConditionPlayerHp: HP in percent at most the value.
	data.CondPlayerHP: func(c *data.Condition) Cond {
		v := float64(c.Int)
		return func(cc ConditionContext) bool {
			e := cc.Caster
			return e != nil && mul(e.CurrentHP(), 100)/float64(e.MaxHP()) <= v
		}
	},
	// ConditionPlayerMp: MP in percent at most the value.
	data.CondPlayerMP: func(c *data.Condition) Cond {
		v := float64(c.Int)
		return func(cc ConditionContext) bool {
			e := cc.Caster
			return e != nil && mul(e.CurrentMP(), 100)/float64(e.MaxMP()) <= v
		}
	},
	// ConditionPlayerCp: CP in percent at least the value.
	data.CondPlayerCP: func(c *data.Condition) Cond {
		v := float64(c.Int)
		return func(cc ConditionContext) bool {
			e := cc.Caster
			return e != nil && mul(e.CurrentCP(), 100)/float64(e.MaxCP()) >= v
		}
	},
	// ConditionPlayerPkCount: PK kills at most the value.
	data.CondPlayerPkcount: func(c *data.Condition) Cond {
		v := c.Int
		return func(cc ConditionContext) bool {
			p := asPlayer(cc.Caster)
			return p != nil && p.PkKills() <= v
		}
	},
	// ConditionPlayerCharges: a player with at least the charges.
	data.CondPlayerCharges: func(c *data.Condition) Cond {
		v := c.Int
		return func(cc ConditionContext) bool {
			return isPlayer(cc.Caster) && cc.Caster.AsPlayer().Charges() >= v
		}
	},
	// ConditionPlayerWeight: the load in percent, without the bonus weight,
	// under the value (or a diet); non-players and a zero limit pass.
	data.CondPlayerWeight: func(c *data.Condition) Cond {
		v := c.Int
		return func(cc ConditionContext) bool {
			p := asPlayer(cc.Caster)
			if p == nil || p.MaxLoad() <= 0 {
				return true
			}
			percent := (p.CurrentLoad() - p.BonusWeightPenalty()) * 100 / p.MaxLoad()
			return percent < v || p.DietMode()
		}
	},
	// ConditionPlayerInvSize: at least the value of free slots.
	data.CondPlayerInvsize: func(c *data.Condition) Cond {
		v := c.Int
		return func(cc ConditionContext) bool {
			p := asPlayer(cc.Caster)
			return p == nil || p.InventoryNonQuestSize() <= p.InventoryLimit()-v
		}
	},
	// ConditionPlayerSex: 0 male, 1 female.
	data.CondPlayerSex: func(c *data.Condition) Cond {
		v := c.Int
		return func(cc ConditionContext) bool {
			p := asPlayer(cc.Caster)
			if p == nil {
				return false
			}
			sex := 0
			if p.IsFemale() {
				sex = 1
			}
			return sex == v
		}
	},
	data.CondPlayerActiveEffectID:    activeEffect(false, false),
	data.CondPlayerActiveEffectIDLvl: activeEffect(false, true),
	data.CondTargetActiveEffectID:    activeEffect(true, false),
	data.CondTargetActiveEffectIDLvl: activeEffect(true, true),
	data.CondPlayerActiveSkillID:     activeSkill(false, false),
	data.CondPlayerActiveSkillIDLvl:  activeSkill(false, true),
	data.CondTargetActiveSkillID:     activeSkill(true, false),
	data.CondTargetActiveSkillIDLvl:  activeSkill(true, true),
	// ConditionPlayerClassIdRestriction.
	data.CondPlayerClassIDRestriction: func(c *data.Condition) Cond {
		ids := slices.Clone(c.Ints)
		return func(cc ConditionContext) bool {
			return isPlayer(cc.Caster) && slices.Contains(ids, cc.Caster.AsPlayer().ClassID())
		}
	},
	// ConditionTargetClassIdRestriction.
	data.CondTargetClassIDRestriction: func(c *data.Condition) Cond {
		ids := slices.Clone(c.Ints)
		return func(cc ConditionContext) bool {
			return isPlayer(cc.Target) && slices.Contains(ids, cc.Target.AsPlayer().ClassID())
		}
	},
	// ConditionPlayerCanEscape.
	data.CondPlayerCanescape: func(c *data.Condition) Cond {
		want := c.Bool
		return func(cc ConditionContext) bool {
			p := asPlayer(cc.Caster)
			return want == (p != nil && p.CanEscape())
		}
	},
	// ConditionPlayerInsideZoneId: a player in one of the zones.
	data.CondPlayerInsidezoneid: func(c *data.Condition) Cond {
		ids := slices.Clone(c.Ints)
		return func(cc ConditionContext) bool {
			if asPlayer(cc.Caster) == nil {
				return false
			}
			for _, id := range ids {
				if cc.Caster.InZone(id) {
					return true
				}
			}
			return false
		}
	},
	// ConditionPlayerCheckAbnormal: an effect of the abnormal type, of at
	// most the level when one is given.
	data.CondPlayerCheckabnormal: func(c *data.Condition) Cond {
		t, lvl := c.Abnormal, c.Int
		return func(cc ConditionContext) bool {
			if cc.Caster == nil {
				return false
			}
			info := cc.Caster.Effects().ByAbnormal(t)
			return info != nil && (lvl == -1 || lvl >= info.skill.AbnormalLevel)
		}
	},
	// ConditionCategoryType: in one of the categories.
	data.CondPlayerCategorytype: func(c *data.Condition) Cond {
		cats := slices.Clone(c.Categories)
		return func(cc ConditionContext) bool {
			if cc.Caster == nil {
				return false
			}
			for _, t := range cats {
				if cc.Caster.InCategory(t) {
					return true
				}
			}
			return false
		}
	},
	// ConditionTargetRace.
	data.CondTargetRace: func(c *data.Condition) Cond {
		var race data.Race
		if len(c.Races) > 0 {
			race = c.Races[0]
		}
		return func(cc ConditionContext) bool { return cc.Target != nil && cc.Target.Race() == race }
	},
	// ConditionTargetLevel: target level at least the value.
	data.CondTargetLevel: func(c *data.Condition) Cond {
		lvl := c.Int
		return func(cc ConditionContext) bool { return cc.Target != nil && cc.Target.Level() >= lvl }
	},
	// ConditionTargetLevelRange.
	data.CondTargetLevelrange: func(c *data.Condition) Cond {
		lo, hi := rangeOf(c)
		return func(cc ConditionContext) bool {
			if cc.Target == nil {
				return false
			}
			l := cc.Target.Level()
			return l >= lo && l <= hi
		}
	},
	// ConditionTargetPlayable.
	data.CondTargetPlayable: func(*data.Condition) Cond {
		return func(cc ConditionContext) bool { return isPlayable(cc.Target) }
	},
	// ConditionTargetNpcId: a player's own target, else the effected, must
	// be an NPC or door with one of the IDs.
	data.CondTargetNpcid: func(c *data.Condition) Cond {
		ids := slices.Clone(c.Ints)
		return func(cc ConditionContext) bool {
			target := cc.Target
			if isPlayer(cc.Caster) {
				target = cc.Caster.Target()
			}
			return target != nil && (isNpc(target) || isDoor(target)) && slices.Contains(ids, target.NpcID())
		}
	},
	// ConditionTargetNpcType: the target is one of the instance types.
	data.CondTargetNpctype: func(c *data.Condition) Cond {
		types := slices.Clone(c.NpcTypes)
		return func(cc ConditionContext) bool {
			if cc.Target == nil {
				return false
			}
			it := cc.Target.InstanceType()
			for _, t := range types {
				if it.Is(t) {
					return true
				}
			}
			return false
		}
	},
	// ConditionTargetAbnormal: the target shows one of the visual effect
	// bits.
	data.CondTargetAbnormal: func(c *data.Condition) Cond {
		mask := uint32(c.Int) //nolint:gosec // a bit mask
		return func(cc ConditionContext) bool {
			return cc.Target != nil && cc.Target.Effects().Visual().Normal&mask != 0
		}
	},
	// ConditionTargetUsesWeaponKind: the target's active weapon is of one of
	// the types.
	data.CondTargetUsing: func(c *data.Condition) Cond {
		mask := c.Int
		return func(cc ConditionContext) bool {
			if cc.Target == nil {
				return false
			}
			w, ok := cc.Target.ActiveWeapon()
			return ok && w.Type.Mask()&mask != 0
		}
	},
	// ConditionTargetMyPartyExceptMe: the target is another member of the
	// caster's party; the caster is told why not.
	data.CondTargetMypartyexceptme: func(c *data.Condition) Cond {
		want := c.Bool
		return func(cc ConditionContext) bool {
			member := true
			p := asPlayer(cc.Caster)
			switch {
			case p == nil || !isPlayer(cc.Target):
				member = false
			case same(cc.Caster, cc.Target):
				send(cc.Caster, MsgCannotUseOnYourself)
				member = false
			case !inParty(cc.Env, cc.Caster) || !sameParty(cc.Env, cc.Caster, cc.Target):
				if cc.Skill != nil {
					send(cc.Caster, MsgS1CannotBeUsedUnsuitable, SkillName(cc.Skill))
				} else {
					send(cc.Caster, MsgS1CannotBeUsedUnsuitable)
				}
				member = false
			}
			return want == member
		}
	},
	// ConditionUsingItemType (using kind): see usingKind.
	data.CondUsingKind: func(c *data.Condition) Cond { return usingKind(c.Int) },
	// ConditionUsingSlotType: a player whose active weapon takes one of the
	// body parts.
	data.CondUsingSlot: func(c *data.Condition) Cond {
		mask := c.Int
		return func(cc ConditionContext) bool {
			if !isPlayer(cc.Caster) {
				return false
			}
			w, ok := cc.Caster.ActiveWeapon()
			return ok && w.BodyPart.Mask()&mask != 0
		}
	},
	// ConditionUsingSkill: the skill of the test has the ID.
	data.CondUsingSkill: func(c *data.Condition) Cond {
		id := c.Int
		return func(cc ConditionContext) bool { return cc.Skill != nil && cc.Skill.ID == id }
	},
	// ConditionSlotItemId: a player with the item, at least the enchant, in
	// the paperdoll slot (item 0: the slot is empty).
	data.CondUsingSlotitem: func(c *data.Condition) Cond {
		var item, slot, enchant int
		if len(c.Ints) >= 3 {
			item, slot, enchant = c.Ints[0], c.Ints[1], c.Ints[2]
		}
		return func(cc ConditionContext) bool {
			if !isPlayer(cc.Caster) {
				return false
			}
			id, ench, ok := cc.Caster.AsPlayer().PaperdollItem(slot)
			if !ok {
				return item == 0
			}
			return id == item && ench >= enchant
		}
	},
	// ConditionWithSkill: whether the test has a skill.
	data.CondGameSkill: func(c *data.Condition) Cond {
		want := c.Bool
		return func(cc ConditionContext) bool { return (cc.Skill != nil) == want }
	},
	// ConditionGameTime: night or day.
	data.CondGameNight: func(c *data.Condition) Cond {
		want := c.Bool
		return func(cc ConditionContext) bool {
			night := cc.Env != nil && cc.Env.World != nil && cc.Env.World.IsNight()
			return night == want
		}
	},
	// ConditionGameChance: Get(100) under the value. Draws once.
	data.CondGameChance: func(c *data.Condition) Cond {
		v := c.Int
		return func(cc ConditionContext) bool {
			return cc.Env != nil && cc.Env.Rand.Get(100) < v
		}
	},
}

// rangeOf returns the (min, max) of a levelrange condition.
func rangeOf(c *data.Condition) (lo, hi int) {
	if len(c.Ints) >= 2 {
		return c.Ints[0], c.Ints[1]
	}
	return 1, 0 // an empty range: Mobius drops a malformed one
}

// asPlayer returns c's acting player, nil for a nil creature.
func asPlayer(c Creature) Player {
	if c == nil {
		return nil
	}
	return c.AsPlayer()
}

func inParty(env *Env, c Creature) bool {
	return env != nil && env.World != nil && env.World.Party(c) != nil
}

func sameParty(env *Env, a, b Creature) bool {
	return env != nil && env.World != nil && env.World.SameParty(a, b)
}

// The states of ConditionPlayerState.
type playerStateKind uint8

const (
	stateResting playerStateKind = iota
	stateMoving
	stateRunning
	stateStanding
	stateFlying
	stateBehind
	stateFront
	stateChaotic
	stateOlympiad
)

// playerState builds ConditionPlayerState.
func playerState(state playerStateKind) func(c *data.Condition) Cond {
	return func(c *data.Condition) Cond {
		want := c.Bool
		return func(cc ConditionContext) bool {
			e := cc.Caster
			if e == nil {
				return !want
			}
			p := e.AsPlayer() // Mobius asPlayer: a summon answers with its owner
			switch state {
			case stateResting:
				if p != nil {
					return p.IsSitting() == want
				}
				return !want
			case stateMoving:
				return e.IsMoving() == want
			case stateRunning:
				return e.IsMoving() == want && e.IsRunning() == want
			case stateStanding:
				if p != nil {
					return want != (p.IsSitting() || e.IsMoving())
				}
				return want != e.IsMoving()
			case stateFlying:
				return e.IsFlying() == want
			case stateBehind:
				return cc.Target != nil && isBehind(e, cc.Target) == want
			case stateFront:
				return cc.Target != nil && isInFrontOf(e, cc.Target) == want
			case stateChaotic:
				if p != nil {
					return (p.Karma() > 0) == want
				}
				return !want
			case stateOlympiad:
				if p != nil {
					return p.InOlympiad() == want
				}
				return !want
			}
			return !want
		}
	}
}

// activeEffect builds ConditionPlayerActiveEffectId and
// ConditionTargetActiveEffectId: an effect of the skill ID, of at least the
// level when one is given.
func activeEffect(target, withLevel bool) func(c *data.Condition) Cond {
	return func(c *data.Condition) Cond {
		id, lvl := c.Int, -1
		if withLevel && len(c.Ints) >= 2 {
			id, lvl = c.Ints[0], c.Ints[1]
		}
		return func(cc ConditionContext) bool {
			e := cc.Caster
			if target {
				e = cc.Target
			}
			if e == nil {
				return false
			}
			info := e.Effects().BySkillID(id)
			return info != nil && (lvl == -1 || lvl <= info.skill.Level)
		}
	}
}

// activeSkill builds ConditionPlayerActiveSkillId and
// ConditionTargetActiveSkillId: a known skill of the ID, of at least the
// level when one is given.
func activeSkill(target, withLevel bool) func(c *data.Condition) Cond {
	return func(c *data.Condition) Cond {
		id, lvl := c.Int, -1
		if withLevel && len(c.Ints) >= 2 {
			id, lvl = c.Ints[0], c.Ints[1]
		}
		return func(cc ConditionContext) bool {
			e := cc.Caster
			if target {
				e = cc.Target
			}
			if e == nil {
				return false
			}
			known := e.SkillLevel(id)
			return known > 0 && (lvl == -1 || lvl <= known)
		}
	}
}

// usingKind is ConditionUsingItemType. An NPC passes with its attack type,
// never with armor kinds. A player's armor kinds test the chest and legs:
// both empty is "no armor"; a chest alone counts only when it is a full
// armor; chest and legs count when they are of one kind; anything else is
// "no armor". Weapon kinds (and legs without a chest) test the worn mask.
func usingKind(mask int) Cond {
	none := data.ArmorNone.Mask()
	armor := mask&(data.ArmorMagic.Mask()|data.ArmorLight.Mask()|data.ArmorHeavy.Mask()) != 0
	return func(cc ConditionContext) bool {
		e := cc.Caster
		if e == nil {
			return false
		}
		if !isPlayer(e) {
			return !armor && mask&e.AttackType().Mask() != 0
		}
		p := e.AsPlayer()
		if armor {
			chest, legs, full := p.ArmorMasks()
			switch {
			case chest == 0 && legs == 0:
				return none&mask == none
			case chest != 0 && legs == 0:
				if full {
					return mask&chest != 0
				}
				return none&mask == none
			case chest != 0:
				if chest == legs {
					return mask&chest != 0
				}
				return none&mask == none
			}
		}
		return mask&p.WornMask() != 0
	}
}

// Precondition is a compiled <conditions> block of a skill: the condition
// the caster must meet and the message it gets when it does not.
type Precondition struct {
	Cond    Cond
	MsgID   int
	AddName bool
	Msg     string
}

// CheckCondition is Skill.checkCondition: it tests the preconditions in
// order with caster and target, and sends the message of the first that
// fails. GMs pass when GMSkillRestriction is off. itemOrWeapon asks for the
// item preconditions, which Interlude skills do not have.
func (s *Skill) CheckCondition(env *Env, caster, target Creature, itemOrWeapon bool) bool {
	if itemOrWeapon || len(s.preconds) == 0 {
		return true
	}
	if p := asPlayer(caster); p != nil && isPlayer(caster) && p.IsGM() && !env.Config.GMSkillRestriction {
		return true
	}
	cc := ConditionContext{Env: env, Caster: caster, Target: target, Skill: s}
	for i := range s.preconds {
		pc := &s.preconds[i]
		if pc.Cond(cc) {
			continue
		}
		switch {
		case pc.MsgID != 0:
			var params []sp.SystemMessageParam
			if pc.AddName {
				params = append(params, skillNameLevel1(s.ID))
			}
			send(caster, sp.SystemMessageID(pc.MsgID), params...) //nolint:gosec // message IDs are small
		case pc.Msg != "":
			if caster != nil {
				caster.SendText(pc.Msg)
			}
		}
		return false
	}
	return true
}

// Preconditions returns the compiled preconditions. The slice must not be
// changed.
func (s *Skill) Preconditions() []Precondition { return s.preconds }

// isBehind is ILocational.isBehind: a stands at target's back.
func isBehind(a, target Creature) bool {
	la, lt := a.Location(), target.Location()
	return stats.PositionOf(la.X, la.Y, lt.X, lt.Y, target.Heading()) == stats.PosBack
}

// isInFrontOf is ILocational.isInFrontOf: a stands in front of target.
func isInFrontOf(a, target Creature) bool {
	la, lt := a.Location(), target.Location()
	return stats.PositionOf(la.X, la.Y, lt.X, lt.Y, target.Heading()) == stats.PosFront
}
