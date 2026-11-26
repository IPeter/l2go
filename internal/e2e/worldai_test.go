package e2e

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/game"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
)

// The world AI walk (Plan 13, Phase 10.9): a GM character sees a door and
// opens and closes it, kills the raid boss Greyclaw Kutus, and boards the
// Talking Island ferry with a ticket. TestWorldAI walks it against the
// in-process stack with a fake clock; TestSmokeWorldAI (smoke_test.go's
// build tag) against a running server.

// IDs of the walk, from the Mobius data.
const (
	// doorGludinHall is a Gludin clan hall door (Doors.xml
	// gludin_agit_003_003), closed at the start.
	doorGludinHall int32 = 17220001
	// npcKutus is the raid boss Greyclaw Kutus; his minions are three Guards
	// of Kutus and a Pawn of Kutus (the template's Privates).
	npcKutus      int32 = 25001
	npcKutusGuard int32 = 25002
	npcKutusPawn  int32 = 25003
	// itemTicketTalkingGludin is the ferry ticket Talking Island to Gludin.
	itemTicketTalkingGludin int32 = 1074
	// sysMsgRaidSuccessful is "Congratulations. Your raid was successful."
	sysMsgRaidSuccessful int32 = 1209
)

// The Talking Island ferry's announcements before and at its departure
// (scripts/vehicles/talking_gludin.lua, Mobius SystemMessageId).
var ferryCalls = []int32{
	981, // THE_FERRY_WILL_LEAVE_FOR_GLUDIN_HARBOR_IN_FIVE_MINUTES
	982, // ... IN_ONE_MINUTE
	983, // THOSE_WISHING_TO_RIDE_SHOULD_MAKE_HASTE_TO_GET_ON
	984, // THE_FERRY_WILL_BE_LEAVING_SOON_FOR_GLUDIN_HARBOR
	985, // THE_FERRY_IS_LEAVING_FOR_GLUDIN_HARBOR
}

// Places of the walk.
var (
	// doorSpot is next to the door (-84495, 155208, -3150).
	doorSpot = testclient.Position{X: -84400, Y: 155208, Z: -3150}
	// kutusSpot is next to Kutus's spawn (-54416, 146480, -2887).
	kutusSpot = testclient.Position{X: -54300, Y: 146480, Z: -2887}
	// dockSpot is on the Talking Island pier, 660 units from where the
	// ferry docks (-96622, 261660, -3610): within the 1,000 of boarding.
	dockSpot = testclient.Position{X: -96622, Y: 261000, Z: -3610}
	// ferryFirstPoint is where the ferry sails first after leaving.
	ferryFirstPoint = testclient.Position{X: -121385, Y: 261660, Z: -3610}
	// deckSpot is where the character stands on the deck.
	deckSpot = testclient.Position{X: 10, Y: -20, Z: 0}
)

// ferryDeparture is when the Talking Island ferry first leaves after the
// server starts: its schedule starts after 3 minutes, then waits 4 minutes,
// 40 s and 20 s (talking_gludin.lua).
const ferryDeparture = 8 * time.Minute

// worldAIWalk is what the world AI walk needs from its stack.
type worldAIWalk struct {
	// pass lets d of game time pass at once (the in-process stack); nil
	// waits in real time.
	pass func(t *testing.T, d time.Duration)
	// spawnRaid makes the raid step kill a Kutus it spawns with //spawn,
	// leaving the spawner's own alone (on a live server).
	spawnRaid bool
	// raidStored checks the stored respawn of the spawner's Kutus, who died
	// between from and to by the server's clock (serverNow). Nil skips it.
	raidStored func(t *testing.T, from, to time.Time)
	serverNow  func() time.Time
	// untilDeparture tells how long until the Talking Island ferry leaves;
	// nil skips the departure (a live server's ferry runs on its own time).
	untilDeparture func() time.Duration
}

// run walks the GM character on c through the steps. c is in the world.
func (wk worldAIWalk) run(t *testing.T, ctx context.Context, c *testclient.GameConn) {
	step := func(name string, fn func(t *testing.T)) {
		t.Helper()
		if !t.Run(name, fn) {
			t.FailNow()
		}
	}
	step("a door: StaticObjectInfo and DoorStatusUpdate, //open and //close", func(t *testing.T) {
		worldAIDoor(t, ctx, c)
	})
	step("a raid kill: the success message, the minions vanish", func(t *testing.T) {
		wk.raidKill(t, ctx, c)
	})
	step("boarding the Talking Island ferry", func(t *testing.T) {
		wk.ferry(t, ctx, c)
	})
}

