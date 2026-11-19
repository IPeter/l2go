package testclient

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

// queueConn returns a connection without a socket whose reader queue the
// test fills with feed, as the background reader would.
func queueConn(limit int) *GameConn {
	return &GameConn{world: newWorld(), q: newQueue(limit)}
}

// feed decodes raw server packets and queues them.
func feed(c *GameConn, bodies ...[]byte) {
	for _, b := range bodies {
		c.q.push(c.receive(b))
	}
}

func actionFailed() []byte { return msg(0x25).b }
func leaveWorld() []byte   { return msg(0x7e).b }
func chatLine(text string) []byte {
	return mobiusCreatureSay(CreatureSay{ObjectID: testOtherID, Type: ChatGeneral, Name: "Pal", Text: text})
}

func isLeaveWorld(p Packet) bool {
	_, ok := p.(LeaveWorld)
	return ok
}

func TestNextInOrder(t *testing.T) {
	c := queueConn(0)
	feed(c, actionFailed(), chatLine("hi"), []byte{0xf8, 1})
	ctx := testContext(t)
	want := []Packet{
		ActionFailed{},
		CreatureSay{ObjectID: testOtherID, Type: ChatGeneral, Name: "Pal", Text: "hi"},
		Unknown{Opcode: 0xf8, Body: []byte{1}},
	}
	for i, w := range want {
		p, err := c.Next(ctx)
		if err != nil {
			t.Fatalf("Next %d: %v", i, err)
		}
		if !reflect.DeepEqual(p, w) {
			t.Errorf("Next %d = %#v, want %#v", i, p, w)
		}
	}
	if n := c.PacketsReceived(); n != 3 {
		t.Errorf("PacketsReceived %d, want 3", n)
	}
	if chat := c.World().Chat(); len(chat) != 1 || chat[0].Text != "hi" {
		t.Errorf("the world model missed the chat line: %+v", chat)
	}
}

// TestWaitForTimeout runs on the synctest clock: the wait ends exactly at
// the deadline, names what it skipped, and leaves the queue usable.
func TestWaitForTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := queueConn(0)
		feed(c, actionFailed(), chatLine("x"))
		ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
		defer cancel()
		start := time.Now()
		_, err := c.WaitFor(ctx, isLeaveWorld)
		if elapsed := time.Since(start); elapsed != 3*time.Second {
			t.Errorf("WaitFor returned after %v, want 3s", elapsed)
		}
		var we *WaitError
		if !errors.As(err, &we) || !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("got %v, want a *WaitError wrapping context.DeadlineExceeded", err)
		}
		if !slices.Equal(we.Seen, []string{"ActionFailed", "CreatureSay"}) || we.SeenCount != 2 {
			t.Errorf("Seen %v (%d)", we.Seen, we.SeenCount)
		}
		if !strings.Contains(err.Error(), "read meanwhile: ActionFailed, CreatureSay") {
			t.Errorf("error %q does not name the skipped packets", err)
		}

		// A packet that comes later is still read in order.
		feed(c, leaveWorld())
		p, err := c.Next(t.Context())
		if err != nil || !isLeaveWorld(p) {
			t.Errorf("Next after the timeout = %v, %v", p, err)
		}
	})
}

func TestAwaitLateArrival(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := queueConn(0)
		go func() {
			time.Sleep(time.Second)
			feed(c, chatLine("early"))
			time.Sleep(time.Second)
			feed(c, chatLine("late"))
		}()
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		start := time.Now()
		say, err := Await(ctx, c, func(s CreatureSay) bool { return s.Text == "late" })
		if err != nil || say.Text != "late" {
			t.Fatalf("Await = %+v, %v", say, err)
		}
		if elapsed := time.Since(start); elapsed != 2*time.Second {
			t.Errorf("Await returned after %v, want 2s", elapsed)
		}
		// The world model saw the skipped line too.
		if n := len(c.World().Chat()); n != 2 {
			t.Errorf("chat log has %d lines, want 2", n)
		}
	})
}

func TestWaitForEmptyQueueTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := queueConn(0)
		ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
		defer cancel()
		_, err := Await[NpcHtmlMessage](ctx, c, nil)
		var we *WaitError
		if !errors.As(err, &we) || we.Want != "NpcHtmlMessage" || we.SeenCount != 0 {
			t.Fatalf("got %v", err)
		}
		if !strings.Contains(err.Error(), "waiting for NpcHtmlMessage") || !strings.Contains(err.Error(), "no packets meanwhile") {
			t.Errorf("error %q", err)
		}
	})
}

func TestWaitForClosedConnection(t *testing.T) {
	c := queueConn(0)
	feed(c, actionFailed())
	c.q.close(fmt.Errorf("%w: EOF", ErrServerClosed))
	ctx := testContext(t)
	_, err := Await[NpcHtmlMessage](ctx, c, nil)
	var we *WaitError
	if !errors.As(err, &we) || !errors.Is(err, ErrServerClosed) || !slices.Equal(we.Seen, []string{"ActionFailed"}) {
		t.Errorf("got %v, want a *WaitError wrapping ErrServerClosed after ActionFailed", err)
	}
	if _, err := c.Next(ctx); !errors.Is(err, ErrServerClosed) {
		t.Errorf("Next after the close: %v", err)
	}
}

func TestQueuedPacketsBeforeClose(t *testing.T) {
	c := queueConn(0)
	feed(c, actionFailed(), leaveWorld())
	c.q.close(fmt.Errorf("%w: EOF", ErrServerClosed))
	ctx := testContext(t)
	for _, want := range []string{"ActionFailed", "LeaveWorld"} {
		p, err := c.Next(ctx)
		if err != nil || p.PacketName() != want {
			t.Fatalf("Next = %v, %v; want %s", p, err, want)
		}
	}
	if _, err := c.Next(ctx); !errors.Is(err, ErrServerClosed) {
		t.Errorf("Next = %v, want ErrServerClosed", err)
	}
}

func TestNextWithDoneContext(t *testing.T) {
	c := queueConn(0)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := c.Next(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("Next on an empty queue = %v, want context.Canceled", err)
	}
	// Queued packets still come first.
	feed(c, actionFailed())
	if p, err := c.Next(ctx); err != nil || p != (ActionFailed{}) {
		t.Errorf("Next = %v, %v", p, err)
	}
}

func TestMalformedPacketInStream(t *testing.T) {
	c := queueConn(0)
	feed(c, msg(0x5f).i32(2).b, actionFailed())
	ctx := testContext(t)
	_, err := c.Next(ctx)
	if !errors.Is(err, ErrMalformed) || !strings.Contains(err.Error(), "RestartResponse: result is 2") {
		t.Errorf("Next = %v, want the malformed RestartResponse", err)
	}
	if p, err := c.Next(ctx); err != nil || p != (ActionFailed{}) {
		t.Errorf("Next after the malformed packet = %v, %v", p, err)
	}

	// A wait stops at a malformed packet.
	feed(c, msg(0x5f).i32(7).b, leaveWorld())
	if _, err := c.WaitFor(ctx, isLeaveWorld); !errors.Is(err, ErrMalformed) {
		t.Errorf("WaitFor = %v, want ErrMalformed", err)
	}
	if p, err := c.Next(ctx); err != nil || !isLeaveWorld(p) {
		t.Errorf("Next = %v, %v; want the LeaveWorld after the malformed packet", p, err)
	}
}

func TestDrain(t *testing.T) {
	var direct GameConn
	if ps, err := direct.Drain(); ps != nil || err != nil {
		t.Errorf("Drain before the reader = %v, %v", ps, err)
	}

	c := queueConn(0)
	feed(c, actionFailed(), msg(0x5f).i32(9).b, leaveWorld())
	ps, err := c.Drain()
	if !slices.Equal(Names(ps), []string{"ActionFailed", "LeaveWorld"}) {
		t.Errorf("Drain = %v", Names(ps))
	}
	if !errors.Is(err, ErrMalformed) || !strings.Contains(err.Error(), "result is 9") {
		t.Errorf("Drain error = %v", err)
	}
	if ps, err := c.Drain(); ps != nil || err != nil {
		t.Errorf("second Drain = %v, %v", ps, err)
	}
}

