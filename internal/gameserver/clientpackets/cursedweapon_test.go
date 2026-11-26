package clientpackets

import "testing"

// RequestCursedWeaponList (D0 22 00) and RequestCursedWeaponLocation (D0 23
// 00) carry nothing after the sub-opcode; extra bytes are ignored, as Mobius
// reads nothing.
func TestCursedWeaponRequests(t *testing.T) {
	for _, body := range [][]byte{{0x22, 0x00}, {0x23, 0x00}} {
		ex, err := ParseExPacket(body)
		if err != nil {
			t.Fatalf("ParseExPacket(% x): %v", body, err)
		}
		switch ex.SubOpcode {
		case ExOpRequestCursedWeaponList:
			if _, err := ParseRequestCursedWeaponList(ex.Body); err != nil {
				t.Errorf("ParseRequestCursedWeaponList: %v", err)
			}
		case ExOpRequestCursedWeaponLocation:
			if _, err := ParseRequestCursedWeaponLocation(append(ex.Body, 1, 2, 3)); err != nil {
				t.Errorf("ParseRequestCursedWeaponLocation: %v", err)
			}
		default:
			t.Errorf("sub-opcode %#x", ex.SubOpcode)
		}
	}
}
