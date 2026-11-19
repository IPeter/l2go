package game

import (
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// chatBody encodes Say2: the text, the type and, for a whisper, the target.
func chatBody(typ serverpackets.ChatType, text, target string) []byte {
	w := packets.NewWriter(8 + 2*len(text) + 2*len(target))
	w.WriteString(text)
	w.WriteInt32(int32(typ))
	if typ == serverpackets.ChatWhisper {
		w.WriteString(target)
	}
	return w.Bytes()
}

// chatSend sends Say2 for a session and waits for the loop.
func chatSend(e *testEnv, sess *fakeSession, typ serverpackets.ChatType, text, target string) {
	e.srv.Handle(sess, clientpackets.OpSay2, chatBody(typ, text, target))
	e.sync()
}

// chatTrio enters Alice (with the given access level), Bob 1,200 units east
// of her and Carol 1,300 units east: they see each other, and Carol is out
// of the general chat range (1,250). Their packets so far are dropped.
func chatTrio(t *testing.T, e *testEnv, aliceAccess storage.AccessLevel) (sa, sb, sc *fakeSession, pa, pb, pc *Player) {
	t.Helper()
	e.createCharacter("alice", "Alice", classHumanFighter, aliceAccess)
	e.createCharacter("bob", "Bob", classHumanFighter, storage.AccessLevelPlayer)
	e.createCharacter("carol", "Carol", classHumanFighter, storage.AccessLevelPlayer)
	sa, sb, sc = newSession("alice"), newSession("bob"), newSession("carol")
	pa, pb, pc = e.enter(sa, 0), e.enter(sb, 0), e.enter(sc, 0)
	e.onLoop(func() {
		home := pa.Position()
		for p, dx := range map[*Player]int32{pb: 1200, pc: 1300} {
			if err := e.srv.MoveObject(p, walkOffset(home, dx, 0, 0)); err != nil {
				t.Errorf("MoveObject: %v", err)
			}
		}
	})
	walkSettle(t, e, sa, sb, sc)
	return sa, sb, sc, pa, pb, pc
}

// chatLine is the CreatureSay a message of p shows as.
func chatLine(p *Player, typ serverpackets.ChatType, name, text string) testclient.CreatureSay {
	return testclient.CreatureSay{ObjectID: p.ObjectID(), Type: testclient.ChatType(typ), Name: name, Text: text}
}

// chatExpect fails unless the CreatureSay packets a session got since the
// last take are want. The other packets are dropped.
func chatExpect(t *testing.T, who string, sess *fakeSession, want ...testclient.CreatureSay) {
	t.Helper()
	if got := only[testclient.CreatureSay](sess.take(t)); !slices.Equal(got, want) {
		t.Errorf("%s heard %+v, want %+v", who, got, want)
	}
}

// chatMessages returns the free-text system messages a session got, and
// fails if it heard chat.
func chatMessages(t *testing.T, who string, sess *fakeSession) []string {
	t.Helper()
	ps := sess.take(t)
	if said := only[testclient.CreatureSay](ps); len(said) != 0 {
		t.Errorf("%s heard %+v", who, said)
	}
	return messages(ps)
}

// General chat reaches the players within 1,250 units and comes back to the
// speaker.
func TestChatGeneralRange(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, sc, pa, _, _ := chatTrio(t, e, storage.AccessLevelPlayer)
	chatSend(e, sa, serverpackets.ChatGeneral, "hello", "")
	line := chatLine(pa, serverpackets.ChatGeneral, "Alice", "hello")
	chatExpect(t, "Alice", sa, line)
	chatExpect(t, "Bob, 1,200 away", sb, line)
	chatExpect(t, "Carol, 1,300 away", sc)
}

// Shout and trade chat follow their mode (Mobius GlobalChat and TradeChat):
// "on" keeps them in the speaker's map region, "global" sends them to
// everyone, "gm" lets only GMs speak (in their region), "off" drops them.
func TestChatShoutAndTradeModes(t *testing.T) {
	for _, tc := range []struct {
		name              string
		mode              config.ChatMode
		gm                bool
		alice, bob, carol bool
	}{
		{"on", config.ChatModeOn, false, true, true, false},
		{"global", config.ChatModeGlobal, false, true, true, true},
		{"gm from a player", config.ChatModeGM, false, false, false, false},
		{"gm from a GM", config.ChatModeGM, true, true, true, false},
		{"off", config.ChatModeOff, true, false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var border int32
			e := newEnv(t, func(gs *config.GameServer) {
				gs.Chat.Shout, gs.Chat.Trade = tc.mode, tc.mode
			}, func(s *Server) {
				// Carol stands in another map region than Alice and Bob.
				s.movement.chatRegion = func(pos world.Position) int {
					if pos.X > border {
						return 2
					}
					return 1
				}
			})
			access := storage.AccessLevelPlayer
			if tc.gm {
				access = storage.AccessLevelAdmin
			}
			sa, sb, sc, pa, _, _ := chatTrio(t, e, access)
			e.onLoop(func() {
				border = pa.Position().X + 1250
				pa.Stats().SetLevel(chatTradeLevel)
			})
			for _, typ := range []serverpackets.ChatType{serverpackets.ChatShout, serverpackets.ChatTrade} {
				e.advance(time.Second) // past the flood protector of the global mode
				chatSend(e, sa, typ, "WTS sword", "")
				line := chatLine(pa, typ, "Alice", "WTS sword")
				for _, h := range []struct {
					name  string
					sess  *fakeSession
					hears bool
				}{{"Alice", sa, tc.alice}, {"Bob", sb, tc.bob}, {"Carol", sc, tc.carol}} {
					var want []testclient.CreatureSay
					if h.hears {
						want = append(want, line)
					}
					chatExpect(t, h.name+" (type "+strconv.Itoa(int(typ))+")", h.sess, want...)
				}
			}
		})
	}
}

