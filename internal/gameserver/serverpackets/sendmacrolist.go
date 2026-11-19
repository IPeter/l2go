package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpSendMacroList is the opcode of SendMacroList.
const OpSendMacroList = 0xe7

// MacroCommandType is the kind of a macro command (the ordinals of Mobius
// MacroType).
type MacroCommandType uint8

// Macro command types.
const (
	MacroCommandNone     MacroCommandType = 0
	MacroCommandSkill    MacroCommandType = 1
	MacroCommandAction   MacroCommandType = 2
	MacroCommandText     MacroCommandType = 3
	MacroCommandShortcut MacroCommandType = 4
	MacroCommandItem     MacroCommandType = 5
	MacroCommandDelay    MacroCommandType = 6
)

// MacroCommand is one line of a macro.
type MacroCommand struct {
	Type    MacroCommandType
	D1      int32 // skill ID, action ID, item ID or delay
	D2      uint8 // shortcut page and slot
	Command string
}

// Macro is one user macro.
type Macro struct {
	ID          int32
	Name        string
	Description string
	Acronym     string
	Icon        uint8
	Commands    []MacroCommand
}

// SendMacroList carries one macro of the player's list. Mobius sends one
// packet per macro, each with the total Count, or a single packet with
// Count 0 and no Macro for an empty list.
type SendMacroList struct {
	// Revision changes whenever the list changes. Mobius starts at 1 and
	// adds 1 before each send, so the list sent by EnterWorld has 2.
	Revision int32
	Count    uint8
	Macro    *Macro // nil for an empty list
}

// NewSendMacroListPacket builds SendMacroList (Mobius
// SendMacroList.writeImpl).
func NewSendMacroListPacket(l SendMacroList) []byte {
	w := packets.NewWriter(8)
	w.WriteUint8(OpSendMacroList)
	w.WriteInt32(l.Revision)
	w.WriteUint8(0) // unknown
	w.WriteUint8(l.Count)
	putBool8(w, l.Macro != nil)
	if m := l.Macro; m != nil {
		w.WriteInt32(m.ID)
		w.WriteString(m.Name)
		w.WriteString(m.Description)
		w.WriteString(m.Acronym)
		w.WriteUint8(m.Icon)
		w.WriteUint8(uint8(len(m.Commands))) //nolint:gosec // a macro has at most 12 commands
		for i, c := range m.Commands {
			w.WriteUint8(uint8(i + 1)) //nolint:gosec // as above
			w.WriteUint8(uint8(c.Type))
			w.WriteInt32(c.D1)
			w.WriteUint8(c.D2)
			w.WriteString(c.Command)
		}
	}
	return w.Bytes()
}
