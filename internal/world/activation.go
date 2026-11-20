package world

import "time"

// Region activation (decision F6) ports Mobius WorldRegion: monsters think,
// regenerate and play animations only where players are.
//
//   - The first player entering an inactive region activates it at once and
//     schedules its neighbourhood (the 3×3×3 regions around it) to activate
//     after TurnOn (startActivation).
//   - When a player leaves a region and no active region around it holds a
//     player, the region schedules a check after TurnOff: then every region
//     of its neighbourhood that has no player in an active region around it
//     deactivates (startDeactivation).
//   - Each region keeps one such timer; scheduling one cancels the other.
//
// A region is active (RegionActive) or not, and counts the active regions
// of its neighbourhood (NeighborsActive is that count above 0). Players are
// the viewers; Mobius's other playables (summons, pets) do not exist yet.
//
// Regions are created lazily. The activation timer creates every region of
// the neighbourhood it activates, as all Mobius regions exist. A region
// created otherwise (an object enters a region no one activated) starts
// inactive and counts the active regions around it, which is what Mobius's
// counters hold for it.

// Activation configures region activation (Mobius GridsAlwaysOn,
// GridNeighborTurnOnTime and GridNeighborTurnOffTime).
type Activation struct {
	// AlwaysOn keeps every region active: there are no timers and no hooks
	// run (Mobius GridsAlwaysOn).
	AlwaysOn bool
	// TurnOn is the delay before the neighbours of a region a player
	// activated become active (Mobius default 1 s).
	TurnOn time.Duration
	// TurnOff is the delay before the regions around a region a player left
	// deactivate if no player is around them (Mobius default 90 s).
	TurnOff time.Duration
}

// RegionHook runs when a region becomes active or inactive (Mobius
// WorldRegion.switchAI), with the objects in it at that moment. It does not
// run for a region without objects.
//
// Activation hooks may run inside World.Add and World.Move (a player entered
// an inactive region): they must not add, move or remove objects then.
// Deactivation hooks run from a timer and may (a monster teleports home).
type RegionHook func(key RegionKey, active bool, objects []Object)

type activation struct {
	alwaysOn        bool
	turnOn, turnOff time.Duration
	hooks           []RegionHook
}

// SetActivation sets how regions activate. A new World keeps every region
// active (as AlwaysOn) until it is called. Call it before objects are added:
// it resets the regions that exist to the new rule and cancels their
// timers. Loop-only.
func (w *World) SetActivation(a Activation) {
	if !w.loop.owned("World.SetActivation") {
		return
	}
	w.act.alwaysOn = a.AlwaysOn
	w.act.turnOn, w.act.turnOff = max(a.TurnOn, 0), max(a.TurnOff, 0)
	for _, r := range w.regions {
		if r.task != 0 {
			w.loop.Cancel(r.task)
			r.task = 0
		}
		r.active = a.AlwaysOn
		r.activeNeighbors = 0
	}
}

// OnRegionActivation adds fn to the hooks run when a region becomes active or
// inactive, in the order added. Loop-only.
func (w *World) OnRegionActivation(fn RegionHook) {
	if !w.loop.owned("World.OnRegionActivation") {
		return
	}
	w.act.hooks = append(w.act.hooks, fn)
}

// RegionActive reports whether o is in an active region (Mobius
// isInActiveRegion). It is false for an object not in the world. Loop-only.
func (w *World) RegionActive(o Object) bool {
	r := o.base().region
	return r != nil && r.active
}

// NeighborsActive reports whether a region around o, its own included, is
// active (Mobius WorldRegion.areNeighborsActive): monsters think only then.
// It is false for an object not in the world. Loop-only.
func (w *World) NeighborsActive(o Object) bool {
	r := o.base().region
	return r != nil && (w.act.alwaysOn || r.activeNeighbors > 0)
}

