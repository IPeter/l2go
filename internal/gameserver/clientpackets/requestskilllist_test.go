package clientpackets

import (
	"bytes"
	"testing"
)

// Mobius RequestSkillList.readImpl reads nothing, so any body parses,
// including garbage.
func TestParseRequestSkillList(t *testing.T) {
	for _, body := range [][]byte{nil, {}, {0x00}, {0xff, 0xff, 0xff}, bytes.Repeat([]byte{0x5a}, 64)} {
		got, err := ParseRequestSkillList(body)
		m67Parsed(t, "RequestSkillList", got, err, RequestSkillList{})
	}
}

func FuzzParseRequestSkillList(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0xff, 0x00})
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := ParseRequestSkillList(data)
		m67Fuzzed(t, data, got, err, 0, false, nil)
	})
}
