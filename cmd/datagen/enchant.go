package main

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/enchant"
)

// The enchant files Mobius's EnchantItemData, EnchantItemGroupsData and
// EnchantItemHPBonusData read.
const (
	enchantItemXML   = "dist/game/data/EnchantItemData.xml"
	enchantGroupsXML = "dist/game/data/EnchantItemGroups.xml"
	enchantHPXML     = "dist/game/data/stats/enchantHPBonus.xml"
)

// The enchant domain converts the enchant scrolls and their rate groups
// into enchant/enchant.json, loaded by enchant.Load. enchantHPBonus.xml is
// only checked: its Interlude table is all zeros, so it adds nothing (see
// stats.ItemModifiers), and a datapack with a bonus fails here.
func init() { registerDomain("enchant", genEnchant) }

// genEnchant reads the enchant data and adds enchant/enchant.json.
func genEnchant(o options) error {
	d, err := readEnchant(o)
	if err != nil {
		return fmt.Errorf("enchant: %w", err)
	}
	if err := d.Check(); err != nil {
		return err
	}
	if err := checkEnchantHPBonus(o); err != nil {
		return fmt.Errorf("enchant: %w", err)
	}
	if err := o.out.addJSON(enchant.File, d); err != nil {
		return err
	}
	fmt.Fprintf(o.log, "enchant: %d scrolls, %d rate groups, %d scroll groups\n", len(d.Scrolls), len(d.RateGroups), len(d.ScrollGroups))
	return nil
}

// readEnchant reads both enchant files.
func readEnchant(o options) (enchant.Data, error) {
	scrolls, err := readEnchantScrolls(o)
	if err != nil {
		return enchant.Data{}, err
	}
	rates, groups, err := readEnchantGroups(o)
	if err != nil {
		return enchant.Data{}, err
	}
	return enchant.Data{Scrolls: scrolls, RateGroups: rates, ScrollGroups: groups}, nil
}

// readEnchantScrolls reads EnchantItemData.xml as Mobius does, sorted by
// ID. A scroll listed twice fails, where Mobius would keep the last.
func readEnchantScrolls(o options) ([]enchant.Scroll, error) {
	doc, err := parseMX(o.src, enchantItemXML)
	if err != nil {
		return nil, err
	}
	root, err := mxRoot(doc)
	if err != nil {
		return nil, err
	}
	var scrolls []enchant.Scroll
	seen := make(map[int]bool)
	for _, e := range root.children() {
		if e.name != "enchant" {
			e.unknown()
			continue
		}
		s := enchant.Scroll{
			ID:          e.reqInt("id"),
			MaxEnchant:  e.intAttr("maxEnchant", enchant.DefaultMaxEnchant),
			BonusRate:   e.floatAttr("bonusRate", 0),
			ScrollGroup: e.intAttr("scrollGroupId", 0),
		}
		grade := e.str("targetGrade", "NONE")
		g, ok := data.ParseCrystalType(grade)
		if !ok {
			e.errorf("unknown targetGrade %q", grade)
		}
		s.Grade = g
		if seen[s.ID] {
			e.errorf("scroll %d is listed twice (Mobius keeps the last)", s.ID)
		}
		seen[s.ID] = true
		for _, it := range e.children() {
			if it.name != "item" {
				it.unknown()
				continue
			}
			group := it.intAttr("altScrollGroupId", -1)
			if group <= -1 {
				group = s.ScrollGroup
			}
			s.Items = append(s.Items, enchant.ScrollItem{ID: it.reqInt("id"), ScrollGroup: group})
		}
		slices.SortFunc(s.Items, func(a, b enchant.ScrollItem) int { return cmp.Compare(a.ID, b.ID) })
		scrolls = append(scrolls, s)
	}
	if err := doc.check(); err != nil {
		return nil, err
	}
	if len(scrolls) == 0 {
		return nil, errors.New(enchantItemXML + ": no scrolls")
	}
	slices.SortFunc(scrolls, func(a, b enchant.Scroll) int { return cmp.Compare(a.ID, b.ID) })
	return scrolls, nil
}

