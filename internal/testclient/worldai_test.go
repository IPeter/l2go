package testclient

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
)

// The cinematic packets decode from the Mobius writeImpl layouts.
func TestDecodeCinematicPackets(t *testing.T) {
	const boss = 0x60000030
	for _, tt := range []struct {
		name string
		raw  []byte
		want Packet
	}{
		// SpecialCamera: d object, force, angle1, angle2, time, duration,
		// relYaw, relPitch, isWide, relAngle, unk.
		{"SpecialCamera", msg(0xc7).i32(boss).i32(1300).i32(270).i32(-20).i32(0).i32(10000).i32(15).i32(0).i32(1).i32(0).i32(0).b,
			SpecialCamera{ObjectID: boss, Force: 1300, Angle1: 270, Angle2: -20, Duration: 10000, RelYaw: 15, Wide: 1}},
		// Earthquake: d x, y, z, intensity, duration, and 0.
		{"Earthquake", msg(0xc4).i32(185708).i32(114298).i32(-8221).i32(20).i32(10).i32(0).b,
			Earthquake{X: 185708, Y: 114298, Z: -8221, Intensity: 20, Duration: 10}},
		// ExRedSky: d duration.
		{"ExRedSky", msg(0xfe).u16(0x40).i32(10).b, ExRedSky{Duration: 10}},
	} {
		got, err := DecodeServerPacket(tt.raw)
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: got %+v, want %+v", tt.name, got, tt.want)
		}
		if got.PacketName() != tt.name {
			t.Errorf("%s: named %q", tt.name, got.PacketName())
		}
		if _, err := DecodeServerPacket(tt.raw[:len(tt.raw)-1]); err == nil {
			t.Errorf("%s: a short packet decoded", tt.name)
		}
	}
	// Mobius always writes the last int of Earthquake as 0.
	if _, err := DecodeServerPacket(msg(0xc4).i32(1).i32(2).i32(3).i32(4).i32(5).i32(1).b); !errors.Is(err, ErrMalformed) {
		t.Errorf("Earthquake with a last int of 1: %v, want ErrMalformed", err)
	}
	if e := (Earthquake{X: 1, Y: 2, Z: 3}); e.Position() != (Position{1, 2, 3}) {
		t.Errorf("Earthquake.Position = %v", e.Position())
	}
}

// The world model keeps the doors, static objects and boats.
func TestWorldDoorsAndStatics(t *testing.T) {
	const door, door2, townMap = 0x60000010, 0x60000011, 0x60000012
	w := NewWorld()
	w.Apply(StaticObjectInfo{StaticID: 17220001, ObjectID: door})
	w.Apply(DoorStatusUpdate{ObjectID: door, Closed: true, DoorID: 17220001, MaxHP: 316500, CurHP: 316500})
	w.Apply(StaticObjectInfo{StaticID: 17220002, ObjectID: door2})
	w.Apply(DoorStatusUpdate{ObjectID: door2, Closed: true, DoorID: 17220002, MaxHP: 316500, CurHP: 316500})
	w.Apply(StaticObjectInfo{StaticID: 24190001, ObjectID: townMap})

	statics := w.StaticObjects()
	if len(statics) != 3 || statics[2] != (StaticObject{ObjectID: townMap, StaticID: 24190001}) {
		t.Errorf("StaticObjects = %+v", statics)
	}
	if ds := w.Doors(); len(ds) != 2 || ds[0].ObjectID != door || ds[1].ObjectID != door2 {
		t.Errorf("Doors = %+v", ds)
	}
	d, ok := w.FindDoor(17220001)
	if !ok || !d.Closed || d.ObjectID != door || d.MaxHP != 316500 {
		t.Errorf("FindDoor = %+v, %v", d, ok)
	}

	// //open: the door's status changes.
	w.Apply(DoorStatusUpdate{ObjectID: door, DoorID: 17220001, MaxHP: 316500, CurHP: 316500})
	if d, _ := w.FindDoor(17220001); d.Closed {
		t.Error("the door stays closed after an open DoorStatusUpdate")
	}
	if _, ok := w.FindDoor(99); ok {
		t.Error("FindDoor found an unknown door")
	}

	// Out of sight: the door is forgotten.
	w.Apply(DeleteObject{ObjectID: door})
	if _, ok := w.FindDoor(17220001); ok || len(w.StaticObjects()) != 2 {
		t.Errorf("after DeleteObject: doors %+v, statics %+v", w.Doors(), w.StaticObjects())
	}

	// A new character starts with nothing.
	w.Apply(CharSelected{ObjectID: testMeID})
	if len(w.Doors()) != 0 || len(w.StaticObjects()) != 0 {
		t.Error("CharSelected kept the doors")
	}
}

