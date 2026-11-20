package data

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

func validObjects() ObjectsData {
	door := func(id int) Door {
		d := Door{ID: id, Name: "door", X: 10, Y: 20, Z: -100, NodeZ: -120, Height: 200, HP: 1000}
		d.Nodes = [4]Point{{0, 0}, {100, 5}, {100, 10}, {0, 10}}
		d.ShowHP, d.Targetable, d.CheckCollision = true, true, true
		return d
	}
	timed := door(3)
	timed.OpenMethod = DoorOpenByTime | DoorOpenBySkill
	timed.CloseTime, timed.OpenTime, timed.RandomTime = 20, 300, 120
	timed.ChildID = 1
	return ObjectsData{
		Doors:         []Door{door(1), door(2), timed},
		StaticObjects: []StaticObject{{ID: 5, Name: "map", Type: StaticTownMap, Texture: "t"}, {ID: 6, Name: "board", Type: StaticSignboard}},
		Fences:        []Fence{{Name: "demo", Width: 100, Length: 100, Height: 3, State: FenceClosed}},
	}
}

func TestNewObjects(t *testing.T) {
	d := validObjects()
	d.Doors[0].Group, d.Doors[1].Group = "g", "g"
	o, err := NewObjects(d)
	if err != nil {
		t.Fatal(err)
	}
	if dr, ok := o.Door(3); !ok || dr.OpenTime != 300 {
		t.Errorf("Door(3) = %+v, %v", dr, ok)
	}
	if _, ok := o.Door(4); ok {
		t.Error("Door(4) found")
	}
	if g := o.DoorGroup("g"); !slices.Equal(g, []int{1, 2}) {
		t.Errorf("DoorGroup(g) = %v", g)
	}
	if g := o.DoorGroups(); !slices.Equal(g, []string{"g"}) {
		t.Errorf("DoorGroups = %v", g)
	}
	if s, ok := o.StaticObject(6); !ok || s.Name != "board" {
		t.Errorf("StaticObject(6) = %+v, %v", s, ok)
	}
	if _, ok := o.StaticObject(7); ok {
		t.Error("StaticObject(7) found")
	}
}

func TestDoorMethods(t *testing.T) {
	d := validObjects().Doors[0]
	if d.SpawnZ() != -120 || d.ZMin() != -120 || d.ZMax() != 80 {
		t.Errorf("SpawnZ %d, ZMin %d, ZMax %d; want -120, -120, 80", d.SpawnZ(), d.ZMin(), d.ZMax())
	}
	d.Z = -200
	if d.SpawnZ() != -200 {
		t.Errorf("SpawnZ = %d, want the pos Z -200 below NodeZ", d.SpawnZ())
	}
	// min(|0-100|, |0-5|) = 5, raised to 20.
	if r := d.CollisionRadius(); r != DoorMinCollisionRadius {
		t.Errorf("CollisionRadius = %d, want %d", r, DoorMinCollisionRadius)
	}
	d.Nodes[1] = Point{-50, 60}
	if r := d.CollisionRadius(); r != 50 {
		t.Errorf("CollisionRadius = %d, want 50", r)
	}
	if d.AutoCloses() {
		t.Error("a door without close time auto-closes")
	}
	d.CloseTime = 10
	if !d.AutoCloses() {
		t.Error("a door with a close time does not auto-close")
	}
	d.ClanHallID = 28
	if d.AutoCloses() {
		t.Error("a clan hall door auto-closes")
	}
	d.ClanHallID, d.OpenMethod = 0, DoorOpenByTime|DoorOpenByClick
	if d.AutoCloses() || !d.OpensBy(DoorOpenByClick) || d.OpensBy(DoorOpenBySkill) {
		t.Error("a timed door auto-closes, or OpensBy is wrong")
	}
	for state, want := range map[string]int{FenceHidden: 0, FenceOpened: 1, FenceClosed: 2, FenceClosedHidden: 0} {
		if got := FenceClientState(state); got != want {
			t.Errorf("FenceClientState(%s) = %d, want %d", state, got, want)
		}
	}
}

