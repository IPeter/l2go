package clientpackets

// Opcodes of the alliance packets (Mobius ClientPackets).
// The parser of RequestSetAllyCrest is in crest.go.
const (
	OpRequestJoinAlly       = 0x82
	OpRequestAnswerJoinAlly = 0x83
	OpAllyLeave             = 0x84
	OpAllyDismiss           = 0x85
	OpRequestDismissAlly    = 0x86
	OpRequestSetAllyCrest   = 0x87
	OpRequestAllyCrest      = 0x88
	OpRequestAllyInfo       = 0x8e
)

// RequestJoinAlly invites a clan leader's clan into the alliance.
type RequestJoinAlly struct {
	TargetID int32 // the invited clan leader's object ID
}

// ParseRequestJoinAlly parses a RequestJoinAlly body (Mobius
// RequestJoinAlly.readImpl): d object ID.
func ParseRequestJoinAlly(data []byte) (RequestJoinAlly, error) {
	v, err := readClanInt32("RequestJoinAlly", data)
	return RequestJoinAlly{TargetID: v}, err
}

// RequestAnswerJoinAlly answers an alliance invitation.
type RequestAnswerJoinAlly struct {
	Answer int32 // 0 refuses, other values accept
}

// ParseRequestAnswerJoinAlly parses a RequestAnswerJoinAlly body (Mobius
// RequestAnswerJoinAlly.readImpl): d answer.
func ParseRequestAnswerJoinAlly(data []byte) (RequestAnswerJoinAlly, error) {
	v, err := readClanInt32("RequestAnswerJoinAlly", data)
	return RequestAnswerJoinAlly{Answer: v}, err
}

// AllyLeave takes the player's clan out of its alliance. It has no fields.
type AllyLeave struct{}

// ParseAllyLeave parses an AllyLeave body. Like Mobius it reads nothing, so
// it never fails.
func ParseAllyLeave([]byte) (AllyLeave, error) { return AllyLeave{}, nil }

// AllyDismiss expels a clan from the alliance.
type AllyDismiss struct {
	ClanName string
}

// ParseAllyDismiss parses an AllyDismiss body (Mobius AllyDismiss.readImpl):
// S clan name.
func ParseAllyDismiss(data []byte) (AllyDismiss, error) {
	s, err := readClanString("AllyDismiss", data)
	return AllyDismiss{ClanName: s}, err
}

// RequestDismissAlly dissolves the alliance. It has no fields.
type RequestDismissAlly struct{}

// ParseRequestDismissAlly parses a RequestDismissAlly body. Like Mobius it
// reads nothing, so it never fails.
func ParseRequestDismissAlly([]byte) (RequestDismissAlly, error) { return RequestDismissAlly{}, nil }

// RequestAllyCrest asks for the image of an alliance crest (AllyCrest).
type RequestAllyCrest struct {
	CrestID int32
}

// ParseRequestAllyCrest parses a RequestAllyCrest body (Mobius
// RequestAllyCrest.readImpl): d crest ID.
func ParseRequestAllyCrest(data []byte) (RequestAllyCrest, error) {
	v, err := readClanInt32("RequestAllyCrest", data)
	return RequestAllyCrest{CrestID: v}, err
}

// RequestAllyInfo asks for the alliance window (AllianceInfo). It has no
// fields.
type RequestAllyInfo struct{}

// ParseRequestAllyInfo parses a RequestAllyInfo body. Like Mobius it reads
// nothing, so it never fails.
func ParseRequestAllyInfo([]byte) (RequestAllyInfo, error) { return RequestAllyInfo{}, nil }
