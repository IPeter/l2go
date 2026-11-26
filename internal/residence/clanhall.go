package residence

import "time"

// Clan hall rules (Plan 14 M11j): the functions an owner rents and their
// fees (Mobius ClanHall and its ClanHallFunction), and an auctionable hall's
// weekly lease (AuctionableHall.FeeTask).

// Clan hall functions (Mobius ClanHall.FUNC_*).
const (
	HallFuncTeleport      = 1
	HallFuncItemCreate    = 2
	HallFuncRestoreHP     = 3
	HallFuncRestoreMP     = 4
	HallFuncRestoreExp    = 5
	HallFuncSupport       = 6
	HallFuncFrontPlatform = 7 // decoration, auctionable halls only
	HallFuncCurtains      = 8 // decoration, auctionable halls only
)

// System messages of the clan halls (Mobius SystemMessageId).
const (
	MsgHallLeaseUnpaid    = 1051 // Payment for your clan hall has not been made. Please make payment to your clan warehouse by $s1 tomorrow.
	MsgHallLeaseRevoked   = 1052 // The clan hall fee is one week overdue; therefore the clan hall ownership has been revoked.
	MsgBidTooLow          = 677  // Your bid price must be higher than the minimum price that can be bid.
	MsgNotEnoughHallAdena = 1005 // There is not enough adena in the clan hall warehouse.
	MsgBidPlaced          = 1006 // You have bid in a clan hall auction.
	MsgAuctionNeedsLevel2 = 673  // Only a clan leader whose clan is of level 2 or higher is allowed to participate in a clan hall auction.
	MsgAlreadyBid         = 676  // Since you have already submitted a bid, you are not allowed to participate in another auction at this time.
	MsgNoOfferings        = 1883 // There are no offerings I own or I made a bid for.
	MsgCannotParticipate  = 680  // You cannot participate in an auction.
	MsgBidCanceled        = 679  // You have canceled your bid.
)

// LeaseRate is AuctionableHall.CH_RATE: the lease is taken every week.
const LeaseRate = 604_800_000 * time.Millisecond

// leaseRecheck is how long after a missed lease the payment is tried again.
// Mobius schedules now + 24 h as the delay (an absolute time); we wait 24 h
// (Plan 14, Mobius quirks: changed).
const leaseRecheck = 24 * time.Hour

// HallFunction is a function an owner rents (Mobius ClanHallFunction, a
// clanhall_functions row): its level, the fee taken every Rate
// milliseconds from the clan warehouse, and when it is due next (Unix
// milliseconds; 0 for a new function, -1 after a raise).
type HallFunction struct {
	Type    int
	Level   int
	Lease   int
	Rate    int64
	EndTime int64
}

// feeLevels are the levels of Mobius's fee switches by function: level
// levels[i] costs fee i+1; any other level costs the last fee (the switch's
// default).
var feeLevels = map[int][]int{
	HallFuncRestoreHP:     {20, 40, 80, 100, 120, 140, 160, 180, 200, 220, 240, 260},
	HallFuncRestoreMP:     {5, 10, 15, 30},
	HallFuncRestoreExp:    {5, 10, 15, 25, 35, 40},
	HallFuncItemCreate:    {1, 2},
	HallFuncTeleport:      {1},
	HallFuncSupport:       {1, 2, 3, 4, 5, 6, 7},
	HallFuncCurtains:      {1},
	HallFuncFrontPlatform: {1},
}

// FeeTable is a function's fees (Feature.ini ClanHall*): the payment period
// in milliseconds and the fee of each level index.
type FeeTable struct {
	Ratio int64
	Fees  []int
}

// FunctionFee is the fee of a function level as ClanHallManager's switches
// give it: level 0 costs nothing; a level of the function's list costs its
// fee; any other level the last one.
func FunctionFee(typ, level int, t FeeTable) int {
	if level == 0 || len(t.Fees) == 0 {
		return 0
	}
	for i, l := range feeLevels[typ] {
		if l == level && i < len(t.Fees) {
			return t.Fees[i]
		}
	}
	return t.Fees[len(t.Fees)-1]
}

// FunctionChange is what Mobius ClanHall.updateFunctions does.
type FunctionChange int

// The changes.
const (
	// FuncAdd rents a new function: the first period is paid from the
	// player's inventory, the next from the clan warehouse.
	FuncAdd FunctionChange = iota
	// FuncRemove deactivates a function (level and fee 0).
	FuncRemove
	// FuncRaise replaces a function with a dearer level: the new fee is
	// paid from the inventory and the next payment is due at once (EndTime
	// -1, which takes no fee).
	FuncRaise
	// FuncKeep changes the level and fee in place, due when it was.
	FuncKeep
)

