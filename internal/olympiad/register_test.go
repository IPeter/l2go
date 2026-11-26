package olympiad

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
)

func applicant(id int32, class int) Applicant {
	return Applicant{CharID: id, Name: "N", ClassID: class, Noble: true, OnBaseClass: true, InventorySize: 10, InventoryLimit: 80}
}

// The checks of registerNoble, in Mobius's order.
func TestRegisterRules(t *testing.T) {
	c := DefaultConfig()
	ok := applicant(1, 88)
	for _, tc := range []struct {
		name     string
		a        Applicant
		classed  bool
		inComp   bool
		inGame   bool
		untilEnd time.Duration
		points   int // the noble's points before, 0 for a new noble
		want     Refusal
		wantMsg  Message
	}{
		{"no competition", ok, false, false, false, time.Hour, 0, Refusal{Msg: MsgNotInProgress}, 0},
		{"not noble", Applicant{CharID: 1, ClassID: 88, OnBaseClass: true, InventoryLimit: 80}, false, true, false, time.Hour, 0, Refusal{Msg: MsgOnlyNobles}, 0},
		{"subclass", Applicant{CharID: 1, ClassID: 88, Noble: true, InventoryLimit: 80}, false, true, false, time.Hour, 0, Refusal{Msg: MsgNoSubclass}, 0},
		{"cursed", Applicant{CharID: 1, ClassID: 88, Noble: true, OnBaseClass: true, CursedWeapon: 8190, InventoryLimit: 80}, false, true, false, time.Hour, 0, Refusal{Msg: MsgCursedWeaponNoOlympiad, Item: 8190}, 0},
		{"80 % full", Applicant{CharID: 1, ClassID: 88, Noble: true, OnBaseClass: true, InventorySize: 64, InventoryLimit: 80}, false, true, false, time.Hour, 0, Refusal{Msg: MsgTooMuchStuff}, 0},
		{"last 10 minutes", ok, false, true, false, 10*time.Minute - time.Millisecond, 0, Refusal{Msg: MsgNoRegistrationLast10}, 0},
		{"in a game", ok, false, true, true, time.Hour, 0, Refusal{Msg: MsgAlreadyNonClassed}, 0},
		{"classed, 2 points", ok, true, true, false, time.Hour, 2, Refusal{Text: "Cannot register when you have less than 3 points"}, 0},
		{"non-classed, 4 points", ok, false, true, false, time.Hour, 4, Refusal{Text: "Cannot register when you have less than 5 points"}, 0},
		{"classed, 3 points", ok, true, true, false, time.Hour, 3, Refusal{}, MsgRegisteredClassed},
		{"non-classed, new", ok, false, true, false, 10 * time.Minute, 0, Refusal{}, MsgRegisteredNonClassed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nobles := &Nobles{}
			if tc.points > 0 {
				nobles.Add(c, 1, 88, "N").Points = tc.points
			}
			var q Queues
			msg, refused := Register(c, nobles, &q, tc.a, tc.classed, tc.inComp, tc.inGame, tc.untilEnd)
			if refused != tc.want || msg != tc.wantMsg {
				t.Errorf("Register = %v, %+v; want %v, %+v", msg, refused, tc.wantMsg, tc.want)
			}
			if !refused.Refused() && !q.IsRegistered(88, 1) {
				t.Error("not on a list after registering")
			}
		})
	}
}

// A second registration is refused, on either list; cancelling needs the
// competition, noblesse, a list, and no match yet.
func TestRegisterTwiceAndCancel(t *testing.T) {
	c := DefaultConfig()
	nobles := &Nobles{}
	var q Queues
	a := applicant(1, 88)
	if _, r := Register(c, nobles, &q, a, true, true, false, time.Hour); r.Refused() {
		t.Fatal(r)
	}
	for _, classed := range []bool{true, false} {
		if _, r := Register(c, nobles, &q, a, classed, true, false, time.Hour); r.Msg != MsgAlreadyClassed {
			t.Errorf("again (classed %v) = %+v", classed, r)
		}
	}
	b := applicant(2, 90)
	Register(c, nobles, &q, b, false, true, false, time.Hour)
	if _, r := Register(c, nobles, &q, b, true, true, false, time.Hour); r.Msg != MsgAlreadyNonClassed {
		t.Errorf("classed after non-classed = %+v", r)
	}
	if cl, non := q.Counts(); cl != 1 || non != 1 {
		t.Errorf("Counts = %d, %d", cl, non)
	}
	for _, tc := range []struct {
		noble, inComp, inGame bool
		id                    int32
		want                  Refusal
	}{
		{true, false, false, 1, Refusal{Msg: MsgNotInProgress}},
		{false, true, false, 1, Refusal{Msg: MsgOnlyNobles}},
		{true, true, false, 9, Refusal{Msg: MsgNotRegistered}},
		{true, true, true, 1, Refusal{Text: "Can't deregister whilst you are already selected for a game"}},
	} {
		if _, r := Unregister(&q, 88, tc.id, tc.noble, tc.inComp, tc.inGame); r != tc.want {
			t.Errorf("Unregister(%+v) = %+v", tc, r)
		}
	}
	if msg, r := Unregister(&q, 88, 1, true, true, false); r.Refused() || msg != MsgRemovedFromList || q.IsRegistered(88, 1) {
		t.Errorf("Unregister = %v, %+v", msg, r)
	}
	if msg, _ := Unregister(&q, 90, 2, true, true, false); msg != MsgRemovedFromList {
		t.Errorf("Unregister non-classed = %v", msg)
	}
}

