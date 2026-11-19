package testclient

import (
	"context"

	"github.com/IPeter/l2go/internal/packets"
)

// Restart points of RequestRestartPoint: the buttons of the window Die
// opens. Mobius sends every other value to the village too.
const (
	RestartVillage  int32 = 0
	RestartClanHall int32 = 1
	RestartCastle   int32 = 2
	RestartSiegeHQ  int32 = 3
	// RestartFixed gets up where the character died, without a teleport.
	RestartFixed int32 = 4
)

// encodeRequestMagicSkillUse builds RequestMagicSkillUse: d skill, d ctrl,
// c shift.
func encodeRequestMagicSkillUse(skillID int32, ctrl, shift bool) []byte {
	w := packets.NewWriter(10)
	w.WriteUint8(OpRequestMagicSkillUse)
	w.WriteInt32(skillID)
	w.WriteInt32(boolInt32(ctrl))
	w.WriteUint8(boolByte(shift))
	return w.Bytes()
}

// encodeRequestAutoSoulShot builds the Ex packet RequestAutoSoulShot: d
// item ID, d 1 to turn the automatic use on, 0 to turn it off.
func encodeRequestAutoSoulShot(itemID int32, on bool) []byte {
	w := packets.NewWriter(11)
	w.WriteUint8(OpExClient)
	w.WriteUint16(ExOpRequestAutoSoulShot)
	w.WriteInt32(itemID)
	w.WriteInt32(boolInt32(on))
	return w.Bytes()
}

// SendAttackRequest sends AttackRequest for objectID from the position the
// world model knows, without waiting; shift sends a shift click.
func (c *GameConn) SendAttackRequest(ctx context.Context, objectID int32, shift bool) error {
	return c.send(ctx, encodeAction(OpAttackRequest, objectID, c.world.Me().Pos, shift))
}

// Attack attacks targetID and returns the character's first Attack with a
// hit on it. Like the client, it targets the object first unless it is the
// target already (Mobius takes an AttackRequest on another object as a
// click that targets it), then sends AttackRequest. The character walks
// into range before it swings; later swings come as more Attack packets.
// Mobius refuses an attack with ActionFailed alone, so a refused attack
// ends with the context; AttackRequest waits for that refusal instead.
func (c *GameConn) Attack(ctx context.Context, targetID int32) (Attack, error) {
	if c.world.Me().TargetID != targetID {
		if _, err := c.Target(ctx, targetID); err != nil {
			return Attack{}, err
		}
	}
	if err := c.SendAttackRequest(ctx, targetID, false); err != nil {
		return Attack{}, err
	}
	me := c.world.Me().ObjectID
	return Await(ctx, c, func(a Attack) bool {
		_, hit := a.HitOn(targetID)
		return a.AttackerID == me && hit
	})
}

// RestartResult is the answer to RestartPoint.
type RestartResult struct {
	// Revive is the character's Revive.
	Revive Revive
	// Teleported is set when the server moved the character: Teleport is
	// its TeleportToLocation and UserInfo the answer to the Appearing that
	// RestartPoint sent.
	Teleported bool
	Teleport   TeleportToLocation
	UserInfo   UserInfo
}

// RestartPoint gets the dead character up at point, a Restart constant
// (RequestRestartPoint), and waits until it lives again: for its Revive
// and, unless point is RestartFixed, for its TeleportToLocation, which it
// answers with Appearing as FinishTeleport does. Mobius revives the
// character just before it teleports it; either order works. Mobius
// ignores the request of a living character, so such a call ends with its
// context.
func (c *GameConn) RestartPoint(ctx context.Context, point int32) (RestartResult, error) {
	if err := c.send(ctx, encodeInt(OpRequestRestartPoint, point)); err != nil {
		return RestartResult{}, err
	}
	me := c.world.Me().ObjectID
	var (
		res     RestartResult
		revived bool
	)
	_, err := c.await(ctx, "the character's Revive and TeleportToLocation", func(p Packet) bool {
		switch v := p.(type) {
		case Revive:
			if v.ObjectID == me {
				res.Revive, revived = v, true
			}
		case TeleportToLocation:
			if v.ObjectID == me {
				res.Teleport, res.Teleported = v, true
			}
		}
		return revived && (res.Teleported || point == RestartFixed)
	})
	if err != nil {
		return res, err
	}
	if res.Teleported {
		u, err := c.Appearing(ctx)
		if err != nil {
			return res, err
		}
		res.UserInfo = u
	}
	return res, nil
}