// gm sends a GM command.
func gm(t *testing.T, ctx context.Context, c *testclient.GameConn, format string, args ...any) {
	t.Helper()
	cmd := fmt.Sprintf(format, args...)
	if err := c.GMCommand(ctx, cmd); err != nil {
		t.Fatalf("GMCommand(%q): %v", cmd, err)
	}
}

// awaitDoor waits for a DoorStatusUpdate of the door doorID that shows it
// closed or open.
func awaitDoor(t *testing.T, ctx context.Context, c *testclient.GameConn, doorID int32, closed bool) testclient.DoorStatusUpdate {
	t.Helper()
	return await(t, ctx, c, func(d testclient.DoorStatusUpdate) bool { return d.DoorID == doorID && d.Closed == closed })
}

// worldAIDoor: next to a Gludin clan hall door the GM sees it
// (StaticObjectInfo with its door ID, then DoorStatusUpdate); //open and
// //close change it, and each change comes as a DoorStatusUpdate.
func worldAIDoor(t *testing.T, ctx context.Context, c *testclient.GameConn) {
	teleport(t, ctx, c, doorSpot)
	var door testclient.Door
	poll(t, ctx, fmt.Sprintf("the door %d", doorGludinHall), func() (bool, error) {
		var ok bool
		door, ok = c.World().FindDoor(doorGludinHall)
		return ok, nil
	})
	if !slices.Contains(c.World().StaticObjects(), testclient.StaticObject{ObjectID: door.ObjectID, StaticID: doorGludinHall}) {
		t.Errorf("no StaticObjectInfo of the door %d (object %d): %+v", doorGludinHall, door.ObjectID, c.World().StaticObjects())
	}
	if door.MaxHP <= 0 || door.CurHP != door.MaxHP || door.Damage != 0 {
		t.Errorf("door status %+v, want a whole door", door)
	}
	if !door.Closed {
		gm(t, ctx, c, "close %d", doorGludinHall)
		awaitDoor(t, ctx, c, doorGludinHall, true)
	}

	gm(t, ctx, c, "open %d", doorGludinHall)
	if d := awaitDoor(t, ctx, c, doorGludinHall, false); d.ObjectID != door.ObjectID {
		t.Errorf("//open: DoorStatusUpdate of object %d, want %d", d.ObjectID, door.ObjectID)
	}
	if d, _ := c.World().FindDoor(doorGludinHall); d.Closed {
		t.Error("the world model shows the door closed after //open")
	}
	gm(t, ctx, c, "close %d", doorGludinHall)
	awaitDoor(t, ctx, c, doorGludinHall, true)
	if d, _ := c.World().FindDoor(doorGludinHall); !d.Closed {
		t.Error("the world model shows the door open after //close")
	}
}

// kutusGroup returns Kutus and his minions among the visible NPCs, leaving
// out the objects of skip: the boss and the object IDs of the minions.
func kutusGroup(w *testclient.World, skip map[int32]bool) (boss testclient.Object, guards, pawns []int32) {
	for _, o := range w.Npcs() {
		if skip[o.ObjectID] || o.Dead {
			continue
		}
		switch o.NpcID {
		case npcKutus:
			boss = o
		case npcKutusGuard:
			guards = append(guards, o.ObjectID)
		case npcKutusPawn:
			pawns = append(pawns, o.ObjectID)
		}
	}
	return boss, guards, pawns
}

