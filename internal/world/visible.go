package world

import "iter"

// Squared ranges with a special meaning.
const (
	noLimit = -1 // no distance limit
	noMatch = -2 // a negative range: nothing is within it (Mobius: distance > range)
)

// Visible yields the objects around o (Mobius World.forEachVisibleObject):
// those in the 3×3×3 regions around its position, in its instance, except o.
// The visibility rule (CanSee) is not applied; check it where it matters.
// o need not be in the world. Don't add, move or remove objects while
// iterating: collect them first.
func (w *World) Visible(o Object) iter.Seq[Object] {
	return func(yield func(Object) bool) {
		w.walk("World.Visible", o.base(), noLimit, false, func(x *Base) bool { return yield(x.self) })
	}
}

// VisibleInRange is Visible limited to objects within r of o (3D, edge
// included, as Mobius forEachVisibleObjectInRange).
func (w *World) VisibleInRange(o Object, r int32) iter.Seq[Object] {
	return func(yield func(Object) bool) {
		w.walk("World.VisibleInRange", o.base(), rangeSq(r), false, func(x *Base) bool { return yield(x.self) })
	}
}

// VisiblePlayers is Visible for viewers only.
func (w *World) VisiblePlayers(o Object) iter.Seq[Viewer] {
	return func(yield func(Viewer) bool) {
		w.walk("World.VisiblePlayers", o.base(), noLimit, true, func(x *Base) bool { return yield(x.viewer) })
	}
}

// VisiblePlayersInRange is VisibleInRange for viewers only.
func (w *World) VisiblePlayersInRange(o Object, r int32) iter.Seq[Viewer] {
	return func(yield func(Viewer) bool) {
		w.walk("World.VisiblePlayersInRange", o.base(), rangeSq(r), true, func(x *Base) bool { return yield(x.viewer) })
	}
}

// Around yields every object in the 3×3×3 regions around pos that is in the
// given instance, for questions about a place rather than an object.
func (w *World) Around(pos Position, instance int32) iter.Seq[Object] {
	return func(yield func(Object) bool) {
		if !w.loop.owned("World.Around") {
			return
		}
		k, ok := RegionOf(pos.X, pos.Y, pos.Z)
		if !ok {
			return
		}
		for _, r := range w.around(k) {
			for _, x := range r.objects {
				if x.instance == instance && !yield(x.self) {
					return
				}
			}
		}
	}
}

// rangeSq returns the squared range for r.
func rangeSq(r int32) int64 {
	if r < 0 {
		return noMatch
	}
	return int64(r) * int64(r)
}

// walk calls fn for the objects (viewers when players is set) around b: in
// its instance, except b, and within the squared range limit unless limit is
// noLimit. It stops when fn returns false.
func (w *World) walk(op string, b *Base, limit int64, players bool, fn func(*Base) bool) {
	if !w.loop.owned(op) || limit == noMatch {
		return
	}
	for _, r := range w.neighbourhood(b) {
		list := r.objects
		if players {
			list = r.viewers
		}
		for _, x := range list {
			if x == b || x.instance != b.instance {
				continue
			}
			if limit != noLimit && b.pos.DistanceSq3D(x.pos) > limit {
				continue
			}
			if !fn(x) {
				return
			}
		}
	}
}
