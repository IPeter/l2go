package clientpackets

// OpRequestItemList is the opcode of RequestItemList.
const OpRequestItemList = 0x0f

// RequestItemList asks for the inventory window (Mobius answers ItemList
// with the window open). It has no fields.
type RequestItemList struct{}

// ParseRequestItemList parses a RequestItemList body. Like Mobius it reads
// nothing, so it never fails.
func ParseRequestItemList([]byte) (RequestItemList, error) { return RequestItemList{}, nil }
