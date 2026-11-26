package residence

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/calendar"
)

func TestFunctionFee(t *testing.T) {
	hp := FeeTable{Ratio: 86_400_000, Fees: []int{700, 800, 1000, 1166, 1500, 1750, 2000, 2250, 2500, 3250, 3750, 4250, 5166}}
	sup := FeeTable{Fees: []int{2500, 5000, 7000, 11000, 21000, 36000, 37000, 364000}}
	tele := FeeTable{Fees: []int{7000, 14000}}
	for _, c := range []struct {
		typ, level int
		t          FeeTable
		want       int
	}{
		{HallFuncRestoreHP, 0, hp, 0},
		{HallFuncRestoreHP, 20, hp, 700},
		{HallFuncRestoreHP, 260, hp, 4250},
		{HallFuncRestoreHP, 300, hp, 5166}, // the switch's default
		{HallFuncRestoreHP, 999, hp, 5166},
		{HallFuncSupport, 7, sup, 37000},
		{HallFuncSupport, 8, sup, 364000},
		{HallFuncTeleport, 1, tele, 7000},
		{HallFuncTeleport, 2, tele, 14000},
	} {
		if got := FunctionFee(c.typ, c.level, c.t); got != c.want {
			t.Errorf("FunctionFee(%d, %d) = %d, want %d", c.typ, c.level, got, c.want)
		}
	}
}

func TestChangeFunction(t *testing.T) {
	old := &HallFunction{Type: HallFuncRestoreHP, Level: 40, Lease: 800}
	for _, c := range []struct {
		old          *HallFunction
		level, lease int
		want         FunctionChange
	}{
		{nil, 20, 700, FuncAdd},
		{old, 0, 0, FuncRemove},
		{old, 80, 1000, FuncRaise},
		{old, 20, 700, FuncKeep},
	} {
		if got := ChangeFunction(c.old, c.level, c.lease); got != c.want {
			t.Errorf("ChangeFunction(%v, %d, %d) = %d, want %d", c.old, c.level, c.lease, got, c.want)
		}
	}
}

func TestFunctionPayment(t *testing.T) {
	f := HallFunction{Lease: 800, Rate: 86_400_000, EndTime: 0}
	// The first run after a rent: paid from the inventory already.
	if pay, end, rm := FunctionPayment(f, false, 1000, 0); pay != 0 || end != 1000+86_400_000 || rm {
		t.Errorf("first run = %d, %d, %v", pay, end, rm)
	}
	if pay, _, rm := FunctionPayment(f, true, 1000, 800); pay != 800 || rm {
		t.Errorf("warehouse run = %d, %v", pay, rm)
	}
	if _, _, rm := FunctionPayment(f, true, 1000, 799); !rm {
		t.Error("a poor warehouse kept the function")
	}
	f.EndTime = -1 // raised: the temporary fee is 0
	if pay, _, rm := FunctionPayment(f, true, 1000, 800); pay != 0 || rm {
		t.Errorf("raised run = %d, %v", pay, rm)
	}
}

func TestLeaseSteps(t *testing.T) {
	const lease = 500_000
	rate := LeaseRate.Milliseconds()
	// A hall just bought pays its first week at once.
	now := int64(1_000_000_000_000)
	st := NextLeaseStep(now, 0, 600_000, lease)
	if !st.Pay || st.PaidUntil != now+rate || !st.Paid || st.Wait != LeaseRate {
		t.Fatalf("first payment %+v", st)
	}
	// A week later the next one; missed weeks are not skipped.
	st = NextLeaseStep(now+rate, now+rate, 600_000, lease)
	if !st.Pay || st.PaidUntil != now+2*rate {
		t.Fatalf("second payment %+v", st)
	}
	// Not enough adena: warned daily, then revoked after a week.
	paid := now + rate
	at := paid
	warns := 0
	for range 20 {
		st = NextLeaseStep(at, paid, 1000, lease)
		if st.Revoke {
			break
		}
		if !st.Warn || st.Paid {
			t.Fatalf("unpaid step %+v", st)
		}
		warns++
		at += st.Wait.Milliseconds()
	}
	if !st.Revoke || warns != 8 || at <= paid+rate {
		t.Fatalf("revoked %v after %d warnings at %d (paid until %d)", st.Revoke, warns, at-paid, rate)
	}
	// Paid in the future: wait.
	if st := NextLeaseStep(now, now+5000, 0, lease); st.Pay || st.Wait != 5*time.Second {
		t.Errorf("future %+v", st)
	}
}

