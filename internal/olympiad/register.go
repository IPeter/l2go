package olympiad

import (
	"slices"
	"time"
)

// Registration (Mobius Olympiad.registerNoble and unRegisterNoble) and the
// waiting lists: one for non-classed matches, one per class for classed
// matches.

// Applicant is what registerNoble checks of a character.
type Applicant struct {
	CharID  int32
	Name    string
	ClassID int
	// Noble, OnBaseClass (the active class is the base class),
	// CursedWeapon (the item ID of a cursed weapon held, 0 for none).
	Noble        bool
	OnBaseClass  bool
	CursedWeapon int
	// InventorySize and InventoryLimit are the items carried and the slots.
	InventorySize, InventoryLimit int
}

// Refusal is why a registration or a cancellation was refused: a system
// message, or Text for Mobius's plain messages. The zero value is none.
type Refusal struct {
	Msg  Message
	Text string
	// Item is the $s1 of MsgCursedWeaponNoOlympiad.
	Item int
}

// Refused reports whether r refuses.
func (r Refusal) Refused() bool { return r.Msg != 0 || r.Text != "" }

// Queues are the waiting lists (Mobius _nonClassBasedRegisters and
// _classBasedRegisters). The classed lists are kept in the order a Java
// HashMap of Integer keys iterates them, which decides what a random pick
// of a class means. The zero value is empty and ready to use.
type Queues struct {
	nonClassed []int32
	classed    map[int][]int32
	// keys are the class keys in insertion order; Mobius removes and puts a
	// key back on every change, which moves it last in its bucket.
	keys []int
	// capacity is the Java HashMap's table size: 16, doubled when the size
	// passes three quarters of it; clear keeps it.
	capacity int
}

// keyOrder returns the classed keys in Java HashMap iteration order: by
// bucket (key & (capacity − 1)), then by insertion within a bucket (a
// resize keeps the relative order).
func (q *Queues) keyOrder() []int {
	capacity := max(q.capacity, 16)
	keys := slices.Clone(q.keys)
	slices.SortStableFunc(keys, func(a, b int) int { return (a & (capacity - 1)) - (b & (capacity - 1)) })
	return keys
}

// putKey is HashMap.put of a key: a new key goes last and may grow the
// table.
func (q *Queues) putKey(class int) {
	if i := slices.Index(q.keys, class); i >= 0 {
		q.keys = slices.Delete(q.keys, i, i+1)
	}
	q.keys = append(q.keys, class)
	if q.capacity == 0 {
		q.capacity = 16
	}
	for len(q.keys) > q.capacity*3/4 {
		q.capacity *= 2
	}
}

// NonClassed returns the non-classed waiting list.
func (q *Queues) NonClassed() []int32 { return slices.Clone(q.nonClassed) }

// Classed returns a class's waiting list.
func (q *Queues) Classed(class int) []int32 { return slices.Clone(q.classed[class]) }

// InClassed reports whether charID waits for a classed match of class.
func (q *Queues) InClassed(class int, charID int32) bool {
	return slices.Contains(q.classed[class], charID)
}

// IsRegistered is Mobius isRegistered: charID waits in the non-classed list
// or in the list of its class.
func (q *Queues) IsRegistered(class int, charID int32) bool {
	return slices.Contains(q.nonClassed, charID) || q.InClassed(class, charID)
}

// Counts is Mobius getWaitingList: the nobles waiting for classed and for
// non-classed matches.
func (q *Queues) Counts() (classed, nonClassed int) {
	for _, l := range q.classed {
		classed += len(l)
	}
	return classed, len(q.nonClassed)
}

// AddClassed and AddNonClassed put a noble on a list.
func (q *Queues) AddClassed(class int, charID int32) {
	if q.classed == nil {
		q.classed = make(map[int][]int32)
	}
	q.classed[class] = append(q.classed[class], charID)
	q.putKey(class)
}

// AddNonClassed puts a noble on the non-classed list.
func (q *Queues) AddNonClassed(charID int32) { q.nonClassed = append(q.nonClassed, charID) }

