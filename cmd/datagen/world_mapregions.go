package main

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"

	"github.com/IPeter/l2go/internal/data"
)

// mapRegionXMLDir holds the map regions (only the files directly in it).
const mapRegionXMLDir = "dist/game/data/mapregion"

// readMapRegions reads the map regions, sorted by name. Mobius reads every
// list, whatever its enabled attribute says.
//
// A map cell listed by two regions is kept only in the region that owns it
// in Mobius: MapRegionData fills its grid by iterating a java.util.HashMap
// of the regions, so the region iterated last wins. resolveMapCells
// reproduces that order.
func readMapRegions(o options) ([]data.MapRegion, error) {
	names, err := fs.Glob(o.src, mapRegionXMLDir+"/*.xml")
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("%s: no map region files found", mapRegionXMLDir)
	}
	var (
		regions []data.MapRegion
		byName  = make(map[string]string) // region name -> file
		errs    []error
	)
	for _, name := range names {
		file := path.Base(name)
		doc, err := parseMX(o.src, name)
		if err != nil {
			return nil, err
		}
		list, err := mxRoot(doc)
		if err != nil {
			return nil, err
		}
		list.ignore("enabled")
		for _, e := range list.children() {
			if e.name != "region" {
				e.unknown()
				continue
			}
			r := parseMapRegion(e, file)
			if prev, dup := byName[r.Name]; dup {
				e.errorf("region %q is also defined in %s (Mobius keeps the one it loads last)", r.Name, prev)
				continue
			}
			byName[r.Name] = file
			regions = append(regions, r)
		}
		if err := doc.check(); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	if err := resolveMapCells(o, regions); err != nil {
		return nil, err
	}
	slices.SortFunc(regions, func(a, b data.MapRegion) int { return cmp.Compare(a.Name, b.Name) })
	return regions, nil
}

// parseMapRegion reads one <region>, as Mobius's MapRegionData does.
func parseMapRegion(e *mxElem, file string) data.MapRegion {
	r := data.MapRegion{
		Name:   e.reqStr("name"),
		Town:   e.reqStr("town"),
		LocID:  e.reqInt("locId"),
		Castle: e.reqInt("castle"),
		BBS:    e.reqInt("bbs"),
		File:   file,
	}
	for _, c := range e.children() {
		switch c.name {
		case "respawnPoint":
			loc := data.Location{X: c.reqInt("X"), Y: c.reqInt("Y"), Z: c.reqInt("Z")}
			// Mobius checks the flags in this order; the first set one wins.
			other, chaotic, banish := c.boolAttr("isOther", false), c.boolAttr("isChaotic", false), c.boolAttr("isBanish", false)
			switch {
			case other:
				r.OtherRespawnPoints = append(r.OtherRespawnPoints, loc)
			case chaotic:
				r.ChaoticRespawnPoints = append(r.ChaoticRespawnPoints, loc)
			case banish:
				r.BanishRespawnPoints = append(r.BanishRespawnPoints, loc)
			default:
				r.RespawnPoints = append(r.RespawnPoints, loc)
			}
		case "map":
			r.Cells = append(r.Cells, data.MapCell{X: c.reqInt("X"), Y: c.reqInt("Y")})
		case "banned":
			name, point := c.reqStr("race"), c.reqStr("point")
			race, ok := data.ParseRace(name) // Mobius: Race.valueOf
			if !ok {
				c.errorf("unknown race %q", name)
				continue
			}
			if _, dup := r.BannedRaces[race]; dup {
				c.errorf("race %s is banned twice (Mobius keeps the last)", name)
				continue
			}
			if r.BannedRaces == nil {
				r.BannedRaces = make(map[data.Race]string)
			}
			r.BannedRaces[race] = point
		default:
			c.unknown()
		}
	}
	return r
}

// resolveMapCells removes each map cell from all regions but the one that
// owns it in Mobius's grid, and logs each decision. regions is in Mobius
// load order (file name order here; Mobius uses the file system's).
func resolveMapCells(o options, regions []data.MapRegion) error {
	keys := make([]string, len(regions))
	for i, r := range regions {
		keys[i] = r.Name
	}
	order, err := javaHashMapOrder(keys)
	if err != nil {
		return err
	}
	owner := make(map[data.MapCell]string)
	for _, i := range order {
		for _, c := range regions[i].Cells {
			owner[c] = regions[i].Name // the last region iterated wins
		}
	}
	for i := range regions {
		r := &regions[i]
		kept := r.Cells[:0]
		for _, c := range r.Cells {
			if dup := slices.Contains(kept, c); dup {
				continue
			}
			if winner := owner[c]; winner != r.Name {
				o.warnf("mapregions: cell %d_%d is listed by %s and %s; Mobius's grid gives it to %s",
					c.X, c.Y, r.Name, winner, winner)
				continue
			}
			kept = append(kept, c)
		}
		if len(kept) == 0 {
			kept = nil
		}
		r.Cells = kept
	}
	return nil
}

// javaHashMapOrder returns the indexes of distinct keys in the iteration
// order of a java.util.HashMap<String, ?> with default settings after the
// keys were put in the given order: by bucket, then by insertion within a
// bucket (resizing keeps that order). The table starts with 16 buckets and
// doubles when it gets more than 3/4 full, or when a bucket reaches 9
// entries while the table is under 64 buckets. It refuses inputs where
// Java would turn a bucket into a tree.
func javaHashMapOrder(keys []string) ([]int, error) {
	hashes := make([]uint32, len(keys))
	for i, k := range keys {
		h := uint32(javaStringHash(k))
		hashes[i] = h ^ h>>16
	}
	capacity := 16
	bucket := func(i int) uint32 { return hashes[i] & uint32(capacity-1) }
	for i := range keys {
		n := 0
		for j := range i + 1 {
			if bucket(j) == bucket(i) {
				n++
			}
		}
		if n >= 9 {
			if capacity >= 64 {
				return nil, fmt.Errorf("mapregions: %d regions share a HashMap bucket; datagen cannot reproduce Mobius's order", n)
			}
			capacity *= 2
		}
		if i+1 > capacity*3/4 {
			capacity *= 2
		}
	}
	order := make([]int, len(keys))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return cmp.Compare(bucket(a), bucket(b)) })
	return order, nil
}

// javaStringHash is Java's String.hashCode for s: s[0]*31^(n-1) + ... +
// s[n-1] over UTF-16 code units, with int overflow.
func javaStringHash(s string) int32 {
	var h int32
	for _, r := range s {
		if r >= 0x10000 {
			r1, r2 := utf16Pair(r)
			h = 31*h + int32(r1)
			h = 31*h + int32(r2)
			continue
		}
		h = 31*h + int32(r)
	}
	return h
}

// utf16Pair returns the UTF-16 surrogate pair of a supplementary rune.
func utf16Pair(r rune) (rune, rune) {
	r -= 0x10000
	return 0xd800 + (r>>10)&0x3ff, 0xdc00 + r&0x3ff
}
