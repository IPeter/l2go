package scenario

import (
	"context"
	"errors"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

// Default ignores (Scenario.Ignore).
var (
	// npcNoise is what NPCs do on their own: the AI moves, turns and
	// animates them on each server's clock, so where an NPC stands when a
	// player first sees it, and its idle animations, differ from run to run
	// (the M3/M4 reference runs).
	npcNoise = []string{"NpcInfo.X", "NpcInfo.Y", "NpcInfo.Z", "NpcInfo.Heading", "SocialAction"}
	// fightNoise is what differs between two fights of the same kind: the
	// positions (they follow where the monster stood and walked), the
	// server's position corrections, and the length of the fight. Social
	// actions stay compared: the level-up animation is one.
	fightNoise = []string{
		"*.X", "*.Y", "*.Z", "*.DestX", "*.DestY", "*.DestZ",
		"*.TargetX", "*.TargetY", "*.TargetZ", "*.Heading",
		"ValidateLocation", "Fight.Swings", "Fight.Packets",
	}
)

// scenarios lists every scenario, in the order -list shows them.
var scenarios = []Scenario{
	{
		Name:        "login-create-enter",
		Description: "log in, get the server list, create a Human Fighter, enter the world, idle 2 s, log out",
		Roles:       []string{"alice"},
		Run:         loginCreateEnter,
		Ignore:      npcNoise,
	},
	{
		Name:        "enter-two-players",
		Description: "two new characters enter at the same spot and see each other; one walks 500 units; both log out",
		Roles:       []string{"alice", "bob"},
		Run:         enterTwoPlayers,
		Ignore:      npcNoise,
	},
	{
		Name:        "chat",
		Description: "two players next to each other: general chat, shout, and a whisper each way",
		Roles:       []string{"alice", "bob"},
		Run:         chat,
		Ignore:      npcNoise,
	},
	{
		Name:        "restart",
		Description: "enter the world, restart to the character screen, enter again, log out",
		Roles:       []string{"alice"},
		Run:         restart,
		Ignore:      npcNoise,
	},
	{
		Name:        "npc-talk",
		Description: "enter in Talking Island, target an NPC in reach (the lowest template), talk to it, follow the first link of its page",
		Roles:       []string{"alice"},
		Run:         npcTalk,
		Ignore:      npcNoise,
	},
	// Milestone 5: items and economy.
	{
		Name:        "items-equip",
		Description: "take off the starting sword, equip the dagger, the sword again (the dagger comes off), take off and put on the pants",
		Roles:       []string{"alice"},
		Run:         itemsEquip,
		Ignore:      npcNoise,
	},
	{
		Name:        "items-bow",
		Description: "GM setup: a bow and 100 wooden arrows; equip the bow (the arrows go on), take it off (they come off)",
		Roles:       []string{"alice"},
		Run:         itemsBow,
		Ignore:      npcNoise,
		GM:          true,
	},
	{
		Name:        "items-drop-pickup",
		Description: "GM setup: 1,000 adena for alice, both on fixed spots; alice drops 100 adena, bob picks them up",
		Roles:       []string{"alice", "bob"},
		Run:         itemsDropPickup,
		Ignore:      npcNoise,
		GM:          true,
	},
	{
		Name:        "shop-buy-sell",
		Description: "GM setup: 10,000 adena, next to Katerina (Talking Island grocer); open the buy list, buy arrows and potions, sell some back",
		Roles:       []string{"alice"},
		Run:         shopBuySell,
		Ignore:      npcNoise,
		GM:          true,
	},
	{
		Name:        "multisell-galladucci",
		Description: "GM setup: C and D crystals, next to Galladucci in Giran; open multisell 300974001, exchange the first entry",
		Roles:       []string{"alice"},
		Run:         multisellGalladucci,
		Ignore:      npcNoise,
		GM:          true,
	},
	{
		Name:        "warehouse-private",
		Description: "GM setup: adena and arrows, next to Wilford (Talking Island); deposit both with the fee, withdraw them",
		Roles:       []string{"alice"},
		Run:         warehousePrivate,
		Ignore:      npcNoise,
		GM:          true,
	},
	// Milestone 6: combat.
	{
		Name:        "kill-monster",
		Description: "walk to the Bearded Keltirs near the start, kill one, see exp, drops and the corpse vanish, log out after the combat stance",
		Roles:       []string{"alice"},
		Run:         killMonster,
		Ignore:      fightNoise,
	},
	{
		Name:        "pvp-death-restart",
		Description: "bob force-attacks alice at the start point until she dies (PvP flag, karma); alice restarts in the village",
		Roles:       []string{"alice", "bob"},
		Run:         pvpDeathRestart,
		Ignore:      fightNoise,
	},
	{
		Name:        "sample-sword",
		Description: "for -sample: a Human Fighter with the starting sword swings at Bearded Keltirs until -swings attacks are recorded",
		Roles:       []string{"alice"},
		Run:         sampleSwings(false),
		Ignore:      fightNoise,
		Sample:      true,
	},
	{
		Name:        "sample-unarmed",
		Description: "for -sample: the same unarmed (Mobius swings fists twice per attack)",
		Roles:       []string{"alice"},
		Run:         sampleSwings(true),
		Ignore:      fightNoise,
		Sample:      true,
	},
	// Milestone 7: skills.
	{
		Name:        "cast-wind-strike",
		Description: "GM setup: a Human Mystic in the Bearded Keltir field; cast Wind Strike at a keltir until it dies (at most 5 casts, with the reuse between them)",
		Roles:       []string{"alice"},
		Run:         castWindStrike,
		Ignore:      fightNoise,
		GM:          true,
	},
	{
		Name:        "learn-skill",
		Description: "GM setup: a level 7 Human Mystic with 1,000 SP next to Baulro (Talking Island); open the skills to learn, learn Wind Strike 2",
		Roles:       []string{"alice"},
		Run:         learnSkill,
		Ignore:      npcNoise,
		GM:          true,
	},
	{
		Name:        "buff-stack",
		Description: "GM setup: a level 10 Human Mystic with Might 1 and Shield 1; cast both on herself, a GM //buff puts Might 3 over Might 1, cast Might 1 again (level 3 stays)",
		Roles:       []string{"alice"},
		Run:         buffStack,
		Ignore:      npcNoise,
		GM:          true,
	},
	{
		Name:        "potion",
		Description: "GM setup: 3 Healing Potions; drink one (short buff, HP ticks), a second within the 10 s reuse is refused, wait until the effect ends",
		Roles:       []string{"alice"},
		Run:         potion,
		Ignore:      npcNoise,
		GM:          true,
	},
	{
		Name:        "soulshot",
		Description: "GM setup: 100 no-grade soulshots in the Bearded Keltir field; use one, turn on automatic soulshots, kill a keltir, turn them off",
		Roles:       []string{"alice"},
		Run:         soulshot,
		Ignore:      fightNoise,
		GM:          true,
	},
	{
		Name:        "newbie-buffs",
		Description: "GM setup: a level 8 Human Mystic (the account's first character) next to the Newbie Guide (Talking Island); ask for support magic and get the buffs",
		Roles:       []string{"alice"},
		Run:         newbieBuffs,
		Ignore:      npcNoise,
		GM:          true,
	},
}

func loginCreateEnter(ctx context.Context, s *Session) error {
	a := s.Player("alice")
	if err := s.Enter(ctx, a); err != nil {
		return err
	}
	return s.Steps(ctx,
		Idle(2*time.Second),
		Step{"alice logs out", a.Logout},
	)
}

// enterBoth brings a and b into the world, one after the other, and waits
// until each sees the other.
func enterBoth(ctx context.Context, s *Session, a, b *Player) error {
	if err := s.Enter(ctx, a); err != nil {
		return err
	}
	if err := s.Enter(ctx, b); err != nil {
		return err
	}
	return s.Steps(ctx,
		Step{a.Role + " sees " + b.Role, func(ctx context.Context) error { return waitSees(ctx, a, b) }},
		Step{b.Role + " sees " + a.Role, func(ctx context.Context) error { return waitSees(ctx, b, a) }},
		// The packets that come with an appearance (CharInfo,
		// RelationChanged, ...) settle before the next step.
		Idle(time.Second),
	)
}

// leaveBoth logs a out, waits until b sees it disappear, and logs b out.
func leaveBoth(ctx context.Context, s *Session, a, b *Player) error {
	aID := a.ObjectID()
	return s.Steps(ctx,
		Step{a.Role + " logs out", a.Logout},
		Step{b.Role + " sees " + a.Role + " leave", func(ctx context.Context) error { return awaitDelete(ctx, b, aID) }},
		Step{b.Role + " logs out", b.Logout},
	)
}

func enterTwoPlayers(ctx context.Context, s *Session) error {
	a, b := s.Player("alice"), s.Player("bob")
	if err := enterBoth(ctx, s, a, b); err != nil {
		return err
	}
	var walk time.Duration
	err := s.Steps(ctx,
		Step{"alice walks 500 units", func(ctx context.Context) error {
			me := a.Game().World().Me()
			walk = walkTime(me.UserInfo, walkDistance)
			_, err := a.Game().MoveTo(ctx, me.Pos.X+walkDistance, me.Pos.Y, me.Pos.Z)
			return err
		}},
		Step{"bob sees alice walk", func(ctx context.Context) error { return awaitMove(ctx, b, a.ObjectID()) }},
		Step{"both wait for the walk to end", func(ctx context.Context) error { return sleep(ctx, walk) }},
	)
	if err != nil {
		return err
	}
	return leaveBoth(ctx, s, a, b)
}

// Chat texts. They are fixed, so that both servers see the same messages.
const (
	chatGeneral      = "refdiff general chat"
	chatShout        = "refdiff shout"
	chatWhisperToBob = "refdiff whisper to bob"
	chatWhisperToA   = "refdiff whisper to alice"
)

// say is a step in which from sends text on channel typ and every player
// of to receives it (the sender's own echo included when it is in to).
func say(from *Player, typ testclient.ChatType, text string, to ...*Player) Step {
	return Step{from.Role + " says " + text, func(ctx context.Context) error {
		if err := from.Game().Say(ctx, typ, text); err != nil {
			return err
		}
		for _, p := range to {
			if err := awaitChat(ctx, p, typ, text); err != nil {
				return err
			}
		}
		return nil
	}}
}

// whisper is a step in which from whispers text to to: to receives it and
// from gets the echo.
func whisper(from, to *Player, text string) Step {
	return Step{from.Role + " whispers to " + to.Role, func(ctx context.Context) error {
		if err := from.Game().Whisper(ctx, to.Char, text); err != nil {
			return err
		}
		if err := awaitChat(ctx, to, testclient.ChatWhisper, text); err != nil {
			return err
		}
		return awaitChat(ctx, from, testclient.ChatWhisper, text)
	}}
}

func chat(ctx context.Context, s *Session) error {
	a, b := s.Player("alice"), s.Player("bob")
	if err := enterBoth(ctx, s, a, b); err != nil {
		return err
	}
	err := s.Steps(ctx,
		say(a, testclient.ChatGeneral, chatGeneral, a, b),
		say(a, testclient.ChatShout, chatShout, a, b),
		whisper(a, b, chatWhisperToBob),
		whisper(b, a, chatWhisperToA),
		Idle(time.Second),
	)
	if err != nil {
		return err
	}
	return leaveBoth(ctx, s, a, b)
}

func restart(ctx context.Context, s *Session) error {
	a := s.Player("alice")
	if err := s.Enter(ctx, a); err != nil {
		return err
	}
	return s.Steps(ctx,
		Idle(time.Second),
		Step{"alice restarts", a.Restart},
		Step{"alice selects the character again", a.Select},
		Step{"alice enters the world again", a.EnterWorld},
		Idle(time.Second),
		Step{"alice logs out", a.Logout},
	)
}

func npcTalk(ctx context.Context, s *Session) error {
	a := s.Player("alice")
	if err := s.Enter(ctx, a); err != nil {
		return err
	}
	var (
		npc  testclient.Object
		page testclient.NpcHtmlMessage
	)
	return s.Steps(ctx,
		// NPCs come with the enter-world sequence; let late ones arrive.
		Idle(time.Second),
		Step{"alice picks an NPC", func(context.Context) error {
			w := a.Game().World()
			o, d, ok := PickTalkable(w.Me().Pos, w.Npcs())
			if !ok {
				return errors.New("no NPC to talk to in sight")
			}
			npc = o
			s.Note("NPC: template %d %q, %.0f units away", o.NpcID, o.Name, d)
			return nil
		}},
		Step{"alice targets the NPC", func(ctx context.Context) error {
			_, err := a.Game().Target(ctx, npc.ObjectID)
			return err
		}},
		Step{"alice talks to the NPC", func(ctx context.Context) error {
			var err error
			page, err = a.Game().Talk(ctx, npc.ObjectID)
			return err
		}},
		Step{"alice follows the first link", func(ctx context.Context) error {
			return followFirstLink(ctx, s, a, page)
		}},
		Idle(time.Second),
		Step{"alice logs out", a.Logout},
	)
}

// followFirstLink clicks the first link of page and waits for the answer:
// the next page, or a teleport, which it finishes the way the client does.
func followFirstLink(ctx context.Context, s *Session, p *Player, page testclient.NpcHtmlMessage) error {
	l, ok := FirstLink(page.HTML)
	if !ok {
		s.Note("the NPC's page has no links")
		return nil
	}
	s.Note("first link: %s", l)
	gc := p.Game()
	if err := l.Send(ctx, gc); err != nil {
		return err
	}
	me := p.ObjectID()
	answer, err := gc.WaitFor(ctx, func(pk testclient.Packet) bool {
		switch v := pk.(type) {
		case testclient.NpcHtmlMessage:
			return true
		case testclient.TeleportToLocation:
			return v.ObjectID == me
		}
		return false
	})
	if err != nil {
		return err
	}
	if _, ok := answer.(testclient.TeleportToLocation); ok {
		_, err := gc.Appearing(ctx)
		return err
	}
	return nil
}
