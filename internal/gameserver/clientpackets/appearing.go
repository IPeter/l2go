package clientpackets

// OpAppearing is the opcode of Appearing.
const OpAppearing = 0x30

// Appearing tells the server that the client finished loading after
// TeleportToLocation. It has no fields.
type Appearing struct{}

// ParseAppearing parses an Appearing body. Like Mobius it reads nothing,
// so it never fails.
func ParseAppearing([]byte) (Appearing, error) { return Appearing{}, nil }
