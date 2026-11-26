package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpAgitDecoInfo is the opcode of AgitDecoInfo (Mobius
// ServerPackets.AGIT_DECO_INFO).
const OpAgitDecoInfo = 0xf7

// AgitDeco is what AgitDecoInfo shows of an auctionable clan hall: its ID,
// grade and the levels of its functions (0 for a function not rented),
// which the client turns into the hall's furniture.
type AgitDeco struct {
	HallID int32
	Grade  int
	// The function levels (Mobius ClanHall.FUNC_*).
	HP, MP, Exp, Teleport, Curtains, ItemCreate, Support, FrontPlatform int
}

// NewAgitDecoInfoPacket builds AgitDecoInfo (Mobius AgitDecoInfo.writeImpl):
//
//	c 0xf7, d hall, then the decoration bytes 0 (none), 1 (small) or 2
//	(large): fireplace (HP), carpet and statue (MP, twice), chandelier
//	(exp), mirror (teleport), crystal (always 0), curtain, magic curtain
//	(item creation), flag (support, twice), front platform, item creation.
func NewAgitDecoInfoPacket(a AgitDeco) []byte {
	w := packets.NewWriter(18)
	w.WriteUint8(OpAgitDecoInfo)
	w.WriteInt32(a.HallID)
	g := a.Grade
	deco := func(level int, small bool) uint8 {
		switch {
		case level == 0:
			return 0
		case small:
			return 1
		}
		return 2
	}
	w.WriteUint8(deco(a.HP, (g == 0 && a.HP < 220) || (g == 1 && a.HP < 160) || (g == 2 && a.HP < 260) || (g == 3 && a.HP < 300)))
	mp := deco(a.MP, ((g == 0 || g == 1) && a.MP < 25) || (g == 2 && a.MP < 30) || (g == 3 && a.MP < 40))
	w.WriteUint8(mp)
	w.WriteUint8(mp)
	w.WriteUint8(deco(a.Exp, (g == 0 && a.Exp < 25) || (g == 1 && a.Exp < 30) || (g == 2 && a.Exp < 40) || (g == 3 && a.Exp < 50)))
	w.WriteUint8(deco(a.Teleport, a.Teleport < 2))
	w.WriteUint8(0) // crystal
	w.WriteUint8(deco(a.Curtains, a.Curtains <= 1))
	item := deco(a.ItemCreate, (g == 0 && a.ItemCreate < 2) || a.ItemCreate < 3)
	w.WriteUint8(item)
	sup := deco(a.Support, (g == 0 && a.Support < 2) || (g == 1 && a.Support < 4) || (g == 2 && a.Support < 5) || (g == 3 && a.Support < 8))
	w.WriteUint8(sup)
	w.WriteUint8(sup)
	w.WriteUint8(deco(a.FrontPlatform, a.FrontPlatform <= 1))
	w.WriteUint8(item)
	return w.Bytes()
}
