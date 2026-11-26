package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// Sources of the castles domain (Plan 14 M11i, decision X6).
const (
	castleSQL            = "dist/db_installer/sql/game/castle.sql"
	castleSiegeGuardsSQL = "dist/db_installer/sql/game/castle_siege_guards.sql"
	siegeIni             = "dist/game/config/Siege.ini"
	siegeScheduleXML     = "dist/game/config/SiegeSchedule.xml"
	mercTicketJava       = "java/org/l2jmobius/gameserver/managers/MercTicketManager.java"
)

// mercTypesPerCastle is MercTicketManager.GUARDIAN_TYPES_COUNT.
const mercTypesPerCastle = 52

// The castles domain converts the castles, their siege schedule, towers and
// guards, and the mercenary tickets; the game loads them with
// data.LoadCastles.
func init() { registerDomain("castles", genCastles) }

func genCastles(o options) error {
	d, err := readCastleData(o)
	if err != nil {
		return fmt.Errorf("castles: %w", err)
	}
	if _, err := data.NewCastles(d); err != nil {
		return err
	}
	if err := o.out.addJSON(data.CastlesFile, d.Castles); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.SiegeGuardsFile, d.Guards); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.MercTicketsFile, d.Tickets); err != nil {
		return err
	}
	fmt.Fprintf(o.log, "castles: %d castles, %d siege guards, %d mercenary tickets\n", len(d.Castles), len(d.Guards), len(d.Tickets))
	return nil
}

func readCastleData(o options) (data.CastleData, error) {
	var d data.CastleData
	t, err := readSQLTable(o.src, castleSQL, "castle")
	if err != nil {
		return d, err
	}
	for _, r := range t.rows {
		var c data.Castle
		if c.ID, err = r.int("id"); err != nil {
			return d, err
		}
		v, err := r.get("name")
		if err != nil {
			return d, err
		}
		c.Name = v.text
		d.Castles = append(d.Castles, c)
	}
	sort.Slice(d.Castles, func(i, j int) bool { return d.Castles[i].ID < d.Castles[j].ID })
	if err := readSiegeSchedule(o.src, d.Castles); err != nil {
		return d, err
	}
	ini, err := readIni(o.src, siegeIni)
	if err != nil {
		return d, err
	}
	maxMercs, err := javaIntArray(o.src, mercTicketJava, "MERCS_MAX_PER_CASTLE")
	if err != nil {
		return d, err
	}
	for i := range d.Castles {
		c := &d.Castles[i]
		if c.ControlTowers, err = iniTowers(ini, c.Name+"ControlTower", false); err != nil {
			return d, err
		}
		if c.FlameTowers, err = iniTowers(ini, c.Name+"FlameTower", true); err != nil {
			return d, err
		}
		if c.ID-1 < len(maxMercs) {
			c.MaxMercenaries = maxMercs[c.ID-1]
		}
		if v, ok := ini[c.Name+"MaxMercenaries"]; ok {
			if c.MaxMercenaries, err = strconv.Atoi(v); err != nil {
				return d, fmt.Errorf("%s: %sMaxMercenaries: %w", siegeIni, c.Name, err)
			}
		}
	}
	if d.Guards, err = readSiegeGuards(o.src); err != nil {
		return d, err
	}
	if d.Tickets, err = readMercTickets(o.src, len(d.Castles)); err != nil {
		return d, err
	}
	return d, nil
}

// readSiegeSchedule fills the schedule fields from SiegeSchedule.xml.
func readSiegeSchedule(fsys fs.FS, castles []data.Castle) error {
	root, err := readXMLTree(fsys, siegeScheduleXML)
	if err != nil {
		return err
	}
	if err := root.checkChildren("schedule"); err != nil {
		return err
	}
	days := map[string]int{"SUNDAY": 0, "MONDAY": 1, "TUESDAY": 2, "WEDNESDAY": 3, "THURSDAY": 4, "FRIDAY": 5, "SATURDAY": 6}
	for _, n := range root.children {
		if err := n.checkAttrs("castleId", "castleName", "siegeEnabled", "day", "hour", "maxConcurrent"); err != nil {
			return err
		}
		id, err := n.intAttr("castleId")
		if err != nil {
			return err
		}
		if id < 1 || id > len(castles) {
			return n.errorf("unknown castle %d", id)
		}
		c := &castles[id-1]
		en, _ := n.attr("siegeEnabled")
		if c.SiegeEnabled, err = parseJavaBool(en); err != nil {
			return n.errorf("siegeEnabled: %v", err)
		}
		day, _ := n.attr("day")
		d, ok := days[day]
		if !ok {
			return n.errorf("unknown day %q", day)
		}
		c.SiegeDay = d
		if c.SiegeHour, err = n.intAttr("hour"); err != nil {
			return err
		}
		if c.MaxConcurrent, err = n.intAttr("maxConcurrent"); err != nil {
			return err
		}
	}
	return nil
}

