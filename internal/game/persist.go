package game

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/IPeter/l2go/internal/storage"
)

// errSaverClosed is given to the done function of a save that came after
// the server stopped saving.
var errSaverClosed = errors.New("game: the server has stopped saving characters")

// Retries of a failed save (decision I2): the first after saveRetryDelay,
// each later one waits twice as long, up to saveRetryMaxDelay. After the
// server stopped, a save gets saveAttemptsAfterClose more tries.
const (
	saveRetryDelay         = 500 * time.Millisecond
	saveRetryMaxDelay      = 30 * time.Second
	saveAttemptsAfterClose = 3
)

// Snapshot returns the player as it is to be stored (Mobius storeCharBase):
// the loaded character with the current position and heading, level,
// vitals and their maximum, and LastAccess set to the loop's time. Items
// holds every item the character owns, carried and in the warehouse, as
// they are now; saving writes only the changes (Server.Save). Loop-only.
func (p *Player) Snapshot() storage.Character {
	c := p.char
	pos := p.Position()
	c.X, c.Y, c.Z, c.Heading = pos.X, pos.Y, pos.Z, pos.Heading
	v := p.stats.View()
	c.Level = int32(p.Level()) //nolint:gosec // levels are small
	c.MaxHP, c.MaxMP, c.MaxCP = float64(v.MaxHP), float64(v.MaxMP), float64(v.MaxCP)
	c.CurHP, c.CurMP, c.CurCP = p.CurHP, p.CurMP, p.CurCP
	c.LastAccess = p.srv.loop.Now()
	c.Items = p.storageItems()
	return c
}

// Save stores a snapshot of p with the changes to its items since its last
// save, in the background (decisions W13 and I2): the loop never waits for
// the database. The saves other players asked for earlier in the tick go
// first, so that an item that moved between characters is written giver
// first. done, if not nil, runs after the write with its error, on the
// saver's goroutine; it must not block. Loop-only.
func (s *Server) Save(p *Player, done func(error)) {
	s.flushSaveRequests()
	s.saver.enqueue(p.Snapshot(), p.takeItemChanges(), done)
}

// RequestSave asks for a save of p at the end of the tick (decision I2):
// every handler that changes items calls it. The saves of a tick are
// written in the order they were asked for. Loop-only.
func (s *Server) RequestSave(p *Player) {
	if p.state != stateInGame || slices.Contains(s.items.saveRequests, p) {
		return
	}
	s.items.saveRequests = append(s.items.saveRequests, p)
}

// flushSaveRequests hands the saves asked for so far to the saver, in
// order. It runs at the end of every tick, and before any other save.
func (s *Server) flushSaveRequests() {
	reqs := s.items.saveRequests
	s.items.saveRequests = nil
	for _, p := range reqs {
		if p.state == stateInGame {
			s.saver.enqueue(p.Snapshot(), p.takeItemChanges(), nil)
		}
	}
}

// saver writes character snapshots in the background, one at a time, in
// the order they arrive (decision I2). A snapshot is merged into the last
// waiting one of its character when nothing waits after that one, or when
// it changes no items; otherwise it waits in line, so that items moving
// between characters are always written giver first. A failed write is
// retried, with a growing delay, before anything after it. enqueue never
// blocks.
type saver struct {
	store      storage.CharacterSaver
	log        *slog.Logger
	timeout    time.Duration
	retryDelay time.Duration
	maxDelay   time.Duration

	mu       sync.Mutex
	queue    []*saveJob         // waiting snapshots, oldest first
	last     map[int32]*saveJob // the newest waiting snapshot per character
	waiting  map[int32]int      // waiting snapshots per character
	inflight int32              // the character being written, or 0
	idle     map[int32]chan struct{}
	closed   bool
	wake     chan struct{}
}

type saveJob struct {
	char    storage.Character
	changes storage.ItemChanges
	dones   []func(error)
}

func newSaver(store storage.CharacterSaver, log *slog.Logger) *saver {
	return &saver{
		store:      store,
		log:        log,
		timeout:    saveTimeout,
		retryDelay: saveRetryDelay,
		maxDelay:   saveRetryMaxDelay,
		last:       make(map[int32]*saveJob),
		waiting:    make(map[int32]int),
		idle:       make(map[int32]chan struct{}),
		wake:       make(chan struct{}, 1),
	}
}

