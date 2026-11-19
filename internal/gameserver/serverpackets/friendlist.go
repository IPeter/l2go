package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpFriendList is the opcode of FriendList.
const OpFriendList = 0xfa

// Friend is one entry of the friend list.
type Friend struct {
	ObjectID int32
	Name     string
	Online   bool
}

// NewFriendListPacket builds FriendList (Mobius FriendList.writeImpl), sent
// once at login. An empty list is valid.
func NewFriendListPacket(friends []Friend) []byte {
	w := packets.NewWriter(5 + len(friends)*48)
	w.WriteUint8(OpFriendList)
	putCount32(w, len(friends))
	for _, f := range friends {
		w.WriteInt32(f.ObjectID)
		w.WriteString(f.Name)
		putBool32(w, f.Online)
		if f.Online {
			w.WriteInt32(f.ObjectID)
		} else {
			w.WriteInt32(0)
		}
	}
	return w.Bytes()
}
