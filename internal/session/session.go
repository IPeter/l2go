// Package session issues and checks the session keys that carry a player
// from the login server to a game server.
//
// The login server issues four random keys per login. The client shows the
// two login keys on later login server requests (server list, server
// login) and all four to the game server, which asks the login server to
// confirm them over the link (package link). A confirmed set is removed, so
// each set lets a player into a game server only once.
package session

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"
)

// DefaultTTL is how long keys stay valid after they were issued or last
// checked, when NewStore gets no TTL.
const DefaultTTL = 2 * time.Minute

// ErrEmptyAccount is returned by Issue for an empty account name.
var ErrEmptyAccount = errors.New("session: empty account name")

// Keys is one set of session keys.
//
// The login server sends LoginOK1 and LoginOK2 in LoginOk and PlayOK1 and
// PlayOK2 in PlayOk. The Interlude client sends all four to the game server
// in AuthLogin, in the order PlayOK2, PlayOK1, LoginOK1, LoginOK2.
type Keys struct {
	LoginOK1, LoginOK2 int32
	PlayOK1, PlayOK2   int32
}

// Store holds the current keys of each account. It is safe for concurrent
// use.
//
// Account names are compared exactly: the login server and the game server
// must use the same spelling (no case folding happens here).
//
// Expired keys are never accepted. They are removed when their account is
// looked up, and by Run.
type Store struct {
	ttl time.Duration
	now func() time.Time // replaced in tests

	mu      sync.Mutex
	entries map[string]entry
}

type entry struct {
	keys    Keys
	expires time.Time
}

// NewStore returns an empty store. Keys expire ttl after they were issued
// or last passed CheckLogin; ttl <= 0 means DefaultTTL.
func NewStore(ttl time.Duration) *Store {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	return &Store{ttl: ttl, now: time.Now, entries: make(map[string]entry)}
}

// Issue creates random keys for account and returns them. They replace any
// earlier keys of that account.
func (s *Store) Issue(account string) (Keys, error) {
	if account == "" {
		return Keys{}, ErrEmptyAccount
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return Keys{}, fmt.Errorf("session: random keys: %w", err)
	}
	k := Keys{
		LoginOK1: int32(binary.LittleEndian.Uint32(b[0:])),  //nolint:gosec // random bits
		LoginOK2: int32(binary.LittleEndian.Uint32(b[4:])),  //nolint:gosec // random bits
		PlayOK1:  int32(binary.LittleEndian.Uint32(b[8:])),  //nolint:gosec // random bits
		PlayOK2:  int32(binary.LittleEndian.Uint32(b[12:])), //nolint:gosec // random bits
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[account] = entry{keys: k, expires: s.now().Add(s.ttl)}
	return k, nil
}

// CheckLogin reports whether account has live keys with the given login
// pair. A match keeps the keys alive for another TTL.
func (s *Store) CheckLogin(account string, loginOK1, loginOK2 int32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	e, ok := s.lookup(account, now)
	if !ok || e.keys.LoginOK1 != loginOK1 || e.keys.LoginOK2 != loginOK2 {
		return false
	}
	e.expires = now.Add(s.ttl)
	s.entries[account] = e
	return true
}

// Consume reports whether account has live keys equal to k (all four
// values). On a match the keys are removed, so they work only once.
func (s *Store) Consume(account string, k Keys) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.lookup(account, s.now())
	if !ok || e.keys != k {
		return false
	}
	delete(s.entries, account)
	return true
}

// Remove forgets the keys of account, if any.
func (s *Store) Remove(account string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, account)
}

// Len returns the number of accounts with live keys.
func (s *Store) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	n := 0
	for _, e := range s.entries {
		if now.Before(e.expires) {
			n++
		}
	}
	return n
}

// Run removes expired keys every half TTL until ctx is done. Without it,
// keys that are never used again stay in memory (they are still rejected).
func (s *Store) Run(ctx context.Context) {
	t := time.NewTicker(max(s.ttl/2, time.Millisecond))
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.removeExpired()
		}
	}
}

// removeExpired deletes every expired entry and returns how many it removed.
func (s *Store) removeExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	n := 0
	for account, e := range s.entries {
		if !now.Before(e.expires) {
			delete(s.entries, account)
			n++
		}
	}
	return n
}

// lookup returns the live entry of account. An expired entry is deleted.
// The caller holds s.mu.
func (s *Store) lookup(account string, now time.Time) (entry, bool) {
	e, ok := s.entries[account]
	if !ok {
		return entry{}, false
	}
	if !now.Before(e.expires) {
		delete(s.entries, account)
		return entry{}, false
	}
	return e, true
}
