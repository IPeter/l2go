package game

import (
	"errors"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// messages returns the texts of the free-text system messages in ps.
func messages(ps []testclient.Packet) []string {
	var out []string
	for _, m := range only[testclient.SystemMessage](ps) {
		if m.ID == testclient.SysMsgText {
			out = append(out, m.Texts()...)
		}
	}
	return out
}

func TestPacketHandlers(t *testing.T) {
	var got []string
	e := newEnv(t, nil, func(s *Server) {
		if err := s.Register(0x77, func(p *Player, body []byte) { got = append(got, p.Name()+":"+string(body)) }); err != nil {
			t.Fatalf("Register: %v", err)
		}
		if err := s.Register(0x77, func(*Player, []byte) {}); !errors.Is(err, ErrDuplicate) {
			t.Errorf("second Register of 0x77 = %v, want ErrDuplicate", err)
		}
		if err := s.Register(clientpackets.OpAppearing, func(*Player, []byte) {}); !errors.Is(err, ErrDuplicate) {
			t.Errorf("Register of the core's Appearing = %v, want ErrDuplicate", err)
		}
		if err := s.RegisterEx(0x99, func(p *Player, body []byte) { got = append(got, "ex:"+string(body)) }); err != nil {
			t.Fatalf("RegisterEx: %v", err)
		}
	})
	e.createCharacter("hero", "Hero", classHumanFighter, storage.AccessLevelPlayer)
	sess := newSession("hero")
	if err := e.srv.SelectCharacter(e.ctx, sess, 0); err != nil {
		t.Fatalf("SelectCharacter: %v", err)
	}

	// Before EnterWorld in-world packets are dropped.
	e.srv.Handle(sess, 0x77, []byte("early"))
	e.sync()
	e.srv.EnterWorld(sess)

	body := []byte("abc")
	e.srv.Handle(sess, 0x77, body)
	body[0] = 'X' // the session may reuse its buffer
	e.srv.HandleEx(sess, 0x99, []byte("def"))
	e.srv.Handle(sess, 0x78, []byte("no handler"))
	e.srv.HandleEx(sess, 0x9a, []byte("no handler"))
	e.srv.Handle(newSession("nobody"), 0x77, []byte("no player"))
	var calls []string
	e.onLoop(func() { calls = slices.Clone(got) })
	if want := []string{"Hero:abc", "ex:def"}; !slices.Equal(calls, want) {
		t.Errorf("handlers got %q, want %q", calls, want)
	}
}

func TestGMCommands(t *testing.T) {
	type call struct {
		player string
		args   []string
	}
	var calls []call
	record := func(p *Player, args []string) { calls = append(calls, call{p.Name(), args}) }
	e := newEnv(t, nil, func(s *Server) {
		if err := s.RegisterGMCommand("echo", storage.AccessLevelAdmin, record); err != nil {
			t.Fatalf("RegisterGMCommand: %v", err)
		}
		if err := s.RegisterGMCommand("master", 100, record); err != nil {
			t.Fatalf("RegisterGMCommand: %v", err)
		}
		if err := s.RegisterGMCommand("echo", storage.AccessLevelAdmin, record); !errors.Is(err, ErrDuplicate) {
			t.Errorf("second RegisterGMCommand = %v, want ErrDuplicate", err)
		}
		if err := s.RegisterGMCommand("two words", storage.AccessLevelAdmin, record); err == nil {
			t.Error("RegisterGMCommand accepted a name with a space")
		}
	})
	e.createCharacter("gm", "Gm", classHumanFighter, storage.AccessLevelAdmin)
	e.createCharacter("player", "Player", classHumanFighter, storage.AccessLevelPlayer)
	gm, pl := newSession("gm"), newSession("player")
	e.enter(gm, 0)
	e.enter(pl, 0)
	gm.take(t)
	pl.take(t)

	for _, tc := range []struct {
		name     string
		sess     *fakeSession
		line     string
		wantCall []call
		wantMsg  []string
	}{
		{"runs with its arguments", gm, "echo a  b", []call{{"Gm", []string{"a", "b"}}}, nil},
		{"without arguments", gm, "echo", []call{{"Gm", nil}}, nil},
		{"unknown", gm, "nope x", nil, []string{"The command 'nope' does not exist!"}},
		{"names match exactly", gm, "Echo", nil, []string{"The command 'Echo' does not exist!"}},
		{"access level too low", gm, "master", nil, []string{"You don't have the access rights to use this command!"}},
		{"not a GM", pl, "echo a", nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e.onLoop(func() { calls = nil })
			e.srv.Handle(tc.sess, clientpackets.OpSendBypassBuildCmd, stringBody(tc.line))
			var got []call
			e.onLoop(func() { got = slices.Clone(calls) })
			if !slices.EqualFunc(got, tc.wantCall, func(a, b call) bool { return a.player == b.player && slices.Equal(a.args, b.args) }) {
				t.Errorf("calls %+v, want %+v", got, tc.wantCall)
			}
			if msgs := messages(tc.sess.take(t)); !slices.Equal(msgs, tc.wantMsg) {
				t.Errorf("messages %q, want %q", msgs, tc.wantMsg)
			}
		})
	}
	if names := e.srv.GMCommands(); !slices.Equal(names, []string{"echo", "master"}) {
		t.Errorf("GMCommands() = %q", names)
	}
}

