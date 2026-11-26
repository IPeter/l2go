package memstore

import (
	"testing"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/storagetest"
)

func TestBBS(t *testing.T) {
	storagetest.TestBBSStore(t, func(*testing.T) (storage.BBSStore, func(error)) {
		s := NewBBS()
		return s, s.FailNext
	})
}

// TestBBSZeroValue checks that a zero BBS works.
func TestBBSZeroValue(t *testing.T) {
	var s BBS
	if err := s.SaveBBSMails(t.Context(), []storage.BBSMail{{ID: 1}}); err != nil {
		t.Fatalf("SaveBBSMails: %v", err)
	}
	if got, err := s.LoadBBSMails(t.Context()); err != nil || len(got) != 1 {
		t.Errorf("LoadBBSMails = %+v, %v", got, err)
	}
}
