package clientpackets

import "testing"

// Mobius RequestItemList.readImpl reads nothing, so any body parses,
// including garbage.
func TestParseRequestItemList(t *testing.T) {
	for _, body := range m3NoFieldBodies {
		if got, err := ParseRequestItemList(body); err != nil || got != (RequestItemList{}) {
			t.Errorf("ParseRequestItemList(% x) = %+v, %v", body, got, err)
		}
	}
}

func FuzzParseRequestItemList(f *testing.F) {
	m3Fuzz(f, ParseRequestItemList, 0, m3NoFieldBodies...)
}
