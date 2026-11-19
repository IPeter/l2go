package clientpackets

// OpRequestShowMiniMap is the opcode of RequestShowMiniMap.
const OpRequestShowMiniMap = 0xcd

// RequestShowMiniMap asks to open the map window. It has no fields; Mobius
// answers with ShowMiniMap.
type RequestShowMiniMap struct{}

// ParseRequestShowMiniMap parses a RequestShowMiniMap body. Mobius reads
// nothing from it, so every body is accepted.
func ParseRequestShowMiniMap([]byte) (RequestShowMiniMap, error) {
	return RequestShowMiniMap{}, nil
}
