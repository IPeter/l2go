package scenario

import (
	"context"
	"fmt"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

// The Milestone 9 scenarios of playing together (Plan 12 Phase 9): a party
// that hunts and shares the exp, a trade, the friend list with its notices,
// and a clan founded at a village master with a second member.
//
// Plan 12 also names alliance-war. It is left out: an alliance needs a clan
// of level 5 and a war needs clan.membersForWar (15) members in each clan,
// which the GM setups of this framework (one session per role) cannot build
// the same way on both servers.

// Settings of the social scenarios.
const (
	// The trade: alice's Stems (store.go) for bob's adena.
	tradeStems     = 5
	tradeAdena     = 100
	friendMessage  = "refdiff friend message"
	clanFoundLevel = 10
	// npcBitz is the village master of Talking Island village.
	npcBitz int32 = 30026
)

// Spots next to Bitz, whose spawn is (-83326, 242964, -3720).
var (
	spotBitzAlice = testclient.Position{X: -83290, Y: 242990, Z: -3720}
	spotBitzBob   = testclient.Position{X: -83250, Y: 242990, Z: -3720}
	// spotKeltirBob is next to keltirField, where bob waits for alice.
	spotKeltirBob = testclient.Position{X: keltirField.X + 60, Y: keltirField.Y, Z: keltirField.Z}
)

// awaitExp waits until p's world model shows the exp message of a kill.
func awaitExp(ctx context.Context, p *Player) (int64, error) {
	var exp int64
	err := poll(ctx, func() bool {
		for _, m := range p.Game().World().SystemMessages() {
			if m.ID == testclient.SysMsgEarnedExpAndSP && len(m.Params) > 0 {
				exp = m.Params[0].Number
				return true
			}
		}
		return false
	})
	if err != nil {
		return 0, fmt.Errorf("%s got no exp: %w", p.Role, err)
	}
	return exp, nil
}

func partyHunt(ctx context.Context, s *Session) error {
	a, b := s.Player("alice"), s.Player("bob")
	if err := s.Prepare(ctx, a, gmMoveTo(a, keltirField)); err != nil {
		return err
	}
	if err := s.Prepare(ctx, b, gmMoveTo(b, spotKeltirBob)); err != nil {
		return err
	}
	if err := enterBoth(ctx, s, a, b); err != nil {
		return err
	}
	var target testclient.Object
	return s.Steps(ctx,
		Step{"alice invites bob with loot Random, bob accepts", func(ctx context.Context) error {
			_, err := testclient.FormParty(ctx, a.Game(), testclient.LootRandom, b.Game())
			return err
		}},
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
		Step{"both get their share of the exp", func(ctx context.Context) error {
			ae, err := awaitExp(ctx, a)
			if err != nil {
				return err
			}
			be, err := awaitExp(ctx, b)
			if err != nil {
				return err
			}
			s.Note("exp: alice %d, bob %d", ae, be)
			return nil
		}},
		Step{"alice waits out the combat stance", func(ctx context.Context) error { return sleep(ctx, stanceWait) }},
		Step{"alice leaves, the party disperses", func(ctx context.Context) error {
			if _, err := a.Game().LeaveParty(ctx); err != nil {
				return err
			}
			id := a.ObjectID()
			_, err := testclient.Await(ctx, b.Game(), func(d testclient.PartySmallWindowDelete) bool { return d.ObjectID == id })
			return err
		}},
		Idle(time.Second),
		Step{"alice logs out", a.Logout},
		Step{"bob logs out", b.Logout},
	)
}

func trade(ctx context.Context, s *Session) error {
	a, b := s.Player("alice"), s.Player("bob")
	if err := s.Prepare(ctx, a, giveItem(a, itemStem, 10), gmMoveTo(a, spotDropAlice)); err != nil {
		return err
	}
	if err := s.Prepare(ctx, b, giveItem(b, itemAdena, 1000), gmMoveTo(b, spotDropBob)); err != nil {
		return err
	}
	if err := enterBoth(ctx, s, a, b); err != nil {
		return err
	}
	var stems, adena testclient.Item
	err := s.Steps(ctx,
		findItems(a, map[int32]*testclient.Item{itemStem: &stems}),
		findItems(b, map[int32]*testclient.Item{itemAdena: &adena}),
		Step{"alice trades 5 Stems for 100 of bob's adena; both confirm", func(ctx context.Context) error {
			res, err := testclient.Trade(ctx, a.Game(), b.Game(),
				[]testclient.TradeOffer{{ObjectID: stems.ObjectID, Count: tradeStems}},
				[]testclient.TradeOffer{{ObjectID: adena.ObjectID, Count: tradeAdena}})
			if err == nil && (res.Requester.Result != testclient.TradeDoneFinished || res.Partner.Result != testclient.TradeDoneFinished) {
				err = fmt.Errorf("the trade ended with %+v", res)
			}
			return err
		}},
		Idle(time.Second),
	)
	if err != nil {
		return err
	}
	return leaveBoth(ctx, s, a, b)
}

func friends(ctx context.Context, s *Session) error {
	a, b := s.Player("alice"), s.Player("bob")
	if err := enterBoth(ctx, s, a, b); err != nil {
		return err
	}
	err := s.Steps(ctx,
		Step{"alice invites bob as a friend", func(ctx context.Context) error {
			if err := a.Game().InviteFriend(ctx, b.Char); err != nil {
				return err
			}
			_, err := b.Game().AwaitFriendInvite(ctx)
			return err
		}},
		Step{"bob accepts", func(ctx context.Context) error {
			if _, err := b.Game().AcceptFriendInvite(ctx); err != nil {
				return err
			}
			_, err := testclient.Await(ctx, a.Game(), func(f testclient.L2Friend) bool { return f.Action == testclient.FriendAdded })
			return err
		}},
		Step{"alice sends bob a friend message", func(ctx context.Context) error {
			if err := a.Game().SendFriendMessage(ctx, b.Char, friendMessage); err != nil {
				return err
			}
			_, err := testclient.Await(ctx, b.Game(), func(m testclient.L2FriendSay) bool { return m.Message == friendMessage })
			return err
		}},
		Idle(time.Second),
		Step{"bob logs out", b.Logout},
		Step{"alice sees bob go offline", func(ctx context.Context) error {
			_, err := testclient.Await(ctx, a.Game(), func(f testclient.FriendStatus) bool { return !f.Online })
			return err
		}},
	)
	if err != nil {
		return err
	}
	if err := s.Enter(ctx, b); err != nil {
		return err
	}
	err = s.Steps(ctx,
		Step{"alice gets bob's log-in notice", func(ctx context.Context) error {
			return awaitSysMsg(ctx, a, testclient.SysMsgFriendLoggedIn)
		}},
		Idle(time.Second),
		Step{"alice deletes bob from the friend list", func(ctx context.Context) error {
			if err := a.Game().DeleteFriend(ctx, b.Char); err != nil {
				return err
			}
			_, err := testclient.Await(ctx, b.Game(), func(f testclient.L2Friend) bool { return f.Action == testclient.FriendRemoved })
			return err
		}},
		Idle(time.Second),
	)
	if err != nil {
		return err
	}
	return leaveBoth(ctx, s, a, b)
}

// clanCreate founds alice's clan under alice's character name: a clan name
// may equal a character name, so the normalizer replaces it like her name
// and both servers compare alike.
func clanCreate(ctx context.Context, s *Session) error {
	a, b := s.Player("alice"), s.Player("bob")
	if err := s.Prepare(ctx, a, setLevel(a, clanFoundLevel), gmMoveTo(a, spotBitzAlice)); err != nil {
		return err
	}
	if err := s.Prepare(ctx, b, gmMoveTo(b, spotBitzBob)); err != nil {
		return err
	}
	if err := enterBoth(ctx, s, a, b); err != nil {
		return err
	}
	err := s.Steps(ctx,
		Step{"alice founds a clan at Bitz", func(ctx context.Context) error {
			npc, err := findNpc(ctx, a, npcBitz)
			if err != nil {
				return err
			}
			all, err := a.Game().CreateClan(ctx, npc.ObjectID, a.Char)
			if err == nil {
				s.Note("clan window: %d members", len(all.Members))
			}
			return err
		}},
		Idle(time.Second),
		Step{"alice invites bob, bob accepts", func(ctx context.Context) error {
			if _, err := testclient.JoinClan(ctx, a.Game(), b.Game(), testclient.PledgeMain); err != nil {
				return err
			}
			name := b.Char
			_, err := testclient.Await(ctx, a.Game(), func(p testclient.PledgeShowMemberListAdd) bool { return p.Name == name })
			return err
		}},
		Idle(time.Second),
	)
	if err != nil {
		return err
	}
	return leaveBoth(ctx, s, a, b)
}
