package stats

import (
	"math"
	"testing"
)

func TestStatNames(t *testing.T) {
	if NumStats != 126 {
		t.Fatalf("NumStats = %d, want 126 (Mobius Stat.java)", NumStats)
	}
	seen := make(map[string]Stat, NumStats)
	for i := range NumStats {
		s := Stat(i)
		name := s.String()
		if name == "" {
			t.Errorf("Stat(%d) has no name", i)
			continue
		}
		if prev, dup := seen[name]; dup {
			t.Errorf("Stat(%d) and Stat(%d) are both named %q", prev, i, name)
		}
		seen[name] = s
		if got, ok := ParseStat(name); !ok || got != s {
			t.Errorf("ParseStat(%q) = %v, %v; want %d, true", name, got, ok, i)
		}
	}
}

// TestStatOrdinals spot-checks that the constants follow the Mobius enum
// order (Stat.ordinal()).
func TestStatOrdinals(t *testing.T) {
	tests := []struct {
		stat Stat
		want int
		name string
	}{
		{MaxHP, 0, "maxHp"},
		{RegHP, 6, "regHp"},
		{RegCP, 7, "regCp"},
		{PDef, 11, "pDef"},
		{PAtk, 13, "pAtk"},
		{PAtkSpd, 16, "pAtkSpd"},
		{MAtkSpd, 17, "mAtkSpd"},
		{SDef, 22, "sDef"},
		{REvas, 38, "rEvas"},
		{RShld, 44, "rShld"},
		{CritRate, 45, "critRate"},
		{MCritRate, 48, "mCritRate"},
		{Cancel, 56, "cancel"},
		{AccCombat, 57, "accCombat"},
		{PAtkRange, 58, "pAtkRange"},
		{RunSpd, 61, "runSpd"},
		{STR, 62, "STR"},
		{MEN, 67, "MEN"},
		{MagicSuccRes, 81, "magicSuccRes"},
		{FirePower, 83, "firePower"},
		{WeightLimit, 100, "weightLimit"},
		{InventoryLimit, 102, "inventoryLimit"},
		{RandomDamage, 125, "randomDamage"},
	}
	for _, tt := range tests {
		if int(tt.stat) != tt.want || tt.stat.String() != tt.name {
			t.Errorf("stat %q = %d named %q, want %d", tt.name, tt.stat, tt.stat.String(), tt.want)
		}
	}
}

func TestParseStat(t *testing.T) {
	if _, ok := ParseStat("PATK"); ok {
		t.Error(`ParseStat("PATK") succeeded; Mobius matches names exactly`)
	}
	if _, ok := ParseStat("unknown"); ok {
		t.Error(`ParseStat("unknown") succeeded`)
	}
	if got := Stat(NumStats).String(); got != "Stat(126)" {
		t.Errorf("Stat(126).String() = %q", got)
	}
	if Stat(NumStats).Valid() || !RandomDamage.Valid() {
		t.Error("Valid is wrong at the end of the range")
	}
}

func TestJavaRound(t *testing.T) {
	tests := []struct {
		in   float64
		want int64
	}{
		{2.5, 3},
		{-2.5, -2}, // ties go toward positive infinity
		{-0.5, 0},
		{-0.6, -1},
		{0.49999999999999994, 0}, // floor(x + 0.5) would give 1
		{33.86335345030997, 34},
		{416.90000000000003, 417},
		{math.NaN(), 0},
		{math.Inf(1), math.MaxInt64},
		{math.Inf(-1), math.MinInt64},
		{1e300, math.MaxInt64},
		{-1e300, math.MinInt64},
	}
	for _, tt := range tests {
		if got := javaRound(tt.in); got != tt.want {
			t.Errorf("javaRound(%v) = %d, want %d", tt.in, got, tt.want)
		}
	}
	// (int) Math.round(x) keeps the low 32 bits of the long.
	if got := javaRoundInt(1<<32 + 5); got != 5 {
		t.Errorf("javaRoundInt(2^32+5) = %d, want 5", got)
	}
}

func TestJavaInt(t *testing.T) {
	tests := []struct {
		in   float64
		want int32
	}{
		{1.9, 1},
		{-1.9, -1},
		{126.4, 126},
		{math.NaN(), 0},
		{3e9, math.MaxInt32},
		{-3e9, math.MinInt32},
		{2147483647.5, math.MaxInt32},
		{math.Inf(1), math.MaxInt32},
		{math.Inf(-1), math.MinInt32},
	}
	for _, tt := range tests {
		if got := javaInt(tt.in); got != tt.want {
			t.Errorf("javaInt(%v) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestF32(t *testing.T) {
	// Mobius keeps collision sizes in floats: 22.8 becomes 22.799999237060547.
	if got, want := f32(22.8), 22.799999237060547; got != want {
		t.Errorf("f32(22.8) = %v, want %v", got, want)
	}
	if got := f32(9); got != 9 {
		t.Errorf("f32(9) = %v, want 9", got)
	}
}
