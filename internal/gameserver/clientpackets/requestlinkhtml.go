package clientpackets

// OpRequestLinkHtml is the opcode of RequestLinkHtml.
const OpRequestLinkHtml = 0x20

// RequestLinkHtml is a click on a "link" action of an HTML page. Link is
// the page to open, relative to the HTML folder. Mobius refuses an empty
// link, a link with "..", and a link the player's link cache does not hold
// as "link " + Link.
type RequestLinkHtml struct {
	Link string
}

// ParseRequestLinkHtml parses a RequestLinkHtml body (Mobius
// RequestLinkHtml.readImpl). Like the Mobius readString it never fails: a
// link without its terminator runs to the end of the data.
func ParseRequestLinkHtml(data []byte) (RequestLinkHtml, error) {
	return RequestLinkHtml{Link: htmlActionText(data)}, nil
}
