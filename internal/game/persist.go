package game

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// errSaverClosed is given to the done function of a save that came after
// the server stopped saving.
var errSaverClosed = errors.New("game: the server has stopped saving characters")

// errNoBatchWriter is the error of a joint save on a character store that
// cannot write batches (storage.BatchWriter).
var errNoBatchWriter = errors.New("game: the character store cannot write joint saves (storage.BatchWriter)")

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
	p.storeSkills(&c) // skill_persist.go
	if p.itemSkill.loaded {
		p.storeShortcuts(&c) // shortcut.go
		p.storeItemReuse(&c) // item_skill.go
	}
	if p.quests.vars != nil {
		c.Vars = p.quests.vars.Map() // quest.go: the player variables
	}
	c.Quests = p.questDocuments()
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
	changes := p.takeSaveChanges()
	s.saver.enqueue(p.Snapshot(), changes, s.questSaveDone(p, changes.Quests, done))
}

// SaveJoint writes the pending changes of several owners in one transaction
// (decisions T5 and T6, storage.BatchWriter): the joint save of a trade, a
// store purchase or a clan warehouse move, and the way to write clans,
// crests, relations and patches to offline characters. Loop-only; the loop
// never waits for the database.
//
// The batch holds a snapshot of each listed player that is in the game with
// the changes to its items and quests since its last save (as Save takes
// them; a player listed twice counts once), plus extra: other owners' item
// changes (a clan warehouse), clan and crest documents, relation changes,
// patches, and further character saves in extra.Saves.
//
// Order: the saves asked for earlier in the tick are handed over first (as
// in Save). The batch then takes the queue position of the earliest pending
// save of a listed player, and every pending write of those players from
// there on, and of whatever those writes touch in turn, merges into it, so
// that nothing written later than before overtakes an older write of the
// same character, clan or crest. Without a pending save of a listed player
// it goes to the end of the queue, after every pending write: a patch to a
// character, a relation change, or the item changes of a clan is written
// after that character's or clan's pending writes. Loading a character
// (SelectCharacter) waits for every pending write that touches it. Patches
// are for characters not in the game: in a merged batch they apply after
// every save (storage.Batch), so they would win over a later snapshot.
//
// A failed write is retried like Save's. When it fails for good (a
// character that does not exist, a clan name taken, or the server stopped
// first), the listed players still in the game are disconnected without a
// save, so that they reload what the database has.
func (s *Server) SaveJoint(players []*Player, extra storage.Batch) {
	s.saveJoint(players, extra, nil)
}

// saveJoint is SaveJoint with done, which, if not nil, runs after the write
// with its error, on the saver's goroutine; it must not block.
func (s *Server) saveJoint(players []*Player, extra storage.Batch, done func(error)) {
	// Hand over only the saves asked for before the first listed player: a
	// listed player's own request goes into the batch (written alone, its
	// snapshot could reach the database without the other side), and the
	// later ones stay for the end of the tick, after the batch.
	reqs := s.items.saveRequests
	cut := len(reqs)
	for i, p := range reqs {
		if slices.Contains(players, p) {
			cut = i
			break
		}
	}
	s.items.saveRequests = reqs[:cut:cut]
	s.flushSaveRequests()
	for _, p := range reqs[cut:] {
		if !slices.Contains(players, p) {
			s.items.saveRequests = append(s.items.saveRequests, p)
		}
	}
	var (
		saves   []storage.CharacterSave
		ids     []int32
		onSaved []func(error)
	)
	for _, p := range players {
		if p == nil || slices.Contains(ids, p.ObjectID()) {
			continue
		}
		ids = append(ids, p.ObjectID())
		if p.state != stateInGame {
			continue // its last save was handed over when it left
		}
		changes := p.takeSaveChanges()
		saves = append(saves, storage.CharacterSave{Character: p.Snapshot(), Changes: changes})
		if qd := s.questSaveDone(p, changes.Quests, nil); qd != nil {
			onSaved = append(onSaved, qd)
		}
	}
	s.saver.enqueueJoint(saves, extra, func(err error) {
		for _, fn := range onSaved {
			fn(err)
		}
		if err != nil && len(ids) > 0 {
			s.loop.Post(func() { s.dropAfterFailedSave(ids, err) })
		}
		if done != nil {
			done(err)
		}
	})
}

