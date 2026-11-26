package main

import (
	"errors"
	"fmt"

	"github.com/IPeter/l2go/internal/data"
)

// seedsXML is the file Mobius CastleManorManager.load reads.
const seedsXML = "dist/game/data/Seeds.xml"

// The manor domain converts Seeds.xml (Plan 14 M11k), loaded by data.Load.
func init() { registerDomain("manor", genSeeds) }

// genSeeds reads the seeds and adds manor/seeds.json, one seed per line, in
// file order.
func genSeeds(o options) error {
	list, err := readSeeds(o)
	if err != nil {
		return fmt.Errorf("seeds: %w", err)
	}
	if err := data.CheckSeeds(list); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.SeedsFile, list); err != nil {
		return err
	}
	castles := make(map[int]bool)
	for _, s := range list {
		castles[s.CastleID] = true
	}
	fmt.Fprintf(o.log, "manor: %d seeds in %d castles\n", len(list), len(castles))
	return nil
}

// readSeeds reads the <crop> elements of each <castle> of the <list> root
// as Mobius parseDocument does: the castle's id, and the crop's id (the
// crop), seedId, mature_Id, reward1, reward2, alternative, level,
// limit_seed and limit_crops. A missing attribute, which Mobius's StatSet
// would refuse, is an error here, and so is an unknown element or attribute.
func readSeeds(o options) ([]data.Seed, error) {
	root, err := readXMLTree(o.src, seedsXML)
	if err != nil {
		return nil, err
	}
	if root.name != "list" {
		return nil, root.errorf("want <list>")
	}
	if err := root.checkChildren("castle"); err != nil {
		return nil, err
	}
	var list []data.Seed
	for _, c := range root.children {
		if err := c.checkAttrs("id"); err != nil {
			return nil, err
		}
		if err := c.checkChildren("crop"); err != nil {
			return nil, err
		}
		castleID, err := c.intAttr("id")
		if err != nil {
			return nil, err
		}
		for _, n := range c.children {
			s, err := readSeed(n, castleID)
			if err != nil {
				return nil, err
			}
			list = append(list, s)
		}
	}
	if len(list) == 0 {
		return nil, errors.New(seedsXML + ": no seeds")
	}
	return list, nil
}

// readSeed reads one <crop>.
func readSeed(n *xmlNode, castleID int) (data.Seed, error) {
	s := data.Seed{CastleID: castleID}
	if err := n.checkAttrs("id", "seedId", "mature_Id", "reward1", "reward2", "alternative", "level", "limit_seed", "limit_crops"); err != nil {
		return s, err
	}
	for name, dst := range map[string]*int{
		"id": &s.CropID, "seedId": &s.SeedID, "mature_Id": &s.MatureID, "reward1": &s.Reward1, "reward2": &s.Reward2,
		"level": &s.Level, "limit_seed": &s.LimitSeeds, "limit_crops": &s.LimitCrops,
	} {
		v, err := n.intAttr(name)
		if err != nil {
			return s, err
		}
		*dst = v
	}
	alt, ok := n.attr("alternative")
	if !ok {
		return s, n.errorf("no alternative attribute")
	}
	v, err := parseJavaBool(alt)
	if err != nil {
		return s, n.errorf("alternative: %v", err)
	}
	s.Alternative = v
	return s, nil
}
