package serverpackets

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/storage"
)

// CharSelectionInfo ends each character with the augmentation of its
// right-hand item (Mobius getPaperdollAugmentationId).
func TestCharSelectionInfoAugmentation(t *testing.T) {
	c := storage.Character{ObjectID: 1, Name: "Augmented", Items: []storage.Item{
		{ObjectID: 2, ItemID: 2499, Count: 1, Slot: storage.PaperdollRHand, Enchant: 3, Augmentation: 14578<<16 | 91},
		{ObjectID: 3, ItemID: 868, Count: 1, Slot: storage.PaperdollRFinger, Augmentation: 5},
	}}
	b := NewCharSelectionInfoPacket("acc", 1, []storage.Character{c}, time.Unix(0, 0))
	if got := int32(binary.LittleEndian.Uint32(b[len(b)-4:])); got != 14578<<16|91 {
		t.Errorf("augmentation %d, want the weapon's %d", got, 14578<<16|91)
	}
	if got := b[len(b)-5]; got != 3 {
		t.Errorf("enchant %d, want 3", got)
	}
}

// The augmentation packets against the Mobius writeImpl of each (c 0xFE, h
// sub-opcode, then the fields).
func TestVariationPackets(t *testing.T) {
	var g m3Golden

	g = m3Golden{}
	g.c(0xfe).h(0x50)
	m3Check(t, "ExShowVariationMakeWindow", NewExShowVariationMakeWindowPacket(), g.b)

	g = m3Golden{}
	g.c(0xfe).h(0x51)
	m3Check(t, "ExShowVariationCancelWindow", NewExShowVariationCancelWindowPacket(), g.b)

	// d item object ID, d item ID, d 1.
	g = m3Golden{}
	g.c(0xfe).h(0x52).d(0x10000123).d(2499).d(1)
	m3Check(t, "ExPutItemResultForVariationMake", NewExPutItemResultForVariationMakePacket(0x10000123, 2499), g.b)

	// d refiner object ID, d life stone ID, d gemstone ID, d gemstone count,
	// d 1.
	g = m3Golden{}
	g.c(0xfe).h(0x53).d(0x10000124).d(8723).d(2130).d(20).d(1)
	m3Check(t, "ExPutIntensiveResultForVariationMake", NewExPutIntensiveResultForVariationMakePacket(ExPutIntensiveResultForVariationMake{
		RefinerObjectID: 0x10000124, LifeStoneItemID: 8723, GemstoneItemID: 2130, GemstoneCount: 20,
	}), g.b)

	// d gemstone object ID, d 0, d count, d 0, d 1.
	g = m3Golden{}
	g.c(0xfe).h(0x54).d(0x10000125).d(0).d(20).d(0).d(1)
	m3Check(t, "ExPutCommissionResultForVariationMake", NewExPutCommissionResultForVariationMakePacket(0x10000125, 20), g.b)

	// d stat12, d stat34, d result.
	g = m3Golden{}
	g.c(0xfe).h(0x55).d(91).d(14578).d(1)
	m3Check(t, "ExVariationResult", NewExVariationResultPacket(91, 14578, 1), g.b)
	g = m3Golden{}
	g.c(0xfe).h(0x55).d(0).d(0).d(0)
	m3Check(t, "ExVariationResult failed", NewExVariationResultPacket(0, 0, 0), g.b)

	// d 0x40A97712, d object ID, d 0x27, d 0x2006, q price, d 1.
	g = m3Golden{}
	g.c(0xfe).h(0x56).d(0x40A97712).d(0x10000123).d(0x27).d(0x2006).q(95000).d(1)
	m3Check(t, "ExPutItemResultForVariationCancel", NewExPutItemResultForVariationCancelPacket(0x10000123, 95000), g.b)

	// d 1 (close the window), d result.
	g = m3Golden{}
	g.c(0xfe).h(0x57).d(1).d(1)
	m3Check(t, "ExVariationCancelResult", NewExVariationCancelResultPacket(1), g.b)
	g = m3Golden{}
	g.c(0xfe).h(0x57).d(1).d(0)
	m3Check(t, "ExVariationCancelResult refused", NewExVariationCancelResultPacket(0), g.b)
}
