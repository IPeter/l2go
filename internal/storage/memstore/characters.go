package memstore

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"sync"
	"time"

	"github.com/IPeter/l2go/internal/storage"
)

// Characters is an in-memory storage.CharacterStore. The zero value is ready
// to use and it is safe for concurrent use.
type Characters struct {
	mu     sync.Mutex
	byID   map[int32]storage.Character
	byName map[string]int32 // storage.NameKey(name) → object ID
}

var _ storage.CharacterStore = (*Characters)(nil)

// NewCharacters returns an empty store.
func NewCharacters() *Characters {
	return &Characters{}
}

// Create implements storage.CharacterStore.
func (s *Characters) Create(ctx context.Context, c *storage.Character) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := storage.NameKey(c.Name)
	if _, ok := s.byName[key]; ok {
		return storage.ErrExists
	}
	if _, ok := s.byID[c.ObjectID]; ok {
		return storage.ErrExists
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = normalizeTime(time.Now())
	}
	if s.byID == nil {
		s.byID = make(map[int32]storage.Character)
		s.byName = make(map[string]int32)
	}
	stored := cloneCharacter(*c)
	stored.CreatedAt = normalizeTime(stored.CreatedAt)
	stored.DeleteTime = normalizeTime(stored.DeleteTime)
	stored.LastAccess = normalizeTime(stored.LastAccess)
	s.byID[c.ObjectID] = stored
	s.byName[key] = c.ObjectID
	return nil
}

// ListByAccount implements storage.CharacterStore.
func (s *Characters) ListByAccount(ctx context.Context, account string) ([]storage.Character, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var list []storage.Character
	for _, c := range s.byID {
		if c.Account == account {
			list = append(list, cloneCharacter(c))
		}
	}
	slices.SortFunc(list, func(a, b storage.Character) int {
		return cmp.Or(a.CreatedAt.Compare(b.CreatedAt), cmp.Compare(a.ObjectID, b.ObjectID))
	})
	return list, nil
}

// CountByAccount implements storage.CharacterStore.
func (s *Characters) CountByAccount(ctx context.Context, account string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, c := range s.byID {
		if c.Account == account {
			n++
		}
	}
	return n, nil
}

// NameExists implements storage.CharacterStore.
func (s *Characters) NameExists(ctx context.Context, name string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.byName[storage.NameKey(name)]
	return ok, nil
}

// Get implements storage.CharacterStore.
func (s *Characters) Get(ctx context.Context, objectID int32) (storage.Character, error) {
	if err := ctx.Err(); err != nil {
		return storage.Character{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.byID[objectID]
	if !ok {
		return storage.Character{}, storage.ErrNotFound
	}
	return cloneCharacter(c), nil
}

// cloneCharacter returns a copy of c that shares no memory with it. A
// character without items gets a nil Items slice, as from MongoDB.
func cloneCharacter(c storage.Character) storage.Character {
	if len(c.Items) == 0 {
		c.Items = nil
	} else {
		c.Items = slices.Clone(c.Items)
	}
	return c
}

// normalizeTime returns t as MongoDB would store it: millisecond precision,
// in UTC, without a monotonic clock reading.
func normalizeTime(t time.Time) time.Time {
	return t.UTC().Truncate(time.Millisecond)
}

// IDs is an in-memory storage.IDAllocator. IDs are unique within one IDs
// value only; they start over in a new process. The zero value is ready to
// use and it is safe for concurrent use.
type IDs struct {
	mu   sync.Mutex
	used int64 // how many IDs were handed out
}

var _ storage.IDAllocator = (*IDs)(nil)

// NewIDs returns an allocator that starts at storage.FirstObjectID.
func NewIDs() *IDs {
	return &IDs{}
}

var errIDsExhausted = errors.New("memstore: object IDs exhausted")

// NextID implements storage.IDAllocator.
func (a *IDs) NextID(ctx context.Context) (int32, error) {
	ids, err := a.NextIDs(ctx, 1)
	if err != nil {
		return 0, err
	}
	return ids[0], nil
}

// NextIDs implements storage.IDAllocator.
func (a *IDs) NextIDs(ctx context.Context, n int) ([]int32, error) {
	if n < 1 {
		return nil, fmt.Errorf("memstore: NextIDs(%d): n must be at least 1", n)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	first := int64(storage.FirstObjectID) + a.used
	if first+int64(n)-1 > math.MaxInt32 {
		return nil, errIDsExhausted
	}
	a.used += int64(n)
	ids := make([]int32, n)
	for i := range ids {
		ids[i] = int32(first + int64(i)) //nolint:gosec // range checked against math.MaxInt32 above
	}
	return ids, nil
}