// dropAfterFailedSave disconnects the players of a joint save that failed
// for good, without saving them, so that they reload from the database
// (decision T5).
func (s *Server) dropAfterFailedSave(ids []int32, err error) {
	for _, id := range ids {
		p, ok := s.Player(world.ObjectID(id))
		if !ok || p.state != stateInGame {
			continue
		}
		s.log.Error("game: a joint save failed, disconnecting the player", append(p.logArgs(), "err", err)...)
		sess := p.session
		s.leave(p, false, nil)
		if sess != nil {
			sess.CloseAfter(leaveWorldPacket, 0)
		}
	}
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
			changes := p.takeSaveChanges()
			s.saver.enqueue(p.Snapshot(), changes, s.questSaveDone(p, changes.Quests, nil))
		}
	}
}

// saveKey names what a save job writes, for the write order: a character,
// or another owner of items, or a clan (all from the object ID space), or a
// crest (crestKey).
type saveKey int64

// crestKey is the key of a crest: crest IDs have their own counter.
func crestKey(id int32) saveKey { return saveKey(int64(id) | 1<<40) }

// saver writes character snapshots and joint saves in the background, one
// job at a time, in the order they arrive (decisions I2, T5 and T6). A
// snapshot is merged into the last waiting job that saves its character
// when nothing waits after that job, or when it changes no items, and in
// both cases only if no waiting job after that one touches the character;
// otherwise it waits in line, so that items moving between characters are
// always written giver first. A joint save goes where enqueueJoint says. A
// failed write is retried, with a growing delay, before anything after it.
// enqueue and enqueueJoint never block.
type saver struct {
	store      storage.CharacterSaver
	batches    storage.BatchWriter // nil when the store cannot write batches
	log        *slog.Logger
	timeout    time.Duration
	retryDelay time.Duration
	maxDelay   time.Duration

	mu       sync.Mutex
	queue    []*saveJob           // waiting jobs, oldest first
	last     map[int32]*saveJob   // the newest waiting job that saves a character
	newest   map[saveKey]*saveJob // the newest waiting job that touches a key
	waiting  map[saveKey]int      // waiting jobs per key
	inflight []saveKey            // the keys of the job being written
	idle     map[saveKey]chan struct{}
	closed   bool
	wake     chan struct{}
}

// saveJob is one write: the saves of one or more characters, and for a
// joint save the rest of a storage.Batch.
type saveJob struct {
	saves []storage.CharacterSave // one per character, in order
	extra storage.Batch           // everything but Saves
	keys  []saveKey               // what it touches, sorted
	dones []func(error)
}

