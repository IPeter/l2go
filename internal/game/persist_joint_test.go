package game

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/storage"
)

// batchStore is a storage.CharacterSaver and storage.BatchWriter that only
// records the writes, in order. The first write waits until release is
// called, so that a test can fill the saver's queue behind it.
type batchStore struct {
	mu      sync.Mutex
	hold    chan struct{}
	started chan struct{}
	writes  []string        // "save 1" or "batch 1,2"
	batches []storage.Batch // every write as a batch
	fail    error           // returned by every write
}

func (s *batchStore) Save(ctx context.Context, c *storage.Character, ch storage.ItemChanges) error {
	return s.record(ctx, "save", storage.Batch{Saves: []storage.CharacterSave{{Character: *c, Changes: ch}}})
}

func (s *batchStore) WriteBatch(ctx context.Context, b storage.Batch) error {
	return s.record(ctx, "batch", b)
}

func (s *batchStore) record(ctx context.Context, kind string, b storage.Batch) error {
	select {
	case s.started <- struct{}{}:
	default:
	}
	s.mu.Lock()
	hold := s.hold
	s.hold = nil
	s.mu.Unlock()
	if hold != nil {
		select {
		case <-hold:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail != nil {
		return s.fail
	}
	ids := make([]string, len(b.Saves))
	for i, cs := range b.Saves {
		ids[i] = fmt.Sprint(cs.Character.ObjectID)
	}
	s.writes = append(s.writes, kind+" "+strings.Join(ids, ","))
	s.batches = append(s.batches, b)
	return nil
}

// jointSaver starts a saver on a batchStore with a save of character 999
// being written and held, so that what the test enqueues waits in line.
// finish releases the held write, closes the saver and waits until
// everything is written; it returns the writes after the held one.
func jointSaver(t *testing.T) (s *saver, store *batchStore, finish func() ([]string, []storage.Batch)) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	hold := make(chan struct{})
	store = &batchStore{hold: hold, started: make(chan struct{}, 1)}
	s = newSaver(store, slog.New(slog.DiscardHandler))
	s.retryDelay = time.Millisecond
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.run(ctx)
	}()
	s.enqueue(storage.Character{ObjectID: 999}, storage.ItemChanges{}, nil)
	select {
	case <-store.started:
	case <-ctx.Done():
		t.Fatal("the first save was not taken up")
	}
	var once sync.Once
	finish = func() ([]string, []storage.Batch) {
		once.Do(func() {
			close(hold)
			s.close()
			select {
			case <-done:
			case <-ctx.Done():
				t.Fatal("the saver did not finish")
			}
		})
		store.mu.Lock()
		defer store.mu.Unlock()
		if len(store.writes) == 0 {
			return nil, nil
		}
		return slices.Clone(store.writes[1:]), slices.Clone(store.batches[1:])
	}
	t.Cleanup(func() { finish() })
	return s, store, finish
}

// upsert is item changes that store one item.
func jointItem(objectID int32) storage.ItemChanges {
	return storage.ItemChanges{Upserts: []storage.Item{{ObjectID: objectID, ItemID: 57, Count: 1}}}
}

func jointChar(id int32) storage.Character { return storage.Character{ObjectID: id} }

func upsertIDs(ch storage.ItemChanges) []int32 {
	var ids []int32
	for _, it := range ch.Upserts {
		ids = append(ids, it.ObjectID)
	}
	return ids
}

// A joint save takes the place of the earliest pending save of a listed
// character and takes in that character's later saves; other saves keep
// their order behind it.
func TestSaverJointTakesTheEarliestPlace(t *testing.T) {
	s, _, finish := jointSaver(t)
	s.enqueue(jointChar(1), jointItem(101), nil)
	s.enqueue(jointChar(3), jointItem(301), nil)
	s.enqueue(jointChar(1), jointItem(102), nil)
	s.enqueueJoint([]storage.CharacterSave{
		{Character: jointChar(1), Changes: jointItem(103)},
		{Character: jointChar(2), Changes: jointItem(201)},
	}, storage.Batch{}, nil)
	writes, batches := finish()
	if want := []string{"batch 1,2", "save 3"}; !slices.Equal(writes, want) {
		t.Fatalf("writes %q, want %q", writes, want)
	}
	b := batches[0]
	if got := upsertIDs(b.Saves[0].Changes); !slices.Equal(got, []int32{101, 102, 103}) {
		t.Errorf("character 1's items in the joint save %v, want all three changes", got)
	}
	if got := upsertIDs(b.Saves[1].Changes); !slices.Equal(got, []int32{201}) {
		t.Errorf("character 2's items in the joint save %v", got)
	}
}

