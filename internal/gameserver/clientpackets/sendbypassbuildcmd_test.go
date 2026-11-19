package clientpackets

import "testing"

// Mobius SendBypassBuildCmd.readImpl: S command, then String.trim().

func TestParseSendBypassBuildCmd(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want string
	}{
		{"teleport", new(m3Body).s("teleport -71338 258271 -3104").b, "teleport -71338 258271 -3104"},
		{"loc", new(m3Body).s("loc").b, "loc"},
		{"trimmed", new(m3Body).s("  gmspeed 2 \t\r\n").b, "gmspeed 2"},
		{"control characters", new(m3Body).s("\x01\x1fheal\x00").b, "heal"}, // the string ends at the zero
		{"inner spaces kept", new(m3Body).s(" a  b ").b, "a  b"},
		{"not ASCII kept", new(m3Body).s(" announce hé　").b, " announce hé　"},
		{"empty", new(m3Body).s("").b, ""},
		{"only spaces", new(m3Body).s("   ").b, ""},
		{"trailing data", append(new(m3Body).s("loc").b, 0x33, 0x00), "loc"},
	}
	for _, tt := range tests {
		got, err := ParseSendBypassBuildCmd(tt.body)
		m3Parsed(t, tt.name, got, err, SendBypassBuildCmd{Command: tt.want})
	}
	m3Truncations(t, "SendBypassBuildCmd", new(m3Body).s("loc").b, ParseSendBypassBuildCmd)
}

func TestParseSendBypassBuildCmdGarbage(t *testing.T) {
	inputs := [][]byte{
		nil,
		{0x6c},                         // half a code unit
		{0x6c, 0x00, 0x6f, 0x00, 0x63}, // no terminator, odd length
		{0x6c, 0x00, 0x6f, 0x00},       // no terminator
	}
	for _, in := range inputs {
		got, err := ParseSendBypassBuildCmd(in)
		m3Failed(t, "SendBypassBuildCmd", got, err)
	}
}

func FuzzParseSendBypassBuildCmd(f *testing.F) {
	m3Fuzz(f, ParseSendBypassBuildCmd, -1, new(m3Body).s("teleport 1 2 3").b, new(m3Body).s(" ").b)
}
