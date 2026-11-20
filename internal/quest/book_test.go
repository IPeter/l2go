package quest

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
)

func TestLoad(t *testing.T) {
	now := testNow
	b, r := newTestBook(&now)
	docs := []storage.Quest{
		{Name: warrior, Vars: vars("<state>", "Started", "cond", "4", VarFlags, "-2147483635", "memoState", "2")},
		{Name: letters, Vars: vars("<state>", "Completed")},
		{Name: gone, Vars: vars("<state>", "Started", "cond", "1")},
		// Variables without <state>: Created.
		{Name: custom, Vars: vars("seen", "1")},
		// An unknown state name: Created.
		{Name: high, Vars: vars("<state>", "Bogus")},
		// No rows: no state.
		{Name: script, Vars: nil},
	}
	unknown := b.Load(docs)
	if !slices.Equal(unknown, []string{gone}) {
		t.Errorf("Load unknown = %v, want [%s]", unknown, gone)
	}
	if b.Dirty() || r.changed != 0 || len(r.events) != 0 {
		t.Errorf("Load changed something: dirty %v, events %v", b.Dirty(), r.events)
	}
	w := b.Get(warrior)
	if w == nil || !w.IsStarted() || w.Cond() != 4 || w.MemoState() != 2 || w.IsSet(VarState) {
		t.Fatalf("warrior loaded as %+v", w)
	}
	if l := b.Get(letters); l == nil || !l.IsCompleted() || l.Vars() != nil {
		t.Errorf("letters loaded as %+v", l)
	}
	if c := b.Get(custom); c == nil || !c.IsCreated() || !c.IsSet("seen") {
		t.Errorf("custom loaded as %+v", c)
	}
	if h := b.Get(high); h == nil || !h.IsCreated() {
		t.Errorf("high loaded as %+v", h)
	}
	if b.Get(script) != nil || b.Get(gone) != nil {
		t.Error("Load made a state of an empty or unknown document")
	}

	// Saving everything gives the documents back, the unknown state name
	// as Mobius writes Created.
	want := []storage.Quest{
		{Name: custom, Vars: vars("seen", "1")},
		{Name: letters, Vars: vars("<state>", "Completed")},
		{Name: warrior, Vars: docs[0].Vars},
		{Name: high, Vars: vars("<state>", "Start")},
	}
	slices.SortFunc(want, func(a, b storage.Quest) int { return compareNames(a.Name, b.Name) })
	if got := b.Documents(); !questsEqual(got, want) {
		t.Errorf("Documents() = %+v, want %+v", got, want)
	}

	// The unknown quest's document is deleted only on request.
	b.DeleteStored(unknown...)
	b.DeleteStored(warrior) // has a state: ignored
	if ch := b.TakeChanges(); !changesEqual(ch, storage.QuestChanges{Deleted: []string{gone}}) {
		t.Errorf("TakeChanges after DeleteStored = %+v, want the unknown quest deleted", ch)
	}
}

