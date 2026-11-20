package serverpackets

// ExOpSetCompassZoneCode is the sub-opcode of ExSetCompassZoneCode.
const ExOpSetCompassZoneCode = 0x32

// CompassZone is the kind of area the client's compass shows (Mobius
// ExSetCompassZoneCode constants).
type CompassZone int32

// The compass zone codes.
const (
	CompassAltered    CompassZone = 0x08
	CompassSiegeWar1  CompassZone = 0x0a
	CompassSiegeWar2  CompassZone = 0x0b
	CompassPeace      CompassZone = 0x0c
	CompassSevenSigns CompassZone = 0x0d
	CompassPvP        CompassZone = 0x0e
	CompassGeneral    CompassZone = 0x0f
)

// NewExSetCompassZoneCodePacket builds ExSetCompassZoneCode (Mobius
// ExSetCompassZoneCode.writeImpl): the zone code.
func NewExSetCompassZoneCodePacket(z CompassZone) []byte {
	w := newExWriter(ExOpSetCompassZoneCode, 4)
	w.WriteInt32(int32(z))
	return w.Bytes()
}
