package game

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/manor"
	"github.com/IPeter/l2go/internal/residence"
	"github.com/IPeter/l2go/internal/social"
	"github.com/IPeter/l2go/internal/storage"
)

// The manor's windows and packets (Plan 14 M11k): the "manor_menu_select"
// bypass of the manor managers (Mobius ManorManager.onNpcManorBypass) and
// the chamberlains (CastleChamberlain.onNpcManorBypass), the info and
// setting windows, the seed shop (RequestBuySeed), the crop sale
// (RequestProcureCropList) and the lords' settings (RequestSetSeed,
// RequestSetCrop). The managers' dialog pages are Lua
// (scripts/npcs/manor_manager).

// manorManagers are the NPCs whose manor bypass the manor manager answers
// (Mobius ManorManager's ON_NPC_MANOR_BYPASS @Id list).
var manorManagers = map[int]bool{
	35644: true, 35645: true, 35319: true, 35366: true, 36456: true, 35512: true, 35558: true,
	35229: true, 35230: true, 35231: true, 35277: true, 35103: true, 35145: true, 35187: true,
}

// manorChamberlains are the castle chamberlains, whose manor bypass the
// lord's windows answer (CastleChamberlain's ON_NPC_MANOR_BYPASS @Id list).
var manorChamberlains = map[int]bool{
	35100: true, 35142: true, 35184: true, 35226: true, 35274: true, 35316: true, 35363: true, 35509: true, 35555: true,
}

// manorHarvesterList is the base of the harvester buylists: the manager's
// list is 300000 + its NPC ID (ManorManager's "Buy harvester").
const manorHarvesterList = 300000

// manorIDOf returns the manor_id parameter of an NPC's template, -1 without
// one (Mobius getParameters().getInt("manor_id", -1)).
func manorIDOf(n *Npc) int32 {
	if v, ok := n.Template.Parameters.Params["manor_id"]; ok {
		if id, err := strconv.ParseInt(strings.TrimSpace(v), 10, 32); err == nil {
			return int32(id)
		}
	}
	return -1
}

// parseManorMenu parses "manor_menu_select?ask=A&state=S&time=T" as Mobius
// RequestBypassToServer does: the three values by position; time 1 is the
// next period. ok is false for a command Mobius fails on.
func parseManorMenu(command string) (ask, state int, next, ok bool) {
	q := command
	if i := strings.IndexByte(command, '?'); i >= 0 {
		q = command[i+1:]
	}
	parts := strings.Split(q, "&")
	if len(parts) < 3 {
		return 0, 0, false, false
	}
	value := func(s string) (string, bool) {
		kv := strings.Split(s, "=")
		if len(kv) < 2 {
			return "", false
		}
		return kv[1], true
	}
	a, ok1 := value(parts[0])
	st, ok2 := value(parts[1])
	t, ok3 := value(parts[2])
	if !ok1 || !ok2 || !ok3 {
		return 0, 0, false, false
	}
	av, err1 := strconv.Atoi(a)
	sv, err2 := strconv.Atoi(st)
	if err1 != nil || err2 != nil {
		return 0, 0, false, false
	}
	return av, sv, t == "1", true
}

// manorMenuSelect handles the "manor_menu_select" bypass (Mobius
// RequestBypassToServer): with the manor allowed, it goes to the last NPC
// the player talked to, if the player can still talk to it and it is a
// manor manager or a chamberlain.
func (s *Server) manorMenuSelect(p *Player, command string) {
	n := s.Npc(p.LastNpc)
	if !s.manor.enabled || n == nil || n.Template == nil || !s.npcCanInteract(p, n) {
		return
	}
	ask, state, next, ok := parseManorMenu(command)
	if !ok {
		s.log.Debug("game: bad manor bypass", append(p.logArgs(), "bypass", command)...)
		return
	}
	switch {
	case manorManagers[n.Template.ID]:
		s.manorManagerMenu(p, n, ask, int32(state), next) //nolint:gosec // a client's int
	case manorChamberlains[n.Template.ID]:
		s.chamberlainManorMenu(p, n, ask, int32(state), next) //nolint:gosec // a client's int
	}
}