// In the global mode a player below GM level speaks at most every 500 ms,
// shout and trade together (Mobius's flood protector).
func TestChatGlobalFloodProtection(t *testing.T) {
	e := newEnv(t, func(gs *config.GameServer) {
		gs.Chat.Shout, gs.Chat.Trade = config.ChatModeGlobal, config.ChatModeGlobal
	})
	sa, sb, sc, pa, _, _ := chatTrio(t, e, storage.AccessLevelPlayer)
	e.onLoop(func() { pa.Stats().SetLevel(chatTradeLevel) })

	chatSend(e, sa, serverpackets.ChatShout, "one", "")
	chatExpect(t, "Carol, the first shout", sc, chatLine(pa, serverpackets.ChatShout, "Alice", "one"))
	sa.take(t)
	sb.take(t)
	chatSend(e, sa, serverpackets.ChatShout, "two", "")
	if msgs := chatMessages(t, "Alice", sa); !slices.Equal(msgs, []string{"Do not spam shout channel."}) {
		t.Errorf("Alice got %q after a second shout", msgs)
	}
	chatSend(e, sa, serverpackets.ChatTrade, "three", "")
	if msgs := chatMessages(t, "Alice", sa); !slices.Equal(msgs, []string{"Do not spam trade channel."}) {
		t.Errorf("Alice got %q after a trade message", msgs)
	}
	chatExpect(t, "Bob, flooded", sb)

	e.advance(500 * time.Millisecond)
	chatSend(e, sa, serverpackets.ChatShout, "four", "")
	chatExpect(t, "Bob after 500 ms", sb, chatLine(pa, serverpackets.ChatShout, "Alice", "four"))
}

// Mobius lets only players of level 20 and up trade-chat. Without world
// data every player is in map region 0.
func TestChatTradeNeedsLevel20(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, sc, pa, _, _ := chatTrio(t, e, storage.AccessLevelPlayer)
	chatSend(e, sa, serverpackets.ChatTrade, "WTB", "")
	if msgs := chatMessages(t, "Alice", sa); !slices.Equal(msgs, []string{"Players can use Trade chat after Lv. 20."}) {
		t.Errorf("Alice got %q", msgs)
	}
	chatExpect(t, "Bob, level 1", sb)

	e.onLoop(func() { pa.Stats().SetLevel(chatTradeLevel) })
	chatSend(e, sa, serverpackets.ChatTrade, "WTB", "")
	line := chatLine(pa, serverpackets.ChatTrade, "Alice", "WTB")
	chatExpect(t, "Alice, level 20", sa, line)
	chatExpect(t, "Bob, level 20", sb, line)
	chatExpect(t, "Carol, level 20", sc, line)
}

// A whisper goes to the player of that name, in any case, and back to the
// speaker as "->Name"; an offline name gets "That player is not online."
func TestChatWhisper(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, sc, pa, _, _ := chatTrio(t, e, storage.AccessLevelPlayer)
	chatSend(e, sa, serverpackets.ChatWhisper, "psst", "bOB")
	chatExpect(t, "Bob", sb, chatLine(pa, serverpackets.ChatWhisper, "Alice", "psst"))
	chatExpect(t, "Alice", sa, chatLine(pa, serverpackets.ChatWhisper, "->Bob", "psst"))
	chatExpect(t, "Carol", sc)

	notOnline := func(what string) {
		t.Helper()
		ps := sa.take(t)
		checkNames(t, what, ps, "SystemMessage")
		if m := ps[0].(testclient.SystemMessage); m.ID != testclient.SysMsgPlayerNotOnline {
			t.Errorf("%s: system message %d, want %d", what, m.ID, testclient.SysMsgPlayerNotOnline)
		}
	}
	chatSend(e, sa, serverpackets.ChatWhisper, "anyone?", "Nobody")
	notOnline("a whisper to nobody")

	e.srv.Logout(sb)
	sa.take(t)
	chatSend(e, sa, serverpackets.ChatWhisper, "still there?", "Bob")
	notOnline("a whisper to a player who logged out")
}

