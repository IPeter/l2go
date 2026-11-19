package clientpackets

import (
	"errors"
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpRequestShortcutReg is the opcode of RequestShortcutReg (Mobius
// REQUEST_SHORT_CUT_REG).
const OpRequestShortcutReg = 0x33

// shortcutSlotsPerPage is the number of slots on one page of the shortcut
// bar. The client numbers the slots across the pages.
const shortcutSlotsPerPage = 12

// shortcutTypeEnd is the number of Mobius ShortcutType values. Mobius looks
// a type of 1 to 6 up in that enum, so 6 is past its end.
const shortcutTypeEnd = 6

// ErrUnknownShortcutType reports the shortcut type 6, which Mobius fails to
// read (its ShortcutType lookup throws) and drops.
var ErrUnknownShortcutType = errors.New("clientpackets: unknown shortcut type")

// RequestShortcutReg puts an item, skill, action, macro or recipe on a slot
// of the shortcut bar. Mobius answers with ShortcutRegister.
type RequestShortcutReg struct {
	// Type is a serverpackets.ShortcutType value: 1 item, 2 skill, 3
	// action, 4 macro, 5 recipe. Mobius reads values below 1 or above 6 as
	// 0 (none), which it ignores.
	Type int32
	// Slot (0-11) and Page are split from the slot number the client sends
	// as Mobius does: number % 12 and number / 12, both rounded toward zero,
	// so a negative number gives a negative slot. Mobius ignores pages
	// outside 0-10.
	Slot, Page int32
	// ID is the item object ID, skill ID, action ID, macro ID or recipe
	// ID, depending on Type.
	ID int32
	// Level is the skill level the client sends. Mobius reads and ignores
	// it: a skill shortcut gets the level the player knows.
	Level int32
}

// ParseRequestShortcutReg parses a RequestShortcutReg body (Mobius
// RequestShortcutReg.readImpl): d type, d slot number, d ID, d level. The
// type 6 fails with ErrUnknownShortcutType.
func ParseRequestShortcutReg(data []byte) (RequestShortcutReg, error) {
	r := packets.NewReader(data)
	typ := r.ReadInt32()
	number := r.ReadInt32()
	req := RequestShortcutReg{
		ID:    r.ReadInt32(),
		Level: r.ReadInt32(),
	}
	if err := r.Err(); err != nil {
		return RequestShortcutReg{}, fmt.Errorf("RequestShortcutReg: %w", err)
	}
	switch {
	case typ == shortcutTypeEnd:
		return RequestShortcutReg{}, fmt.Errorf("RequestShortcutReg: %w: %d", ErrUnknownShortcutType, typ)
	case typ >= 1 && typ < shortcutTypeEnd:
		req.Type = typ
	}
	req.Slot, req.Page = number%shortcutSlotsPerPage, number/shortcutSlotsPerPage
	return req, nil
}
