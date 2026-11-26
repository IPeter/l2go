//go:build integration

package mongostore

import (
	"testing"

	"github.com/IPeter/l2go/internal/storage/storagetest"
)

func TestClanHalls(t *testing.T) {
	storagetest.TestClanHallStore(t, func(t *testing.T) (storagetest.ClanHallStores, func(error)) {
		s, _ := openTestStore(t)
		chars := s.Characters()
		return storagetest.ClanHallStores{Batches: chars, Halls: chars, Clans: s.Clans()}, s.fault.arm
	})
}
