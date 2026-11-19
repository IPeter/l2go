package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpRequestTargetCanceld is the opcode of RequestTargetCanceld (the
// spelling is the one Mobius uses).
const OpRequestTargetCanceld = 0x37

// RequestTargetCanceld asks to drop the current target.
type RequestTargetCanceld struct {
	// Unselect is 0 when Mobius aborts a cast in progress first, and drops
	// the target only if nothing is being cast. Any other value only drops
	// the target.
	Unselect uint16
}

// ParseRequestTargetCanceld parses a RequestTargetCanceld body (Mobius
// RequestTargetCanceld.readImpl).
func ParseRequestTargetCanceld(data []byte) (RequestTargetCanceld, error) {
	r := packets.NewReader(data)
	p := RequestTargetCanceld{Unselect: r.ReadUint16()}
	if err := r.Err(); err != nil {
		return RequestTargetCanceld{}, fmt.Errorf("RequestTargetCanceld: %w", err)
	}
	return p, nil
}