// manorManagerMenu is ManorManager.onNpcManorBypass.
func (s *Server) manorManagerMenu(p *Player, n *Npc, ask int, state int32, next bool) {
	if s.manor.mode == manor.Maintenance {
		p.SendSystemMessage(manorMsgMaintenance)
		return
	}
	templateID := manorIDOf(n)
	castleID := state
	if state == -1 {
		castleID = templateID
	}
	switch ask {
	case 1: // seed purchase
		if templateID != castleID {
			p.SendSystemMessage(manorMsgOnlySeedsOf, serverpackets.CastleNameParam(templateID))
			return
		}
		if pkt := s.buyListSeedPacket(p, castleID); pkt != nil {
			p.Send(pkt)
		}
	case 2: // crop sales
		if pkt := s.sellCropListPacket(p, castleID); pkt != nil {
			p.Send(pkt)
		}
	case 3: // seed info
		p.Send(s.seedInfoPacket(castleID, next, false))
	case 4: // crop info
		p.Send(s.cropInfoPacket(castleID, next, false))
	case 5: // basic info
		p.Send(s.manorDefaultInfoPacket(false))
	case 6: // buy a harvester
		s.showBuyWindow(p, n, manorHarvesterList+n.Template.ID)
	case 9: // the castles that buy a crop (the state is the crop)
		p.Send(s.procureCropDetailPacket(state))
	default:
		s.log.Warn("game: unknown manor request", append(p.logArgs(), "ask", ask)...)
	}
}

// chamberlainManorMenu is CastleChamberlain.onNpcManorBypass: for the owner
// clan's members and GMs, the lord's view of the lists and the settings
// windows, which open outside the approved period only.
func (s *Server) chamberlainManorMenu(p *Player, n *Npc, ask int, state int32, next bool) {
	c := s.npcCastle(n)
	if c == nil || (!p.IsGM() && (p.clan.clan == nil || p.ClanID() != c.ownerID)) {
		return
	}
	f := &s.manor
	if f.mode == manor.Maintenance {
		p.SendSystemMessage(manorMsgMaintenance)
		return
	}
	castleID := state
	if state == -1 {
		castleID = c.ID
	}
	switch ask {
	case 3:
		p.Send(s.seedInfoPacket(castleID, next, true))
	case 4:
		p.Send(s.cropInfoPacket(castleID, next, true))
	case 5:
		p.Send(s.manorDefaultInfoPacket(true))
	case 7, 8:
		if f.mode == manor.Approved {
			p.SendSystemMessage(manorMsgCannotSetUp)
			return
		}
		if ask == 7 {
			p.Send(s.seedSettingPacket(castleID))
		} else {
			p.Send(s.cropSettingPacket(castleID))
		}
	default:
		s.log.Warn("game: unknown chamberlain manor request", append(p.logArgs(), "ask", ask)...)
	}
}

// ----- the windows -----

// seedRewards returns a seed's level and rewards as the packets show them.
func seedRewards(seed *manor.Seed) serverpackets.SeedRewards {
	if seed == nil {
		return serverpackets.SeedRewards{}
	}
	return serverpackets.SeedRewards{Level: int32(seed.Level), Reward1: int32(seed.Reward1), Reward2: int32(seed.Reward2)} //nolint:gosec // small
}

