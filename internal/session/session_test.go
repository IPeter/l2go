package session

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeClock is a manually advanced clock for expiry tests.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{t: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func newTestStore(ttl time.Duration) (*Store, *fakeClock) {
	s := NewStore(ttl)
	clock := newFakeClock()
	s.now = clock.Now
	return s, clock
}

func (s *Store) size() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries)
}

func TestNewStoreDefaultTTL(t *testing.T) {
	for _, ttl := range []time.Duration{0, -time.Second} {
		if got := NewStore(ttl).ttl; got != DefaultTTL {
			t.Errorf("NewStore(%v).ttl = %v, want %v", ttl, got, DefaultTTL)
		}
	}
	if got := NewStore(time.Second).ttl; got != time.Second {
		t.Errorf("NewStore(1s).ttl = %v, want 1s", got)
	}
}

func TestIssueCheckConsume(t *testing.T) {
	s, _ := newTestStore(time.Minute)
	k, err := s.Issue("alice")
	if err != nil {
		t.Fatal(err)
	}
	if !s.CheckLogin("alice", k.LoginOK1, k.LoginOK2) {
		t.Error("CheckLogin with the issued keys failed")
	}
	if !s.CheckLogin("alice", k.LoginOK1, k.LoginOK2) {
		t.Error("CheckLogin is not repeatable")
	}
	if got := s.Len(); got != 1 {
		t.Errorf("Len = %d, want 1", got)
	}
	if !s.Consume("alice", k) {
		t.Fatal("Consume with the issued keys failed")
	}
	if s.Consume("alice", k) {
		t.Error("Consume accepted the same keys twice")
	}
	if s.CheckLogin("alice", k.LoginOK1, k.LoginOK2) {
		t.Error("CheckLogin accepted consumed keys")
	}
	if got := s.Len(); got != 0 {
		t.Errorf("Len after Consume = %d, want 0", got)
	}
}

func TestIssueEmptyAccount(t *testing.T) {
	s := NewStore(time.Minute)
	if _, err := s.Issue(""); !errors.Is(err, ErrEmptyAccount) {
		t.Errorf("Issue(\"\") = %v, want ErrEmptyAccount", err)
	}
}

func TestIssueIsRandom(t *testing.T) {
	s := NewStore(time.Minute)
	seen := make(map[Keys]bool)
	for i := range 100 {
		k, err := s.Issue(fmt.Sprintf("user%d", i))
		if err != nil {
			t.Fatal(err)
		}
		if seen[k] {
			t.Fatalf("Issue returned the same keys twice: %+v", k)
		}
		seen[k] = true
	}
}

func TestIssueReplacesEarlierKeys(t *testing.T) {
	s, _ := newTestStore(time.Minute)
	old, err := s.Issue("alice")
	if err != nil {
		t.Fatal(err)
	}
	k, err := s.Issue("alice")
	if err != nil {
		t.Fatal(err)
	}
	if old == k {
		t.Fatal("second Issue returned the same keys")
	}
	if s.CheckLogin("alice", old.LoginOK1, old.LoginOK2) {
		t.Error("CheckLogin accepted replaced keys")
	}
	if s.Consume("alice", old) {
		t.Error("Consume accepted replaced keys")
	}
	if !s.Consume("alice", k) {
		t.Error("Consume rejected the current keys")
	}
}

func TestWrongKeys(t *testing.T) {
	s, _ := newTestStore(time.Minute)
	k, err := s.Issue("alice")
	if err != nil {
		t.Fatal(err)
	}
	if s.CheckLogin("alice", k.LoginOK1+1, k.LoginOK2) {
		t.Error("CheckLogin accepted a wrong LoginOK1")
	}
	if s.CheckLogin("alice", k.LoginOK1, k.LoginOK2+1) {
		t.Error("CheckLogin accepted a wrong LoginOK2")
	}
	if s.CheckLogin("bob", k.LoginOK1, k.LoginOK2) {
		t.Error("CheckLogin accepted alice's keys for bob")
	}
	if s.CheckLogin("Alice", k.LoginOK1, k.LoginOK2) {
		t.Error("CheckLogin folded the account name's case")
	}

	wrong := []Keys{
		{k.LoginOK1 ^ 1, k.LoginOK2, k.PlayOK1, k.PlayOK2},
		{k.LoginOK1, k.LoginOK2 ^ 1, k.PlayOK1, k.PlayOK2},
		{k.LoginOK1, k.LoginOK2, k.PlayOK1 ^ 1, k.PlayOK2},
		{k.LoginOK1, k.LoginOK2, k.PlayOK1, k.PlayOK2 ^ 1},
		// The order the client sends them in, not swapped back.
		{k.PlayOK2, k.PlayOK1, k.LoginOK1, k.LoginOK2},
	}
	for _, w := range wrong {
		if s.Consume("alice", w) {
			t.Errorf("Consume accepted wrong keys %+v (issued %+v)", w, k)
		}
	}
	if s.Consume("bob", k) {
		t.Error("Consume accepted alice's keys for bob")
	}
	// Wrong attempts do not burn the real keys.
	if !s.Consume("alice", k) {
		t.Error("Consume rejected the right keys after wrong attempts")
	}
}