// The classed lists iterate in Java HashMap order: by key & 15 while the
// table has 16 buckets, insertion order within a bucket; a change moves a
// key last.
func TestQueueOrder(t *testing.T) {
	c := DefaultConfig()
	c.ClassedParticipants = 1
	var q Queues
	for _, class := range []int{104, 88, 120, 93} { // buckets 8, 8, 8, 13
		q.AddClassed(class, int32(class))
	}
	if got := c.ReadyClasses(&q); !slices.Equal(got, []int{104, 88, 120, 93}) {
		t.Errorf("ReadyClasses = %v", got)
	}
	q.AddClassed(88, 1) // moves 88 to the end of its bucket
	if got := c.ReadyClasses(&q); !slices.Equal(got, []int{104, 120, 88, 93}) {
		t.Errorf("ReadyClasses after a change = %v", got)
	}
	q.Remove(104, 104) // empties 104, which stays (and moves)
	if got := c.ReadyClasses(&q); !slices.Equal(got, []int{120, 88, 93}) {
		t.Errorf("ReadyClasses after a removal = %v", got)
	}
	q.Clear()
	if cl, non := q.Counts(); cl != 0 || non != 0 || c.ReadyClasses(&q) != nil {
		t.Error("Clear left nobles")
	}
}

// Match making with Mobius's draws: non-classed matches fill the first
// half of the stadiums, classed ones prefer the second half.
func TestMakeMatches(t *testing.T) {
	c := DefaultConfig()
	c.NonClassedParticipants, c.ClassedParticipants = 4, 2
	var q Queues
	for id := int32(1); id <= 5; id++ {
		q.AddNonClassed(id)
	}
	q.AddClassed(88, 10)
	q.AddClassed(88, 11)
	// Stadium 0: a non-classed match: two draws (no class drawn, as the
	// non-classed list has a pair). Stadium 1: the same. Stadium 2: one
	// noble left, so the classed test draws a class (1 ready: Get(1)), the
	// branch's test draws again, the match draws its class and two nobles.
	r := rnd.NewScripted(t,
		rnd.Get(5, 4), rnd.Get(4, 0), // 5 and 1
		rnd.Get(3, 1), rnd.Get(2, 1), // 3 and 4
		rnd.Get(1, 0), rnd.Get(1, 0), rnd.Get(1, 0), rnd.Get(2, 1), rnd.Get(1, 0), // 11 and 10
		rnd.Get(1, 0), // stadium 3: the classed test fails, the loop ends
	)
	busy := map[int]bool{}
	got := c.MakeMatches(r, &q, func(i int) bool { return !busy[i] })
	r.Done()
	want := []Pairing{
		{Stadium: 0, Type: NonClassed, Players: [2]int32{5, 1}},
		{Stadium: 1, Type: NonClassed, Players: [2]int32{3, 4}},
		{Stadium: 2, Type: Classed, Players: [2]int32{11, 10}},
	}
	if !slices.Equal(got, want) {
		t.Errorf("MakeMatches = %+v, want %+v", got, want)
	}
	if !slices.Equal(q.NonClassed(), []int32{2}) || len(q.Classed(88)) != 0 {
		t.Errorf("left %v and %v", q.NonClassed(), q.Classed(88))
	}

	// Not enough waiting: nothing, and no draw.
	var q2 Queues
	q2.AddNonClassed(1)
	q2.AddNonClassed(2)
	if got := c.MakeMatches(rnd.NewScripted(t), &q2, func(int) bool { return true }); got != nil {
		t.Errorf("MakeMatches with 2 of 4 = %+v", got)
	}

	// A busy stadium is skipped; the second half prefers classed matches.
	c.NonClassedParticipants = 2
	var q3 Queues
	q3.AddClassed(90, 1)
	q3.AddClassed(90, 2)
	// Each busy stadium of the first half still runs the loop's test (a
	// class draw); stadium 11 draws for the test, the branch, the class and
	// two nobles; stadium 12's test ends the loop.
	var draws []rnd.Draw
	for range Stadiums / 2 {
		draws = append(draws, rnd.Get(1, 0))
	}
	draws = append(draws, rnd.Get(1, 0), rnd.Get(1, 0), rnd.Get(1, 0), rnd.Get(2, 0), rnd.Get(1, 0), rnd.Get(1, 0))
	r3 := rnd.NewScripted(t, draws...)
	got = c.MakeMatches(r3, &q3, func(i int) bool { return i >= Stadiums/2 })
	r3.Done()
	if want := []Pairing{{Stadium: 11, Type: Classed, Players: [2]int32{1, 2}}}; !slices.Equal(got, want) {
		t.Errorf("MakeMatches in the second half = %+v", got)
	}
}
