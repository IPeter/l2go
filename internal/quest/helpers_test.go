package quest

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/storage"
)

// Quests the tests know.
const (
	warrior = "Q00401_PathOfTheWarrior"
	letters = "Q00001_LettersOfLove"
	custom  = "CustomQuest"
	script  = "NpcScript"
	high    = "Q20000_High"
	gone    = "Q00999_Unloaded" // stored, but no script by this name
)

var testQuests = map[string]Info{
	warrior: {ID: 401, Name: warrior},
	letters: {ID: 1, Name: letters},
	custom:  {ID: 5000, Name: custom, Custom: true},
	script:  {ID: -1, Name: script},
	high:    {ID: 20000, Name: high},
}

func testLookup(name string) (Info, bool) {
	q, ok := testQuests[name]
	return q, ok
}

// recorder is a Listener that records what it is told, in order.
type recorder struct {
	events  []string
	changed int
}

func (r *recorder) QuestList() {
	r.events = append(r.events, "QuestList")
}

func (r *recorder) QuestMark(id int32) {
	r.events = append(r.events, fmt.Sprintf("QuestMark %d", id))
}

func (r *recorder) PlaySound(sound string) {
	r.events = append(r.events, "Sound "+sound)
}

func (r *recorder) RemoveDeathNotify(st *State) {
	r.events = append(r.events, "RemoveDeathNotify "+st.Name())
}

func (r *recorder) RemoveQuestItems(st *State) {
	r.events = append(r.events, "RemoveQuestItems "+st.Name())
}

func (r *recorder) Changed() { r.changed++ }

// take returns the recorded events and forgets them.
func (r *recorder) take() []string {
	ev := r.events
	r.events = nil
	return ev
}

// testNow is the fake clock of the tests: 05:10:20.345678 CET.
var testNow = time.Date(2026, time.November, 20, 5, 10, 20, 345_678_000, time.FixedZone("CET", 3600))

// newTestBook returns a Book with the test quests, a recorder and a clock
// that returns *now.
func newTestBook(now *time.Time) (*Book, *recorder) {
	r := &recorder{}
	return NewBook(testLookup, r, func() time.Time { return *now }), r
}

// loaded returns a Book that loaded docs, with nothing recorded yet.
func loaded(t *testing.T, docs ...storage.Quest) (*Book, *recorder) {
	t.Helper()
	now := testNow
	b, r := newTestBook(&now)
	if unknown := b.Load(docs); len(unknown) > 0 {
		t.Fatalf("Load: unknown quests %v", unknown)
	}
	if b.Dirty() || r.changed != 0 || len(r.events) != 0 {
		t.Fatalf("Load changed something: dirty %v, changed %d, events %v", b.Dirty(), r.changed, r.events)
	}
	return b, r
}

// vars builds a variable map from name, value pairs.
func vars(kv ...string) map[string]string {
	m := make(map[string]string, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return m
}

// checkEvents fails the test unless the recorder got exactly want.
func checkEvents(t *testing.T, r *recorder, want ...string) {
	t.Helper()
	if got := r.take(); !slices.Equal(got, want) {
		t.Errorf("events %q, want %q", got, want)
	}
}

// u is the Java int with these 32 bits, for writing flags in hex.
func u(bits uint32) int32 { return int32(bits) } //nolint:gosec // two's complement on purpose
