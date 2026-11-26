//go:build integration

package mongostore

import (
	"testing"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/storagetest"
)

func TestSummons(t *testing.T) {
	storagetest.TestSummonStore(t, func(t *testing.T) (storage.SummonStore, func(error)) {
		s, _ := openTestStore(t)
		return s.Summons(), s.fault.arm
	})
}
