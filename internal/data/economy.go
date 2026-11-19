package data

// Economy data: the NPC shops (buy lists), the multisell lists and the
// merchant price configs that cmd/datagen converts from the L2J Mobius
// Interlude datapack. It is loaded with LoadEconomy, separately from the
// player and item data of Load and the world data of LoadWorld; it checks
// its references into those when given them (see EconomyRefs).

import (
	"cmp"
	"fmt"
	"io/fs"
	"math"
	"os"
	"slices"
)

// Paths of the economy data files, relative to the data folder. cmd/datagen
// stores a file bigger than 10 MB gzipped, under its name plus ".gz"; the
// loader reads either form.
const (
	BuyListsFile     = "buylists/buylists.json"
	MultisellsFile   = "multisell/multisell.json"
	PriceConfigsFile = "merchant/price_configs.json"
)

// The special item IDs of multisell ingredients and products (Mobius
// MultisellData): points that are paid or given instead of an item.
const (
	MultisellPcCafePoints   = -100
	MultisellClanReputation = -200
	MultisellFame           = -300
)

// MultisellAnyNpc in Multisell.NpcIDs lets every NPC open the list.
const MultisellAnyNpc = -1

// maxMultisellEnchant bounds enchant levels, which the packets send as
// 16-bit numbers.
const maxMultisellEnchant = 65535

// maxPriceConfigCastle is the highest castle ID: Interlude has 9 castles,
// Gludio (1) to Schuttgart (9).
const maxPriceConfigCastle = 9

// IsSpecialMultisellItem reports whether id is one of the special multisell
// item IDs (MultisellPcCafePoints, MultisellClanReputation, MultisellFame).
func IsSpecialMultisellItem(id int) bool {
	return id == MultisellPcCafePoints || id == MultisellClanReputation || id == MultisellFame
}

// BuyList is an NPC shop, Mobius BuyListHolder, from
// dist/game/data/buylists/<id>.xml. A merchant opens it with the bypass
// "Buy <id>" (Mobius Merchant.showBuyWindow), and only the NPCs of NpcIDs
// may: a list without NPCs is opened by GM commands (the admin shops) or,
// in Mobius, by its community board (list 423).
type BuyList struct {
	// ID is the list ID, the number of its file. By convention it is the
	// ID of its NPC times 100 plus a list number (3000400 is the first list
	// of NPC 30004), but only NpcIDs decides who may open it.
	ID int `json:"id"`
	// NpcIDs are the NPCs that may open the list, sorted.
	NpcIDs []int `json:"npcIds,omitempty"`
	// Products are the items on sale, in file order (the order the shop
	// window shows).
	Products []BuyListProduct `json:"products,omitempty"`
}

// BuyListProduct is an item on sale, Mobius Product.
type BuyListProduct struct {
	ItemID int `json:"itemId"`
	// Price is the price of one item in adena before tax, as Mobius
	// Product.getPrice returns it: the list's price, or the item's
	// reference price when the list gives none. Like Mobius with
	// CorrectPrices on (its default), a list price below the item's sell
	// price (half its reference price) is raised to it in lists that name
	// NPCs. The buyer pays Price * (1 + base tax + castle tax), see
	// PriceConfig.
	Price int `json:"price"`
	// Count is the stock: how many the shop holds when full. 0 means
	// unlimited (Mobius -1).
	Count int `json:"count,omitempty"`
	// RestockDelay is the time in minutes after the first sale until the
	// stock is full again (Mobius restock_delay). Mobius refills a stock
	// without one at its next check, within a minute.
	RestockDelay int `json:"restockDelay,omitempty"`
}

// LimitedStock reports whether the product has a stock (Mobius
// hasLimitedStock).
func (p *BuyListProduct) LimitedStock() bool { return p.Count > 0 }

// AllowsNpc reports whether the NPC may open the list (Mobius
// isNpcAllowed).
func (l *BuyList) AllowsNpc(npcID int) bool {
	_, found := slices.BinarySearch(l.NpcIDs, npcID)
	return found
}

