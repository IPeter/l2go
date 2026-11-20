package game

import (
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/world"
)

// The static objects (Mobius StaticObject and StaticObjectClick): town
// maps, thrones and signboards, which show a map or a page when used.

const (
	// staticTombSignboard is the signboard with a page of its own.
	staticTombSignboard = 24230101
	// The signboard pages, under the HTML folder.
	staticTombPage = "signboard/tomb_of_crystalgolem.htm"
	staticPvPPage  = "signboard/pvp_signboard.htm"
	// staticInteractionRange is Mobius Npc.INTERACTION_DISTANCE, which
	// StaticObjectClick measures in 2D.
	staticInteractionRange = 250
)

// StaticObject is a static object in the world. It lives on the world loop.
type StaticObject struct {
	world.Base

	// Template is the object's data.
	Template *data.StaticObject

	srv *Server
}

var (
	_ InfoProvider  = (*StaticObject)(nil)
	_ objectClicker = (*StaticObject)(nil)
)

// staticApproach is a player walking up to a static object to use it
// (Mobius INTERACT intention).
type staticApproach struct {
	obj     *StaticObject
	dest    world.Position
	moving  bool
	started time.Time
}

// spawnStaticObject puts the static object of t into the world (Mobius
// StaticObjectData.addObject).
func (s *Server) spawnStaticObject(t *data.StaticObject) error {
	id, err := s.world.IDs().Next()
	if err != nil {
		return err
	}
	o := &StaticObject{
		Base:     world.NewBase(id, world.Position{X: int32(t.X), Y: int32(t.Y), Z: int32(t.Z)}), //nolint:gosec // world coordinates fit
		Template: t,
		srv:      s,
	}
	if err := s.AddObject(o); err != nil {
		_ = s.world.IDs().Release(id)
		return err
	}
	s.doors.statics = append(s.doors.statics, o)
	return nil
}

// StaticObjects returns the static objects in the world, in data order. The
// slice must not be modified. Loop-only.
func (s *Server) StaticObjects() []*StaticObject { return s.doors.statics }

// Name returns the object's name.
func (o *StaticObject) Name() string { return o.Template.Name }

// InfoFor returns what a player that starts to see the object gets (Mobius
// StaticObject.sendInfo): StaticObjectInfo (InfoProvider).
func (o *StaticObject) InfoFor(*Player) [][]byte {
	return [][]byte{serverpackets.NewStaticObjectInfoPacket(serverpackets.StaticObjectInfo{
		StaticID: int32(o.Template.ID), //nolint:gosec // static object IDs fit
		ObjectID: int32(o.ID()),
	})}
}

// click is Mobius StaticObjectClick: the first click targets the object; a
// click on the target uses it within 250 (2D), else walks up to it first.
func (o *StaticObject) click(p *Player, interact bool) {
	s := o.srv
	if p.Target != world.Object(o) {
		s.SetTarget(p, o)
		return
	}
	if !interact {
		return
	}
	if p.Position().Distance2D(o.Position()) > staticInteractionRange {
		s.approachStatic(p, o)
		return
	}
	s.useStatic(p, o)
}

// useStatic shows what a static object shows: a signboard its page, a town
// map the map (Mobius StaticObjectClick in range). A throne does nothing.
func (s *Server) useStatic(p *Player, o *StaticObject) {
	switch o.Template.Type {
	case data.StaticSignboard:
		file := staticPvPPage
		if o.Template.ID == staticTombSignboard {
			file = staticTombPage
		}
		content, ok := s.htmlFile(p, file)
		if !ok {
			content = "<html><body>Signboard is missing:<br>" + htmlPrefix + file + "</body></html>"
		}
		s.sendPage(p, newPage(o.ID(), content))
	case data.StaticTownMap:
		texture := o.Template.Texture
		if texture == "" {
			texture = "none" // Mobius's default
		}
		p.Send(serverpackets.NewShowTownMapPacket(serverpackets.ShowTownMap{
			Texture: "town_map." + texture,
			X:       int32(o.Template.MapX), //nolint:gosec // map coordinates fit
			Y:       int32(o.Template.MapY), //nolint:gosec // as above
		}))
	}
}

// approachStatic walks p up to o to use it (Mobius setIntentionInteract and
// maybeMoveToPawn), as approachNpc does for an NPC; checkStaticApproaches
// clicks again once p arrived.
func (s *Server) approachStatic(p *Player, o *StaticObject) {
	delete(s.doors.approaches, p)
	if p.Sitting {
		return
	}
	s.playerIntendInteract(p)
	if !s.MovePlayerTo(p, o.Position(), targetApproachOffset) {
		return
	}
	a := &staticApproach{obj: o, started: s.Now()}
	if m := p.Move; m != nil {
		a.dest, a.moving = m.Destination, true
	}
	s.doors.approaches[p] = a
	s.world.Broadcast(p, npcMoveToPawnPacket(p, o, targetApproachOffset), true)
}

// checkStaticApproaches is the tick hook of the walks up to static objects:
// a player that stopped clicks the object again (Mobius doInteract calls
// onAction), which uses it in range; one that walks elsewhere, lost the
// object as its target or left gives up.
func (s *Server) checkStaticApproaches(now time.Time) {
	for p, a := range s.doors.approaches {
		o := a.obj
		switch {
		case !p.InGame() || !p.InWorld() || p.Target != world.Object(o) || !o.InWorld():
			delete(s.doors.approaches, p)
		case p.Move == nil:
			delete(s.doors.approaches, p)
			if p.Position().Distance2D(o.Position()) <= staticInteractionRange {
				s.useStatic(p, o)
			}
			p.SendActionFailed()
		case !a.moving || p.Move.Destination != a.dest, now.Sub(a.started) > targetApproachTimeout:
			delete(s.doors.approaches, p)
		}
	}
}
