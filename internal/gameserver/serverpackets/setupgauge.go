package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpSetupGauge is the opcode of SetupGauge.
const OpSetupGauge = 0x6d

// GaugeColor is the color of the bar SetupGauge shows.
type GaugeColor int32

// Gauge colors (Mobius SetupGauge.BLUE, RED and CYAN; green is only named in
// a comment there). The comments say what Mobius uses each one for.
const (
	GaugeBlue  GaugeColor = 0 // casting a skill, crafting
	GaugeRed   GaugeColor = 1 // drawing a bow
	GaugeCyan  GaugeColor = 2 // breath under water
	GaugeGreen GaugeColor = 3 // a pet's food
)

// SetupGauge shows a bar over the player's status window that fills up, such
// as the casting bar. Only the player it concerns gets it; Mobius takes the
// player's object ID too but does not send it.
type SetupGauge struct {
	Color GaugeColor
	// Time is the current value of the bar and MaxTime the full one. For a
	// timed bar both are its duration in milliseconds: Mobius sends the
	// same value twice. The pet food bar sends the food left and the full
	// stomach instead.
	Time    int32
	MaxTime int32
}

// NewSetupGaugePacket builds SetupGauge (Mobius SetupGauge.writeImpl).
func NewSetupGaugePacket(g SetupGauge) []byte {
	w := packets.NewWriter(13)
	w.WriteUint8(OpSetupGauge)
	w.WriteInt32(int32(g.Color))
	w.WriteInt32(g.Time)
	w.WriteInt32(g.MaxTime)
	return w.Bytes()
}
