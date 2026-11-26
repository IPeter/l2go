package clientpackets

import "testing"

func TestParseRequestWriteHeroWords(t *testing.T) {
	if ExOpRequestWriteHeroWords != 0x0c || ExOpRequestOlympiadObserverEnd != 0x12 || ExOpRequestOlympiadMatchList != 0x13 {
		t.Error("wrong sub-opcodes")
	}
	got, err := ParseRequestWriteHeroWords([]byte{'H', 0, 'i', 0, 0, 0})
	if err != nil || got.Words != "Hi" {
		t.Errorf("ParseRequestWriteHeroWords = %+v, %v", got, err)
	}
	if _, err := ParseRequestWriteHeroWords([]byte{'H'}); err == nil {
		t.Error("a cut string parsed")
	}
	ex, err := ParseExPacket([]byte{0x12, 0x00})
	if err != nil || ex.SubOpcode != ExOpRequestOlympiadObserverEnd || len(ex.Body) != 0 {
		t.Errorf("ParseExPacket = %+v, %v", ex, err)
	}
}

func FuzzParseRequestWriteHeroWords(f *testing.F) {
	m3Fuzz(f, ParseRequestWriteHeroWords, -1, []byte{'H', 0, 0, 0})
}
