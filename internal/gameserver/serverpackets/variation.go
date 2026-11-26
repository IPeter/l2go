package serverpackets

// The augmentation windows (Plan 14, M11d): the Ex packets FE 50-57 Mobius
// sends for the refine and cancel windows.

// Sub-opcodes of the augmentation packets (Mobius ServerPackets).
const (
	ExOpShowVariationMakeWindow             = 0x50
	ExOpShowVariationCancelWindow           = 0x51
	ExOpPutItemResultForVariationMake       = 0x52
	ExOpPutIntensiveResultForVariationMake  = 0x53
	ExOpPutCommissionResultForVariationMake = 0x54
	ExOpVariationResult                     = 0x55
	ExOpPutItemResultForVariationCancel     = 0x56
	ExOpVariationCancelResult               = 0x57
)

// NewExShowVariationMakeWindowPacket builds ExShowVariationMakeWindow, which
// opens the augmentation window. It has no fields.
func NewExShowVariationMakeWindowPacket() []byte {
	return newExWriter(ExOpShowVariationMakeWindow, 0).Bytes()
}

// NewExShowVariationCancelWindowPacket builds ExShowVariationCancelWindow,
// which opens the window that removes an augmentation. It has no fields.
func NewExShowVariationCancelWindowPacket() []byte {
	return newExWriter(ExOpShowVariationCancelWindow, 0).Bytes()
}

// NewExPutItemResultForVariationMakePacket builds
// ExPutItemResultForVariationMake, the answer to an item put into the
// augmentation window: d object ID, d item ID, d 1.
func NewExPutItemResultForVariationMakePacket(objectID, itemID int32) []byte {
	w := newExWriter(ExOpPutItemResultForVariationMake, 12)
	w.WriteInt32(objectID)
	w.WriteInt32(itemID)
	w.WriteInt32(1)
	return w.Bytes()
}

// ExPutIntensiveResultForVariationMake is the answer to a life stone put
// into the augmentation window: the gemstones it asks for next.
type ExPutIntensiveResultForVariationMake struct {
	RefinerObjectID int32
	LifeStoneItemID int32
	GemstoneItemID  int32
	GemstoneCount   int32
}

// NewExPutIntensiveResultForVariationMakePacket builds it: d refiner object
// ID, d life stone item ID, d gemstone item ID, d gemstone count, d 1.
func NewExPutIntensiveResultForVariationMakePacket(p ExPutIntensiveResultForVariationMake) []byte {
	w := newExWriter(ExOpPutIntensiveResultForVariationMake, 20)
	w.WriteInt32(p.RefinerObjectID)
	w.WriteInt32(p.LifeStoneItemID)
	w.WriteInt32(p.GemstoneItemID)
	w.WriteInt32(p.GemstoneCount)
	w.WriteInt32(1)
	return w.Bytes()
}

// NewExPutCommissionResultForVariationMakePacket builds
// ExPutCommissionResultForVariationMake, the answer to the gemstones put
// into the augmentation window: d gemstone object ID, d 0, d count, d 0,
// d 1.
func NewExPutCommissionResultForVariationMakePacket(gemstoneObjectID, count int32) []byte {
	w := newExWriter(ExOpPutCommissionResultForVariationMake, 20)
	w.WriteInt32(gemstoneObjectID)
	w.WriteInt32(0)
	w.WriteInt32(count)
	w.WriteInt32(0)
	w.WriteInt32(1)
	return w.Bytes()
}

// NewExVariationResultPacket builds ExVariationResult, the outcome of
// RequestRefine: d the low option, d the high option, d 1 on success (0, 0,
// 0 on a failure).
func NewExVariationResultPacket(stat12, stat34, result int32) []byte {
	w := newExWriter(ExOpVariationResult, 12)
	w.WriteInt32(stat12)
	w.WriteInt32(stat34)
	w.WriteInt32(result)
	return w.Bytes()
}

// NewExPutItemResultForVariationCancelPacket builds
// ExPutItemResultForVariationCancel, the answer to an augmented item put
// into the cancel window, with the fee: d 0x40A97712, d object ID, d 0x27,
// d 0x2006, q price, d 1 (Mobius writes these constants as they are).
func NewExPutItemResultForVariationCancelPacket(objectID int32, price int64) []byte {
	w := newExWriter(ExOpPutItemResultForVariationCancel, 28)
	w.WriteInt32(0x40A97712)
	w.WriteInt32(objectID)
	w.WriteInt32(0x27)
	w.WriteInt32(0x2006)
	w.WriteUint64(uint64(price)) //nolint:gosec // same bits as the Java long
	w.WriteInt32(1)
	return w.Bytes()
}

// NewExVariationCancelResultPacket builds ExVariationCancelResult, the
// outcome of RequestRefineCancel: d 1 (close the window), d result (1 done,
// 0 refused).
func NewExVariationCancelResultPacket(result int32) []byte {
	w := newExWriter(ExOpVariationCancelResult, 8)
	w.WriteInt32(1)
	w.WriteInt32(result)
	return w.Bytes()
}