// readEnchantGroups reads EnchantItemGroups.xml as Mobius does: the rate
// groups in file order, their ranges "min-max" or "n" (a range Mobius
// cannot read is left out, as Mobius does), and the scroll groups sorted by
// ID, each rate's slots ORed together (a slot Mobius does not know fails).
func readEnchantGroups(o options) ([]enchant.RateGroup, []enchant.ScrollGroup, error) {
	doc, err := parseMX(o.src, enchantGroupsXML)
	if err != nil {
		return nil, nil, err
	}
	root, err := mxRoot(doc)
	if err != nil {
		return nil, nil, err
	}
	var (
		rates  []enchant.RateGroup
		groups []enchant.ScrollGroup
	)
	for _, e := range root.children() {
		switch e.name {
		case "enchantRateGroup":
			g := enchant.RateGroup{Name: e.reqStr("name"), Chances: []enchant.RangeChance{}}
			for _, c := range e.children() {
				if c.name != "current" {
					c.unknown()
					continue
				}
				r, ok := enchantRange(c.reqStr("enchant"))
				r.Chance = c.reqFloat("chance")
				if !ok {
					o.warnf("enchant: rate group %s: range %q left out, as Mobius does", g.Name, c.str("enchant", ""))
					continue
				}
				g.Chances = append(g.Chances, r)
			}
			rates = append(rates, g)
		case "enchantScrollGroup":
			sg := enchant.ScrollGroup{ID: e.reqInt("id"), Rates: []enchant.RateItem{}}
			for _, r := range e.children() {
				if r.name != "enchantRate" {
					r.unknown()
					continue
				}
				sg.Rates = append(sg.Rates, readEnchantRate(r))
			}
			groups = append(groups, sg)
		default:
			e.unknown()
		}
	}
	if err := doc.check(); err != nil {
		return nil, nil, err
	}
	slices.SortFunc(groups, func(a, b enchant.ScrollGroup) int { return cmp.Compare(a.ID, b.ID) })
	return rates, groups, nil
}

// readEnchantRate reads an <enchantRate> as EnchantItemGroupsData does.
func readEnchantRate(r *mxElem) enchant.RateItem {
	rate := enchant.RateItem{Group: r.reqStr("group")}
	for _, it := range r.children() {
		if it.name != "item" {
			it.unknown()
			continue
		}
		if slot, ok := it.attr("slot"); ok {
			part, known := data.BodyPartByName(slot)
			if !known {
				it.errorf("unknown slot %q", slot)
			}
			rate.Slots |= part.Mask()
		}
		if _, ok := it.attr("magicWeapon"); ok {
			magic := it.boolAttr("magicWeapon", false)
			rate.MagicWeapon = &magic
		}
		if _, ok := it.attr("itemId"); ok {
			rate.ItemIDs = append(rate.ItemIDs, it.intAttr("itemId", 0))
		}
	}
	slices.Sort(rate.ItemIDs)
	rate.ItemIDs = slices.Compact(rate.ItemIDs)
	return rate
}

// enchantRange parses a range of levels as EnchantItemGroupsData does:
// "min-max" or a single number; false for anything else.
func enchantRange(s string) (enchant.RangeChance, bool) {
	num := func(t string) (int, bool) {
		if t == "" || strings.Trim(t, "0123456789") != "" {
			return 0, false
		}
		n, err := parseJavaInt(t)
		return n, err == nil
	}
	if lo, hi, ok := strings.Cut(s, "-"); ok {
		a, ok1 := num(lo)
		b, ok2 := num(hi)
		return enchant.RangeChance{Min: a, Max: b}, ok1 && ok2
	}
	n, ok := num(s)
	return enchant.RangeChance{Min: n, Max: n}, ok
}

// checkEnchantHPBonus checks that every bonus of enchantHPBonus.xml is 0.
func checkEnchantHPBonus(o options) error {
	doc, err := parseMX(o.src, enchantHPXML)
	if err != nil {
		return err
	}
	root, err := mxRoot(doc)
	if err != nil {
		return err
	}
	for _, e := range root.children() {
		if e.name != "enchantHP" {
			e.unknown()
			continue
		}
		grade := e.reqStr("grade")
		for _, b := range e.children() {
			if b.name != "bonus" {
				b.unknown()
				continue
			}
			if v := b.textFloat(); v != 0 {
				b.errorf("grade %s has an enchant HP bonus of %v; L2Go leaves the bonus out (all zeros in Interlude)", grade, v)
			}
		}
	}
	return doc.check()
}
