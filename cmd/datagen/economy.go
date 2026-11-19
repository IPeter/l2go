package main

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// The economy domain converts the NPC shops (buy lists), the multisell
// lists and the merchant price configs, loaded by data.LoadEconomy.
const (
	// buyListXMLDir and multisellXMLDir hold one list per file, named by
	// its ID. Mobius reads only the files directly in them, and their
	// custom/ folders only with CustomBuyListLoad and CustomMultisellLoad,
	// extras L2Go keeps off.
	buyListXMLDir   = "dist/game/data/buylists"
	multisellXMLDir = "dist/game/data/multisell"
	priceConfigXML  = "dist/game/data/MerchantPriceConfig.xml"
)

func init() { registerDomain("economy", genEconomy) }

// genEconomy reads the economy data, checks it with data.NewEconomy against
// the items and NPCs it names, and adds its files:
//
//   - buylists/buylists.json and multisell/multisell.json: a JSON array with
//     each list on a line of its own, followed by its products or entries,
//     one per line;
//   - merchant/price_configs.json, one config per line.
func genEconomy(o options) error {
	refs, err := readEconomyRefs(o)
	if err != nil {
		return err
	}
	var d data.EconomyData
	if d.BuyLists, err = readBuyLists(o, refs); err != nil {
		return fmt.Errorf("buylists: %w", err)
	}
	if d.Multisells, err = readMultisells(o, refs); err != nil {
		return fmt.Errorf("multisell: %w", err)
	}
	if d.PriceConfigs, err = readPriceConfigs(o); err != nil {
		return fmt.Errorf("merchant price configs: %w", err)
	}
	if _, err := data.NewEconomy(d, refs.lookups()); err != nil {
		return err
	}

	err = econAddNested(o.out, data.BuyListsFile, "products", d.BuyLists, func(l data.BuyList) (data.BuyList, []data.BuyListProduct) {
		products := l.Products
		l.Products = nil
		return l, products
	})
	if err != nil {
		return err
	}
	err = econAddNested(o.out, data.MultisellsFile, "entries", d.Multisells, func(m data.Multisell) (data.Multisell, []data.MultisellEntry) {
		entries := m.Entries
		m.Entries = nil
		return m, entries
	})
	if err != nil {
		return err
	}
	configs, err := econNested(data.PriceConfigs{Default: d.PriceConfigs.Default}, "configs", d.PriceConfigs.Configs)
	if err != nil {
		return fmt.Errorf("encode %s: %w", data.PriceConfigsFile, err)
	}
	if err := o.out.addSized(data.PriceConfigsFile, append(configs, '\n')); err != nil {
		return err
	}

	products, entries := 0, 0
	for _, l := range d.BuyLists {
		products += len(l.Products)
	}
	for _, m := range d.Multisells {
		entries += len(m.Entries)
	}
	fmt.Fprintf(o.log, "economy: %d buy lists with %d products, %d multisell lists with %d entries, %d merchant price configs\n",
		len(d.BuyLists), products, len(d.Multisells), entries, len(d.PriceConfigs.Configs))
	return nil
}

// economyRefs are the items and NPCs the economy data names, read as the
// items and world domains read them.
type economyRefs struct {
	items map[int]*data.Item
	npcs  map[int]*data.NpcTemplate
}

// readEconomyRefs reads the item templates (the buy lists need their
// reference prices) and the NPC templates. Their warnings are left to the
// items and world domains.
func readEconomyRefs(o options) (*economyRefs, error) {
	quiet := o
	quiet.log = io.Discard
	items, err := readItems(quiet)
	if err != nil {
		return nil, fmt.Errorf("items: %w", err)
	}
	npcs, err := readNpcs(quiet)
	if err != nil {
		return nil, fmt.Errorf("NPCs: %w", err)
	}
	r := &economyRefs{
		items: make(map[int]*data.Item, len(items)),
		npcs:  make(map[int]*data.NpcTemplate, len(npcs)),
	}
	for i := range items {
		r.items[items[i].ID] = &items[i]
	}
	for i := range npcs {
		r.npcs[npcs[i].ID] = &npcs[i]
	}
	return r, nil
}