func TestUserCommands(t *testing.T) {
	e := newEnv(t, nil, func(s *Server) {
		if err := s.RegisterUserCommand(clientpackets.UserCmdLoc, func(p *Player) { p.SendMessage("you are here") }); err != nil {
			t.Fatalf("RegisterUserCommand: %v", err)
		}
		if err := s.RegisterUserCommand(clientpackets.UserCmdLoc, func(*Player) {}); !errors.Is(err, ErrDuplicate) {
			t.Errorf("second RegisterUserCommand = %v, want ErrDuplicate", err)
		}
	})
	e.createCharacter("gm", "Gm", classHumanFighter, storage.AccessLevelAdmin)
	e.createCharacter("player", "Player", classHumanFighter, storage.AccessLevelPlayer)
	gm, pl := newSession("gm"), newSession("player")
	e.enter(gm, 0)
	e.enter(pl, 0)
	gm.take(t)
	pl.take(t)

	e.srv.Handle(pl, clientpackets.OpBypassUserCmd, int32Body(clientpackets.UserCmdLoc))
	e.srv.Handle(pl, clientpackets.OpBypassUserCmd, int32Body(99))
	e.sync()
	if msgs := messages(pl.take(t)); !slices.Equal(msgs, []string{"you are here"}) {
		t.Errorf("player got %q, want the /loc answer only", msgs)
	}
	// An unknown command tells a GM, as Mobius does.
	e.srv.Handle(gm, clientpackets.OpBypassUserCmd, int32Body(99))
	e.sync()
	if msgs := messages(gm.take(t)); !slices.Equal(msgs, []string{"User commandID 99 not implemented yet."}) {
		t.Errorf("GM got %q", msgs)
	}
}

func TestEnterWorldRequestsAreAnswered(t *testing.T) {
	e := newEnv(t, nil)
	e.createCharacter("hero", "Hero", classHumanFighter, storage.AccessLevelPlayer)
	sess := newSession("hero")
	e.enter(sess, 0)
	sess.take(t)

	e.srv.HandleEx(sess, clientpackets.ExOpRequestManorList, nil)
	e.srv.Handle(sess, clientpackets.OpRequestQuestList, nil)
	e.srv.Handle(sess, clientpackets.OpRequestSkillCoolTime, nil)
	e.sync()
	ps := sess.take(t)
	checkNames(t, "answers", ps, "ExSendManorList", "QuestList", "SkillCoolTime")
	manor, ok := ps[0].(testclient.Unknown)
	if sub, isEx := manor.ExOpcode(); !ok || !isEx || sub != serverpackets.ExOpSendManorList {
		t.Errorf("manor answer %v, want ExSendManorList", ps[0])
	}
	if q := ps[1].(testclient.QuestList); len(q.Quests) != 0 {
		t.Errorf("QuestList has %d quests, want none", len(q.Quests))
	}
}

