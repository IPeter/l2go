package clientpackets

import "testing"

// Mobius RequestMagicSkillUse.readImpl: d skill ID, d ctrl (!= 0),
// c shift (!= 0) (9 bytes).
func TestParseRequestMagicSkillUse(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want RequestMagicSkillUse
	}{
		{"click", new(m67Body).d(1177).d(0).c(0).b, RequestMagicSkillUse{SkillID: 1177}},
		{"ctrl", new(m67Body).d(1177).d(1).c(0).b, RequestMagicSkillUse{SkillID: 1177, CtrlPressed: true}},
		{"shift", new(m67Body).d(56).d(0).c(1).b, RequestMagicSkillUse{SkillID: 56, ShiftPressed: true}},
		// Unlike RequestActionUse, Mobius compares with 0: any other value
		// is pressed.
		{"ctrl 2, shift 255", new(m67Body).d(3).d(2).c(0xff).b, RequestMagicSkillUse{SkillID: 3, CtrlPressed: true, ShiftPressed: true}},
		{"ctrl -1, shift 0x80", new(m67Body).d(3).d(-1).c(0x80).b, RequestMagicSkillUse{SkillID: 3, CtrlPressed: true, ShiftPressed: true}},
		{"ctrl 0x100", new(m67Body).d(3).d(0x100).c(0).b, RequestMagicSkillUse{SkillID: 3, CtrlPressed: true}},
		{"trailing data", new(m67Body).d(1204).d(0).c(0).c(9).b, RequestMagicSkillUse{SkillID: 1204}},
		{"garbage", []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, RequestMagicSkillUse{SkillID: -1, CtrlPressed: true, ShiftPressed: true}},
	}
	for _, tt := range tests {
		got, err := ParseRequestMagicSkillUse(tt.body)
		m67Parsed(t, tt.name, got, err, tt.want)
	}
	m67Truncations(t, "RequestMagicSkillUse", new(m67Body).d(1177).d(1).c(1).b, ParseRequestMagicSkillUse)
}

func FuzzParseRequestMagicSkillUse(f *testing.F) {
	f.Add([]byte{})
	f.Add(new(m67Body).d(1177).d(0).c(0).b)
	f.Add(new(m67Body).d(56).d(1).c(1).c(0).b)
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := ParseRequestMagicSkillUse(data)
		if !m67Fuzzed(t, data, got, err, 9, false, nil) {
			return
		}
		want := RequestMagicSkillUse{
			SkillID:      m67Int(data, 0),
			CtrlPressed:  m67Int(data, 4) != 0,
			ShiftPressed: data[8] != 0,
		}
		if got != want {
			t.Errorf("% x: got %+v, want %+v", data, got, want)
		}
	})
}
