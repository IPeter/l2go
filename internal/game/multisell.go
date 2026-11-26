package game

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/world"
)

// The multisell feature (multisell*.go, Plan 08 Phase 16, decision I10):
// the bypasses "multisell <id>" and "exc_multisell <id>" open a multisell
// list in MultiSellList pages (Mobius MultisellData.separateAndSend), and
// MultiSellChoose exchanges ingredients for products (Mobius
// MultiSellChoose). Unlike Mobius, every check runs before anything is
// taken, so an exchange is all or nothing.
func init() { registerFeature("multisell", setupMultisell) }

// multisellEntryFactor multiplies a list entry's ID into the ID the client
// sees (Mobius PreparedEntry).
const multisellEntryFactor = 100_000

// System messages of the exchanges (multisell, warehouse).
const (
	exchangeMsgNotClanMember     serverpackets.SystemMessageID = 212  // "You are not a clan member and cannot perform this action."
	exchangeMsgAcquiredEnchanted serverpackets.SystemMessageID = 371  // "Acquired +$s1 $s2."
	exchangeMsgWeightLimit       serverpackets.SystemMessageID = 422  // "You have exceeded the weight limit."
	exchangeMsgQuantityExceeded  serverpackets.SystemMessageID = 1036 // "You have exceeded the quantity that can be inputted."
	exchangeMsgShortOfPoints     serverpackets.SystemMessageID = 1710 // "You are short of accumulated points."
)

// multisellFeature is the multisell state on the Server (Server.multisell).
// Loop-only.
type multisellFeature struct {
	players map[*Player]*multisellState
}

// multisellState is a player's multisell state.
type multisellState struct {
	list *preparedMultisell // the open list (Mobius Player.getMultiSell), nil if none
	next time.Time          // flood protection of MultiSellChoose
}

// preparedMultisell is a multisell list as one player opened it (Mobius
// PreparedListContainer).
type preparedMultisell struct {
	tmpl *data.Multisell
	// npc is the NPC the list was opened at, 0 for none: MultiSellChoose
	// needs the same one (Mobius checkNpcObjectId).
	npc                 world.ObjectID
	maintainEnchantment bool
	entries             []multisellEntry
}

// multisellEntry is one prepared exchange (Mobius PreparedEntry).
type multisellEntry struct {
	id int32 // the entry ID the client sends back
	// stackable is false when a product does not stack: then only one
	// exchange at a time (Mobius Entry.isStackable).
	stackable   bool
	ingredients []multisellItem // adena merged and last
	products    []multisellItem
}

// multisellItem is a prepared ingredient or product (Mobius Ingredient).
type multisellItem struct {
	itemID  int32
	count   int64
	enchant int32      // the level the entry shows and checks
	tmpl    *data.Item // nil for the special IDs
}

// stackable reports whether the item stacks; an item without a template
// counts as stackable (Mobius Ingredient.isStackable).
func (it multisellItem) stackable() bool { return it.tmpl == nil || it.tmpl.Stackable }

// special reports whether the item is one of the special multisell IDs
// (points instead of an item).
func (it multisellItem) special() bool { return it.itemID < 0 }

// armorOrWeapon reports whether t is a weapon or an armor (Mobius
// Ingredient.isArmorOrWeapon).
func armorOrWeapon(t *data.Item) bool { return t != nil && (t.Weapon != nil || t.Armor != nil) }

// setupMultisell registers MultiSellChoose and the player hooks.
func setupMultisell(s *Server) error {
	s.multisell.players = make(map[*Player]*multisellState)
	s.OnPlayerLeave(func(p *Player) { delete(s.multisell.players, p) })
	return errors.Join(
		s.Register(clientpackets.OpMultiSellChoose, s.handleMultiSellChoose),
	)
}

// multisellStateOf returns p's multisell state, made on first use.
func (s *Server) multisellStateOf(p *Player) *multisellState {
	st := s.multisell.players[p]
	if st == nil {
		st = &multisellState{}
		s.multisell.players[p] = st
	}
	return st
}

