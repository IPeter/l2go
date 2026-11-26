package main

import (
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// Sources of the clan halls domain (Plan 14 M11j, decision X6).
const (
	clanHallSQL          = "dist/db_installer/sql/game/clanhall.sql"
	siegableClanHallSQL  = "dist/db_installer/sql/game/siegable_clanhall.sql"
	clanHallGuardsSQL    = "dist/db_installer/sql/game/clanhall_siege_guards.sql"
	auctionSQL           = "dist/db_installer/sql/game/auction.sql"
	clanHallAuctionsJava = "java/org/l2jmobius/gameserver/managers/ClanHallAuctionManager.java"
)

// The clan halls domain converts the auctionable and contestable halls, the
// contestable halls' siege guards and the NPC auctions; the game loads them
// with data.LoadClanHalls.
func init() { registerDomain("clanhalls", genClanHalls) }

func genClanHalls(o options) error {
	d, err := readClanHallData(o.src)
	if err != nil {
		return fmt.Errorf("clanhalls: %w", err)
	}
	if _, err := data.NewClanHalls(d); err != nil {
		return err
	}
	if err := o.out.addJSON(data.ClanHallsFile, d.Halls); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.ClanHallGuardsFile, d.Guards); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.ClanHallAuctionsFile, d.Auctions); err != nil {
		return err
	}
	fmt.Fprintf(o.log, "clanhalls: %d halls, %d siege guards, %d NPC auctions\n", len(d.Halls), len(d.Guards), len(d.Auctions))
	return nil
}

func readClanHallData(fsys fs.FS) (data.ClanHallData, error) {
	var d data.ClanHallData
	t, err := readSQLTable(fsys, clanHallSQL, "clanhall")
	if err != nil {
		return d, err
	}
	str := func(r sqlRow, col string) (string, error) {
		v, err := r.get(col)
		return v.text, err
	}
	for _, r := range t.rows {
		var h data.ClanHall
		if h.ID, err = r.int("id"); err != nil {
			return d, err
		}
		if h.Lease, err = r.int("lease"); err != nil {
			return d, err
		}
		if h.Grade, err = r.int("Grade"); err != nil {
			return d, err
		}
		for _, f := range []struct {
			col string
			dst *string
		}{{"name", &h.Name}, {"desc", &h.Desc}, {"location", &h.Location}} {
			if *f.dst, err = str(r, f.col); err != nil {
				return d, err
			}
		}
		d.Halls = append(d.Halls, h)
	}
	st, err := readSQLTable(fsys, siegableClanHallSQL, "siegable_clanhall")
	if err != nil {
		return d, err
	}
	for _, r := range st.rows {
		h := data.ClanHall{Siegable: true}
		if h.ID, err = r.int("clanHallId"); err != nil {
			return d, err
		}
		if h.SiegeLength, err = r.long("siegeLength"); err != nil {
			return d, err
		}
		for _, f := range []struct {
			col string
			dst *string
		}{{"name", &h.Name}, {"desc", &h.Desc}, {"location", &h.Location}} {
			if *f.dst, err = str(r, f.col); err != nil {
				return d, err
			}
		}
		sched, err := str(r, "schedule_config")
		if err != nil {
			return d, err
		}
		if h.Schedule, err = parseHallSchedule(sched); err != nil {
			return d, r.errorf("schedule_config: %v", err)
		}
		d.Halls = append(d.Halls, h)
	}
	sort.Slice(d.Halls, func(i, j int) bool { return d.Halls[i].ID < d.Halls[j].ID })
	if d.Guards, err = readClanHallGuards(fsys); err != nil {
		return d, err
	}
	if d.Auctions, err = readNpcAuctions(fsys); err != nil {
		return d, err
	}
	return d, nil
}

// parseHallSchedule reads "DD;MM;YY;HH;mm" as SiegableHall does; Mobius
// falls back to 7;0;0;12;0 for a value without five parts.
func parseHallSchedule(s string) ([5]int, error) {
	out := [5]int{7, 0, 0, 12, 0}
	parts := strings.Split(s, ";")
	if len(parts) != 5 {
		return out, fmt.Errorf("%q: want 5 parts", s)
	}
	for i, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return out, fmt.Errorf("%q: %w", s, err)
		}
		out[i] = n
	}
	return out, nil
}

// readClanHallGuards reads clanhall_siege_guards.sql.
func readClanHallGuards(fsys fs.FS) ([]data.ClanHallGuard, error) {
	t, err := readSQLTable(fsys, clanHallGuardsSQL, "clanhall_siege_guards")
	if err != nil {
		return nil, err
	}
	out := make([]data.ClanHallGuard, 0, len(t.rows))
	for _, r := range t.rows {
		var g data.ClanHallGuard
		for _, f := range []struct {
			col string
			dst *int
		}{
			{"clanHallId", &g.HallID}, {"npcId", &g.NpcID}, {"x", &g.X}, {"y", &g.Y}, {"z", &g.Z},
			{"heading", &g.Heading}, {"respawnDelay", &g.RespawnDelay},
		} {
			if *f.dst, err = r.int(f.col); err != nil {
				return nil, err
			}
		}
		boss, err := r.get("isSiegeBoss")
		if err != nil {
			return nil, err
		}
		g.SiegeBoss = boss.text == "true"
		out = append(out, g)
	}
	return out, nil
}

// readNpcAuctions reads the rows of ClanHallAuctionManager.ITEM_INIT_DATA,
// INSERT values written as Java strings, with the columns of auction.sql.
func readNpcAuctions(fsys fs.FS) ([]data.ClanHallAuction, error) {
	rows, err := javaStringArray(fsys, clanHallAuctionsJava, "ITEM_INIT_DATA")
	if err != nil {
		return nil, err
	}
	b, err := fs.ReadFile(fsys, auctionSQL)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", auctionSQL, err)
	}
	script := string(b) + ";\nINSERT INTO `auction` VALUES " + strings.Join(rows, ",\n") + ";\n"
	t, err := parseSQLTable(script, clanHallAuctionsJava, "auction")
	if err != nil {
		return nil, fmt.Errorf("parse %s ITEM_INIT_DATA: %w", clanHallAuctionsJava, err)
	}
	out := make([]data.ClanHallAuction, 0, len(t.rows))
	for _, r := range t.rows {
		var a data.ClanHallAuction
		if a.HallID, err = r.int("itemId"); err != nil {
			return nil, err
		}
		v, err := r.get("itemName")
		if err != nil {
			return nil, err
		}
		a.ItemName = v.text
		if a.StartingBid, err = r.long("startingBid"); err != nil {
			return nil, err
		}
		if a.EndDate, err = r.long("endDate"); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].HallID < out[j].HallID })
	return out, nil
}

// javaStringArray reads the string literals of the initializer of the
// String array name in a Java file: "name = { "a", "b" };".
func javaStringArray(fsys fs.FS, file, name string) ([]string, error) {
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
	end := strings.Index(src[i:], "};")
	if open < 0 || end < open {
		return nil, fmt.Errorf("%s: %s has no initializer", file, name)
	}
	body := src[i+open+1 : i+end]
	var out []string
	for j := 0; j < len(body); j++ {
		if body[j] != '"' {
			continue
		}
		k := strings.IndexByte(body[j+1:], '"')
		if k < 0 {
			return nil, fmt.Errorf("%s: %s: unterminated string", file, name)
		}
		out = append(out, body[j+1:j+1+k])
		j += k + 1
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: %s is empty", file, name)
	}
	return out, nil
}
