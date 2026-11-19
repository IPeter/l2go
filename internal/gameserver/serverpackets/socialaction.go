package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpSocialAction is the opcode of SocialAction.
const OpSocialAction = 0x2d

// SocialActionLevelUp is the action Mobius plays when a character gains a
// level. Players may only ask for actions 2 to 13 themselves (see
// clientpackets.RequestSocialAction).
const SocialActionLevelUp = 15

// SocialAction plays an animation (a greeting, a bow, a dance...) on a
// creature, for everyone who sees it.
type SocialAction struct {
	ObjectID int32
	ActionID int32
}

// NewSocialActionPacket builds SocialAction (Mobius SocialAction.writeImpl).
func NewSocialActionPacket(s SocialAction) []byte {
	w := packets.NewWriter(9)
	w.WriteUint8(OpSocialAction)
	w.WriteInt32(s.ObjectID)
	w.WriteInt32(s.ActionID)
	return w.Bytes()
}
