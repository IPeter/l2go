package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpRequestShortcutDel is the opcode of RequestShortcutDel (Mobius
// REQUEST_SHORT_CUT_DEL).
const OpRequestShortcutDel = 0x35

// RequestShortcutDel clears a slot of the shortcut bar. When the slot had a
// shortcut, Mobius answers with the whole bar (ShortcutInit).
type RequestShortcutDel struct {
	// Slot and Page are split from the slot number the client sends, as in
	// RequestShortcutReg. Mobius ignores pages outside 0-10.
	Slot, Page int32
}

// ParseRequestShortcutDel parses a RequestShortcutDel body (Mobius
// RequestShortcutDel.readImpl): d slot number.
func ParseRequestShortcutDel(data []byte) (RequestShortcutDel, error) {
	r := packets.NewReader(data)
	number := r.ReadInt32()
	if err := r.Err(); err != nil {
		return RequestShortcutDel{}, fmt.Errorf("RequestShortcutDel: %w", err)
	}
	return RequestShortcutDel{
		Slot: number % shortcutSlotsPerPage,
		Page: number / shortcutSlotsPerPage,
	}, nil
}