// Product returns the product of the list that sells the item.
func (l *BuyList) Product(itemID int) (*BuyListProduct, bool) {
	for i := range l.Products {
		if l.Products[i].ItemID == itemID {
			return &l.Products[i], true
		}
	}
	return nil, false
}

// Multisell is a multisell list, Mobius ListContainer, from
// dist/game/data/multisell/<id>.xml (custom/ is left out): exchanges of
// ingredients for products. An NPC opens it with the bypass "multisell
// <id>" (every entry) or "exc_multisell <id>" (only the entries whose first
// ingredient the player has, not equipped).
type Multisell struct {
	ID int `json:"id"`
	// ApplyTaxes taxes the tax ingredients when the NPC is in a town whose
	// castle has an owner (Mobius PreparedListContainer).
	ApplyTaxes bool `json:"applyTaxes,omitempty"`
	// MaintainEnchantment keeps, in the exc_multisell view, the enchant
	// level of the weapon or armor the player gives on the weapons and
	// armor the entry gives.
	MaintainEnchantment bool `json:"maintainEnchantment,omitempty"`
	// NpcIDs are the NPCs that may open the list, sorted; MultisellAnyNpc
	// lets every NPC. Without NPCs only GMs may open it (Mobius
	// MultisellData.separateAndSend).
	NpcIDs []int `json:"npcIds,omitempty"`
	// Entries are the exchanges, in file order.
	Entries []MultisellEntry `json:"entries,omitempty"`
}

// MultisellEntry is one exchange of a list, Mobius Entry.
type MultisellEntry struct {
	// ID is the Mobius entry ID: the position of the entry in its file,
	// from 1. The MultiSellList packet sends ID * 100000 (plus the enchant
	// level in the MaintainEnchantment view).
	ID          int             `json:"id"`
	Ingredients []MultisellItem `json:"ingredients,omitempty"`
	Products    []MultisellItem `json:"products"`
}

// MultisellItem is an ingredient or a product of an entry, Mobius
// Ingredient.
type MultisellItem struct {
	// ItemID is the item, or one of the special Multisell* IDs.
	ItemID int `json:"itemId"`
	// Count is how many. Like Mobius with CorrectPrices on, an entry of
	// one adena ingredient and one product never costs less adena than the
	// product's sell price (half its reference price) times its count.
	Count int64 `json:"count"`
	// Enchant is the enchant level (enchantmentLevel in the XML).
	Enchant int `json:"enchant,omitempty"`
	// TaxIngredient, on an adena ingredient only, makes the count the base
	// of the castle tax instead of a price: the player pays the tax on it
	// when the list applies taxes, nothing otherwise (Mobius
	// isTaxIngredient, PreparedEntry). Ingredients only.
	TaxIngredient bool `json:"taxIngredient,omitempty"`
	// MaintainIngredient keeps the ingredient, such as a recipe, with the
	// player, unless AltBlacksmithUseRecipes is on (the Mobius default),
	// which takes it like any other. Ingredients only.
	MaintainIngredient bool `json:"maintainIngredient,omitempty"`
}

// Special reports whether the item is one of the special multisell IDs.
func (it MultisellItem) Special() bool { return IsSpecialMultisellItem(it.ItemID) }

// AllowsNpc reports whether the NPC may open the list: it is one of
// NpcIDs, or NpcIDs holds MultisellAnyNpc. Pass 0 for a list opened without
// an NPC, which only MultisellAnyNpc allows.
func (m *Multisell) AllowsNpc(npcID int) bool {
	if _, anyNpc := slices.BinarySearch(m.NpcIDs, MultisellAnyNpc); anyNpc {
		return true
	}
	_, found := slices.BinarySearch(m.NpcIDs, npcID)
	return found
}

