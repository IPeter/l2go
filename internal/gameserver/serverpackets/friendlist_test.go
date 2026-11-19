package serverpackets

import "testing"

func TestFriendListPacket(t *testing.T) {
	// Mobius FriendList.writeImpl: c 0xFA, d count, then per friend
	// d object ID, S name, d online, d object ID if online else 0.
	m3Check(t, "empty FriendList", NewFriendListPacket(nil), unhex(t, "fa 00000000"))

	friends := []Friend{
		{ObjectID: 0x10000010, Name: "Alice", Online: true},
		{ObjectID: 0x10000020, Name: "Bob", Online: false},
	}
	var g m3Golden
	g.c(0xfa).d(2)
	g.d(0x10000010).s("Alice").d(1).d(0x10000010)
	g.d(0x10000020).s("Bob").d(0).d(0)
	m3Check(t, "FriendList", NewFriendListPacket(friends), g.b)
}
