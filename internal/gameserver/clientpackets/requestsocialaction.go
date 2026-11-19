package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpRequestSocialAction is the opcode of RequestSocialAction.
const OpRequestSocialAction = 0x1b

// The social actions a player may ask for (Mobius RequestSocialAction).
// The others are played by the server, such as the level-up animation;
// Mobius treats a request for one as an illegal action.
const (
	MinSocialAction = 2
	MaxSocialAction = 13
)

// RequestSocialAction asks to play a social action (a greeting, a bow, a
// dance...) from the Actions window.
type RequestSocialAction struct {
	ActionID int32
}

// Allowed reports whether players may ask for the action.
func (r RequestSocialAction) Allowed() bool {
	return r.ActionID >= MinSocialAction && r.ActionID <= MaxSocialAction
}

// ParseRequestSocialAction parses a RequestSocialAction body (Mobius
// RequestSocialAction.readImpl).
func ParseRequestSocialAction(data []byte) (RequestSocialAction, error) {
	r := packets.NewReader(data)
	p := RequestSocialAction{ActionID: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return RequestSocialAction{}, fmt.Errorf("RequestSocialAction: %w", err)
	}
	return p, nil
}