// Entry returns the entry with the given ID (MultisellEntry.ID).
func (m *Multisell) Entry(id int) (*MultisellEntry, bool) {
	i, found := slices.BinarySearchFunc(m.Entries, id, func(e MultisellEntry, id int) int { return cmp.Compare(e.ID, id) })
	if !found {
		return nil, false
	}
	return &m.Entries[i], true
}

// PriceConfigs are the merchant price configs, Mobius
// MerchantPriceConfigTable, from dist/game/data/MerchantPriceConfig.xml.
type PriceConfigs struct {
	// Default is the ID of the config of a merchant that is near none of
	// the configs' zones.
	Default int `json:"default"`
	// Configs are sorted by ID.
	Configs []PriceConfig `json:"configs,omitempty"`
}

// PriceConfig is a merchant tax region, Mobius MerchantPriceConfig. A
// merchant sells a product for Price * (1 + BaseTax/100 + the castle's tax
// rate), the castle's rate counting only when the castle has an owner
// (Mobius RequestBuyItem); the castle receives its part.
type PriceConfig struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// BaseTax is the tax in percent that always applies.
	BaseTax int `json:"baseTax"`
	// CastleID is the castle whose tax applies too, 0 if none (Mobius -1).
	CastleID int `json:"castleId,omitempty"`
	// ZoneID is the town zone whose merchants use the config, 0 if none.
	// The zone need not exist: the castle configs name zones the Interlude
	// zone data does not have, so Mobius uses them only as the default.
	ZoneID int `json:"zoneId,omitempty"`
}

// BaseTaxRate returns BaseTax as a rate (Mobius getBaseTaxRate).
func (c *PriceConfig) BaseTaxRate() float64 { return float64(c.BaseTax) / 100 }

// EconomyData is the content of the economy data files, as decoded from
// JSON.
type EconomyData struct {
	BuyLists     []BuyList    // BuyListsFile, sorted by ID
	Multisells   []Multisell  // MultisellsFile, sorted by ID
	PriceConfigs PriceConfigs // PriceConfigsFile
}

// economyFiles pairs each economy file with the field it decodes into.
func (d *EconomyData) economyFiles() []struct {
	name string
	v    any
} {
	return []struct {
		name string
		v    any
	}{
		{BuyListsFile, &d.BuyLists},
		{MultisellsFile, &d.Multisells},
		{PriceConfigsFile, &d.PriceConfigs},
	}
}

// Economy is the checked economy data with its lookups. The exported
// slices keep the order of the files. An Economy must not be modified after
// NewEconomy or LoadEconomy returns, and is then safe for concurrent use;
// the pointers and slices its lookups return point into it and must not be
// modified either.
type Economy struct {
	EconomyData

	buyLists      map[int]int          // list ID -> index in BuyLists
	npcBuyLists   map[int][]*BuyList   // NPC ID -> the lists naming it, by ID
	multisells    map[int]int          // list ID -> index in Multisells
	npcMultisells map[int][]*Multisell // NPC ID -> the lists naming it, by ID
	priceConfigs  map[int]int          // config ID -> index in PriceConfigs.Configs
}

// EconomyRefs looks up what the economy data refers to, so NewEconomy can
// check the references. A nil function skips its check. NewEconomyRefs
// makes them from loaded data.
type EconomyRefs struct {
	// Item returns the item template with the given ID, like
	// Data.ItemTemplate.
	Item func(id int) (*Item, bool)
	// Npc returns the NPC template with the given ID, like World.Npc.
	Npc func(id int) (*NpcTemplate, bool)
}

// NewEconomyRefs returns the references into the items of d and the NPCs
// of w. A nil d, a d loaded without the items domain and a nil w leave
// their check out.
func NewEconomyRefs(d *Data, w *World) EconomyRefs {
	var r EconomyRefs
	if d != nil && d.Loaded("items") {
		r.Item = d.ItemTemplate
	}
	if w != nil {
		r.Npc = w.Npc
	}
	return r
}

