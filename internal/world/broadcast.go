package world

// Broadcast sends body to every player that sees o (Mobius
// Creature.broadcastPacket): the viewers around o, in its instance, for which
// CanSee(viewer, o) holds. With self set and o a viewer, o gets body first
// (Mobius Player.broadcastPacket). Every receiver gets the same slice: don't
// modify it afterwards.
func (w *World) Broadcast(o Object, body []byte, self bool) {
	w.broadcast("World.Broadcast", o, body, noLimit, self)
}

// BroadcastInRange is Broadcast limited to the viewers within r of o (3D,
// edge included). A negative r reaches nobody but o itself.
func (w *World) BroadcastInRange(o Object, body []byte, r int32, self bool) {
	w.broadcast("World.BroadcastInRange", o, body, rangeSq(r), self)
}

func (w *World) broadcast(op string, o Object, body []byte, limit int64, self bool) {
	if !w.loop.owned(op) {
		return
	}
	if self {
		if v, ok := o.(Viewer); ok {
			Send(v, body)
		}
	}
	if limit == noMatch {
		return
	}
	b := o.base()
	for _, r := range w.neighbourhood(b) {
		for _, x := range r.viewers {
			if x == b || x.instance != b.instance {
				continue
			}
			if limit != noLimit && b.pos.DistanceSq3D(x.pos) > limit {
				continue
			}
			if w.CanSee(x.viewer, o) {
				Send(x.viewer, body)
			}
		}
	}
}

// BroadcastAll sends body to every online player, in or out of the world
// (Mobius World.broadcastToAllOnlinePlayers).
func (w *World) BroadcastAll(body []byte) {
	if !w.loop.owned("World.BroadcastAll") {
		return
	}
	for _, v := range w.players {
		Send(v, body)
	}
}

// Update is a set of refreshes an object needs. MarkUpdate merges them per
// tick. The game layer gives the bits their meaning; these two are the usual
// ones, and the other bits are free.
type Update uint32

const (
	// UpdateUserInfo means the player needs its own UserInfo.
	UpdateUserInfo Update = 1 << iota
	// UpdateCharInfo means the viewers around the object need its CharInfo
	// (or NpcInfo).
	UpdateCharInfo
)

type pendingUpdate struct {
	obj Object
	u   Update
}

// updates is the per-tick coalescing buffer (decision W6).
type updates struct {
	list  []pendingUpdate
	spare []pendingUpdate
	index map[*Base]int
	flush func(o Object, u Update)
}

// MarkUpdate records that o needs the refreshes in u. Marks merge until the
// end of the tick; then the flush function (SetUpdateFlush) gets each marked
// object once, in the order they were first marked.
func (w *World) MarkUpdate(o Object, u Update) {
	if u == 0 || !w.loop.owned("World.MarkUpdate") {
		return
	}
	b := o.base()
	if i, ok := w.upd.index[b]; ok {
		w.upd.list[i].u |= u
		return
	}
	w.upd.index[b] = len(w.upd.list)
	w.upd.list = append(w.upd.list, pendingUpdate{obj: o, u: u})
}

// SetUpdateFlush sets the function that sends merged refreshes. The world
// calls it on the loop at the end of every tick, after all OnTick hooks. An
// object may have left the world since it was marked; marks made while
// flushing wait for the next tick.
func (w *World) SetUpdateFlush(fn func(o Object, u Update)) {
	if !w.loop.owned("World.SetUpdateFlush") {
		return
	}
	w.upd.flush = fn
}

// FlushUpdates sends the pending refreshes now, for example right before a
// packet that has to follow them. Without a flush function they are dropped.
func (w *World) FlushUpdates() {
	if len(w.upd.list) == 0 || !w.loop.owned("World.FlushUpdates") {
		return
	}
	list := w.upd.list
	w.upd.list = w.upd.spare[:0]
	w.upd.spare = nil
	clear(w.upd.index)
	if fn := w.upd.flush; fn != nil {
		for _, p := range list {
			w.loop.call("update flush", func() { fn(p.obj, p.u) })
		}
	}
	clear(list)
	if w.upd.spare == nil {
		w.upd.spare = list[:0]
	}
}
