//go:build integration

package mongostore

import (
	"testing"

	"github.com/IPeter/l2go/internal/storage/storagetest"
)

func TestCastles(t *testing.T) {
	storagetest.TestCastleStore(t, func(t *testing.T) (storagetest.CastleStores, func(error)) {
		s, _ := openTestStore(t)
		chars := s.Characters()
		return storagetest.CastleStores{Batches: chars, Castles: chars, Clans: s.Clans()}, s.fault.arm
	})
}
