package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpDlgAnswer is the opcode of DlgAnswer (Mobius DLG_ANSWER).
const OpDlgAnswer = 0xc5

// DlgAnswer is the player's answer to a ConfirmDlg.
type DlgAnswer struct {
	// MessageID is the system message the question showed.
	MessageID int32
	// Answer is 1 for yes, 0 for no.
	Answer int32
	// RequesterID is the ConfirmDlg's requester ID, sent back.
	RequesterID int32
}

// ParseDlgAnswer parses a DlgAnswer body (Mobius DlgAnswer.readImpl):
// d message ID, d answer, d requester ID.
func ParseDlgAnswer(data []byte) (DlgAnswer, error) {
	r := packets.NewReader(data)
	req := DlgAnswer{
		MessageID:   r.ReadInt32(),
		Answer:      r.ReadInt32(),
		RequesterID: r.ReadInt32(),
	}
	if err := r.Err(); err != nil {
		return DlgAnswer{}, fmt.Errorf("DlgAnswer: %w", err)
	}
	return req, nil
}
