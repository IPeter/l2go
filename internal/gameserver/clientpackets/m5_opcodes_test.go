package clientpackets

import "testing"

// The opcodes are those of Mobius ClientPackets; the limits are the Mobius
// readImpl checks with the default Player.ini.
func TestM5ClientOpcodes(t *testing.T) {
	tests := []struct {
		name      string
		got, want int
	}{
		{"RequestItemList", OpRequestItemList, 0x0f},
		{"RequestUnEquipItem", OpRequestUnEquipItem, 0x11},
		{"RequestDropItem", OpRequestDropItem, 0x12},
		{"UseItem", OpUseItem, 0x14},
		{"RequestSellItem", OpRequestSellItem, 0x1e},
		{"RequestBuyItem", OpRequestBuyItem, 0x1f},
		{"SendWareHouseDepositList", OpSendWareHouseDepositList, 0x31},
		{"SendWareHouseWithDrawList", OpSendWareHouseWithDrawList, 0x32},
		{"RequestDestroyItem", OpRequestDestroyItem, 0x59},
		{"MultiSellChoose", OpMultiSellChoose, 0xa7},
		{"MAX_ITEM_IN_PACKET: max(80, 100, 250)", MaxItemsInPacket, 250},
		{"RequestBuyItem count check", MaxBuyCount, 10000},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}
