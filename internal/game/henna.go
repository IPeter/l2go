package game

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/henna"
	"github.com/IPeter/l2go/internal/storage"
)

// Hennas (Plan 14, M11c): the dyes symbol makers draw and remove. The rules
// are in internal/henna; this file adapts them to the packets (Mobius
// RequestHenna* and the Henna* server packets), the player (Mobius
// Player.addHenna, removeHenna, restoreHenna and the class change) and
// storage (Character.Hennas, by class index, decision X3). The symbol
// maker's dialog is scripts/npcs/symbol_maker (Lua), which opens the windows
// with player:show_henna_equip_list() and show_henna_remove_list().
//
// Decision X10: drawing and removing a dye need the last NPC the player
// talked to to be a symbol maker within interaction range (250); Mobius
// checks neither. GMs need none, as for selling (I9).

func init() { registerFeature("henna", setupHenna) }

// System messages of the hennas.
const (
	sysMsgSymbolAdded      serverpackets.SystemMessageID = 877 // "The symbol has been added."
	sysMsgSymbolDeleted    serverpackets.SystemMessageID = 878 // "The symbol has been deleted."
	sysMsgSymbolCannotDraw serverpackets.SystemMessageID = 899 // "The symbol cannot be drawn."
	sysMsgNoSymbolSlot     serverpackets.SystemMessageID = 900 // "No slot exists to draw the symbol."
)

// hennaFeature is the feature's state on the Server (Server.henna).
type hennaFeature struct {
	// table is the dye data, nil when the data folder has none (no dye
	// exists then). Replaced by //reload henna; read on the loop.
	table *data.Hennas
}

// playerHenna is the feature's state of a player (Player.henna). The dyes
// of every class index are in Player.char.Hennas, which is replaced on each
// change and never changed in place (a snapshot shares it with the saver).
// Loop-only.
type playerHenna struct {
	// classIndex is the class whose dyes are worn: 0, the base class, until
	// subclasses switch it (SwapHennas).
	classIndex int32
	// worn are the dyes of classIndex.
	worn henna.Worn
}

// setupHenna loads the dyes and registers the packets.
func setupHenna(s *Server) error {
	if err := s.loadHennaTable(); err != nil {
		return err
	}
	return errors.Join(
		s.Register(clientpackets.OpRequestHennaItemList, s.handleHennaItemList),
		s.Register(clientpackets.OpRequestHennaItemInfo, s.handleHennaItemInfo),
		s.Register(clientpackets.OpRequestHennaEquip, s.handleHennaEquip),
		s.Register(clientpackets.OpRequestHennaRemoveList, s.handleHennaRemoveList),
		s.Register(clientpackets.OpRequestHennaItemRemoveInfo, s.handleHennaItemRemoveInfo),
		s.Register(clientpackets.OpRequestHennaRemove, s.handleHennaRemove),
	)
}

// loadHennaTable reads the dye data of the data folder. Without the file no
// dye exists (logged).
func (s *Server) loadHennaTable() error {
	dir := s.settings.DataDir
	if dir == "" {
		return nil
	}
	switch t, err := data.LoadHennas(dir); {
	case err == nil:
		s.henna.table = t
	case errors.Is(err, fs.ErrNotExist):
		s.log.Info("game: no henna data, symbol makers have no dyes", "file", filepath.Join(dir, data.HennasFile))
	default:
		return fmt.Errorf("hennas: %w", err)
	}
	return nil
}

// reloadHennas is //reload henna (Mobius AdminReload "henna"): the dye data
// is read again. Worn dyes keep their old data until the player's dyes are
// loaded again, as in Mobius.
func (s *Server) reloadHennas(p *Player) {
	if err := s.loadHennaTable(); err != nil {
		s.log.Error("game: //reload henna failed", append(p.logArgs(), "err", err)...)
		p.SendMessage("Reloading the henna data failed: " + err.Error())
		return
	}
	s.tellGMs(p.Name() + ": Reloaded Henna data.")
}