// Without a pending save of a listed character, a joint save goes after
// every pending write: a patch or a relation change is written after the
// pending saves of its characters, and loading a character waits for it.
func TestSaverJointAfterPendingWrites(t *testing.T) {
	s, _, finish := jointSaver(t)
	s.enqueue(jointChar(1), jointItem(101), nil)
	s.enqueue(jointChar(2), jointItem(201), nil)
	s.enqueueJoint(nil, storage.Batch{
		Patches:   []storage.CharacterPatch{{ObjectID: 1, Fields: storage.PatchClanID}},
		Relations: storage.RelationChanges{Added: storage.FriendRelations(1, 3)},
	}, nil)
	// Character 3 has only the relation pending: loading it waits.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := s.wait(ctx, 3); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("wait for character 3 with the relation pending: %v, want a timeout", err)
	}
	writes, batches := finish()
	if want := []string{"save 1", "save 2", "batch "}; !slices.Equal(writes, want) {
		t.Fatalf("writes %q, want %q", writes, want)
	}
	if b := batches[2]; len(b.Patches) != 1 || len(b.Relations.Added) != 2 {
		t.Errorf("joint save %+v", b)
	}
	if err := s.wait(context.Background(), 3); err != nil {
		t.Errorf("wait after the write: %v", err)
	}
}

// A joint save also takes in what the writes it takes in touch, so that no
// write moves past an older write of the same character.
func TestSaverJointMergesTransitively(t *testing.T) {
	s, _, finish := jointSaver(t)
	s.enqueue(jointChar(3), jointItem(301), nil) // C
	s.enqueue(jointChar(1), jointItem(101), nil) // A
	s.enqueueJoint(nil, storage.Batch{Relations: storage.RelationChanges{Added: storage.FriendRelations(1, 2)}}, nil)
	s.enqueue(jointChar(4), jointItem(401), nil) // D, untouched
	s.enqueueJoint([]storage.CharacterSave{
		{Character: jointChar(2), Changes: jointItem(201)}, // B
		{Character: jointChar(3), Changes: jointItem(302)}, // C
	}, storage.Batch{}, nil)
	writes, batches := finish()
	// C's save is first: the relation batch touches B and A, so A's save
	// comes along, and the result takes C's place.
	if want := []string{"batch 3,1,2", "save 4"}; !slices.Equal(writes, want) {
		t.Fatalf("writes %q, want %q", writes, want)
	}
	if b := batches[0]; len(b.Relations.Added) != 2 || !slices.Equal(upsertIDs(b.Saves[0].Changes), []int32{301, 302}) {
		t.Errorf("joint save %+v", b)
	}
}

// A plain save does not merge into an earlier save of its character when a
// later write touches that character.
func TestSaverSaveStaysBehindALaterBatch(t *testing.T) {
	s, _, finish := jointSaver(t)
	first := jointChar(1)
	first.X = 1
	s.enqueue(first, storage.ItemChanges{}, nil)
	s.enqueueJoint(nil, storage.Batch{Relations: storage.RelationChanges{Added: storage.FriendRelations(1, 2)}}, nil)
	second := jointChar(1)
	second.X = 2
	s.enqueue(second, storage.ItemChanges{}, nil)
	writes, batches := finish()
	if want := []string{"save 1", "batch ", "save 1"}; !slices.Equal(writes, want) {
		t.Fatalf("writes %q, want %q", writes, want)
	}
	if batches[0].Saves[0].Character.X != 1 || batches[2].Saves[0].Character.X != 2 {
		t.Error("the snapshots were written out of order")
	}
}

// A joint save that fails for good (a missing character) is not retried,
// and its done function gets the error.
func TestSaverJointPermanentFailure(t *testing.T) {
	s, store, finish := jointSaver(t)
	store.mu.Lock()
	store.fail = storage.ErrNotFound
	store.mu.Unlock()
	got := make(chan error, 1)
	s.enqueueJoint([]storage.CharacterSave{{Character: jointChar(1)}, {Character: jointChar(2)}}, storage.Batch{}, func(err error) { got <- err })
	finish()
	if err := <-got; !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("done got %v, want ErrNotFound", err)
	}
}

