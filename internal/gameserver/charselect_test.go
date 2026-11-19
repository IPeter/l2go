package gameserver

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
)

// testClock is a clock tests move by hand, for Server.now.
type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func newTestClock() *testClock {
	return &testClock{now: time.Date(2026, time.November, 19, 12, 0, 0, 0, time.UTC)}
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) Add(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// clockEnv is an environment whose server reads the time from clock.
func clockEnv(t *testing.T, clock *testClock, configure func(*config.GameServer)) *env {
	t.Helper()
	return newEnv(t, envOptions{
		configure: configure,
		server:    func(s *Server) { s.now = clock.Now },
	})
}

const sevenDays = 7 * 24 * time.Hour

// stored returns the stored character with name of account.
func stored(t *testing.T, e *env, account, name string) (storage.Character, bool) {
	t.Helper()
	list, err := e.chars.ListByAccount(testCtx(t), account)
	if err != nil {
		t.Fatal(err)
	}
	for _, ch := range list {
		if ch.Name == name {
			return ch, true
		}
	}
	return storage.Character{}, false
}

func deleteSlot(t *testing.T, c *testclient.GameConn, slot int32) testclient.DeleteResult {
	t.Helper()
	res, err := c.DeleteCharacter(testCtx(t), slot)
	if err != nil {
		t.Fatalf("DeleteCharacter(%d): %v", slot, err)
	}
	return res
}

func restoreSlot(t *testing.T, c *testclient.GameConn, slot int32) testclient.CharacterList {
	t.Helper()
	list, err := c.RestoreCharacter(testCtx(t), slot)
	if err != nil {
		t.Fatalf("RestoreCharacter(%d): %v", slot, err)
	}
	return list
}

// Delete marks the character for deletion in 7 days (Mobius
// DeleteCharAfterDays): CharDeleteSuccess, then the list with the time
// left. Restore clears it.
func TestCharacterDeleteAndRestore(t *testing.T) {
	clock := newTestClock()
	e := clockEnv(t, clock, nil)
	c, _ := withCharacters(t, e, "player1", "Doomed", "Keeper")

	res := deleteSlot(t, c, 0)
	if !res.Deleted || res.Failed {
		t.Fatalf("delete answer: %+v, want CharDeleteSuccess", res)
	}
	doomed, ok := res.Characters.Find("Doomed")
	if !ok || doomed.DeleteTimer != int32(sevenDays/time.Second) {
		t.Errorf("Doomed listed %v with delete timer %d, want %d", ok, doomed.DeleteTimer, int32(sevenDays/time.Second))
	}
	if keeper, _ := res.Characters.Find("Keeper"); keeper.DeleteTimer != 0 {
		t.Errorf("Keeper delete timer %d, want 0", keeper.DeleteTimer)
	}
	ch, _ := stored(t, e, "player1", "Doomed")
	if want := clock.Now().Add(sevenDays); !ch.DeleteTime.Equal(want) {
		t.Errorf("stored delete time %v, want %v", ch.DeleteTime, want)
	}

	// The timer counts down.
	clock.Add(24 * time.Hour)
	list := restoreSlot(t, c, 5) // an empty slot: only the list
	if d, _ := list.Find("Doomed"); d.DeleteTimer != int32(6*24*time.Hour/time.Second) {
		t.Errorf("a day later: delete timer %d, want 6 days", d.DeleteTimer)
	}

	list = restoreSlot(t, c, 0)
	if d, ok := list.Find("Doomed"); !ok || d.DeleteTimer != 0 {
		t.Errorf("after restore: listed %v, delete timer %d, want 0", ok, d.DeleteTimer)
	}
	if ch, _ := stored(t, e, "player1", "Doomed"); ch.Deleting() {
		t.Errorf("stored delete time %v after restore, want none", ch.DeleteTime)
	}
}

// A character whose delete time has passed is deleted when the list is
// built, as in Mobius; there is no background job.
func TestCharacterPurgedWhenListed(t *testing.T) {
	clock := newTestClock()
	e := clockEnv(t, clock, nil)
	c, _ := withCharacters(t, e, "player1", "Doomed", "Keeper")
	deleteSlot(t, c, 0)

	// At the delete time itself it is still listed (Mobius: now > time).
	clock.Add(sevenDays)
	if list := restoreSlot(t, c, 9); len(list.Characters) != 2 {
		t.Fatalf("at the delete time: %d characters listed, want 2", len(list.Characters))
	}
	if _, ok := stored(t, e, "player1", "Doomed"); !ok {
		t.Fatal("deleted at the delete time; Mobius waits until it has passed")
	}

	clock.Add(time.Millisecond)
	list := restoreSlot(t, c, 9)
	if _, ok := list.Find("Doomed"); ok || len(list.Characters) != 1 {
		t.Errorf("after the delete time: list %+v, want only Keeper", list.Characters)
	}
	if _, ok := stored(t, e, "player1", "Doomed"); ok {
		t.Error("the character is still stored")
	}
	if taken, _ := e.chars.NameExists(testCtx(t), "Doomed"); taken {
		t.Error("the name is still taken")
	}
	// The slots follow the new list: slot 1 is empty now.
	if res := deleteSlot(t, c, 1); res.Deleted || res.Failed {
		t.Errorf("deleting the old slot 1: %+v, want only the list", res)
	}
	if ch, _ := stored(t, e, "player1", "Keeper"); ch.Deleting() {
		t.Error("Keeper was marked for deletion through a stale slot")
	}
}

// Expired characters are purged from the very first list too.
func TestCharacterPurgedAtLogin(t *testing.T) {
	e := newEnv(t, envOptions{})
	ctx := testCtx(t)
	for _, ch := range []storage.Character{
		{ObjectID: 1, Account: "player1", Name: "Expired", DeleteTime: time.Now().Add(-time.Minute)},
		{ObjectID: 2, Account: "player1", Name: "Pending", DeleteTime: time.Now().Add(time.Hour)},
		{ObjectID: 3, Account: "player1", Name: "Normal"},
	} {
		if err := e.chars.Create(ctx, &ch); err != nil {
			t.Fatal(err)
		}
	}
	_, list := e.login(t, "player1")
	if _, ok := list.Find("Expired"); ok || len(list.Characters) != 2 {
		t.Errorf("list %+v, want Pending and Normal", list.Characters)
	}
	if p, _ := list.Find("Pending"); p.DeleteTimer <= 0 || p.DeleteTimer > 3600 {
		t.Errorf("Pending delete timer %d, want about an hour", p.DeleteTimer)
	}
	if _, err := e.chars.Get(ctx, 1); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Get(expired) = %v, want ErrNotFound", err)
	}
}

