package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpRequestRestartPoint is the opcode of RequestRestartPoint.
const OpRequestRestartPoint = 0x6d

// Restart points: the buttons of the window a dead player gets with
// serverpackets.Die (the cases of Mobius RequestRestartPoint.portPlayer).
const (
	RestartPointVillage  = 0 // the nearest village; Mobius also sends unknown values there
	RestartPointClanHall = 1
	RestartPointCastle   = 2
	RestartPointSiegeHQ  = 3
	// RestartPointFixed gets up where the player died, with a
	// self-resurrection item or as a GM.
	RestartPointFixed = 4
	// RestartPointAgathion is a to-do in Mobius: nothing happens.
	RestartPointAgathion = 5
	// RestartPointJail is the point Mobius uses for a jailed player,
	// whatever the player chose.
	RestartPointJail = 27
)

// RequestRestartPoint is a dead player's choice of where to get up.
type RequestRestartPoint struct {
	// Point is one of the RestartPoint values, not checked here. Mobius
	// sends unknown values to the village, and overrides the choice of a
	// jailed player (jail) and of a festival participant (point 5).
	Point int32
}

// ParseRequestRestartPoint parses a RequestRestartPoint body (Mobius
// RequestRestartPoint.readImpl): one int.
func ParseRequestRestartPoint(data []byte) (RequestRestartPoint, error) {
	r := packets.NewReader(data)
	req := RequestRestartPoint{Point: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return RequestRestartPoint{}, fmt.Errorf("RequestRestartPoint: %w", err)
	}
	return req, nil
}