// closeMultisell forgets p's open list (Mobius setMultiSell(null)).
func (s *Server) closeMultisell(p *Player) {
	if st := s.multisell.players[p]; st != nil {
		st.list = nil
	}
}

// bypassMultisell handles "multisell <id>" and "exc_multisell <id>" (Mobius
// bypass handler Multisell): the list, all of it or only the entries whose
// first ingredient p carries (exc_multisell). A bad list ID is logged.
func (s *Server) bypassMultisell(p *Player, n *Npc, word, command string) {
	listID, err := strconv.Atoi(strings.TrimSpace(command[len(word):]))
	if err != nil {
		s.log.Warn("game: bad multisell bypass", append(p.logArgs(), "bypass", command)...)
		return
	}
	s.OpenMultisell(p, n, listID, word == "exc_multisell")
}

// OpenMultisell sends p the multisell list in MultiSellList pages and keeps
// it open for MultiSellChoose (Mobius MultisellData.separateAndSend). A list
// opens only at an NPC its <npcs> name, unless it names -1; a GM opens any
// list, with a warning. inventoryOnly shows only the entries whose first
// ingredient p carries, not worn (exc_multisell). n may be nil. Loop-only.
func (s *Server) OpenMultisell(p *Player, n *Npc, listID int, inventoryOnly bool) {
	var tmpl *data.Multisell
	ok := false
	if s.economy != nil {
		tmpl, ok = s.economy.Multisell(listID)
	}
	if !ok {
		s.log.Warn("game: unknown multisell list", append(p.logArgs(), "list", listID, "npc", npcTemplateID(n))...)
		return
	}
	if !tmpl.AllowsNpc(data.MultisellAnyNpc) && (n == nil || n.Template == nil || !tmpl.AllowsNpc(n.Template.ID)) {
		if !p.IsGM() {
			s.log.Warn("game: multisell list not allowed at this NPC", append(p.logArgs(), "list", listID, "npc", npcTemplateID(n))...)
			return
		}
		p.SendMessage(multisellRestrictedText(listID))
	}
	list := s.prepareMultisell(tmpl, p, n, inventoryOnly)
	for _, page := range serverpackets.NewMultiSellListPages(list.packet()) {
		p.Send(page)
	}
	s.multisellStateOf(p).list = list
}

// npcTemplateID returns the template ID of n for the log, 0 for none.
func npcTemplateID(n *Npc) int {
	if n == nil || n.Template == nil {
		return 0
	}
	return n.Template.ID
}

// multisellRestrictedText is the warning a GM gets for a list it may use
// only as a GM.
func multisellRestrictedText(listID int) string {
	return "Multisell " + strconv.Itoa(listID) + " is restricted. Under current conditions cannot be used. Only GMs are allowed to use it."
}

// prepareMultisell prepares the list for p (Mobius PreparedListContainer):
// every entry, or with inventoryOnly the entries whose first ingredient p
// carries unworn, keeping that item's enchant level on the weapons and
// armor of a list with maintainEnchantment. Without castles no tax applies.
func (s *Server) prepareMultisell(tmpl *data.Multisell, p *Player, n *Npc, inventoryOnly bool) *preparedMultisell {
	list := &preparedMultisell{tmpl: tmpl, maintainEnchantment: tmpl.MaintainEnchantment}
	if n != nil {
		list.npc = n.ID()
	}
	if !inventoryOnly {
		for i := range tmpl.Entries {
			list.entries = append(list.entries, s.prepareMultisellEntry(&tmpl.Entries[i], nil, false))
		}
		return list
	}
	// Mobius getUniqueItems / getUniqueItemsByEnchantLevel: the first item
	// of each template (and enchant level), adena left out, worn or not.
	var unique []*inventory.Item
	for _, it := range p.inv.Items() {
		if it.ItemID() == inventory.AdenaID || it.ItemID() == inventory.AncientAdenaID {
			continue
		}
		dup := false
		for _, u := range unique {
			if u.ItemID() == it.ItemID() && (!list.maintainEnchantment || u.Enchant() == it.Enchant()) {
				dup = true
				break
			}
		}
		if !dup {
			unique = append(unique, it)
		}
	}
	for i := range tmpl.Entries {
		e := &tmpl.Entries[i]
		if len(e.Ingredients) == 0 {
			continue
		}
		first := e.Ingredients[0].ItemID
		for _, it := range unique {
			if !it.Equipped() && int(it.ItemID()) == first {
				list.entries = append(list.entries, s.prepareMultisellEntry(e, it, list.maintainEnchantment))
				break
			}
		}
	}
	return list
}

