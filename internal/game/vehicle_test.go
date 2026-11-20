package game

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The ferry tests run the real Talking Island–Gludin schedule
// (scripts/vehicles/talking_gludin.lua) on the fake clock.

var (
	// tiDock and gludinDock are where the ferry anchors.
	tiDock     = world.Position{X: -96622, Y: 261660, Z: -3610}
	gludinDock = world.Position{X: -95686, Y: 150514, Z: -3610}
	// tiOust is where a player without a ticket is put ashore.
	tiOust = world.Position{X: -96777, Y: 258970, Z: -3623}
)

const (
	ticketTalkingToGludin = 1074
	ticketGludinToTalking = 1075
)

// newBoatEnv starts a server with the world data and the Talking Island–
// Gludin ferry.
func newBoatEnv(t *testing.T) *testEnv {
	t.Helper()
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join("..", "..", "scripts", filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	return newDialogEnv(t, dialogEnvOptions{scripts: map[string]string{
		"lib/boat.lua":                read("lib/boat.lua"),
		"vehicles/talking_gludin.lua": read("vehicles/talking_gludin.lua"),
	}})
}

// theBoat returns the one boat of the world.
func theBoat(t *testing.T, e *testEnv) *Boat {
	t.Helper()
	var boats []*Boat
	e.onLoop(func() {
		for _, b := range e.srv.vehicles.boats {
			boats = append(boats, b)
		}
	})
	if len(boats) != 1 {
		t.Fatalf("%d boats, want 1", len(boats))
	}
	return boats[0]
}

// vehicleBody encodes a request body of int32 fields.
func vehicleBody(fields ...int32) []byte {
	b := make([]byte, 0, 4*len(fields))
	for _, f := range fields {
		b = binary.LittleEndian.AppendUint32(b, uint32(f))
	}
	return b
}

// boatSays returns the message IDs of the boat announcements in ps.
func boatSays(ps []testclient.Packet) []int32 {
	var out []int32
	for _, s := range only[testclient.CreatureSay](ps) {
		if s.Type == testclient.ChatBoat && s.NameID == 801 {
			out = append(out, s.MessageID)
		}
	}
	return out
}

// sailTime is how long the engine takes to sail from from along path: each
// leg takes the first whole number of game ticks in which speed / 10 per
// tick covers it (walkTrack.step).
func sailTime(from world.Position, path []scripting.VehiclePoint) time.Duration {
	var ticks int64
	for _, p := range path {
		d := math.Hypot(float64(p.X)-float64(from.X), float64(p.Y)-float64(from.Y))
		ticks += int64(math.Floor(d*10/float64(p.Speed))) + 1
		from = boatPointPos(p)
	}
	return time.Duration(ticks) * world.GameTick
}

// heard is a boat announcement and when it came.
type heard struct {
	at  time.Duration
	msg int32
}

// A full Talking Island–Gludin cycle: the calls at Mobius's times, the
// departure with its tickets (a player without one is put ashore), the
// passengers carried along, a passenger who logs out saved ashore, a late
// arrival who sees the ship moving and its passengers on board, the calls
// at Gludin, the way back, and the first call of the next cycle.
func TestBoatTalkingGludinCycle(t *testing.T) {
	e := newBoatEnv(t)
	boat := theBoat(t, e)
	if pos := e.positionOf(boat); pos.X != tiDock.X || pos.Y != tiDock.Y || pos.Heading != 32768 {
		t.Fatalf("the ferry starts at %v, want the Talking Island dock", pos)
	}
	var route scripting.VehicleRoute
	e.onLoop(func() { route = e.srv.vehicles.schedules[0].route })

	near := world.Position{X: tiDock.X, Y: tiDock.Y - 600, Z: tiDock.Z}
	ticket := func(id int32) []storage.Item {
		return []storage.Item{{ItemID: id, Count: 1, Slot: storage.PaperdollNone, Mana: storage.NoMana}}
	}
	alice, pa := e.enterChar(testChar{account: "alice", name: "Alice", pos: near, items: ticket(ticketTalkingToGludin)})
	bob, pb := e.enterChar(testChar{account: "bob", name: "Bob", pos: near})
	eve, pe := e.enterChar(testChar{account: "eve", name: "Eve", pos: near, items: ticket(ticketTalkingToGludin)})
	carol, _ := e.enterChar(testChar{account: "carol", name: "Carol", pos: world.Position{X: 83400, Y: 147943, Z: -3404}})
	// Boarding before the ferry runs its schedule: Alice and Bob get on.
	board := func(sess *fakeSession, deck int32) {
		e.srv.Handle(sess, clientpackets.OpRequestGetOnVehicle, vehicleBody(boat.ObjectID(), deck, -10, 20))
		e.sync()
	}
	board(alice, 5)
	board(bob, 6)
	board(eve, 7)
	ons := only[testclient.GetOnVehicle](bob.take(t))
	if len(ons) != 3 || ons[0].ObjectID != pa.ObjectID() || ons[0].BoatID != boat.ObjectID() || ons[0].X != 5 {
		t.Fatalf("Bob saw %+v; want the three boardings, Alice's first at deck x 5", ons)
	}
	var onBoard, peace bool
	var at world.Position
	e.onLoop(func() {
		onBoard, peace, at = pa.Vehicle() == boat, pa.Zones.Inside(ZonePeace), pa.Position()
	})
	if !onBoard || !peace || at.X != tiDock.X || at.Y != tiDock.Y {
		t.Fatalf("Alice on board %v, in peace %v, at %v; want on board at the dock in peace", onBoard, peace, at)
	}
	alice.take(t)
	eve.take(t)

	var bobHeard []heard
	var aliceGot, bobGot []testclient.Packet
	elapsed := time.Duration(0)
	step := func(d time.Duration) {
		e.advance(d)
		elapsed += d
		ab := bob.take(t)
		for _, m := range boatSays(ab) {
			bobHeard = append(bobHeard, heard{elapsed, m})
		}
		bobGot = append(bobGot, ab...)
		aliceGot = append(aliceGot, alice.take(t)...)
		if len(boatSays(carol.take(t))) != 0 {
			t.Fatalf("Carol, far away, heard a boat at %v", elapsed)
		}
		eve.take(t)
	}
	for elapsed < 480*time.Second {
		step(time.Second)
	}

	// The departure: Alice paid, Bob had no ticket and is ashore.
	var aliceTickets, bobOn int64
	var passengers []*Player
	e.onLoop(func() {
		if it := pa.inv.ByItemID(ticketTalkingToGludin); it != nil {
			aliceTickets = it.Count()
		}
		if pb.Vehicle() != nil {
			bobOn = 1
		}
		passengers = boat.Passengers()
	})
	if aliceTickets != 0 || bobOn != 0 || len(passengers) != 2 {
		t.Errorf("at the departure: Alice has %d tickets, Bob on board %d, %d passengers; want 0, 0, 2", aliceTickets, bobOn, len(passengers))
	}
	if len(only[testclient.InventoryUpdate](aliceGot)) == 0 {
		t.Error("Alice got no InventoryUpdate for her ticket")
	}
	if msgs := systemMessages(bobGot, int32(sysMsgNoTicket)); len(msgs) != 1 {
		t.Errorf("Bob got %d \"no ticket\" messages, want 1", len(msgs))
	}
	tps := only[testclient.TeleportToLocation](bobGot)
	if len(tps) != 1 || abs32(tps[0].X-tiOust.X) > maxOffsetOnTeleport || abs32(tps[0].Y-tiOust.Y) > maxOffsetOnTeleport {
		t.Errorf("Bob's teleports %+v; want one to the oust point %v", tps, tiOust)
	}
	starts := only[testclient.VehicleStarted](aliceGot)
	deps := only[testclient.VehicleDeparture](aliceGot)
	first := route.Steps[3].Path[0]
	if len(starts) != 1 || starts[0].State != 1 || len(deps) != 1 || deps[0].MoveSpeed != 180 || deps[0].RotationSpeed != 800 ||
		deps[0].X != int32(first.X) || deps[0].Y != int32(first.Y) {
		t.Errorf("Alice saw %+v and %+v; want VehicleStarted 1 and VehicleDeparture to %+v at 180/800", starts, deps, first)
	}
	horns := only[testclient.PlaySound](bobGot)
	if len(horns) != 1 || horns[0].Sound != boatSound || horns[0].ObjectID != boat.ObjectID() || horns[0].X != tiDock.X {
		t.Errorf("the departure horn: %+v", horns)
	}

	// Under way: the passengers move with the ship; UserInfo names it.
	step(30 * time.Second)
	bp := e.positionOf(boat)
	if ap := e.position(pa); ap.X != bp.X || ap.Y != bp.Y || bp.X == tiDock.X {
		t.Errorf("after 30 s at sea the boat is at %v and Alice at %v; want her with the moving boat", bp, ap)
	}
	var vehicleID int32
	e.onLoop(func() { pa.sendUserInfo() })
	for _, u := range only[testclient.UserInfo](alice.take(t)) {
		vehicleID = u.VehicleID
	}
	if vehicleID != boat.ObjectID() {
		t.Errorf("Alice's UserInfo names vehicle %d, want %d", vehicleID, boat.ObjectID())
	}

	// A late arrival next to the ship sees it moving, and Alice on its deck.
	bp = e.positionOf(boat)
	dave, _ := e.enterCharRaw(testChar{account: "dave", name: "Dave", pos: world.Position{X: bp.X, Y: bp.Y + 300, Z: bp.Z}})
	names := daveSaw(dave.take(t), boat.ObjectID(), pa.ObjectID())
	if !follows(names, "VehicleInfo", "VehicleDeparture") || !follows(names, "CharInfo", "GetOnVehicle") || len(names) != 4 {
		t.Errorf("Dave saw %q about the ferry and Alice; want VehicleInfo then VehicleDeparture, and CharInfo on deck then GetOnVehicle", names)
	}

	// Eve logs out at sea and is saved ashore, off the ship.
	e.srv.Logout(eve)
	saved := e.store.waitSave(t, pe.ObjectID())
	if math.Hypot(float64(saved.X-bp.X), float64(saved.Y-bp.Y)) < 3000 {
		t.Errorf("Eve was saved at %d, %d, next to the ship", saved.X, saved.Y)
	}
	e.onLoop(func() { passengers = boat.Passengers() })
	if len(passengers) != 1 {
		t.Errorf("%d passengers after Eve left, want 1", len(passengers))
	}

	// On to Gludin.
	toGludin := 480*time.Second + sailTime(tiDock, route.Steps[3].Path)
	arrive := toGludin + sailTime(boatPointPos(route.Steps[3].Path[len(route.Steps[3].Path)-1]), route.Steps[7].Path)
	for !heardMsg(bobHeard, 986) && elapsed < arrive+5*time.Second {
		step(time.Second)
	}
	if bp := e.positionOf(boat); bp.X != gludinDock.X || bp.Y != gludinDock.Y {
		t.Fatalf("at %v the ferry is at %v, want the Gludin dock", elapsed, bp)
	}
	e.srv.Handle(alice, clientpackets.OpRequestGetOffVehicle, vehicleBody(boat.ObjectID(), gludinDock.X+300, gludinDock.Y, gludinDock.Z))
	e.sync()
	e.onLoop(func() { onBoard, peace, at = pa.Vehicle() != nil, pa.Zones.Inside(ZonePeace), pa.Position() })
	if onBoard || at.X != gludinDock.X+300 {
		t.Errorf("Alice after getting off: on board %v at %v", onBoard, at)
	}
	offs := only[testclient.GetOffVehicle](alice.take(t))
	if len(offs) != 1 || offs[0].X != gludinDock.X+300 {
		t.Errorf("Alice saw %+v; want her GetOffVehicle", offs)
	}

	// Back to Talking Island and the first call of the next cycle.
	leaveGludin := arrive + 600*time.Second
	back := leaveGludin + sailTime(gludinDock, route.Steps[12].Path)
	home := back + sailTime(boatPointPos(route.Steps[12].Path[len(route.Steps[12].Path)-1]), route.Steps[16].Path)
	for elapsed < home+301*time.Second {
		step(time.Second)
	}

	want := []heard{
		{180 * time.Second, 981},
		{420 * time.Second, 982}, {420 * time.Second, 983},
		{460 * time.Second, 984},
		{480 * time.Second, 985},
		{780 * time.Second, 1159},
		{1080 * time.Second, 1160},
		{1320 * time.Second, 1161},
		{arrive, 986}, {arrive, 987},
		{arrive + 300*time.Second, 988},
		{arrive + 540*time.Second, 989}, {arrive + 540*time.Second, 983},
		{arrive + 580*time.Second, 990},
		{leaveGludin, 991},
		{leaveGludin + 150*time.Second, 1191},
		{leaveGludin + 450*time.Second, 1192},
		{leaveGludin + 690*time.Second, 1193},
		{home, 979}, {home, 980},
		{home + 300*time.Second, 981},
	}
	if len(bobHeard) != len(want) {
		t.Fatalf("Bob heard %d announcements, want %d:\n got %v\nwant %v", len(bobHeard), len(want), bobHeard, want)
	}
	for i, w := range want {
		g := bobHeard[i]
		// The calls the timers make come on the second; those after a path
		// within two seconds of the sailing time (the test steps a second at
		// a time).
		slack := time.Duration(0)
		if w.at > 1320*time.Second {
			slack = 2 * time.Second
		}
		if g.msg != w.msg || g.at < w.at || g.at > w.at+slack {
			t.Errorf("announcement %d: %d at %v, want %d at %v", i, g.msg, g.at, w.msg, w.at)
		}
	}
	var docks [boatDockSlots]bool
	e.onLoop(func() { docks = e.srv.vehicles.docks })
	if !docks[1] || docks[2] {
		t.Errorf("docks %v; want Talking Island busy and Gludin free", docks)
	}
}

// follows reports whether b comes right after a in names.
func follows(names []string, a, b string) bool {
	for i := 0; i+1 < len(names); i++ {
		if names[i] == a {
			return names[i+1] == b
		}
	}
	return false
}

func heardMsg(hs []heard, msg int32) bool {
	for _, h := range hs {
		if h.msg == msg {
			return true
		}
	}
	return false
}

// daveSaw returns the names of the packets about the boat and the
// passenger, in order.
func daveSaw(ps []testclient.Packet, boatID, passengerID int32) []string {
	var out []string
	for _, p := range ps {
		switch p := p.(type) {
		case testclient.VehicleInfo:
			if p.ObjectID == boatID {
				out = append(out, p.PacketName())
			}
		case testclient.VehicleDeparture:
			if p.ObjectID == boatID {
				out = append(out, p.PacketName())
			}
		case testclient.CharInfoPacket:
			if p.ObjectID == passengerID && p.VehicleID == boatID && p.X == 5 {
				out = append(out, p.PacketName())
			}
		case testclient.GetOnVehicle:
			if p.ObjectID == passengerID && p.BoatID == boatID {
				out = append(out, p.PacketName())
			}
		}
	}
	return out
}

// enterCharRaw is enterChar without dropping the first packets.
func (e *testEnv) enterCharRaw(c testChar) (*fakeSession, *Player) {
	e.t.Helper()
	ch := e.createCharacter(c.account, c.name, classHumanFighter, c.access)
	ch.X, ch.Y, ch.Z = c.pos.X, c.pos.Y, c.pos.Z
	if err := e.store.Update(e.ctx, &ch); err != nil {
		e.t.Fatalf("Update: %v", err)
	}
	sess := newSession(c.account)
	p := e.enter(sess, 0)
	return sess, p
}

// positionOf returns an object's position, read on the loop.
func (e *testEnv) positionOf(o world.Object) world.Position {
	var pos world.Position
	e.onLoop(func() { pos = o.Position() })
	return pos
}

// Boarding needs the ship standing and within 1,000; a passenger may only
// name its own ship; getting off needs the ship standing and a point
// within 1,000. Walking on the deck shows MoveToLocationInVehicle, and a
// stop StopMoveInVehicle, to the others.
func TestBoatBoardingRules(t *testing.T) {
	e := newBoatEnv(t)
	boat := theBoat(t, e)
	far, _ := e.enterChar(testChar{account: "far", name: "Far", pos: world.Position{X: tiDock.X, Y: tiDock.Y - 1500, Z: tiDock.Z}})
	alice, pa := e.enterChar(testChar{account: "alice", name: "Alice", pos: world.Position{X: tiDock.X, Y: tiDock.Y - 600, Z: tiDock.Z}})
	bob, _ := e.enterChar(testChar{account: "bob", name: "Bob", pos: world.Position{X: tiDock.X, Y: tiDock.Y - 700, Z: tiDock.Z}})
	send := func(sess *fakeSession, op byte, fields ...int32) {
		e.srv.Handle(sess, op, vehicleBody(fields...))
		e.sync()
	}

	send(far, clientpackets.OpRequestGetOnVehicle, boat.ObjectID(), 0, 0, 0)
	if ps := far.take(t); len(only[testclient.ActionFailed](ps)) != 1 || len(only[testclient.GetOnVehicle](ps)) != 0 {
		t.Errorf("boarding from 1,500 away: %q; want ActionFailed", testclient.Names(ps))
	}
	send(alice, clientpackets.OpRequestGetOnVehicle, boat.ObjectID()+1, 0, 0, 0)
	if ps := alice.take(t); len(only[testclient.ActionFailed](ps)) != 1 {
		t.Errorf("boarding an unknown boat: %q; want ActionFailed", testclient.Names(ps))
	}
	var peaceBefore bool
	e.onLoop(func() { peaceBefore = pa.Zones.Inside(ZonePeace) })
	send(alice, clientpackets.OpRequestGetOnVehicle, boat.ObjectID(), 1, 2, 3)
	alice.take(t)
	bob.take(t)

	// Walking on the deck.
	send(alice, clientpackets.OpRequestMoveToLocationInVehicle, boat.ObjectID(), 50, 60, 70, 1, 2, 3)
	moves := only[testclient.MoveToLocationInVehicle](bob.take(t))
	if len(moves) != 1 || moves[0] != (testclient.MoveToLocationInVehicle{ObjectID: pa.ObjectID(), BoatID: boat.ObjectID(), DestX: 50, DestY: 60, DestZ: 70, X: 1, Y: 2, Z: 3}) {
		t.Errorf("Bob saw %+v; want Alice walking on the deck", moves)
	}
	send(alice, clientpackets.OpCannotMoveAnymoreInVehicle, boat.ObjectID(), 40, 50, 60, 1000)
	stops := only[testclient.StopMoveInVehicle](bob.take(t))
	if len(stops) != 1 || stops[0].X != 40 || stops[0].Heading != 1000 {
		t.Errorf("Bob saw %+v; want Alice stopping at 40, 50, 60", stops)
	}
	alice.take(t)
	// A passenger's position reports change nothing.
	send(alice, clientpackets.OpValidatePosition, tiDock.X+500, tiDock.Y, tiDock.Z, 0, boat.ObjectID())
	if pos := e.position(pa); pos.X != tiDock.X {
		t.Errorf("a passenger's ValidatePosition moved her to %v", pos)
	}

	// Getting off too far away fails; next to the ship it works.
	send(alice, clientpackets.OpRequestGetOffVehicle, boat.ObjectID(), tiDock.X, tiDock.Y-1200, tiDock.Z)
	if ps := alice.take(t); len(only[testclient.ActionFailed](ps)) != 1 || len(only[testclient.GetOffVehicle](ps)) != 0 {
		t.Errorf("getting off 1,200 away: %q", testclient.Names(ps))
	}
	send(alice, clientpackets.OpRequestGetOffVehicle, boat.ObjectID(), tiDock.X, tiDock.Y-500, tiDock.Z)
	names := testclient.Names(bob.take(t))
	stop, off := -1, -1
	for i, n := range names {
		switch n {
		case "StopMoveInVehicle":
			stop = i
		case "GetOffVehicle":
			off = i
		}
	}
	if stop < 0 || off < stop {
		t.Errorf("Bob saw %q when Alice got off; want StopMoveInVehicle, then GetOffVehicle", names)
	}
	var peace, aboard bool
	e.onLoop(func() { peace, aboard = pa.Zones.Inside(ZonePeace), pa.Vehicle() != nil })
	if peace != peaceBefore || aboard {
		t.Errorf("Alice ashore: in peace %v (before boarding %v), aboard %v", peace, peaceBefore, aboard)
	}
}

// The docks are Mobius's array of three: the Rune dock (3) never becomes
// busy.
func TestBoatDocks(t *testing.T) {
	e := newEnv(t, nil)
	e.onLoop(func() {
		s := e.srv
		s.dockShip(2, true)
		s.dockShip(3, true)
		if !s.dockBusy(2) || s.dockBusy(3) || s.dockBusy(1) {
			t.Errorf("docks: Gludin %v, Rune %v, Talking Island %v; want true, false, false", s.dockBusy(2), s.dockBusy(3), s.dockBusy(1))
		}
	})
}
