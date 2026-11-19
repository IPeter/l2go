package clientpackets

import (
	"math"
	"testing"

	"github.com/IPeter/l2go/internal/packets"
)

// Mobius RequestAcquireSkillInfo.readImpl and RequestAcquireSkill.readImpl
// are the same: d skill ID, d level, d AcquireSkillType.values()[type]
// (12 bytes). A type outside the enum throws, so Mobius drops the packet.

// m67AcquireValid are bodies Mobius reads, with the fields they give.
var m67AcquireValid = []struct {
	name              string
	body              []byte
	skill, level, typ int32
}{
	{"class skill", new(m67Body).d(3).d(2).d(AcquireSkillClass).b, 3, 2, 0},
	{"fishing skill", new(m67Body).d(1312).d(1).d(AcquireSkillFishing).b, 1312, 1, 1},
	{"clan skill", new(m67Body).d(370).d(3).d(AcquireSkillPledge).b, 370, 3, 2},
	// The ID and level are checked later, by the handler.
	{"zero ID and level", new(m67Body).d(0).d(0).d(AcquireSkillClass).b, 0, 0, 0},
	{"negative ID, huge level", new(m67Body).d(-5).d(2000).d(AcquireSkillClass).b, -5, 2000, 0},
	{"trailing data", new(m67Body).d(1177).d(1).d(AcquireSkillClass).c(1).b, 1177, 1, 0},
}

// m67AcquireBadTypes are types Mobius fails to read.
var m67AcquireBadTypes = []int32{-1, 3, 4, 99, math.MaxInt32, math.MinInt32}

func TestParseRequestAcquireSkillInfo(t *testing.T) {
	for _, tt := range m67AcquireValid {
		got, err := ParseRequestAcquireSkillInfo(tt.body)
		m67Parsed(t, tt.name, got, err, RequestAcquireSkillInfo{SkillID: tt.skill, Level: tt.level, Type: tt.typ})
	}
	for _, typ := range m67AcquireBadTypes {
		got, err := ParseRequestAcquireSkillInfo(new(m67Body).d(1177).d(1).d(typ).b)
		m67Failed(t, "type", got, err, ErrUnknownAcquireSkillType)
	}
	m67Truncations(t, "RequestAcquireSkillInfo", new(m67Body).d(1177).d(1).d(AcquireSkillPledge).b, ParseRequestAcquireSkillInfo)
	// A short body is short whatever its type bytes are.
	got, err := ParseRequestAcquireSkillInfo(new(m67Body).d(1177).d(1).c(0xff).c(0xff).c(0xff).b)
	m67Failed(t, "short, bad type", got, err, packets.ErrShortPacket)
}

func TestParseRequestAcquireSkill(t *testing.T) {
	for _, tt := range m67AcquireValid {
		got, err := ParseRequestAcquireSkill(tt.body)
		m67Parsed(t, tt.name, got, err, RequestAcquireSkill{SkillID: tt.skill, Level: tt.level, Type: tt.typ})
	}
	for _, typ := range m67AcquireBadTypes {
		got, err := ParseRequestAcquireSkill(new(m67Body).d(1177).d(1).d(typ).b)
		m67Failed(t, "type", got, err, ErrUnknownAcquireSkillType)
	}
	m67Truncations(t, "RequestAcquireSkill", new(m67Body).d(1177).d(1).d(AcquireSkillFishing).b, ParseRequestAcquireSkill)
}

// m67AcquireSeeds adds the seeds of the acquire skill fuzz tests.
func m67AcquireSeeds(f *testing.F) {
	f.Add([]byte{})
	f.Add(new(m67Body).d(3).d(2).d(AcquireSkillClass).b)
	f.Add(new(m67Body).d(370).d(3).d(AcquireSkillPledge).c(0).b)
	f.Add(new(m67Body).d(1).d(1).d(3).b)
}

// m67AcquireRejected reports whether Mobius fails to read the type of a
// body of at least 12 bytes.
func m67AcquireRejected(data []byte) bool {
	if len(data) < 12 {
		return false
	}
	typ := m67Int(data, 8)
	return typ < 0 || typ > 2
}

func FuzzParseRequestAcquireSkillInfo(f *testing.F) {
	m67AcquireSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := ParseRequestAcquireSkillInfo(data)
		if !m67Fuzzed(t, data, got, err, 12, m67AcquireRejected(data), ErrUnknownAcquireSkillType) {
			return
		}
		want := RequestAcquireSkillInfo{SkillID: m67Int(data, 0), Level: m67Int(data, 4), Type: m67Int(data, 8)}
		if got != want {
			t.Errorf("% x: got %+v, want %+v", data, got, want)
		}
	})
}

func FuzzParseRequestAcquireSkill(f *testing.F) {
	m67AcquireSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := ParseRequestAcquireSkill(data)
		if !m67Fuzzed(t, data, got, err, 12, m67AcquireRejected(data), ErrUnknownAcquireSkillType) {
			return
		}
		want := RequestAcquireSkill{SkillID: m67Int(data, 0), Level: m67Int(data, 4), Type: m67Int(data, 8)}
		if got != want {
			t.Errorf("% x: got %+v, want %+v", data, got, want)
		}
	})
}
