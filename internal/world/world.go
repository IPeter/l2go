package world

import (
	"errors"
	"iter"
	"time"
)

// Errors of World.
var (
	ErrInWorld     = errors.New("world: object is in the world")
	ErrOutOfWorld  = errors.New("world: position is outside the world")
	ErrDuplicateID = errors.New("world: another object has this ID")
	ErrInvalidID   = errors.New("world: object ID is zero")
)

// World is the state the loop owns: the objects in the world by ID, the
// region grid that decides who sees whom, the online players, coalesced
// updates and game time. Its methods are loop-only, except Loop, GameTime
// and IDs.
type World struct {
	loop     *Loop
	regions  map[uint32]*region
	objects  map[ObjectID]*Base
	players  []Viewer
	playerIx map[ObjectID]int
	canSee   func(v Viewer, o Object) bool
	upd      updates
	gameTime *GameTime
	ids      *IDAllocator
	act      activation
}

// New returns an empty world run by loop. It adds a tick hook for game time
// (OnTick) and one that flushes coalesced updates (OnTickEnd), so call it
// before Run or on the loop.
func New(loop *Loop) *World {
	w := &World{
		loop:     loop,
		regions:  make(map[uint32]*region),
		objects:  make(map[ObjectID]*Base),
		playerIx: make(map[ObjectID]int),
		upd:      updates{index: make(map[*Base]int)},
		gameTime: newGameTime(loop),
		ids:      NewIDAllocator(),
		act:      activation{alwaysOn: true},
	}
	loop.OnTick(w.gameTime.update)
	loop.OnTickEnd(func(time.Time) { w.FlushUpdates() })
	return w
}

// Loop returns the loop that owns the world.
func (w *World) Loop() *Loop { return w.loop }

// GameTime returns the world's game clock. Its readers are safe for
// concurrent use, so the session layer may keep it to fill CharSelected.
func (w *World) GameTime() *GameTime { return w.gameTime }

// IDs returns the allocator of runtime object IDs (0x60000000 up).
func (w *World) IDs() *IDAllocator { return w.ids }

// Add puts o into the world at its position (Mobius spawnMe). The Delta
// holds the objects around it in its instance; nothing is sent. Errors:
// ErrInWorld, ErrInvalidID, ErrDuplicateID, ErrOutOfWorld, ErrNotOnLoop.
func (w *World) Add(o Object) (Delta, error) {
	if !w.loop.owned("World.Add") {
		return Delta{}, ErrNotOnLoop
	}
	b := o.base()
	switch {
	case b.region != nil:
		return Delta{}, ErrInWorld
	case b.id == 0:
		return Delta{}, ErrInvalidID
	}
	if _, dup := w.objects[b.id]; dup {
		return Delta{}, ErrDuplicateID
	}
	k, ok := RegionOf(b.pos.X, b.pos.Y, b.pos.Z)
	if !ok {
		return Delta{}, ErrOutOfWorld
	}
	b.self = o
	b.viewer, _ = o.(Viewer)
	r := w.regionAt(k, true)
	r.add(b)
	w.objects[b.id] = b
	w.entered(r, b)

	d := Delta{Subject: o, w: w}
	for _, n := range r.neighbors {
		for _, x := range n.objects {
			if x != b && x.instance == b.instance {
				d.Entered = append(d.Entered, x.self)
			}
		}
	}
	return d, nil
}

// Remove takes o out of the world (Mobius decayMe). The Delta holds the
// objects that were around it, in every instance as in Mobius; nothing is
// sent. For an object not in the world it is empty.
func (w *World) Remove(o Object) Delta {
	if !w.loop.owned("World.Remove") {
		return Delta{}
	}
	b := o.base()
	r := b.region
	if r == nil {
		return Delta{}
	}
	playable := b.viewer != nil
	r.remove(b)
	delete(w.objects, b.id)
	b.self, b.viewer = nil, nil
	w.left(r, playable)

	d := Delta{Subject: o, w: w}
	for _, n := range r.neighbors {
		for _, x := range n.objects {
			d.Left = append(d.Left, x.self)
		}
	}
	return d
}

