package manor

import (
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/calendar"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
)

func TestInitialMode(t *testing.T) {
	s := DefaultSettings()
	for _, c := range []struct {
		hour, minute int
		want         Mode
	}{
		{0, 0, Modifiable}, {5, 59, Modifiable}, {6, 0, Modifiable}, {6, 1, Approved}, {12, 0, Approved},
		{19, 59, Approved}, {20, 0, Maintenance}, {20, 5, Maintenance}, {20, 6, Modifiable}, {20, 59, Modifiable},
		// Mobius quirk: the first minutes of the later hours are APPROVED.
		{21, 3, Approved}, {21, 6, Modifiable}, {23, 0, Approved}, {23, 30, Modifiable},
	} {
		if got := InitialMode(c.hour, c.minute, s); got != c.want {
			t.Errorf("InitialMode(%02d:%02d) = %v, want %v", c.hour, c.minute, got, c.want)
		}
	}
	if Approved.Next() != Maintenance || Maintenance.Next() != Modifiable || Modifiable.Next() != Approved || Disabled.Next() != Disabled {
		t.Error("Next cycle broken")
	}
	if Maintenance.String() != "MAINTENANCE" || Disabled.String() != "DISABLED" {
		t.Error("mode names")
	}
}

// Two days of mode changes from a Wednesday noon: the change times keep the
// start's milliseconds, as Java's Calendar does.
func TestNextChangeTwoDays(t *testing.T) {
	cal := calendar.New(time.UTC)
	s := DefaultSettings()
	now := time.Date(2026, 11, 25, 12, 0, 0, 437e6, time.UTC)
	mode := InitialMode(12, 0, s)
	want := []struct {
		at   time.Time
		mode Mode
	}{
		{time.Date(2026, 11, 25, 20, 0, 0, 437e6, time.UTC), Maintenance},
		{time.Date(2026, 11, 25, 20, 6, 0, 437e6, time.UTC), Modifiable},
		{time.Date(2026, 11, 26, 6, 0, 0, 437e6, time.UTC), Approved},
		{time.Date(2026, 11, 26, 20, 0, 0, 437e6, time.UTC), Maintenance},
		{time.Date(2026, 11, 26, 20, 6, 0, 437e6, time.UTC), Modifiable},
		{time.Date(2026, 11, 27, 6, 0, 0, 437e6, time.UTC), Approved},
	}
	for i, w := range want {
		next := NextChange(mode, now, cal, s)
		if !next.Equal(w.at) {
			t.Fatalf("change %d from %v at %v: %v, want %v", i, mode, now, next, w.at)
		}
		now, mode = next, mode.Next()
		if mode != w.mode {
			t.Fatalf("change %d: mode %v, want %v", i, mode, w.mode)
		}
	}
	if !NextChange(Disabled, now, cal, s).IsZero() {
		t.Error("a disabled manor changes")
	}
	// At the change time itself the change is a second away.
	at := time.Date(2026, 11, 27, 20, 0, 0, 0, time.UTC)
	if got := NextChange(Approved, at, cal, s); !got.Equal(at.Add(time.Second)) {
		t.Errorf("NextChange at the time = %v", got)
	}
}

func TestJavaHashOrder(t *testing.T) {
	// 16 buckets: 17 and 33 share bucket 1 with 1, in insertion order.
	if got := javaHashOrder([]int{33, 2, 1, 17}); !slices.Equal(got, []int{0, 2, 3, 1}) {
		t.Errorf("javaHashOrder = %v", got)
	}
	// 13 keys make 32 buckets: 16 and 48 share bucket 16, after 1-11, and
	// keep their insertion order.
	keys := []int{48, 16, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	got := javaHashOrder(keys)
	var vals []int
	for _, i := range got {
		vals = append(vals, keys[i])
	}
	if !slices.Equal(vals, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 48, 16}) {
		t.Errorf("javaHashOrder values = %v", vals)
	}
}

var loadSeeds = sync.OnceValues(func() (*data.Data, error) {
	return data.Load(filepath.Join("..", "..", "data"))
})

func realTable(t *testing.T) *Table {
	t.Helper()
	d, err := loadSeeds()
	if err != nil {
		t.Fatal(err)
	}
	return NewTable(d.Seeds, func(id int) (int64, bool) {
		it, ok := d.ItemTemplate(id)
		if !ok {
			return 0, false
		}
		return int64(it.Price), true
	}, 1)
}

