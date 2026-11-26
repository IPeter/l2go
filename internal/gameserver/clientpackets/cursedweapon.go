package clientpackets

// Sub-opcodes of the cursed weapon packets (Mobius ExClientPackets).
const (
	ExOpRequestCursedWeaponList     = 0x22
	ExOpRequestCursedWeaponLocation = 0x23
)

// RequestCursedWeaponList asks for the cursed weapons (the server answers
// with ExCursedWeaponList). It has no fields.
type RequestCursedWeaponList struct{}

// ParseRequestCursedWeaponList parses a RequestCursedWeaponList body, the
// ExPacket.Body after the sub-opcode. Like Mobius it reads nothing, so it
// never fails.
func ParseRequestCursedWeaponList([]byte) (RequestCursedWeaponList, error) {
	return RequestCursedWeaponList{}, nil
}

// RequestCursedWeaponLocation asks where the cursed weapons are (the
// server answers with ExCursedWeaponLocation when one is in the world). It
// has no fields.
type RequestCursedWeaponLocation struct{}

// ParseRequestCursedWeaponLocation parses a RequestCursedWeaponLocation
// body. Like Mobius it reads nothing, so it never fails.
func ParseRequestCursedWeaponLocation([]byte) (RequestCursedWeaponLocation, error) {
	return RequestCursedWeaponLocation{}, nil
}
