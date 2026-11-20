package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpRequestQuestAbort is the opcode of RequestQuestAbort.
const OpRequestQuestAbort = 0x64

// RequestQuestAbort is the abort button of the quest log. Mobius ignores an
// unknown quest, a quest the player has no state of, and a completed one;
// otherwise it exits the quest (its state and quest items go) and sends
// QuestList.
type RequestQuestAbort struct {
	QuestID int32
}

// ParseRequestQuestAbort parses a RequestQuestAbort body (Mobius
// RequestQuestAbort.readImpl): d quest ID.
func ParseRequestQuestAbort(data []byte) (RequestQuestAbort, error) {
	r := packets.NewReader(data)
	req := RequestQuestAbort{QuestID: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return RequestQuestAbort{}, fmt.Errorf("RequestQuestAbort: %w", err)
	}
	return req, nil
}
