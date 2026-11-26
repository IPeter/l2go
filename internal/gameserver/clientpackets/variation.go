package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// The augmentation windows (Plan 14, M11d): the Ex packets D0 29-2E the
// refine and cancel windows send (Mobius ExClientPackets).

// Sub-opcodes of the augmentation packets.
const (
	ExOpRequestConfirmTargetItem  = 0x29
	ExOpRequestConfirmRefinerItem = 0x2a
	ExOpRequestConfirmGemStone    = 0x2b
	ExOpRequestRefine             = 0x2c
	ExOpRequestConfirmCancelItem  = 0x2d
	ExOpRequestRefineCancel       = 0x2e
)

// RequestConfirmRefinerItem puts a life stone into the augmentation window.
type RequestConfirmRefinerItem struct {
	TargetObjectID  int32
	RefinerObjectID int32
}

// RequestRefine names the item, the life stone and the gemstones of an
// augmentation: RequestConfirmGemStone puts the gemstones into the window,
// RequestRefine augments.
type RequestRefine struct {
	TargetObjectID   int32
	RefinerObjectID  int32
	GemstoneObjectID int32
	GemstoneCount    int32
}

// ParseRequestObjectID parses the body of RequestConfirmTargetItem,
// RequestConfirmCancelItem and RequestRefineCancel (Mobius readImpl: d
// object ID), the data after the sub-opcode.
func ParseRequestObjectID(data []byte) (int32, error) {
	r := packets.NewReader(data)
	id := r.ReadInt32()
	if err := r.Err(); err != nil {
		return 0, fmt.Errorf("augmentation request: %w", err)
	}
	return id, nil
}

// ParseRequestConfirmRefinerItem parses a RequestConfirmRefinerItem body:
// d target object ID, d refiner object ID.
func ParseRequestConfirmRefinerItem(data []byte) (RequestConfirmRefinerItem, error) {
	r := packets.NewReader(data)
	req := RequestConfirmRefinerItem{TargetObjectID: r.ReadInt32(), RefinerObjectID: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return RequestConfirmRefinerItem{}, fmt.Errorf("RequestConfirmRefinerItem: %w", err)
	}
	return req, nil
}

// ParseRequestRefine parses a RequestConfirmGemStone or RequestRefine body:
// d target object ID, d refiner object ID, d gemstone object ID, d gemstone
// count.
func ParseRequestRefine(data []byte) (RequestRefine, error) {
	r := packets.NewReader(data)
	req := RequestRefine{
		TargetObjectID:   r.ReadInt32(),
		RefinerObjectID:  r.ReadInt32(),
		GemstoneObjectID: r.ReadInt32(),
		GemstoneCount:    r.ReadInt32(),
	}
	if err := r.Err(); err != nil {
		return RequestRefine{}, fmt.Errorf("RequestRefine: %w", err)
	}
	return req, nil
}