// The committed Seeds.xml: 258 seeds; seeds of one crop agree on what
// getSeedByCrop(cropId) gives, so its map order does not matter.
func TestRealTable(t *testing.T) {
	tb := realTable(t)
	if tb.Len() != 258 {
		t.Fatalf("%d seeds", tb.Len())
	}
	for _, s := range tb.Seeds() {
		first := tb.SeedByCrop(s.CropID)
		if first.MatureID != s.MatureID || first.Reward1 != s.Reward1 || first.Reward2 != s.Reward2 || first.Level != s.Level {
			t.Errorf("seed %d and %d of crop %d differ", first.SeedID, s.SeedID, s.CropID)
		}
		if tb.SeedByCropIn(s.CropID, s.CastleID) == nil {
			t.Errorf("crop %d not in castle %d", s.CropID, s.CastleID)
		}
	}
	s := tb.Seed(5016) // Seed: Dark Coda, Gludio
	if s == nil || s.CastleID != 1 || s.CropID != 5073 || s.SeedPrice != 5 || s.SeedLimit != 8100 || s.SeedMinPrice() != 3 || s.SeedMaxPrice() != 50 {
		t.Errorf("Seed(5016) = %+v", s)
	}
	if !tb.IsCrop(5073) || tb.IsCrop(5016) || len(tb.CropIDs()) != len(tb.Crops()) {
		t.Error("crop IDs")
	}
	for id := 1; id <= 9; id++ {
		if len(tb.SeedsFor(id)) == 0 {
			t.Errorf("castle %d has no seeds", id)
		}
	}
}

// testTable is two seeds of castle 1 and one of castle 2, all of one crop
// pair.
func testTable() *Table {
	list := []data.Seed{
		{CastleID: 1, SeedID: 5016, CropID: 5073, MatureID: 5103, Reward1: 1864, Reward2: 1878, Level: 10, LimitSeeds: 100, LimitCrops: 50},
		{CastleID: 1, SeedID: 5650, CropID: 5818, MatureID: 5838, Reward1: 1864, Reward2: 1878, Alternative: true, Level: 10, LimitSeeds: 200, LimitCrops: 20},
		{CastleID: 2, SeedID: 5221, CropID: 5073, MatureID: 5103, Reward1: 1864, Reward2: 1878, Level: 10, LimitSeeds: 100, LimitCrops: 50},
	}
	prices := map[int]int64{5016: 10, 5650: 20, 5221: 10, 5073: 100, 5818: 200}
	return NewTable(list, func(id int) (int64, bool) { p, ok := prices[id]; return p, ok }, 2)
}

func TestFilters(t *testing.T) {
	tb := testTable()
	got := tb.FilterProduction([]Production{
		{SeedID: 5016, Amount: 200, Price: 6, StartAmount: 200},   // limit 100×2, min 6: kept
		{SeedID: 5016, Amount: 201, Price: 50, StartAmount: 201},  // over the limit
		{SeedID: 5650, Amount: 10, Price: 201, StartAmount: 10},   // over 10× the price
		{SeedID: 5650, Amount: 10, Price: 11, StartAmount: 10},    // under 60 %
		{SeedID: 9999, Amount: 1, Price: 1, StartAmount: 1},       // unknown
		{SeedID: 5650, Amount: 400, Price: 200, StartAmount: 400}, // kept
	})
	if len(got) != 2 || got[0].SeedID != 5016 || got[1].StartAmount != 400 {
		t.Errorf("FilterProduction = %+v", got)
	}
	crops := tb.FilterProcure([]Procure{
		{CropID: 5073, Amount: 100, Price: 60, StartAmount: 100, Reward: 1}, // kept
		{CropID: 5818, Amount: 41, Price: 200, StartAmount: 41, Reward: 2},  // over 20×2
		{CropID: 5073, Amount: 1, Price: 1001, StartAmount: 1, Reward: 1},   // too dear
	}, 1)
	if len(crops) != 1 || crops[0].CropID != 5073 {
		t.Errorf("FilterProcure = %+v", crops)
	}
	if got := tb.FilterProcure([]Procure{{CropID: 5818, Amount: 1, Price: 200, StartAmount: 1}}, 2); len(got) != 0 {
		t.Errorf("a crop of another castle kept: %+v", got)
	}
}

