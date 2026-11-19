package game

import (
	"errors"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// actionBody encodes RequestActionUse: the action, Ctrl and Shift (off).
func actionBody(actionID int32) []byte {
	w := packets.NewWriter(9)
	w.WriteInt32(actionID)
	w.WriteInt32(0)
	w.WriteUint8(0)
	return w.Bytes()
}

// actionUse sends RequestActionUse for a session and waits for the loop.
func actionUse(e *testEnv, sess *fakeSession, actionID int32) {
	e.srv.Handle(sess, clientpackets.OpRequestActionUse, actionBody(actionID))
	e.sync()
}

// Action 0 sits down and stands up, with Mobius's 2.5 s animations: a sitting
// player, and one standing up, cannot walk, and a request during an
// animation is ignored.
func TestSitAndStand(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, pa, _ := twoPlayers(t, e)
	walkSettle(t, e, sa, sb)
	home := e.position(pa)

	actionUse(e, sa, clientpackets.ActionUseSitStand)
	ps := sa.take(t)
	checkNames(t, "Alice sits down", ps, "ChangeWaitType")
	if w := ps[0].(testclient.ChangeWaitType); w.ObjectID != pa.ObjectID() || w.WaitType != testclient.WaitSitting ||
		w.X != home.X || w.Y != home.Y || w.Z != home.Z {
		t.Errorf("ChangeWaitType %+v, want Alice sitting at %v", w, home)
	}
	checkNames(t, "Bob sees Alice sit down", sb.take(t), "ChangeWaitType")

	walkClick(e, sa, home, walkOffset(home, 100, 0, 0))
	checkNames(t, "a click while sitting", sa.take(t), "ActionFailed")
	actionUse(e, sa, clientpackets.ActionUseSitStand)
	checkNames(t, "a request while sitting down", sa.take(t))

	e.advance(sitAnimation)
	actionUse(e, sa, clientpackets.ActionUseSitStand)
	ps = sa.take(t)
	checkNames(t, "Alice stands up", ps, "ChangeWaitType")
	if w := ps[0].(testclient.ChangeWaitType); w.WaitType != testclient.WaitStanding {
		t.Errorf("ChangeWaitType %+v, want standing", w)
	}
	checkNames(t, "Bob sees Alice stand up", sb.take(t), "ChangeWaitType")
	walkClick(e, sa, home, walkOffset(home, 100, 0, 0))
	checkNames(t, "a click while standing up", sa.take(t), "ActionFailed")

	e.advance(sitAnimation)
	walkClick(e, sa, home, walkOffset(home, 100, 0, 0))
	checkNames(t, "a click after standing up", sa.take(t), "MoveToLocation")
}

// Sitting down while walking waits for the walk to arrive (Mobius NextAction
// on ARRIVED).
func TestSitWhenTheWalkArrives(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, pa, _ := twoPlayers(t, e)
	walkSettle(t, e, sa, sb)
	home := e.position(pa)
	walkClick(e, sa, home, walkOffset(home, 300, 0, 0))
	sa.take(t)
	sb.take(t)

	actionUse(e, sa, clientpackets.ActionUseSitStand)
	checkNames(t, "a sit request while walking", sa.take(t))
	e.advance(3 * time.Second)
	checkNames(t, "Alice arrives and sits", sa.take(t), "MoveToLocation", "ChangeWaitType")
	checkNames(t, "Bob", sb.take(t), "MoveToLocation", "ChangeWaitType")
	var sitting bool
	e.onLoop(func() { sitting = pa.Sitting })
	if !sitting {
		t.Error("Alice does not sit after her walk")
	}
}

// /sit (ChangeWaitType2 0) stops a walk first; /stand (1) stands up.
func TestSitCommandStopsTheWalk(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, pa, _ := twoPlayers(t, e)
	walkSettle(t, e, sa, sb)
	home := e.position(pa)
	walkClick(e, sa, home, walkOffset(home, 300, 0, 0))
	e.advance(world.GameTick)
	sa.take(t)
	sb.take(t)

	e.srv.Handle(sa, opChangeWaitType2, walkInts(0))
	e.sync()
	checkNames(t, "/sit while walking", sa.take(t), "StopMove", "ChangeWaitType")
	checkNames(t, "Bob", sb.take(t), "StopMove", "ChangeWaitType")
	if walking, _ := walkState(e, pa); walking {
		t.Error("still walking after /sit")
	}

	e.advance(sitAnimation)
	e.srv.Handle(sa, opChangeWaitType2, walkInts(1))
	e.sync()
	ps := sa.take(t)
	checkNames(t, "/stand", ps, "ChangeWaitType")
	if w := ps[0].(testclient.ChangeWaitType); w.WaitType != testclient.WaitStanding {
		t.Errorf("ChangeWaitType %+v, want standing", w)
	}
}

// Action 1 switches between running and walking: everyone gets
// ChangeMoveType at once, then UserInfo and CharInfo with the new gait, and
// the speed changes. /run and /walk (ChangeMoveType2) do the same; asking
// for the gait the player has changes nothing.
func TestWalkRunToggle(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, pa, _ := twoPlayers(t, e)
	walkSettle(t, e, sa, sb)

	actionUse(e, sa, clientpackets.ActionUseWalkRun)
	ps := sa.take(t)
	checkNames(t, "Alice switches to walking", ps, "ChangeMoveType")
	if c := ps[0].(testclient.ChangeMoveType); c.ObjectID != pa.ObjectID() || c.Running {
		t.Errorf("ChangeMoveType %+v, want Alice walking", c)
	}
	checkNames(t, "Bob", sb.take(t), "ChangeMoveType")
	e.advance(world.GameTick)
	ps = sa.take(t)
	checkNames(t, "Alice at the end of the tick", ps, "UserInfo")
	if ps[0].(testclient.UserInfo).Running {
		t.Error("UserInfo shows Alice running")
	}
	pbs := sb.take(t)
	checkNames(t, "Bob at the end of the tick", pbs, "CharInfo")
	if pbs[0].(testclient.CharInfoPacket).Running {
		t.Error("CharInfo shows Alice running")
	}
	var speed float64
	e.onLoop(func() { speed = pa.Stats().MoveSpeed() })
	if math.Abs(speed-88) > 1e-9 { // walk speed 80 × DEX bonus 1.1
		t.Errorf("walking speed %v, want 88", speed)
	}

	e.srv.Handle(sa, opChangeMoveType2, walkInts(1))
	e.srv.Handle(sa, opChangeMoveType2, walkInts(1))
	e.sync()
	ps = sa.take(t)
	checkNames(t, "/run twice", ps, "ChangeMoveType")
	if !ps[0].(testclient.ChangeMoveType).Running {
		t.Errorf("ChangeMoveType %+v, want running", ps[0])
	}
	e.advance(world.GameTick)
	checkNames(t, "Alice after /run", sa.take(t), "UserInfo")
	checkNames(t, "Bob after /run", sb.take(t), "ChangeMoveType", "CharInfo")
}

// Other features add actions with RegisterActionUse; Mobius ignores the
// actions it does not know.
func TestRegisterActionUse(t *testing.T) {
	var got []int32
	e := newEnv(t, nil, func(s *Server) {
		if err := s.RegisterActionUse(9999, func(_ *Player, req clientpackets.RequestActionUse) {
			got = append(got, req.ActionID)
		}); err != nil {
			t.Fatalf("RegisterActionUse: %v", err)
		}
		if err := s.RegisterActionUse(clientpackets.ActionUseSitStand, func(*Player, clientpackets.RequestActionUse) {}); !errors.Is(err, ErrDuplicate) {
			t.Errorf("RegisterActionUse of sit/stand = %v, want ErrDuplicate", err)
		}
		if err := s.RegisterActionUse(9998, nil); err == nil {
			t.Error("RegisterActionUse accepted a nil handler")
		}
	})
	sa, sb, _, _ := twoPlayers(t, e)
	walkSettle(t, e, sa, sb)

	actionUse(e, sa, 9999)
	actionUse(e, sa, 9997)
	var calls []int32
	e.onLoop(func() { calls = slices.Clone(got) })
	if !slices.Equal(calls, []int32{9999}) {
		t.Errorf("handled actions %v, want [9999]", calls)
	}
	checkNames(t, "an unknown action", sa.take(t))
}