// prepareMultisellEntry prepares an entry (Mobius PreparedEntry): the ID
// times 100,000 (plus the enchant level of item when maintainEnchantment),
// the adena ingredients merged into one listed last (tax ingredients cost
// nothing without a castle owner), and the enchant levels: item's on the
// weapons and armor with maintainEnchantment, a product's default level
// otherwise.
func (s *Server) prepareMultisellEntry(e *data.MultisellEntry, item *inventory.Item, maintainEnchantment bool) multisellEntry {
	out := multisellEntry{id: int32(min(e.ID, math.MaxInt32/multisellEntryFactor) * multisellEntryFactor), stackable: true} //nolint:gosec // clamped
	if maintainEnchantment && item != nil {
		out.id += item.Enchant()
	}
	var adena int64
	var kept *int32 // the enchant level of item, once a weapon or armor ingredient took it
	for _, ing := range e.Ingredients {
		if ing.ItemID == adenaID {
			if !ing.TaxIngredient {
				adena += ing.Count
			}
			continue
		}
		mi := s.multisellItem(ing)
		if maintainEnchantment && item != nil && armorOrWeapon(mi.tmpl) {
			level := item.Enchant()
			kept = &level
			mi.enchant = level
		}
		out.ingredients = append(out.ingredients, mi)
	}
	if adena > 0 {
		out.ingredients = append(out.ingredients, s.multisellItem(data.MultisellItem{ItemID: adenaID, Count: adena}))
	}
	for _, pr := range e.Products {
		mi := s.multisellItem(pr)
		if !mi.stackable() {
			out.stackable = false
		}
		switch {
		case maintainEnchantment && armorOrWeapon(mi.tmpl):
			if kept != nil {
				mi.enchant = *kept
			}
		case armorOrWeapon(mi.tmpl) && mi.tmpl.DefaultEnchant > 0:
			mi.enchant = int32(mi.tmpl.DefaultEnchant) //nolint:gosec // small template values
		}
		out.products = append(out.products, mi)
	}
	return out
}

// multisellItem returns the prepared item of an ingredient or product.
func (s *Server) multisellItem(it data.MultisellItem) multisellItem {
	mi := multisellItem{itemID: int32(it.ItemID), count: it.Count, enchant: int32(it.Enchant)} //nolint:gosec // checked by the data loader
	if it.ItemID > 0 {
		if t, ok := s.data.ItemTemplate(it.ItemID); ok {
			mi.tmpl = t
		}
	}
	return mi
}

// entry returns the prepared entry with the client's entry ID.
func (l *preparedMultisell) entry(id int32) *multisellEntry {
	for i := range l.entries {
		if l.entries[i].id == id {
			return &l.entries[i]
		}
	}
	return nil
}

// packet returns the list as MultiSellList writes it.
func (l *preparedMultisell) packet() serverpackets.MultiSellList {
	out := serverpackets.MultiSellList{ListID: int32(l.tmpl.ID), Entries: make([]serverpackets.MultiSellEntry, 0, len(l.entries))} //nolint:gosec // list IDs fit in int32
	for _, e := range l.entries {
		pe := serverpackets.MultiSellEntry{EntryID: e.id}
		for _, pr := range e.products {
			pe.Products = append(pe.Products, pr.packetItem())
		}
		for _, ing := range e.ingredients {
			pe.Ingredients = append(pe.Ingredients, ing.packetItem())
		}
		out.Entries = append(out.Entries, pe)
	}
	return out
}

