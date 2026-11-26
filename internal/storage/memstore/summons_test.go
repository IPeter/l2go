package memstore

import (
	"testing"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/storagetest"
)

func TestSummons(t *testing.T) {
	storagetest.TestSummonStore(t, func(*testing.T) (storage.SummonStore, func(error)) {
		s := NewSummons()
		return s, s.FailNext
	})
}
