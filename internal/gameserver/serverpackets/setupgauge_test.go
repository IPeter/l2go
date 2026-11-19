package serverpackets

import "testing"

// Mobius SetupGauge.writeImpl: c 0x6D; d color, d time, d time2. The object
// ID Mobius takes is not written.
func TestSetupGaugePacket(t *testing.T) {
	tests := []struct {
		name string
		in   SetupGauge
		want string
	}{
		// A 4-second cast: Mobius sends the time twice.
		{"casting", SetupGauge{Color: GaugeBlue, Time: 4000, MaxTime: 4000}, "6d 00000000 a00f0000 a00f0000"},
		{"bow", SetupGauge{Color: GaugeRed, Time: 1520, MaxTime: 1520}, "6d 01000000 f0050000 f0050000"},
		{"breath, cleared", SetupGauge{Color: GaugeCyan}, "6d 02000000 00000000 00000000"},
		{"pet food", SetupGauge{Color: GaugeGreen, Time: 5000, MaxTime: 10000}, "6d 03000000 88130000 10270000"},
	}
	for _, tt := range tests {
		m67Check(t, tt.name, NewSetupGaugePacket(tt.in), m67Hex(t, tt.want))
	}
}
