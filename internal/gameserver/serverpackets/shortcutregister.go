package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpShortcutRegister is the opcode of ShortcutRegister (Mobius
// SHORT_CUT_REGISTER).
const OpShortcutRegister = 0x44

// NewShortcutRegisterPacket builds ShortcutRegister (Mobius
// ShortcutRegister.writeImpl): one new or changed slot of the shortcut bar,
// the answer to RequestShortcutReg.
//
// The layout is shorter than a ShortcutInit entry: an item shortcut sends
// only its ID, a skill shortcut its ID, its level and a zero byte, and
// every shortcut, ShortcutNone too, ends with an int 1.
func NewShortcutRegisterPacket(sc Shortcut) []byte {
	w := packets.NewWriter(22)
	w.WriteUint8(OpShortcutRegister)
	w.WriteInt32(int32(sc.Type))
	w.WriteInt32(sc.Slot + sc.Page*shortcutsPerPage)
	switch sc.Type {
	case ShortcutItem, ShortcutAction, ShortcutMacro, ShortcutRecipe:
		w.WriteInt32(sc.ID)
	case ShortcutSkill:
		w.WriteInt32(sc.ID)
		w.WriteInt32(sc.Level)
		w.WriteUint8(0) // C5 addition
	}
	w.WriteInt32(1)
	return w.Bytes()
}
