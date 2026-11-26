package game

import (
	"time"

	"github.com/IPeter/l2go/internal/residence"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The clan hall functions (Mobius ClanHall.ClanHallFunction and
// updateFunctions): the owner rents a function at the hall manager, paying
// the first period from the player's inventory; then the clan warehouse pays
// every period, and a function it cannot pay stops. HP and MP restore speed
// up the members' regeneration inside the hall (Formulas.calcHpRegen and
// calcMpRegen), exp restore gives back exp on a restart to the hall.

// hallFunction is a rented function with its payment task.
type hallFunction struct {
	residence.HallFunction
	// fromWarehouse is Mobius _cwh: the next payment comes from the clan
	// warehouse (false for the first run after a rent or a raise).
	fromWarehouse bool
	timer         world.TimerID // 0 for none
}

// feeTable returns the fees of a function from the clanHalls config.
func (s *Server) feeTable(typ int) residence.FeeTable {
	c := s.halls.cfg
	switch typ {
	case residence.HallFuncTeleport:
		return residence.FeeTable{Ratio: c.TeleportFeeRatio, Fees: c.TeleportFees}
	case residence.HallFuncItemCreate:
		return residence.FeeTable{Ratio: c.ItemCreationFeeRatio, Fees: c.ItemCreationFees}
	case residence.HallFuncRestoreHP:
		return residence.FeeTable{Ratio: c.HpRegenFeeRatio, Fees: c.HpRegenFees}
	case residence.HallFuncRestoreMP:
		return residence.FeeTable{Ratio: c.MpRegenFeeRatio, Fees: c.MpRegenFees}
	case residence.HallFuncRestoreExp:
		return residence.FeeTable{Ratio: c.ExpRegenFeeRatio, Fees: c.ExpRegenFees}
	case residence.HallFuncSupport:
		return residence.FeeTable{Ratio: c.SupportFeeRatio, Fees: c.SupportFees}
	case residence.HallFuncCurtains:
		return residence.FeeTable{Ratio: c.CurtainFeeRatio, Fees: c.CurtainFees}
	case residence.HallFuncFrontPlatform:
		return residence.FeeTable{Ratio: c.FrontPlatformFeeRatio, Fees: c.FrontPlatformFees}
	}
	return residence.FeeTable{}
}

// Function returns a rented function (Mobius getFunction), false for none.
func (h *ClanHall) Function(typ int) (residence.HallFunction, bool) {
	if f := h.functions[typ]; f != nil {
		return f.HallFunction, true
	}
	return residence.HallFunction{}, false
}

// updateHallFunction is ClanHall.updateFunctions for p: the fee comes out of
// p's inventory first (false when p has too little), then the function is
// rented, removed, raised or changed in place. The player and the hall are
// saved in one transaction. Deviation: a raised function's old payment task
// stops (Mobius keeps it running, charging twice).
func (s *Server) updateHallFunction(p *Player, h *ClanHall, typ, level, lease int) bool {
	if lease > 0 && (p.inv.Adena() < int64(lease) || !p.TakeItems(int(adenaID), int64(lease))) {
		return false
	}
	rate := s.feeTable(typ).Ratio
	old := h.functions[typ]
	var oldFn *residence.HallFunction
	if old != nil {
		oldFn = &old.HallFunction
	}
	switch residence.ChangeFunction(oldFn, level, lease) {
	case residence.FuncAdd:
		f := &hallFunction{HallFunction: residence.HallFunction{Type: typ, Level: level, Lease: lease, Rate: rate}}
		h.functions[typ] = f
		s.scheduleHallFunction(h, f, 0)
	case residence.FuncRemove:
		s.removeHallFunction(h, typ)
	case residence.FuncRaise:
		s.removeHallFunction(h, typ)
		f := &hallFunction{HallFunction: residence.HallFunction{Type: typ, Level: level, Lease: lease, Rate: rate, EndTime: -1}}
		h.functions[typ] = f
		s.scheduleHallFunction(h, f, 0)
	case residence.FuncKeep:
		old.Lease, old.Level = lease, level
	}
	s.SaveJoint([]*Player{p}, storage.Batch{ClanHalls: []storage.ClanHall{h.document()}})
	return true
}

// removeHallFunction is ClanHall.removeFunction (not saved).
func (s *Server) removeHallFunction(h *ClanHall, typ int) {
	if f := h.functions[typ]; f != nil && f.timer != 0 {
		s.loop.Cancel(f.timer)
	}
	delete(h.functions, typ)
}

// scheduleHallFunction (re)arms a function's payment task.
func (s *Server) scheduleHallFunction(h *ClanHall, f *hallFunction, d time.Duration) {
	if f.timer != 0 {
		s.loop.Cancel(f.timer)
	}
	f.timer = s.loop.After(d, func() {
		f.timer = 0
		s.hallFunctionTask(h, f)
	})
}

// hallFunctionTask is ClanHallFunction.FunctionTask.run: the warehouse pays
// the period (the first one after a rent was paid by the player), or the
// function stops. The hall and the warehouse are saved in one transaction.
func (s *Server) hallFunctionTask(h *ClanHall, f *hallFunction) {
	cl := h.Owner()
	if cl == nil || h.functions[f.Type] != f {
		return
	}
	wh := s.ClanWarehouse(cl)
	now := s.Now().UnixMilli()
	pay, end, remove := residence.FunctionPayment(f.HallFunction, f.fromWarehouse, now, wh.Adena())
	if remove {
		s.removeHallFunction(h, f.Type)
		h.save()
		return
	}
	f.EndTime = end
	var b storage.Batch
	if pay > 0 {
		if _, err := wh.DestroyByItemID(adenaID, int64(pay)); err != nil {
			s.log.Error("game: clan hall function fee not taken", "hall", h.ID, "function", f.Type, "err", err)
		}
		if ch := wh.Tracker().Take(); !noItemChanges(ch) {
			b.Items = []storage.OwnerItemChanges{ownerItemChanges(cl.ID, ch)}
		}
		wh.TakeUpdates()
	}
	b.ClanHalls = []storage.ClanHall{h.document()}
	s.SaveJoint(nil, b)
	f.fromWarehouse = true
	s.scheduleHallFunction(h, f, time.Duration(f.Rate)*time.Millisecond)
}

// hallRegen adds the clan hall restore functions to a player's regeneration
// (Formulas.calcHpRegen and calcMpRegen): in its clan's hall, HP and MP
// regenerate 1 + level/100 times as fast.
func (s *Server) hallRegen(p *Player, z *stats.RegenZone) {
	if !p.Zones.Inside(ZoneClanHall) {
		return
	}
	h := s.ClanHallByOwner(p.Clan())
	if h == nil || !h.Contains(p.Position()) {
		return
	}
	if f := h.functions[residence.HallFuncRestoreHP]; f != nil {
		z.HPMul = append(z.HPMul, 1+float64(f.Level)/100)
	}
	if f := h.functions[residence.HallFuncRestoreMP]; f != nil {
		z.MPMul = append(z.MPMul, 1+float64(f.Level)/100)
	}
}
