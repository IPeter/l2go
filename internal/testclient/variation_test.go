package testclient

import (
	"bytes"
	"errors"
	"testing"
)

// The augmentation packets, transcribed from the Mobius writeImpl.
func TestDecodeVariationPackets(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want Packet
	}{
		{"ExShowVariationMakeWindow", msg(0xfe).u16(0x50).b, ExShowVariationMakeWindow{}},
		{"ExShowVariationCancelWindow", msg(0xfe).u16(0x51).b, ExShowVariationCancelWindow{}},
		{"ExPutItemResultForVariationMake", msg(0xfe).u16(0x52).i32(0x10000123).i32(2499).i32(1).b,
			ExPutItemResultForVariationMake{ObjectID: 0x10000123, ItemID: 2499, Result: 1}},
		{"ExPutIntensiveResultForVariationMake", msg(0xfe).u16(0x53).i32(0x10000124).i32(8723).i32(2130).i32(20).i32(1).b,
			ExPutIntensiveResultForVariationMake{RefinerObjectID: 0x10000124, LifeStoneItemID: 8723, GemstoneItemID: 2130, GemstoneCount: 20, Result: 1}},
		{"ExPutCommissionResultForVariationMake", msg(0xfe).u16(0x54).i32(0x10000125).i32(0).i32(20).i32(0).i32(1).b,
			ExPutCommissionResultForVariationMake{GemstoneObjectID: 0x10000125, Count: 20, Result: 1}},
		{"ExVariationResult", msg(0xfe).u16(0x55).i32(91).i32(14578).i32(1).b, ExVariationResult{Stat12: 91, Stat34: 14578, Result: 1}},
		{"ExPutItemResultForVariationCancel", msg(0xfe).u16(0x56).i32(0x40A97712).i32(0x10000123).i32(0x27).i32(0x2006).i64(95000).i32(1).b,
			ExPutItemResultForVariationCancel{Unknown1: 0x40A97712, ObjectID: 0x10000123, Unknown2: 0x27, Unknown3: 0x2006, Price: 95000, Result: 1}},
		{"ExVariationCancelResult", msg(0xfe).u16(0x57).i32(1).i32(0).b, ExVariationCancelResult{CloseWindow: 1}},
	}
	for _, tt := range tests {
		checkDecode(t, tt.name, tt.body, tt.want)
		if got := serverPacketName(tt.body); got != tt.want.PacketName() {
			t.Errorf("%s: serverPacketName = %q", tt.name, got)
		}
	}
	// Extra bytes are a layout error.
	if _, err := DecodeServerPacket(msg(0xfe).u16(0x57).i32(1).i32(0).u8(0).b); !errors.Is(err, ErrMalformed) {
		t.Errorf("ExVariationCancelResult with an extra byte: %v, want ErrMalformed", err)
	}
}

// The requests follow the Mobius readImpl of each.
func TestEncodeVariationRequests(t *testing.T) {
	tests := []struct {
		name string
		got  []byte
		want []byte
	}{
		{"RequestConfirmTargetItem", encodeExInts(ExOpRequestConfirmTargetItem, 7), msg(0xd0).u16(0x29).i32(7).b},
		{"RequestConfirmRefinerItem", encodeExInts(ExOpRequestConfirmRefinerItem, 7, 8), msg(0xd0).u16(0x2a).i32(7).i32(8).b},
		{"RequestConfirmGemStone", encodeExInts(ExOpRequestConfirmGemStone, 7, 8, 9, 20), msg(0xd0).u16(0x2b).i32(7).i32(8).i32(9).i32(20).b},
		{"RequestRefine", encodeExInts(ExOpRequestRefine, 7, 8, 9, 20), msg(0xd0).u16(0x2c).i32(7).i32(8).i32(9).i32(20).b},
		{"RequestConfirmCancelItem", encodeExInts(ExOpRequestConfirmCancelItem, 7), msg(0xd0).u16(0x2d).i32(7).b},
		{"RequestRefineCancel", encodeExInts(ExOpRequestRefineCancel, 7), msg(0xd0).u16(0x2e).i32(7).b},
	}
	for _, tt := range tests {
		if !bytes.Equal(tt.got, tt.want) {
			t.Errorf("%s: % x, want % x", tt.name, tt.got, tt.want)
		}
		if got := clientPacketName(tt.got); got != tt.name {
			t.Errorf("clientPacketName = %q, want %q", got, tt.name)
		}
	}
}
