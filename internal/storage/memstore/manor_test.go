package memstore

import (
	"testing"

	"github.com/IPeter/l2go/internal/storage/storagetest"
)

func TestManorStore(t *testing.T) {
	storagetest.TestManorStore(t, func(*testing.T) (storagetest.ManorStores, func(error)) {
		s := NewCharacters()
		return storagetest.ManorStores{Batches: s, Manor: s, Castles: s}, func(err error) {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.failNext = err
		}
	})
}