// loadHennas gives a selected player the dyes of its base class (Mobius
// restoreHenna in Player.load), before the skills set the HP within the
// maxima. It runs on the loop at character selection; it sends nothing.
func (s *Server) loadHennas(p *Player) {
	s.restoreHennas(p, 0)
}

// SwapHennas puts on p the dyes of another class index and sends HennaInfo
// (Mobius setActiveClass step 6: restoreHenna, then HennaInfo). The
// subclass feature calls it when p switches its active class, after the new
// class template is set, and at character selection when p logged out on a
// subclass. The caller sends UserInfo. Loop-only.
func (s *Server) SwapHennas(p *Player, classIndex int32) {
	s.restoreHennas(p, classIndex)
	if p.InGame() {
		p.Send(hennaInfoPacket(p))
	}
}

// restoreHennas sets p's worn dyes to the stored ones of the class index and
// their stat bonus.
func (s *Server) restoreHennas(p *Player, classIndex int32) {
	p.henna.classIndex = classIndex
	p.henna.worn = henna.Restore(s.henna.table, p.char.Hennas[classIndex])
	p.stats.SetHenna(p.henna.worn.Bonus())
}

// hennasChanged applies a change of p's worn dyes (Mobius addHenna and
// removeHenna): the stat bonus (recalcHennaStats), the stored dyes of the
// class, HennaInfo, UserInfo and a save.
func (s *Server) hennasChanged(p *Player) {
	p.stats.SetHenna(p.henna.worn.Bonus())
	stored := make(map[int32]storage.HennaSlots, len(p.char.Hennas)+1)
	for k, v := range p.char.Hennas {
		stored[k] = v
	}
	stored[p.henna.classIndex] = p.henna.worn.IDs()
	p.char.Hennas = storage.CloneHennas(stored)
	p.Send(hennaInfoPacket(p))
	p.UpdateUserInfo()
	s.RequestSave(p)
}

// hennaInfoPacket is HennaInfo with p's dyes and their stat bonus (Mobius
// HennaInfo).
func hennaInfoPacket(p *Player) []byte {
	b := p.henna.worn.Bonus()
	// Mobius writes them with writeByte; the sums stay far within a byte.
	info := serverpackets.HennaInfo{
		INT: int8(b.INT), //nolint:gosec // see above
		STR: int8(b.STR), //nolint:gosec // see above
		CON: int8(b.CON), //nolint:gosec // see above
		MEN: int8(b.MEN), //nolint:gosec // see above
		DEX: int8(b.DEX), //nolint:gosec // see above
		WIT: int8(b.WIT), //nolint:gosec // see above
	}
	for _, h := range p.henna.worn.List() {
		info.DyeIDs = append(info.DyeIDs, int32(h.DyeID)) //nolint:gosec // dye IDs are small
	}
	return serverpackets.NewHennaInfoPacket(info)
}

// hennaClassLevel returns the level of p's class, which sets its dye slots:
// 0 for a base class, 1 to 3 after each class change.
func hennaClassLevel(p *Player) int { return data.PlayerClass(p.ClassID()).Level() }

// hennaPlayer is what the henna rules need of p, with the dye item's count
// of h (none for a nil h).
func hennaPlayer(p *Player, h *data.Henna) henna.Player {
	hp := henna.Player{ClassID: p.ClassID(), ClassLevel: hennaClassLevel(p), Adena: p.inv.Adena()}
	if h != nil && p.inv.Inventory != nil {
		hp.DyeCount = p.inv.Count(int32(h.DyeItemID), -1, true) //nolint:gosec // item IDs fit in int32
	}
	return hp
}

// ShowHennaEquipList opens the "draw a symbol" window (Mobius
// player.sendPacket(new HennaEquipList(player)), the symbol maker's "Draw"
// link; scripting.PlayerHenna).
func (p *Player) ShowHennaEquipList() { p.Send(p.srv.hennaEquipListPacket(p)) }

