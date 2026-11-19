package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpEnterWorld is the opcode of EnterWorld.
const OpEnterWorld = 0x03

// enterWorldUnknownBlock is the size of each of the two unknown byte blocks
// of EnterWorld.
const enterWorldUnknownBlock = 32

// EnterWorld asks to enter the world with the character chosen by
// CharacterSelect, once the loading screen is done.
type EnterWorld struct {
	// Tracert is the client's traceroute: five IPv4 addresses. Mobius
	// passes it to the login server and logs it; it is not trusted.
	Tracert [5][4]byte
}

// ParseEnterWorld parses an EnterWorld body (Mobius EnterWorld.readImpl):
// 32 unknown bytes, four unknown ints, 32 unknown bytes, an unknown int and
// the traceroute, 104 bytes in all. Like Mobius it needs all of them; a
// shorter body is an error.
func ParseEnterWorld(data []byte) (EnterWorld, error) {
	r := packets.NewReader(data)
	r.ReadBytes(enterWorldUnknownBlock)
	for range 4 {
		r.ReadInt32()
	}
	r.ReadBytes(enterWorldUnknownBlock)
	r.ReadInt32()
	var e EnterWorld
	for i := range e.Tracert {
		for j := range e.Tracert[i] {
			e.Tracert[i][j] = r.ReadUint8()
		}
	}
	if err := r.Err(); err != nil {
		return EnterWorld{}, fmt.Errorf("EnterWorld: %w", err)
	}
	return e, nil
}
