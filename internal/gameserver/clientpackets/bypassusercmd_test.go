package clientpackets

import "testing"

// Mobius BypassUserCmd.readImpl: d command ID (4 bytes).

func TestParseBypassUserCmd(t *testing.T) {
	body := []byte{0x00, 0x00, 0x00, 0x00}
	got, err := ParseBypassUserCmd(body)
	m3Parsed(t, "BypassUserCmd /loc", got, err, BypassUserCmd{Command: UserCmdLoc})

	got, err = ParseBypassUserCmd([]byte{77, 0, 0, 0, 0xaa})
	m3Parsed(t, "BypassUserCmd with trailing data", got, err, BypassUserCmd{Command: 77})

	m3Truncations(t, "BypassUserCmd", body, ParseBypassUserCmd)

	got, err = ParseBypassUserCmd([]byte{0xde, 0xad, 0xbe, 0xef})
	m3Parsed(t, "BypassUserCmd garbage", got, err, BypassUserCmd{Command: -0x10415222}) // 0xefbeadde
}

func FuzzParseBypassUserCmd(f *testing.F) {
	m3Fuzz(f, ParseBypassUserCmd, 4, []byte{0, 0, 0, 0}, []byte{52, 0})
}
