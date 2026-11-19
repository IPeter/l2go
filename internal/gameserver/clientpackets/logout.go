package clientpackets

// OpLogout is the opcode of Logout.
const OpLogout = 0x09

// Logout asks to leave the game. It has no fields.
type Logout struct{}

// ParseLogout parses a Logout body. Like Mobius it reads nothing, so it
// never fails.
func ParseLogout([]byte) (Logout, error) { return Logout{}, nil }
