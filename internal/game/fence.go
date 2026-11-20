package game

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/world"
)

// The fences (Mobius Fence and FenceData): boxes that block walking while
// closed. The Interlude data has one, the demo fence.

// Fence is a fence in the world. It lives on the world loop.
type Fence struct {
	world.Base

	// Template is the fence's data.
	Template *data.Fence

	state string // a data.Fence* state
	box   geo.Fence
	// layers are the object IDs of the fence's upper layers: a fence of
	// height n shows n fences, one per object ID (Mobius _heightFences).
	layers []world.ObjectID
}

var _ InfoProvider = (*Fence)(nil)

// spawnFence puts the fence of t into the world (Mobius
// FenceData.spawnFence): one object ID per layer of its height.
func (s *Server) spawnFence(t *data.Fence) error {
	ids := s.world.IDs()
	id, err := ids.Next()
	if err != nil {
		return err
	}
	pos := world.Position{X: int32(t.X), Y: int32(t.Y), Z: int32(t.Z)} //nolint:gosec // world coordinates fit
	f := &Fence{
		Base:     world.NewBase(id, pos),
		Template: t,
		state:    t.State,
		box:      geo.NewFence(pos.X, pos.Y, int32(t.Width), int32(t.Length), pos.Z), //nolint:gosec // fence sizes fit
	}
	for range t.Height - 1 {
		lid, err := ids.Next()
		if err != nil {
			return err
		}
		f.layers = append(f.layers, lid)
	}
	if err := s.AddObject(f); err != nil {
		return err
	}
	s.doors.fences = append(s.doors.fences, f)
	if k, ok := world.RegionOf(pos.X, pos.Y, pos.Z); ok {
		for _, nk := range regionNeighbourhood(k) {
			s.doors.fenceNear[nk] = append(s.doors.fenceNear[nk], f)
		}
	}
	return nil
}

// Fences returns the fences in the world, in data order. The slice must not
// be modified. Loop-only.
func (s *Server) Fences() []*Fence { return s.doors.fences }

// State returns the fence's state, a data.Fence* value.
func (f *Fence) State() string { return f.state }

// blocks reports whether the fence's state blocks walking (Mobius
// FenceState.isGeodataEnabled: closed, shown or not).
func (f *Fence) blocks() bool {
	return f.state == data.FenceClosed || f.state == data.FenceClosedHidden
}

// FencesBetween reports whether a closed fence stands between from and to
// (Mobius FenceData.checkIfFenceBetween): of the fences indexed in the
// region of from, one in the instance whose box the segment crosses
// (geo.Fence.Between). It is the fence part of the composed geodata view
// (A2). Loop-only.
func (s *Server) FencesBetween(from, to geo.Point, instance int32) bool {
	k, ok := world.RegionOf(from.X, from.Y, from.Z)
	if !ok {
		return false
	}
	for _, f := range s.doors.fenceNear[k] {
		if f.blocks() && f.InstanceID() == instance && f.box.Between(from, to) {
			return true
		}
	}
	return false
}

// SetFenceState changes a fence's state and shows it to the players around
// (Mobius Fence.setState and broadcastInfo). Loop-only.
func (s *Server) SetFenceState(f *Fence, state string) {
	f.state = state
	for v := range s.world.VisiblePlayers(f) {
		if vp, ok := v.(*Player); ok && s.world.CanSee(v, f) {
			for _, b := range f.InfoFor(vp) {
				vp.Send(b)
			}
		}
	}
}

// InfoFor returns what a player that starts to see the fence gets (Mobius
// Fence.sendInfo): ExColosseumFenceInfo for the fence, then for each upper
// layer (InfoProvider).
func (f *Fence) InfoFor(*Player) [][]byte {
	out := make([][]byte, 0, 1+len(f.layers))
	out = append(out, f.info(f.ID()))
	for _, id := range f.layers {
		out = append(out, f.info(id))
	}
	return out
}

func (f *Fence) info(id world.ObjectID) []byte {
	pos := f.Position()
	return serverpackets.NewExColosseumFenceInfoPacket(serverpackets.ExColosseumFenceInfo{
		ObjectID: int32(id),
		State:    int32(data.FenceClientState(f.state)), //nolint:gosec // 0 to 2
		X:        pos.X, Y: pos.Y, Z: pos.Z,
		Width:  int32(f.Template.Width),  //nolint:gosec // fence sizes fit
		Length: int32(f.Template.Length), //nolint:gosec // as above
	})
}