// seedInfoPacket is ExShowSeedInfo: nothing for the next period until it is
// approved, nor for an unknown castle.
func (s *Server) seedInfoPacket(castleID int32, next, hide bool) []byte {
	f := &s.manor
	out := serverpackets.ExShowSeedInfo{HideButtons: hide, ManorID: castleID}
	if mc := s.manorCastle(castleID); mc != nil && (!next || f.mode == manor.Approved) {
		for _, sp := range mc.Products(next) {
			out.Seeds = append(out.Seeds, serverpackets.SeedInfo{SeedID: int32(sp.SeedID), Amount: itemCount32(sp.Amount), //nolint:gosec // item IDs
				StartAmount: itemCount32(sp.StartAmount), Price: itemCount32(sp.Price), SeedRewards: seedRewards(f.table.Seed(sp.SeedID))})
		}
	}
	return serverpackets.NewExShowSeedInfoPacket(out)
}

// cropInfoPacket is ExShowCropInfo, with seedInfoPacket's rules.
func (s *Server) cropInfoPacket(castleID int32, next, hide bool) []byte {
	f := &s.manor
	out := serverpackets.ExShowCropInfo{HideButtons: hide, ManorID: castleID}
	if mc := s.manorCastle(castleID); mc != nil && (!next || f.mode == manor.Approved) {
		for _, cp := range mc.Procures(next) {
			out.Crops = append(out.Crops, serverpackets.CropInfo{CropID: int32(cp.CropID), Amount: itemCount32(cp.Amount), //nolint:gosec // item IDs
				StartAmount: itemCount32(cp.StartAmount), Price: itemCount32(cp.Price), Reward: int8(cp.Reward), //nolint:gosec // 1-2
				SeedRewards: seedRewards(f.table.SeedByCrop(cp.CropID))})
		}
	}
	return serverpackets.NewExShowCropInfoPacket(out)
}

// manorDefaultInfoPacket is ExShowManorDefaultInfo: every crop.
func (s *Server) manorDefaultInfoPacket(hide bool) []byte {
	var crops []serverpackets.ManorDefaultCrop
	for _, seed := range s.manor.table.Crops() {
		//nolint:gosec // item IDs, levels and reference prices fit
		crops = append(crops, serverpackets.ManorDefaultCrop{CropID: int32(seed.CropID), Level: int32(seed.Level),
			SeedPrice: itemCount32(seed.SeedPrice), CropPrice: itemCount32(seed.CropPrice), Reward1: int32(seed.Reward1), Reward2: int32(seed.Reward2)})
	}
	return serverpackets.NewExShowManorDefaultInfoPacket(hide, crops)
}

// seedSettingPacket is ExShowSeedSetting: the castle's seeds with their
// limits and both periods.
func (s *Server) seedSettingPacket(castleID int32) []byte {
	f := &s.manor
	mc := s.manorCastle(castleID)
	var seeds []serverpackets.SeedSetting
	for _, seed := range f.table.SeedsFor(int(castleID)) {
		//nolint:gosec // item IDs fit
		st := serverpackets.SeedSetting{SeedID: int32(seed.SeedID), SeedRewards: seedRewards(seed), Limit: itemCount32(seed.SeedLimit),
			ReferencePrice: itemCount32(seed.SeedPrice), MinPrice: itemCount32(seed.SeedMinPrice()), MaxPrice: itemCount32(seed.SeedMaxPrice())}
		if mc != nil {
			if sp := mc.Product(seed.SeedID, false); sp != nil {
				st.Start, st.Price = itemCount32(sp.StartAmount), itemCount32(sp.Price)
			}
			if sp := mc.Product(seed.SeedID, true); sp != nil {
				st.NextStart, st.NextPrice = itemCount32(sp.StartAmount), itemCount32(sp.Price)
			}
		}
		seeds = append(seeds, st)
	}
	return serverpackets.NewExShowSeedSettingPacket(castleID, seeds)
}

