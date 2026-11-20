package scenario

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

// The Milestone 7 scenarios (Plan 10, Phase 7.8): casting, learning a
// skill, buff stacking, a potion, soulshots and the newbie buffer. The
// deterministic fields are what they compare: cast and reuse times in
// MagicSkillUse, the trainer's AcquireSkillList and AcquireSkillInfo, the
// SkillList, the AbnormalStatusUpdate entries, ShortBuffStatusUpdate and
// ExRegenMax. Except cast-wind-strike and soulshot, which fight, every one
// starts from a fixed spot set up with //move_to; all set their characters
// up with GM commands.

// Skills the scenarios use (Mobius skill IDs).
const (
	skillWindStrike int32 = 1177
	skillMight      int32 = 1068
	skillShield     int32 = 1040
)

// Settings of the scenarios.
const (
	// maxCasts is how many Wind Strikes cast-wind-strike casts on one
	// monster before it finishes the fight with its fists.
	maxCasts = 5
	// learnLevel is the level at which Baulro teaches Wind Strike 2, and
	// learnSP the SP the setup gives for it (it costs 240).
	learnLevel = 7
	learnSP    = 1000
	// buffLevel gives the buff-stack character the MP for three casts.
	buffLevel = 10
	// newbieLevel is the lowest level of the newbie buffer (Wind Walk).
	newbieLevel = 8
	// mightGMLevel is the level of Might the GM buff puts over the cast
	// one in buff-stack.
	mightGMLevel = 3
	// potionLimit outlasts the 15 s of a Healing Potion.
	potionLimit = 25 * time.Second
)

// setLevel is a GM step: //set_level gives p the level (testclient.LevelTo).
func setLevel(p *Player, level int32) Step {
	return Step{fmt.Sprintf("%s goes to level %d (GM)", p.Role, level), func(ctx context.Context) error {
		if _, err := p.Game().LevelTo(ctx, level); err != nil {
			return gmHint(p, err)
		}
		return nil
	}}
}

// addSP is a GM step: //add_exp_sp gives p sp skill points.
func addSP(p *Player, sp int32) Step {
	return Step{fmt.Sprintf("%s gets %d SP (GM)", p.Role, sp), func(ctx context.Context) error {
		if _, err := p.Game().AddSP(ctx, sp); err != nil {
			return gmHint(p, err)
		}
		return nil
	}}
}

// addSkill is a GM step: //add_skill gives p the skill level, and the step
// waits until the SkillList has it.
func addSkill(p *Player, id, level int32) Step {
	return Step{fmt.Sprintf("%s gets skill %d level %d (GM)", p.Role, id, level), func(ctx context.Context) error {
		gc := p.Game()
		if err := gc.TargetSelf(ctx); err != nil {
			return err
		}
		if err := gc.GMCommand(ctx, "add_skill "+strconv.Itoa(int(id))+" "+strconv.Itoa(int(level))); err != nil {
			return err
		}
		if err := poll(ctx, func() bool { return gc.World().SkillLevel(id) == level }); err != nil {
			return gmHint(p, fmt.Errorf("skill %d is at level %d, want %d: %w", id, gc.World().SkillLevel(id), level, err))
		}
		return nil
	}}
}

// reuseWait is how long to wait after cast c before casting again: its
// reuse, and settle for the packets after it.
func reuseWait(c testclient.Cast) time.Duration {
	return time.Duration(max(c.Use.ReuseDelay, 0))*time.Millisecond + settle
}

// awaitDeath waits up to d for the object targetID to die (or leave sight)
// and reports whether it did. A death of p's character ends it with
// testclient.ErrDied.
func awaitDeath(ctx context.Context, p *Player, targetID int32, d time.Duration) (bool, error) {
	w := p.Game().World()
	dctx, cancel := context.WithTimeout(ctx, d)
	defer cancel()
	err := poll(dctx, func() bool {
		o, ok := w.Object(targetID)
		return !ok || o.Dead || w.Me().Dead
	})
	switch {
	case ctx.Err() != nil:
		return false, ctx.Err()
	case err != nil:
		return false, nil
	case w.Me().Dead:
		return false, testclient.ErrDied
	}
	return true, nil
}