// LoadEconomy reads and checks the economy data files in dir; see
// NewEconomy for refs.
func LoadEconomy(dir string, refs EconomyRefs) (*Economy, error) {
	e, err := LoadEconomyFS(os.DirFS(dir), refs)
	if err != nil {
		return nil, fmt.Errorf("%w (data folder %s)", err, dir)
	}
	return e, nil
}

// LoadEconomyFS reads and checks the economy data files in fsys. Each file
// may be stored as name.json or name.json.gz (see readJSON).
func LoadEconomyFS(fsys fs.FS, refs EconomyRefs) (*Economy, error) {
	var d EconomyData
	for _, f := range d.economyFiles() {
		if err := readJSON(fsys, f.name, f.v); err != nil {
			return nil, err
		}
	}
	return NewEconomy(d, refs)
}

// NewEconomy checks the economy data and builds its lookups. The slices are
// used as they are, not copied. Each part is checked (see checkBuyLists,
// checkMultisells and checkPriceConfigs), and so are the references into
// the items and NPCs that refs can look up.
//
// All problems are reported together in one error wrapping ErrInvalid
// (with a long list cut short).
func NewEconomy(d EconomyData, refs EconomyRefs) (*Economy, error) {
	e := &Economy{
		EconomyData:   d,
		buyLists:      make(map[int]int, len(d.BuyLists)),
		npcBuyLists:   make(map[int][]*BuyList),
		multisells:    make(map[int]int, len(d.Multisells)),
		npcMultisells: make(map[int][]*Multisell),
		priceConfigs:  make(map[int]int, len(d.PriceConfigs.Configs)),
	}
	var p problems
	e.checkBuyLists(&p, refs)
	e.checkMultisells(&p, refs)
	e.checkPriceConfigs(&p)
	if err := p.err(); err != nil {
		return nil, err
	}
	return e, nil
}

// BuyList returns the buy list with the given ID.
func (e *Economy) BuyList(id int) (*BuyList, bool) {
	i, ok := e.buyLists[id]
	if !ok {
		return nil, false
	}
	return &e.BuyLists[i], true
}

// NpcBuyLists returns the buy lists the NPC may open, sorted by ID, or nil.
func (e *Economy) NpcBuyLists(npcID int) []*BuyList { return e.npcBuyLists[npcID] }

// Multisell returns the multisell list with the given ID.
func (e *Economy) Multisell(id int) (*Multisell, bool) {
	i, ok := e.multisells[id]
	if !ok {
		return nil, false
	}
	return &e.Multisells[i], true
}

// NpcMultisells returns the multisell lists that name the NPC, sorted by
// ID, or nil. The lists that allow every NPC (MultisellAnyNpc) are not
// included.
func (e *Economy) NpcMultisells(npcID int) []*Multisell { return e.npcMultisells[npcID] }

// PriceConfig returns the merchant price config with the given ID.
func (e *Economy) PriceConfig(id int) (*PriceConfig, bool) {
	i, ok := e.priceConfigs[id]
	if !ok {
		return nil, false
	}
	return &e.PriceConfigs.Configs[i], true
}

// DefaultPriceConfig returns the config of the merchants near none of the
// configs' zones.
func (e *Economy) DefaultPriceConfig() *PriceConfig {
	c, _ := e.PriceConfig(e.PriceConfigs.Default)
	return c
}

// PriceConfigFor returns the price config of a merchant as Mobius picks it
// when the merchant spawns (MerchantPriceConfigTable): the first config, by
// ID, whose zone near reports, else the default. Mobius asks whether the
// zone touches the 32768x32768 zone region the merchant stands in, so near
// should answer that, not whether the merchant is inside the zone.
func (e *Economy) PriceConfigFor(near func(zoneID int) bool) *PriceConfig {
	for i := range e.PriceConfigs.Configs {
		c := &e.PriceConfigs.Configs[i]
		if c.ZoneID > 0 && near(c.ZoneID) {
			return c
		}
	}
	return e.DefaultPriceConfig()
}

