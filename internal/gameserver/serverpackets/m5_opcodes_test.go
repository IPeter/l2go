package serverpackets

import "testing"

// The opcodes are those of Mobius ServerPackets, and the constants those of
// the Mobius classes named.
func TestM5ServerOpcodes(t *testing.T) {
	tests := []struct {
		name      string
		got, want int
	}{
		{"SpawnItem", OpSpawnItem, 0x0b},
		{"DropItem", OpDropItem, 0x0c},
		{"GetItem", OpGetItem, 0x0d},
		{"SellList", OpSellList, 0x10},
		{"BuyList", OpBuyList, 0x11},
		{"WareHouseDepositList", OpWareHouseDepositList, 0x41},
		{"WareHouseWithdrawalList", OpWareHouseWithdrawalList, 0x42},
		{"EquipUpdate", OpEquipUpdate, 0x4b},
		{"EnchantResult", OpEnchantResult, 0x81},
		{"MultiSellList", OpMultiSellList, 0xd0},
		{"MultisellData.PAGE_SIZE", MultiSellPageSize, 40},
		{"WareHouseDepositList.PRIVATE", int(WarehousePrivate), 1},
		{"WareHouseDepositList.CLAN", int(WarehouseClan), 2},
		{"WareHouseDepositList.CASTLE", int(WarehouseCastle), 3},
		{"WareHouseDepositList.FREIGHT", int(WarehouseFreight), 4},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}