// castUntilDead casts skillID at targetID until it dies, at most maxCasts
// times with the reuse between the casts, and finishes a survivor with
// melee attacks. It returns the casts.
func castUntilDead(ctx context.Context, p *Player, targetID, skillID int32) (int, error) {
	casts := 0
	for casts < maxCasts {
		c, err := p.Game().CastOn(ctx, targetID, skillID)
		if err != nil {
			return casts, err
		}
		casts++
		dead, err := awaitDeath(ctx, p, targetID, reuseWait(c))
		if err != nil || dead {
			return casts, err
		}
	}
	if _, err := p.Game().Attack(ctx, targetID); err != nil {
		return casts, err
	}
	_, err := p.Game().FightUntilDead(ctx, targetID)
	return casts, err
}

func castWindStrike(ctx context.Context, s *Session) error {
	a := s.Player("alice")
	a.Template = HumanMystic
	if err := s.Prepare(ctx, a, gmMoveTo(a, keltirField)); err != nil {
		return err
	}
	if err := s.Enter(ctx, a); err != nil {
		return err
	}
	var target testclient.Object
	return s.Steps(ctx,
		Idle(time.Second),
		Long(walkLimit, Step{"alice finds a Bearded Keltir", func(ctx context.Context) error {
			var err error
			target, err = huntMonster(ctx, a, npcBeardedKeltir, keltirField)
			if err == nil {
				s.Note("monster: template %d %q", target.NpcID, target.Name)
			}
			return err
		}}),
		Long(fightLimit, Step{"alice casts Wind Strike until it dies", func(ctx context.Context) error {
			n, err := castUntilDead(ctx, a, target.ObjectID, skillWindStrike)
			s.Note("the monster took %d Wind Strikes", n)
			return err
		}}),
		Step{"alice waits out the combat stance", func(ctx context.Context) error { return sleep(ctx, stanceWait) }},
		Step{"alice logs out", a.Logout},
	)
}

func learnSkill(ctx context.Context, s *Session) error {
	a := s.Player("alice")
	a.Template = HumanMystic
	err := s.Prepare(ctx, a,
		setLevel(a, learnLevel),
		addSP(a, learnSP),
		gmMoveTo(a, spotBaulro),
	)
	if err != nil {
		return err
	}
	if err := s.Enter(ctx, a); err != nil {
		return err
	}
	var page testclient.NpcHtmlMessage
	return s.Steps(ctx,
		Idle(time.Second),
		Step{"alice talks to Baulro", func(ctx context.Context) error {
			var err error
			page, err = talkTo(ctx, s, a, npcBaulro)
			return err
		}},
		Step{"alice opens the skills to learn", func(ctx context.Context) error {
			if err := openLink(ctx, a, page, "_SkillList"); err != nil {
				return err
			}
			list, err := testclient.Await[testclient.AcquireSkillList](ctx, a.Game(), nil)
			if err != nil {
				return err
			}
			ws, ok := list.Find(skillWindStrike)
			if !ok {
				return fmt.Errorf("the list of %d skills has no Wind Strike", len(list.Skills))
			}
			s.Note("%d skills to learn; Wind Strike %d for %d SP", len(list.Skills), ws.NextLevel, ws.SPCost)
			return nil
		}},
		Idle(settle),
		Step{"alice learns Wind Strike 2", func(ctx context.Context) error {
			info, _, err := a.Game().LearnSkill(ctx, skillWindStrike, 2, testclient.AcquireSkillClass)
			s.Note("Wind Strike 2 costs %d SP", info.SPCost)
			return err
		}},
		Idle(time.Second),
		Step{"alice logs out", a.Logout},
	)
}

// castSelf is a step in which p casts skillID on itself and waits for its
// icon; *last keeps the cast for the reuse wait of the next step.
func castSelf(s *Session, p *Player, name string, skillID int32, last *testclient.Cast) Step {
	return Step{name, func(ctx context.Context) error {
		c, err := p.Game().CastOn(ctx, p.ObjectID(), skillID)
		if err != nil {
			return err
		}
		*last = c
		e, err := p.Game().AwaitEffect(ctx, skillID)
		if err == nil {
			s.Note("skill %d: icon level %d, %d s", skillID, e.Level, e.Duration)
		}
		return err
	}}
}

