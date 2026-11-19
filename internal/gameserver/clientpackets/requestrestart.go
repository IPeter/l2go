package clientpackets

// OpRequestRestart is the opcode of RequestRestart.
const OpRequestRestart = 0x46

// RequestRestart asks to leave the world and go back to character
// selection. It has no fields.
type RequestRestart struct{}

// ParseRequestRestart parses a RequestRestart body. Like Mobius it reads
// nothing, so it never fails.
func ParseRequestRestart([]byte) (RequestRestart, error) { return RequestRestart{}, nil }
