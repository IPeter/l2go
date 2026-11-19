package serverpackets

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/packets"
)

// Fixed values Mobius writes around each base stat of a template. The
// client ignores them.
const (
	templateStatMax = 0x46
	templateStatMin = 0x0a
)

// NewCharacterSuccessPacket builds NewCharacterSuccess, which lists the
// classes the character creation screen offers, in the given order.
func NewCharacterSuccessPacket(templates []data.Template) []byte {
	w := packets.NewWriter(5 + len(templates)*80)
	w.WriteUint8(OpNewCharacterSuccess)
	w.WriteInt32(int32(len(templates))) //nolint:gosec // a handful of templates
	for _, t := range templates {
		w.WriteInt32(int32(t.Race))    //nolint:gosec // small game data values
		w.WriteInt32(int32(t.ClassID)) //nolint:gosec // small game data values
		s := t.Stats
		for _, v := range []int{s.STR, s.DEX, s.CON, s.INT, s.WIT, s.MEN} {
			w.WriteInt32(templateStatMax)
			w.WriteInt32(int32(v)) //nolint:gosec // small game data values
			w.WriteInt32(templateStatMin)
		}
	}
	return w.Bytes()
}