// testNpc is a minimal NPC in the world. Its world.Object ID method (the
// object ID) and scripting.NpcRef's (the template ID) clash, so the
// script ref is a separate type, as a real NPC needs too.
type testNpc struct {
	world.Base
}

// testNpcRef is the NPC as scripts see it.
type testNpcRef struct {
	npc *testNpc
	id  int
}

func (r testNpcRef) ObjectID() int32 { return int32(r.npc.ID()) }
func (r testNpcRef) ID() int         { return r.id }
func (r testNpcRef) Name() string    { return "Newbie Guide" }

func TestScriptOutputNpcSay(t *testing.T) {
	e := newEnv(t, nil)
	e.createCharacter("hero", "Hero", classHumanFighter, storage.AccessLevelPlayer)
	sess := newSession("hero")
	p := e.enter(sess, 0)
	sess.take(t)

	npc := &testNpc{}
	ref := testNpcRef{npc: npc, id: 30598}
	e.onLoop(func() {
		id, err := e.srv.World().IDs().Next()
		if err != nil {
			t.Errorf("IDs: %v", err)
			return
		}
		pos := p.Position()
		pos.X += 50
		npc.Base = world.NewBase(id, pos)
		if err := e.srv.AddObject(npc); err != nil {
			t.Errorf("AddObject: %v", err)
		}
		e.srv.ScriptOutput().NpcSay(ref, "Welcome!")
	})
	ps := sess.take(t)
	checkNames(t, "after npc:say", ps, "NpcSay")
	say := ps[0].(testclient.NpcSay)
	if say.ObjectID != ref.ObjectID() || say.NpcID() != 30598 || say.Text != "Welcome!" || say.Type != testclient.ChatType(serverpackets.ChatGeneral) {
		t.Errorf("NpcSay %+v", say)
	}

	// An NPC out of the world says nothing.
	e.onLoop(func() {
		e.srv.RemoveObject(npc)
		e.srv.ScriptOutput().NpcSay(ref, "Gone")
	})
	if ps := sess.take(t); len(only[testclient.NpcSay](ps)) != 0 {
		t.Errorf("an NPC out of the world spoke: %q", testclient.Names(ps))
	}
}

func TestScriptOutputBeforeNew(t *testing.T) {
	out := NewScriptOutput()
	// Not connected to a server yet: nothing happens, nothing breaks.
	npc := &testNpc{Base: world.NewBase(world.FirstRuntimeID, world.Position{})}
	out.NpcSay(testNpcRef{npc: npc, id: 1}, "hello")
}

func TestFeaturesRunInNameOrder(t *testing.T) {
	featuresMu.Lock()
	saved := features
	features = nil
	featuresMu.Unlock()
	t.Cleanup(func() {
		featuresMu.Lock()
		features = saved
		featuresMu.Unlock()
	})

	var order []string
	registerFeature("zones", func(*Server) error { order = append(order, "zones"); return nil })
	registerFeature("chat", func(s *Server) error {
		order = append(order, "chat")
		return s.Register(0x38, func(*Player, []byte) {})
	})
	newEnv(t, nil)
	if !slices.Equal(order, []string{"chat", "zones"}) {
		t.Errorf("features ran in the order %q, want [chat zones]", order)
	}

	// A feature that fails stops New, naming the feature.
	registerFeature("broken", func(*Server) error { return errors.New("no data") })
	cfg := config.Default()
	if _, err := New(Options{Config: cfg, ServerID: cfg.GameServers[0].ID, Data: testGameData(t), Characters: newRecordingStore()}); err == nil {
		t.Error("New with a failing feature succeeded")
	}
}