// packetItem returns the item as MultiSellList writes it: the display ID,
// body part and type 2 of its template, or its own ID without one.
func (it multisellItem) packetItem() serverpackets.MultiSellItem {
	out := serverpackets.MultiSellItem{
		ItemID:  it.itemID,
		Count:   itemCount32(it.count),
		Enchant: int16(min(max(it.enchant, math.MinInt16), math.MaxInt16)), //nolint:gosec // clamped
	}
	if it.tmpl == nil {
		out.NoTemplate = true
		return out
	}
	//nolint:gosec // display IDs, masks and list types fit
	out.ItemID, out.BodyPart, out.Type2 = int32(it.tmpl.DisplayID), int32(it.tmpl.BodyPartMask), int16(it.tmpl.Type2)
	return out
}

// handleMultiSellChoose handles MultiSellChoose (0xA7) in Mobius's order: a
// flood click is ignored; a bad amount or another list closes the list; the
// NPC must be the one the list opened at, allowed and within 250 (a GM is
// warned and goes on); then the exchange.
func (s *Server) handleMultiSellChoose(p *Player, body []byte) {
	req, err := clientpackets.ParseMultiSellChoose(body)
	if err != nil {
		s.log.Debug("game: bad MultiSellChoose", append(p.logArgs(), "err", err)...)
		return
	}
	st := s.multisellStateOf(p)
	if s.flood(&st.next, time.Duration(s.gs.Economy.FloodMultisell)) {
		return // the open list stays valid for the next click
	}
	if req.Amount < 1 || int64(req.Amount) > int64(s.gs.Economy.MultisellAmountLimit) {
		st.list = nil
		return
	}
	list := st.list
	if list == nil || int32(list.tmpl.ID) != req.ListID { //nolint:gosec // list IDs fit in int32
		st.list = nil
		return
	}
	if !list.tmpl.AllowsNpc(data.MultisellAnyNpc) && !s.multisellNpcOK(p, list) {
		if !p.IsGM() {
			st.list = nil
			return
		}
		p.SendMessage(multisellRestrictedText(list.tmpl.ID))
	}
	e := list.entry(req.EntryID)
	if e == nil {
		return
	}
	s.multisellExchange(p, list, e, int64(req.Amount))
}

// multisellNpcOK reports whether p's last NPC may serve the list: it is
// allowed, is the NPC the list opened at, is in p's instance and within
// interaction range in 3D.
func (s *Server) multisellNpcOK(p *Player, list *preparedMultisell) bool {
	if p.LastNpc == 0 {
		return false
	}
	n := s.Npc(p.LastNpc)
	return n != nil && n.Template != nil && list.tmpl.AllowsNpc(n.Template.ID) &&
		(list.npc == 0 || list.npc == n.ID()) &&
		p.InstanceID() == n.InstanceID() &&
		p.Position().InRange3D(n.Position(), s.npcInteractionRange())
}

// multisellTake is one item an exchange takes, and how many of it. exact is
// set for an item taken at exactly its ingredient's enchant level, whose
// augmentation goes to a product (Mobius MultiSellChoose).
type multisellTake struct {
	it    *inventory.Item
	count int64
	exact bool
}