func TestWorldBoats(t *testing.T) {
	const boat, boat2, other = 0x60000020, 0x60000021, 0x10000002
	w := NewWorld()
	w.Apply(CharSelected{ObjectID: testMeID, X: -96622, Y: 260800, Z: -3610})
	w.Apply(VehicleInfo{ObjectID: boat, X: -96622, Y: 261660, Z: -3610, Heading: 32768})
	w.Apply(VehicleInfo{ObjectID: boat2, X: -50000, Y: 261660, Z: -3610})

	b, ok := w.NearestBoat()
	if !ok || b.ObjectID != boat || b.Moving || b.Pos != (Position{-96622, 261660, -3610}) || b.Heading != 32768 {
		t.Fatalf("NearestBoat = %+v, %v", b, ok)
	}

	// Two players board; the character's deck position is kept.
	w.Apply(GetOnVehicle{ObjectID: other, BoatID: boat, X: 10, Y: 20, Z: 30})
	w.Apply(GetOnVehicle{ObjectID: testMeID, BoatID: boat, X: -5, Y: 6, Z: -7})
	if id, deck := w.OnBoat(); id != boat || deck != (Position{-5, 6, -7}) {
		t.Errorf("OnBoat = %d, %v", id, deck)
	}
	if b, _ := w.Boat(boat); !slices.Equal(b.Passengers, []int32{testMeID, other}) {
		t.Errorf("passengers %v", b.Passengers)
	}
	w.Apply(MoveToLocationInVehicle{ObjectID: testMeID, BoatID: boat, DestX: 1, DestY: 2, DestZ: 3})
	if _, deck := w.OnBoat(); deck != (Position{1, 2, 3}) {
		t.Errorf("deck after a walk on it: %v", deck)
	}
	w.Apply(StopMoveInVehicle{ObjectID: testMeID, BoatID: boat, X: 4, Y: 5, Z: 6})
	if _, deck := w.OnBoat(); deck != (Position{4, 5, 6}) {
		t.Errorf("deck after stopping on it: %v", deck)
	}

	// The departure.
	w.Apply(VehicleStarted{ObjectID: boat, State: 1})
	w.Apply(VehicleDeparture{ObjectID: boat, MoveSpeed: 180, RotationSpeed: 800, X: -121385, Y: 261660, Z: -3610})
	b, _ = w.Boat(boat)
	if !b.Moving || b.Dest != (Position{-121385, 261660, -3610}) || b.MoveSpeed != 180 || b.RotationSpeed != 800 {
		t.Errorf("boat under way = %+v", b)
	}
	w.Apply(VehicleStarted{ObjectID: boat, State: 0})
	if b, _ := w.Boat(boat); b.Moving {
		t.Error("VehicleStarted 0 left the boat moving")
	}

	// The other player moves to the second boat; the character gets off.
	w.Apply(GetOnVehicle{ObjectID: other, BoatID: boat2})
	if b, _ := w.Boat(boat); !slices.Equal(b.Passengers, []int32{testMeID}) {
		t.Errorf("passengers of the first boat %v", b.Passengers)
	}
	w.Apply(GetOffVehicle{ObjectID: testMeID, BoatID: boat, X: -96777, Y: 258970, Z: -3623})
	if id, _ := w.OnBoat(); id != 0 || w.Me().Pos != (Position{-96777, 258970, -3623}) {
		t.Errorf("after GetOffVehicle: boat %d, at %v", id, w.Me().Pos)
	}
	if b, _ := w.Boat(boat); len(b.Passengers) != 0 {
		t.Errorf("passengers after leaving %v", b.Passengers)
	}

	// A boat out of sight is forgotten, with the character's place on it.
	w.Apply(GetOnVehicle{ObjectID: testMeID, BoatID: boat2})
	w.Apply(DeleteObject{ObjectID: boat2})
	if bs := w.Boats(); len(bs) != 1 || bs[0].ObjectID != boat {
		t.Errorf("Boats after DeleteObject = %+v", bs)
	}
	if id, _ := w.OnBoat(); id != 0 {
		t.Errorf("OnBoat after the boat left sight = %d", id)
	}
}

func TestGameClock(t *testing.T) {
	tests := []struct {
		minutes int32
		want    time.Duration
	}{
		{0, 0},
		{359, 0},
		// 06:00: 18 game hours, 3 real hours.
		{360, 18 * time.Hour / 6},
		{600, 14 * 60 * 10 * time.Second},
		{1439, 10 * time.Second},
		// The next day's 00:30.
		{1440 + 30, 0},
	}
	for _, tt := range tests {
		if got := UntilNight(tt.minutes); got != tt.want {
			t.Errorf("UntilNight(%d) = %v, want %v", tt.minutes, got, tt.want)
		}
	}

	at := time.Unix(1000, 0)
	g := GameClock{Minutes: 1430, At: at}
	if m := g.MinutesAt(at.Add(time.Minute)); m != 1436 {
		t.Errorf("one real minute after 23:50: %d, want 1436", m)
	}
	if m := g.MinutesAt(at.Add(2 * time.Minute)); m != 2 || !g.NightAt(at.Add(2*time.Minute)) {
		t.Errorf("two real minutes after 23:50: %d, want 2 (night)", m)
	}
	if g.NightAt(at) {
		t.Error("23:50 is night")
	}
	g.Passed = time.Hour // 360 game minutes
	if m := g.MinutesAt(at); m != 350 {
		t.Errorf("an hour passed: %d, want 350", m)
	}
	if m := g.MinutesAt(at.Add(-time.Hour)); m != 350 {
		t.Errorf("a time before CharSelected: %d, want 350", m)
	}
}