func newSaver(store storage.CharacterSaver, log *slog.Logger) *saver {
	batches, _ := store.(storage.BatchWriter)
	return &saver{
		store:      store,
		batches:    batches,
		log:        log,
		timeout:    saveTimeout,
		retryDelay: saveRetryDelay,
		maxDelay:   saveRetryMaxDelay,
		last:       make(map[int32]*saveJob),
		newest:     make(map[saveKey]*saveJob),
		waiting:    make(map[saveKey]int),
		idle:       make(map[saveKey]chan struct{}),
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
	id := c.ObjectID
	j := s.last[id]
	if j != nil && s.newest[saveKey(id)] == j && (noItemChanges(changes) || s.queue[len(s.queue)-1] == j) {
		cs := j.save(id)
		cs.Character = c
		cs.Changes = mergeItemChanges(cs.Changes, changes)
	} else {
		j = &saveJob{saves: []storage.CharacterSave{{Character: c, Changes: changes}}}
		j.keys = j.touches()
		s.queue = append(s.queue, j)
		s.count(j)
	}
	if done != nil {
		j.dones = append(j.dones, done)
	}
	s.mu.Unlock()
	s.signal()
}

// enqueueJoint hands a joint save to the saver: the character saves (one
// per character) and the rest of extra, written in one transaction
// (Server.SaveJoint says where it goes in the queue). done, if not nil,
// runs after the write. Safe for concurrent use.
func (s *saver) enqueueJoint(saves []storage.CharacterSave, extra storage.Batch, done func(error)) {
	nj := &saveJob{}
	mergeJob(nj, &saveJob{saves: append(slices.Clone(saves), extra.Saves...), extra: extra})
	if len(nj.saves) == 0 && nj.extra.Empty() {
		if done != nil {
			done(nil)
		}
		return
	}
	if done != nil {
		nj.dones = append(nj.dones, done)
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		s.log.Error("game: joint save dropped: the server has stopped saving", "keys", nj.keys)
		for _, d := range nj.dones {
			d(errSaverClosed)
		}
		return
	}
	first := slices.IndexFunc(s.queue, func(j *saveJob) bool {
		return slices.ContainsFunc(j.saves, func(cs storage.CharacterSave) bool {
			return nj.save(cs.Character.ObjectID) != nil
		})
	})
	if first < 0 {
		s.queue = append(s.queue, nj)
		s.count(nj)
		s.mu.Unlock()
		s.signal()
		return
	}
	// Merge every waiting job from first on that touches what the joint
	// save touches, and what those touch in turn: none of them may move
	// past a job that touches the same keys.
	keys := slices.Clone(nj.keys)
	merged := make([]bool, len(s.queue))
	for changed := true; changed; {
		changed = false
		for i := first; i < len(s.queue); i++ {
			if merged[i] || !touchesAny(s.queue[i], keys) {
				continue
			}
			merged[i], changed = true, true
			keys = unionKeys(keys, s.queue[i].keys)
		}
	}
	joint := &saveJob{}
	for i := first; i < len(s.queue); i++ {
		if merged[i] {
			mergeJob(joint, s.queue[i])
		}
	}
	mergeJob(joint, nj)
	queue := append(slices.Clone(s.queue[:first]), joint)
	for i := first; i < len(s.queue); i++ {
		if !merged[i] {
			queue = append(queue, s.queue[i])
		}
	}
	s.queue = queue
	s.reindex()
	s.mu.Unlock()
	s.signal()
}

func (s *saver) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// count adds a job that was put at the end of the queue to the indexes.
func (s *saver) count(j *saveJob) {
	for _, k := range j.keys {
		s.newest[k] = j
		s.waiting[k]++
	}
	for _, cs := range j.saves {
		s.last[cs.Character.ObjectID] = j
	}
}

// reindex builds the indexes of the queue again.
func (s *saver) reindex() {
	clear(s.last)
	clear(s.newest)
	clear(s.waiting)
	for _, j := range s.queue {
		s.count(j)
	}
}

// wait returns once nothing that touches the character waits or is being
// written, or with ctx's error. Safe for concurrent use.
func (s *saver) wait(ctx context.Context, objectID int32) error {
	k := saveKey(objectID)
	s.mu.Lock()
	if s.waiting[k] == 0 && !slices.Contains(s.inflight, k) {
		s.mu.Unlock()
		return nil
	}
	ch := s.idle[k]
	if ch == nil {
		ch = make(chan struct{})
		s.idle[k] = ch
	}
	s.mu.Unlock()
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// close makes run return once the jobs handed over so far are written.
// Later ones are dropped (and logged).
func (s *saver) close() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	s.signal()
}

// run writes jobs until close was called and none is left. Each write gets
// its own timeout under ctx.
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
		for _, k := range job.keys {
			if s.newest[k] == job {
				delete(s.newest, k)
			}
			if s.waiting[k]--; s.waiting[k] <= 0 {
				delete(s.waiting, k)
			}
		}
		for _, cs := range job.saves {
			if id := cs.Character.ObjectID; s.last[id] == job {
				delete(s.last, id)
			}
		}
		s.inflight = job.keys
		s.mu.Unlock()

		err := s.writeWithRetry(ctx, job)
		for _, done := range job.dones {
			done(err)
		}

		s.mu.Lock()
		s.inflight = nil
		for _, k := range job.keys {
			if s.waiting[k] == 0 {
				if ch := s.idle[k]; ch != nil {
					close(ch)
					delete(s.idle, k)
				}
			}
		}
		s.mu.Unlock()
	}
}