// compareNames is the order of storage: plain string order.
func compareNames(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func TestLoadSetsCondOnlyFromInts(t *testing.T) {
	b, _ := loaded(t, storage.Quest{Name: warrior, Vars: vars("<state>", "Started", "cond", "x")})
	st := b.Get(warrior)
	if st.Cond() != 0 {
		t.Errorf("Cond() = %d, want 0 for a cond that is not an int", st.Cond())
	}
	if v, _ := st.Get(VarCond); v != "x" {
		t.Errorf("cond variable = %q, want it kept", v)
	}
}

func TestCreatedStatesAreNotSaved(t *testing.T) {
	b, r := loaded(t)
	st := b.State(warrior, true)
	if b.State(warrior, true) != st || b.State(warrior, false) != st {
		t.Error("State returned another state for the same quest")
	}
	if b.State(letters, false) != nil {
		t.Error("State(create false) made a state")
	}
	if b.Dirty() || r.changed != 0 {
		t.Fatal("a new state has something to save")
	}
	// Set and unset: Mobius inserted and deleted the row, so nothing is
	// stored; the save deletes the document, which may not exist.
	st.Set("x", "1")
	st.Unset("x")
	if ch := b.TakeChanges(); !changesEqual(ch, storage.QuestChanges{Deleted: []string{warrior}}) {
		t.Errorf("TakeChanges = %+v, want the document deleted", ch)
	}
}

func TestChangedOncePerSave(t *testing.T) {
	b, r := loaded(t)
	st := b.State(warrior, true)
	st.StartQuest()
	st.SetCond(2, false)
	b.State(letters, true).StartQuest()
	if r.changed != 1 {
		t.Errorf("Changed called %d times before the save, want once", r.changed)
	}
	ch := b.TakeChanges()
	if len(ch.Upserts) != 2 || b.Dirty() {
		t.Fatalf("TakeChanges = %+v, dirty after %v", ch, b.Dirty())
	}
	if ch := b.TakeChanges(); !ch.Empty() {
		t.Errorf("second TakeChanges = %+v, want nothing", ch)
	}
	st.SetCond(3, false)
	if r.changed != 2 {
		t.Errorf("Changed called %d times after the next change, want 2", r.changed)
	}
}

func TestSaveFailedKeepsQuestsDirty(t *testing.T) {
	b, r := loaded(t, storage.Quest{Name: letters, Vars: vars("<state>", "Started", "cond", "1")})
	st := b.State(warrior, true)
	st.StartQuest()
	b.Get(letters).ExitQuest(true, false)
	first := b.TakeChanges()
	if b.Dirty() {
		t.Fatal("dirty after TakeChanges")
	}

	b.SaveFailed(first)
	if !b.Dirty() || r.changed != 2 {
		t.Fatalf("after SaveFailed: dirty %v, Changed %d times; want dirty, a new save asked for", b.Dirty(), r.changed)
	}
	// The retry writes the quests as they are now.
	st.SetCond(3, false)
	retry := b.TakeChanges()
	want := storage.QuestChanges{
		Upserts: []storage.Quest{{Name: warrior, Vars: vars("<state>", "Started", "cond", "3", VarFlags, "-2147483643")}},
		Deleted: []string{letters},
	}
	if !changesEqual(retry, want) {
		t.Errorf("TakeChanges after SaveFailed = %+v, want %+v", retry, want)
	}
}

func TestNewReplacesState(t *testing.T) {
	b, _ := loaded(t, storage.Quest{Name: warrior, Vars: vars("<state>", "Started", "cond", "3")})
	old := b.Get(warrior)
	st := b.New(warrior)
	if b.Get(warrior) != st || st == old || !old.Detached() || !st.IsCreated() {
		t.Error("New did not replace the state")
	}
	old.Set("x", "1")
	if b.Dirty() {
		t.Error("a change to the replaced state was saved")
	}
}

func TestActive(t *testing.T) {
	b, _ := loaded(t,
		storage.Quest{Name: warrior, Vars: vars("<state>", "Started", "cond", "4", VarFlags, "-2147483635")},
		storage.Quest{Name: letters, Vars: vars("<state>", "Started", "cond", "2")},
		storage.Quest{Name: custom, Vars: vars("<state>", "Started", "cond", "7")},
		// IDs -1 and 20000: not listed.
		storage.Quest{Name: script, Vars: vars("<state>", "Started", "cond", "1")},
		storage.Quest{Name: high, Vars: vars("<state>", "Started", "cond", "1")},
	)
	b.State("Q00002_Unloaded", true).Set(VarCond, "1") // no script: not listed
	b.Load([]storage.Quest{{Name: letters + "x", Vars: vars("<state>", "Started")}})
	done := b.New("q00003_lower") // not loaded either
	done.SetState(Started)

	// Mobius order: by name, ignoring case; the flags are negative, so
	// the step is sent.
	want := []Entry{{ID: 5000, Status: 7}, {ID: 1, Status: 2}, {ID: 401, Status: 4}}
	if got := b.Active(); !slices.Equal(got, want) {
		t.Errorf("Active() = %+v, want %+v", got, want)
	}
	b.Get(letters).ExitQuest(false, false)
	if got := b.Active(); len(got) != 2 {
		t.Errorf("Active() after completing one = %+v, want 2 entries", got)
	}
}

func TestStatesOrder(t *testing.T) {
	b, _ := loaded(t)
	for _, name := range []string{"b", "C", "a", "A"} {
		b.New(name)
	}
	var names []string
	for _, st := range b.States() {
		names = append(names, st.Name())
	}
	if want := []string{"A", "a", "b", "C"}; !slices.Equal(names, want) {
		t.Errorf("States() order %v, want %v", names, want)
	}
}

// failingSaver fails its next Save; otherwise it saves to a memstore.
type failingSaver struct {
	*memstore.Characters
	fail error
}

func (s *failingSaver) Save(ctx context.Context, c *storage.Character, ch storage.ItemChanges) error {
	if err := s.fail; err != nil {
		s.fail = nil
		return err
	}
	return s.Characters.Save(ctx, c, ch)
}

// TestSaveAndReload runs the write-behind against a store: a quest walk is
// saved with the items' changes, a failed save keeps the quests dirty, and a
// new Book loads what was saved (the relog).
func TestSaveAndReload(t *testing.T) {
	ctx := t.Context()
	store := &failingSaver{Characters: memstore.NewCharacters()}
	c := storage.Character{ObjectID: 1, Account: "alice", Name: "Quester"}
	if err := store.Create(ctx, &c); err != nil {
		t.Fatalf("Create: %v", err)
	}
	stored, err := store.Get(ctx, 1)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	b, _ := loaded(t, stored.Quests...)

	save := func() error {
		t.Helper()
		ch := b.TakeChanges()
		err := store.Save(ctx, &c, storage.ItemChanges{Quests: ch})
		if err != nil {
			b.SaveFailed(ch)
		}
		return err
	}

	w := b.State(warrior, true)
	w.StartQuest()
	w.SetCond(3, true)
	b.State(letters, true).StartQuest()
	b.State(custom, true).Set("seen", "1")
	if err := save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	// A failed write: nothing reaches the store, the quests stay dirty.
	b.Get(letters).ExitQuest(false, true)
	w.SetCond(4, false)
	injected := errors.New("injected")
	store.fail = injected
	if err := save(); !errors.Is(err, injected) {
		t.Fatalf("save error = %v, want the injected failure", err)
	}
	if !b.Dirty() {
		t.Fatal("quests not dirty after a failed save")
	}
	got, err := store.Get(ctx, 1)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if q := questByName(got.Quests, letters); q == nil || q.Vars[VarState] != "Started" {
		t.Errorf("the failed save reached the store: %+v", got.Quests)
	}
	if err := save(); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if b.Dirty() {
		t.Error("dirty after the retry")
	}

	// The relog: a new Book from the stored documents is the same.
	got, err = store.Get(ctx, 1)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !questsEqual(got.Quests, b.Documents()) {
		t.Errorf("stored %+v, want %+v", got.Quests, b.Documents())
	}
	again, _ := loaded(t, got.Quests...)
	for _, name := range []string{warrior, letters, custom} {
		x, y := b.Get(name), again.Get(name)
		if y == nil || x.State() != y.State() || x.Cond() != y.Cond() || !maps.Equal(x.Vars(), y.Vars()) {
			t.Errorf("%s after the relog: %+v, want %+v", name, y, x)
		}
	}
	if w := again.Get(warrior); w.Cond() != 4 || !w.IsStarted() {
		t.Errorf("warrior after the relog: %v, cond %d", w.State(), w.Cond())
	}
	if !slices.Equal(again.Active(), b.Active()) {
		t.Errorf("Active() after the relog = %+v, want %+v", again.Active(), b.Active())
	}

	// Abort: the document goes.
	again.Get(warrior).ExitQuest(true, false)
	b = again
	if err := save(); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, _ = store.Get(ctx, 1)
	if questByName(got.Quests, warrior) != nil {
		t.Errorf("aborted quest still stored: %+v", got.Quests)
	}
}

func questByName(quests []storage.Quest, name string) *storage.Quest {
	for i := range quests {
		if quests[i].Name == name {
			return &quests[i]
		}
	}
	return nil
}

func questsEqual(a, b []storage.Quest) bool {
	return slices.EqualFunc(a, b, func(x, y storage.Quest) bool {
		return x.Name == y.Name && maps.Equal(x.Vars, y.Vars)
	})
}