// waitReuse is a step that waits out the reuse of *last.
func waitReuse(last *testclient.Cast) Step {
	return Step{"wait for the reuse", func(ctx context.Context) error { return sleep(ctx, reuseWait(*last)) }}
}

func buffStack(ctx context.Context, s *Session) error {
	a := s.Player("alice")
	a.Template = HumanMystic
	err := s.Prepare(ctx, a,
		setLevel(a, buffLevel),
		addSkill(a, skillMight, 1),
		addSkill(a, skillShield, 1),
		gmMoveTo(a, spotGuide),
	)
	if err != nil {
		return err
	}
	if err := s.Enter(ctx, a); err != nil {
		return err
	}
	var last testclient.Cast
	return s.Steps(ctx,
		Idle(time.Second),
		Step{"alice targets herself", func(ctx context.Context) error { return a.Game().TargetSelf(ctx) }},
		castSelf(s, a, "alice casts Might 1 on herself", skillMight, &last),
		waitReuse(&last),
		castSelf(s, a, "alice casts Shield 1 on herself", skillShield, &last),
		waitReuse(&last),
		Step{fmt.Sprintf("a GM buff puts Might %d over Might 1", mightGMLevel), func(ctx context.Context) error {
			if err := a.Game().GMCommand(ctx, fmt.Sprintf("buff %d %d", skillMight, mightGMLevel)); err != nil {
				return err
			}
			return poll(ctx, func() bool {
				e, ok := a.Game().World().Effect(skillMight)
				return ok && int32(e.Level) == mightGMLevel
			})
		}},
		Idle(settle),
		Step{"alice casts Might 1 again: the higher level stays", func(ctx context.Context) error {
			if _, err := a.Game().CastOn(ctx, a.ObjectID(), skillMight); err != nil {
				return err
			}
			if err := sleep(ctx, settle); err != nil {
				return err
			}
			e, _ := a.Game().World().Effect(skillMight)
			s.Note("Might after the second cast: level %d, %d s; %d icons", e.Level, e.Duration, len(a.Game().World().Effects()))
			return nil
		}},
		Idle(time.Second),
		Step{"alice logs out", a.Logout},
	)
}

func potion(ctx context.Context, s *Session) error {
	a := s.Player("alice")
	if err := s.Prepare(ctx, a, giveItem(a, itemHealingPotionM, 3), gmMoveTo(a, spotKaterina)); err != nil {
		return err
	}
	if err := s.Enter(ctx, a); err != nil {
		return err
	}
	var bottle testclient.Item
	drink := func(ctx context.Context) error { return a.Game().UseItem(ctx, bottle.ObjectID) }
	return s.Steps(ctx,
		Idle(time.Second),
		findItems(a, map[int32]*testclient.Item{itemHealingPotionM: &bottle}),
		Step{"alice drinks a Healing Potion", func(ctx context.Context) error {
			if err := drink(ctx); err != nil {
				return err
			}
			b, err := testclient.Await(ctx, a.Game(), func(b testclient.ShortBuffStatusUpdate) bool { return b.SkillID != 0 })
			if err == nil {
				s.Note("short buff: skill %d level %d, %d s", b.SkillID, b.Level, b.Duration)
			}
			return err
		}},
		Idle(settle),
		Step{"alice drinks another within the reuse: refused", func(ctx context.Context) error {
			if err := drink(ctx); err != nil {
				return err
			}
			m, err := testclient.Await[testclient.SystemMessage](ctx, a.Game(), nil)
			if err == nil {
				s.Note("refusal: system message %d", m.ID)
			}
			return err
		}},
		Long(potionLimit, Step{"the potion's effect ends", func(ctx context.Context) error {
			_, err := testclient.Await(ctx, a.Game(), func(b testclient.ShortBuffStatusUpdate) bool { return b.SkillID == 0 })
			return err
		}}),
		Idle(time.Second),
		Step{"alice logs out", a.Logout},
	)
}

