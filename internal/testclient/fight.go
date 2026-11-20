package testclient

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"
)

// The playing helpers of Milestones 6 and 7: finding a monster, fighting it
// to its death, casting a skill until it takes effect, opening a trainer's
// skill list and the GM steps that set a character up (level, target).
// They build on the request methods of combat.go and on the World model.

var (
	// ErrDied ends a fight in which the character died.
	ErrDied = errors.New("testclient: the character died")

	// ErrTargetLost ends a fight whose target disappeared from sight
	// (DeleteObject) before it died.
	ErrTargetLost = errors.New("testclient: the target disappeared")

	// ErrNothingToLearn is returned by LearnableSkills when the trainer
	// answers with AcquireSkillDone: nothing is left to learn at the
	// character's level.
	ErrNothingToLearn = errors.New("testclient: the trainer has nothing to teach")
)

// worldPoll is how often the helpers that wait on the World model look at
// it.
const worldPoll = 50 * time.Millisecond

// pollWorld calls done every worldPoll until it is true or ctx ends.
func pollWorld(ctx context.Context, done func() bool) error {
	t := time.NewTicker(worldPoll)
	defer t.Stop()
	for !done() {
		select {
		case <-t.C:
		case <-ctx.Done():
			return fmt.Errorf("testclient: %w", ctx.Err())
		}
	}
	return nil
}

// PickMonster returns the monster of objs to attack and its distance from
// me: a living attackable NPC (shown by NpcInfo) of template npcID, or of any
// template when npcID is 0; the nearest one, the lower object ID on a tie.
func PickMonster(me Position, objs []Object, npcID int32) (Object, float64, bool) {
	var (
		best     Object
		bestDist = math.Inf(1)
		found    bool
	)
	for _, o := range objs {
		if o.Kind != KindNpc || !o.Attackable || o.Dead || (npcID != 0 && o.NpcID != npcID) {
			continue
		}
		d := o.Pos.DistanceTo(me)
		if !found || d < bestDist || (d == bestDist && o.ObjectID < best.ObjectID) {
			best, bestDist, found = o, d, true
		}
	}
	return best, bestDist, found
}

// HuntMonster returns a monster of template npcID (any template for 0) to
// attack, the nearest one the World model shows. When none is in sight, the
// character walks towards field until one shows up.
func (c *GameConn) HuntMonster(ctx context.Context, npcID int32, field Position) (Object, error) {
	pick := func() (Object, bool) {
		o, _, ok := PickMonster(c.world.Me().Pos, c.world.Npcs(), npcID)
		return o, ok
	}
	if o, ok := pick(); ok {
		return o, nil
	}
	if _, err := c.MoveTo(ctx, field.X, field.Y, field.Z); err != nil {
		return Object{}, err
	}
	var o Object
	err := pollWorld(ctx, func() bool {
		var ok bool
		o, ok = pick()
		return ok
	})
	if err != nil {
		return o, fmt.Errorf("no monster %d in sight on the way to %d %d: %w", npcID, field.X, field.Y, err)
	}
	return o, nil
}

// FightUntilDead waits until the monster or player targetID that the
// character attacks dies, and returns how many swings (the character's
// Attack packets) came meanwhile. Mobius keeps swinging on its own after the
// first AttackRequest, so FightUntilDead sends nothing; call Attack (or
// CastSkill) first. A death of the character ends it with ErrDied, the
// target leaving sight with ErrTargetLost. A target the World model shows
// dead already ends it at once.
func (c *GameConn) FightUntilDead(ctx context.Context, targetID int32) (int, error) {
	me := c.world.Me().ObjectID
	if c.world.Me().Dead {
		return 0, ErrDied
	}
	if o, ok := c.world.Object(targetID); ok && o.Dead {
		return 0, nil
	}
	swings := 0
	for {
		p, err := c.await(ctx, "the death of the target", func(p Packet) bool {
			switch v := p.(type) {
			case Attack:
				return v.AttackerID == me
			case Die:
				return v.ObjectID == targetID || v.ObjectID == me
			case DeleteObject:
				return v.ObjectID == targetID
			}
			return false
		})
		if err != nil {
			return swings, err
		}
		switch v := p.(type) {
		case Attack:
			swings++
		case Die:
			if v.ObjectID == me {
				return swings, ErrDied
			}
			return swings, nil
		case DeleteObject:
			return swings, ErrTargetLost
		}
	}
}

// Cast is a cast that took effect: the character's MagicSkillUse and the
// MagicSkillLaunched that followed it.
type Cast struct {
	Use      MagicSkillUse
	Launched MagicSkillLaunched
}

// CastOn casts skillID on targetID and waits until it takes effect: it
// targets the object first unless it is the target already (targetID 0
// keeps the current target, and the character's own ID casts on itself),
// casts (CastSkill) and waits for the character's MagicSkillLaunched of the
// skill. Mobius refuses a cast with a system message and ActionFailed, and
// an interrupted cast ends with MagicSkillCanceled; both end with the
// context, the cancel with an error naming it.
func (c *GameConn) CastOn(ctx context.Context, targetID, skillID int32) (Cast, error) {
	if targetID != 0 && c.world.Me().TargetID != targetID {
		if _, err := c.Target(ctx, targetID); err != nil {
			return Cast{}, err
		}
	}
	use, err := c.CastSkill(ctx, skillID, false, false)
	if err != nil {
		return Cast{}, err
	}
	me := c.world.Me().ObjectID
	var launched MagicSkillLaunched
	canceled := false
	_, err = c.await(ctx, "the MagicSkillLaunched of the cast", func(p Packet) bool {
		switch v := p.(type) {
		case MagicSkillLaunched:
			if v.CasterID == me && v.SkillID == skillID {
				launched = v
				return true
			}
		case MagicSkillCanceled:
			if v.ObjectID == me {
				canceled = true
				return true
			}
		}
		return false
	})
	if err != nil {
		return Cast{Use: use}, err
	}
	if canceled {
		return Cast{Use: use}, fmt.Errorf("testclient: the cast of skill %d was canceled", skillID)
	}
	return Cast{Use: use, Launched: launched}, nil
}