// readIni reads the key = value lines of a Mobius .ini file.
func readIni(fsys fs.FS, name string) (map[string]string, error) {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	out := make(map[string]string)
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' || line[0] == '!' {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out, sc.Err()
}

// iniTowers reads prefix1, prefix2, … as SiegeManager.load does: x, y, z,
// npc and, for flame towers, the zone IDs.
func iniTowers(ini map[string]string, prefix string, flame bool) ([]data.TowerSpawn, error) {
	var out []data.TowerSpawn
	for i := 1; i < 0xFF; i++ {
		v, ok := ini[prefix+strconv.Itoa(i)]
		if !ok {
			break
		}
		parts := strings.Split(v, ",")
		if len(parts) < 4 {
			return nil, fmt.Errorf("%s: %s%d: want x,y,z,npc", siegeIni, prefix, i)
		}
		nums := make([]int, len(parts))
		for j, p := range parts {
			n, err := strconv.Atoi(strings.TrimSpace(p))
			if err != nil {
				return nil, fmt.Errorf("%s: %s%d: %w", siegeIni, prefix, i, err)
			}
			nums[j] = n
		}
		t := data.TowerSpawn{X: nums[0], Y: nums[1], Z: nums[2], NpcID: nums[3]}
		if flame {
			t.Zones = nums[4:]
		}
		out = append(out, t)
	}
	return out, nil
}

// readSiegeGuards reads castle_siege_guards.sql (the rows with isHired 0).
func readSiegeGuards(fsys fs.FS) ([]data.SiegeGuard, error) {
	t, err := readSQLTable(fsys, castleSiegeGuardsSQL, "castle_siege_guards")
	if err != nil {
		return nil, err
	}
	var out []data.SiegeGuard
	for _, r := range t.rows {
		hired, err := r.int("isHired")
		if err != nil {
			return nil, err
		}
		if hired != 0 {
			continue
		}
		var g data.SiegeGuard
		for _, f := range []struct {
			col string
			dst *int
		}{
			{"castleId", &g.CastleID}, {"id", &g.ID}, {"npcId", &g.NpcID}, {"x", &g.X}, {"y", &g.Y}, {"z", &g.Z},
			{"heading", &g.Heading}, {"respawnDelay", &g.RespawnDelay},
		} {
			if *f.dst, err = r.int(f.col); err != nil {
				return nil, err
			}
		}
		out = append(out, g)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].CastleID != out[j].CastleID {
			return out[i].CastleID < out[j].CastleID
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// readMercTickets reads the ticket tables of MercTicketManager.java.
func readMercTickets(fsys fs.FS, castles int) ([]data.MercTicket, error) {
	items, err := javaIntArray(fsys, mercTicketJava, "ITEM_IDS")
	if err != nil {
		return nil, err
	}
	npcs, err := javaIntArray(fsys, mercTicketJava, "NPC_IDS")
	if err != nil {
		return nil, err
	}
	maxes, err := javaIntArray(fsys, mercTicketJava, "MAX_MERC_PER_TYPE")
	if err != nil {
		return nil, err
	}
	if len(items) != len(npcs) || len(items) != len(maxes) || len(items) != castles*mercTypesPerCastle {
		return nil, fmt.Errorf("%s: %d items, %d NPCs, %d limits for %d castles", mercTicketJava, len(items), len(npcs), len(maxes), castles)
	}
	out := make([]data.MercTicket, len(items))
	for i := range items {
		out[i] = data.MercTicket{ItemID: items[i], NpcID: npcs[i], CastleID: i/mercTypesPerCastle + 1,
			Index: i % mercTypesPerCastle, MaxPerType: maxes[i]}
	}
	return out, nil
}

// javaIntArray reads the initializer of the int array name in a Java file:
// "name = { 1, 2, … };" (comments stripped).
func javaIntArray(fsys fs.FS, file, name string) ([]int, error) {
	b, err := fs.ReadFile(fsys, file)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", file, err)
	}
	src := stripJavaComments(string(b))
	i := strings.Index(src, " "+name+" =")
	if i < 0 {
		return nil, fmt.Errorf("%s: no array %s", file, name)
	}
	open := strings.IndexByte(src[i:], '{')
	end := strings.IndexByte(src[i:], '}')
	if open < 0 || end < open {
		return nil, fmt.Errorf("%s: %s has no initializer", file, name)
	}
	var out []int
	for _, f := range strings.Split(src[i+open+1:i+end], ",") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		n, err := strconv.Atoi(f)
		if err != nil {
			return nil, fmt.Errorf("%s: %s: %w", file, name, err)
		}
		out = append(out, n)
	}
	return out, nil
}
