package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// Opcodes of the henna packets (Mobius ClientPackets REQUEST_HENNA_*): the
// symbol maker's windows (M11c).
const (
	OpRequestHennaItemList       = 0xba
	OpRequestHennaItemInfo       = 0xbb
	OpRequestHennaEquip          = 0xbc
	OpRequestHennaRemoveList     = 0xbd
	OpRequestHennaItemRemoveInfo = 0xbe
	OpRequestHennaRemove         = 0xbf
)

// RequestHenna is the body of every henna packet: one int. For the lists
// (RequestHennaItemList, RequestHennaRemoveList) Mobius reads and ignores it;
// for the others it is the dye (symbol) ID.
type RequestHenna struct {
	SymbolID int32
}

// parseRequestHenna parses a henna packet body (Mobius readImpl: d).
func parseRequestHenna(name string, data []byte) (RequestHenna, error) {
	r := packets.NewReader(data)
	id := r.ReadInt32()
	if err := r.Err(); err != nil {
		return RequestHenna{}, fmt.Errorf("%s: %w", name, err)
	}
	return RequestHenna{SymbolID: id}, nil
}

// ParseRequestHennaItemList parses RequestHennaItemList (0xBA): d unknown.
func ParseRequestHennaItemList(data []byte) (RequestHenna, error) {
	return parseRequestHenna("RequestHennaItemList", data)
}

// ParseRequestHennaItemInfo parses RequestHennaItemInfo (0xBB): d symbol ID.
func ParseRequestHennaItemInfo(data []byte) (RequestHenna, error) {
	return parseRequestHenna("RequestHennaItemInfo", data)
}

// ParseRequestHennaEquip parses RequestHennaEquip (0xBC): d symbol ID.
func ParseRequestHennaEquip(data []byte) (RequestHenna, error) {
	return parseRequestHenna("RequestHennaEquip", data)
}

// ParseRequestHennaRemoveList parses RequestHennaRemoveList (0xBD): d
// unknown.
func ParseRequestHennaRemoveList(data []byte) (RequestHenna, error) {
	return parseRequestHenna("RequestHennaRemoveList", data)
}

// ParseRequestHennaItemRemoveInfo parses RequestHennaItemRemoveInfo (0xBE): d
// symbol ID.
func ParseRequestHennaItemRemoveInfo(data []byte) (RequestHenna, error) {
	return parseRequestHenna("RequestHennaItemRemoveInfo", data)
}

// ParseRequestHennaRemove parses RequestHennaRemove (0xBF): d symbol ID.
func ParseRequestHennaRemove(data []byte) (RequestHenna, error) {
	return parseRequestHenna("RequestHennaRemove", data)
}
