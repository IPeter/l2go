package testclient

// The clan hall packets (Plan 14 M11j). The reader follows the writeImpl of
// the Mobius class of the same name, field by field.

// OpAgitDecoInfo is the game server opcode of AgitDecoInfo (Mobius
// ServerPackets.AGIT_DECO_INFO).
const OpAgitDecoInfo byte = 0xf7

func init() {
	serverPackets[OpAgitDecoInfo] = serverPacket{"AgitDecoInfo", readAgitDecoInfo}
}

// AgitDecoInfo is the furniture of an auctionable clan hall (0xf7): each
// byte is 0 (none), 1 (small) or 2 (large).
type AgitDecoInfo struct {
	HallID int32
	// Deco are the 13 bytes in Mobius's order: fireplace (HP), carpet and
	// statue (MP), chandelier (exp), mirror (teleport), crystal, curtain,
	// magic curtain (item creation), flag twice (support), front platform,
	// item creation.
	Deco [13]uint8
}

// PacketName implements Packet.
func (AgitDecoInfo) PacketName() string { return "AgitDecoInfo" }

func readAgitDecoInfo(f *fields) Packet {
	var p AgitDecoInfo
	p.HallID = f.i32()
	for i := range p.Deco {
		p.Deco[i] = f.u8()
		if p.Deco[i] > 2 {
			f.failf("decoration %d is %d, want 0 to 2", i, p.Deco[i])
		}
	}
	return p
}