// ShowHennaRemoveList opens the "delete a symbol" window (Mobius
// HennaRemoveList, the symbol maker's "Remove" link; scripting.PlayerHenna).
func (p *Player) ShowHennaRemoveList() { p.Send(hennaRemoveListPacket(p)) }

// hennaEquipListPacket is HennaEquipList: the dyes p's class may wear whose
// dye item p carries, after the count of all the class's dyes (Mobius).
func (s *Server) hennaEquipListPacket(p *Player) []byte {
	list := s.henna.table.ForClass(p.ClassID())
	l := serverpackets.HennaEquipList{Adena: itemCount32(p.inv.Adena()), Listed: int32(len(list))} //nolint:gosec // 180 dyes at most
	for _, h := range list {
		if p.inv.Inventory == nil || p.inv.ByItemID(int32(h.DyeItemID)) == nil { //nolint:gosec // item IDs fit in int32
			continue
		}
		l.Entries = append(l.Entries, hennaEntry(h, h.WearCount, h.WearFee))
	}
	return serverpackets.NewHennaEquipListPacket(l)
}

// hennaRemoveListPacket is HennaRemoveList: p's dyes with what removing them
// gives back and costs.
func hennaRemoveListPacket(p *Player) []byte {
	l := serverpackets.HennaRemoveList{
		Adena:      itemCount32(p.inv.Adena()),
		EmptySlots: int32(p.henna.worn.EmptySlots(hennaClassLevel(p))), //nolint:gosec // 0-3
	}
	for _, h := range p.henna.worn.List() {
		l.Entries = append(l.Entries, hennaEntry(h, h.CancelCount, h.CancelFee))
	}
	return serverpackets.NewHennaRemoveListPacket(l)
}

// hennaEntry is a line of the dye windows; the data's values fit in int32.
func hennaEntry(h *data.Henna, count, fee int) serverpackets.HennaEntry {
	return serverpackets.HennaEntry{
		DyeID:     int32(h.DyeID),     //nolint:gosec // see above
		DyeItemID: int32(h.DyeItemID), //nolint:gosec // see above
		Count:     int32(count),       //nolint:gosec // see above
		Fee:       int32(fee),         //nolint:gosec // see above
	}
}

// hennaItemInfo is the body of HennaItemDrawInfo (sign 1) and
// HennaItemRemoveInfo (sign -1): the dye, its count and fee, whether p's
// class may wear it, p's adena, and p's base stats now and with the dye's
// stats added (drawing) or taken away (removing), uncapped, as Mobius
// shows them.
func hennaItemInfo(p *Player, h *data.Henna, count, fee, sign int) serverpackets.HennaItemInfo {
	v := p.stats.View()
	stat := func(now int32, d int) serverpackets.HennaStat {
		return serverpackets.HennaStat{Now: now, After: now + int32(sign*d)} //nolint:gosec // small
	}
	e := hennaEntry(h, count, fee)
	return serverpackets.HennaItemInfo{
		DyeID:     e.DyeID,
		DyeItemID: e.DyeItemID,
		Count:     e.Count,
		Fee:       e.Fee,
		Allowed:   h.AllowedClass(p.ClassID()),
		Adena:     itemCount32(p.inv.Adena()),
		INT:       stat(v.INT, h.INT),
		STR:       stat(v.STR, h.STR),
		CON:       stat(v.CON, h.CON),
		MEN:       stat(v.MEN, h.MEN),
		DEX:       stat(v.DEX, h.DEX),
		WIT:       stat(v.WIT, h.WIT),
	}
}

// handleHennaItemList handles RequestHennaItemList (0xBA): the "draw"
// window again.
func (s *Server) handleHennaItemList(p *Player, body []byte) {
	if _, err := clientpackets.ParseRequestHennaItemList(body); err != nil {
		s.log.Debug("game: bad RequestHennaItemList", append(p.logArgs(), "err", err)...)
		return
	}
	p.ShowHennaEquipList()
}