// cropSettingPacket is ExShowCropSetting: the castle's crops with their
// limits and both periods.
func (s *Server) cropSettingPacket(castleID int32) []byte {
	f := &s.manor
	mc := s.manorCastle(castleID)
	var crops []serverpackets.CropSetting
	for _, seed := range f.table.SeedsFor(int(castleID)) {
		//nolint:gosec // item IDs fit
		st := serverpackets.CropSetting{CropID: int32(seed.CropID), SeedRewards: seedRewards(seed), Limit: itemCount32(seed.CropLimit),
			MinPrice: itemCount32(seed.CropMinPrice()), MaxPrice: itemCount32(seed.CropMaxPrice())}
		if mc != nil {
			if cp := mc.CropProcure(seed.CropID, false); cp != nil {
				st.Start, st.Price, st.Reward = itemCount32(cp.StartAmount), itemCount32(cp.Price), int8(cp.Reward) //nolint:gosec // 1-2
			}
			if cp := mc.CropProcure(seed.CropID, true); cp != nil {
				st.NextStart, st.NextPrice, st.NextReward = itemCount32(cp.StartAmount), itemCount32(cp.Price), int8(cp.Reward) //nolint:gosec // 1-2
			}
		}
		crops = append(crops, st)
	}
	return serverpackets.NewExShowCropSettingPacket(castleID, crops)
}

// sellCropListPacket is ExShowSellCropList: the crops p carries (the first
// stack of each, in Mobius's map order) with what the castle buys of them;
// nil for an unknown castle (Mobius fails).
func (s *Server) sellCropListPacket(p *Player, castleID int32) []byte {
	f := &s.manor
	mc := s.manorCastle(castleID)
	if mc == nil {
		return nil
	}
	var held []int
	items := make(map[int]*inventory.Item)
	for _, cropID := range f.table.CropIDs() {
		if it := p.inv.ByItemID(int32(cropID)); it != nil { //nolint:gosec // item IDs
			held = append(held, cropID)
			items[cropID] = it
		}
	}
	var crops []serverpackets.SellCrop
	for _, i := range manor.JavaHashOrder(held) {
		it := items[held[i]]
		c := serverpackets.SellCrop{ObjectID: it.ObjectID(), CropID: it.ItemID(), SeedRewards: seedRewards(f.table.SeedByCrop(held[i])),
			Count: itemCount32(it.Count())}
		if cp := mc.CropProcure(held[i], false); cp != nil && cp.Amount > 0 {
			c.Procured, c.ManorID, c.Amount, c.Price, c.Reward = true, castleID, itemCount32(cp.Amount), itemCount32(cp.Price), int8(cp.Reward) //nolint:gosec // 1-2
		}
		crops = append(crops, c)
	}
	return serverpackets.NewExShowSellCropListPacket(castleID, crops)
}

// procureCropDetailPacket is ExShowProcureCropDetail: the castles that still
// buy the crop, in castle order.
func (s *Server) procureCropDetailPacket(cropID int32) []byte {
	var castles []serverpackets.ProcureCastle
	for _, c := range s.Castles() {
		mc := s.manorCastle(c.ID)
		if mc == nil {
			continue
		}
		if cp := mc.CropProcure(int(cropID), false); cp != nil && cp.Amount > 0 {
			castles = append(castles, serverpackets.ProcureCastle{CastleID: c.ID, Amount: itemCount32(cp.Amount), Price: itemCount32(cp.Price), Reward: int8(cp.Reward)}) //nolint:gosec // 1-2
		}
	}
	return serverpackets.NewExShowProcureCropDetailPacket(cropID, castles)
}

// buyListSeedPacket is BuyListSeed: the seeds the castle sells that are
// left and priced; nil for an unknown castle.
func (s *Server) buyListSeedPacket(p *Player, castleID int32) []byte {
	mc := s.manorCastle(castleID)
	if mc == nil {
		return nil
	}
	var seeds []serverpackets.BuyListSeedEntry
	for _, sp := range mc.Production {
		if sp.Amount > 0 && sp.Price > 0 {
			seeds = append(seeds, serverpackets.BuyListSeedEntry{SeedID: int32(sp.SeedID), Amount: itemCount32(sp.Amount), Price: itemCount32(sp.Price)}) //nolint:gosec // item IDs
		}
	}
	return serverpackets.NewBuyListSeedPacket(itemCount32(p.inv.Adena()), castleID, seeds)
}