// multisellExchange makes amount exchanges of entry e (Mobius
// MultiSellChoose after the list checks): the amount rule, weight, slots,
// the merged ingredient counts; then it takes the ingredients (worn items
// never, the lowest enchant level first unless the level matters) and gives
// the products, with Mobius's messages, ItemList and StatusUpdate. Every
// check runs before anything changes.
func (s *Server) multisellExchange(p *Player, list *preparedMultisell, e *multisellEntry, amount int64) {
	if !e.stackable && amount > 1 {
		s.log.Warn("game: multisell amount above 1 for an entry that does not stack", append(p.logArgs(), "list", list.tmpl.ID, "entry", e.id)...)
		s.closeMultisell(p)
		return
	}
	var slots, weight int64
	for _, pr := range e.products {
		if pr.special() {
			continue
		}
		switch {
		case !pr.stackable():
			slots += pr.count * amount
		case p.inv.ByItemID(pr.itemID) == nil:
			slots++
		}
		if pr.tmpl != nil {
			weight += pr.count * amount * int64(pr.tmpl.Weight)
		}
	}
	if !p.inv.ValidateWeight(weight) {
		p.SendSystemMessage(exchangeMsgWeightLimit)
		return
	}
	if !p.inv.ValidateCapacity(slots) {
		p.SendSystemMessage(sysMsgInventoryFull)
		return
	}
	merged, ok := mergeMultisellIngredients(e.ingredients)
	if !ok {
		p.SendSystemMessage(exchangeMsgQuantityExceeded)
		return
	}
	for _, ing := range merged {
		need := ing.count * amount
		if need > math.MaxInt32 {
			p.SendSystemMessage(exchangeMsgQuantityExceeded)
			return
		}
		if ing.special() {
			s.refuseSpecialIngredient(p, ing.itemID)
			return
		}
		// AltBlacksmithUseRecipes is on (Mobius default), so a
		// maintainIngredient ingredient is taken like any other.
		enchant := int32(-1)
		if list.maintainEnchantment || ing.enchant > 0 {
			enchant = ing.enchant
		}
		if p.inv.Count(ing.itemID, enchant, false) < need {
			p.SendMessage(strconv.FormatInt(need, 10) + " unit(s) of the item " + ing.name() + " is/are required.")
			return
		}
	}
	for _, pr := range e.products {
		if pr.special() {
			// Points (PC café, clan reputation, fame) have no owner yet;
			// Mobius would take the ingredients and give them.
			s.log.Warn("game: multisell entry gives points, refused", append(p.logArgs(), "list", list.tmpl.ID, "entry", e.id, "item", pr.itemID)...)
			return
		}
	}
	plan, ok := s.planMultisellTake(p, list, e, amount)
	if !ok {
		s.log.Warn("game: multisell ingredients cannot be taken", append(p.logArgs(), "list", list.tmpl.ID, "entry", e.id)...)
		s.closeMultisell(p)
		return
	}
	// The augmentations of the items taken at an exact enchant level, in the
	// order taken, go to the products (Mobius's augmentation list).
	var augs []int32
	for _, t := range plan {
		if t.exact && t.it.Augmented() {
			augs = append(augs, t.it.Augmentation())
		}
		if !s.DestroyItem(p, t.it, t.count, true) {
			s.log.Error("game: multisell ingredient not destroyed", append(p.logArgs(), "itemObjectId", t.it.ObjectID(), "count", t.count)...)
			s.closeMultisell(p)
			return
		}
	}
	s.giveMultisellProducts(p, list, e, amount, augs)
	s.SendItemList(p, false)
	p.Send(loadStatusUpdate(p))
}

// name returns the template's name, or the item ID without one.
func (it multisellItem) name() string {
	if it.tmpl != nil {
		return it.tmpl.Name
	}
	return strconv.Itoa(int(it.itemID))
}

// mergeMultisellIngredients merges the ingredients of one item and enchant
// level into one (Mobius: "2 swords = 1 dual"). It fails when a merged
// count passes the int range.
func mergeMultisellIngredients(ings []multisellItem) ([]multisellItem, bool) {
	out := make([]multisellItem, 0, len(ings))
	for _, ing := range ings {
		found := false
		for i := len(out) - 1; i >= 0; i-- {
			if out[i].itemID == ing.itemID && out[i].enchant == ing.enchant {
				if out[i].count+ing.count > math.MaxInt32 {
					return nil, false
				}
				out[i].count += ing.count
				found = true
				break
			}
		}
		if !found {
			out = append(out, ing)
		}
	}
	return out, true
}

