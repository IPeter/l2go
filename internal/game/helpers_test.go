package game

import (
	"context"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// testStart is when the fake clock starts: 13:00 UTC, which is 06:00 game
// time (13 real hours are 3 game days and 6 game hours after midnight).
var testStart = time.Date(2026, 11, 19, 13, 0, 0, 0, time.UTC)

// testGameTime is the game time at testStart, in minutes.
const testGameTime = 6 * 60

// Human Fighter (class 0) and its starting items (initialEquipment.xml).
const (
	classHumanFighter = 0
	classHumanMystic  = 10
	itemSquireSword   = 2369 // equipped, right hand
	itemSquireShirt   = 1146 // equipped, chest
	itemSquirePants   = 1147 // equipped, legs
)

var loadTestData = sync.OnceValues(func() (*data.Data, error) {
	return data.Load(filepath.Join("..", "..", "data"))
})

func testGameData(t *testing.T) *data.Data {
	t.Helper()
	d, err := loadTestData()
	if err != nil {
		t.Fatalf("load the game data: %v", err)
	}
	return d
}

// testEnv is a running game server on a fake clock, with an in-memory
// character store.
type testEnv struct {
	t     *testing.T
	ctx   context.Context // for the test's calls; ends with the test
	srv   *Server
	store *recordingStore
	ids   *memstore.IDs
	clock *world.FakeClock
	data  *data.Data
	// stop stops the server and waits for Run to return. The test's cleanup
	// calls it too; later calls do nothing.
	stop func()
}

// newEnv starts a server. configure, if not nil, changes the game server's
// config first; autosave is off unless it turns it on. Extra features can be
// registered on the server before it runs with setup.
func newEnv(t *testing.T, configure func(gs *config.GameServer), setup ...func(s *Server)) *testEnv {
	t.Helper()
	d := testGameData(t)
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval = 0
	if configure != nil {
		configure(gs)
	}
	store := newRecordingStore()
	clock := world.NewFakeClock(testStart)
	ids := memstore.NewIDs()
	srv, err := New(Options{
		Config:         cfg,
		ServerID:       gs.ID,
		Data:           d,
		Characters:     store,
		IDs:            ids,
		Clock:          clock,
		CheckLoopOwner: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for _, fn := range setup {
		fn(srv)
	}

	runCtx, stopRun := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Run(runCtx) }()
	var once sync.Once
	stop := func() {
		once.Do(func() {
			stopRun()
			if err := <-done; err != nil {
				t.Errorf("Run: %v", err)
			}
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(func() {
		stop()
		cancel()
	})
	return &testEnv{t: t, ctx: ctx, srv: srv, store: store, ids: ids, clock: clock, data: d, stop: stop}
}

// onLoop runs fn on the loop and waits for it.
func (e *testEnv) onLoop(fn func()) {
	e.t.Helper()
	if err := e.srv.OnLoop(e.ctx, fn); err != nil {
		e.t.Fatalf("OnLoop: %v", err)
	}
}

// sync waits until the loop has run everything posted so far, and every
// timer and tick due on the fake clock.
func (e *testEnv) sync() { e.onLoop(func() {}) }

// advance moves the fake clock by d and waits until the loop has run what
// became due.
func (e *testEnv) advance(d time.Duration) {
	e.t.Helper()
	e.clock.Advance(d)
	e.sync()
}

// createCharacter stores a new level 1 character of a starting class at its
// first creation point with its initial items, the equipped ones worn.
func (e *testEnv) createCharacter(account, name string, classID int, access storage.AccessLevel) storage.Character {
	e.t.Helper()
	tmpl, ok := e.data.Template(classID)
	if !ok {
		e.t.Fatalf("no starting class %d", classID)
	}
	start := e.data.InitialItems(classID)
	ids, err := e.ids.NextIDs(e.ctx, 1+len(start))
	if err != nil {
		e.t.Fatalf("NextIDs: %v", err)
	}
	items := make([]storage.Item, len(start))
	for i, it := range start {
		items[i] = storage.Item{ObjectID: ids[1+i], ItemID: int32(it.ItemID), Count: it.Count, Slot: storage.PaperdollNone}
		if it.Equipped {
			items[i].Slot = e.slotOf(it.ItemID)
		}
	}
	loc := tmpl.CreationPoints[0]
	c := storage.Character{
		ObjectID:    ids[0],
		Account:     account,
		Name:        name,
		Race:        int32(tmpl.Race),
		ClassID:     int32(classID),
		BaseClassID: int32(classID),
		Level:       1,
		CurHP:       tmpl.Level1.HP,
		MaxHP:       tmpl.Level1.HP,
		CurMP:       tmpl.Level1.MP,
		MaxMP:       tmpl.Level1.MP,
		MaxCP:       tmpl.Level1.CP,
		X:           int32(loc.X),
		Y:           int32(loc.Y),
		Z:           int32(loc.Z),
		AccessLevel: access,
		Items:       items,
	}
	if err := e.store.Create(e.ctx, &c); err != nil {
		e.t.Fatalf("Create: %v", err)
	}
	return c
}

// slotOf returns the paperdoll slot an item is worn in.
func (e *testEnv) slotOf(itemID int) storage.PaperdollSlot {
	e.t.Helper()
	it, ok := e.data.ItemTemplate(itemID)
	if !ok {
		e.t.Fatalf("no item %d", itemID)
	}
	switch part := it.Part(); part {
	case data.BodyPartLRHand:
		return storage.PaperdollRHand
	case data.BodyPartFullArmor:
		return storage.PaperdollChest
	case data.BodyPartLREar:
		return storage.PaperdollREar
	case data.BodyPartLRFinger:
		return storage.PaperdollRFinger
	default:
		if s := part.PaperdollSlot(); s >= 0 && s < storage.PaperdollTotalSlots {
			return storage.PaperdollSlot(s)
		}
	}
	e.t.Fatalf("item %d cannot be worn", itemID)
	return storage.PaperdollNone
}

// enter selects the character in slot for sess and enters the world with
// it. The packets stay in the session.
func (e *testEnv) enter(sess *fakeSession, slot int) *Player {
	e.t.Helper()
	if err := e.srv.SelectCharacter(e.ctx, sess, slot); err != nil {
		e.t.Fatalf("SelectCharacter: %v", err)
	}
	e.srv.EnterWorld(sess)
	var p *Player
	var inGame bool
	e.onLoop(func() {
		p = e.srv.playerOf(sess)
		inGame = p != nil && p.InGame()
	})
	if !inGame {
		e.t.Fatalf("%s is not in the game after EnterWorld", sess.account)
	}
	return p
}

// position returns a player's position, read on the loop.
func (e *testEnv) position(p *Player) world.Position {
	var pos world.Position
	e.onLoop(func() { pos = p.Position() })
	return pos
}

// inWorld reports whether a player is in the world, read on the loop.
func (e *testEnv) inWorld(p *Player) bool {
	var in bool
	e.onLoop(func() { in = p.InWorld() })
	return in
}

// fakeSession is a Session that records what the game does with it.
type fakeSession struct {
	account string

	mu       sync.Mutex
	state    State
	bodies   [][]byte
	closed   bool
	final    []byte
	delay    time.Duration
	returned int // calls of ReturnToCharacterSelect
	notify   chan struct{}
}

var _ Session = (*fakeSession)(nil)

func newSession(account string) *fakeSession {
	return &fakeSession{account: account, state: StateAuthenticated, notify: make(chan struct{}, 1)}
}

// testPlayKey is the PlayOK1 session key of every fake session.
const testPlayKey int32 = 0x5eed

func (s *fakeSession) Send(body []byte) {
	s.mu.Lock()
	s.bodies = append(s.bodies, body)
	s.mu.Unlock()
	s.signal()
}

func (s *fakeSession) signal() {
	select {
	case s.notify <- struct{}{}:
	default:
	}
}

func (s *fakeSession) Account() string { return s.account }

func (s *fakeSession) PlayKey1() int32 { return testPlayKey }

func (s *fakeSession) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *fakeSession) SetState(st State) {
	s.mu.Lock()
	s.state = st
	s.mu.Unlock()
}

func (s *fakeSession) ReturnToCharacterSelect(context.Context) {
	s.mu.Lock()
	s.returned++
	s.state = StateAuthenticated
	s.mu.Unlock()
	s.signal()
}

func (s *fakeSession) CloseAfter(final []byte, delay time.Duration) {
	s.mu.Lock()
	s.closed, s.final, s.delay = true, final, delay
	s.mu.Unlock()
	s.signal()
}

func (s *fakeSession) RemoteAddr() string { return "test/" + s.account }

// take decodes and removes the packets sent so far.
func (s *fakeSession) take(t *testing.T) []testclient.Packet {
	t.Helper()
	s.mu.Lock()
	bodies := s.bodies
	s.bodies = nil
	s.mu.Unlock()
	ps := make([]testclient.Packet, 0, len(bodies))
	for _, b := range bodies {
		p, err := testclient.DecodeServerPacket(b)
		if err != nil {
			t.Fatalf("decode % x: %v", b, err)
		}
		ps = append(ps, p)
	}
	return ps
}

// closedWith returns the final packet the session was closed with, decoded,
// or false when it is open.
func (s *fakeSession) closedWith(t *testing.T) (testclient.Packet, time.Duration, bool) {
	t.Helper()
	s.mu.Lock()
	closed, final, delay := s.closed, s.final, s.delay
	s.mu.Unlock()
	if !closed {
		return nil, 0, false
	}
	if final == nil {
		return nil, delay, true
	}
	p, err := testclient.DecodeServerPacket(final)
	if err != nil {
		t.Fatalf("decode the final packet % x: %v", final, err)
	}
	return p, delay, true
}

// waitFor waits until cond holds; cond runs under the session's lock.
func (s *fakeSession) waitFor(t *testing.T, what string, cond func(s *fakeSession) bool) {
	t.Helper()
	timeout := time.NewTimer(10 * time.Second)
	defer timeout.Stop()
	for {
		s.mu.Lock()
		ok := cond(s)
		s.mu.Unlock()
		if ok {
			return
		}
		select {
		case <-s.notify:
		case <-timeout.C:
			t.Fatalf("timed out waiting for %s", what)
		}
	}
}

// recordingStore is the in-memory store; it also keeps every saved snapshot
// and can hold saves back.
type recordingStore struct {
	*memstore.Characters

	mu      sync.Mutex
	saves   []storage.Character
	hold    chan struct{}          // while not nil, saves wait until it is closed
	started chan int32             // the object ID of every save that starts
	saved   chan storage.Character // every finished save
}

func newRecordingStore() *recordingStore {
	return &recordingStore{
		Characters: memstore.NewCharacters(),
		started:    make(chan int32, 256),
		saved:      make(chan storage.Character, 256),
	}
}

func (s *recordingStore) Update(ctx context.Context, c *storage.Character) error {
	select {
	case s.started <- c.ObjectID:
	default:
	}
	s.mu.Lock()
	hold := s.hold
	s.mu.Unlock()
	if hold != nil {
		select {
		case <-hold:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if err := s.Characters.Update(ctx, c); err != nil {
		return err
	}
	snap := *c
	snap.Items = slices.Clone(c.Items)
	s.mu.Lock()
	s.saves = append(s.saves, snap)
	s.mu.Unlock()
	select {
	case s.saved <- snap:
	default:
	}
	return nil
}

// holdSaves makes saves wait until the returned function is called.
func (s *recordingStore) holdSaves() (release func()) {
	ch := make(chan struct{})
	s.mu.Lock()
	s.hold = ch
	s.mu.Unlock()
	return func() {
		s.mu.Lock()
		s.hold = nil
		s.mu.Unlock()
		close(ch)
	}
}

// saveCount returns the number of saves of a character so far.
func (s *recordingStore) saveCount(objectID int32) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, c := range s.saves {
		if c.ObjectID == objectID {
			n++
		}
	}
	return n
}

// waitSave waits for the next save of a character and returns it.
func (s *recordingStore) waitSave(t *testing.T, objectID int32) storage.Character {
	t.Helper()
	timeout := time.NewTimer(10 * time.Second)
	defer timeout.Stop()
	for {
		select {
		case c := <-s.saved:
			if c.ObjectID == objectID {
				return c
			}
		case <-timeout.C:
			t.Fatalf("character %d was not saved", objectID)
			return storage.Character{}
		}
	}
}

// stringBody encodes a client packet body that is one string.
func stringBody(s string) []byte {
	w := packets.NewWriter(2*len(s) + 2)
	w.WriteString(s)
	return w.Bytes()
}

// int32Body encodes a client packet body that is one int.
func int32Body(v int32) []byte {
	w := packets.NewWriter(4)
	w.WriteInt32(v)
	return w.Bytes()
}

// checkNames fails the test unless the packets have the given names, in
// order.
func checkNames(t *testing.T, what string, ps []testclient.Packet, want ...string) {
	t.Helper()
	if got := testclient.Names(ps); !slices.Equal(got, want) {
		t.Fatalf("%s:\n got %q\nwant %q", what, got, want)
	}
}

// only returns the packets of type T.
func only[T testclient.Packet](ps []testclient.Packet) []T { return testclient.FindAll[T](ps) }
