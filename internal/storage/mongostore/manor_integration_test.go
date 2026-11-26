//go:build integration

package mongostore

import (
	"testing"

	"github.com/IPeter/l2go/internal/storage/storagetest"
)

func TestManor(t *testing.T) {
	storagetest.TestManorStore(t, func(t *testing.T) (storagetest.ManorStores, func(error)) {
		s, _ := openTestStore(t)
		chars := s.Characters()
		return storagetest.ManorStores{Batches: chars, Manor: chars, Castles: chars}, s.fault.arm
	})
}