func TestExpect(t *testing.T) {
	c := queueConn(0)
	feed(c, actionFailed(), leaveWorld())
	ctx := testContext(t)
	if _, err := Expect[ActionFailed](ctx, c); err != nil {
		t.Fatal(err)
	}
	_, err := Expect[RestartResponse](ctx, c)
	var upe *UnexpectedPacketError
	if !errors.As(err, &upe) || upe.Name != "LeaveWorld" || upe.Want != "RestartResponse" || upe.Opcode != 0x7e {
		t.Errorf("Expect[RestartResponse] = %v", err)
	}
}

func TestQueueLimit(t *testing.T) {
	c := queueConn(2)
	feed(c, actionFailed(), chatLine("1"), chatLine("2"))
	if n := c.Dropped(); n != 1 {
		t.Errorf("Dropped %d, want 1", n)
	}
	ctx := testContext(t)
	for _, want := range []string{"1", "2"} {
		s, err := Expect[CreatureSay](ctx, c)
		if err != nil || s.Text != want {
			t.Errorf("Expect = %+v, %v; want line %s", s, err, want)
		}
	}
	if n := len(c.World().Chat()); n != 2 {
		t.Errorf("the world model saw %d chat lines, want 2", n)
	}
}

func TestPacketHelpers(t *testing.T) {
	ps := []Packet{
		UserInfo{ObjectID: 5},
		ValidateLocation{ObjectID: 9},
		ActionFailed{},
		ValidateLocation{ObjectID: 5},
		SkillList{},
		ActionFailed{},
	}
	if v, ok := Find[ValidateLocation](ps); !ok || v.ObjectID != 9 {
		t.Errorf("Find[ValidateLocation] = %+v, %v", v, ok)
	}
	if _, ok := Find[LeaveWorld](ps); ok {
		t.Error("Find[LeaveWorld] found one")
	}
	if all := FindAll[ActionFailed](ps); len(all) != 2 {
		t.Errorf("FindAll[ActionFailed] = %v", all)
	}
	wantNames := []string{"UserInfo", "ValidateLocation", "ActionFailed", "ValidateLocation", "SkillList", "ActionFailed"}
	if !slices.Equal(Names(ps), wantNames) {
		t.Errorf("Names = %v", Names(ps))
	}

	ready := []struct {
		name string
		seen []Packet
		want bool
	}{
		{"all", ps, true},
		{"other object's location", ps[:3], false},
		{"no ActionFailed after the location", []Packet{UserInfo{ObjectID: 5}, ValidateLocation{ObjectID: 5}, SkillList{}}, false},
		{"ActionFailed before the location", []Packet{UserInfo{ObjectID: 5}, ActionFailed{}, ValidateLocation{ObjectID: 5}, SkillList{}}, false},
		{"no SkillList yet", []Packet{UserInfo{ObjectID: 5}, ValidateLocation{ObjectID: 5}, ActionFailed{}}, false},
		{"no UserInfo", []Packet{ValidateLocation{ObjectID: 1}, ActionFailed{}, SkillList{}}, true},
		{"SkillList first", []Packet{SkillList{}, UserInfo{ObjectID: 5}, ValidateLocation{ObjectID: 5}, ActionFailed{}}, true},
	}
	for _, tt := range ready {
		if got := WorldReady(tt.seen); got != tt.want {
			t.Errorf("WorldReady(%s) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestWaitErrorListsAtMost64Names(t *testing.T) {
	we := &WaitError{Want: "LeaveWorld", Err: context.DeadlineExceeded}
	for range maxSeenNames + 3 {
		we.see("ActionFailed")
	}
	if len(we.Seen) != maxSeenNames || we.SeenCount != maxSeenNames+3 {
		t.Errorf("Seen %d names, count %d", len(we.Seen), we.SeenCount)
	}
	if !strings.HasSuffix(we.Error(), ", ... 3 more)") {
		t.Errorf("Error() = %q", we.Error())
	}
}
