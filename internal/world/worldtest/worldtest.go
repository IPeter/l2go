// Package worldtest has test doubles for package world: plain objects,
// players whose packets go to a recorder, and the recorder itself.
package worldtest

import (
	"slices"
	"sync"

	"github.com/IPeter/l2go/internal/world"
)

// Object is a plain world object, such as an NPC or an item on the ground.
type Object struct {
	world.Base
	Name string
}

// NewObject returns an object with the given ID at pos.
func NewObject(id world.ObjectID, pos world.Position) *Object {
	return &Object{Base: world.NewBase(id, pos)}
}

// Player is a viewer whose packets go to Rec.
type Player struct {
	world.Base
	Name string
	GM   bool // sees invisible objects
	Rec  *Recorder
}

// NewPlayer returns a player with the given ID at pos and an empty recorder.
func NewPlayer(id world.ObjectID, pos world.Position) *Player {
	return &Player{Base: world.NewBase(id, pos), Rec: &Recorder{}}
}

// Sink returns the player's recorder, or nil without one.
func (p *Player) Sink() world.Sink {
	if p.Rec == nil {
		return nil
	}
	return p.Rec
}

// SeesInvisible reports whether the player is a GM.
func (p *Player) SeesInvisible() bool { return p.GM }

// Recorder is a Sink that keeps every body it gets, in order. It is safe for
// concurrent use.
type Recorder struct {
	mu     sync.Mutex
	bodies [][]byte
}

// Send records body (not a copy: broadcasts share it).
func (r *Recorder) Send(body []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bodies = append(r.bodies, body)
}

// Bodies returns the recorded bodies in the order received.
func (r *Recorder) Bodies() [][]byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.bodies)
}

// Strings returns the recorded bodies as strings, for easy comparisons.
func (r *Recorder) Strings() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.bodies))
	for i, b := range r.bodies {
		out[i] = string(b)
	}
	return out
}

// Len returns the number of recorded bodies.
func (r *Recorder) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.bodies)
}

// Reset forgets the recorded bodies.
func (r *Recorder) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bodies = nil
}