// raidKill: Kutus stands with his three guards and his pawn. The GM kills
// him with //kill: he dies, the GM gets the raid success message and the
// minions vanish (DeleteObject). On the in-process stack his respawn, 12 to
// 60 hours later, is stored.
func (wk worldAIWalk) raidKill(t *testing.T, ctx context.Context, c *testclient.GameConn) {
	teleport(t, ctx, c, kutusSpot)
	skip := map[int32]bool{}
	if wk.spawnRaid {
		// What the GM sees now is the spawner's group; the next Kutus is the
		// GM's own.
		time.Sleep(time.Second)
		for _, o := range c.World().Npcs() {
			skip[o.ObjectID] = true
		}
		if err := c.CancelTarget(ctx); err != nil {
			t.Fatalf("CancelTarget: %v", err)
		}
		gm(t, ctx, c, "spawn %d", npcKutus)
	}
	var (
		boss          testclient.Object
		guards, pawns []int32
	)
	poll(t, ctx, "Kutus with his three guards and his pawn", func() (bool, error) {
		boss, guards, pawns = kutusGroup(c.World(), skip)
		return boss.ObjectID != 0 && len(guards) == 3 && len(pawns) == 1, nil
	})
	if !boss.Attackable {
		t.Errorf("Kutus %+v is not attackable", boss)
	}
	if _, err := c.Target(ctx, boss.ObjectID); err != nil {
		t.Fatalf("Target(Kutus): %v", err)
	}

	var from time.Time
	if wk.serverNow != nil {
		from = wk.serverNow()
	}
	gm(t, ctx, c, "kill")
	minions := map[int32]bool{}
	for _, id := range append(guards, pawns...) {
		minions[id] = true
	}
	died, success := false, false
	_, err := c.WaitFor(ctx, func(p testclient.Packet) bool {
		switch v := p.(type) {
		case testclient.Die:
			died = died || v.ObjectID == boss.ObjectID
		case testclient.SystemMessage:
			success = success || v.ID == sysMsgRaidSuccessful
		case testclient.DeleteObject:
			delete(minions, v.ObjectID)
		}
		return died && success && len(minions) == 0
	})
	if err != nil {
		t.Fatalf("after //kill: Kutus died %v, raid success message %v, minions left %v: %v", died, success, minions, err)
	}
	if o, ok := c.World().Object(boss.ObjectID); !ok || !o.Dead {
		t.Errorf("Kutus in the world model: %+v, %v; want dead", o, ok)
	}
	if wk.raidStored != nil {
		wk.raidStored(t, from, wk.serverNow())
	}
}

// ferry: the GM gets a ticket and boards the Talking Island ferry, which
// stands at the pier. On the in-process stack time passes to the
// departure: the calls come, the ticket is taken, the ferry sails, and the
// passenger goes with it.
func (wk worldAIWalk) ferry(t *testing.T, ctx context.Context, c *testclient.GameConn) {
	w := c.World()
	gm(t, ctx, c, "create_item %d 1", itemTicketTalkingGludin)
	poll(t, ctx, "the ferry ticket", func() (bool, error) { return w.ItemCount(itemTicketTalkingGludin) == 1, nil })

	teleport(t, ctx, c, dockSpot)
	var boat testclient.Boat
	poll(t, ctx, "the ferry", func() (bool, error) {
		var ok bool
		boat, ok = w.NearestBoat()
		return ok, nil
	})
	if wk.untilDeparture == nil && boat.Moving {
		t.Skipf("the ferry is under way (%+v); boarding needs it at the pier", boat)
	}
	if d := boat.Pos.DistanceTo(dockSpot); d > 1000 {
		if wk.untilDeparture == nil {
			t.Skipf("the ferry is %.0f units away, not at the pier", d)
		}
		t.Fatalf("the ferry is %.0f units from the pier spot: %+v", d, boat)
	}
	g, err := c.BoardShip(ctx, boat.ObjectID, deckSpot)
	if err != nil {
		t.Fatalf("BoardShip: %v", err)
	}
	if g.BoatID != boat.ObjectID || g.Position() != deckSpot {
		t.Errorf("GetOnVehicle %+v, want boat %d at %v", g, boat.ObjectID, deckSpot)
	}
	if id, deck := w.OnBoat(); id != boat.ObjectID || deck != deckSpot {
		t.Errorf("OnBoat = %d, %v", id, deck)
	}
	if wk.untilDeparture == nil {
		return
	}

	// Time passes to the departure and a little beyond.
	if _, err := c.Drain(); err != nil {
		t.Fatalf("Drain: %v", err)
	}
	wk.pass(t, wk.untilDeparture()+2*time.Second)
	var calls []int32
	departed, started := false, false
	_, err = c.WaitFor(ctx, func(p testclient.Packet) bool {
		switch v := p.(type) {
		case testclient.CreatureSay:
			if v.Type == testclient.ChatBoat {
				calls = append(calls, v.MessageID)
			}
		case testclient.VehicleStarted:
			started = started || (v.ObjectID == boat.ObjectID && v.State == 1)
		case testclient.VehicleDeparture:
			departed = departed || (v.ObjectID == boat.ObjectID && v.Destination() == ferryFirstPoint)
		}
		return started && departed && inOrder(ferryCalls, calls)
	})
	if err != nil {
		t.Fatalf("waiting for the departure: calls %v (want %v), started %v, departed %v: %v", calls, ferryCalls, started, departed, err)
	}
	poll(t, ctx, "the ticket taken", func() (bool, error) { return w.ItemCount(itemTicketTalkingGludin) == 0, nil })
	if id, _ := w.OnBoat(); id != boat.ObjectID {
		t.Errorf("after the departure the character is on boat %d, want %d", id, boat.ObjectID)
	}

	// Under way, the passenger moves with the ferry: /loc tells where.
	wk.pass(t, 20*time.Second)
	if err := c.UserCommand(ctx, testclient.UserCmdLoc); err != nil {
		t.Fatalf("UserCommand: %v", err)
	}
	m := await(t, ctx, c, func(m testclient.SystemMessage) bool { _, ok := messageLocation(m); return ok })
	loc, _ := messageLocation(m)
	if loc[0] > int64(dockSpot.X)-1000 {
		t.Errorf("20 s after the departure the passenger is at %v, want well west of the pier %v", loc, dockSpot)
	}
}

