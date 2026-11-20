package game

import (
	"context"
	"path/filepath"
	"slices"
	"strconv"
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

// The clan tests run a server with the world data (for a village master),
// the in-memory clan store and fixture pages.

// npcBitz is a village master (VillageMasterFighter) of Talking Island.
const npcBitz = 30026

// clanTestPages are the fixture pages the clan code reads.
var clanTestPages = map[string]string{
	"scripts/village_master/ClanMaster/9000-07-success.htm":     "<html><body>LEADER SUCCESS</body></html>",
	"scripts/village_master/ClanMaster/9000-07-in-progress.htm": "<html><body>LEADER IN PROGRESS</body></html>",
	"scripts/village_master/ClanMaster/9000-07-canceled.htm":    "<html><body>LEADER CANCELED</body></html>",
	"clanNotice.htm":        "<html><body>%clan_name%: %notice_text%</body></html>",
	"admin/claninfo.htm":    "<html><body>%clan_name% %clan_leader% %clan_level% %clan_points% %clan_players_count% %clan_ally% %current_player_name%</body></html>",
	"admin/clanchanges.htm": "<html><body><table>%data%</table></body></html>",
	"admin/game_menu.htm":   "<html><body>GAME MENU</body></html>",
}

// clanVMPage offers the village master's clan commands.
const clanVMPage = `<html><body>
<a action="bypass -h npc_%objectId%_create_clan $name">Create</a>
<a action="bypass -h npc_%objectId%_increase_clan_level">Level</a>
<a action="bypass -h npc_%objectId%_dissolve_clan">Dissolve</a>
<a action="bypass -h npc_%objectId%_recover_clan">Recover</a>
<a action="bypass -h npc_%objectId%_change_clan_leader $name">Leader</a>
<a action="bypass -h npc_%objectId%_cancel_clan_leader_change">Cancel</a>
</body></html>`

// newClanEnv starts a server with world data, the fixture pages and the
// clan store, on a fake clock at start (zero: testStart). With prev, the
// server uses prev's store and object IDs: a restart.
func newClanEnv(t *testing.T, prev *testEnv, start time.Time, configure func(gs *config.GameServer)) *testEnv {
	t.Helper()
	d := testGameData(t)
	w := testWorldData(t)
	dataDir := t.TempDir()
	for name, content := range clanTestPages {
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
		IDs:            ids,
		Passives:       testPassives(t),
		Categories:     testCategories(t),
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

// clanChar describes a character for enterClanChar.
type clanChar struct {
	account, name string
	level         int32
	sp            int32
	adena         int64
	class         int
	access        storage.AccessLevel
	items         []storage.Item
}

// createClanChar stores a character as c describes, at the village.
func (e *testEnv) createClanChar(c clanChar) storage.Character {
	e.t.Helper()
	ch := e.createCharacter(c.account, c.name, c.class, c.access)
	ch.X, ch.Y, ch.Z = tiVillage.X, tiVillage.Y, tiVillage.Z
	if c.level > 1 {
		ch.Level = c.level
		ch.Exp = e.srv.ExpForLevel(int(c.level))
	}
	ch.SP = c.sp
	extra := slices.Clone(c.items)
	if c.adena > 0 {
		extra = append(extra, storage.Item{ItemID: adenaID, Count: c.adena, Slot: storage.PaperdollNone, Mana: storage.NoMana})
	}
	if len(extra) > 0 {
		ids, err := e.ids.NextIDs(e.ctx, len(extra))
		if err != nil {
			e.t.Fatalf("NextIDs: %v", err)
		}
		for i := range extra {
			extra[i].ObjectID = ids[i]
		}
		ch.Items = append(ch.Items, extra...)
	}
	if err := e.store.Update(e.ctx, &ch); err != nil {
		e.t.Fatalf("Update: %v", err)
	}
	return ch
}

// enterClanChar creates a character as c describes and enters the world
// with it. The packets sent so far are dropped.
func (e *testEnv) enterClanChar(c clanChar) (*fakeSession, *Player) {
	e.t.Helper()
	e.createClanChar(c)
	return e.clanLogin(c.account)
}

// clanLogin enters the world with the account's character; the packets of
// entering are dropped.
func (e *testEnv) clanLogin(account string) (*fakeSession, *Player) {
	e.t.Helper()
	sess := newSession(account)
	p := e.enter(sess, 0)
	e.sync()
	sess.take(e.t)
	return sess, p
}

// clanLogout logs the session's player out and waits for its save.
func (e *testEnv) clanLogout(sess *fakeSession, p *Player) {
	e.t.Helper()
	id := p.ObjectID()
	e.srv.Logout(sess)
	e.flushClanWrites(id)
}

// flushClanWrites hands the pending saves to the saver and waits until the
// writes touching ids are done.
func (e *testEnv) flushClanWrites(ids ...int32) {
	e.t.Helper()
	e.onLoop(e.srv.flushSaveRequests)
	for _, id := range ids {
		if err := e.srv.saver.wait(e.ctx, id); err != nil {
			e.t.Fatalf("wait for the writes of %d: %v", id, err)
		}
	}
}

// storedClan reads a clan from the store after its writes.
func (e *testEnv) storedClan(id int32) (storage.Clan, error) {
	e.t.Helper()
	e.flushClanWrites(id)
	return e.store.Clans().GetClan(e.ctx, id)
}

// storedCharacter reads a character from the store after its writes.
func (e *testEnv) storedCharacter(id int32) storage.Character {
	e.t.Helper()
	e.flushClanWrites(id)
	c, err := e.store.Get(e.ctx, id)
	if err != nil {
		e.t.Fatalf("Get %d: %v", id, err)
	}
	return c
}

// vmCommand runs a village master command for p: the clan page first (for
// the link cache), then the bypass; it returns what p got.
func (e *testEnv) vmCommand(sess *fakeSession, p *Player, n *Npc, command string) []testclient.Packet {
	e.t.Helper()
	e.onLoop(func() { e.srv.sendPage(p, newPage(n.ID(), clanVMPage).replaceObjectID(n.ID())) })
	return e.bypass(sess, "npc_"+strconv.Itoa(int(n.ID()))+"_"+command)
}

// foundClan makes p found a clan named name on the loop.
func (e *testEnv) foundClan(p *Player, name string) *Clan {
	e.t.Helper()
	var c *Clan
	e.onLoop(func() { c = e.srv.CreateClan(p, name) })
	if c == nil {
		e.t.Fatalf("%s could not create the clan %s", p.Name(), name)
	}
	return c
}

// clanSend sends a client packet and returns what the session got.
func (e *testEnv) clanSend(sess *fakeSession, op byte, body []byte) []testclient.Packet {
	e.t.Helper()
	sess.take(e.t)
	e.srv.Handle(sess, op, body)
	e.sync()
	return sess.take(e.t)
}

// clanSendEx sends an extended client packet and returns what the session
// got.
func (e *testEnv) clanSendEx(sess *fakeSession, sub uint16, body []byte) []testclient.Packet {
	e.t.Helper()
	sess.take(e.t)
	e.srv.HandleEx(sess, sub, body)
	e.sync()
	return sess.take(e.t)
}

// inviteToClan makes from invite to into unit and answers with answer; it
// returns what the invited player got.
func (e *testEnv) inviteToClan(fromSess *fakeSession, from *Player, toSess *fakeSession, to *Player, unit, answer int32) []testclient.Packet {
	e.t.Helper()
	toSess.take(e.t)
	ps := e.clanSend(fromSess, clientpackets.OpRequestJoinPledge, clanInts(to.ObjectID(), unit))
	if len(ps) != 0 {
		e.t.Fatalf("the inviter got %q %v", testclient.Names(ps), sysMsgs(ps))
	}
	if asks := only[testclient.AskJoinPledge](toSess.take(e.t)); len(asks) != 1 || asks[0].RequestorID != from.ObjectID() {
		e.t.Fatalf("AskJoinPledge = %+v", asks)
	}
	return e.clanSend(toSess, clientpackets.OpRequestAnswerJoinPledge, clanInts(answer))
}

// clanInts encodes a body of ints.
func clanInts(vs ...int32) []byte {
	w := packets.NewWriter(4 * len(vs))
	for _, v := range vs {
		w.WriteInt32(v)
	}
	return w.Bytes()
}

// clanStrings encodes a body of strings.
func clanStrings(ss ...string) []byte {
	w := packets.NewWriter(64)
	for _, s := range ss {
		w.WriteString(s)
	}
	return w.Bytes()
}

// clanHasMsg reports whether ps has the system message id.
func clanHasMsg(ps []testclient.Packet, id int32) bool {
	return slices.Contains(sysMsgs(ps), id)
}

// clanWantMsg fails unless ps has the system message.
func clanWantMsg(t *testing.T, what string, ps []testclient.Packet, id int32) {
	t.Helper()
	if !slices.Contains(sysMsgs(ps), id) {
		t.Errorf("%s: no system message %d in %v (%q)", what, id, sysMsgs(ps), testclient.Names(ps))
	}
}

// clanUserInfo returns the last UserInfo of ps.
func clanUserInfo(t *testing.T, ps []testclient.Packet) testclient.UserInfo {
	t.Helper()
	uis := only[testclient.UserInfo](ps)
	if len(uis) == 0 {
		t.Fatalf("no UserInfo in %q", testclient.Names(ps))
	}
	return uis[len(uis)-1]
}