// checkBuyLists checks the buy lists and indexes them:
//
//   - there are lists, and list IDs are unique and not negative;
//   - NPC IDs are positive and unique within a list, and name existing
//     NPCs (with refs.Npc);
//   - item IDs are positive and unique within a list, and name existing
//     items (with refs.Item);
//   - prices, counts and restock delays are not negative, and prices and
//     counts fit the 32-bit numbers of the packets.
func (e *Economy) checkBuyLists(p *problems, refs EconomyRefs) {
	if len(e.BuyLists) == 0 {
		p.addf("%s: no buy lists", BuyListsFile)
	}
	for i := range e.BuyLists {
		l := &e.BuyLists[i]
		where := fmt.Sprintf("%s: list %d", BuyListsFile, l.ID)
		if l.ID < 0 {
			p.addf("%s: list ID must not be negative", where)
			continue
		}
		if _, dup := e.buyLists[l.ID]; dup {
			p.addf("%s: duplicate list ID", where)
			continue
		}
		e.buyLists[l.ID] = i
		for _, id := range checkEconomyNpcs(p, where, l.NpcIDs, false, refs) {
			e.npcBuyLists[id] = append(e.npcBuyLists[id], l)
		}
		items := make(map[int]bool, len(l.Products))
		for _, pr := range l.Products {
			at := fmt.Sprintf("%s: item %d", where, pr.ItemID)
			switch {
			case pr.ItemID <= 0:
				p.addf("%s: item ID must be positive", at)
			case items[pr.ItemID]:
				p.addf("%s: listed twice", at)
			case refs.Item != nil:
				if _, ok := refs.Item(pr.ItemID); !ok {
					p.addf("%s: does not exist in %s", at, ItemsFile)
				}
			}
			items[pr.ItemID] = true
			if pr.Price < 0 || pr.Price > math.MaxInt32 {
				p.addf("%s: price %d is out of range 0-%d", at, pr.Price, math.MaxInt32)
			}
			if pr.Count < 0 || pr.Count > math.MaxInt32 || pr.RestockDelay < 0 {
				p.addf("%s: count %d and restock delay %d must not be negative", at, pr.Count, pr.RestockDelay)
			}
		}
	}
	for _, lists := range e.npcBuyLists {
		slices.SortFunc(lists, func(a, b *BuyList) int { return cmp.Compare(a.ID, b.ID) })
	}
}

// checkMultisells checks the multisell lists and indexes them:
//
//   - there are lists, and list IDs are unique and not negative;
//   - NPC IDs are positive (or MultisellAnyNpc) and unique within a list,
//     and name existing NPCs (with refs.Npc);
//   - entry IDs are positive and increase, and each entry has products;
//   - item IDs are positive or special, and name existing items (with
//     refs.Item); counts are positive and fit 32 bits, enchant levels are
//     in 0-65535, and only ingredients are tax or maintained ingredients.
func (e *Economy) checkMultisells(p *problems, refs EconomyRefs) {
	if len(e.Multisells) == 0 {
		p.addf("%s: no multisell lists", MultisellsFile)
	}
	for i := range e.Multisells {
		m := &e.Multisells[i]
		where := fmt.Sprintf("%s: list %d", MultisellsFile, m.ID)
		if m.ID < 0 {
			p.addf("%s: list ID must not be negative", where)
			continue
		}
		if _, dup := e.multisells[m.ID]; dup {
			p.addf("%s: duplicate list ID", where)
			continue
		}
		e.multisells[m.ID] = i
		for _, id := range checkEconomyNpcs(p, where, m.NpcIDs, true, refs) {
			e.npcMultisells[id] = append(e.npcMultisells[id], m)
		}
		last := 0
		for _, en := range m.Entries {
			at := fmt.Sprintf("%s: entry %d", where, en.ID)
			if en.ID <= last {
				p.addf("%s: entry IDs must be positive and increasing (after %d)", at, last)
			}
			last = max(last, en.ID)
			if len(en.Products) == 0 {
				p.addf("%s: no products", at)
			}
			for _, it := range en.Ingredients {
				checkMultisellItem(p, at+": ingredient", it, true, refs)
			}
			for _, it := range en.Products {
				checkMultisellItem(p, at+": product", it, false, refs)
			}
		}
	}
	for _, lists := range e.npcMultisells {
		slices.SortFunc(lists, func(a, b *Multisell) int { return cmp.Compare(a.ID, b.ID) })
	}
}