// Players below GM level may send 105 characters, counted in UTF-16 units
// as Java does; longer messages get Mobius's system message. GMs may send
// more, and hero voice to everyone.
func TestChatLengthLimit(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, _, pa, _, _ := chatTrio(t, e, storage.AccessLevelPlayer)
	ok := strings.Repeat("a", 105)
	chatSend(e, sa, serverpackets.ChatGeneral, ok, "")
	chatExpect(t, "Bob, 105 characters", sb, chatLine(pa, serverpackets.ChatGeneral, "Alice", ok))
	sa.take(t)
	for _, text := range []string{strings.Repeat("a", 106), strings.Repeat("a", 104) + "😀"} {
		chatSend(e, sa, serverpackets.ChatGeneral, text, "")
		ps := sa.take(t)
		checkNames(t, "a message of 106 units", ps, "SystemMessage")
		if m := ps[0].(testclient.SystemMessage); m.ID != int32(serverpackets.SysMsgChatTooLong) {
			t.Errorf("system message %d, want %d", m.ID, serverpackets.SysMsgChatTooLong)
		}
		chatExpect(t, "Bob, too long", sb)
	}

	g := newEnv(t, nil)
	ga, gb, gc, gm, _, _ := chatTrio(t, g, storage.AccessLevelAdmin)
	long := strings.Repeat("a", 200)
	chatSend(g, ga, serverpackets.ChatGeneral, long, "")
	chatExpect(t, "Bob, a GM's long message", gb, chatLine(gm, serverpackets.ChatGeneral, "Alice", long))
	ga.take(t)
	chatSend(g, ga, serverpackets.ChatHeroVoice, "hear ye", "")
	voice := chatLine(gm, serverpackets.ChatHeroVoice, "Alice", "hear ye")
	chatExpect(t, "the GM, hero voice", ga, voice)
	chatExpect(t, "Bob, hero voice", gb, voice)
	chatExpect(t, "Carol, hero voice", gc, voice)
}

// An unknown chat type or an empty text, which no client sends, gets
// ActionFailed and disconnects the player with LeaveWorld after saving it,
// as in Mobius.
func TestChatBadMessagesDisconnect(t *testing.T) {
	for _, tc := range []struct {
		name string
		typ  serverpackets.ChatType
		text string
	}{
		{"an unknown chat type", 99, "hi"},
		{"an empty text", serverpackets.ChatGeneral, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv(t, nil)
			sa, sb, _, pa, _, _ := chatTrio(t, e, storage.AccessLevelPlayer)
			chatSend(e, sa, tc.typ, tc.text, "")
			if ps := sa.take(t); len(ps) == 0 || ps[0].PacketName() != "ActionFailed" {
				t.Errorf("Alice got %q, want ActionFailed first", testclient.Names(ps))
			}
			final, delay, closed := sa.closedWith(t)
			if _, ok := final.(testclient.LeaveWorld); !closed || !ok || delay != 0 {
				t.Errorf("closed %v with %v after %v, want LeaveWorld at once", closed, final, delay)
			}
			e.store.waitSave(t, pa.ObjectID())
			if e.inWorld(pa) {
				t.Error("Alice is still in the world")
			}
			checkNames(t, "Bob sees Alice go", sb.take(t), "DeleteObject")
		})
	}
}

// The Lua events.on_chat handlers may change a message or drop it; a text
// they make too long is cut to the limit.
func TestChatLuaHook(t *testing.T) {
	h := scripttest.LoadSource(t, map[string]string{"events/chat.lua": `
events.on_chat(function(ev)
  if ev.text == "drop me" then return false end
  if ev.text == "long" then return "` + strings.Repeat("b", 300) + `" end
  if ev.type == "whisper" then return "(" .. ev.target .. ") " .. ev.text end
  return (ev.text:gsub("darn", "d**n"))
end)
`})
	e := newEnv(t, nil, func(s *Server) { s.scripts = h.Engine() })
	sa, sb, _, pa, _, _ := chatTrio(t, e, storage.AccessLevelPlayer)

	chatSend(e, sa, serverpackets.ChatGeneral, "darn it", "")
	chatExpect(t, "Bob, a changed message", sb, chatLine(pa, serverpackets.ChatGeneral, "Alice", "d**n it"))
	sa.take(t)

	chatSend(e, sa, serverpackets.ChatGeneral, "drop me", "")
	chatExpect(t, "Alice, a dropped message", sa)
	chatExpect(t, "Bob, a dropped message", sb)

	chatSend(e, sa, serverpackets.ChatWhisper, "hi", "Bob")
	chatExpect(t, "Bob, a changed whisper", sb, chatLine(pa, serverpackets.ChatWhisper, "Alice", "(Bob) hi"))
	sa.take(t)

	chatSend(e, sa, serverpackets.ChatGeneral, "long", "")
	chatExpect(t, "Bob, a long reply", sb, chatLine(pa, serverpackets.ChatGeneral, "Alice", strings.Repeat("b", 105)))
}

