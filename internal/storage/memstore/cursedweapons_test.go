package memstore

import (
	"testing"

	"github.com/IPeter/l2go/internal/storage/storagetest"
)

func TestCursedWeaponStore(t *testing.T) {
	storagetest.TestCursedWeaponStore(t, func(*testing.T) (storagetest.CursedWeaponStores, func(error)) {
		s := NewCharacters()
		return storagetest.CursedWeaponStores{Characters: s, Batches: s, CursedWeapons: s}, func(err error) {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.failNext = err
		}
	})
}