// With deleteCharAfterDays 0 the character is deleted at once, as the
// Mobius code does, and its name is free again.
func TestCharacterDeleteAtOnce(t *testing.T) {
	e := newEnv(t, envOptions{configure: func(gs *config.GameServer) { gs.World.DeleteCharAfterDays = 0 }})
	c, _ := withCharacters(t, e, "player1", "Gone", "Stays")
	res := deleteSlot(t, c, 0)
	if !res.Deleted {
		t.Fatalf("delete answer: %+v, want CharDeleteSuccess", res)
	}
	if _, ok := res.Characters.Find("Gone"); ok || len(res.Characters.Characters) != 1 {
		t.Errorf("list %+v, want only Stays", res.Characters.Characters)
	}
	if _, ok := stored(t, e, "player1", "Gone"); ok {
		t.Error("the character is still stored")
	}
	wantCreated(t, c, humanFighter("Gone"))
}

// An empty slot gets only the list, as in Mobius.
func TestCharacterDeleteEmptySlot(t *testing.T) {
	e := newEnv(t, envOptions{})
	c, _ := withCharacters(t, e, "player1", "Lonely")
	for _, slot := range []int32{1, -1, 1 << 30} {
		res := deleteSlot(t, c, slot)
		if res.Deleted || res.Failed || len(res.Characters.Characters) != 1 {
			t.Errorf("slot %d: %+v, want only the list", slot, res)
		}
	}
	if ch, _ := stored(t, e, "player1", "Lonely"); ch.Deleting() {
		t.Error("a character was marked for deletion")
	}
}

// failingDeletes is a store whose deletes fail.
type failingDeletes struct{ storage.CharacterStore }

var errDiskFull = errors.New("disk full")

func (failingDeletes) Delete(context.Context, int32) error { return errDiskFull }

func (failingDeletes) SetDeleteTime(context.Context, int32, time.Time) error { return errDiskFull }

// A storage error gets only the list, as Mobius does when
// markToDeleteChar fails.
func TestCharacterDeleteStoreError(t *testing.T) {
	e := newEnv(t, envOptions{wrapStore: func(s storage.CharacterStore) storage.CharacterStore {
		return failingDeletes{s}
	}})
	c, _ := withCharacters(t, e, "player1", "Sturdy")
	if res := deleteSlot(t, c, 0); res.Deleted || res.Failed || len(res.Characters.Characters) != 1 {
		t.Errorf("delete answer %+v, want only the list", res)
	}
	if list := restoreSlot(t, c, 0); len(list.Characters) != 1 {
		t.Errorf("restore answer %+v, want the list", list)
	}
}

func TestCharacterDeleteRestoreMalformed(t *testing.T) {
	for _, op := range []byte{clientpackets.OpCharacterDelete, clientpackets.OpCharacterRestore} {
		e := newEnv(t, envOptions{})
		c, _ := withCharacters(t, e, "player1", "Victim")
		sendAll(t, c, []rawPacket{{op, []byte{0, 0}}})
		if err := c.ExpectClosed(testCtx(t)); err != nil {
			t.Errorf("0x%02x: %v", op, err)
		}
		if ch, _ := stored(t, e, "player1", "Victim"); ch.Deleting() {
			t.Errorf("0x%02x: the character was marked for deletion", op)
		}
	}
}

// After a restart the list is built again, purge included.
func TestRestartListPurges(t *testing.T) {
	clock := newTestClock()
	e := clockEnv(t, clock, nil)
	c, _ := withCharacters(t, e, "player1", "Player", "Doomed")
	deleteSlot(t, c, 1)
	enter(t, e, c)
	clock.Add(sevenDays + time.Second)
	list, err := c.Restart(testCtx(t))
	if err != nil {
		t.Fatalf("Restart: %v", err)
	}
	if _, ok := list.Find("Doomed"); ok || len(list.Characters) != 1 {
		t.Errorf("list after restart %+v, want only Player", list.Characters)
	}
}