// handleHennaItemInfo handles RequestHennaItemInfo (0xBB): what drawing a
// dye costs and does. An unknown dye gets ActionFailed (logged unless 0).
func (s *Server) handleHennaItemInfo(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestHennaItemInfo(body)
	if err != nil {
		s.log.Debug("game: bad RequestHennaItemInfo", append(p.logArgs(), "err", err)...)
		return
	}
	h := s.henna.table.Henna(int(req.SymbolID))
	if h == nil {
		if req.SymbolID != 0 {
			s.log.Warn("game: RequestHennaItemInfo of an unknown dye", append(p.logArgs(), "dye", req.SymbolID)...)
		}
		p.SendActionFailed()
		return
	}
	p.Send(serverpackets.NewHennaItemDrawInfoPacket(hennaItemInfo(p, h, h.WearCount, h.WearFee, 1)))
}

// handleHennaRemoveList handles RequestHennaRemoveList (0xBD): the "delete"
// window again.
func (s *Server) handleHennaRemoveList(p *Player, body []byte) {
	if _, err := clientpackets.ParseRequestHennaRemoveList(body); err != nil {
		s.log.Debug("game: bad RequestHennaRemoveList", append(p.logArgs(), "err", err)...)
		return
	}
	p.ShowHennaRemoveList()
}

// handleHennaItemRemoveInfo handles RequestHennaItemRemoveInfo (0xBE): what
// removing a dye gives back, costs and does. Dye 0 is ignored; an unknown
// dye gets ActionFailed (logged).
func (s *Server) handleHennaItemRemoveInfo(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestHennaItemRemoveInfo(body)
	if err != nil {
		s.log.Debug("game: bad RequestHennaItemRemoveInfo", append(p.logArgs(), "err", err)...)
		return
	}
	if req.SymbolID == 0 {
		return
	}
	h := s.henna.table.Henna(int(req.SymbolID))
	if h == nil {
		s.log.Warn("game: RequestHennaItemRemoveInfo of an unknown dye", append(p.logArgs(), "dye", req.SymbolID)...)
		p.SendActionFailed()
		return
	}
	p.Send(serverpackets.NewHennaItemRemoveInfoPacket(hennaItemInfo(p, h, h.CancelCount, h.CancelFee, -1)))
}

// symbolMakerNear reports whether p may draw or remove dyes (decision X10):
// a GM always, a player when the NPC it talked to last is a symbol maker in
// its instance within interaction range (3D).
func (s *Server) symbolMakerNear(p *Player) bool {
	if p.IsGM() {
		return true
	}
	n := s.Npc(p.LastNpc)
	if n == nil || !n.InWorld() || !henna.IsSymbolMaker(n.Template.ID) {
		return false
	}
	return p.InstanceID() == n.InstanceID() && p.Position().InRange3D(n.Position(), s.npcInteractionRange())
}

// hennaFlood is Mobius's transaction flood protector of the henna packets.
func (s *Server) hennaFlood(p *Player) bool {
	return s.flood(&p.items.nextTransaction, time.Duration(s.gs.Economy.FloodTransaction))
}