func TestLeaseFirstWait(t *testing.T) {
	rate := LeaseRate.Milliseconds()
	now := int64(1_000_000_000_000)
	if w := LeaseFirstWait(now, now+10_000, true, false); w != 10*time.Second {
		t.Errorf("paid ahead: %v", w)
	}
	if w := LeaseFirstWait(now, now, false, true); w != 0 {
		t.Errorf("forced: %v", w)
	}
	if w := LeaseFirstWait(now, now-1000, false, false); w != 24*time.Hour {
		t.Errorf("unpaid with grace: %v", w)
	}
	if w := LeaseFirstWait(now, now-rate+1000, false, false); w != time.Second {
		t.Errorf("unpaid, last day: %v", w)
	}
}

func TestAuctionBids(t *testing.T) {
	a := &Auction{ID: 22, ItemName: "Moonstone Hall", StartingBid: 20_000_000}
	if r := a.PlaceBid(BidRequest{ClanID: 1, Bid: 19_000_000, Warehouse: 1e9}); r.Placed || r.Msg != MsgBidTooLow {
		t.Fatalf("low bid %+v", r)
	}
	if r := a.PlaceBid(BidRequest{ClanID: 1, Bid: 20_000_000, Warehouse: 1}); r.Placed || r.Msg != MsgNotEnoughHallAdena {
		t.Fatalf("poor bid %+v", r)
	}
	r := a.PlaceBid(BidRequest{ClanID: 1, ClanName: "Lions", LeaderName: "Alice", PlayerName: "Alice", Bid: 20_000_000, Warehouse: 1e9, Now: 5})
	if !r.Placed || r.Take != 20_000_000 || r.Msg != MsgBidPlaced || r.Outbid != 0 {
		t.Fatalf("first bid %+v", r)
	}
	// Equal is too low; a higher bid outbids Lions.
	if r := a.PlaceBid(BidRequest{ClanID: 2, Bid: 20_000_000, Warehouse: 1e9}); r.Placed || r.Msg != MsgBidTooLow {
		t.Fatalf("equal bid %+v", r)
	}
	r = a.PlaceBid(BidRequest{ClanID: 2, ClanName: "Wolves", LeaderName: "Bob", PlayerName: "Bob", Bid: 21_000_000, Warehouse: 1e9})
	if !r.Placed || r.Take != 21_000_000 || r.Outbid != 1 {
		t.Fatalf("second bid %+v", r)
	}
	// The leader raising pays the difference.
	r = a.PlaceBid(BidRequest{ClanID: 2, LeaderName: "Bob", Bid: 22_000_000, Warehouse: 1e9})
	if !r.Placed || r.Take != 1_000_000 || len(a.Bidders) != 2 {
		t.Fatalf("raise %+v (%d bidders)", r, len(a.Bidders))
	}
	if h, _ := a.Highest(); h.ClanID != 2 || h.Bid != 22_000_000 || a.MinimumBid() != 22_000_000 {
		t.Errorf("highest %+v", h)
	}
	if a.End() != AuctionSold {
		t.Error("an auction with bids does not sell")
	}
	if b, ok := a.RemoveBidder(2); !ok || b.Bid != 22_000_000 || len(a.Bidders) != 1 {
		t.Errorf("RemoveBidder = %+v, %v", b, ok)
	}
	if got := Refund(22_000_000, true, 0); got != 19_800_000 {
		t.Errorf("Refund = %d", got)
	}
	if got := Refund(100, false, MaxAdena-10); got != 10 {
		t.Errorf("capped Refund = %d", got)
	}
	empty := &Auction{SellerID: 0}
	if empty.End() != AuctionRestart {
		t.Error("an NPC auction without bids does not restart")
	}
	empty.SellerID = 7
	if empty.End() != AuctionDropped {
		t.Error("a sale without bids is not dropped")
	}
	if w, end := AuctionWait(100, 50); w != 0 || end != 100+AuctionLength.Milliseconds() {
		t.Errorf("AuctionWait past = %v, %d", w, end)
	}
}

