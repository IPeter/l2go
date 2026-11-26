package game

import (
	"bytes"
	"context"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The crest and clan skill tests run newClanEnv's server with the crest
// store, the skill data and the village master's skill pages.

// clanExtraPages are the pages of the clan skills, beside clanTestPages.
var clanExtraPages = map[string]string{
	"villagemaster/NotClanLeader.htm": "<html><body>NOT CLAN LEADER</body></html>",
	"villagemaster/NoMoreSkills.htm":  "<html><body>NO MORE SKILLS</body></html>",
}

// newClanExtrasEnv is newClanEnv with the crest store and the skill data.
// With prev, the server uses prev's store and object IDs: a restart.
func newClanExtrasEnv(t *testing.T, prev *testEnv, start time.Time, configure func(gs *config.GameServer)) *testEnv {
	t.Helper()
	d := testGameData(t)
	w := testWorldData(t)
	dataDir := t.TempDir()
	pages := maps.Clone(clanTestPages)
	maps.Copy(pages, clanExtraPages)
	for name, content := range pages {
		writeDialogFile(t, filepath.Join(dataDir, "html", filepath.FromSlash(name)), content)
	}
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval = 0
	gs.World.SpawnProtection = 0
	gs.DataDir = dataDir
	if configure != nil {
		configure(gs)
	}
	store, ids := newRecordingStore(), memstore.NewIDs()
	if prev != nil {
		store, ids = prev.store, prev.ids
	}
	if start.IsZero() {
		start = testStart
	}
	clock := world.NewFakeClock(start)
	srv, err := New(Options{
		Config:         cfg,
		ServerID:       gs.ID,
		Data:           d,
		World:          w,
		Characters:     store,
		Clans:          store.Clans(),
		Crests:         store.Crests(),
		IDs:            ids,
		Passives:       testPassives(t),
		Categories:     testCategories(t),
		Skills:         testSkills(t),
		Clock:          clock,
		CheckLoopOwner: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
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
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(func() {
		stop()
		cancel()
	})
	return &testEnv{t: t, ctx: ctx, srv: srv, store: store, ids: ids, clock: clock, data: d, stop: stop}
}

// crestBody encodes a crest upload: d length, then the image.
func crestBody(length int32, data []byte) []byte {
	w := packets.NewWriter(4 + len(data))
	w.WriteInt32(length)
	w.WriteBytes(data)
	return w.Bytes()
}

// crestImage is an image of n bytes made of b.
func crestImage(n int, b byte) []byte { return bytes.Repeat([]byte{b}, n) }

// uploadCrest sends a clan crest upload (large for the large crest), waits
// until the crest ID is reserved and applied, and returns what the session
// got.
func (e *testEnv) uploadCrest(sess *fakeSession, large bool, body []byte) []testclient.Packet {
	e.t.Helper()
	sess.take(e.t)
	if large {
		e.srv.HandleEx(sess, clientpackets.ExOpRequestExSetPledgeCrestLarge, body)
	} else {
		e.srv.Handle(sess, clientpackets.OpRequestSetPledgeCrest, body)
	}
	e.sync()
	e.srv.crests.pending.Wait()
	e.sync()
	return sess.take(e.t)
}

// clanCrestIDs returns the clan's crest IDs, read on the loop.
func (e *testEnv) clanCrestIDs(c *Clan) (crest, large int32) {
	e.onLoop(func() { crest, large = c.CrestID, c.CrestLargeID })
	return crest, large
}

// storedCrestIDs returns the IDs of the stored crests.
func (e *testEnv) storedCrestIDs() []int32 {
	e.t.Helper()
	list, err := e.store.Crests().LoadCrests(e.ctx)
	if err != nil {
		e.t.Fatalf("LoadCrests: %v", err)
	}
	var ids []int32
	for _, cr := range list {
		ids = append(ids, cr.ID)
	}
	return ids
}

// The clan crest: the refusals (size, clan level, privilege), an upload
// shown on the members' UserInfo and CharInfo, a new ID for a new image
// with the old crest deleted, the requests in game, and deletion.
func TestClanCrestUpload(t *testing.T) {
	e := newClanExtrasEnv(t, nil, time.Time{}, nil)
	sessA, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	sessB, b := e.enterClanChar(clanChar{account: "bob", name: "Bob", level: 20})
	sessV, _ := e.enterClanChar(clanChar{account: "view", name: "Viewer", level: 20})
	c := e.foundClan(a, "Dragons")
	e.inviteToClan(sessA, a, sessB, b, 0, 1)
	img := crestImage(clientpackets.MaxPledgeCrestSize, 7)

	clanWantMsg(t, "too large", e.uploadCrest(sessA, false, crestBody(clientpackets.MaxPledgeCrestSize+1, nil)), int32(crestMsgImageSize))
	clanWantMsg(t, "clan level 0", e.uploadCrest(sessA, false, crestBody(int32(len(img)), img)), int32(crestMsgNeedsClanLevel3))
	e.onLoop(func() { e.srv.ChangeClanLevel(c, 3) })
	clanWantMsg(t, "no privilege", e.uploadCrest(sessB, false, crestBody(int32(len(img)), img)), int32(clanMsgNotAuthorized))
	if id, _ := e.clanCrestIDs(c); id != 0 {
		t.Fatalf("crest %d after the refusals", id)
	}
	if ps := e.uploadCrest(sessV, false, crestBody(int32(len(img)), img)); len(ps) != 0 {
		t.Errorf("an upload without a clan got %q", testclient.Names(ps))
	}

	ps := e.uploadCrest(sessA, false, crestBody(int32(len(img)), img))
	if msgs := messages(ps); !slices.Equal(msgs, []string{crestMsgRegisteredText}) {
		t.Errorf("upload messages %q", msgs)
	}
	first, _ := e.clanCrestIDs(c)
	if first == 0 {
		t.Fatal("no crest after the upload")
	}
	sessB.take(t)
	sessV.take(t)
	e.flushInfo()
	for _, sess := range []*fakeSession{sessA, sessB} {
		if ui := clanUserInfo(t, sess.take(t)); ui.ClanCrestID != first {
			t.Errorf("%s: UserInfo crest %d, want %d", sess.account, ui.ClanCrestID, first)
		}
	}
	seen := map[int32]int32{}
	for _, ci := range only[testclient.CharInfoPacket](sessV.take(t)) {
		seen[ci.ObjectID] = ci.ClanCrestID
	}
	if seen[a.ObjectID()] != first || seen[b.ObjectID()] != first {
		t.Errorf("the viewer's CharInfo crests %v, want %d for both members", seen, first)
	}
	crests := only[testclient.PledgeCrest](e.clanSend(sessV, clientpackets.OpRequestPledgeCrest, clanInts(first)))
	if len(crests) != 1 || crests[0].CrestID != first || !bytes.Equal(crests[0].Data, img) {
		t.Errorf("PledgeCrest %+v", crests)
	}
	if doc, err := e.storedClan(c.ID); err != nil || doc.CrestID != first {
		t.Errorf("stored clan crest %d, %v", doc.CrestID, err)
	}
	if ids := e.storedCrestIDs(); !slices.Equal(ids, []int32{first}) {
		t.Errorf("stored crests %v", ids)
	}

	// A new image gets a new ID; the old crest is gone.
	img2 := crestImage(64, 9)
	e.uploadCrest(sessA, false, crestBody(int32(len(img2)), img2))
	second, _ := e.clanCrestIDs(c)
	if second == 0 || second == first {
		t.Fatalf("second crest %d after %d", second, first)
	}
	crests = only[testclient.PledgeCrest](e.clanSend(sessV, clientpackets.OpRequestPledgeCrest, clanInts(first)))
	if len(crests) != 1 || crests[0].CrestID != first || len(crests[0].Data) != 0 {
		t.Errorf("PledgeCrest of the old crest %+v", crests)
	}
	e.flushClanWrites(c.ID)
	if ids := e.storedCrestIDs(); !slices.Equal(ids, []int32{second}) {
		t.Errorf("stored crests %v, want [%d]", ids, second)
	}

	// Deleting: an empty image.
	clanWantMsg(t, "deleted", e.uploadCrest(sessA, false, crestBody(0, nil)), int32(crestMsgDeleted))
	if id, _ := e.clanCrestIDs(c); id != 0 {
		t.Errorf("crest %d after the delete", id)
	}
	if ps := e.uploadCrest(sessA, false, crestBody(0, nil)); clanHasMsg(ps, int32(crestMsgDeleted)) {
		t.Error("deleting a missing crest said it was deleted")
	}
	e.flushClanWrites(c.ID)
	if ids := e.storedCrestIDs(); len(ids) != 0 {
		t.Errorf("stored crests %v after the delete", ids)
	}
}

// The large crest: its own limit (a larger image is ignored), the clan
// level, the message, ExPledgeEmblem, deletion; and the dissolution
// refusal for both crests.
func TestClanCrestLargeAndDissolution(t *testing.T) {
	e := newClanExtrasEnv(t, nil, time.Time{}, nil)
	sessA, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	c := e.foundClan(a, "Dragons")
	img := crestImage(clientpackets.MaxPledgeCrestLargeSize, 3)

	if ps := e.uploadCrest(sessA, true, crestBody(clientpackets.MaxPledgeCrestLargeSize+1, nil)); len(ps) != 0 {
		t.Errorf("a too large image got %q", testclient.Names(ps))
	}
	clanWantMsg(t, "clan level 0", e.uploadCrest(sessA, true, crestBody(int32(len(img)), img)), int32(crestMsgNeedsClanLevel3))
	e.onLoop(func() { e.srv.ChangeClanLevel(c, 3) })
	clanWantMsg(t, "registered", e.uploadCrest(sessA, true, crestBody(int32(len(img)), img)), int32(crestMsgLargeRegistered))
	_, large := e.clanCrestIDs(c)
	if large == 0 {
		t.Fatal("no large crest")
	}
	emblems := only[testclient.ExPledgeEmblem](e.clanSendEx(sessA, clientpackets.ExOpRequestExPledgeCrestLarge, clanInts(large)))
	if len(emblems) != 1 || emblems[0].CrestID != large || !bytes.Equal(emblems[0].Data, img) {
		t.Errorf("ExPledgeEmblem %+v", emblems)
	}
	if doc, err := e.storedClan(c.ID); err != nil || doc.CrestLargeID != large {
		t.Errorf("stored large crest %d, %v", doc.CrestLargeID, err)
	}

	// During a dissolution neither crest changes.
	e.onLoop(func() { c.DissolvingExpiry = e.srv.Now().Add(time.Hour).UnixMilli() })
	clanWantMsg(t, "dissolving", e.uploadCrest(sessA, false, crestBody(4, crestImage(4, 1))), int32(crestMsgDissolving))
	clanWantMsg(t, "dissolving (large)", e.uploadCrest(sessA, true, crestBody(0, nil)), int32(crestMsgDissolving))
	if crest, l := e.clanCrestIDs(c); crest != 0 || l != large {
		t.Errorf("crests %d/%d during the dissolution", crest, l)
	}
	e.onLoop(func() { c.DissolvingExpiry = 0 })
	clanWantMsg(t, "deleted", e.uploadCrest(sessA, true, crestBody(0, nil)), int32(crestMsgDeleted))
	if _, l := e.clanCrestIDs(c); l != 0 {
		t.Errorf("large crest %d after the delete", l)
	}
}

// RequestPledgeCrest is answered only in the game: not before the
// character is selected, nor while it enters; then with the image.
func TestClanCrestRequestBeforeGame(t *testing.T) {
	e := newClanExtrasEnv(t, nil, time.Time{}, nil)
	sessA, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	c := e.foundClan(a, "Dragons")
	e.onLoop(func() { e.srv.ChangeClanLevel(c, 3) })
	img := crestImage(32, 5)
	e.uploadCrest(sessA, false, crestBody(int32(len(img)), img))
	id, _ := e.clanCrestIDs(c)

	e.createClanChar(clanChar{account: "view", name: "Viewer", level: 20})
	sess := newSession("view")
	request := func() []testclient.PledgeCrest {
		t.Helper()
		sess.take(t)
		e.srv.Handle(sess, clientpackets.OpRequestPledgeCrest, clanInts(id))
		e.sync()
		return only[testclient.PledgeCrest](sess.take(t))
	}
	if got := request(); len(got) != 0 {
		t.Errorf("answered on the character screen: %+v", got)
	}
	if err := e.srv.SelectCharacter(e.ctx, sess, 0); err != nil {
		t.Fatalf("SelectCharacter: %v", err)
	}
	if got := request(); len(got) != 0 {
		t.Errorf("answered while entering: %+v", got)
	}
	e.srv.EnterWorld(sess)
	e.sync()
	if got := request(); len(got) != 1 || got[0].CrestID != id || !bytes.Equal(got[0].Data, img) {
		t.Errorf("in game: %+v", got)
	}
}

// After a restart the crests are back; crests no clan uses are deleted and
// crest IDs whose crest is gone are cleared (Mobius CrestTable.load).
func TestClanCrestRestart(t *testing.T) {
	e := newClanExtrasEnv(t, nil, time.Time{}, nil)
	sessA, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	c := e.foundClan(a, "Dragons")
	e.onLoop(func() { e.srv.ChangeClanLevel(c, 3) })
	img := crestImage(48, 4)
	e.uploadCrest(sessA, false, crestBody(int32(len(img)), img))
	id, _ := e.clanCrestIDs(c)
	other := e.foundClanChar(t, "Wolves")
	e.onLoop(func() {
		other.CrestLargeID = 999 // a crest that does not exist
		e.srv.saveClan(other)
	})
	e.clanLogout(sessA, a)
	e.flushClanWrites(c.ID, other.ID)
	orphan := storage.Crest{ID: 500, Type: storage.CrestAlly, Data: []byte{1}}
	if err := e.store.WriteBatch(e.ctx, storage.Batch{Crests: []storage.Crest{orphan}}); err != nil {
		t.Fatalf("WriteBatch: %v", err)
	}
	e.stop()

	e2 := newClanExtrasEnv(t, e, testStart.Add(time.Hour), nil)
	var crestID, largeID int32
	var data []byte
	e2.onLoop(func() {
		crestID = e2.srv.Clan(c.ID).CrestID
		largeID = e2.srv.Clan(other.ID).CrestLargeID
		data = e2.srv.crestData(id)
	})
	if crestID != id || !bytes.Equal(data, img) {
		t.Errorf("restored crest %d with %d bytes, want %d", crestID, len(data), id)
	}
	if largeID != 0 {
		t.Errorf("missing large crest %d not cleared", largeID)
	}
	e2.flushClanWrites(other.ID)
	if ids := e2.storedCrestIDs(); !slices.Equal(ids, []int32{id}) {
		t.Errorf("stored crests after the restart %v, want [%d]", ids, id)
	}
	if doc, err := e2.storedClan(other.ID); err != nil || doc.CrestLargeID != 0 {
		t.Errorf("stored Wolves large crest %d, %v", doc.CrestLargeID, err)
	}
}

// foundClanChar makes a new character found a clan named name.
func (e *testEnv) foundClanChar(t *testing.T, name string) *Clan {
	t.Helper()
	_, p := e.enterClanChar(clanChar{account: "f" + strings.ToLower(name), name: "F" + name, level: 40})
	return e.foundClan(p, name)
}

// The alliance crest helper: newCrest and applyAllyCrest give every clan of
// the alliance the crest, write it and delete the old one; onlyThisClan
// changes one clan.
func TestClanAllyCrestHelper(t *testing.T) {
	e := newClanExtrasEnv(t, nil, time.Time{}, nil)
	lead := e.foundClanChar(t, "Dragons")
	member := e.foundClanChar(t, "Wolves")
	e.onLoop(func() {
		for _, c := range []*Clan{lead, member} {
			c.AllyID, c.AllyName = lead.ID, "Kingdom"
			e.srv.saveClan(c)
		}
	})
	set := func(data []byte) {
		e.onLoop(func() {
			e.srv.newCrest(storage.CrestAlly, data, func(id int32, b *storage.Batch) {
				e.srv.applyAllyCrest(lead, id, false, *b)
			})
		})
		e.srv.crests.pending.Wait()
		e.sync()
	}
	set([]byte{1, 2, 3})
	var first, got int32
	e.onLoop(func() { first, got = lead.AllyCrestID, member.AllyCrestID })
	if first == 0 || got != first {
		t.Fatalf("alliance crests %d/%d", first, got)
	}
	set([]byte{4, 5})
	var second int32
	e.onLoop(func() { second, got = lead.AllyCrestID, member.AllyCrestID })
	if second == first || got != second {
		t.Errorf("alliance crests %d/%d after %d", second, got, first)
	}
	if doc, err := e.storedClan(member.ID); err != nil || doc.AllyCrestID != second {
		t.Errorf("stored member alliance crest %d, %v", doc.AllyCrestID, err)
	}
	if ids := e.storedCrestIDs(); !slices.Equal(ids, []int32{second}) {
		t.Errorf("stored crests %v, want [%d]", ids, second)
	}
	e.onLoop(func() { e.srv.applyAllyCrest(member, 0, true, storage.Batch{}) })
	e.onLoop(func() { first, got = lead.AllyCrestID, member.AllyCrestID })
	if first != second || got != 0 {
		t.Errorf("after onlyThisClan: %d/%d", first, got)
	}
}
