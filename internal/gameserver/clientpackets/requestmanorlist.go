package clientpackets

// RequestManorList asks for the castles with a manor (the server answers
// with ExSendManorList). It is the extended packet ExOpRequestManorList and
// has no fields; the client sends it while entering the world.
type RequestManorList struct{}

// ParseRequestManorList parses a RequestManorList body, the ExPacket.Body
// after the sub-opcode. Like Mobius it reads nothing, so it never fails.
func ParseRequestManorList([]byte) (RequestManorList, error) { return RequestManorList{}, nil }
