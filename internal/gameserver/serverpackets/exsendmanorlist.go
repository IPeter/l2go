package serverpackets

import (
	"cmp"
	"slices"
	"strings"
)

// ExOpSendManorList is the sub-opcode of ExSendManorList.
const ExOpSendManorList = 0x1b

// ManorCastle is a castle with a manor.
type ManorCastle struct {
	ID   int32 // the castle (residence) ID
	Name string
}

// InterludeCastles returns the nine castles of Interlude, from the Mobius
// castle table. L2Go has no castles yet, so the manor list is always these.
func InterludeCastles() []ManorCastle {
	return []ManorCastle{
		{1, "Gludio"},
		{2, "Dion"},
		{3, "Giran"},
		{4, "Oren"},
		{5, "Aden"},
		{6, "Innadril"},
		{7, "Goddard"},
		{8, "Rune"},
		{9, "Schuttgart"},
	}
}

// NewExSendManorListPacket builds ExSendManorList (Mobius
// ExSendManorList.writeImpl), the answer to RequestManorList. Like Mobius
// it sorts the castles by ID and sends the names in lower case.
func NewExSendManorListPacket(castles []ManorCastle) []byte {
	sorted := slices.SortedStableFunc(slices.Values(castles), func(a, b ManorCastle) int {
		return cmp.Compare(a.ID, b.ID)
	})
	w := newExWriter(ExOpSendManorList, 4+len(sorted)*24)
	putCount32(w, len(sorted))
	for _, c := range sorted {
		w.WriteInt32(c.ID)
		w.WriteString(strings.ToLower(c.Name))
	}
	return w.Bytes()
}
