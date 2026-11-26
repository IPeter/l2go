package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// Sub-opcodes of the Olympiad and hero requests (Plan 14, M11h; Mobius
// ExClientPackets REQUEST_WRITE_HERO_WORDS, REQUEST_OLYMPIAD_OBSERVER_END
// and REQUEST_OLYMPIAD_MATCH_LIST). The last two read nothing.
const (
	ExOpRequestWriteHeroWords      = 0x0c
	ExOpRequestOlympiadObserverEnd = 0x12
	ExOpRequestOlympiadMatchList   = 0x13
)

// RequestWriteHeroWords sets a hero's words, shown on its diary page.
type RequestWriteHeroWords struct {
	Words string
}

// ParseRequestWriteHeroWords parses the body of RequestWriteHeroWords, the
// data after the sub-opcode (Mobius readImpl): S words.
func ParseRequestWriteHeroWords(data []byte) (RequestWriteHeroWords, error) {
	r := packets.NewReader(data)
	req := RequestWriteHeroWords{Words: r.ReadString()}
	if err := r.Err(); err != nil {
		return RequestWriteHeroWords{}, fmt.Errorf("RequestWriteHeroWords: %w", err)
	}
	return req, nil
}
