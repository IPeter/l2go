package scenario

import (
	"context"
	"errors"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

// scenarios lists every scenario, in the order -list shows them.
var scenarios = []Scenario{
	{
		Name:        "login-create-enter",
		Description: "log in, get the server list, create a Human Fighter, enter the world, idle 2 s, log out",
		Roles:       []string{"alice"},
		Run:         loginCreateEnter,
	},
	{
		Name:        "enter-two-players",
		Description: "two new characters enter at the same spot and see each other; one walks 500 units; both log out",
		Roles:       []string{"alice", "bob"},
		Run:         enterTwoPlayers,
	},
	{
		Name:        "chat",
		Description: "two players next to each other: general chat, shout, and a whisper each way",
		Roles:       []string{"alice", "bob"},
		Run:         chat,
	},
	{
		Name:        "restart",
		Description: "enter the world, restart to the character screen, enter again, log out",
		Roles:       []string{"alice"},
		Run:         restart,
	},
	{
		Name:        "npc-talk",
		Description: "enter in Talking Island, target the nearest NPC, talk to it, follow the first link of its page",
		Roles:       []string{"alice"},
		Run:         npcTalk,
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
		Step{"alice finds the nearest NPC", func(context.Context) error {
			w := a.Game().World()
			o, d, ok := NearestTalkable(w.Me().Pos, w.Npcs())
			if !ok {
				return errors.New("no NPC to talk to in sight")
			}
			npc = o
			s.Note("nearest NPC: template %d %q, %.0f units away", o.NpcID, o.Name, d)
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
