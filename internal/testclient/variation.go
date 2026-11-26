package testclient

import (
	"context"

	"github.com/IPeter/l2go/internal/packets"
)

// The augmentation windows (Plan 14, M11d): the Ex server packets FE 50-57
// and the Ex client packets D0 29-2E, after Mobius.

// Ex server sub-opcodes of augmentation.
const (
	ExOpShowVariationMakeWindow             uint16 = 0x50
	ExOpShowVariationCancelWindow           uint16 = 0x51
	ExOpPutItemResultForVariationMake       uint16 = 0x52
	ExOpPutIntensiveResultForVariationMake  uint16 = 0x53
	ExOpPutCommissionResultForVariationMake uint16 = 0x54
	ExOpVariationResult                     uint16 = 0x55
	ExOpPutItemResultForVariationCancel     uint16 = 0x56
	ExOpVariationCancelResult               uint16 = 0x57
)

// Ex client sub-opcodes of augmentation.
const (
	ExOpRequestConfirmTargetItem  uint16 = 0x29
	ExOpRequestConfirmRefinerItem uint16 = 0x2a
	ExOpRequestConfirmGemStone    uint16 = 0x2b
	ExOpRequestRefine             uint16 = 0x2c
	ExOpRequestConfirmCancelItem  uint16 = 0x2d
	ExOpRequestRefineCancel       uint16 = 0x2e
)

func init() {
	for sub, p := range map[uint16]serverPacket{
		ExOpShowVariationMakeWindow:             {"ExShowVariationMakeWindow", readExShowVariationMakeWindow},
		ExOpShowVariationCancelWindow:           {"ExShowVariationCancelWindow", readExShowVariationCancelWindow},
		ExOpPutItemResultForVariationMake:       {"ExPutItemResultForVariationMake", readExPutItemResultForVariationMake},
		ExOpPutIntensiveResultForVariationMake:  {"ExPutIntensiveResultForVariationMake", readExPutIntensiveResultForVariationMake},
		ExOpPutCommissionResultForVariationMake: {"ExPutCommissionResultForVariationMake", readExPutCommissionResultForVariationMake},
		ExOpVariationResult:                     {"ExVariationResult", readExVariationResult},
		ExOpPutItemResultForVariationCancel:     {"ExPutItemResultForVariationCancel", readExPutItemResultForVariationCancel},
		ExOpVariationCancelResult:               {"ExVariationCancelResult", readExVariationCancelResult},
	} {
		exServerPackets[sub] = p
	}
	for sub, name := range map[uint16]string{
		ExOpRequestConfirmTargetItem:  "RequestConfirmTargetItem",
		ExOpRequestConfirmRefinerItem: "RequestConfirmRefinerItem",
		ExOpRequestConfirmGemStone:    "RequestConfirmGemStone",
		ExOpRequestRefine:             "RequestRefine",
		ExOpRequestConfirmCancelItem:  "RequestConfirmCancelItem",
		ExOpRequestRefineCancel:       "RequestRefineCancel",
	} {
		exClientPacketNames[sub] = name
	}
}

// ExShowVariationMakeWindow opens the augmentation window.
type ExShowVariationMakeWindow struct{}

// PacketName implements Packet.
func (ExShowVariationMakeWindow) PacketName() string { return "ExShowVariationMakeWindow" }

func readExShowVariationMakeWindow(*fields) Packet { return ExShowVariationMakeWindow{} }

// ExShowVariationCancelWindow opens the window that removes augmentations.
type ExShowVariationCancelWindow struct{}

// PacketName implements Packet.
func (ExShowVariationCancelWindow) PacketName() string { return "ExShowVariationCancelWindow" }

func readExShowVariationCancelWindow(*fields) Packet { return ExShowVariationCancelWindow{} }

// ExPutItemResultForVariationMake accepts the item to augment: d object
// ID, d item ID, d 1.
type ExPutItemResultForVariationMake struct {
	ObjectID int32
	ItemID   int32
	Result   int32
}

// PacketName implements Packet.
func (ExPutItemResultForVariationMake) PacketName() string { return "ExPutItemResultForVariationMake" }

func readExPutItemResultForVariationMake(f *fields) Packet {
	return ExPutItemResultForVariationMake{ObjectID: f.i32(), ItemID: f.i32(), Result: f.i32()}
}

// ExPutIntensiveResultForVariationMake accepts the life stone and names
// the gemstones: d refiner object ID, d life stone ID, d gemstone ID, d
// gemstone count, d 1.
type ExPutIntensiveResultForVariationMake struct {
	RefinerObjectID int32
	LifeStoneItemID int32
	GemstoneItemID  int32
	GemstoneCount   int32
	Result          int32
}

// PacketName implements Packet.
func (ExPutIntensiveResultForVariationMake) PacketName() string {
	return "ExPutIntensiveResultForVariationMake"
}

func readExPutIntensiveResultForVariationMake(f *fields) Packet {
	return ExPutIntensiveResultForVariationMake{
		RefinerObjectID: f.i32(), LifeStoneItemID: f.i32(), GemstoneItemID: f.i32(), GemstoneCount: f.i32(), Result: f.i32(),
	}
}