// The maintenance and approval bookkeeping of one castle over two days.
func TestMaintenanceAndApprove(t *testing.T) {
	tb := testTable()
	c := &Castle{
		Production: []Production{{SeedID: 5016, Amount: 30, Price: 10, StartAmount: 100}},
		Procure: []Procure{
			{CropID: 5073, Amount: 40, Price: 100, StartAmount: 50, Reward: 1}, // 10 sold: 9 mature
			{CropID: 5818, Amount: 19, Price: 200, StartAmount: 20, Reward: 2}, // 1 sold: 0.9 → 1 with the roll
			{CropID: 5818, Amount: 0, Price: 0, StartAmount: 0},                // not bought
		},
	}
	c.SetNextProduction([]Production{{SeedID: 5016, Amount: 100, Price: 12, StartAmount: 100}})
	c.SetNextProcure([]Procure{{CropID: 5073, Amount: 10, Price: 100, StartAmount: 10, Reward: 2}})
	if cost := tb.Cost(c, true); cost != 100*10+10*100 {
		t.Fatalf("next cost %d", cost)
	}
	if income := c.SeedIncome(); income != 70*10 {
		t.Errorf("seed income %d", income)
	}
	r := rnd.NewScripted(t, rnd.Get(99, 89))
	crops, treasury := tb.Maintenance(c, 5000, r)
	if want := []Harvest{{MatureID: 5103, Count: 9}, {MatureID: 5838, Count: 1}}; !slices.Equal(crops, want) {
		t.Errorf("harvest %+v, want %+v", crops, want)
	}
	// The unspent money: 40×100 + 19×200.
	if treasury != 5000+4000+3800 {
		t.Errorf("treasury %d", treasury)
	}
	if len(c.Production) != 1 || c.Production[0].Price != 12 || len(c.ProductionNext) != 1 || !c.ProductionShared {
		t.Errorf("after the maintenance: %+v", c)
	}
	// A purchase lowers the shared next period too (Mobius quirk).
	if !c.DecreaseSeed(5016, 30) || c.ProductionNext[0].Amount != 70 || c.DecreaseSeed(5016, 71) {
		t.Errorf("shared decrease: %+v", c)
	}
	// The approval pays the next period.
	treasury, cleared := tb.Approve(c, treasury, func(int) bool { return true })
	if cleared || treasury != 12800-tb.Cost(c, true) {
		t.Errorf("approve: treasury %d, cleared %v", treasury, cleared)
	}
	// A full warehouse and a poor castle empty the next period.
	_, cleared = tb.Approve(c, 1, func(int) bool { return false })
	if !cleared || c.ProductionNext != nil || c.ProcureNext != nil {
		t.Errorf("approve when poor: %+v", c)
	}
	// A poor castle keeps its lists when the warehouse fits, and pays nothing.
	c.SetNextProcure([]Procure{{CropID: 5073, Amount: 10, Price: 100, StartAmount: 10}})
	if tr, cl := tb.Approve(c, 5, func(int) bool { return true }); cl || tr != 5 || len(c.ProcureNext) != 1 {
		t.Errorf("approve too poor to pay: %d %v", tr, cl)
	}
	// A treasury that cannot pay the new period empties the next one; the
	// running one keeps its amounts.
	c.Procure[0].Amount = 3
	_, treasury = tb.Maintenance(c, 0, rnd.NewScripted(t))
	if c.ProductionNext != nil || c.ProcureNext != nil || len(c.Procure) != 1 || c.Procure[0].Amount != 10 || treasury != 300 {
		t.Errorf("poor maintenance: %+v, treasury %d", c, treasury)
	}
	c.Reset()
	if c.Production != nil || c.ProcureShared {
		t.Error("Reset")
	}
}

func TestSowAndHarvestChances(t *testing.T) {
	s := &Seed{Seed: data.Seed{Level: 20}}
	alt := &Seed{Seed: data.Seed{Level: 20, Alternative: true}}
	for _, c := range []struct {
		seed          *Seed
		player, mob   int
		want, harvest int
	}{
		{s, 20, 20, 90, 100},
		{alt, 20, 20, 20, 100},
		{s, 20, 25, 90, 100},
		{s, 20, 27, 70, 90},  // 2 levels over the seed's range, 2 over the gap
		{s, 40, 13, -30, 1},  // 2 under the seed's range, 22 over the gap
		{alt, 40, 40, -55, 100},
	} {
		if got := SowChance(c.seed, c.player, c.mob); got != c.want {
			t.Errorf("SowChance(%d, %d) = %d, want %d", c.player, c.mob, got, c.want)
		}
		if got := HarvestChance(c.player, c.mob); got != c.harvest {
			t.Errorf("HarvestChance(%d, %d) = %d, want %d", c.player, c.mob, got, c.harvest)
		}
	}
	if HarvestCount(s, 20, nil, 1) != 1 || HarvestCount(s, 30, []int{4303, 4416, 4305}, 2) != (1*2*4+5)*2 {
		t.Error("HarvestCount")
	}
	if !KeptWhenSeeded(57) || !KeptWhenSeeded(6361) || KeptWhenSeeded(1864) {
		t.Error("KeptWhenSeeded")
	}
}

// Seeded distribution: 90 % sown, 50 % harvested at a 14-level gap.
func TestSowDistribution(t *testing.T) {
	s := &Seed{Seed: data.Seed{Level: 20}}
	r := rnd.New(11)
	const n = 100000
	sown, harvested := 0, 0
	for range n {
		if SowSucceeds(r, s, 20, 20) {
			sown++
		}
		if HarvestSucceeds(r, 34, 20) {
			harvested++
		}
	}
	// Rnd.get(99) < 90 is 90/99, < 55 is 55/99.
	for _, c := range []struct {
		name      string
		got, want float64
	}{{"sown", float64(sown) / n, 90.0 / 99}, {"harvested", float64(harvested) / n, 55.0 / 99}} {
		if d := c.got - c.want; d > 0.01 || d < -0.01 {
			t.Errorf("%s rate %.4f, want %.4f", c.name, c.got, c.want)
		}
	}
}
