package game

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

func TestZoneRules(t *testing.T) {
	zone := func(typ string, params ...string) *data.Zone {
		z := &data.Zone{ID: 1, Type: typ}
		for i := 0; i+1 < len(params); i += 2 {
			z.Params = append(z.Params, data.ZoneParam{Name: params[i], Value: params[i+1]})
		}
		return z
	}
	const humanFighter, humanMystic, human = 0, 10, 0
	for _, tc := range []struct {
		name   string
		zone   *data.Zone
		flags  []ZoneFlag
		water  bool
		swamp  float64
		danger bool
		// affected: whether a level 20 human fighter and a level 20 human
		// mystic are affected.
		fighter, mystic bool
	}{
		{"water", zone("WaterZone"), []ZoneFlag{ZoneWater}, true, 0, false, true, true},
		{"peace", zone("PeaceZone"), []ZoneFlag{ZonePeace}, false, 0, false, true, true},
		{"peace without stores", zone("PeaceZone", "allowStore", "false"), []ZoneFlag{ZonePeace, ZoneNoStore}, false, 0, false, true, true},
		{"town", zone("TownZone", "townId", "1", "taxById", "1"), []ZoneFlag{ZoneTown}, false, 0, false, true, true},
		{"no store", zone("NoStoreZone"), []ZoneFlag{ZoneNoStore}, false, 0, false, true, true},
		{"arena", zone("ArenaZone"), []ZoneFlag{ZonePvP}, false, 0, false, true, true},
		{"jail", zone("JailZone"), []ZoneFlag{ZoneJail, ZoneNoSummonFriend}, false, 0, false, true, true},
		{"swamp", zone("SwampZone"), []ZoneFlag{ZoneSwamp}, false, 0.5, false, true, true},
		{"swamp with a bonus", zone("SwampZone", "move_bonus", "0.7"), []ZoneFlag{ZoneSwamp}, false, 0.7, false, true, true},
		{"castle swamp outside sieges", zone("SwampZone", "castleId", "3"), nil, false, 0, false, true, true},
		{"siege zone outside sieges", zone("SiegeZone", "castleId", "3"), nil, false, 0, false, true, true},
		{"effect zone", zone("EffectZone", "chance", "100"), []ZoneFlag{ZoneAltered, ZoneDangerArea}, false, 0, true, true, true},
		{"effect zone without the icon", zone("EffectZone", "showDangerIcon", "false"), []ZoneFlag{ZoneAltered}, false, 0, false, true, true},
		{"effect zone for NPCs", zone("EffectZone", "targetClass", "Npc"), []ZoneFlag{ZoneAltered, ZoneDangerArea}, false, 0, true, false, false},
		{"no bookmark", zone("ConditionZone", "NoBookmark", "true"), []ZoneFlag{ZoneNoBookmark}, false, 0, false, true, true},
		{"olympiad", zone("OlympiadStadiumZone"), []ZoneFlag{ZonePvP, ZoneNoSummonFriend, ZoneNoLanding, ZoneNoRestart, ZoneNoBookmark}, false, 0, false, true, true},
		{"disabled", zone("ArenaZone", "default_enabled", "false"), []ZoneFlag{ZonePvP}, false, 0, false, false, false},
		{"high level", zone("NoPvPZone", "affectedLvlMin", "40"), []ZoneFlag{ZoneNoPvP}, false, 0, false, false, false},
		{"elves only", zone("MotherTreeZone", "affectedRace", "1"), []ZoneFlag{ZoneMotherTree}, false, 0, false, false, false},
		{"humans and elves", zone("MotherTreeZone", "affectedRace", "1", "affectedRace", "0"), []ZoneFlag{ZoneMotherTree}, false, 0, false, true, true},
		{"fighters", zone("ScriptZone", "affectedClassType", "Fighter"), []ZoneFlag{ZoneScript}, false, 0, false, true, false},
		{"mystics", zone("ScriptZone", "affectedClassType", "Mystic"), []ZoneFlag{ZoneScript}, false, 0, false, false, true},
		{"one class", zone("HqZone", "affectedClassId", "10"), []ZoneFlag{ZoneHQ}, false, 0, false, false, true},
		{"NPCs only", zone("NoStoreZone", "targetClass", "Npc"), []ZoneFlag{ZoneNoStore}, false, 0, false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newZoneRule(tc.zone)
			if !slices.Equal(r.flags, tc.flags) || r.water != tc.water || r.swamp != tc.swamp || r.danger != tc.danger {
				t.Errorf("rule flags %v water %v swamp %v danger %v, want %v %v %v %v", r.flags, r.water, r.swamp, r.danger, tc.flags, tc.water, tc.swamp, tc.danger)
			}
			fighter := !r.check || r.affects(20, humanFighter, human)
			mystic := !r.check || r.affects(20, humanMystic, human)
			if fighter != tc.fighter || mystic != tc.mystic {
				t.Errorf("affects a fighter %v, a mystic %v; want %v, %v", fighter, mystic, tc.fighter, tc.mystic)
			}
		})
	}
}

