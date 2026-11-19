package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// CharCreateFailReason is the reason code of CharCreateFail. The client
// shows a fixed message for each.
type CharCreateFailReason int32

// Interlude reason codes (Mobius CharCreateFail.java).
const (
	CharCreateFailCreationFailed      CharCreateFailReason = 0x00 // "Your character creation has failed."
	CharCreateFailTooManyCharacters   CharCreateFailReason = 0x01 // "You cannot create another character. ..."
	CharCreateFailNameExists          CharCreateFailReason = 0x02 // "This name already exists."
	CharCreateFailNameTooLong         CharCreateFailReason = 0x03 // "... cannot exceed 16 characters ..." (also an empty name)
	CharCreateFailIncorrectName       CharCreateFailReason = 0x04 // "Incorrect name. Please try again."
	CharCreateFailCreationNotAllowed  CharCreateFailReason = 0x05 // "Characters cannot be created from this server."
	CharCreateFailChooseAnotherServer CharCreateFailReason = 0x06 // "... choose another server."
)

// charCreateFailByName maps the reason names scripts give to
// events.reject (see scripting.Reasons) to protocol codes. This table is
// the only place that ties the two together.
var charCreateFailByName = map[string]CharCreateFailReason{
	"creation_failed":      CharCreateFailCreationFailed,
	"too_many_characters":  CharCreateFailTooManyCharacters,
	"name_exists":          CharCreateFailNameExists,
	"name_too_long":        CharCreateFailNameTooLong,
	"incorrect_name":       CharCreateFailIncorrectName,
	"creation_not_allowed": CharCreateFailCreationNotAllowed,
}

// CharCreateFailReasonByName returns the code for a script reason name.
func CharCreateFailReasonByName(name string) (CharCreateFailReason, bool) {
	r, ok := charCreateFailByName[name]
	return r, ok
}

// NewCharCreateFailPacket builds CharCreateFail.
func NewCharCreateFailPacket(reason CharCreateFailReason) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpCharCreateFail)
	w.WriteInt32(int32(reason))
	return w.Bytes()
}