// manorMerchant returns the merchant p talked to last when p can still talk
// to it, else nil (getLastFolkNPC instanceof Merchant and canInteract).
func (s *Server) manorMerchant(p *Player) *Npc {
	n := s.Npc(p.LastNpc)
	if n == nil || n.Template == nil || !s.npcIs(n, "Merchant") || !s.npcCanInteract(p, n) {
		return nil
	}
	return n
}

// ----- the seed shop -----

// handleRequestBuySeed handles RequestBuySeed (0xC4) in Mobius's order: the
// flood check, a refused list, the maintenance, the castle, the manor
// manager of that castle, each seed (on sale, priced, enough left, within
// MaxAdena), the weight, slots and adena; then each seed is paid, taken from
// the castle's stock (paid back when it ran out) and given, and the castle's
// treasury gets the money ("$s1 adena disappeared."). The player, the
// treasury and the stock are saved together.
func (s *Server) handleRequestBuySeed(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestBuySeed(body)
	if err != nil {
		s.log.Debug("game: bad RequestBuySeed", append(p.logArgs(), "err", err)...)
		return
	}
	if s.flood(&p.items.nextTransaction, time.Duration(s.gs.Economy.FloodTransaction)) {
		p.SendMessage("You are buying seeds too fast!")
		return
	}
	if req.Items == nil || s.manor.mode == manor.Maintenance {
		p.SendActionFailed()
		return
	}
	c, mc := s.CastleByID(req.ManorID), s.manorCastle(req.ManorID)
	if c == nil || mc == nil {
		p.SendActionFailed()
		return
	}
	if n := s.manorMerchant(p); n == nil || manorIDOf(n) != req.ManorID {
		p.SendActionFailed()
		return
	}
	maxAdena := s.items.cfg.MaxAdena
	var total, weight, slots int64
	for _, it := range req.Items {
		count := int64(it.Count)
		sp := mc.Product(int(it.SeedID), false)
		if sp == nil || sp.Price <= 0 || sp.Amount < count || maxAdena/count < sp.Price {
			p.SendActionFailed()
			return
		}
		total += sp.Price * count
		if total > maxAdena {
			s.illegalAction(p, fmt.Sprintf("tried to purchase over %d adena worth of goods", maxAdena))
			p.SendActionFailed()
			return
		}
		t, ok := s.data.ItemTemplate(int(it.SeedID))
		if !ok {
			p.SendActionFailed()
			return
		}
		weight += count * int64(t.Weight)
		switch {
		case !t.Stackable:
			slots += count
		case p.inv.ByItemID(it.SeedID) == nil:
			slots++
		}
	}
	switch {
	case !p.inv.ValidateWeight(weight):
		p.SendSystemMessage(shopMsgWeightLimit)
		return
	case !p.inv.ValidateCapacity(slots):
		p.SendSystemMessage(sysMsgInventoryFull)
		return
	case total < 0 || p.inv.Adena() < total:
		p.SendSystemMessage(sysMsgNotEnoughAdena)
		return
	}
	for _, it := range req.Items {
		count := int64(it.Count)
		price := mc.Product(int(it.SeedID), false).Price * count
		if !s.ReduceAdena(p, price, false) {
			total -= price
			continue
		}
		if !mc.DecreaseSeed(int(it.SeedID), count) {
			s.AddAdena(p, price, false)
			total -= price
			continue
		}
		if _, err := s.AddItem(p, it.SeedID, count, -1, true, false); err != nil {
			s.log.Error("game: cannot give bought seeds", append(p.logArgs(), "item", it.SeedID, "count", count, "err", err)...)
		}
	}
	if total <= 0 {
		return
	}
	if v, ok := residence.AddNoTax(c.ownerID > 0, c.treasury, total); ok {
		c.treasury = v
	}
	p.SendSystemMessage(sysMsgAdenaDisappeared, serverpackets.NumberParam(itemCount32(total)))
	s.SaveJoint([]*Player{p}, storage.Batch{Castles: []storage.Castle{c.document()},
		ManorCastles: []storage.ManorCastle{manorCastleDoc(c.ID, mc)}})
}

