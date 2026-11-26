package clientpackets

// ExOpRequestExFishRanking is the sub-opcode of the extended packet
// RequestExFishRanking (Mobius ExClientPackets REQUEST_EX_FISH_RANKING).
const ExOpRequestExFishRanking = 0x1f

// RequestExFishRanking asks for the running fishing championship's ranking
// (the client's fish ranking button). It has no fields.
type RequestExFishRanking struct{}

// ParseRequestExFishRanking parses a RequestExFishRanking body, the data
// after the sub-opcode. Like Mobius it reads nothing, so it never fails.
func ParseRequestExFishRanking([]byte) (RequestExFishRanking, error) {
	return RequestExFishRanking{}, nil
}
