package mongostore

import (
	"cmp"
	"slices"

	"github.com/IPeter/l2go/internal/storage"
)

// hennaDoc is one worn dye of storage.Character.Hennas, a row of Mobius
// character_hennas with its column names (decision X3: per class index).
type hennaDoc struct {
	ClassIndex int32 `bson:"class_index"`
	Slot       int32 `bson:"slot"` // 1-3
	SymbolID   int32 `bson:"symbol_id"`
}

// toHennaDocs returns the rows of the worn dyes, by class index and slot,
// nil for none.
func toHennaDocs(hennas map[int32]storage.HennaSlots) []hennaDoc {
	var docs []hennaDoc
	for class, slots := range hennas {
		for i, id := range slots {
			if id != 0 {
				docs = append(docs, hennaDoc{ClassIndex: class, Slot: int32(i + 1), SymbolID: id}) //nolint:gosec // slots 1-3
			}
		}
	}
	slices.SortFunc(docs, func(a, b hennaDoc) int {
		return cmp.Or(cmp.Compare(a.ClassIndex, b.ClassIndex), cmp.Compare(a.Slot, b.Slot))
	})
	return docs
}

// fromHennaDocs returns the dyes of the rows, nil for none. Rows with a slot
// outside 1-3 or symbol 0 are skipped, as Mobius restoreHenna skips them.
func fromHennaDocs(docs []hennaDoc) map[int32]storage.HennaSlots {
	hennas := make(map[int32]storage.HennaSlots)
	for _, d := range docs {
		if d.Slot < 1 || d.Slot > 3 || d.SymbolID == 0 {
			continue
		}
		s := hennas[d.ClassIndex]
		s[d.Slot-1] = d.SymbolID
		hennas[d.ClassIndex] = s
	}
	return storage.CloneHennas(hennas)
}