// RegionState returns whether the region k exists, is active, and how many
// regions of its neighbourhood are active. For tests and GM tools.
// Loop-only.
func (w *World) RegionState(k RegionKey) (exists, active bool, activeNeighbors int) {
	if !k.valid() {
		return false, false, 0
	}
	r := w.regions[k.pack()]
	if r == nil {
		return false, false, 0
	}
	return true, r.active, int(r.activeNeighbors)
}

// entered runs after b was added to r (Mobius addVisibleObject): the first
// player in an inactive region activates it.
func (w *World) entered(r *region, b *Base) {
	if b.viewer != nil && !r.active && !w.act.alwaysOn {
		w.startActivation(r)
	}
}

// left runs after an object was taken out of r (Mobius removeVisibleObject):
// when it was a player and no player is around any more, the region schedules
// the deactivation of its neighbourhood.
func (w *World) left(r *region, playable bool) {
	if playable && !w.act.alwaysOn && w.neighborsEmpty(r) {
		w.startDeactivation(r)
	}
}

// setActive turns r on or off (Mobius setActive): the regions around it count
// it, and the hooks see its objects.
func (w *World) setActive(r *region, active bool) {
	if r.active == active {
		return
	}
	r.active = active
	d := int32(1)
	if !active {
		d = -1
	}
	for _, n := range r.neighbors {
		n.activeNeighbors += d
	}
	if len(r.objects) == 0 || len(w.act.hooks) == 0 {
		return
	}
	objs := make([]Object, len(r.objects))
	for i, b := range r.objects {
		objs[i] = b.self
	}
	for _, fn := range w.act.hooks {
		_ = w.loop.call("region hook", func() { fn(r.key, active, objs) }) // a panic is logged by call
	}
}

// neighborsEmpty reports whether no active region around r holds a player
// (Mobius areNeighborsEmpty).
func (w *World) neighborsEmpty(r *region) bool {
	for _, n := range r.neighbors {
		if n.active && len(n.viewers) > 0 {
			return false
		}
	}
	return true
}

// startActivation activates r now and its neighbourhood after TurnOn,
// replacing r's pending timer (Mobius startActivation).
func (w *World) startActivation(r *region) {
	w.setActive(r, true)
	if r.task != 0 {
		w.loop.Cancel(r.task)
	}
	var id TimerID
	id = w.loop.After(w.act.turnOn, func() {
		if r.task == id {
			r.task = 0
		}
		for _, k := range surrounding(r.key) {
			w.setActive(w.regionAt(k, true), true)
		}
	})
	r.task = id
}

// startDeactivation schedules, after TurnOff, the deactivation of every
// region around r with no player around it, replacing r's pending timer
// (Mobius startDeactivation).
func (w *World) startDeactivation(r *region) {
	if r.task != 0 {
		w.loop.Cancel(r.task)
	}
	var id TimerID
	id = w.loop.After(w.act.turnOff, func() {
		if r.task == id {
			r.task = 0
		}
		for _, k := range surrounding(r.key) {
			if n := w.regions[k.pack()]; n != nil && w.neighborsEmpty(n) {
				w.setActive(n, false)
			}
		}
	})
	r.task = id
}

// surrounding returns the keys of k's neighbourhood in Mobius's order
// (WorldRegion.setSurroundingRegions): the regions in x, y, z order, with k
// swapped into the first place.
func surrounding(k RegionKey) []RegionKey {
	keys := make([]RegionKey, 0, 27)
	self := 0
	for dx := int32(-1); dx <= 1; dx++ {
		for dy := int32(-1); dy <= 1; dy++ {
			for dz := int32(-1); dz <= 1; dz++ {
				nk := RegionKey{k.X + dx, k.Y + dy, k.Z + dz}
				if !nk.valid() {
					continue
				}
				if nk == k {
					self = len(keys)
				}
				keys = append(keys, nk)
			}
		}
	}
	keys[0], keys[self] = keys[self], keys[0]
	return keys
}
