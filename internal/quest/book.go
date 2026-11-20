package quest

import (
	"cmp"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/storage"
)

// Book is one character's quests: Mobius Player's quest map with the
// persistence of Mobius Quest (playerEnter, the *InDb methods) turned into
// write-behind documents (decision Q4).
//
// A change marks its quest dirty; TakeChanges returns the documents of the
// dirty quests for the next save and SaveFailed marks them dirty again.
// States that were never saved and have no variables (a Created state just
// looked at) are not saved, as in Mobius.
type Book struct {
	lookup   Lookup
	listener Listener
	now      func() time.Time
	states   map[string]*State
	dirty    map[string]struct{}
}

// NewBook returns an empty Book. lookup finds the loaded quest scripts (nil
// knows none); l hears about packets, sounds and items (nil ignores them);
// now is the clock for daily quests (nil is time.Now), on the loop the
// loop's clock.
func NewBook(lookup Lookup, l Listener, now func() time.Time) *Book {
	if lookup == nil {
		lookup = func(string) (Info, bool) { return Info{}, false }
	}
	if l == nil {
		l = NopListener{}
	}
	if now == nil {
		now = time.Now
	}
	return &Book{
		lookup:   lookup,
		listener: l,
		now:      now,
		states:   make(map[string]*State),
		dirty:    make(map[string]struct{}),
	}
}

// quest looks a quest script up.
func (b *Book) quest(name string) (Info, bool) { return b.lookup(name) }

// Load adds the stored quests of a character, as Get returned them (Mobius
// playerEnter): a document's "<state>" gives the state (Created for a
// document without it or with an unknown name), its other entries are the
// variables, and "cond" the step. Loading changes nothing to save.
//
// Documents of quests that lookup does not know are skipped and their names
// returned, in document order; they stay stored unless the caller deletes
// them (general.autoDeleteInvalidQuestData) with DeleteStored.
func (b *Book) Load(docs []storage.Quest) (unknown []string) {
	for _, d := range docs {
		if _, ok := b.lookup(d.Name); !ok {
			unknown = append(unknown, d.Name)
			continue
		}
		if len(d.Vars) == 0 {
			continue // no rows, no state
		}
		st := b.states[d.Name]
		if st == nil {
			st = &State{book: b, name: d.Name}
			b.states[d.Name] = st
		}
		for _, k := range slices.Sorted(maps.Keys(d.Vars)) {
			if k == VarState {
				st.state = StateFromName(d.Vars[k])
				st.stored = true
				continue
			}
			st.setInternal(k, d.Vars[k])
		}
	}
	return unknown
}

// DeleteStored marks quests without a state in the Book for deletion from
// storage at the next save, such as the unknown quests Load returned.
func (b *Book) DeleteStored(names ...string) {
	for _, name := range names {
		if b.states[name] == nil {
			b.markDirty(name)
		}
	}
}

// Get returns the state of a quest, or nil (Mobius
// getQuestState(player, false)).
func (b *Book) Get(name string) *State { return b.states[name] }

// State returns the state of a quest; if there is none and create is true,
// it creates one, Created and empty (Mobius getQuestState(player, create)).
// A new state is saved only once it changes.
func (b *Book) State(name string, create bool) *State {
	if st := b.states[name]; st != nil || !create {
		return st
	}
	return b.New(name)
}

// New creates a new, Created and empty state of a quest and returns it
// (Mobius newQuestState). It replaces the quest's state if it had one; the
// old state is detached.
func (b *Book) New(name string) *State {
	if old := b.states[name]; old != nil {
		old.detached = true
	}
	st := &State{book: b, name: name}
	b.states[name] = st
	return st
}

// remove takes st out of the Book and marks its document for deletion.
func (b *Book) remove(st *State) {
	if b.states[st.name] == st {
		delete(b.states, st.name)
		b.markDirty(st.name)
	}
	st.detached = true
}

// States returns every state in Mobius's order: by name, ignoring case.
func (b *Book) States() []*State {
	list := slices.Collect(maps.Values(b.states))
	slices.SortFunc(list, func(x, y *State) int {
		return cmp.Or(cmp.Compare(strings.ToLower(x.name), strings.ToLower(y.name)), cmp.Compare(x.name, y.name))
	})
	return list
}

// Entry is one quest of the QuestList packet.
type Entry struct {
	ID     int32 // the quest ID
	Status int32 // the skipped-step flags if positive, else the step
}

// Active returns the QuestList entries (Mobius getAllActiveQuests and
// QuestList.writeImpl): the started quests whose script is loaded with an ID
// from 1 to MaxListID, in States order. Its length is what the quest limit
// counts.
func (b *Book) Active() []Entry {
	var list []Entry
	for _, st := range b.States() {
		if !st.IsStarted() {
			continue
		}
		q, ok := b.lookup(st.name)
		if !ok || q.ID < 1 || q.ID > MaxListID {
			continue
		}
		status := st.GetInt(VarFlags)
		if status <= 0 {
			status = st.Cond()
		}
		list = append(list, Entry{ID: q.ID, Status: status})
	}
	return list
}

// markDirty marks a quest to be saved and tells the listener when the Book
// had nothing to save before.
func (b *Book) markDirty(name string) {
	if _, ok := b.dirty[name]; ok {
		return
	}
	clean := len(b.dirty) == 0
	b.dirty[name] = struct{}{}
	if clean {
		b.listener.Changed()
	}
}

// Dirty reports whether the Book has changes to save.
func (b *Book) Dirty() bool { return len(b.dirty) > 0 }

// TakeChanges returns the changes to save, by quest name, and forgets them:
// the full document of every dirty quest that has a state with something to
// store, and the deletion of the others. Pass them to the save
// (storage.ItemChanges.Quests); if the save fails, give them to SaveFailed.
func (b *Book) TakeChanges() storage.QuestChanges {
	var ch storage.QuestChanges
	for _, name := range slices.Sorted(maps.Keys(b.dirty)) {
		if doc, ok := b.document(name); ok {
			ch.Upserts = append(ch.Upserts, doc)
		} else {
			ch.Deleted = append(ch.Deleted, name)
		}
	}
	clear(b.dirty)
	return ch
}

// SaveFailed marks the quests of changes that could not be written dirty
// again, so the next save writes them as they are then.
func (b *Book) SaveFailed(changes storage.QuestChanges) {
	for _, q := range changes.Upserts {
		b.markDirty(q.Name)
	}
	for _, name := range changes.Deleted {
		b.markDirty(name)
	}
}

// Documents returns the documents of every quest with something to store,
// in name order: what a full save of the character stores.
func (b *Book) Documents() []storage.Quest {
	var docs []storage.Quest
	for _, name := range slices.Sorted(maps.Keys(b.states)) {
		if doc, ok := b.document(name); ok {
			docs = append(docs, doc)
		}
	}
	return docs
}

// document returns the stored form of a quest's state: one entry per Mobius
// character_quests row. It returns false when Mobius would have no row: no
// state, or a Created state never saved with no variables.
func (b *Book) document(name string) (storage.Quest, bool) {
	st := b.states[name]
	if st == nil {
		return storage.Quest{}, false
	}
	vars := maps.Clone(st.vars)
	if st.stored {
		if vars == nil {
			vars = make(map[string]string, 1)
		}
		vars[VarState] = st.state.Name()
	}
	if len(vars) == 0 {
		return storage.Quest{}, false
	}
	return storage.Quest{Name: name, Vars: vars}, true
}