func TestNewObjectsErrors(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(d *ObjectsData)
		want   string
	}{
		{"duplicate door", func(d *ObjectsData) { d.Doors[1].ID = 1 }, "door 1: duplicate ID"},
		{"unsorted doors", func(d *ObjectsData) { d.Doors[0].ID = 9 }, "door 2: doors are not sorted"},
		{"no name", func(d *ObjectsData) { d.Doors[0].Name = "" }, "door 1: no name"},
		{"height", func(d *ObjectsData) { d.Doors[0].Height = 0 }, "height 0 must be positive"},
		{"hp", func(d *ObjectsData) { d.Doors[0].HP = 0 }, "HP 0 must be positive"},
		{"open method", func(d *ObjectsData) { d.Doors[0].OpenMethod = 32 }, "unknown open method bits 32"},
		{"action", func(d *ObjectsData) { d.Doors[0].MasterOpen = "act_flip" }, `unknown master action "act_flip"`},
		{"negative time", func(d *ObjectsData) { d.Doors[0].CloseTime = -1 }, "door 1: negative time"},
		{"untimed open time", func(d *ObjectsData) { d.Doors[0].OpenTime = 5 }, "open or random time on a door that does not open by time"},
		{"child", func(d *ObjectsData) { d.Doors[2].ChildID = 7 }, "child door 7 does not exist"},
		{"own child", func(d *ObjectsData) { d.Doors[2].ChildID = 3 }, "the door is its own child"},
		{"static order", func(d *ObjectsData) { d.StaticObjects[1].ID = 5 }, "static object 5: IDs are not sorted and unique"},
		{"static type", func(d *ObjectsData) { d.StaticObjects[0].Type = 3 }, "unknown type 3"},
		{"fence state", func(d *ObjectsData) { d.Fences[0].State = "closed" }, `unknown state "closed"`},
		{"fence size", func(d *ObjectsData) { d.Fences[0].Height = 0 }, "size 100x100x0 must be positive"},
	} {
		d := validObjects()
		tt.change(&d)
		_, err := NewObjects(d)
		if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: %v, want an error mentioning %q", tt.name, err, tt.want)
		}
	}
}

func TestLoadObjectsFS(t *testing.T) {
	fsys := fstest.MapFS{
		DoorsFile:         {Data: []byte(`[{"id":1,"name":"d","x":0,"y":0,"z":0,"nodes":[{"x":0,"y":0},{"x":1,"y":0},{"x":1,"y":1},{"x":0,"y":1}],"nodeZ":0,"height":10,"hp":1,"pDef":0,"mDef":0,"showHp":true,"targetable":true,"checkCollision":true}]`)},
		StaticObjectsFile: {Data: []byte(`[]`)},
		FencesFile:        {Data: []byte(`[]`)},
	}
	if o, err := LoadObjectsFS(fsys); err != nil || len(o.Doors) != 1 {
		t.Fatalf("LoadObjectsFS: %v", err)
	}
	fsys[FencesFile] = &fstest.MapFile{Data: []byte(`[{"name":"f","bad":1}]`)}
	if _, err := LoadObjectsFS(fsys); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Errorf("unknown field: %v", err)
	}
	if _, err := LoadObjects(t.TempDir()); err == nil || !strings.Contains(err.Error(), "data folder") {
		t.Errorf("LoadObjects of an empty folder: %v", err)
	}
}