// Remove is the list part of Mobius unRegisterNoble and
// removeDisconnectedCompetitor: the noble leaves the non-classed list, or
// else its class's list (whose key stays, even when empty, as in Mobius).
// It reports whether it was on one.
func (q *Queues) Remove(class int, charID int32) bool {
	if i := slices.Index(q.nonClassed, charID); i >= 0 {
		q.nonClassed = slices.Delete(q.nonClassed, i, i+1)
		return true
	}
	l, ok := q.classed[class]
	if !ok {
		return false
	}
	i := slices.Index(l, charID)
	if i >= 0 {
		l = slices.Delete(l, i, i+1)
	}
	q.classed[class] = l
	q.putKey(class)
	return i >= 0
}

// Clear is Mobius clearRegistered: both lists empty, the table size kept.
func (q *Queues) Clear() {
	q.nonClassed = nil
	q.classed = nil
	q.keys = nil
}

// Register is the rule of Mobius registerNoble for a. inComp is whether the
// competition runs, untilEnd the time until it ends, inGame whether a is
// already selected for a match. On success the noble is in nobles (with
// StartPoints when new) and on a list, and msg is the "registered" message;
// otherwise refused says why. A new noble stays in nobles even when its
// points are too few, as in Mobius.
func Register(c Config, nobles *Nobles, q *Queues, a Applicant, classed, inComp, inGame bool, untilEnd time.Duration) (msg Message, refused Refusal) {
	switch {
	case !inComp:
		return 0, Refusal{Msg: MsgNotInProgress}
	case !a.Noble:
		return 0, Refusal{Msg: MsgOnlyNobles}
	case !a.OnBaseClass:
		return 0, Refusal{Msg: MsgNoSubclass}
	case a.CursedWeapon != 0:
		return 0, Refusal{Msg: MsgCursedWeaponNoOlympiad, Item: a.CursedWeapon}
	case float64(a.InventoryLimit)*0.8 <= float64(a.InventorySize):
		return 0, Refusal{Msg: MsgTooMuchStuff}
	case untilEnd < 10*time.Minute:
		return 0, Refusal{Msg: MsgNoRegistrationLast10}
	case q.InClassed(a.ClassID, a.CharID):
		return 0, Refusal{Msg: MsgAlreadyClassed}
	case slices.Contains(q.nonClassed, a.CharID) || inGame:
		return 0, Refusal{Msg: MsgAlreadyNonClassed}
	}
	n := nobles.Add(c, a.CharID, a.ClassID, a.Name)
	if classed && n.Points < 3 {
		return 0, Refusal{Text: "Cannot register when you have less than 3 points"}
	}
	if !classed && n.Points < 5 {
		return 0, Refusal{Text: "Cannot register when you have less than 5 points"}
	}
	if classed {
		q.AddClassed(a.ClassID, a.CharID)
		return MsgRegisteredClassed, Refusal{}
	}
	q.AddNonClassed(a.CharID)
	return MsgRegisteredNonClassed, Refusal{}
}

// Unregister is the rule of Mobius unRegisterNoble: on success the noble
// left its list and msg is "removed from the waiting list".
func Unregister(q *Queues, classID int, charID int32, noble, inComp, inGame bool) (msg Message, refused Refusal) {
	switch {
	case !inComp:
		return 0, Refusal{Msg: MsgNotInProgress}
	case !noble:
		return 0, Refusal{Msg: MsgOnlyNobles}
	case !q.IsRegistered(classID, charID):
		return 0, Refusal{Msg: MsgNotRegistered}
	case inGame:
		return 0, Refusal{Text: "Can't deregister whilst you are already selected for a game"}
	}
	q.Remove(classID, charID)
	return MsgRemovedFromList, Refusal{}
}

// ReadyClasses is Mobius hasEnoughRegisteredClassed: the classes with at
// least ClassedParticipants waiting, in HashMap order; nil for none.
func (c Config) ReadyClasses(q *Queues) []int {
	var out []int
	for _, class := range q.keyOrder() {
		if len(q.classed[class]) >= c.ClassedParticipants {
			out = append(out, class)
		}
	}
	return out
}

