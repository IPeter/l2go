package main

import (
	"fmt"
	"io/fs"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/IPeter/l2go/internal/data"
)

// Sources of the sevensigns domain (Plan 14 M11l, decision X6).
const (
	dimensionalRiftSQL = "dist/db_installer/sql/game/dimensional_rift.sql"
	dimensionalRiftXML = "dist/game/data/DimensionalRift.xml"
	randomSpawnSQL     = "dist/db_installer/sql/game/random_spawn.sql"
	randomSpawnLocSQL  = "dist/db_installer/sql/game/random_spawn_loc.sql"
	festivalJava       = javaRoot + "mechanics/sevensigns/SevenSignsFestival.java"
)

// The sevensigns domain converts the Dimensional Rift's rooms and spawns,
// the Festival of Darkness spawn tables (Java-only, read from the source)
// and the random spawns of the Seven Signs NPCs; the game loads them with
// data.LoadSevenSigns.
func init() { registerDomain("sevensigns", genSevenSigns) }

func genSevenSigns(o options) error {
	var d data.SevenSigns
	var err error
	if d.Rift, err = readRiftRooms(o.src); err != nil {
		return err
	}
	if d.Festival, err = readFestivalSpawns(o.src); err != nil {
		return err
	}
	if d.AutoSpawns, err = readAutoSpawns(o.src); err != nil {
		return err
	}
	if err := d.Check(); err != nil {
		return err
	}
	if err := o.out.addJSON(data.SevenSignsFile, d); err != nil {
		return err
	}
	spawns, chests := 0, 0
	for _, r := range d.Rift {
		spawns += len(r.Spawns)
	}
	for _, l := range append(slices.Clone(d.Festival.DawnChest), d.Festival.DuskChest...) {
		chests += len(l)
	}
	fmt.Fprintf(o.log, "sevensigns: %d rift rooms with %d spawn lines, %d festival chests, %d auto spawn groups\n",
		len(d.Rift), spawns, chests, len(d.AutoSpawns))
	return nil
}

// readRiftRooms reads dimensional_rift.sql and the spawns of
// DimensionalRift.xml (DimensionalRiftManager.loadRooms and loadSpawns).
func readRiftRooms(fsys fs.FS) ([]data.RiftRoom, error) {
	t, err := readSQLTable(fsys, dimensionalRiftSQL, "dimensional_rift")
	if err != nil {
		return nil, err
	}
	var rooms []data.RiftRoom
	for _, r := range t.rows {
		var room data.RiftRoom
		for _, f := range []struct {
			col string
			dst *int
		}{
			{"type", &room.Type}, {"room_id", &room.Room}, {"xMin", &room.XMin}, {"xMax", &room.XMax}, {"yMin", &room.YMin},
			{"yMax", &room.YMax}, {"zMin", &room.ZMin}, {"zMax", &room.ZMax}, {"xT", &room.TeleX}, {"yT", &room.TeleY}, {"zT", &room.TeleZ},
		} {
			if *f.dst, err = r.int(f.col); err != nil {
				return nil, err
			}
		}
		boss, err := r.int("boss")
		if err != nil {
			return nil, err
		}
		room.Boss = boss > 0
		rooms = append(rooms, room)
	}
	sort.SliceStable(rooms, func(i, j int) bool {
		if rooms[i].Type != rooms[j].Type {
			return rooms[i].Type < rooms[j].Type
		}
		return rooms[i].Room < rooms[j].Room
	})
	root, err := readXMLTree(fsys, dimensionalRiftXML)
	if err != nil {
		return nil, err
	}
	if err := root.checkChildren("area"); err != nil {
		return nil, err
	}
	for _, area := range root.children {
		if err := area.checkAttrs("type"); err != nil {
			return nil, err
		}
		if err := area.checkChildren("room"); err != nil {
			return nil, err
		}
		typ, err := area.intAttr("type")
		if err != nil {
			return nil, err
		}
		for _, rn := range area.children {
			if err := rn.checkAttrs("id"); err != nil {
				return nil, err
			}
			if err := rn.checkChildren("spawn"); err != nil {
				return nil, err
			}
			id, err := rn.intAttr("id")
			if err != nil {
				return nil, err
			}
			i := slices.IndexFunc(rooms, func(r data.RiftRoom) bool { return r.Type == typ && r.Room == id })
			if i < 0 {
				// Mobius logs an error for every spawn of an unknown room.
				return nil, rn.errorf("room %d/%d is not in %s", typ, id, dimensionalRiftSQL)
			}
			for _, sn := range rn.children {
				if err := sn.checkAttrs("mobId", "count", "delay"); err != nil {
					return nil, err
				}
				if err := sn.checkChildren(); err != nil {
					return nil, err
				}
				var s data.RiftSpawn
				if s.NpcID, err = sn.intAttr("mobId"); err != nil {
					return nil, err
				}
				if s.Count, err = sn.intAttr("count"); err != nil {
					return nil, err
				}
				if s.Delay, err = sn.intAttr("delay"); err != nil {
					return nil, err
				}
				rooms[i].Spawns = append(rooms[i].Spawns, s)
			}
		}
	}
	return rooms, nil
}