// handleHennaEquip handles RequestHennaEquip (0xBC, Mobius): with a free
// slot, a known dye the class may wear, enough dyes and the fee, the dye is
// drawn (HennaInfo, UserInfo), the dyes and the fee are taken and "The
// symbol has been added." follows. Otherwise "No slot exists to draw the
// symbol." or "The symbol cannot be drawn." and ActionFailed; asking for a
// dye the class may not wear is punished as an exploit (players only).
func (s *Server) handleHennaEquip(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestHennaEquip(body)
	if err != nil {
		s.log.Debug("game: bad RequestHennaEquip", append(p.logArgs(), "err", err)...)
		return
	}
	if s.hennaFlood(p) {
		return
	}
	if !s.symbolMakerNear(p) {
		p.SendActionFailed()
		return
	}
	h := s.henna.table.Henna(int(req.SymbolID))
	res, forbidden := henna.CheckEquip(p.henna.worn, h, hennaPlayer(p, h))
	switch res {
	case henna.EquipNoSlot:
		p.SendSystemMessage(sysMsgNoSymbolSlot)
		p.SendActionFailed()
		return
	case henna.EquipUnknown:
		s.log.Warn("game: RequestHennaEquip of an unknown dye", append(p.logArgs(), "dye", req.SymbolID)...)
		p.SendActionFailed()
		return
	case henna.EquipRefused:
		p.SendSystemMessage(sysMsgSymbolCannotDraw)
		if forbidden && !p.IsGM() {
			s.illegalAction(p, fmt.Sprintf("tried to add the forbidden henna %d", h.DyeID))
		}
		p.SendActionFailed()
		return
	}
	p.henna.worn.Add(h)
	s.hennasChanged(p)
	s.DestroyItemByItemID(p, int32(h.DyeItemID), int64(h.WearCount), true) //nolint:gosec // item IDs fit in int32
	s.ReduceAdena(p, int64(h.WearFee), false)
	p.SendSystemMessage(sysMsgSymbolAdded)
}

// handleHennaRemove handles RequestHennaRemove (0xBF, Mobius): the first
// slot with the dye is emptied when p has the cancel fee (removeHenna),
// else "You do not have enough adena." and ActionFailed; a dye p does not
// wear gets ActionFailed (logged).
func (s *Server) handleHennaRemove(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestHennaRemove(body)
	if err != nil {
		s.log.Debug("game: bad RequestHennaRemove", append(p.logArgs(), "err", err)...)
		return
	}
	if s.hennaFlood(p) || !s.symbolMakerNear(p) {
		p.SendActionFailed()
		return
	}
	slot, res := henna.CheckRemove(p.henna.worn, int(req.SymbolID), p.inv.Adena())
	switch res {
	case henna.RemoveNotWorn:
		s.log.Warn("game: RequestHennaRemove of a dye not worn", append(p.logArgs(), "dye", req.SymbolID)...)
		p.SendActionFailed()
	case henna.RemoveNoAdena:
		p.SendSystemMessage(sysMsgNotEnoughAdena)
		p.SendActionFailed()
	default:
		s.removeHenna(p, slot)
	}
}

// removeHenna empties a slot of p (Mobius Player.removeHenna): HennaInfo and
// UserInfo, the dyes given back (the cancel count) and the cancel fee taken
// when p has it, "You have earned $s2 $s1(s)." and "The symbol has been
// deleted.". It reports whether the slot held a dye.
func (s *Server) removeHenna(p *Player, slot int) bool {
	h := p.henna.worn.Remove(slot)
	if h == nil {
		return false
	}
	s.hennasChanged(p)
	itemID := int32(h.DyeItemID) //nolint:gosec // item IDs fit in int32
	if h.CancelCount > 0 && p.inv.Inventory != nil {
		// Mobius adds to the inventory directly: no "obtained" message.
		if _, err := p.inv.Create(itemID, int64(h.CancelCount)); err != nil {
			s.log.Error("game: cannot give back the dyes of a removed henna", append(p.logArgs(), "item", itemID, "err", err)...)
		} else {
			s.itemsChanged(p, true, nil)
		}
	}
	s.ReduceAdena(p, int64(h.CancelFee), false)
	p.SendSystemMessage(sysMsgEarnedS2S1, s.itemNameParam(itemID), serverpackets.NumberParam(int32(h.CancelCount))) //nolint:gosec // small
	p.SendSystemMessage(sysMsgSymbolDeleted)
	return true
}

// hennaClassChanged removes the dyes p's new class may not wear (Mobius
// setPlayerClass: removeHenna for each), with the usual refund and fee. It
// runs from ChangeClass after the class is set.
func (s *Server) hennaClassChanged(p *Player) {
	for _, slot := range p.henna.worn.Disallowed(p.ClassID()) {
		s.removeHenna(p, slot)
	}
}
