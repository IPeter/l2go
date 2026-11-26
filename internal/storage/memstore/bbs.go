package memstore

import (
	"context"
	"maps"
	"slices"
	"sync"

	"github.com/IPeter/l2go/internal/storage"
)

// BBS is an in-memory storage.BBSStore. The zero value is ready to use and
// it is safe for concurrent use.
type BBS struct {
	mu        sync.Mutex
	mails     map[int32]storage.BBSMail
	favorites map[int32]storage.BBSFavorite

	// failNext, if set, fails the next write before it changes anything.
	failNext error
}

var _ storage.BBSStore = (*BBS)(nil)

// NewBBS returns an empty store.
func NewBBS() *BBS { return &BBS{} }

// FailNext makes the next write fail with err before it changes anything.
// For tests of the callers.
func (s *BBS) FailNext(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failNext = err
}

// write runs fn under the lock unless the next write is to fail.
func (s *BBS) write(ctx context.Context, fn func()) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.failNext; err != nil {
		s.failNext = nil
		return err
	}
	fn()
	return nil
}

// LoadBBSMails implements storage.BBSStore.
func (s *BBS) LoadBBSMails(ctx context.Context) ([]storage.BBSMail, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []storage.BBSMail
	for _, id := range slices.Sorted(maps.Keys(s.mails)) {
		out = append(out, s.mails[id])
	}
	return out, nil
}

// SaveBBSMails implements storage.BBSStore.
func (s *BBS) SaveBBSMails(ctx context.Context, mails []storage.BBSMail) error {
	return s.write(ctx, func() {
		if s.mails == nil {
			s.mails = make(map[int32]storage.BBSMail)
		}
		for _, m := range mails {
			s.mails[m.ID] = m
		}
	})
}

// DeleteBBSMail implements storage.BBSStore.
func (s *BBS) DeleteBBSMail(ctx context.Context, id int32) error {
	return s.write(ctx, func() { delete(s.mails, id) })
}

// LoadBBSFavorites implements storage.BBSStore.
func (s *BBS) LoadBBSFavorites(ctx context.Context) ([]storage.BBSFavorite, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []storage.BBSFavorite
	for _, id := range slices.Sorted(maps.Keys(s.favorites)) {
		out = append(out, s.favorites[id])
	}
	return out, nil
}

// SaveBBSFavorite implements storage.BBSStore.
func (s *BBS) SaveBBSFavorite(ctx context.Context, f storage.BBSFavorite) error {
	return s.write(ctx, func() {
		if s.favorites == nil {
			s.favorites = make(map[int32]storage.BBSFavorite)
		}
		s.favorites[f.ID] = f
	})
}

// DeleteBBSFavorite implements storage.BBSStore.
func (s *BBS) DeleteBBSFavorite(ctx context.Context, playerID, id int32) error {
	return s.write(ctx, func() {
		if f, ok := s.favorites[id]; ok && f.PlayerID == playerID {
			delete(s.favorites, id)
		}
	})
}