// readFestivalSpawns reads the FESTIVAL_*_SPAWNS initializers of
// SevenSignsFestival.java.
func readFestivalSpawns(fsys fs.FS) (data.Festival, error) {
	var f data.Festival
	b, err := fs.ReadFile(fsys, festivalJava)
	if err != nil {
		return f, fmt.Errorf("read %s: %w", festivalJava, err)
	}
	src := stripJavaComments(string(b))
	flat := []struct {
		name string
		dst  *[]data.FestivalSpawn
		npc  bool
	}{
		{"FESTIVAL_DAWN_PLAYER_SPAWNS", &f.DawnPlayer, false}, {"FESTIVAL_DUSK_PLAYER_SPAWNS", &f.DuskPlayer, false},
		{"FESTIVAL_DAWN_WITCH_SPAWNS", &f.DawnWitch, true}, {"FESTIVAL_DUSK_WITCH_SPAWNS", &f.DuskWitch, true},
	}
	for _, l := range flat {
		tree, err := javaArrayInit(src, l.name)
		if err != nil {
			return f, fmt.Errorf("%s: %w", festivalJava, err)
		}
		if *l.dst, err = festivalSpawnList(tree, l.npc); err != nil {
			return f, fmt.Errorf("%s: %s: %w", festivalJava, l.name, err)
		}
	}
	nested := []struct {
		name string
		dst  *[][]data.FestivalSpawn
	}{
		{"FESTIVAL_DAWN_PRIMARY_SPAWNS", &f.DawnPrimary}, {"FESTIVAL_DUSK_PRIMARY_SPAWNS", &f.DuskPrimary},
		{"FESTIVAL_DAWN_SECONDARY_SPAWNS", &f.DawnSecondary}, {"FESTIVAL_DUSK_SECONDARY_SPAWNS", &f.DuskSecondary},
		{"FESTIVAL_DAWN_CHEST_SPAWNS", &f.DawnChest}, {"FESTIVAL_DUSK_CHEST_SPAWNS", &f.DuskChest},
	}
	for _, l := range nested {
		tree, err := javaArrayInit(src, l.name)
		if err != nil {
			return f, fmt.Errorf("%s: %w", festivalJava, err)
		}
		for i, sub := range tree.list {
			if sub.list == nil {
				return f, fmt.Errorf("%s: %s[%d] is not a list", festivalJava, l.name, i)
			}
			spawns, err := festivalSpawnList(sub, true)
			if err != nil {
				return f, fmt.Errorf("%s: %s[%d]: %w", festivalJava, l.name, i, err)
			}
			*l.dst = append(*l.dst, spawns)
		}
	}
	return f, nil
}

// festivalSpawnList converts a list of int rows: x, y, z, heading and, with
// npc, the NPC ID.
func festivalSpawnList(tree javaInit, npc bool) ([]data.FestivalSpawn, error) {
	want := 4
	if npc {
		want = 5
	}
	out := []data.FestivalSpawn{}
	for i, row := range tree.list {
		nums, ok := row.ints()
		if !ok || len(nums) != want {
			return nil, fmt.Errorf("row %d: want %d ints", i, want)
		}
		s := data.FestivalSpawn{X: nums[0], Y: nums[1], Z: nums[2], Heading: nums[3]}
		if npc {
			s.NpcID = nums[4]
		}
		out = append(out, s)
	}
	return out, nil
}

// javaInit is a Java array initializer: a number, or a list of them.
type javaInit struct {
	num  int
	list []javaInit // nil for a number
}

// ints returns the numbers of a list of numbers.
func (j javaInit) ints() ([]int, bool) {
	if j.list == nil {
		return nil, false
	}
	out := make([]int, len(j.list))
	for i, e := range j.list {
		if e.list != nil {
			return nil, false
		}
		out[i] = e.num
	}
	return out, true
}

// javaArrayInit parses the initializer of the array name in comment-free
// Java source: "name = { … };" with nested braces, int literals and
// trailing commas.
func javaArrayInit(src, name string) (javaInit, error) {
	i := strings.Index(src, " "+name+" =")
	if i < 0 {
		return javaInit{}, fmt.Errorf("no array %s", name)
	}
	open := strings.IndexByte(src[i:], '{')
	if open < 0 {
		return javaInit{}, fmt.Errorf("%s has no initializer", name)
	}
	p := &javaInitParser{s: src, pos: i + open}
	v, err := p.value()
	if err != nil {
		return javaInit{}, fmt.Errorf("%s: %w", name, err)
	}
	if v.list == nil {
		return javaInit{}, fmt.Errorf("%s: not a list", name)
	}
	return v, nil
}

