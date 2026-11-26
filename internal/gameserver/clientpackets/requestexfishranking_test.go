package clientpackets

import "testing"

// RequestExFishRanking (D0 1F) reads nothing, so any body parses.
func TestParseRequestExFishRanking(t *testing.T) {
	if ExOpRequestExFishRanking != 0x1f {
		t.Errorf("sub-opcode = %#x, want 0x1f", ExOpRequestExFishRanking)
	}
	for _, body := range m3NoFieldBodies {
		if got, err := ParseRequestExFishRanking(body); err != nil || got != (RequestExFishRanking{}) {
			t.Errorf("ParseRequestExFishRanking(% x) = %+v, %v", body, got, err)
		}
	}
	ex, err := ParseExPacket([]byte{0x1f, 0x00})
	if err != nil || ex.SubOpcode != ExOpRequestExFishRanking || len(ex.Body) != 0 {
		t.Errorf("ParseExPacket = %+v, %v", ex, err)
	}
}

func FuzzParseRequestExFishRanking(f *testing.F) {
	m3Fuzz(f, ParseRequestExFishRanking, 0, m3NoFieldBodies...)
}
