package memstore

import (
	"testing"

	"github.com/IPeter/l2go/internal/storage/storagetest"
)

func TestCastleStore(t *testing.T) {
	storagetest.TestCastleStore(t, func(*testing.T) (storagetest.CastleStores, func(error)) {
		s := NewCharacters()
		return storagetest.CastleStores{Batches: s, Castles: s, Clans: s.Clans()}, func(err error) {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.failNext = err
		}
	})
}