// pointIn returns a point inside z: a point of a grid over its bounds that
// the shape contains, at the middle of its height.
func pointIn(t *testing.T, z *data.Zone) world.Position {
	t.Helper()
	minX, minY, maxX, maxY := z.Shape.Bounds()
	zMid := (z.Shape.LowZ() + z.Shape.HighZ()) / 2
	for i := 1; i < 20; i++ {
		for j := 1; j < 20; j++ {
			x, y := minX+(maxX-minX)*i/20, minY+(maxY-minY)*j/20
			if z.Shape.Contains(x, y, zMid) {
				return world.Position{X: int32(x), Y: int32(y), Z: int32(zMid)}
			}
		}
	}
	t.Fatalf("no point found inside zone %d", z.ID)
	return world.Position{}
}

// firstZone returns the first zone of the world data of type typ with an
// NPoly shape.
func firstZone(t *testing.T, typ string) *data.Zone {
	t.Helper()
	w := testWorldData(t)
	for i := range w.Zones {
		if z := &w.Zones[i]; z.Type == typ && z.Shape.Type == data.ShapeNPoly {
			return z
		}
	}
	t.Fatalf("no %s in the world data", typ)
	return nil
}

func TestZoneIndexFindsZones(t *testing.T) {
	w := testWorldData(t)
	ix := newZoneIndex(w)
	for _, typ := range []string{"WaterZone", "TownZone", "PeaceZone"} {
		z := firstZone(t, typ)
		pos := pointIn(t, z)
		var found []int
		ix.at(pos, func(r *zoneRule) bool { found = append(found, r.zone.ID); return true })
		if !slices.Contains(found, z.ID) {
			t.Errorf("zones at %v of %s %d: %v", pos, typ, z.ID, found)
		}
		// Every zone found contains the point.
		for _, id := range found {
			if zz, _ := w.Zone(id); !zz.Shape.Contains(int(pos.X), int(pos.Y), int(pos.Z)) {
				t.Errorf("zone %d found at %v does not contain it", id, pos)
			}
		}
	}
}