// lookups returns the references for data.NewEconomy.
func (r *economyRefs) lookups() data.EconomyRefs {
	return data.EconomyRefs{
		Item: func(id int) (*data.Item, bool) {
			it, ok := r.items[id]
			return it, ok
		},
		Npc: func(id int) (*data.NpcTemplate, bool) {
			n, ok := r.npcs[id]
			return n, ok
		},
	}
}

// readBuyLists reads the buy lists, sorted by ID, as Mobius's BuyListData
// does:
//
//   - a product without a price costs the item's reference price;
//   - a price below the item's sell price (half its reference price), in a
//     list that named NPCs before it, is raised to the sell price
//     (CorrectPrices, on by default);
//   - an item listed twice keeps its first place and takes its last values
//     (the products are a LinkedHashMap);
//   - products of items that are not in the item data are left out.
//
// NPCs that are not in the NPC data are left out too: they never spawn.
func readBuyLists(o options, refs *economyRefs) ([]data.BuyList, error) {
	files, err := econListFiles(o, buyListXMLDir, "buylists")
	if err != nil {
		return nil, err
	}
	var (
		lists        []data.BuyList
		errs         []error
		missingNpcs  = make(map[int]bool)
		missingItems []string // list/item
		raised       []string
		twice        []string
	)
	for _, f := range files {
		doc, err := parseMX(o.src, f.path)
		if err != nil {
			return nil, err
		}
		root, err := mxRoot(doc)
		if err != nil {
			return nil, err
		}
		l := data.BuyList{ID: f.id}
		named := false             // Mobius has created the list's NPC set
		place := make(map[int]int) // item ID -> index in l.Products
		for _, e := range root.children() {
			switch e.name {
			case "npcs":
				for _, id := range econNpcs(e) {
					named = true
					if refs.npcs[id] == nil {
						missingNpcs[id] = true
					} else if !slices.Contains(l.NpcIDs, id) {
						l.NpcIDs = append(l.NpcIDs, id)
					}
				}
			case "item":
				id := econReqInt(e, "id")
				price := econInt(e, "price", -1)
				restock := econLong(e, "restock_delay", -1)
				count := econInt(e, "count", -1)
				it := refs.items[id]
				if it == nil {
					missingItems = append(missingItems, fmt.Sprintf("%d/%d", l.ID, id))
					continue
				}
				pr := data.BuyListProduct{ItemID: id, Price: price}
				switch sell := it.Price / 2; {
				case price < 0:
					pr.Price = it.Price
				case named && sell > price:
					pr.Price = sell
					raised = append(raised, fmt.Sprintf("%d/%d", l.ID, id))
				}
				switch {
				case count == 0:
					e.errorf("count 0 is a stock that is always empty; leave count out for an unlimited one")
				case count > 0:
					pr.Count = count
				}
				if restock > 0 {
					pr.RestockDelay = int(restock)
				}
				if i, dup := place[id]; dup {
					l.Products[i] = pr
					twice = append(twice, fmt.Sprintf("%d/%d", l.ID, id))
					continue
				}
				place[id] = len(l.Products)
				l.Products = append(l.Products, pr)
			default:
				e.unknown()
			}
		}
		slices.Sort(l.NpcIDs)
		lists = append(lists, l)
		if err := doc.check(); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	if len(missingItems) > 0 {
		o.warnf("buylists: left out %d products of items that are not in the item data, as Mobius does (list/item): %s",
			len(missingItems), econJoin(missingItems, 20))
	}
	if len(missingNpcs) > 0 {
		o.warnf("buylists: left out NPCs that are not in the NPC data: %s", joinInts(slices.Sorted(maps.Keys(missingNpcs)), 20))
	}
	if len(raised) > 0 {
		o.warnf("buylists: raised %d prices below the item's sell price to it, as Mobius's CorrectPrices does (list/item): %s",
			len(raised), econJoin(raised, 20))
	}
	if len(twice) > 0 {
		o.warnf("buylists: items listed twice keep their first place and last values, as in Mobius (list/item): %s", econJoin(twice, 20))
	}
	return lists, nil
}

// readMultisells reads the multisell lists, sorted by ID, as Mobius's
// MultisellData does: the entries are numbered from 1 in file order, and an
// entry of one adena ingredient and one product never costs less adena than
// the product's sell price times its count (CorrectPrices, on by default).
//
// An entry naming an item that is not in the item data is left out (Mobius
// shows it, then fails the exchange); the other entries keep their IDs. NPCs
// that are not in the NPC data are left out: they never spawn.
func readMultisells(o options, refs *economyRefs) ([]data.Multisell, error) {
	files, err := econListFiles(o, multisellXMLDir, "multisell")
	if err != nil {
		return nil, err
	}
	var (
		lists       []data.Multisell
		errs        []error
		missingNpcs = make(map[int]bool)
		dropped     []string // list/entry
		raised      []string
	)
	for _, f := range files {
		doc, err := parseMX(o.src, f.path)
		if err != nil {
			return nil, err
		}
		root, err := mxRoot(doc)
		if err != nil {
			return nil, err
		}
		// useRate, which Mobius reads but never uses, is left unread, so
		// datagen refuses it.
		m := data.Multisell{
			ID:                  f.id,
			ApplyTaxes:          root.boolAttr("applyTaxes", false),
			MaintainEnchantment: root.boolAttr("maintainEnchantment", false),
		}
		entryID := 0
		for _, e := range root.children() {
			switch e.name {
			case "npcs":
				for _, id := range econNpcs(e) {
					if id != data.MultisellAnyNpc && refs.npcs[id] == nil {
						missingNpcs[id] = true
					} else if !slices.Contains(m.NpcIDs, id) {
						m.NpcIDs = append(m.NpcIDs, id)
					}
				}
			case "item":
				entryID++
				en := parseMultisellEntry(e, entryID)
				if !refs.knowsItems(en) {
					dropped = append(dropped, fmt.Sprintf("%d/%d", m.ID, en.ID))
					continue
				}
				if correctMultisellPrice(&en, refs) {
					raised = append(raised, fmt.Sprintf("%d/%d", m.ID, en.ID))
				}
				m.Entries = append(m.Entries, en)
			default:
				e.unknown()
			}
		}
		slices.Sort(m.NpcIDs)
		lists = append(lists, m)
		if err := doc.check(); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	if len(dropped) > 0 {
		o.warnf("multisell: left out %d entries naming items that are not in the item data (list/entry): %s",
			len(dropped), econJoin(dropped, 20))
	}
	if len(missingNpcs) > 0 {
		o.warnf("multisell: left out NPCs that are not in the NPC data: %s", joinInts(slices.Sorted(maps.Keys(missingNpcs)), 20))
	}
	if len(raised) > 0 {
		o.warnf("multisell: raised %d adena prices below the product's sell price to it, as Mobius's CorrectPrices does (list/entry): %s",
			len(raised), econJoin(raised, 20))
	}
	return lists, nil
}

// parseMultisellEntry reads an <item> of a multisell list, as Mobius's
// parseEntry and Ingredient do. Products take no ingredient flags.
func parseMultisellEntry(e *mxElem, id int) data.MultisellEntry {
	en := data.MultisellEntry{ID: id}
	for _, c := range e.children() {
		switch c.name {
		case "ingredient":
			en.Ingredients = append(en.Ingredients, data.MultisellItem{
				ItemID:             econReqInt(c, "id"),
				Count:              int64(econReqInt(c, "count")),
				Enchant:            econInt(c, "enchantmentLevel", 0),
				TaxIngredient:      c.boolAttr("isTaxIngredient", false),
				MaintainIngredient: c.boolAttr("maintainIngredient", false),
			})
		case "production":
			en.Products = append(en.Products, data.MultisellItem{
				ItemID:  econReqInt(c, "id"),
				Count:   int64(econReqInt(c, "count")),
				Enchant: econInt(c, "enchantmentLevel", 0),
			})
		default:
			c.unknown()
		}
	}
	return en
}

// knowsItems reports whether every item of the entry is in the item data
// or is a special multisell ID.
func (r *economyRefs) knowsItems(en data.MultisellEntry) bool {
	for _, it := range slices.Concat(en.Ingredients, en.Products) {
		if !it.Special() && r.items[it.ItemID] == nil {
			return false
		}
	}
	return true
}

// correctMultisellPrice applies Mobius's CorrectPrices to an entry of one
// adena ingredient and one product: the adena count is raised to the
// product's sell price (half its reference price) times its count. The sum
// is a Java int, so it wraps on overflow as it does in Mobius. It reports
// whether the count was raised.
func correctMultisellPrice(en *data.MultisellEntry, refs *economyRefs) bool {
	if len(en.Ingredients) != 1 || len(en.Products) != 1 || en.Ingredients[0].ItemID != adenaID {
		return false
	}
	var total int32
	for _, p := range en.Products {
		it := refs.items[p.ItemID]
		if it == nil {
			continue
		}
		total += int32(p.Count) * int32(it.Price/2) //nolint:gosec // Java int arithmetic, as in Mobius
	}
	if int64(total) <= en.Ingredients[0].Count {
		return false
	}
	en.Ingredients[0].Count = int64(total)
	return true
}

// readPriceConfigs reads MerchantPriceConfig.xml as Mobius's
// MerchantPriceConfigTable does, sorted by ID. A castle or zone ID that is
// missing (Mobius -1), 0 or negative means none.
func readPriceConfigs(o options) (data.PriceConfigs, error) {
	doc, err := parseMX(o.src, priceConfigXML)
	if err != nil {
		return data.PriceConfigs{}, err
	}
	root := doc.root
	if root.name != "merchantPriceConfig" {
		return data.PriceConfigs{}, fmt.Errorf("%s: root element is <%s>, want <merchantPriceConfig>", priceConfigXML, root.name)
	}
	pc := data.PriceConfigs{Default: econReqInt(root, "defaultPriceConfig")}
	ids := make(map[int]bool)
	for _, e := range root.children() {
		if e.name != "priceConfig" {
			e.unknown()
			continue
		}
		c := data.PriceConfig{
			ID:       econReqInt(e, "id"),
			Name:     e.reqStr("name"),
			BaseTax:  econReqInt(e, "baseTax"),
			CastleID: max(econInt(e, "castleId", 0), 0),
			ZoneID:   max(econInt(e, "zoneId", 0), 0),
		}
		if ids[c.ID] {
			e.errorf("config ID %d is used twice (Mobius keeps the last)", c.ID)
			continue
		}
		ids[c.ID] = true
		pc.Configs = append(pc.Configs, c)
	}
	if err := doc.check(); err != nil {
		return data.PriceConfigs{}, err
	}
	slices.SortFunc(pc.Configs, func(a, b data.PriceConfig) int { return cmp.Compare(a.ID, b.ID) })
	return pc, nil
}

// econListFile is a list file Mobius loads.
type econListFile struct {
	path string
	id   int
}

// econListFiles returns the list files of dir, sorted by list ID, picked as
// Mobius's BuyListData and MultisellData pick them: the files directly in
// dir whose name is a number plus ".xml" (in any case), the number being
// the list ID. It warns about the folders and other XML files there, which
// Mobius does not load; other files (readme.txt) are skipped quietly.
func econListFiles(o options, dir, what string) ([]econListFile, error) {
	entries, err := fs.ReadDir(o.src, dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	var (
		files   []econListFile
		names   = make(map[int]string) // list ID -> file name
		skipped []string
	)
	for _, de := range entries {
		name := de.Name()
		if de.IsDir() {
			skipped = append(skipped, name+"/")
			continue
		}
		num, isXML := strings.CutSuffix(strings.ToLower(name), ".xml")
		if !isXML {
			continue
		}
		if num == "" || strings.Trim(num, "0123456789") != "" {
			skipped = append(skipped, name)
			continue
		}
		p := path.Join(dir, name)
		if !strings.HasSuffix(name, ".xml") {
			return nil, fmt.Errorf("%s: Mobius loads it but cannot read its ID (it strips a lower-case .xml)", p)
		}
		id, err := parseJavaInt(num)
		if err != nil {
			return nil, fmt.Errorf("%s: list ID: %w", p, err)
		}
		if prev, dup := names[id]; dup {
			return nil, fmt.Errorf("%s: list ID %d is also the ID of %s", p, id, prev)
		}
		names[id] = name
		files = append(files, econListFile{path: p, id: id})
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%s: no list files found", dir)
	}
	if len(skipped) > 0 {
		o.warnf("%s: left out %s: Mobius loads only <id>.xml files directly in %s (custom/ only with its Custom*Load option, an extra L2Go keeps off)",
			what, strings.Join(skipped, ", "), dir)
	}
	slices.SortFunc(files, func(a, b econListFile) int { return cmp.Compare(a.id, b.id) })
	return files, nil
}

// econNpcs reads an <npcs> element: the number in each <npc>.
func econNpcs(e *mxElem) []int {
	var ids []int
	for _, c := range e.children() {
		if c.name != "npc" {
			c.unknown()
			continue
		}
		id, err := parseJavaInt(c.text)
		if err != nil {
			c.errorf("text: %v", err)
		}
		ids = append(ids, id)
	}
	return ids
}

// econInt returns an attribute parsed as Java's Integer.parseInt, which
// the economy loaders of Mobius use (decimal only, unlike mxElem.intAttr),
// or def if it is missing.
func econInt(e *mxElem, name string, def int) int {
	v, ok := e.attr(name)
	if !ok {
		return def
	}
	n, err := parseJavaInt(v)
	if err != nil {
		e.errorf("attribute %s: %v", name, err)
	}
	return n
}

// econReqInt returns a required attribute parsed as Integer.parseInt.
func econReqInt(e *mxElem, name string) int {
	if _, ok := e.attr(name); !ok {
		e.errorf("missing attribute %s", name)
		return 0
	}
	return econInt(e, name, 0)
}

// econLong returns an attribute parsed as Java's Long.parseLong, or def if
// it is missing.
func econLong(e *mxElem, name string, def int64) int64 {
	v, ok := e.attr(name)
	if !ok {
		return def
	}
	n, err := parseJavaLong(v)
	if err != nil {
		e.errorf("attribute %s: %v", name, err)
	}
	return n
}

// econJoin formats up to limit strings, then how many more there are.
func econJoin(items []string, limit int) string {
	if len(items) <= limit {
		return strings.Join(items, ", ")
	}
	return fmt.Sprintf("%s and %d more", strings.Join(items[:limit], ", "), len(items)-limit)
}

// econAddNested stores groups as a JSON array with each group on a line of
// its own, its list field (named key) following with one element per line:
// big lists still diff per element. split returns a group without its list
// (see econNested) and the list.
func econAddNested[G, E any](o *output, name, key string, groups []G, split func(G) (G, []E)) error {
	var b bytes.Buffer
	b.WriteString("[")
	for i, g := range groups {
		head, elems := split(g)
		gb, err := econNested(head, key, elems)
		if err != nil {
			return fmt.Errorf("encode %s: element %d: %w", name, i, err)
		}
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString("\n")
		b.Write(gb)
	}
	if len(groups) > 0 {
		b.WriteString("\n")
	}
	b.WriteString("]\n")
	return o.addSized(name, b.Bytes())
}

// econNested encodes head as compact JSON with elems added as its field
// key, one element per line. head must leave that field out: it is the
// struct with the list emptied, and the list field is tagged omitempty.
// Without elements, head is encoded as it is.
func econNested[E any](head any, key string, elems []E) ([]byte, error) {
	h, err := marshalCompact(head)
	if err != nil {
		return nil, err
	}
	if len(elems) == 0 {
		return h, nil
	}
	if len(h) < 2 || h[0] != '{' || h[len(h)-1] != '}' {
		return nil, fmt.Errorf("%T is not encoded as a JSON object", head)
	}
	var b bytes.Buffer
	b.Write(h[:len(h)-1])
	if len(h) > 2 {
		b.WriteString(",")
	}
	b.WriteString(`"` + key + `":[`)
	for i, el := range elems {
		eb, err := marshalCompact(el)
		if err != nil {
			return nil, fmt.Errorf("%s %d: %w", key, i, err)
		}
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString("\n")
		b.Write(eb)
	}
	b.WriteString("\n]}")
	return b.Bytes(), nil
}