// A message that links an item (char 8 ... ID=objectId ... char 8) passes
// only when the speaker has the item, and may be 500 characters long
// (Mobius parseAndPublishItem).
func TestChatItemLinks(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, _, pa, _, _ := chatTrio(t, e, storage.AccessLevelPlayer)
	var own int32
	e.onLoop(func() { own = pa.Inventory().Items()[0].ObjectID })
	link := func(id int32) string {
		return "look \bType=1 \tID=" + strconv.Itoa(int(id)) + " \tColor=0 \tTitle=\x1bSword\x1b\b here"
	}

	chatSend(e, sa, serverpackets.ChatGeneral, link(own), "")
	chatExpect(t, "Bob, Alice's item", sb, chatLine(pa, serverpackets.ChatGeneral, "Alice", link(own)))
	sa.take(t)
	for _, text := range []string{
		link(own + 999),
		"broken \bID= here\b",
		"no end \bID=" + strconv.Itoa(int(own)) + " here",
		"no id \b here \b",
	} {
		chatSend(e, sa, serverpackets.ChatGeneral, text, "")
		chatExpect(t, "Alice, "+strconv.Quote(text), sa)
		chatExpect(t, "Bob, "+strconv.Quote(text), sb)
	}

	long := link(own) + strings.Repeat("a", 200)
	chatSend(e, sa, serverpackets.ChatGeneral, long, "")
	chatExpect(t, "Bob, a long message with a link", sb, chatLine(pa, serverpackets.ChatGeneral, "Alice", long))
}

// Party, clan, alliance, hero and petition chat answer as Mobius does for a
// player without a party, clan, alliance, hero status or petition; the GM
// channel has no handler.
func TestChatChannelsWithoutGroups(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, _, _, _, _ := chatTrio(t, e, storage.AccessLevelPlayer)
	for _, tc := range []struct {
		typ serverpackets.ChatType
		msg string
	}{
		{serverpackets.ChatParty, "You are not in a party."},
		{serverpackets.ChatClan, "You are not in a clan."},
		{serverpackets.ChatAlliance, "You are not in an alliance."},
		{serverpackets.ChatHeroVoice, "Only Heroes can enter the Hero channel."},
	} {
		chatSend(e, sa, tc.typ, "hi", "")
		if msgs := chatMessages(t, "Alice", sa); !slices.Equal(msgs, []string{tc.msg}) {
			t.Errorf("type %d: Alice got %q, want %q", tc.typ, msgs, tc.msg)
		}
		chatExpect(t, "Bob", sb)
	}

	chatSend(e, sa, serverpackets.ChatPetitionPlayer, "help", "")
	ps := sa.take(t)
	checkNames(t, "petition chat", ps, "SystemMessage")
	if m := ps[0].(testclient.SystemMessage); m.ID != int32(chatSysMsgNoPetition) {
		t.Errorf("system message %d, want %d", m.ID, chatSysMsgNoPetition)
	}

	chatSend(e, sa, serverpackets.ChatGM, "hi", "")
	checkNames(t, "the GM channel", sa.take(t))
	chatExpect(t, "Bob", sb)
}

// chatTestWorld is the world data of data/, loaded once.
var chatTestWorld = sync.OnceValues(func() (*data.World, error) {
	return data.LoadWorld(filepath.Join("..", "..", "data"))
})

// Shout and trade chat stay in Mobius's map region (getMapRegionLocId):
// Talking Island Town is 910, Gludio Castle Town 912. Without world data
// every position is in region 0.
func TestChatMapRegion(t *testing.T) {
	ti := world.Position{X: -71338, Y: 258271, Z: -3104}
	var s Server
	if got := s.chatMapRegion(ti); got != 0 {
		t.Errorf("region without world data %d, want 0", got)
	}
	w, err := chatTestWorld()
	if err != nil {
		t.Fatalf("load the world data: %v", err)
	}
	s.worldData = w
	for _, tc := range []struct {
		name string
		pos  world.Position
		want int
	}{
		{"the Talking Island start", ti, 910},
		{"Talking Island village", world.Position{X: -84000, Y: 243000, Z: -3700}, 910},
		{"Gludio", gludio, 912},
	} {
		if got := s.chatMapRegion(tc.pos); got != tc.want {
			t.Errorf("%s: region %d, want %d", tc.name, got, tc.want)
		}
	}
}
