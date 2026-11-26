//go:build integration

package mongostore

import (
	"testing"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/storagetest"
)

func TestSevenSigns(t *testing.T) {
	storagetest.TestSevenSignsStore(t, func(t *testing.T) (storage.SevenSignsStore, func(error)) {
		s, _ := openTestStore(t)
		return s.SevenSigns(), s.fault.arm
	})
}
