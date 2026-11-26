package game

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/cursedweapon"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The cursed weapons of the data.
const (
	cwZariche  = 8190
	cwAkamanah = 8689
)

// newCursedEnv starts a server with world and skill data on store and
// clock, so that a test can stop it and start another on the same store
// (a restart).
func newCursedEnv(t *testing.T, store *recordingStore, clock *world.FakeClock, ids *memstore.IDs) *testEnv {
	t.Helper()
	d := testGameData(t)
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval = 0
	gs.World.SpawnProtection = 0
	gs.DataDir = filepath.Join("..", "..", "data") // the admin pages
	srv, err := New(Options{
		Config:         cfg,
		ServerID:       gs.ID,
		Data:           d,
		World:          testWorldData(t),
		Skills:         testSkills(t),
		Characters:     store,
		Relations:      store.Relations(),
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(func() {
		stop()
		cancel()
	})
	return &testEnv{t: t, ctx: ctx, srv: srv, store: store, ids: ids, clock: clock, data: d, stop: stop}
}

// forcedRand answers Get(n) with scripted values while the next one is for
// that n, and passes every other draw to its source: a test forces the
// cursed weapon rolls and leaves combat and drops to chance.
type forcedRand struct {
	rnd.Source
	gets [][2]int // {n, value}
}

func (f *forcedRand) Get(n int) int {
	if len(f.gets) > 0 && f.gets[0][0] == n {
		v := f.gets[0][1]
		f.gets = f.gets[1:]
		return v
	}
	return f.Source.Get(n)
}

// force makes the server's next Get draws of the given bounds answer the
// values, in order.
func (e *testEnv) force(gets ...[2]int) *forcedRand {
	e.t.Helper()
	var f *forcedRand
	e.onLoop(func() {
		if fr, ok := e.srv.rnd.(*forcedRand); ok {
			fr.gets = append(fr.gets, gets...)
			f = fr
			return
		}
		f = &forcedRand{Source: e.srv.rnd, gets: gets}
		e.srv.rnd = f
	})
	return f
}

// left returns the forced draws not made yet, read on the loop.
func (e *testEnv) left(f *forcedRand) int {
	var n int
	e.onLoop(func() { n = len(f.gets) })
	return n
}

// cwNpc returns the first spawnable NPC whose type is typ and none of not.
func cwNpc(t *testing.T, w *data.World, typ string, not ...string) int {
	t.Helper()
	for i := range w.Npcs {
		n := &w.Npcs[i]
		if !w.NpcTypeIs(n.Type, typ) || !w.Spawnable(n.ID) {
			continue
		}
		if slices.ContainsFunc(not, func(x string) bool { return w.NpcTypeIs(n.Type, x) }) {
			continue
		}
		return n.ID
	}
	t.Fatalf("no NPC of type %s", typ)
	return 0
}

// cwWeapon returns the state of a weapon, read on the loop.
func (e *testEnv) cwWeapon(itemID int32) cursedweapon.Weapon {
	var w cursedweapon.Weapon
	e.onLoop(func() { w = *e.srv.cursed.set.Get(itemID) })
	return w
}

// cwRows returns the stored cursed weapon rows once every save is written.
func (e *testEnv) cwRows() []storage.CursedWeapon {
	e.t.Helper()
	e.sync()
	if err := e.srv.saver.cwWaitIdle(e.ctx); err != nil {
		e.t.Fatalf("wait for the saves: %v", err)
	}
	rows, err := e.store.LoadCursedWeapons(e.ctx)
	if err != nil {
		e.t.Fatalf("LoadCursedWeapons: %v", err)
	}
	return rows
}

// cwWaitIdle returns once the saver has nothing waiting or being written.
func (s *saver) cwWaitIdle(ctx context.Context) error {
	for {
		s.mu.Lock()
		idle := len(s.queue) == 0 && s.inflight == nil
		s.mu.Unlock()
		if idle {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond):
		}
	}
}

// cwKarma returns a player's karma, PKs and weapon level, read on the loop.
func (e *testEnv) cwKarma(p *Player) (karma, pks, level int32) {
	e.onLoop(func() { karma, pks, level = p.char.Karma, p.char.PKKills, p.cursedWeaponLevel() })
	return
}

// cwMessages returns the IDs of the system messages among ps.
func cwMessages(ps []testclient.Packet) []int32 {
	var ids []int32
	for _, m := range only[testclient.SystemMessage](ps) {
		ids = append(ids, m.ID)
	}
	return ids
}

// cwDropFor makes a monster next to p drop the weapon of the first weapon
// roll (Akamanah rolls first, in Mobius's map order) and returns the
// ground item.
func (e *testEnv) cwDropFor(p *Player, first bool) *GroundItem {
	e.t.Helper()
	w := testWorldData(e.t)
	pos := e.position(p)
	n := e.spawnNpc(cwNpc(e.t, w, "Monster", cursedweapon.ExcludedNpcTypes...), world.Position{X: pos.X + 50, Y: pos.Y, Z: pos.Z})
	if first {
		e.force([2]int{cursedweapon.DropBound, 0})
	} else {
		e.force([2]int{cursedweapon.DropBound, 1}, [2]int{cursedweapon.DropBound, 0})
	}
	var g *GroundItem
	e.onLoop(func() {
		e.srv.cursedCheckDrop(n, p)
		id := int32(cwAkamanah)
		if !first {
			id = cwZariche
		}
		g = e.srv.cursed.rt[id].ground
		e.srv.DespawnNpc(n) // it would fight the player as the clock runs
	})
	if g == nil {
		e.t.Fatal("no weapon on the ground")
	}
	return g
}

// The drop roll: Rnd.get(100000) < 1 per weapon not in the world, in map
// order; excluded NPCs roll nothing.
func TestCursedWeaponDrop(t *testing.T) {
	e := newCursedEnv(t, newRecordingStore(), world.NewFakeClock(testStart), memstore.NewIDs())
	sess, p := e.enterChar(testChar{account: "hunter", name: "Hunter"})
	w := testWorldData(t)
	pos := e.position(p)
	at := world.Position{X: pos.X + 50, Y: pos.Y, Z: pos.Z}

	// Guards and grand bosses never roll.
	for _, typ := range []string{"Guard", "GrandBoss"} {
		n := e.spawnNpc(cwNpc(t, w, typ), at)
		f := e.force([2]int{cursedweapon.DropBound, 0})
		e.onLoop(func() { e.srv.cursedCheckDrop(n, p) })
		if e.left(f) != 1 {
			t.Errorf("a %s rolled for a cursed weapon", typ)
		}
		e.onLoop(func() { f.gets = nil })
	}

	// A plain monster rolls for both weapons; 1 is no drop.
	n := e.spawnNpc(cwNpc(t, w, "Monster", cursedweapon.ExcludedNpcTypes...), at)
	f := e.force([2]int{cursedweapon.DropBound, 1}, [2]int{cursedweapon.DropBound, 1})
	e.onLoop(func() { e.srv.cursedCheckDrop(n, p) })
	if e.left(f) != 0 {
		t.Error("a monster did not roll for both weapons")
	}
	if zw, aw := e.cwWeapon(cwZariche), e.cwWeapon(cwAkamanah); zw.Active() || aw.Active() {
		t.Fatal("a roll of 1 dropped a weapon")
	}

	// The second roll (Zariche) drops: it falls by the corpse, the sky turns
	// red, the earth shakes, everyone is told, the life starts.
	sess.take(t)
	g := e.cwDropFor(p, false)
	ps := sess.take(t)
	if len(only[testclient.ExRedSky](ps)) != 1 || len(only[testclient.Earthquake](ps)) != 1 {
		t.Errorf("no red sky and earthquake: %v", testclient.Names(ps))
	}
	if !slices.Contains(cwMessages(ps), int32(cwMsgDroppedIn)) {
		t.Errorf("no drop announcement: %v", cwMessages(ps))
	}
	zw := e.cwWeapon(cwZariche)
	if !zw.Dropped || zw.Activated || zw.EndTime != testStart.Add(300*time.Minute).UnixMilli() {
		t.Errorf("Zariche after the drop: %+v", zw)
	}
	var timer world.TimerID
	e.onLoop(func() { timer = g.timer })
	if timer != 0 || !e.inWorldObj(g) {
		t.Error("the weapon on the ground has an auto-destroy timer or is not in the world")
	}
	// A weapon in the world rolls no more: only Akamanah rolls.
	f = e.force([2]int{cursedweapon.DropBound, 5}, [2]int{cursedweapon.DropBound, 5})
	e.onLoop(func() { e.srv.cursedCheckDrop(n, p) })
	if e.left(f) != 1 {
		t.Error("the weapon on the ground rolled again")
	}
	// Nothing is stored for a weapon on the ground.
	if rows := e.cwRows(); rows != nil {
		t.Errorf("rows after the drop: %+v", rows)
	}
}

func (e *testEnv) inWorldObj(o *GroundItem) bool {
	var in bool
	e.onLoop(func() { in = o.InWorld() })
	return in
}

// The pickup turns the player into the holder: karma 9,999,999 and no PKs
// kept for later, the skills, the weapon worn, full vitals, the
// announcement, and the row saved with the player.
func TestCursedWeaponPickUp(t *testing.T) {
	e := newCursedEnv(t, newRecordingStore(), world.NewFakeClock(testStart), memstore.NewIDs())
	sess, p := e.enterChar(testChar{account: "hunter", name: "Hunter", karma: 720})
	e.onLoop(func() { p.char.PKKills = 3; p.CurHP = 10 })
	g := e.cwDropFor(p, true)
	sess.take(t)
	e.onLoop(func() { e.srv.doPickUp(p, g) })
	e.advance(time.Second) // the social action and the skill list
	ps := sess.take(t)

	if karma, pks, level := e.cwKarma(p); karma != cursedweapon.HolderKarma || pks != 0 || level != 0 {
		t.Errorf("holder: karma %d, PKs %d, level %d", karma, pks, level)
	}
	w := e.cwWeapon(cwAkamanah)
	if !w.Activated || w.PlayerID != p.ObjectID() || w.PlayerKarma != 720 || w.PlayerPKKills != 3 {
		t.Errorf("weapon after the pickup: %+v", w)
	}
	var equipped, cursed bool
	var hp, maxHP float64
	var skillIDs []int
	e.onLoop(func() {
		rh := p.inv.Paperdoll(storage.PaperdollRHand)
		equipped = rh != nil && rh.ItemID() == cwAkamanah
		cursed, hp, maxHP = p.CursedWeaponEquipped(), p.CurHP, float64(p.MaxHP())
		for _, sk := range p.Skills() {
			skillIDs = append(skillIDs, sk.ID)
		}
	})
	if !equipped || !cursed || hp != maxHP {
		t.Errorf("equipped %v, cursed %v, HP %v of %v", equipped, cursed, hp, maxHP)
	}
	for _, id := range []int{3629, cursedweapon.SkillVoidBurst, cursedweapon.SkillVoidFlow} {
		if !slices.Contains(skillIDs, id) {
			t.Errorf("skill %d missing: %v", id, skillIDs)
		}
	}
	msgs := cwMessages(ps)
	for _, id := range []int32{int32(sysMsgEquippedS1), int32(cwMsgOwnerAppeared)} {
		if !slices.Contains(msgs, id) {
			t.Errorf("message %d missing: %v", id, msgs)
		}
	}
	if slices.Contains(msgs, int32(sysMsgAttentionPickedUp)) {
		t.Error("the pickup of a cursed weapon was announced as a weapon")
	}
	if len(only[testclient.SocialAction](ps)) == 0 {
		t.Error("no social action")
	}
	rows := e.cwRows()
	if len(rows) != 1 || rows[0].ItemID != cwAkamanah || rows[0].CharID != p.ObjectID() || rows[0].PlayerKarma != 720 ||
		rows[0].PlayerPKKills != 3 || rows[0].EndTime != w.EndTime || rows[0].ItemObjectID != g.item.ObjectID() {
		t.Errorf("stored rows %+v", rows)
	}
	c, _ := e.store.Get(e.ctx, p.ObjectID())
	if c.Karma != cursedweapon.HolderKarma {
		t.Errorf("stored karma %d", c.Karma)
	}

	// The Lua scripts see it (player:is_cursed_weapon_equipped()).
	var ref scripting.PlayerRef = p
	if cw, ok := ref.(scripting.PlayerCursedWeapon); !ok || !cursedOnLoop(e, cw) {
		t.Error("the scripts do not see the cursed weapon")
	}

	// The holder cannot unequip it, cast other skills, shout or invite.
	e.srv.Handle(sess, clientpackets.OpRequestUnEquipItem, int32Body(int32(storage.PaperdollRHand)))
	e.sync()
	e.onLoop(func() { equipped = p.inv.Paperdoll(storage.PaperdollRHand) != nil })
	if !equipped {
		t.Error("the holder took the cursed weapon off")
	}
	e.onLoop(func() {
		if cursedSkillAllowed(p, 3) || !cursedSkillAllowed(p, cursedweapon.SkillVoidFlow) {
			t.Error("the holder's skills are not limited to Void Burst and Void Flow")
		}
	})
}

func cursedOnLoop(e *testEnv, cw scripting.PlayerCursedWeapon) bool {
	var v bool
	e.onLoop(func() { v = cw.CursedWeaponEquipped() })
	return v
}

// Kills: the PK count is the kill count, the skill steps up every 10
// kills, each kill takes 3 minutes off, and the row follows.
func TestCursedWeaponKills(t *testing.T) {
	e := newCursedEnv(t, newRecordingStore(), world.NewFakeClock(testStart), memstore.NewIDs())
	_, p := e.enterChar(testChar{account: "holder", name: "Holder", access: storage.AccessLevelAdmin})
	_, v := e.enterChar(testChar{account: "victim", name: "Victim"})
	e.onLoop(func() { e.srv.RunGMCommand(p, "cw_add 8190") })
	w0 := e.cwWeapon(cwZariche)
	for range 10 {
		e.onLoop(func() { e.srv.onKillUpdatePvPKarma(p, v) })
	}
	w := e.cwWeapon(cwZariche)
	if w.NbKills != 10 || w.EndTime != w0.EndTime-30*60*1000 {
		t.Errorf("after 10 kills: %d kills, end %d (was %d)", w.NbKills, w.EndTime, w0.EndTime)
	}
	if karma, pks, level := e.cwKarma(p); karma != cursedweapon.HolderKarma || pks != 10 || level != 1 {
		t.Errorf("holder after 10 kills: karma %d, PKs %d, level %d", karma, pks, level)
	}
	var skill int
	e.onLoop(func() { skill = p.SkillLevel(3603) })
	if skill != 2 {
		t.Errorf("skill 3603 level %d after 10 kills, want 2", skill)
	}
	if rows := e.cwRows(); len(rows) != 1 || rows[0].NbKills != 10 || rows[0].EndTime != w.EndTime {
		t.Errorf("stored rows %+v", rows)
	}
	// The victim of a holder drops nothing and loses no exp; the holder's
	// kill gives no karma.
	e.onLoop(func() {
		if !cursedKillerOf(p) || cursedKillerOf(v) {
			t.Error("cursedKillerOf is wrong")
		}
	})
}

// The holder's death: Rnd.get(100) <= 50 vanishes the weapon, anything
// higher drops it; either way the holder gets its karma and PKs back.
func TestCursedWeaponHolderDeath(t *testing.T) {
	for _, c := range []struct {
		roll   int
		vanish bool
	}{{50, true}, {51, false}} {
		e := newCursedEnv(t, newRecordingStore(), world.NewFakeClock(testStart), memstore.NewIDs())
		sess, p := e.enterChar(testChar{account: "holder", name: "Holder", access: storage.AccessLevelAdmin, karma: 100})
		_, k := e.enterChar(testChar{account: "killer", name: "Killer"})
		e.onLoop(func() { e.srv.RunGMCommand(p, "cw_add zariche") })
		if !e.cwWeapon(cwZariche).Activated {
			t.Fatal("//cw_add did not activate Zariche")
		}
		sess.take(t)
		e.force([2]int{cursedweapon.DisappearBound, c.roll})
		e.onLoop(func() { e.srv.Kill(p, k) })
		ps := sess.take(t)
		w := e.cwWeapon(cwZariche)
		var has, onGround bool
		e.onLoop(func() {
			has = p.inv.ByItemID(cwZariche) != nil
			onGround = e.srv.cursed.rt[cwZariche].ground != nil
		})
		karma, pks, level := e.cwKarma(p)
		switch {
		case has:
			t.Errorf("roll %d: the dead holder still has the weapon", c.roll)
		case c.vanish && (w.Active() || onGround || !slices.Contains(cwMessages(ps), int32(cwMsgDisappeared))):
			t.Errorf("roll %d: the weapon did not vanish: %+v, ground %v", c.roll, w, onGround)
		case !c.vanish && (!w.Dropped || w.Activated || !onGround || !slices.Contains(cwMessages(ps), int32(cwMsgDroppedIn))):
			t.Errorf("roll %d: the weapon did not drop: %+v, ground %v", c.roll, w, onGround)
		}
		// The karma came back (100), then the death took it (below 200: 0).
		if karma != 0 || pks != 0 || level != 0 {
			t.Errorf("roll %d: after the death karma %d, PKs %d, level %d", c.roll, karma, pks, level)
		}
		if rows := e.cwRows(); rows != nil {
			t.Errorf("roll %d: rows after the death %+v", c.roll, rows)
		}
	}
}

// The life ends after 300 minutes on the fake clock: the online holder
// gets its karma back and loses the weapon and skills; everyone is told.
func TestCursedWeaponExpiry(t *testing.T) {
	e := newCursedEnv(t, newRecordingStore(), world.NewFakeClock(testStart), memstore.NewIDs())
	sess, p := e.enterChar(testChar{account: "holder", name: "Holder", access: storage.AccessLevelAdmin, karma: 720})
	e.onLoop(func() { e.srv.RunGMCommand(p, "cw_add 8689") })
	e.advance(299 * time.Minute)
	if !e.cwWeapon(cwAkamanah).Activated {
		t.Fatal("the weapon ended before its time")
	}
	sess.take(t)
	e.advance(time.Minute + 36*time.Second)
	ps := sess.take(t)
	if w := e.cwWeapon(cwAkamanah); w.Active() {
		t.Fatalf("the weapon lives after 300 minutes: %+v", w)
	}
	var has bool
	var skill int
	e.onLoop(func() { has, skill = p.inv.ByItemID(cwAkamanah) != nil, p.SkillLevel(3629) })
	if karma, _, _ := e.cwKarma(p); karma != 720 || has || skill != 0 {
		t.Errorf("after the end: karma %d, has the weapon %v, skill %d", karma, has, skill)
	}
	if !slices.Contains(cwMessages(ps), int32(cwMsgDisappeared)) {
		t.Errorf("no disappearance message: %v", cwMessages(ps))
	}
	if rows := e.cwRows(); rows != nil {
		t.Errorf("rows after the end %+v", rows)
	}
}

// A restart restores the holder: the timer runs on, the holder entering
// gets the skills and its minutes left. A weapon whose life ended while the
// server was down ends at startup, with its offline holder's copy, and its
// karma and PKs written back.
func TestCursedWeaponRestart(t *testing.T) {
	store, ids := newRecordingStore(), memstore.NewIDs()
	clock := world.NewFakeClock(testStart)
	e := newCursedEnv(t, store, clock, ids)
	_, p := e.enterChar(testChar{account: "holder", name: "Holder", access: storage.AccessLevelAdmin, karma: 720})
	_, q := e.enterChar(testChar{account: "other", name: "Other", access: storage.AccessLevelAdmin, karma: 30})
	holder, other := p.ObjectID(), q.ObjectID()
	e.onLoop(func() {
		e.srv.RunGMCommand(p, "cw_add 8190")
		e.srv.RunGMCommand(q, "cw_add 8689")
	})
	// Akamanah's life is cut to 10 minutes, so it ends during the downtime.
	e.onLoop(func() { e.srv.cursed.set.Get(cwAkamanah).EndTime = testStart.Add(10 * time.Minute).UnixMilli() })
	e.onLoop(func() { e.srv.cursedSave(e.srv.cursed.set.Get(cwAkamanah), q) })
	e.stop()

	clock.Advance(time.Hour)
	e2 := newCursedEnv(t, store, clock, ids)
	if w := e2.cwWeapon(cwAkamanah); w.Active() {
		t.Errorf("Akamanah lives after its end: %+v", w)
	}
	oc, err := store.Get(e2.ctx, other)
	if err != nil {
		t.Fatal(err)
	}
	if oc.Karma != 30 || slices.ContainsFunc(oc.Items, func(it storage.Item) bool { return it.ItemID == cwAkamanah }) {
		t.Errorf("Akamanah's offline holder: karma %d, items %+v", oc.Karma, oc.Items)
	}
	w := e2.cwWeapon(cwZariche)
	if !w.Activated || w.PlayerID != holder || w.PlayerKarma != 720 {
		t.Fatalf("Zariche after the restart: %+v", w)
	}
	sess := newSession("holder")
	p2 := e2.enter(sess, 0)
	e2.advance(time.Second)
	ps := sess.take(t)
	var skill int
	e2.onLoop(func() { skill = p2.SkillLevel(3603) })
	if skill != 1 || !slices.Contains(cwMessages(ps), int32(cwMsgMinutesLeft)) || !slices.Contains(cwMessages(ps), int32(cwMsgOwnerLoggedIn)) {
		t.Errorf("the holder entering: skill %d, messages %v", skill, cwMessages(ps))
	}
	ui := only[testclient.UserInfo](ps)
	if len(ui) == 0 || ui[0].Karma != cursedweapon.HolderKarma {
		t.Errorf("the holder's UserInfo: %+v", ui)
	}
	for _, m := range only[testclient.SystemMessage](ps) {
		if m.ID == int32(cwMsgMinutesLeft) && (len(m.Params) != 2 || m.Params[1].Number != 240) {
			t.Errorf("minutes left %+v, want 240", m.Params)
		}
	}
}

// A stray copy, one its owner does not hold, is deleted at enter world.
func TestCursedWeaponStrayCopy(t *testing.T) {
	e := newCursedEnv(t, newRecordingStore(), world.NewFakeClock(testStart), memstore.NewIDs())
	sess := newSession("stray")
	ch := e.createCharacter("stray", "Stray", classHumanFighter, storage.AccessLevelPlayer)
	id, _ := e.ids.NextID(e.ctx)
	ch.Items = append(ch.Items, storage.Item{ObjectID: id, ItemID: cwZariche, Count: 1, Mana: storage.NoMana})
	if err := e.store.Update(e.ctx, &ch); err != nil {
		t.Fatal(err)
	}
	p := e.enter(sess, 0)
	var has bool
	e.onLoop(func() { has = p.inv.ByItemID(cwZariche) != nil })
	if has {
		t.Error("the stray Zariche is still there")
	}
	if !slices.Contains(cwMessages(sess.take(t)), int32(sysMsgS1Disappeared)) {
		t.Error("no disappearance message for the stray copy")
	}
}

// At startup a copy of a weapon nobody holds is deleted and its owner's
// karma and PKs set to the weapon's (0), as Mobius controlPlayers does.
func TestCursedWeaponControlPlayers(t *testing.T) {
	store := newRecordingStore()
	ids := memstore.NewIDs()
	idsOf, _ := ids.NextIDs(context.Background(), 2)
	c := storage.Character{ObjectID: idsOf[0], Account: "stray", Name: "Stray", Level: 1, Karma: 500, PKKills: 4,
		Items: []storage.Item{{ObjectID: idsOf[1], ItemID: cwAkamanah, Count: 1, Slot: storage.PaperdollRHand, Mana: storage.NoMana}}}
	if err := store.Create(context.Background(), &c); err != nil {
		t.Fatal(err)
	}
	e := newCursedEnv(t, store, world.NewFakeClock(testStart), ids)
	got, err := store.Get(e.ctx, c.ObjectID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Karma != 0 || got.PKKills != 0 || len(got.Items) != 0 {
		t.Errorf("after startup: karma %d, PKs %d, items %+v", got.Karma, got.PKKills, got.Items)
	}
}

// The packets D0:22 and D0:23 and the GM commands.
func TestCursedWeaponPacketsAndGM(t *testing.T) {
	e := newCursedEnv(t, newRecordingStore(), world.NewFakeClock(testStart), memstore.NewIDs())
	sess, p := e.enterChar(testChar{account: "gm", name: "Gm", access: storage.AccessLevelAdmin})

	e.srv.HandleEx(sess, clientpackets.ExOpRequestCursedWeaponList, nil)
	e.srv.HandleEx(sess, clientpackets.ExOpRequestCursedWeaponLocation, nil)
	e.sync()
	ps := sess.take(t)
	lists := only[testclient.ExCursedWeaponList](ps)
	if len(lists) != 1 || !slices.Equal(lists[0].ItemIDs, []int32{cwAkamanah, cwZariche}) {
		t.Errorf("ExCursedWeaponList = %+v", lists)
	}
	if len(only[testclient.ExCursedWeaponLocation](ps)) != 0 {
		t.Error("a location list without a weapon in the world")
	}

	gm := func(line string) []testclient.Packet {
		t.Helper()
		sess.take(t)
		e.onLoop(func() { e.srv.RunGMCommand(p, line) })
		e.sync()
		return sess.take(t)
	}
	texts := func(ps []testclient.Packet) string {
		var b strings.Builder
		for _, s := range only[testclient.CreatureSay](ps) {
			b.WriteString(s.Text + "\n")
		}
		return b.String()
	}
	if got := texts(gm("cw_info")); !strings.Contains(got, "> Blood Sword Akamanah (8689)") || !strings.Contains(got, "Don't exist in the world.") {
		t.Errorf("//cw_info:\n%s", got)
	}
	if got := texts(gm("cw_add")); !strings.Contains(got, cwUsage) || !strings.Contains(got, "Unknown cursed weapon ID.") {
		t.Errorf("//cw_add without a weapon:\n%s", got)
	}
	if got := texts(gm("cw_add 57")); !strings.Contains(got, "Unknown cursed weapon ID.") {
		t.Errorf("//cw_add 57:\n%s", got)
	}
	gm("cw_add blood_sword")
	if karma, _, _ := e.cwKarma(p); karma != cursedweapon.HolderKarma {
		t.Fatalf("//cw_add: karma %d", karma)
	}
	if got := texts(gm("cw_add 8689")); !strings.Contains(got, "This cursed weapon is already active.") {
		t.Errorf("//cw_add twice:\n%s", got)
	}
	pages := only[testclient.NpcHtmlMessage](gm("cw_info_menu"))
	if len(pages) != 1 || !strings.Contains(pages[0].HTML, "Weilder:</td><td>Gm") || !strings.Contains(pages[0].HTML, "admin_cw_add 8190") {
		t.Errorf("//cw_info_menu: %+v", pages)
	}

	e.srv.HandleEx(sess, clientpackets.ExOpRequestCursedWeaponLocation, nil)
	e.sync()
	locs := only[testclient.ExCursedWeaponLocation](sess.take(t))
	pos := e.position(p)
	if len(locs) != 1 || len(locs[0].Weapons) != 1 ||
		locs[0].Weapons[0] != (testclient.CursedWeaponLocation{ItemID: cwAkamanah, Activated: true, X: pos.X, Y: pos.Y, Z: pos.Z}) {
		t.Errorf("ExCursedWeaponLocation = %+v", locs)
	}

	if got := texts(gm("cw_goto 8190")); got != "" {
		t.Errorf("//cw_goto to a weapon out of the world: chat %q", got)
	}
	gm("cw_remove akamanah")
	if w := e.cwWeapon(cwAkamanah); w.Active() {
		t.Errorf("//cw_remove left %+v", w)
	}
	if karma, _, _ := e.cwKarma(p); karma != 0 {
		t.Errorf("after //cw_remove karma %d", karma)
	}
}