// Move sets o's position (Mobius setXYZ). When an object in the world
// crosses into another region, the Delta holds the objects of the regions
// that left and entered its neighbourhood; otherwise it is empty. An object
// not in the world just gets the position. A position outside the world
// returns ErrOutOfWorld and changes nothing (Mobius stops a player at its
// last valid position and deletes an NPC).
func (w *World) Move(o Object, pos Position) (Delta, error) {
	if !w.loop.owned("World.Move") {
		return Delta{}, ErrNotOnLoop
	}
	b := o.base()
	old := b.region
	if old == nil {
		b.pos = pos
		return Delta{}, nil
	}
	k, ok := RegionOf(pos.X, pos.Y, pos.Z)
	if !ok {
		return Delta{}, ErrOutOfWorld
	}
	b.pos = pos
	if k == old.key {
		return Delta{}, nil
	}
	r := w.regionAt(k, true)
	// Mobius setXYZ: the old region loses the object (which may schedule its
	// deactivation) before the new one gets it (which may activate it).
	old.remove(b)
	w.left(old, b.viewer != nil)
	r.add(b)
	w.entered(r, b)

	d := Delta{Subject: o, w: w}
	for _, n := range old.neighbors {
		if n.key.Adjacent(k) {
			continue
		}
		for _, x := range n.objects {
			d.Left = append(d.Left, x.self)
		}
	}
	for _, n := range r.neighbors {
		if n.key.Adjacent(old.key) {
			continue
		}
		for _, x := range n.objects {
			if x != b && x.instance == b.instance {
				d.Entered = append(d.Entered, x.self)
			}
		}
	}
	return d, nil
}

// Find returns the object in the world with the given ID (Mobius
// World.findObject).
func (w *World) Find(id ObjectID) (Object, bool) {
	if !w.loop.owned("World.Find") {
		return nil, false
	}
	b, ok := w.objects[id]
	if !ok {
		return nil, false
	}
	return b.self, true
}

// Len returns the number of objects in the world.
func (w *World) Len() int { return len(w.objects) }

// Objects returns every object in the world, in no order (Mobius
// World.getVisibleObjects()). The world must not change while it runs.
func (w *World) Objects() iter.Seq[Object] {
	return func(yield func(Object) bool) {
		if !w.loop.owned("World.Objects") {
			return
		}
		for _, b := range w.objects {
			if !yield(b.self) {
				return
			}
		}
	}
}

// Regions returns the number of regions created so far.
func (w *World) Regions() int { return len(w.regions) }

// CanSee reports whether v sees o by the visibility rule (Mobius
// WorldObject.isVisibleFor). By default o is visible unless it is invisible
// and v does not see invisible objects. Regions and instances are not
// checked here.
func (w *World) CanSee(v Viewer, o Object) bool {
	if w.canSee != nil {
		return w.canSee(v, o)
	}
	return VisibleByDefault(v, o)
}

// VisibleByDefault is CanSee's default rule.
func VisibleByDefault(v Viewer, o Object) bool {
	return !o.base().invisible || v.SeesInvisible()
}

// SetVisibilityRule replaces CanSee's rule, for example so scripts can hide
// an NPC from some players (Mobius OnNpcCanBeSeen). Nil restores the
// default.
func (w *World) SetVisibilityRule(rule func(v Viewer, o Object) bool) {
	if !w.loop.owned("World.SetVisibilityRule") {
		return
	}
	w.canSee = rule
}

// AddPlayer registers v as online, so Player and BroadcastAll reach it
// whether or not it is in the world (a teleporting player is online but not
// in the world). It returns ErrDuplicateID if another online player has the
// same ID; Mobius then disconnects both.
func (w *World) AddPlayer(v Viewer) error {
	if !w.loop.owned("World.AddPlayer") {
		return ErrNotOnLoop
	}
	id := v.ID()
	if _, dup := w.playerIx[id]; dup {
		return ErrDuplicateID
	}
	w.playerIx[id] = len(w.players)
	w.players = append(w.players, v)
	return nil
}

// RemovePlayer unregisters v. It does nothing when another viewer is
// registered under v's ID, so a stale session can't remove a new one.
func (w *World) RemovePlayer(v Viewer) {
	if !w.loop.owned("World.RemovePlayer") {
		return
	}
	id := v.ID()
	i, ok := w.playerIx[id]
	if !ok || w.players[i].base() != v.base() {
		return
	}
	last := len(w.players) - 1
	moved := w.players[last]
	w.players[i] = moved
	w.playerIx[moved.ID()] = i
	w.players[last] = nil
	w.players = w.players[:last]
	delete(w.playerIx, id)
}

// Player returns the online player with the given ID.
func (w *World) Player(id ObjectID) (Viewer, bool) {
	if !w.loop.owned("World.Player") {
		return nil, false
	}
	i, ok := w.playerIx[id]
	if !ok {
		return nil, false
	}
	return w.players[i], true
}

// Players yields the online players. Don't add or remove players while
// iterating.
func (w *World) Players() iter.Seq[Viewer] {
	return func(yield func(Viewer) bool) {
		if !w.loop.owned("World.Players") {
			return
		}
		for _, v := range w.players {
			if !yield(v) {
				return
			}
		}
	}
}

// PlayerCount returns the number of online players.
func (w *World) PlayerCount() int { return len(w.players) }
