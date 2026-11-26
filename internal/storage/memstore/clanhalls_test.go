package memstore

import (
	"testing"

	"github.com/IPeter/l2go/internal/storage/storagetest"
)

func TestClanHallStore(t *testing.T) {
	storagetest.TestClanHallStore(t, func(*testing.T) (storagetest.ClanHallStores, func(error)) {
		s := NewCharacters()
		return storagetest.ClanHallStores{Batches: s, Halls: s, Clans: s.Clans()}, func(err error) {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.failNext = err
		}
	})
}