// refuseSpecialIngredient tells p why it lacks the points a special
// ingredient asks for (Mobius MultisellData.hasSpecialIngredient). There are
// no PC café points, clans or fame yet, so p never has them.
func (s *Server) refuseSpecialIngredient(p *Player, id int32) {
	switch id {
	case data.MultisellPcCafePoints:
		p.SendSystemMessage(exchangeMsgShortOfPoints)
	case data.MultisellClanReputation:
		p.SendSystemMessage(exchangeMsgNotClanMember)
	case data.MultisellFame:
		p.SendMessage("You don't have enough reputation to do that.")
	}
}

// planMultisellTake picks the items the exchange takes, in the order of the
// entry's ingredients, without changing anything (Mobius's take loop): a
// stack from the first stack of the item; items that do not stack one by
// one, unworn, of exactly the ingredient's level when the list keeps
// enchant levels or the level is above 0, else the lowest level first. It
// fails when the inventory cannot cover them all.
func (s *Server) planMultisellTake(p *Player, list *preparedMultisell, e *multisellEntry, amount int64) ([]multisellTake, bool) {
	var plan []multisellTake
	taken := make(map[*inventory.Item]int64)
	for _, ing := range e.ingredients {
		need := ing.count * amount
		first := p.inv.ByItemID(ing.itemID)
		if first == nil {
			return nil, false
		}
		if first.Stackable() {
			if first.Count()-taken[first] < need {
				return nil, false
			}
			taken[first] += need
			plan = append(plan, multisellTake{it: first, count: need})
			continue
		}
		exact := list.maintainEnchantment || ing.enchant > 0
		for range need {
			var pick *inventory.Item
			for _, it := range p.inv.Items() {
				if it.ItemID() != ing.itemID || it.Equipped() || taken[it] > 0 || (exact && it.Enchant() != ing.enchant) {
					continue
				}
				if pick == nil || it.Enchant() < pick.Enchant() {
					pick = it
				}
				if exact || pick.Enchant() == 0 {
					break
				}
			}
			if pick == nil {
				return nil, false
			}
			taken[pick] = 1
			plan = append(plan, multisellTake{it: pick, count: 1, exact: exact})
		}
	}
	return plan, true
}

// giveMultisellProducts gives the products of amount exchanges with Mobius's
// messages: a stack at once, other items one by one, at the entry's enchant
// level when the list keeps enchant levels or the level is above 0; then the
// i-th item of each such product gets the i-th of augs, the augmentations of
// the ingredients taken (Mobius MultiSellChoose).
func (s *Server) giveMultisellProducts(p *Player, list *preparedMultisell, e *multisellEntry, amount int64, augs []int32) {
	for _, pr := range e.products {
		n := pr.count * amount
		if pr.stackable() {
			if _, err := s.AddItem(p, pr.itemID, n, -1, false, false); err != nil {
				s.log.Error("game: multisell product not given", append(p.logArgs(), "item", pr.itemID, "count", n, "err", err)...)
			}
		} else {
			keep := list.maintainEnchantment || pr.enchant > 0
			enchant := int32(-1)
			if keep {
				enchant = pr.enchant
			}
			for i := range n {
				it, err := s.AddItem(p, pr.itemID, 1, enchant, false, false)
				if err != nil {
					s.log.Error("game: multisell product not given", append(p.logArgs(), "item", pr.itemID, "err", err)...)
					break
				}
				if keep && it != nil && i < int64(len(augs)) {
					it.SetAugmentation(augs[i])
				}
			}
		}
		name := s.itemNameParam(pr.itemID)
		switch {
		case n > 1:
			p.SendSystemMessage(sysMsgEarnedS2S1, name, serverpackets.NumberParam(itemCount32(n)))
		case list.maintainEnchantment && pr.enchant > 0:
			p.SendSystemMessage(exchangeMsgAcquiredEnchanted, serverpackets.NumberParam(pr.enchant), name)
		default:
			p.SendSystemMessage(sysMsgEarnedS1, name)
		}
	}
}
