package world

// Sink receives the packets for one client. The game server's connection
// writer implements it: it encrypts a copy of each body and sends it, in
// order.
type Sink interface {
	// Send queues an encoded packet body (opcode first, not encrypted). It
	// runs on the loop, so it must not block: a client that cannot keep up
	// is the sink's to deal with (disconnect it). The body may be shared
	// with every other receiver of a broadcast, so it must not be modified,
	// by the sink or by the sender after handing it over.
	Send(body []byte)
}

// SinkFunc adapts a function to Sink.
type SinkFunc func(body []byte)

// Send calls f(body).
func (f SinkFunc) Send(body []byte) { f(body) }

// Object is anything in the world. Implement it by embedding Base in a
// struct and using a pointer to that struct (decision W4); Base's unexported
// method keeps other implementations out.
type Object interface {
	ID() ObjectID
	Position() Position
	base() *Base
}

// Viewer is an object that receives packets: a player.
type Viewer interface {
	Object
	// Sink returns the client's sink, or nil (a nil interface, not a nil
	// pointer) when it has none.
	Sink() Sink
	// SeesInvisible reports whether the viewer sees invisible objects
	// (Mobius: GMs).
	SeesInvisible() bool
}

// Send hands body to v's sink, if it has one.
func Send(v Viewer, body []byte) {
	if s := v.Sink(); s != nil {
		s.Send(body)
	}
}

// Base is the world's part of an object: its ID, position, instance and
// invisibility, and its place in the region grid. Embed it by value:
//
//	type Npc struct {
//		world.Base
//		// ...
//	}
//
//	npc := &Npc{Base: world.NewBase(id, pos)}
//
// Its methods are loop-only, like all world state. Don't copy a Base while
// its object is in the world.
type Base struct {
	id        ObjectID
	pos       Position
	instance  int32
	invisible bool

	// Set by World.Add, cleared by World.Remove.
	self   Object
	viewer Viewer // self as a Viewer, or nil
	region *region
	slot   int32 // index in region.objects
	vslot  int32 // index in region.viewers, or -1
}

// NewBase returns a Base for an object with the given ID at pos, in the main
// world (instance 0) and visible.
func NewBase(id ObjectID, pos Position) Base { return Base{id: id, pos: pos} }

func (b *Base) base() *Base { return b }

// ID returns the object's ID.
func (b *Base) ID() ObjectID { return b.id }

// Position returns the object's position.
func (b *Base) Position() Position { return b.pos }

// InstanceID returns the object's instance; 0 is the main world.
func (b *Base) InstanceID() int32 { return b.instance }

// Invisible reports whether the object is invisible (Mobius
// WorldObject.isInvisible).
func (b *Base) Invisible() bool { return b.invisible }

// InWorld reports whether the object is in the world (between World.Add and
// World.Remove).
func (b *Base) InWorld() bool { return b.region != nil }

// Region returns the region the object is in, or false when it is not in the
// world.
func (b *Base) Region() (RegionKey, bool) {
	if b.region == nil {
		return RegionKey{}, false
	}
	return b.region.key, true
}

// SetHeading changes the direction the object faces.
func (b *Base) SetHeading(h int32) { b.pos.Heading = h }

// SetInvisible changes the invisibility flag. It sends nothing; Mobius
// setInvisible then sends DeleteObject to the players around that no longer
// see the object, and its info to those that still do (World.CanSee).
func (b *Base) SetInvisible(invisible bool) { b.invisible = invisible }

// SetID changes the ID of an object that is not in the world; otherwise it
// returns ErrInWorld.
func (b *Base) SetID(id ObjectID) error {
	if b.region != nil {
		return ErrInWorld
	}
	b.id = id
	return nil
}

// SetPosition places an object that is not in the world; otherwise it
// returns ErrInWorld (use World.Move).
func (b *Base) SetPosition(pos Position) error {
	if b.region != nil {
		return ErrInWorld
	}
	b.pos = pos
	return nil
}

// SetInstanceID moves an object that is not in the world to another
// instance; otherwise it returns ErrInWorld. Like a Mobius teleport, remove
// the object, change its instance and add it again.
func (b *Base) SetInstanceID(instance int32) error {
	if b.region != nil {
		return ErrInWorld
	}
	b.instance = instance
	return nil
}