// NonClassedReady is Mobius hasEnoughRegisteredNonClassed.
func (c Config) NonClassedReady(q *Queues) bool { return len(q.nonClassed) >= c.NonClassedParticipants }

// randomClass is Mobius getRandomClassList: the class of a random ready
// class (one draw), -1 when none is ready.
func randomClass(r Rand, ready []int) int {
	if len(ready) == 0 {
		return -1
	}
	return ready[r.Get(len(ready))]
}

// nextOpponents is Mobius nextOpponents: two random nobles taken from the
// list (two draws), nil when fewer than two wait.
func nextOpponents(r Rand, list []int32) (pair []int32, rest []int32) {
	if len(list)/2 < 1 {
		return nil, list
	}
	first := r.Get(len(list))
	a := list[first]
	list = slices.Delete(slices.Clone(list), first, first+1)
	second := r.Get(len(list))
	b := list[second]
	list = slices.Delete(list, second, second+1)
	return []int32{a, b}, list
}

// Pairing is a match the manager makes: the stadium, its type and the two
// nobles (the first is player one).
type Pairing struct {
	Stadium int
	Type    CompetitionType
	Players [2]int32
}

// MakeMatches is one round of Mobius OlympiadManager.run for the stadiums
// that are free: with enough nobles waiting, each free stadium from the
// first gets a match while two nobles of a ready list wait; the first half
// prefers non-classed matches, the second classed ones. Every random pick
// is Mobius's: a class is drawn for each test of the classed lists, and
// two nobles for each match. The chosen nobles leave their lists.
func (c Config) MakeMatches(r Rand, q *Queues, free func(stadium int) bool) []Pairing {
	ready := c.ReadyClasses(q)
	readyNonClassed := c.NonClassedReady(q)
	if ready == nil && !readyNonClassed {
		return nil
	}
	// classedPair is existNextOpponents(getRandomClassList(...)): a class
	// drawn, and whether two of its nobles wait.
	classedPair := func() bool {
		class := randomClass(r, ready)
		return class >= 0 && len(q.classed[class])/2 >= 1
	}
	nonClassedPair := func() bool { return len(q.nonClassed)/2 >= 1 }
	// take makes a match of two nobles of list (two draws); ok is false
	// when fewer than two wait: Mobius's game then fails and the stadium
	// is tried again.
	take := func(i int, typ CompetitionType, list *[]int32) (Pairing, bool) {
		pair, rest := nextOpponents(r, *list)
		*list = rest
		if pair == nil {
			return Pairing{}, false
		}
		return Pairing{Stadium: i, Type: typ, Players: [2]int32{pair[0], pair[1]}}, true
	}
	takeClassed := func(i int) (Pairing, bool) {
		class := randomClass(r, ready)
		l := q.classed[class]
		p, ok := take(i, Classed, &l)
		q.classed[class] = l
		return p, ok
	}
	var out []Pairing
	retries := 0
	for i := 0; i < Stadiums; i++ {
		// Mobius tests the non-classed list first, so a class is drawn only
		// when it has no pair.
		if !nonClassedPair() && !classedPair() {
			break
		}
		if !free(i) {
			continue
		}
		var p Pairing
		made, failed := false, false
		if i < Stadiums/2 {
			switch {
			case readyNonClassed && nonClassedPair():
				p, made = take(i, NonClassed, &q.nonClassed)
			case ready != nil && classedPair():
				p, made = takeClassed(i)
				failed = !made
			}
		} else {
			switch {
			case ready != nil && classedPair():
				p, made = takeClassed(i)
				failed = !made
			case readyNonClassed && nonClassedPair():
				p, made = take(i, NonClassed, &q.nonClassed)
			}
		}
		switch {
		case made:
			out = append(out, p)
		case failed && retries < maxMatchRetries:
			// The class drawn for the match had no pair: Mobius tries the
			// stadium again (i--).
			retries++
			i--
		}
	}
	return out
}

// maxMatchRetries bounds the retries of a stadium whose classed match drew
// a class without a pair (Mobius retries without a bound).
const maxMatchRetries = 1000
