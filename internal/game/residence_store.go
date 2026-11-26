package game

import (
	"slices"

	"github.com/IPeter/l2go/internal/storage"
)

// castleKey is the saver's key of a castle document (persist.go).
func castleKey(id int32) saveKey { return saveKey(int64(id) | 3<<40) }

// mergeCastles adds the castles of b, written after a, to a: a castle of b
// replaces the one of a with its ID.
func mergeCastles(a, b *storage.Batch) {
	for _, c := range b.Castles {
		if i := slices.IndexFunc(a.Castles, func(x storage.Castle) bool { return x.ID == c.ID }); i >= 0 {
			a.Castles[i] = c
		} else {
			a.Castles = append(a.Castles, c)
		}
	}
}