func TestWaitForNight(t *testing.T) {
	ctx := testContext(t)
	c := &GameConn{world: newWorld()}
	if _, err := c.WaitForNight(ctx, nil); !errors.Is(err, ErrNoGameClock) {
		t.Errorf("WaitForNight before CharSelected = %v", err)
	}

	// 10:00: 14 game hours to midnight, plus a game minute.
	c.world.Apply(CharSelected{ObjectID: testMeID, GameTime: 600})
	var passed []time.Duration
	pass := func(_ context.Context, d time.Duration) error {
		passed = append(passed, d)
		return nil
	}
	want := 14*time.Hour*10/60 + RealGameMinute
	d, err := c.WaitForNight(ctx, pass)
	if err != nil || d != want || !slices.Equal(passed, []time.Duration{want}) {
		t.Fatalf("WaitForNight = %v, %v (passed %v), want %v", d, err, passed, want)
	}
	clock, _ := c.world.GameClock()
	if !clock.NightAt(time.Now()) || clock.MinutesAt(time.Now()) > 2 {
		t.Errorf("the clock after the wait: %+v at %d", clock, clock.MinutesAt(time.Now()))
	}
	// At night it returns at once.
	if d, err := c.WaitForNight(ctx, pass); d != 0 || err != nil || len(passed) != 1 {
		t.Errorf("WaitForNight at night = %v, %v", d, err)
	}

	// A failing pass is returned and nothing counts as passed.
	c.world.Apply(CharSelected{ObjectID: testMeID, GameTime: 700})
	boom := errors.New("boom")
	if _, err := c.WaitForNight(ctx, func(context.Context, time.Duration) error { return boom }); !errors.Is(err, boom) {
		t.Errorf("WaitForNight with a failing pass = %v", err)
	}
	if clock, _ := c.world.GameClock(); clock.Passed != 0 {
		t.Errorf("passed %v after a failed pass", clock.Passed)
	}

	// Without pass it sleeps, until the context ends.
	short, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel()
	if _, err := c.WaitForNight(short, nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("WaitForNight sleeping past the context = %v", err)
	}
}

// BoardShip boards the nearest boat; a refusal is ErrBoardRefused.
func TestBoardShip(t *testing.T) {
	const boat = 0x60000020
	pos := Position{-96622, 260800, -3610}
	deck := Position{-20, 30, -5}
	info := msg(0x59).i32(boat).i32(-96622).i32(261660).i32(-3610).i32(32768).b
	addr := startFakeGame(t, func(f *fakeGame) {
		f.enterQuick(pos, testItems(), info)
		// Refused first (the boat is under way), then accepted.
		f.expect(msg(0x42).i32(boat).i32(deck.X).i32(deck.Y).i32(deck.Z).b)
		f.send(msg(0x25).b)
		f.expect(msg(0x42).i32(boat).i32(deck.X).i32(deck.Y).i32(deck.Z).b)
		f.send(msg(0x5c).i32(0x10000002).i32(boat).i32(1).i32(2).i32(3).b) // another player
		f.send(msg(0x5c).i32(testMeID).i32(boat).i32(deck.X).i32(deck.Y).i32(deck.Z).b)
		f.drain()
	})
	ctx := testContext(t)
	c := clientEnterQuick(t, ctx, addr)
	if err := pollWorld(ctx, func() bool { _, ok := c.World().NearestBoat(); return ok }); err != nil {
		t.Fatalf("no boat in sight: %v", err)
	}
	if _, err := c.BoardShip(ctx, 0, deck); !errors.Is(err, ErrBoardRefused) {
		t.Fatalf("BoardShip refused = %v, want ErrBoardRefused", err)
	}
	g, err := c.BoardShip(ctx, boat, deck)
	if err != nil || g != (GetOnVehicle{ObjectID: testMeID, BoatID: boat, X: deck.X, Y: deck.Y, Z: deck.Z}) {
		t.Fatalf("BoardShip = %+v, %v", g, err)
	}
	if id, d := c.World().OnBoat(); id != boat || d != deck {
		t.Errorf("OnBoat = %d, %v", id, d)
	}

	empty := &GameConn{world: newWorld()}
	if _, err := empty.BoardShip(ctx, 0, deck); !errors.Is(err, ErrNoBoat) {
		t.Errorf("BoardShip without a boat = %v, want ErrNoBoat", err)
	}
}