// inOrder reports whether want appears in got in order, with anything in
// between.
func inOrder(want, got []int32) bool {
	i := 0
	for _, g := range got {
		if i < len(want) && g == want[i] {
			i++
		}
	}
	return i == len(want)
}

// withWorldAI gives the in-process stack the doors (data/objects), the raid
// bosses (data/bosses, data/spawns/raidbosses.json) and an in-memory boss
// store.
func withWorldAI(t *testing.T, bosses storage.BossStore) func(*game.Options) {
	return func(o *game.Options) {
		dir := o.Config.GameServers[0].DataDir
		objects, err := data.LoadObjects(dir)
		if err != nil {
			t.Fatal(err)
		}
		b, err := data.LoadBosses(dir)
		if err != nil {
			t.Fatal(err)
		}
		if err := b.CheckWorld(o.World); err != nil {
			t.Fatal(err)
		}
		o.Objects, o.Bosses, o.BossStore = objects, b, bosses
	}
}

// TestWorldAI walks a GM through a door, a raid kill and the ferry on the
// in-process stack. The fake clock lets the 8 minutes before the ferry
// leaves pass at once.
func TestWorldAI(t *testing.T) {
	bosses := memstore.NewBosses()
	s := startStack(t, withWorldAI(t, bosses))
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	const account, password, name = "worldai", "Secret-1", "Wanderer"
	c := gameLogin(t, ctx, s, account, password)
	res, err := c.CreateCharacter(ctx, humanFighter(name))
	if err != nil || !res.Created {
		t.Fatalf("CreateCharacter: %+v, %v", res, err)
	}
	s.makeGM(t, account)
	if _, err := c.SelectCharacter(ctx, 0); err != nil {
		t.Fatalf("SelectCharacter: %v", err)
	}
	if _, err := c.EnterWorld(ctx); err != nil {
		t.Fatalf("EnterWorld: %v", err)
	}

	worldAIWalk{
		pass:      s.passTime,
		serverNow: s.clock.Now,
		raidStored: func(t *testing.T, from, to time.Time) {
			var row storage.RaidBossState
			// The boss state is written in the background.
			poll(t, ctx, "Kutus's stored respawn", func() (bool, error) {
				rows, err := bosses.LoadRaidBosses(ctx)
				if err != nil {
					return false, err
				}
				i := slices.IndexFunc(rows, func(r storage.RaidBossState) bool { return r.BossID == npcKutus })
				if i < 0 || rows[i].RespawnTime == 0 {
					return false, nil
				}
				row = rows[i]
				return true, nil
			})
			at := time.UnixMilli(row.RespawnTime)
			if lo, hi := from.Add(12*time.Hour), to.Add(60*time.Hour); at.Before(lo) || at.After(hi) {
				t.Errorf("Kutus respawns at %v, want between %v and %v (12 to 60 hours after the kill)", at, lo, hi)
			}
		},
		untilDeparture: func() time.Duration { return max(clockStart.Add(ferryDeparture).Sub(s.clock.Now()), 0) },
	}.run(t, ctx, c)
}