// RequestRestartPoint asks to get up at point, a Restart constant, without
// waiting, for negative tests; RestartPoint waits for the revival.
func (c *GameConn) RequestRestartPoint(ctx context.Context, point int32) error {
	return c.send(ctx, encodeInt(OpRequestRestartPoint, point))
}

// SkillListBypass is the trainer link that opens the list of skills to
// learn ("Learn skills" on a trainer's page): npc_<object ID>_SkillList.
func SkillListBypass(npcObjectID int32) string {
	return "npc_" + strconv.Itoa(int(npcObjectID)) + "_SkillList"
}

// LearnableSkills opens the skill list of the trainer npcObjectID (the
// SkillList bypass, see SkillListBypass) and returns its AcquireSkillList.
// When the trainer has nothing to teach, Mobius sends AcquireSkillDone
// instead, and LearnableSkills returns ErrNothingToLearn. The character
// must stand in reach of the trainer (talk to it first).
func (c *GameConn) LearnableSkills(ctx context.Context, npcObjectID int32) (AcquireSkillList, error) {
	if err := c.Bypass(ctx, SkillListBypass(npcObjectID)); err != nil {
		return AcquireSkillList{}, err
	}
	p, err := c.await(ctx, "AcquireSkillList or AcquireSkillDone", func(p Packet) bool {
		switch p.(type) {
		case AcquireSkillList, AcquireSkillDone:
			return true
		}
		return false
	})
	if err != nil {
		return AcquireSkillList{}, err
	}
	if l, ok := p.(AcquireSkillList); ok {
		return l, nil
	}
	return AcquireSkillList{}, ErrNothingToLearn
}

// LearnSkill learns skillID at level from the trainer whose list is open,
// the way the client does: it asks what the level costs
// (RequestAcquireSkillInfo), then learns it (RequestAcquireSkill), and
// returns the AcquireSkillInfo and the SkillList that has the skill.
func (c *GameConn) LearnSkill(ctx context.Context, skillID, level, typ int32) (AcquireSkillInfo, SkillList, error) {
	info, err := c.AcquireSkillInfo(ctx, skillID, level, typ)
	if err != nil {
		return info, SkillList{}, err
	}
	list, err := c.AcquireSkill(ctx, skillID, level, typ)
	return info, list, err
}

// TargetSelf targets the character itself (a click on it) unless it is the
// target already. GM commands such as //set_level and //add_skill act on
// the target.
func (c *GameConn) TargetSelf(ctx context.Context) error {
	me := c.world.Me()
	if me.ObjectID == 0 {
		return errors.New("testclient: TargetSelf before entering the world")
	}
	if me.TargetID == me.ObjectID {
		return nil
	}
	_, err := c.Target(ctx, me.ObjectID)
	return err
}

// LevelTo sets the character's level with the GM command //set_level: it
// targets the character, sends the command and waits for the UserInfo with
// the new level. The character needs GM rights; without them the command
// is ignored and the call ends with the context. A character at the level
// already gets nothing sent.
func (c *GameConn) LevelTo(ctx context.Context, level int32) (UserInfo, error) {
	if me := c.world.Me(); me.HasUserInfo && me.UserInfo.Level == level {
		return me.UserInfo, nil
	}
	if err := c.TargetSelf(ctx); err != nil {
		return UserInfo{}, err
	}
	if err := c.GMCommand(ctx, "set_level "+strconv.Itoa(int(level))); err != nil {
		return UserInfo{}, err
	}
	return Await(ctx, c, func(u UserInfo) bool { return u.Level == level })
}

// AddSP gives the character sp skill points with the GM command
// //add_exp_sp 0 sp and waits for the UserInfo that shows them. The
// character needs GM rights; without them the call ends with the context.
func (c *GameConn) AddSP(ctx context.Context, sp int32) (UserInfo, error) {
	if err := c.TargetSelf(ctx); err != nil {
		return UserInfo{}, err
	}
	want := c.world.Me().UserInfo.SP + sp
	if err := c.GMCommand(ctx, "add_exp_sp 0 "+strconv.Itoa(int(sp))); err != nil {
		return UserInfo{}, err
	}
	return Await(ctx, c, func(u UserInfo) bool { return u.SP >= want })
}

// AwaitEffect waits until the World model shows the character's buff
// skillID (AbnormalStatusUpdate) and returns it.
func (c *GameConn) AwaitEffect(ctx context.Context, skillID int32) (Effect, error) {
	var e Effect
	err := pollWorld(ctx, func() bool {
		var ok bool
		e, ok = c.world.Effect(skillID)
		return ok
	})
	if err != nil {
		return e, fmt.Errorf("no effect of skill %d: %w", skillID, err)
	}
	return e, nil
}