func TestUnknownAccount(t *testing.T) {
	s := NewStore(time.Minute)
	if s.CheckLogin("nobody", 0, 0) {
		t.Error("CheckLogin accepted an unknown account")
	}
	if s.Consume("nobody", Keys{}) {
		t.Error("Consume accepted an unknown account")
	}
}

func TestRemove(t *testing.T) {
	s, _ := newTestStore(time.Minute)
	k, err := s.Issue("alice")
	if err != nil {
		t.Fatal(err)
	}
	s.Remove("alice")
	s.Remove("alice") // removing twice is harmless
	if s.Consume("alice", k) {
		t.Error("Consume accepted removed keys")
	}
}

func TestExpiry(t *testing.T) {
	s, clock := newTestStore(time.Minute)
	k, err := s.Issue("alice")
	if err != nil {
		t.Fatal(err)
	}
	clock.Advance(time.Minute - time.Nanosecond)
	if !s.CheckLogin("alice", k.LoginOK1, k.LoginOK2) {
		t.Fatal("keys expired too early")
	}
	// CheckLogin extended the lifetime by another TTL.
	clock.Advance(time.Minute - time.Nanosecond)
	if s.Len() != 1 {
		t.Fatal("CheckLogin did not extend the lifetime")
	}
	clock.Advance(time.Nanosecond)
	if s.Len() != 0 {
		t.Error("Len counts expired keys")
	}
	if s.Consume("alice", k) {
		t.Error("Consume accepted expired keys")
	}
	if s.size() != 0 {
		t.Error("the expired entry was not removed on lookup")
	}
}

func TestExpiredKeysFailCheckLogin(t *testing.T) {
	s, clock := newTestStore(time.Minute)
	k, err := s.Issue("alice")
	if err != nil {
		t.Fatal(err)
	}
	clock.Advance(time.Minute)
	if s.CheckLogin("alice", k.LoginOK1, k.LoginOK2) {
		t.Error("CheckLogin accepted expired keys")
	}
}

func TestFailedCheckDoesNotExtend(t *testing.T) {
	s, clock := newTestStore(time.Minute)
	k, err := s.Issue("alice")
	if err != nil {
		t.Fatal(err)
	}
	clock.Advance(30 * time.Second)
	s.CheckLogin("alice", k.LoginOK1+1, k.LoginOK2)
	clock.Advance(30 * time.Second)
	if s.Consume("alice", k) {
		t.Error("a failed CheckLogin extended the lifetime")
	}
}

func TestRemoveExpired(t *testing.T) {
	s, clock := newTestStore(time.Minute)
	for _, a := range []string{"a", "b", "c"} {
		if _, err := s.Issue(a); err != nil {
			t.Fatal(err)
		}
	}
	clock.Advance(30 * time.Second)
	if _, err := s.Issue("d"); err != nil {
		t.Fatal(err)
	}
	clock.Advance(30 * time.Second)
	if n := s.removeExpired(); n != 3 {
		t.Errorf("removeExpired = %d, want 3", n)
	}
	if s.size() != 1 || s.Len() != 1 {
		t.Errorf("size, Len = %d, %d; want 1, 1", s.size(), s.Len())
	}
}

func TestRunRemovesExpiredUntilCancelled(t *testing.T) {
	s, clock := newTestStore(10 * time.Millisecond)
	if _, err := s.Issue("alice"); err != nil {
		t.Fatal(err)
	}
	clock.Advance(time.Hour)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.Run(ctx)
	}()

	deadline := time.Now().Add(5 * time.Second)
	for s.size() != 0 {
		if time.Now().After(deadline) {
			t.Fatal("Run did not remove the expired entry")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not stop after cancel")
	}
}

func TestConcurrentUse(t *testing.T) {
	s := NewStore(time.Minute)
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() {
			account := fmt.Sprintf("user%d", i%10)
			for range 20 {
				k, err := s.Issue(account)
				if err != nil {
					t.Error(err)
					return
				}
				s.CheckLogin(account, k.LoginOK1, k.LoginOK2)
				s.Consume(account, k)
				_ = s.Len()
				s.removeExpired()
			}
			s.Remove(account)
		})
	}
	wg.Wait()
}

// Many game servers racing to confirm the same keys: exactly one wins.
func TestConsumeOnlyOnceUnderContention(t *testing.T) {
	s := NewStore(time.Minute)
	k, err := s.Issue("alice")
	if err != nil {
		t.Fatal(err)
	}
	var wins atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for range 50 {
		wg.Go(func() {
			<-start
			if s.Consume("alice", k) {
				wins.Add(1)
			}
		})
	}
	close(start)
	wg.Wait()
	if got := wins.Load(); got != 1 {
		t.Errorf("%d goroutines consumed the keys, want 1", got)
	}
}
