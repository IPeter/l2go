package clientpackets

import (
	"encoding/binary"
	"errors"
	"fmt"
	"testing"

	"github.com/IPeter/l2go/internal/packets"
)

// The helpers of the M6 (combat) and M7 (skills) parser tests.

// m67Body builds a client packet body field by field, as the client sends
// it: c byte, d int.
type m67Body struct{ b []byte }

func (m *m67Body) c(v byte) *m67Body { m.b = append(m.b, v); return m }

func (m *m67Body) d(v int32) *m67Body {
	m.b = binary.LittleEndian.AppendUint32(m.b, uint32(v))
	return m
}

// m67Int reads the little-endian int at off, as the client wrote it.
func m67Int(data []byte, off int) int32 {
	return int32(binary.LittleEndian.Uint32(data[off:]))
}

// m67Parsed checks a successful parse.
func m67Parsed[T comparable](t *testing.T, name string, got T, err error, want T) {
	t.Helper()
	if err != nil {
		t.Errorf("%s: %v", name, err)
		return
	}
	if got != want {
		t.Errorf("%s:\n got %+v\nwant %+v", name, got, want)
	}
}

// m67Failed checks a failed parse: an error wrapping want, and the zero
// value.
func m67Failed[T comparable](t *testing.T, name string, got T, err, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Errorf("%s: error %v, want %v", name, err, want)
	}
	var zero T
	if got != zero {
		t.Errorf("%s: got %+v with the error, want the zero value", name, got)
	}
}

// m67Truncations checks that every proper prefix of full fails with
// ErrShortPacket.
func m67Truncations[T comparable](t *testing.T, name string, full []byte, parse func([]byte) (T, error)) {
	t.Helper()
	for n := range len(full) {
		got, err := parse(full[:n])
		m67Failed(t, fmt.Sprintf("%s, %d of %d bytes", name, n, len(full)), got, err, packets.ErrShortPacket)
	}
}

// m67Fuzzed checks the parse of fuzzed data: it must fail with
// ErrShortPacket below size bytes, with invalid when the data is long
// enough but rejected, and succeed otherwise. It reports whether the parse
// succeeded, so that the caller can check the fields.
func m67Fuzzed[T comparable](t *testing.T, data []byte, got T, err error, size int, rejected bool, invalid error) bool {
	t.Helper()
	switch {
	case len(data) < size:
		m67Failed(t, fmt.Sprintf("% x", data), got, err, packets.ErrShortPacket)
		return false
	case rejected:
		m67Failed(t, fmt.Sprintf("% x", data), got, err, invalid)
		return false
	case err != nil:
		t.Errorf("% x: %v", data, err)
		return false
	}
	return true
}

// The opcodes are those of Mobius ClientPackets and ExClientPackets
// (Interlude).
func TestM67ClientOpcodes(t *testing.T) {
	tests := []struct {
		name      string
		got, want int
	}{
		{"RequestMagicSkillUse", OpRequestMagicSkillUse, 0x2f},
		{"RequestShortcutReg", OpRequestShortcutReg, 0x33},
		{"RequestShortcutDel", OpRequestShortcutDel, 0x35},
		{"RequestSkillList", OpRequestSkillList, 0x3f},
		{"RequestAcquireSkillInfo", OpRequestAcquireSkillInfo, 0x6b},
		{"RequestAcquireSkill", OpRequestAcquireSkill, 0x6c},
		{"RequestRestartPoint", OpRequestRestartPoint, 0x6d},
		{"Ex RequestAutoSoulShot", ExOpRequestAutoSoulShot, 0x05},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s opcode = %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}

// The values are those of Mobius: the restart point cases of
// RequestRestartPoint, the AcquireSkillType ordinals, and the
// RequestAutoSoulShot types.
func TestM67ClientEnums(t *testing.T) {
	tests := []struct {
		name      string
		got, want int
	}{
		{"village", RestartPointVillage, 0},
		{"clan hall", RestartPointClanHall, 1},
		{"castle", RestartPointCastle, 2},
		{"siege HQ", RestartPointSiegeHQ, 3},
		{"fixed", RestartPointFixed, 4},
		{"agathion", RestartPointAgathion, 5},
		{"jail", RestartPointJail, 27},
		{"AcquireSkillType.CLASS", AcquireSkillClass, 0},
		{"AcquireSkillType.FISHING", AcquireSkillFishing, 1},
		{"AcquireSkillType.PLEDGE", AcquireSkillPledge, 2},
		{"auto soulshot off", AutoSoulShotOff, 0},
		{"auto soulshot on", AutoSoulShotOn, 1},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.got, tt.want)
		}
	}
}