// enqueue hands a snapshot and the changes to its items to the saver. done,
// if not nil, runs after the write. Safe for concurrent use.
func (s *saver) enqueue(c storage.Character, changes storage.ItemChanges, done func(error)) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		s.log.Error("game: character not saved: the server has stopped saving", "objectId", c.ObjectID, "name", c.Name)
		if done != nil {
			done(errSaverClosed)
		}
		return
	}
	j := s.last[c.ObjectID]
	if j != nil && (noItemChanges(changes) || s.queue[len(s.queue)-1] == j) {
		j.char = c
		j.changes = mergeItemChanges(j.changes, changes)
	} else {
		j = &saveJob{char: c, changes: changes}
		s.queue = append(s.queue, j)
		s.last[c.ObjectID] = j
		s.waiting[c.ObjectID]++
	}
	if done != nil {
		j.dones = append(j.dones, done)
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
	if s.waiting[objectID] == 0 && s.inflight != objectID {
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
		job := s.queue[0]
		s.queue[0] = nil
		s.queue = s.queue[1:]
		id := job.char.ObjectID
		if s.last[id] == job {
			delete(s.last, id)
		}
		if s.waiting[id]--; s.waiting[id] <= 0 {
			delete(s.waiting, id)
		}
		s.inflight = id
		s.mu.Unlock()

		err := s.writeWithRetry(ctx, job)
		for _, done := range job.dones {
			done(err)
		}

		s.mu.Lock()
		s.inflight = 0
		if s.waiting[id] == 0 {
			if ch := s.idle[id]; ch != nil {
				close(ch)
				delete(s.idle, id)
			}
		}
		s.mu.Unlock()
	}
}

// writeWithRetry writes a snapshot, retrying a failed write until it
// succeeds, the character is gone (storage.ErrNotFound), ctx ends, or the
// saver closed and the last tries failed too.
func (s *saver) writeWithRetry(ctx context.Context, job *saveJob) error {
	delay := s.retryDelay
	afterClose := 0
	for {
		err := s.write(ctx, job)
		if err == nil || errors.Is(err, storage.ErrNotFound) || ctx.Err() != nil {
			return err
		}
		s.mu.Lock()
		closed := s.closed
		s.mu.Unlock()
		if closed {
			if afterClose++; afterClose >= saveAttemptsAfterClose {
				s.log.Error("game: giving up saving the character", "objectId", job.char.ObjectID, "name", job.char.Name, "err", err)
				return err
			}
		}
		t := time.NewTimer(delay)
		select {
		case <-t.C:
		case <-ctx.Done():
			t.Stop()
			return err
		}
		delay = min(2*delay, s.maxDelay)
	}
}

func (s *saver) write(ctx context.Context, job *saveJob) error {
	c := &job.char
	wctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	if err := s.store.Save(wctx, c, job.changes); err != nil {
		s.log.Error("game: cannot save character", "objectId", c.ObjectID, "name", c.Name, "err", err)
		return err
	}
	s.log.Debug("character saved", "objectId", c.ObjectID, "name", c.Name, "x", c.X, "y", c.Y, "z", c.Z,
		"upserts", len(job.changes.Upserts), "destroyed", len(job.changes.Destroyed), "left", len(job.changes.Left))
	return nil
}

// noItemChanges reports whether ch changes no item.
func noItemChanges(ch storage.ItemChanges) bool {
	return len(ch.Upserts) == 0 && len(ch.Destroyed) == 0 && len(ch.Left) == 0
}

// mergeItemChanges returns the changes of a, then b, as one set of changes
// with the meaning of storage.CharacterSaver.Save: each change of b
// replaces what a says about the same item.
func mergeItemChanges(a, b storage.ItemChanges) storage.ItemChanges {
	if noItemChanges(b) {
		return a
	}
	out := storage.ItemChanges{
		Upserts:   slices.Clone(a.Upserts),
		Destroyed: slices.Clone(a.Destroyed),
		Left:      slices.Clone(a.Left),
	}
	drop := func(id int32) {
		out.Upserts = slices.DeleteFunc(out.Upserts, func(it storage.Item) bool { return it.ObjectID == id })
		out.Destroyed = slices.DeleteFunc(out.Destroyed, func(x int32) bool { return x == id })
		out.Left = slices.DeleteFunc(out.Left, func(x int32) bool { return x == id })
	}
	for _, it := range b.Upserts {
		drop(it.ObjectID)
		out.Upserts = append(out.Upserts, it)
	}
	for _, id := range b.Destroyed {
		drop(id)
		out.Destroyed = append(out.Destroyed, id)
	}
	for _, id := range b.Left {
		if slices.Contains(out.Destroyed, id) {
			continue
		}
		drop(id)
		out.Left = append(out.Left, id)
	}
	return out
}
