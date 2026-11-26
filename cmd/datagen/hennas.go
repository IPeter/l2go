package main

import (
	"fmt"
	"slices"

	"github.com/IPeter/l2go/internal/data"
)

// hennaListXML is the dye list Mobius's HennaData reads.
const hennaListXML = "dist/game/data/stats/hennaList.xml"

// The hennas domain converts hennaList.xml, loaded by data.LoadHennas.
func init() { registerDomain("hennas", genHennas) }

// genHennas reads the dyes and adds hennas/hennas.json, one dye per line,
// by dye ID.
func genHennas(o options) error {
	list, err := readHennas(o)
	if err != nil {
		return fmt.Errorf("hennas: %w", err)
	}
	if _, err := data.NewHennas(list); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.HennasFile, list); err != nil {
		return err
	}
	fmt.Fprintf(o.log, "hennas: %d dyes\n", len(list))
	return nil
}

// readHennas reads the dyes as Mobius's HennaData.parseHenna does: the
// <henna> attributes, the <stats> (a missing stat is 0), <wear> and
// <cancel> counts and fees, and the <classId> list, kept as a sorted set
// (Mobius keeps a HashSet). A dye ID listed twice fails, where Mobius would
// keep the last; so does a class ID Mobius does not know (it would store
// null). dyeName and dyeLevel are not used by Mobius and left out.
func readHennas(o options) ([]data.Henna, error) {
	root, err := readXMLTree(o.src, hennaListXML)
	if err != nil {
		return nil, err
	}
	if root.name != "list" {
		return nil, root.errorf("want <list>")
	}
	if err := root.checkChildren("henna"); err != nil {
		return nil, err
	}
	var list []data.Henna
	seen := make(map[int]bool)
	for _, e := range root.children {
		h, err := readHenna(e)
		if err != nil {
			return nil, err
		}
		if seen[h.DyeID] {
			return nil, e.errorf("dye %d is listed twice (Mobius keeps the last)", h.DyeID)
		}
		seen[h.DyeID] = true
		list = append(list, h)
	}
	slices.SortFunc(list, func(a, b data.Henna) int { return a.DyeID - b.DyeID })
	return list, nil
}

func readHenna(e *xmlNode) (data.Henna, error) {
	var h data.Henna
	if err := e.checkAttrs("dyeId", "dyeName", "dyeItemId", "dyeLevel"); err != nil {
		return h, err
	}
	if err := e.checkChildren("stats", "wear", "cancel", "classId"); err != nil {
		return h, err
	}
	var err error
	if h.DyeID, err = e.intAttr("dyeId"); err != nil {
		return h, err
	}
	if h.DyeItemID, err = e.intAttr("dyeItemId"); err != nil {
		return h, err
	}
	var wear, cancel bool
	for _, c := range e.children {
		switch c.name {
		case "stats":
			if err := c.checkAttrs("str", "con", "dex", "int", "men", "wit"); err != nil {
				return h, err
			}
			for _, f := range []struct {
				name string
				dst  *int
			}{{"str", &h.STR}, {"con", &h.CON}, {"dex", &h.DEX}, {"int", &h.INT}, {"men", &h.MEN}, {"wit", &h.WIT}} {
				if _, ok := c.attr(f.name); !ok {
					continue
				}
				if *f.dst, err = c.intAttr(f.name); err != nil {
					return h, err
				}
			}
		case "wear", "cancel":
			if err := c.checkAttrs("count", "fee"); err != nil {
				return h, err
			}
			count, err := c.intAttr("count")
			if err != nil {
				return h, err
			}
			fee, err := c.intAttr("fee")
			if err != nil {
				return h, err
			}
			if c.name == "wear" {
				h.WearCount, h.WearFee, wear = count, fee, true
			} else {
				h.CancelCount, h.CancelFee, cancel = count, fee, true
			}
		case "classId":
			id, err := c.intText()
			if err != nil {
				return h, err
			}
			if !data.PlayerClass(id).Valid() {
				return h, c.errorf("unknown class %d", id)
			}
			if i, found := slices.BinarySearch(h.Classes, id); !found {
				h.Classes = slices.Insert(h.Classes, i, id)
			}
		}
	}
	switch {
	case !wear:
		return h, e.errorf("dye %d has no <wear>", h.DyeID)
	case !cancel:
		return h, e.errorf("dye %d has no <cancel>", h.DyeID)
	}
	if h.Classes == nil {
		h.Classes = []int{}
	}
	return h, nil
}