func TestNextHallSiege(t *testing.T) {
	cal := calendar.New(time.UTC)
	now := time.Date(2026, 11, 25, 15, 30, 10, 123_000_000, time.UTC)
	got := NextHallSiege(cal, now, [5]int{14, 0, 0, 12, 0})
	if want := time.Date(2026, 12, 9, 12, 0, 0, 123_000_000, time.UTC); !got.Equal(want) {
		t.Errorf("NextHallSiege = %v, want %v", got, want)
	}
	// A month added to January 31 is February 28.
	got = NextHallSiege(cal, time.Date(2027, 1, 31, 0, 0, 0, 0, time.UTC), [5]int{0, 1, 0, 20, 30})
	if want := time.Date(2027, 2, 28, 20, 30, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("NextHallSiege + 1 month = %v, want %v", got, want)
	}
	if !HallSiegeDue(now, now.Add(59*time.Minute)) || HallSiegeDue(now, now.Add(2*time.Hour)) {
		t.Error("HallSiegeDue")
	}
}

func TestCheckHallRegister(t *testing.T) {
	base := HallRegisterCheck{ClanID: 5, ClanLevel: 4, MinLevel: 4, MaxAttackers: 2, MinLevelMessage: "low"}
	for _, c := range []struct {
		name string
		mod  func(*HallRegisterCheck)
		want HallRegisterResult
	}{
		{"ok", func(*HallRegisterCheck) {}, HallRegisterResult{OK: true}},
		{"level", func(c *HallRegisterCheck) { c.ClanLevel = 3 }, HallRegisterResult{Text: "low"}},
		{"waiting", func(c *HallRegisterCheck) { c.Status = HallWaitingBattle }, HallRegisterResult{SysMsg: MsgRegistrationDeadline, WithArg: true}},
		{"running", func(c *HallRegisterCheck) { c.Status = HallRunning }, HallRegisterResult{SysMsg: MsgNotTimeToRegister}},
		{"owner", func(c *HallRegisterCheck) { c.OwnerID = 5 }, HallRegisterResult{SysMsg: MsgOwnerIsDefender}},
		{"hall owner", func(c *HallRegisterCheck) { c.ClanHideoutID = 22 }, HallRegisterResult{SysMsg: MsgOwnsAnotherCastle}},
		{"twice", func(c *HallRegisterCheck) { c.Attacker = true }, HallRegisterResult{SysMsg: MsgAlreadyRequested}},
		{"other hall", func(c *HallRegisterCheck) { c.OtherHall = true }, HallRegisterResult{SysMsg: MsgAnotherSiegeSameDay}},
		{"full", func(c *HallRegisterCheck) { c.Attackers = 2 }, HallRegisterResult{SysMsg: MsgNoMoreAttackers}},
	} {
		chk := base
		c.mod(&chk)
		if got := CheckHallRegister(chk); got != c.want {
			t.Errorf("%s: %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestMostDamage(t *testing.T) {
	order := []int32{3, 1, 2}
	if got := MostDamage(order, map[int32]int64{3: 100, 1: 250, 2: 250}); got != 1 {
		t.Errorf("MostDamage = %d, want 1 (the first to reach the most)", got)
	}
	if got := MostDamage(nil, nil); got != 0 {
		t.Errorf("MostDamage of nothing = %d", got)
	}
}