// loadRepoObjects loads the committed world objects.
func loadRepoObjects(t *testing.T) *Objects {
	t.Helper()
	if _, err := os.Stat(filepath.Join(repoData, DoorsFile)); err != nil {
		t.Skipf("no object data in %s: %v", repoData, err)
	}
	o, err := LoadObjects(repoData)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

// Spot checks of the committed objects against Mobius Doors.xml,
// StaticObjects.xml and FenceData.xml.
func TestRepoObjects(t *testing.T) {
	o := loadRepoObjects(t)
	open, timed, grouped := 0, 0, 0
	for i := range o.Doors {
		d := &o.Doors[i]
		if d.DefaultOpen {
			open++
		}
		if d.OpensBy(DoorOpenByTime) {
			timed++
		}
		if d.Group != "" {
			grouped++
		}
	}
	// The Interlude datapack of 2026-11: 547 doors, 26 open at start, 7
	// doors in 3 groups; 29 have the open-by-time bit (Devil's Isle 2,
	// Tower of Insolence 3, and the 24 Sea of Spores doors of open_method
	// 10, skill and time).
	if len(o.Doors) != 547 || open != 26 || grouped != 7 || len(o.DoorGroups()) != 3 || timed != 29 {
		t.Errorf("%d doors, %d open, %d grouped in %d groups, %d timed; want 547, 26, 7 in 3, 29",
			len(o.Doors), open, grouped, len(o.DoorGroups()), timed)
	}
	if len(o.StaticObjects) != 29 || len(o.Fences) != 1 {
		t.Errorf("%d static objects, %d fences; want 29, 1", len(o.StaticObjects), len(o.Fences))
	}

	// <door id="19160001" name="temple_gate" open_method="8" level="3" close_time="10" height="256"
	// baseHpMax="126600" hp_showable="false" basePDef="644" baseMDef="518" pos="-16363,-36798,-10535"
	// node1="-16647,-36862" node2="-16088,-36860" node3="-16081,-36733" node4="-16649,-36733" nodeZ="-10790"
	// default_status="close" />
	d, ok := o.Door(19160001)
	if !ok {
		t.Fatal("door 19160001 not found")
	}
	want := Door{
		ID: 19160001, Name: "temple_gate", X: -16363, Y: -36798, Z: -10535,
		Nodes: [4]Point{{-16647, -36862}, {-16088, -36860}, {-16081, -36733}, {-16649, -36733}},
		NodeZ: -10790, Height: 256, HP: 126600, PDef: 644, MDef: 518, Level: 3,
		OpenMethod: DoorOpenBySkill, Targetable: true, CloseTime: 10, CheckCollision: true,
	}
	if *d != want {
		t.Errorf("door 19160001 =\n%+v\nwant\n%+v", *d, want)
	}
	if d.SpawnZ() != -10790 || d.ZMax() != -10534 || d.CollisionRadius() != 20 || !d.AutoCloses() {
		t.Errorf("door 19160001: SpawnZ %d, ZMax %d, radius %d, auto-closes %v", d.SpawnZ(), d.ZMax(), d.CollisionRadius(), d.AutoCloses())
	}
	// hubris_3F_001: open_method="2" close_time="120" open_time="120" random_time="120"
	// child_id_event="23180002"; its child opens and closes with it.
	h, _ := o.Door(23180001)
	c, _ := o.Door(23180002)
	if h == nil || c == nil || h.ChildID != 23180002 || h.OpenTime != 120 || h.RandomTime != 120 || h.CloseTime != 120 ||
		h.AutoCloses() || c.MasterOpen != DoorActOpen || c.MasterClose != DoorActClose {
		t.Errorf("hubris doors = %+v, %+v", h, c)
	}
	// A clan hall door: gludin_agit_003_003, clanhall_id="28".
	if g, ok := o.Door(17220001); !ok || g.ClanHallID != 28 || !g.OpensBy(DoorOpenByClick) || g.ShowHP {
		t.Errorf("door 17220001 = %+v, %v", g, ok)
	}
	if g := o.DoorGroup("sanctuary_parent"); !slices.Equal(g, []int{19160010, 19160011}) {
		t.Errorf("DoorGroup(sanctuary_parent) = %v", g)
	}

	// <object id="17250001" name="town_map_talking_1" type="0" x="-82873" y="244808" z="-3717"
	// texture="town_map_talking_t00" map_x="365" map_y="152" />
	if s, ok := o.StaticObject(17250001); !ok || *s != (StaticObject{ID: 17250001, Name: "town_map_talking_1", Type: StaticTownMap,
		X: -82873, Y: 244808, Z: -3717, Texture: "town_map_talking_t00", MapX: 365, MapY: 152}) {
		t.Errorf("StaticObject(17250001) = %+v, %v", s, ok)
	}
	if s, ok := o.StaticObject(22220001); !ok || s.Type != StaticSignboard || s.Texture != "" {
		t.Errorf("StaticObject(22220001) = %+v, %v", s, ok)
	}
	// <fence name="demo" x="-114552" y="-251256" z="-2992" width="100" length="100" height="3" state="CLOSED" />
	if f := o.Fences[0]; f != (Fence{Name: "demo", X: -114552, Y: -251256, Z: -2992, Width: 100, Length: 100, Height: 3, State: FenceClosed}) {
		t.Errorf("fence = %+v", f)
	}
}
