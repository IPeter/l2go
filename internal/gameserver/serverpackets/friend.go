package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// Opcodes of the friend packets (Mobius ServerPackets; FriendAddRequest is
// ASK_JOIN_FRIEND, FriendStatusPacket FRIEND_STATUS and L2FriendSay
// FRIEND_RECV_MSG). FriendList is OpFriendList.
const (
	OpFriendAddRequest   = 0x7d
	OpL2Friend           = 0xfb
	OpFriendStatusPacket = 0xfc
	OpL2FriendSay        = 0xfd
)

// The actions of L2Friend (Mobius: 1 add, 2 modify, 3 remove).
const (
	L2FriendAdd    int32 = 1
	L2FriendModify int32 = 2
	L2FriendRemove int32 = 3
)

// NewFriendAddRequestPacket builds FriendAddRequest (Mobius
// FriendAddRequest.writeImpl), the friend invitation window: S inviter
// name, d 0.
func NewFriendAddRequestPacket(requestor string) []byte {
	w := packets.NewWriter(7 + 2*len(requestor))
	w.WriteUint8(OpFriendAddRequest)
	w.WriteString(requestor)
	w.WriteInt32(0)
	return w.Bytes()
}

// L2Friend is a change of one entry of the friend window.
type L2Friend struct {
	// Action is L2FriendAdd, L2FriendModify or L2FriendRemove.
	Action int32
	Name   string
	Online bool
	// ObjectID is 0 when Mobius builds the packet from a name only (the
	// removal of a friend).
	ObjectID int32
}

// NewL2FriendPacket builds L2Friend (Mobius L2Friend.writeImpl): d action,
// d 0, S name, d online, d object ID.
func NewL2FriendPacket(f L2Friend) []byte {
	w := packets.NewWriter(15 + 2*len(f.Name))
	w.WriteUint8(OpL2Friend)
	w.WriteInt32(f.Action)
	w.WriteInt32(0)
	w.WriteString(f.Name)
	putBool32(w, f.Online)
	w.WriteInt32(f.ObjectID)
	return w.Bytes()
}

// NewFriendStatusPacket builds FriendStatusPacket (Mobius
// FriendStatusPacket.writeImpl), a friend logging in or out: d online,
// S name, d object ID.
func NewFriendStatusPacket(online bool, name string, objectID int32) []byte {
	w := packets.NewWriter(11 + 2*len(name))
	w.WriteUint8(OpFriendStatusPacket)
	putBool32(w, online)
	w.WriteString(name)
	w.WriteInt32(objectID)
	return w.Bytes()
}

// NewL2FriendSayPacket builds L2FriendSay (Mobius L2FriendSay.writeImpl),
// a friend message: d 0, S receiver, S sender, S message. The receiver
// comes before the sender.
func NewL2FriendSayPacket(sender, receiver, message string) []byte {
	w := packets.NewWriter(11 + 2*(len(sender)+len(receiver)+len(message)))
	w.WriteUint8(OpL2FriendSay)
	w.WriteInt32(0)
	w.WriteString(receiver)
	w.WriteString(sender)
	w.WriteString(message)
	return w.Bytes()
}
