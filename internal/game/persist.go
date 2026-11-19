package game

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/IPeter/l2go/internal/storage"
)

// characterUpdater is the part of storage.CharacterStore the saver uses: it
// replaces the state of a stored character, items included.
type characterUpdater interface {
	Update(ctx context.Context, c *storage.Character) error
}

// errSaverClosed is given to the done function of a save that came after
// the server stopped saving.
var errSaverClosed = errors.New("game: the server has stopped saving characters")

// Snapshot returns the player as it is to be stored (Mobius storeCharBase):
// the loaded character with the current position and heading, level,
// vitals and their maximum, items, and LastAccess set to the loop's time.
// Loop-only.
func (p *Player) Snapshot() storage.Character {
	c := p.char
	pos := p.Position()
	c.X, c.Y, c.Z, c.Heading = pos.X, pos.Y, pos.Z, pos.Heading
	v := p.stats.View()
	c.Level = int32(p.Level()) //nolint:gosec // levels are small
	c.MaxHP, c.MaxMP, c.MaxCP = float64(v.MaxHP), float64(v.MaxMP), float64(v.MaxCP)
	c.CurHP, c.CurMP, c.CurCP = p.CurHP, p.CurMP, p.CurCP
	c.LastAccess = p.srv.loop.Now()
	c.Items = p.inv.storageItems()
	return c
}

// Save stores a snapshot of p in the background (decision W13): the loop
// never waits for the database. done, if not nil, runs after the write with
// its error, on the saver's goroutine; it must not block. Loop-only.
func (s *Server) Save(p *Player, done func(error)) {
	s.saver.enqueue(p.Snapshot(), done)
}

// saver writes character snapshots in the background, one at a time. A
// newer snapshot of a character replaces one that still waits, and a
// snapshot that arrives while the character is being written is written
// after that write, so writes never go back in time. enqueue never blocks.
type saver struct {
	store   characterUpdater
	log     *slog.Logger
	timeout time.Duration

	mu       sync.Mutex
	pending  map[int32]*saveJob // the newest unwritten snapshot per character
	queue    []int32            // characters with a pending snapshot, oldest first
	inflight int32              // the character being written, or 0
	idle     map[int32]chan struct{}
	closed   bool
	wake     chan struct{}
}

type saveJob struct {
	char  storage.Character
	dones []func(error)
}

func newSaver(store characterUpdater, log *slog.Logger) *saver {
	return &saver{
		store:   store,
		log:     log,
		timeout: saveTimeout,
		pending: make(map[int32]*saveJob),
		idle:    make(map[int32]chan struct{}),
		wake:    make(chan struct{}, 1),
	}
}

// enqueue hands a snapshot to the saver. done, if not nil, runs after the
// write. Safe for concurrent use.
func (s *saver) enqueue(c storage.Character, done func(error)) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		s.log.Error("game: character not saved: the server has stopped saving", "objectId", c.ObjectID, "name", c.Name)
		if done != nil {
			done(errSaverClosed)
		}
		return
	}
	if j, ok := s.pending[c.ObjectID]; ok {
		j.char = c
		if done != nil {
			j.dones = append(j.dones, done)
		}
	} else {
		j := &saveJob{char: c}
		if done != nil {
			j.dones = append(j.dones, done)
		}
		s.pending[c.ObjectID] = j
		s.queue = append(s.queue, c.ObjectID)
	}
	s.mu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// wait returns once the character has no snapshot waiting or being written,
// or with ctx's error. Safe for concurrent use.
func (s *saver) wait(ctx context.Context, objectID int32) error {
	s.mu.Lock()
	if s.pending[objectID] == nil && s.inflight != objectID {
		s.mu.Unlock()
		return nil
	}
	ch := s.idle[objectID]
	if ch == nil {
		ch = make(chan struct{})
		s.idle[objectID] = ch
	}
	s.mu.Unlock()
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// close makes run return once the snapshots handed over so far are written.
// Later snapshots are dropped (and logged).
func (s *saver) close() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// run writes snapshots until close was called and none is left. Each write
// gets its own timeout under ctx.
func (s *saver) run(ctx context.Context) {
	for {
		s.mu.Lock()
		if len(s.queue) == 0 {
			closed := s.closed
			s.mu.Unlock()
			if closed {
				return
			}
			<-s.wake
			continue
		}
		id := s.queue[0]
		s.queue[0] = 0
		s.queue = s.queue[1:]
		job := s.pending[id]
		delete(s.pending, id)
		s.inflight = id
		s.mu.Unlock()

		err := s.write(ctx, &job.char)
		for _, done := range job.dones {
			done(err)
		}

		s.mu.Lock()
		s.inflight = 0
		if s.pending[id] == nil {
			if ch := s.idle[id]; ch != nil {
				close(ch)
				delete(s.idle, id)
			}
		}
		s.mu.Unlock()
	}
}

func (s *saver) write(ctx context.Context, c *storage.Character) error {
	wctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	if err := s.store.Update(wctx, c); err != nil {
		s.log.Error("game: cannot save character", "objectId", c.ObjectID, "name", c.Name, "err", err)
		return err
	}
	s.log.Debug("character saved", "objectId", c.ObjectID, "name", c.Name, "x", c.X, "y", c.Y, "z", c.Z)
	return nil
}