// ----- the crop sale -----

// cropSale is a checked line of RequestProcureCropList.
type cropSale struct {
	item     clientpackets.ProcureCropItem
	castle   *manor.Castle
	price    int64 // the crop's price at that manor
	rewardID int
}

// handleRequestProcureCropList handles RequestProcureCropList (D0 09) in
// Mobius's order: a refused list, the maintenance, the manor manager of
// its castle, each crop (carried, bought by its manor with enough left),
// the weight and slots of the rewards; then each crop is sold: nothing for a
// price under one reward ("Failed in trading ..."), a 5 % fee for selling to
// another castle's manor, the manor's stock lowered, the crop destroyed and
// the reward given. The player and the manors' stocks are saved together.
//
// Mobius quirk, kept: the stock is lowered before the fee and the crop are
// taken, so a sale that fails there still lowers it.
func (s *Server) handleRequestProcureCropList(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestProcureCropList(body)
	if err != nil {
		s.log.Debug("game: bad RequestProcureCropList", append(p.logArgs(), "err", err)...)
		return
	}
	if req.Items == nil {
		return
	}
	f := &s.manor
	if f.mode == manor.Maintenance {
		p.SendActionFailed()
		return
	}
	n := s.manorMerchant(p)
	var castle *Castle
	if n != nil {
		castle = s.npcCastle(n)
	}
	if castle == nil || manorIDOf(n) != castle.ID {
		p.SendActionFailed()
		return
	}
	sales := make([]cropSale, 0, len(req.Items))
	var weight, slots int64
	for _, it := range req.Items {
		item := p.inv.ByObjectID(it.ObjectID)
		if item == nil || item.Count() < int64(it.Count) || item.ItemID() != it.ItemID {
			p.SendActionFailed()
			return
		}
		mc := s.manorCastle(it.ManorID)
		var cp *manor.Procure
		if mc != nil {
			cp = mc.CropProcure(int(it.ItemID), false)
		}
		seed := f.table.SeedByCrop(int(it.ItemID))
		if cp == nil || cp.Amount < int64(it.Count) || seed == nil {
			p.SendActionFailed()
			return
		}
		rewardID := seed.Reward(cp.Reward)
		t, ok := s.data.ItemTemplate(rewardID)
		if !ok {
			p.SendActionFailed()
			return
		}
		weight += int64(it.Count) * int64(t.Weight)
		switch {
		case !t.Stackable:
			slots += int64(it.Count)
		case p.inv.ByItemID(int32(rewardID)) == nil: //nolint:gosec // item IDs
			slots++
		}
		sales = append(sales, cropSale{item: it, castle: mc, price: cp.Price, rewardID: rewardID})
	}
	if !p.inv.ValidateWeight(weight) {
		p.SendSystemMessage(shopMsgWeightLimit)
		return
	}
	if !p.inv.ValidateCapacity(slots) {
		p.SendSystemMessage(sysMsgInventoryFull)
		return
	}
	touched := make(map[int32]bool)
	for _, sale := range sales {
		it := sale.item
		t, _ := s.data.ItemTemplate(sale.rewardID)
		rewardPrice := int64(t.Price)
		if rewardPrice == 0 {
			continue
		}
		price := int64(it.Count) * sale.price
		rewards := price / rewardPrice
		failed := func() {
			p.SendSystemMessage(manorMsgFailedTrading, s.itemNameParam(it.ItemID), serverpackets.NumberParam(it.Count))
		}
		if rewards < 1 {
			failed()
			continue
		}
		var fee int64
		if castle.ID != it.ManorID {
			fee = int64(float64(price) * 0.05)
		}
		if fee != 0 && p.inv.Adena() < fee {
			failed()
			p.SendSystemMessage(sysMsgNotEnoughAdena)
			continue
		}
		if !sale.castle.DecreaseCrop(int(it.ItemID), int64(it.Count)) {
			continue
		}
		touched[it.ManorID] = true
		if fee > 0 && !s.ReduceAdena(p, fee, true) {
			continue
		}
		item := p.inv.ByObjectID(it.ObjectID)
		if item == nil || !s.DestroyItem(p, item, int64(it.Count), true) {
			continue
		}
		if _, err := s.AddItem(p, int32(sale.rewardID), rewards, -1, true, false); err != nil { //nolint:gosec // item IDs
			s.log.Error("game: cannot give a crop reward", append(p.logArgs(), "item", sale.rewardID, "count", rewards, "err", err)...)
		}
	}
	var b storage.Batch
	for _, c := range s.Castles() {
		if touched[c.ID] {
			b.ManorCastles = append(b.ManorCastles, manorCastleDoc(c.ID, s.manorCastle(c.ID)))
		}
	}
	s.SaveJoint([]*Player{p}, b)
}

