package game

import (
	"errors"
	"maps"
	"slices"
	"strconv"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
)

// The door GM commands (Mobius AdminDoorControl): //open and //close with a
// door ID or on the targeted door, //openall, //closeall and //showdoors.

// The colours //showdoors draws in (java.awt.Color).
const (
	doorShowOpen   = 0x00ff00 // Color.GREEN
	doorShowClosed = 0xff0000 // Color.RED
	doorShowOff    = 0x000000 // Color.BLACK
	// doorShowZ is the Z of the drawing's middle and of the "off" line.
	doorShowZ = -16000
)

func (s *Server) setupDoorGM() error {
	admin := storage.AccessLevelAdmin
	return errors.Join(
		s.RegisterGMCommand("open", admin, s.gmDoor(true)),
		s.RegisterGMCommand("close", admin, s.gmDoor(false)),
		s.RegisterGMCommand("openall", admin, s.gmDoorAll(true)),
		s.RegisterGMCommand("closeall", admin, s.gmDoorAll(false)),
		s.RegisterGMCommand("showdoors", admin, s.gmShowDoors),
	)
}

// gmDoor is //open and //close: "//open <door ID>" opens that door,
// "//open" the targeted one ("Incorrect target." otherwise). A bad or
// unknown door ID does nothing, as in Mobius (it logs the exception).
func (s *Server) gmDoor(open bool) GMCommand {
	return func(p *Player, args []string) {
		var d *Door
		if len(args) == 0 {
			t, ok := p.Target.(*Door)
			if !ok {
				p.SendMessage("Incorrect target.")
				return
			}
			d = t
		} else {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				s.log.Warn("game: problem with AdminDoorControl", append(p.logArgs(), "err", err)...)
				return
			}
			if d = s.doors.byID[id]; d == nil {
				return
			}
		}
		if open {
			s.openDoor(d)
		} else {
			s.closeDoor(d)
		}
	}
}

// gmDoorAll is //openall and //closeall: every door, in door ID order.
func (s *Server) gmDoorAll(open bool) GMCommand {
	return func(*Player, []string) {
		for _, d := range s.doors.list {
			if open {
				s.openDoor(d)
			} else {
				s.closeDoor(d)
			}
		}
	}
}

// gmShowDoors is //showdoors: each door the GM sees and has not been shown
// yet is drawn (green open, red closed) with "Found door <ID>";
// "//showdoors off" replaces each drawing with a black dot and forgets them.
func (s *Server) gmShowDoors(p *Player, args []string) {
	pos := p.Position()
	if len(args) > 0 && args[0] == "off" {
		for _, id := range slices.Sorted(maps.Keys(s.doors.shown[p])) {
			p.Send(serverpackets.NewExServerPrimitivePacket(serverpackets.ExServerPrimitive{
				Name: "Door" + strconv.Itoa(id), X: pos.X, Y: pos.Y, Z: doorShowZ,
				Lines: []serverpackets.PrimitiveLine{{
					PrimitivePoint: serverpackets.PrimitivePoint{Color: doorShowOff, X: pos.X, Y: pos.Y, Z: doorShowZ},
					X2:             pos.X, Y2: pos.Y, Z2: doorShowZ,
				}},
			}))
		}
		delete(s.doors.shown, p)
		return
	}
	shown := s.doors.shown[p]
	if shown == nil {
		shown = make(map[int]bool)
		s.doors.shown[p] = shown
	}
	for o := range s.world.Visible(p) {
		d, ok := o.(*Door)
		if !ok || shown[d.Template.ID] {
			continue
		}
		shown[d.Template.ID] = true
		p.Send(doorDrawing(d, pos.X, pos.Y))
		p.SendMessage("Found door " + strconv.Itoa(d.Template.ID))
	}
}

// doorDrawing is the //showdoors drawing of a door: its two boxes and the
// diagonals, in Mobius's line order.
func doorDrawing(d *Door, x, y int32) []byte {
	color := int32(doorShowClosed)
	if d.open {
		color = doorShowOpen
	}
	nx, ny := d.shape.X, d.shape.Y
	lo, hi := d.shape.ZMin, d.shape.ZMax
	line := func(i int, zi int32, j int, zj int32) serverpackets.PrimitiveLine {
		return serverpackets.PrimitiveLine{
			PrimitivePoint: serverpackets.PrimitivePoint{Color: color, X: nx[i], Y: ny[i], Z: zi},
			X2:             nx[j], Y2: ny[j], Z2: zj,
		}
	}
	return serverpackets.NewExServerPrimitivePacket(serverpackets.ExServerPrimitive{
		Name: "Door" + strconv.Itoa(d.Template.ID), X: x, Y: y, Z: doorShowZ,
		Lines: []serverpackets.PrimitiveLine{
			// box 1
			line(0, lo, 1, lo), line(1, lo, 2, hi), line(2, hi, 3, hi), line(3, hi, 0, lo),
			// box 2
			line(0, hi, 1, hi), line(1, hi, 2, lo), line(2, lo, 3, lo), line(3, lo, 0, hi),
			// diagonals
			line(0, lo, 1, hi), line(2, lo, 3, hi), line(0, hi, 1, lo), line(2, hi, 3, lo),
		},
	})
}