type javaInitParser struct {
	s   string
	pos int
}

func (p *javaInitParser) skipSpace() {
	for p.pos < len(p.s) && unicode.IsSpace(rune(p.s[p.pos])) {
		p.pos++
	}
}

func (p *javaInitParser) value() (javaInit, error) {
	p.skipSpace()
	if p.pos >= len(p.s) {
		return javaInit{}, fmt.Errorf("unexpected end")
	}
	if p.s[p.pos] != '{' {
		start := p.pos
		for p.pos < len(p.s) && (p.s[p.pos] == '-' || p.s[p.pos] == 'x' || p.s[p.pos] == 'X' ||
			(p.s[p.pos] >= '0' && p.s[p.pos] <= '9') || (p.s[p.pos] >= 'a' && p.s[p.pos] <= 'f') || (p.s[p.pos] >= 'A' && p.s[p.pos] <= 'F')) {
			p.pos++
		}
		n, err := strconv.ParseInt(p.s[start:p.pos], 0, 32)
		if err != nil {
			return javaInit{}, fmt.Errorf("offset %d: not an int literal: %q", start, p.s[start:p.pos])
		}
		return javaInit{num: int(n)}, nil
	}
	p.pos++ // {
	out := javaInit{list: []javaInit{}}
	for {
		p.skipSpace()
		if p.pos >= len(p.s) {
			return javaInit{}, fmt.Errorf("unterminated list")
		}
		if p.s[p.pos] == '}' {
			p.pos++
			return out, nil
		}
		v, err := p.value()
		if err != nil {
			return javaInit{}, err
		}
		out.list = append(out.list, v)
		p.skipSpace()
		switch {
		case p.pos < len(p.s) && p.s[p.pos] == ',':
			p.pos++
		case p.pos < len(p.s) && p.s[p.pos] == '}':
		default:
			return javaInit{}, fmt.Errorf("offset %d: want , or }", p.pos)
		}
	}
}

// readAutoSpawns reads the random_spawn groups of the Seven Signs NPCs
// with their random_spawn_loc locations (AutoSpawnHandler.restoreSpawnData).
func readAutoSpawns(fsys fs.FS) ([]data.AutoSpawn, error) {
	t, err := readSQLTable(fsys, randomSpawnSQL, "random_spawn")
	if err != nil {
		return nil, err
	}
	locs, err := readSQLTable(fsys, randomSpawnLocSQL, "random_spawn_loc")
	if err != nil {
		return nil, err
	}
	byGroup := map[int][]data.AutoSpawnLoc{}
	for _, r := range locs.rows {
		g, err := r.int("groupId")
		if err != nil {
			return nil, err
		}
		var l data.AutoSpawnLoc
		for _, f := range []struct {
			col string
			dst *int
		}{{"x", &l.X}, {"y", &l.Y}, {"z", &l.Z}, {"heading", &l.Heading}} {
			if *f.dst, err = r.int(f.col); err != nil {
				return nil, err
			}
		}
		byGroup[g] = append(byGroup[g], l)
	}
	var out []data.AutoSpawn
	for _, r := range t.rows {
		var a data.AutoSpawn
		for _, f := range []struct {
			col string
			dst *int
		}{{"groupId", &a.GroupID}, {"npcId", &a.NpcID}, {"count", &a.Count}, {"initialDelay", &a.InitialDelay},
			{"respawnDelay", &a.RespawnDelay}, {"despawnDelay", &a.DespawnDelay}} {
			if *f.dst, err = r.int(f.col); err != nil {
				return nil, err
			}
		}
		if !slices.Contains(data.SevenSignsNpcIDs, a.NpcID) {
			continue
		}
		for _, f := range []struct {
			col string
			dst *bool
		}{{"broadcastSpawn", &a.BroadcastSpawn}, {"randomSpawn", &a.RandomSpawn}} {
			v, err := r.get(f.col)
			if err != nil {
				return nil, err
			}
			// Boolean.parseBoolean: "true" in any case, anything else false.
			*f.dst = strings.EqualFold(v.text, "true")
		}
		a.Locations = byGroup[a.GroupID]
		if len(a.Locations) == 0 {
			return nil, r.errorf("group %d has no location in %s", a.GroupID, randomSpawnLocSQL)
		}
		out = append(out, a)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].GroupID < out[j].GroupID })
	return out, nil
}
