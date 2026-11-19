package serverpackets

import (
	"encoding/binary"
	"encoding/hex"
	"strings"
	"testing"
)

// The helpers of the M6 (combat) and M7 (skills) golden tests. Each test
// builds the expected packet by hand from the Mobius writeImpl quoted above
// it.

// m67Golden builds an expected packet field by field, like the Mobius
// WriteBuffer, without the production packets.Writer or put helpers:
// c byte, h short, d int, q long.
type m67Golden struct{ b []byte }

func (g *m67Golden) c(v byte) *m67Golden { g.b = append(g.b, v); return g }

func (g *m67Golden) h(v int16) *m67Golden {
	g.b = binary.LittleEndian.AppendUint16(g.b, uint16(v))
	return g
}

func (g *m67Golden) d(v int32) *m67Golden {
	g.b = binary.LittleEndian.AppendUint32(g.b, uint32(v))
	return g
}

func (g *m67Golden) q(v int64) *m67Golden {
	g.b = binary.LittleEndian.AppendUint64(g.b, uint64(v))
	return g
}

// m67Hex decodes hex written with spaces and line breaks.
func m67Hex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.Join(strings.Fields(s), ""))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// m67Check compares a built packet with the expected bytes and, on a
// mismatch, reports the first differing offset as well.
func m67Check(t *testing.T, name string, got, want []byte) {
	t.Helper()
	if string(got) == string(want) {
		return
	}
	at := min(len(got), len(want))
	for i := range at {
		if got[i] != want[i] {
			at = i
			break
		}
	}
	t.Errorf("%s: first difference at offset %d (got %d bytes, want %d)\n got % x\nwant % x", name, at, len(got), len(want), got, want)
}

// The opcodes are those of Mobius ServerPackets (Interlude).
func TestM67ServerOpcodes(t *testing.T) {
	tests := []struct {
		name      string
		got, want int
	}{
		{"Attack", OpAttack, 0x05},
		{"Die", OpDie, 0x06},
		{"Revive", OpRevive, 0x07},
		{"AutoAttackStart", OpAutoAttackStart, 0x2b},
		{"AutoAttackStop", OpAutoAttackStop, 0x2c},
		{"ShortcutRegister", OpShortcutRegister, 0x44},
		{"MagicSkillUse", OpMagicSkillUse, 0x48},
		{"MagicSkillCanceled", OpMagicSkillCanceled, 0x49},
		{"SetupGauge", OpSetupGauge, 0x6d},
		{"MagicSkillLaunched", OpMagicSkillLaunched, 0x76},
		{"AcquireSkillList", OpAcquireSkillList, 0x8a},
		{"AcquireSkillInfo", OpAcquireSkillInfo, 0x8b},
		{"AcquireSkillDone", OpAcquireSkillDone, 0x8e},
		{"PartySpelled", OpPartySpelled, 0xee},
		{"ShortBuffStatusUpdate", OpShortBuffStatusUpdate, 0xf4},
		{"Ex ExAutoSoulShot", ExOpAutoSoulShot, 0x12},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s opcode = %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}

// The enum values are those of Mobius: Hit.HITFLAG_*, the AcquireSkillType
// ordinals, the SetupGauge colors and the PartySpelled creature types.
func TestM67ServerEnums(t *testing.T) {
	tests := []struct {
		name      string
		got, want int64
	}{
		{"HITFLAG_USESS", HitFlagSoulshot, 0x10},
		{"HITFLAG_CRIT", HitFlagCritical, 0x20},
		{"HITFLAG_SHLD", HitFlagShield, 0x40},
		{"HITFLAG_MISS", HitFlagMiss, 0x80},
		{"AcquireSkillType.CLASS", int64(AcquireSkillClass), 0},
		{"AcquireSkillType.FISHING", int64(AcquireSkillFishing), 1},
		{"AcquireSkillType.PLEDGE", int64(AcquireSkillPledge), 2},
		{"SetupGauge.BLUE", int64(GaugeBlue), 0},
		{"SetupGauge.RED", int64(GaugeRed), 1},
		{"SetupGauge.CYAN", int64(GaugeCyan), 2},
		{"green", int64(GaugeGreen), 3},
		{"player", int64(PartySpelledPlayer), 0},
		{"pet", int64(PartySpelledPet), 1},
		{"servitor", int64(PartySpelledServitor), 2},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}