// ChangeFunction is the branch of ClanHall.updateFunctions for a function
// rented (old, nil for none) and the new level and lease.
func ChangeFunction(old *HallFunction, level, lease int) FunctionChange {
	switch {
	case old == nil:
		return FuncAdd
	case level == 0 && lease == 0:
		return FuncRemove
	case lease-old.Lease > 0:
		return FuncRaise
	}
	return FuncKeep
}

// FunctionPayment is ClanHallFunction.FunctionTask.run for a hall with an
// owner: fromWarehouse is the task's _cwh (false for the first run after a
// rent or raise, paid from the inventory already). It returns the fee to
// take from the warehouse (0 for none), the next end time, or remove when
// the warehouse cannot pay the fee. A function at EndTime -1 (raised) takes
// its temporary fee, which is always 0 in Mobius.
func FunctionPayment(f HallFunction, fromWarehouse bool, now, warehouseAdena int64) (pay int, end int64, remove bool) {
	if fromWarehouse && warehouseAdena < int64(f.Lease) {
		return 0, f.EndTime, true
	}
	fee := f.Lease
	if f.EndTime == -1 {
		fee = 0
	}
	if !fromWarehouse {
		fee = 0
	}
	return fee, now + f.Rate, false
}

// FunctionFirstWait is ClanHallFunction.initializeTask: the time until the
// function's task runs.
func FunctionFirstWait(f HallFunction, now int64) time.Duration {
	if f.EndTime > now {
		return time.Duration(f.EndTime-now) * time.Millisecond
	}
	return 0
}

// LeaseStep is one run of AuctionableHall.FeeTask.
type LeaseStep struct {
	// Pay takes the lease from the clan warehouse; PaidUntil and Paid are the
	// lease state after the run.
	Pay       bool
	PaidUntil int64
	Paid      bool
	// Warn tells the clan the lease is unpaid (MsgHallLeaseUnpaid); Revoke
	// frees the hall (MsgHallLeaseRevoked), which goes back to an NPC
	// auction.
	Warn, Revoke bool
	// Save writes the hall.
	Save bool
	// Wait is when the task runs again (not after Revoke).
	Wait time.Duration
}

// NextLeaseStep is AuctionableHall.FeeTask.run for a hall with an owner at
// now (Unix milliseconds), with the warehouse's adena and the lease. A lease
// paid until the future waits for it; otherwise the warehouse pays the week
// (from the old paid-until date, so missed weeks are not skipped, or from
// now for a hall just bought), or the clan is warned every day for a week
// and then loses the hall.
func NextLeaseStep(now, paidUntil, warehouseAdena int64, lease int) LeaseStep {
	rate := LeaseRate.Milliseconds()
	st := LeaseStep{PaidUntil: paidUntil}
	if paidUntil > now {
		st.Paid = true
		st.Wait = time.Duration(paidUntil-now) * time.Millisecond
		return st
	}
	if warehouseAdena >= int64(lease) {
		if paidUntil != 0 {
			for st.PaidUntil <= now {
				st.PaidUntil += rate
			}
		} else {
			st.PaidUntil = now + rate
		}
		st.Pay, st.Paid, st.Save = true, true, true
		st.Wait = time.Duration(st.PaidUntil-now) * time.Millisecond
		return st
	}
	if now > paidUntil+rate {
		st.Revoke = true
		return st
	}
	st.Warn, st.Save = true, true
	if now+leaseRecheck.Milliseconds() <= paidUntil+rate {
		st.Wait = leaseRecheck
	} else {
		// Mobius waits until paidUntil + rate exactly, then once more; the
		// extra millisecond lands past it so the hall is revoked.
		st.Wait = time.Duration(paidUntil+rate-now+1) * time.Millisecond
	}
	return st
}

// LeaseFirstWait is AuctionableHall.initialyzeTask: the time until the fee
// task first runs. forced is set when a clan just got the hall (setOwner),
// which pays at once. A missed lease is checked again after leaseRecheck (the
// Mobius quirk changed, see NextLeaseStep).
func LeaseFirstWait(now, paidUntil int64, paid, forced bool) time.Duration {
	rate := LeaseRate.Milliseconds()
	switch {
	case paidUntil > now:
		return time.Duration(paidUntil-now) * time.Millisecond
	case !paid && !forced:
		if now+leaseRecheck.Milliseconds() <= paidUntil+rate {
			return leaseRecheck
		}
		return time.Duration(max(paidUntil+rate-now, 0)) * time.Millisecond
	}
	return 0
}
