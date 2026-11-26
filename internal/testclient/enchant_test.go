package testclient

import (
	"bytes"
	"testing"
)

// The enchant packets as Mobius writes and reads them.

// mobiusRequestEnchantItem: d object (the support item is commented out).
func mobiusRequestEnchantItem(objectID int32) []byte { return msg(0x58).i32(objectID).b }

// mobiusRequestCrystallizeItem: d object, d count.
func mobiusRequestCrystallizeItem(objectID, count int32) []byte {
	return msg(0x72).i32(objectID).i32(count).b
}

// mobiusChooseInventoryItem: c 0x6F, d item ID.
func mobiusChooseInventoryItem(itemID int32) []byte { return msg(0x6f).i32(itemID).b }

func TestDecodeEnchantPackets(t *testing.T) {
	checkDecode(t, "ChooseInventoryItem", mobiusChooseInventoryItem(955), ChooseInventoryItem{ItemID: 955})
	if _, err := DecodeServerPacket(msg(0x6f).b); err == nil {
		t.Error("a ChooseInventoryItem without its item decoded")
	}
	if got := serverPacketName(mobiusChooseInventoryItem(1)); got != "ChooseInventoryItem" {
		t.Errorf("serverPacketName = %q", got)
	}
}

func TestEncodeEnchantRequests(t *testing.T) {
	if got, want := encodeInt(OpRequestEnchantItem, 0x10000007), mobiusRequestEnchantItem(0x10000007); !bytes.Equal(got, want) {
		t.Errorf("RequestEnchantItem % x, want % x", got, want)
	}
	if got, want := encodeInts(OpRequestCrystallizeItem, 9, 1), mobiusRequestCrystallizeItem(9, 1); !bytes.Equal(got, want) {
		t.Errorf("RequestCrystallizeItem % x, want % x", got, want)
	}
	for raw, want := range map[string]string{
		string(mobiusRequestEnchantItem(1)):        "RequestEnchantItem",
		string(mobiusRequestCrystallizeItem(1, 1)): "RequestCrystallizeItem",
	} {
		if got := clientPacketName([]byte(raw)); got != want {
			t.Errorf("clientPacketName = %q, want %q", got, want)
		}
	}
}

// TestEnchantFlow uses a scroll, enchants a sword and crystallizes an armor
// against a fake server, with Mobius's order of answers.
func TestEnchantFlow(t *testing.T) {
	pos := Position{-71338, 258271, -3104}
	scroll := Item{Type1: 4, ObjectID: 0x10000020, ItemID: 955, Count: 1, Type2: 5, Mana: -1}
	sword := Item{Type1: 0, ObjectID: 0x10000021, ItemID: 123, Count: 1, Type2: 0, BodyPart: 0x0080, Mana: -1}
	armor := Item{Type1: 1, ObjectID: 0x10000022, ItemID: 60, Count: 1, Type2: 1, BodyPart: 0x8000, Mana: -1}
	inv := append(testItems(), scroll, sword, armor)
	sword1 := sword
	sword1.Enchant = 1

	addr := startFakeGame(t, func(f *fakeGame) {
		f.enterQuick(pos, inv)

		// The scroll opens the enchant window.
		f.expect(mobiusUseItem(scroll.ObjectID, false))
		f.send(mobiusChooseInventoryItem(955))
		f.send(msg(0x64).i32(303).i32(0).b)

		// The sword: the message, the result, then the inventory.
		f.expect(mobiusRequestEnchantItem(sword.ObjectID))
		f.send(mobiusSMItem(62, 123))
		f.send(msg(0x81).i32(1).b)
		f.send(mobiusInventoryUpdate(InventoryUpdate{Items: []ItemChange{
			{Change: ItemRemoved, Item: scroll}, {Change: ItemModified, Item: sword1},
		}}))

		// The armor is crystallized.
		f.expect(mobiusRequestCrystallizeItem(armor.ObjectID, 1))
		f.send(mobiusInventoryUpdate(InventoryUpdate{Items: []ItemChange{{Change: ItemRemoved, Item: armor}}}))
		f.drain()
	})

	ctx := testContext(t)
	c := clientEnterQuick(t, ctx, addr)
	w := c.World()

	ch, err := c.UseEnchantScroll(ctx, scroll.ObjectID)
	if err != nil || ch.ItemID != 955 {
		t.Fatalf("UseEnchantScroll = %+v, %v", ch, err)
	}
	r, err := c.EnchantItem(ctx, sword.ObjectID)
	if err != nil || r.Result != 1 {
		t.Fatalf("EnchantItem = %+v, %v", r, err)
	}
	if ic, err := c.AwaitItemChange(ctx, sword.ObjectID); err != nil || ic.Enchant != 1 {
		t.Fatalf("the sword after the enchant = %+v, %v", ic, err)
	}
	if _, ok := w.Item(scroll.ObjectID); ok {
		t.Error("the scroll is still in the inventory")
	}
	ic, err := c.CrystallizeItem(ctx, armor.ObjectID, 1)
	if err != nil || ic.Change != ItemRemoved {
		t.Fatalf("CrystallizeItem = %+v, %v", ic, err)
	}
	if _, ok := w.Item(armor.ObjectID); ok {
		t.Error("the armor is still in the inventory")
	}
}
