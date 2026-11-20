package scenario

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

// The Milestone 6 scenarios (Plan 09, "Test strategy"): a kill, a death and
// a restart, and the swing samples of refdiff -sample. Fights are random:
// the recording package collapses the swings of a fight into one Fight
// entry and hides random damage, HP and loot, and the scenarios ignore
// positions, which follow where the monster happened to stand.

// Time limits of the long steps.
const (
	// walkLimit covers a walk across the keltir field (about 4,500 units
	// at a level 1 run speed of 115) plus the first swing.
	walkLimit = 75 * time.Second
	// fightLimit covers a level 1 fight to the death.
	fightLimit = 2 * time.Minute
	// swingTime is the time budget of one swing of a Sample scenario,
	// walks to the next monster included.
	swingTime = 3 * time.Second
)

func killMonster(ctx context.Context, s *Session) error {
	a := s.Player("alice")
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
		Long(walkLimit, Step{"alice attacks it", func(ctx context.Context) error {
			_, err := a.Game().Attack(ctx, target.ObjectID)
			return err
		}}),
		Long(fightLimit, Step{"alice fights until it dies", func(ctx context.Context) error {
			n, err := fightUntilDead(ctx, a, target.ObjectID)
			s.Note("the fight took %d more swings", n)
			return err
		}}),
		// Exp, drops and the corpse's decay (7 s) come meanwhile.
		Step{"alice waits out the combat stance", func(ctx context.Context) error { return sleep(ctx, stanceWait) }},
		Step{"alice logs out", a.Logout},
	)
}

func pvpDeathRestart(ctx context.Context, s *Session) error {
	a, b := s.Player("alice"), s.Player("bob")
	if err := enterBoth(ctx, s, a, b); err != nil {
		return err
	}
	aliceID := a.ObjectID()
	return s.Steps(ctx,
		Long(walkLimit, Step{"bob force-attacks alice", func(ctx context.Context) error {
			_, err := b.Game().Attack(ctx, aliceID)
			if errors.Is(err, context.DeadlineExceeded) {
				err = fmt.Errorf("%w (the start point may be a peace zone)", err)
			}
			return err
		}}),
		Long(fightLimit, Step{"bob fights until alice dies", func(ctx context.Context) error {
			n, err := fightUntilDead(ctx, b, aliceID)
			s.Note("the fight took %d more swings", n)
			return err
		}}),
		Step{"alice sees her death", func(ctx context.Context) error {
			return poll(ctx, func() bool { return a.Game().World().Me().Dead })
		}},
		Idle(2*time.Second),
		Step{"alice restarts in the village", func(ctx context.Context) error {
			res, err := a.Game().RestartPoint(ctx, testclient.RestartVillage)
			if err != nil {
				return err
			}
			t, u := res.Teleport, res.UserInfo
			hp := 0
			if u.MaxHP > 0 {
				hp = int(u.HP) * 100 / int(u.MaxHP)
			}
			s.Note("restart point %d %d %d with %d%% HP", t.X, t.Y, t.Z, hp)
			return nil
		}},
		Step{"both wait out the combat stance", func(ctx context.Context) error { return sleep(ctx, stanceWait) }},
		Step{"alice logs out", a.Logout},
		Step{"bob logs out", b.Logout},
	)
}

// sampleSwings runs a Sample scenario: alice (with the starting sword, or
// unarmed) swings at Bearded Keltirs until Session.Swings attacks are
// recorded. A dead alice restarts in the village and walks back.
func sampleSwings(unarmed bool) func(ctx context.Context, s *Session) error {
	return func(ctx context.Context, s *Session) error {
		a := s.Player("alice")
		if err := s.Enter(ctx, a); err != nil {
			return err
		}
		steps := []Step{Idle(time.Second)}
		if unarmed {
			steps = append(steps, Step{"alice takes off the sword", func(ctx context.Context) error {
				sword, err := inventoryItem(a, itemSquireSword)
				if err != nil {
					return err
				}
				if err := a.Game().UnequipItem(ctx, testclient.PaperdollRHand); err != nil {
					return err
				}
				_, err = a.Game().AwaitItemChange(ctx, sword.ObjectID)
				return err
			}})
		}
		limit := time.Duration(s.Swings())*swingTime + walkLimit
		steps = append(steps, Long(limit, Step{fmt.Sprintf("alice swings %d times", s.Swings()), func(ctx context.Context) error {
			return swingAtMonsters(ctx, s, a, s.Swings())
		}}))
		return s.Steps(ctx, steps...)
	}
}

// swingAtMonsters makes p fight Bearded Keltirs one after the other until
// it swung n times.
func swingAtMonsters(ctx context.Context, s *Session, p *Player, n int) error {
	swings, kills, deaths := 0, 0, 0
	defer func() { s.Note("%d swings, %d kills, %d deaths", swings, kills, deaths) }()
	for swings < n {
		target, err := huntMonster(ctx, p, npcBeardedKeltir, keltirField)
		if err != nil {
			return err
		}
		if _, err := p.Game().Attack(ctx, target.ObjectID); err != nil {
			return err
		}
		swings++
		more, err := fightUntilDead(ctx, p, target.ObjectID)
		swings += more
		switch {
		case errors.Is(err, errDied):
			deaths++
			if err := sleep(ctx, 2*time.Second); err != nil {
				return err
			}
			if _, err := p.Game().RestartPoint(ctx, testclient.RestartVillage); err != nil {
				return err
			}
		case err != nil:
			return err
		default:
			kills++
		}
		// The loot and the exp come, and the next click waits for them.
		if err := sleep(ctx, settle); err != nil {
			return err
		}
		_, _ = p.Game().Drain()
	}
	return nil
}