// ----- the lords' settings -----

// manorLord reports whether p may set castleID's manor now (RequestSetSeed
// and RequestSetCrop): the modifiable period, a clan that owns the castle,
// the CASTLE_MANOR privilege and an NPC p can still talk to. Otherwise p
// gets ActionFailed.
func (s *Server) manorLord(p *Player, castleID int32) bool {
	clan := p.clan.clan
	n := s.Npc(p.LastNpc)
	if s.manor.mode != manor.Modifiable || clan == nil || clan.CastleID != castleID || !p.HasClanAccess(social.AccessCastleManor) ||
		n == nil || !s.npcCanInteract(p, n) || s.manorCastle(castleID) == nil {
		p.SendActionFailed()
		return false
	}
	return true
}

// handleRequestSetSeed handles RequestSetSeed (D0 0A): the lord's seeds of
// the next period, the ones within the limits and price bounds, replace the
// list and are saved.
func (s *Server) handleRequestSetSeed(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestSetSeed(body)
	if err != nil {
		s.log.Debug("game: bad RequestSetSeed", append(p.logArgs(), "err", err)...)
		return
	}
	if len(req.Items) == 0 || !s.manorLord(p, req.ManorID) {
		return
	}
	list := make([]manor.Production, 0, len(req.Items))
	for _, it := range req.Items {
		list = append(list, manor.Production{SeedID: int(it.SeedID), Amount: int64(it.Sales), Price: int64(it.Price), StartAmount: int64(it.Sales)})
	}
	s.manorCastle(req.ManorID).SetNextProduction(s.manor.table.FilterProduction(list))
	s.saveManorCastle(req.ManorID)
}

// handleRequestSetCrop handles RequestSetCrop (D0 0B): the lord's crops of
// the next period, with RequestSetSeed's rules.
func (s *Server) handleRequestSetCrop(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestSetCrop(body)
	if err != nil {
		s.log.Debug("game: bad RequestSetCrop", append(p.logArgs(), "err", err)...)
		return
	}
	if len(req.Items) == 0 || !s.manorLord(p, req.ManorID) {
		return
	}
	list := make([]manor.Procure, 0, len(req.Items))
	for _, it := range req.Items {
		list = append(list, manor.Procure{CropID: int(it.CropID), Amount: int64(it.Sales), Price: int64(it.Price), StartAmount: int64(it.Sales), Reward: int(it.Reward)})
	}
	s.manorCastle(req.ManorID).SetNextProcure(s.manor.table.FilterProcure(list, int(req.ManorID)))
	s.saveManorCastle(req.ManorID)
}