func TestWaterZoneSwimSpeed(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{})
	water := firstZone(t, "WaterZone")
	inWater := pointIn(t, water)
	sess, p := e.enterChar(testChar{account: "hero", name: "Hero", pos: tiStart})

	move := func(pos world.Position) testclient.UserInfo {
		t.Helper()
		e.onLoop(func() {
			if err := e.srv.MoveObject(p, pos); err != nil {
				t.Errorf("MoveObject: %v", err)
			}
		})
		e.advance(e.srv.Settings().TickInterval)
		uis := only[testclient.UserInfo](sess.take(t))
		if len(uis) == 0 {
			t.Fatalf("no UserInfo after moving to %v", pos)
		}
		return uis[len(uis)-1]
	}

	ui := move(inWater)
	var inside, swimming bool
	var ids []int
	var speed, swimSpeed float64
	e.onLoop(func() {
		inside = p.Zones.Inside(ZoneWater)
		swimming = p.Stats().Motion().InWater
		ids = slices.Clone(p.Zones.IDs())
		speed, swimSpeed = p.Stats().MoveSpeed(), p.Stats().SwimRunSpeed()
	})
	if !inside || !swimming || !slices.Contains(ids, water.ID) {
		t.Errorf("in the water: flag %v, swimming %v, zones %v", inside, swimming, ids)
	}
	if speed != swimSpeed {
		t.Errorf("move speed in the water %v, want the swim run speed %v", speed, swimSpeed)
	}
	if !ui.InWater {
		t.Error("UserInfo in the water does not say so")
	}

	ui = move(tiStart)
	e.onLoop(func() {
		inside = p.Zones.Inside(ZoneWater)
		swimming = p.Stats().Motion().InWater
		ids = slices.Clone(p.Zones.IDs())
	})
	if inside || swimming || slices.Contains(ids, water.ID) {
		t.Errorf("out of the water: flag %v, swimming %v, zones %v", inside, swimming, ids)
	}
	if ui.InWater {
		t.Error("UserInfo out of the water still says in water")
	}
}

func TestZonesOnEnterAndAfterTeleport(t *testing.T) {
	w := testWorldData(t)
	town, ok := w.Zone(11022) // Talking Island
	if !ok || town.Type != "TownZone" || !town.Shape.Contains(int(tiVillage.X), int(tiVillage.Y), int(tiVillage.Z)) {
		t.Fatalf("the Talking Island town zone does not hold %v", tiVillage)
	}
	e := newDialogEnv(t, dialogEnvOptions{})
	sess, p := e.enterChar(testChar{account: "hero", name: "Hero", pos: tiVillage})
	var inTown bool
	e.onLoop(func() { inTown = p.Zones.Inside(ZoneTown) && p.Zones.In(town.ID) })
	if !inTown {
		t.Error("entering the world in the village: not in the town zone")
	}

	// A teleport into the water: the zones change once the player appears.
	water := firstZone(t, "WaterZone")
	dest := pointIn(t, water)
	dest.Z -= teleportZOffset
	e.onLoop(func() {
		if err := e.srv.Teleport(p, dest); err != nil {
			t.Errorf("Teleport: %v", err)
		}
	})
	e.srv.Handle(sess, clientpackets.OpAppearing, nil)
	e.sync()
	var inWater bool
	e.onLoop(func() {
		inTown = p.Zones.Inside(ZoneTown)
		inWater = p.Zones.Inside(ZoneWater) && p.Stats().Motion().InWater
	})
	if inTown || !inWater {
		t.Errorf("after the teleport: in town %v, in water %v; want false, true", inTown, inWater)
	}
}

func TestZonesWaitForAHundredUnits(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{})
	water := firstZone(t, "WaterZone")
	pos := pointIn(t, water)
	_, p := e.enterChar(testChar{account: "hero", name: "Hero", pos: pos})
	in := func() bool {
		var ok bool
		e.onLoop(func() { ok = p.Zones.In(water.ID) })
		return ok
	}
	if !in() {
		t.Fatal("not in the water zone after entering the world in it")
	}
	// Forget the zone behind the feature's back: small moves do not look
	// at the zones again, a move of 100 units does.
	e.onLoop(func() {
		p.Zones.Remove(water.ID)
		_ = e.srv.MoveObject(p, world.Position{X: pos.X, Y: pos.Y, Z: pos.Z + 1})
	})
	if in() {
		t.Error("a 1-unit move worked out the zones again")
	}
	e.onLoop(func() {
		e.srv.revalidateZones(p, true)
	})
	if !in() {
		t.Error("a forced revalidation did not find the zone")
	}
}
