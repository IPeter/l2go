package world

import (
	"container/heap"
	"runtime/debug"
	"slices"
	"time"
)

// minEvery is the shortest Every interval.
const minEvery = time.Millisecond

// TimerID identifies a timer of the loop. Zero is never a valid ID.
type TimerID uint64

// HookID identifies a tick hook. Zero is never a valid ID.
type HookID uint64

type timer struct {
	id    TimerID
	when  time.Time
	every time.Duration // zero for After
	seq   uint64        // orders timers with the same deadline
	index int           // position in the heap; -1 when not in it
	fn    func()
}

type tickHook struct {
	id HookID
	fn func(now time.Time)
}

// After runs fn on the loop once d has passed, counted from Now. It returns
// 0 if CheckOwner refused the call. Loop-only.
func (l *Loop) After(d time.Duration, fn func()) TimerID {
	if !l.owned("Loop.After") {
		return 0
	}
	return l.addTimer(max(d, 0), 0, fn)
}

// Every runs fn on the loop every d, the first time once d has passed.
// Intervals under a millisecond count as one. Like a fixed-rate schedule, a
// loop that fell behind runs fn once for every missed interval. It returns 0
// if CheckOwner refused the call. Loop-only.
func (l *Loop) Every(d time.Duration, fn func()) TimerID {
	if !l.owned("Loop.Every") {
		return 0
	}
	d = max(d, minEvery)
	return l.addTimer(d, d, fn)
}

func (l *Loop) addTimer(d, every time.Duration, fn func()) TimerID {
	l.lastTimer++
	l.seq++
	t := &timer{id: l.lastTimer, when: l.Now().Add(d), every: every, seq: l.seq, fn: fn}
	heap.Push(&l.timers, t)
	l.timerByID[t.id] = t
	return t.id
}

// Cancel stops a timer and reports whether it was still pending: false for
// an unknown ID and for an After timer that already ran. A timer may cancel
// itself from its own function. Loop-only.
func (l *Loop) Cancel(id TimerID) bool {
	if !l.owned("Loop.Cancel") {
		return false
	}
	t, ok := l.timerByID[id]
	if !ok {
		return false
	}
	delete(l.timerByID, id)
	if t.index >= 0 {
		heap.Remove(&l.timers, t.index)
	}
	return true
}

// runTimer runs t, the earliest timer, and reschedules it if it repeats.
func (l *Loop) runTimer(t *timer) {
	at := t.when
	if t.every > 0 {
		t.when = at.Add(t.every)
		l.seq++
		t.seq = l.seq
		heap.Fix(&l.timers, t.index)
	} else {
		heap.Pop(&l.timers)
		delete(l.timerByID, t.id)
	}
	l.now = at
	l.inEvent = true
	_ = l.call("timer", t.fn) // a panic is logged by call; the loop goes on
	l.inEvent = false
}

// OnTick adds fn to the hooks run on every tick, in the order added. now is
// the tick's scheduled time. It returns 0 if CheckOwner refused the call.
// Loop-only.
func (l *Loop) OnTick(fn func(now time.Time)) HookID {
	if !l.owned("Loop.OnTick") {
		return 0
	}
	l.lastHook++
	l.hooks = append(l.hooks, tickHook{l.lastHook, fn})
	return l.lastHook
}

// OnTickEnd adds fn to the hooks run on every tick after all OnTick hooks,
// in the order added: for work that must see the whole tick, such as
// flushing coalesced updates. It returns 0 if CheckOwner refused the call.
// Loop-only.
func (l *Loop) OnTickEnd(fn func(now time.Time)) HookID {
	if !l.owned("Loop.OnTickEnd") {
		return 0
	}
	l.lastHook++
	l.endHooks = append(l.endHooks, tickHook{l.lastHook, fn})
	return l.lastHook
}

// RemoveHook removes a tick hook and reports whether it existed. A hook
// removed while a tick runs still runs in that tick. Loop-only.
func (l *Loop) RemoveHook(id HookID) bool {
	if !l.owned("Loop.RemoveHook") {
		return false
	}
	for _, list := range []*[]tickHook{&l.hooks, &l.endHooks} {
		i := slices.IndexFunc(*list, func(h tickHook) bool { return h.id == id })
		if i >= 0 {
			// A new slice, so a tick in progress keeps iterating the old one.
			*list = slices.Concat((*list)[:i], (*list)[i+1:])
			return true
		}
	}
	return false
}

// runTick runs the hooks of the tick scheduled at at.
func (l *Loop) runTick(at time.Time) {
	l.now = at
	l.inEvent = true
	for _, h := range l.hooks {
		l.callHook(h.fn, at)
	}
	for _, h := range l.endHooks {
		l.callHook(h.fn, at)
	}
	l.inEvent = false
}

func (l *Loop) callHook(fn func(time.Time), at time.Time) {
	defer func() {
		if v := recover(); v != nil {
			l.log.Error("world: tick hook panicked", "panic", v, "stack", string(debug.Stack()))
		}
	}()
	fn(at)
}

// timerHeap orders timers by deadline, then by seq.
type timerHeap []*timer

func (h timerHeap) Len() int { return len(h) }

func (h timerHeap) Less(i, j int) bool {
	if a, b := h[i].when, h[j].when; !a.Equal(b) {
		return a.Before(b)
	}
	return h[i].seq < h[j].seq
}

func (h timerHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *timerHeap) Push(x any) {
	t := x.(*timer)
	t.index = len(*h)
	*h = append(*h, t)
}

func (h *timerHeap) Pop() any {
	old := *h
	n := len(old) - 1
	t := old[n]
	old[n] = nil
	t.index = -1
	*h = old[:n]
	return t
}
