package data

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

// sampleSevenSigns is the smallest valid Seven Signs data.
func sampleSevenSigns() SevenSigns {
	var d SevenSigns
	for typ := RiftTypeWaiting; typ <= RiftTypeHero; typ++ {
		for room := range 2 {
			d.Rift = append(d.Rift, RiftRoom{Type: typ, Room: room * 9, XMin: 0, XMax: 100, YMin: 0, YMax: 100, ZMin: -10, ZMax: 10,
				Boss: room == 1, Spawns: []RiftSpawn{{NpcID: 25339, Count: 1, Delay: 300}}})
		}
	}
	spawns := func(npc int) []FestivalSpawn {
		out := make([]FestivalSpawn, FestivalCount)
		for i := range out {
			out[i] = FestivalSpawn{X: i, NpcID: npc}
		}
		return out
	}
	lists := func() [][]FestivalSpawn {
		out := make([][]FestivalSpawn, FestivalCount)
		for i := range out {
			out[i] = []FestivalSpawn{{X: i, Heading: -1, NpcID: 18009}}
		}
		return out
	}
	d.Festival = Festival{DawnPlayer: spawns(0), DuskPlayer: spawns(0), DawnWitch: spawns(31132), DuskWitch: spawns(31142),
		DawnPrimary: lists(), DuskPrimary: lists(), DawnSecondary: lists(), DuskSecondary: lists(), DawnChest: lists(), DuskChest: lists()}
	d.AutoSpawns = []AutoSpawn{{GroupID: 11, NpcID: 31113, Count: 1, InitialDelay: -1, RespawnDelay: -1, DespawnDelay: -1,
		Locations: []AutoSpawnLoc{{X: 1, Y: 2, Z: 3, Heading: -1}}}}
	return d
}

func TestSevenSignsCheck(t *testing.T) {
	d := sampleSevenSigns()
	if err := d.Check(); err != nil {
		t.Fatalf("Check: %v", err)
	}
	if r, ok := d.RiftRoom(3, 9); !ok || !r.Boss {
		t.Errorf("RiftRoom(3, 9) = %+v, %v", r, ok)
	}
	if _, ok := d.RiftRoom(3, 5); ok {
		t.Error("RiftRoom(3, 5) found")
	}
	if rooms := d.RiftRooms(0); len(rooms) != 2 || rooms[0].Room != 0 || rooms[1].Room != 9 {
		t.Errorf("RiftRooms(0) = %+v", rooms)
	}
	if l := d.AutoSpawnsOf(31113); len(l) != 1 || l[0].GroupID != 11 {
		t.Errorf("AutoSpawnsOf = %+v", l)
	}
	r := RiftRoom{XMin: 0, XMax: 100, YMin: 0, YMax: 100, ZMin: -10, ZMax: 10}
	for _, c := range []struct {
		x, y, z int
		in      bool
	}{{0, 0, 0, true}, {99, 99, 10, true}, {100, 50, 0, false}, {50, 100, 0, false}, {50, 50, 11, false}, {-1, 50, 0, false}} {
		if got := r.Contains(c.x, c.y, c.z); got != c.in {
			t.Errorf("Contains(%d, %d, %d) = %v", c.x, c.y, c.z, got)
		}
	}

	for _, c := range []struct {
		name string
		edit func(d *SevenSigns)
		want string
	}{
		{"no hero rooms", func(d *SevenSigns) { d.Rift = d.Rift[:len(d.Rift)-2] }, "no rift rooms of type 6"},
		{"unsorted", func(d *SevenSigns) { d.Rift[0], d.Rift[1] = d.Rift[1], d.Rift[0] }, "must be sorted"},
		{"bad box", func(d *SevenSigns) { d.Rift[3].XMax = -5 }, "bad type or box"},
		{"bad spawn", func(d *SevenSigns) { d.Rift[3].Spawns[0].Count = 0 }, "bad spawn"},
		{"four witches", func(d *SevenSigns) { d.Festival.DuskWitch = d.Festival.DuskWitch[:4] }, "duskWitch has 4 spawns"},
		{"a player spawn with an NPC", func(d *SevenSigns) { d.Festival.DawnPlayer[1].NpcID = 5 }, "dawnPlayer: bad NPC"},
		{"empty chests", func(d *SevenSigns) { d.Festival.DuskChest[2] = nil }, "duskChest 2 has no spawns"},
		{"a foreign NPC", func(d *SevenSigns) { d.AutoSpawns[0].NpcID = 32014 }, "not a Seven Signs NPC"},
		{"no locations", func(d *SevenSigns) { d.AutoSpawns[0].Locations = nil }, "no count or locations"},
	} {
		t.Run(c.name, func(t *testing.T) {
			d := sampleSevenSigns()
			c.edit(&d)
			err := d.Check()
			if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), c.want) {
				t.Errorf("Check = %v, want %q", err, c.want)
			}
		})
	}
}

func TestLoadSevenSigns(t *testing.T) {
	if _, err := LoadSevenSignsFS(fstest.MapFS{}); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing file: error %v does not wrap fs.ErrNotExist", err)
	}
	dir := t.TempDir()
	if _, err := LoadSevenSigns(dir); err == nil || !strings.Contains(err.Error(), dir) || !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error %v: want it to name %s and wrap fs.ErrNotExist", err, dir)
	}
	bad := fstest.MapFS{SevenSignsFile: {Data: []byte(`{"rift":[],"festival":{},"autoSpawns":[]}`)}}
	if _, err := LoadSevenSignsFS(bad); !errors.Is(err, ErrInvalid) {
		t.Errorf("empty data: %v", err)
	}
	// The committed file loads.
	d, err := LoadSevenSigns("../../data")
	if errors.Is(err, fs.ErrNotExist) {
		t.Skip("no committed Seven Signs data")
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Rift) != 56 || len(d.Festival.DawnChest) != FestivalCount || len(d.AutoSpawnsOf(31113)) != 1 {
		t.Errorf("committed data: %d rooms, %d chest lists", len(d.Rift), len(d.Festival.DawnChest))
	}
}