// An empty joint save writes nothing and is done at once.
func TestSaverJointEmpty(t *testing.T) {
	s, _, finish := jointSaver(t)
	called := false
	s.enqueueJoint(nil, storage.Batch{}, func(err error) { called = err == nil })
	if writes, _ := finish(); len(writes) != 0 || !called {
		t.Errorf("writes %q, done called %v", writes, called)
	}
}

func TestMergeBatch(t *testing.T) {
	ab := storage.Relation{Owner: 1, Target: 2, Type: storage.RelationFriend}
	blk := storage.Relation{Owner: 1, Target: 3, Type: storage.RelationBlock}
	var a storage.Batch
	mergeBatch(&a, &storage.Batch{
		Relations: storage.RelationChanges{Added: []storage.Relation{ab}, Removed: []storage.Relation{blk}},
		Clans:     []storage.Clan{{ID: 10, Name: "Old"}, {ID: 11}},
		Items:     []storage.OwnerItemChanges{{Owner: 10, Upserts: []storage.Item{{ObjectID: 1}}}},
		Patches:   []storage.CharacterPatch{{ObjectID: 1}},
	})
	mergeBatch(&a, &storage.Batch{
		Relations:    storage.RelationChanges{Removed: []storage.Relation{ab}, Added: []storage.Relation{blk}},
		Clans:        []storage.Clan{{ID: 10, Name: "New"}},
		DeletedClans: []int32{11},
		Items:        []storage.OwnerItemChanges{{Owner: 10, Destroyed: []int32{1}}},
		Patches:      []storage.CharacterPatch{{ObjectID: 2}},
	})
	if !slices.Equal(a.Relations.Removed, []storage.Relation{ab}) || !slices.Equal(a.Relations.Added, []storage.Relation{blk}) {
		t.Errorf("relations %+v: the later change of each relation should win", a.Relations)
	}
	if len(a.Clans) != 1 || a.Clans[0].Name != "New" || !slices.Equal(a.DeletedClans, []int32{11}) {
		t.Errorf("clans %+v, deleted %v", a.Clans, a.DeletedClans)
	}
	if len(a.Items) != 1 || len(a.Items[0].Upserts) != 0 || !slices.Equal(a.Items[0].Destroyed, []int32{1}) {
		t.Errorf("clan items %+v", a.Items)
	}
	if len(a.Patches) != 2 || a.Patches[0].ObjectID != 1 {
		t.Errorf("patches %+v", a.Patches)
	}
}

// SaveJoint writes the players' pending changes with the extra writes, and
// a joint save that fails for good disconnects the players without a save.
func TestSaveJointOnTheServer(t *testing.T) {
	e := newEnv(t, nil)
	e.createCharacter("alice", "Alice", classHumanFighter, storage.AccessLevelPlayer)
	e.createCharacter("bob", "Bob", classHumanFighter, storage.AccessLevelPlayer)
	sa, sb := newSession("alice"), newSession("bob")
	pa, pb := e.enter(sa, 0), e.enter(sb, 0)

	done := make(chan error, 1)
	e.onLoop(func() {
		e.srv.saveJoint([]*Player{pa, pb, pa}, storage.Batch{
			Relations: storage.RelationChanges{Added: storage.FriendRelations(pa.ObjectID(), pb.ObjectID())},
		}, func(err error) { done <- err })
	})
	if err := <-done; err != nil {
		t.Fatalf("joint save: %v", err)
	}
	rels, err := e.store.Relations().Relations(e.ctx, pa.ObjectID())
	if err != nil || len(rels) != 1 || rels[0].Target != pb.ObjectID() {
		t.Errorf("Alice's relations %+v (%v)", rels, err)
	}

	// A save of a character that does not exist fails the whole batch.
	saves := e.store.saveCount(pa.ObjectID())
	e.onLoop(func() {
		e.srv.SaveJoint([]*Player{pa}, storage.Batch{Saves: []storage.CharacterSave{{Character: storage.Character{ObjectID: 1 << 30}}}})
	})
	sa.waitFor(t, "Alice's disconnection", func(s *fakeSession) bool { return s.closed })
	var inGame bool
	e.onLoop(func() { inGame = pa.InGame() })
	if inGame {
		t.Error("Alice is still in the game after a failed joint save")
	}
	if n := e.store.saveCount(pa.ObjectID()); n != saves {
		t.Errorf("Alice was saved %d more times after the failure, want none", n-saves)
	}
	if !e.inWorld(pb) {
		t.Error("Bob, not in the failed joint save, left the world")
	}
}
