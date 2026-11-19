package game

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// socialRequest sends RequestSocialAction from sess and waits for it.
func socialRequest(e *testEnv, sess *fakeSession, id int32) {
	e.srv.Handle(sess, clientpackets.OpRequestSocialAction, int32Body(id))
	e.sync()
}

func TestSocialAction(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, pa, _ := twoPlayers(t, e)
	socialRequest(e, sa, 4)
	want := testclient.SocialAction{ObjectID: pa.ObjectID(), ActionID: 4}
	for who, sess := range map[string]*fakeSession{"Alice": sa, "Bob": sb} {
		ps := sess.take(t)
		checkNames(t, who+" after Alice's bow", ps, "SocialAction")
		if got := ps[0].(testclient.SocialAction); got != want {
			t.Errorf("%s got %+v, want %+v", who, got, want)
		}
	}
}

func TestSocialActionNeedsAnIdlePlayer(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, pa, _ := twoPlayers(t, e)
	for _, tc := range []struct {
		name  string
		set   func()
		reset func()
	}{
		{"sitting", func() { pa.Sitting = true }, func() { pa.Sitting = false }},
		{"walking", func() {
			pos := pa.Position()
			pa.Move = &MoveState{Origin: pos, Destination: world.Position{X: pos.X + 500, Y: pos.Y, Z: pos.Z}, Start: e.srv.Now(), Speed: 120, Running: true}
		}, func() { pa.Move = nil }},
	} {
		e.onLoop(tc.set)
		socialRequest(e, sa, 2)
		if ps := slices.Concat(sa.take(t), sb.take(t)); len(only[testclient.SocialAction](ps)) != 0 {
			t.Errorf("%s: a social action was played", tc.name)
		}
		e.onLoop(tc.reset)
	}
	socialRequest(e, sa, 13)
	if n := len(only[testclient.SocialAction](sa.take(t))); n != 1 {
		t.Errorf("idle again: %d social actions, want 1", n)
	}
}

func TestIllegalSocialActionKicks(t *testing.T) {
	e := newEnv(t, nil)
	e.createCharacter("alice", "Alice", classHumanFighter, storage.AccessLevelPlayer)
	e.createCharacter("gm", "Gm", classHumanFighter, storage.AccessLevelAdmin)
	sa, sg := newSession("alice"), newSession("gm")
	e.enter(sa, 0)
	e.enter(sg, 0)
	sa.take(t)
	sg.take(t)

	// The level-up animation is the server's to play (Mobius allows 2-13).
	socialRequest(e, sa, 15)
	socialRequest(e, sg, 1)
	const warning = "You will be kicked for illegal action, GM informed."
	for who, sess := range map[string]*fakeSession{"Alice": sa, "the GM": sg} {
		ps := sess.take(t)
		if len(only[testclient.SocialAction](ps)) != 0 || !slices.Equal(messages(ps), []string{warning}) {
			t.Errorf("%s got %q (%q), want only the warning", who, testclient.Names(ps), messages(ps))
		}
	}
	e.advance(socialKickDelay - time.Millisecond)
	if _, _, closed := sa.closedWith(t); closed {
		t.Fatal("Alice was kicked before 5 seconds")
	}
	e.advance(time.Millisecond)
	final, _, closed := sa.closedWith(t)
	if !closed {
		t.Fatal("Alice was not kicked after 5 seconds")
	}
	if _, ok := final.(testclient.LeaveWorld); !ok {
		t.Errorf("Alice was kicked with %v, want LeaveWorld", final)
	}
	if _, _, closed := sg.closedWith(t); closed {
		t.Error("the GM was kicked")
	}
}
