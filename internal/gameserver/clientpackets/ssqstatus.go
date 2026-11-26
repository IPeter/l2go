package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpRequestSSQStatus is the opcode of RequestSSQStatus.
const OpRequestSSQStatus = 0xc7

// RequestSSQStatus asks for a page of the Record of the Seven Signs (the
// server answers with SSQStatus).
type RequestSSQStatus struct {
	// Page is the page, 1-4; not checked here.
	Page int
}

// ParseRequestSSQStatus parses a RequestSSQStatus body (Mobius
// RequestSSQStatus.readImpl): one byte.
func ParseRequestSSQStatus(data []byte) (RequestSSQStatus, error) {
	r := packets.NewReader(data)
	req := RequestSSQStatus{Page: int(r.ReadUint8())}
	if err := r.Err(); err != nil {
		return RequestSSQStatus{}, fmt.Errorf("RequestSSQStatus: %w", err)
	}
	return req, nil
}