// RequestMagicSkillUse asks to cast skillID (RequestMagicSkillUse) without
// waiting: ctrl forces the skill on a target that is not hostile, shift
// casts without walking into range.
func (c *GameConn) RequestMagicSkillUse(ctx context.Context, skillID int32, ctrl, shift bool) error {
	return c.send(ctx, encodeRequestMagicSkillUse(skillID, ctrl, shift))
}

// CastSkill casts skillID (RequestMagicSkillUse) and returns the
// character's MagicSkillUse for it; SetupGauge, and MagicSkillLaunched when
// the skill takes effect, come after it. Mobius refuses a cast with a
// system message and ActionFailed, so a refused cast ends with the
// context; negative tests use RequestMagicSkillUse.
func (c *GameConn) CastSkill(ctx context.Context, skillID int32, ctrl, shift bool) (MagicSkillUse, error) {
	if err := c.RequestMagicSkillUse(ctx, skillID, ctrl, shift); err != nil {
		return MagicSkillUse{}, err
	}
	me := c.world.Me().ObjectID
	return Await(ctx, c, func(m MagicSkillUse) bool { return m.CasterID == me && m.SkillID == skillID })
}

// AcquireSkillInfo asks what learning skillID at level costs
// (RequestAcquireSkillInfo; typ is an AcquireSkill constant) and returns
// the AcquireSkillInfo that answers. Mobius ignores a skill the trainer
// does not teach, so such a call ends with the context.
func (c *GameConn) AcquireSkillInfo(ctx context.Context, skillID, level, typ int32) (AcquireSkillInfo, error) {
	if err := c.send(ctx, encodeInts(OpRequestAcquireSkillInfo, skillID, level, typ)); err != nil {
		return AcquireSkillInfo{}, err
	}
	return Await(ctx, c, func(a AcquireSkillInfo) bool { return a.SkillID == skillID && a.Level == level })
}

// RequestAcquireSkill asks to learn skillID at level (RequestAcquireSkill)
// without waiting, for negative tests: Mobius refuses with a system message
// and sends the trainer's list again.
func (c *GameConn) RequestAcquireSkill(ctx context.Context, skillID, level, typ int32) error {
	return c.send(ctx, encodeInts(OpRequestAcquireSkill, skillID, level, typ))
}

// AcquireSkill learns skillID at level from the trainer whose list is open
// (RequestAcquireSkill; typ is an AcquireSkill constant) and returns the
// SkillList that has it. Mobius then sends the trainer's next
// AcquireSkillList, or AcquireSkillDone when nothing is left to learn. A
// refused request ends with the context; negative tests use
// RequestAcquireSkill.
func (c *GameConn) AcquireSkill(ctx context.Context, skillID, level, typ int32) (SkillList, error) {
	if err := c.RequestAcquireSkill(ctx, skillID, level, typ); err != nil {
		return SkillList{}, err
	}
	return Await(ctx, c, func(l SkillList) bool { return skillLevel(l.Skills, skillID) == level })
}

// RegisterShortcut puts sc in its slot of the shortcut bar
// (RequestShortcutReg: type, slot plus 12 times the page, ID and level;
// Rest is not sent) and returns the ShortcutRegister that confirms it.
// Mobius ignores a skill the character does not know, so such a call ends
// with the context.
func (c *GameConn) RegisterShortcut(ctx context.Context, sc Shortcut) (ShortcutRegister, error) {
	if err := c.send(ctx, encodeInts(OpRequestShortcutReg, sc.Type, sc.Slot, sc.ID, sc.Level)); err != nil {
		return ShortcutRegister{}, err
	}
	return Await(ctx, c, func(r ShortcutRegister) bool { return r.Slot == sc.Slot })
}

// DeleteShortcut empties a slot of the shortcut bar (RequestShortcutDel;
// slot plus 12 times the page). Mobius does not answer, so the world model
// drops the slot at once.
func (c *GameConn) DeleteShortcut(ctx context.Context, slot int32) error {
	if err := c.send(ctx, encodeInt(OpRequestShortcutDel, slot)); err != nil {
		return err
	}
	c.world.deleteShortcut(slot)
	return nil
}

// AutoSoulShot turns the automatic use of the shot item itemID on or off
// (the Ex packet RequestAutoSoulShot) and returns the ExAutoSoulShot that
// confirms it. Mobius answers nothing when the item is not in the
// inventory, so such a call ends with the context.
func (c *GameConn) AutoSoulShot(ctx context.Context, itemID int32, on bool) (ExAutoSoulShot, error) {
	if err := c.send(ctx, encodeRequestAutoSoulShot(itemID, on)); err != nil {
		return ExAutoSoulShot{}, err
	}
	return Await(ctx, c, func(a ExAutoSoulShot) bool { return a.ItemID == itemID && a.Enabled == on })
}