// checkMultisellItem checks an ingredient or a product of a multisell
// entry.
func checkMultisellItem(p *problems, where string, it MultisellItem, ingredient bool, refs EconomyRefs) {
	at := fmt.Sprintf("%s %d", where, it.ItemID)
	switch {
	case it.Special():
	case it.ItemID <= 0:
		p.addf("%s: item ID must be positive or a special multisell ID", at)
	case refs.Item != nil:
		if _, ok := refs.Item(it.ItemID); !ok {
			p.addf("%s: does not exist in %s", at, ItemsFile)
		}
	}
	if it.Count <= 0 || it.Count > math.MaxInt32 {
		p.addf("%s: count %d is out of range 1-%d", at, it.Count, math.MaxInt32)
	}
	if it.Enchant < 0 || it.Enchant > maxMultisellEnchant {
		p.addf("%s: enchant level %d is out of range 0-%d", at, it.Enchant, maxMultisellEnchant)
	}
	if !ingredient && (it.TaxIngredient || it.MaintainIngredient) {
		p.addf("%s: only ingredients can be tax or maintained ingredients", at)
	}
}

// checkEconomyNpcs checks the NPC IDs of a list, which must be sorted,
// positive (or MultisellAnyNpc if anyNpc) and unique, and name existing
// NPCs when refs can tell. It returns the IDs of real NPCs to index.
func checkEconomyNpcs(p *problems, where string, ids []int, anyNpc bool, refs EconomyRefs) []int {
	var npcs []int
	for i, id := range ids {
		switch {
		case i > 0 && id <= ids[i-1]:
			p.addf("%s: NPC IDs must be sorted and unique (%d after %d)", where, id, ids[i-1])
			continue
		case anyNpc && id == MultisellAnyNpc:
			continue
		case id <= 0:
			p.addf("%s: NPC %d: NPC IDs must be positive", where, id)
			continue
		case refs.Npc != nil:
			if _, ok := refs.Npc(id); !ok {
				p.addf("%s: NPC %d does not exist in %s", where, id, NpcsFile)
			}
		}
		npcs = append(npcs, id)
	}
	return npcs
}

// checkPriceConfigs checks the merchant price configs and indexes them:
// there are configs, sorted by unique ID, with names; taxes and zone IDs
// are not negative, castle IDs are in 0-9; the default config exists.
func (e *Economy) checkPriceConfigs(p *problems) {
	c := &e.PriceConfigs
	if len(c.Configs) == 0 {
		p.addf("%s: no price configs", PriceConfigsFile)
		return
	}
	for i := range c.Configs {
		pc := &c.Configs[i]
		where := fmt.Sprintf("%s: config %d (%q)", PriceConfigsFile, pc.ID, pc.Name)
		if i > 0 && pc.ID <= c.Configs[i-1].ID {
			p.addf("%s: IDs must be unique and increasing (after %d)", where, c.Configs[i-1].ID)
			continue
		}
		e.priceConfigs[pc.ID] = i
		if pc.Name == "" {
			p.addf("%s: no name", where)
		}
		if pc.BaseTax < 0 || pc.ZoneID < 0 {
			p.addf("%s: base tax %d and zone ID %d must not be negative", where, pc.BaseTax, pc.ZoneID)
		}
		if pc.CastleID < 0 || pc.CastleID > maxPriceConfigCastle {
			p.addf("%s: castle %d is out of range 0-%d", where, pc.CastleID, maxPriceConfigCastle)
		}
	}
	if _, ok := e.priceConfigs[c.Default]; !ok {
		p.addf("%s: the default config %d does not exist", PriceConfigsFile, c.Default)
	}
}