// writeWithRetry writes a job, retrying a failed write until it succeeds,
// fails for good (a character that does not exist, a clan name taken, no
// batch writer), ctx ends, or the saver closed and the last tries failed
// too.
func (s *saver) writeWithRetry(ctx context.Context, job *saveJob) error {
	delay := s.retryDelay
	afterClose := 0
	for {
		err := s.write(ctx, job)
		if err == nil || errors.Is(err, storage.ErrNotFound) || errors.Is(err, storage.ErrExists) ||
			errors.Is(err, errNoBatchWriter) || ctx.Err() != nil {
			return err
		}
		s.mu.Lock()
		closed := s.closed
		s.mu.Unlock()
		if closed {
			if afterClose++; afterClose >= saveAttemptsAfterClose {
				s.log.Error("game: giving up saving", append(job.logArgs(), "err", err)...)
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
	wctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	if len(job.saves) == 1 && job.extra.Empty() {
		cs := &job.saves[0]
		c := &cs.Character
		if err := s.store.Save(wctx, c, cs.Changes); err != nil {
			s.log.Error("game: cannot save character", "objectId", c.ObjectID, "name", c.Name, "err", err)
			return err
		}
		s.log.Debug("character saved", "objectId", c.ObjectID, "name", c.Name, "x", c.X, "y", c.Y, "z", c.Z,
			"upserts", len(cs.Changes.Upserts), "destroyed", len(cs.Changes.Destroyed), "left", len(cs.Changes.Left))
		return nil
	}
	if s.batches == nil {
		s.log.Error("game: cannot write a joint save", append(job.logArgs(), "err", errNoBatchWriter)...)
		return errNoBatchWriter
	}
	b := job.extra
	b.Saves = job.saves
	if err := s.batches.WriteBatch(wctx, b); err != nil {
		s.log.Error("game: cannot write a joint save", append(job.logArgs(), "err", err)...)
		return err
	}
	s.log.Debug("joint save written", job.logArgs()...)
	return nil
}

// logArgs names what a job writes.
func (j *saveJob) logArgs() []any {
	ids := make([]int32, len(j.saves))
	for i, cs := range j.saves {
		ids[i] = cs.Character.ObjectID
	}
	return []any{"characters", ids, "keys", j.keys}
}

// save returns the job's save of a character, or nil.
func (j *saveJob) save(id int32) *storage.CharacterSave {
	for i := range j.saves {
		if j.saves[i].Character.ObjectID == id {
			return &j.saves[i]
		}
	}
	return nil
}

// touches returns the keys of what the job writes, sorted: the saved
// characters, the owners of other item changes, deleted and stored clans,
// the clans of the alliance crests' alliances, the characters on both ends
// of a relation, patched characters, and the crests.
func (j *saveJob) touches() []saveKey {
	var keys []saveKey
	add := func(k saveKey) {
		if i, found := slices.BinarySearch(keys, k); !found {
			keys = slices.Insert(keys, i, k)
		}
	}
	b := &j.extra
	for _, cs := range j.saves {
		add(saveKey(cs.Character.ObjectID))
	}
	for _, it := range b.Items {
		add(saveKey(it.Owner))
	}
	for _, id := range b.DeletedClans {
		add(saveKey(id))
	}
	for _, c := range b.Clans {
		add(saveKey(c.ID))
	}
	for _, a := range b.AllyCrests {
		add(saveKey(a.AllyID))
	}
	for _, id := range b.DeletedCrests {
		add(crestKey(id))
	}
	for _, c := range b.Crests {
		add(crestKey(c.ID))
	}
	for _, list := range [][]storage.Relation{b.Relations.Added, b.Relations.Removed} {
		for _, r := range list {
			add(saveKey(r.Owner))
			add(saveKey(r.Target))
		}
	}
	for _, p := range b.Patches {
		add(saveKey(p.ObjectID))
	}
	return keys
}

// touchesAny reports whether the job touches one of keys (sorted).
func touchesAny(j *saveJob, keys []saveKey) bool {
	for _, k := range j.keys {
		if _, found := slices.BinarySearch(keys, k); found {
			return true
		}
	}
	return false
}

// unionKeys returns the sorted keys of a and b (both sorted).
func unionKeys(a, b []saveKey) []saveKey {
	out := slices.Clone(a)
	for _, k := range b {
		if i, found := slices.BinarySearch(out, k); !found {
			out = slices.Insert(out, i, k)
		}
	}
	return out
}

// mergeJob adds src, written after dst, to dst, so that writing dst once
// stores what writing both in order would: a character's newer snapshot
// replaces the older one and their item changes merge; see mergeBatch for
// the rest.
func mergeJob(dst, src *saveJob) {
	for _, cs := range src.saves {
		if d := dst.save(cs.Character.ObjectID); d != nil {
			d.Character = cs.Character
			d.Changes = mergeItemChanges(d.Changes, cs.Changes)
		} else {
			dst.saves = append(dst.saves, cs)
		}
	}
	mergeBatch(&dst.extra, &src.extra)
	dst.dones = append(dst.dones, src.dones...)
	dst.keys = dst.touches()
}

// mergeBatch adds b, written after a, to a (Saves aside). Item changes of
// the same owner merge as in mergeItemChanges; a stored clan or crest
// replaces the stored or deleted one with its ID and a deleted one drops
// the stored one; an alliance crest replaces the one of its alliance; a
// removed relation cancels an added one and the other way round (a Batch
// applies removals first); patches keep their order. a's slices are only
// appended to or rebuilt, never shared with b.
func mergeBatch(a, b *storage.Batch) {
	for _, oc := range b.Items {
		i := slices.IndexFunc(a.Items, func(x storage.OwnerItemChanges) bool { return x.Owner == oc.Owner })
		if i < 0 {
			a.Items = append(a.Items, ownerItemChanges(oc.Owner, mergeItemChanges(storage.ItemChanges{}, itemChangesOf(oc))))
			continue
		}
		a.Items[i] = ownerItemChanges(oc.Owner, mergeItemChanges(itemChangesOf(a.Items[i]), itemChangesOf(oc)))
	}
	for _, id := range b.DeletedClans {
		a.Clans = slices.DeleteFunc(a.Clans, func(c storage.Clan) bool { return c.ID == id })
		if !slices.Contains(a.DeletedClans, id) {
			a.DeletedClans = append(a.DeletedClans, id)
		}
	}
	for _, c := range b.Clans {
		a.DeletedClans = slices.DeleteFunc(a.DeletedClans, func(id int32) bool { return id == c.ID })
		if i := slices.IndexFunc(a.Clans, func(x storage.Clan) bool { return x.ID == c.ID }); i >= 0 {
			a.Clans[i] = c
		} else {
			a.Clans = append(a.Clans, c)
		}
	}
	for _, ac := range b.AllyCrests {
		if i := slices.IndexFunc(a.AllyCrests, func(x storage.AllyCrest) bool { return x.AllyID == ac.AllyID }); i >= 0 {
			a.AllyCrests[i] = ac
		} else {
			a.AllyCrests = append(a.AllyCrests, ac)
		}
	}
	for _, id := range b.DeletedCrests {
		a.Crests = slices.DeleteFunc(a.Crests, func(c storage.Crest) bool { return c.ID == id })
		if !slices.Contains(a.DeletedCrests, id) {
			a.DeletedCrests = append(a.DeletedCrests, id)
		}
	}
	for _, c := range b.Crests {
		a.DeletedCrests = slices.DeleteFunc(a.DeletedCrests, func(id int32) bool { return id == c.ID })
		if i := slices.IndexFunc(a.Crests, func(x storage.Crest) bool { return x.ID == c.ID }); i >= 0 {
			a.Crests[i] = c
		} else {
			a.Crests = append(a.Crests, c)
		}
	}
	rel := &a.Relations
	for _, r := range b.Relations.Removed {
		rel.Added = slices.DeleteFunc(rel.Added, func(x storage.Relation) bool { return x == r })
		if !slices.Contains(rel.Removed, r) {
			rel.Removed = append(rel.Removed, r)
		}
	}
	for _, r := range b.Relations.Added {
		rel.Removed = slices.DeleteFunc(rel.Removed, func(x storage.Relation) bool { return x == r })
		if !slices.Contains(rel.Added, r) {
			rel.Added = append(rel.Added, r)
		}
	}
	a.Patches = append(a.Patches, b.Patches...)
}

func itemChangesOf(oc storage.OwnerItemChanges) storage.ItemChanges {
	return storage.ItemChanges{Upserts: oc.Upserts, Destroyed: oc.Destroyed, Left: oc.Left}
}

func ownerItemChanges(owner int32, ch storage.ItemChanges) storage.OwnerItemChanges {
	return storage.OwnerItemChanges{Owner: owner, Upserts: ch.Upserts, Destroyed: ch.Destroyed, Left: ch.Left}
}

// noItemChanges reports whether ch changes no item.
func noItemChanges(ch storage.ItemChanges) bool {
	return len(ch.Upserts) == 0 && len(ch.Destroyed) == 0 && len(ch.Left) == 0
}

// mergeItemChanges returns the changes of a, then b, as one set of changes
// with the meaning of storage.CharacterSaver.Save: each change of b
// replaces what a says about the same item, and the quest changes merge per
// quest, the newer winning (storage.QuestChanges.Merge).
func mergeItemChanges(a, b storage.ItemChanges) storage.ItemChanges {
	quests := a.Quests.Merge(b.Quests)
	if noItemChanges(b) {
		a.Quests = quests
		return a
	}
	out := storage.ItemChanges{
		Upserts:   slices.Clone(a.Upserts),
		Destroyed: slices.Clone(a.Destroyed),
		Left:      slices.Clone(a.Left),
		Quests:    quests,
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