// ExPutCommissionResultForVariationMake accepts the gemstones: d object ID,
// d 0, d count, d 0, d 1.
type ExPutCommissionResultForVariationMake struct {
	GemstoneObjectID int32
	Unknown1         int32
	Count            int32
	Unknown2         int32
	Result           int32
}

// PacketName implements Packet.
func (ExPutCommissionResultForVariationMake) PacketName() string {
	return "ExPutCommissionResultForVariationMake"
}

func readExPutCommissionResultForVariationMake(f *fields) Packet {
	return ExPutCommissionResultForVariationMake{GemstoneObjectID: f.i32(), Unknown1: f.i32(), Count: f.i32(), Unknown2: f.i32(), Result: f.i32()}
}

// ExVariationResult is the outcome of RequestRefine: the two options and 1,
// or 0, 0, 0 for a failure.
type ExVariationResult struct {
	Stat12 int32
	Stat34 int32
	Result int32
}

// PacketName implements Packet.
func (ExVariationResult) PacketName() string { return "ExVariationResult" }

func readExVariationResult(f *fields) Packet {
	return ExVariationResult{Stat12: f.i32(), Stat34: f.i32(), Result: f.i32()}
}

// ExPutItemResultForVariationCancel accepts the item whose augmentation is
// to go, with the fee: d 0x40A97712, d object ID, d 0x27, d 0x2006, q
// price, d 1.
type ExPutItemResultForVariationCancel struct {
	Unknown1 int32 // 0x40A97712
	ObjectID int32
	Unknown2 int32 // 0x27
	Unknown3 int32 // 0x2006
	Price    int64
	Result   int32
}

// PacketName implements Packet.
func (ExPutItemResultForVariationCancel) PacketName() string {
	return "ExPutItemResultForVariationCancel"
}

func readExPutItemResultForVariationCancel(f *fields) Packet {
	return ExPutItemResultForVariationCancel{Unknown1: f.i32(), ObjectID: f.i32(), Unknown2: f.i32(), Unknown3: f.i32(), Price: f.i64(), Result: f.i32()}
}

// ExVariationCancelResult is the outcome of RequestRefineCancel: d 1
// (close the window), d result (1 removed, 0 refused).
type ExVariationCancelResult struct {
	CloseWindow int32
	Result      int32
}

// PacketName implements Packet.
func (ExVariationCancelResult) PacketName() string { return "ExVariationCancelResult" }

func readExVariationCancelResult(f *fields) Packet {
	return ExVariationCancelResult{CloseWindow: f.i32(), Result: f.i32()}
}

// encodeExInts builds an Ex client packet of d fields.
func encodeExInts(sub uint16, vs ...int32) []byte {
	w := packets.NewWriter(3 + 4*len(vs))
	w.WriteUint8(OpExClient)
	w.WriteUint16(sub)
	for _, v := range vs {
		w.WriteInt32(v)
	}
	return w.Bytes()
}

// SendConfirmTargetItem puts an item into the augmentation window
// (RequestConfirmTargetItem: d object ID), without waiting.
func (c *GameConn) SendConfirmTargetItem(ctx context.Context, objectID int32) error {
	return c.send(ctx, encodeExInts(ExOpRequestConfirmTargetItem, objectID))
}

// SendConfirmRefinerItem puts a life stone into the augmentation window
// (RequestConfirmRefinerItem: d target, d refiner), without waiting.
func (c *GameConn) SendConfirmRefinerItem(ctx context.Context, targetID, refinerID int32) error {
	return c.send(ctx, encodeExInts(ExOpRequestConfirmRefinerItem, targetID, refinerID))
}

// SendConfirmGemStone puts the gemstones into the augmentation window
// (RequestConfirmGemStone: d target, d refiner, d gemstone, d count),
// without waiting.
func (c *GameConn) SendConfirmGemStone(ctx context.Context, targetID, refinerID, gemID, count int32) error {
	return c.send(ctx, encodeExInts(ExOpRequestConfirmGemStone, targetID, refinerID, gemID, count))
}

// Refine augments an item (RequestRefine: d target, d refiner, d gemstone,
// d count) and waits for the ExVariationResult; the item updates come
// before it.
func (c *GameConn) Refine(ctx context.Context, targetID, refinerID, gemID, count int32) (ExVariationResult, error) {
	if err := c.send(ctx, encodeExInts(ExOpRequestRefine, targetID, refinerID, gemID, count)); err != nil {
		return ExVariationResult{}, err
	}
	return Await[ExVariationResult](ctx, c, nil)
}

// SendConfirmCancelItem puts an augmented item into the cancel window
// (RequestConfirmCancelItem: d object ID), without waiting.
func (c *GameConn) SendConfirmCancelItem(ctx context.Context, objectID int32) error {
	return c.send(ctx, encodeExInts(ExOpRequestConfirmCancelItem, objectID))
}

// RefineCancel removes an item's augmentation (RequestRefineCancel: d
// object ID) and waits for the ExVariationCancelResult; the fee's
// messages come before it.
func (c *GameConn) RefineCancel(ctx context.Context, objectID int32) (ExVariationCancelResult, error) {
	if err := c.send(ctx, encodeExInts(ExOpRequestRefineCancel, objectID)); err != nil {
		return ExVariationCancelResult{}, err
	}
	return Await[ExVariationCancelResult](ctx, c, nil)
}
