package clientpackets

import (
	"encoding/binary"
	"unicode/utf16"
)

// OpRequestBypassToServer is the opcode of RequestBypassToServer.
const OpRequestBypassToServer = 0x21

// RequestBypassToServer is a click on a "bypass" action of an HTML page.
// Command is the text after "bypass" (and after its "-h" flag), with the
// values the player typed in place of the $parameters.
type RequestBypassToServer struct {
	Command string
}

// ParseRequestBypassToServer parses a RequestBypassToServer body (Mobius
// RequestBypassToServer.readImpl). Like the Mobius readString it never
// fails: a command without its terminator runs to the end of the data, and
// an empty body is an empty command, which Mobius answers by disconnecting
// the player.
func ParseRequestBypassToServer(data []byte) (RequestBypassToServer, error) {
	return RequestBypassToServer{Command: htmlActionText(data)}, nil
}

// htmlActionText reads the only field of RequestBypassToServer and
// RequestLinkHtml the way the Mobius ReadablePacket.readString does: UTF-16LE
// code units up to a zero unit, or up to the end of the data if there is
// none. An odd last byte is ignored.
func htmlActionText(data []byte) string {
	units := make([]uint16, 0, len(data)/2)
	for i := 0; i+1 < len(data); i += 2 {
		u := binary.LittleEndian.Uint16(data[i:])
		if u == 0 {
			break
		}
		units = append(units, u)
	}
	return string(utf16.Decode(units))
}