func soulshot(ctx context.Context, s *Session) error {
	a := s.Player("alice")
	if err := s.Prepare(ctx, a, giveItem(a, itemSoulshotNG, 100), gmMoveTo(a, keltirField)); err != nil {
		return err
	}
	if err := s.Enter(ctx, a); err != nil {
		return err
	}
	var (
		shots  testclient.Item
		target testclient.Object
	)
	return s.Steps(ctx,
		Idle(time.Second),
		findItems(a, map[int32]*testclient.Item{itemSoulshotNG: &shots}),
		Step{"alice uses a soulshot", func(ctx context.Context) error {
			if err := a.Game().UseItem(ctx, shots.ObjectID); err != nil {
				return err
			}
			me := a.ObjectID()
			m, err := testclient.Await(ctx, a.Game(), func(m testclient.MagicSkillUse) bool { return m.CasterID == me })
			if err == nil {
				s.Note("soulshot animation: skill %d", m.SkillID)
			}
			return err
		}},
		Idle(settle),
		Step{"alice turns on automatic soulshots", func(ctx context.Context) error {
			_, err := a.Game().AutoSoulShot(ctx, itemSoulshotNG, true)
			return err
		}},
		Long(walkLimit, Step{"alice finds a Bearded Keltir", func(ctx context.Context) error {
			var err error
			target, err = huntMonster(ctx, a, npcBeardedKeltir, keltirField)
			return err
		}}),
		Long(walkLimit, Step{"alice attacks it", func(ctx context.Context) error {
			_, err := a.Game().Attack(ctx, target.ObjectID)
			return err
		}}),
		Long(fightLimit, Step{"alice fights until it dies", func(ctx context.Context) error {
			n, err := fightUntilDead(ctx, a, target.ObjectID)
			s.Note("the fight took %d more swings; %d soulshots left", n, a.Game().World().ItemCount(itemSoulshotNG))
			return err
		}}),
		Step{"alice waits out the combat stance", func(ctx context.Context) error { return sleep(ctx, stanceWait) }},
		Step{"alice turns off automatic soulshots", func(ctx context.Context) error {
			_, err := a.Game().AutoSoulShot(ctx, itemSoulshotNG, false)
			return err
		}},
		Idle(time.Second),
		Step{"alice logs out", a.Logout},
	)
}

func newbieBuffs(ctx context.Context, s *Session) error {
	a := s.Player("alice")
	a.Template = HumanMystic
	if err := s.Prepare(ctx, a, setLevel(a, newbieLevel), gmMoveTo(a, spotGuide)); err != nil {
		return err
	}
	if err := s.Enter(ctx, a); err != nil {
		return err
	}
	var page testclient.NpcHtmlMessage
	return s.Steps(ctx,
		Idle(time.Second),
		Step{"alice talks to the Newbie Guide", func(ctx context.Context) error {
			var err error
			page, err = talkTo(ctx, s, a, npcNewbieGuide)
			return err
		}},
		Step{"alice opens the support magic page", func(ctx context.Context) error {
			if err := openLink(ctx, a, page, "SupportMagic.htm"); err != nil {
				return err
			}
			var err error
			page, err = testclient.Await[testclient.NpcHtmlMessage](ctx, a.Game(), nil)
			return err
		}},
		Step{"alice asks for support magic", func(ctx context.Context) error {
			if err := openLink(ctx, a, page, "_SupportMagic"); err != nil {
				return err
			}
			guide := page.ObjectID
			m, err := testclient.Await(ctx, a.Game(), func(m testclient.MagicSkillUse) bool { return m.CasterID == guide })
			if err == nil {
				s.Note("the guide casts skill %d level %d", m.SkillID, m.Level)
			}
			return err
		}},
		Step{"alice gets the buffs", func(ctx context.Context) error {
			w := a.Game().World()
			if err := poll(ctx, func() bool { return len(w.Effects()) > 0 }); err != nil {
				return errors.Join(errors.New("no buff icon"), err)
			}
			if err := sleep(ctx, 2*time.Second); err != nil {
				return err
			}
			for _, e := range w.Effects() {
				s.Note("buff %d level %d, %d s", e.SkillID, e.Level, e.Duration)
			}
			return nil
		}},
		Idle(time.Second),
		Step{"alice logs out", a.Logout},
	)
}
